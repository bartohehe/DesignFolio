package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"designfolio/internal/media"
	"designfolio/internal/model"
	"designfolio/internal/store"
)

// uploadedAsset opisuje jeden plik w odpowiedzi na upload.
type uploadedAsset struct {
	Asset     *model.Asset `json:"asset"`
	Duplicate bool         `json:"duplicate"`
}

type uploadResponse struct {
	Uploaded []uploadedAsset `json:"uploaded"`
	Skipped  []skippedFile   `json:"skipped"`
}

type skippedFile struct {
	Filename string `json:"filename"`
	Reason   string `json:"reason"`
}

// uploadAssets przyjmuje wiele plików naraz z formularza multipart.
//
// Używamy r.MultipartReader() zamiast r.ParseMultipartForm(), bo ten drugi
// wczytuje CAŁE żądanie do pamięci i plików tymczasowych, zanim odda sterowanie.
// Przy modelu 3D albo serii zdjęć z aparatu to setki megabajtów. MultipartReader
// daje strumień: każdy plik jest przepisywany na dysk w locie.
func (h *handlers) uploadAssets(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")

	item, err := h.store.GetItem(r.Context(), itemID)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, CodeNotFound, "nie ma itemu o id "+itemID)
		return
	}
	if err != nil {
		writeInternal(w, err, "uploadAssets: pobranie itemu")
		return
	}

	// Górny limit na całe żądanie. Pojedynczy plik ma własny limit w media.Storage.
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUpload*maxFilesPerRequest)

	reader, err := r.MultipartReader()
	if err != nil {
		writeErr(w, http.StatusUnsupportedMediaType, CodeUnsupportedType,
			"oczekiwano formularza multipart/form-data")
		return
	}

	resp := uploadResponse{Uploaded: []uploadedAsset{}, Skipped: []skippedFile{}}

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			writeErr(w, http.StatusBadRequest, CodeBadRequest, "uszkodzony formularz: "+err.Error())
			return
		}

		// Interesują nas tylko części z plikiem; zwykłe pola formularza pomijamy.
		if part.FileName() == "" {
			_ = part.Close()
			continue
		}

		uploaded, skipped := h.saveOnePart(r, item.ID, part)
		_ = part.Close()

		if skipped != nil {
			resp.Skipped = append(resp.Skipped, *skipped)
			continue
		}
		if uploaded != nil {
			resp.Uploaded = append(resp.Uploaded, *uploaded)
		}
	}

	if len(resp.Uploaded) == 0 && len(resp.Skipped) == 0 {
		writeErr(w, http.StatusBadRequest, CodeBadRequest, "żądanie nie zawierało żadnego pliku")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// maxFilesPerRequest ogranicza sumaryczny rozmiar żądania do wielokrotności
// limitu pojedynczego pliku.
const maxFilesPerRequest = 20

// saveOnePart zapisuje jeden plik. Zwraca albo wynik, albo powód pominięcia —
// jeden zły plik nie może wywrócić całego uploadu, w którym reszta jest w porządku.
func (h *handlers) saveOnePart(
	r *http.Request, itemID string, part io.Reader,
) (*uploadedAsset, *skippedFile) {
	filename := "plik"
	if named, ok := part.(interface{ FileName() string }); ok {
		filename = named.FileName()
	}

	assetID := uuid.NewString()

	saved, err := h.media.Save(assetID, filename, part)
	if err != nil {
		if errors.Is(err, media.ErrTooLarge) {
			return nil, &skippedFile{Filename: filename, Reason: "plik przekracza limit rozmiaru"}
		}
		slog.Error("zapis pliku nie powiódł się", "plik", filename, "err", err)
		return nil, &skippedFile{Filename: filename, Reason: "nie udało się zapisać"}
	}

	// Deduplikacja w obrębie itemu: to samo zdjęcie wrzucone dwa razy
	// nie zakłada drugiego rekordu. Plik z dysku sprzątamy.
	if existing, err := h.store.FindAssetByChecksum(r.Context(), itemID, saved.Checksum); err == nil {
		if removeErr := h.media.Remove(assetID); removeErr != nil {
			slog.Warn("nie udało się usunąć duplikatu z dysku", "asset", assetID, "err", removeErr)
		}
		return &uploadedAsset{Asset: existing, Duplicate: true}, nil
	} else if !errors.Is(err, store.ErrNotFound) {
		slog.Error("sprawdzenie duplikatu nie powiodło się", "err", err)
	}

	asset := model.Asset{
		ID:       assetID,
		ItemID:   itemID,
		Filename: saved.Filename,
		Mime:     saved.Mime,
		Size:     saved.Size,
		Kind:     saved.Kind,
		Width:    saved.Width,
		Height:   saved.Height,
		Checksum: saved.Checksum,
		HasThumb: saved.HasThumb,
	}

	if err := h.store.CreateAsset(r.Context(), asset); err != nil {
		// Zapis do bazy padł, więc plik na dysku jest niczyj — kasujemy go od razu.
		if removeErr := h.media.Remove(assetID); removeErr != nil {
			slog.Warn("nie udało się posprzątać po nieudanym zapisie", "asset", assetID, "err", removeErr)
		}
		slog.Error("zapis metadanych pliku nie powiódł się", "plik", filename, "err", err)
		return nil, &skippedFile{Filename: filename, Reason: "nie udało się zapisać metadanych"}
	}

	stored, err := h.store.GetAsset(r.Context(), assetID)
	if err != nil {
		return nil, &skippedFile{Filename: filename, Reason: "zapisany, ale nie udało się go odczytać"}
	}
	return &uploadedAsset{Asset: stored}, nil
}

// arrangeRequest opisuje nową kolejność plików i wybór okładki.
type arrangeRequest struct {
	Order []string `json:"order"`
	Cover *string  `json:"cover"`
}

func (h *handlers) arrangeAssets(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")

	var req arrangeRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	cover := ""
	if req.Cover != nil {
		cover = *req.Cover
	}

	err := h.store.ArrangeAssets(r.Context(), itemID, req.Order, cover)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, CodeNotFound, err.Error())
		return
	}
	if err != nil {
		writeInternal(w, err, "arrangeAssets")
		return
	}

	assets, err := h.store.AssetsForItem(r.Context(), itemID)
	if err != nil {
		writeInternal(w, err, "arrangeAssets: odczyt")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assets": assets})
}

func (h *handlers) deleteAsset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Zwracany identyfikator itemu nie jest tu potrzebny — indeks wyszukiwania
	// obejmuje tytuł, opis i tagi, więc zmiana plików go nie dotyczy.
	_, err := h.store.DeleteAsset(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, CodeNotFound, "nie ma pliku o id "+id)
		return
	}
	if err != nil {
		writeInternal(w, err, "deleteAsset")
		return
	}

	// Rekord zniknął — teraz dysk. Odwrotna kolejność zostawiłaby rekord
	// wskazujący na nieistniejący plik.
	if err := h.media.Remove(id); err != nil {
		slog.Warn("nie udało się usunąć plików z dysku", "asset", id, "err", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

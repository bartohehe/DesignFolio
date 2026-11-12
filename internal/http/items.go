package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"designfolio/internal/model"
	"designfolio/internal/store"
)

// maxJSONBody chroni przed przypadkowym wysłaniem ogromnego dokumentu.
// Opis w markdownie ma limit 100 kB, więc 1 MB to duży zapas.
const maxJSONBody = 1 << 20

// decodeJSON czyta ciało żądania do wskazanej struktury.
//
// DisallowUnknownFields sprawia, że literówka w nazwie pola ("titel")
// kończy się czytelnym błędem, a nie po cichu zignorowaną zmianą.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeErr(w, http.StatusRequestEntityTooLarge, CodeTooLarge, "ciało żądania jest za duże")
			return false
		}
		writeErr(w, http.StatusBadRequest, CodeBadRequest, "nieprawidłowy JSON: "+err.Error())
		return false
	}

	// Drugie wywołanie Decode musi zwrócić io.EOF. Cokolwiek innego znaczy,
	// że po pierwszym obiekcie w ciele było jeszcze coś — prawie zawsze pomyłka.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeErr(w, http.StatusBadRequest, CodeBadRequest, "oczekiwano dokładnie jednego obiektu JSON")
		return false
	}
	return true
}

// writeValidationErr zamienia błąd walidacji z pakietu model na odpowiedź HTTP.
func writeValidationErr(w http.ResponseWriter, err error) bool {
	var ve model.ValidationError
	if errors.As(err, &ve) {
		writeErr(w, http.StatusBadRequest, CodeBadRequest, ve.Error())
		return true
	}
	return false
}

func (h *handlers) listItems(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := store.ListFilter{
		Query: q.Get("q"),
		Sort:  q.Get("sort"),
		Order: q.Get("order"),
		// Limit przycinamy tutaj, na granicy z użytkownikiem — store ufa
		// swoim wywołującym i sam nie narzuca górnej wartości.
		Limit:  min(atoiOr(q.Get("limit"), 0), store.MaxLimit),
		Offset: atoiOr(q.Get("offset"), 0),
	}

	// Parametry wielowartościowe przyjmujemy w dwóch formach:
	// ?kind=photo&kind=drawing oraz ?kind=photo,drawing.
	for _, raw := range multiValue(q["kind"]) {
		kind := model.Kind(raw)
		if !kind.Valid() {
			writeErr(w, http.StatusBadRequest, CodeBadRequest, "nieznany kind: "+raw)
			return
		}
		filter.Kinds = append(filter.Kinds, kind)
	}

	for _, raw := range multiValue(q["status"]) {
		status := model.Status(raw)
		if !status.Valid() {
			writeErr(w, http.StatusBadRequest, CodeBadRequest, "nieznany status: "+raw)
			return
		}
		filter.Statuses = append(filter.Statuses, status)
	}

	for _, raw := range multiValue(q["tag"]) {
		if slug := model.Slugify(raw); slug != "" {
			filter.TagSlugs = append(filter.TagSlugs, slug)
		}
	}

	if raw := q.Get("favorite"); raw != "" {
		fav, err := strconv.ParseBool(raw)
		if err != nil {
			writeErr(w, http.StatusBadRequest, CodeBadRequest, "favorite musi być true albo false")
			return
		}
		filter.Favorite = &fav
	}

	result, err := h.store.ListItems(r.Context(), filter)
	if err != nil {
		writeInternal(w, err, "listItems")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *handlers) createItem(w http.ResponseWriter, r *http.Request) {
	var in model.ItemInput
	if !decodeJSON(w, r, &in) {
		return
	}

	in.Normalize()
	if err := in.Validate(); err != nil {
		if writeValidationErr(w, err) {
			return
		}
		writeInternal(w, err, "createItem: walidacja")
		return
	}

	item, err := h.store.CreateItem(r.Context(), in)
	if err != nil {
		writeInternal(w, err, "createItem")
		return
	}

	w.Header().Set("Location", "/api/items/"+item.ID)
	writeJSON(w, http.StatusCreated, item)
}

func (h *handlers) getItem(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "idOrSlug")

	item, err := h.store.GetItem(r.Context(), idOrSlug)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, CodeNotFound, "nie ma itemu o identyfikatorze "+idOrSlug)
		return
	}
	if err != nil {
		writeInternal(w, err, "getItem")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *handlers) updateItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var patch model.ItemPatch
	if !decodeJSON(w, r, &patch) {
		return
	}

	patch.Normalize()
	if err := patch.Validate(); err != nil {
		if writeValidationErr(w, err) {
			return
		}
		writeInternal(w, err, "updateItem: walidacja")
		return
	}

	item, err := h.store.UpdateItem(r.Context(), id, patch)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, CodeNotFound, "nie ma itemu o id "+id)
		return
	}
	if err != nil {
		writeInternal(w, err, "updateItem")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *handlers) deleteItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	assetIDs, err := h.store.DeleteItem(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, CodeNotFound, "nie ma itemu o id "+id)
		return
	}
	if err != nil {
		writeInternal(w, err, "deleteItem")
		return
	}

	// Baza jest już po commicie, więc sprzątamy dysk. Nieudane kasowanie
	// zostawia osierocony katalog — logujemy je, ale nie zwracamy błędu:
	// z punktu widzenia użytkownika item zniknął i to jest prawda.
	for _, assetID := range assetIDs {
		if err := h.media.Remove(assetID); err != nil {
			slog.Warn("nie udało się usunąć plików", "asset", assetID, "err", err)
		}
	}

	// 204: operacja się udała i nie ma czego zwracać.
	w.WriteHeader(http.StatusNoContent)
}

// multiValue rozwija ?tag=a,b&tag=c do []string{"a", "b", "c"}.
func multiValue(values []string) []string {
	out := []string{}
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func atoiOr(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

package httpapi

import (
	"errors"
	"io/fs"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"designfolio/internal/store"
)

// serveFile oddaje zawartość pliku w jednym z trzech wariantów:
// original, thumb, preview.
func (h *handlers) serveFile(w http.ResponseWriter, r *http.Request) {
	assetID := chi.URLParam(r, "assetID")
	variant := chi.URLParam(r, "variant")

	switch variant {
	case "original", "thumb", "preview":
	default:
		writeErr(w, http.StatusBadRequest, CodeBadRequest,
			"wariant musi być jednym z: original, thumb, preview")
		return
	}

	asset, err := h.store.GetAsset(r.Context(), assetID)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, CodeNotFound, "nie ma pliku o id "+assetID)
		return
	}
	if err != nil {
		writeInternal(w, err, "serveFile: odczyt metadanych")
		return
	}

	file, path, err := h.media.Open(assetID, asset.Filename, variant)
	if errors.Is(err, fs.ErrNotExist) {
		writeErr(w, http.StatusNotFound, CodeNotFound, "nie ma wariantu "+variant+" tego pliku")
		return
	}
	if err != nil {
		writeInternal(w, err, "serveFile: otwarcie pliku")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		writeInternal(w, err, "serveFile: stat")
		return
	}

	// ETag z sumy kontrolnej treści: ta sama zawartość zawsze da ten sam znacznik,
	// a plik pod danym id nigdy się nie zmienia (edycja = nowy asset).
	// Dlatego można go cache'ować na zawsze.
	w.Header().Set("ETag", strconv.Quote(asset.Checksum+"-"+variant))
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")

	if variant == "original" {
		// inline, nie attachment: przeglądarka ma pokazać zdjęcie albo PDF,
		// a nie od razu je pobierać. Nazwa i tak trafia do okna zapisu.
		w.Header().Set("Content-Disposition", "inline; filename*=UTF-8''"+urlEncode(asset.Filename))
	}

	// ServeContent sam obsłuży nagłówki Range (przewijanie dużych plików),
	// If-None-Match i Content-Type wywiedziony z rozszerzenia.
	http.ServeContent(w, r, path, info.ModTime(), file)
}

// urlEncode koduje nazwę pliku na potrzeby nagłówka Content-Disposition
// w składni RFC 5987, żeby polskie znaki nie rozjechały się w przeglądarce.
func urlEncode(s string) string {
	const hex = "0123456789ABCDEF"
	out := make([]byte, 0, len(s))

	for i := 0; i < len(s); i++ {
		c := s[i]
		// Zbiór znaków bezpiecznych wg RFC 5987 (attr-char).
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '!' || c == '#' || c == '$' || c == '&' || c == '+' || c == '-' ||
			c == '.' || c == '^' || c == '_' || c == '`' || c == '|' || c == '~' {
			out = append(out, c)
			continue
		}
		out = append(out, '%', hex[c>>4], hex[c&0x0F])
	}
	return string(out)
}

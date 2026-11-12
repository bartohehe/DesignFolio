package httpapi

import (
	"errors"
	"io/fs"
	"net/http"
	"strings"
)

// spaHandler serwuje zbudowany frontend z systemu plików (w prod: z binarki,
// przez go:embed). Kluczowa jest obsługa "SPA fallback": React Router obsługuje
// ścieżki typu /items/moj-projekt po stronie przeglądarki, więc gdy plik o takiej
// nazwie nie istnieje, oddajemy index.html i to router decyduje, co pokazać.
//
// Gdy dist jest puste (frontend niezbudowany — normalne w trybie dev, gdzie
// stronę serwuje Vite), pokazujemy krótką instrukcję zamiast 404.
func spaHandler(dist fs.FS) http.Handler {
	if dist == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(
				"Frontend nie jest wbudowany w tę binarkę.\n\n" +
					"Tryb dev:  cd web && npm run dev   (strona pod http://localhost:5173)\n" +
					"Tryb prod: cd web && npm run build, potem przebuduj serwer.\n",
			))
		})
	}

	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// fs.FS używa ścieżek bez wiodącego ukośnika ("assets/app.js", nie "/assets/app.js").
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}

		// Vite dokleja hash do nazw plików w assets/, więc ta sama nazwa zawsze
		// znaczy tę samą treść — te można cache'ować na zawsze. WSZYSTKO inne,
		// a przede wszystkim index.html, musi być odświeżane przy każdym wejściu:
		// to on wskazuje na aktualne, zahaszowane pliki. Zapamiętany stary
		// index.html po wdrożeniu prowadzi do plików, których już nie ma —
		// czyli do białej strony.
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		info, err := fs.Stat(dist, name)
		switch {
		case err == nil && !info.IsDir():
			fileServer.ServeHTTP(w, r)

		case errors.Is(err, fs.ErrNotExist) || (err == nil && info.IsDir()):
			// Nieznana ścieżka → oddajemy powłokę aplikacji.
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)

		default:
			writeInternal(w, err, "static: stat")
		}
	})
}

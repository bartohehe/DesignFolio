// Package httpapi składa router i trzyma handlery HTTP.
//
// Nazwa pakietu celowo różni się od nazwy katalogu (internal/http): gdyby pakiet
// nazywał się "http", każdy plik importujący net/http musiałby go aliasować.
// Go tego nie zabrania — nazwa pakietu i katalogu to dwie osobne rzeczy.
package httpapi

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"

	"designfolio/internal/media"
	"designfolio/internal/store"
)

// Options to wszystko, czego router potrzebuje z zewnątrz. Dzięki temu
// pakiet nie sięga sam po konfigurację ani po zmienne globalne.
type Options struct {
	// Version trafia do /api/health — przydatne, żeby sprawdzić, co właściwie chodzi.
	Version string

	// Token pusty = middleware autoryzacji wyłączony.
	Token string

	// Store to warstwa danych. Wymagany.
	Store *store.Store

	// Media obsługuje pliki na dysku. Wymagane.
	Media *media.Storage

	// MaxUploadBytes to limit rozmiaru pojedynczego pliku.
	MaxUploadBytes int64

	// Static to system plików ze zbudowanym frontendem; nil, gdy go nie ma.
	// W trybie dev jest nil — stronę serwuje Vite, a on proxuje /api tutaj,
	// więc requesty są same-origin i nie potrzebujemy CORS-a.
	Static fs.FS
}

// handlers zbiera zależności współdzielone przez wszystkie handlery.
// Metody na tej strukturze zastępują globalne zmienne i długie listy
// argumentów przekazywanych do każdej funkcji z osobna.
type handlers struct {
	store     *store.Store
	media     *media.Storage
	maxUpload int64
}

// NewRouter buduje kompletny handler aplikacji: API pod /api, reszta to frontend.
func NewRouter(opts Options) http.Handler {
	h := &handlers{
		store:     opts.Store,
		media:     opts.Media,
		maxUpload: opts.MaxUploadBytes,
	}

	r := chi.NewRouter()

	r.Use(recoverer)
	r.Use(requestLogger)

	r.Route("/api", func(api chi.Router) {
		api.Use(requireTokenForWrites(opts.Token))

		api.Get("/health", healthHandler(opts.Version, opts.Store))

		api.Route("/items", func(items chi.Router) {
			items.Get("/", h.listItems)
			items.Post("/", h.createItem)

			// Odczyt przyjmuje id albo slug, zapisy wymagają id — slug
			// zmienia się przy zmianie tytułu, więc nie jest stabilnym uchwytem.
			items.Get("/{idOrSlug}", h.getItem)
			items.Patch("/{id}", h.updateItem)
			items.Delete("/{id}", h.deleteItem)

			// Pliki itemu: wrzucanie i układanie.
			items.Post("/{id}/assets", h.uploadAssets)
			items.Patch("/{id}/assets", h.arrangeAssets)
		})

		api.Delete("/assets/{id}", h.deleteAsset)

		// Zawartość pliku. Wariant: original | thumb | preview.
		api.Get("/files/{assetID}/{variant}", h.serveFile)

		api.Get("/tags", h.listTags)
		api.Get("/facets", h.getFacets)
		api.Get("/timeline", h.timeline)
		api.Get("/export", h.exportCatalog)

		// Nieznana ścieżka pod /api ma zwrócić błąd w kształcie API,
		// a nie HTML-ową powłokę frontendu.
		api.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeErr(w, http.StatusNotFound, CodeNotFound, "nie ma takiego endpointu: "+r.URL.Path)
		})
		api.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			writeErr(w, http.StatusMethodNotAllowed, CodeBadRequest, "metoda "+r.Method+" nie jest tu obsługiwana")
		})
	})

	// Wszystko poza /api obsługuje frontend.
	r.Handle("/*", spaHandler(opts.Static))

	return r
}

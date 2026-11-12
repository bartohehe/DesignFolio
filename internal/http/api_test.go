// Testy handlerów jadą przez prawdziwy router i prawdziwą bazę SQLite
// w pliku tymczasowym. Bez mocków — sprawdzamy to, co faktycznie pojedzie
// na produkcję, razem z migracjami, CHECK-ami i kluczami obcymi.
//
// Dlaczego plik, a nie ":memory:"? Baza w pamięci jest prywatna dla
// pojedynczego połączenia, a *sql.DB to pula połączeń — druga goroutine
// zobaczyłaby pustą bazę bez tabel.
package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	httpapi "designfolio/internal/http"
	"designfolio/internal/media"
	"designfolio/internal/model"
	"designfolio/internal/store"
)

// --- infrastruktura testowa -------------------------------------------------

func newTestServer(t *testing.T, token string) *httptest.Server {
	t.Helper()

	dir := t.TempDir()
	st, err := store.Open(context.Background(), filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("otwarcie bazy testowej: %v", err)
	}
	// Cleanupy lecą w odwrotnej kolejności rejestracji, więc baza zamknie się
	// przed sprzątaniem katalogu przez t.TempDir — na Windowsie to konieczne,
	// bo otwartego pliku nie da się usunąć.
	t.Cleanup(func() { _ = st.Close() })

	srv := httptest.NewServer(httpapi.NewRouter(httpapi.Options{
		Version:        "test",
		Token:          token,
		Store:          st,
		Media:          media.NewStorage(filepath.Join(dir, "files"), 10<<20),
		MaxUploadBytes: 10 << 20,
	}))
	t.Cleanup(srv.Close)

	return srv
}

type response struct {
	status int
	body   []byte
}

// decode rozpakowuje ciało odpowiedzi do wskazanej struktury.
func (r response) decode(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.body, dst); err != nil {
		t.Fatalf("nie mogę sparsować odpowiedzi %s: %v", r.body, err)
	}
}

// errorCode wyciąga kod błędu z odpowiedzi API.
func (r response) errorCode(t *testing.T) string {
	t.Helper()
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	r.decode(t, &body)
	return body.Error.Code
}

func do(t *testing.T, srv *httptest.Server, method, path, body string, headers ...[2]string) response {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}

	req, err := http.NewRequest(method, srv.URL+path, reader)
	if err != nil {
		t.Fatalf("budowa żądania: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, h := range headers {
		req.Header.Set(h[0], h[1])
	}

	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("wysłanie żądania %s %s: %v", method, path, err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("odczyt odpowiedzi: %v", err)
	}
	return response{status: res.StatusCode, body: raw}
}

func mustCreate(t *testing.T, srv *httptest.Server, body string) model.Item {
	t.Helper()

	res := do(t, srv, http.MethodPost, "/api/items", body)
	if res.status != http.StatusCreated {
		t.Fatalf("tworzenie itemu: status %d, ciało %s", res.status, res.body)
	}

	var item model.Item
	res.decode(t, &item)
	return item
}

// --- health -----------------------------------------------------------------

func TestHealth(t *testing.T) {
	srv := newTestServer(t, "")

	res := do(t, srv, http.MethodGet, "/api/health", "")
	if res.status != http.StatusOK {
		t.Fatalf("status = %d, oczekiwano 200; ciało %s", res.status, res.body)
	}

	var body struct {
		Status string `json:"status"`
		DB     string `json:"db"`
	}
	res.decode(t, &body)

	if body.Status != "ok" || body.DB != "ok" {
		t.Errorf("health = %+v, oczekiwano status=ok db=ok", body)
	}
}

// --- tworzenie i walidacja --------------------------------------------------

func TestCreateItem(t *testing.T) {
	srv := newTestServer(t, "")

	item := mustCreate(t, srv, `{
		"title": "Wieża Eiffla — szkic",
		"description": "Ołówek 2B",
		"kind": "drawing",
		"status": "done",
		"favorite": true,
		"happened_at": "2024-05-01T10:00:00Z",
		"metadata": {"medium": "ołówek", "czas_h": 3},
		"tags": ["Szkic", "architektura"]
	}`)

	if item.ID == "" {
		t.Error("id jest puste")
	}
	if want := "wieza-eiffla-szkic"; item.Slug != want {
		t.Errorf("slug = %q, oczekiwano %q", item.Slug, want)
	}
	if item.Kind != model.KindDrawing || item.Status != model.StatusDone {
		t.Errorf("kind/status = %q/%q", item.Kind, item.Status)
	}
	if !item.Favorite {
		t.Error("favorite = false, oczekiwano true")
	}
	if item.HappenedAt == nil || item.HappenedAt.Year() != 2024 {
		t.Errorf("happened_at = %v", item.HappenedAt)
	}
	if len(item.Tags) != 2 {
		t.Fatalf("liczba tagów = %d, oczekiwano 2: %+v", len(item.Tags), item.Tags)
	}
	if item.ExternalURL != nil {
		t.Errorf("external_url = %v, oczekiwano null", *item.ExternalURL)
	}
}

func TestCreateItemDefaults(t *testing.T) {
	srv := newTestServer(t, "")

	item := mustCreate(t, srv, `{"title": "Bez niczego", "kind": "note"}`)

	if item.Status != model.StatusWIP {
		t.Errorf("status = %q, oczekiwano domyślnego wip", item.Status)
	}
	if string(item.Metadata) != "{}" {
		t.Errorf("metadata = %s, oczekiwano {}", item.Metadata)
	}
	if item.Tags == nil {
		t.Error("tags = null, oczekiwano pustej tablicy")
	}
	if item.CreatedAt.IsZero() || item.UpdatedAt.IsZero() {
		t.Error("brakuje created_at albo updated_at")
	}
}

func TestCreateItemValidation(t *testing.T) {
	srv := newTestServer(t, "")

	tests := []struct {
		name string
		body string
	}{
		{"pusty tytuł", `{"title": "   ", "kind": "note"}`},
		{"brak kind", `{"title": "X"}`},
		{"nieznany kind", `{"title": "X", "kind": "kosmos"}`},
		{"nieznany status", `{"title": "X", "kind": "note", "status": "prawie"}`},
		{"url bez schematu", `{"title": "X", "kind": "link", "external_url": "example.com"}`},
		{"url ze złym schematem", `{"title": "X", "kind": "link", "external_url": "ftp://example.com"}`},
		{"metadata jako tablica", `{"title": "X", "kind": "note", "metadata": [1, 2]}`},
		{"metadata jako liczba", `{"title": "X", "kind": "note", "metadata": 42}`},
		{"nieznane pole", `{"titel": "literówka", "kind": "note"}`},
		{"niepoprawny JSON", `{"title": `},
		{"dwa obiekty w ciele", `{"title":"a","kind":"note"}{"title":"b","kind":"note"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := do(t, srv, http.MethodPost, "/api/items", tt.body)
			if res.status != http.StatusBadRequest {
				t.Fatalf("status = %d, oczekiwano 400; ciało %s", res.status, res.body)
			}
			if code := res.errorCode(t); code != "bad_request" {
				t.Errorf("kod błędu = %q, oczekiwano bad_request", code)
			}
		})
	}
}

func TestSlugCollision(t *testing.T) {
	srv := newTestServer(t, "")

	first := mustCreate(t, srv, `{"title": "Kubek", "kind": "print3d"}`)
	second := mustCreate(t, srv, `{"title": "Kubek", "kind": "print3d"}`)
	third := mustCreate(t, srv, `{"title": "Kubek", "kind": "print3d"}`)

	if first.Slug != "kubek" {
		t.Errorf("pierwszy slug = %q, oczekiwano kubek", first.Slug)
	}
	if second.Slug != "kubek-2" {
		t.Errorf("drugi slug = %q, oczekiwano kubek-2", second.Slug)
	}
	if third.Slug != "kubek-3" {
		t.Errorf("trzeci slug = %q, oczekiwano kubek-3", third.Slug)
	}
}

func TestSlugFallbackForUnslugifiableTitle(t *testing.T) {
	srv := newTestServer(t, "")

	item := mustCreate(t, srv, `{"title": "🎨🖌️", "kind": "drawing"}`)
	if item.Slug != "item" {
		t.Errorf("slug = %q, oczekiwano zastępczego 'item'", item.Slug)
	}
}

// --- pobieranie -------------------------------------------------------------

func TestGetItemByIDAndSlug(t *testing.T) {
	srv := newTestServer(t, "")
	created := mustCreate(t, srv, `{"title": "Lampa", "kind": "print3d", "tags": ["druk"]}`)

	for _, key := range []string{created.ID, created.Slug} {
		res := do(t, srv, http.MethodGet, "/api/items/"+key, "")
		if res.status != http.StatusOK {
			t.Fatalf("GET /api/items/%s: status %d", key, res.status)
		}

		var got model.Item
		res.decode(t, &got)
		if got.ID != created.ID {
			t.Errorf("id = %q, oczekiwano %q", got.ID, created.ID)
		}
		if len(got.Tags) != 1 || got.Tags[0].Slug != "druk" {
			t.Errorf("tagi = %+v, oczekiwano jednego o slugu 'druk'", got.Tags)
		}
	}
}

func TestGetItemNotFound(t *testing.T) {
	srv := newTestServer(t, "")

	res := do(t, srv, http.MethodGet, "/api/items/nie-ma-takiego", "")
	if res.status != http.StatusNotFound {
		t.Fatalf("status = %d, oczekiwano 404", res.status)
	}
	if code := res.errorCode(t); code != "not_found" {
		t.Errorf("kod = %q, oczekiwano not_found", code)
	}
}

// --- aktualizacja -----------------------------------------------------------

func TestPatchOnlyTouchesGivenFields(t *testing.T) {
	srv := newTestServer(t, "")
	created := mustCreate(t, srv, `{
		"title": "Stary tytuł",
		"description": "opis",
		"kind": "project",
		"external_url": "https://example.com/repo",
		"tags": ["go", "sqlite"]
	}`)

	res := do(t, srv, http.MethodPatch, "/api/items/"+created.ID, `{"status": "done"}`)
	if res.status != http.StatusOK {
		t.Fatalf("status = %d, ciało %s", res.status, res.body)
	}

	var updated model.Item
	res.decode(t, &updated)

	if updated.Status != model.StatusDone {
		t.Errorf("status = %q, oczekiwano done", updated.Status)
	}
	// Wszystko, czego nie było w żądaniu, musi zostać nietknięte.
	if updated.Title != created.Title || updated.Description != created.Description {
		t.Errorf("PATCH nadpisał pola, których nie podano: %+v", updated)
	}
	if updated.ExternalURL == nil || *updated.ExternalURL != "https://example.com/repo" {
		t.Errorf("external_url = %v, oczekiwano niezmienionego", updated.ExternalURL)
	}
	if len(updated.Tags) != 2 {
		t.Errorf("tagi = %+v, oczekiwano niezmienionych dwóch", updated.Tags)
	}
	if updated.Slug != created.Slug {
		t.Errorf("slug zmienił się na %q mimo braku zmiany tytułu", updated.Slug)
	}
}

func TestPatchNullClearsField(t *testing.T) {
	srv := newTestServer(t, "")
	created := mustCreate(t, srv, `{
		"title": "Z linkiem",
		"kind": "link",
		"external_url": "https://example.com",
		"happened_at": "2023-01-01T00:00:00Z"
	}`)

	res := do(t, srv, http.MethodPatch, "/api/items/"+created.ID,
		`{"external_url": null, "happened_at": null}`)
	if res.status != http.StatusOK {
		t.Fatalf("status = %d, ciało %s", res.status, res.body)
	}

	var updated model.Item
	res.decode(t, &updated)

	if updated.ExternalURL != nil {
		t.Errorf("external_url = %v, oczekiwano null po jawnym wyczyszczeniu", *updated.ExternalURL)
	}
	if updated.HappenedAt != nil {
		t.Errorf("happened_at = %v, oczekiwano null", updated.HappenedAt)
	}
}

func TestPatchTitleChangesSlug(t *testing.T) {
	srv := newTestServer(t, "")
	created := mustCreate(t, srv, `{"title": "Pierwsza wersja", "kind": "note"}`)

	res := do(t, srv, http.MethodPatch, "/api/items/"+created.ID, `{"title": "Druga wersja"}`)
	var updated model.Item
	res.decode(t, &updated)

	if updated.Slug != "druga-wersja" {
		t.Errorf("slug = %q, oczekiwano druga-wersja", updated.Slug)
	}
}

func TestPatchReplacesTags(t *testing.T) {
	srv := newTestServer(t, "")
	created := mustCreate(t, srv, `{"title": "Tagowany", "kind": "note", "tags": ["a", "b"]}`)

	res := do(t, srv, http.MethodPatch, "/api/items/"+created.ID, `{"tags": ["c"]}`)
	var updated model.Item
	res.decode(t, &updated)

	if len(updated.Tags) != 1 || updated.Tags[0].Slug != "c" {
		t.Fatalf("tagi = %+v, oczekiwano wyłącznie 'c'", updated.Tags)
	}

	// Puste [] musi usunąć wszystkie tagi.
	res = do(t, srv, http.MethodPatch, "/api/items/"+created.ID, `{"tags": []}`)
	res.decode(t, &updated)
	if len(updated.Tags) != 0 {
		t.Errorf("tagi = %+v, oczekiwano pustej listy", updated.Tags)
	}
}

func TestPatchNotFound(t *testing.T) {
	srv := newTestServer(t, "")

	res := do(t, srv, http.MethodPatch, "/api/items/brak-takiego", `{"status": "done"}`)
	if res.status != http.StatusNotFound {
		t.Fatalf("status = %d, oczekiwano 404", res.status)
	}
}

func TestPatchValidation(t *testing.T) {
	srv := newTestServer(t, "")
	created := mustCreate(t, srv, `{"title": "X", "kind": "note"}`)

	tests := []struct {
		name string
		body string
	}{
		{"pusty tytuł", `{"title": "  "}`},
		{"tytuł null", `{"title": null}`},
		{"zły kind", `{"kind": "nieistnieje"}`},
		{"tags null", `{"tags": null}`},
		{"metadata null", `{"metadata": null}`},
		{"favorite null", `{"favorite": null}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := do(t, srv, http.MethodPatch, "/api/items/"+created.ID, tt.body)
			if res.status != http.StatusBadRequest {
				t.Fatalf("status = %d, oczekiwano 400; ciało %s", res.status, res.body)
			}
		})
	}
}

// --- usuwanie ---------------------------------------------------------------

func TestDeleteItem(t *testing.T) {
	srv := newTestServer(t, "")
	created := mustCreate(t, srv, `{"title": "Do usunięcia", "kind": "other", "tags": ["x"]}`)

	res := do(t, srv, http.MethodDelete, "/api/items/"+created.ID, "")
	if res.status != http.StatusNoContent {
		t.Fatalf("status = %d, oczekiwano 204; ciało %s", res.status, res.body)
	}

	if res := do(t, srv, http.MethodGet, "/api/items/"+created.ID, ""); res.status != http.StatusNotFound {
		t.Errorf("po usunięciu GET zwrócił %d, oczekiwano 404", res.status)
	}
	if res := do(t, srv, http.MethodDelete, "/api/items/"+created.ID, ""); res.status != http.StatusNotFound {
		t.Errorf("powtórny DELETE zwrócił %d, oczekiwano 404", res.status)
	}

	// Sam tag ma przetrwać usunięcie itemu — znika tylko powiązanie.
	res = do(t, srv, http.MethodGet, "/api/tags", "")
	var tags struct {
		Tags []model.Tag `json:"tags"`
	}
	res.decode(t, &tags)
	if len(tags.Tags) != 1 || tags.Tags[0].Count != 0 {
		t.Errorf("tagi po usunięciu itemu = %+v, oczekiwano jednego z count=0", tags.Tags)
	}
}

// --- listowanie i filtry ----------------------------------------------------

// seedItems tworzy stały zestaw danych do testów listowania.
func seedItems(t *testing.T, srv *httptest.Server) {
	t.Helper()

	bodies := []string{
		`{"title": "Alfa", "kind": "project", "status": "done", "favorite": true,
		  "happened_at": "2024-03-01T00:00:00Z", "tags": ["go", "cli"]}`,
		`{"title": "Beta", "kind": "project", "status": "wip",
		  "happened_at": "2024-01-01T00:00:00Z", "tags": ["go"]}`,
		`{"title": "Gamma", "kind": "drawing", "status": "done", "favorite": true,
		  "happened_at": "2024-02-01T00:00:00Z", "tags": ["szkic"]}`,
		`{"title": "Delta", "kind": "print3d", "status": "archived",
		  "happened_at": "2023-12-01T00:00:00Z"}`,
	}
	for _, b := range bodies {
		mustCreate(t, srv, b)
	}
}

func listTitles(t *testing.T, srv *httptest.Server, query string) []string {
	t.Helper()

	res := do(t, srv, http.MethodGet, "/api/items"+query, "")
	if res.status != http.StatusOK {
		t.Fatalf("GET /api/items%s: status %d, ciało %s", query, res.status, res.body)
	}

	var result store.ListResult
	res.decode(t, &result)

	titles := make([]string, 0, len(result.Items))
	for _, it := range result.Items {
		titles = append(titles, it.Title)
	}
	return titles
}

func TestListFilters(t *testing.T) {
	srv := newTestServer(t, "")
	seedItems(t, srv)

	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{
			name:  "domyślnie wszystko, malejąco po happened_at",
			query: "",
			want:  []string{"Alfa", "Gamma", "Beta", "Delta"},
		},
		{
			name:  "sortowanie rosnąco",
			query: "?order=asc",
			want:  []string{"Delta", "Beta", "Gamma", "Alfa"},
		},
		{
			name:  "sortowanie po tytule",
			query: "?sort=title&order=asc",
			want:  []string{"Alfa", "Beta", "Delta", "Gamma"},
		},
		{
			name:  "filtr po kind",
			query: "?kind=project",
			want:  []string{"Alfa", "Beta"},
		},
		{
			name:  "kilka kind po przecinku",
			query: "?kind=drawing,print3d",
			want:  []string{"Gamma", "Delta"},
		},
		{
			name:  "kilka kind jako powtórzony parametr",
			query: "?kind=drawing&kind=print3d",
			want:  []string{"Gamma", "Delta"},
		},
		{
			name:  "filtr po statusie",
			query: "?status=done",
			want:  []string{"Alfa", "Gamma"},
		},
		{
			name:  "tylko ulubione",
			query: "?favorite=true",
			want:  []string{"Alfa", "Gamma"},
		},
		{
			name:  "tylko nieulubione",
			query: "?favorite=false",
			want:  []string{"Beta", "Delta"},
		},
		{
			name:  "filtr po tagu",
			query: "?tag=go",
			want:  []string{"Alfa", "Beta"},
		},
		{
			name:  "dwa tagi łączy AND, nie OR",
			query: "?tag=go,cli",
			want:  []string{"Alfa"},
		},
		{
			name:  "kind i status razem",
			query: "?kind=project&status=wip",
			want:  []string{"Beta"},
		},
		{
			name:  "nic nie pasuje",
			query: "?tag=nieistniejacy",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := listTitles(t, srv, tt.query)
			if len(got) != len(tt.want) {
				t.Fatalf("dostałem %v, oczekiwano %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("dostałem %v, oczekiwano %v", got, tt.want)
				}
			}
		})
	}
}

func TestListPagination(t *testing.T) {
	srv := newTestServer(t, "")
	seedItems(t, srv)

	res := do(t, srv, http.MethodGet, "/api/items?limit=2&offset=1", "")
	var result store.ListResult
	res.decode(t, &result)

	if result.Total != 4 {
		t.Errorf("total = %d, oczekiwano 4 (liczy wszystkie, nie tylko stronę)", result.Total)
	}
	if len(result.Items) != 2 {
		t.Fatalf("na stronie %d itemów, oczekiwano 2", len(result.Items))
	}
	if result.Items[0].Title != "Gamma" || result.Items[1].Title != "Beta" {
		t.Errorf("strona = %q, %q; oczekiwano Gamma, Beta", result.Items[0].Title, result.Items[1].Title)
	}
	if result.Limit != 2 || result.Offset != 1 {
		t.Errorf("limit/offset w odpowiedzi = %d/%d", result.Limit, result.Offset)
	}
}

func TestListRejectsUnknownFilterValues(t *testing.T) {
	srv := newTestServer(t, "")

	for _, query := range []string{"?kind=kosmos", "?status=prawie", "?favorite=może"} {
		res := do(t, srv, http.MethodGet, "/api/items"+query, "")
		if res.status != http.StatusBadRequest {
			t.Errorf("GET /api/items%s: status %d, oczekiwano 400", query, res.status)
		}
	}
}

func TestListIgnoresBadSortAndOversizedLimit(t *testing.T) {
	srv := newTestServer(t, "")
	seedItems(t, srv)

	// Nieznane sortowanie i absurdalny limit mają wrócić do wartości domyślnych,
	// a nie wysypać zapytania — to jedyna ochrona przed wstrzyknięciem
	// nazwy kolumny do ORDER BY.
	res := do(t, srv, http.MethodGet, "/api/items?sort=DROP+TABLE+items&limit=99999", "")
	if res.status != http.StatusOK {
		t.Fatalf("status = %d, ciało %s", res.status, res.body)
	}

	var result store.ListResult
	res.decode(t, &result)
	if result.Limit != store.MaxLimit {
		t.Errorf("limit = %d, oczekiwano przycięcia do %d", result.Limit, store.MaxLimit)
	}
	if len(result.Items) != 4 {
		t.Errorf("liczba itemów = %d, oczekiwano 4", len(result.Items))
	}
}

// --- tagi -------------------------------------------------------------------

func TestTagsAreReusedCaseInsensitively(t *testing.T) {
	srv := newTestServer(t, "")

	mustCreate(t, srv, `{"title": "Jeden", "kind": "note", "tags": ["Druk 3D"]}`)
	mustCreate(t, srv, `{"title": "Dwa", "kind": "note", "tags": ["druk 3d"]}`)

	res := do(t, srv, http.MethodGet, "/api/tags", "")
	var body struct {
		Tags []model.Tag `json:"tags"`
	}
	res.decode(t, &body)

	if len(body.Tags) != 1 {
		t.Fatalf("tagi = %+v, oczekiwano jednego wspólnego", body.Tags)
	}
	if body.Tags[0].Count != 2 {
		t.Errorf("count = %d, oczekiwano 2", body.Tags[0].Count)
	}
	// Zachowujemy pisownię z pierwszego użycia.
	if body.Tags[0].Name != "Druk 3D" {
		t.Errorf("name = %q, oczekiwano 'Druk 3D'", body.Tags[0].Name)
	}
}

func TestDuplicateTagsInOneItemCollapse(t *testing.T) {
	srv := newTestServer(t, "")

	item := mustCreate(t, srv, `{"title": "X", "kind": "note", "tags": ["Go", "go", "  GO  "]}`)
	if len(item.Tags) != 1 {
		t.Errorf("tagi = %+v, oczekiwano jednego po deduplikacji", item.Tags)
	}
}

// --- token ------------------------------------------------------------------

func TestTokenGuardsWritesOnly(t *testing.T) {
	const token = "sekret"
	srv := newTestServer(t, token)

	// Odczyt jest otwarty nawet przy ustawionym tokenie.
	if res := do(t, srv, http.MethodGet, "/api/items", ""); res.status != http.StatusOK {
		t.Errorf("GET bez tokenu: status %d, oczekiwano 200", res.status)
	}

	// Zapis bez tokenu jest odrzucany.
	res := do(t, srv, http.MethodPost, "/api/items", `{"title": "X", "kind": "note"}`)
	if res.status != http.StatusUnauthorized {
		t.Fatalf("POST bez tokenu: status %d, oczekiwano 401", res.status)
	}
	if code := res.errorCode(t); code != "unauthorized" {
		t.Errorf("kod = %q, oczekiwano unauthorized", code)
	}

	// Zły token też.
	res = do(t, srv, http.MethodPost, "/api/items", `{"title": "X", "kind": "note"}`,
		[2]string{"X-App-Token", "zły"})
	if res.status != http.StatusUnauthorized {
		t.Errorf("POST ze złym tokenem: status %d, oczekiwano 401", res.status)
	}

	// Z poprawnym tokenem przechodzi.
	res = do(t, srv, http.MethodPost, "/api/items", `{"title": "X", "kind": "note"}`,
		[2]string{"X-App-Token", token})
	if res.status != http.StatusCreated {
		t.Errorf("POST z tokenem: status %d, oczekiwano 201; ciało %s", res.status, res.body)
	}
}

// --- kształt błędów ---------------------------------------------------------

func TestUnknownAPIPathReturnsJSONError(t *testing.T) {
	srv := newTestServer(t, "")

	res := do(t, srv, http.MethodGet, "/api/nie-ma-tego", "")
	if res.status != http.StatusNotFound {
		t.Fatalf("status = %d, oczekiwano 404", res.status)
	}
	// Kluczowe: pod /api nigdy nie może wyjść HTML powłoki frontendu.
	if code := res.errorCode(t); code != "not_found" {
		t.Errorf("kod = %q, oczekiwano not_found; ciało %s", code, res.body)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	srv := newTestServer(t, "")

	res := do(t, srv, http.MethodPut, "/api/items", `{}`)
	if res.status != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, oczekiwano 405; ciało %s", res.status, res.body)
	}
}

package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"designfolio/internal/store"
)

// seedForSearch tworzy zestaw, w którym każdy przypadek testowy
// da się jednoznacznie rozstrzygnąć.
func seedForSearch(t *testing.T, srv *httptest.Server) {
	t.Helper()

	bodies := []string{
		`{"title": "Uchwyt na słuchawki", "kind": "print3d",
		  "description": "Klips wsuwany pod blat biurka.", "tags": ["druk 3d", "biurko"]}`,
		`{"title": "Wieża ciśnień", "kind": "drawing",
		  "description": "Szkic z natury, ołówek 2B.", "tags": ["szkic", "architektura"]}`,
		`{"title": "Notatki o SQLite", "kind": "note",
		  "description": "WAL zdejmuje problem blokad przy odczycie.", "tags": ["sqlite"]}`,
		`{"title": "Organizer na kable", "kind": "print3d",
		  "description": "Zatrzask zamiast wkrętu.", "tags": ["druk 3d"]}`,
	}
	for _, b := range bodies {
		mustCreate(t, srv, b)
	}
}

func searchTitles(t *testing.T, srv *httptest.Server, query string) []string {
	t.Helper()

	res := do(t, srv, http.MethodGet, "/api/items?q="+url.QueryEscape(query), "")
	if res.status != http.StatusOK {
		t.Fatalf("szukanie %q: status %d, ciało %s", query, res.status, res.body)
	}

	var result store.ListResult
	res.decode(t, &result)

	titles := make([]string, 0, len(result.Items))
	for _, it := range result.Items {
		titles = append(titles, it.Title)
	}
	return titles
}

func TestSearch(t *testing.T) {
	srv := newTestServer(t, "")
	seedForSearch(t, srv)

	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"po tytule", "uchwyt", []string{"Uchwyt na słuchawki"}},
		{"po fragmencie opisu", "zatrzask", []string{"Organizer na kable"}},
		{"po tagu", "architektura", []string{"Wieża ciśnień"}},
		{"przedrostek wystarcza", "orga", []string{"Organizer na kable"}},
		{"bez polskich znaków", "wieza", []string{"Wieża ciśnień"}},
		{"bez znaków diakrytycznych w opisie", "olowek", []string{"Wieża ciśnień"}},
		{"wielkość liter nie ma znaczenia", "SQLITE", []string{"Notatki o SQLite"}},
		{"nic nie pasuje", "helikopter", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := searchTitles(t, srv, tt.query)
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

func TestSearchRequiresAllWords(t *testing.T) {
	srv := newTestServer(t, "")
	seedForSearch(t, srv)

	// Oba słowa muszą wystąpić — "druk" jest tagiem obu druków 3D,
	// ale "kable" tylko jednego.
	got := searchTitles(t, srv, "druk kable")
	if len(got) != 1 || got[0] != "Organizer na kable" {
		t.Errorf("dostałem %v, oczekiwano wyłącznie organizera", got)
	}
}

func TestSearchSurvivesFTSOperators(t *testing.T) {
	srv := newTestServer(t, "")
	seedForSearch(t, srv)

	// Znaki, które w składni FTS5 są operatorami. Wstawione surowo
	// wywróciłyby zapytanie błędem składni — mają zostać potraktowane
	// jak zwykły tekst.
	for _, query := range []string{
		`uchwyt"`, `"`, `AND`, `NOT NOT`, `*`, `^ ( ) -`, `uchwyt AND OR`, `sqlite*`,
	} {
		res := do(t, srv, http.MethodGet, "/api/items?q="+url.QueryEscape(query), "")
		if res.status != http.StatusOK {
			t.Errorf("zapytanie %q: status %d, ciało %s", query, res.status, res.body)
		}
	}
}

func TestSearchReflectsEdits(t *testing.T) {
	srv := newTestServer(t, "")
	item := mustCreate(t, srv, `{"title": "Pierwotny tytuł", "kind": "note"}`)

	if got := searchTitles(t, srv, "pierwotny"); len(got) != 1 {
		t.Fatalf("nowy item nie trafił do indeksu: %v", got)
	}

	// Zmiana tytułu ma zaktualizować indeks — stary tytuł przestaje pasować.
	do(t, srv, http.MethodPatch, "/api/items/"+item.ID, `{"title": "Zmieniony tytuł"}`)

	if got := searchTitles(t, srv, "pierwotny"); len(got) != 0 {
		t.Errorf("stary tytuł nadal w indeksie: %v", got)
	}
	if got := searchTitles(t, srv, "zmieniony"); len(got) != 1 {
		t.Errorf("nowy tytuł nie trafił do indeksu: %v", got)
	}

	// Dopisanie tagu też musi być widoczne w wyszukiwaniu.
	do(t, srv, http.MethodPatch, "/api/items/"+item.ID, `{"tags": ["kaligrafia"]}`)
	if got := searchTitles(t, srv, "kaligrafia"); len(got) != 1 {
		t.Errorf("nowy tag nie trafił do indeksu: %v", got)
	}

	// Usunięcie itemu musi go usunąć również z indeksu.
	do(t, srv, http.MethodDelete, "/api/items/"+item.ID, "")
	if got := searchTitles(t, srv, "zmieniony"); len(got) != 0 {
		t.Errorf("usunięty item nadal w wynikach: %v", got)
	}
}

func TestSearchCombinesWithFilters(t *testing.T) {
	srv := newTestServer(t, "")
	seedForSearch(t, srv)

	res := do(t, srv, http.MethodGet, "/api/items?q=druk&kind=print3d&status=wip", "")
	var result store.ListResult
	res.decode(t, &result)

	if result.Total != 2 {
		t.Errorf("total = %d, oczekiwano 2 druków 3D pasujących do zapytania", result.Total)
	}

	// Zawężenie rodzajem, którego nie ma wśród trafień, ma dać pustkę.
	res = do(t, srv, http.MethodGet, "/api/items?q=druk&kind=photo", "")
	res.decode(t, &result)
	if result.Total != 0 {
		t.Errorf("total = %d, oczekiwano 0", result.Total)
	}
}

func TestSearchRanksTitleAboveDescription(t *testing.T) {
	srv := newTestServer(t, "")

	mustCreate(t, srv, `{"title": "Zupełnie co innego", "kind": "note",
	  "description": "Gdzieś w środku pada słowo lampa i tyle."}`)
	mustCreate(t, srv, `{"title": "Lampa nocna", "kind": "print3d",
	  "description": "Abażur z białego PETG."}`)

	got := searchTitles(t, srv, "lampa")
	if len(got) != 2 {
		t.Fatalf("dostałem %v, oczekiwano dwóch trafień", got)
	}
	if got[0] != "Lampa nocna" {
		t.Errorf("pierwszy wynik to %q — trafienie w tytule ma być wyżej niż w opisie", got[0])
	}
}

package httpapi_test

import (
	"net/http"
	"strings"
	"testing"

	"designfolio/internal/model"
	"designfolio/internal/store"
)

func TestExportContainsEverything(t *testing.T) {
	srv := newTestServer(t, "")

	item := mustCreate(t, srv, `{
		"title": "Z wszystkim",
		"kind": "print3d",
		"description": "Opis do eksportu",
		"favorite": true,
		"happened_at": "2025-04-01T00:00:00Z",
		"metadata": {"drukarka": "Prusa MK4", "czas_h": 3.5},
		"tags": ["druk 3d", "test"]
	}`)
	uploadAssets(t, srv, item.ID,
		uploadFile{field: "files", filename: "a.png", content: pngBytes(t, 100, 100)})
	mustCreate(t, srv, `{"title": "Drugi", "kind": "note"}`)

	res := do(t, srv, http.MethodGet, "/api/export", "")
	if res.status != http.StatusOK {
		t.Fatalf("status = %d, ciało %s", res.status, res.body)
	}

	var export store.Export
	res.decode(t, &export)

	if export.Version != store.ExportFormatVersion {
		t.Errorf("version = %d, oczekiwano %d", export.Version, store.ExportFormatVersion)
	}
	if export.Counts.Items != 2 || export.Counts.Assets != 1 || export.Counts.Tags != 2 {
		t.Errorf("liczniki = %+v, oczekiwano 2 itemów, 1 pliku, 2 tagów", export.Counts)
	}
	if export.ExportedAt.IsZero() {
		t.Error("brak znacznika czasu eksportu")
	}

	var found *model.Item
	for i := range export.Items {
		if export.Items[i].ID == item.ID {
			found = &export.Items[i]
		}
	}
	if found == nil {
		t.Fatal("item nie znalazł się w eksporcie")
	}

	// Eksport ma być kompletny: metadane, tagi i pliki razem z itemem.
	if len(found.Tags) != 2 {
		t.Errorf("tagi w eksporcie = %+v", found.Tags)
	}
	if len(found.Assets) != 1 || found.Assets[0].Filename != "a.png" {
		t.Errorf("pliki w eksporcie = %+v", found.Assets)
	}
	if !strings.Contains(string(found.Metadata), "Prusa MK4") {
		t.Errorf("metadane w eksporcie = %s", found.Metadata)
	}
	if found.HappenedAt == nil {
		t.Error("brak happened_at w eksporcie")
	}
}

func TestExportSetsDownloadHeader(t *testing.T) {
	srv := newTestServer(t, "")

	httpRes, err := srv.Client().Get(srv.URL + "/api/export")
	if err != nil {
		t.Fatalf("pobranie eksportu: %v", err)
	}
	defer httpRes.Body.Close()

	disposition := httpRes.Header.Get("Content-Disposition")
	if !strings.HasPrefix(disposition, "attachment;") {
		t.Errorf("Content-Disposition = %q, oczekiwano załącznika", disposition)
	}
	if !strings.Contains(disposition, "designfolio-") {
		t.Errorf("nazwa pliku w %q nie zawiera nazwy aplikacji", disposition)
	}
}

func TestExportOfEmptyCatalog(t *testing.T) {
	srv := newTestServer(t, "")

	res := do(t, srv, http.MethodGet, "/api/export", "")
	var export store.Export
	res.decode(t, &export)

	// Pusty katalog ma dać puste tablice, a nie null — inaczej narzędzia
	// po drugiej stronie muszą obsługiwać dwa przypadki.
	if export.Items == nil || export.Tags == nil {
		t.Errorf("puste kolekcje wyszły jako null: items=%v tags=%v", export.Items, export.Tags)
	}
	if export.Counts.Items != 0 {
		t.Errorf("liczba itemów = %d, oczekiwano 0", export.Counts.Items)
	}
}

func TestTimelineReturnsEverythingSorted(t *testing.T) {
	srv := newTestServer(t, "")

	mustCreate(t, srv, `{"title": "Starszy", "kind": "note", "happened_at": "2023-01-01T00:00:00Z"}`)
	mustCreate(t, srv, `{"title": "Najnowszy", "kind": "note", "happened_at": "2026-01-01T00:00:00Z"}`)
	mustCreate(t, srv, `{"title": "Środkowy", "kind": "note", "happened_at": "2024-06-01T00:00:00Z"}`)
	// Bez daty — liczy się data dodania, więc trafia na sam początek.
	mustCreate(t, srv, `{"title": "Bez daty", "kind": "note"}`)

	res := do(t, srv, http.MethodGet, "/api/timeline", "")
	if res.status != http.StatusOK {
		t.Fatalf("status = %d, ciało %s", res.status, res.body)
	}

	var body struct {
		Items []model.Item `json:"items"`
		Total int          `json:"total"`
	}
	res.decode(t, &body)

	if body.Total != 4 {
		t.Fatalf("total = %d, oczekiwano 4", body.Total)
	}

	want := []string{"Bez daty", "Najnowszy", "Środkowy", "Starszy"}
	for i, title := range want {
		if body.Items[i].Title != title {
			t.Errorf("pozycja %d = %q, oczekiwano %q", i, body.Items[i].Title, title)
		}
	}
}

func TestTimelineIgnoresPagination(t *testing.T) {
	srv := newTestServer(t, "")

	// Oś czasu ma pokazać wszystko naraz, także powyżej domyślnego limitu listy.
	for i := 0; i < store.DefaultLimit+5; i++ {
		mustCreate(t, srv, `{"title": "Rzecz numer `+string(rune('A'+i%26))+`", "kind": "note"}`)
	}

	res := do(t, srv, http.MethodGet, "/api/timeline", "")
	var body struct {
		Total int `json:"total"`
	}
	res.decode(t, &body)

	if body.Total != store.DefaultLimit+5 {
		t.Errorf("total = %d, oczekiwano %d — oś czasu nie stronicuje",
			body.Total, store.DefaultLimit+5)
	}
}

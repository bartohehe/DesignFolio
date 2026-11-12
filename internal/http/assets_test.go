package httpapi_test

import (
	"bytes"
	"image"
	"image/color"
	_ "image/jpeg" // rejestracja dekodera — miniatury bywają JPEG-ami
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"designfolio/internal/model"
)

// --- pomocnicze --------------------------------------------------------------

// pngBytes generuje prawdziwy obrazek PNG o podanych wymiarach.
// Prawdziwy, a nie atrapa — cały sens tych testów polega na tym,
// że przechodzi przez dekoder i skalowanie.
func pngBytes(t *testing.T, width, height int) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.NRGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: uint8((x + y) % 256),
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("kodowanie testowego PNG: %v", err)
	}
	return buf.Bytes()
}

type uploadFile struct {
	field    string
	filename string
	content  []byte
}

// uploadAssets buduje żądanie multipart i wysyła je pod /api/items/{id}/assets.
func uploadAssets(t *testing.T, srv *httptest.Server, itemID string, files ...uploadFile) response {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for _, f := range files {
		part, err := writer.CreateFormFile(f.field, f.filename)
		if err != nil {
			t.Fatalf("budowa formularza: %v", err)
		}
		if _, err := part.Write(f.content); err != nil {
			t.Fatalf("zapis części formularza: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("zamknięcie formularza: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/items/"+itemID+"/assets", &body)
	if err != nil {
		t.Fatalf("budowa żądania: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("wysłanie uploadu: %v", err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("odczyt odpowiedzi: %v", err)
	}
	return response{status: res.StatusCode, body: raw}
}

type uploadResult struct {
	Uploaded []struct {
		Asset     model.Asset `json:"asset"`
		Duplicate bool        `json:"duplicate"`
	} `json:"uploaded"`
	Skipped []struct {
		Filename string `json:"filename"`
		Reason   string `json:"reason"`
	} `json:"skipped"`
}

// --- testy -------------------------------------------------------------------

func TestUploadImageGeneratesDerivatives(t *testing.T) {
	srv := newTestServer(t, "")
	item := mustCreate(t, srv, `{"title": "Z obrazkiem", "kind": "photo"}`)

	res := uploadAssets(t, srv, item.ID, uploadFile{
		field: "files", filename: "zdjęcie testowe.png", content: pngBytes(t, 900, 600),
	})
	if res.status != http.StatusCreated {
		t.Fatalf("status = %d, oczekiwano 201; ciało %s", res.status, res.body)
	}

	var result uploadResult
	res.decode(t, &result)

	if len(result.Uploaded) != 1 {
		t.Fatalf("wgrano %d plików, oczekiwano 1; pominięte: %+v", len(result.Uploaded), result.Skipped)
	}

	asset := result.Uploaded[0].Asset
	if asset.Kind != model.AssetImage {
		t.Errorf("kind = %q, oczekiwano image", asset.Kind)
	}
	if !asset.HasThumb {
		t.Error("has_thumb = false, oczekiwano wygenerowanych miniatur")
	}
	if asset.Width == nil || *asset.Width != 900 || asset.Height == nil || *asset.Height != 600 {
		t.Errorf("wymiary = %v×%v, oczekiwano 900×600", asset.Width, asset.Height)
	}
	if asset.Checksum == "" {
		t.Error("brak sumy kontrolnej")
	}
	// Polskie znaki w nazwie mają przetrwać, ukośniki i ścieżki — nie.
	if asset.Filename != "zdjęcie-testowe.png" {
		t.Errorf("filename = %q, oczekiwano oczyszczonej nazwy z polskimi znakami", asset.Filename)
	}

	// Wszystkie trzy warianty muszą być serwowane.
	for _, variant := range []string{"original", "thumb", "preview"} {
		res := do(t, srv, http.MethodGet, "/api/files/"+asset.ID+"/"+variant, "")
		if res.status != http.StatusOK {
			t.Errorf("wariant %s: status %d, oczekiwano 200", variant, res.status)
		}
		if len(res.body) == 0 {
			t.Errorf("wariant %s: pusta odpowiedź", variant)
		}
	}

	// Sprawdzamy wymiary w pikselach, a nie rozmiar w bajtach: przeskalowany
	// obraz bywa CIĘŻSZY od oryginału (resampling wprowadza szum, którego PNG
	// nie skompresuje tak dobrze jak gładki gradient źródłowy).
	thumbBounds := decodeBounds(t, do(t, srv, http.MethodGet, "/api/files/"+asset.ID+"/thumb", "").body)
	if thumbBounds.Dx() != 400 || thumbBounds.Dy() != 266 {
		t.Errorf("miniatura %d×%d, oczekiwano 400×266 (proporcja 900×600 zachowana)",
			thumbBounds.Dx(), thumbBounds.Dy())
	}

	// Podgląd jest większy od miniatury, ale nie powiększa oryginału:
	// 900 px źródła mieści się w limicie 1600, więc zostaje bez zmian.
	previewBounds := decodeBounds(t, do(t, srv, http.MethodGet, "/api/files/"+asset.ID+"/preview", "").body)
	if previewBounds.Dx() != 900 {
		t.Errorf("podgląd %d px szerokości, oczekiwano 900 — nie powiększamy w górę", previewBounds.Dx())
	}
}

func decodeBounds(t *testing.T, raw []byte) image.Rectangle {
	t.Helper()
	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("dekodowanie obrazu: %v", err)
	}
	return decoded.Bounds()
}

func TestUploadSmallImageIsNotUpscaled(t *testing.T) {
	srv := newTestServer(t, "")
	item := mustCreate(t, srv, `{"title": "Mały", "kind": "photo"}`)

	res := uploadAssets(t, srv, item.ID, uploadFile{
		field: "files", filename: "maly.png", content: pngBytes(t, 120, 80),
	})
	var result uploadResult
	res.decode(t, &result)

	thumb := do(t, srv, http.MethodGet, "/api/files/"+result.Uploaded[0].Asset.ID+"/thumb", "")
	decoded, _, err := image.Decode(bytes.NewReader(thumb.body))
	if err != nil {
		t.Fatalf("dekodowanie miniatury: %v", err)
	}
	if decoded.Bounds().Dx() != 120 {
		t.Errorf("szerokość = %d, oczekiwano 120 — małych obrazów nie powiększamy",
			decoded.Bounds().Dx())
	}
}

func TestUploadNonImageKeepsFileWithoutThumbs(t *testing.T) {
	srv := newTestServer(t, "")
	item := mustCreate(t, srv, `{"title": "Model", "kind": "print3d"}`)

	stl := []byte("solid test\nfacet normal 0 0 1\nendsolid test\n")
	res := uploadAssets(t, srv, item.ID, uploadFile{field: "files", filename: "kubek.stl", content: stl})

	var result uploadResult
	res.decode(t, &result)

	asset := result.Uploaded[0].Asset
	if asset.Kind != model.AssetModel {
		t.Errorf("kind = %q, oczekiwano model", asset.Kind)
	}
	if asset.HasThumb {
		t.Error("has_thumb = true, model 3D nie ma miniatury")
	}
	if asset.Width != nil || asset.Height != nil {
		t.Errorf("wymiary = %v/%v, oczekiwano null dla nie-obrazu", asset.Width, asset.Height)
	}

	// Oryginał ma być do pobrania, miniatura ma nie istnieć.
	if res := do(t, srv, http.MethodGet, "/api/files/"+asset.ID+"/original", ""); res.status != http.StatusOK {
		t.Errorf("original: status %d, oczekiwano 200", res.status)
	}
	if res := do(t, srv, http.MethodGet, "/api/files/"+asset.ID+"/thumb", ""); res.status != http.StatusNotFound {
		t.Errorf("thumb: status %d, oczekiwano 404", res.status)
	}
}

func TestUploadDeduplicatesByChecksum(t *testing.T) {
	srv := newTestServer(t, "")
	item := mustCreate(t, srv, `{"title": "Dwa razy to samo", "kind": "photo"}`)
	content := pngBytes(t, 300, 200)

	first := uploadAssets(t, srv, item.ID, uploadFile{field: "files", filename: "a.png", content: content})
	var firstResult uploadResult
	first.decode(t, &firstResult)

	// Ten sam plik pod inną nazwą — treść decyduje, nie nazwa.
	second := uploadAssets(t, srv, item.ID, uploadFile{field: "files", filename: "kopia.png", content: content})
	var secondResult uploadResult
	second.decode(t, &secondResult)

	if !secondResult.Uploaded[0].Duplicate {
		t.Error("duplicate = false, oczekiwano rozpoznania duplikatu")
	}
	if secondResult.Uploaded[0].Asset.ID != firstResult.Uploaded[0].Asset.ID {
		t.Error("duplikat wskazuje na inny asset niż pierwotny")
	}

	detail := do(t, srv, http.MethodGet, "/api/items/"+item.ID, "")
	var withAssets model.Item
	detail.decode(t, &withAssets)
	if len(withAssets.Assets) != 1 {
		t.Errorf("item ma %d plików, oczekiwano 1", len(withAssets.Assets))
	}
}

func TestUploadMultipleFilesAtOnce(t *testing.T) {
	srv := newTestServer(t, "")
	item := mustCreate(t, srv, `{"title": "Seria", "kind": "photo"}`)

	res := uploadAssets(t, srv, item.ID,
		uploadFile{field: "files", filename: "1.png", content: pngBytes(t, 200, 300)},
		uploadFile{field: "files", filename: "2.png", content: pngBytes(t, 300, 200)},
		uploadFile{field: "files", filename: "3.png", content: pngBytes(t, 250, 250)},
	)
	var result uploadResult
	res.decode(t, &result)

	if len(result.Uploaded) != 3 {
		t.Fatalf("wgrano %d, oczekiwano 3; pominięte %+v", len(result.Uploaded), result.Skipped)
	}

	// Kolejność ma odpowiadać kolejności wysyłki.
	for i, up := range result.Uploaded {
		if up.Asset.SortOrder != i {
			t.Errorf("plik %d ma sort_order %d", i, up.Asset.SortOrder)
		}
	}
}

func TestArrangeAssetsSetsOrderAndCover(t *testing.T) {
	srv := newTestServer(t, "")
	item := mustCreate(t, srv, `{"title": "Do poukładania", "kind": "photo"}`)

	res := uploadAssets(t, srv, item.ID,
		uploadFile{field: "files", filename: "a.png", content: pngBytes(t, 200, 200)},
		uploadFile{field: "files", filename: "b.png", content: pngBytes(t, 210, 200)},
		uploadFile{field: "files", filename: "c.png", content: pngBytes(t, 220, 200)},
	)
	var uploaded uploadResult
	res.decode(t, &uploaded)

	ids := []string{
		uploaded.Uploaded[0].Asset.ID,
		uploaded.Uploaded[1].Asset.ID,
		uploaded.Uploaded[2].Asset.ID,
	}

	// Odwracamy kolejność i robimy okładkę z ostatniego.
	body := `{"order": ["` + ids[2] + `", "` + ids[1] + `", "` + ids[0] + `"], "cover": "` + ids[1] + `"}`
	arrange := do(t, srv, http.MethodPatch, "/api/items/"+item.ID+"/assets", body)
	if arrange.status != http.StatusOK {
		t.Fatalf("status = %d, ciało %s", arrange.status, arrange.body)
	}

	detail := do(t, srv, http.MethodGet, "/api/items/"+item.ID, "")
	var withAssets model.Item
	detail.decode(t, &withAssets)

	if len(withAssets.Assets) != 3 {
		t.Fatalf("liczba plików = %d", len(withAssets.Assets))
	}
	if withAssets.Assets[0].ID != ids[2] || withAssets.Assets[2].ID != ids[0] {
		t.Error("kolejność plików nie została odwrócona")
	}
	if withAssets.Cover == nil || withAssets.Cover.ID != ids[1] {
		t.Errorf("okładka = %v, oczekiwano %s", withAssets.Cover, ids[1])
	}

	// Tylko jeden plik może być okładką.
	covers := 0
	for _, a := range withAssets.Assets {
		if a.IsCover {
			covers++
		}
	}
	if covers != 1 {
		t.Errorf("liczba okładek = %d, oczekiwano dokładnie 1", covers)
	}
}

func TestArrangeRejectsForeignAsset(t *testing.T) {
	srv := newTestServer(t, "")
	first := mustCreate(t, srv, `{"title": "Pierwszy", "kind": "photo"}`)
	second := mustCreate(t, srv, `{"title": "Drugi", "kind": "photo"}`)

	res := uploadAssets(t, srv, second.ID,
		uploadFile{field: "files", filename: "obcy.png", content: pngBytes(t, 100, 100)})
	var uploaded uploadResult
	res.decode(t, &uploaded)

	// Próba przypisania cudzego pliku jako okładki musi się odbić.
	body := `{"order": [], "cover": "` + uploaded.Uploaded[0].Asset.ID + `"}`
	arrange := do(t, srv, http.MethodPatch, "/api/items/"+first.ID+"/assets", body)
	if arrange.status != http.StatusNotFound {
		t.Errorf("status = %d, oczekiwano 404 przy pliku z innego itemu", arrange.status)
	}
}

func TestCoverFallsBackToFirstImage(t *testing.T) {
	srv := newTestServer(t, "")
	item := mustCreate(t, srv, `{"title": "Bez wskazanej okładki", "kind": "photo"}`)

	uploadAssets(t, srv, item.ID,
		uploadFile{field: "files", filename: "pierwszy.png", content: pngBytes(t, 200, 200)},
		uploadFile{field: "files", filename: "drugi.png", content: pngBytes(t, 300, 200)},
	)

	// Na liście item ma mieć okładkę mimo braku jawnego wyboru.
	list := do(t, srv, http.MethodGet, "/api/items", "")
	var result struct {
		Items []model.Item `json:"items"`
	}
	list.decode(t, &result)

	var found *model.Item
	for i := range result.Items {
		if result.Items[i].ID == item.ID {
			found = &result.Items[i]
		}
	}
	if found == nil {
		t.Fatal("item nie znalazł się na liście")
	}
	if found.Cover == nil {
		t.Fatal("cover = null, oczekiwano zastępczej okładki z pierwszego zdjęcia")
	}
	if found.Cover.Filename != "pierwszy.png" {
		t.Errorf("okładka = %q, oczekiwano pierwszego zdjęcia", found.Cover.Filename)
	}
}

func TestDeleteAssetRemovesFile(t *testing.T) {
	srv := newTestServer(t, "")
	item := mustCreate(t, srv, `{"title": "Do skasowania", "kind": "photo"}`)

	res := uploadAssets(t, srv, item.ID,
		uploadFile{field: "files", filename: "x.png", content: pngBytes(t, 200, 200)})
	var uploaded uploadResult
	res.decode(t, &uploaded)
	assetID := uploaded.Uploaded[0].Asset.ID

	del := do(t, srv, http.MethodDelete, "/api/assets/"+assetID, "")
	if del.status != http.StatusNoContent {
		t.Fatalf("status = %d, oczekiwano 204", del.status)
	}

	if res := do(t, srv, http.MethodGet, "/api/files/"+assetID+"/original", ""); res.status != http.StatusNotFound {
		t.Errorf("plik nadal serwowany: status %d", res.status)
	}
	if res := do(t, srv, http.MethodDelete, "/api/assets/"+assetID, ""); res.status != http.StatusNotFound {
		t.Errorf("powtórne usunięcie: status %d, oczekiwano 404", res.status)
	}
}

func TestDeleteItemRemovesItsFiles(t *testing.T) {
	srv := newTestServer(t, "")
	item := mustCreate(t, srv, `{"title": "Z plikami", "kind": "photo"}`)

	res := uploadAssets(t, srv, item.ID,
		uploadFile{field: "files", filename: "a.png", content: pngBytes(t, 200, 200)},
		uploadFile{field: "files", filename: "b.png", content: pngBytes(t, 200, 300)},
	)
	var uploaded uploadResult
	res.decode(t, &uploaded)

	if del := do(t, srv, http.MethodDelete, "/api/items/"+item.ID, ""); del.status != http.StatusNoContent {
		t.Fatalf("usunięcie itemu: status %d", del.status)
	}

	for _, up := range uploaded.Uploaded {
		res := do(t, srv, http.MethodGet, "/api/files/"+up.Asset.ID+"/original", "")
		if res.status != http.StatusNotFound {
			t.Errorf("plik %s przetrwał usunięcie itemu: status %d", up.Asset.Filename, res.status)
		}
	}
}

func TestServeFileSetsCachingHeaders(t *testing.T) {
	srv := newTestServer(t, "")
	item := mustCreate(t, srv, `{"title": "Nagłówki", "kind": "photo"}`)

	res := uploadAssets(t, srv, item.ID,
		uploadFile{field: "files", filename: "x.png", content: pngBytes(t, 200, 200)})
	var uploaded uploadResult
	res.decode(t, &uploaded)
	asset := uploaded.Uploaded[0].Asset

	httpRes, err := srv.Client().Get(srv.URL + "/api/files/" + asset.ID + "/thumb")
	if err != nil {
		t.Fatalf("pobranie miniatury: %v", err)
	}
	defer httpRes.Body.Close()

	etag := httpRes.Header.Get("ETag")
	if etag == "" {
		t.Error("brak nagłówka ETag")
	}

	// Powtórne żądanie z ETagiem ma dostać 304, a nie całą treść.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/files/"+asset.ID+"/thumb", nil)
	req.Header.Set("If-None-Match", etag)
	cached, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("żądanie warunkowe: %v", err)
	}
	defer cached.Body.Close()

	if cached.StatusCode != http.StatusNotModified {
		t.Errorf("status = %d, oczekiwano 304 przy pasującym ETagu", cached.StatusCode)
	}
}

func TestUploadRejectsUnknownVariantAndMissingItem(t *testing.T) {
	srv := newTestServer(t, "")

	res := uploadAssets(t, srv, "nie-ma-takiego",
		uploadFile{field: "files", filename: "x.png", content: pngBytes(t, 10, 10)})
	if res.status != http.StatusNotFound {
		t.Errorf("upload do nieistniejącego itemu: status %d, oczekiwano 404", res.status)
	}

	item := mustCreate(t, srv, `{"title": "T", "kind": "photo"}`)
	up := uploadAssets(t, srv, item.ID,
		uploadFile{field: "files", filename: "x.png", content: pngBytes(t, 10, 10)})
	var uploaded uploadResult
	up.decode(t, &uploaded)

	bad := do(t, srv, http.MethodGet, "/api/files/"+uploaded.Uploaded[0].Asset.ID+"/gigant", "")
	if bad.status != http.StatusBadRequest {
		t.Errorf("nieznany wariant: status %d, oczekiwano 400", bad.status)
	}
}

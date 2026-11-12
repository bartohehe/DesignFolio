package media

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"

	// Rejestracja dekoderów formatów, których standardowa biblioteka nie zna.
	// Importy "na efekt uboczny" — pakiety wpisują się do rejestru image.Decode.
	_ "golang.org/x/image/webp"
)

// Rozmiary miniatur: dłuższy bok w pikselach. Oryginał zostaje zawsze nietknięty.
const (
	ThumbSize   = 400  // kafelek w siatce
	PreviewSize = 1600 // galeria i lightbox
)

// jpegQuality — 82 to standardowy kompromis: różnicy nie widać,
// a plik jest kilkukrotnie mniejszy niż przy 100.
const jpegQuality = 82

// derivativeExt zwraca rozszerzenie miniatury dla danego formatu źródłowego.
//
// PNG dla formatów, które mogą mieć przezroczystość — spłaszczenie kanału alfa
// na sztywny kolor tła zepsułoby skany rysunków i grafiki z wyciętym tłem.
// JPEG dla reszty, bo jest wielokrotnie mniejszy.
func derivativeExt(sourceFormat string) string {
	switch sourceFormat {
	case "png", "gif", "webp":
		return ".png"
	default:
		return ".jpg"
	}
}

// imageInfo to wynik analizy obrazu.
type imageInfo struct {
	Width  int
	Height int
	// HasThumb mówi, czy udało się wygenerować miniatury.
	HasThumb bool
}

// makeDerivatives wczytuje obraz z pliku i zapisuje obok niego thumb i preview.
//
// Zwraca wymiary oryginału. Gdy plik okaże się nieczytelny jako obraz
// (uszkodzony, egzotyczny format), zwracamy błąd — wołający decyduje,
// czy to powód do odrzucenia całego uploadu. W tym projekcie nie jest:
// plik zostaje zapisany jako zwykły załącznik bez miniatur.
func makeDerivatives(originalPath, dir string) (imageInfo, error) {
	file, err := os.Open(originalPath)
	if err != nil {
		return imageInfo{}, fmt.Errorf("media: otwarcie obrazu: %w", err)
	}
	defer file.Close()

	// image.Decode zwraca też nazwę formatu — potrzebujemy jej, żeby wiedzieć,
	// czy źródło mogło mieć przezroczystość.
	src, format, err := image.Decode(file)
	if err != nil {
		return imageInfo{}, fmt.Errorf("media: dekodowanie obrazu: %w", err)
	}

	bounds := src.Bounds()
	info := imageInfo{Width: bounds.Dx(), Height: bounds.Dy()}

	ext := derivativeExt(format)

	for _, variant := range []struct {
		name string
		size int
	}{
		{"thumb", ThumbSize},
		{"preview", PreviewSize},
	} {
		resized := fitWithin(src, variant.size)
		out := filepath.Join(dir, variant.name+ext)

		// imaging.Save wybiera koder po rozszerzeniu pliku.
		if err := imaging.Save(resized, out, imaging.JPEGQuality(jpegQuality)); err != nil {
			return info, fmt.Errorf("media: zapis %s: %w", variant.name, err)
		}
	}

	info.HasThumb = true
	return info, nil
}

// fitWithin skaluje obraz tak, żeby zmieścił się w kwadracie o boku max,
// zachowując proporcje. Obrazu mniejszego niż max NIE powiększamy —
// rozmycie z interpolacji jest gorsze niż mała miniatura.
func fitWithin(src image.Image, max int) image.Image {
	bounds := src.Bounds()
	if bounds.Dx() <= max && bounds.Dy() <= max {
		return src
	}
	// Fit skaluje do wnętrza prostokąta; Lanczos daje najostrzejszy wynik
	// przy pomniejszaniu, a to jedyny kierunek, w którym tu skalujemy.
	return imaging.Fit(src, max, max, imaging.Lanczos)
}

// FindDerivative szuka miniatury danego wariantu, sprawdzając oba możliwe
// rozszerzenia. Dzięki temu nie musimy trzymać formatu w bazie.
func FindDerivative(dir, variant string) (string, bool) {
	for _, ext := range []string{".jpg", ".png"} {
		path := filepath.Join(dir, variant+ext)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

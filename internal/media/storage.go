package media

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"designfolio/internal/model"
)

// ErrTooLarge zwracamy, gdy pojedynczy plik przekroczy limit.
var ErrTooLarge = errors.New("media: plik przekracza limit")

// Storage zarządza katalogiem data/files.
//
// Układ na dysku jest celowo przewidywalny i czytelny bez aplikacji:
//
//	data/files/<asset-id>/<oryginalna-nazwa>   ← zawsze zachowany oryginał
//	data/files/<asset-id>/thumb.jpg|.png       ← dłuższy bok 400 px
//	data/files/<asset-id>/preview.jpg|.png     ← dłuższy bok 1600 px
type Storage struct {
	root       string
	maxFileSize int64
}

func NewStorage(root string, maxFileSize int64) *Storage {
	return &Storage{root: root, maxFileSize: maxFileSize}
}

// Dir zwraca katalog jednego assetu.
func (s *Storage) Dir(assetID string) string {
	return filepath.Join(s.root, assetID)
}

// Saved to komplet metadanych powstałych przy zapisie pliku.
type Saved struct {
	Filename string
	Mime     string
	Size     int64
	Kind     model.AssetKind
	Width    *int
	Height   *int
	Checksum string
	HasThumb bool
}

// Save zapisuje strumień na dysk i — dla obrazów — generuje miniatury.
//
// Zapis idzie najpierw do pliku tymczasowego w tym samym katalogu, a dopiero
// udany transfer kończy się zmianą nazwy. Przerwany upload nie zostawia więc
// obciętego pliku pod właściwą nazwą; os.Rename w obrębie jednego wolumenu
// jest operacją atomową.
func (s *Storage) Save(assetID, rawFilename string, r io.Reader) (*Saved, error) {
	filename := SafeFilename(rawFilename)
	dir := s.Dir(assetID)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("media: utworzenie katalogu %s: %w", dir, err)
	}

	tmpPath := filepath.Join(dir, ".upload.tmp")
	tmp, err := os.Create(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("media: plik tymczasowy: %w", err)
	}

	hasher := sha256.New()
	head := &headBuffer{}

	// io.MultiWriter rozdziela ten sam strumień na trzy odbiorniki naraz:
	// plik, funkcję skrótu i bufor na pierwsze bajty (do rozpoznania typu).
	// Dzięki temu czytamy dane dokładnie raz.
	limited := io.LimitReader(r, s.maxFileSize+1)
	size, copyErr := io.Copy(io.MultiWriter(tmp, hasher, head), limited)

	closeErr := tmp.Close()

	cleanup := func() { _ = os.RemoveAll(dir) }

	if copyErr != nil {
		cleanup()
		return nil, fmt.Errorf("media: zapis pliku: %w", copyErr)
	}
	if closeErr != nil {
		cleanup()
		return nil, fmt.Errorf("media: zamknięcie pliku: %w", closeErr)
	}
	if size > s.maxFileSize {
		cleanup()
		return nil, ErrTooLarge
	}
	if size == 0 {
		cleanup()
		return nil, fmt.Errorf("media: plik %q jest pusty", filename)
	}

	result := &Saved{
		Filename: filename,
		Size:     size,
		Checksum: hex.EncodeToString(hasher.Sum(nil)),
		Mime:     DetectMime(filename, head.bytes),
	}
	result.Kind = KindForMime(result.Mime, filename)

	if result.Kind == model.AssetImage {
		info, err := makeDerivatives(tmpPath, dir)
		if err != nil {
			// Plik z rozszerzeniem obrazu, którego nie umiemy zdekodować
			// (np. .psd albo uszkodzony .png), zostaje jako zwykły załącznik.
			// Odrzucanie całego uploadu byłoby gorsze niż brak miniatury.
			slog.Warn("nie udało się zrobić miniatur", "plik", filename, "err", err)
			result.Kind = model.AssetOther
		} else {
			result.Width = &info.Width
			result.Height = &info.Height
			result.HasThumb = info.HasThumb
		}
	}

	finalPath := filepath.Join(dir, filename)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		cleanup()
		return nil, fmt.Errorf("media: finalizacja pliku %s: %w", filename, err)
	}
	return result, nil
}

// Open otwiera wskazany wariant pliku. Zwraca też ścieżkę, bo wołający
// potrzebuje jej do http.ServeContent (nazwa decyduje o Content-Type).
func (s *Storage) Open(assetID, filename, variant string) (*os.File, string, error) {
	dir := s.Dir(assetID)

	var path string
	switch variant {
	case "original":
		// SafeFilename ponownie, mimo że nazwa pochodzi z bazy: gdyby
		// kiedykolwiek trafiła tam wartość ze ścieżką, to jest ostatnie miejsce,
		// w którym da się to zatrzymać.
		path = filepath.Join(dir, SafeFilename(filename))
	case "thumb", "preview":
		found, ok := FindDerivative(dir, variant)
		if !ok {
			return nil, "", os.ErrNotExist
		}
		path = found
	default:
		return nil, "", fmt.Errorf("media: nieznany wariant %q", variant)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	return file, path, nil
}

// Remove kasuje cały katalog assetu razem z miniaturami.
func (s *Storage) Remove(assetID string) error {
	if err := os.RemoveAll(s.Dir(assetID)); err != nil {
		return fmt.Errorf("media: usunięcie plików assetu %s: %w", assetID, err)
	}
	return nil
}

// headBuffer zbiera pierwsze bajty strumienia do rozpoznania typu zawartości,
// przepuszczając wszystko dalej bez zmian.
type headBuffer struct {
	bytes []byte
}

const headSize = 512

func (h *headBuffer) Write(p []byte) (int, error) {
	if missing := headSize - len(h.bytes); missing > 0 {
		if len(p) < missing {
			missing = len(p)
		}
		h.bytes = append(h.bytes, p[:missing]...)
	}
	return len(p), nil
}

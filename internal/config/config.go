// Package config wczytuje ustawienia aplikacji ze zmiennych środowiskowych
// i z opcjonalnego pliku .env. Celowo bez zewnętrznej biblioteki — parser .env
// to kilkadziesiąt linii, a dzięki temu widać dokładnie, skąd bierze się każda wartość.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config trzyma całą konfigurację runtime'u. Jest tworzony raz w main
// i przekazywany dalej — nigdzie nie sięgamy do os.Getenv poza tym plikiem.
type Config struct {
	// Addr to adres nasłuchu, np. "127.0.0.1:8080".
	// Domyślnie localhost, bo aplikacja nie ma być wystawiona na świat.
	Addr string

	// DataDir to katalog z bazą i plikami: <DataDir>/designfolio.db, <DataDir>/files/...
	DataDir string

	// Token, jeśli niepusty, włącza sprawdzanie nagłówka X-App-Token
	// dla metod zapisu (POST/PUT/PATCH/DELETE). Pusty = middleware wyłączony.
	Token string

	// MaxUploadMB to limit rozmiaru pojedynczego requestu z plikami.
	MaxUploadMB int64

	// Dev włącza bardziej gadatliwe logi i CORS dla serwera Vite.
	Dev bool
}

// DBPath zwraca ścieżkę do pliku bazy.
func (c Config) DBPath() string { return filepath.Join(c.DataDir, "designfolio.db") }

// FilesDir zwraca katalog, w którym lądują oryginały i miniatury.
func (c Config) FilesDir() string { return filepath.Join(c.DataDir, "files") }

// MaxUploadBytes przelicza limit na bajty.
func (c Config) MaxUploadBytes() int64 { return c.MaxUploadMB << 20 }

// Load czyta .env (jeśli istnieje), a potem zmienne środowiskowe.
// Kolejność ma znaczenie: prawdziwe zmienne środowiskowe wygrywają z .env,
// żeby dało się nadpisać ustawienie przy uruchomieniu bez edycji pliku.
func Load(envPath string) (Config, error) {
	if err := loadDotEnv(envPath); err != nil {
		return Config{}, fmt.Errorf("config: wczytanie %s: %w", envPath, err)
	}

	cfg := Config{
		Addr:        envOr("APP_ADDR", "127.0.0.1:8080"),
		DataDir:     envOr("DATA_DIR", "./data"),
		Token:       os.Getenv("APP_TOKEN"),
		MaxUploadMB: 512,
		Dev:         envOr("APP_ENV", "prod") == "dev",
	}

	if raw := os.Getenv("MAX_UPLOAD_MB"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("config: MAX_UPLOAD_MB musi być dodatnią liczbą, dostałem %q", raw)
		}
		cfg.MaxUploadMB = n
	}

	abs, err := filepath.Abs(cfg.DataDir)
	if err != nil {
		return Config{}, fmt.Errorf("config: ścieżka bezwzględna dla DATA_DIR: %w", err)
	}
	cfg.DataDir = abs

	return cfg, nil
}

// EnsureDirs tworzy katalogi na dane, jeśli ich nie ma.
func (c Config) EnsureDirs() error {
	for _, dir := range []string{c.DataDir, c.FilesDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("config: utworzenie katalogu %s: %w", dir, err)
		}
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadDotEnv wczytuje plik w formacie KEY=VALUE. Obsługuje komentarze (#),
// puste linie, prefiks "export " i cudzysłowy wokół wartości.
// Brak pliku to nie błąd — .env jest opcjonalny.
func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		// errors.Is zamiast porównania == , bo os.Open owija błąd w *PathError.
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		key, value, found := strings.Cut(line, "=")
		if !found {
			return fmt.Errorf("linia %d: oczekiwano KEY=VALUE, dostałem %q", lineNo, line)
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)

		// Nie nadpisujemy tego, co już jest w środowisku procesu.
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("linia %d: ustawienie %s: %w", lineNo, key, err)
			}
		}
	}
	return scanner.Err()
}

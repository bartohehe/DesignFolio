// Package store to cała warstwa danych: otwarcie SQLite, migracje i zapytania.
// Nie zna HTTP — przyjmuje i zwraca typy z pakietu model.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	// Sterownik rejestruje się pod nazwą "sqlite" w bloku init().
	// Pusty identyfikator importu (_) mówi: "potrzebuję efektu ubocznego,
	// nie używam żadnej nazwy z tego pakietu".
	_ "modernc.org/sqlite"
)

// ErrNotFound zwracamy zamiast sql.ErrNoRows, żeby warstwa HTTP nie musiała
// wiedzieć, że pod spodem jest w ogóle baza danych.
var ErrNotFound = errors.New("store: nie znaleziono")

// Store opakowuje pulę połączeń. *sql.DB to nie jedno połączenie,
// tylko pula — jest bezpieczna do użycia z wielu goroutine naraz.
type Store struct {
	db *sql.DB
}

// Open otwiera bazę, ustawia PRAGMA i wykonuje zaległe migracje.
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("store: otwarcie bazy %s: %w", path, err)
	}

	// SQLite pozwala na wielu czytelników, ale tylko jednego piszącego naraz.
	// Przy kilku połączeniach nadmiarowe zapisy czekają (busy_timeout niżej),
	// zamiast od razu dostawać SQLITE_BUSY.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(time.Hour)

	// sql.Open jest leniwe — nie łączy się, dopóki nie polecą zapytania.
	// Ping wymusza połączenie teraz, żeby błąd wyszedł przy starcie.
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: połączenie z bazą %s: %w", path, err)
	}

	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	// Indeks wyszukiwania jest budowany przez kod, nie przez migrację —
	// przy pierwszym starcie po dodaniu FTS trzeba go uzupełnić.
	indexed, err := s.backfillIndexIfEmpty(ctx)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if indexed > 0 {
		slog.Info("zbudowano indeks wyszukiwania", "itemow", indexed)
	}

	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Ping sprawdza, czy baza odpowiada — używane przez /api/health.
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// DB daje dostęp do surowej puli. Potrzebne w testach i przy eksporcie.
func (s *Store) DB() *sql.DB { return s.db }

// dsn buduje connection string z PRAGMA-mi zapisanymi w URL-u.
//
// Dlaczego nie zwykłe "PRAGMA ..." po otwarciu? Bo *sql.DB to pula:
// PRAGMA wykonane raz dotyczyłoby jednego połączenia, a kolejne wstawałyby
// bez tych ustawień. Zapisane w DSN są stosowane do każdego połączenia w puli.
func dsn(path string) string {
	pragmas := []string{
		// WAL: czytelnicy nie blokują piszącego i odwrotnie.
		"journal_mode(WAL)",
		// KLUCZOWE: SQLite domyślnie IGNORUJE klucze obce.
		// Bez tego ON DELETE CASCADE po prostu nie zadziała.
		"foreign_keys(1)",
		// Czekaj do 5 s, zanim zwrócisz SQLITE_BUSY.
		"busy_timeout(5000)",
		// NORMAL + WAL to standardowy kompromis: bezpieczne przy padzie
		// aplikacji, znacznie szybsze niż FULL.
		"synchronous(NORMAL)",
	}

	q := url.Values{}
	for _, p := range pragmas {
		q.Add("_pragma", p)
	}
	// ToSlash, bo na Windowsie ścieżka ma backslashe, a to jest URI.
	return "file:" + filepath.ToSlash(path) + "?" + q.Encode()
}

//go:embed migrations/*.sql
var migrationFS embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

// migrate wykonuje wszystkie migracje, których jeszcze nie ma w bazie.
func (s *Store) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			name       TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)`)
	if err != nil {
		return fmt.Errorf("store: tabela schema_migrations: %w", err)
	}

	applied, err := s.appliedVersions(ctx)
	if err != nil {
		return err
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.version] {
			continue
		}
		if err := s.applyMigration(ctx, m); err != nil {
			return err
		}
		slog.Info("migracja wykonana", "version", m.version, "name", m.name)
	}
	return nil
}

func (s *Store) appliedVersions(ctx context.Context) (map[int]bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("store: odczyt wykonanych migracji: %w", err)
	}
	// defer rows.Close() jest obowiązkowy — bez tego połączenie wraca do puli
	// dopiero przy garbage collection i pula potrafi się zakleszczyć.
	defer rows.Close()

	applied := map[int]bool{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("store: odczyt wersji migracji: %w", err)
		}
		applied[v] = true
	}
	// rows.Err() łapie błędy, które wystąpiły W TRAKCIE iteracji —
	// pętla po rows.Next() sama ich nie zgłasza.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iteracja po migracjach: %w", err)
	}
	return applied, nil
}

func (s *Store) applyMigration(ctx context.Context, m migration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: start transakcji migracji %d: %w", m.version, err)
	}
	// Rollback po udanym Commit jest bezpieczny — zwraca sql.ErrTxDone,
	// które tu ignorujemy. To standardowy idiom: jedna linia zamiast
	// wołania Rollback w każdej gałęzi błędu.
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return fmt.Errorf("store: migracja %d (%s): %w", m.version, m.name, err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
		m.version, m.name, formatTime(time.Now()))
	if err != nil {
		return fmt.Errorf("store: zapis wersji migracji %d: %w", m.version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit migracji %d: %w", m.version, err)
	}
	return nil
}

// loadMigrations czyta wbudowane pliki i sortuje je po numerze wersji.
// Nazwa pliku musi mieć postać <numer>_<nazwa>.sql, np. 0001_init.sql.
func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("store: odczyt katalogu migracji: %w", err)
	}

	out := make([]migration, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}

		numStr, name, found := strings.Cut(strings.TrimSuffix(e.Name(), ".sql"), "_")
		if !found {
			return nil, fmt.Errorf("store: zła nazwa migracji %q, oczekiwano <numer>_<nazwa>.sql", e.Name())
		}
		version, err := strconv.Atoi(numStr)
		if err != nil {
			return nil, fmt.Errorf("store: zły numer migracji w %q: %w", e.Name(), err)
		}

		content, err := migrationFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("store: odczyt migracji %s: %w", e.Name(), err)
		}

		out = append(out, migration{version: version, name: name, sql: string(content)})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// --- pomocnicze konwersje czasu --------------------------------------------

// formatTime zapisuje czas w UTC w formacie RFC3339 z milisekundami.
// Stała precyzja jest ważna: gdyby część dat miała milisekundy, a część nie,
// sortowanie tekstowe dawałoby błędne wyniki.
func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

func parseTime(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02T15:04:05.000Z", s)
	if err == nil {
		return t, nil
	}
	// Zapasowe parsowanie: daty wpisane ręcznie przez API mogą przyjść
	// w zwykłym RFC3339, bez stałej liczby cyfr po przecinku.
	return time.Parse(time.RFC3339, s)
}

// nullTimeToPtr zamienia nullowalną kolumnę TEXT na *time.Time.
func nullTimeToPtr(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid || ns.String == "" {
		return nil, nil
	}
	t, err := parseTime(ns.String)
	if err != nil {
		return nil, fmt.Errorf("store: parsowanie daty %q: %w", ns.String, err)
	}
	return &t, nil
}

// ptrToNullTime robi konwersję w drugą stronę, przy zapisie.
func ptrToNullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

// nullStringToPtr zamienia nullowalny TEXT na *string.
func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

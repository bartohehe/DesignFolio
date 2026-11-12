package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"designfolio/internal/model"
)

// querier obejmuje wspólny zestaw metod *sql.DB i *sql.Tx.
// Dzięki niemu jedna funkcja pomocnicza działa i wewnątrz transakcji,
// i poza nią — bez duplikowania kodu.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// ListTags zwraca wszystkie tagi wraz z liczbą itemów, do których są przypięte.
// Sortowanie: najpierw najczęściej używane, potem alfabetycznie.
func (s *Store) ListTags(ctx context.Context) ([]model.Tag, error) {
	const q = `
		SELECT t.id, t.name, t.slug, COUNT(it.item_id) AS cnt
		FROM tags t
		LEFT JOIN item_tags it ON it.tag_id = t.id
		GROUP BY t.id, t.name, t.slug
		ORDER BY cnt DESC, t.name COLLATE NOCASE ASC`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store: lista tagów: %w", err)
	}
	defer rows.Close()

	// Pusty slice zamiast nil — inaczej JSON wyszedłby jako null, a frontend
	// musiałby to obsługiwać osobno. Zawsze zwracamy [].
	tags := []model.Tag{}
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Count); err != nil {
			return nil, fmt.Errorf("store: odczyt tagu: %w", err)
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iteracja po tagach: %w", err)
	}
	return tags, nil
}

// ensureTags zamienia nazwy tagów na rekordy, tworząc brakujące w locie.
// Dopasowanie idzie po slugu, więc "Druk 3D", "druk 3d" i "DRUK-3D"
// trafiają w ten sam tag; zachowana zostaje pisownia z pierwszego użycia.
func ensureTags(ctx context.Context, q querier, names []string) ([]model.Tag, error) {
	tags := make([]model.Tag, 0, len(names))
	now := formatTime(time.Now())

	for _, name := range names {
		slug := model.Slugify(name)
		if slug == "" {
			continue // sama interpunkcja albo emoji — nie ma z czego zrobić tagu
		}

		var t model.Tag
		err := q.QueryRowContext(ctx,
			`SELECT id, name, slug FROM tags WHERE slug = ?`, slug,
		).Scan(&t.ID, &t.Name, &t.Slug)

		switch {
		case err == nil:
			tags = append(tags, t)

		case errors.Is(err, sql.ErrNoRows):
			t = model.Tag{ID: uuid.NewString(), Name: name, Slug: slug}
			_, err := q.ExecContext(ctx,
				`INSERT INTO tags (id, name, slug, created_at) VALUES (?, ?, ?, ?)`,
				t.ID, t.Name, t.Slug, now)
			if err != nil {
				return nil, fmt.Errorf("store: utworzenie tagu %q: %w", name, err)
			}
			tags = append(tags, t)

		default:
			return nil, fmt.Errorf("store: wyszukanie tagu %q: %w", name, err)
		}
	}
	return tags, nil
}

// setItemTags ustawia dokładnie taki zestaw tagów, jaki podano.
// Najprostsze poprawne podejście: skasuj powiązania i wstaw od nowa.
// Przy maksymalnie 30 tagach na item nie ma sensu kombinować z różnicami.
func setItemTags(ctx context.Context, q querier, itemID string, names []string) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM item_tags WHERE item_id = ?`, itemID); err != nil {
		return fmt.Errorf("store: usunięcie starych tagów itemu: %w", err)
	}

	tags, err := ensureTags(ctx, q, names)
	if err != nil {
		return err
	}

	for _, t := range tags {
		_, err := q.ExecContext(ctx,
			`INSERT INTO item_tags (item_id, tag_id) VALUES (?, ?)`, itemID, t.ID)
		if err != nil {
			return fmt.Errorf("store: przypięcie tagu %q: %w", t.Slug, err)
		}
	}
	return nil
}

// tagsForItems pobiera tagi dla wielu itemów jednym zapytaniem.
//
// To rozwiązanie problemu N+1: zamiast osobnego zapytania na każdy item
// (przy 100 itemach = 101 zapytań), robimy jedno z klauzulą IN.
func tagsForItems(ctx context.Context, q querier, itemIDs []string) (map[string][]model.Tag, error) {
	result := make(map[string][]model.Tag, len(itemIDs))
	if len(itemIDs) == 0 {
		return result, nil
	}

	// Liczba placeholderów musi odpowiadać liczbie argumentów, więc budujemy
	// "?, ?, ?" dynamicznie. To jedyny bezpieczny sposób na IN — sklejanie
	// wartości bezpośrednio w SQL otwierałoby drogę do SQL injection.
	placeholders := strings.Repeat("?, ", len(itemIDs)-1) + "?"
	args := make([]any, len(itemIDs))
	for i, id := range itemIDs {
		args[i] = id
	}

	query := `
		SELECT it.item_id, t.id, t.name, t.slug
		FROM item_tags it
		JOIN tags t ON t.id = it.tag_id
		WHERE it.item_id IN (` + placeholders + `)
		ORDER BY t.name COLLATE NOCASE ASC`

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: tagi dla itemów: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var itemID string
		var t model.Tag
		if err := rows.Scan(&itemID, &t.ID, &t.Name, &t.Slug); err != nil {
			return nil, fmt.Errorf("store: odczyt tagu itemu: %w", err)
		}
		result[itemID] = append(result[itemID], t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iteracja po tagach itemów: %w", err)
	}
	return result, nil
}

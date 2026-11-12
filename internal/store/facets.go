package store

import (
	"context"
	"fmt"

	"designfolio/internal/model"
)

// Facet to jedna pozycja w indeksie: wartość filtra i ile itemów ją ma.
type Facet struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// Facets zasilają boczny indeks w UI. Zwracamy komplet możliwych wartości,
// także tych z zerem — dzięki temu lista filtrów nie skacze, gdy ostatni
// item danego rodzaju zniknie.
type Facets struct {
	Total     int     `json:"total"`
	Favorites int     `json:"favorites"`
	Kinds     []Facet `json:"kinds"`
	Statuses  []Facet `json:"statuses"`
}

// FacetCounts liczy itemy w rozbiciu na rodzaj i status.
func (s *Store) FacetCounts(ctx context.Context) (*Facets, error) {
	byKind, err := s.countBy(ctx, "kind")
	if err != nil {
		return nil, err
	}
	byStatus, err := s.countBy(ctx, "status")
	if err != nil {
		return nil, err
	}

	f := &Facets{
		Kinds:    make([]Facet, 0, len(model.AllKinds())),
		Statuses: make([]Facet, 0, len(model.AllStatuses())),
	}

	// Kolejność bierzemy z modelu, nie z bazy — ma być stała i sensowna
	// dla człowieka, a nie zależna od tego, co akurat jest najliczniejsze.
	for _, k := range model.AllKinds() {
		f.Kinds = append(f.Kinds, Facet{Value: string(k), Count: byKind[string(k)]})
	}
	for _, st := range model.AllStatuses() {
		f.Statuses = append(f.Statuses, Facet{Value: string(st), Count: byStatus[string(st)]})
	}

	row := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(favorite), 0) FROM items`)
	if err := row.Scan(&f.Total, &f.Favorites); err != nil {
		return nil, fmt.Errorf("store: zliczenie itemów i ulubionych: %w", err)
	}
	return f, nil
}

// countBy grupuje itemy po jednej kolumnie. Nazwa kolumny NIE pochodzi
// od użytkownika — jest wpisana na sztywno w wywołaniach powyżej.
func (s *Store) countBy(ctx context.Context, column string) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+column+`, COUNT(*) FROM items GROUP BY `+column)
	if err != nil {
		return nil, fmt.Errorf("store: zliczenie po %s: %w", column, err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var value string
		var count int
		if err := rows.Scan(&value, &count); err != nil {
			return nil, fmt.Errorf("store: odczyt licznika %s: %w", column, err)
		}
		counts[value] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iteracja po licznikach %s: %w", column, err)
	}
	return counts, nil
}

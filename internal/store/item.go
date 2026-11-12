package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"designfolio/internal/model"
)

// itemColumns trzymamy w jednym miejscu, bo ta sama lista jest używana
// przy pobieraniu pojedynczego itemu i przy listowaniu. Kolejność musi
// zgadzać się z scanItem niżej. Tabela items jest wszędzie aliasowana jako "i",
// żeby ta jedna stała pasowała do obu zapytań.
const itemColumns = `
	i.id, i.title, i.slug, i.description, i.kind, i.status, i.favorite,
	i.external_url, i.metadata, i.happened_at, i.created_at, i.updated_at`

// rowScanner obejmuje *sql.Row i *sql.Rows — obie mają metodę Scan
// o tej samej sygnaturze, więc jedna funkcja skanująca obsłuży oba przypadki.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanItem(sc rowScanner) (*model.Item, error) {
	var (
		it          model.Item
		favorite    int64
		externalURL sql.NullString
		metadata    []byte
		happenedAt  sql.NullString
		createdAt   string
		updatedAt   string
	)

	err := sc.Scan(
		&it.ID, &it.Title, &it.Slug, &it.Description, &it.Kind, &it.Status, &favorite,
		&externalURL, &metadata, &happenedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err // wołający decyduje, czy sql.ErrNoRows to błąd
	}

	it.Favorite = favorite != 0
	it.ExternalURL = nullStringToPtr(externalURL)

	// Kopiujemy bajty: bufor zwrócony przez Scan może zostać nadpisany
	// przy kolejnym rows.Next(). To klasyczna pułapka database/sql.
	it.Metadata = append(json.RawMessage(nil), metadata...)
	if len(it.Metadata) == 0 {
		it.Metadata = json.RawMessage(`{}`)
	}

	if it.HappenedAt, err = nullTimeToPtr(happenedAt); err != nil {
		return nil, err
	}
	if it.CreatedAt, err = parseTime(createdAt); err != nil {
		return nil, fmt.Errorf("store: created_at itemu %s: %w", it.ID, err)
	}
	if it.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return nil, fmt.Errorf("store: updated_at itemu %s: %w", it.ID, err)
	}

	it.Tags = []model.Tag{} // nigdy nil, żeby JSON miał [] a nie null
	return &it, nil
}

// CreateItem zapisuje nowy item razem z tagami.
func (s *Store) CreateItem(ctx context.Context, in model.ItemInput) (*model.Item, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: start transakcji: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	id := uuid.NewString()
	slug, err := uniqueSlug(ctx, tx, model.Slugify(in.Title), "")
	if err != nil {
		return nil, err
	}
	now := time.Now()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO items (
			id, title, slug, description, kind, status, favorite,
			external_url, metadata, happened_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, in.Title, slug, in.Description, string(in.Kind), string(in.Status), boolToInt(in.Favorite),
		in.ExternalURL, string(in.Metadata), ptrToNullTime(in.HappenedAt),
		formatTime(now), formatTime(now),
	)
	if err != nil {
		return nil, fmt.Errorf("store: zapis itemu: %w", err)
	}

	if err := setItemTags(ctx, tx, id, in.Tags); err != nil {
		return nil, err
	}

	// Indeks wyszukiwania aktualizujemy w tej samej transakcji, co dane —
	// wycofanie transakcji cofa jedno i drugie.
	if err := reindexItem(ctx, tx, id); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit tworzenia itemu: %w", err)
	}
	return s.GetItem(ctx, id)
}

// GetItem znajduje item po UUID albo po slugu — frontend używa w URL-ach
// sluga, ale API musi też przyjąć id.
func (s *Store) GetItem(ctx context.Context, idOrSlug string) (*model.Item, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+itemColumns+` FROM items i WHERE i.id = ? OR i.slug = ?`, idOrSlug, idOrSlug)

	it, err := scanItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: pobranie itemu %q: %w", idOrSlug, err)
	}

	tagsByItem, err := tagsForItems(ctx, s.db, []string{it.ID})
	if err != nil {
		return nil, err
	}
	if tags, ok := tagsByItem[it.ID]; ok {
		it.Tags = tags
	}

	covers, err := coversForItems(ctx, s.db, []string{it.ID})
	if err != nil {
		return nil, err
	}
	it.Cover = covers[it.ID]

	if it.Assets, err = s.AssetsForItem(ctx, it.ID); err != nil {
		return nil, err
	}
	return it, nil
}

// UpdateItem nanosi częściową zmianę. Pola nieobecne w żądaniu zostają
// nietknięte — o tym, które przyszły, decyduje model.Optional.
func (s *Store) UpdateItem(ctx context.Context, id string, p model.ItemPatch) (*model.Item, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: start transakcji: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Sprawdzamy istnienie w tej samej transakcji, żeby między sprawdzeniem
	// a UPDATE nikt nie zdążył itemu skasować.
	var currentTitle, currentSlug string
	err = tx.QueryRowContext(ctx, `SELECT title, slug FROM items WHERE id = ?`, id).
		Scan(&currentTitle, &currentSlug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: sprawdzenie itemu %s: %w", id, err)
	}

	// Budujemy listę "kolumna = ?" tylko z pól, które faktycznie przyszły.
	sets := []string{}
	args := []any{}

	addSet := func(column string, value any) {
		sets = append(sets, column+" = ?")
		args = append(args, value)
	}

	if title, ok := p.Title.Get(); ok {
		addSet("title", title)
		// Slug idzie za tytułem, ale tylko gdy tytuł naprawdę się zmienił —
		// inaczej edycja opisu psułaby istniejące linki.
		if title != currentTitle {
			slug, err := uniqueSlug(ctx, tx, model.Slugify(title), id)
			if err != nil {
				return nil, err
			}
			addSet("slug", slug)
		}
	}
	if description, ok := p.Description.Get(); ok {
		addSet("description", description)
	}
	if kind, ok := p.Kind.Get(); ok {
		addSet("kind", string(kind))
	}
	if status, ok := p.Status.Get(); ok {
		addSet("status", string(status))
	}
	if favorite, ok := p.Favorite.Get(); ok {
		addSet("favorite", boolToInt(favorite))
	}
	if metadata, ok := p.Metadata.Get(); ok {
		addSet("metadata", string(metadata))
	}
	// Poniższe dwa pola są nullowalne: Set=true przy Value=nil znaczy
	// "wyczyść", więc nie możemy użyć Get(), który skleja oba przypadki.
	if p.ExternalURL.Set {
		if p.ExternalURL.Value == nil {
			addSet("external_url", nil)
		} else {
			addSet("external_url", *p.ExternalURL.Value)
		}
	}
	if p.HappenedAt.Set {
		addSet("happened_at", ptrToNullTime(p.HappenedAt.Value))
	}

	if len(sets) > 0 {
		addSet("updated_at", formatTime(time.Now()))
		args = append(args, id)

		query := `UPDATE items SET ` + strings.Join(sets, ", ") + ` WHERE id = ?`
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return nil, fmt.Errorf("store: aktualizacja itemu %s: %w", id, err)
		}
	}

	if p.Tags.Set && p.Tags.Value != nil {
		if err := setItemTags(ctx, tx, id, *p.Tags.Value); err != nil {
			return nil, err
		}
		// Zmiana samych tagów też jest zmianą itemu.
		if len(sets) == 0 {
			if _, err := tx.ExecContext(ctx,
				`UPDATE items SET updated_at = ? WHERE id = ?`, formatTime(time.Now()), id); err != nil {
				return nil, fmt.Errorf("store: aktualizacja updated_at: %w", err)
			}
		}
	}

	if err := reindexItem(ctx, tx, id); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit aktualizacji itemu: %w", err)
	}
	return s.GetItem(ctx, id)
}

// DeleteItem usuwa item i zwraca identyfikatory jego plików, żeby wołający
// mógł posprzątać dysk. Powiązania w item_tags i rekordy assetów znikają
// przez ON DELETE CASCADE — działa to tylko dlatego, że w DSN włączyliśmy
// PRAGMA foreign_keys(1).
//
// Kolejność jest przemyślana: najpierw commit w bazie, potem kasowanie plików.
// Osierocony plik na dysku jest nieszkodliwy; rekord wskazujący na nieistniejący
// plik psuje widok.
func (s *Store) DeleteItem(ctx context.Context, id string) ([]string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: start transakcji: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, `SELECT id FROM assets WHERE item_id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("store: pliki usuwanego itemu: %w", err)
	}
	assetIDs := []string{}
	for rows.Next() {
		var assetID string
		if err := rows.Scan(&assetID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("store: odczyt id pliku: %w", err)
		}
		assetIDs = append(assetIDs, assetID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("store: iteracja po plikach itemu: %w", err)
	}
	rows.Close()

	res, err := tx.ExecContext(ctx, `DELETE FROM items WHERE id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("store: usunięcie itemu %s: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("store: liczba usuniętych wierszy: %w", err)
	}
	if affected == 0 {
		return nil, ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit usunięcia itemu: %w", err)
	}
	return assetIDs, nil
}

// --- listowanie -------------------------------------------------------------

// ListFilter to komplet parametrów listy. Zerowa wartość struktury
// (wszystko puste) oznacza "wszystkie itemy, domyślne sortowanie".
type ListFilter struct {
	Kinds    []model.Kind
	Statuses []model.Status
	TagSlugs []string
	Favorite *bool
	// Query to fraza z pola wyszukiwania. Pusta = brak wyszukiwania.
	Query  string
	Sort   string // relevance | happened_at | created_at | updated_at | title
	Order  string // asc | desc
	Limit  int
	Offset int

	// match to Query przetłumaczone na składnię FTS5. Ustawiane w normalize;
	// puste oznacza, że nie ma czego szukać (np. wpisano same znaki
	// interpunkcyjne) i zapytanie leci bez JOIN-a z indeksem.
	match string
}

// ListResult niesie stronę wyników razem z informacją, ile jest wszystkich.
type ListResult struct {
	Items  []model.Item `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

// Domyślne i maksymalne wartości stronicowania.
const (
	DefaultLimit = 60
	MaxLimit     = 200
)

// sortColumns to biała lista — nazwa kolumny nigdy nie może pochodzić
// wprost od użytkownika, bo trafia do SQL-a bez placeholdera.
var sortColumns = map[string]string{
	"happened_at": "COALESCE(i.happened_at, i.created_at)",
	"created_at":  "i.created_at",
	"updated_at":  "i.updated_at",
	"title":       "i.title COLLATE NOCASE",
}

// ListItems zwraca stronę itemów pasujących do filtra.
func (s *Store) ListItems(ctx context.Context, f ListFilter) (*ListResult, error) {
	f.normalize()

	from, where, args := f.buildFromWhere()

	var total int
	countQuery := `SELECT COUNT(*)` + from + where
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("store: zliczenie itemów: %w", err)
	}

	// Drugie kryterium sortowania (i.id) daje stabilną kolejność, gdy
	// pierwsze kryterium ma remisy — bez tego stronicowanie potrafi
	// pokazać ten sam item na dwóch stronach.
	query := `SELECT ` + itemColumns + from + where + `
		ORDER BY ` + f.orderExpression() + `, i.id ASC
		LIMIT ? OFFSET ?`

	pageArgs := append(append([]any{}, args...), f.Limit, f.Offset)

	rows, err := s.db.QueryContext(ctx, query, pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("store: lista itemów: %w", err)
	}
	defer rows.Close()

	items := []model.Item{}
	ids := []string{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("store: odczyt itemu z listy: %w", err)
		}
		items = append(items, *it)
		ids = append(ids, it.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iteracja po itemach: %w", err)
	}

	tagsByItem, err := tagsForItems(ctx, s.db, ids)
	if err != nil {
		return nil, err
	}
	covers, err := coversForItems(ctx, s.db, ids)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if tags, ok := tagsByItem[items[i].ID]; ok {
			items[i].Tags = tags
		}
		items[i].Cover = covers[items[i].ID]
	}

	return &ListResult{Items: items, Total: total, Limit: f.Limit, Offset: f.Offset}, nil
}

func (f *ListFilter) normalize() {
	f.Query = strings.TrimSpace(f.Query)
	f.match = buildMatchQuery(f.Query)

	// Fraza złożona z samej interpunkcji ("***", "???") nie da się przełożyć
	// na zapytanie FTS. Traktujemy ją jak brak wyszukiwania, zamiast budować
	// zapytanie, które nie ma po czym sortować.
	searching := f.match != ""

	// Sortowanie przechodzi przez białą listę — nazwa kolumny trafia do SQL-a
	// bez placeholdera, więc nie może pochodzić wprost od użytkownika.
	// "relevance" jest dozwolone wyłącznie przy wyszukiwaniu: bez JOIN-a
	// z indeksem funkcja bm25 nie ma do czego się odwołać.
	_, knownColumn := sortColumns[f.Sort]
	if !knownColumn && !(f.Sort == "relevance" && searching) {
		if searching {
			f.Sort = "relevance"
		} else {
			f.Sort = "happened_at"
		}
	}

	if !strings.EqualFold(f.Order, "asc") {
		f.Order = "DESC"
	} else {
		f.Order = "ASC"
	}
	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	// Górnego limitu NIE narzucamy tutaj: store obsługuje też wywołania
	// wewnętrzne (oś czasu, eksport), które potrzebują większych porcji.
	// Wartość od użytkownika przycina warstwa HTTP — tam, gdzie wchodzi.
	if f.Offset < 0 {
		f.Offset = 0
	}
}

// orderExpression zwraca wyrażenie do ORDER BY. Nigdy nie zawiera niczego,
// co pochodzi od użytkownika — sortowanie przechodzi przez białą listę.
func (f ListFilter) orderExpression() string {
	if f.Sort == "relevance" {
		// bm25 zwraca wartości ujemne: im mniejsza, tym lepsze dopasowanie,
		// więc rosnąco = od najtrafniejszych. Wagi kolumn (title, description,
		// tags) podnoszą znaczenie tytułu i tagu nad treścią opisu.
		return `bm25(items_fts, 10.0, 1.0, 5.0) ASC`
	}
	return sortColumns[f.Sort] + ` ` + f.Order
}

// buildFromWhere skleja klauzule FROM i WHERE. Wszystkie wartości idą przez
// placeholdery (?), więc nic pochodzącego od użytkownika nie ląduje w SQL-u.
func (f ListFilter) buildFromWhere() (string, string, []any) {
	from := ` FROM items i`
	conds := []string{}
	args := []any{}

	if f.match != "" {
		// Bez aliasu: FTS5 wymaga nazwy tabeli po lewej stronie MATCH.
		from += ` JOIN items_fts ON items_fts.rowid = i.rowid`
		conds = append(conds, `items_fts MATCH ?`)
		args = append(args, f.match)
	}

	if len(f.Kinds) > 0 {
		placeholders := make([]string, len(f.Kinds))
		for i, k := range f.Kinds {
			placeholders[i] = "?"
			args = append(args, string(k))
		}
		conds = append(conds, `i.kind IN (`+strings.Join(placeholders, ", ")+`)`)
	}

	if len(f.Statuses) > 0 {
		placeholders := make([]string, len(f.Statuses))
		for i, st := range f.Statuses {
			placeholders[i] = "?"
			args = append(args, string(st))
		}
		conds = append(conds, `i.status IN (`+strings.Join(placeholders, ", ")+`)`)
	}

	if f.Favorite != nil {
		conds = append(conds, `i.favorite = ?`)
		args = append(args, boolToInt(*f.Favorite))
	}

	if len(f.TagSlugs) > 0 {
		// Semantyka AND: item musi mieć WSZYSTKIE wskazane tagi.
		// Liczymy, ile z podanych tagów item posiada, i porównujemy z liczbą podanych.
		placeholders := make([]string, len(f.TagSlugs))
		for i, slug := range f.TagSlugs {
			placeholders[i] = "?"
			args = append(args, slug)
		}
		conds = append(conds, `(
			SELECT COUNT(DISTINCT t.slug)
			FROM item_tags it
			JOIN tags t ON t.id = it.tag_id
			WHERE it.item_id = i.id AND t.slug IN (`+strings.Join(placeholders, ", ")+`)
		) = ?`)
		args = append(args, len(f.TagSlugs))
	}

	if len(conds) == 0 {
		return from, "", args
	}
	return from, " WHERE " + strings.Join(conds, " AND "), args
}

// --- pomocnicze -------------------------------------------------------------

// uniqueSlug zwraca slug wolny w tabeli items. Przy kolizji dokleja -2, -3, …
// excludeID pozwala pominąć własny rekord przy aktualizacji.
func uniqueSlug(ctx context.Context, q querier, base, excludeID string) (string, error) {
	if base == "" {
		base = "item" // tytuł złożony wyłącznie z emoji albo interpunkcji
	}

	for attempt := 1; attempt <= 1000; attempt++ {
		candidate := base
		if attempt > 1 {
			candidate = fmt.Sprintf("%s-%d", base, attempt)
		}

		var exists int
		err := q.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM items WHERE slug = ? AND id <> ?`, candidate, excludeID).
			Scan(&exists)
		if err != nil {
			return "", fmt.Errorf("store: sprawdzenie sluga %q: %w", candidate, err)
		}
		if exists == 0 {
			return candidate, nil
		}
	}

	// Tysiąc itemów o identycznym tytule to sytuacja, której nie ma sensu
	// dalej obsługiwać elegancko — dokładamy fragment UUID i kończymy.
	return base + "-" + uuid.NewString()[:8], nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

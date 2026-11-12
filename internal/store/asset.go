package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"designfolio/internal/model"
)

// assetColumns — kolejność musi zgadzać się ze scanAsset.
const assetColumns = `
	a.id, a.item_id, a.filename, a.mime, a.size, a.kind,
	a.width, a.height, a.checksum, a.sort_order, a.is_cover, a.has_thumb, a.created_at`

func scanAsset(sc rowScanner) (*model.Asset, error) {
	var (
		as        model.Asset
		width     sql.NullInt64
		height    sql.NullInt64
		isCover   int64
		hasThumb  int64
		createdAt string
	)

	err := sc.Scan(
		&as.ID, &as.ItemID, &as.Filename, &as.Mime, &as.Size, &as.Kind,
		&width, &height, &as.Checksum, &as.SortOrder, &isCover, &hasThumb, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	// Wymiary są znane tylko dla obrazów — dla modelu 3D czy PDF-a zostają nil.
	if width.Valid {
		w := int(width.Int64)
		as.Width = &w
	}
	if height.Valid {
		h := int(height.Int64)
		as.Height = &h
	}
	as.IsCover = isCover != 0
	as.HasThumb = hasThumb != 0

	if as.CreatedAt, err = parseTime(createdAt); err != nil {
		return nil, fmt.Errorf("store: created_at assetu %s: %w", as.ID, err)
	}
	return &as, nil
}

// coversForItems pobiera okładki wielu itemów jednym zapytaniem —
// ta sama ochrona przed problemem N+1 co w tagsForItems.
//
// Gdy item nie ma jawnie oznaczonej okładki, ale ma jakieś zdjęcie,
// za okładkę robi pierwszy obrazek w kolejności. Dzięki temu kafelek
// pokazuje coś sensownego, zanim użytkownik cokolwiek wybierze ręcznie.
func coversForItems(ctx context.Context, q querier, itemIDs []string) (map[string]*model.Asset, error) {
	covers := make(map[string]*model.Asset, len(itemIDs))
	if len(itemIDs) == 0 {
		return covers, nil
	}

	placeholders := strings.Repeat("?, ", len(itemIDs)-1) + "?"
	args := make([]any, len(itemIDs))
	for i, id := range itemIDs {
		args[i] = id
	}

	// Sortowanie: najpierw jawna okładka (is_cover DESC), potem kolejność
	// ustawiona przez użytkownika. Pierwszy wiersz dla danego itemu wygrywa.
	query := `
		SELECT ` + assetColumns + `
		FROM assets a
		WHERE a.item_id IN (` + placeholders + `) AND a.kind = 'image'
		ORDER BY a.item_id, a.is_cover DESC, a.sort_order ASC, a.created_at ASC`

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: okładki itemów: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		as, err := scanAsset(rows)
		if err != nil {
			return nil, fmt.Errorf("store: odczyt okładki: %w", err)
		}
		if _, taken := covers[as.ItemID]; !taken {
			covers[as.ItemID] = as
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iteracja po okładkach: %w", err)
	}
	return covers, nil
}

// AssetsForItem zwraca wszystkie pliki itemu w kolejności ustawionej
// przez użytkownika.
func (s *Store) AssetsForItem(ctx context.Context, itemID string) ([]model.Asset, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+assetColumns+` FROM assets a WHERE a.item_id = ?
		 ORDER BY a.sort_order ASC, a.created_at ASC`, itemID)
	if err != nil {
		return nil, fmt.Errorf("store: pliki itemu %s: %w", itemID, err)
	}
	defer rows.Close()

	assets := []model.Asset{}
	for rows.Next() {
		as, err := scanAsset(rows)
		if err != nil {
			return nil, fmt.Errorf("store: odczyt pliku: %w", err)
		}
		assets = append(assets, *as)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iteracja po plikach: %w", err)
	}
	return assets, nil
}

// GetAsset zwraca pojedynczy plik — używane przy serwowaniu zawartości.
func (s *Store) GetAsset(ctx context.Context, id string) (*model.Asset, error) {
	as, err := scanAsset(s.db.QueryRowContext(ctx,
		`SELECT `+assetColumns+` FROM assets a WHERE a.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: pobranie pliku %s: %w", id, err)
	}
	return as, nil
}

// FindAssetByChecksum szuka pliku o tej samej treści w obrębie jednego itemu.
// Służy do deduplikacji: wrzucenie tego samego zdjęcia dwa razy nie ma sensu.
func (s *Store) FindAssetByChecksum(ctx context.Context, itemID, checksum string) (*model.Asset, error) {
	as, err := scanAsset(s.db.QueryRowContext(ctx,
		`SELECT `+assetColumns+` FROM assets a WHERE a.item_id = ? AND a.checksum = ?`,
		itemID, checksum))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: szukanie duplikatu: %w", err)
	}
	return as, nil
}

// CreateAsset zapisuje metadane pliku. Sam plik leży już na dysku —
// kolejność jest ważna: gdyby zapis do bazy padł, zostaje osierocony plik
// (nieszkodliwy), a nie rekord bez pliku (psuje widok).
func (s *Store) CreateAsset(ctx context.Context, a model.Asset) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: start transakcji: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Nowy plik ląduje na końcu listy.
	var nextOrder int
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(sort_order), -1) + 1 FROM assets WHERE item_id = ?`, a.ItemID).
		Scan(&nextOrder)
	if err != nil {
		return fmt.Errorf("store: kolejność nowego pliku: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO assets (
			id, item_id, filename, mime, size, kind,
			width, height, checksum, sort_order, is_cover, has_thumb, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		a.ID, a.ItemID, a.Filename, a.Mime, a.Size, string(a.Kind),
		intPtrArg(a.Width), intPtrArg(a.Height), a.Checksum, nextOrder,
		boolToInt(a.HasThumb), formatTime(time.Now()))
	if err != nil {
		return fmt.Errorf("store: zapis pliku %s: %w", a.Filename, err)
	}

	if err := touchItem(ctx, tx, a.ItemID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit zapisu pliku: %w", err)
	}
	return nil
}

// DeleteAsset usuwa rekord i zwraca item, do którego należał —
// wołający potrzebuje go, żeby odświeżyć indeks wyszukiwania.
func (s *Store) DeleteAsset(ctx context.Context, id string) (itemID string, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("store: start transakcji: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	err = tx.QueryRowContext(ctx, `SELECT item_id FROM assets WHERE id = ?`, id).Scan(&itemID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: sprawdzenie pliku %s: %w", id, err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM assets WHERE id = ?`, id); err != nil {
		return "", fmt.Errorf("store: usunięcie pliku %s: %w", id, err)
	}
	if err := touchItem(ctx, tx, itemID); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("store: commit usunięcia pliku: %w", err)
	}
	return itemID, nil
}

// ArrangeAssets ustawia kolejność plików i wskazuje okładkę.
//
// coverID może być pusty — wtedy żaden plik nie jest jawną okładką
// i kafelek bierze pierwszy obrazek z listy.
func (s *Store) ArrangeAssets(ctx context.Context, itemID string, order []string, coverID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: start transakcji: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for position, assetID := range order {
		res, err := tx.ExecContext(ctx,
			`UPDATE assets SET sort_order = ? WHERE id = ? AND item_id = ?`,
			position, assetID, itemID)
		if err != nil {
			return fmt.Errorf("store: kolejność pliku %s: %w", assetID, err)
		}
		// Warunek item_id chroni przed przestawianiem cudzych plików;
		// brak trafienia znaczy, że przyszedł identyfikator z innego itemu.
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("store: wynik zmiany kolejności: %w", err)
		}
		if affected == 0 {
			return fmt.Errorf("%w: plik %s nie należy do itemu %s", ErrNotFound, assetID, itemID)
		}
	}

	// Okładkę zdejmujemy ze wszystkich, zanim nadamy ją jednemu — inaczej
	// unikalny indeks częściowy (jedna okładka na item) odrzuciłby UPDATE.
	if _, err := tx.ExecContext(ctx,
		`UPDATE assets SET is_cover = 0 WHERE item_id = ?`, itemID); err != nil {
		return fmt.Errorf("store: zdjęcie okładki: %w", err)
	}
	if coverID != "" {
		res, err := tx.ExecContext(ctx,
			`UPDATE assets SET is_cover = 1 WHERE id = ? AND item_id = ?`, coverID, itemID)
		if err != nil {
			return fmt.Errorf("store: ustawienie okładki: %w", err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("store: wynik ustawienia okładki: %w", err)
		}
		if affected == 0 {
			return fmt.Errorf("%w: plik %s nie należy do itemu %s", ErrNotFound, coverID, itemID)
		}
	}

	if err := touchItem(ctx, tx, itemID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit układu plików: %w", err)
	}
	return nil
}

// touchItem odświeża updated_at itemu po zmianie jego plików.
func touchItem(ctx context.Context, q querier, itemID string) error {
	if _, err := q.ExecContext(ctx,
		`UPDATE items SET updated_at = ? WHERE id = ?`, formatTime(time.Now()), itemID); err != nil {
		return fmt.Errorf("store: aktualizacja updated_at itemu %s: %w", itemID, err)
	}
	return nil
}

// intPtrArg zamienia *int na wartość akceptowaną przez sterownik:
// nil musi trafić do bazy jako NULL, nie jako zero.
func intPtrArg(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

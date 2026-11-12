package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"designfolio/internal/model"
)

// Indeks trzyma tekst SPROWADZONY DO ASCII (bez polskich znaków, małymi
// literami) — patrz model.FoldDiacritics. To samo robimy z frazą wpisaną
// przez użytkownika, więc obie strony porównania są w tej samej postaci:
// "słuchawki" znajdzie się po "sluchawki" i odwrotnie.
//
// Dlaczego nie zostawić tego tokenizerowi? Opcja remove_diacritics w unicode61
// rozkłada ó, ż i ć, ale ł jest w Unicode osobną literą i zostaje nietknięte.
// Skoro i tak musimy złożyć jedno słowo ręcznie, robimy całość w jednym miejscu.

// reindexItem odświeża wpis itemu w indeksie wyszukiwania.
//
// Wołane z każdej ścieżki zapisu w TEJ SAMEJ transakcji co zmiana danych —
// dzięki temu indeks nigdy nie rozjeżdża się z tabelą items, nawet gdy
// transakcja się wycofa.
func reindexItem(ctx context.Context, q querier, itemID string) error {
	// FTS5 nie ma UPSERT-a, więc najpierw kasujemy stary wpis.
	if err := deleteFromIndex(ctx, q, itemID); err != nil {
		return err
	}

	var (
		rowid       int64
		title       string
		description string
		tags        sql.NullString
	)
	err := q.QueryRowContext(ctx, `
		SELECT
			i.rowid, i.title, i.description,
			(SELECT group_concat(t.name, ' ')
			 FROM item_tags it JOIN tags t ON t.id = it.tag_id
			 WHERE it.item_id = i.id)
		FROM items i WHERE i.id = ?`, itemID).
		Scan(&rowid, &title, &description, &tags)

	if errors.Is(err, sql.ErrNoRows) {
		return nil // item właśnie zniknął — nie ma czego indeksować
	}
	if err != nil {
		return fmt.Errorf("store: odczyt itemu %s do indeksu: %w", itemID, err)
	}

	_, err = q.ExecContext(ctx,
		`INSERT INTO items_fts (rowid, title, description, tags) VALUES (?, ?, ?, ?)`,
		rowid,
		model.FoldDiacritics(title),
		model.FoldDiacritics(description),
		model.FoldDiacritics(tags.String))
	if err != nil {
		return fmt.Errorf("store: indeksowanie itemu %s: %w", itemID, err)
	}
	return nil
}

func deleteFromIndex(ctx context.Context, q querier, itemID string) error {
	_, err := q.ExecContext(ctx,
		`DELETE FROM items_fts WHERE rowid = (SELECT rowid FROM items WHERE id = ?)`, itemID)
	if err != nil {
		return fmt.Errorf("store: usunięcie itemu %s z indeksu: %w", itemID, err)
	}
	return nil
}

// buildMatchQuery zamienia to, co użytkownik wpisał w pole wyszukiwania,
// na wyrażenie zrozumiałe dla FTS5.
//
// Nie da się wstawić surowego tekstu do MATCH: znaki takie jak ", *, ^, -
// i słowa AND/OR/NOT są w składni FTS5 operatorami, więc zapytanie
// "rysunek (szkic" wysypałoby się błędem składni. Rozbijamy więc tekst
// na słowa i każde opakowujemy w cudzysłów, dopisując * (dopasowanie
// przedrostkowe — "wiez" znajdzie "wieżę").
//
// Wszystkie słowa muszą wystąpić: przy katalogu na kilkaset rzeczy zawężanie
// jest użyteczniejsze niż poszerzanie.
//
// Zwraca pusty string, gdy w zapytaniu nie było ani jednej litery czy cyfry —
// wołający musi wtedy potraktować to jak brak wyszukiwania.
func buildMatchQuery(input string) string {
	folded := model.FoldDiacritics(input)

	tokens := strings.FieldsFunc(folded, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	quoted := make([]string, 0, len(tokens))
	for _, token := range tokens {
		// Podwojony cudzysłów to sposób FTS5 na cudzysłów w treści.
		safe := strings.ReplaceAll(token, `"`, `""`)
		quoted = append(quoted, `"`+safe+`"*`)
	}

	return strings.Join(quoted, " AND ")
}

// ReindexAll przebudowuje cały indeks od zera. Wołane przy starcie, gdy indeks
// jest pusty mimo istniejących itemów (świeżo dodana migracja FTS albo ręczna
// ingerencja w bazę), oraz dostępne do wywołania wprost.
func (s *Store) ReindexAll(ctx context.Context) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("store: start transakcji: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM items_fts`); err != nil {
		return 0, fmt.Errorf("store: czyszczenie indeksu: %w", err)
	}

	rows, err := tx.QueryContext(ctx, `SELECT id FROM items`)
	if err != nil {
		return 0, fmt.Errorf("store: lista itemów do indeksu: %w", err)
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("store: odczyt id do indeksu: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("store: iteracja po itemach: %w", err)
	}
	rows.Close()

	for _, id := range ids {
		if err := reindexItem(ctx, tx, id); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("store: commit przebudowy indeksu: %w", err)
	}
	return len(ids), nil
}

// backfillIndexIfEmpty uzupełnia indeks, gdy w bazie są itemy, a indeks jest
// pusty. Dzieje się tak dokładnie raz: przy pierwszym uruchomieniu wersji,
// która dodała migrację FTS do istniejącego katalogu.
func (s *Store) backfillIndexIfEmpty(ctx context.Context) (int, error) {
	var indexed, items int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM items_fts`).Scan(&indexed); err != nil {
		return 0, fmt.Errorf("store: rozmiar indeksu: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM items`).Scan(&items); err != nil {
		return 0, fmt.Errorf("store: liczba itemów: %w", err)
	}
	if indexed > 0 || items == 0 {
		return 0, nil
	}
	return s.ReindexAll(ctx)
}

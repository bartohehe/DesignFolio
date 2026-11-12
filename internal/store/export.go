package store

import (
	"context"
	"fmt"
	"time"

	"designfolio/internal/model"
)

// Export to cały katalog w jednej strukturze — itemy z tagami i plikami
// plus słownik wszystkich tagów.
//
// Format jest celowo prosty i samoopisujący: gdyby ta aplikacja kiedyś
// przestała istnieć, plik JSON ma dać się przeczytać bez niej.
type Export struct {
	Version    int          `json:"version"`
	ExportedAt time.Time    `json:"exported_at"`
	Counts     ExportCounts `json:"counts"`
	Items      []model.Item `json:"items"`
	Tags       []model.Tag  `json:"tags"`
}

type ExportCounts struct {
	Items  int `json:"items"`
	Assets int `json:"assets"`
	Tags   int `json:"tags"`
}

// ExportFormatVersion podbijamy, gdy zmieni się kształt eksportu.
const ExportFormatVersion = 1

// ExportAll zbiera całą zawartość katalogu.
//
// Bez stronicowania: eksport ma być kompletny z definicji. Przy katalogu
// tej wielkości (tysiące itemów) to i tak kilka megabajtów JSON-a.
func (s *Store) ExportAll(ctx context.Context) (*Export, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+itemColumns+` FROM items i ORDER BY COALESCE(i.happened_at, i.created_at) DESC, i.id ASC`)
	if err != nil {
		return nil, fmt.Errorf("store: eksport itemów: %w", err)
	}
	defer rows.Close()

	items := []model.Item{}
	ids := []string{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("store: odczyt itemu do eksportu: %w", err)
		}
		items = append(items, *it)
		ids = append(ids, it.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iteracja przy eksporcie: %w", err)
	}

	tagsByItem, err := tagsForItems(ctx, s.db, ids)
	if err != nil {
		return nil, err
	}

	assetsByItem, err := s.allAssetsByItem(ctx)
	if err != nil {
		return nil, err
	}

	assetCount := 0
	for i := range items {
		if tags, ok := tagsByItem[items[i].ID]; ok {
			items[i].Tags = tags
		}
		if assets, ok := assetsByItem[items[i].ID]; ok {
			items[i].Assets = assets
			assetCount += len(assets)

			for _, a := range assets {
				if a.IsCover {
					cover := a
					items[i].Cover = &cover
				}
			}
		}
	}

	tags, err := s.ListTags(ctx)
	if err != nil {
		return nil, err
	}

	return &Export{
		Version:    ExportFormatVersion,
		ExportedAt: time.Now().UTC(),
		Counts: ExportCounts{
			Items:  len(items),
			Assets: assetCount,
			Tags:   len(tags),
		},
		Items: items,
		Tags:  tags,
	}, nil
}

// allAssetsByItem pobiera wszystkie pliki jednym zapytaniem i grupuje je
// po itemie — znowu unikamy zapytania na każdy item z osobna.
func (s *Store) allAssetsByItem(ctx context.Context) (map[string][]model.Asset, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+assetColumns+` FROM assets a ORDER BY a.item_id, a.sort_order ASC`)
	if err != nil {
		return nil, fmt.Errorf("store: eksport plików: %w", err)
	}
	defer rows.Close()

	byItem := map[string][]model.Asset{}
	for rows.Next() {
		as, err := scanAsset(rows)
		if err != nil {
			return nil, fmt.Errorf("store: odczyt pliku do eksportu: %w", err)
		}
		byItem[as.ItemID] = append(byItem[as.ItemID], *as)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iteracja po plikach przy eksporcie: %w", err)
	}
	return byItem, nil
}

// TimelineLimit chroni przed zwróceniem całej bazy w jednym żądaniu,
// gdyby katalog urósł ponad wszelkie oczekiwania.
const TimelineLimit = 2000

// Timeline zwraca itemy uporządkowane chronologicznie, bez stronicowania.
// Grupowaniem po latach i miesiącach zajmuje się interfejs — data jest
// w każdym itemie, więc nie ma sensu robić tego dwa razy.
func (s *Store) Timeline(ctx context.Context) ([]model.Item, error) {
	result, err := s.ListItems(ctx, ListFilter{
		Sort:  "happened_at",
		Order: "desc",
		Limit: TimelineLimit,
	})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

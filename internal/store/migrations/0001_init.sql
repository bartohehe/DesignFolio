-- Podstawowy schemat: itemy, assety, tagi.
--
-- Konwencje w całej bazie:
--   * identyfikatory to UUID w kolumnach TEXT,
--   * daty to TEXT w formacie RFC3339 UTC — sortują się leksykograficznie
--     i są czytelne, gdy ktoś zajrzy do bazy przez sqlite3,
--   * wartości logiczne to INTEGER 0/1 (SQLite nie ma typu BOOLEAN),
--   * CHECK-i pilnują enumów po stronie bazy, niezależnie od kodu Go.

CREATE TABLE items (
    id           TEXT PRIMARY KEY,
    title        TEXT NOT NULL,
    slug         TEXT NOT NULL UNIQUE,
    description  TEXT NOT NULL DEFAULT '',
    kind         TEXT NOT NULL CHECK (kind IN
                     ('project', 'drawing', 'print3d', 'photo', 'note', 'link', 'other')),
    status       TEXT NOT NULL DEFAULT 'wip' CHECK (status IN ('wip', 'done', 'archived')),
    favorite     INTEGER NOT NULL DEFAULT 0 CHECK (favorite IN (0, 1)),
    external_url TEXT,
    metadata     TEXT NOT NULL DEFAULT '{}',
    happened_at  TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE INDEX idx_items_kind ON items (kind);
CREATE INDEX idx_items_status ON items (status);
CREATE INDEX idx_items_created_at ON items (created_at DESC);
-- COALESCE w indeksie, bo dokładnie tak sortujemy oś czasu:
-- kiedy powstało, a gdy nie wiadomo — kiedy zostało dodane.
CREATE INDEX idx_items_happened_at ON items (COALESCE(happened_at, created_at) DESC);
-- Indeks częściowy: zajmuje tyle miejsca, ile jest ulubionych, a nie ile jest itemów.
CREATE INDEX idx_items_favorite ON items (id) WHERE favorite = 1;

CREATE TABLE assets (
    id         TEXT PRIMARY KEY,
    item_id    TEXT NOT NULL REFERENCES items (id) ON DELETE CASCADE,
    filename   TEXT NOT NULL,
    mime       TEXT NOT NULL,
    size       INTEGER NOT NULL,
    kind       TEXT NOT NULL CHECK (kind IN ('image', 'model', 'document', 'archive', 'other')),
    width      INTEGER,
    height     INTEGER,
    checksum   TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_cover   INTEGER NOT NULL DEFAULT 0 CHECK (is_cover IN (0, 1)),
    has_thumb  INTEGER NOT NULL DEFAULT 0 CHECK (has_thumb IN (0, 1)),
    created_at TEXT NOT NULL
);

CREATE INDEX idx_assets_item ON assets (item_id, sort_order);
CREATE INDEX idx_assets_checksum ON assets (checksum);
-- Unikalny indeks częściowy = "najwyżej jedna okładka na item",
-- pilnowane przez bazę, a nie przez poprawność kodu.
CREATE UNIQUE INDEX idx_assets_one_cover ON assets (item_id) WHERE is_cover = 1;

CREATE TABLE tags (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
);

CREATE TABLE item_tags (
    item_id TEXT NOT NULL REFERENCES items (id) ON DELETE CASCADE,
    tag_id  TEXT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (item_id, tag_id)
);

-- Klucz główny (item_id, tag_id) obsługuje szukanie tagów danego itemu.
-- Ten indeks obsługuje kierunek odwrotny: itemy danego tagu.
CREATE INDEX idx_item_tags_tag ON item_tags (tag_id);

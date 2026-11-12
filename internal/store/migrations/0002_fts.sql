-- Wyszukiwanie pełnotekstowe po tytule, opisie i tagach.
--
-- Tabela FTS5 trzyma własną kopię tekstu (nie jest "external content"),
-- a jej rowid celowo zrównujemy z items.rowid — dzięki temu wynik
-- wyszukiwania łączy się z itemami jednym JOIN-em, bez dodatkowej kolumny.
--
-- Do tabeli trafia tekst już sprowadzony do ASCII przez model.FoldDiacritics
-- (patrz internal/store/search.go). remove_diacritics zostaje jako druga linia
-- obrony, ale samo nie wystarcza: rozkłada ó, ż i ć, natomiast ł jest
-- w Unicode osobną literą i przechodzi nietknięte.
CREATE VIRTUAL TABLE items_fts USING fts5(
    title,
    description,
    tags,
    tokenize = 'unicode61 remove_diacritics 2'
);

-- Tabela zostaje pusta. Wypełnia ją kod Go: przy starcie, gdy wykryje
-- itemy bez wpisów w indeksie (Store.backfillIndexIfEmpty), a potem
-- na bieżąco funkcja reindexItem przy każdym zapisie.
--
-- Nie robimy tego w SQL-u z dwóch powodów: zwijanie polskich znaków wymaga
-- kodu, a triggery i tak nie ogarnęłyby agregacji tagów z tabeli łączącej.
-- Jedna funkcja w Go to jedno miejsce do czytania i jedno do debugowania.

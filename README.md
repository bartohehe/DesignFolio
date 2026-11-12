# DesignFolio

Prywatny, self-hostowany katalog rzeczy cyfrowych: projektów programistycznych,
rysunków, modeli i druków 3D, notatek, zdjęć i linków.

Aplikacja jest jednoosobowa i nie jest publikowana. Nie ma logowania, userów ani
panelu admina — wszystko dodaje się z poziomu UI, a serwer domyślnie słucha
wyłącznie na `127.0.0.1`.

Stan aplikacji to jeden katalog: `data/`. Skopiowanie go jest pełnym backupem.

---

## Szybki start (dev)

Potrzebne: **Go 1.25+** (tyle wymaga sterownik `modernc.org/sqlite`), **Node 20+**.
Dwa terminale.

Terminal 1 — API:

```bash
go run ./cmd/server
```

Terminal 2 — frontend z hot reloadem:

```bash
cd web && npm install && npm run dev
```

Otwórz **http://localhost:5173**. Vite proxuje `/api/*` na `127.0.0.1:8080`,
więc przeglądarka widzi jeden origin i nie ma mowy o CORS-ie.

## Uruchomienie produkcyjne

```bash
docker compose up --build
```

Aplikacja stoi pod **http://127.0.0.1:8080** — API i frontend z jednej binarki.
Dane lądują w `./data` (wolumen podpięty do kontenera).

Bez Dockera, jedną binarką:

```bash
cd web && npm ci && npm run build && cd .. && go build -o designfolio ./cmd/server
```

Frontend jest wtedy wbudowany w plik wykonywalny przez `go:embed` — na serwer
kopiuje się jeden plik.

## Konfiguracja

Wszystko przez zmienne środowiskowe albo plik `.env` (skopiuj z `.env.example`).
Zmienne środowiskowe mają pierwszeństwo przed `.env`.

| Zmienna         | Domyślnie        | Opis                                                       |
| --------------- | ---------------- | ---------------------------------------------------------- |
| `APP_ADDR`      | `127.0.0.1:8080` | Adres nasłuchu. `0.0.0.0:8080` = widoczne w sieci lokalnej. |
| `DATA_DIR`      | `./data`         | Katalog z bazą i plikami.                                   |
| `APP_TOKEN`     | _(puste)_        | Puste = brak autoryzacji. Niepuste = zapisy wymagają `X-App-Token`. |
| `MAX_UPLOAD_MB` | `512`            | Limit rozmiaru jednego uploadu.                             |
| `APP_ENV`       | `prod`           | `dev` = logi tekstowe i poziom debug.                       |

Flaga `-addr` nadpisuje `APP_ADDR`, flaga `-env` wskazuje inny plik `.env`.

> **Wystawiasz na sieć domową?** Ustaw `APP_TOKEN` i zmień mapowanie portu
> w `docker-compose.yml` z `127.0.0.1:8080:8080` na `8080:8080`.

Gdy `APP_TOKEN` jest ustawiony, przeglądarka musi go znać, żeby zapisywać.
Nie trzeba niczego konfigurować z góry: pierwszy odrzucony zapis sam otwiera
okno „Token zapisu", a po wpisaniu wartości wystarczy powtórzyć operację.
Token zostaje w `localStorage` tej przeglądarki; zmienić albo usunąć go można
przyciskiem „Token zapisu" na dole marginesu.

## Struktura katalogów

```
cmd/server/          punkt wejścia: config, wiring, graceful shutdown
internal/
  config/            wczytywanie .env i zmiennych środowiskowych
  http/              router chi, middleware, handlery (pakiet `httpapi`)
  store/             SQLite: migracje, zapytania — cała logika danych
  media/             zapis plików na dysk, miniatury, rozpoznawanie typów
  model/             struktury domenowe: Item, Asset, Tag
web/
  embed.go           wbudowuje web/dist w binarkę (pakiet Go w katalogu frontendu)
  src/
    api/             klient HTTP i typy odpowiedzi (bez `any`)
    components/      komponenty UI
    routes/          widoki podpięte pod React Router
    styles/tokens.css  JEDYNE miejsce z wartościami designu
data/                baza + pliki (gitignored) — to się backupuje
scripts/             skrypty backupu
```

Zasada podziału: handlery w `internal/http` tylko parsują żądanie i wołają
`store`/`media`. Zero SQL w warstwie HTTP, zero HTTP w warstwie danych.

### Dlaczego pakiet w `internal/http` nazywa się `httpapi`

Gdyby nazywał się `http`, każdy plik importujący `net/http` musiałby go
aliasować. Go pozwala, żeby nazwa pakietu różniła się od nazwy katalogu.

## API

| Metoda | Ścieżka               | Opis                                              |
| ------ | --------------------- | ------------------------------------------------- |
| GET    | `/api/health`         | status serwera, wersja, stan bazy                 |
| GET    | `/api/items`          | lista z filtrowaniem i stronicowaniem             |
| POST   | `/api/items`          | nowy item (tagi tworzone w locie)                 |
| GET    | `/api/items/{idOrSlug}` | pojedynczy item po id **albo** slugu            |
| PATCH  | `/api/items/{id}`     | częściowa aktualizacja                            |
| DELETE | `/api/items/{id}`     | usunięcie (204)                                   |
| POST   | `/api/items/{id}/assets` | wrzucenie plików (multipart, wiele naraz)      |
| PATCH  | `/api/items/{id}/assets` | kolejność plików i wybór okładki               |
| DELETE | `/api/assets/{id}`    | usunięcie pliku (rekord + dysk)                   |
| GET    | `/api/files/{id}/{wariant}` | treść pliku: `original`, `thumb`, `preview` |
| GET    | `/api/tags`           | wszystkie tagi z licznikami użyć                  |
| GET    | `/api/facets`         | liczniki per rodzaj i stan — zasilają indeks w UI |
| GET    | `/api/timeline`       | wszystko chronologicznie, bez stronicowania       |
| GET    | `/api/export`         | cały katalog jako jeden JSON (załącznik)          |

Parametry `GET /api/items`:

| Parametr   | Przykład                        | Uwagi                                             |
| ---------- | ------------------------------- | ------------------------------------------------- |
| `kind`     | `?kind=photo,drawing`           | wiele wartości: po przecinku lub powtórzony klucz  |
| `status`   | `?status=done`                  | `wip` \| `done` \| `archived`                      |
| `tag`      | `?tag=go,cli`                   | **AND** — item musi mieć wszystkie podane tagi     |
| `favorite` | `?favorite=true`                |                                                   |
| `q`        | `?q=sluchawki`                  | wyszukiwanie pełnotekstowe (FTS5)                 |
| `sort`     | `?sort=title`                   | `relevance` (przy `q`), `happened_at` (domyślne), `created_at`, `updated_at`, `title` |
| `order`    | `?order=asc`                    | domyślnie `desc`                                  |
| `limit`    | `?limit=20`                     | domyślnie 60, maksymalnie 200                     |
| `offset`   | `?offset=60`                    |                                                   |

Sortowanie po `happened_at` używa `COALESCE(happened_at, created_at)` — item bez
ręcznie ustawionej daty trafia tam, kiedy został dodany.

### PATCH: trzy stany pola

`PATCH` rozróżnia pole nieobecne od jawnego `null`:

```jsonc
{}                          // nie ruszaj niczego
{ "external_url": null }    // wyczyść pole
{ "external_url": "https://…" } // ustaw wartość
```

Odpowiada za to typ generyczny `model.Optional[T]`
([internal/model/optional.go](internal/model/optional.go)).

Kolejne endpointy (upload, wyszukiwarka, eksport) dochodzą w następnych etapach.

Błędy zawsze mają ten sam kształt:

```json
{ "error": { "code": "not_found", "message": "nie ma takiego itemu" } }
```

Kody: `bad_request`, `not_found`, `conflict`, `unauthorized`,
`payload_too_large`, `unsupported_media_type`, `internal`.

## Widoki

| Adres                   | Co to jest                                            |
| ----------------------- | ----------------------------------------------------- |
| `/`                     | arkusz — masonry z indeksem filtrów na marginesie      |
| `/nowa`                 | formularz nowej rzeczy                                 |
| `/rzecz/{slug}`         | widok szczegółów: opis, podgląd 3D, pliki, metadane    |
| `/rzecz/{slug}/edycja`  | edycja (pola, nie pliki)                               |
| `/os-czasu`             | oś czasu — wszystko pogrupowane po latach i miesiącach |
| `/tagi`                 | wszystkie tagi, wielkość napisu niesie liczbę użyć     |

Filtry arkusza siedzą w adresie (`/?kind=print3d&favorite=true`), więc działa
przycisk Wstecz, a widok da się odświeżyć albo zapisać w zakładkach.

## Design

Dark-first, ciepły grafit zamiast czerni, jeden akcent (bursztyn) używany
oszczędnie. Typografia niesie charakter: Space Grotesk na nagłówki, Inter na
treść, JetBrains Mono na metadane, tagi i liczby. Fonty są self-hostowane
(paczki npm), więc aplikacja działa bez internetu.

Wszystkie kolory, rozmiary, promienie i czasy animacji są zdefiniowane
w `web/src/styles/tokens.css` (Tailwind v4 konfiguruje się w CSS, nie w JS).
W komponentach używamy wyłącznie klas zbudowanych z tych tokenów.

Pełny opis systemu — kolor, typografia, układ, ruch, stany i zasady, których
łatwo nie dochować — jest w [DESIGN.md](DESIGN.md). Kontekst produktowy
(kto, po co, jakie decyzje są wiążące) w [PRODUCT.md](PRODUCT.md).

### Metadane zależne od rodzaju

Backend trzyma `metadata` jako dowolny JSON, a schemat nazwanych pól mieszka
we froncie w [web/src/lib/kinds.ts](web/src/lib/kinds.ts) — dołożenie pola nie
wymaga migracji bazy. Formularz zachowuje klucze spoza schematu bieżącego
rodzaju: item mógł kiedyś być drukiem 3D, a dziś być „inne", i te dane nie mogą
zniknąć przy edycji.

## Baza danych

SQLite przez `modernc.org/sqlite` — sterownik napisany w czystym Go, więc binarka
buduje się z `CGO_ENABLED=0` i nie wymaga kompilatora C.

Migracje to ponumerowane pliki w [internal/store/migrations/](internal/store/migrations/),
wbudowane w binarkę przez `go:embed` i wykonywane przy starcie. Runner zapisuje
wykonane wersje w tabeli `schema_migrations` i puszcza każdą migrację w transakcji.
Żeby dodać kolejną, wystarczy nowy plik `0002_cokolwiek.sql`.

PRAGMA są ustawiane w connection stringu, a nie zapytaniem po otwarciu —
`*sql.DB` to pula połączeń, więc pojedyncze `PRAGMA` dotyczyłoby tylko jednego
z nich. Włączone: `journal_mode=WAL`, `foreign_keys=1` (SQLite domyślnie **ignoruje**
klucze obce, bez tego `ON DELETE CASCADE` nie działa), `busy_timeout=5000`,
`synchronous=NORMAL`.

Konwencje: identyfikatory to UUID w kolumnach `TEXT`, daty to `TEXT` w RFC3339 UTC
ze stałą precyzją (sortują się leksykograficznie), wartości logiczne to `INTEGER` 0/1.

## Pliki

Układ na dysku jest przewidywalny i czytelny bez aplikacji:

```
data/files/<asset-id>/<oryginalna-nazwa>   ← oryginał, zawsze zachowany
data/files/<asset-id>/thumb.jpg|.png       ← dłuższy bok 400 px
data/files/<asset-id>/preview.jpg|.png     ← dłuższy bok 1600 px
```

Miniatury powstają przy uploadzie, tylko dla obrazów. Format wynika ze źródła:
PNG, gdy oryginał mógł mieć przezroczystość (png/gif/webp), JPEG w pozostałych
przypadkach. Obrazu mniejszego niż próg **nie powiększamy** — rozmycie
z interpolacji jest gorsze niż mała miniatura.

Upload idzie strumieniem (`r.MultipartReader`), nie przez `ParseMultipartForm` —
ten drugi wczytałby całe żądanie do pamięci, a modele 3D i serie zdjęć potrafią
mieć setki megabajtów. Plik ląduje najpierw pod nazwą tymczasową i dopiero udany
transfer kończy się `os.Rename`, więc przerwana wysyłka nie zostawia obciętego
pliku pod właściwą nazwą.

Deduplikacja działa po sumie SHA-256 w obrębie itemu: to samo zdjęcie wrzucone
drugi raz nie zakłada nowego rekordu.

Kolejność plików zmienia się przeciąganiem kafelków na stronie rzeczy
(natywne HTML5 drag & drop, bez biblioteki) albo strzałkami — te drugie
działają z klawiatury.

## Wyszukiwanie

SQLite FTS5 po tytule, opisie i tagach. Indeks utrzymuje `reindexItem`
w [internal/store/search.go](internal/store/search.go), wołane w tej samej
transakcji co zapis — nie ma triggerów, bo i tak nie ogarnęłyby agregacji tagów.

**Polskie znaki.** Do indeksu trafia tekst sprowadzony do ASCII
(`model.FoldDiacritics`) i tak samo traktowana jest wpisana fraza, więc
„słuchawki" znajdzie się po „sluchawki" i odwrotnie. Sam `remove_diacritics`
w tokenizerze nie wystarcza: rozkłada ó, ż i ć, ale **ł jest w Unicode osobną
literą** i przechodzi nietknięte.

Fraza użytkownika nie trafia do `MATCH` surowa — `"`, `*`, `^`, `-` oraz
AND/OR/NOT są w FTS5 operatorami. Rozbijamy ją na słowa, każde cytujemy
i dopisujemy `*` (dopasowanie przedrostkowe). Wszystkie słowa muszą wystąpić.

## Backup

```bash
./scripts/backup.sh
```

```powershell
.\scripts\backup.ps1 -Destination D:\kopie -Keep 10
```

Backup to kopia katalogu `data/` — nic więcej. Odtworzenie polega na skopiowaniu
go z powrotem; sprawdzone: serwer wystartował na katalogu z kopii i zobaczył
komplet danych.

> **Uwaga.** W trybie WAL baza to trzy pliki (`.db`, `.db-wal`, `.db-shm`).
> Skrypt kopiuje wszystkie, ale dla gwarancji spójności zatrzymaj aplikację
> przed backupem: `docker compose stop`. Skrypt ostrzega, gdy wykryje
> działający serwer.

Eksport metadanych do jednego pliku JSON jest osobno, pod `/api/export`
(link „Eksport JSON" na dole marginesu).

## Testy

```bash
go test ./...
```

Testy handlerów jadą przez prawdziwy router i prawdziwą bazę SQLite w pliku
tymczasowym — bez mocków, razem z migracjami i CHECK-ami. Baza `:memory:`
nie nadaje się do tego celu: jest prywatna dla pojedynczego połączenia,
a `*sql.DB` to pula.

```bash
cd web && npm run typecheck
```

## Stan prac

- [x] **1** — szkielet: Go serwuje `/api/health`, React + Vite z hot reloadem, Docker
- [x] **2** — baza, migracje, model `Item`, CRUD API, testy handlerów
- [x] **3** — lista itemów, widok szczegółów, formularz dodawania
- [x] **4** — upload plików, miniatury, galeria
- [x] **5** — tagi, filtry, wyszukiwarka FTS5
- [x] **6** — podgląd 3D, timeline, eksport i backup
- [x] **7** — przejście po designie

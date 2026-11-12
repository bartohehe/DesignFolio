# Design

Spisane ze zbudowanego interfejsu (etapy 1–7), nie z zamiarów. Źródłem prawdy dla
wartości jest [web/src/styles/tokens.css](web/src/styles/tokens.css) — ten dokument
opisuje, jak ich używać.

## Świat wizualny

**Arkusz stykowy z indeksem na marginesie.** Katalog jest jedną taflą klatek,
a nie siatką jednakowych kart pod poziomym paskiem filtrów. Margines po lewej
niesie indeks: rodzaje i stany z licznikami. Klatki rozdziela tło i odstęp —
w całym interfejsie nie ma ani jednego cienia i ani jednej ramki wokół karty.

Ulubione są zakreślone otwartym pierścieniem przechylonym o 12°, jak kółkiem
kredką na odbitce. To jedyne miejsce, gdzie akcent pojawia się jako rysunek,
a nie jako kolor tekstu czy tła.

## Kolor

Skala szarości jest ciepła (odcień 60–70 w OKLCH), nie neutralna — zdjęcia
i skany rysunków wyglądają na niej naturalnie.

| Token | Rola |
| --- | --- |
| `canvas` | tło strony |
| `surface` | klatka tekstowa, panel, chip |
| `raised` | pole formularza, hover |
| `line` | włoskowe dzielniki, obrys |
| `ink` | tekst główny |
| `muted` | tekst drugorzędny, opisy |
| `faint` | etykiety mono, liczniki, podpowiedzi |
| `accent` | jeden bursztyn |
| `accent-dim` | hover na wypełnieniu akcentu |
| `danger`, `success` | stany |

**Strategia: powściągliwa.** Neutralna baza plus jeden akcent. Bursztyn ma
dokładnie cztery zastosowania: aktywny filtr w indeksie, przycisk główny,
zaznaczenie ulubionego, pierścień focusa. Nic więcej — jeśli akcent pojawia
się gdzieś piąty raz, coś poszło nie tak.

**Kontrast jest mierzony, nie szacowany.** Wartości `ink`, `muted` i `faint`
dobrano tak, żeby najtrudniejszy przypadek — drobny tekst mono na `surface` —
przechodził AA. Zmierzone na zbudowanej stronie: ink 14,1:1, muted 6,8:1,
faint 5,5:1. Pełne przejście po wszystkich węzłach tekstowych arkusza,
widoku szczegółów i formularza: zero naruszeń.

## Typografia

Trzy role, każda ze swoim zadaniem:

- **Space Grotesk** — nagłówki i tytuły klatek. Jedyny nośnik charakteru.
- **Inter** — treść, opisy, etykiety kontrolek.
- **JetBrains Mono** — metadane, tagi, liczby, nazwy pól. Nigdy dla przycisków
  ani zdań: mono jest tu językiem danych, nie kostiumem „technicznego" wyglądu.

Fonty są self-hostowane paczkami `@fontsource-variable/*` — aplikacja działa
bez internetu.

Skala 1.25, lista zamknięta: `2xs 0.694rem` → `4xl 3.052rem`. Rozmiar spoza
listy oznacza błąd, nie wyjątek. Nagłówki od `2xl` w górę mają ujemny tracking.
Miara tekstu ciągłego: `max-w-prose`.

**Tytuł zawsze prowadzi.** Rodzaj i stan idą pod tytułem jako jego opis —
w projekcie nie ma etykiet zapowiadających nagłówek.

## Układ

Asymetryczny, bez centrowanej kolumny. Na `lg` i wyżej:
`grid-cols-[16rem_minmax(0,1fr)]` — margines 16 rem, arkusz zajmuje resztę
z prawym marginesem `pr-16`. Poniżej `lg` margines przenosi się nad treść
i zwija do rozwijanego indeksu; nic z funkcji nie znika.

**Masonry** liczy kolumny z rzeczywistej szerokości kontenera (`ResizeObserver`,
docelowa kolumna 340 px, maksymalnie 4), a itemy rozdaje po kolei `i % kolumny`.
Kolumny CSS odpadły, bo układają treść od góry do dołu — przy sortowaniu
od najnowszych trzy pierwsze rzeczy wylądowałyby jedna pod drugą.

Widok szczegółów rozdziela się na dwie kolumny dopiero od `xl`. Przy 1024 px
kolumna opisu schodziła do ~47 znaków w wierszu, a wygodna miara to 65–75.

## Klatki

Dwa warianty, wybierane obecnością okładki:

- **z okładką** — zdjęcie prowadzi, proporcja z prawdziwych wymiarów pliku
  przycięta do zakresu 0,62–1,6, podpis leży pod nim wprost na tle arkusza;
- **bez okładki** — mniejsza klatka tekstowa na własnym tle `surface`.

Item bez zdjęcia nie dostaje placeholdera. Notatka i link to pełnoprawne rzeczy,
a nie uszkodzone obrazki.

Na klatce jest tytuł, rodzaj i fragment opisu. Tagów i dat tam nie ma —
to świadoma decyzja o gęstości, nie przeoczenie.

## Ruch

Jedna krzywa (`--ease-out`), dwa czasy: 150 ms dla koloru, 200 ms dla ruchu.
Dwa utility kodują tę zasadę: `transition-ui` (wyłącznie `transform`
i `opacity`) oraz `transition-tint` (kolory). Zero bounce, zero sprężyn.

Ruch jest jeden na widok: klatka tekstowa unosi się o 2 px, okładka skaluje
o 2%. Przy dociąganiu wyników arkusz przygasa do 60% zamiast znikać i wracać
jako szkielet.

`prefers-reduced-motion: reduce` ścina wszystkie animacje globalnie.

## Ikony

Własny zestaw rysowany w siatce 24×24, jedną grubością kreski (1,5) i jednym
zakończeniem ([web/src/components/icons.tsx](web/src/components/icons.tsx)).
Żadnych emoji i żadnej biblioteki. Ikony rodzajów są metaforami narzędzia,
nie obrazkami rzeczy: nawiasy = kod, stalówka = rysunek, sześcian warstwowy =
druk, kadr z horyzontem = zdjęcie.

Ikona przy etykiecie tekstowej ma `aria-hidden`; ikona jako jedyna treść
przycisku wymaga `aria-label` albo `sr-only`.

## Stany

Zaprojektowane, nie doklejone:

- **ładowanie** — szkielet powtarza rytm arkusza (klatki o różnych, ale stałych
  wysokościach), nie jeden szary prostokąt;
- **pusto** — rysunek pustych klatek, wyjaśnienie i akcja główna; osobna wersja
  dla „katalog jest pusty" i dla „filtry nic nie zwróciły";
- **błąd** — nazywa problem i drogę wyjścia, ze szczegółem technicznym włącznie
  (jedynym użytkownikiem jest osoba, która to pisze);
- **filtr bez wyników** — wiersz indeksu wygaszony i nieaktywny, chyba że
  akurat jest włączony.

## Widoki

Cztery, wszystkie w tej samej gramatyce:

- **Arkusz** (`/`) — masonry z indeksem na marginesie. Wyszukiwarka nad
  indeksem, trafienia podświetlone `<mark>` w tytule i fragmencie opisu.
- **Rzecz** (`/rzecz/{slug}`) — tytuł prowadzi, opis w mierze `max-w-prose`,
  podgląd 3D dla modeli, półka plików, metadane w kolumnie obok (od `xl`).
- **Oś czasu** (`/os-czasu`) — lata jako duże cyfry mono, miesiące w kolumnie
  po lewej, wpisy jako lista do przebiegnięcia wzrokiem. Miniatury są tu
  drobnym znacznikiem, a nie drugą siatką.
- **Tagi** (`/tagi`) — wielkość napisu niesie liczbę użyć, ale **skokowo**,
  czterema stopniami ze skali typograficznej. Płynnie wyliczony rozmiar
  byłby wartością spoza systemu.

## Podgląd 3D

three.js ładowany przez `React.lazy` — schodzi do osobnej paczki (ok. 560 kB)
i pobiera się dopiero przy rzeczy, która ma model `.stl` albo `.3mf`. Model
dostaje ten sam ciepły bursztyn co akcent interfejsu, oświetlenie kierunkowe
plus wypełnienie (bez niego spody są czarne), a kamera sama kadruje obiekt
niezależnie od skali pliku.

## Dostępność

- Kontrast AA na całym interfejsie, zmierzony programowo.
- Focus: `outline` 2 px w kolorze akcentu z odstępem 2 px, przez `:focus-visible`
  (nie miga przy kliknięciu myszą). `outline` zamiast `box-shadow`, bo nie da się
  go przyciąć przez `overflow: hidden`.
- Filtry to `<button aria-pressed>`, nie linki udające przełączniki.
- Stan filtrów mieszka w URL — działa Wstecz i odświeżenie.
- Usuwanie potwierdzane dwustopniowo w miejscu, bez okna modalnego.
- Kolejność plików: przeciąganie myszą ma zawsze odpowiednik klawiaturowy
  (strzałki na kafelku). Przeciągany kafelek przygasa do 40%, cel unosi się
  o 4 px i dostaje obrys w kolorze akcentu.
- Okna modalne (podgląd, token) to natywny `<dialog>` z `m-auto` — preflight
  Tailwinda zeruje margines, przez co bez tej klasy modal przykleja się do
  lewego górnego rogu. Escape obsługiwany jawnie, bo nie każdy silnik zgłasza
  natywne `cancel`.
- Natywne `<select>` i `<input type="date">` zamiast własnych rozwijań.
- Interfejs po polsku; komunikaty nazywają problem i wyjście.

## Zasady, które łatwo złamać

1. **Żadnych wartości spoza tokenów.** Ani jednego `bg-[#…]`, `text-[13px]`,
   `style={{ margin: 12 }}`.
2. **Dwie klasy koloru tekstu na jednym elemencie to błąd.** Mają tę samą wagę,
   więc o wyniku decyduje kolejność w arkuszu, nie w atrybucie `class`.
   Stan włączony dostaje własny wariant przycisku (`mark`), nie doklejaną klasę.
3. **Karta w karcie nigdy.** Klatka jest płaska.
4. **Mono tylko dla danych.** Przyciski i zdania idą Interem.
5. **Nowy rozmiar tekstu = zmiana skali w tokenach**, nie wartość w komponencie.

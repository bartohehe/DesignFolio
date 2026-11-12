# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Go 1.25 (chi, database/sql, modernc.org/sqlite) serwuje API i wbudowany frontend.
React 18 + TypeScript + Vite + Tailwind v4, TanStack Query, React Router.
Uruchomienie: `docker compose up` (jeden kontener) albo tryb dev z osobnym Vite i proxy na API.
Stack pinowany przez użytkownika, nie do zmiany bez pytania.

## Users

Jeden użytkownik — właściciel katalogu. Programista, który jednocześnie rysuje,
drukuje w 3D, robi zdjęcia i pisze notatki. Uczy się Go przy okazji tego projektu.
Pracuje na Windowsie (dev) i na małym serwerze z Linuxem (prod).
Nie ma innych użytkowników, nie ma logowania, nie ma ról.

## Product Purpose

Prywatny katalog wszystkiego, co użytkownik zrobił cyfrowo: projektów programistycznych,
rysunków, modeli i druków 3D, notatek, zdjęć i linków. Aplikacja nie jest publikowana —
chodzi na `127.0.0.1` albo w sieci domowej.

Sukces: rzecz sprzed dwóch lat da się znaleźć w kilka sekund, a odłożenie nowej rzeczy
jest na tyle bezbolesne, że użytkownik faktycznie to robi zamiast zostawiać pliki
w losowych katalogach.

## Positioning

Nie portfolio do pokazywania i nie menedżer plików. To osobisty rejestr twórczy,
w którym druk 3D, commit i szkic ołówkiem leżą obok siebie jako równorzędne rzeczy,
opisane wspólnym zestawem pól plus metadanymi właściwymi dla swojego rodzaju.

## Operating Context

- Dodawanie i przeglądanie w mniej więcej równych proporcjach — obie ścieżki
  muszą być pierwszorzędne.
- Pliki (zdjęcia, modele .stl/.3mf, dokumenty) leżą na dysku w `data/files/`,
  metadane w SQLite. Backup to skopiowanie katalogu `data/`.
- Katalog rośnie powoli i nieregularnie: kilka rzeczy w tygodniu, czasem seria
  po jednej sesji rysowania albo druku.
- Sporo itemów nigdy nie dostanie zdjęcia (notatki, linki) — puste kafelki
  nie mogą wyglądać na błąd.

## Capabilities and Constraints

**Model danych.** `Item` (id, title, slug, description w markdownie, kind, status,
created_at, updated_at, happened_at, favorite, external_url, metadata JSON),
`Asset` (plik przypięty do itemu, miniatury 400/1600, okładka, kolejność),
`Tag` + relacja wiele-do-wielu, tagi tworzone w locie.

**Kind** (7): `project`, `drawing`, `print3d`, `photo`, `note`, `link`, `other`.
**Status** (3): `wip`, `done`, `archived`.

**Metadata per kind — gotowe, nazwane pola**, nie generyczny edytor klucz-wartość.
Decyzja użytkownika: spójność danych ważniejsza niż pełna swoboda.
Przykłady: `project` → język, repo url, stack; `print3d` → drukarka, materiał,
czas druku, warstwa, ustawienia slicera; `drawing` → narzędzie/medium, czas pracy.

**Kafelek w siatce** pokazuje: tytuł, kind, fragment opisu oraz okładkę, jeśli item ma
przypięte zdjęcie. Item bez zdjęcia dostaje mniejszy kafelek tekstowy — nie placeholder.
Tagi i daty nie wchodzą na kafelek (decyzja użytkownika).

**Bezpieczeństwo.** Bez systemu userów. Opcjonalny `APP_TOKEN` sprawdzany tylko
dla metod zapisu; pusty = wyłączony.

**Stan prac.** Etapy 1–2 gotowe (szkielet, baza, CRUD API, testy). Etap 3 to lista,
widok szczegółów i formularz. Upload plików, wyszukiwarka FTS, podgląd 3D, timeline
i eksport są zaplanowane na etapy 4–6.

## Brand Commitments

Nazwa: **DesignFolio**.

Użytkownik zapinał kierunek wizualny w briefie i potwierdził go w rundzie pytań —
te decyzje są wiążące:

- dark-first, tło ciepły grafit, nie czerń;
- jeden mocny kolor akcentu (ciepły bursztyn), używany oszczędnie;
- typografia jako główny nośnik charakteru: Space Grotesk (nagłówki),
  Inter (treść), JetBrains Mono (metadane, tagi, liczby);
- layout asymetryczny, edytorski, masonry o różnych proporcjach — nie sztywne kwadraty;
- karty bez ciężkich cieni i grubych ramek: separacja przez tło, odstęp i typografię;
- mikro-interakcje 150–200 ms, wyłącznie transform/opacity, zero bounce;
- stany puste, ładowania i błędów zaprojektowane, nie doklejone;
- tokeny designu wyłącznie w `web/src/styles/tokens.css`, zero hardkodowanych wartości.

## Evidence on Hand

Katalog jest pusty — użytkownik nie dostarczył jeszcze żadnych realnych itemów,
zdjęć ani modeli. Dane pokazowe w trakcie budowy są syntetyczne i muszą być
oznaczone jako takie; nie wolno ich podawać za dorobek użytkownika.

Nie istnieją: logo, gotowe assety graficzne, treści marketingowe.
Nie wolno ich zmyślać ani sugerować, że istnieją.

## Product Principles

1. **Jednoosobowość jest cechą, nie ograniczeniem.** Zero ekranów logowania,
   zero pustych paneli admina, zero „zaproś zespół". Każdy piksel służy jednej osobie.
2. **Odłożenie rzeczy musi być tańsze niż niezałożenie jej.** Tytuł i kind wystarczą;
   reszta jest opcjonalna i można ją uzupełnić później.
3. **Rzeczy bez obrazka są równie ważne.** Notatka i link to pełnoprawne itemy,
   a nie uszkodzone zdjęcia.
4. **Katalog ma przetrwać aplikację.** Pliki zostają na dysku pod czytelnymi ścieżkami,
   baza jest jednym plikiem, eksport zwraca wszystko jako JSON.
5. **Kod jest materiałem do nauki Go.** Czytelność wygrywa ze sprytem;
   nieoczywiste konstrukcje mają komentarz.

## Accessibility & Inclusion

Kontrast minimum WCAG AA. Pełna obsługa klawiatury z widocznym focusem.
Poszanowanie `prefers-reduced-motion`. Interfejs po polsku.

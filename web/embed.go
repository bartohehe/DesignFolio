// Package web wbudowuje zbudowany frontend w binarkę serwera.
//
// Ten plik leży w katalogu web/, bo dyrektywa go:embed nie potrafi wyjść
// katalog wyżej (żadnych ".." we wzorcu). Trzymanie pakietu Go obok kodu
// frontendu jest tu najprostszym rozwiązaniem.
package web

import (
	"embed"
	"io/fs"
)

// Wzorzec "all:dist" zamiast "dist" ma znaczenie: bez prefiksu all: embed
// pomija pliki zaczynające się od kropki lub podkreślnika — a repo zawiera
// dist/.gitkeep po to, żeby ten embed w ogóle się kompilował przed pierwszym
// `npm run build`.
//
//go:embed all:dist
var dist embed.FS

// Dist zwraca system plików z zawartością web/dist, przycięty tak, że
// "index.html" leży w jego korzeniu. Drugi wynik mówi, czy frontend
// faktycznie został zbudowany — jeśli nie, serwer pokaże instrukcję
// zamiast pustej strony.
func Dist() (fs.FS, bool) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}
	return sub, true
}

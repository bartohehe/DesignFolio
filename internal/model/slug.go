package model

import "strings"

// diacritics mapuje znaki diakrytyczne na odpowiedniki ASCII.
// Robimy to ręczną tablicą zamiast przez golang.org/x/text/unicode/norm,
// bo lista realnie potrzebnych znaków jest krótka, a zależność byłaby duża.
var diacritics = map[rune]string{
	'ą': "a", 'ć': "c", 'ę': "e", 'ł': "l", 'ń': "n",
	'ó': "o", 'ś': "s", 'ź': "z", 'ż': "z",
	'á': "a", 'à': "a", 'ä': "a", 'â': "a", 'å': "a", 'ã': "a",
	'é': "e", 'è': "e", 'ë': "e", 'ê': "e",
	'í': "i", 'ì': "i", 'ï': "i", 'î': "i",
	'ò': "o", 'ö': "o", 'ô': "o", 'õ': "o", 'ø': "o",
	'ú': "u", 'ù': "u", 'ü': "u", 'û': "u",
	'ý': "y", 'ÿ': "y", 'ñ': "n", 'ç': "c",
	'š': "s", 'č': "c", 'ř': "r", 'ž': "z", 'ě': "e", 'ů': "u", 'ď': "d", 'ť': "t",
	'ß': "ss", 'æ': "ae", 'œ': "oe", 'đ': "d",
}

// MaxSlugLen ogranicza długość sluga, żeby URL-e pozostały czytelne.
const MaxSlugLen = 80

// FoldDiacritics sprowadza tekst do małych liter bez znaków diakrytycznych,
// zachowując spacje i interpunkcję. "Ołówek 2B" → "olowek 2b".
//
// Potrzebne w wyszukiwarce. Wbudowany w SQLite tokenizer unicode61 z opcją
// remove_diacritics radzi sobie z ó, ż czy ć (rozkładają się na literę bazową
// plus znak łączący), ale NIE z ł — to w Unicode osobna litera, a nie "l ze
// znakiem". Bez tej funkcji "słuchawki" nie znalazłyby się po wpisaniu
// "sluchawki", co w polskim tekście jest częstym przypadkiem.
func FoldDiacritics(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range strings.ToLower(s) {
		if repl, ok := diacritics[r]; ok {
			b.WriteString(repl)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Slugify zamienia dowolny tekst na fragment URL-a: małe litery ASCII,
// cyfry i myślniki. "Rysunek — Wieża Eiffla #2" → "rysunek-wieza-eiffla-2".
//
// Zwraca pusty string, jeśli po oczyszczeniu nic nie zostało (np. sam emoji).
// Wołający musi wtedy podstawić własną wartość zastępczą.
func Slugify(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	lastDash := true // true na starcie, żeby slug nie zaczął się od myślnika

	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if repl, ok := diacritics[r]; ok {
				b.WriteString(repl)
				lastDash = false
				continue
			}
			// Każdy inny znak (spacja, myślnik, interpunkcja, emoji)
			// staje się jednym myślnikiem — ale nigdy dwoma pod rząd.
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	slug := strings.Trim(b.String(), "-")
	if len(slug) > MaxSlugLen {
		slug = strings.Trim(slug[:MaxSlugLen], "-")
	}
	return slug
}

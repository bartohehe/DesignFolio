package model_test

import (
	"strings"
	"testing"

	"designfolio/internal/model"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Prosty tytuł", "prosty-tytul"},
		{"ŻÓŁĆ gęślą jaźń", "zolc-gesla-jazn"},
		{"Wieża Eiffla — szkic #2", "wieza-eiffla-szkic-2"},
		{"  spacje   dookoła  ", "spacje-dookola"},
		{"wiele---myślników", "wiele-myslnikow"},
		{"CamelCase123", "camelcase123"},
		{"Grüße aus München", "grusse-aus-munchen"},
		{"---", ""},
		{"", ""},
		{"🎨", ""},
		{"a/b\\c", "a-b-c"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := model.Slugify(tt.in); got != tt.want {
				t.Errorf("Slugify(%q) = %q, oczekiwano %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSlugifyTruncatesAndTrims(t *testing.T) {
	long := strings.Repeat("bardzo ", 40) // znacznie ponad limit

	got := model.Slugify(long)
	if len(got) > model.MaxSlugLen {
		t.Errorf("długość = %d, oczekiwano najwyżej %d", len(got), model.MaxSlugLen)
	}
	// Przycięcie nie może zostawić myślnika na końcu.
	if strings.HasSuffix(got, "-") || strings.HasPrefix(got, "-") {
		t.Errorf("slug %q ma myślnik na brzegu", got)
	}
}

func TestNormalizeTags(t *testing.T) {
	got := model.NormalizeTags([]string{"  Go  ", "go", "GO", "", "   ", "SQLite"})

	if len(got) != 2 {
		t.Fatalf("tagi = %v, oczekiwano dwóch", got)
	}
	// Zachowana jest pisownia i kolejność pierwszego wystąpienia.
	if got[0] != "Go" || got[1] != "SQLite" {
		t.Errorf("tagi = %v, oczekiwano [Go SQLite]", got)
	}
}

// Package model trzyma typy domenowe i ich walidację.
// Nie zna ani SQL-a, ani HTTP — dzięki temu te same struktury krążą
// między warstwami bez tłumaczenia z jednej reprezentacji na drugą.
package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Kind mówi, czym jest dana rzecz w katalogu.
//
// To zwykły string opakowany we własny typ — Go nie ma enumów, ale taki
// "named type" sprawia, że kompilator nie pozwoli podstawić dowolnego stringa
// tam, gdzie oczekiwany jest Kind.
type Kind string

const (
	KindProject Kind = "project"
	KindDrawing Kind = "drawing"
	KindPrint3D Kind = "print3d"
	KindPhoto   Kind = "photo"
	KindNote    Kind = "note"
	KindLink    Kind = "link"
	KindOther   Kind = "other"
)

// AllKinds zwraca wszystkie dozwolone wartości — używane przy walidacji
// i przez frontend do zbudowania filtrów.
func AllKinds() []Kind {
	return []Kind{KindProject, KindDrawing, KindPrint3D, KindPhoto, KindNote, KindLink, KindOther}
}

func (k Kind) Valid() bool {
	for _, v := range AllKinds() {
		if k == v {
			return true
		}
	}
	return false
}

// Status to etap życia rzeczy.
type Status string

const (
	StatusWIP      Status = "wip"
	StatusDone     Status = "done"
	StatusArchived Status = "archived"
)

func AllStatuses() []Status {
	return []Status{StatusWIP, StatusDone, StatusArchived}
}

func (s Status) Valid() bool {
	for _, v := range AllStatuses() {
		if s == v {
			return true
		}
	}
	return false
}

// AssetKind grupuje pliki według tego, jak da się je pokazać w UI.
type AssetKind string

const (
	AssetImage    AssetKind = "image"
	AssetModel    AssetKind = "model"    // .stl, .3mf — podgląd 3D
	AssetDocument AssetKind = "document" // pdf, txt, md
	AssetArchive  AssetKind = "archive"  // zip, 7z
	AssetOther    AssetKind = "other"
)

// Asset to plik przypięty do itemu. Same bajty leżą na dysku
// w data/files/<id>/ — w bazie trzymamy wyłącznie metadane.
type Asset struct {
	ID        string    `json:"id"`
	ItemID    string    `json:"item_id"`
	Filename  string    `json:"filename"`
	Mime      string    `json:"mime"`
	Size      int64     `json:"size"`
	Kind      AssetKind `json:"kind"`
	Width     *int      `json:"width"`  // tylko dla obrazów
	Height    *int      `json:"height"` // tylko dla obrazów
	Checksum  string    `json:"checksum"`
	SortOrder int       `json:"sort_order"`
	IsCover   bool      `json:"is_cover"`
	HasThumb  bool      `json:"has_thumb"`
	CreatedAt time.Time `json:"created_at"`
}

// Tag to etykieta. Count jest wypełniany tylko przy listowaniu tagów.
type Tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Count int    `json:"count,omitempty"`
}

// Item to jedna rzecz w portfolio.
//
// Pola opcjonalne są wskaźnikami, żeby w JSON-ie wychodziło `null`,
// a nie mylące "" albo "0001-01-01T00:00:00Z".
type Item struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Slug        string          `json:"slug"`
	Description string          `json:"description"`
	Kind        Kind            `json:"kind"`
	Status      Status          `json:"status"`
	Favorite    bool            `json:"favorite"`
	ExternalURL *string         `json:"external_url"`
	Metadata    json.RawMessage `json:"metadata"`
	HappenedAt  *time.Time      `json:"happened_at"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Tags        []Tag           `json:"tags"`

	// Cover to okładka używana na kafelku w siatce. Nil, gdy item nie ma
	// przypiętego zdjęcia — wtedy UI rysuje mniejszy kafelek tekstowy.
	Cover *Asset `json:"cover"`

	// Assets wypełniamy tylko w widoku pojedynczego itemu — na liście
	// byłoby to setki niepotrzebnych rekordów.
	Assets []Asset `json:"assets,omitempty"`
}

// ValidationError niesie nazwę pola, żeby frontend mógł podświetlić
// konkretny input, a nie tylko pokazać ogólny komunikat.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Ograniczenia długości. Trzymamy je jako stałe, bo pojawiają się
// w walidacji i w komunikatach błędów.
const (
	MaxTitleLen       = 200
	MaxDescriptionLen = 100_000
	MaxURLLen         = 2000
	MaxTagLen         = 40
	MaxTagsPerItem    = 30
)

// ItemInput to dane do utworzenia itemu (POST). Wszystkie pola opcjonalne
// mają sensowne wartości domyślne ustawiane w Normalize.
type ItemInput struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Kind        Kind            `json:"kind"`
	Status      Status          `json:"status"`
	Favorite    bool            `json:"favorite"`
	ExternalURL *string         `json:"external_url"`
	Metadata    json.RawMessage `json:"metadata"`
	HappenedAt  *time.Time      `json:"happened_at"`
	Tags        []string        `json:"tags"`
}

// Normalize porządkuje dane wejściowe: przycina białe znaki, ustawia
// wartości domyślne i czyści listę tagów. Wołane przed Validate.
func (in *ItemInput) Normalize() {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)

	if in.Status == "" {
		in.Status = StatusWIP
	}
	if len(in.Metadata) == 0 || string(in.Metadata) == "null" {
		in.Metadata = json.RawMessage(`{}`)
	}
	in.ExternalURL = normalizeURL(in.ExternalURL)
	in.Tags = NormalizeTags(in.Tags)
}

// Validate sprawdza reguły domenowe. Zwraca pierwszy napotkany problem —
// przy jednoosobowej aplikacji z formularzem po stronie UI to wystarcza,
// a kod jest o połowę krótszy niż zbieranie listy błędów.
func (in ItemInput) Validate() error {
	if in.Title == "" {
		return ValidationError{Field: "title", Message: "tytuł jest wymagany"}
	}
	if len([]rune(in.Title)) > MaxTitleLen {
		return ValidationError{Field: "title", Message: fmt.Sprintf("tytuł może mieć najwyżej %d znaków", MaxTitleLen)}
	}
	if len(in.Description) > MaxDescriptionLen {
		return ValidationError{Field: "description", Message: "opis jest za długi"}
	}
	if !in.Kind.Valid() {
		return ValidationError{Field: "kind", Message: fmt.Sprintf("dozwolone wartości: %s", joinKinds())}
	}
	if !in.Status.Valid() {
		return ValidationError{Field: "status", Message: "dozwolone wartości: wip, done, archived"}
	}
	if err := validateURL(in.ExternalURL); err != nil {
		return err
	}
	if err := ValidateMetadata(in.Metadata); err != nil {
		return err
	}
	if len(in.Tags) > MaxTagsPerItem {
		return ValidationError{Field: "tags", Message: fmt.Sprintf("najwyżej %d tagów na item", MaxTagsPerItem)}
	}
	return nil
}

// ItemPatch to częściowa aktualizacja (PATCH). Każde pole jest Optional,
// bo musimy odróżnić trzy sytuacje: pola nie było w żądaniu, pole ustawiono
// na null (wyczyść), pole ustawiono na wartość.
type ItemPatch struct {
	Title       Optional[string]          `json:"title"`
	Description Optional[string]          `json:"description"`
	Kind        Optional[Kind]            `json:"kind"`
	Status      Optional[Status]          `json:"status"`
	Favorite    Optional[bool]            `json:"favorite"`
	ExternalURL Optional[string]          `json:"external_url"`
	Metadata    Optional[json.RawMessage] `json:"metadata"`
	HappenedAt  Optional[time.Time]       `json:"happened_at"`
	Tags        Optional[[]string]        `json:"tags"`
}

// Normalize porządkuje tylko te pola, które faktycznie przyszły w żądaniu.
func (p *ItemPatch) Normalize() {
	if p.Title.Set && p.Title.Value != nil {
		trimmed := strings.TrimSpace(*p.Title.Value)
		p.Title.Value = &trimmed
	}
	if p.Description.Set && p.Description.Value != nil {
		trimmed := strings.TrimSpace(*p.Description.Value)
		p.Description.Value = &trimmed
	}
	if p.ExternalURL.Set {
		p.ExternalURL.Value = normalizeURL(p.ExternalURL.Value)
	}
	if p.Tags.Set && p.Tags.Value != nil {
		cleaned := NormalizeTags(*p.Tags.Value)
		p.Tags.Value = &cleaned
	}
}

func (p ItemPatch) Validate() error {
	if p.Title.Set {
		if p.Title.Value == nil || *p.Title.Value == "" {
			return ValidationError{Field: "title", Message: "tytuł nie może być pusty"}
		}
		if len([]rune(*p.Title.Value)) > MaxTitleLen {
			return ValidationError{Field: "title", Message: fmt.Sprintf("tytuł może mieć najwyżej %d znaków", MaxTitleLen)}
		}
	}
	if p.Description.Set && p.Description.Value != nil && len(*p.Description.Value) > MaxDescriptionLen {
		return ValidationError{Field: "description", Message: "opis jest za długi"}
	}
	if p.Kind.Set {
		if p.Kind.Value == nil || !p.Kind.Value.Valid() {
			return ValidationError{Field: "kind", Message: fmt.Sprintf("dozwolone wartości: %s", joinKinds())}
		}
	}
	if p.Status.Set {
		if p.Status.Value == nil || !p.Status.Value.Valid() {
			return ValidationError{Field: "status", Message: "dozwolone wartości: wip, done, archived"}
		}
	}
	if p.Favorite.Set && p.Favorite.Value == nil {
		return ValidationError{Field: "favorite", Message: "oczekiwano true albo false"}
	}
	if p.ExternalURL.Set {
		if err := validateURL(p.ExternalURL.Value); err != nil {
			return err
		}
	}
	if p.Metadata.Set {
		if p.Metadata.Value == nil {
			return ValidationError{Field: "metadata", Message: "metadata nie może być null — użyj {}"}
		}
		if err := ValidateMetadata(*p.Metadata.Value); err != nil {
			return err
		}
	}
	if p.Tags.Set {
		if p.Tags.Value == nil {
			return ValidationError{Field: "tags", Message: "tags nie może być null — użyj []"}
		}
		if len(*p.Tags.Value) > MaxTagsPerItem {
			return ValidationError{Field: "tags", Message: fmt.Sprintf("najwyżej %d tagów na item", MaxTagsPerItem)}
		}
	}
	return nil
}

// ValidateMetadata pilnuje, że metadata to obiekt JSON, a nie tablica
// czy liczba. Dzięki temu zapytania po polach metadanych mają sens.
func ValidateMetadata(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return ValidationError{Field: "metadata", Message: "oczekiwano obiektu JSON, np. {\"drukarka\": \"Prusa MK4\"}"}
	}
	return nil
}

// NormalizeTags przycina, usuwa puste i deduplikuje tagi po slugu,
// zachowując kolejność podaną przez użytkownika.
func NormalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := make(map[string]bool, len(tags))

	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if len([]rune(t)) > MaxTagLen {
			t = string([]rune(t)[:MaxTagLen])
		}
		key := Slugify(t)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
	}
	return out
}

func normalizeURL(u *string) *string {
	if u == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*u)
	if trimmed == "" {
		return nil // pusty string traktujemy jak brak URL-a
	}
	return &trimmed
}

func validateURL(u *string) error {
	if u == nil {
		return nil
	}
	if len(*u) > MaxURLLen {
		return ValidationError{Field: "external_url", Message: "adres jest za długi"}
	}
	parsed, err := url.Parse(*u)
	if err != nil {
		return ValidationError{Field: "external_url", Message: "to nie wygląda na poprawny adres"}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ValidationError{Field: "external_url", Message: "adres musi zaczynać się od http:// albo https://"}
	}
	if parsed.Host == "" {
		return ValidationError{Field: "external_url", Message: "adres musi zawierać nazwę hosta"}
	}
	return nil
}

func joinKinds() string {
	names := make([]string, 0, len(AllKinds()))
	for _, k := range AllKinds() {
		names = append(names, string(k))
	}
	return strings.Join(names, ", ")
}

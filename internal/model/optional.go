package model

import "encoding/json"

// Optional rozwiązuje klasyczny problem PATCH-a: jak odróżnić
// "tego pola nie było w żądaniu" od "ustaw to pole na null".
//
// Zwykły wskaźnik (*string) rozróżnia tylko dwa stany, a potrzebujemy trzech:
//
//	{}                        → Set=false, Value=nil  → nie ruszaj pola
//	{"external_url": null}    → Set=true,  Value=nil  → wyczyść pole
//	{"external_url": "http…"} → Set=true,  Value=&"…" → ustaw wartość
//
// Alternatywą byłby podwójny wskaźnik (**string), ale to się fatalnie czyta.
// Optional jest typem generycznym, więc jedna definicja obsługuje wszystkie pola.
type Optional[T any] struct {
	Set   bool
	Value *T
}

// UnmarshalJSON jest wołany przez encoding/json TYLKO wtedy, gdy klucz
// faktycznie wystąpił w dokumencie. To właśnie ten fakt wykorzystujemy,
// ustawiając Set na true — brak wywołania oznacza brak pola.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true

	if string(data) == "null" {
		o.Value = nil
		return nil
	}

	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

// MarshalJSON dopełnia parę, żeby typ dało się też serializować
// (przydatne w testach i przy logowaniu).
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if o.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*o.Value)
}

// Get zwraca wartość i informację, czy w ogóle została podana.
// Skraca warunki w kodzie store'a do jednej linii.
func (o Optional[T]) Get() (T, bool) {
	var zero T
	if !o.Set || o.Value == nil {
		return zero, false
	}
	return *o.Value, true
}

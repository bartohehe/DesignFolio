package httpapi

import "net/http"

type tagsResponse struct {
	Tags []tagJSON `json:"tags"`
}

// tagJSON powtarza kształt model.Tag, ale z zawsze obecnym polem count.
// W model.Tag count ma `omitempty`, bo przy itemie liczba nie ma sensu —
// tutaj jest przeciwnie: zero też jest informacją.
type tagJSON struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Count int    `json:"count"`
}

func (h *handlers) listTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.store.ListTags(r.Context())
	if err != nil {
		writeInternal(w, err, "listTags")
		return
	}

	out := make([]tagJSON, 0, len(tags))
	for _, t := range tags {
		out = append(out, tagJSON{ID: t.ID, Name: t.Name, Slug: t.Slug, Count: t.Count})
	}
	writeJSON(w, http.StatusOK, tagsResponse{Tags: out})
}

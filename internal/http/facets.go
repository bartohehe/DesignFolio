package httpapi

import "net/http"

func (h *handlers) getFacets(w http.ResponseWriter, r *http.Request) {
	facets, err := h.store.FacetCounts(r.Context())
	if err != nil {
		writeInternal(w, err, "getFacets")
		return
	}
	writeJSON(w, http.StatusOK, facets)
}

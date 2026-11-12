package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"designfolio/internal/model"
)

// exportCatalog oddaje cały katalog jako jeden plik JSON.
//
// Nie zawiera samych plików — te leżą w data/files i backupuje się je
// kopiując katalog (patrz scripts/). Eksport opisuje, co jest czym.
func (h *handlers) exportCatalog(w http.ResponseWriter, r *http.Request) {
	data, err := h.store.ExportAll(r.Context())
	if err != nil {
		writeInternal(w, err, "exportCatalog")
		return
	}

	filename := fmt.Sprintf("designfolio-%s.json", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	writeJSON(w, http.StatusOK, data)
}

type timelineResponse struct {
	Items []model.Item `json:"items"`
	Total int          `json:"total"`
}

// timeline zwraca itemy w porządku chronologicznym, bez stronicowania.
// Grupowanie po latach i miesiącach robi interfejs.
func (h *handlers) timeline(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.Timeline(r.Context())
	if err != nil {
		writeInternal(w, err, "timeline")
		return
	}
	writeJSON(w, http.StatusOK, timelineResponse{Items: items, Total: len(items)})
}

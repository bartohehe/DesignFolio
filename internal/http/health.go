package httpapi

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"designfolio/internal/store"
)

type healthResponse struct {
	Status  string    `json:"status"`
	Version string    `json:"version"`
	Go      string    `json:"go"`
	DB      string    `json:"db"`
	Time    time.Time `json:"time"`
}

// healthHandler zwraca domknięcie (closure) zamiast zwykłej funkcji-handlera.
// Dzięki temu zależności "wsiąkają" do środka i nie musimy trzymać ich
// w zmiennych globalnych. Ten sam wzorzec powtarza się w całym projekcie.
func healthHandler(version string, st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := healthResponse{
			Status:  "ok",
			Version: version,
			Go:      runtime.Version(),
			DB:      "ok",
			Time:    time.Now().UTC(),
		}

		// Health ma odpowiadać szybko, więc baza dostaje własny, krótki limit czasu.
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := st.Ping(ctx); err != nil {
			resp.Status = "degraded"
			resp.DB = "error: " + err.Error()
			writeJSON(w, http.StatusServiceUnavailable, resp)
			return
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

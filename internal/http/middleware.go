package httpapi

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"time"
)

// requireTokenForWrites sprawdza nagłówek X-App-Token, ale tylko dla metod,
// które coś zmieniają. Odczyt zostaje otwarty — aplikacja i tak chodzi na localhoście,
// a token jest tylko zabezpieczeniem "przed przypadkiem" w sieci domowej.
//
// Gdy token jest pusty, middleware nie robi nic i przepuszcza wszystko.
func requireTokenForWrites(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				got := r.Header.Get("X-App-Token")
				// ConstantTimeCompare zamiast ==, żeby czas porównania nie zdradzał,
				// ile początkowych znaków tokenu się zgadza. Przy jednym userze na
				// localhoście to nadmiar ostrożności, ale kosztuje jedną linię.
				if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
					writeErr(w, http.StatusUnauthorized, CodeUnauthorized, "brakujący lub błędny nagłówek X-App-Token")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// statusRecorder podgląda status odpowiedzi, żeby dało się go zalogować.
// http.ResponseWriter sam z siebie nie pozwala odczytać zapisanego statusu.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	// Handler może nigdy nie zawołać WriteHeader — wtedy Go domyślnie wysyła 200.
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// requestLogger loguje każde żądanie jedną linią przez log/slog.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"dur", time.Since(start).Round(time.Microsecond),
		)
	})
}

// recoverer łapie panikę w handlerze i zamienia ją na 500 zamiast ubijać
// całe połączenie. W runtime nie panikujemy celowo, ale nil pointer w nowym
// kodzie nie powinien wywalać serwera.
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panika w handlerze", "path", r.URL.Path, "panic", rec)
				writeErr(w, http.StatusInternalServerError, CodeInternal, "coś poszło nie tak po stronie serwera")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

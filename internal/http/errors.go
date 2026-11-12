package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Ustalone kody błędów. Trzymamy je jako stałe, żeby frontend mógł na nich
// polegać, a literówka w handlerze nie przeszła niezauważona.
const (
	CodeBadRequest      = "bad_request"
	CodeNotFound        = "not_found"
	CodeConflict        = "conflict"
	CodeUnauthorized    = "unauthorized"
	CodeTooLarge        = "payload_too_large"
	CodeUnsupportedType = "unsupported_media_type"
	CodeInternal        = "internal"
)

// errorBody to jedyny kształt błędu, jaki wychodzi z API:
//
//	{ "error": { "code": "...", "message": "..." } }
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeJSON serializuje wartość do JSON-a z podanym statusem.
//
// Uwaga na kolejność: nagłówki i status muszą polecieć PRZED zapisem ciała.
// Po pierwszym w.Write() status jest już wysłany i WriteHeader nic nie zmieni.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Nie da się już naprawić odpowiedzi — klient dostał połowę JSON-a.
		// Jedyne sensowne wyjście to log.
		slog.Error("zapis odpowiedzi JSON nie powiódł się", "err", err)
	}
}

// writeErr odsyła błąd w ustalonym kształcie.
func writeErr(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

// writeInternal loguje prawdziwy błąd, ale klientowi pokazuje ogólny komunikat.
// Szczegóły techniczne zostają w logu serwera.
func writeInternal(w http.ResponseWriter, err error, op string) {
	slog.Error("błąd wewnętrzny", "op", op, "err", err)
	writeErr(w, http.StatusInternalServerError, CodeInternal, "coś poszło nie tak po stronie serwera")
}

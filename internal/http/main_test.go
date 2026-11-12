package httpapi_test

import (
	"io"
	"log/slog"
	"os"
	"testing"
)

// TestMain wycisza logi aplikacji na czas testów.
//
// Bez tego każdy request wypisuje linię do stdout i prawdziwy wynik testów
// tonie w kilkuset wierszach szumu. Logi błędów nadal są w kodzie — po prostu
// nie trafiają na terminal.
func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

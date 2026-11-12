// Command server uruchamia DesignFolio: API pod /api oraz — w trybie prod —
// zbudowany frontend wbudowany w tę samą binarkę.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"designfolio/internal/config"
	httpapi "designfolio/internal/http"
	"designfolio/internal/media"
	"designfolio/internal/store"
	"designfolio/web"
)

// version podmieniamy przy budowaniu:
//
//	go build -ldflags "-X main.version=1.0.0" ./cmd/server
var version = "dev"

func main() {
	// Cała logika siedzi w run(), żeby dało się użyć defer i zwracać błędy
	// zamiast wołać os.Exit w środku (os.Exit pomija odroczone wywołania defer).
	if err := run(); err != nil {
		slog.Error("serwer zakończył pracę z błędem", "err", err)
		os.Exit(1)
	}
}

func run() error {
	envPath := flag.String("env", ".env", "ścieżka do pliku .env (opcjonalny)")
	addrFlag := flag.String("addr", "", "adres nasłuchu, nadpisuje APP_ADDR")
	flag.Parse()

	cfg, err := config.Load(*envPath)
	if err != nil {
		return err
	}
	if *addrFlag != "" {
		cfg.Addr = *addrFlag
	}

	setupLogger(cfg.Dev)

	if err := cfg.EnsureDirs(); err != nil {
		return err
	}

	// Otwarcie bazy wykonuje też zaległe migracje.
	// Kontekst z limitem czasu chroni przed zawieszeniem startu, gdyby plik
	// bazy był zablokowany przez inny proces.
	dbCtx, cancelDB := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelDB()

	st, err := store.Open(dbCtx, cfg.DBPath())
	if err != nil {
		return err
	}
	defer func() {
		if err := st.Close(); err != nil {
			slog.Error("zamykanie bazy", "err", err)
		}
	}()

	static, built := web.Dist()
	if !built {
		slog.Warn("frontend niezbudowany — serwuję samo API",
			"podpowiedź", "cd web && npm run dev (dev) albo npm run build (prod)")
	}

	handler := httpapi.NewRouter(httpapi.Options{
		Version:        version,
		Token:          cfg.Token,
		Store:          st,
		Media:          media.NewStorage(cfg.FilesDir(), cfg.MaxUploadBytes()),
		MaxUploadBytes: cfg.MaxUploadBytes(),
		Static:         static,
	})

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: handler,
		// Limit na same nagłówki. Świadomie NIE ustawiamy ReadTimeout ani
		// WriteTimeout: w etapie 4 dojdzie upload i pobieranie dużych plików
		// (modele 3D, zdjęcia w oryginale), a globalny timeout ucinałby je
		// w połowie. Aplikacja chodzi na localhoście, więc ryzyko slowloris
		// jest tu czysto teoretyczne.
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// NotifyContext daje kontekst, który zostaje anulowany przy Ctrl+C
	// albo przy `docker stop` (SIGTERM). To sygnał do zamykania serwera.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Kanał na błąd z goroutine serwera — bez tego błąd startu (np. zajęty port)
	// zginąłby w tle, a program czekałby w nieskończoność na sygnał.
	errCh := make(chan error, 1)
	go func() {
		slog.Info("serwer wystartował",
			"addr", cfg.Addr,
			"data", cfg.DataDir,
			"token", cfg.Token != "",
			"frontend", built,
			"version", version,
		)
		// ErrServerClosed to normalne zakończenie po Shutdown, nie awaria.
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("nasłuch na %s: %w", cfg.Addr, err)
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("zamykam serwer...")
	}

	// Shutdown przestaje przyjmować nowe połączenia i czeka, aż bieżące
	// requesty się dokończą — maksymalnie 15 sekund.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("zamykanie serwera: %w", err)
	}
	slog.Info("serwer zatrzymany")
	return nil
}

// setupLogger ustawia globalny logger slog: czytelny tekst w dev, JSON w prod
// (łatwiejszy do przeszukiwania, gdy leci do pliku albo do dockerowych logów).
func setupLogger(dev bool) {
	level := slog.LevelInfo
	if dev {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}

	var h slog.Handler
	if dev {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(h))
}

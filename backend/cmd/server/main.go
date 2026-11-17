package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"catch-me/backend/api"
	"catch-me/backend/controller"
	"catch-me/backend/db"
	"catch-me/backend/googleplaces"
	"catch-me/backend/service"

	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
)

//go:generate go run ../openapi -out ../../api/openapi.json

func main() {
	godotenv.Load("../.env")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "catch-me.sqlite", "SQLite database file (parent directory must exist)")
	googlePlacesAPIKey := flag.String("google-places-api-key", os.Getenv("GOOGLE_MAPS_KEY"), "Google Places API (New) key used to find landmarks for game routes (env GOOGLE_MAPS_KEY); required")
	skipDefault, _ := strconv.ParseBool(os.Getenv("TEST_SKIP_START_POSITION_CHECK"))
	skipStartPositionCheck := flag.Bool("test-skip-start-position-check", skipDefault, "Development only: bypass runner Ready position checks; keep real location updates (env TEST_SKIP_START_POSITION_CHECK)")
	healthcheck := flag.Bool("healthcheck", false, "Probe GET /health on -addr and exit 0 if healthy (container HEALTHCHECK; the runtime image has no shell or curl)")
	flag.Parse()

	if *healthcheck {
		os.Exit(probeHealth(*addr))
	}

	// Colored logs that name the source file and line of every entry.
	slog.SetDefault(slog.New(tint.NewTextHandler(os.Stderr, &tint.Options{AddSource: true, TimeFormat: time.DateTime})))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *addr, *dbPath, *googlePlacesAPIKey, *skipStartPositionCheck); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}

// probeHealth returns the process exit code of a request to the local /health endpoint.
func probeHealth(addr string) int {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return 1
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + net.JoinHostPort("127.0.0.1", port) + "/health")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func run(ctx context.Context, addr, dbPath, googlePlacesAPIKey string, skipStartPositionCheck bool) error {
	if googlePlacesAPIKey == "" {
		return errors.New("a Google Places API key is required: set -google-places-api-key or GOOGLE_MAPS_KEY")
	}
	store, err := db.Open(ctx, dbPath)
	if err != nil {
		return err
	}
	defer store.Close()
	if skipStartPositionCheck {
		slog.Warn("TEST MODE: runner start-position check disabled; real position updates remain enabled")
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	games := service.NewGameService(store, googleplaces.NewClient(googleplaces.DefaultBaseURL, googlePlacesAPIKey, ""), skipStartPositionCheck)
	go deleteExpiredPositions(ctx, games)

	server := &http.Server{
		Handler:           api.NewHandler(controller.NewGameController(games)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		// Above the service's 8-second budget for landmark lookups, below the app's 15-second wait.
		WriteTimeout: 12 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(listener)
	}()
	slog.Info("Catch me! API listening", "addr", listener.Addr().String())

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return err
		}
		return nil
	}
}

// deleteExpiredPositions enforces the position retention at startup
// and hourly until shutdown.
func deleteExpiredPositions(ctx context.Context, games *service.GameService) {
	for {
		if err := games.DeleteExpiredPositions(ctx); err != nil && ctx.Err() == nil {
			slog.Error("delete expired positions", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Hour):
		}
	}
}

package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"catch-me/backend/api"
	"catch-me/backend/controller"
	"catch-me/backend/db"
	"catch-me/backend/googleplaces"
	"catch-me/backend/service"

	"github.com/joho/godotenv"
)

//go:generate go tool swag init -g main.go -d .,../../controller --requiredByDefault --outputTypes json --output ../../api

// @title Catch me! API
// @version 0.1.0
// @description API generated from Go handler annotations and response structs.
// @BasePath /
// @securityDefinitions.apikey PlayerToken
// @in header
// @name Authorization
// @description "Bearer " followed by the playerToken returned when joining a game.
func main() {
	godotenv.Load("../.env")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "catch-me.sqlite", "SQLite database file (parent directory must exist)")
	googlePlacesAPIKey := flag.String("google-places-api-key", os.Getenv("GOOGLE_MAPS_KEY"), "Google Places API (New) key used to find landmarks for game routes (env GOOGLE_MAPS_KEY); required")
	skipStartPositionCheck := flag.Bool("test-skip-start-position-check", false, "Development only: bypass runner Ready position checks; keep real location updates")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *addr, *dbPath, *googlePlacesAPIKey, *skipStartPositionCheck); err != nil {
		log.Printf("server failed: %v", err)
		os.Exit(1)
	}
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
		log.Print("TEST MODE: runner start-position check disabled; real position updates remain enabled")
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
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(listener)
	}()
	log.Printf("Catch me! API listening on %s", listener.Addr())

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Print("shutting down")
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
			log.Print(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Hour):
		}
	}
}

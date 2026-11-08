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

	"catch-me/backend/internal/api"
	"catch-me/backend/internal/controller"
	"catch-me/backend/internal/db"
	"catch-me/backend/internal/service"
)

// @title Catch me! API
// @version 0.1.0
// @description API generated from Go handler annotations and response structs.
// @BasePath /
// @securityDefinitions.apikey PlayerToken
// @in header
// @name Authorization
// @description "Bearer " followed by the playerToken returned when joining a game.
func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	webOrigin := flag.String("web-origin", "http://localhost:8081", "Allowed browser origin; empty disables browser access")
	dbPath := flag.String("db", "catch-me.sqlite", "SQLite database file (parent directory must exist)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *addr, *webOrigin, *dbPath); err != nil {
		log.Printf("server failed: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, addr, webOrigin, dbPath string) error {
	store, err := db.Open(ctx, dbPath)
	if err != nil {
		return err
	}
	defer store.Close()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler:           api.NewHandler(webOrigin, controller.NewGameController(service.NewGameService(store))),
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

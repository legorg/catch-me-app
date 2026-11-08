// Package api registers endpoints and applies HTTP middleware.
package api

import (
	"net/http"

	"catch-me/backend/internal/controller"
)

func NewHandler(webOrigin string, games *controller.GameController) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", controller.Health)
	mux.HandleFunc("/api/games", games.Create)
	mux.HandleFunc("/api/games/{sessionId}", games.Get)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		controller.WriteError(w, http.StatusNotFound, "not found")
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Add("Vary", "Origin")
		if origin := r.Header.Get("Origin"); origin != "" {
			if webOrigin == "" || origin != webOrigin {
				controller.WriteError(w, http.StatusForbidden, "origin not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

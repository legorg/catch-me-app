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
	mux.HandleFunc("/api/players", games.Join)
	mux.HandleFunc("/api/me", games.Leave)
	mux.HandleFunc("/api/me/game", games.State)
	mux.HandleFunc("/api/me/ready", games.SetReady)
	mux.HandleFunc("/api/me/route", games.RegenerateRoute)
	mux.HandleFunc("/api/me/hunter-team", games.SetHunterTeam)
	mux.HandleFunc("/api/me/position", games.UpdatePosition)
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

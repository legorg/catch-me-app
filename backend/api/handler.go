// Package api registers endpoints and applies HTTP middleware.
package api

import (
	"net/http"

	"catch-me/backend/controller"
)

func NewHandler(games *controller.GameController) http.Handler {
	mux := http.NewServeMux()
	mount(mux, operations(games))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		controller.WriteError(w, http.StatusNotFound, "not found")
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r)
	})
}

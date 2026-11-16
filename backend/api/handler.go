// Package api registers endpoints and applies HTTP middleware.
package api

import (
	"log/slog"
	"net/http"
	"time"

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
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		mux.ServeHTTP(recorder, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", recorder.status, "duration", time.Since(start), "remote", r.RemoteAddr)
	})
}

// statusRecorder remembers the status code written by a handler for request logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

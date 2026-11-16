package controller

import (
	"net/http"
)

// Health reports whether the backend is running.
func Health(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok", Service: "catch-me-backend"})
}

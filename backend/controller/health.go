package controller

import (
	"net/http"
)

// Health reports whether the backend is running.
// @Summary Check backend health
// @ID getHealth
// @Produce json
// @Success 200 {object} HealthResponse "Backend is running"
// @Failure 403 {object} APIError "Browser origin is not allowed"
// @Router /health [get]
func Health(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok", Service: "catch-me-backend"})
}

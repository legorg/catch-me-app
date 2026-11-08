package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"
)

// NewHandler builds the API without starting a server, so handlers are easy to test.
func NewHandler(webOrigin string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/api/demo", demoGreeting)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, APIError{Error: "not found"})
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Add("Vary", "Origin")

		if origin := r.Header.Get("Origin"); origin != "" {
			if webOrigin == "" || origin != webOrigin {
				writeJSON(w, http.StatusForbidden, APIError{Error: "origin not allowed"})
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

// health reports whether the backend is running.
// @Summary Check backend health
// @ID getHealth
// @Produce json
// @Success 200 {object} HealthResponse "Backend is running"
// @Failure 403 {object} APIError "Browser origin is not allowed"
// @Router /health [get]
func health(w http.ResponseWriter, r *http.Request) {
	if !allowGet(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok", Service: "catch-me-backend"})
}

// demoGreeting returns a dummy message demonstrating the generated client.
// @Summary Return a dummy greeting from Go
// @ID getDemoGreeting
// @Produce json
// @Param name query string false "Player name; whitespace is trimmed, blank defaults to Runner, trimmed names are limited to 60 Unicode characters" default(Runner)
// @Success 200 {object} DemoResponse "Greeting demonstrating frontend/backend communication"
// @Failure 400 {object} APIError "Invalid player name"
// @Failure 403 {object} APIError "Browser origin is not allowed"
// @Router /api/demo [get]
func demoGreeting(w http.ResponseWriter, r *http.Request) {
	if !allowGet(w, r) {
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		name = "Runner"
	}
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) > 60 {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "name must be valid UTF-8 and at most 60 characters"})
		return
	}
	writeJSON(w, http.StatusOK, DemoResponse{
		Message:    fmt.Sprintf("Hello, %s! Your Expo app is connected to Go.", name),
		PlayerName: name,
		Service:    "catch-me-backend",
	})
}

func allowGet(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	w.Header().Set("Allow", "GET, HEAD")
	writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
	return false
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("could not write JSON response: %v", err)
	}
}

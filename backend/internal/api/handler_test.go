package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewHandler("", nil).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" || body["service"] != "catch-me-backend" {
		t.Fatalf("unexpected body: %v", body)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache control = %q", got)
	}
}

func TestRouting(t *testing.T) {
	for _, test := range []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"unknown route", http.MethodGet, "/missing", http.StatusNotFound},
		{"nested health route", http.MethodGet, "/health/missing", http.StatusNotFound},
		{"unsupported method", http.MethodPost, "/health", http.StatusMethodNotAllowed},
		{"head", http.MethodHead, "/health", http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			NewHandler("", nil).ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d", recorder.Code, test.status)
			}
			if test.status == http.StatusMethodNotAllowed && recorder.Header().Get("Allow") != "GET, HEAD" {
				t.Fatal("missing allowed methods")
			}
		})
	}
}

func TestCORS(t *testing.T) {
	for _, test := range []struct {
		name          string
		allowedOrigin string
		origin        string
		method        string
		status        int
		allowOrigin   string
	}{
		{"native request", "http://localhost:8081", "", http.MethodGet, http.StatusOK, ""},
		{"allowed browser", "http://localhost:8081", "http://localhost:8081", http.MethodGet, http.StatusOK, "http://localhost:8081"},
		{"preflight", "http://localhost:8081", "http://localhost:8081", http.MethodOptions, http.StatusNoContent, "http://localhost:8081"},
		{"other browser", "http://localhost:8081", "https://example.com", http.MethodGet, http.StatusForbidden, ""},
		{"rejected preflight", "http://localhost:8081", "https://example.com", http.MethodOptions, http.StatusForbidden, ""},
		{"browser access disabled", "", "http://localhost:8081", http.MethodGet, http.StatusForbidden, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/health", nil)
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			recorder := httptest.NewRecorder()
			NewHandler(test.allowedOrigin, nil).ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d", recorder.Code, test.status)
			}
			if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != test.allowOrigin {
				t.Fatalf("allowed origin = %q, want %q", got, test.allowOrigin)
			}
		})
	}
}

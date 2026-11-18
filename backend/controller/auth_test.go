package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"catch-me/backend/game"
)

func TestRequirePlayerToken(t *testing.T) {
	token := game.NewPlayerToken()
	for _, tc := range []struct {
		name, authorization string
		valid               bool
	}{
		{"missing", "", false},
		{"wrong scheme", "Basic " + token.String(), false},
		{"lowercase scheme", "bearer " + token.String(), false},
		{"empty", "Bearer ", false},
		{"too short", "Bearer abc", false},
		{"non hexadecimal", "Bearer " + strings.Repeat("z", 64), false},
		{"valid", "Bearer " + token.String(), true},
		{"surrounding spaces", "Bearer  " + token.String() + " ", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Authorization", tc.authorization)
			w := httptest.NewRecorder()
			got, ok := requirePlayerToken(w, r)
			if ok != tc.valid {
				t.Fatalf("valid = %v, want %v", ok, tc.valid)
			}
			if tc.valid {
				if got != token || w.Flushed || w.Body.Len() != 0 || len(w.Header()) != 0 {
					t.Fatal("valid token must be returned without writing a response")
				}
				return
			}
			if got != (game.PlayerToken{}) {
				t.Fatal("invalid token must return the zero value")
			}
			assertUnauthorized(t, w)
		})
	}
}

func TestPlayerHandlersRejectMissingToken(t *testing.T) {
	// No service: unauthenticated requests must return before using it.
	c := NewGameController(nil)
	for _, tc := range []struct {
		name, method string
		handler      http.HandlerFunc
	}{
		{"state", http.MethodGet, c.State},
		{"finish", http.MethodPost, c.Finish},
		{"caught", http.MethodPost, c.MarkCaught},
		{"ready", http.MethodPut, c.SetReady},
		{"route", http.MethodPost, c.RegenerateRoute},
		{"hunter team", http.MethodPut, c.SetHunterTeam},
		{"position", http.MethodPut, c.UpdatePosition},
		{"joker", http.MethodPost, c.UseJoker},
		{"add sketch", http.MethodPost, c.Sketches},
		{"clear sketches", http.MethodDelete, c.Sketches},
		{"leave", http.MethodDelete, c.Leave},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.handler(w, httptest.NewRequest(tc.method, "/", nil))
			assertUnauthorized(t, w)
		})
	}
}

func assertUnauthorized(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("expected 401 with bearer challenge: %d %v", w.Code, w.Header())
	}
	if got := w.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", got)
	}
	var body APIError
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Error != game.ErrUnauthorized.Error() {
		t.Fatalf("unexpected error response: %s", w.Body)
	}
}

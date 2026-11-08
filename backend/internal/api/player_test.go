package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"catch-me/backend/internal/controller"
	"catch-me/backend/internal/service"
)

func send(t *testing.T, handler http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func decode[T any](t *testing.T, recorder *httptest.ResponseRecorder, status int) T {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, status, recorder.Body)
	}
	var value T
	if err := json.Unmarshal(recorder.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestLobbyFlow(t *testing.T) {
	handler := NewHandler("", controller.NewGameController(service.NewGameService(testStore(t))))
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":52.52,"longitude":13.405}`), 201)
	codes := created.JoinCodes

	runner := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "",
		`{"code":" `+strings.ToLower(codes.Runner)+` ","name":" Ada "}`), 201)
	if len(runner.PlayerToken) != 64 || runner.State.Team != "runner" || runner.State.JoinCode != codes.Runner ||
		runner.State.Status != "lobby" || runner.State.StartsAt != "" || runner.State.SessionID != created.Game.SessionID ||
		len(runner.State.Players) != 1 || runner.State.Players[0].Name != "Ada" || runner.State.PlayerID != runner.State.Players[0].PlayerID {
		t.Fatalf("unexpected join response: %+v", runner)
	}
	hunter := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+codes.Hunter+`","name":"Bob"}`), 201)

	stateRecorder := send(t, handler, "GET", "/api/me/game", runner.PlayerToken, "")
	state := decode[controller.GameStateResponse](t, stateRecorder, 200)
	if len(state.Players) != 2 || state.Playground != created.Game.Playground {
		t.Fatalf("unexpected state: %+v", state)
	}
	if strings.Contains(stateRecorder.Body.String(), codes.Hunter) || strings.Contains(stateRecorder.Body.String(), runner.PlayerToken) {
		t.Fatal("state leaks the other team's code or a token")
	}

	decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/ready", runner.PlayerToken, `{"ready":true}`), 200)
	state = decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/ready", hunter.PlayerToken, `{"ready":true}`), 200)
	if state.Status != "countdown" || state.StartsAt == "" || state.ServerTime == "" || state.Team != "hunter" {
		t.Fatalf("countdown not started: %+v", state)
	}

	if recorder := send(t, handler, "DELETE", "/api/me", hunter.PlayerToken, ""); recorder.Code != 204 || recorder.Body.Len() != 0 {
		t.Fatalf("leave: %d %s", recorder.Code, recorder.Body)
	}
	if recorder := send(t, handler, "GET", "/api/me/game", hunter.PlayerToken, ""); recorder.Code != 401 || recorder.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("left player still authorized: %d", recorder.Code)
	}
	state = decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", runner.PlayerToken, ""), 200)
	if state.Status != "lobby" || state.StartsAt != "" || len(state.Players) != 1 {
		t.Fatalf("leave did not cancel countdown: %+v", state)
	}
}

func TestLobbyErrors(t *testing.T) {
	handler := NewHandler("", controller.NewGameController(service.NewGameService(testStore(t))))
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":0,"longitude":0}`), 201)
	runner := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Ada"}`), 201)
	for _, tc := range []struct {
		name, method, path, token, body string
		status                          int
	}{
		{"unknown code", "POST", "/api/players", "", `{"code":"ZZZZZZ","name":"Bob"}`, 404},
		{"invalid code", "POST", "/api/players", "", `{"code":"ABC","name":"Bob"}`, 400},
		{"missing code", "POST", "/api/players", "", `{"name":"Bob"}`, 400},
		{"empty name", "POST", "/api/players", "", `{"code":"` + created.JoinCodes.Hunter + `","name":"  "}`, 400},
		{"long name", "POST", "/api/players", "", `{"code":"` + created.JoinCodes.Hunter + `","name":"` + strings.Repeat("x", 21) + `"}`, 400},
		{"duplicate name", "POST", "/api/players", "", `{"code":"` + created.JoinCodes.Hunter + `","name":"ADA"}`, 409},
		{"unknown join field", "POST", "/api/players", "", `{"code":"ZZZZZZ","name":"Bob","team":"hunter"}`, 400},
		{"wrong join method", "GET", "/api/players", "", "", 405},
		{"missing token", "GET", "/api/me/game", "", "", 401},
		{"malformed token", "GET", "/api/me/game", "abc", "", 401},
		{"unknown token", "GET", "/api/me/game", strings.Repeat("a", 64), "", 401},
		{"ready without value", "PUT", "/api/me/ready", runner.PlayerToken, `{}`, 400},
		{"ready wrong type", "PUT", "/api/me/ready", runner.PlayerToken, `{"ready":"yes"}`, 400},
		{"ready without token", "PUT", "/api/me/ready", "", `{"ready":true}`, 401},
		{"wrong ready method", "POST", "/api/me/ready", runner.PlayerToken, `{"ready":true}`, 405},
		{"wrong leave method", "GET", "/api/me", runner.PlayerToken, "", 405},
		{"leave without token", "DELETE", "/api/me", "", "", 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := send(t, handler, tc.method, tc.path, tc.token, tc.body)
			if body := decode[controller.APIError](t, recorder, tc.status); body.Error == "" {
				t.Fatal("expected error message")
			}
		})
	}
}

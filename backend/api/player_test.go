package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"catch-me/backend/controller"
	"catch-me/backend/game"
	"catch-me/backend/service"
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

// ownEntry is the requester's player entry. Joins can share a timestamp, so the order is not fixed.
func ownEntry(state controller.GameStateResponse) controller.PlayerResponse {
	for _, p := range state.Players {
		if p.PlayerID == state.PlayerID {
			return p
		}
	}
	return controller.PlayerResponse{}
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
	// The start-cell check depends on a per-runner cell that responses no longer expose.
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, true)))
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":52.52,"longitude":13.405}`), 201)
	codes := created.JoinCodes

	runner := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "",
		`{"code":" `+codes.Runner+` ","name":" Ada "}`), 201)
	if len(runner.PlayerToken) != 64 || runner.State.Team != "runner" || runner.State.JoinCodes != codes ||
		runner.State.Status != "lobby" || runner.State.StartsAt != nil || runner.State.SessionID != created.Game.SessionID ||
		len(runner.State.Players) != 1 || runner.State.Players[0].Name != "Ada" || runner.State.PlayerID != runner.State.Players[0].PlayerID {
		t.Fatalf("unexpected join response: %+v", runner)
	}
	hunter := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+codes.Hunter+`","name":"Bob"}`), 201)
	if runner.State.Route == nil || *runner.State.Route != created.Game.Route || created.Game.Route.DistanceMeters <= 0 {
		t.Fatalf("runner must get the game route: %+v %+v", runner.State.Route, created.Game.Route)
	}
	if hunter.State.Route != nil {
		t.Fatalf("hunter must not get the route: %+v", hunter.State.Route)
	}
	if recorder := send(t, handler, "GET", "/api/me/game", hunter.PlayerToken, ""); strings.Contains(recorder.Body.String(), `"route"`) {
		t.Fatal("hunter state leaks the route")
	}
	decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/route", runner.PlayerToken, ""), 200)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/route", hunter.PlayerToken, ""), 409)

	stateRecorder := send(t, handler, "GET", "/api/me/game", runner.PlayerToken, "")
	state := decode[controller.GameStateResponse](t, stateRecorder, 200)
	if len(state.Players) != 2 || state.Playground != created.Game.Playground {
		t.Fatalf("unexpected state: %+v", state)
	}
	if state.JoinCodes != codes || strings.Contains(stateRecorder.Body.String(), runner.PlayerToken) {
		t.Fatal("state must show both join codes and never a token")
	}

	start := [2]float64{created.Game.Playground.Center.Latitude, created.Game.Playground.Center.Longitude}
	positionBody := fmt.Sprintf(`{"latitude":%v,"longitude":%v,"accuracyMeters":8}`, start[0], start[1])
	state = decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/position", runner.PlayerToken, positionBody), 200)
	if own := ownEntry(state).Position; own == nil || own.Latitude != start[0] || own.AccuracyMeters != 8 {
		t.Fatalf("own position missing: %+v", own)
	}
	if recorder := send(t, handler, "GET", "/api/me/game", hunter.PlayerToken, ""); strings.Contains(recorder.Body.String(), `"position"`) {
		t.Fatal("hunter sees a runner position before the game runs")
	}

	decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/ready", runner.PlayerToken, `{"ready":true}`), 200)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/route", runner.PlayerToken, ""), 409)
	decode[controller.APIError](t, send(t, handler, "PUT", "/api/me/ready", hunter.PlayerToken, `{"ready":true}`), 409)
	decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/hunter-team", hunter.PlayerToken, `{"name":"Wolves"}`), 200)
	state = decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/ready", hunter.PlayerToken, `{"ready":true}`), 200)
	if state.Status != "countdown" || state.StartsAt == nil || state.ServerTime.IsZero() || state.Team != "hunter" {
		t.Fatalf("countdown not started: %+v", state)
	}

	if recorder := send(t, handler, "DELETE", "/api/me", hunter.PlayerToken, ""); recorder.Code != 204 || recorder.Body.Len() != 0 {
		t.Fatalf("leave: %d %s", recorder.Code, recorder.Body)
	}
	if recorder := send(t, handler, "GET", "/api/me/game", hunter.PlayerToken, ""); recorder.Code != 401 || recorder.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("left player still authorized: %d", recorder.Code)
	}
	state = decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", runner.PlayerToken, ""), 200)
	if state.Status != "lobby" || state.StartsAt != nil || len(state.Players) != 1 {
		t.Fatalf("leave did not cancel countdown: %+v", state)
	}
}

func TestStartPositionOverrideKeepsRealLocationUpdates(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, true)))
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":0,"longitude":0}`), 201)
	runner := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Ada"}`), 201)
	hunter := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Hunter+`","name":"Bob"}`), 201)
	decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/ready", runner.PlayerToken, `{"ready":true}`), 200)
	decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/hunter-team", hunter.PlayerToken, `{"name":"Wolves"}`), 200)
	countdown := decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/ready", hunter.PlayerToken, `{"ready":true}`), 200)
	for _, body := range []string{`{"latitude":1,"longitude":2,"accuracyMeters":8}`, `{"latitude":3,"longitude":4,"accuracyMeters":8}`} {
		state := decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/position", runner.PlayerToken, body), 200)
		if !state.StartPositionCheckDisabled || state.Status != "countdown" || !state.StartsAt.Equal(*countdown.StartsAt) || !ownEntry(state).Ready {
			t.Fatal("test override failed to preserve readiness and countdown")
		}
		var reported struct{ Latitude, Longitude float64 }
		_ = json.Unmarshal([]byte(body), &reported)
		own := ownEntry(state).Position
		if own == nil || own.Latitude != reported.Latitude || own.Longitude != reported.Longitude {
			t.Fatal("real reported position was replaced or not updated")
		}
	}
}

// centerLandmarks "finds" the center of every cell it is asked about.
type centerLandmarks struct{}

func (centerLandmarks) Landmarks(_ context.Context, cell game.Bounds) ([]game.Landmark, error) {
	return []game.Landmark{{Title: "Center", Coordinates: cell.Center()}}, nil
}

func TestLobbyErrors(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, false)))
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":0,"longitude":0}`), 201)
	runner := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Ada"}`), 201)
	for _, tc := range []struct {
		name, method, path, token, body string
		status                          int
	}{
		{"unknown code", "POST", "/api/players", "", `{"code":"000000","name":"Bob"}`, 404},
		{"invalid code", "POST", "/api/players", "", `{"code":"ABC","name":"Bob"}`, 400},
		{"letters in code", "POST", "/api/players", "", `{"code":"K7M2QX","name":"Bob"}`, 400},
		{"missing code", "POST", "/api/players", "", `{"name":"Bob"}`, 400},
		{"empty name", "POST", "/api/players", "", `{"code":"` + created.JoinCodes.Hunter + `","name":"  "}`, 400},
		{"long name", "POST", "/api/players", "", `{"code":"` + created.JoinCodes.Hunter + `","name":"` + strings.Repeat("x", 21) + `"}`, 400},
		{"duplicate name", "POST", "/api/players", "", `{"code":"` + created.JoinCodes.Hunter + `","name":"ADA"}`, 409},
		{"unknown join field", "POST", "/api/players", "", `{"code":"123456","name":"Bob","team":"hunter"}`, 400},
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
		{"route without token", "POST", "/api/me/route", "", "", 401},
		{"wrong route method", "GET", "/api/me/route", runner.PlayerToken, "", 405},
		{"position without token", "PUT", "/api/me/position", "", `{"latitude":0,"longitude":0,"accuracyMeters":5}`, 401},
		{"position missing accuracy", "PUT", "/api/me/position", runner.PlayerToken, `{"latitude":0,"longitude":0}`, 400},
		{"position out of range", "PUT", "/api/me/position", runner.PlayerToken, `{"latitude":85,"longitude":0,"accuracyMeters":5}`, 400},
		{"position bad accuracy", "PUT", "/api/me/position", runner.PlayerToken, `{"latitude":0,"longitude":0,"accuracyMeters":-1}`, 400},
		{"wrong position method", "POST", "/api/me/position", runner.PlayerToken, "", 405},
		{"ready away from the start", "PUT", "/api/me/ready", runner.PlayerToken, `{"ready":true}`, 409},
		{"hunter team without token", "PUT", "/api/me/hunter-team", "", `{"name":"Wolves"}`, 401},
		{"hunter team without name", "PUT", "/api/me/hunter-team", runner.PlayerToken, `{}`, 400},
		{"hunter team name too long", "PUT", "/api/me/hunter-team", runner.PlayerToken, `{"name":"` + strings.Repeat("x", 21) + `"}`, 400},
		{"runner joins hunter team", "PUT", "/api/me/hunter-team", runner.PlayerToken, `{"name":"Wolves"}`, 409},
		{"wrong hunter team method", "POST", "/api/me/hunter-team", runner.PlayerToken, `{"name":"Wolves"}`, 405},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := send(t, handler, tc.method, tc.path, tc.token, tc.body)
			if body := decode[controller.APIError](t, recorder, tc.status); body.Error == "" {
				t.Fatal("expected error message")
			}
		})
	}
}

func TestUseJokerRefusesWhatTheRulesForbid(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, true)))
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":0,"longitude":0}`), 201)
	runner := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Ada"}`), 201)
	hunter := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Hunter+`","name":"Bob"}`), 201)
	body := `{"kind":"showRunners"}`
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", "", body), 401)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", hunter.PlayerToken, `{"kind":"teleport"}`), 400)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", hunter.PlayerToken, body), 409)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", runner.PlayerToken, body), 409)
	decode[controller.APIError](t, send(t, handler, "GET", "/api/me/jokers", hunter.PlayerToken, ""), 405)
	for _, token := range []string{runner.PlayerToken, hunter.PlayerToken} {
		state := decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", token, ""), 200)
		if state.Jokers == nil || len(state.Jokers) != 0 {
			t.Fatalf("a player without a hunter team has no jokers: %+v", state.Jokers)
		}
	}
	decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/hunter-team", hunter.PlayerToken, `{"name":"Wolves"}`), 200)
	state := decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", hunter.PlayerToken, ""), 200)
	if len(state.Jokers) != 1 || state.Jokers[0].Kind != "showRunners" || state.Jokers[0].Available || state.Jokers[0].UsedAt != nil {
		t.Fatalf("a hunter team has the joker, unavailable before the game runs: %+v", state.Jokers)
	}
}

func TestSketchesRefuseWhatTheRulesForbid(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, true)))
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":0,"longitude":0}`), 201)
	runner := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Ada"}`), 201)
	hunter := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Hunter+`","name":"Bob"}`), 201)
	decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/hunter-team", hunter.PlayerToken, `{"name":"Wolves"}`), 200)
	line := `{"points":[{"latitude":0,"longitude":0},{"latitude":0.001,"longitude":0.001}]}`
	many := `{"points":[` + strings.Repeat(`{"latitude":0,"longitude":0},`, 500) + `{"latitude":0,"longitude":0}]}`
	huge := `{"points":[` + strings.Repeat(`{"latitude":0,"longitude":0},`, 3000) + `{"latitude":0,"longitude":0}]}`
	for _, tc := range []struct {
		name, method, token, body string
		status                    int
	}{
		{"without token", "POST", "", line, 401},
		{"one point", "POST", hunter.PlayerToken, `{"points":[{"latitude":0,"longitude":0}]}`, 400},
		{"too many points", "POST", hunter.PlayerToken, many, 400},
		{"missing longitude", "POST", hunter.PlayerToken, `{"points":[{"latitude":0},{"latitude":1,"longitude":1}]}`, 400},
		{"point out of range", "POST", hunter.PlayerToken, `{"points":[{"latitude":85,"longitude":0},{"latitude":0,"longitude":0}]}`, 400},
		{"body too large", "POST", hunter.PlayerToken, huge, 413},
		{"before the game runs", "POST", hunter.PlayerToken, line, 409},
		{"runner draws", "POST", runner.PlayerToken, line, 409},
		{"runner clears", "DELETE", runner.PlayerToken, "", 409},
		{"clear before the game runs", "DELETE", hunter.PlayerToken, "", 409},
		{"wrong method", "GET", hunter.PlayerToken, "", 405},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if body := decode[controller.APIError](t, send(t, handler, tc.method, "/api/me/sketches", tc.token, tc.body), tc.status); body.Error == "" {
				t.Fatal("expected error message")
			}
		})
	}
	for _, token := range []string{runner.PlayerToken, hunter.PlayerToken} {
		if state := decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", token, ""), 200); state.Sketches == nil || len(state.Sketches) != 0 {
			t.Fatalf("sketches must be an empty list: %+v", state.Sketches)
		}
	}
}

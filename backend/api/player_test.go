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
	base := state.Base()
	for _, p := range base.Players {
		if p.PlayerID == base.PlayerID {
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
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":52.52,"longitude":13.405,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`), 201)
	codes := created.JoinCodes

	runner := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "",
		`{"code":" `+codes.Runner+` ","name":" Ada "}`), 201)
	runnerState, isRunner := runner.State.Runner()
	runnerBase := runner.State.Base()
	if len(runner.PlayerToken) != 64 || !isRunner || runnerState.Team != "runner" || runnerBase.JoinCodes != codes ||
		runnerBase.Status != "lobby" || runnerBase.StartsAt != nil || runnerBase.SessionID != created.Game.SessionID ||
		len(runnerBase.Players) != 1 || runnerBase.Players[0].Name != "Ada" || runnerBase.PlayerID != runnerBase.Players[0].PlayerID {
		t.Fatalf("unexpected join response: %+v", runner)
	}
	hunter := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+codes.Hunter+`","name":"Bob"}`), 201)
	if runnerState.Route != created.Game.Route || created.Game.Route.DistanceMeters <= 0 {
		t.Fatalf("runner must get the game route: %+v %+v", runnerState.Route, created.Game.Route)
	}
	if _, isRunner := hunter.State.Runner(); isRunner {
		t.Fatalf("hunter must not get the runner state: %+v", hunter.State)
	}
	if recorder := send(t, handler, "GET", "/api/me/game", hunter.PlayerToken, ""); strings.Contains(recorder.Body.String(), `"route"`) {
		t.Fatal("hunter state leaks the route")
	}
	if runnerState.RouteRegenerationsLeft != 2 {
		t.Fatalf("runner must see both route regenerations left: %+v", runnerState)
	}
	regenerated, _ := decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/route", runner.PlayerToken, ""), 200).Runner()
	if regenerated.Route == created.Game.Route || regenerated.RouteRegenerationsLeft != 1 {
		t.Fatalf("route not regenerated: %+v", regenerated)
	}
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/route", hunter.PlayerToken, ""), 409)

	stateRecorder := send(t, handler, "GET", "/api/me/game", runner.PlayerToken, "")
	state := decode[controller.GameStateResponse](t, stateRecorder, 200)
	if len(state.Base().Players) != 2 || state.Base().Playground != created.Game.Playground {
		t.Fatalf("unexpected state: %+v", state)
	}
	if state.Base().JoinCodes != codes || strings.Contains(stateRecorder.Body.String(), runner.PlayerToken) {
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
	_, isHunter := state.Hunter()
	if state.Base().Status != "countdown" || state.Base().StartsAt == nil || state.Base().ServerTime.IsZero() || !isHunter {
		t.Fatalf("countdown not started: %+v", state)
	}

	if recorder := send(t, handler, "DELETE", "/api/me", hunter.PlayerToken, ""); recorder.Code != 204 || recorder.Body.Len() != 0 {
		t.Fatalf("leave: %d %s", recorder.Code, recorder.Body)
	}
	if recorder := send(t, handler, "GET", "/api/me/game", hunter.PlayerToken, ""); recorder.Code != 401 || recorder.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("left player still authorized: %d", recorder.Code)
	}
	state = decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", runner.PlayerToken, ""), 200)
	if state.Base().Status != "lobby" || state.Base().StartsAt != nil || len(state.Base().Players) != 1 {
		t.Fatalf("leave did not cancel countdown: %+v", state)
	}
}

func TestStartPositionOverrideKeepsRealLocationUpdates(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, true)))
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`), 201)
	runner := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Ada"}`), 201)
	hunter := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Hunter+`","name":"Bob"}`), 201)
	decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/ready", runner.PlayerToken, `{"ready":true}`), 200)
	decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/hunter-team", hunter.PlayerToken, `{"name":"Wolves"}`), 200)
	countdown := decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/ready", hunter.PlayerToken, `{"ready":true}`), 200)
	for _, body := range []string{`{"latitude":1,"longitude":2,"accuracyMeters":8}`, `{"latitude":3,"longitude":4,"accuracyMeters":8}`} {
		state := decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/position", runner.PlayerToken, body), 200)
		if !state.Base().StartPositionCheckDisabled || state.Base().Status != "countdown" || !state.Base().StartsAt.Equal(*countdown.Base().StartsAt) || !ownEntry(state).Ready {
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
	return []game.Landmark{{Title: "Center", Coordinates: boundsCenter(cell)}}, nil
}

func TestLobbyErrors(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, false)))
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`), 201)
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
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`), 201)
	runner := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Ada"}`), 201)
	hunter := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Hunter+`","name":"Bob"}`), 201)
	body := `{"kind":"showRunners"}`
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", "", body), 401)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", hunter.PlayerToken, `{"kind":"teleport"}`), 400)
	for _, invalid := range []string{`{}`, `{"kind":"queryZone"}`, `{"kind":"queryZone","cell":"M1"}`, `{"kind":"queryZone","cell":"A13"}`, `{"kind":"showRunners","cell":"A1"}`, `{"kind":"showRunners","latitude":0,"longitude":0}`} {
		decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", hunter.PlayerToken, invalid), 400)
	}
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", hunter.PlayerToken, body), 409)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", runner.PlayerToken, body), 409)
	decode[controller.APIError](t, send(t, handler, "GET", "/api/me/jokers", hunter.PlayerToken, ""), 405)
	// The jokers as a client reads them.
	type gameJokers struct {
		Team   string `json:"team"`
		Jokers []struct {
			Kind      string  `json:"kind"`
			Available bool    `json:"available"`
			UsedAt    *string `json:"usedAt"`
			Found     *bool   `json:"found"`
		} `json:"jokers"`
	}
	runnerGame := decode[gameJokers](t, send(t, handler, "GET", "/api/me/game", runner.PlayerToken, ""), 200)
	if j := runnerGame.Jokers; runnerGame.Team != "runner" || len(j) != 3 || j[0].Kind != "zoneOnly" || j[1].Kind != "fakePosition" || j[2].Kind != "showHunters" || j[0].Available {
		t.Fatalf("runners have their jokers, unavailable before the game runs: %+v", runnerGame)
	}
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", runner.PlayerToken, `{"kind":"zoneOnly"}`), 409)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", runner.PlayerToken, `{"kind":"fakePosition","latitude":0,"longitude":0}`), 409)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", runner.PlayerToken, `{"kind":"showHunters"}`), 409)
	for _, invalid := range []string{
		`{"kind":"zoneOnly","cell":"A1"}`, `{"kind":"zoneOnly","latitude":0,"longitude":0}`, `{"kind":"fakePosition"}`,
		`{"kind":"fakePosition","latitude":0}`, `{"kind":"fakePosition","latitude":95,"longitude":0}`, `{"kind":"showHunters","cell":"A1"}`,
	} {
		decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", runner.PlayerToken, invalid), 400)
	}
	teamless := decode[gameJokers](t, send(t, handler, "GET", "/api/me/game", hunter.PlayerToken, ""), 200)
	if teamless.Jokers == nil || len(teamless.Jokers) != 0 {
		t.Fatalf("a hunter without a team has no jokers: %+v", teamless.Jokers)
	}
	decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/hunter-team", hunter.PlayerToken, `{"name":"Wolves"}`), 200)
	hunterGame := decode[gameJokers](t, send(t, handler, "GET", "/api/me/game", hunter.PlayerToken, ""), 200)
	if j := hunterGame.Jokers; len(j) != 3 || j[0].Kind != "showRunners" || j[1].Kind != "queryZone" || j[2].Kind != "compass" ||
		j[0].Available || j[1].Available || j[2].Available || j[0].UsedAt != nil || j[1].Found != nil {
		t.Fatalf("a hunter team has all its jokers, unavailable before the game runs: %+v", j)
	}
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", hunter.PlayerToken, `{"kind":"queryZone","cell":"C4"}`), 409)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", hunter.PlayerToken, `{"kind":"showHunters"}`), 409)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", hunter.PlayerToken, `{"kind":"compass"}`), 409)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", hunter.PlayerToken, `{"kind":"compass","cell":"A1"}`), 400)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", runner.PlayerToken, `{"kind":"compass"}`), 409)
}

func TestSketchesRefuseWhatTheRulesForbid(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, true)))
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`), 201)
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
	runnerGame := decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", runner.PlayerToken, ""), 200)
	if _, isHunter := runnerGame.Hunter(); isHunter {
		t.Fatalf("a runner has no sketches: %+v", runnerGame)
	}
	hunterGame := decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", hunter.PlayerToken, ""), 200)
	if h, _ := hunterGame.Hunter(); h.Sketches == nil || len(h.Sketches) != 0 {
		t.Fatalf("sketches must be an empty list: %+v", h.Sketches)
	}
}

// boundsCenter is the middle of the bounds.
func boundsCenter(b game.Bounds) game.Coordinates {
	c, _ := game.ParseCoordinates((b.North+b.South)/2, (b.East+b.West)/2)
	return c
}

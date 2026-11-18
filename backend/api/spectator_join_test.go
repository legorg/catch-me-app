package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"catch-me/backend/controller"
	"catch-me/backend/game"
	"catch-me/backend/service"
)

func TestDirectSpectatorJoinHTTPFlow(t *testing.T) {
	store := testStore(t)
	handler := NewHandler(controller.NewGameController(service.NewGameService(store, centerLandmarks{}, true)))
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":52.52,"longitude":13.405,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`), 201)
	codes := created.JoinCodes
	if len(codes.Spectator) != 6 || codes.Spectator == codes.Runner || codes.Spectator == codes.Hunter {
		t.Fatal("missing separate spectator invitation")
	}
	join := func(code, name string) controller.JoinGameResponse {
		return decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+code+`","name":"`+name+`"}`), 201)
	}
	viewer := join(codes.Spectator, "Viewer")
	runner := join(codes.Runner, "Ada")
	other := join(codes.Runner, "Cy")
	hunter := join(codes.Hunter, "Bob")
	checkSpectator := func(state controller.GameStateResponse, phase string) {
		t.Helper()
		v, ok := state.Spectator()
		if !ok || v.Team != "spectator" || v.Role != "spectator" || string(v.Status) != phase || len(v.Jokers) != 0 || v.JoinCodes != codes {
			t.Fatalf("incorrect direct spectator state: %+v", state.Base())
		}
		if v.Route != created.Game.Route {
			t.Fatalf("spectator did not receive the runner route: %+v", v.Route)
		}
		var raw map[string]json.RawMessage
		data, err := json.Marshal(state)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"routeRegenerationsLeft", "sketches", "history", "positions", "playerToken", "tokenHash"} {
			if _, present := raw[field]; present {
				t.Fatalf("spectator variant exposes %s", field)
			}
		}
		for _, p := range v.Players {
			if p.Team == "spectator" && (p.Ready || p.Runner != nil || p.Position != nil || p.HunterTeam != "") {
				t.Fatalf("spectator was modeled as playing: %+v", p)
			}
			if phase != "running" && (p.Position != nil || len(p.Reveals) != 0 || p.Runner != nil && p.Runner.CaughtPosition != nil) {
				t.Fatal("spectator received locations outside running phase")
			}
		}
	}
	checkSpectator(viewer.State, "lobby")
	decode[controller.APIError](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+codes.Runner+`","name":"viewer"}`), 409)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+codes.Spectator+`","name":"Injected","role":"player"}`), 400)
	for _, player := range []controller.JoinGameResponse{runner, other, hunter} {
		decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/position", player.PlayerToken, `{"latitude":52.52,"longitude":13.405,"accuracyMeters":5}`), 200)
	}
	checkSpectator(decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", viewer.PlayerToken, ""), 200), "lobby")
	decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/hunter-team", hunter.PlayerToken, `{"name":"Pack"}`), 200)
	var countdown controller.GameStateResponse
	for _, player := range []controller.JoinGameResponse{runner, other, hunter} {
		countdown = decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/ready", player.PlayerToken, `{"ready":true}`), 200)
	}
	if countdown.Base().Status != "countdown" || countdown.Base().StartsAt == nil {
		t.Fatal("unready spectator blocked countdown")
	}
	late := join(codes.Spectator, "Late")
	checkSpectator(late.State, "countdown")
	if !late.State.Base().StartsAt.Equal(*countdown.Base().StartsAt) {
		t.Fatal("spectator join restarted countdown")
	}
	if response := send(t, handler, "DELETE", "/api/me", late.PlayerToken, ""); response.Code != 204 {
		t.Fatalf("spectator could not leave before start: %d", response.Code)
	}
	decode[controller.APIError](t, send(t, handler, "GET", "/api/me/game", late.PlayerToken, ""), 401)
	stillCounting := decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", viewer.PlayerToken, ""), 200)
	if !stillCounting.Base().StartsAt.Equal(*countdown.Base().StartsAt) {
		t.Fatal("spectator leave restarted countdown")
	}
	for _, request := range []struct{ method, path, body string }{
		{"PUT", "ready", `{"ready":true}`}, {"PUT", "ready", `{"ready":false}`},
		{"PUT", "position", `{"latitude":52.52,"longitude":13.405,"accuracyMeters":5}`},
		{"PUT", "hunter-team", `{"name":"Pack"}`}, {"POST", "route", ""},
		{"POST", "caught", ""}, {"POST", "finish", ""}, {"POST", "jokers", `{"kind":"showHunters"}`},
		{"POST", "sketches", `{"points":[{"latitude":52.52,"longitude":13.405},{"latitude":52.521,"longitude":13.406}]}`},
		{"DELETE", "sketches", ""},
	} {
		decode[controller.APIError](t, send(t, handler, request.method, "/api/me/"+request.path, viewer.PlayerToken, request.body), 409)
	}
	code, _ := game.ParseJoinCode(codes.Runner)
	if _, err := store.UpdateLobbyByJoinCode(context.Background(), code, time.Now().UTC(), func(l *game.Lobby) error {
		l.Game.StartsAt = time.Now().UTC().Add(-time.Minute)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	running := decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", viewer.PlayerToken, ""), 200)
	checkSpectator(running, "running")
	for _, p := range running.Base().Players {
		if p.Team != "spectator" && (p.Position == nil || len(p.Reveals) != 0) {
			t.Fatal("direct spectator did not receive real player reports")
		}
	}
	decode[controller.APIError](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+codes.Spectator+`","name":"Too late"}`), 409)
	decode[controller.APIError](t, send(t, handler, "DELETE", "/api/me", viewer.PlayerToken, ""), 409)
	decode[controller.APIError](t, send(t, handler, "GET", "/api/games/"+string(created.Game.SessionID)+"/result", "", ""), 404)
	decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/caught", runner.PlayerToken, ""), 200)
	decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/finish", other.PlayerToken, ""), 200)
	ended := decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", viewer.PlayerToken, ""), 200)
	checkSpectator(ended, "ended")
	decode[controller.APIError](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+codes.Spectator+`","name":"Ended"}`), 409)
	result := decode[controller.GameResultResponse](t, send(t, handler, "GET", "/api/games/"+string(created.Game.SessionID)+"/result", "", ""), 200)
	var found bool
	for _, p := range result.Players {
		if p.PlayerID == viewer.State.Base().PlayerID {
			found = true
			if p.Team != "spectator" || p.Runner != nil || p.HunterTeam != "" {
				t.Fatal("result fabricated a playing team/outcome for direct spectator")
			}
		}
	}
	if !found {
		t.Fatal("result lost direct spectator identity")
	}
}

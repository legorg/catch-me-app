package api

import (
	"context"
	"testing"
	"time"

	"catch-me/backend/controller"
	"catch-me/backend/game"
	"catch-me/backend/service"
)

func TestConvertedSpectatorHTTPPermissions(t *testing.T) {
	for _, action := range []string{"caught", "finish"} {
		t.Run(action, func(t *testing.T) {
			store := testStore(t)
			handler := NewHandler(controller.NewGameController(service.NewGameService(store, centerLandmarks{}, true)))
			created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":52.52,"longitude":13.405}`), 201)
			join := func(code, name string) controller.JoinGameResponse {
				return decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+code+`","name":"`+name+`"}`), 201)
			}
			runner := join(created.JoinCodes.Runner, "Ada")
			mate := join(created.JoinCodes.Runner, "Cy")
			hunter := join(created.JoinCodes.Hunter, "Bob")
			other := join(created.JoinCodes.Hunter, "Dan")
			for i, player := range []controller.JoinGameResponse{hunter, other} {
				team := []string{"Wolves", "Foxes"}[i]
				decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/hunter-team", player.PlayerToken, `{"name":"`+team+`"}`), 200)
			}
			for _, player := range []controller.JoinGameResponse{runner, mate, hunter, other} {
				decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/position", player.PlayerToken, `{"latitude":52.52,"longitude":13.405,"accuracyMeters":5}`), 200)
			}
			if runner.State.Base().Role != controller.Role(game.RolePlayer) {
				t.Fatal("new runner already has spectator permission")
			}
			code, _ := game.ParseJoinCode(created.JoinCodes.Runner)
			if _, err := store.UpdateLobbyByJoinCode(context.Background(), code, time.Now().UTC(), func(l *game.Lobby) error {
				l.Game.StartsAt = time.Now().UTC().Add(-time.Minute)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			converted := decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/"+action, runner.PlayerToken, ""), 200)
			check := func(state controller.GameStateResponse) {
				t.Helper()
				if state.Base().Role != controller.Role(game.RoleSpectator) || state.Base().Status != "running" {
					t.Fatalf("missing spectator permission: %+v", state.Base())
				}
				for _, p := range state.Base().Players {
					if p.PlayerID == state.Base().PlayerID && action == "caught" {
						if p.Runner.CaughtPosition == nil || p.Position != nil {
							t.Fatal("lost catch marker")
						}
					} else if p.Position == nil || len(p.Reveals) != 0 {
						t.Fatalf("missing real spectator report: %+v", p)
					}
				}
			}
			check(converted)
			check(decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", runner.PlayerToken, ""), 200))
			check(decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/"+action, runner.PlayerToken, ""), 200))
			for _, player := range []controller.JoinGameResponse{mate, hunter, other} {
				state := decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", player.PlayerToken, ""), 200)
				if state.Base().Role != controller.Role(game.RolePlayer) {
					t.Fatal("active player received spectator role")
				}
				for _, p := range state.Base().Players {
					if p.Team == "hunter" && p.PlayerID != state.Base().PlayerID && p.Position != nil {
						t.Fatal("unrestricted hunter feed leaked to active player")
					}
					if action == "finish" && p.PlayerID == converted.Base().PlayerID && p.Position != nil {
						t.Fatal("finish report leaked to active player")
					}
				}
			}
			for _, request := range []struct{ method, path, body string }{
				{"PUT", "position", `{"latitude":52.52,"longitude":13.405,"accuracyMeters":5}`},
				{"PUT", "ready", `{"ready":false}`},
				{"PUT", "hunter-team", `{"name":"Wolves"}`},
				{"POST", "jokers", `{"kind":"showHunters"}`},
				{"POST", "route", ""},
				{"DELETE", "sketches", ""},
				{"POST", "sketches", `{"points":[{"latitude":52.52,"longitude":13.405},{"latitude":52.521,"longitude":13.406}]}`},
			} {
				decode[controller.APIError](t, send(t, handler, request.method, "/api/me/"+request.path, runner.PlayerToken, request.body), 409)
			}
			decode[controller.APIError](t, send(t, handler, "DELETE", "/api/me", runner.PlayerToken, ""), 409)
			opposite := "finish"
			if action == "finish" {
				opposite = "caught"
			}
			decode[controller.APIError](t, send(t, handler, "POST", "/api/me/"+opposite, runner.PlayerToken, ""), 409)
			decode[controller.APIError](t, send(t, handler, "GET", "/api/me/game", "", ""), 401)
			decode[controller.APIError](t, send(t, handler, "GET", "/api/games/"+string(created.Game.SessionID)+"/result", "", ""), 404)
			decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/caught", mate.PlayerToken, ""), 200)
			ended := decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", runner.PlayerToken, ""), 200)
			if ended.Base().Role != controller.Role(game.RoleSpectator) || ended.Base().Status != "ended" {
				t.Fatal("spectator lost game-end state")
			}
			for _, p := range ended.Base().Players {
				if p.Position != nil || len(p.Reveals) != 0 || p.Runner != nil && p.Runner.CaughtPosition != nil {
					t.Fatal("spectator live feed continued after game end")
				}
			}
		})
	}
}

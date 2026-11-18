package api

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"catch-me/backend/controller"
	"catch-me/backend/game"
	"catch-me/backend/service"
)

func TestFinishHTTPFlow(t *testing.T) {
	for _, skip := range []bool{false, true} {
		t.Run(fmt.Sprintf("override=%v", skip), func(t *testing.T) {
			store := testStore(t)
			handler := NewHandler(controller.NewGameController(service.NewGameService(store, centerLandmarks{}, skip)))
			created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":52.52,"longitude":13.405}`), 201)
			runner := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Ada"}`), 201)
			hunter := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Hunter+`","name":"Bob"}`), 201)
			for _, tc := range []struct {
				method, token string
				status        int
			}{
				{"GET", runner.PlayerToken, 405}, {"POST", "", 401}, {"POST", strings.Repeat("a", 64), 401},
				{"POST", runner.PlayerToken, 409}, {"POST", hunter.PlayerToken, 409},
			} {
				decode[controller.APIError](t, send(t, handler, tc.method, "/api/me/finish", tc.token, ""), tc.status)
			}
			code, _ := game.ParseJoinCode(created.JoinCodes.Runner)
			if _, err := store.UpdateLobbyByJoinCode(context.Background(), code, time.Now().UTC(), func(l *game.Lobby) error {
				l.Game.StartsAt = time.Now().UTC().Add(-time.Hour)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if !skip {
				// Coordinates in the finish body cannot substitute for a stored report.
				decode[controller.APIError](t, send(t, handler, "POST", "/api/me/finish", runner.PlayerToken, `{"latitude":52.52,"longitude":13.405}`), 409)
				runnerState, _ := runner.State.Runner()
				stop := runnerState.Route.Stop.Coordinates
				body := fmt.Sprintf(`{"latitude":%v,"longitude":%v,"accuracyMeters":5}`, stop.Latitude, stop.Longitude)
				decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/position", runner.PlayerToken, body), 200)
			}
			state := decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/finish", runner.PlayerToken, ""), 200)
			finished := ownEntry(state)
			if finished.Runner == nil || finished.Runner.Status != controller.RunnerStatus(game.RunnerFinished) || finished.Runner.EndedAt == nil || state.Base().Status != "running" || state.Base().StartPositionCheckDisabled != skip {
				t.Fatalf("unexpected finish state: %+v", state)
			}
			for _, token := range []string{runner.PlayerToken, hunter.PlayerToken} {
				view := decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", token, ""), 200)
				if _, ok := view.Runner(); token == hunter.PlayerToken && ok {
					t.Fatal("route leaked to hunter")
				}
				for _, p := range view.Base().Players {
					if p.PlayerID == finished.PlayerID && (p.Runner == nil || p.Runner.Status != controller.RunnerStatus(game.RunnerFinished) || p.Runner.EndedAt == nil || !p.Runner.EndedAt.Equal(*finished.Runner.EndedAt) || p.Position != nil || p.Runner.CaughtPosition != nil || len(p.Reveals) != 0) {
						t.Fatalf("finish coordinates leaked: %+v", p)
					}
				}
			}
			retry := decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/finish", runner.PlayerToken, ""), 200)
			if p := ownEntry(retry); p.Runner == nil || p.Runner.EndedAt == nil || !p.Runner.EndedAt.Equal(*finished.Runner.EndedAt) {
				t.Fatal("retry changed finish")
			}
			decode[controller.APIError](t, send(t, handler, "PUT", "/api/me/position", runner.PlayerToken, `{"latitude":0,"longitude":0,"accuracyMeters":5}`), 409)
			decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", runner.PlayerToken, `{"kind":"showHunters"}`), 409)
			decode[controller.APIError](t, send(t, handler, "POST", "/api/me/caught", runner.PlayerToken, ""), 409)
			decode[controller.APIError](t, send(t, handler, "POST", "/api/me/finish", hunter.PlayerToken, ""), 409)
		})
	}
}

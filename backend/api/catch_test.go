package api

import (
	"context"
	"strings"
	"testing"
	"time"

	"catch-me/backend/controller"
	"catch-me/backend/game"
	"catch-me/backend/service"
)

func TestMarkCaughtHTTPFlow(t *testing.T) {
	for _, withPosition := range []bool{false, true} {
		name := "without position"
		if withPosition {
			name = "with position"
		}
		t.Run(name, func(t *testing.T) {
			store := testStore(t)
			handler := NewHandler(controller.NewGameController(service.NewGameService(store, centerLandmarks{}, false)))
			created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":52.52,"longitude":13.405}`), 201)
			runner := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Ada"}`), 201)
			other := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Cy"}`), 201)
			hunter := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Hunter+`","name":"Bob"}`), 201)
			for _, tc := range []struct {
				method, token string
				status        int
			}{
				{"GET", runner.PlayerToken, 405},
				{"POST", "", 401},
				{"POST", strings.Repeat("a", 64), 401},
				{"POST", runner.PlayerToken, 409},
				{"POST", hunter.PlayerToken, 409},
			} {
				decode[controller.APIError](t, send(t, handler, tc.method, "/api/me/caught", tc.token, ""), tc.status)
			}
			var last *controller.PositionResponse
			if withPosition {
				state := decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/position", runner.PlayerToken, `{"latitude":52.52,"longitude":13.405,"accuracyMeters":8}`), 200)
				last = ownEntry(state).Position
			}
			// Start without sleeping for the countdown. No hunter ever reports a position.
			code, _ := game.ParseJoinCode(created.JoinCodes.Runner)
			if _, err := store.UpdateLobbyByJoinCode(context.Background(), code, time.Now().UTC(), func(l *game.Lobby) error {
				l.Game.StartsAt = time.Now().UTC().Add(-time.Hour)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			state := decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/caught", runner.PlayerToken, ""), 200)
			caught := ownEntry(state)
			if caught.Runner == nil || caught.Runner.Status != controller.RunnerStatus(game.RunnerCaught) || caught.Runner.EndedAt == nil || caught.Position != nil || len(caught.Reveals) != 0 {
				t.Fatalf("expected caught state without normal positions or reveals: %+v", caught)
			}
			if withPosition && (caught.Runner.CaughtPosition == nil || *caught.Runner.CaughtPosition != *last) {
				t.Fatalf("catch location must be the last report: %+v", caught)
			}
			if !withPosition && caught.Runner.CaughtPosition != nil {
				t.Fatal("catch without a position must not invent one")
			}
			// The caught player remains authenticated and gets the same restricted
			// runner view, not privileged spectator access to hidden positions.
			for _, token := range []string{runner.PlayerToken, other.PlayerToken, hunter.PlayerToken} {
				view := decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", token, ""), 200)
				for _, p := range view.Base().Players {
					if p.PlayerID == caught.PlayerID {
						if p.Runner == nil || p.Runner.Status != controller.RunnerStatus(game.RunnerCaught) || p.Runner.EndedAt == nil || !p.Runner.EndedAt.Equal(*caught.Runner.EndedAt) || p.Position != nil || len(p.Reveals) != 0 || (p.Runner.CaughtPosition != nil) != withPosition {
							t.Fatalf("all viewers must see the same catch immediately: %+v", p)
						}
					}
				}
			}
			retry := decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/caught", runner.PlayerToken, ""), 200)
			if p := ownEntry(retry); p.Runner == nil || p.Runner.EndedAt == nil || !p.Runner.EndedAt.Equal(*caught.Runner.EndedAt) {
				t.Fatal("retry changed the catch time")
			}
			decode[controller.APIError](t, send(t, handler, "PUT", "/api/me/position", runner.PlayerToken, `{"latitude":0,"longitude":0,"accuracyMeters":5}`), 409)
			decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", runner.PlayerToken, `{"kind":"showHunters"}`), 409)
			decode[controller.APIError](t, send(t, handler, "POST", "/api/me/caught", hunter.PlayerToken, ""), 409)
		})
	}
}

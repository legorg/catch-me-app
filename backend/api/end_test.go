package api

import (
	"context"
	"testing"
	"time"

	"catch-me/backend/controller"
	"catch-me/backend/game"
	"catch-me/backend/service"
)

func TestAutomaticGameEndHTTPFlow(t *testing.T) {
	for _, lastAction := range []string{"caught", "finish"} {
		t.Run(lastAction, func(t *testing.T) {
			store := testStore(t)
			handler := NewHandler(controller.NewGameController(service.NewGameService(store, centerLandmarks{}, true)))
			created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":52.52,"longitude":13.405}`), 201)
			ada := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Ada"}`), 201)
			cy := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Cy"}`), 201)
			bob := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Hunter+`","name":"Bob"}`), 201)
			decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/hunter-team", bob.PlayerToken, `{"name":"Pack"}`), 200)
			if ada.State.Base().EndedAt != nil {
				t.Fatal("lobby must not have an end time")
			}
			code, _ := game.ParseJoinCode(created.JoinCodes.Runner)
			if _, err := store.UpdateLobbyByJoinCode(context.Background(), code, time.Now().UTC(), func(l *game.Lobby) error {
				l.Game.StartsAt = time.Now().UTC().Add(-time.Hour)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			first := decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/finish", ada.PlayerToken, ""), 200)
			if first.Base().Status != "running" || first.Base().EndedAt != nil {
				t.Fatal("the first finish must not end the game")
			}
			final := decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/"+lastAction, cy.PlayerToken, ""), 200)
			end := final.Base().EndedAt
			if final.Base().Status != "ended" || end == nil || !end.Equal(final.Base().ServerTime) || final.Base().NextRevealAt != nil {
				t.Fatalf("final action must return the ended state immediately: %+v", final.Base())
			}
			for _, token := range []string{ada.PlayerToken, cy.PlayerToken, bob.PlayerToken} {
				state := decode[controller.GameStateResponse](t, send(t, handler, "GET", "/api/me/game", token, ""), 200)
				if state.Base().Status != "ended" || state.Base().EndedAt == nil || !state.Base().EndedAt.Equal(*end) || state.Base().NextRevealAt != nil {
					t.Fatal("all authenticated players must see the same game end")
				}
				decode[controller.APIError](t, send(t, handler, "PUT", "/api/me/position", token, `{"latitude":0,"longitude":0,"accuracyMeters":5}`), 409)
				decode[controller.APIError](t, send(t, handler, "PUT", "/api/me/ready", token, `{"ready":false}`), 409)
				decode[controller.APIError](t, send(t, handler, "DELETE", "/api/me", token, ""), 409)
			}
			for _, tc := range []struct{ token, action string }{{ada.PlayerToken, "finish"}, {cy.PlayerToken, lastAction}} {
				retry := decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/"+tc.action, tc.token, ""), 200)
				if retry.Base().EndedAt == nil || !retry.Base().EndedAt.Equal(*end) {
					t.Fatal("retry changed the persisted end time")
				}
			}
			decode[controller.APIError](t, send(t, handler, "POST", "/api/me/jokers", bob.PlayerToken, `{"kind":"showRunners"}`), 409)
			decode[controller.APIError](t, send(t, handler, "POST", "/api/me/sketches", bob.PlayerToken, `{"points":[{"latitude":52.52,"longitude":13.405},{"latitude":52.53,"longitude":13.415}]}`), 409)
			decode[controller.APIError](t, send(t, handler, "DELETE", "/api/me/sketches", bob.PlayerToken, ""), 409)
			decode[controller.APIError](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Late"}`), 409)
		})
	}
}

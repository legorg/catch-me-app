package api

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"catch-me/backend/controller"
	"catch-me/backend/game"
	"catch-me/backend/service"
)

func TestTimelineForSpectatorsOnly(t *testing.T) {
	store := testStore(t)
	handler := NewHandler(controller.NewGameController(service.NewGameService(store, centerLandmarks{}, true)))
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":52.52,"longitude":13.405,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`), 201)
	join := func(code, name string) controller.JoinGameResponse {
		return decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+code+`","name":"`+name+`"}`), 201)
	}
	ada := join(created.JoinCodes.Runner, "Ada")
	cy := join(created.JoinCodes.Runner, "Cy")
	bob := join(created.JoinCodes.Hunter, "Bob")
	wanda := join(created.JoinCodes.Spectator, "Wanda")
	decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/hunter-team", bob.PlayerToken, `{"name":"Wolves"}`), 200)

	type event struct {
		Kind, HunterTeam, Joker string
		At                      time.Time
		Runner                  *controller.TimelinePlayer
	}
	type timeline struct {
		ServerTime time.Time
		Events     []event
	}
	if got := decode[timeline](t, send(t, handler, "GET", "/api/me/timeline", wanda.PlayerToken, ""), 200); len(got.Events) != 0 || got.ServerTime.IsZero() {
		t.Fatalf("lobby timeline must be empty: %+v", got)
	}

	code, _ := game.ParseJoinCode(created.JoinCodes.Runner)
	if _, err := store.UpdateLobbyByJoinCode(context.Background(), code, time.Now().UTC(), func(l *game.Lobby) error {
		l.Game.StartsAt = time.Now().UTC().Add(-time.Minute)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/jokers", bob.PlayerToken, `{"kind":"showRunners"}`), 200)
	decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/jokers", cy.PlayerToken, `{"kind":"zoneOnly"}`), 200)
	decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/caught", ada.PlayerToken, ""), 200)

	for _, viewer := range []controller.JoinGameResponse{wanda, ada} {
		recorder := send(t, handler, "GET", "/api/me/timeline", viewer.PlayerToken, "")
		if body := recorder.Body.String(); strings.Contains(body, "latitude") || strings.Contains(body, "cell") {
			t.Fatalf("timeline exposes coordinates or cells: %s", body)
		}
		got := decode[timeline](t, recorder, 200)
		kinds := make([]string, 0, len(got.Events))
		for _, e := range got.Events {
			kinds = append(kinds, e.Kind)
		}
		if !slices.Equal(kinds, []string{"gameStarted", "hunterJokerUsed", "runnerJokerUsed", "runnerCaught"}) {
			t.Fatalf("unexpected events: %v", kinds)
		}
		hunterJoker, runnerJoker, caught := got.Events[1], got.Events[2], got.Events[3]
		if hunterJoker.HunterTeam != "Wolves" || hunterJoker.Joker != "showRunners" || hunterJoker.Runner != nil {
			t.Fatalf("unexpected hunter joker event: %+v", hunterJoker)
		}
		if runnerJoker.Joker != "zoneOnly" || runnerJoker.Runner == nil || runnerJoker.Runner.Name != "Cy" || runnerJoker.HunterTeam != "" {
			t.Fatalf("unexpected runner joker event: %+v", runnerJoker)
		}
		if caught.Runner == nil || caught.Runner.Name != "Ada" || caught.Runner.PlayerID != ada.State.Base().PlayerID || caught.Joker != "" {
			t.Fatalf("unexpected catch event: %+v", caught)
		}
	}

	for _, active := range []controller.JoinGameResponse{cy, bob} {
		decode[controller.APIError](t, send(t, handler, "GET", "/api/me/timeline", active.PlayerToken, ""), 409)
	}
	decode[controller.APIError](t, send(t, handler, "GET", "/api/me/timeline", "", ""), 401)
	decode[controller.APIError](t, send(t, handler, "POST", "/api/me/timeline", wanda.PlayerToken, ""), 405)
}

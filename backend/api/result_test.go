package api

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"catch-me/backend/controller"
	"catch-me/backend/game"
	"catch-me/backend/service"
)

func TestPublicGameResultHTTPFlow(t *testing.T) {
	store := testStore(t)
	handler := NewHandler(controller.NewGameController(service.NewGameService(store, centerLandmarks{}, true)))
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":52.52,"longitude":13.405}`), 201)
	path := "/api/games/" + string(created.Game.SessionID) + "/result"
	unknown := send(t, handler, "GET", "/api/games/"+game.NewSessionID().String()+"/result", "", "")
	decode[controller.APIError](t, unknown, 404)
	assertHidden := func() {
		t.Helper()
		response := send(t, handler, "GET", path, "", "")
		decode[controller.APIError](t, response, 404)
		if response.Body.String() != unknown.Body.String() {
			t.Fatal("unfinished and unknown games must be indistinguishable")
		}
	}
	assertHidden() // Empty lobby.
	ada := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Ada"}`), 201)
	cy := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Runner+`","name":"Cy"}`), 201)
	bob := decode[controller.JoinGameResponse](t, send(t, handler, "POST", "/api/players", "", `{"code":"`+created.JoinCodes.Hunter+`","name":"Bob"}`), 201)
	decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/hunter-team", bob.PlayerToken, `{"name":"Pack"}`), 200)
	for _, token := range []string{ada.PlayerToken, cy.PlayerToken, bob.PlayerToken} {
		decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/position", token, `{"latitude":52.52,"longitude":13.405,"accuracyMeters":5}`), 200)
		decode[controller.GameStateResponse](t, send(t, handler, "PUT", "/api/me/ready", token, `{"ready":true}`), 200)
	}
	assertHidden() // Countdown.
	start := time.Now().UTC().Add(-time.Hour)
	code, _ := game.ParseJoinCode(created.JoinCodes.Runner)
	if _, err := store.UpdateLobbyByJoinCode(context.Background(), code, time.Now().UTC(), func(l *game.Lobby) error {
		l.Game.StartsAt = start
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	assertHidden() // Running, including with a valid participant token.
	decode[controller.APIError](t, send(t, handler, "GET", path, ada.PlayerToken, ""), 404)
	caught := decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/caught", ada.PlayerToken, ""), 200)
	assertHidden() // A caught runner alone does not publish the result.
	finished := decode[controller.GameStateResponse](t, send(t, handler, "POST", "/api/me/finish", cy.PlayerToken, ""), 200)
	response := send(t, handler, "GET", path, "", "")
	result := decode[controller.GameResultResponse](t, response, 200)
	if response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("WWW-Authenticate") != "" {
		t.Fatal("public results must not require authentication or allow caching")
	}
	end := finished.Base().EndedAt
	if result.SessionID != created.Game.SessionID || !result.StartsAt.Equal(start) || end == nil || !result.EndedAt.Equal(*end) || result.DurationSeconds != int64(end.Sub(start)/time.Second) || len(result.Players) != 3 {
		t.Fatalf("unexpected result: %+v", result)
	}
	byName := map[string]controller.ResultPlayerResponse{}
	for _, p := range result.Players {
		byName[p.Name] = p
	}
	for _, tc := range []struct {
		name   string
		status game.RunnerStatus
		state  controller.GameStateResponse
	}{{"Ada", game.RunnerCaught, caught}, {"Cy", game.RunnerFinished, finished}} {
		p := byName[tc.name]
		original := ownEntry(tc.state)
		if p.Team != controller.Team(game.Runners) || p.Runner == nil || p.Runner.Status != controller.RunnerOutcome(tc.status) || !p.Runner.EndedAt.Equal(*original.Runner.EndedAt) || p.PlayerID != original.PlayerID || p.HunterTeam != "" {
			t.Fatalf("runner outcome changed: %+v", p)
		}
	}
	if p := byName["Bob"]; p.Team != controller.Team(game.Hunters) || p.HunterTeam != "Pack" || p.Runner != nil {
		t.Fatalf("hunter affiliation missing or runner outcome invented: %+v", p)
	}
	// An exact JSON allowlist prevents leaking any present or future gameplay
	// fields, including catch locations, route coordinates, join codes and hashes.
	raw := decode[map[string]json.RawMessage](t, response, 200)
	assertResultFields(t, raw, "sessionId", "startsAt", "endedAt", "durationSeconds", "players")
	var players []map[string]json.RawMessage
	if err := json.Unmarshal(raw["players"], &players); err != nil {
		t.Fatal(err)
	}
	for _, p := range players {
		var team string
		if err := json.Unmarshal(p["team"], &team); err != nil {
			t.Fatal(err)
		}
		if team == "runner" {
			assertResultFields(t, p, "playerId", "name", "team", "runner")
			var runner map[string]json.RawMessage
			if err := json.Unmarshal(p["runner"], &runner); err != nil {
				t.Fatal(err)
			}
			assertResultFields(t, runner, "status", "endedAt")
		} else {
			assertResultFields(t, p, "playerId", "name", "team", "hunterTeam")
		}
	}
	// The public lookup must not interpret an unrelated/invalid bearer token.
	public := decode[controller.GameResultResponse](t, send(t, handler, "GET", path, "invalid", ""), 200)
	if public.SessionID != result.SessionID {
		t.Fatal("public lookup depends on authentication")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	decode[controller.APIError](t, send(t, handler, "GET", path, "", ""), 500)
}

func TestGameResultRejectsInvalidIDsAndMethods(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, false)))
	for _, id := range []string{"123456", "invalid", strings.Repeat("a", 31), strings.Repeat("a", 33), strings.Repeat("A", 32), strings.Repeat("z", 32)} {
		decode[controller.APIError](t, send(t, handler, "GET", "/api/games/"+id+"/result", "", ""), 400)
	}
	for _, method := range []string{"POST", "PUT", "DELETE", "HEAD"} {
		response := send(t, handler, method, "/api/games/"+game.NewSessionID().String()+"/result", "", "")
		if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET" {
			t.Fatalf("unsupported method %s: %d", method, response.Code)
		}
	}
}

func assertResultFields(t *testing.T, object map[string]json.RawMessage, expected ...string) {
	t.Helper()
	if len(object) != len(expected) {
		t.Fatalf("unexpected public result fields: %v", object)
	}
	for field := range object {
		if !slices.Contains(expected, field) {
			t.Fatalf("unexpected public result field %s", field)
		}
	}
}

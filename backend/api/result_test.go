package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
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
	created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", `{"latitude":52.52,"longitude":13.405,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`), 201)
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
	assertResultFields(t, raw, "sessionId", "startsAt", "endedAt", "durationSeconds", "playground", "route", "players", "events")
	if !reflect.DeepEqual(result.Playground, created.Game.Playground) || !reflect.DeepEqual(result.Route, created.Game.Route) {
		t.Fatalf("result playground/route differ from the created game: %+v %+v", result.Playground, result.Route)
	}
	var events []map[string]json.RawMessage
	if err := json.Unmarshal(raw["events"], &events); err != nil {
		t.Fatal(err)
	}
	kinds := make([]string, 0, len(events))
	for _, e := range events {
		var kind string
		if err := json.Unmarshal(e["kind"], &kind); err != nil {
			t.Fatal(err)
		}
		kinds = append(kinds, kind)
		if kind == "runnerCaught" || kind == "runnerFinished" {
			assertResultFields(t, e, "kind", "at", "runner")
		} else {
			assertResultFields(t, e, "kind", "at")
		}
	}
	if !slices.Equal(kinds, []string{"gameStarted", "runnerCaught", "runnerFinished", "gameEnded"}) {
		t.Fatalf("result must contain the whole timeline: %v", kinds)
	}
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
			assertResultFields(t, p, "playerId", "name", "team", "runner", "track")
			var runner map[string]json.RawMessage
			if err := json.Unmarshal(p["runner"], &runner); err != nil {
				t.Fatal(err)
			}
			assertResultFields(t, runner, "status", "endedAt")
		} else {
			assertResultFields(t, p, "playerId", "name", "team", "hunterTeam", "track")
		}
		// Each player reported once in the lobby, which lies inside the backdated game.
		var track []map[string]json.RawMessage
		if err := json.Unmarshal(p["track"], &track); err != nil || len(track) != 1 {
			t.Fatalf("track missing: %s %v", p["track"], err)
		}
		assertResultFields(t, track[0], "latitude", "longitude", "accuracyMeters", "recordedAt")
	}
	if track := byName["Bob"].Track; track == nil || len(*track) != 1 || (*track)[0].Latitude != 52.52 || (*track)[0].AccuracyMeters != 5 {
		t.Fatalf("hunter track wrong: %+v", track)
	}
	// The public lookup must not interpret an unrelated/invalid bearer token.
	public := decode[controller.GameResultResponse](t, send(t, handler, "GET", path, "invalid", ""), 200)
	if public.SessionID != result.SessionID {
		t.Fatal("public lookup depends on authentication")
	}
	// Clients that accept gzip get the same JSON compressed; others (above) get it plain.
	request := httptest.NewRequest("GET", path, nil)
	request.Header.Set("Accept-Encoding", "gzip")
	compressed := httptest.NewRecorder()
	handler.ServeHTTP(compressed, request)
	if compressed.Header().Get("Content-Encoding") != "gzip" || compressed.Header().Get("Vary") != "Accept-Encoding" || response.Header().Get("Content-Encoding") != "" {
		t.Fatalf("wrong encoding headers: %v / %v", compressed.Header(), response.Header())
	}
	reader, err := gzip.NewReader(compressed.Body)
	if err != nil {
		t.Fatal(err)
	}
	if body, err := io.ReadAll(reader); err != nil || !bytes.Equal(body, response.Body.Bytes()) || compressed.Body.Len() >= response.Body.Len() {
		t.Fatalf("gzip body differs or is not smaller: %v", err)
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

func TestAcceptsGzip(t *testing.T) {
	for header, want := range map[string]bool{
		"": false, "identity": false, "gzip": true, "GZIP": true, "deflate, gzip;q=0.8": true, "br, gzip": true,
		"gzip;q=0": false, "gzip; q=0.000": false, "x-gzip": false,
	} {
		if acceptsGzip(header) != want {
			t.Errorf("acceptsGzip(%q) = %v, want %v", header, !want, want)
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

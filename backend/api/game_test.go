package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"catch-me/backend/controller"
	"catch-me/backend/db"
	domain "catch-me/backend/game"
	"catch-me/backend/service"
)

func testStore(t *testing.T) *db.Store {
	t.Helper()
	store, err := db.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestCreateGame(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, false)))
	request := httptest.NewRequest(http.MethodPost, "/api/games", strings.NewReader(`{"latitude":52.52,"longitude":13.405}`))
	request.Header.Set("Content-Type", "application/json")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, request)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body)
	}
	var createdGame controller.CreateGameResponse
	if err := json.Unmarshal(created.Body.Bytes(), &createdGame); err != nil {
		t.Fatal(err)
	}
	game := createdGame.Game
	if _, err := domain.ParseSessionID(game.SessionID); err != nil || game.CreatedAt.IsZero() {
		t.Fatalf("invalid game metadata: %+v", game)
	}
	runnerCode, runnerErr := domain.ParseJoinCode(createdGame.JoinCodes.Runner)
	hunterCode, hunterErr := domain.ParseJoinCode(createdGame.JoinCodes.Hunter)
	if runnerErr != nil || hunterErr != nil || runnerCode == hunterCode {
		t.Fatalf("invalid join codes: %+v", createdGame.JoinCodes)
	}
	center, _ := domain.ParseCoordinates(52.52, 13.405)
	p, _ := domain.NewPlayground(center, domain.DefaultCellSize(), domain.DefaultGridSize())
	want := controller.Playground{Center: controller.Coordinates{Latitude: 52.52, Longitude: 13.405}, Bounds: controller.Bounds(p.Bounds()), Rows: 6, Columns: 6, CellSizeMeters: 2000}
	if game.Playground != want {
		t.Fatalf("unexpected playground: %+v", game.Playground)
	}
}

func TestGameErrors(t *testing.T) {
	store := testStore(t)
	handler := NewHandler(controller.NewGameController(service.NewGameService(store, centerLandmarks{}, false)))
	for _, tc := range []struct {
		name, method, path, body string
		status                   int
	}{
		{"empty", "POST", "/api/games", "", 400},
		{"missing coordinate", "POST", "/api/games", `{"latitude":0}`, 400},
		{"null coordinate", "POST", "/api/games", `{"latitude":null,"longitude":0}`, 400},
		{"null body", "POST", "/api/games", `null`, 400},
		{"wrong type", "POST", "/api/games", `{"latitude":"0","longitude":0}`, 400},
		{"zero size", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":0}`, 400},
		{"negative size", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":-100}`, 400},
		{"too small", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":99}`, 400},
		{"too large", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":5100}`, 400},
		{"wrong increment", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":1550}`, 400},
		{"fractional meters", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":1500.5}`, 400},
		{"string size", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":"2000"}`, 400},
		{"grid too small", "POST", "/api/games", `{"latitude":0,"longitude":0,"gridSize":2}`, 400},
		{"grid too large", "POST", "/api/games", `{"latitude":0,"longitude":0,"gridSize":13}`, 400},
		{"zero reveal interval", "POST", "/api/games", `{"latitude":0,"longitude":0,"revealIntervalMinutes":0}`, 400},
		{"reveal interval too long", "POST", "/api/games", `{"latitude":0,"longitude":0,"revealIntervalMinutes":61}`, 400},
		{"fractional reveal interval", "POST", "/api/games", `{"latitude":0,"longitude":0,"revealIntervalMinutes":0.5}`, 400},
		{"unknown field", "POST", "/api/games", `{"latitude":0,"longitude":0,"rows":8}`, 400},
		{"two objects", "POST", "/api/games", `{"latitude":0,"longitude":0}{}`, 400},
		{"too far north", "POST", "/api/games", `{"latitude":81,"longitude":0}`, 400},
		{"longitude out of range", "POST", "/api/games", `{"latitude":0,"longitude":181}`, 400},
		{"date line", "POST", "/api/games", `{"latitude":0,"longitude":180}`, 400},
		{"large body", "POST", "/api/games", strings.Repeat(" ", 1025), 413},
		{"large trailing body", "POST", "/api/games", `{"latitude":0,"longitude":0}` + strings.Repeat(" ", 1025), 413},
		{"wrong create method", "GET", "/api/games", "", 405},
		{"removed load route", "GET", "/api/games/00112233445566778899aabbccddeeff", "", 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
			if recorder.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, tc.status, recorder.Body)
			}
			var body controller.APIError
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Error == "" {
				t.Fatalf("expected JSON error: %s", recorder.Body)
			}
		})
	}
	_ = store.Close()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("POST", "/api/games", strings.NewReader(`{"latitude":0,"longitude":0}`)))
	if recorder.Code != 500 || strings.Contains(recorder.Body.String(), "sql") {
		t.Fatalf("storage errors should be generic: %d %s", recorder.Code, recorder.Body)
	}
}

func TestCreateGameCellSizes(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, false)))
	for _, tc := range []struct {
		body string
		size int
	}{
		{`{"latitude":0,"longitude":0}`, 2000},
		{`{"latitude":0,"longitude":0,"cellSizeMeters":null}`, 2000},
		{`{"latitude":0,"longitude":0,"cellSizeMeters":100}`, 100},
		{`{"latitude":0,"longitude":0,"cellSizeMeters":1500}`, 1500},
		{`{"latitude":0,"longitude":0,"cellSizeMeters":2000}`, 2000},
		{`{"latitude":0,"longitude":0,"cellSizeMeters":5000}`, 5000},
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest("POST", "/api/games", strings.NewReader(tc.body)))
		var created controller.CreateGameResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil || recorder.Code != 201 || created.Game.Playground.CellSizeMeters != tc.size {
			t.Fatalf("unexpected size: %d %s", recorder.Code, recorder.Body)
		}
	}
}

func TestCreateGameGridSizeAndRevealInterval(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, false)))
	for _, tc := range []struct {
		body           string
		cells, minutes int
	}{
		{`{"latitude":0,"longitude":0}`, 6, 30},
		{`{"latitude":0,"longitude":0,"gridSize":null,"revealIntervalMinutes":null}`, 6, 30},
		{`{"latitude":0,"longitude":0,"gridSize":3,"revealIntervalMinutes":1}`, 3, 1},
		{`{"latitude":0,"longitude":0,"gridSize":12,"revealIntervalMinutes":60}`, 12, 60},
	} {
		created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", tc.body), 201)
		if p := created.Game.Playground; p.Rows != tc.cells || p.Columns != tc.cells || created.Game.RevealIntervalMinutes != tc.minutes {
			t.Fatalf("%s: unexpected settings: %+v", tc.body, created.Game)
		}
	}
}

// Smoke test only: ensure the generated contract describes the real routes.
func TestOpenAPIRoutes(t *testing.T) {
	data, err := os.ReadFile("openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	type schema struct {
		Ref           string `json:"$ref"`
		Required      []string
		OneOf         []schema `json:"oneOf"`
		Enum          []any
		Properties    map[string]schema
		Discriminator struct {
			PropertyName string `json:"propertyName"`
		}
	}
	var spec struct {
		OpenAPI string
		Paths   map[string]map[string]struct {
			OperationID string `json:"operationId"`
			Security    []map[string][]string
			Responses   map[string]struct {
				Content map[string]struct{ Schema schema }
			}
		}
		Components struct {
			Schemas         map[string]schema
			SecuritySchemes map[string]struct{ Type, Name, In string }
		}
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(spec.OpenAPI, "3.1") {
		t.Fatalf("expected OpenAPI 3.1, got %q", spec.OpenAPI)
	}
	if scheme := spec.Components.SecuritySchemes["PlayerToken"]; scheme.Type != "apiKey" || scheme.In != "header" || scheme.Name != "Authorization" {
		t.Fatalf("unexpected player token scheme: %+v", scheme)
	}
	const union = "ControllerGameStateResponse"
	for _, tc := range []struct{ path, method, id, status, schema string }{
		{"/health", "get", "getHealth", "200", "ControllerHealthResponse"},
		{"/api/games", "post", "createGame", "201", "ControllerCreateGameResponse"},
		{"/api/players", "post", "joinGame", "201", "ControllerJoinGameResponse"},
		{"/api/me/game", "get", "getGameState", "200", union},
		{"/api/me/ready", "put", "setReady", "200", union},
		{"/api/me/route", "post", "regenerateRoute", "200", union},
		{"/api/me/hunter-team", "put", "setHunterTeam", "200", union},
		{"/api/me/position", "put", "updatePosition", "200", union},
		{"/api/me/jokers", "post", "activateJoker", "200", union},
		{"/api/me/sketches", "post", "addSketch", "200", union},
		{"/api/me/sketches", "delete", "clearSketches", "200", union},
	} {
		operation := spec.Paths[tc.path][tc.method]
		ref := operation.Responses[tc.status].Content["application/json"].Schema.Ref
		defined := spec.Components.Schemas[tc.schema]
		if operation.OperationID != tc.id || ref != "#/components/schemas/"+tc.schema ||
			(tc.schema != union && len(defined.Required) == 0) {
			t.Fatalf("missing operation/schema: %s %s", tc.method, tc.path)
		}
		if secured := strings.HasPrefix(tc.path, "/api/me"); secured != (len(operation.Security) == 1 && len(operation.Security[0]) == 1) {
			t.Fatalf("wrong security requirement: %s %s", tc.method, tc.path)
		}
	}
	if _, ok := spec.Paths["/api/me"]["delete"].Responses["204"]; !ok {
		t.Fatal("leaving the game must respond 204")
	}

	state := spec.Components.Schemas[union]
	if len(state.OneOf) != 2 || state.Discriminator.PropertyName != "team" {
		t.Fatalf("the game state must be a oneOf discriminated by team: %+v", state)
	}
	for i, team := range []string{"runner", "hunter"} {
		name := strings.TrimPrefix(state.OneOf[i].Ref, "#/components/schemas/")
		variant := spec.Components.Schemas[name]
		if got := variant.Properties["team"].Enum; len(got) != 1 || got[0] != team || !slices.Contains(variant.Required, "team") {
			t.Fatalf("%s must have team %q only: %+v", name, team, variant.Properties["team"])
		}
	}
}

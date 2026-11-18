package api

import (
	"context"
	"encoding/json"
	"math"
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
	request := httptest.NewRequest(http.MethodPost, "/api/games", strings.NewReader(`{"latitude":52.52,"longitude":13.405,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`))
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
		{"missing coordinate", "POST", "/api/games", `{"latitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`, 400},
		{"null coordinate", "POST", "/api/games", `{"latitude":null,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`, 400},
		{"null body", "POST", "/api/games", `null`, 400},
		{"wrong type", "POST", "/api/games", `{"latitude":"0","longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`, 400},
		{"zero size", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":0,"gridSize":6,"revealIntervalMinutes":30}`, 400},
		{"negative size", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":-100,"gridSize":6,"revealIntervalMinutes":30}`, 400},
		{"too small", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":99,"gridSize":6,"revealIntervalMinutes":30}`, 400},
		{"too large", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":5100,"gridSize":6,"revealIntervalMinutes":30}`, 400},
		{"wrong increment", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":1550,"gridSize":6,"revealIntervalMinutes":30}`, 400},
		{"fractional meters", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":1500.5,"gridSize":6,"revealIntervalMinutes":30}`, 400},
		{"string size", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":"2000","gridSize":6,"revealIntervalMinutes":30}`, 400},
		{"grid too small", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":2,"revealIntervalMinutes":30}`, 400},
		{"grid too large", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":13,"revealIntervalMinutes":30}`, 400},
		{"zero reveal interval", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":0}`, 400},
		{"reveal interval too long", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":61}`, 400},
		{"fractional reveal interval", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":0.5}`, 400},
		{"unknown field", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30,"rows":8}`, 400},
		{"two objects", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}{}`, 400},
		{"too far north", "POST", "/api/games", `{"latitude":81,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`, 400},
		{"longitude out of range", "POST", "/api/games", `{"latitude":0,"longitude":181,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`, 400},
		{"date line", "POST", "/api/games", `{"latitude":0,"longitude":180,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`, 400},
		{"large body", "POST", "/api/games", strings.Repeat(" ", 1025), 413},
		{"large trailing body", "POST", "/api/games", `{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}` + strings.Repeat(" ", 1025), 413},
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
	handler.ServeHTTP(recorder, httptest.NewRequest("POST", "/api/games", strings.NewReader(`{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`)))
	if recorder.Code != 500 || strings.Contains(recorder.Body.String(), "sql") {
		t.Fatalf("storage errors should be generic: %d %s", recorder.Code, recorder.Body)
	}
}

func TestCreateGameRequiresSettings(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, false)))
	for _, field := range []string{"cellSizeMeters", "gridSize", "revealIntervalMinutes"} {
		for _, missing := range []bool{true, false} {
			name := field + "/null"
			if missing {
				name = field + "/missing"
			}
			t.Run(name, func(t *testing.T) {
				body := map[string]any{"latitude": 0, "longitude": 0, "cellSizeMeters": 2000, "gridSize": 6, "revealIntervalMinutes": 30}
				if missing {
					delete(body, field)
				} else {
					body[field] = nil
				}
				data, err := json.Marshal(body)
				if err != nil {
					t.Fatal(err)
				}
				response := decode[controller.APIError](t, send(t, handler, "POST", "/api/games", "", string(data)), 400)
				if response.Error == "" {
					t.Fatal("expected validation error")
				}
			})
		}
	}
}

func TestCreateGameCellSizes(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, false)))
	for _, tc := range []struct {
		body string
		size int
	}{
		{`{"latitude":0,"longitude":0,"cellSizeMeters":100,"gridSize":6,"revealIntervalMinutes":30}`, 100},
		{`{"latitude":0,"longitude":0,"cellSizeMeters":1500,"gridSize":6,"revealIntervalMinutes":30}`, 1500},
		{`{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`, 2000},
		{`{"latitude":0,"longitude":0,"cellSizeMeters":5000,"gridSize":6,"revealIntervalMinutes":30}`, 5000},
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
		{`{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`, 6, 30},
		{`{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":3,"revealIntervalMinutes":1}`, 3, 1},
		{`{"latitude":0,"longitude":0,"cellSizeMeters":2000,"gridSize":12,"revealIntervalMinutes":60}`, 12, 60},
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
		Type          json.RawMessage
		Default       json.RawMessage
		Minimum       *float64
		Maximum       *float64
		MultipleOf    *float64
		MaxItems      *int
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
			Parameters  []struct {
				Name, In string
				Required bool
				Schema   struct {
					Pattern              string
					MinLength, MaxLength int
				}
			}
			RequestBody json.RawMessage
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
	create := spec.Components.Schemas["ControllerCreateGameRequest"]
	for field, parse := range map[string]func(int) error{
		"cellSizeMeters":        func(v int) error { _, err := domain.ParseCellSize(v); return err },
		"gridSize":              func(v int) error { _, err := domain.ParseGridSize(v); return err },
		"revealIntervalMinutes": func(v int) error { _, err := domain.ParseRevealInterval(v); return err },
	} {
		property := create.Properties[field]
		if !slices.Contains(create.Required, field) || string(property.Type) != `"integer"` || len(property.Default) != 0 {
			t.Fatalf("create setting %s must be a required non-null integer without a server default", field)
		}
		if property.Minimum == nil || property.Maximum == nil {
			t.Fatalf("create setting %s must document its limits", field)
		}
		low, high := int(*property.Minimum), int(*property.Maximum)
		for _, v := range []int{low - 1, low, low + 1, high - 1, high, high + 1} {
			allowed := v >= low && v <= high && (property.MultipleOf == nil || math.Mod(float64(v), *property.MultipleOf) == 0)
			if (parse(v) == nil) != allowed {
				t.Errorf("create setting %s: schema and domain disagree on %d", field, v)
			}
		}
	}
	for _, tc := range []struct {
		schema, field string
		max           int
	}{
		{"ControllerGameStateRunnerResponse", "jokers", 3},
		{"ControllerGameStateHunterResponse", "jokers", 3},
		{"ControllerGameStateSpectatorResponse", "jokers", 0},
		{"ControllerGameStateHunterResponse", "sketches", 200},
	} {
		limit := spec.Components.Schemas[tc.schema].Properties[tc.field].MaxItems
		if limit == nil || *limit != tc.max {
			t.Errorf("%s.%s must document maxItems %d", tc.schema, tc.field, tc.max)
		}
	}
	const union = "ControllerGameStateResponse"
	for _, tc := range []struct {
		name   string
		values []any
	}{
		{"ControllerTeam", []any{"runner", "hunter", "spectator"}},
		{"ControllerRole", []any{"player", "spectator"}},
		{"ControllerGameStatus", []any{"lobby", "countdown", "running", "ended"}},
		{"ControllerRunnerOutcome", []any{"caught", "finished"}},
		{"ControllerJokerKind", []any{"showRunners", "queryZone", "zoneOnly", "fakePosition", "showHunters", "compass"}},
	} {
		if got := spec.Components.Schemas[tc.name].Enum; !slices.Equal(got, tc.values) {
			t.Fatalf("%s enum changed: %v", tc.name, got)
		}
	}
	player := spec.Components.Schemas["ControllerPlayerResponse"]
	if runner, ok := player.Properties["runner"]; !ok || runner.Ref != "#/components/schemas/ControllerRunnerStateResponse" || slices.Contains(player.Required, "runner") {
		t.Fatal("player schema must document runner-only state")
	}
	for _, old := range []string{"caughtAt", "caughtPosition", "finishedAt"} {
		if _, ok := player.Properties[old]; ok {
			t.Fatalf("obsolete player field %s", old)
		}
	}
	runner := spec.Components.Schemas["ControllerRunnerStateResponse"]
	status := spec.Components.Schemas[strings.TrimPrefix(runner.Properties["status"].Ref, "#/components/schemas/")]
	if !slices.Equal(status.Enum, []any{"active", "caught", "finished"}) || !slices.Contains(runner.Required, "status") {
		t.Fatalf("runner status must be required and enumerate implemented states: %+v", runner)
	}
	for _, optional := range []string{"endedAt", "caughtPosition"} {
		if _, ok := runner.Properties[optional]; !ok || slices.Contains(runner.Required, optional) {
			t.Fatalf("runner schema missing optional %s", optional)
		}
	}
	for _, tc := range []struct{ path, method, id, status, schema string }{
		{"/health", "get", "getHealth", "200", "ControllerHealthResponse"},
		{"/api/games", "post", "createGame", "201", "ControllerCreateGameResponse"},
		{"/api/games/{sessionId}/result", "get", "getGameResult", "200", "ControllerGameResultResponse"},
		{"/api/players", "post", "joinGame", "201", "ControllerJoinGameResponse"},
		{"/api/me/game", "get", "getGameState", "200", union},
		{"/api/me/timeline", "get", "getTimeline", "200", "ControllerTimelineResponse"},
		{"/api/me/caught", "post", "markCaught", "200", union},
		{"/api/me/finish", "post", "finish", "200", union},
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
	resultOperation := spec.Paths["/api/games/{sessionId}/result"]["get"]
	if len(resultOperation.Parameters) != 1 || len(resultOperation.RequestBody) != 0 {
		t.Fatal("result lookup must have one path parameter and no request body")
	}
	param := resultOperation.Parameters[0]
	if param.Name != "sessionId" || param.In != "path" || !param.Required || param.Schema.Pattern != "^[a-f0-9]{32}$" || param.Schema.MinLength != 32 || param.Schema.MaxLength != 32 {
		t.Fatalf("result lookup must document the complete game ID: %+v", param)
	}
	for _, tc := range []struct {
		name   string
		fields []string
	}{
		{"ControllerGameResultResponse", []string{"sessionId", "startsAt", "endedAt", "durationSeconds", "players"}},
		{"ControllerResultPlayerResponse", []string{"playerId", "name", "team", "hunterTeam", "runner"}},
		{"ControllerRunnerResultResponse", []string{"status", "endedAt"}},
	} {
		definition := spec.Components.Schemas[tc.name]
		if len(definition.Properties) != len(tc.fields) {
			t.Fatalf("public result schema has unexpected fields: %s", tc.name)
		}
		for field := range definition.Properties {
			if !slices.Contains(tc.fields, field) {
				t.Fatalf("public result schema exposes %s in %s", field, tc.name)
			}
		}
	}

	state := spec.Components.Schemas[union]
	if len(state.OneOf) != 3 || state.Discriminator.PropertyName != "team" {
		t.Fatalf("the game state must be a oneOf discriminated by team: %+v", state)
	}
	for i, team := range []string{"runner", "hunter", "spectator"} {
		name := strings.TrimPrefix(state.OneOf[i].Ref, "#/components/schemas/")
		variant := spec.Components.Schemas[name]
		if got := variant.Properties["team"].Enum; len(got) != 1 || got[0] != team || !slices.Contains(variant.Required, "team") {
			t.Fatalf("%s must have team %q only: %+v", name, team, variant.Properties["team"])
		}
		if _, ok := variant.Properties["endedAt"]; !ok || slices.Contains(variant.Required, "endedAt") {
			t.Fatalf("%s must document the optional game end time", name)
		}
		if !slices.Contains(variant.Required, "role") {
			t.Fatalf("%s must document viewer permission", name)
		}
		if team != "hunter" {
			if _, ok := variant.Properties["route"]; !ok || !slices.Contains(variant.Required, "route") {
				t.Fatalf("%s must document the required runner route", name)
			}
		} else if _, ok := variant.Properties["route"]; ok {
			t.Fatal("hunter schema exposes the runner route")
		}
		if team == "spectator" {
			for _, hidden := range []string{"routeRegenerationsLeft", "sketches"} {
				if _, ok := variant.Properties[hidden]; ok {
					t.Fatalf("direct spectator schema exposes %s", hidden)
				}
			}
		}
	}
	codes := spec.Components.Schemas["ControllerJoinCodes"]
	if _, ok := codes.Properties["spectator"]; !ok || !slices.Contains(codes.Required, "spectator") {
		t.Fatal("spectator code must be documented and required")
	}
	events := spec.Components.Schemas["ControllerTimelineEventResponse"]
	if len(events.OneOf) != 6 || events.Discriminator.PropertyName != "kind" {
		t.Fatalf("timeline events must be a oneOf discriminated by kind: %+v", events)
	}
	for _, variant := range events.OneOf {
		definition := spec.Components.Schemas[strings.TrimPrefix(variant.Ref, "#/components/schemas/")]
		if len(definition.Required) != len(definition.Properties) {
			t.Fatalf("timeline event %s must have only required fields", variant.Ref)
		}
		for field := range definition.Properties {
			if !slices.Contains([]string{"kind", "at", "hunterTeam", "runner", "joker"}, field) {
				t.Fatalf("timeline event %s exposes %s", variant.Ref, field)
			}
		}
	}
}

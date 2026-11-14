package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"catch-me/backend/internal/controller"
	"catch-me/backend/internal/db"
	domain "catch-me/backend/internal/game"
	"catch-me/backend/internal/service"
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
	handler := NewHandler("http://localhost:8081", controller.NewGameController(service.NewGameService(testStore(t), false)))
	request := httptest.NewRequest(http.MethodPost, "/api/games", strings.NewReader(`{"latitude":52.52,"longitude":13.405}`))
	request.Header.Set("Origin", "http://localhost:8081")
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
	if _, err := domain.ParseSessionID(game.SessionID); err != nil || game.CreatedAt == "" {
		t.Fatalf("invalid game metadata: %+v", game)
	}
	runnerCode, runnerErr := domain.ParseJoinCode(createdGame.JoinCodes.Runner)
	hunterCode, hunterErr := domain.ParseJoinCode(createdGame.JoinCodes.Hunter)
	if runnerErr != nil || hunterErr != nil || runnerCode == hunterCode {
		t.Fatalf("invalid join codes: %+v", createdGame.JoinCodes)
	}
	center, _ := domain.ParseCoordinates(52.52, 13.405)
	p, _ := domain.NewPlayground(center, domain.DefaultCellSize())
	want := controller.Playground{Center: controller.Coordinates{Latitude: 52.52, Longitude: 13.405}, Bounds: controller.Bounds(p.Bounds()), Rows: 6, Columns: 6, CellSizeMeters: 2000}
	if game.Playground != want {
		t.Fatalf("unexpected playground: %+v", game.Playground)
	}
	preflight := httptest.NewRequest(http.MethodOptions, "/api/me/ready", nil)
	preflight.Header.Set("Origin", "http://localhost:8081")
	preflight.Header.Set("Access-Control-Request-Headers", "authorization, content-type")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, preflight)
	if recorder.Code != 204 || !strings.Contains(recorder.Header().Get("Access-Control-Allow-Methods"), "PUT") ||
		recorder.Header().Get("Access-Control-Allow-Headers") != "Authorization, Content-Type" {
		t.Fatal("authenticated JSON preflight not supported")
	}
}

func TestGameErrors(t *testing.T) {
	store := testStore(t)
	handler := NewHandler("", controller.NewGameController(service.NewGameService(store, false)))
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
	handler := NewHandler("", controller.NewGameController(service.NewGameService(testStore(t), false)))
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

// Smoke test only: ensure the generated contract describes the real routes.
func TestOpenAPIRoutes(t *testing.T) {
	data, err := os.ReadFile("../../api/swagger.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Swagger string
		Paths   map[string]map[string]struct {
			OperationID string `json:"operationId"`
			Responses   map[string]struct {
				Schema struct {
					Ref string `json:"$ref"`
				}
			}
		}
		Definitions map[string]struct{ Required []string }
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Swagger != "2.0" {
		t.Fatal("expected Swagger 2.0")
	}
	for _, tc := range []struct{ path, method, id, status, schema string }{
		{"/health", "get", "getHealth", "200", "HealthResponse"},
		{"/api/games", "post", "createGame", "201", "CreateGameResponse"},
		{"/api/players", "post", "joinGame", "201", "JoinGameResponse"},
		{"/api/me/game", "get", "getGameState", "200", "GameStateResponse"},
		{"/api/me/ready", "put", "setReady", "200", "GameStateResponse"},
		{"/api/me/route", "post", "regenerateRoute", "200", "GameStateResponse"},
		{"/api/me/position", "put", "updatePosition", "200", "GameStateResponse"},
	} {
		operation := spec.Paths[tc.path][tc.method]
		if operation.OperationID != tc.id || operation.Responses[tc.status].Schema.Ref != "#/definitions/"+tc.schema || len(spec.Definitions[tc.schema].Required) == 0 {
			t.Fatalf("missing operation/schema: %s %s", tc.method, tc.path)
		}
	}
}

package api

import (
	"context"
	"encoding/json"
	"testing"

	"catch-me/backend/controller"
	"catch-me/backend/game"
	"catch-me/backend/service"
)

const defaultJokersJSON = `{"showRunners":{"allowance":1},"queryZone":{"allowance":1},"compass":{"allowance":1},"trap":{"allowance":1},"zoneOnly":{"allowance":1},"fakePosition":{"allowance":1},"showHunters":{"allowance":1}}`

func gameBodyWithJokers(t *testing.T, jokers any) string {
	t.Helper()
	body := map[string]any{
		"latitude": 52.52, "longitude": 13.405, "cellSizeMeters": 2000,
		"gridSize": 6, "revealIntervalMinutes": 30, "jokers": jokers,
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func defaultJokerSettings(t *testing.T) map[string]any {
	t.Helper()
	var settings map[string]any
	if err := json.Unmarshal([]byte(defaultJokersJSON), &settings); err != nil {
		t.Fatal(err)
	}
	return settings
}

func TestCreateGameRequiresEveryJokerAllowance(t *testing.T) {
	handler := NewHandler(controller.NewGameController(service.NewGameService(testStore(t), centerLandmarks{}, false)))
	decode[controller.APIError](t, send(t, handler, "POST", "/api/games", "", `{"latitude":52.52,"longitude":13.405,"cellSizeMeters":2000,"gridSize":6,"revealIntervalMinutes":30}`), 400)
	for _, jokers := range []any{nil, "wrong type", []any{}, map[string]any{}} {
		decode[controller.APIError](t, send(t, handler, "POST", "/api/games", "", gameBodyWithJokers(t, jokers)), 400)
	}
	for _, kind := range []string{"showRunners", "queryZone", "compass", "trap", "zoneOnly", "fakePosition", "showHunters"} {
		for _, invalid := range []string{"missing kind", "null kind", "missing allowance", "null allowance", "negative", "too many", "fractional", "string", "unknown field"} {
			t.Run(kind+"/"+invalid, func(t *testing.T) {
				settings := defaultJokerSettings(t)
				switch invalid {
				case "missing kind":
					delete(settings, kind)
				case "null kind":
					settings[kind] = nil
				case "missing allowance":
					settings[kind] = map[string]any{}
				case "null allowance":
					settings[kind] = map[string]any{"allowance": nil}
				case "negative":
					settings[kind] = map[string]any{"allowance": -1}
				case "too many":
					settings[kind] = map[string]any{"allowance": 6}
				case "fractional":
					settings[kind] = map[string]any{"allowance": 1.5}
				case "string":
					settings[kind] = map[string]any{"allowance": "1"}
				case "unknown field":
					settings[kind] = map[string]any{"allowance": 1, "radiusMeters": 100}
				}
				decode[controller.APIError](t, send(t, handler, "POST", "/api/games", "", gameBodyWithJokers(t, settings)), 400)
			})
		}
	}
	settings := defaultJokerSettings(t)
	settings["teleport"] = map[string]any{"allowance": 1}
	decode[controller.APIError](t, send(t, handler, "POST", "/api/games", "", gameBodyWithJokers(t, settings)), 400)
}

func TestCreateGamePersistsExplicitJokerAllowances(t *testing.T) {
	store := testStore(t)
	handler := NewHandler(controller.NewGameController(service.NewGameService(store, centerLandmarks{}, false)))
	kinds := []game.JokerKind{game.JokerShowRunners, game.JokerQueryZone, game.JokerCompass, game.JokerTrap, game.JokerZoneOnly, game.JokerFakePosition, game.JokerShowHunters}
	for _, allZero := range []bool{false, true} {
		settings := defaultJokerSettings(t)
		for i, kind := range kinds {
			uses := i % 6
			if allZero {
				uses = 0
			}
			settings[string(kind)] = map[string]any{"allowance": uses}
		}
		created := decode[controller.CreateGameResponse](t, send(t, handler, "POST", "/api/games", "", gameBodyWithJokers(t, settings)), 201)
		code, _ := game.ParseJoinCode(created.JoinCodes.Runner)
		lobby, err := store.UpdateLobbyByJoinCode(context.Background(), code, created.Game.CreatedAt, nil)
		if err != nil {
			t.Fatal(err)
		}
		for i, kind := range kinds {
			want := i % 6
			if allZero {
				want = 0
			}
			if got := lobby.Game.JokerSettings.Allowance(kind).Max(); got != want {
				t.Fatalf("%s allowance: got %d, want %d", kind, got, want)
			}
		}
	}
}

package controller

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"catch-me/backend/game"
)

// The kind constants and the enum tags of the variants cannot reference each
// other, so every domain event is mapped to its own variant with its tag's kind.
func TestTimelineEventKindsMatchTheirVariants(t *testing.T) {
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	name, _ := game.ParsePlayerName("Ada")
	team, _ := game.ParseHunterTeamName("Wolves")
	runner := game.Player{ID: game.NewPlayerID(), Name: name, Team: game.Runners}
	lobby := game.Lobby{Players: []game.Player{runner}}
	// In the order of JSONSchemaOneOf.
	events := []game.Event{
		game.StartEvent{At: at},
		game.HunterJokerEvent{Team: team, Joker: game.JokerShowRunners, At: at},
		game.RunnerJokerEvent{Runner: runner.ID, Joker: game.JokerZoneOnly, At: at},
		game.CatchEvent{Runner: runner.ID, At: at},
		game.FinishEvent{Runner: runner.ID, At: at},
		game.EndEvent{At: at},
	}
	variants := TimelineEventResponse{}.JSONSchemaOneOf()
	if len(events) != len(variants) {
		t.Fatalf("%d events for %d variants", len(events), len(variants))
	}
	for i, event := range events {
		mapped, err := timelineEventResponse(lobby, event)
		if err != nil {
			t.Fatal(err)
		}
		variant := reflect.TypeOf(variants[i])
		if reflect.TypeOf(mapped) != variant {
			t.Fatalf("%T mapped to %T, want %s", event, mapped, variant)
		}
		data, err := json.Marshal(TimelineEventResponse{mapped})
		if err != nil {
			t.Fatal(err)
		}
		var probe struct{ Kind string }
		if err := json.Unmarshal(data, &probe); err != nil {
			t.Fatal(err)
		}
		field, _ := variant.FieldByName("Kind")
		if want := field.Tag.Get("enum"); probe.Kind != want {
			t.Fatalf("%s has kind %q, its schema says %q", variant.Name(), probe.Kind, want)
		}
	}
	if _, err := timelineEventResponse(game.Lobby{}, game.CatchEvent{Runner: runner.ID, At: at}); err == nil {
		t.Fatal("an event of an unknown player must not lose its runner")
	}
}

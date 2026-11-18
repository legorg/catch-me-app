package controller

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"catch-me/backend/game"
)

// The kind constants and the enum tags of the variants cannot reference each
// other, so every domain reveal is mapped to its own variant with its tag's kind.
func TestRevealKindsMatchTheirVariants(t *testing.T) {
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	point, _ := game.ParseCoordinates(52.52, 13.405)
	position, _ := game.NewPosition(point, 5, at)
	cell, _ := game.ParseCell("C4")
	// In the order of JSONSchemaOneOf.
	reveals := []game.HunterReveal{
		game.PositionReveal{At: at, Position: position},
		game.ZoneReveal{At: at, Cell: cell},
		game.OutsideReveal{At: at},
	}
	variants := RevealResponse{}.JSONSchemaOneOf()
	if len(reveals) != len(variants) {
		t.Fatalf("%d reveals for %d variants", len(reveals), len(variants))
	}
	for i, reveal := range reveals {
		mapped := revealResponse(reveal)
		variant := reflect.TypeOf(variants[i])
		if reflect.TypeOf(mapped.variant) != variant {
			t.Fatalf("%T mapped to %T, want %s", reveal, mapped.variant, variant)
		}
		data, err := json.Marshal(mapped)
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
}

package controller

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"catch-me/backend/game"
)

// The tags only document the limits (in the OpenAPI document); the domain
// enforces them. This keeps both in step.
func TestCreateGameTagsMatchTheDomain(t *testing.T) {
	for field, domain := range map[string]struct {
		parse    func(int) error
		fallback int
	}{
		"CellSizeMeters":        {func(v int) error { _, err := game.ParseCellSize(v); return err }, game.DefaultCellSize().Meters()},
		"GridSize":              {func(v int) error { _, err := game.ParseGridSize(v); return err }, game.DefaultGridSize().Cells()},
		"RevealIntervalMinutes": {func(v int) error { _, err := game.ParseRevealInterval(v); return err }, game.DefaultRevealInterval().Minutes()},
	} {
		f, ok := reflect.TypeFor[CreateGameRequest]().FieldByName(field)
		if !ok {
			t.Fatalf("CreateGameRequest has no field %s", field)
		}
		rules := map[string]int{}
		for _, rule := range strings.Split(f.Tag.Get("validate"), ",") {
			if name, value, ok := strings.Cut(rule, "="); ok {
				rules[name], _ = strconv.Atoi(value)
			}
		}
		lowest, highest := rules["gte"], rules["lte"]
		if f.Tag.Get("minimum") != strconv.Itoa(lowest) || f.Tag.Get("maximum") != strconv.Itoa(highest) {
			t.Errorf("%s: minimum/maximum tags differ from validate %q", field, f.Tag.Get("validate"))
		}
		if f.Tag.Get("default") != strconv.Itoa(domain.fallback) {
			t.Errorf("%s: default tag %q, domain default %d", field, f.Tag.Get("default"), domain.fallback)
		}
		if domain.parse(lowest) != nil || domain.parse(highest) != nil {
			t.Errorf("%s: the domain rejects the documented limits %d and %d", field, lowest, highest)
		}
		if domain.parse(lowest-1) == nil || domain.parse(highest+1) == nil {
			t.Errorf("%s: the domain accepts values beyond the documented limits %d and %d", field, lowest, highest)
		}
	}
}

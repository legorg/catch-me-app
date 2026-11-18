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
	for field, parse := range map[string]func(int) error{
		"CellSizeMeters":        func(v int) error { _, err := game.ParseCellSize(v); return err },
		"GridSize":              func(v int) error { _, err := game.ParseGridSize(v); return err },
		"RevealIntervalMinutes": func(v int) error { _, err := game.ParseRevealInterval(v); return err },
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
		if parse(lowest) != nil || parse(highest) != nil {
			t.Errorf("%s: the domain rejects the documented limits %d and %d", field, lowest, highest)
		}
		if parse(lowest-1) == nil || parse(highest+1) == nil {
			t.Errorf("%s: the domain accepts values beyond the documented limits %d and %d", field, lowest, highest)
		}
	}
}

func TestCoordinatesResponse(t *testing.T) {
	for _, want := range []Coordinates{
		{Latitude: 52.52, Longitude: 13.405},
		{Latitude: -33.8688, Longitude: -70.6693},
		{},
	} {
		point, err := game.ParseCoordinates(want.Latitude, want.Longitude)
		if err != nil {
			t.Fatal(err)
		}
		if got := coordinatesResponse(point); got != want {
			t.Fatalf("coordinates = %+v, want %+v", got, want)
		}
	}
}

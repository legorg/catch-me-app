package game

import (
	"errors"
	"testing"
)

func TestParseJokerAllowance(t *testing.T) {
	for _, max := range []int{-1, 0, 1, 2, 3, 4, 5, 6} {
		allowance, err := ParseJokerAllowance(max)
		if max < 0 || max > 5 {
			var validation *ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("allowance %d: expected validation error, got %v", max, err)
			}
		} else if err != nil || allowance.Max() != max {
			t.Fatalf("allowance %d: got %d, %v", max, allowance.Max(), err)
		}
	}
}

func TestJokerSettingsDefaultsAndZero(t *testing.T) {
	defaults := DefaultJokerSettings()
	var zero JokerSettings
	kinds := append(append([]JokerKind{}, hunterJokers...), runnerJokers...)
	if len(zero.allowances()) != len(kinds) {
		t.Fatal("every joker kind needs exactly one allowance field")
	}
	for _, kind := range kinds {
		if defaults.Allowance(kind).Max() != 1 || zero.Allowance(kind).Max() != 0 {
			t.Fatalf("%s: defaults must allow one use; zero must disable", kind)
		}
	}
	if defaults.Allowance("unknown").Max() != 0 {
		t.Fatal("unknown joker must not receive an allowance")
	}
	if !defaults.ZoneAlert.Enabled || zero.ZoneAlert.Enabled {
		t.Fatal("the zone alert must be on by default and off in the zero value")
	}
}

func TestJokerSettingsAllowancesAreIndependent(t *testing.T) {
	s := JokerSettings{
		ShowRunners: JokerAllowance{max: 0}, QueryZone: JokerAllowance{max: 1},
		Compass: JokerAllowance{max: 2}, Trap: TrapSettings{JokerAllowance: JokerAllowance{max: 3}},
		ZoneOnly: JokerAllowance{max: 4}, FakePosition: JokerAllowance{max: 5},
		ShowHunters: JokerAllowance{max: 2},
	}
	want := map[JokerKind]int{
		JokerShowRunners: 0, JokerQueryZone: 1, JokerCompass: 2, JokerTrap: 3,
		JokerZoneOnly: 4, JokerFakePosition: 5, JokerShowHunters: 2,
	}
	for kind, max := range want {
		if got := s.Allowance(kind).Max(); got != max {
			t.Fatalf("%s allowance: got %d, want %d", kind, got, max)
		}
	}
}

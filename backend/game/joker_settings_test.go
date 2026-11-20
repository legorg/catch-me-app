package game

import (
	"errors"
	"testing"
)

func TestParseJokerAllowance(t *testing.T) {
	for _, uses := range []int{-1, 0, 1, 2, 3, 4, 5, 6} {
		allowance, err := ParseJokerAllowance(uses)
		if uses < 0 || uses > 5 {
			var validation *ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("allowance %d: expected validation error, got %v", uses, err)
			}
		} else if err != nil || allowance.Uses() != uses {
			t.Fatalf("allowance %d: got %d, %v", uses, allowance.Uses(), err)
		}
	}
}

func TestJokerSettingsDefaultsAndZero(t *testing.T) {
	defaults := DefaultJokerSettings()
	var zero JokerSettings
	for _, kind := range append(append([]JokerKind{}, hunterJokers...), runnerJokers...) {
		if defaults.Allowance(kind).Uses() != 1 || zero.Allowance(kind).Uses() != 0 {
			t.Fatalf("%s: defaults must allow one use; zero must disable", kind)
		}
	}
	if defaults.Allowance("unknown").Uses() != 0 {
		t.Fatal("unknown joker must not receive an allowance")
	}
}

func TestJokerSettingsAllowancesAreIndependent(t *testing.T) {
	s := JokerSettings{
		ShowRunners: JokerAllowance{uses: 0}, QueryZone: JokerAllowance{uses: 1},
		Compass: JokerAllowance{uses: 2}, Trap: TrapSettings{JokerAllowance: JokerAllowance{uses: 3}},
		ZoneOnly: JokerAllowance{uses: 4}, FakePosition: JokerAllowance{uses: 5},
		ShowHunters: JokerAllowance{uses: 2},
	}
	want := map[JokerKind]int{
		JokerShowRunners: 0, JokerQueryZone: 1, JokerCompass: 2, JokerTrap: 3,
		JokerZoneOnly: 4, JokerFakePosition: 5, JokerShowHunters: 2,
	}
	for kind, uses := range want {
		if got := s.Allowance(kind).Uses(); got != uses {
			t.Fatalf("%s allowance: got %d, want %d", kind, got, uses)
		}
	}
}

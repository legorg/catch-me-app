package game

import (
	"errors"
	"testing"
	"time"
)

func TestTrapJokerRules(t *testing.T) {
	l, runner, bob, dan, eve, running := jokerLobby(t)
	at := running.Add(time.Minute)
	point := cellCenter(l.Game.Playground, Cell{column: 2, row: 3})
	outside, _ := ParseCoordinates(0, 0)
	var rule *RuleError
	var validation *ValidationError
	if err := l.UseJoker(bob.ID, Trap{Point: outside}, at); !errors.As(err, &validation) {
		t.Fatalf("the point must lie inside the playground: %v", err)
	}
	if err := l.UseJoker(runner.ID, Trap{Point: point}, at); !errors.As(err, &rule) {
		t.Fatalf("runners have no trap: %v", err)
	}
	if err := l.UseJoker(bob.ID, Trap{Point: point}, at); err != nil {
		t.Fatal(err)
	}
	if err := l.UseJoker(dan.ID, Trap{Point: point}, at.Add(time.Second)); !errors.As(err, &rule) {
		t.Fatalf("a hunter team can place one trap: %v", err)
	}
	got := l.JokersFor(dan, at)[3]
	if len(got.Uses) != 1 {
		t.Fatalf("expected one placed trap: %+v", got)
	}
	if use, ok := got.Uses[0].(TrapUse); got.Available || !ok || use != (TrapUse{Team: bob.HunterTeam, At: at, Point: point}) {
		t.Fatalf("a teammate sees the placed, not yet triggered trap: %+v", got)
	}
	if !got.Uses[0].(TrapUse).ActiveAt().Equal(at.Add(20 * time.Minute)) {
		t.Fatal("a trap becomes active 20 minutes after placement")
	}
	if other := l.JokersFor(eve, at)[3]; !other.Available || len(other.Uses) != 0 {
		t.Fatalf("another hunter team sees nothing of it: %+v", other)
	}
	for _, j := range l.JokersFor(runner, at) {
		if j.Kind == JokerTrap {
			t.Fatalf("runners are not told: %+v", j)
		}
	}
	if err := l.UseJoker(eve.ID, Trap{Point: point}, at); err != nil {
		t.Fatalf("another hunter team has its own trap: %v", err)
	}
}

func TestTrapTriggersOnTheFirstReportNearbyAfterActivation(t *testing.T) {
	point, _ := ParseCoordinates(48, 11)
	activeAt := time.Date(2026, 11, 8, 12, 20, 0, 0, time.UTC)
	report := func(northMeters float64, at time.Time) JokerPosition {
		// One degree of latitude is about 111,195 m.
		c, _ := ParseCoordinates(48+northMeters/111195, 11)
		return JokerPosition{Position: Position{Coordinates: c, RecordedAt: at}}
	}
	for _, tc := range []struct {
		name    string
		reports []JokerPosition
		want    time.Time
	}{
		{"no reports", nil, time.Time{}},
		{"only before activation", []JokerPosition{report(0, activeAt.Add(-time.Second))}, time.Time{}},
		{"outside the radius", []JokerPosition{report(110, activeAt.Add(time.Minute))}, time.Time{}},
		{"at activation", []JokerPosition{report(0, activeAt)}, activeAt},
		{"earliest inside wins, in any order", []JokerPosition{
			report(50, activeAt.Add(3*time.Minute)),
			report(200, activeAt.Add(time.Minute)),
			report(90, activeAt.Add(2*time.Minute)),
		}, activeAt.Add(2 * time.Minute)},
	} {
		if got := TrapTriggeredAt(point, activeAt, tc.reports); !got.Equal(tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestTrapSearchBoundsContainTheRadius(t *testing.T) {
	point, _ := ParseCoordinates(48, 11)
	b := TrapSearchBounds(point)
	if !b.Contains(48+99.0/111195, 11) || b.Contains(48+150.0/111195, 11) {
		t.Fatalf("bounds must contain the 100 m radius and little more: %+v", b)
	}
}

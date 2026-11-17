package game

import (
	"errors"
	"testing"
)

func TestCellParsing(t *testing.T) {
	for _, label := range []string{"A1", "F6", "C4"} {
		if cell, err := ParseCell(label); err != nil || cell.String() != label {
			t.Fatalf("cell %s did not round trip", label)
		}
	}
	var validation *ValidationError
	for _, label := range []string{"", "A", "A0", "A7", "G1", "a1", "A10"} {
		if _, err := ParseCell(label); !errors.As(err, &validation) {
			t.Fatalf("invalid cell accepted: %q", label)
		}
	}
	for _, used := range []int{-1, MaxRouteRegenerations + 1} {
		if _, err := ParseRouteRegenerations(used); !errors.As(err, &validation) {
			t.Fatalf("invalid route regeneration count accepted: %d", used)
		}
	}
}

// routeVia is a route that starts at the given landmark title.
func routeVia(l Lobby, title string) Route {
	route := l.Game.Route
	route.Start.Title = title
	return route
}

func TestRegenerateRoute(t *testing.T) {
	l := newLobby()
	runner := join(t, &l, Runners, "Ada", start)
	mate := join(t, &l, Runners, "Cy", start)
	hunter := join(t, &l, Hunters, "Bob", start)
	var rule *RuleError
	if err := l.RegenerateRoute(runner.ID, l.Game.Route, start); !errors.As(err, &rule) || l.Game.RouteRegenerations != 0 {
		t.Fatal("unchanged route accepted")
	}
	if err := l.RegenerateRoute(hunter.ID, routeVia(l, "Other"), start); !errors.As(err, &rule) {
		t.Fatal("hunter changed the route")
	}
	// Every runner may change it; the count is shared.
	for i, id := range []PlayerID{runner.ID, mate.ID} {
		next := routeVia(l, string(rune('A'+i)))
		if err := l.RegenerateRoute(id, next, start); err != nil {
			t.Fatal(err)
		}
		if l.Game.Route != next || l.Game.RouteRegenerationsLeft() != MaxRouteRegenerations-1-i {
			t.Fatalf("route not replaced: %+v", l.Game)
		}
	}
	if err := l.CanRegenerateRoute(runner.ID, start); !errors.As(err, &rule) {
		t.Fatal("regeneration beyond the limit accepted")
	}

	l = newLobby()
	runner = join(t, &l, Runners, "Ada", start)
	mate = join(t, &l, Runners, "Cy", start)
	_ = l.SetReady(mate.ID, true, start, false)
	if err := l.CanRegenerateRoute(runner.ID, start); !errors.As(err, &rule) {
		t.Fatal("route changed while a runner is ready")
	}
	hunter = join(t, &l, Hunters, "Bob", start)
	_ = l.SetReady(hunter.ID, true, start, false)
	_ = l.SetReady(runner.ID, true, start, false)
	if l.Game.Status(start.Add(CountdownDuration)) != StatusRunning {
		t.Fatal("game did not start")
	}
	if err := l.RegenerateRoute(runner.ID, routeVia(l, "Other"), start.Add(CountdownDuration)); !errors.As(err, &rule) {
		t.Fatal("route changed after the start")
	}
}

package game

import (
	"errors"
	"testing"
)

func TestRandomRoutesConnectOppositeEdges(t *testing.T) {
	starts := map[int]bool{}
	for range 500 {
		route := randomRoute(MaxRouteRegenerations)
		parsed, err := ParseRunnerRoute(route.Start.String(), route.End.String(), route.RegenerationsLeft)
		if err != nil || parsed != route {
			t.Fatalf("invalid route %v: %v", route, err)
		}
		starts[route.Start.row] = true
	}
	if !starts[0] || !starts[GridSize-1] || len(starts) != 2 {
		t.Fatalf("routes must start on both edges and only there: %v", starts)
	}
}

func TestCellAndRouteParsing(t *testing.T) {
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
	for _, tc := range []struct {
		start, end string
		left       int
	}{{"A2", "B5", 2}, {"A1", "B1", 2}, {"A1", "B5", 2}, {"A1", "B6", 3}, {"A1", "B6", -1}} {
		if _, err := ParseRunnerRoute(tc.start, tc.end, tc.left); !errors.As(err, &validation) {
			t.Fatalf("invalid route accepted: %+v", tc)
		}
	}
	if _, err := ParseRunnerRoute("F6", "A1", 0); err != nil {
		t.Fatal("south-to-north route rejected")
	}
}

func TestRunnersGetRoutesHuntersDoNot(t *testing.T) {
	l := newLobby()
	runner := join(t, &l, Runners, "Ada", start)
	hunter := join(t, &l, Hunters, "Bob", start)
	if runner.Route.IsZero() || runner.Route.RegenerationsLeft != MaxRouteRegenerations || !hunter.Route.IsZero() {
		t.Fatalf("unexpected routes: %+v %+v", runner.Route, hunter.Route)
	}
}

func TestRegenerateRoute(t *testing.T) {
	l := newLobby()
	runner := join(t, &l, Runners, "Ada", start)
	hunter := join(t, &l, Hunters, "Bob", start)
	previous := runner.Route
	for left := MaxRouteRegenerations - 1; left >= 0; left-- {
		if err := l.RegenerateRoute(runner.ID, start); err != nil {
			t.Fatal(err)
		}
		current, _ := l.Player(runner.ID)
		if current.Route.RegenerationsLeft != left || (current.Route.Start == previous.Start && current.Route.End == previous.End) {
			t.Fatalf("route not regenerated: %+v after %+v", current.Route, previous)
		}
		previous = current.Route
	}
	var rule *RuleError
	if err := l.RegenerateRoute(runner.ID, start); !errors.As(err, &rule) {
		t.Fatal("regeneration beyond the limit accepted")
	}
	if err := l.RegenerateRoute(hunter.ID, start); !errors.As(err, &rule) {
		t.Fatal("hunter got a route")
	}

	l = newLobby()
	runner = join(t, &l, Runners, "Ada", start)
	_ = l.SetReady(runner.ID, true, start, false)
	if err := l.RegenerateRoute(runner.ID, start); !errors.As(err, &rule) {
		t.Fatal("ready runner changed route")
	}
	hunter = join(t, &l, Hunters, "Bob", start)
	_ = l.SetReady(hunter.ID, true, start, false)
	_ = l.SetReady(runner.ID, false, start, false)
	_ = l.SetReady(runner.ID, true, start, false)
	if err := l.RegenerateRoute(runner.ID, start.Add(CountdownDuration)); !errors.As(err, &rule) {
		t.Fatal("route changed after the start")
	}
}

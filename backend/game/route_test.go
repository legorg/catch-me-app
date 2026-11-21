package game

import (
	"errors"
	"testing"
)

func TestCellParsing(t *testing.T) {
	for _, label := range []string{"A1", "F6", "C4", "L12", "A10"} {
		if cell, err := ParseCell(label); err != nil || cell.String() != label {
			t.Fatalf("cell %s did not round trip", label)
		}
	}
	var validation *ValidationError
	for _, label := range []string{"", "A", "A0", "A01", "A+1", "A13", "M1", "a1", "A1 "} {
		if _, err := ParseCell(label); !errors.As(err, &validation) {
			t.Fatalf("invalid cell accepted: %q", label)
		}
	}
	center, _ := ParseCoordinates(52.52, 13.405)
	small, _ := ParseGridSize(3)
	p, _ := NewPlayground(center, DefaultCellSize(), small)
	c3, _ := ParseCell("C3")
	d1, _ := ParseCell("D1")
	a4, _ := ParseCell("A4")
	if !p.HasCell(c3) || p.HasCell(d1) || p.HasCell(a4) {
		t.Fatal("a 3 by 3 grid has exactly the cells A1 to C3")
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

func TestCustomRoute(t *testing.T) {
	l := newLobby()
	p := l.Game.Playground
	last := p.Rows() - 1
	south, north := cellCenter(p, Cell{column: 1, row: last}), cellCenter(p, Cell{column: 4})
	route, err := NewCustomRoute(p, south, north)
	if err != nil {
		t.Fatal(err)
	}
	if route != (Route{Start: Landmark{Coordinates: south}, Stop: Landmark{Coordinates: north}}) {
		t.Fatalf("unexpected route: %+v", route)
	}
	outside, _ := ParseCoordinates(p.Bounds().North+0.01, north.Longitude())
	var validation *ValidationError
	for _, points := range [][2]Coordinates{
		{north, south},                       // swapped
		{cellCenter(p, Cell{row: 1}), north}, // start not in the southern row
		{south, cellCenter(p, Cell{row: 1})}, // stop not in the northern row
		{south, outside},                     // stop outside the playground
	} {
		if _, err := NewCustomRoute(p, points[0], points[1]); !errors.As(err, &validation) {
			t.Fatalf("invalid custom route accepted: %+v", points)
		}
	}

	l.Game.Route, l.Game.CustomRoute = route, true
	runner := join(t, &l, Runners, "Ada", start)
	if l.Game.RouteRegenerationsLeft() != 0 {
		t.Fatal("a custom route has regenerations left")
	}
	var rule *RuleError
	if err := l.RegenerateRoute(runner.ID, routeVia(l, "Other"), start); !errors.As(err, &rule) || l.Game.Route != route {
		t.Fatal("custom route regenerated")
	}
}

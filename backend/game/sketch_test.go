package game

import (
	"errors"
	"testing"
	"time"
)

func sketchLine(t *testing.T, points int) SketchLine {
	t.Helper()
	coordinates := make([]Coordinates, points)
	for i := range coordinates {
		coordinates[i], _ = ParseCoordinates(52.52+float64(i)/1000, 13.405)
	}
	line, err := NewSketchLine(coordinates)
	if err != nil {
		t.Fatal(err)
	}
	return line
}

func TestNewSketchLine(t *testing.T) {
	var validation *ValidationError
	for _, n := range []int{0, 1, MaxSketchPoints + 1} {
		if _, err := NewSketchLine(make([]Coordinates, n)); !errors.As(err, &validation) {
			t.Fatalf("%d points accepted: %v", n, err)
		}
	}
	if got := sketchLine(t, MaxSketchPoints).Points(); len(got) != MaxSketchPoints {
		t.Fatalf("points lost: %d", len(got))
	}
}

func TestSketchRules(t *testing.T) {
	l, runner, bob, _, _, running := jokerLobby(t)
	line := sketchLine(t, 2)
	var rule *RuleError
	if err := l.AddSketch(bob.ID, line, running.Add(-time.Second)); !errors.As(err, &rule) {
		t.Fatalf("sketching before the game runs must be refused: %v", err)
	}
	if err := l.AddSketch(runner.ID, line, running); !errors.As(err, &rule) {
		t.Fatalf("runners cannot sketch: %v", err)
	}
	if err := l.ClearSketches(runner.ID, running); !errors.As(err, &rule) {
		t.Fatalf("runners cannot clear sketches: %v", err)
	}
	if err := l.AddSketch(PlayerID{}, line, running); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("unknown player: %v", err)
	}
	for range MaxSketchesPerTeam {
		if err := l.AddSketch(bob.ID, line, running); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.AddSketch(bob.ID, line, running); !errors.As(err, &rule) {
		t.Fatalf("sketch limit not enforced: %v", err)
	}
}

func TestSketchesAreVisibleToTheirTeamOnly(t *testing.T) {
	l, runner, bob, dan, eve, running := jokerLobby(t)
	at := running.Add(time.Minute)
	if err := l.AddSketch(bob.ID, sketchLine(t, 3), at); err != nil {
		t.Fatal(err)
	}
	if err := l.AddSketch(eve.ID, sketchLine(t, 2), at); err != nil {
		t.Fatal(err)
	}
	got := l.SketchesFor(dan, at)
	if len(got) != 1 || len(got[0].Line.Points()) != 3 || !got[0].CreatedAt.Equal(at) || got[0].Team != bob.HunterTeam {
		t.Fatalf("a teammate must see the team's sketch: %+v", got)
	}
	if got := l.SketchesFor(eve, at); len(got) != 1 || len(got[0].Line.Points()) != 2 {
		t.Fatalf("another hunter team sees only its own sketches: %+v", got)
	}
	if got := l.SketchesFor(runner, at); len(got) != 0 {
		t.Fatalf("runners never see sketches: %+v", got)
	}
	if got := l.SketchesFor(dan, running.Add(-time.Second)); len(got) != 0 {
		t.Fatalf("no sketches before the game runs: %+v", got)
	}

	// Any member clears all of the team's sketches, also those others drew.
	if err := l.ClearSketches(dan.ID, at); err != nil {
		t.Fatal(err)
	}
	if got := l.SketchesFor(bob, at); len(got) != 0 {
		t.Fatalf("team sketches not cleared: %+v", got)
	}
	if got := l.SketchesFor(eve, at); len(got) != 1 {
		t.Fatalf("clearing must not touch other hunter teams: %+v", got)
	}
}

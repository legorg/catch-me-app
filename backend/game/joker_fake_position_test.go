package game

import (
	"errors"
	"testing"
	"time"
)

func TestFakePositionJokerRules(t *testing.T) {
	l, runner, _, _, _, running := jokerLobby(t)
	cy := l.Players[1]
	point := cellCenter(l.Game.Playground, Cell{column: 0, row: 0})
	outside, _ := ParseCoordinates(0, 0)
	fake := func(at Coordinates) JokerChoice { return FakePosition{Point: at} }
	var rule *RuleError
	var validation *ValidationError
	if err := l.UseJoker(runner.ID, zoneOnly, running.Add(10*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := l.UseJoker(runner.ID, fake(point), running.Add(20*time.Second)); !errors.As(err, &rule) {
		t.Fatalf("a runner can only have one joker waiting for the next reveal: %v", err)
	}
	// At the reveal time, the zone joker has acted.
	revealed := running.Add(RevealInterval)
	if err := l.UseJoker(runner.ID, fake(outside), revealed); !errors.As(err, &validation) {
		t.Fatalf("the point must lie inside the playground: %v", err)
	}
	if err := l.UseJoker(runner.ID, fake(point), revealed); err != nil {
		t.Fatal(err)
	}
	if err := l.UseJoker(cy.ID, fake(point), revealed); !errors.As(err, &rule) {
		t.Fatalf("the runners share one use: %v", err)
	}
	if got := l.JokersFor(cy, revealed); got[1].Available || got[1].Use != (FakePositionUse{Runner: runner.ID, At: revealed, Point: point}) {
		t.Fatalf("every runner sees who used it and the shown point: %+v", got[1])
	}
}

func TestFakePositionReplacesTheNextRevealOnly(t *testing.T) {
	l, runner, bob, _, _, running := jokerLobby(t)
	first, second, third := running.Add(RevealInterval), running.Add(2*RevealInterval), running.Add(3*RevealInterval)
	actual := position(t, l, "C4", second)
	l.Reveals = map[PlayerID][]Reveal{runner.ID: {{Position: actual, At: first}, {Position: actual, At: second}, {Position: actual, At: third}}}
	point := cellCenter(l.Game.Playground, Cell{column: 0, row: 0})
	if err := l.UseJoker(runner.ID, FakePosition{Point: point}, first.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var got []HunterReveal
	for _, v := range l.VisibleTo(bob, third) {
		if v.ID == runner.ID {
			got = v.Reveals
		}
	}
	if len(got) != 3 || got[0] != (PositionReveal{At: first, Position: actual}) || got[2] != (PositionReveal{At: third, Position: actual}) {
		t.Fatalf("only the next reveal is affected: %+v", got)
	}
	want := PositionReveal{At: second, Position: Position{Coordinates: point, AccuracyMeters: actual.AccuracyMeters, RecordedAt: actual.RecordedAt}}
	if got[1] != want {
		t.Fatalf("the next reveal must look like a normal one at the chosen point: %+v", got[1])
	}
	if l.Reveals[runner.ID][1].Position != actual {
		t.Fatal("the stored reveals stay unchanged")
	}
}

package game

import (
	"errors"
	"testing"
	"time"
)

var showHunters JokerChoice = ShowHunters{}

func TestShowHuntersJokerRules(t *testing.T) {
	l, runner, bob, _, _, running := jokerLobby(t)
	cy := l.Players[1]
	at := running.Add(time.Minute)
	var rule *RuleError
	if err := l.UseJoker(runner.ID, showHunters, running.Add(-time.Second)); !errors.As(err, &rule) {
		t.Fatalf("a joker before the game runs must be refused: %v", err)
	}
	if err := l.UseJoker(bob.ID, showHunters, at); !errors.As(err, &rule) {
		t.Fatalf("hunters do not have runner jokers: %v", err)
	}
	// A pending reveal joker does not block it, and it does not block one.
	if err := l.UseJoker(runner.ID, zoneOnly, at); err != nil {
		t.Fatal(err)
	}
	if err := l.UseJoker(runner.ID, showHunters, at); err != nil {
		t.Fatalf("showHunters acts at once and ignores pending reveal jokers: %v", err)
	}
	if err := l.UseJoker(cy.ID, showHunters, at.Add(time.Minute)); !errors.As(err, &rule) {
		t.Fatalf("the runners share one use: %v", err)
	}
	if err := l.UseJoker(cy.ID, FakePosition{Point: cellCenter(l.Game.Playground, Cell{})}, at.Add(time.Minute)); err != nil {
		t.Fatalf("showHunters must not block another runner's reveal joker: %v", err)
	}
}

func TestShowHuntersShowsEveryHunterToAllRunnersOnly(t *testing.T) {
	l, runner, bob, dan, eve, running := jokerLobby(t)
	cy := l.Players[1]
	at := running.Add(time.Minute)
	if err := l.UseJoker(runner.ID, showHunters, at); err != nil {
		t.Fatal(err)
	}
	got := l.JokersFor(cy, at)
	if len(got) != 3 || got[2].Kind != JokerShowHunters || got[2].Available || len(got[2].Uses) != 1 {
		t.Fatalf("every runner sees the used joker: %+v", got)
	}
	use, ok := got[2].Uses[0].(ShowHuntersUse)
	if !ok || use.Runner != runner.ID || !use.At.Equal(at) || len(use.Positions) != 3 {
		t.Fatalf("the use must show every hunter of every hunter team: %+v", got[2].Uses[0])
	}
	for i, hunter := range []Player{bob, dan, eve} {
		if use.Positions[i].PlayerID != hunter.ID || use.Positions[i].Position != hunter.Position {
			t.Fatalf("hunter %d has the wrong position: %+v", i, use.Positions[i])
		}
	}
	for _, j := range l.JokersFor(bob, at) {
		if j.Kind == JokerShowHunters || len(j.Uses) != 0 {
			t.Fatalf("hunters are not told about runner jokers: %+v", j)
		}
	}
	for _, v := range l.VisibleTo(runner, at) {
		if v.Team == Hunters && !v.Position.IsZero() {
			t.Fatalf("runners still get no live hunter position: %+v", v)
		}
	}
}

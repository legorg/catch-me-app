package game

import (
	"errors"
	"testing"
	"time"
)

var zoneOnly JokerChoice = ZoneOnly{}

func TestZoneOnlyJokerRules(t *testing.T) {
	l, runner, bob, _, _, running := jokerLobby(t)
	cy := l.Players[1]
	at := running.Add(time.Minute)
	var rule *RuleError
	if err := l.UseJoker(runner.ID, zoneOnly, running.Add(-time.Second)); !errors.As(err, &rule) {
		t.Fatalf("a joker before the game runs must be refused: %v", err)
	}
	if err := l.UseJoker(bob.ID, zoneOnly, at); !errors.As(err, &rule) {
		t.Fatalf("hunters do not have runner jokers: %v", err)
	}
	if err := l.UseJoker(runner.ID, zoneOnly, at); err != nil {
		t.Fatal(err)
	}
	if err := l.UseJoker(cy.ID, zoneOnly, at.Add(time.Minute)); !errors.As(err, &rule) {
		t.Fatalf("the runners share one use: %v", err)
	}
	if got := l.JokersFor(cy, at); len(got) != 2 || got[0].Available || got[0].Use != (ZoneOnlyUse{Runner: runner.ID, At: at}) || !got[1].Available {
		t.Fatalf("every runner sees who used it and when: %+v", got)
	}
	for _, j := range l.JokersFor(bob, at) {
		if j.Kind == JokerZoneOnly || !j.Available {
			t.Fatalf("hunters are not told about runner jokers: %+v", j)
		}
	}
}

func TestZoneOnlyHidesThePositionOfTheNextRevealOnly(t *testing.T) {
	l, runner, bob, _, _, running := jokerLobby(t)
	cy := l.Players[1]
	first, second, third := running.Add(revealInterval), running.Add(2*revealInterval), running.Add(3*revealInterval)
	inC6 := position(t, l, "C6", first)
	// The runner moves on after using the joker: the zone is where they are at the reveal.
	inC4 := position(t, l, "C4", second)
	outside, _ := ParseCoordinates(0, 0)
	away, _ := NewPosition(outside, 5, first)
	l.Reveals = map[PlayerID][]Reveal{
		runner.ID: {{Position: inC6, At: first}, {Position: inC4, At: second}, {Position: inC4, At: third}},
		cy.ID:     {{Position: away, At: first}, {Position: away, At: second}},
	}
	// Used exactly at a reveal time: that reveal has already happened, so the next one shows the zone.
	if err := l.UseJoker(runner.ID, zoneOnly, first); err != nil {
		t.Fatal(err)
	}
	reveals := func(id PlayerID) []HunterReveal {
		for _, v := range l.VisibleTo(bob, third) {
			if v.ID == id {
				return v.Reveals
			}
		}
		t.Fatal("player missing")
		return nil
	}
	got := reveals(runner.ID)
	if len(got) != 3 || got[0] != (PositionReveal{At: first, Position: inC6}) || got[2] != (PositionReveal{At: third, Position: inC4}) {
		t.Fatalf("only the next reveal is affected: %+v", got)
	}
	if got[1] != (ZoneReveal{At: second, Cell: Cell{column: 2, row: 3}}) {
		t.Fatalf("the next reveal shows only the cell at the reveal time: %+v", got[1])
	}
	if l.Reveals[runner.ID][1].Position != inC4 {
		t.Fatal("the stored reveals stay unchanged")
	}
	if got := reveals(cy.ID); got[0] != (PositionReveal{At: first, Position: away}) || got[1] != (PositionReveal{At: second, Position: away}) {
		t.Fatalf("other runners' reveals are unaffected: %+v", got)
	}

	// Outside the playground the zone reveal has no cell.
	l.Jokers = []JokerUse{ZoneOnlyUse{Runner: cy.ID, At: first.Add(time.Second)}}
	if got := reveals(cy.ID)[1]; got != (OutsideReveal{At: second}) {
		t.Fatalf("a zone reveal outside the playground has neither cell nor position: %+v", got)
	}
}

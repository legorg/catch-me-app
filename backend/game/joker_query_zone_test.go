package game

import (
	"errors"
	"testing"
	"time"
)

func TestQueryZoneAnswersOnlyYesOrNo(t *testing.T) {
	l, runner, bob, dan, eve, running := jokerLobby(t)
	at := running.Add(time.Minute)
	query := func(states []jokerState) jokerState {
		t.Helper()
		if len(states) != 4 || states[1].Kind != JokerQueryZone {
			t.Fatalf("expected the query-zone joker: %+v", states)
		}
		return states[1]
	}
	// Both runners stand at the route start in C6.
	occupied := QueryZone{Cell: Cell{column: 2, row: 5}}
	empty := QueryZone{Cell: Cell{column: 0, row: 0}}
	var validation *ValidationError
	if err := l.UseJoker(bob.ID, QueryZone{Cell: Cell{column: 6, row: 0}}, at); !errors.As(err, &validation) {
		t.Fatalf("G1 is outside a 6 by 6 grid: %v", err)
	}
	if err := l.UseJoker(bob.ID, occupied, at); err != nil {
		t.Fatal(err)
	}
	if got := query(l.JokersFor(dan, at)); got.Available || len(got.Uses) != 1 || got.Uses[0] != (QueryZoneUse{Team: bob.HunterTeam, At: at, Cell: Cell{column: 2, row: 5}, Found: true}) {
		t.Fatalf("a teammate must see yes for C6, without any positions: %+v", got)
	}
	if got := l.JokersFor(dan, at)[0]; !got.Available {
		t.Fatalf("querying a zone does not use up show runners: %+v", got)
	}
	var rule *RuleError
	if err := l.UseJoker(dan.ID, empty, at); !errors.As(err, &rule) {
		t.Fatalf("a team can query a zone once: %v", err)
	}
	if got := query(l.JokersFor(eve, at)); !got.Available || len(got.Uses) != 0 {
		t.Fatalf("another hunter team sees nothing of it: %+v", got)
	}
	if err := l.UseJoker(eve.ID, empty, at); err != nil {
		t.Fatal(err)
	}
	if got := query(l.JokersFor(eve, at)); len(got.Uses) != 1 || got.Uses[0] != (QueryZoneUse{Team: eve.HunterTeam, At: at, Cell: Cell{}, Found: false}) {
		t.Fatalf("no runner is in A1: %+v", got)
	}
	if err := l.UseJoker(runner.ID, empty, at); !errors.As(err, &rule) {
		t.Fatalf("runners cannot query a zone: %v", err)
	}
}

func TestQueryZoneUsesTheRotatedGrid(t *testing.T) {
	for _, degrees := range []int{30, 90} {
		l, _, bob, _, eve, running := jokerLobby(t)
		northAligned := rotateGrid(t, &l, degrees)
		at := running.Add(time.Minute)
		c6 := Cell{column: 2, row: 5}
		point := cellCenter(l.Game.Playground, c6)
		// Without the rotation the point would lie in another cell.
		unrotated, inside := northAligned.CellAt(point)
		if !inside || unrotated == c6 {
			t.Fatalf("%d°: the point must lie in another north-aligned cell, got %s (inside %v)", degrees, unrotated, inside)
		}
		for i, p := range l.Players {
			if p.Team == Runners {
				l.Players[i].Position, _ = NewPosition(point, 5, at)
			}
		}
		if err := l.UseJoker(bob.ID, QueryZone{Cell: c6}, at); err != nil {
			t.Fatal(err)
		}
		if err := l.UseJoker(eve.ID, QueryZone{Cell: unrotated}, at); err != nil {
			t.Fatal(err)
		}
		if got := l.JokersFor(bob, at)[1].Uses; len(got) != 1 || !got[0].(QueryZoneUse).Found {
			t.Fatalf("%d°: the runners are in the rotated C6: %+v", degrees, got)
		}
		if got := l.JokersFor(eve, at)[1].Uses; len(got) != 1 || got[0].(QueryZoneUse).Found {
			t.Fatalf("%d°: no runner is in the rotated %s: %+v", degrees, unrotated, got)
		}
	}
}

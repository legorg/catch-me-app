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
	if got := query(l.JokersFor(dan, at)); got.Available || got.Use != (QueryZoneUse{Team: bob.HunterTeam, At: at, Cell: Cell{column: 2, row: 5}, Found: true}) {
		t.Fatalf("a teammate must see yes for C6, without any positions: %+v", got)
	}
	if got := l.JokersFor(dan, at)[0]; !got.Available {
		t.Fatalf("querying a zone does not use up show runners: %+v", got)
	}
	var rule *RuleError
	if err := l.UseJoker(dan.ID, empty, at); !errors.As(err, &rule) {
		t.Fatalf("a team can query a zone once: %v", err)
	}
	if got := query(l.JokersFor(eve, at)); !got.Available || got.Use != nil {
		t.Fatalf("another hunter team sees nothing of it: %+v", got)
	}
	if err := l.UseJoker(eve.ID, empty, at); err != nil {
		t.Fatal(err)
	}
	if got := query(l.JokersFor(eve, at)); got.Use != (QueryZoneUse{Team: eve.HunterTeam, At: at, Cell: Cell{}, Found: false}) {
		t.Fatalf("no runner is in A1: %+v", got)
	}
	if err := l.UseJoker(runner.ID, empty, at); !errors.As(err, &rule) {
		t.Fatalf("runners cannot query a zone: %v", err)
	}
}

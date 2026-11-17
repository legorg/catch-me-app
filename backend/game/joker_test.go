package game

import (
	"errors"
	"testing"
	"time"
)

// jokerLobby has runners Ada and Cy, and hunters Bob and Dan (team Pack) and Eve (team Wolves); it is running from the returned time.
func jokerLobby(t *testing.T) (l Lobby, runner, bob, dan, eve Player, running time.Time) {
	t.Helper()
	l = newLobby()
	runner = join(t, &l, Runners, "Ada", start)
	join(t, &l, Runners, "Cy", start)
	bob = join(t, &l, Hunters, "Bob", start)
	dan = join(t, &l, Hunters, "Dan", start)
	eve = join(t, &l, Hunters, "Eve", start)
	setHunterTeam(t, &l, eve.ID, "wolves", start)
	eve, _ = l.Player(eve.ID)
	for _, p := range l.Players {
		if err := l.SetReady(p.ID, true, start, false); err != nil {
			t.Fatal(err)
		}
	}
	return l, runner, bob, dan, eve, l.Game.StartsAt
}

var showRunners = JokerChoice{Kind: JokerShowRunners}

func TestParseJokerKind(t *testing.T) {
	if kind, err := ParseJokerKind("showRunners"); err != nil || kind != JokerShowRunners {
		t.Fatalf("known joker rejected: %v", err)
	}
	var validation *ValidationError
	if _, err := ParseJokerKind("teleport"); !errors.As(err, &validation) {
		t.Fatalf("unknown joker accepted: %v", err)
	}
}

func TestShowRunnersJokerRules(t *testing.T) {
	l, runner, bob, dan, eve, running := jokerLobby(t)
	var rule *RuleError
	if err := l.UseJoker(bob.ID, showRunners, running.Add(-time.Second)); !errors.As(err, &rule) {
		t.Fatalf("a joker before the game runs must be refused: %v", err)
	}
	if err := l.UseJoker(runner.ID, showRunners, running); !errors.As(err, &rule) {
		t.Fatalf("runners have no jokers: %v", err)
	}
	if err := l.UseJoker(PlayerID{}, showRunners, running); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("unknown player: %v", err)
	}

	if err := l.UseJoker(bob.ID, showRunners, running.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := l.UseJoker(bob.ID, showRunners, running.Add(2*time.Minute)); !errors.As(err, &rule) {
		t.Fatalf("a joker can only be used once: %v", err)
	}
	if err := l.UseJoker(dan.ID, showRunners, running.Add(2*time.Minute)); !errors.As(err, &rule) {
		t.Fatalf("a team shares one use: %v", err)
	}
	if err := l.UseJoker(eve.ID, showRunners, running.Add(2*time.Minute)); err != nil {
		t.Fatalf("another hunter team has its own joker: %v", err)
	}
}

func TestJokersAreVisibleToTheirTeamOnly(t *testing.T) {
	l, runner, bob, dan, eve, running := jokerLobby(t)
	at := running.Add(time.Minute)
	only := func(states []jokerState) jokerState {
		t.Helper()
		if len(states) != 2 || states[0].Kind != JokerShowRunners || states[1].Kind != JokerQueryZone {
			t.Fatalf("expected both hunter jokers: %+v", states)
		}
		return states[0]
	}
	if got := only(l.JokersFor(bob, running.Add(-time.Second))); got.Available {
		t.Fatal("not available before the game runs")
	}
	if got := only(l.JokersFor(bob, at)); !got.Available || !got.UsedAt.IsZero() || len(got.Positions) != 0 {
		t.Fatalf("unused joker: %+v", got)
	}
	if err := l.UseJoker(bob.ID, showRunners, at); err != nil {
		t.Fatal(err)
	}
	used := only(l.JokersFor(dan, at))
	if used.Available || !used.UsedAt.Equal(at) || len(used.Positions) != 2 {
		t.Fatalf("a teammate must see the result of the use: %+v", used)
	}
	if got := only(l.JokersFor(eve, at)); !got.Available || len(got.Positions) != 0 {
		t.Fatalf("another hunter team sees nothing of it: %+v", got)
	}
	if got := l.JokersFor(runner, at); len(got) != 1 || got[0].Kind != JokerZoneOnly || !got[0].Available {
		t.Fatalf("runners see only their own jokers: %+v", got)
	}
}

func TestParseJokerChoice(t *testing.T) {
	if got, err := ParseJokerChoice("queryZone", "C6"); err != nil || got.Kind != JokerQueryZone || got.Cell.String() != "C6" {
		t.Fatalf("query with a cell rejected: %+v, %v", got, err)
	}
	if got, err := ParseJokerChoice("showRunners", ""); err != nil || got != showRunners {
		t.Fatalf("show runners rejected: %+v, %v", got, err)
	}
	var validation *ValidationError
	for _, c := range []struct{ kind, cell string }{{"queryZone", ""}, {"queryZone", "G1"}, {"showRunners", "A1"}, {"teleport", ""}} {
		if _, err := ParseJokerChoice(c.kind, c.cell); !errors.As(err, &validation) {
			t.Fatalf("%s with cell %q must be invalid: %v", c.kind, c.cell, err)
		}
	}
}

func TestQueryZoneAnswersOnlyYesOrNo(t *testing.T) {
	l, runner, bob, dan, eve, running := jokerLobby(t)
	at := running.Add(time.Minute)
	query := func(states []jokerState) jokerState {
		t.Helper()
		if len(states) != 2 || states[1].Kind != JokerQueryZone {
			t.Fatalf("expected the query-zone joker: %+v", states)
		}
		return states[1]
	}
	// Both runners stand at the route start in C6.
	occupied, _ := ParseJokerChoice("queryZone", "C6")
	empty, _ := ParseJokerChoice("queryZone", "A1")
	if err := l.UseJoker(bob.ID, occupied, at); err != nil {
		t.Fatal(err)
	}
	if got := query(l.JokersFor(dan, at)); got.Available || !got.UsedAt.Equal(at) || got.Cell.String() != "C6" || !got.Found || len(got.Positions) != 0 {
		t.Fatalf("a teammate must see yes for C6, without any positions: %+v", got)
	}
	if got := l.JokersFor(dan, at)[0]; !got.Available {
		t.Fatalf("querying a zone does not use up show runners: %+v", got)
	}
	var rule *RuleError
	if err := l.UseJoker(dan.ID, empty, at); !errors.As(err, &rule) {
		t.Fatalf("a team can query a zone once: %v", err)
	}
	if got := query(l.JokersFor(eve, at)); !got.Available || got.Found {
		t.Fatalf("another hunter team sees nothing of it: %+v", got)
	}
	if err := l.UseJoker(eve.ID, empty, at); err != nil {
		t.Fatal(err)
	}
	if got := query(l.JokersFor(eve, at)); got.Cell.String() != "A1" || got.Found {
		t.Fatalf("no runner is in A1: %+v", got)
	}
	if err := l.UseJoker(runner.ID, empty, at); !errors.As(err, &rule) {
		t.Fatalf("runners cannot query a zone: %v", err)
	}
}

var zoneOnly = JokerChoice{Kind: JokerZoneOnly}

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
	if got := l.JokersFor(cy, at); len(got) != 1 || got[0].Available || !got[0].UsedAt.Equal(at) || got[0].UsedBy != runner.ID {
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
	first, second, third := running.Add(RevealInterval), running.Add(2*RevealInterval), running.Add(3*RevealInterval)
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
	reveals := func(id PlayerID) []Reveal {
		for _, v := range l.VisibleTo(bob, third) {
			if v.ID == id {
				return v.Reveals
			}
		}
		t.Fatal("player missing")
		return nil
	}
	got := reveals(runner.ID)
	if len(got) != 3 || got[0] != l.Reveals[runner.ID][0] || got[2] != l.Reveals[runner.ID][2] {
		t.Fatalf("only the next reveal is affected: %+v", got)
	}
	if !got[1].ZoneOnly || !got[1].Position.IsZero() || !got[1].InPlayground || got[1].Cell.String() != "C4" || !got[1].At.Equal(second) {
		t.Fatalf("the next reveal shows only the cell at the reveal time: %+v", got[1])
	}
	if l.Reveals[runner.ID][1].ZoneOnly {
		t.Fatal("the stored reveals stay unchanged")
	}
	if got := reveals(cy.ID); got[0].ZoneOnly || got[1].ZoneOnly {
		t.Fatalf("other runners' reveals are unaffected: %+v", got)
	}

	// Outside the playground the zone reveal has no cell.
	l.Jokers = []JokerUse{{Kind: JokerZoneOnly, PlayerID: cy.ID, UsedAt: first.Add(time.Second)}}
	if got := reveals(cy.ID)[1]; !got.ZoneOnly || got.InPlayground || !got.Position.IsZero() {
		t.Fatalf("a zone reveal outside the playground has neither cell nor position: %+v", got)
	}
}

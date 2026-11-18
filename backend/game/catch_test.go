package game

import (
	"errors"
	"testing"
	"time"
)

func TestMarkCaughtRules(t *testing.T) {
	l, runner, hunter, _, _, running := jokerLobby(t)
	var rule *RuleError
	if err := l.MarkCaught(NewPlayerID(), running); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("unknown player: %v", err)
	}
	if err := l.MarkCaught(hunter.ID, running); !errors.As(err, &rule) {
		t.Fatalf("hunter must not mark themselves caught: %v", err)
	}
	if err := l.MarkCaught(runner.ID, running.Add(-time.Second)); !errors.As(err, &rule) {
		t.Fatalf("catch during countdown must be refused: %v", err)
	}
	lobby := newLobby()
	beforeStart := join(t, &lobby, Runners, "Before start", start)
	if err := lobby.MarkCaught(beforeStart.ID, start); !errors.As(err, &rule) {
		t.Fatalf("catch in lobby must be refused: %v", err)
	}
	// Remove hunter reports; the runner's report is also old. Neither matters.
	for i := range l.Players {
		if l.Players[i].Team == Hunters {
			l.Players[i].Position = Position{}
		}
	}
	at := running.Add(time.Hour)
	if err := l.MarkCaught(runner.ID, at); err != nil {
		t.Fatal(err)
	}
	if err := l.MarkCaught(runner.ID, at.Add(time.Minute)); err != nil {
		t.Fatalf("retry should be harmless: %v", err)
	}
	caught, _ := l.Player(runner.ID)
	if caught.Runner.Status() != RunnerCaught || caught.Runner.EndedAt() != at || caught.Position != runner.Position {
		t.Fatalf("catch time and last report must remain frozen: %+v", caught)
	}
	if err := l.MarkCaught(l.Players[1].ID, at); err != nil {
		t.Fatal(err)
	}
	if l.Game.Status(at) != StatusRunning {
		t.Fatal("even catching every runner must not end the game yet")
	}
}

func TestCaughtRunnerReplacesLivePositionAndRevealsForEveryone(t *testing.T) {
	l, runner, hunter, _, _, running := jokerLobby(t)
	at := running.Add(time.Second)
	actual := position(t, l, "C4", at)
	if err := l.UpdatePosition(runner.ID, actual, at, false); err != nil {
		t.Fatal(err)
	}
	// Even a pending fake position must not replace the real catch location.
	if err := l.UseJoker(runner.ID, FakePosition{Point: cellCenter(l.Game.Playground, Cell{})}, at); err != nil {
		t.Fatal(err)
	}
	l.Reveals = map[PlayerID][]Reveal{runner.ID: {{At: at, Position: actual}}}
	if err := l.MarkCaught(runner.ID, at); err != nil {
		t.Fatal(err)
	}
	for _, viewer := range []Player{runner, l.Players[1], hunter} {
		for _, seenAt := range []time.Time{at, at.Add(2 * revealInterval)} {
			for _, p := range l.VisibleTo(viewer, seenAt) {
				if p.ID != runner.ID {
					continue
				}
				if p.Runner.Status() != RunnerCaught || p.Runner.EndedAt() != at || p.CaughtPosition != actual || !p.Position.IsZero() || len(p.Reveals) != 0 {
					t.Fatalf("everyone must get only the frozen catch, immediately and later: %+v", p)
				}
			}
		}
	}
}

func TestCaughtRunnerCannotPlayAndIsExcludedFromNewHunterJokers(t *testing.T) {
	l, runner, hunter, _, _, running := jokerLobby(t)
	at := running.Add(time.Second)
	for id, cell := range map[PlayerID]string{runner.ID: "A1", l.Players[1].ID: "F6", hunter.ID: "C3"} {
		if err := l.UpdatePosition(id, position(t, l, cell, at), at, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.MarkCaught(runner.ID, at); err != nil {
		t.Fatal(err)
	}
	var rule *RuleError
	if err := l.UpdatePosition(runner.ID, position(t, l, "B2", at), at, true); !errors.As(err, &rule) {
		t.Fatalf("even the development override must not allow further reports: %v", err)
	}
	for _, choice := range []JokerChoice{ZoneOnly{}, FakePosition{Point: l.Game.Playground.Center()}, ShowHunters{}} {
		if err := l.UseJoker(runner.ID, choice, at); !errors.As(err, &rule) {
			t.Fatalf("caught runner must not use %s: %v", choice.Kind(), err)
		}
	}
	caught, _ := l.Player(runner.ID)
	for _, joker := range l.JokersFor(caught, at) {
		if joker.Available {
			t.Fatalf("caught runner must not have available jokers: %+v", joker)
		}
	}
	cell, _ := ParseCell("A1")
	for _, choice := range []JokerChoice{ShowRunners{}, QueryZone{Cell: cell}, Compass{}} {
		if err := l.UseJoker(hunter.ID, choice, at); err != nil {
			t.Fatal(err)
		}
	}
	for _, use := range l.Jokers {
		switch u := use.(type) {
		case ShowRunnersUse:
			if len(u.Positions) != 1 || u.Positions[0].PlayerID != l.Players[1].ID {
				t.Fatalf("show runners must exclude the caught runner: %+v", u)
			}
		case QueryZoneUse:
			if u.Found {
				t.Fatal("caught runner must not count in a zone query")
			}
		case CompassUse:
			if len(u.Bearings) != 1 {
				t.Fatalf("compass must point only at active runners: %+v", u)
			}
		}
	}
}

func TestMarkCaughtWithoutAReportedPosition(t *testing.T) {
	l, runner, hunter, _, _, running := jokerLobby(t)
	l.Players[l.index(runner.ID)].Position = Position{}
	if err := l.MarkCaught(runner.ID, running); err != nil {
		t.Fatalf("catch must not require a report: %v", err)
	}
	for _, p := range l.VisibleTo(hunter, running) {
		if p.ID == runner.ID && (p.Runner.Status() != RunnerCaught || p.Runner.EndedAt() != running || !p.CaughtPosition.IsZero()) {
			t.Fatalf("catch without a position should only expose the status: %+v", p)
		}
	}
}

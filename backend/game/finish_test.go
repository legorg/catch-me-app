package game

import (
	"errors"
	"testing"
	"time"
)

func TestFinishPositionGateAndOverride(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		age                   time.Duration
		where                 string
		latitudeOffset        float64
		missing, skip, wantOK bool
	}{
		{name: "fresh stop", where: "stop", wantOK: true},
		{name: "near stop", latitudeOffset: 0.0003, wantOK: true},
		{name: "outside radius", latitudeOffset: 0.0006},
		{name: "30 seconds inclusive", where: "stop", age: 30 * time.Second, wantOK: true},
		{name: "stale", where: "stop", age: 30*time.Second + time.Nanosecond},
		{name: "start is not stop", where: "start"},
		{name: "missing", missing: true},
		{name: "override missing", missing: true, skip: true, wantOK: true},
		{name: "override stale", where: "stop", age: time.Hour, skip: true, wantOK: true},
		{name: "override distant", where: "start", skip: true, wantOK: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l, runner, _, _, _, running := jokerLobby(t)
			at := running.Add(time.Hour)
			var report Position
			if !tc.missing {
				point := l.Game.Route.Stop.Coordinates
				if tc.where == "start" {
					point = l.Game.Route.Start.Coordinates
				}
				point, _ = ParseCoordinates(point.Latitude()+tc.latitudeOffset, point.Longitude())
				report, _ = NewPosition(point, 5, at.Add(-tc.age))
			}
			l.Players[l.index(runner.ID)].Position = report
			err := l.Finish(runner.ID, at, tc.skip)
			var rule *RuleError
			if tc.wantOK && err != nil || !tc.wantOK && !errors.As(err, &rule) {
				t.Fatalf("finish: %v, wantOK=%v", err, tc.wantOK)
			}
			p, _ := l.Player(runner.ID)
			if (p.Runner.Status() == RunnerFinished) != tc.wantOK || p.Position != report {
				t.Fatalf("unexpected finish or changed report: %+v", p)
			}
			if tc.wantOK {
				// Retry succeeds without a fresh position or an override.
				if err := l.Finish(runner.ID, at.Add(time.Hour), false); err != nil {
					t.Fatal(err)
				}
				p, _ = l.Player(runner.ID)
				if p.Runner.EndedAt() != at {
					t.Fatal("retry changed finish time")
				}
			}
		})
	}
}

func TestFinishRulesAndRetirement(t *testing.T) {
	l, runner, hunter, _, _, running := jokerLobby(t)
	var rule *RuleError
	if err := l.Finish(NewPlayerID(), running, true); !errors.Is(err, ErrUnauthorized) {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id PlayerID
		at time.Time
	}{
		{hunter.ID, running}, {runner.ID, running.Add(-time.Second)},
	} {
		if err := l.Finish(tc.id, tc.at, true); !errors.As(err, &rule) {
			t.Fatalf("override bypassed role/phase: %v", err)
		}
	}
	lobby := newLobby()
	beforeStart := join(t, &lobby, Runners, "Before start", start)
	if err := lobby.Finish(beforeStart.ID, start, true); !errors.As(err, &rule) {
		t.Fatal("finished in lobby")
	}
	if err := l.MarkCaught(l.Players[1].ID, running); err != nil {
		t.Fatal(err)
	}
	if err := l.Finish(l.Players[1].ID, running, true); !errors.As(err, &rule) {
		t.Fatal("caught runner finished")
	}
	if err := l.Finish(runner.ID, running, true); err != nil {
		t.Fatal(err)
	}
	if err := l.MarkCaught(runner.ID, running); !errors.As(err, &rule) {
		t.Fatal("finished runner was caught")
	}
	if err := l.UpdatePosition(runner.ID, runner.Position, running, true); !errors.As(err, &rule) {
		t.Fatal("finished runner reported")
	}
	for _, choice := range []JokerChoice{ZoneOnly{}, FakePosition{Point: l.Game.Playground.Center()}, ShowHunters{}} {
		if err := l.UseJoker(runner.ID, choice, running); !errors.As(err, &rule) {
			t.Fatalf("finished runner used %s", choice.Kind())
		}
	}
	p, _ := l.Player(runner.ID)
	for _, joker := range l.JokersFor(p, running) {
		if joker.Available {
			t.Fatal("available joker after finish")
		}
	}
	if l.Game.Status(running) != StatusRunning {
		t.Fatal("finish must not end the game yet")
	}
}

func TestFinishHidesCoordinatesAndExcludesNewHunterJokers(t *testing.T) {
	l, runner, hunter, _, _, running := jokerLobby(t)
	report, _ := NewPosition(l.Game.Route.Stop.Coordinates, 5, running)
	l.Players[l.index(runner.ID)].Position = report
	l.Reveals = map[PlayerID][]Reveal{runner.ID: {{At: running, Position: report}}}
	if err := l.Finish(runner.ID, running, false); err != nil {
		t.Fatal(err)
	}
	for _, viewer := range []Player{runner, l.Players[1], hunter} {
		for _, at := range []time.Time{running, running.Add(2 * revealInterval)} {
			for _, p := range l.VisibleTo(viewer, at) {
				if p.ID == runner.ID && (p.Runner.Status() != RunnerFinished || p.Runner.EndedAt() != running || !p.Position.IsZero() || !p.CaughtPosition.IsZero() || len(p.Reveals) != 0) {
					t.Fatalf("finish exposed coordinates: %+v", p)
				}
			}
		}
	}
	cell, _ := l.Game.Playground.CellAt(report.Coordinates)
	for _, choice := range []JokerChoice{ShowRunners{}, QueryZone{Cell: cell}, Compass{}} {
		if err := l.UseJoker(hunter.ID, choice, running); err != nil {
			t.Fatal(err)
		}
	}
	for _, use := range l.Jokers {
		switch u := use.(type) {
		case ShowRunnersUse:
			if len(u.Positions) != 1 || u.Positions[0].PlayerID == runner.ID {
				t.Fatalf("finished runner in snapshot: %+v", u)
			}
		case QueryZoneUse:
			if u.Found {
				t.Fatal("finished runner in query")
			}
		case CompassUse:
			if len(u.Bearings) != 1 {
				t.Fatal("finished runner in compass")
			}
		}
	}
}

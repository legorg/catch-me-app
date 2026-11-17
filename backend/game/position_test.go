package game

import (
	"errors"
	"testing"
	"time"
)

func position(t *testing.T, l Lobby, cell string, at time.Time) Position {
	t.Helper()
	c, err := ParseCell(cell)
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewPosition(cellCenter(l.Game.Playground, c), 5, at)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCellAtMatchesCellCenter(t *testing.T) {
	l := newLobby()
	for row := range gridSize {
		for column := range gridSize {
			cell := Cell{column: column, row: row}
			if got, ok := l.Game.Playground.CellAt(cellCenter(l.Game.Playground, cell)); !ok || got != cell {
				t.Fatalf("cell %s maps to %s", cell, got)
			}
		}
	}
	b := l.Game.Playground.Bounds()
	corner, _ := ParseCoordinates(b.South, b.East)
	if got, ok := l.Game.Playground.CellAt(corner); !ok || got.String() != "F6" {
		t.Fatalf("south-east corner maps to %s", got)
	}
	outside, _ := ParseCoordinates(b.North+0.001, b.West)
	if _, ok := l.Game.Playground.CellAt(outside); ok {
		t.Fatal("outside position mapped to a cell")
	}
	if _, err := NewPosition(outside, -1, start); err == nil {
		t.Fatal("negative accuracy accepted")
	}
}

func TestRunnerMustBeNearStartToGetReady(t *testing.T) {
	l := newLobby()
	name, _ := ParsePlayerName("Ada")
	runner, _ := l.Join(l.Game.JoinCodes.Runner, name, NewPlayerToken(), start)
	var rule *RuleError
	if err := l.SetReady(runner.ID, true, start, false); !errors.As(err, &rule) {
		t.Fatal("ready without position accepted")
	}
	startAt := l.Game.Route.Start.Coordinates
	nearby := func(latOffset float64) Position {
		c, _ := ParseCoordinates(startAt.Latitude()+latOffset, startAt.Longitude())
		p, err := NewPosition(c, 5, start)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	// 0.0006 degrees of latitude are about 67 m, 0.0003 about 33 m.
	if err := l.UpdatePosition(runner.ID, nearby(0.0006), start, false); err != nil {
		t.Fatal(err)
	}
	if err := l.SetReady(runner.ID, true, start, false); !errors.As(err, &rule) {
		t.Fatal("ready more than 50 m from the start accepted")
	}
	_ = l.UpdatePosition(runner.ID, nearby(0.0003), start, false)
	if err := l.SetReady(runner.ID, true, start.Add(startPositionMaxAge+time.Second), false); !errors.As(err, &rule) {
		t.Fatal("stale position accepted")
	}
	if err := l.SetReady(runner.ID, true, start, false); err != nil {
		t.Fatal(err)
	}
	if err := l.SetReady(runner.ID, false, start.Add(time.Hour), false); err != nil {
		t.Fatal("un-ready must never need a position")
	}
}

func TestLeavingStartCancelsReady(t *testing.T) {
	l := newLobby()
	runner := join(t, &l, Runners, "Ada", start)
	hunter := join(t, &l, Hunters, "Bob", start)
	_ = l.SetReady(runner.ID, true, start, false)
	_ = l.SetReady(hunter.ID, true, start, false)
	if l.Game.Status(start) != StatusCountdown {
		t.Fatal("countdown not started")
	}
	if err := l.UpdatePosition(runner.ID, position(t, l, "C3", start.Add(time.Second)), start.Add(time.Second), false); err != nil {
		t.Fatal(err)
	}
	if p, _ := l.Player(runner.ID); p.Ready || l.Game.Status(start.Add(time.Second)) != StatusLobby {
		t.Fatal("moving away from the start must cancel ready and the countdown")
	}
	moveToStart(t, &l, runner.ID, start.Add(2*time.Second))
	_ = l.SetReady(runner.ID, true, start.Add(2*time.Second), false)
	running := l.Game.StartsAt
	if err := l.UpdatePosition(runner.ID, position(t, l, "C3", running), running, false); err != nil {
		t.Fatal(err)
	}
	if p, _ := l.Player(runner.ID); !p.Ready || l.Game.Status(running) != StatusRunning {
		t.Fatal("moving while running must not change ready state")
	}
}

func runningLobby(t *testing.T) (Lobby, Player, Player, Player) {
	t.Helper()
	l := newLobby()
	runner := join(t, &l, Runners, "Ada", start)
	mate := join(t, &l, Runners, "Cy", start)
	hunter := join(t, &l, Hunters, "Bob", start)
	for _, p := range []Player{runner, mate, hunter} {
		if err := l.SetReady(p.ID, true, start, false); err != nil {
			t.Fatal(err)
		}
	}
	return l, runner, mate, hunter
}

func TestRevealSchedule(t *testing.T) {
	l, _, _, _ := runningLobby(t)
	began := l.Game.StartsAt
	if !l.Game.latestReveal(began.Add(RevealInterval-time.Second)).IsZero() || l.Game.NextReveal(began) != began.Add(RevealInterval) {
		t.Fatal("first reveal must be one interval after the start")
	}
	if l.Game.latestReveal(began.Add(2*RevealInterval+time.Second)) != began.Add(2*RevealInterval) {
		t.Fatal("latest reveal wrong")
	}
	if !l.Game.NextReveal(began.Add(-time.Second)).IsZero() {
		t.Fatal("no reveals before the start")
	}
}

func TestRevealTimes(t *testing.T) {
	l, _, _, _ := runningLobby(t)
	began := l.Game.StartsAt
	if len(l.Game.RevealTimes(began.Add(RevealInterval-time.Second))) != 0 || len(l.Game.RevealTimes(began.Add(-time.Second))) != 0 {
		t.Fatal("no reveals before the first one")
	}
	times := l.Game.RevealTimes(began.Add(3*RevealInterval + time.Second))
	if len(times) != 3 || times[0] != began.Add(RevealInterval) || times[2] != l.Game.latestReveal(began.Add(3*RevealInterval+time.Second)) {
		t.Fatalf("all reveal times, oldest first, expected: %v", times)
	}
}

func TestVisibility(t *testing.T) {
	l, runner, mate, hunter := runningLobby(t)
	lobbyTime := start.Add(time.Second)
	find := func(visible []visiblePlayer, id PlayerID) visiblePlayer {
		for _, v := range visible {
			if v.ID == id {
				return v
			}
		}
		t.Fatal("player missing")
		return visiblePlayer{}
	}
	me, _ := l.Player(runner.ID)
	asRunner := l.VisibleTo(me, lobbyTime)
	if find(asRunner, runner.ID).Position.IsZero() || !find(asRunner, mate.ID).Position.IsZero() {
		t.Fatal("before the start only the own position is visible")
	}

	running := l.Game.StartsAt.Add(time.Minute)
	_ = l.UpdatePosition(hunter.ID, position(t, l, "C3", running), running, false)
	asRunner = l.VisibleTo(me, running)
	if find(asRunner, mate.ID).Position.IsZero() || !find(asRunner, hunter.ID).Position.IsZero() {
		t.Fatal("runners see teammates but never hunters")
	}
	h, _ := l.Player(hunter.ID)
	asHunter := l.VisibleTo(h, running)
	if !find(asHunter, runner.ID).Position.IsZero() || find(asHunter, hunter.ID).Position.IsZero() {
		t.Fatal("hunters see no runner positions before a reveal")
	}
	revealed := l.Game.StartsAt.Add(RevealInterval)
	r := l.Players[l.index(runner.ID)]
	earlier := Reveal{Position: r.Position, At: revealed.Add(-RevealInterval)}
	latest := Reveal{Position: r.Position, At: revealed}
	l.Reveals = map[PlayerID][]Reveal{runner.ID: {earlier, latest}}
	v := find(l.VisibleTo(h, revealed), runner.ID)
	if len(v.Reveals) != 2 || v.Reveals[0] != (PositionReveal{At: earlier.At, Position: earlier.Position}) ||
		v.Reveals[1] != (PositionReveal{At: latest.At, Position: latest.Position}) || !v.Position.IsZero() {
		t.Fatalf("hunter must see every reveal, not the live position: %+v", v)
	}
	// Reveals reach hunters only, and only for runners.
	if got := find(l.VisibleTo(me, revealed), runner.ID).Reveals; len(got) != 0 {
		t.Fatalf("runners must not see reveals: %+v", got)
	}
	if got := find(l.VisibleTo(h, revealed), hunter.ID).Reveals; len(got) != 0 {
		t.Fatalf("hunters have no reveals: %+v", got)
	}
	if got := find(l.VisibleTo(h, lobbyTime), runner.ID).Reveals; len(got) != 0 {
		t.Fatalf("no reveals before the game runs: %+v", got)
	}
}

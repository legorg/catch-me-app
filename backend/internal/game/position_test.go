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
	p, err := NewPosition(l.Game.Playground.CellCenter(c), 5, at)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCellAtMatchesCellCenter(t *testing.T) {
	l := newLobby()
	for row := range GridSize {
		for column := range GridSize {
			cell := Cell{column: column, row: row}
			if got, ok := l.Game.Playground.CellAt(l.Game.Playground.CellCenter(cell)); !ok || got != cell {
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

func TestRunnerMustBeInStartCellToGetReady(t *testing.T) {
	l := newLobby()
	name, _ := ParsePlayerName("Ada")
	runner, _ := l.Join(l.Game.JoinCodes.Runner, name, NewPlayerToken(), start)
	var rule *RuleError
	if err := l.SetReady(runner.ID, true, start, false); !errors.As(err, &rule) {
		t.Fatal("ready without position accepted")
	}
	wrong := "A3"
	if err := l.UpdatePosition(runner.ID, position(t, l, wrong, start), start, false); err != nil {
		t.Fatal(err)
	}
	if err := l.SetReady(runner.ID, true, start, false); !errors.As(err, &rule) {
		t.Fatal("ready outside start cell accepted")
	}
	_ = l.UpdatePosition(runner.ID, position(t, l, runner.Route.Start.String(), start), start, false)
	if err := l.SetReady(runner.ID, true, start.Add(StartCellMaxAge+time.Second), false); !errors.As(err, &rule) {
		t.Fatal("stale position accepted")
	}
	if err := l.SetReady(runner.ID, true, start, false); err != nil {
		t.Fatal(err)
	}
	if err := l.SetReady(runner.ID, false, start.Add(time.Hour), false); err != nil {
		t.Fatal("un-ready must never need a position")
	}
}

func TestLeavingStartCellCancelsReady(t *testing.T) {
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
		t.Fatal("leaving the start cell must cancel ready and the countdown")
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
	if !l.Game.LatestReveal(began.Add(RevealInterval-time.Second)).IsZero() || l.Game.NextReveal(began) != began.Add(RevealInterval) {
		t.Fatal("first reveal must be one interval after the start")
	}
	if l.Game.LatestReveal(began.Add(2*RevealInterval+time.Second)) != began.Add(2*RevealInterval) {
		t.Fatal("latest reveal wrong")
	}
	if !l.Game.NextReveal(began.Add(-time.Second)).IsZero() {
		t.Fatal("no reveals before the start")
	}
}

func TestVisibility(t *testing.T) {
	l, runner, mate, hunter := runningLobby(t)
	lobbyTime := start.Add(time.Second)
	find := func(visible []VisiblePlayer, id PlayerID) VisiblePlayer {
		for _, v := range visible {
			if v.ID == id {
				return v
			}
		}
		t.Fatal("player missing")
		return VisiblePlayer{}
	}
	me, _ := l.Player(runner.ID)
	asRunner := l.VisibleTo(me, lobbyTime)
	if find(asRunner, runner.ID).Position.IsZero() || !find(asRunner, mate.ID).Position.IsZero() {
		t.Fatal("before the start only the own position is visible")
	}
	if find(asRunner, mate.ID).Route.IsZero() || !find(asRunner, hunter.ID).Route.IsZero() {
		t.Fatal("runners see teammates' routes")
	}

	running := l.Game.StartsAt.Add(time.Minute)
	_ = l.UpdatePosition(hunter.ID, position(t, l, "C3", running), running, false)
	asRunner = l.VisibleTo(me, running)
	if find(asRunner, mate.ID).Position.IsZero() || !find(asRunner, hunter.ID).Position.IsZero() {
		t.Fatal("runners see teammates but never hunters")
	}
	h, _ := l.Player(hunter.ID)
	asHunter := l.VisibleTo(h, running)
	if !find(asHunter, runner.ID).Position.IsZero() || !find(asHunter, runner.ID).Route.IsZero() || find(asHunter, hunter.ID).Position.IsZero() {
		t.Fatal("hunters see no runner positions or routes before a reveal")
	}
	revealed := l.Game.StartsAt.Add(RevealInterval)
	r := &l.Players[l.index(runner.ID)]
	r.Reveal = Reveal{Position: r.Position, At: revealed}
	v := find(l.VisibleTo(h, revealed), runner.ID)
	if v.Position.IsZero() || v.RevealedAt != revealed || !v.Route.IsZero() {
		t.Fatalf("hunter must see the revealed position: %+v", v)
	}
}

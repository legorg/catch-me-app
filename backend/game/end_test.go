package game

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestLastRunnerEndsGame(t *testing.T) {
	for _, outcomes := range [][2]RunnerStatus{
		{RunnerCaught, RunnerCaught}, {RunnerFinished, RunnerFinished},
		{RunnerCaught, RunnerFinished}, {RunnerFinished, RunnerCaught},
	} {
		t.Run(string(outcomes[0])+"/"+string(outcomes[1]), func(t *testing.T) {
			l, runner, _, _, _, running := jokerLobby(t)
			other := l.Players[1]
			complete := func(id PlayerID, outcome RunnerStatus, at time.Time) error {
				if outcome == RunnerCaught {
					return l.MarkCaught(id, at)
				}
				return l.Finish(id, at, true)
			}
			first, last := running.Add(time.Second), running.Add(time.Minute)
			if err := complete(runner.ID, outcomes[0], first); err != nil {
				t.Fatal(err)
			}
			if l.Game.Status(first) != StatusRunning || !l.Game.EndedAt.IsZero() {
				t.Fatal("an active runner remains; the game must continue")
			}
			if err := complete(other.ID, outcomes[1], last); err != nil {
				t.Fatal(err)
			}
			if l.Game.Status(last) != StatusEnded || !l.Game.EndedAt.Equal(last) || !l.Game.NextReveal(last).IsZero() {
				t.Fatalf("last outcome must end the game and stop reveals: %+v", l.Game)
			}
			for i, p := range []Player{runner, other} {
				before, _ := l.Player(p.ID)
				if err := complete(p.ID, outcomes[i], last.Add(time.Hour)); err != nil {
					t.Fatalf("same-outcome retry after the end: %v", err)
				}
				after, _ := l.Player(p.ID)
				if before != after || !l.Game.EndedAt.Equal(last) {
					t.Fatal("retry changed a runner or the game end time")
				}
			}
		})
	}
}

func TestEndedGameRejectsFurtherChanges(t *testing.T) {
	l, runner, hunter, _, _, running := jokerLobby(t)
	if err := l.MarkCaught(runner.ID, running); err != nil {
		t.Fatal(err)
	}
	if err := l.Finish(l.Players[1].ID, running, true); err != nil {
		t.Fatal(err)
	}
	at := running.Add(time.Hour)
	name, _ := ParsePlayerName("Late player")
	line, _ := NewSketchLine([]Coordinates{l.Game.Route.Start.Coordinates, l.Game.Route.Stop.Coordinates})
	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"hunter position", func() error { return l.UpdatePosition(hunter.ID, hunter.Position, at, true) }},
		{"runner position", func() error { return l.UpdatePosition(runner.ID, runner.Position, at, true) }},
		{"joker", func() error { return l.UseJoker(hunter.ID, ShowRunners{}, at) }},
		{"sketch", func() error { return l.AddSketch(hunter.ID, line, at) }},
		{"clear sketches", func() error { return l.ClearSketches(hunter.ID, at) }},
		{"ready", func() error { return l.SetReady(hunter.ID, false, at, true) }},
		{"hunter team", func() error { return l.SetHunterTeam(hunter.ID, hunter.HunterTeam, at) }},
		{"route", func() error { return l.RegenerateRoute(runner.ID, l.Game.Route, at) }},
		{"leave", func() error { return l.Leave(hunter.ID, at) }},
		{"join", func() error { _, err := l.Join(l.Game.JoinCodes.Runner, name, NewPlayerToken(), at); return err }},
		{"change caught to finished", func() error { return l.Finish(runner.ID, at, true) }},
		{"change finished to caught", func() error { return l.MarkCaught(l.Players[1].ID, at) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := Lobby{Game: l.Game, Players: append([]Player(nil), l.Players...)}
			var rule *RuleError
			if err := tc.run(); !errors.As(err, &rule) {
				t.Fatalf("expected a rule error: %v", err)
			}
			if l.Game != before.Game || !reflect.DeepEqual(l.Players, before.Players) {
				t.Fatal("rejected action changed the game or players")
			}
		})
	}
	for _, p := range l.Players {
		for _, j := range l.JokersFor(p, at) {
			if j.Available {
				t.Fatal("ended game offers an available joker")
			}
		}
	}
}

func TestEndRequiresRunningGameWithRunners(t *testing.T) {
	for _, l := range []Lobby{
		{},
		{Game: Game{StartsAt: start}},
		{Game: Game{StartsAt: start}, Players: []Player{{Team: Hunters}}},
		{Players: []Player{{Team: Runners}}},
		{Game: Game{StartsAt: start.Add(time.Minute)}, Players: []Player{{Team: Runners}}},
	} {
		l.endIfAllRunnersDone(start)
		if !l.Game.EndedAt.IsZero() {
			t.Fatal("empty, hunter-only, active-runner or pre-start game ended")
		}
	}
}

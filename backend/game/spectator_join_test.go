package game

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func joinSpectator(t *testing.T, l *Lobby, name string, at time.Time) Player {
	t.Helper()
	parsed, _ := ParsePlayerName(name)
	p, err := l.Join(l.Game.JoinCodes.Spectator, parsed, NewPlayerToken(), at)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSpectatorJoinCodesAndCapacity(t *testing.T) {
	l := newLobby()
	codes := l.Game.JoinCodes
	if codes.Spectator == codes.Runner || codes.Spectator == codes.Hunter || len(codes.Spectator.String()) != 6 {
		t.Fatal("spectator code must be a separate six-digit code")
	}
	if team, ok := codes.TeamFor(codes.Spectator); !ok || team != Spectators {
		t.Fatal("spectator code selects wrong affiliation")
	}
	if _, ok := (JoinCodes{}).TeamFor(JoinCode{}); ok {
		t.Fatal("empty code must not permit joining")
	}
	for i := range maxPlayers {
		p := joinSpectator(t, &l, fmt.Sprintf("Viewer %d", i), start)
		if p.Team != Spectators || p.Role() != RoleSpectator || p.active() || p.Ready || !p.Position.IsZero() {
			t.Fatalf("spectator joined as a player: %+v", p)
		}
	}
	if l.canStart() || !l.Game.StartsAt.IsZero() {
		t.Fatal("spectators alone can start a game")
	}
	name, _ := ParsePlayerName("Extra")
	var rule *RuleError
	for _, code := range []JoinCode{codes.Runner, codes.Hunter, codes.Spectator} {
		if _, err := l.Join(code, name, NewPlayerToken(), start); !errors.As(err, &rule) {
			t.Fatalf("spectators must count toward capacity: %v", err)
		}
	}
}

func TestSpectatorJoiningLeavingDoesNotAffectCountdown(t *testing.T) {
	l := newLobby()
	watcher := joinSpectator(t, &l, "Viewer", start)
	duplicate, _ := ParsePlayerName(" viewer ")
	if _, err := l.Join(l.Game.JoinCodes.Runner, duplicate, NewPlayerToken(), start); err == nil {
		t.Fatal("name uniqueness must include spectators")
	}
	runner := join(t, &l, Runners, "Ada", start)
	hunter := join(t, &l, Hunters, "Bob", start)
	if err := l.SetReady(runner.ID, true, start, false); err != nil {
		t.Fatal(err)
	}
	if l.canStart() {
		t.Fatal("spectator substituted for an unready hunter")
	}
	if err := l.SetReady(hunter.ID, true, start, false); err != nil {
		t.Fatal(err)
	}
	scheduled := l.Game.StartsAt
	if scheduled != start.Add(CountdownDuration) {
		t.Fatal("unready spectator blocked start")
	}
	late := joinSpectator(t, &l, "Late", start.Add(time.Second))
	for _, p := range []Player{watcher, late} {
		if err := l.Leave(p.ID, start.Add(2*time.Second)); err != nil {
			t.Fatal(err)
		}
		if l.Game.StartsAt != scheduled {
			t.Fatal("spectator joining/leaving restarted countdown")
		}
	}
	name, _ := ParsePlayerName("Too late")
	for _, at := range []time.Time{scheduled, scheduled.Add(time.Second)} {
		if _, err := l.Join(l.Game.JoinCodes.Spectator, name, NewPlayerToken(), at); err == nil {
			t.Fatal("spectator joined at/after start")
		}
	}
	l.Game.EndedAt = scheduled.Add(time.Minute)
	if _, err := l.Join(l.Game.JoinCodes.Spectator, name, NewPlayerToken(), l.Game.EndedAt); err == nil {
		t.Fatal("spectator joined an ended game")
	}
}

func TestDirectSpectatorPermissionsAndVisibility(t *testing.T) {
	l := newLobby()
	viewer := joinSpectator(t, &l, "Viewer", start)
	runner := join(t, &l, Runners, "Ada", start)
	join(t, &l, Hunters, "Bob", start)
	line, _ := NewSketchLine([]Coordinates{l.Game.Route.Start.Coordinates, l.Game.Route.Stop.Coordinates})
	team, _ := ParseHunterTeamName("Pack")
	cell, _ := ParseCell("C3")
	for _, phase := range []Status{StatusLobby, StatusCountdown, StatusRunning, StatusEnded} {
		l.Game.StartsAt, l.Game.EndedAt = time.Time{}, time.Time{}
		if phase == StatusCountdown {
			l.Game.StartsAt = start.Add(time.Second)
		}
		if phase == StatusRunning || phase == StatusEnded {
			l.Game.StartsAt = start.Add(-time.Minute)
		}
		if phase == StatusEnded {
			l.Game.EndedAt = start
		}
		for _, action := range []func() error{
			func() error { return l.SetReady(viewer.ID, true, start, true) },
			func() error { return l.SetReady(viewer.ID, false, start, true) },
			func() error { return l.SetHunterTeam(viewer.ID, team, start) },
			func() error { return l.UpdatePosition(viewer.ID, runner.Position, start, true) },
			func() error { return l.MarkCaught(viewer.ID, start) },
			func() error { return l.Finish(viewer.ID, start, true) },
			func() error { return l.CanRegenerateRoute(viewer.ID, start) },
			func() error { return l.AddSketch(viewer.ID, line, start) },
			func() error { return l.ClearSketches(viewer.ID, start) },
		} {
			var rule *RuleError
			if err := action(); !errors.As(err, &rule) {
				t.Fatalf("spectator action allowed in %s: %v", phase, err)
			}
		}
		for _, choice := range []JokerChoice{ShowRunners{}, QueryZone{Cell: cell}, Compass{}, ShowHunters{}, ZoneOnly{}, FakePosition{Point: l.Game.Playground.Center()}} {
			if err := l.UseJoker(viewer.ID, choice, start); err == nil {
				t.Fatalf("spectator used %s", choice.Kind())
			}
		}
		if len(l.JokersFor(viewer, start)) != 0 || len(l.SketchesFor(viewer, start)) != 0 {
			t.Fatal("direct spectator received team-specific gameplay data")
		}
		for _, p := range l.VisibleTo(viewer, start) {
			wantPosition := phase == StatusRunning && p.Team != Spectators
			if !p.Position.IsZero() != wantPosition || len(p.Reveals) != 0 {
				t.Fatalf("wrong spectator visibility in %s: %+v", phase, p)
			}
		}
	}
	// Retiring the only runner must end the game despite the direct spectator.
	l.Game.EndedAt = time.Time{}
	if err := l.MarkCaught(runner.ID, start); err != nil || l.Game.Status(start) != StatusEnded {
		t.Fatalf("spectator blocked automatic game end: %v", err)
	}
}

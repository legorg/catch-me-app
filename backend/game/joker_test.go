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

func TestParseJokerKind(t *testing.T) {
	if kind, err := ParseJokerKind("showRunners"); err != nil || kind != JokerShowRunners {
		t.Fatalf("known joker rejected: %v", err)
	}
	var validation *ValidationError
	if _, err := ParseJokerKind("teleport"); !errors.As(err, &validation) {
		t.Fatalf("unknown joker accepted: %v", err)
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
	if got := only(l.JokersFor(bob, at)); !got.Available || got.Use != nil {
		t.Fatalf("unused joker: %+v", got)
	}
	if err := l.UseJoker(bob.ID, showRunners, at); err != nil {
		t.Fatal(err)
	}
	used := only(l.JokersFor(dan, at))
	if use, ok := used.Use.(ShowRunnersUse); used.Available || !ok || use.Team != bob.HunterTeam || !use.At.Equal(at) || len(use.Positions) != 2 {
		t.Fatalf("a teammate must see the result of the use: %+v", used)
	}
	if got := only(l.JokersFor(eve, at)); !got.Available || got.Use != nil {
		t.Fatalf("another hunter team sees nothing of it: %+v", got)
	}
	if got := l.JokersFor(runner, at); len(got) != 2 || got[0].Kind != JokerZoneOnly || got[1].Kind != JokerFakePosition || !got[0].Available {
		t.Fatalf("runners see only their own jokers: %+v", got)
	}
}

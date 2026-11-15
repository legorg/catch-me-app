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

func TestShowRunnersJokerRules(t *testing.T) {
	l, runner, bob, dan, eve, running := jokerLobby(t)
	var rule *RuleError
	if err := l.UseJoker(bob.ID, JokerShowRunners, running.Add(-time.Second)); !errors.As(err, &rule) {
		t.Fatalf("a joker before the game runs must be refused: %v", err)
	}
	if err := l.UseJoker(runner.ID, JokerShowRunners, running); !errors.As(err, &rule) {
		t.Fatalf("runners have no jokers: %v", err)
	}
	if err := l.UseJoker(PlayerID{}, JokerShowRunners, running); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("unknown player: %v", err)
	}

	if err := l.UseJoker(bob.ID, JokerShowRunners, running.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := l.UseJoker(bob.ID, JokerShowRunners, running.Add(2*time.Minute)); !errors.As(err, &rule) {
		t.Fatalf("a joker can only be used once: %v", err)
	}
	if err := l.UseJoker(dan.ID, JokerShowRunners, running.Add(2*time.Minute)); !errors.As(err, &rule) {
		t.Fatalf("a team shares one use: %v", err)
	}
	if err := l.UseJoker(eve.ID, JokerShowRunners, running.Add(2*time.Minute)); err != nil {
		t.Fatalf("another hunter team has its own joker: %v", err)
	}
}

func TestJokersAreVisibleToTheirTeamOnly(t *testing.T) {
	l, runner, bob, dan, eve, running := jokerLobby(t)
	at := running.Add(time.Minute)
	only := func(states []JokerState) JokerState {
		t.Helper()
		if len(states) != 1 || states[0].Kind != JokerShowRunners {
			t.Fatalf("expected the show-runners joker: %+v", states)
		}
		return states[0]
	}
	if got := only(l.JokersFor(bob, running.Add(-time.Second))); got.Available {
		t.Fatal("not available before the game runs")
	}
	if got := only(l.JokersFor(bob, at)); !got.Available || !got.UsedAt.IsZero() || len(got.Positions) != 0 {
		t.Fatalf("unused joker: %+v", got)
	}
	if err := l.UseJoker(bob.ID, JokerShowRunners, at); err != nil {
		t.Fatal(err)
	}
	used := only(l.JokersFor(dan, at))
	if used.Available || !used.UsedAt.Equal(at) || len(used.Positions) != 2 {
		t.Fatalf("a teammate must see the result of the use: %+v", used)
	}
	if got := only(l.JokersFor(eve, at)); !got.Available || len(got.Positions) != 0 {
		t.Fatalf("another hunter team sees nothing of it: %+v", got)
	}
	if got := l.JokersFor(runner, at); len(got) != 0 {
		t.Fatalf("runners have no jokers: %+v", got)
	}
}

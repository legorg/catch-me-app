package game

import (
	"errors"
	"slices"
	"testing"
	"time"
)

var showRunners JokerChoice = ShowRunners{}

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
	var validation *ValidationError
	if err := l.UseJoker(bob.ID, nil, running); !errors.As(err, &validation) {
		t.Fatalf("a missing choice is invalid: %v", err)
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

func TestRunnersLearnWhichTeamLocatedThemAndWhen(t *testing.T) {
	l, ada, bob, _, eve, running := jokerLobby(t)
	cy := l.Players[1]
	first, caught, second := running.Add(time.Minute), running.Add(2*time.Minute), running.Add(3*time.Minute)
	steps := []func() error{
		func() error { return l.UseJoker(bob.ID, showRunners, first) },
		func() error { return l.MarkCaught(ada.ID, caught) },
		func() error { return l.UseJoker(eve.ID, showRunners, second) },
		func() error { return l.UseJoker(eve.ID, Compass{}, second) },
	}
	for _, step := range steps {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	ada, _ = l.Player(ada.ID)
	if got := l.LocatedFor(cy); !slices.Equal(got, []Located{{Team: bob.HunterTeam, At: first}, {Team: eve.HunterTeam, At: second}}) {
		t.Fatalf("Cy must learn both uses: %+v", got)
	}
	if got := l.LocatedFor(ada); !slices.Equal(got, []Located{{Team: bob.HunterTeam, At: first}}) {
		t.Fatalf("a use after the catch did not show Ada: %+v", got)
	}
	if got := l.LocatedFor(bob); got != nil {
		t.Fatalf("hunters are not told: %+v", got)
	}
}

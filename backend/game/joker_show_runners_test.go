package game

import (
	"errors"
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

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"catch-me/backend/game"
)

func TestMarkCaughtUsesAuthenticatedPlayerAndServerTime(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	repository := newMemoryRepository()
	s := newService(repository, at)
	token := game.NewPlayerToken()
	runner := game.Player{ID: game.NewPlayerID(), Team: game.Runners, TokenHash: token.Hash()}
	saved := game.Game{SessionID: game.NewSessionID(), StartsAt: at.Add(-time.Minute)}
	repository.lobbies[saved.SessionID] = game.Lobby{Game: saved, Players: []game.Player{runner}}
	view, err := s.MarkCaught(ctx, token)
	if err != nil || view.Me.ID != runner.ID || view.Me.Runner.Status() != game.RunnerCaught || view.Me.Runner.EndedAt() != at || view.Now != at {
		t.Fatalf("catch must use the authenticated player and server time: %+v %v", view, err)
	}
	if _, err := s.State(ctx, token); err != nil {
		t.Fatalf("caught runner must remain able to read game state: %v", err)
	}
	if _, err := s.MarkCaught(ctx, game.NewPlayerToken()); !errors.Is(err, game.ErrUnauthorized) {
		t.Fatalf("unknown token must be refused: %v", err)
	}
	failure := errors.New("storage unavailable")
	repository.err = failure
	if _, err := s.MarkCaught(ctx, token); !errors.Is(err, failure) {
		t.Fatalf("storage errors must remain inspectable: %v", err)
	}
}

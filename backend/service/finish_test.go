package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"catch-me/backend/game"
)

func TestFinishUsesServerTimeAndSharedOverride(t *testing.T) {
	for _, skip := range []bool{false, true} {
		ctx := context.Background()
		at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
		repository := newMemoryRepository()
		s := newService(repository, at)
		s.skipStartPositionCheck = skip
		token := game.NewPlayerToken()
		runner := game.Player{ID: game.NewPlayerID(), Team: game.Runners, TokenHash: token.Hash()}
		saved := game.Game{SessionID: game.NewSessionID(), StartsAt: at.Add(-time.Minute)}
		if !skip {
			runner.Position, _ = game.NewPosition(saved.Route.Stop.Coordinates, 5, at)
		}
		repository.lobbies[saved.SessionID] = game.Lobby{Game: saved, Players: []game.Player{runner}}
		view, err := s.Finish(ctx, token)
		if err != nil || view.Me.ID != runner.ID || view.Me.Runner.Status() != game.RunnerFinished || view.Me.Runner.EndedAt() != at || view.Now != at || view.StartPositionCheckDisabled != skip {
			t.Fatalf("finish must use authenticated player/server settings: %+v %v", view, err)
		}
		if _, err := s.State(ctx, token); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Finish(ctx, game.NewPlayerToken()); !errors.Is(err, game.ErrUnauthorized) {
			t.Fatal(err)
		}
		failure := errors.New("storage unavailable")
		repository.err = failure
		if _, err := s.Finish(ctx, token); !errors.Is(err, failure) {
			t.Fatal(err)
		}
	}
}

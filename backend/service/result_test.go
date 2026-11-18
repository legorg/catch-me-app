package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"catch-me/backend/game"
)

func TestResultLookupUsesFullIDWithoutPlayerToken(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	repository := newMemoryRepository()
	s := newService(repository, at)
	id := game.NewSessionID()
	if _, err := s.Result(ctx, id); !errors.Is(err, game.ErrNotFound) {
		t.Fatalf("unknown ID: %v", err)
	}
	repository.lobbies[id] = game.Lobby{Game: game.Game{SessionID: id, StartsAt: at}}
	if _, err := s.Result(ctx, id); !errors.Is(err, game.ErrNotFound) {
		t.Fatalf("running game must stay hidden: %v", err)
	}
	l := repository.lobbies[id]
	l.Game.EndedAt = at.Add(time.Minute)
	repository.lobbies[id] = l
	result, err := s.Result(ctx, id)
	if err != nil || result.SessionID != id || result.EndedAt != l.Game.EndedAt {
		t.Fatalf("ended game lookup failed: %+v %v", result, err)
	}
	failure := errors.New("storage unavailable")
	repository.err = failure
	if _, err := s.Result(ctx, id); !errors.Is(err, failure) {
		t.Fatalf("storage errors must remain inspectable: %v", err)
	}
}

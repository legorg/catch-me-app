package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"catch-me/backend/game"
)

func TestGameEndIsAtomicAndSurvivesReopening(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "ended.sqlite")
	store := open(t, path)
	saved := create(t, store)
	runner, token := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	_, hunterToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now)
	start := now.Add(time.Minute)
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), now, func(l *game.Lobby) error {
		l.Game.StartsAt = start
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	endedAt := start.Add(time.Minute)
	failure := errors.New("abort transaction")
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), endedAt, func(l *game.Lobby) error {
		if err := l.MarkCaught(runner.ID, endedAt); err != nil {
			return err
		}
		return failure
	}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	unchanged := load(t, store, saved.JoinCodes.Runner, endedAt)
	if unchanged.Game.Status(endedAt) != game.StatusRunning || unchanged.Players[0].Runner.Status() != game.RunnerActive {
		t.Fatal("rollback must preserve both the game and runner state")
	}
	// Force a write failure after saveLobby has written the game's end time.
	if err := store.db.Exec(`CREATE TRIGGER reject_runner_update BEFORE UPDATE ON runners
		BEGIN SELECT RAISE(ABORT, 'test write failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), endedAt, func(l *game.Lobby) error {
		return l.MarkCaught(runner.ID, endedAt)
	}); err == nil {
		t.Fatal("expected the injected runner write failure")
	}
	unchanged = load(t, store, saved.JoinCodes.Runner, endedAt)
	if !unchanged.Game.EndedAt.IsZero() || unchanged.Players[0].Runner.Status() != game.RunnerActive {
		t.Fatal("a failed runner write must roll back the game's end time too")
	}
	if err := store.db.Exec("DROP TRIGGER reject_runner_update").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), endedAt, func(l *game.Lobby) error {
		return l.MarkCaught(runner.ID, endedAt)
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = open(t, path)
	later := endedAt.Add(time.Hour)
	for _, playerToken := range []game.PlayerToken{token, hunterToken} {
		l, err := store.UpdateLobbyByToken(ctx, playerToken.Hash(), later, nil)
		if err != nil || l.Game.Status(later) != game.StatusEnded || !l.Game.EndedAt.Equal(endedAt) || !l.Game.StartsAt.Equal(start) {
			t.Fatalf("all players must load the persisted game end: %+v %v", l.Game, err)
		}
	}
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), later, func(l *game.Lobby) error {
		return l.MarkCaught(runner.ID, later)
	}); err != nil {
		t.Fatal(err)
	}
	l := load(t, store, saved.JoinCodes.Runner, later)
	p, _ := l.Player(runner.ID)
	if !l.Game.EndedAt.Equal(endedAt) || !p.Runner.EndedAt().Equal(endedAt) {
		t.Fatal("retries must preserve game and runner outcomes")
	}
}

func TestConcurrentRunnerOutcomesEndGame(t *testing.T) {
	ctx := context.Background()
	store := open(t, ":memory:")
	saved := create(t, store)
	ada, adaToken := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	cy, cyToken := join(t, store, saved.JoinCodes.Runner, "Cy", now)
	join(t, store, saved.JoinCodes.Hunter, "Bob", now)
	at := now.Add(time.Minute)
	if _, err := store.UpdateLobbyByToken(ctx, adaToken.Hash(), now, func(l *game.Lobby) error {
		l.Game.StartsAt = at
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	begin, results := make(chan struct{}), make(chan error, 2)
	for i, token := range []game.PlayerToken{adaToken, cyToken} {
		go func() {
			<-begin
			_, err := store.UpdateLobbyByToken(ctx, token.Hash(), at, func(l *game.Lobby) error {
				if i == 0 {
					return l.MarkCaught(ada.ID, at)
				}
				return l.Finish(cy.ID, at, true)
			})
			results <- err
		}()
	}
	close(begin)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	l := load(t, store, saved.JoinCodes.Runner, at)
	a, _ := l.Player(ada.ID)
	c, _ := l.Player(cy.ID)
	if l.Game.Status(at) != game.StatusEnded || !l.Game.EndedAt.Equal(at) || a.Runner.Status() != game.RunnerCaught || c.Runner.Status() != game.RunnerFinished {
		t.Fatalf("concurrent outcomes were lost: %+v", l)
	}
}

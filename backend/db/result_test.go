package db

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"catch-me/backend/game"
)

func TestResultLookupPersistsAndDoesNotNeedPositionHistory(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "result.sqlite")
	store := open(t, path)
	saved := create(t, store)
	ada, adaToken := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	cy, cyToken := join(t, store, saved.JoinCodes.Runner, "Cy", now.Add(time.Second))
	hunter, _ := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(2*time.Second))
	if _, err := store.Result(ctx, game.NewSessionID()); !errors.Is(err, game.ErrNotFound) {
		t.Fatalf("unknown game: %v", err)
	}
	start := now.Add(time.Minute)
	for _, startsAt := range []time.Time{{}, start} {
		if _, err := store.UpdateLobbyByToken(ctx, adaToken.Hash(), now, func(l *game.Lobby) error {
			l.Game.StartsAt = startsAt
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Result(ctx, saved.SessionID); !errors.Is(err, game.ErrNotFound) {
			t.Fatalf("unended game returned a result: %v", err)
		}
	}
	if _, err := store.UpdateLobbyByToken(ctx, adaToken.Hash(), start, func(l *game.Lobby) error {
		return l.Finish(ada.ID, start, true)
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Result(ctx, saved.SessionID); !errors.Is(err, game.ErrNotFound) {
		t.Fatal("game with an active runner returned a result")
	}
	end := start.Add(37 * time.Second)
	if _, err := store.UpdateLobbyByToken(ctx, cyToken.Hash(), end, func(l *game.Lobby) error {
		return l.MarkCaught(cy.ID, end)
	}); err != nil {
		t.Fatal(err)
	}
	result, err := store.Result(ctx, saved.SessionID)
	if err != nil || result.SessionID != saved.SessionID || result.StartsAt != start || result.EndedAt != end || len(result.Players) != 3 {
		t.Fatalf("result lookup failed: %+v %v", result, err)
	}
	if result.Players[0].ID != ada.ID || result.Players[0].Runner.Status() != game.RunnerFinished || result.Players[1].ID != cy.ID || result.Players[1].Runner.Status() != game.RunnerCaught || result.Players[2].HunterTeam != hunter.HunterTeam {
		t.Fatal("result lost join order, outcomes or hunter teams")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = open(t, path)
	// An invalid position would fail gameplay loading, but results must not read it.
	if err := store.db.Model(&positionRow{}).Where("player_id = ?", ada.ID.String()).Update("latitude", 999).Error; err != nil {
		t.Fatal(err)
	}
	for _, expire := range []bool{false, true} {
		if expire {
			if err := store.DeletePositionsBefore(ctx, start.Add(time.Nanosecond)); err != nil {
				t.Fatal(err)
			}
		}
		reloaded, err := store.Result(ctx, saved.SessionID)
		if err != nil || !reflect.DeepEqual(reloaded, result) {
			t.Fatalf("reopen/position retention changed the result: %+v %v", reloaded, err)
		}
	}
}

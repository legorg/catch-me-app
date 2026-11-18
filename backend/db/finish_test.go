package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"catch-me/backend/game"
)

func TestFinishPersistenceAndHistoricalJokers(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "finished.sqlite")
	store := open(t, path)
	saved := create(t, store)
	runner, token := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	before, beforeToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
	after, afterToken := join(t, store, saved.JoinCodes.Hunter, "Dan", now.Add(2*time.Second))
	wolves, _ := game.ParseHunterTeamName("Wolves")
	if _, err := store.UpdateLobbyByToken(ctx, afterToken.Hash(), now.Add(3*time.Second), func(l *game.Lobby) error {
		return l.SetHunterTeam(after.ID, wolves, now.Add(3*time.Second))
	}); err != nil {
		t.Fatal(err)
	}
	start := now.Add(time.Minute)
	if err := store.db.Model(&gameRow{ID: saved.SessionID.String()}).Update("starts_at", start).Error; err != nil {
		t.Fatal(err)
	}
	// An earlier joker snapshot must not be updated to the later finish position.
	use := func(p game.Player, token game.PlayerToken, at time.Time, choice game.JokerChoice) {
		t.Helper()
		if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), at, func(l *game.Lobby) error {
			return l.UseJoker(p.ID, choice, at)
		}); err != nil {
			t.Fatal(err)
		}
	}
	use(before, beforeToken, start.Add(time.Second), game.ShowRunners{})
	finishPosition, _ := game.NewPosition(saved.Route.Stop.Coordinates, 5, start.Add(2*time.Second))
	move(t, store, token, finishPosition)
	countBefore := count(t, store)
	at := start.Add(3 * time.Second)
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), at, func(l *game.Lobby) error {
		return l.Finish(runner.ID, at, false)
	}); err != nil {
		t.Fatal(err)
	}
	// At the exact finish timestamp, every hunter joker must exclude this runner.
	origin, _ := game.NewPosition(saved.Playground.Center(), 5, at)
	move(t, store, afterToken, origin)
	cell, _ := saved.Playground.CellAt(saved.Route.Stop.Coordinates)
	for _, choice := range []game.JokerChoice{game.ShowRunners{}, game.QueryZone{Cell: cell}, game.Compass{}} {
		use(after, afterToken, at, choice)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = open(t, path)
	later := at.Add(2 * revealInterval)
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), later, func(l *game.Lobby) error {
		return l.Finish(runner.ID, later, false)
	}); err != nil {
		t.Fatal(err)
	}
	var rule *game.RuleError
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), later, func(l *game.Lobby) error {
		return l.UpdatePosition(runner.ID, finishPosition, later, true)
	}); !errors.As(err, &rule) {
		t.Fatalf("report after reopening: %v", err)
	}
	if count(t, store) != countBefore+1 {
		t.Fatal("finish or retry changed position history")
	}
	lobby := load(t, store, saved.JoinCodes.Hunter, later)
	p, _ := lobby.Player(runner.ID)
	if p.Runner.Status() != game.RunnerFinished || p.Runner.EndedAt() != at || p.Position != finishPosition || len(lobby.Reveals[runner.ID]) != 0 {
		t.Fatalf("finish/history not preserved: %+v", p)
	}
	for _, v := range lobby.VisibleTo(after, later) {
		if v.ID == runner.ID && (v.Runner.Status() != game.RunnerFinished || v.Runner.EndedAt() != at || !v.Position.IsZero() || !v.CaughtPosition.IsZero() || len(v.Reveals) != 0) {
			t.Fatalf("finish leaked: %+v", v)
		}
	}
	if len(lobby.Jokers) != 4 {
		t.Fatalf("expected four joker snapshots: %+v", lobby.Jokers)
	}
	for i, use := range lobby.Jokers {
		switch u := use.(type) {
		case game.ShowRunnersUse:
			if i == 0 {
				if len(u.Positions) != 1 || u.Positions[0].Position.Coordinates != saved.Route.Start.Coordinates {
					t.Fatalf("historical snapshot changed: %+v", u)
				}
			} else if len(u.Positions) != 0 {
				t.Fatalf("finish in new snapshot: %+v", u)
			}
		case game.QueryZoneUse:
			if u.Found {
				t.Fatal("finish in query")
			}
		case game.CompassUse:
			if len(u.Bearings) != 0 {
				t.Fatal("finish in compass")
			}
		}
	}
	if err := store.DeletePositionsBefore(ctx, start.Add(time.Nanosecond)); err != nil {
		t.Fatal(err)
	}
	lobby = load(t, store, saved.JoinCodes.Hunter, later)
	p, _ = lobby.Player(runner.ID)
	if p.Runner.Status() != game.RunnerFinished || p.Runner.EndedAt() != at || !p.Position.IsZero() {
		t.Fatal("retention must remove positions, not finish status")
	}
}

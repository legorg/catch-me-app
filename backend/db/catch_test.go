package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"catch-me/backend/game"
)

func TestCatchSurvivesReopeningAndObeysPositionRetention(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "caught.sqlite")
	store := open(t, path)
	saved := create(t, store)
	runner, token := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	hunter, _ := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
	start := now.Add(time.Minute)
	if err := store.db.Model(&gameRow{ID: saved.SessionID.String()}).Update("starts_at", start).Error; err != nil {
		t.Fatal(err)
	}
	actual, _ := game.NewPosition(saved.Playground.Center(), 8, start.Add(time.Second))
	move(t, store, token, actual)
	at := start.Add(2 * time.Second)
	before := count(t, store)
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), at, func(l *game.Lobby) error {
		return l.MarkCaught(runner.ID, at)
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = open(t, path)
	later := at.Add(2 * revealInterval)
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), later, func(l *game.Lobby) error {
		return l.MarkCaught(runner.ID, later)
	}); err != nil {
		t.Fatalf("retry after reopening must preserve the catch: %v", err)
	}
	var rule *game.RuleError
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), later, func(l *game.Lobby) error {
		return l.UpdatePosition(runner.ID, actual, later, true)
	}); !errors.As(err, &rule) {
		t.Fatalf("further position must be refused after reopening: %v", err)
	}
	if count(t, store) != before {
		t.Fatal("catching and rejected reports must not add or delete position history")
	}
	lobby := load(t, store, saved.JoinCodes.Hunter, later)
	p, _ := lobby.Player(runner.ID)
	if p.CaughtAt != at || p.Position != actual || len(lobby.Reveals[runner.ID]) != 0 {
		t.Fatalf("catch and last report must survive reopening, without further reveals: %+v", p)
	}
	for _, v := range lobby.VisibleTo(hunter, later) {
		if v.ID == runner.ID && (v.CaughtAt != at || v.CaughtPosition != actual || !v.Position.IsZero() || len(v.Reveals) != 0) {
			t.Fatalf("hunter must only get the frozen catch: %+v", v)
		}
	}
	if err := store.DeletePositionsBefore(ctx, start.Add(time.Nanosecond)); err != nil {
		t.Fatal(err)
	}
	lobby = load(t, store, saved.JoinCodes.Hunter, later)
	for _, v := range lobby.VisibleTo(hunter, later) {
		if v.ID == runner.ID && (v.CaughtAt != at || !v.CaughtPosition.IsZero()) {
			t.Fatalf("retention must remove the catch coordinates, not the caught status: %+v", v)
		}
	}
}

func TestHunterJokerSnapshotsExcludeRunnersCaughtAtUseTime(t *testing.T) {
	for _, kind := range []game.JokerKind{game.JokerShowRunners, game.JokerQueryZone, game.JokerCompass} {
		t.Run(string(kind), func(t *testing.T) {
			ctx := context.Background()
			store := open(t, ":memory:")
			saved := create(t, store)
			runner, token := join(t, store, saved.JoinCodes.Runner, "Ada", now)
			otherRunner, otherToken := join(t, store, saved.JoinCodes.Runner, "Cy", now.Add(time.Second))
			beforeHunter, beforeToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(2*time.Second))
			afterHunter, afterToken := join(t, store, saved.JoinCodes.Hunter, "Dan", now.Add(3*time.Second))
			wolves, _ := game.ParseHunterTeamName("Wolves")
			if _, err := store.UpdateLobbyByToken(ctx, afterToken.Hash(), now.Add(4*time.Second), func(l *game.Lobby) error {
				return l.SetHunterTeam(afterHunter.ID, wolves, now.Add(4*time.Second))
			}); err != nil {
				t.Fatal(err)
			}
			start := now.Add(time.Minute)
			if err := store.db.Model(&gameRow{ID: saved.SessionID.String()}).Update("starts_at", start).Error; err != nil {
				t.Fatal(err)
			}
			otherPosition, _ := game.NewPosition(saved.Route.Stop.Coordinates, 5, start)
			move(t, store, otherToken, otherPosition)
			origin, _ := game.NewPosition(saved.Playground.Center(), 5, start)
			move(t, store, beforeToken, origin)
			move(t, store, afterToken, origin)
			cell, _ := saved.Playground.CellAt(saved.Route.Start.Coordinates)
			var choice game.JokerChoice
			switch kind {
			case game.JokerShowRunners:
				choice = game.ShowRunners{}
			case game.JokerQueryZone:
				choice = game.QueryZone{Cell: cell}
			case game.JokerCompass:
				choice = game.Compass{}
			}
			use := func(p game.Player, token game.PlayerToken, at time.Time) {
				t.Helper()
				if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), at, func(l *game.Lobby) error {
					return l.UseJoker(p.ID, choice, at)
				}); err != nil {
					t.Fatal(err)
				}
			}
			use(beforeHunter, beforeToken, start.Add(time.Second))
			caughtAt := start.Add(2 * time.Second)
			if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), caughtAt, func(l *game.Lobby) error {
				return l.MarkCaught(runner.ID, caughtAt)
			}); err != nil {
				t.Fatal(err)
			}
			// Use at the exact catch timestamp, not only strictly after it.
			use(afterHunter, afterToken, caughtAt)
			lobby := load(t, store, saved.JoinCodes.Hunter, caughtAt.Add(revealInterval))
			if len(lobby.Jokers) != 2 {
				t.Fatalf("expected both snapshots: %+v", lobby.Jokers)
			}
			for i, use := range lobby.Jokers {
				switch u := use.(type) {
				case game.ShowRunnersUse:
					if len(u.Positions) != 2-i || (i == 1 && u.Positions[0].PlayerID != otherRunner.ID) {
						t.Fatalf("snapshot must include only runners active at use: %+v", u)
					}
				case game.QueryZoneUse:
					if u.Found != (i == 0) {
						t.Fatalf("zone answer must use catch state at use: %+v", u)
					}
				case game.CompassUse:
					if len(u.Bearings) != 2-i {
						t.Fatalf("compass must use catch state at use: %+v", u)
					}
				}
			}
		})
	}
}

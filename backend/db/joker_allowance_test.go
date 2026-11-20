package db

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"catch-me/backend/game"
)

func createWithJokerAllowance(t *testing.T, store *Store, uses int) game.Game {
	t.Helper()
	saved := fixture(t, 52.52, 2000, strings.Repeat("a", 32))
	a, err := game.ParseJokerAllowance(uses)
	if err != nil {
		t.Fatal(err)
	}
	saved.JokerSettings = game.JokerSettings{
		ShowRunners: a, QueryZone: a, Compass: a, Trap: game.TrapSettings{JokerAllowance: a},
		ZoneOnly: a, FakePosition: a, ShowHunters: a,
	}
	if err := store.Create(context.Background(), saved); err != nil {
		t.Fatal(err)
	}
	return saved
}

func storedJokerUse(store *Store, token game.PlayerToken, choice game.JokerChoice, at time.Time) (game.Lobby, error) {
	return store.UpdateLobbyByToken(context.Background(), token.Hash(), at, func(l *game.Lobby) error {
		me, ok := l.PlayerByToken(token.Hash())
		if !ok {
			return game.ErrUnauthorized
		}
		return l.UseJoker(me.ID, choice, at)
	})
}

func TestJokerScopeIndexIsNotUnique(t *testing.T) {
	store := open(t, ":memory:")
	var indexes []struct {
		Name   string
		Unique int
	}
	if err := store.db.Raw("PRAGMA index_list('jokers')").Scan(&indexes).Error; err != nil {
		t.Fatal(err)
	}
	found := false
	for _, index := range indexes {
		if index.Unique != 0 {
			t.Fatalf("joker uses must not have a unique scope index: %+v", index)
		}
		if index.Name == "joker_scope" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing joker scope index")
	}
}

func TestEveryJokerKeepsMultipleResultsAfterReopening(t *testing.T) {
	for _, kind := range []game.JokerKind{
		game.JokerShowRunners, game.JokerQueryZone, game.JokerCompass, game.JokerTrap,
		game.JokerZoneOnly, game.JokerFakePosition, game.JokerShowHunters,
	} {
		t.Run(string(kind), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "jokers.sqlite")
			store := open(t, path)
			saved := createWithJokerAllowance(t, store, 2)
			runner, runnerToken := join(t, store, saved.JoinCodes.Runner, "Ada", now)
			hunter, hunterToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
			startsAt := now.Add(time.Minute)
			if err := store.db.Model(&gameRow{ID: saved.SessionID.String()}).Update("starts_at", startsAt).Error; err != nil {
				t.Fatal(err)
			}
			actor, token := hunter, hunterToken
			if kind == game.JokerZoneOnly || kind == game.JokerFakePosition || kind == game.JokerShowHunters {
				actor, token = runner, runnerToken
			}
			var want []game.JokerUse
			var lastChoice game.JokerChoice
			for i, cellName := range []string{"A1", "B1"} {
				cell, _ := game.ParseCell(cellName)
				point := boundsCenter(saved.Playground.CellBounds(cell))
				choices := map[game.JokerKind]game.JokerChoice{
					game.JokerShowRunners: game.ShowRunners{}, game.JokerQueryZone: game.QueryZone{Cell: cell},
					game.JokerCompass: game.Compass{}, game.JokerTrap: game.Trap{Point: point},
					game.JokerZoneOnly: game.ZoneOnly{}, game.JokerFakePosition: game.FakePosition{Point: point},
					game.JokerShowHunters: game.ShowHunters{},
				}
				at := startsAt.Add(time.Duration(i)*revealInterval + 2*time.Second)
				runnerPoint, hunterPoint := saved.Route.Start.Coordinates, saved.Playground.Center()
				if i == 1 {
					runnerPoint, hunterPoint = hunterPoint, runnerPoint
				}
				if kind == game.JokerQueryZone && i == 0 {
					runnerPoint = point
				}
				runnerReport, _ := game.NewPosition(runnerPoint, 5, at.Add(-time.Second))
				hunterReport, _ := game.NewPosition(hunterPoint, 5, at.Add(-time.Second))
				move(t, store, runnerToken, runnerReport)
				move(t, store, hunterToken, hunterReport)
				lastChoice = choices[kind]
				lobby, err := storedJokerUse(store, token, lastChoice, at)
				if err != nil || len(lobby.Jokers) != i+1 {
					t.Fatalf("use %d: got %+v, %v", i+1, lobby.Jokers, err)
				}
				want = append(want, lobby.Jokers[i])
			}
			if kind == game.JokerTrap {
				triggerAt := startsAt.Add(revealInterval + game.TrapActivationDelay + 3*time.Second)
				for i, use := range want {
					trap := use.(game.TrapUse)
					trap.TriggeredAt = triggerAt.Add(time.Duration(i) * time.Second)
					report, _ := game.NewPosition(trap.Point, 5, trap.TriggeredAt)
					move(t, store, runnerToken, report)
					want[i] = trap
				}
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			store = open(t, path)
			at := startsAt.Add(2*revealInterval + time.Second)
			lobby := load(t, store, saved.JoinCodes.Hunter, at)
			if !reflect.DeepEqual(lobby.Jokers, want) {
				t.Fatalf("individual joker results changed after reopening:\n got %+v\nwant %+v", lobby.Jokers, want)
			}
			found := false
			for _, state := range lobby.JokersFor(actor, at) {
				if state.Kind == kind {
					found = true
				}
				if state.Kind == kind && (state.Available || !reflect.DeepEqual(state.Uses, want)) {
					t.Fatalf("incorrect persisted remaining uses/history: %+v", state)
				}
			}
			if !found {
				t.Fatalf("missing persisted joker %s", kind)
			}
			if kind == game.JokerFakePosition || kind == game.JokerZoneOnly {
				for _, visible := range lobby.VisibleTo(hunter, at) {
					if visible.ID != runner.ID {
						continue
					}
					if len(visible.Reveals) != 2 {
						t.Fatalf("expected two reveals: %+v", visible.Reveals)
					}
					for i, reveal := range visible.Reveals {
						if kind == game.JokerFakePosition {
							shown, ok := reveal.(game.PositionReveal)
							if !ok || shown.Position.Coordinates != want[i].(game.FakePositionUse).Point {
								t.Fatalf("fake use %d did not affect its own reveal: %+v", i, reveal)
							}
						} else if _, ok := reveal.(game.ZoneReveal); !ok {
							t.Fatalf("zone use %d exposed a real position: %+v", i, reveal)
						}
					}
				}
			}
			var rule *game.RuleError
			if _, err := storedJokerUse(store, token, lastChoice, at); !errors.As(err, &rule) {
				t.Fatalf("reopening must not restore spent allowance: %v", err)
			}
			if got := load(t, store, saved.JoinCodes.Hunter, at).Jokers; !reflect.DeepEqual(got, want) {
				t.Fatalf("rejected use changed the persisted history: %+v", got)
			}
		})
	}
}

func TestConcurrentJokerUsesCannotExceedTheAllowance(t *testing.T) {
	store := open(t, ":memory:")
	saved := createWithJokerAllowance(t, store, 2)
	join(t, store, saved.JoinCodes.Runner, "Ada", now)
	_, hunterToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
	startsAt := now.Add(time.Minute)
	if err := store.db.Model(&gameRow{ID: saved.SessionID.String()}).Update("starts_at", startsAt).Error; err != nil {
		t.Fatal(err)
	}
	errorsCh := make(chan error, 6)
	var wg sync.WaitGroup
	for i := 0; i < cap(errorsCh); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := storedJokerUse(store, hunterToken, game.ShowRunners{}, startsAt)
			errorsCh <- err
		}()
	}
	wg.Wait()
	close(errorsCh)
	accepted := 0
	for err := range errorsCh {
		if err == nil {
			accepted++
			continue
		}
		var rule *game.RuleError
		if !errors.As(err, &rule) {
			t.Fatalf("expected a domain rejection, not a database error: %v", err)
		}
	}
	if got := load(t, store, saved.JoinCodes.Hunter, startsAt).Jokers; accepted != 2 || len(got) != 2 {
		t.Fatalf("concurrent uses exceeded allowance: accepted %d, stored %+v", accepted, got)
	}
}

func TestJokerUsesRollBackTogetherWhenTheAllowanceIsExceeded(t *testing.T) {
	store := open(t, ":memory:")
	saved := createWithJokerAllowance(t, store, 2)
	join(t, store, saved.JoinCodes.Runner, "Ada", now)
	hunter, hunterToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
	startsAt := now.Add(time.Minute)
	if err := store.db.Model(&gameRow{ID: saved.SessionID.String()}).Update("starts_at", startsAt).Error; err != nil {
		t.Fatal(err)
	}
	_, err := store.UpdateLobbyByToken(context.Background(), hunterToken.Hash(), startsAt, func(l *game.Lobby) error {
		for i := 0; i < 3; i++ {
			if err := l.UseJoker(hunter.ID, game.ShowRunners{}, startsAt); err != nil {
				return err
			}
		}
		return nil
	})
	var rule *game.RuleError
	if !errors.As(err, &rule) {
		t.Fatalf("expected exhausted allowance: %v", err)
	}
	if got := load(t, store, saved.JoinCodes.Hunter, startsAt).Jokers; len(got) != 0 {
		t.Fatalf("failed transaction left partial joker uses: %+v", got)
	}
	if _, err := storedJokerUse(store, hunterToken, game.ShowRunners{}, startsAt); err != nil {
		t.Fatalf("rollback spent the allowance: %v", err)
	}
}

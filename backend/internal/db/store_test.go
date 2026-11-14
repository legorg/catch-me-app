package db

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"catch-me/backend/internal/game"
)

var now = time.Date(2026, 11, 8, 12, 0, 0, 123456789, time.UTC)

func fixture(t *testing.T, latitude float64, meters int, id string) game.Game {
	t.Helper()
	sessionID, err := game.ParseSessionID(id)
	if err != nil {
		t.Fatal(err)
	}
	center, err := game.ParseCoordinates(latitude, 13.405)
	if err != nil {
		t.Fatal(err)
	}
	size, err := game.ParseCellSize(meters)
	if err != nil {
		t.Fatal(err)
	}
	p, err := game.NewPlayground(center, size)
	if err != nil {
		t.Fatal(err)
	}
	return game.Game{SessionID: sessionID, CreatedAt: now, Playground: p, JoinCodes: game.NewJoinCodes()}
}

func open(t *testing.T, path string) *Store {
	t.Helper()
	store, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func create(t *testing.T, store *Store) game.Game {
	t.Helper()
	saved := fixture(t, 52.52, 2000, strings.Repeat("a", 32))
	if err := store.Create(context.Background(), saved); err != nil {
		t.Fatal(err)
	}
	return saved
}

func load(t *testing.T, store *Store, code game.JoinCode, at time.Time) game.Lobby {
	t.Helper()
	lobby, err := store.UpdateLobbyByJoinCode(context.Background(), code, at, nil)
	if err != nil {
		t.Fatal(err)
	}
	return lobby
}

// join adds a player; runners report a position in their start cell.
func join(t *testing.T, store *Store, code game.JoinCode, name string, at time.Time) (game.Player, game.PlayerToken) {
	t.Helper()
	parsed, _ := game.ParsePlayerName(name)
	token := game.NewPlayerToken()
	var player game.Player
	_, err := store.UpdateLobbyByJoinCode(context.Background(), code, at, func(l *game.Lobby) error {
		var err error
		if player, err = l.Join(code, parsed, token, at); err != nil || player.Route.IsZero() {
			return err
		}
		player.Position, _ = game.NewPosition(l.Game.Playground.CellCenter(player.Route.Start), 5, at)
		return l.UpdatePosition(player.ID, player.Position, at, false)
	})
	if err != nil {
		t.Fatal(err)
	}
	return player, token
}

func move(t *testing.T, store *Store, token game.PlayerToken, position game.Position) {
	t.Helper()
	if _, err := store.UpdateLobbyByToken(context.Background(), token.Hash(), position.RecordedAt, func(l *game.Lobby) error {
		me, _ := l.PlayerByToken(token.Hash())
		return l.UpdatePosition(me.ID, position, position.RecordedAt, false)
	}); err != nil {
		t.Fatal(err)
	}
}

func count(t *testing.T, store *Store) int64 {
	t.Helper()
	var n int64
	if err := store.db.Model(&positionRow{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestStorePersistsAfterReopening(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "games.sqlite")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	fixtures := []game.Game{fixture(t, 52.52, 100, strings.Repeat("a", 32)), fixture(t, 0, 5000, strings.Repeat("b", 32))}
	for _, saved := range fixtures {
		if err := store.Create(ctx, saved); err != nil {
			t.Fatal(err)
		}
	}
	_ = store.Close()
	reopened := open(t, path)
	for _, want := range fixtures {
		if got := load(t, reopened, want.JoinCodes.Hunter, now); !reflect.DeepEqual(got.Game, want) || len(got.Players) != 0 {
			t.Fatalf("game did not persist: %+v", got)
		}
	}
	duplicate := fixtures[0]
	duplicate.JoinCodes = game.NewJoinCodes()
	if err := reopened.Create(ctx, duplicate); err == nil {
		t.Fatal("duplicate ID accepted")
	}
	clash := fixture(t, 52.52, 2000, strings.Repeat("c", 32))
	clash.JoinCodes.Runner = fixtures[0].JoinCodes.Hunter
	if err := reopened.Create(ctx, clash); !errors.Is(err, game.ErrJoinCodeTaken) {
		t.Fatalf("code collision across columns not detected: %v", err)
	}
	if _, err := reopened.UpdateLobbyByJoinCode(ctx, game.NewJoinCode(), now, nil); !errors.Is(err, game.ErrNotFound) {
		t.Fatalf("not-found not translated: %v", err)
	}
}

func TestOpenRejectsMissingParent(t *testing.T) {
	if _, err := Open(context.Background(), filepath.Join(t.TempDir(), "missing", "games.sqlite")); err == nil {
		t.Fatal("database opened in missing directory")
	}
}

// Invalid stored data must be a server error, never a client validation error.
func TestInvalidStoredDataIsNotAValidationError(t *testing.T) {
	store := open(t, ":memory:")
	saved := create(t, store)
	_, token := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	for _, corrupt := range []string{"UPDATE positions SET latitude = 95", "UPDATE games SET cell_size_meters = 1550"} {
		if err := store.db.Exec(corrupt).Error; err != nil {
			t.Fatal(err)
		}
		_, err := store.UpdateLobbyByToken(context.Background(), token.Hash(), now, nil)
		var validation *game.ValidationError
		if err == nil || errors.As(err, &validation) {
			t.Fatalf("invalid stored data after %q: %v", corrupt, err)
		}
	}
}

func TestLobbyPersists(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lobby.sqlite")
	store := open(t, path)
	saved := create(t, store)
	runner, runnerToken := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	hunter, hunterToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
	for _, token := range []game.PlayerToken{runnerToken, hunterToken} {
		if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), now, func(l *game.Lobby) error {
			me, _ := l.PlayerByToken(token.Hash())
			return l.SetReady(me.ID, true, now.Add(time.Second), false)
		}); err != nil {
			t.Fatal(err)
		}
	}
	_ = store.Close()

	store = open(t, path)
	lobby := load(t, store, saved.JoinCodes.Runner, now)
	runner.Ready, hunter.Ready = true, true
	if !lobby.Game.StartsAt.Equal(now.Add(time.Second+game.CountdownDuration)) || !reflect.DeepEqual(lobby.Players, []game.Player{runner, hunter}) {
		t.Fatalf("lobby not persisted: %+v", lobby)
	}
	if _, err := store.UpdateLobbyByToken(ctx, hunterToken.Hash(), now, func(l *game.Lobby) error { return l.Leave(hunter.ID, now) }); err != nil {
		t.Fatal(err)
	}
	if lobby := load(t, store, saved.JoinCodes.Runner, now); len(lobby.Players) != 1 || !lobby.Game.StartsAt.IsZero() {
		t.Fatalf("leave not persisted: %+v", lobby)
	}
	if _, err := store.UpdateLobbyByToken(ctx, hunterToken.Hash(), now, nil); !errors.Is(err, game.ErrNotFound) {
		t.Fatal("removed player's token still works")
	}
	if _, err := store.UpdateLobbyByToken(ctx, runnerToken.Hash(), now, func(l *game.Lobby) error {
		_ = l.SetReady(runner.ID, false, now, false)
		return l.RegenerateRoute(runner.ID, now)
	}); err != nil {
		t.Fatal(err)
	}
	if got := load(t, store, saved.JoinCodes.Runner, now).Players[0].Route; got.RegenerationsLeft != game.MaxRouteRegenerations-1 || got == runner.Route {
		t.Fatalf("regenerated route not persisted: %+v", got)
	}
	failed := errors.New("rule failed")
	if _, err := store.UpdateLobbyByJoinCode(ctx, saved.JoinCodes.Runner, now, func(l *game.Lobby) error {
		l.Players = nil
		return failed
	}); !errors.Is(err, failed) {
		t.Fatal("change error lost")
	}
	if lobby := load(t, store, saved.JoinCodes.Runner, now); len(lobby.Players) != 1 {
		t.Fatal("failed change was saved")
	}
}

func TestPositionsAndReveals(t *testing.T) {
	ctx := context.Background()
	store := open(t, ":memory:")
	saved := create(t, store)
	runner, token := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
	start := now.Add(time.Minute)
	if err := store.db.Model(&gameRow{ID: saved.SessionID.String()}).Update("starts_at", start).Error; err != nil {
		t.Fatal(err)
	}
	reveal := start.Add(game.RevealInterval)
	first, _ := game.NewPosition(saved.Playground.CellCenter(runner.Route.Start), 12.5, reveal.Add(-time.Second))
	second, _ := game.NewPosition(saved.Playground.Center(), 3, reveal)
	move(t, store, token, first)
	if got := load(t, store, saved.JoinCodes.Runner, reveal.Add(-time.Second)).Players[0]; got.Position != first || !got.Reveal.IsZero() {
		t.Fatalf("no reveal before the first reveal time: %+v", got)
	}
	move(t, store, token, second)
	// A reveal shows the last position strictly before the reveal time.
	players := load(t, store, saved.JoinCodes.Runner, reveal.Add(time.Minute)).Players
	if got := players[0]; got.Position != second || got.Reveal != (game.Reveal{Position: first, At: reveal}) {
		t.Fatalf("latest position or reveal wrong: %+v %+v", got.Position, got.Reveal)
	}
	if got := players[1]; !got.Position.IsZero() || !got.Reveal.IsZero() {
		t.Fatalf("hunter without positions got one: %+v", got)
	}
	if got := load(t, store, saved.JoinCodes.Runner, reveal.Add(game.RevealInterval)).Players[0].Reveal; got.Position != second {
		t.Fatalf("second reveal missing: %+v", got)
	}

	if count(t, store) != 3 {
		t.Fatal("every reported position must be kept")
	}
	// Retention counts from the start, not from the report.
	if err := store.DeletePositionsBefore(ctx, start); err != nil || count(t, store) != 3 {
		t.Fatalf("positions deleted before their retention ended: %v", err)
	}
	if err := store.DeletePositionsBefore(ctx, start.Add(time.Nanosecond)); err != nil || count(t, store) != 0 {
		t.Fatalf("expired positions kept: %v", err)
	}
}

func TestPositionsOfUnstartedGamesAndLeavingPlayers(t *testing.T) {
	ctx := context.Background()
	store := open(t, ":memory:")
	saved := create(t, store)
	ada, adaToken := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	join(t, store, saved.JoinCodes.Runner, "Cy", now)
	if _, err := store.UpdateLobbyByToken(ctx, adaToken.Hash(), now, func(l *game.Lobby) error { return l.Leave(ada.ID, now) }); err != nil {
		t.Fatal(err)
	}
	if count(t, store) != 1 {
		t.Fatal("leaving must delete the player's positions")
	}
	// A game that never started expires counting from its creation.
	if err := store.DeletePositionsBefore(ctx, now.Add(time.Nanosecond)); err != nil || count(t, store) != 0 {
		t.Fatalf("positions of an unstarted game kept: %v", err)
	}
}

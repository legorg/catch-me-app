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

func load(t *testing.T, store *Store, code game.JoinCode) game.Lobby {
	t.Helper()
	lobby, err := store.UpdateLobbyByJoinCode(context.Background(), code, nil)
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
	_, err := store.UpdateLobbyByJoinCode(context.Background(), code, func(l *game.Lobby) error {
		var err error
		if player, err = l.Join(code, parsed, token, at); err != nil || player.Route.IsZero() {
			return err
		}
		position, _ := game.NewPosition(l.Game.Playground.CellCenter(player.Route.Start), 5, at)
		player.Position = position
		return l.UpdatePosition(player.ID, position, at, false)
	})
	if err != nil {
		t.Fatal(err)
	}
	return player, token
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
		if got := load(t, reopened, want.JoinCodes.Hunter); !reflect.DeepEqual(got.Game, want) || len(got.Players) != 0 {
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
	if _, err := reopened.UpdateLobbyByJoinCode(ctx, game.NewJoinCode(), nil); !errors.Is(err, game.ErrNotFound) {
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
	saved := fixture(t, 52.52, 2000, strings.Repeat("a", 32))
	if err := store.Create(context.Background(), saved); err != nil {
		t.Fatal(err)
	}
	if err := store.db.Exec("UPDATE games SET cell_size_meters = 1550").Error; err != nil {
		t.Fatal(err)
	}
	_, err := store.UpdateLobbyByJoinCode(context.Background(), saved.JoinCodes.Runner, nil)
	var validation *game.ValidationError
	if err == nil || errors.As(err, &validation) {
		t.Fatalf("invalid stored game: %v", err)
	}
}

func TestLobbyPersists(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lobby.sqlite")
	store := open(t, path)
	saved := fixture(t, 52.52, 2000, strings.Repeat("a", 32))
	if err := store.Create(ctx, saved); err != nil {
		t.Fatal(err)
	}
	runner, runnerToken := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	hunter, hunterToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
	for _, token := range []game.PlayerToken{runnerToken, hunterToken} {
		if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), func(l *game.Lobby) error {
			me, _ := l.PlayerByToken(token.Hash())
			return l.SetReady(me.ID, true, now.Add(time.Second), false)
		}); err != nil {
			t.Fatal(err)
		}
	}
	_ = store.Close()

	store = open(t, path)
	lobby := load(t, store, saved.JoinCodes.Runner)
	runner.Ready, hunter.Ready = true, true
	if !lobby.Game.StartsAt.Equal(now.Add(time.Second+game.CountdownDuration)) || !reflect.DeepEqual(lobby.Players, []game.Player{runner, hunter}) {
		t.Fatalf("lobby not persisted: %+v", lobby)
	}
	if _, err := store.UpdateLobbyByToken(ctx, hunterToken.Hash(), func(l *game.Lobby) error { return l.Leave(hunter.ID, now) }); err != nil {
		t.Fatal(err)
	}
	if lobby := load(t, store, saved.JoinCodes.Runner); len(lobby.Players) != 1 || !lobby.Game.StartsAt.IsZero() {
		t.Fatalf("leave not persisted: %+v", lobby)
	}
	if _, err := store.UpdateLobbyByToken(ctx, hunterToken.Hash(), nil); !errors.Is(err, game.ErrNotFound) {
		t.Fatal("removed player's token still works")
	}
	if _, err := store.UpdateLobbyByToken(ctx, runnerToken.Hash(), func(l *game.Lobby) error {
		_ = l.SetReady(runner.ID, false, now, false)
		return l.RegenerateRoute(runner.ID, now)
	}); err != nil {
		t.Fatal(err)
	}
	if got := load(t, store, saved.JoinCodes.Runner).Players[0].Route; got.RegenerationsLeft != game.MaxRouteRegenerations-1 || got == runner.Route {
		t.Fatalf("regenerated route not persisted: %+v", got)
	}
	failed := errors.New("rule failed")
	if _, err := store.UpdateLobbyByJoinCode(ctx, saved.JoinCodes.Runner, func(l *game.Lobby) error {
		l.Players = nil
		return failed
	}); !errors.Is(err, failed) {
		t.Fatal("change error lost")
	}
	if lobby := load(t, store, saved.JoinCodes.Runner); len(lobby.Players) != 1 {
		t.Fatal("failed change was saved")
	}
}

func TestPositionsRevealsAndHistoryPersist(t *testing.T) {
	ctx := context.Background()
	store := open(t, ":memory:")
	saved := fixture(t, 52.52, 2000, strings.Repeat("a", 32))
	if err := store.Create(ctx, saved); err != nil {
		t.Fatal(err)
	}
	runner, token := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	start := now.Add(time.Minute)
	if err := store.db.Model(&gameRow{ID: saved.SessionID.String()}).Update("starts_at", start).Error; err != nil {
		t.Fatal(err)
	}
	update := func(position game.Position) {
		t.Helper()
		if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), func(l *game.Lobby) error {
			return l.UpdatePosition(runner.ID, position, position.RecordedAt, false)
		}); err != nil {
			t.Fatal(err)
		}
	}
	first, _ := game.NewPosition(saved.Playground.CellCenter(runner.Route.Start), 12.5, start.Add(game.RevealInterval-time.Second))
	second, _ := game.NewPosition(saved.Playground.Center(), 3, start.Add(game.RevealInterval+time.Second))
	update(first)
	update(second)
	got := load(t, store, saved.JoinCodes.Runner).Players[0]
	if got.Position != second || got.Reveal != (game.Reveal{Position: first, At: start.Add(game.RevealInterval)}) {
		t.Fatalf("position or reveal not persisted: %+v %+v", got.Position, got.Reveal)
	}

	var history []historyRow
	if err := store.db.Order("id").Find(&history).Error; err != nil {
		t.Fatal(err)
	}
	if len(history) != 3 {
		t.Fatalf("every reported position must be kept: %+v", history)
	}
	oldest, _ := parsePosition(history[0].Position)
	latest, _ := parsePosition(history[2].Position)
	if oldest != runner.Position || latest != second {
		t.Fatalf("every reported position must be kept in order: %+v", history)
	}
	// Retention counts from the start, not from the report.
	if err := store.DeleteHistoryBefore(ctx, start); err != nil {
		t.Fatal(err)
	}
	if count(t, store) != 3 {
		t.Fatal("history deleted before its retention ended")
	}
	if err := store.DeleteHistoryBefore(ctx, start.Add(time.Nanosecond)); err != nil {
		t.Fatal(err)
	}
	if count(t, store) != 0 {
		t.Fatal("expired history kept")
	}
}

func TestHistoryOfUnstartedGamesAndLeavingPlayers(t *testing.T) {
	ctx := context.Background()
	store := open(t, ":memory:")
	saved := fixture(t, 52.52, 2000, strings.Repeat("a", 32))
	if err := store.Create(ctx, saved); err != nil {
		t.Fatal(err)
	}
	ada, adaToken := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	join(t, store, saved.JoinCodes.Runner, "Cy", now)
	if _, err := store.UpdateLobbyByToken(ctx, adaToken.Hash(), func(l *game.Lobby) error { return l.Leave(ada.ID, now) }); err != nil {
		t.Fatal(err)
	}
	if count(t, store) != 1 {
		t.Fatal("leaving must delete the player's history")
	}
	// A game that never started expires counting from its creation.
	if err := store.DeleteHistoryBefore(ctx, now.Add(time.Nanosecond)); err != nil || count(t, store) != 0 {
		t.Fatalf("history of an unstarted game kept: %v", err)
	}
}

func count(t *testing.T, store *Store) int64 {
	t.Helper()
	var n int64
	if err := store.db.Model(&historyRow{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

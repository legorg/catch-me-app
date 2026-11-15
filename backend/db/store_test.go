package db

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"catch-me/backend/game"
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
	route := game.Route{
		Start: game.Landmark{Title: "Start", Coordinates: p.CellBounds(p.SouthEdgeCells()[0]).Center()},
		Stop:  game.Landmark{Title: "Stop", Coordinates: p.CellBounds(p.NorthEdgeCells()[0]).Center()},
	}
	return game.Game{SessionID: sessionID, CreatedAt: now, Playground: p, JoinCodes: game.NewJoinCodes(), Route: route}
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

// join adds a player; runners report a position at the start, and
// hunters join the hunter team "Pack".
func join(t *testing.T, store *Store, code game.JoinCode, name string, at time.Time) (game.Player, game.PlayerToken) {
	t.Helper()
	parsed, _ := game.ParsePlayerName(name)
	token := game.NewPlayerToken()
	var player game.Player
	_, err := store.UpdateLobbyByJoinCode(context.Background(), code, at, func(l *game.Lobby) error {
		var err error
		if player, err = l.Join(code, parsed, token, at); err != nil {
			return err
		}
		if player.Team == game.Hunters {
			player.HunterTeam, _ = game.ParseHunterTeamName("Pack")
			return l.SetHunterTeam(player.ID, player.HunterTeam, at)
		}
		player.Position, _ = game.NewPosition(l.Game.Route.Start.Coordinates, 5, at)
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
	if before := load(t, store, saved.JoinCodes.Runner, reveal.Add(-time.Second)); before.Players[0].Position != first || len(before.Reveals) != 0 {
		t.Fatalf("no reveal before the first reveal time: %+v %+v", before.Players[0], before.Reveals)
	}
	move(t, store, token, second)
	// A reveal shows the last position strictly before the reveal time.
	lobby := load(t, store, saved.JoinCodes.Runner, reveal.Add(game.RevealInterval/2))
	if got := lobby.Players[0]; got.Position != second {
		t.Fatalf("latest position wrong: %+v", got.Position)
	}
	if got := lobby.Reveals[runner.ID]; len(got) != 1 || got[0] != (game.Reveal{Position: first, At: reveal}) {
		t.Fatalf("reveal wrong: %+v", got)
	}
	if got := lobby.Players[1]; !got.Position.IsZero() || len(lobby.Reveals) != 1 {
		t.Fatalf("hunter without positions got one: %+v %+v", got, lobby.Reveals)
	}
	if got := load(t, store, saved.JoinCodes.Runner, reveal.Add(game.RevealInterval)).Reveals[runner.ID]; len(got) != 2 || got[1].Position != second {
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

func TestRevealsKeepEveryReveal(t *testing.T) {
	store := open(t, ":memory:")
	saved := create(t, store)
	// Joining reports a first position at `now`.
	runner, token := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
	start := now.Add(time.Minute)
	if err := store.db.Model(&gameRow{ID: saved.SessionID.String()}).Update("starts_at", start).Error; err != nil {
		t.Fatal(err)
	}
	reveal := func(n time.Duration) time.Time { return start.Add(n * game.RevealInterval) }
	first, _ := game.NewPosition(saved.Playground.CellCenter(runner.Route.Start), 4, reveal(1).Add(time.Second))
	second, _ := game.NewPosition(saved.Playground.Center(), 3, reveal(3).Add(-time.Second))
	move(t, store, token, first)
	move(t, store, token, second)

	lobby := load(t, store, saved.JoinCodes.Runner, reveal(3).Add(time.Second))
	// Reveal 1 shows the position from joining, reveal 2 the first report (nothing new before reveal 2), reveal 3 the second.
	all := lobby.Reveals[runner.ID]
	if len(all) != 3 || all[0].At != reveal(1) || all[0].Position.RecordedAt != now ||
		all[1].At != reveal(2) || all[1].Position != first || all[2].At != reveal(3) || all[2].Position != second {
		t.Fatalf("reveals wrong: %+v", all)
	}
	if len(lobby.Reveals) != 1 {
		t.Fatalf("only the runner has reveals: %+v", lobby.Reveals)
	}
	if first := load(t, store, saved.JoinCodes.Runner, reveal(1).Add(time.Second)); len(first.Reveals[runner.ID]) != 1 {
		t.Fatal("exactly the first reveal expected")
	}
}

func TestUsedJokerIsStoredAndShowsPositionsAtTheTimeOfUse(t *testing.T) {
	ctx := context.Background()
	store := open(t, ":memory:")
	saved := create(t, store)
	runner, runnerToken := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	hunter, hunterToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
	start := now.Add(time.Minute)
	if err := store.db.Model(&gameRow{ID: saved.SessionID.String()}).Update("starts_at", start).Error; err != nil {
		t.Fatal(err)
	}
	before, _ := game.NewPosition(saved.Playground.CellCenter(runner.Route.Start), 4, start.Add(10*time.Second))
	after, _ := game.NewPosition(saved.Playground.Center(), 3, start.Add(70*time.Second))
	move(t, store, runnerToken, before)
	usedAt := start.Add(60 * time.Second)
	use := func(at time.Time) error {
		_, err := store.UpdateLobbyByToken(ctx, hunterToken.Hash(), at, func(l *game.Lobby) error {
			return l.UseJoker(hunter.ID, game.JokerChoice{Kind: game.JokerShowRunners}, at)
		})
		return err
	}
	if err := use(usedAt); err != nil {
		t.Fatal(err)
	}
	move(t, store, runnerToken, after)

	lobby := load(t, store, saved.JoinCodes.Hunter, start.Add(100*time.Second))
	if len(lobby.Jokers) != 1 || lobby.Jokers[0].Kind != game.JokerShowRunners || !lobby.Jokers[0].UsedAt.Equal(usedAt) || lobby.Jokers[0].Team != hunter.HunterTeam {
		t.Fatalf("used joker not stored: %+v", lobby.Jokers)
	}
	got := lobby.Jokers[0].Positions
	if len(got) != 1 || got[0].PlayerID != runner.ID || got[0].Position != before {
		t.Fatalf("the joker must show where the runner was when it was used, not later: %+v", got)
	}
	var rule *game.RuleError
	if err := use(start.Add(90 * time.Second)); !errors.As(err, &rule) {
		t.Fatalf("a second use must be refused: %v", err)
	}
	if count := load(t, store, saved.JoinCodes.Hunter, start.Add(100*time.Second)); len(count.Jokers) != 1 {
		t.Fatal("a refused use was saved")
	}
}

func TestQueriedZoneIsStoredAndAnsweredAtTheTimeOfUse(t *testing.T) {
	ctx := context.Background()
	store := open(t, ":memory:")
	saved := create(t, store)
	runner, runnerToken := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	hunter, hunterToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
	start := now.Add(time.Minute)
	if err := store.db.Model(&gameRow{ID: saved.SessionID.String()}).Update("starts_at", start).Error; err != nil {
		t.Fatal(err)
	}
	before, _ := game.NewPosition(saved.Playground.CellCenter(runner.Route.Start), 4, start.Add(10*time.Second))
	after, _ := game.NewPosition(saved.Playground.Center(), 3, start.Add(70*time.Second))
	move(t, store, runnerToken, before)
	usedAt := start.Add(60 * time.Second)
	choice, err := game.ParseJokerChoice("queryZone", runner.Route.Start.String())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateLobbyByToken(ctx, hunterToken.Hash(), usedAt, func(l *game.Lobby) error {
		return l.UseJoker(hunter.ID, choice, usedAt)
	}); err != nil {
		t.Fatal(err)
	}
	move(t, store, runnerToken, after)

	lobby := load(t, store, saved.JoinCodes.Hunter, start.Add(100*time.Second))
	if len(lobby.Jokers) != 1 {
		t.Fatalf("used joker not stored: %+v", lobby.Jokers)
	}
	got := lobby.Jokers[0]
	if got.Kind != game.JokerQueryZone || got.Cell != runner.Route.Start || !got.Found || len(got.Positions) != 0 || !got.UsedAt.Equal(usedAt) {
		t.Fatalf("the query must answer yes for where the runner was when it was used, without positions: %+v", got)
	}
}

func TestSketchesPersistUntilClearedOrExpired(t *testing.T) {
	ctx := context.Background()
	store := open(t, ":memory:")
	saved := create(t, store)
	join(t, store, saved.JoinCodes.Runner, "Ada", now)
	hunter, hunterToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
	start := now.Add(time.Minute)
	if err := store.db.Model(&gameRow{ID: saved.SessionID.String()}).Update("starts_at", start).Error; err != nil {
		t.Fatal(err)
	}
	at := start.Add(time.Minute)
	first, _ := game.ParseCoordinates(52.52, 13.405)
	second, _ := game.ParseCoordinates(52.53, 13.415)
	line, _ := game.NewSketchLine([]game.Coordinates{first, second})
	update := func(change func(l *game.Lobby) error) error {
		_, err := store.UpdateLobbyByToken(ctx, hunterToken.Hash(), at, change)
		return err
	}
	for range 2 {
		if err := update(func(l *game.Lobby) error { return l.AddSketch(hunter.ID, line, at) }); err != nil {
			t.Fatal(err)
		}
	}
	sketches := load(t, store, saved.JoinCodes.Hunter, at).Sketches
	if len(sketches) != 2 || sketches[0].ID == sketches[1].ID || sketches[0].Team != hunter.HunterTeam ||
		!sketches[0].CreatedAt.Equal(at) || !reflect.DeepEqual(sketches[0].Line, line) {
		t.Fatalf("sketches not stored: %+v", sketches)
	}

	if err := update(func(l *game.Lobby) error { return l.ClearSketches(hunter.ID, at) }); err != nil {
		t.Fatal(err)
	}
	if got := load(t, store, saved.JoinCodes.Hunter, at).Sketches; len(got) != 0 {
		t.Fatalf("cleared sketches still stored: %+v", got)
	}

	if err := update(func(l *game.Lobby) error { return l.AddSketch(hunter.ID, line, at) }); err != nil {
		t.Fatal(err)
	}
	// Retention counts from the start, like positions.
	if err := store.DeletePositionsBefore(ctx, start); err != nil || len(load(t, store, saved.JoinCodes.Hunter, at).Sketches) != 1 {
		t.Fatalf("sketch deleted before its retention ended: %v", err)
	}
	if err := store.DeletePositionsBefore(ctx, start.Add(time.Nanosecond)); err != nil || len(load(t, store, saved.JoinCodes.Hunter, at).Sketches) != 0 {
		t.Fatalf("expired sketch kept: %v", err)
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

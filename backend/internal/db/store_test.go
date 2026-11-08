package db

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"catch-me/backend/internal/game"
)

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
	return game.Game{SessionID: sessionID, CreatedAt: time.Date(2026, 11, 8, 12, 0, 0, 123456789, time.UTC), Playground: p}
}

func TestStorePersistsAfterReopening(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "games.sqlite")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	fixtures := []game.Game{fixture(t, 52.52, 100, strings.Repeat("a", 32)), fixture(t, 52.52, 5000, strings.Repeat("b", 32))}
	for _, saved := range fixtures {
		if err := store.Create(ctx, saved); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	for _, want := range fixtures {
		got, err := reopened.Get(ctx, want.SessionID)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("game did not persist: %+v, %v", got, err)
		}
	}
	if _, err := reopened.Get(ctx, game.NewSessionID()); !errors.Is(err, game.ErrNotFound) {
		t.Fatalf("not-found not translated: %v", err)
	}
	if err := reopened.Create(ctx, fixtures[0]); err == nil {
		t.Fatal("duplicate ID accepted")
	}
	if err := reopened.client.Game.UpdateOneID(fixtures[0].SessionID.String()).SetCellSizeMeters(1550).Exec(ctx); err == nil {
		t.Fatal("Ent schema validation bypassed")
	}
}

const legacySchema = `CREATE TABLE games (
	session_id TEXT PRIMARY KEY, created_at TEXT NOT NULL,
	latitude REAL NOT NULL, longitude REAL NOT NULL,
	north REAL NOT NULL, south REAL NOT NULL, east REAL NOT NULL, west REAL NOT NULL,
	rows INTEGER NOT NULL, columns INTEGER NOT NULL, cell_size_meters INTEGER NOT NULL
)`

func insertLegacy(t *testing.T, database *sql.DB, id, createdAt string, latitude float64, p game.Playground) {
	t.Helper()
	b := p.Bounds()
	_, err := database.Exec(`INSERT INTO games VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, createdAt, latitude, p.Center().Longitude(), b.North, b.South, b.East, b.West, p.Rows(), p.Columns(), p.CellSize().Meters())
	if err != nil {
		t.Fatal(err)
	}
}

// Create the exact table used before Ent, then prove startup and subsequent
// writes preserve its existing IDs, coordinates, sizes, and timestamp precision.
func TestLegacyDatabaseCompatibility(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy.sqlite")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Exec(legacySchema); err != nil {
		t.Fatal(err)
	}
	fixtures := []game.Game{fixture(t, 0, 100, strings.Repeat("a", 32)), fixture(t, 52.52, 1500, strings.Repeat("b", 32)), fixture(t, 52.52, 2000, strings.Repeat("c", 32))}
	for _, saved := range fixtures {
		insertLegacy(t, legacy, saved.SessionID.String(), "2026-11-08T12:00:00.123456789Z", saved.Playground.Center().Latitude(), saved.Playground)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		store, err := Open(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range fixtures {
			got, err := store.Get(ctx, want.SessionID)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("legacy data changed: %+v, %v", got, err)
			}
		}
		if attempt == 0 {
			added := fixture(t, 52.52, 5000, strings.Repeat("d", 32))
			if err := store.Create(ctx, added); err != nil {
				t.Fatal(err)
			}
			fixtures = append(fixtures, added)
		}
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

// Rows written outside the store are parsed on load; invalid data must be a
// server error, never a client validation error.
func TestInvalidStoredDataIsNotAValidationError(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "invalid.sqlite")
	database, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(legacySchema); err != nil {
		t.Fatal(err)
	}
	valid := fixture(t, 52.52, 2000, strings.Repeat("a", 32)).Playground
	insertLegacy(t, database, strings.Repeat("a", 32), "not a time", 52.52, valid)
	insertLegacy(t, database, strings.Repeat("b", 32), "2026-11-08T12:00:00Z", 85, valid)
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, id := range []string{strings.Repeat("a", 32), strings.Repeat("b", 32)} {
		sessionID, _ := game.ParseSessionID(id)
		_, err := store.Get(ctx, sessionID)
		var validation *game.ValidationError
		if err == nil || errors.As(err, &validation) {
			t.Fatalf("invalid stored game: %v", err)
		}
	}
}

func TestOpenRejectsMissingParent(t *testing.T) {
	if _, err := Open(context.Background(), filepath.Join(t.TempDir(), "missing", "games.sqlite")); err == nil {
		t.Fatal("database opened in missing directory")
	}
}

func TestJoinCodesAndLobbyPersist(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lobby.sqlite")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	saved := fixture(t, 52.52, 2000, strings.Repeat("a", 32))
	saved.JoinCodes = game.NewJoinCodes()
	if err := store.Create(ctx, saved); err != nil {
		t.Fatal(err)
	}
	clash := fixture(t, 52.52, 2000, strings.Repeat("b", 32))
	clash.JoinCodes = game.JoinCodes{Runner: saved.JoinCodes.Hunter, Hunter: game.NewJoinCode()}
	if err := store.Create(ctx, clash); !errors.Is(err, game.ErrJoinCodeTaken) {
		t.Fatalf("code collision across columns not detected: %v", err)
	}

	now := time.Date(2026, 11, 8, 12, 0, 0, 123456789, time.UTC)
	names := map[game.Team]string{game.Runners: "Ada", game.Hunters: "Bob"}
	tokens := map[game.Team]game.PlayerToken{}
	for i, team := range []game.Team{game.Runners, game.Hunters} {
		name, _ := game.ParsePlayerName(names[team])
		tokens[team] = game.NewPlayerToken()
		joinedAt := now.Add(time.Duration(i) * time.Second)
		_, err := store.UpdateLobbyByJoinCode(ctx, saved.JoinCodes.For(team), func(l *game.Lobby) error {
			player, err := l.Join(saved.JoinCodes.For(team), name, tokens[team], joinedAt)
			if err != nil {
				return err
			}
			return l.SetReady(player.ID, true, joinedAt)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	lobby, err := store.UpdateLobbyByToken(ctx, tokens[game.Hunters].Hash(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if lobby.Game.JoinCodes != saved.JoinCodes || !lobby.Game.StartsAt.Equal(now.Add(time.Second+game.CountdownDuration)) || len(lobby.Players) != 2 {
		t.Fatalf("lobby not persisted: %+v", lobby)
	}
	if lobby.Players[0].Name.String() != "Ada" || !lobby.Players[0].Ready || !lobby.Players[0].JoinedAt.Equal(now) {
		t.Fatalf("player not persisted: %+v", lobby.Players[0])
	}
	me, _ := lobby.PlayerByToken(tokens[game.Hunters].Hash())
	if _, err := store.UpdateLobbyByToken(ctx, me.TokenHash, func(l *game.Lobby) error { return l.Leave(me.ID, now) }); err != nil {
		t.Fatal(err)
	}
	lobby, err = store.UpdateLobbyByJoinCode(ctx, saved.JoinCodes.Runner, nil)
	if err != nil || len(lobby.Players) != 1 || !lobby.Game.StartsAt.IsZero() {
		t.Fatalf("leave not persisted: %+v %v", lobby, err)
	}
	if _, err := store.UpdateLobbyByToken(ctx, me.TokenHash, nil); !errors.Is(err, game.ErrNotFound) {
		t.Fatal("removed player's token still works")
	}
	failed := errors.New("rule failed")
	if _, err := store.UpdateLobbyByJoinCode(ctx, saved.JoinCodes.Runner, func(l *game.Lobby) error {
		l.Players = nil
		return failed
	}); !errors.Is(err, failed) {
		t.Fatal("change error lost")
	}
	if lobby, _ := store.UpdateLobbyByJoinCode(ctx, saved.JoinCodes.Runner, nil); len(lobby.Players) != 1 {
		t.Fatal("failed change was saved")
	}
}

func TestRoutesPersist(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "routes.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	saved := fixture(t, 52.52, 2000, strings.Repeat("a", 32))
	saved.JoinCodes = game.NewJoinCodes()
	if err := store.Create(ctx, saved); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	token := game.NewPlayerToken()
	name, _ := game.ParsePlayerName("Ada")
	var joined game.Player
	if _, err := store.UpdateLobbyByJoinCode(ctx, saved.JoinCodes.Runner, func(l *game.Lobby) error {
		joined, err = l.Join(saved.JoinCodes.Runner, name, token, now)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	load := func() game.Player {
		t.Helper()
		lobby, err := store.UpdateLobbyByToken(ctx, token.Hash(), nil)
		if err != nil {
			t.Fatal(err)
		}
		return lobby.Players[0]
	}
	if loaded := load(); !loaded.SameState(joined) || loaded.Route == nil {
		t.Fatalf("route not persisted: %+v", loaded.Route)
	}
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), func(l *game.Lobby) error { return l.RegenerateRoute(joined.ID, now) }); err != nil {
		t.Fatal(err)
	}
	if loaded := load(); loaded.Route.RegenerationsLeft != game.MaxRouteRegenerations-1 || *loaded.Route == *joined.Route {
		t.Fatalf("regenerated route not persisted: %+v", loaded.Route)
	}

	// Runners who joined before routes existed load without a route.
	if err := store.client.Player.UpdateOneID(joined.ID.String()).ClearRouteStart().ClearRouteEnd().ClearRouteRegenerationsLeft().Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if loaded := load(); loaded.Route != nil {
		t.Fatal("missing route not loaded as nil")
	}
	if err := store.client.Player.UpdateOneID(joined.ID.String()).SetRouteStart("A1").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), nil); err == nil {
		t.Fatal("incomplete stored route accepted")
	}
}

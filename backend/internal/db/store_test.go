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

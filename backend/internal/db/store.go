// Package db implements game persistence with Ent and a pure-Go SQLite driver.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"catch-me/backend/internal/db/ent"
	"catch-me/backend/internal/db/ent/migrate"
	"catch-me/backend/internal/game"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

type Store struct{ client *ent.Client }

func Open(ctx context.Context, path string) (*Store, error) {
	// DSN pragmas apply to every pooled connection, not only the first one.
	database, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	database.SetMaxOpenConns(1)
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, database)))
	// Ent's development migration is append-only by default: no dropped columns
	// or indexes. The schema maps to the legacy table; tests cover existing data.
	if err := client.Schema.Create(ctx, migrate.WithDropColumn(false), migrate.WithDropIndex(false)); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("initialize game schema: %w", err)
	}
	return &Store{client: client}, nil
}

func (s *Store) Close() error { return s.client.Close() }

func (s *Store) Create(ctx context.Context, saved game.Game) error {
	p := saved.Playground
	center, bounds := p.Center(), p.Bounds()
	_, err := s.client.Game.Create().SetID(saved.SessionID.String()).SetCreatedAt(saved.CreatedAt.Format(time.RFC3339Nano)).
		SetLatitude(center.Latitude()).SetLongitude(center.Longitude()).
		SetNorth(bounds.North).SetSouth(bounds.South).SetEast(bounds.East).SetWest(bounds.West).
		SetRowCount(p.Rows()).SetColumnCount(p.Columns()).SetCellSizeMeters(p.CellSize().Meters()).Save(ctx)
	if err != nil {
		return fmt.Errorf("insert game: %w", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id game.SessionID) (game.Game, error) {
	row, err := s.client.Game.Get(ctx, id.String())
	if ent.IsNotFound(err) {
		return game.Game{}, game.ErrNotFound
	}
	if err != nil {
		return game.Game{}, fmt.Errorf("select game: %w", err)
	}
	saved, err := parseRow(id, row)
	if err != nil {
		// %v, not %w: invalid stored data is a server error, not a client validation error.
		return game.Game{}, fmt.Errorf("parse stored game: %v", err)
	}
	return saved, nil
}

// parseRow rebuilds the playground from its center and cell size. The stored
// bounds and grid dimensions are derived values kept for table compatibility.
func parseRow(id game.SessionID, row *ent.Game) (game.Game, error) {
	createdAt, err := time.Parse(time.RFC3339Nano, row.CreatedAt)
	if err != nil {
		return game.Game{}, err
	}
	center, err := game.ParseCoordinates(row.Latitude, row.Longitude)
	if err != nil {
		return game.Game{}, err
	}
	cellSize, err := game.ParseCellSize(row.CellSizeMeters)
	if err != nil {
		return game.Game{}, err
	}
	playground, err := game.NewPlayground(center, cellSize)
	if err != nil {
		return game.Game{}, err
	}
	return game.Game{SessionID: id, CreatedAt: createdAt, Playground: playground}, nil
}

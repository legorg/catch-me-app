// Package db implements game persistence with Ent and a pure-Go SQLite driver.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"time"

	"catch-me/backend/internal/db/ent"
	entgame "catch-me/backend/internal/db/ent/game"
	"catch-me/backend/internal/db/ent/migrate"
	entplayer "catch-me/backend/internal/db/ent/player"
	"catch-me/backend/internal/db/ent/predicate"
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
	// One connection serializes all transactions, so read-modify-write lobby
	// updates cannot interleave.
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
	return s.inTx(ctx, func(tx *ent.Tx) error {
		runner, hunter := optionalCode(saved.JoinCodes.Runner), optionalCode(saved.JoinCodes.Hunter)
		var codes []string
		for _, code := range []*string{runner, hunter} {
			if code != nil {
				codes = append(codes, *code)
			}
		}
		if len(codes) > 0 {
			// Unique indexes cover each column; this also prevents a runner code
			// equal to another game's hunter code.
			taken, err := tx.Game.Query().Where(entgame.Or(entgame.RunnerCodeIn(codes...), entgame.HunterCodeIn(codes...))).Exist(ctx)
			if err != nil {
				return fmt.Errorf("check join codes: %w", err)
			}
			if taken {
				return game.ErrJoinCodeTaken
			}
		}
		p := saved.Playground
		center, bounds := p.Center(), p.Bounds()
		_, err := tx.Game.Create().SetID(saved.SessionID.String()).SetCreatedAt(saved.CreatedAt.Format(time.RFC3339Nano)).
			SetLatitude(center.Latitude()).SetLongitude(center.Longitude()).
			SetNorth(bounds.North).SetSouth(bounds.South).SetEast(bounds.East).SetWest(bounds.West).
			SetRowCount(p.Rows()).SetColumnCount(p.Columns()).SetCellSizeMeters(p.CellSize().Meters()).
			SetNillableRunnerCode(runner).SetNillableHunterCode(hunter).SetNillableStartsAt(optionalTime(saved.StartsAt)).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("insert game: %w", err)
		}
		return nil
	})
}

func (s *Store) Get(ctx context.Context, id game.SessionID) (game.Game, error) {
	row, err := s.client.Game.Get(ctx, id.String())
	if ent.IsNotFound(err) {
		return game.Game{}, game.ErrNotFound
	}
	if err != nil {
		return game.Game{}, fmt.Errorf("select game: %w", err)
	}
	saved, err := parseGame(row)
	if err != nil {
		// %v, not %w: invalid stored data is a server error, not a client validation error.
		return game.Game{}, fmt.Errorf("parse stored game: %v", err)
	}
	return saved, nil
}

// UpdateLobbyByJoinCode loads the game with this runner or hunter code and its
// players, applies change, and saves the result in one transaction. A nil
// change only loads.
func (s *Store) UpdateLobbyByJoinCode(ctx context.Context, code game.JoinCode, change func(*game.Lobby) error) (game.Lobby, error) {
	return s.updateLobby(ctx, entgame.Or(entgame.RunnerCode(code.String()), entgame.HunterCode(code.String())), change)
}

// UpdateLobbyByToken is UpdateLobbyByJoinCode for the game of the player with this token hash.
func (s *Store) UpdateLobbyByToken(ctx context.Context, hash game.TokenHash, change func(*game.Lobby) error) (game.Lobby, error) {
	return s.updateLobby(ctx, entgame.HasPlayersWith(entplayer.TokenHash(hash.String())), change)
}

func (s *Store) updateLobby(ctx context.Context, where predicate.Game, change func(*game.Lobby) error) (game.Lobby, error) {
	var lobby game.Lobby
	err := s.inTx(ctx, func(tx *ent.Tx) error {
		row, err := tx.Game.Query().Where(where).
			WithPlayers(func(q *ent.PlayerQuery) { q.Order(entplayer.ByJoinedAt(), entplayer.ByID()) }).
			Only(ctx)
		if ent.IsNotFound(err) {
			return game.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("select lobby: %w", err)
		}
		before, err := parseLobby(row)
		if err != nil {
			return fmt.Errorf("parse stored lobby: %v", err)
		}
		lobby = game.Lobby{Game: before.Game, Players: slices.Clone(before.Players)}
		if change == nil {
			return nil
		}
		if err := change(&lobby); err != nil {
			return err
		}
		return saveLobby(ctx, tx, before, lobby)
	})
	return lobby, err
}

func saveLobby(ctx context.Context, tx *ent.Tx, before, after game.Lobby) error {
	if !before.Game.StartsAt.Equal(after.Game.StartsAt) {
		update := tx.Game.UpdateOneID(after.Game.SessionID.String())
		if after.Game.StartsAt.IsZero() {
			update.ClearStartsAt()
		} else {
			update.SetStartsAt(after.Game.StartsAt)
		}
		if err := update.Exec(ctx); err != nil {
			return fmt.Errorf("update start time: %w", err)
		}
	}
	removed := map[game.PlayerID]game.Player{}
	for _, p := range before.Players {
		removed[p.ID] = p
	}
	for _, p := range after.Players {
		previous, existed := removed[p.ID]
		delete(removed, p.ID)
		var err error
		start, end, left := routeColumns(p.Route)
		switch {
		case !existed:
			err = tx.Player.Create().SetID(p.ID.String()).SetGameID(after.Game.SessionID.String()).
				SetTeam(entplayer.Team(p.Team)).SetName(p.Name.String()).SetTokenHash(p.TokenHash.String()).
				SetReady(p.Ready).SetJoinedAt(p.JoinedAt).
				SetNillableRouteStart(start).SetNillableRouteEnd(end).SetNillableRouteRegenerationsLeft(left).Exec(ctx)
		case !previous.SameState(p):
			// Only ready state and routes change after joining; routes are replaced, never removed.
			err = tx.Player.UpdateOneID(p.ID.String()).SetReady(p.Ready).
				SetNillableRouteStart(start).SetNillableRouteEnd(end).SetNillableRouteRegenerationsLeft(left).Exec(ctx)
		}
		if err != nil {
			return fmt.Errorf("save player: %w", err)
		}
	}
	for id := range removed {
		if err := tx.Player.DeleteOneID(id.String()).Exec(ctx); err != nil {
			return fmt.Errorf("delete player: %w", err)
		}
	}
	return nil
}

func (s *Store) inTx(ctx context.Context, run func(*ent.Tx) error) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := run(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func optionalCode(code game.JoinCode) *string {
	if code == (game.JoinCode{}) {
		return nil
	}
	value := code.String()
	return &value
}

func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func routeColumns(route *game.RunnerRoute) (start, end *string, regenerationsLeft *int) {
	if route == nil {
		return nil, nil, nil
	}
	from, to, left := route.Start.String(), route.End.String(), route.RegenerationsLeft
	return &from, &to, &left
}

func parseLobby(row *ent.Game) (game.Lobby, error) {
	saved, err := parseGame(row)
	if err != nil {
		return game.Lobby{}, err
	}
	lobby := game.Lobby{Game: saved}
	for _, p := range row.Edges.Players {
		player, err := parsePlayer(p)
		if err != nil {
			return game.Lobby{}, err
		}
		lobby.Players = append(lobby.Players, player)
	}
	return lobby, nil
}

// parseGame rebuilds the playground from its center and cell size. The stored
// bounds and grid dimensions are derived values kept for table compatibility.
func parseGame(row *ent.Game) (game.Game, error) {
	id, err := game.ParseSessionID(row.ID)
	if err != nil {
		return game.Game{}, err
	}
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
	saved := game.Game{SessionID: id, CreatedAt: createdAt, Playground: playground}
	for _, code := range []struct {
		stored *string
		target *game.JoinCode
	}{{row.RunnerCode, &saved.JoinCodes.Runner}, {row.HunterCode, &saved.JoinCodes.Hunter}} {
		if code.stored != nil {
			if *code.target, err = game.ParseJoinCode(*code.stored); err != nil {
				return game.Game{}, err
			}
		}
	}
	if row.StartsAt != nil {
		saved.StartsAt = row.StartsAt.UTC()
	}
	return saved, nil
}

func parsePlayer(row *ent.Player) (game.Player, error) {
	id, err := game.ParsePlayerID(row.ID)
	if err != nil {
		return game.Player{}, err
	}
	name, err := game.ParsePlayerName(row.Name)
	if err != nil {
		return game.Player{}, err
	}
	team, err := game.ParseTeam(string(row.Team))
	if err != nil {
		return game.Player{}, err
	}
	hash, err := game.ParseTokenHash(row.TokenHash)
	if err != nil {
		return game.Player{}, err
	}
	player := game.Player{ID: id, Name: name, Team: team, Ready: row.Ready, TokenHash: hash, JoinedAt: row.JoinedAt.UTC()}
	switch {
	case row.RouteStart != nil && row.RouteEnd != nil && row.RouteRegenerationsLeft != nil && team == game.Runners:
		route, err := game.ParseRunnerRoute(*row.RouteStart, *row.RouteEnd, *row.RouteRegenerationsLeft)
		if err != nil {
			return game.Player{}, err
		}
		player.Route = &route
	case row.RouteStart != nil || row.RouteEnd != nil || row.RouteRegenerationsLeft != nil:
		return game.Player{}, fmt.Errorf("player %s has an incomplete route or is not a runner", row.ID)
	}
	return player, nil
}

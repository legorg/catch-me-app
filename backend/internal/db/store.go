// Package db stores games with GORM and the pure-Go modernc.org/sqlite driver.
package db

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"catch-me/backend/internal/game"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
)

// Table rows. As in the domain, zero values mean "none": hunters have no
// route, and a zero RecordedAt or RevealedAt means no position or reveal.
type gameRow struct {
	ID                  string `gorm:"primaryKey"`
	CreatedAt           time.Time
	Latitude, Longitude float64
	CellSizeMeters      int
	RunnerCode          string `gorm:"uniqueIndex;not null"`
	HunterCode          string `gorm:"uniqueIndex;not null"`
	StartsAt            time.Time
	Players             []playerRow `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE"`
}

type playerRow struct {
	ID                     string `gorm:"primaryKey"`
	GameID                 string `gorm:"index;not null"`
	Team, Name             string
	TokenHash              string `gorm:"uniqueIndex;not null"`
	Ready                  bool
	JoinedAt               time.Time
	RouteStart, RouteEnd   string
	RouteRegenerationsLeft int
	Position               positionRow `gorm:"embedded;embeddedPrefix:position_"`
	Reveal                 positionRow `gorm:"embedded;embeddedPrefix:reveal_"`
	RevealedAt             time.Time
	History                []historyRow `gorm:"foreignKey:PlayerID;constraint:OnDelete:CASCADE"`
}

type positionRow struct {
	Latitude, Longitude, AccuracyMeters float64
	RecordedAt                          time.Time
}

// historyRow is one reported position, kept for a replay after the game.
type historyRow struct {
	ID       uint64      `gorm:"primaryKey"`
	PlayerID string      `gorm:"index;not null"`
	Position positionRow `gorm:"embedded"`
}

func (gameRow) TableName() string    { return "games" }
func (playerRow) TableName() string  { return "players" }
func (historyRow) TableName() string { return "position_history" }

type Store struct{ db *gorm.DB }

func Open(ctx context.Context, path string) (*Store, error) {
	// DSN parameters apply to every pooled connection. _time_format stores times
	// as sortable text; all times are UTC, so SQL can compare them.
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_time_format=sqlite"
	// Silent logger: SQL logs would contain player positions.
	db, err := gorm.Open(sqlite.New(sqlite.Config{DriverName: "sqlite", DSN: dsn}), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	database, err := db.DB()
	if err != nil {
		return nil, err
	}
	// One connection serializes all transactions, so read-modify-write lobby
	// updates cannot interleave.
	database.SetMaxOpenConns(1)
	// AutoMigrate creates missing tables, columns, and indexes; it never drops any.
	if err := db.WithContext(ctx).AutoMigrate(&gameRow{}, &playerRow{}, &historyRow{}); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	database, err := s.db.DB()
	if err != nil {
		return err
	}
	return database.Close()
}

func (s *Store) Create(ctx context.Context, g game.Game) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Also prevents a runner code equal to another game's hunter code.
		codes := []string{g.JoinCodes.Runner.String(), g.JoinCodes.Hunter.String()}
		var taken int64
		if err := tx.Model(&gameRow{}).Where("runner_code IN ? OR hunter_code IN ?", codes, codes).Count(&taken).Error; err != nil {
			return fmt.Errorf("check join codes: %w", err)
		}
		if taken > 0 {
			return game.ErrJoinCodeTaken
		}
		center := g.Playground.Center()
		row := gameRow{
			ID: g.SessionID.String(), CreatedAt: g.CreatedAt, Latitude: center.Latitude(), Longitude: center.Longitude(),
			CellSizeMeters: g.Playground.CellSize().Meters(), RunnerCode: codes[0], HunterCode: codes[1], StartsAt: g.StartsAt,
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("insert game: %w", err)
		}
		return nil
	})
}

// UpdateLobbyByJoinCode loads the game with this runner or hunter code and its
// players, applies change, and saves the result in one transaction. A nil
// change only loads.
func (s *Store) UpdateLobbyByJoinCode(ctx context.Context, code game.JoinCode, change func(*game.Lobby) error) (game.Lobby, error) {
	return s.updateLobby(ctx, change, "runner_code = ? OR hunter_code = ?", code.String(), code.String())
}

// UpdateLobbyByToken is UpdateLobbyByJoinCode for the game of the player with this token hash.
func (s *Store) UpdateLobbyByToken(ctx context.Context, hash game.TokenHash, change func(*game.Lobby) error) (game.Lobby, error) {
	return s.updateLobby(ctx, change, "id = (SELECT game_id FROM players WHERE token_hash = ?)", hash.String())
}

func (s *Store) updateLobby(ctx context.Context, change func(*game.Lobby) error, where string, args ...any) (game.Lobby, error) {
	var lobby game.Lobby
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row gameRow
		err := tx.Preload("Players", func(q *gorm.DB) *gorm.DB { return q.Order("joined_at, id") }).Where(where, args...).Take(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return game.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("select lobby: %w", err)
		}
		before, err := parseLobby(row)
		if err != nil {
			// %v, not %w: invalid stored data is a server error, not a client validation error.
			return fmt.Errorf("parse stored lobby: %v", err)
		}
		lobby = game.Lobby{Game: before.Game, Players: slices.Clone(before.Players)}
		if change == nil {
			return nil
		}
		if err := change(&lobby); err != nil {
			return err
		}
		return saveLobby(tx, before, lobby)
	})
	return lobby, err
}

// DeleteHistoryBefore deletes the position history of games that started (or,
// if never started, were created) before cutoff. A zero start sorts first.
func (s *Store) DeleteHistoryBefore(ctx context.Context, cutoff time.Time) error {
	err := s.db.WithContext(ctx).Where(`player_id IN (SELECT players.id FROM players JOIN games ON games.id = players.game_id
		WHERE max(games.created_at, games.starts_at) < ?)`, cutoff.UTC()).Delete(&historyRow{}).Error
	if err != nil {
		return fmt.Errorf("delete position history: %w", err)
	}
	return nil
}

func saveLobby(tx *gorm.DB, before, after game.Lobby) error {
	if !before.Game.StartsAt.Equal(after.Game.StartsAt) {
		if err := tx.Model(&gameRow{ID: after.Game.SessionID.String()}).Update("starts_at", after.Game.StartsAt).Error; err != nil {
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
		if existed && previous == p {
			continue
		}
		row := toPlayerRow(after.Game.SessionID, p)
		save := tx.Create
		if existed {
			save = tx.Save
		}
		if err := save(&row).Error; err != nil {
			return fmt.Errorf("save player: %w", err)
		}
		if p.Position != previous.Position && !p.Position.IsZero() {
			if err := tx.Create(&historyRow{PlayerID: row.ID, Position: row.Position}).Error; err != nil {
				return fmt.Errorf("save position history: %w", err)
			}
		}
	}
	for id := range removed {
		// Also deletes the player's position history (ON DELETE CASCADE).
		if err := tx.Delete(&playerRow{ID: id.String()}).Error; err != nil {
			return fmt.Errorf("delete player: %w", err)
		}
	}
	return nil
}

func toPlayerRow(gameID game.SessionID, p game.Player) playerRow {
	row := playerRow{
		ID: p.ID.String(), GameID: gameID.String(), Team: string(p.Team), Name: p.Name.String(),
		TokenHash: p.TokenHash.String(), Ready: p.Ready, JoinedAt: p.JoinedAt,
		Position: toPositionRow(p.Position), Reveal: toPositionRow(p.Reveal.Position), RevealedAt: p.Reveal.At,
	}
	if !p.Route.IsZero() {
		row.RouteStart, row.RouteEnd, row.RouteRegenerationsLeft = p.Route.Start.String(), p.Route.End.String(), p.Route.RegenerationsLeft
	}
	return row
}

func toPositionRow(p game.Position) positionRow {
	return positionRow{Latitude: p.Coordinates.Latitude(), Longitude: p.Coordinates.Longitude(), AccuracyMeters: p.AccuracyMeters, RecordedAt: p.RecordedAt}
}

// parseLobby rebuilds domain values through their parsers, so invalid stored
// data is rejected. The playground is derived from center and cell size.
func parseLobby(row gameRow) (game.Lobby, error) {
	id, err1 := game.ParseSessionID(row.ID)
	center, err2 := game.ParseCoordinates(row.Latitude, row.Longitude)
	cellSize, err3 := game.ParseCellSize(row.CellSizeMeters)
	runner, err4 := game.ParseJoinCode(row.RunnerCode)
	hunter, err5 := game.ParseJoinCode(row.HunterCode)
	if err := errors.Join(err1, err2, err3, err4, err5); err != nil {
		return game.Lobby{}, err
	}
	playground, err := game.NewPlayground(center, cellSize)
	if err != nil {
		return game.Lobby{}, err
	}
	lobby := game.Lobby{Game: game.Game{
		SessionID: id, CreatedAt: row.CreatedAt.UTC(), Playground: playground,
		JoinCodes: game.JoinCodes{Runner: runner, Hunter: hunter}, StartsAt: row.StartsAt.UTC(),
	}}
	for _, p := range row.Players {
		player, err := parsePlayer(p)
		if err != nil {
			return game.Lobby{}, err
		}
		lobby.Players = append(lobby.Players, player)
	}
	return lobby, nil
}

func parsePlayer(row playerRow) (game.Player, error) {
	id, err1 := game.ParsePlayerID(row.ID)
	name, err2 := game.ParsePlayerName(row.Name)
	team, err3 := game.ParseTeam(row.Team)
	hash, err4 := game.ParseTokenHash(row.TokenHash)
	position, err5 := parsePosition(row.Position)
	revealed, err6 := parsePosition(row.Reveal)
	var route game.RunnerRoute
	var err7 error
	if team == game.Runners {
		route, err7 = game.ParseRunnerRoute(row.RouteStart, row.RouteEnd, row.RouteRegenerationsLeft)
	}
	if err := errors.Join(err1, err2, err3, err4, err5, err6, err7); err != nil {
		return game.Player{}, err
	}
	return game.Player{
		ID: id, Name: name, Team: team, Ready: row.Ready, TokenHash: hash, JoinedAt: row.JoinedAt.UTC(),
		Route: route, Position: position, Reveal: game.Reveal{Position: revealed, At: row.RevealedAt.UTC()},
	}, nil
}

func parsePosition(row positionRow) (game.Position, error) {
	if row.RecordedAt.IsZero() {
		return game.Position{}, nil
	}
	coordinates, err := game.ParseCoordinates(row.Latitude, row.Longitude)
	if err != nil {
		return game.Position{}, err
	}
	return game.NewPosition(coordinates, row.AccuracyMeters, row.RecordedAt.UTC())
}

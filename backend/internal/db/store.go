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

// Table rows. A zero StartsAt means not scheduled; hunters have an empty route.
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
	Team, Name, HunterTeam string
	TokenHash              string `gorm:"uniqueIndex;not null"`
	Ready                  bool
	JoinedAt               time.Time
	RouteStart, RouteEnd   string
	RouteRegenerationsLeft int
	Positions              []positionRow `gorm:"foreignKey:PlayerID;constraint:OnDelete:CASCADE"`
}

// positionRow is one reported position. A player's newest row is their current
// position; all rows together are the history for a replay after the game.
type positionRow struct {
	ID                                  uint64 `gorm:"primaryKey"`
	PlayerID                            string `gorm:"index;not null"`
	Latitude, Longitude, AccuracyMeters float64
	RecordedAt                          time.Time
}

func (gameRow) TableName() string     { return "games" }
func (playerRow) TableName() string   { return "players" }
func (positionRow) TableName() string { return "positions" }

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
	if err := db.WithContext(ctx).AutoMigrate(&gameRow{}, &playerRow{}, &positionRow{}); err != nil {
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
// players as of now, applies change, and saves the result in one transaction.
// A nil change only loads.
func (s *Store) UpdateLobbyByJoinCode(ctx context.Context, code game.JoinCode, now time.Time, change func(*game.Lobby) error) (game.Lobby, error) {
	return s.updateLobby(ctx, now, change, "runner_code = ? OR hunter_code = ?", code.String(), code.String())
}

// UpdateLobbyByToken is UpdateLobbyByJoinCode for the game of the player with this token hash.
func (s *Store) UpdateLobbyByToken(ctx context.Context, hash game.TokenHash, now time.Time, change func(*game.Lobby) error) (game.Lobby, error) {
	return s.updateLobby(ctx, now, change, "id = (SELECT game_id FROM players WHERE token_hash = ?)", hash.String())
}

func (s *Store) updateLobby(ctx context.Context, now time.Time, change func(*game.Lobby) error, where string, args ...any) (game.Lobby, error) {
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
		before, err := loadLobby(tx, row, now)
		if err != nil {
			return err
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

// DeletePositionsBefore deletes the positions of games that started (or, if
// never started, were created) before cutoff. A zero start sorts first.
func (s *Store) DeletePositionsBefore(ctx context.Context, cutoff time.Time) error {
	err := s.db.WithContext(ctx).Where(`player_id IN (SELECT players.id FROM players JOIN games ON games.id = players.game_id
		WHERE max(games.created_at, games.starts_at) < ?)`, cutoff.UTC()).Delete(&positionRow{}).Error
	if err != nil {
		return fmt.Errorf("delete positions: %w", err)
	}
	return nil
}

// saveLobby writes changed games and players. A changed position is a new row;
// positions are never updated.
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
		if !existed || withoutPositions(previous) != withoutPositions(p) {
			row := toPlayerRow(after.Game.SessionID, p)
			save := tx.Create
			if existed {
				save = tx.Save
			}
			if err := save(&row).Error; err != nil {
				return fmt.Errorf("save player: %w", err)
			}
		}
		if p.Position != previous.Position && !p.Position.IsZero() {
			position := positionRow{
				PlayerID: p.ID.String(), Latitude: p.Position.Coordinates.Latitude(), Longitude: p.Position.Coordinates.Longitude(),
				AccuracyMeters: p.Position.AccuracyMeters, RecordedAt: p.Position.RecordedAt,
			}
			if err := tx.Create(&position).Error; err != nil {
				return fmt.Errorf("save position: %w", err)
			}
		}
	}
	for id := range removed {
		// Also deletes the player's positions (ON DELETE CASCADE).
		if err := tx.Delete(&playerRow{ID: id.String()}).Error; err != nil {
			return fmt.Errorf("delete player: %w", err)
		}
	}
	return nil
}

// withoutPositions keeps the fields stored in the players table.
func withoutPositions(p game.Player) game.Player {
	p.Position, p.Reveal = game.Position{}, game.Reveal{}
	return p
}

func toPlayerRow(gameID game.SessionID, p game.Player) playerRow {
	row := playerRow{
		ID: p.ID.String(), GameID: gameID.String(), Team: string(p.Team), Name: p.Name.String(),
		TokenHash: p.TokenHash.String(), Ready: p.Ready, JoinedAt: p.JoinedAt, HunterTeam: p.HunterTeam.String(),
	}
	if !p.Route.IsZero() {
		row.RouteStart, row.RouteEnd, row.RouteRegenerationsLeft = p.Route.Start.String(), p.Route.End.String(), p.Route.RegenerationsLeft
	}
	return row
}

// loadLobby rebuilds domain values through their parsers, so invalid stored
// data is a server error. It adds each player's latest position and, for
// runners, their reveal: the last position before the latest reveal time.
func loadLobby(tx *gorm.DB, row gameRow, now time.Time) (game.Lobby, error) {
	lobby, err := parseLobby(row)
	if err != nil {
		// %v, not %w: invalid stored data is a server error, not a client validation error.
		return game.Lobby{}, fmt.Errorf("parse stored lobby: %v", err)
	}
	latest, err := lastPositions(tx, row.Players, time.Time{})
	if err != nil {
		return game.Lobby{}, err
	}
	var revealed map[string]game.Position
	due := lobby.Game.LatestReveal(now)
	if !due.IsZero() {
		if revealed, err = lastPositions(tx, row.Players, due); err != nil {
			return game.Lobby{}, err
		}
	}
	for i := range lobby.Players {
		p := &lobby.Players[i]
		p.Position = latest[p.ID.String()]
		if position, ok := revealed[p.ID.String()]; ok && p.Team == game.Runners {
			p.Reveal = game.Reveal{Position: position, At: due}
		}
	}
	return lobby, nil
}

// lastPositions returns each player's newest position, only counting positions
// recorded before the given time unless it is zero.
func lastPositions(tx *gorm.DB, players []playerRow, before time.Time) (map[string]game.Position, error) {
	ids := make([]string, len(players))
	for i, p := range players {
		ids[i] = p.ID
	}
	newest := tx.Model(&positionRow{}).Select("max(id)").Where("player_id IN ?", ids).Group("player_id")
	if !before.IsZero() {
		newest = newest.Where("recorded_at < ?", before)
	}
	var rows []positionRow
	if err := tx.Where("id IN (?)", newest).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("select positions: %w", err)
	}
	positions := map[string]game.Position{}
	for _, row := range rows {
		position, err := parsePosition(row)
		if err != nil {
			return nil, fmt.Errorf("parse stored position: %v", err)
		}
		positions[row.PlayerID] = position
	}
	return positions, nil
}

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
	var route game.RunnerRoute
	var hunterTeam game.HunterTeamName
	var err5 error
	if team == game.Runners {
		route, err5 = game.ParseRunnerRoute(row.RouteStart, row.RouteEnd, row.RouteRegenerationsLeft)
	} else if row.HunterTeam != "" {
		hunterTeam, err5 = game.ParseHunterTeamName(row.HunterTeam)
	}
	if err := errors.Join(err1, err2, err3, err4, err5); err != nil {
		return game.Player{}, err
	}
	return game.Player{
		ID: id, Name: name, Team: team, Ready: row.Ready, TokenHash: hash, JoinedAt: row.JoinedAt.UTC(), HunterTeam: hunterTeam, Route: route,
	}, nil
}

func parsePosition(row positionRow) (game.Position, error) {
	coordinates, err := game.ParseCoordinates(row.Latitude, row.Longitude)
	if err != nil {
		return game.Position{}, err
	}
	return game.NewPosition(coordinates, row.AccuracyMeters, row.RecordedAt.UTC())
}

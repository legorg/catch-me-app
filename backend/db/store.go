// Package db stores games with GORM and the pure-Go modernc.org/sqlite driver.
package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"catch-me/backend/game"
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
	StartTitle          string
	StartLat, StartLon  float64
	StopTitle           string
	StopLat, StopLon    float64
	Players             []playerRow `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE"`
	Jokers              []jokerRow  `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE"`
	Sketches            []sketchRow `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE"`
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

// jokerRow is one used joker. What it showed is derived from the positions at UsedAt.
// A hunter team can use each kind once; Team is stored lowercase so spelling does not matter.
type jokerRow struct {
	ID     uint64    `gorm:"primaryKey"`
	GameID string    `gorm:"not null;uniqueIndex:joker_once"`
	Team   string    `gorm:"not null;uniqueIndex:joker_once"`
	Kind   string    `gorm:"not null;uniqueIndex:joker_once"`
	Name   string    `gorm:"not null"`
	UsedAt time.Time `gorm:"not null"`
}

// sketchRow is one line a hunter team drew. It belongs to the team, so it has no
// player and stays when its author leaves. Points is a JSON array of
// [latitude, longitude] pairs.
type sketchRow struct {
	ID        string    `gorm:"primaryKey"`
	GameID    string    `gorm:"index;not null"`
	Team      string    `gorm:"not null"`
	Points    string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (gameRow) TableName() string     { return "games" }
func (jokerRow) TableName() string    { return "jokers" }
func (playerRow) TableName() string   { return "players" }
func (positionRow) TableName() string { return "positions" }
func (sketchRow) TableName() string   { return "sketches" }

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
	if err := db.WithContext(ctx).AutoMigrate(&gameRow{}, &playerRow{}, &positionRow{}, &jokerRow{}, &sketchRow{}); err != nil {
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
			StartTitle: g.Route.Start.Title, StartLat: g.Route.Start.Coordinates.Latitude(), StartLon: g.Route.Start.Coordinates.Longitude(),
			StopTitle: g.Route.Stop.Title, StopLat: g.Route.Stop.Coordinates.Latitude(), StopLon: g.Route.Stop.Coordinates.Longitude(),
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
		lobby = game.Lobby{Game: before.Game, Players: slices.Clone(before.Players), Reveals: before.Reveals, Jokers: slices.Clone(before.Jokers), Sketches: slices.Clone(before.Sketches)}
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

// DeletePositionsBefore deletes the positions and sketches of games that
// started (or, if never started, were created) before cutoff. A zero start sorts first.
func (s *Store) DeletePositionsBefore(ctx context.Context, cutoff time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where(`player_id IN (SELECT players.id FROM players JOIN games ON games.id = players.game_id
			WHERE max(games.created_at, games.starts_at) < ?)`, cutoff.UTC()).Delete(&positionRow{}).Error
		if err != nil {
			return fmt.Errorf("delete positions: %w", err)
		}
		err = tx.Where(`game_id IN (SELECT id FROM games WHERE max(created_at, starts_at) < ?)`, cutoff.UTC()).Delete(&sketchRow{}).Error
		if err != nil {
			return fmt.Errorf("delete sketches: %w", err)
		}
		return nil
	})
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
	// Jokers are only ever added.
	for _, use := range after.Jokers[min(len(before.Jokers), len(after.Jokers)):] {
		row := jokerRow{
			GameID: after.Game.SessionID.String(), Team: strings.ToLower(use.Team.String()), Kind: string(use.Kind),
			Name: use.Team.String(), UsedAt: use.UsedAt,
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("save joker: %w", err)
		}
	}
	if err := saveSketches(tx, after.Game.SessionID, before.Sketches, after.Sketches); err != nil {
		return err
	}
	for id := range removed {
		// Also deletes the player's positions (ON DELETE CASCADE).
		if err := tx.Delete(&playerRow{ID: id.String()}).Error; err != nil {
			return fmt.Errorf("delete player: %w", err)
		}
	}
	return nil
}

// saveSketches inserts added sketches and deletes removed ones; sketches are never changed.
func saveSketches(tx *gorm.DB, gameID game.SessionID, before, after []game.Sketch) error {
	removed := map[game.SketchID]bool{}
	for _, s := range before {
		removed[s.ID] = true
	}
	for _, s := range after {
		if removed[s.ID] {
			delete(removed, s.ID)
			continue
		}
		points := make([][2]float64, 0, len(s.Line.Points()))
		for _, p := range s.Line.Points() {
			points = append(points, [2]float64{p.Latitude(), p.Longitude()})
		}
		encoded, err := json.Marshal(points)
		if err != nil {
			return fmt.Errorf("encode sketch: %w", err)
		}
		row := sketchRow{ID: s.ID.String(), GameID: gameID.String(), Team: s.Team.String(), Points: string(encoded), CreatedAt: s.CreatedAt}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("save sketch: %w", err)
		}
	}
	if len(removed) == 0 {
		return nil
	}
	ids := make([]string, 0, len(removed))
	for id := range removed {
		ids = append(ids, id.String())
	}
	if err := tx.Where("id IN ?", ids).Delete(&sketchRow{}).Error; err != nil {
		return fmt.Errorf("delete sketches: %w", err)
	}
	return nil
}

// withoutPositions keeps the fields stored in the players table.
func withoutPositions(p game.Player) game.Player {
	p.Position = game.Position{}
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
// runners, their reveals: at each reveal time the last position before it.
func loadLobby(tx *gorm.DB, row gameRow, now time.Time) (game.Lobby, error) {
	lobby, err := parseLobby(row)
	if err != nil {
		// %v, not %w: invalid stored data is a server error, not a client validation error.
		return game.Lobby{}, fmt.Errorf("parse stored lobby: %v", err)
	}
	latest, err := lastPositions(tx, row.Players)
	if err != nil {
		return game.Lobby{}, err
	}
	times := lobby.Game.RevealTimes(now)
	var runnerIDs []string
	for _, p := range lobby.Players {
		if p.Team == game.Runners {
			runnerIDs = append(runnerIDs, p.ID.String())
		}
	}
	var jokerRows []jokerRow
	if err := tx.Where("game_id = ?", row.ID).Order("id").Find(&jokerRows).Error; err != nil {
		return game.Lobby{}, fmt.Errorf("select jokers: %w", err)
	}
	// One query for the positions both reveals and jokers are read from.
	until := time.Time{}
	if len(times) > 0 {
		until = times[len(times)-1]
	}
	for _, j := range jokerRows {
		if j.UsedAt.After(until) {
			until = j.UsedAt
		}
	}
	history, err := runnerHistory(tx, runnerIDs, until)
	if err != nil {
		return game.Lobby{}, err
	}
	reveals, err := runnerReveals(history, times)
	if err != nil {
		return game.Lobby{}, err
	}
	for _, j := range jokerRows {
		use, err := parseJoker(j, history)
		if err != nil {
			return game.Lobby{}, err
		}
		lobby.Jokers = append(lobby.Jokers, use)
	}
	var sketchRows []sketchRow
	if err := tx.Where("game_id = ?", row.ID).Order("created_at, id").Find(&sketchRows).Error; err != nil {
		return game.Lobby{}, fmt.Errorf("select sketches: %w", err)
	}
	for _, s := range sketchRows {
		sketch, err := parseSketch(s)
		if err != nil {
			return game.Lobby{}, fmt.Errorf("parse stored sketch: %v", err)
		}
		lobby.Sketches = append(lobby.Sketches, sketch)
	}
	lobby.Reveals = map[game.PlayerID][]game.Reveal{}
	for i := range lobby.Players {
		p := &lobby.Players[i]
		p.Position = latest[p.ID.String()]
		if all := reveals[p.ID.String()]; len(all) > 0 {
			lobby.Reveals[p.ID] = all
		}
	}
	return lobby, nil
}

// runnerHistory returns each runner's positions recorded up to `until`, oldest first.
func runnerHistory(tx *gorm.DB, runnerIDs []string, until time.Time) (map[string][]positionRow, error) {
	byPlayer := map[string][]positionRow{}
	if len(runnerIDs) == 0 || until.IsZero() {
		return byPlayer, nil
	}
	var rows []positionRow
	err := tx.Where("player_id IN ? AND recorded_at <= ?", runnerIDs, until).Order("player_id, recorded_at, id").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("select positions: %w", err)
	}
	for _, row := range rows {
		byPlayer[row.PlayerID] = append(byPlayer[row.PlayerID], row)
	}
	return byPlayer, nil
}

// runnerReveals returns, per runner, their reveal at every given time (oldest
// first): the last position recorded before it. Times before the runner's first
// position have no reveal.
func runnerReveals(history map[string][]positionRow, times []time.Time) (map[string][]game.Reveal, error) {
	reveals := map[string][]game.Reveal{}
	for id, positions := range history {
		next := 0 // positions[:next] were recorded before the current reveal time
		for _, at := range times {
			for next < len(positions) && positions[next].RecordedAt.Before(at) {
				next++
			}
			if next == 0 {
				continue
			}
			position, err := parsePosition(positions[next-1])
			if err != nil {
				return nil, fmt.Errorf("parse stored position: %v", err)
			}
			reveals[id] = append(reveals[id], game.Reveal{Position: position, At: at})
		}
	}
	return reveals, nil
}

// parseJoker adds what a joker showed: each runner's last position at or before the use.
func parseJoker(row jokerRow, history map[string][]positionRow) (game.JokerUse, error) {
	kind, err := game.ParseJokerKind(row.Kind)
	if err != nil {
		return game.JokerUse{}, fmt.Errorf("parse stored joker: %v", err)
	}
	team, err := game.ParseHunterTeamName(row.Name)
	if err != nil {
		return game.JokerUse{}, fmt.Errorf("parse stored joker: %v", err)
	}
	use := game.JokerUse{Kind: kind, Team: team, UsedAt: row.UsedAt.UTC()}
	if kind != game.JokerShowRunners {
		return use, nil
	}
	for id, positions := range history {
		last := -1
		for i, p := range positions {
			if p.RecordedAt.After(row.UsedAt) {
				break
			}
			last = i
		}
		if last < 0 {
			continue
		}
		playerID, err := game.ParsePlayerID(id)
		if err != nil {
			return game.JokerUse{}, fmt.Errorf("parse stored joker: %v", err)
		}
		position, err := parsePosition(positions[last])
		if err != nil {
			return game.JokerUse{}, fmt.Errorf("parse stored position: %v", err)
		}
		use.Positions = append(use.Positions, game.JokerPosition{PlayerID: playerID, Position: position})
	}
	// Map order is random; keep the response stable.
	slices.SortFunc(use.Positions, func(a, b game.JokerPosition) int { return strings.Compare(a.PlayerID.String(), b.PlayerID.String()) })
	return use, nil
}

// lastPositions returns each player's newest position.
func lastPositions(tx *gorm.DB, players []playerRow) (map[string]game.Position, error) {
	ids := make([]string, len(players))
	for i, p := range players {
		ids[i] = p.ID
	}
	newest := tx.Model(&positionRow{}).Select("max(id)").Where("player_id IN ?", ids).Group("player_id")
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
	start, err1 := game.ParseCoordinates(row.StartLat, row.StartLon)
	stop, err2 := game.ParseCoordinates(row.StopLat, row.StopLon)
	if err := errors.Join(err1, err2); err != nil {
		return game.Lobby{}, err
	}
	lobby.Game.Route = game.Route{
		Start: game.Landmark{Title: row.StartTitle, Coordinates: start},
		Stop:  game.Landmark{Title: row.StopTitle, Coordinates: stop},
	}
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

func parseSketch(row sketchRow) (game.Sketch, error) {
	id, err1 := game.ParseSketchID(row.ID)
	team, err2 := game.ParseHunterTeamName(row.Team)
	var pairs [][2]float64
	err3 := json.Unmarshal([]byte(row.Points), &pairs)
	if err := errors.Join(err1, err2, err3); err != nil {
		return game.Sketch{}, err
	}
	points := make([]game.Coordinates, 0, len(pairs))
	for _, pair := range pairs {
		point, err := game.ParseCoordinates(pair[0], pair[1])
		if err != nil {
			return game.Sketch{}, err
		}
		points = append(points, point)
	}
	line, err := game.NewSketchLine(points)
	if err != nil {
		return game.Sketch{}, err
	}
	return game.Sketch{ID: id, Team: team, Line: line, CreatedAt: row.CreatedAt.UTC()}, nil
}

func parsePosition(row positionRow) (game.Position, error) {
	coordinates, err := game.ParseCoordinates(row.Latitude, row.Longitude)
	if err != nil {
		return game.Position{}, err
	}
	return game.NewPosition(coordinates, row.AccuracyMeters, row.RecordedAt.UTC())
}

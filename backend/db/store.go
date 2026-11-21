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

// Table rows. A zero StartsAt means not scheduled.
type gameRow struct {
	ID                  string `gorm:"primaryKey"`
	CreatedAt           time.Time
	Latitude, Longitude float64
	CellSizeMeters      int
	// Older rows predate these settings and get the defaults.
	GridSize              int    `gorm:"not null;default:6"`
	RevealIntervalMinutes int    `gorm:"not null;default:30"`
	RunnerCode            string `gorm:"uniqueIndex;not null"`
	HunterCode            string `gorm:"uniqueIndex;not null"`
	SpectatorCode         string `gorm:"uniqueIndex;not null"`
	StartsAt              time.Time
	EndedAt               *time.Time
	StartTitle            string
	StartLat, StartLon    float64
	StopTitle             string
	StopLat, StopLon      float64
	// RouteRegenerations is how often the runners replaced the route.
	RouteRegenerations int            `gorm:"not null;default:0"`
	Players            []playerRow    `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE"`
	Jokers             []jokerRow     `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE"`
	Sketches           []sketchRow    `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE"`
	ZoneAlerts         []zoneAlertRow `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE"`

	JokerSettings jokerSettingsRow `gorm:"embedded;embeddedPrefix:joker_"`
}

// Pointers preserve an explicit allowance of 0 (or a disabled zone alert) when
// GORM inserts a row with a default. Older games receive the default for each
// newly added column in AutoMigrate.
type jokerSettingsRow struct {
	ShowRunnersAllowance  *int  `gorm:"not null;default:1"`
	QueryZoneAllowance    *int  `gorm:"not null;default:1"`
	CompassAllowance      *int  `gorm:"not null;default:1"`
	TrapAllowance         *int  `gorm:"not null;default:1"`
	ZoneOnlyAllowance     *int  `gorm:"not null;default:1"`
	FakePositionAllowance *int  `gorm:"not null;default:1"`
	ShowHuntersAllowance  *int  `gorm:"not null;default:1"`
	ZoneAlertEnabled      *bool `gorm:"not null;default:true"`
}

// The route_start, route_end, and route_regenerations_left columns of older
// databases held per-runner routes; they are no longer read or written.
type playerRow struct {
	ID                     string `gorm:"primaryKey"`
	GameID                 string `gorm:"index;not null"`
	Team, Name, HunterTeam string
	TokenHash              string `gorm:"uniqueIndex;not null"`
	Ready                  bool
	JoinedAt               time.Time
	Runner                 *runnerRow    `gorm:"foreignKey:PlayerID;constraint:OnDelete:CASCADE"`
	Positions              []positionRow `gorm:"foreignKey:PlayerID;constraint:OnDelete:CASCADE"`
}

// runnerRow is the one-to-one gameplay state of a runner, never a hunter.
type runnerRow struct {
	PlayerID string     `gorm:"primaryKey;not null"`
	Status   string     `gorm:"not null"`
	EndedAt  *time.Time `gorm:"check:runner_outcome,(status = 'active' AND ended_at IS NULL) OR (status IN ('caught','finished') AND ended_at IS NOT NULL)"`
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
// Allowances are enforced by the domain within the serialized lobby transaction.
// Team is stored lowercase so spelling does not matter.
// Shared runner jokers have an empty Team and Name. Personal fake position
// jokers use the runner's ID as Team (the personal ownership scope). All runner jokers
// record who used them in PlayerID; hunter jokers have an empty PlayerID.
// Cell is the queried cell (queryZone), empty otherwise. Latitude and
// Longitude are the shown point (fakePosition), the origin (compass), or the
// trap's point (trap), zero otherwise.
type jokerRow struct {
	ID                  uint64    `gorm:"primaryKey"`
	GameID              string    `gorm:"not null;index:joker_scope"`
	Team                string    `gorm:"not null;index:joker_scope"`
	Kind                string    `gorm:"not null;index:joker_scope"`
	Name                string    `gorm:"not null"`
	PlayerID            string    `gorm:"not null;default:''"`
	Cell                string    `gorm:"not null;default:''"`
	Latitude, Longitude float64   `gorm:"not null;default:0"`
	UsedAt              time.Time `gorm:"not null"`
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

// zoneAlertRow is one runner's zone alert about one hunter. The unique index
// backs up the domain rule of at most one alert per runner and hunter.
type zoneAlertRow struct {
	ID       uint64    `gorm:"primaryKey"`
	GameID   string    `gorm:"not null;uniqueIndex:zone_alert_once"`
	RunnerID string    `gorm:"not null;uniqueIndex:zone_alert_once"`
	HunterID string    `gorm:"not null;uniqueIndex:zone_alert_once"`
	Cell     string    `gorm:"not null"`
	RaisedAt time.Time `gorm:"not null"`
}

func (gameRow) TableName() string     { return "games" }
func (jokerRow) TableName() string    { return "jokers" }
func (playerRow) TableName() string   { return "players" }
func (runnerRow) TableName() string   { return "runners" }
func (positionRow) TableName() string { return "positions" }
func (sketchRow) TableName() string   { return "sketches" }

func (zoneAlertRow) TableName() string { return "zone_alerts" }

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
	if err := db.WithContext(ctx).AutoMigrate(&gameRow{}, &playerRow{}, &runnerRow{}, &positionRow{}, &jokerRow{}, &sketchRow{}, &zoneAlertRow{}); err != nil {
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
		// Prevent collisions across all invitation columns, not just within one affiliation.
		codes := []string{g.JoinCodes.Runner.String(), g.JoinCodes.Hunter.String(), g.JoinCodes.Spectator.String()}
		var taken int64
		if err := tx.Model(&gameRow{}).Where("runner_code IN ? OR hunter_code IN ? OR spectator_code IN ?", codes, codes, codes).Count(&taken).Error; err != nil {
			return fmt.Errorf("check join codes: %w", err)
		}
		if taken > 0 {
			return game.ErrJoinCodeTaken
		}
		center := g.Playground.Center()
		row := gameRow{
			ID: g.SessionID.String(), CreatedAt: g.CreatedAt, Latitude: center.Latitude(), Longitude: center.Longitude(),
			CellSizeMeters: g.Playground.CellSize().Meters(), GridSize: g.Playground.GridSize().Cells(), RevealIntervalMinutes: g.RevealInterval.Minutes(),
			RunnerCode: codes[0], HunterCode: codes[1], SpectatorCode: codes[2], StartsAt: g.StartsAt,
			StartTitle: g.Route.Start.Title, StartLat: g.Route.Start.Coordinates.Latitude(), StartLon: g.Route.Start.Coordinates.Longitude(),
			StopTitle: g.Route.Stop.Title, StopLat: g.Route.Stop.Coordinates.Latitude(), StopLon: g.Route.Stop.Coordinates.Longitude(),
			RouteRegenerations: g.RouteRegenerations,
			JokerSettings:      toJokerSettingsRow(g.JokerSettings),
		}
		if !g.EndedAt.IsZero() {
			row.EndedAt = &g.EndedAt
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("insert game: %w", err)
		}
		return nil
	})
}

// UpdateLobbyByJoinCode loads the game with this runner, hunter or spectator code and its
// players as of now, applies change, and saves the result in one transaction.
// A nil change only loads.
func (s *Store) UpdateLobbyByJoinCode(ctx context.Context, code game.JoinCode, now time.Time, change func(*game.Lobby) error) (game.Lobby, error) {
	return s.updateLobby(ctx, now, change, "runner_code = ? OR hunter_code = ? OR spectator_code = ?", code.String(), code.String(), code.String())
}

// UpdateLobbyByToken is UpdateLobbyByJoinCode for the game of the player with this token hash.
func (s *Store) UpdateLobbyByToken(ctx context.Context, hash game.TokenHash, now time.Time, change func(*game.Lobby) error) (game.Lobby, error) {
	return s.updateLobby(ctx, now, change, "id = (SELECT game_id FROM players WHERE token_hash = ?)", hash.String())
}

func (s *Store) updateLobby(ctx context.Context, now time.Time, change func(*game.Lobby) error, where string, args ...any) (game.Lobby, error) {
	var lobby game.Lobby
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row gameRow
		err := tx.Preload("Players", func(q *gorm.DB) *gorm.DB { return q.Order("joined_at, id") }).Preload("Players.Runner").Where(where, args...).Take(&row).Error
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
		lobby = game.Lobby{
			Game: before.Game, Players: slices.Clone(before.Players), Reveals: before.Reveals,
			Jokers: slices.Clone(before.Jokers), Sketches: slices.Clone(before.Sketches), ZoneAlerts: slices.Clone(before.ZoneAlerts),
		}
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

// DeleteGamesBefore deletes games that ended before cutoff, or, if they never
// ended, started (or, if never started, were created) before cutoff, with all
// their players, runner states, positions, jokers, sketches and zone alerts. A zero start
// sorts first. It also deletes orphaned rows whose game or player is gone, e.g.
// left over from older schemas without enforced foreign keys.
func (s *Store) DeleteGamesBefore(ctx context.Context, cutoff time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("coalesce(ended_at, max(created_at, starts_at)) < ?", cutoff.UTC()).Delete(&gameRow{}).Error
		if err != nil {
			return fmt.Errorf("delete games: %w", err)
		}
		// Children first, so a removed player's rows are orphaned in the same pass.
		orphans := []struct {
			row   any
			where string
		}{
			{&playerRow{}, "game_id NOT IN (SELECT id FROM games)"},
			{&jokerRow{}, "game_id NOT IN (SELECT id FROM games)"},
			{&sketchRow{}, "game_id NOT IN (SELECT id FROM games)"},
			{&zoneAlertRow{}, "game_id NOT IN (SELECT id FROM games)"},
			{&runnerRow{}, "player_id NOT IN (SELECT id FROM players)"},
			{&positionRow{}, "player_id NOT IN (SELECT id FROM players)"},
		}
		for _, o := range orphans {
			if err := tx.Where(o.where).Delete(o.row).Error; err != nil {
				return fmt.Errorf("delete orphaned rows: %w", err)
			}
		}
		return nil
	})
}

// saveLobby writes changed games and players. A changed position is a new row;
// positions are never updated.
func saveLobby(tx *gorm.DB, before, after game.Lobby) error {
	if !before.Game.EndedAt.Equal(after.Game.EndedAt) {
		if err := tx.Model(&gameRow{ID: after.Game.SessionID.String()}).Update("ended_at", after.Game.EndedAt).Error; err != nil {
			return fmt.Errorf("update end time: %w", err)
		}
	}
	if !before.Game.StartsAt.Equal(after.Game.StartsAt) {
		if err := tx.Model(&gameRow{ID: after.Game.SessionID.String()}).Update("starts_at", after.Game.StartsAt).Error; err != nil {
			return fmt.Errorf("update start time: %w", err)
		}
	}
	if before.Game.Route != after.Game.Route || before.Game.RouteRegenerations != after.Game.RouteRegenerations {
		r := after.Game.Route
		err := tx.Model(&gameRow{ID: after.Game.SessionID.String()}).Updates(map[string]any{
			"start_title": r.Start.Title, "start_lat": r.Start.Coordinates.Latitude(), "start_lon": r.Start.Coordinates.Longitude(),
			"stop_title": r.Stop.Title, "stop_lat": r.Stop.Coordinates.Latitude(), "stop_lon": r.Stop.Coordinates.Longitude(),
			"route_regenerations": after.Game.RouteRegenerations,
		}).Error
		if err != nil {
			return fmt.Errorf("update route: %w", err)
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
		if p.Team == game.Runners && (!existed || previous.Runner != p.Runner) {
			row := toRunnerRow(p)
			if err := tx.Save(&row).Error; err != nil {
				return fmt.Errorf("save runner: %w", err)
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
		row := jokerRow{GameID: after.Game.SessionID.String(), Kind: string(use.Kind()), UsedAt: use.UsedAt()}
		switch u := use.(type) {
		case game.ShowRunnersUse:
			row.Team, row.Name = strings.ToLower(u.Team.String()), u.Team.String()
		case game.QueryZoneUse:
			row.Team, row.Name, row.Cell = strings.ToLower(u.Team.String()), u.Team.String(), u.Cell.String()
		case game.CompassUse:
			row.Team, row.Name = strings.ToLower(u.Team.String()), u.Team.String()
			row.Latitude, row.Longitude = u.Origin.Latitude(), u.Origin.Longitude()
		case game.TrapUse:
			row.Team, row.Name = strings.ToLower(u.Team.String()), u.Team.String()
			row.Latitude, row.Longitude = u.Point.Latitude(), u.Point.Longitude()
		case game.ZoneOnlyUse:
			row.PlayerID = u.Runner.String()
		case game.FakePositionUse:
			row.PlayerID, row.Latitude, row.Longitude = u.Runner.String(), u.Point.Latitude(), u.Point.Longitude()
			row.Team = row.PlayerID
		case game.ShowHuntersUse:
			row.PlayerID = u.Runner.String()
		default:
			return fmt.Errorf("save joker: unknown joker %q", use.Kind())
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("save joker: %w", err)
		}
	}
	if err := saveSketches(tx, after.Game.SessionID, before.Sketches, after.Sketches); err != nil {
		return err
	}
	// Zone alerts are only ever added.
	for _, a := range after.ZoneAlerts[min(len(before.ZoneAlerts), len(after.ZoneAlerts)):] {
		row := zoneAlertRow{
			GameID: after.Game.SessionID.String(), RunnerID: a.Runner.String(), HunterID: a.Hunter.String(),
			Cell: a.Cell.String(), RaisedAt: a.At,
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("save zone alert: %w", err)
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
	p.Runner = game.RunnerState{}
	return p
}

func toPlayerRow(gameID game.SessionID, p game.Player) playerRow {
	row := playerRow{
		ID: p.ID.String(), GameID: gameID.String(), Team: string(p.Team), Name: p.Name.String(),
		TokenHash: p.TokenHash.String(), Ready: p.Ready, JoinedAt: p.JoinedAt, HunterTeam: p.HunterTeam.String(),
	}
	return row
}

func toRunnerRow(p game.Player) runnerRow {
	row := runnerRow{PlayerID: p.ID.String(), Status: string(p.Runner.Status())}
	if at := p.Runner.EndedAt(); !at.IsZero() {
		row.EndedAt = &at
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
	var runnerIDs, hunterIDs []string
	for _, p := range lobby.Players {
		if p.Team == game.Runners {
			runnerIDs = append(runnerIDs, p.ID.String())
		} else if p.Team == game.Hunters {
			hunterIDs = append(hunterIDs, p.ID.String())
		}
	}
	jokerRows, err := selectJokers(tx, row.ID)
	if err != nil {
		return game.Lobby{}, err
	}
	// One query for the runner positions both reveals and jokers are read from.
	// Hunter positions are only read for a used showHunters joker; traps query
	// their own reports.
	until, huntersUntil := time.Time{}, time.Time{}
	if len(times) > 0 {
		until = times[len(times)-1]
	}
	for _, j := range jokerRows {
		switch {
		case j.Kind == string(game.JokerTrap):
		case j.Kind == string(game.JokerShowHunters):
			if j.UsedAt.After(huntersUntil) {
				huntersUntil = j.UsedAt
			}
		case j.UsedAt.After(until):
			until = j.UsedAt
		}
	}
	history, err := positionHistory(tx, runnerIDs, until)
	if err != nil {
		return game.Lobby{}, err
	}
	hunterHistory, err := positionHistory(tx, hunterIDs, huntersUntil)
	if err != nil {
		return game.Lobby{}, err
	}
	reveals, err := runnerReveals(history, times)
	if err != nil {
		return game.Lobby{}, err
	}
	for _, j := range jokerRows {
		var use game.JokerUse
		if j.Kind == string(game.JokerTrap) {
			use, err = parseTrap(tx, j, runnerIDs)
		} else {
			use, err = parseJoker(j, history, hunterHistory, lobby)
		}
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
	if lobby.ZoneAlerts, err = selectZoneAlerts(tx, row.ID); err != nil {
		return game.Lobby{}, err
	}
	lobby.Reveals = map[game.PlayerID][]game.Reveal{}
	for i := range lobby.Players {
		p := &lobby.Players[i]
		p.Position = latest[p.ID.String()]
		if all := reveals[p.ID.String()]; p.Runner.Status() == game.RunnerActive && len(all) > 0 {
			lobby.Reveals[p.ID] = all
		}
	}
	return lobby, nil
}

// selectJokers returns the game's used jokers in order of use.
func selectJokers(tx *gorm.DB, gameID string) ([]jokerRow, error) {
	var rows []jokerRow
	if err := tx.Where("game_id = ?", gameID).Order("id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("select jokers: %w", err)
	}
	return rows, nil
}

// selectZoneAlerts returns the game's zone alerts, oldest first.
func selectZoneAlerts(tx *gorm.DB, gameID string) ([]game.ZoneAlert, error) {
	var rows []zoneAlertRow
	if err := tx.Where("game_id = ?", gameID).Order("id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("select zone alerts: %w", err)
	}
	alerts := make([]game.ZoneAlert, 0, len(rows))
	for _, row := range rows {
		runner, err1 := game.ParsePlayerID(row.RunnerID)
		hunter, err2 := game.ParsePlayerID(row.HunterID)
		cell, err3 := game.ParseCell(row.Cell)
		if err := errors.Join(err1, err2, err3); err != nil {
			return nil, fmt.Errorf("parse stored zone alert: %v", err)
		}
		alerts = append(alerts, game.ZoneAlert{Runner: runner, Hunter: hunter, Cell: cell, At: row.RaisedAt.UTC()})
	}
	return alerts, nil
}

// positionHistory returns each player's positions recorded up to `until`, oldest first.
func positionHistory(tx *gorm.DB, playerIDs []string, until time.Time) (map[string][]positionRow, error) {
	byPlayer := map[string][]positionRow{}
	if len(playerIDs) == 0 || until.IsZero() {
		return byPlayer, nil
	}
	var rows []positionRow
	err := tx.Where("player_id IN ? AND recorded_at <= ?", playerIDs, until).Order("player_id, recorded_at, id").Find(&rows).Error
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

// parseJoker rebuilds a used joker. A showing joker gets what it showed, from
// each player's last position at or before the use: the runners' positions
// (showRunners), whether a runner was in the cell (queryZone), the directions
// to the runners (compass), or the hunters' positions (showHunters). Reveal
// jokers only record who used them (and the point for fakePosition).
func parseJoker(row jokerRow, runners, hunters map[string][]positionRow, lobby game.Lobby) (game.JokerUse, error) {
	kind, err := game.ParseJokerKind(row.Kind)
	if err != nil {
		return nil, fmt.Errorf("parse stored joker: %v", err)
	}
	at := row.UsedAt.UTC()
	switch kind {
	case game.JokerZoneOnly, game.JokerFakePosition, game.JokerShowHunters:
		runner, err := game.ParsePlayerID(row.PlayerID)
		if err != nil {
			return nil, fmt.Errorf("parse stored joker: %v", err)
		}
		switch kind {
		case game.JokerZoneOnly:
			return game.ZoneOnlyUse{Runner: runner, At: at}, nil
		case game.JokerShowHunters:
			positions, err := positionsAt(hunters, row.UsedAt)
			if err != nil {
				return nil, err
			}
			return game.ShowHuntersUse{Runner: runner, At: at, Positions: positions}, nil
		}
		point, err := game.ParseCoordinates(row.Latitude, row.Longitude)
		if err != nil {
			return nil, fmt.Errorf("parse stored joker: %v", err)
		}
		return game.FakePositionUse{Runner: runner, At: at, Point: point}, nil
	}
	team, err := game.ParseHunterTeamName(row.Name)
	if err != nil {
		return nil, fmt.Errorf("parse stored joker: %v", err)
	}
	positions, err := positionsAt(runners, row.UsedAt)
	if err != nil {
		return nil, err
	}
	// Rebuild the result using only runners active at the time of this use.
	// Earlier snapshots stay historical; later jokers exclude caught/finished runners.
	positions = slices.DeleteFunc(positions, func(position game.JokerPosition) bool {
		player, ok := lobby.Player(position.PlayerID)
		return !ok || !player.Runner.ActiveAt(at)
	})
	switch kind {
	case game.JokerShowRunners:
		return game.ShowRunnersUse{Team: team, At: at, Positions: positions}, nil
	case game.JokerQueryZone:
		cell, err := game.ParseCell(row.Cell)
		if err != nil {
			return nil, fmt.Errorf("parse stored joker: %v", err)
		}
		return game.QueryZoneUse{Team: team, At: at, Cell: cell, Found: lobby.Game.Playground.AnyInCell(positions, cell)}, nil
	case game.JokerCompass:
		origin, err := game.ParseCoordinates(row.Latitude, row.Longitude)
		if err != nil {
			return nil, fmt.Errorf("parse stored joker: %v", err)
		}
		return game.CompassUse{Team: team, At: at, Origin: origin, Bearings: game.CompassBearings(origin, positions)}, nil
	}
	return nil, fmt.Errorf("parse stored joker: unknown joker %q", kind)
}

// parseTrap rebuilds a placed trap. Whether it triggered is derived from the
// runners' reports at or after its activation near its point; runners report
// only while active, so these are reports of active runners.
func parseTrap(tx *gorm.DB, row jokerRow, runnerIDs []string) (game.JokerUse, error) {
	team, err := game.ParseHunterTeamName(row.Name)
	if err != nil {
		return nil, fmt.Errorf("parse stored joker: %v", err)
	}
	point, err := game.ParseCoordinates(row.Latitude, row.Longitude)
	if err != nil {
		return nil, fmt.Errorf("parse stored joker: %v", err)
	}
	use := game.TrapUse{Team: team, At: row.UsedAt.UTC(), Point: point}
	if len(runnerIDs) == 0 {
		return use, nil
	}
	b := game.TrapSearchBounds(point)
	var rows []positionRow
	err = tx.Where("player_id IN ? AND recorded_at >= ? AND latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?",
		runnerIDs, use.ActiveAt(), b.South, b.North, b.West, b.East).Order("recorded_at, id").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("select trap positions: %w", err)
	}
	reports := make([]game.JokerPosition, 0, len(rows))
	for _, r := range rows {
		position, err := parsePosition(r)
		if err != nil {
			return nil, fmt.Errorf("parse stored position: %v", err)
		}
		reports = append(reports, game.JokerPosition{Position: position})
	}
	use.TriggeredAt = game.TrapTriggeredAt(point, use.ActiveAt(), reports)
	return use, nil
}

// positionsAt returns each player's last position at or before `at`, sorted by
// player ID; players without one are left out.
func positionsAt(history map[string][]positionRow, at time.Time) ([]game.JokerPosition, error) {
	var positions []game.JokerPosition
	for id, rows := range history {
		last := -1
		for i, p := range rows {
			if p.RecordedAt.After(at) {
				break
			}
			last = i
		}
		if last < 0 {
			continue
		}
		playerID, err := game.ParsePlayerID(id)
		if err != nil {
			return nil, fmt.Errorf("parse stored joker: %v", err)
		}
		position, err := parsePosition(rows[last])
		if err != nil {
			return nil, fmt.Errorf("parse stored position: %v", err)
		}
		positions = append(positions, game.JokerPosition{PlayerID: playerID, Position: position})
	}
	// Map order is random; keep the response stable.
	slices.SortFunc(positions, func(a, b game.JokerPosition) int { return strings.Compare(a.PlayerID.String(), b.PlayerID.String()) })
	return positions, nil
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
	gridSize, err6 := game.ParseGridSize(row.GridSize)
	revealInterval, err7 := game.ParseRevealInterval(row.RevealIntervalMinutes)
	spectator, err8 := game.ParseJoinCode(row.SpectatorCode)
	jokerSettings, err9 := parseJokerSettings(row.JokerSettings)
	if err := errors.Join(err1, err2, err3, err4, err5, err6, err7, err8, err9); err != nil {
		return game.Lobby{}, err
	}
	playground, err := game.NewPlayground(center, cellSize, gridSize)
	if err != nil {
		return game.Lobby{}, err
	}
	lobby := game.Lobby{Game: game.Game{
		SessionID: id, CreatedAt: row.CreatedAt.UTC(), Playground: playground, RevealInterval: revealInterval, JokerSettings: jokerSettings,
		JoinCodes: game.JoinCodes{Runner: runner, Hunter: hunter, Spectator: spectator}, StartsAt: row.StartsAt.UTC(),
	}}
	if row.EndedAt != nil {
		lobby.Game.EndedAt = row.EndedAt.UTC()
	}
	start, err1 := game.ParseCoordinates(row.StartLat, row.StartLon)
	stop, err2 := game.ParseCoordinates(row.StopLat, row.StopLon)
	regenerations, err3 := game.ParseRouteRegenerations(row.RouteRegenerations)
	if err := errors.Join(err1, err2, err3); err != nil {
		return game.Lobby{}, err
	}
	lobby.Game.Route = game.Route{
		Start: game.Landmark{Title: row.StartTitle, Coordinates: start},
		Stop:  game.Landmark{Title: row.StopTitle, Coordinates: stop},
	}
	lobby.Game.RouteRegenerations = regenerations
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
	var hunterTeam game.HunterTeamName
	var err5 error
	if team == game.Hunters && row.HunterTeam != "" {
		hunterTeam, err5 = game.ParseHunterTeamName(row.HunterTeam)
	}
	if err := errors.Join(err1, err2, err3, err4, err5); err != nil {
		return game.Player{}, err
	}
	player := game.Player{
		ID: id, Name: name, Team: team, Ready: row.Ready, TokenHash: hash, JoinedAt: row.JoinedAt.UTC(), HunterTeam: hunterTeam,
	}
	if team == game.Runners {
		if row.Runner == nil {
			return game.Player{}, fmt.Errorf("runner state missing")
		}
		var at time.Time
		if row.Runner.EndedAt != nil {
			at = *row.Runner.EndedAt
		}
		state, err := game.ParseRunnerState(row.Runner.Status, at)
		if err != nil {
			return game.Player{}, err
		}
		player.Runner = state
	} else if row.Runner != nil {
		return game.Player{}, fmt.Errorf("non-runners cannot have runner state")
	}
	return player, nil
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

package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"catch-me/backend/game"
	_ "modernc.org/sqlite"
)

type fixtureSummary struct {
	id         game.SessionID
	start, end time.Time
	positions  int
}

// generate only INSERTs, in one transaction. Schema creation/migration belongs
// to normal backend startup, not this development tool. The SQL deliberately
// stays here, outside the production store and server.
func generate(ctx context.Context, path string, now time.Time) (fixtureSummary, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fixtureSummary{}, fmt.Errorf("existing development database: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fixtureSummary{}, fmt.Errorf("database must be an existing regular file")
	}
	database, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_time_format=sqlite")
	if err != nil {
		return fixtureSummary{}, err
	}
	defer database.Close()
	database.SetMaxOpenConns(1)
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fixtureSummary{}, err
	}
	defer tx.Rollback()
	end := now.UTC().Truncate(time.Second).Add(-5 * time.Minute)
	fixture := fixtureSummary{id: game.NewSessionID(), start: end.Add(-4 * time.Hour), end: end}
	if err := insertGame(ctx, tx, fixture); err != nil {
		return fixtureSummary{}, fmt.Errorf("insert fixture game: %w", err)
	}
	var runners []string
	for index := range 20 {
		id := game.NewPlayerID().String()
		if index < 4 {
			runners = append(runners, id)
		}
		count, err := insertPlayer(ctx, tx, fixture, index, id)
		if err != nil {
			return fixtureSummary{}, fmt.Errorf("insert fixture player %d: %w", index+1, err)
		}
		fixture.positions += count
	}
	if err := insertExtras(ctx, tx, fixture, runners); err != nil {
		return fixtureSummary{}, fmt.Errorf("insert fixture jokers/sketches: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fixtureSummary{}, fmt.Errorf("commit fixture: %w", err)
	}
	return fixture, nil
}

func insertGame(ctx context.Context, tx *sql.Tx, fixture fixtureSummary) error {
	// Check across all three invitation columns, just as the normal store does.
	var codes game.JoinCodes
	for {
		codes = game.NewJoinCodes()
		a, b, c := codes.Runner.String(), codes.Hunter.String(), codes.Spectator.String()
		var taken int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM games WHERE runner_code IN (?, ?, ?) OR hunter_code IN (?, ?, ?) OR spectator_code IN (?, ?, ?)`, a, b, c, a, b, c, a, b, c).Scan(&taken); err != nil {
			return err
		}
		if taken == 0 {
			break
		}
	}
	startLat, startLon := coordinates(0, -2500)
	stopLat, stopLon := coordinates(0, 2500)
	_, err := tx.ExecContext(ctx, `INSERT INTO games
		(id, created_at, latitude, longitude, cell_size_meters, grid_size, reveal_interval_minutes,
		runner_code, hunter_code, spectator_code, starts_at, ended_at, start_title, start_lat, start_lon, stop_title, stop_lat, stop_lon, route_regenerations)
		VALUES (?, ?, ?, ?, 1000, 6, 30, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`,
		fixture.id.String(), fixture.start.Add(-30*time.Minute), centerLat, centerLon,
		codes.Runner.String(), codes.Hunter.String(), codes.Spectator.String(), fixture.start, fixture.end,
		"TEST Start (synthetisch)", startLat, startLon, "TEST Ziel (synthetisch)", stopLat, stopLon)
	return err
}

func insertPlayer(ctx context.Context, tx *sql.Tx, fixture fixtureSummary, index int, id string) (int, error) {
	team, name, hunterTeam := "runner", fmt.Sprintf("TEST Runner %d", index+1), ""
	path := runnerPaths[0]
	status := "caught"
	if index < 4 {
		path = runnerPaths[index]
		if index >= 2 {
			status = "finished"
		}
	} else {
		hunterTeam = hunterTeams[(index-4)/4]
		team, name = "hunter", fmt.Sprintf("TEST Hunter %02d", index-3)
		path = hunterPaths[(index-4)/4]
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO players (id, game_id, team, name, hunter_team, token_hash, ready, joined_at) VALUES (?, ?, ?, ?, ?, ?, 1, ?)`,
		id, fixture.id.String(), team, name, hunterTeam, game.NewPlayerToken().Hash().String(), fixture.start.Add(-25*time.Minute).Add(time.Duration(index)*time.Second))
	if err != nil {
		return 0, err
	}
	last := time.Duration(path[len(path)-1].minute) * time.Minute
	if index < 4 {
		if _, err := tx.ExecContext(ctx, `INSERT INTO runners (player_id, status, ended_at) VALUES (?, ?, ?)`, id, status, fixture.start.Add(last)); err != nil {
			return 0, err
		}
	}
	statement, err := tx.PrepareContext(ctx, `INSERT INTO positions (player_id, latitude, longitude, accuracy_meters, recorded_at) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer statement.Close()
	count := 0
	for elapsed := time.Duration(0); elapsed <= last; elapsed += 5 * time.Second {
		lat, lon := playerPosition(index, elapsed.Minutes())
		accuracy := 4 + float64((count+index)%13)*0.3
		if _, err := statement.ExecContext(ctx, id, lat, lon, accuracy, fixture.start.Add(elapsed)); err != nil {
			return 0, err
		}
		count++
	}
	return count, nil
}

func insertExtras(ctx context.Context, tx *sql.Tx, fixture fixtureSummary, runners []string) error {
	insertJoker := func(team, name, player string, kind game.JokerKind, minute int, cell string, lat, lon float64) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO jokers (game_id, team, name, player_id, kind, used_at, cell, latitude, longitude) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			fixture.id.String(), team, name, player, string(kind), fixture.start.Add(time.Duration(minute)*time.Minute), cell, lat, lon)
		return err
	}
	for team, name := range hunterTeams {
		for i, kind := range []game.JokerKind{game.JokerShowRunners, game.JokerQueryZone, game.JokerCompass} {
			minute := 35 + team*30 + i*20
			lat, lon := 0.0, 0.0
			cell := ""
			if kind == game.JokerQueryZone {
				cell = "C3"
			}
			if kind == game.JokerCompass {
				lat, lon = playerPosition(4+team*4, float64(minute))
			}
			if err := insertJoker(strings.ToLower(name), name, "", kind, minute, cell, lat, lon); err != nil {
				return err
			}
		}
	}
	for i, kind := range []game.JokerKind{game.JokerShowHunters, game.JokerZoneOnly} {
		if err := insertJoker("", "", runners[i], kind, 15+i*10, "", 0, 0); err != nil {
			return err
		}
	}
	for i, runner := range runners {
		lat, lon := coordinates(float64(i*700-1000), 400)
		// After the zone-only reveal at minute 30, before any runner retires.
		if err := insertJoker(runner, "", runner, game.JokerFakePosition, 40+i*5, "", lat, lon); err != nil {
			return err
		}
	}
	for team, name := range hunterTeams {
		var points [][2]float64
		for _, minute := range []float64{60, 70, 80, 90} {
			lat, lon := playerPosition(4+team*4, minute)
			points = append(points, [2]float64{lat, lon})
		}
		encoded, err := json.Marshal(points)
		if err != nil {
			return err
		}
		var id [16]byte
		_, _ = rand.Read(id[:]) // Never fails since Go 1.24.
		if _, err := tx.ExecContext(ctx, `INSERT INTO sketches (id, game_id, team, points, created_at) VALUES (?, ?, ?, ?, ?)`,
			fmt.Sprintf("%x", id), fixture.id.String(), strings.ToLower(name), string(encoded), fixture.start.Add(50*time.Minute)); err != nil {
			return err
		}
	}
	return nil
}

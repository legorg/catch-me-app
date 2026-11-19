package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"catch-me/backend/db"
	"catch-me/backend/game"
	"catch-me/backend/geo"
)

func TestGenerateCompletedGame(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "fixture.sqlite")
	store, err := db.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	now := time.Date(2026, 11, 19, 16, 0, 0, 123, time.FixedZone("test", 3600))
	fixture, err := generate(ctx, path, now)
	if err != nil {
		t.Fatal(err)
	}
	if fixture.positions != 54740 || !fixture.end.Equal(now.Truncate(time.Second).Add(-5*time.Minute)) || fixture.end.Sub(fixture.start) != 4*time.Hour {
		t.Fatalf("wrong fixture size/times: %+v", fixture)
	}
	result, err := store.Result(ctx, fixture.id)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Players) != 20 || len(result.Events) != 24 || !result.StartsAt.Equal(fixture.start) || !result.EndedAt.Equal(fixture.end) {
		t.Fatal("result lost players, events or game times")
	}
	if result.Playground.Rows() != 6 || result.Playground.CellSize().Meters() != 1000 {
		t.Fatal("wrong playground")
	}
	startCell, _ := result.Playground.CellAt(result.Route.Start.Coordinates)
	stopCell, _ := result.Playground.CellAt(result.Route.Stop.Coordinates)
	if !strings.HasSuffix(startCell.String(), "6") || !strings.HasSuffix(stopCell.String(), "1") {
		t.Fatal("route must connect the southern and northern edge rows")
	}
	teams := map[string]int{}
	points := 0
	for i, player := range result.Players {
		if !strings.HasPrefix(player.Name.String(), "TEST ") {
			t.Fatal("fixture players must be recognizable")
		}
		end := result.EndedAt
		if i < 4 {
			status := game.RunnerCaught
			if i >= 2 {
				status = game.RunnerFinished
			}
			if player.Team != game.Runners || player.Runner.Status() != status {
				t.Fatal("wrong runner outcome")
			}
			end = player.Runner.EndedAt()
		} else {
			if player.Team != game.Hunters {
				t.Fatal("expected hunter")
			}
			teams[player.HunterTeam.String()]++
		}
		track := player.Track
		if len(track) != int(end.Sub(result.StartsAt)/(5*time.Second))+1 || !track[0].RecordedAt.Equal(result.StartsAt) || !track[len(track)-1].RecordedAt.Equal(end) {
			t.Fatalf("incomplete track for %s", player.Name)
		}
		points += len(track)
		for j, point := range track {
			if _, inside := result.Playground.CellAt(point.Coordinates); !inside {
				t.Fatal("track left playground")
			}
			if point.AccuracyMeters < 4 || point.AccuracyMeters > 8 {
				t.Fatal("unexpected accuracy")
			}
			if j > 0 {
				previous := track[j-1]
				if point.RecordedAt.Sub(previous.RecordedAt) != 5*time.Second || distance(previous.Coordinates, point.Coordinates) > 20 {
					t.Fatal("track has a timestamp gap or implausible jump")
				}
			}
		}
		if i < 4 && distance(track[0].Coordinates, result.Route.Start.Coordinates) > 1 {
			t.Fatal("runner did not start at start landmark")
		}
		if i == 2 || i == 3 {
			if distance(track[len(track)-1].Coordinates, result.Route.Stop.Coordinates) > 1 {
				t.Fatal("finished runner did not reach destination")
			}
		}
	}
	if points != fixture.positions || len(teams) != 4 {
		t.Fatal("track points or hunter teams missing")
	}
	for _, count := range teams {
		if count != 4 {
			t.Fatal("each hunter team must have four members")
		}
	}
	jokers := map[game.JokerKind]int{}
	for i, event := range result.Events {
		if i > 0 && event.OccurredAt().Before(result.Events[i-1].OccurredAt()) {
			t.Fatal("timeline not chronological")
		}
		switch event := event.(type) {
		case game.HunterJokerEvent:
			jokers[event.Joker]++
		case game.RunnerJokerEvent:
			jokers[event.Joker]++
		}
	}
	wantJokers := map[game.JokerKind]int{game.JokerShowRunners: 4, game.JokerQueryZone: 4, game.JokerCompass: 4, game.JokerShowHunters: 1, game.JokerZoneOnly: 1, game.JokerFakePosition: 4}
	if !reflect.DeepEqual(jokers, wantJokers) {
		t.Fatalf("joker coverage: %v", jokers)
	}
	database, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	// The normal gameplay loader must also accept the persisted extras, not
	// only the result projection (which deliberately never reads sketches).
	var hashText string
	if err := database.QueryRow(`SELECT token_hash FROM players WHERE id = ?`, result.Players[0].ID.String()).Scan(&hashText); err != nil {
		t.Fatal(err)
	}
	hash, err := game.ParseTokenHash(hashText)
	if err != nil {
		t.Fatal(err)
	}
	lobby, err := store.UpdateLobbyByToken(ctx, hash, now, nil)
	if err != nil || len(lobby.Jokers) != 18 || len(lobby.Sketches) != 4 || lobby.Game.Status(now) != game.StatusEnded {
		t.Fatalf("gameplay loader rejected fixture: %v", err)
	}
	// Repeated runs add games, without replacing existing IDs, codes or data.
	second, err := generate(ctx, path, now.Add(time.Minute))
	if err != nil || second.id == fixture.id {
		t.Fatalf("second fixture: %v", err)
	}
	unchanged, err := store.Result(ctx, fixture.id)
	if err != nil || !reflect.DeepEqual(result, unchanged) {
		t.Fatal("generating another fixture modified the first game")
	}
	counts := tableCounts(t, database)
	if _, err := database.Exec(`CREATE TRIGGER reject_fixture BEFORE INSERT ON sketches BEGIN SELECT RAISE(ABORT, 'test rollback'); END`); err != nil {
		t.Fatal(err)
	}
	// This fails late, after all positions and jokers were inserted. Every
	// new row must roll back, across all six tables.
	if _, err := generate(ctx, path, now); err == nil {
		t.Fatal("expected insert failure")
	}
	if !reflect.DeepEqual(counts, tableCounts(t, database)) {
		t.Fatal("failed fixture left partial rows or removed existing data")
	}
	if err := store.DeleteGamesBefore(ctx, now.Add(-game.GameRetention)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(counts, tableCounts(t, database)) {
		t.Fatal("fresh fixtures expired early")
	}
	if err := store.DeleteGamesBefore(ctx, fixture.end.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Result(ctx, fixture.id); !errors.Is(err, game.ErrNotFound) {
		t.Fatal("expired fixture remains accessible")
	}
	if _, err := store.Result(ctx, second.id); err != nil {
		t.Fatalf("cleanup removed the newer fixture: %v", err)
	}
	wantCounts := map[string]int{"games": 1, "players": 20, "runners": 4, "positions": 54740, "jokers": 18, "sketches": 4}
	if got := tableCounts(t, database); !reflect.DeepEqual(got, wantCounts) {
		t.Fatalf("wrong persisted counts after cleanup: %v", got)
	}
	rows, err := database.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() || rows.Err() != nil {
		t.Fatal("fixture has broken foreign keys")
	}
}

func tableCounts(t *testing.T, database *sql.DB) map[string]int {
	t.Helper()
	counts := map[string]int{}
	for _, table := range []string{"games", "players", "runners", "positions", "jokers", "sketches"} {
		var count int
		if err := database.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		counts[table] = count
	}
	return counts
}

func distance(a, b game.Coordinates) float64 {
	return geo.DistanceMeters(a.Latitude(), a.Longitude(), b.Latitude(), b.Longitude())
}

func TestGenerateRequiresExistingSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.sqlite")
	if _, err := generate(context.Background(), path, time.Now()); err == nil {
		t.Fatal("expected missing database error")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("generator created a database")
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := generate(context.Background(), path, time.Now()); err == nil {
		t.Fatal("generator must not create missing tables")
	}
}

func TestPathsIncludePausesAndTeamSeparation(t *testing.T) {
	for _, path := range runnerPaths {
		// A path either stops along the way or ends in a catch before 4 h.
		hasPause := false
		for i := 1; i < len(path); i++ {
			if path[i].east == path[i-1].east && path[i].north == path[i-1].north {
				hasPause = true
			}
		}
		if !hasPause {
			t.Fatal("runner path has no pause")
		}
	}
	aLat, aLon := playerPosition(4, 90)
	bLat, bLon := playerPosition(5, 90)
	if gap := geo.DistanceMeters(aLat, aLon, bLat, bLon); gap < 10 || gap > 15 {
		t.Fatalf("hunter teammates should walk close but not overlap: %f m", gap)
	}
}

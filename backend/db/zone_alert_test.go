package db

import (
	"context"
	"slices"
	"testing"
	"time"

	"catch-me/backend/game"
)

func TestZoneAlertsPersistOncePerRunnerAndHunter(t *testing.T) {
	ctx := context.Background()
	store := open(t, ":memory:")
	saved := create(t, store)
	ada, adaToken := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	bob, bobToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now)
	start := now.Add(time.Minute)
	if _, err := store.UpdateLobbyByToken(ctx, adaToken.Hash(), start, func(l *game.Lobby) error {
		l.Game.StartsAt = start
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	center := saved.Playground.Center()
	cell, _ := saved.Playground.CellAt(center)
	report := func(token game.PlayerToken, at time.Time) {
		t.Helper()
		position, _ := game.NewPosition(center, 5, at)
		move(t, store, token, position)
	}
	report(adaToken, start.Add(time.Second))
	report(bobToken, start.Add(2*time.Second))
	report(bobToken, start.Add(3*time.Second))
	report(adaToken, start.Add(4*time.Second))
	want := []game.ZoneAlert{{Runner: ada.ID, Hunter: bob.ID, Cell: cell, At: start.Add(2 * time.Second)}}
	if got := load(t, store, saved.JoinCodes.Runner, start.Add(5*time.Second)).ZoneAlerts; !slices.Equal(got, want) {
		t.Fatalf("stored zone alerts = %+v; want %+v", got, want)
	}

	end := start.Add(time.Minute)
	if _, err := store.UpdateLobbyByToken(ctx, adaToken.Hash(), end, func(l *game.Lobby) error { return l.MarkCaught(ada.ID, end) }); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteGamesBefore(ctx, end.Add(game.GameRetention+time.Nanosecond)); err != nil {
		t.Fatal(err)
	}
	var left int64
	if err := store.db.Model(&zoneAlertRow{}).Count(&left).Error; err != nil || left != 0 {
		t.Fatalf("zone alerts of a deleted game kept: %d, %v", left, err)
	}
}

func TestZoneAlertIndexIsUnique(t *testing.T) {
	store := open(t, ":memory:")
	var indexes []struct {
		Name   string
		Unique int
	}
	if err := store.db.Raw("PRAGMA index_list('zone_alerts')").Scan(&indexes).Error; err != nil {
		t.Fatal(err)
	}
	for _, index := range indexes {
		if index.Name == "zone_alert_once" && index.Unique == 1 {
			return
		}
	}
	t.Fatalf("missing unique zone alert index: %+v", indexes)
}

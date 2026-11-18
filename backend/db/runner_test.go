package db

import (
	"context"
	"testing"
	"time"

	"catch-me/backend/game"
)

func TestRunnerRowsAndConstraints(t *testing.T) {
	store := open(t, ":memory:")
	saved := create(t, store)
	runner, token := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	hunter, _ := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
	var rows []runnerRow
	if err := store.db.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].PlayerID != runner.ID.String() || rows[0].Status != "active" || rows[0].EndedAt != nil {
		t.Fatalf("expected only the active runner's row: %+v", rows)
	}
	for _, tc := range []struct {
		status string
		at     *time.Time
	}{
		{"active", &now}, {"caught", nil}, {"finished", nil}, {"unknown", &now},
	} {
		if err := store.db.Model(&runnerRow{PlayerID: runner.ID.String()}).Updates(map[string]any{"status": tc.status, "ended_at": tc.at}).Error; err == nil {
			t.Fatalf("database accepted contradictory state: %+v", tc)
		}
	}
	if err := store.db.Create(&runnerRow{PlayerID: runner.ID.String(), Status: "active"}).Error; err == nil {
		t.Fatal("duplicate runner state accepted")
	}
	if err := store.db.Create(&runnerRow{PlayerID: game.NewPlayerID().String(), Status: "active"}).Error; err == nil {
		t.Fatal("orphan runner state accepted")
	}
	// Invalid role associations are also rejected while parsing stored data.
	if err := store.db.Create(&runnerRow{PlayerID: hunter.ID.String(), Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateLobbyByToken(context.Background(), token.Hash(), now, nil); err == nil {
		t.Fatal("hunter with runner state was accepted")
	}
	if err := store.db.Delete(&runnerRow{PlayerID: hunter.ID.String()}).Error; err != nil {
		t.Fatal(err)
	}
	// Leaving the lobby removes runner state and positions via the player FK.
	if _, err := store.UpdateLobbyByToken(context.Background(), token.Hash(), now, func(l *game.Lobby) error { return l.Leave(runner.ID, now) }); err != nil {
		t.Fatal(err)
	}
	var n int64
	if err := store.db.Model(&runnerRow{}).Count(&n).Error; err != nil || n != 0 {
		t.Fatalf("runner state survived player deletion: %d %v", n, err)
	}
	if count(t, store) != 0 {
		t.Fatal("positions survived player deletion")
	}
}

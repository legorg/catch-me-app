package db

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"catch-me/backend/game"
)

func TestJokerSettingsPersistAfterReopening(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "games.sqlite")
	store := open(t, path)
	fixtures := []game.Game{
		fixture(t, 52.52, 100, strings.Repeat("a", 32)),
		fixture(t, 52.52, 100, strings.Repeat("b", 32)),
		fixture(t, 52.52, 100, strings.Repeat("c", 32)),
	}
	fixtures[0].JokerSettings = game.JokerSettings{}
	s := &fixtures[1].JokerSettings
	fields := []*game.JokerAllowance{&s.ShowRunners, &s.QueryZone, &s.Compass, &s.Trap.JokerAllowance, &s.ZoneOnly, &s.FakePosition, &s.ShowHunters}
	for i, field := range fields {
		*field, _ = game.ParseJokerAllowance(i % 6)
	}
	for _, saved := range fixtures {
		if err := store.Create(ctx, saved); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := open(t, path)
	for _, saved := range fixtures {
		if got := load(t, reopened, saved.JoinCodes.Hunter, now).Game.JokerSettings; got != saved.JokerSettings {
			t.Fatalf("joker settings did not persist: got %+v, want %+v", got, saved.JokerSettings)
		}
	}
}

func TestOlderGamesReceiveDefaultJokerSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "games.sqlite")
	store := open(t, path)
	saved := create(t, store)
	// Only this test's temporary database is changed: emulate the schema before
	// joker settings existed, then let Open add the columns to the existing row.
	for _, column := range []string{
		"joker_show_runners_allowance", "joker_query_zone_allowance", "joker_compass_allowance",
		"joker_trap_allowance", "joker_zone_only_allowance", "joker_fake_position_allowance",
		"joker_show_hunters_allowance", "joker_zone_alert_enabled",
	} {
		if err := store.db.Exec("ALTER TABLE games DROP COLUMN " + column).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := open(t, path)
	if got := load(t, reopened, saved.JoinCodes.Runner, now).Game.JokerSettings; got != game.DefaultJokerSettings() {
		t.Fatalf("older game did not receive defaults: %+v", got)
	}
}

func TestInvalidStoredJokerAllowanceIsNotAValidationError(t *testing.T) {
	store := open(t, ":memory:")
	saved := create(t, store)
	for _, uses := range []int{-1, 6} {
		if err := store.db.Model(&gameRow{}).Where("id = ?", saved.SessionID.String()).Update("joker_trap_allowance", uses).Error; err != nil {
			t.Fatal(err)
		}
		_, err := store.UpdateLobbyByJoinCode(context.Background(), saved.JoinCodes.Hunter, now, nil)
		var validation *game.ValidationError
		if err == nil || errors.As(err, &validation) {
			t.Fatalf("invalid stored allowance %d: expected server error, got %v", uses, err)
		}
	}
}

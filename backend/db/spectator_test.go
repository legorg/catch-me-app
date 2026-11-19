package db

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"catch-me/backend/game"
)

func TestSpectatorCodeAndIdentitySurviveReopening(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "spectator.sqlite")
	store := open(t, path)
	saved := create(t, store)
	name, _ := game.ParsePlayerName("Viewer")
	token := game.NewPlayerToken()
	lobby, err := store.UpdateLobbyByJoinCode(ctx, saved.JoinCodes.Spectator, now, func(l *game.Lobby) error {
		_, err := l.Join(saved.JoinCodes.Spectator, name, token, now)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	id := lobby.Players[0].ID
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = open(t, path)
	lobby, err = store.UpdateLobbyByToken(ctx, token.Hash(), now, nil)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := lobby.Player(id)
	if !ok || p.Team != game.Spectators || p.Role() != game.RoleSpectator || p.Ready || !p.Position.IsZero() || lobby.Game.JoinCodes != saved.JoinCodes {
		t.Fatal("reopening lost spectator identity or invitation")
	}
	var rows int64
	if err := store.db.Model(&runnerRow{}).Where("player_id = ?", id.String()).Count(&rows).Error; err != nil || rows != 0 {
		t.Fatalf("spectator has runner state: %d, %v", rows, err)
	}
	if count(t, store) != 0 {
		t.Fatal("spectator joining wrote position history")
	}
	endedAt := now.Add(time.Hour)
	if _, err := store.UpdateLobbyByToken(ctx, token.Hash(), endedAt, func(l *game.Lobby) error {
		l.Game.StartsAt, l.Game.EndedAt = now, endedAt
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	result, err := store.Result(ctx, saved.SessionID)
	if err != nil || len(result.Players) != 1 || result.Players[0].Team != game.Spectators || result.Players[0].ID != id || result.Players[0].Track != nil {
		t.Fatalf("spectator result affiliation lost: %+v, %v", result, err)
	}
}

func TestJoinCodeCollisionsAcrossAllThreeAffiliations(t *testing.T) {
	for _, existing := range []string{"runner", "hunter", "spectator"} {
		for _, incoming := range []string{"runner", "hunter", "spectator"} {
			t.Run(existing+"/"+incoming, func(t *testing.T) {
				store := open(t, filepath.Join(t.TempDir(), "codes.sqlite"))
				first := create(t, store)
				next := fixture(t, 52.52, 2000, strings.Repeat("b", 32))
				codes := map[string]game.JoinCode{"runner": first.JoinCodes.Runner, "hunter": first.JoinCodes.Hunter, "spectator": first.JoinCodes.Spectator}
				switch incoming {
				case "runner":
					next.JoinCodes.Runner = codes[existing]
				case "hunter":
					next.JoinCodes.Hunter = codes[existing]
				case "spectator":
					next.JoinCodes.Spectator = codes[existing]
				}
				if err := store.Create(context.Background(), next); !errors.Is(err, game.ErrJoinCodeTaken) {
					t.Fatalf("collision accepted: %v", err)
				}
			})
		}
	}
}

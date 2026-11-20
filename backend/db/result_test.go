package db

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"catch-me/backend/game"
)

func TestResultLookupPersistsWithTracksFromStartToEnd(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "result.sqlite")
	store := open(t, path)
	saved := create(t, store)
	ada, adaToken := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	cy, cyToken := join(t, store, saved.JoinCodes.Runner, "Cy", now.Add(time.Second))
	hunter, hunterToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(2*time.Second))
	if _, err := store.Result(ctx, game.NewSessionID()); !errors.Is(err, game.ErrNotFound) {
		t.Fatalf("unknown game: %v", err)
	}
	start := now.Add(time.Minute)
	for _, startsAt := range []time.Time{{}, start} {
		if _, err := store.UpdateLobbyByToken(ctx, adaToken.Hash(), now, func(l *game.Lobby) error {
			l.Game.StartsAt = startsAt
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Result(ctx, saved.SessionID); !errors.Is(err, game.ErrNotFound) {
			t.Fatalf("unended game returned a result: %v", err)
		}
	}
	if _, err := store.UpdateLobbyByToken(ctx, adaToken.Hash(), start, func(l *game.Lobby) error {
		return l.Finish(ada.ID, start, true)
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Result(ctx, saved.SessionID); !errors.Is(err, game.ErrNotFound) {
		t.Fatal("game with an active runner returned a result")
	}
	report := func(offset time.Duration) game.Position {
		p, _ := game.NewPosition(saved.Playground.Center(), 4, start.Add(offset))
		return p
	}
	move(t, store, cyToken, report(10*time.Second))
	move(t, store, cyToken, report(12*time.Second))
	move(t, store, hunterToken, report(15*time.Second))
	used := start.Add(20 * time.Second)
	if _, err := store.UpdateLobbyByToken(ctx, hunterToken.Hash(), used, func(l *game.Lobby) error {
		return l.UseJoker(hunter.ID, game.ShowRunners{}, used)
	}); err != nil {
		t.Fatal(err)
	}
	end := start.Add(37 * time.Second)
	if _, err := store.UpdateLobbyByToken(ctx, cyToken.Hash(), end, func(l *game.Lobby) error {
		return l.MarkCaught(cy.ID, end)
	}); err != nil {
		t.Fatal(err)
	}
	// Gameplay rejects reports after the end; a stored one must still not reach the track.
	late := positionRow{PlayerID: hunter.ID.String(), Latitude: 52.52, Longitude: 13.405, AccuracyMeters: 4, RecordedAt: end.Add(time.Second)}
	if err := store.db.Create(&late).Error; err != nil {
		t.Fatal(err)
	}
	result, err := store.Result(ctx, saved.SessionID)
	if err != nil || result.SessionID != saved.SessionID || result.StartsAt != start || result.EndedAt != end || len(result.Players) != 3 {
		t.Fatalf("result lookup failed: %+v %v", result, err)
	}
	if result.Players[0].ID != ada.ID || result.Players[0].Runner.Status() != game.RunnerFinished || result.Players[1].ID != cy.ID || result.Players[1].Runner.Status() != game.RunnerCaught || result.Players[2].HunterTeam != hunter.HunterTeam {
		t.Fatal("result lost join order, outcomes or hunter teams")
	}
	wantEvents := []game.Event{
		game.StartEvent{At: start},
		game.FinishEvent{Runner: ada.ID, At: start},
		game.HunterJokerEvent{Team: hunter.HunterTeam, Joker: game.JokerShowRunners, At: used},
		game.CatchEvent{Runner: cy.ID, At: end},
		game.EndEvent{At: end},
	}
	if !reflect.DeepEqual(result.Events, wantEvents) {
		t.Fatalf("result events = %+v, want %+v", result.Events, wantEvents)
	}
	// Lobby reports (Ada's and Cy's at joining) and the late report are left out.
	wantTracks := [][]game.Position{{}, {report(10 * time.Second), report(12 * time.Second)}, {report(15 * time.Second)}}
	for i, want := range wantTracks {
		if !reflect.DeepEqual(result.Players[i].Track, want) {
			t.Fatalf("track of %s = %+v, want %+v", result.Players[i].Name, result.Players[i].Track, want)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = open(t, path)
	// An invalid lobby position would fail gameplay loading, but results must not read it.
	if err := store.db.Model(&positionRow{}).Where("player_id = ? AND recorded_at < ?", ada.ID.String(), start).Update("latitude", 999).Error; err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.Result(ctx, saved.SessionID)
	if err != nil || !reflect.DeepEqual(reloaded, result) {
		t.Fatalf("reopening changed the result: %+v %v", reloaded, err)
	}
	if err := store.DeleteGamesBefore(ctx, end.Add(time.Nanosecond)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Result(ctx, saved.SessionID); !errors.Is(err, game.ErrNotFound) {
		t.Fatalf("deleted game returned a result: %v", err)
	}
}

func TestResultTimelineShowsATrapTrigger(t *testing.T) {
	ctx := context.Background()
	store := open(t, ":memory:")
	saved := create(t, store)
	ada, adaToken := join(t, store, saved.JoinCodes.Runner, "Ada", now)
	hunter, hunterToken := join(t, store, saved.JoinCodes.Hunter, "Bob", now.Add(time.Second))
	start := now.Add(time.Minute)
	if err := store.db.Model(&gameRow{ID: saved.SessionID.String()}).Update("starts_at", start).Error; err != nil {
		t.Fatal(err)
	}
	placed := start.Add(time.Minute)
	if _, err := store.UpdateLobbyByToken(ctx, hunterToken.Hash(), placed, func(l *game.Lobby) error {
		return l.UseJoker(hunter.ID, game.Trap{Point: saved.Playground.Center()}, placed)
	}); err != nil {
		t.Fatal(err)
	}
	triggered := placed.Add(game.TrapActivationDelay + time.Minute)
	report, _ := game.NewPosition(saved.Playground.Center(), 4, triggered)
	move(t, store, adaToken, report)
	end := triggered.Add(time.Minute)
	if _, err := store.UpdateLobbyByToken(ctx, adaToken.Hash(), end, func(l *game.Lobby) error {
		return l.MarkCaught(ada.ID, end)
	}); err != nil {
		t.Fatal(err)
	}
	result, err := store.Result(ctx, saved.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	wantEvents := []game.Event{
		game.StartEvent{At: start},
		game.HunterJokerEvent{Team: hunter.HunterTeam, Joker: game.JokerTrap, At: placed},
		game.TrapTriggeredEvent{Team: hunter.HunterTeam, At: triggered},
		game.CatchEvent{Runner: ada.ID, At: end},
		game.EndEvent{At: end},
	}
	if !reflect.DeepEqual(result.Events, wantEvents) {
		t.Fatalf("result events = %+v, want %+v", result.Events, wantEvents)
	}
}

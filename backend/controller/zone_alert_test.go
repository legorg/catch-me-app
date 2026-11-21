package controller

import (
	"testing"
	"time"

	"catch-me/backend/game"
	"catch-me/backend/service"
)

func TestRunnerStateListsOwnZoneAlertsAndShowRunnersUses(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	name := func(n string) game.PlayerName { parsed, _ := game.ParsePlayerName(n); return parsed }
	pack, _ := game.ParseHunterTeamName("Pack")
	ada := game.Player{ID: game.NewPlayerID(), Name: name("Ada"), Team: game.Runners}
	cy := game.Player{ID: game.NewPlayerID(), Name: name("Cy"), Team: game.Runners}
	bob := game.Player{ID: game.NewPlayerID(), Name: name("Bob"), Team: game.Hunters, HunterTeam: pack}
	cell, _ := game.ParseCell("C4")
	lobby := game.Lobby{
		Game:       game.Game{StartsAt: now.Add(-time.Minute), RevealInterval: game.DefaultRevealInterval(), JokerSettings: game.DefaultJokerSettings()},
		Players:    []game.Player{ada, cy, bob},
		ZoneAlerts: []game.ZoneAlert{{Runner: ada.ID, Hunter: bob.ID, Cell: cell, At: now}},
	}
	state := func(me game.Player) GameStateRunnerResponse {
		t.Helper()
		runner, ok := gameStateResponse(service.PlayerView{Lobby: lobby, Me: me, Now: now}).Runner()
		if !ok {
			t.Fatal("expected the runner variant")
		}
		return runner
	}
	want := ZoneAlertResponse{At: now, HunterID: PlayerID(bob.ID.String()), Cell: "C4"}
	if got := state(ada).ZoneAlerts; len(got) != 1 || got[0] != want {
		t.Fatalf("Ada's zone alerts = %+v; want %+v", got, want)
	}
	if got := state(cy).ZoneAlerts; got == nil || len(got) != 0 {
		t.Fatalf("Cy must get an empty list, not Ada's alerts: %+v", got)
	}
	if got := state(cy).LocatedBy; got == nil || len(got) != 0 {
		t.Fatalf("without a showRunners use the list is empty: %+v", got)
	}
	lobby.Jokers = []game.JokerUse{game.ShowRunnersUse{Team: pack, At: now}}
	if got := state(cy).LocatedBy; len(got) != 1 || got[0] != (LocatedByResponse{At: now, HunterTeam: "Pack"}) {
		t.Fatalf("runners learn the team and time of a showRunners use: %+v", got)
	}
}

package controller

import (
	"testing"
	"time"

	"catch-me/backend/game"
	"catch-me/backend/service"
)

func TestGameStateShowsSketchesToTheOwnHunterTeamOnly(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	pack, _ := game.ParseHunterTeamName("Pack")
	wolves, _ := game.ParseHunterTeamName("Wolves")
	bob := game.Player{ID: game.NewPlayerID(), Team: game.Hunters, HunterTeam: pack}
	eve := game.Player{ID: game.NewPlayerID(), Team: game.Hunters, HunterTeam: wolves}
	ada := game.Player{ID: game.NewPlayerID(), Team: game.Runners}
	lobby := game.Lobby{Game: game.Game{StartsAt: now.Add(-time.Minute)}, Players: []game.Player{ada, bob, eve}}
	first, _ := game.ParseCoordinates(52.52, 13.405)
	second, _ := game.ParseCoordinates(52.53, 13.415)
	line, _ := game.NewSketchLine([]game.Coordinates{first, second})
	if err := lobby.AddSketch(bob.ID, line, now); err != nil {
		t.Fatal(err)
	}

	got := gameStateResponse(service.PlayerView{Lobby: lobby, Me: bob, Now: now}).Sketches
	if len(got) != 1 || len(got[0].SketchID) != 32 || !got[0].CreatedAt.Equal(now) ||
		len(got[0].Points) != 2 || got[0].Points[1] != (Coordinates{Latitude: 52.53, Longitude: 13.415}) {
		t.Fatalf("the team's sketch is missing: %+v", got)
	}
	for _, viewer := range []game.Player{ada, eve} {
		if got := gameStateResponse(service.PlayerView{Lobby: lobby, Me: viewer, Now: now}).Sketches; got == nil || len(got) != 0 {
			t.Fatalf("%s must get an empty list: %+v", viewer.Team, got)
		}
	}
}

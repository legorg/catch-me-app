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

	bobState, ok := gameStateResponse(service.PlayerView{Lobby: lobby, Me: bob, Now: now}).Hunter()
	if !ok {
		t.Fatal("a hunter must get the hunter state")
	}
	got := bobState.Sketches
	if len(got) != 1 || len(got[0].SketchID) != 32 || !got[0].CreatedAt.Equal(now) ||
		len(got[0].Points) != 2 || got[0].Points[1] != (Coordinates{Latitude: 52.53, Longitude: 13.415}) {
		t.Fatalf("the team's sketch is missing: %+v", got)
	}
	if _, ok := gameStateResponse(service.PlayerView{Lobby: lobby, Me: ada, Now: now}).Hunter(); ok {
		t.Fatal("a runner must not get the hunter state")
	}
	eveState, _ := gameStateResponse(service.PlayerView{Lobby: lobby, Me: eve, Now: now}).Hunter()
	if eveState.Sketches == nil || len(eveState.Sketches) != 0 {
		t.Fatalf("another hunter team must get an empty list: %+v", eveState.Sketches)
	}
}

func TestGameStateShowsTheQueriedZoneAndItsAnswer(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	pack, _ := game.ParseHunterTeamName("Pack")
	bob := game.Player{ID: game.NewPlayerID(), Team: game.Hunters, HunterTeam: pack}
	ada := game.Player{ID: game.NewPlayerID(), Team: game.Runners}
	lobby := game.Lobby{Game: game.Game{StartsAt: now.Add(-time.Minute)}, Players: []game.Player{ada, bob}}
	choice, _ := game.ParseJokerChoice("queryZone", "A1")
	if err := lobby.UseJoker(bob.ID, choice, now); err != nil {
		t.Fatal(err)
	}

	bobState, _ := gameStateResponse(service.PlayerView{Lobby: lobby, Me: bob, Now: now}).Hunter()
	got := bobState.Jokers
	if len(got) != 2 || got[0].Kind != "showRunners" || got[0].Cell != "" || got[0].Found != nil {
		t.Fatalf("an unused joker has no answer: %+v", got)
	}
	// "No" must be sent as false, not left out.
	if got[1].Kind != "queryZone" || got[1].Available || got[1].Cell != "A1" || got[1].Found == nil || *got[1].Found || len(got[1].Positions) != 0 {
		t.Fatalf("the query must show the cell and the answer, without positions: %+v", got[1])
	}
}

func TestGameStateShowsOnlyTheZoneAfterTheZoneJoker(t *testing.T) {
	start := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	center, _ := game.ParseCoordinates(52.52, 13.405)
	playground, _ := game.NewPlayground(center, game.DefaultCellSize())
	pack, _ := game.ParseHunterTeamName("Pack")
	bob := game.Player{ID: game.NewPlayerID(), Team: game.Hunters, HunterTeam: pack}
	ada := game.Player{ID: game.NewPlayerID(), Team: game.Runners}
	cy := game.Player{ID: game.NewPlayerID(), Team: game.Runners}
	lobby := game.Lobby{Game: game.Game{StartsAt: start, Playground: playground}, Players: []game.Player{ada, cy, bob}}
	first, second := start.Add(game.RevealInterval), start.Add(2*game.RevealInterval)
	position, _ := game.NewPosition(center, 5, first.Add(-time.Second))
	lobby.Reveals = map[game.PlayerID][]game.Reveal{ada.ID: {{Position: position, At: first}, {Position: position, At: second}}}
	if err := lobby.UseJoker(ada.ID, game.JokerChoice{Kind: game.JokerZoneOnly}, first.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	now := second.Add(time.Second)

	bobState, _ := gameStateResponse(service.PlayerView{Lobby: lobby, Me: bob, Now: now}).Hunter()
	if len(bobState.Jokers) != 2 {
		t.Fatalf("hunters only get their own jokers: %+v", bobState.Jokers)
	}
	reveals := bobState.Players[0].Reveals
	if len(reveals) != 2 || reveals[0].Position == nil || reveals[0].ZoneOnly {
		t.Fatalf("the reveal before the joker keeps its position: %+v", reveals)
	}
	cell, _ := playground.CellAt(center)
	if reveals[1].Position != nil || !reveals[1].ZoneOnly || reveals[1].Cell != Cell(cell.String()) || !reveals[1].RevealedAt.Equal(second) {
		t.Fatalf("the next reveal shows only the cell: %+v", reveals[1])
	}
	cyState, _ := gameStateResponse(service.PlayerView{Lobby: lobby, Me: cy, Now: now}).Runner()
	if len(cyState.Jokers) != 1 || cyState.Jokers[0].Kind != "zoneOnly" || cyState.Jokers[0].Available || cyState.Jokers[0].UsedBy != PlayerID(ada.ID.String()) {
		t.Fatalf("every runner sees who used the runner joker: %+v", cyState.Jokers)
	}
}

package controller

import (
	"encoding/json"
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
	lobby := game.Lobby{Game: game.Game{StartsAt: now.Add(-time.Minute), RevealInterval: game.DefaultRevealInterval()}, Players: []game.Player{ada, bob, eve}}
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
	center, _ := game.ParseCoordinates(52.52, 13.405)
	playground, _ := game.NewPlayground(center, game.DefaultCellSize(), game.DefaultGridSize())
	lobby := game.Lobby{Game: game.Game{StartsAt: now.Add(-time.Minute), Playground: playground, RevealInterval: game.DefaultRevealInterval()}, Players: []game.Player{ada, bob}}
	a1, _ := game.ParseCell("A1")
	choice := game.QueryZone{Cell: a1}
	if err := lobby.UseJoker(bob.ID, choice, now); err != nil {
		t.Fatal(err)
	}

	bobState, _ := gameStateResponse(service.PlayerView{Lobby: lobby, Me: bob, Now: now}).Hunter()
	// An unused joker has no result; "no" must be sent as false, not left out.
	want := `[{"kind":"showRunners","available":true},{"kind":"queryZone","available":false,"usedAt":"2026-11-08T12:00:00Z","cell":"A1","found":false}]`
	if got := jsonOf(t, bobState.Jokers); got != want {
		t.Fatalf("jokers:\n got %s\nwant %s", got, want)
	}
}

// everyMinute keeps the expected reveal times short.
var everyMinute, _ = game.ParseRevealInterval(1)

// jsonOf is v as the client receives it.
func jsonOf(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestGameStateShowsOnlyTheZoneAfterTheZoneJoker(t *testing.T) {
	start := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	center, _ := game.ParseCoordinates(52.52, 13.405)
	playground, _ := game.NewPlayground(center, game.DefaultCellSize(), game.DefaultGridSize())
	pack, _ := game.ParseHunterTeamName("Pack")
	bob := game.Player{ID: game.NewPlayerID(), Team: game.Hunters, HunterTeam: pack}
	ada := game.Player{ID: game.NewPlayerID(), Team: game.Runners}
	cy := game.Player{ID: game.NewPlayerID(), Team: game.Runners}
	lobby := game.Lobby{Game: game.Game{StartsAt: start, Playground: playground, RevealInterval: everyMinute}, Players: []game.Player{ada, cy, bob}}
	first, second := start.Add(time.Minute), start.Add(2*time.Minute)
	position, _ := game.NewPosition(center, 5, first.Add(-time.Second))
	lobby.Reveals = map[game.PlayerID][]game.Reveal{ada.ID: {{Position: position, At: first}, {Position: position, At: second}}}
	if err := lobby.UseJoker(ada.ID, game.ZoneOnly{}, first.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	now := second.Add(time.Second)

	bobState, _ := gameStateResponse(service.PlayerView{Lobby: lobby, Me: bob, Now: now}).Hunter()
	if len(bobState.Jokers) != 2 {
		t.Fatalf("hunters only get their own jokers: %+v", bobState.Jokers)
	}
	reveals := bobState.Players[0].Reveals
	cell, _ := playground.CellAt(center)
	want := `[{"kind":"position","revealedAt":"2026-11-08T12:01:00Z","position":{"latitude":52.52,"longitude":13.405,"accuracyMeters":5,"recordedAt":"2026-11-08T12:00:59Z"}},` +
		`{"kind":"zone","revealedAt":"2026-11-08T12:02:00Z","cell":"` + cell.String() + `"}]`
	if got := jsonOf(t, reveals); got != want {
		t.Fatalf("only the reveal after the joker shows just the cell:\n got %s\nwant %s", got, want)
	}
	cyState, _ := gameStateResponse(service.PlayerView{Lobby: lobby, Me: cy, Now: now}).Runner()
	want = `{"kind":"zoneOnly","available":false,"usedAt":"2026-11-08T12:01:01Z","usedBy":"` + ada.ID.String() + `"}`
	if got := jsonOf(t, cyState.Jokers[0]); len(cyState.Jokers) != 3 || got != want {
		t.Fatalf("every runner sees who used the runner joker:\n got %s\nwant %s", got, want)
	}
}

func TestGameStateShowsTheFakePointAsANormalReveal(t *testing.T) {
	start := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	center, _ := game.ParseCoordinates(52.52, 13.405)
	point, _ := game.ParseCoordinates(52.53, 13.415)
	playground, _ := game.NewPlayground(center, game.DefaultCellSize(), game.DefaultGridSize())
	pack, _ := game.ParseHunterTeamName("Pack")
	bob := game.Player{ID: game.NewPlayerID(), Team: game.Hunters, HunterTeam: pack}
	ada := game.Player{ID: game.NewPlayerID(), Team: game.Runners}
	lobby := game.Lobby{Game: game.Game{StartsAt: start, Playground: playground, RevealInterval: everyMinute}, Players: []game.Player{ada, bob}}
	first := start.Add(time.Minute)
	position, _ := game.NewPosition(center, 5, first.Add(-time.Second))
	lobby.Reveals = map[game.PlayerID][]game.Reveal{ada.ID: {{Position: position, At: first}}}
	if err := lobby.UseJoker(ada.ID, game.FakePosition{Point: point}, start.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	now := first.Add(time.Second)

	bobState, _ := gameStateResponse(service.PlayerView{Lobby: lobby, Me: bob, Now: now}).Hunter()
	reveals := bobState.Players[0].Reveals
	want := `[{"kind":"position","revealedAt":"2026-11-08T12:01:00Z","position":{"latitude":52.53,"longitude":13.415,"accuracyMeters":5,"recordedAt":"2026-11-08T12:00:59Z"}}]`
	if got := jsonOf(t, reveals); got != want {
		t.Fatalf("hunters must see the chosen point like a normal reveal:\n got %s\nwant %s", got, want)
	}
	adaState, _ := gameStateResponse(service.PlayerView{Lobby: lobby, Me: ada, Now: now}).Runner()
	want = `{"kind":"fakePosition","available":false,"usedAt":"2026-11-08T12:00:01Z","usedBy":"` + ada.ID.String() + `","point":{"latitude":52.53,"longitude":13.415}}`
	if got := jsonOf(t, adaState.Jokers[1]); got != want {
		t.Fatalf("runners see the point they showed:\n got %s\nwant %s", got, want)
	}
}

func TestGameStateShowsTheHunterPositionsToRunnersOnly(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	pack, _ := game.ParseHunterTeamName("Pack")
	where, _ := game.ParseCoordinates(52.52, 13.405)
	position, _ := game.NewPosition(where, 5, now.Add(-time.Second))
	bob := game.Player{ID: game.NewPlayerID(), Team: game.Hunters, HunterTeam: pack, Position: position}
	ada := game.Player{ID: game.NewPlayerID(), Team: game.Runners}
	lobby := game.Lobby{Game: game.Game{StartsAt: now.Add(-time.Minute), RevealInterval: game.DefaultRevealInterval()}, Players: []game.Player{ada, bob}}
	if err := lobby.UseJoker(ada.ID, game.ShowHunters{}, now); err != nil {
		t.Fatal(err)
	}

	adaState, _ := gameStateResponse(service.PlayerView{Lobby: lobby, Me: ada, Now: now}).Runner()
	want := `{"kind":"showHunters","available":false,"usedAt":"2026-11-08T12:00:00Z","usedBy":"` + ada.ID.String() + `",` +
		`"positions":[{"playerId":"` + bob.ID.String() + `","latitude":52.52,"longitude":13.405,"accuracyMeters":5,"recordedAt":"2026-11-08T11:59:59Z"}]}`
	if got := jsonOf(t, adaState.Jokers[2]); len(adaState.Jokers) != 3 || got != want {
		t.Fatalf("runners see where the hunters were:\n got %s\nwant %s", got, want)
	}
	for _, p := range adaState.Players {
		if p.PlayerID == PlayerID(bob.ID.String()) && p.Position != nil {
			t.Fatal("runners still get no live hunter position")
		}
	}
	bobState, _ := gameStateResponse(service.PlayerView{Lobby: lobby, Me: bob, Now: now}).Hunter()
	if got := jsonOf(t, bobState.Jokers); got != `[{"kind":"showRunners","available":true},{"kind":"queryZone","available":true}]` {
		t.Fatalf("hunters are not told: %s", got)
	}
}

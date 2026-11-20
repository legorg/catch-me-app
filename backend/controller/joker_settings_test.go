package controller

import (
	"encoding/json"
	"testing"
	"time"

	"catch-me/backend/game"
	"catch-me/backend/service"
)

func TestJokerSettingsResponseIncludesEveryConfiguredAllowance(t *testing.T) {
	allowance := func(uses int) game.JokerAllowance {
		a, err := game.ParseJokerAllowance(uses)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	settings := game.JokerSettings{
		ShowRunners: allowance(0), QueryZone: allowance(1), Compass: allowance(2),
		Trap: game.TrapSettings{JokerAllowance: allowance(3)}, ZoneOnly: allowance(4),
		FakePosition: allowance(5), ShowHunters: allowance(0),
	}
	want := `{"showRunners":{"allowance":0},"queryZone":{"allowance":1},"compass":{"allowance":2},"trap":{"allowance":3},"zoneOnly":{"allowance":4},"fakePosition":{"allowance":5},"showHunters":{"allowance":0}}`
	if got := jsonOf(t, jokerSettingsResponse(settings)); got != want {
		t.Fatalf("configured allowances changed:\n got %s\nwant %s", got, want)
	}
	created := gameResponse(game.Game{JokerSettings: settings})
	if got := jsonOf(t, created.JokerSettings); got != want {
		t.Fatalf("creation response must include the same settings: %s", got)
	}
}

func TestGameStateSettingsAndRemainingUsesRespectOwnershipAndPhase(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	pack, _ := game.ParseHunterTeamName("Pack")
	wolves, _ := game.ParseHunterTeamName("Wolves")
	ada := game.Player{ID: game.NewPlayerID(), Team: game.Runners}
	cy := game.Player{ID: game.NewPlayerID(), Team: game.Runners}
	bob := game.Player{ID: game.NewPlayerID(), Team: game.Hunters, HunterTeam: pack}
	dan := game.Player{ID: game.NewPlayerID(), Team: game.Hunters, HunterTeam: pack}
	eve := game.Player{ID: game.NewPlayerID(), Team: game.Hunters, HunterTeam: wolves}
	viewer := game.Player{ID: game.NewPlayerID(), Team: game.Spectators}
	center, _ := game.ParseCoordinates(52.52, 13.405)
	playground, _ := game.NewPlayground(center, game.DefaultCellSize(), game.DefaultGridSize())
	a, _ := game.ParseJokerAllowance(2)
	settings := game.JokerSettings{
		ShowRunners: a, Compass: a, Trap: game.TrapSettings{JokerAllowance: a},
		ZoneOnly: a, FakePosition: a, ShowHunters: a,
	}
	lobby := game.Lobby{
		Game:    game.Game{StartsAt: now.Add(-time.Minute), Playground: playground, RevealInterval: game.DefaultRevealInterval(), JokerSettings: settings},
		Players: []game.Player{ada, cy, bob, dan, eve, viewer},
	}
	assertState := func(player game.Player, at time.Time) GameStateResponse {
		t.Helper()
		me, _ := lobby.Player(player.ID)
		state := gameStateResponse(service.PlayerView{Lobby: lobby, Me: me, Now: at})
		if jsonOf(t, state.Base().JokerSettings) != jsonOf(t, jokerSettingsResponse(settings)) {
			t.Fatalf("settings changed for %s in phase %s", player.Team, state.Base().Status)
		}
		return state
	}
	assertRemaining := func(player game.Player, kind game.JokerKind, remaining int, available bool, at time.Time) {
		t.Helper()
		var state struct {
			Jokers []struct {
				Kind      game.JokerKind
				Remaining *int
				Available bool
			}
		}
		if err := json.Unmarshal([]byte(jsonOf(t, assertState(player, at))), &state); err != nil {
			t.Fatal(err)
		}
		for _, joker := range state.Jokers {
			if joker.Kind == kind {
				if joker.Remaining == nil || *joker.Remaining != remaining || joker.Available != available {
					t.Fatalf("incorrect %s status for %s: %+v", kind, player.ID, joker)
				}
				return
			}
		}
		t.Fatalf("missing permitted joker %s", kind)
	}
	for _, at := range []time.Time{now.Add(-2 * time.Minute), now} {
		for _, player := range lobby.Players {
			state := assertState(player, at)
			if player.Team == game.Spectators {
				spectator, _ := state.Spectator()
				if len(spectator.Jokers) != 0 {
					t.Fatal("spectators must receive settings, not joker uses")
				}
			}
		}
	}
	assertRemaining(bob, game.JokerShowRunners, 2, false, now.Add(-2*time.Minute))
	assertRemaining(bob, game.JokerQueryZone, 0, false, now)
	for _, use := range []struct {
		player game.Player
		choice game.JokerChoice
	}{
		{bob, game.ShowRunners{}},
		{ada, game.ShowHunters{}},
		{ada, game.FakePosition{Point: center}},
	} {
		if err := lobby.UseJoker(use.player.ID, use.choice, now); err != nil {
			t.Fatal(err)
		}
	}
	assertRemaining(bob, game.JokerShowRunners, 1, true, now)
	assertRemaining(dan, game.JokerShowRunners, 1, true, now)
	assertRemaining(eve, game.JokerShowRunners, 2, true, now)
	assertRemaining(ada, game.JokerShowHunters, 1, true, now)
	assertRemaining(cy, game.JokerShowHunters, 1, true, now)
	assertRemaining(ada, game.JokerFakePosition, 1, true, now)
	assertRemaining(cy, game.JokerFakePosition, 2, true, now)
	if err := lobby.MarkCaught(ada.ID, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	assertRemaining(ada, game.JokerFakePosition, 1, false, now.Add(time.Second))
	if err := lobby.MarkCaught(cy.ID, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	assertRemaining(bob, game.JokerShowRunners, 1, false, now.Add(2*time.Second))
	assertRemaining(eve, game.JokerShowRunners, 2, false, now.Add(2*time.Second))
	assertState(viewer, now.Add(2*time.Second))
}

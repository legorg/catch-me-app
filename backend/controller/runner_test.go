package controller

import (
	"encoding/json"
	"testing"
	"time"

	"catch-me/backend/game"
	"catch-me/backend/service"
)

func TestRunnerResponseContract(t *testing.T) {
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	point, _ := game.ParseCoordinates(52.52, 13.405)
	report, _ := game.NewPosition(point, 5, at.Add(-time.Second))
	for _, status := range []game.RunnerStatus{game.RunnerActive, game.RunnerCaught, game.RunnerFinished} {
		for _, withReport := range []bool{false, true} {
			end := at
			if status == game.RunnerActive {
				end = time.Time{}
			}
			runnerState, err := game.ParseRunnerState(string(status), end)
			if err != nil {
				t.Fatal(err)
			}
			runner := game.Player{ID: game.NewPlayerID(), Team: game.Runners, Runner: runnerState}
			if withReport {
				runner.Position = report
			}
			hunter := game.Player{ID: game.NewPlayerID(), Team: game.Hunters}
			lobby := game.Lobby{Game: game.Game{StartsAt: at.Add(-time.Minute), RevealInterval: game.DefaultRevealInterval()}, Players: []game.Player{runner, hunter}}
			for _, viewer := range []game.Player{runner, hunter} {
				players := gameStateResponse(service.PlayerView{Lobby: lobby, Me: viewer, Now: at}).Base().Players
				got := players[0]
				if got.Runner == nil || string(got.Runner.Status) != string(status) || (got.Runner.EndedAt == nil) != (status == game.RunnerActive) {
					t.Fatalf("incorrect runner state: %+v", got)
				}
				if got.Runner.EndedAt != nil && !got.Runner.EndedAt.Equal(at) {
					t.Fatal("incorrect end time")
				}
				if (got.Runner.CaughtPosition != nil) != (status == game.RunnerCaught && withReport) {
					t.Fatalf("catch coordinates have wrong visibility: %+v", got)
				}
				if status != game.RunnerActive && (got.Position != nil || len(got.Reveals) != 0) {
					t.Fatal("completed runner exposed ordinary position/reveals")
				}
				if players[1].Runner != nil {
					t.Fatal("hunter received runner state")
				}
				// Check real JSON, not only Go structs: obsolete fields must disappear.
				var raw []map[string]json.RawMessage
				if err := json.Unmarshal([]byte(jsonOf(t, players)), &raw); err != nil {
					t.Fatal(err)
				}
				for _, p := range raw {
					for _, old := range []string{"caughtAt", "finishedAt", "caughtPosition"} {
						if _, ok := p[old]; ok {
							t.Fatalf("obsolete top-level field %s", old)
						}
					}
				}
				if _, ok := raw[1]["runner"]; ok {
					t.Fatal("hunter JSON must omit runner, not send null")
				}
				var stateJSON map[string]json.RawMessage
				if err := json.Unmarshal(raw[0]["runner"], &stateJSON); err != nil {
					t.Fatal(err)
				}
				if _, ok := stateJSON["endedAt"]; ok != (status != game.RunnerActive) {
					t.Fatal("active JSON must omit endedAt; completed JSON must include it")
				}
				if _, ok := stateJSON["caughtPosition"]; ok != (status == game.RunnerCaught && withReport) {
					t.Fatal("JSON must omit catch coordinates unless caught with a retained report")
				}
			}
		}
	}
}

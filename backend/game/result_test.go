package game

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestResultIsOnlyAvailableAfterGameEnd(t *testing.T) {
	l, runner, hunter, _, _, running := jokerLobby(t)
	for _, startsAt := range []time.Time{{}, running.Add(time.Hour), running} {
		before := l
		before.Game.StartsAt = startsAt
		if _, err := before.Result(nil); !errors.Is(err, ErrNotFound) {
			t.Fatalf("unfinished game exposed a result: %v", err)
		}
	}
	if err := l.Finish(runner.ID, running, true); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Result(nil); !errors.Is(err, ErrNotFound) {
		t.Fatal("first finish exposed a result while another runner is active")
	}
	endedAt := running.Add(time.Minute)
	if err := l.MarkCaught(l.Players[1].ID, endedAt); err != nil {
		t.Fatal(err)
	}
	result, err := l.Result(nil)
	if err != nil || result.SessionID != l.Game.SessionID || result.StartsAt != running || result.EndedAt != endedAt || len(result.Players) != len(l.Players) {
		t.Fatalf("unexpected result: %+v %v", result, err)
	}
	wantEvents := []Event{
		StartEvent{At: running},
		FinishEvent{Runner: runner.ID, At: running},
		CatchEvent{Runner: l.Players[1].ID, At: endedAt},
		EndEvent{At: endedAt},
	}
	if !slices.Equal(result.Events, wantEvents) {
		t.Fatalf("result events = %+v, want the whole timeline %+v", result.Events, wantEvents)
	}
	if got, ok := result.Player(runner.ID); !ok || got.Name != runner.Name {
		t.Fatal("result player lookup failed")
	}
	for i, player := range l.Players {
		got := result.Players[i]
		if got.ID != player.ID || got.Name != player.Name || got.Team != player.Team || got.HunterTeam != player.HunterTeam || got.Runner != player.Runner {
			t.Fatal("result lost original team, hunter team or runner outcome")
		}
	}
	if result.Players[0].Runner.Status() != RunnerFinished || result.Players[1].Runner.Status() != RunnerCaught || result.Players[2].HunterTeam != hunter.HunterTeam {
		t.Fatal("mixed outcomes or hunter team missing")
	}
	result.Players[0].Name = PlayerName{}
	if l.Players[0].Name != runner.Name {
		t.Fatal("result player slice aliases the lobby")
	}
}

func TestResultTracks(t *testing.T) {
	l, runner, hunter, _, _, running := jokerLobby(t)
	end := running.Add(time.Minute)
	l.Game.EndedAt = end
	spectator := Player{ID: NewPlayerID(), Team: Spectators}
	l.Players = append(l.Players, spectator)
	report := func(offset time.Duration) Position {
		p, _ := NewPosition(l.Game.Playground.Center(), 5, running.Add(offset))
		return p
	}
	histories := map[PlayerID][]Position{
		// Lobby report, then reports thinned to 5 s, then the last report kept.
		runner.ID: {report(-time.Second), report(0), report(3 * time.Second), report(5 * time.Second), report(9 * time.Second), report(10 * time.Second), report(12 * time.Second)},
		// Reports after the end are left out.
		hunter.ID:    {report(30 * time.Second), report(2 * time.Minute)},
		spectator.ID: {report(0)},
	}
	result, err := l.Result(histories)
	if err != nil {
		t.Fatal(err)
	}
	want := []Position{report(0), report(5 * time.Second), report(10 * time.Second), report(12 * time.Second)}
	if got, _ := result.Player(runner.ID); !slices.Equal(got.Track, want) {
		t.Fatalf("runner track = %+v, want %+v", got.Track, want)
	}
	if got, _ := result.Player(hunter.ID); !slices.Equal(got.Track, []Position{report(30 * time.Second)}) {
		t.Fatalf("hunter track = %+v", got.Track)
	}
	if got, _ := result.Player(l.Players[1].ID); got.Track == nil || len(got.Track) != 0 {
		t.Fatal("runner without reports needs an empty, non-nil track")
	}
	if got, _ := result.Player(spectator.ID); got.Track != nil {
		t.Fatal("direct spectators have no track")
	}
}

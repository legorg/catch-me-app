package game

import (
	"errors"
	"reflect"
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
	if result.Route != l.Game.Route || !reflect.DeepEqual(result.Playground, l.Game.Playground) {
		t.Fatal("result lost the playground or route")
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

func TestTrackPointLimit(t *testing.T) {
	l, _, _, _, _, start := jokerLobby(t)
	for _, count := range []int{MaxTrackPoints - 1, MaxTrackPoints, MaxTrackPoints + 1, 2*MaxTrackPoints + 17} {
		history := make([]Position, count)
		for i := range history {
			at := start.Add(time.Duration(i) * TrackMinInterval)
			if i == count-1 {
				at = at.Add(-time.Second) // Keep the final report even below the usual interval.
			}
			history[i], _ = NewPosition(l.Game.Playground.Center(), 5, at)
		}
		before := slices.Clone(history)
		got := track(history, start, history[count-1].RecordedAt)
		if len(got) != min(count, MaxTrackPoints) {
			t.Fatalf("%d reports: got %d track points", count, len(got))
		}
		if got[0] != history[0] || got[len(got)-1] != history[count-1] {
			t.Fatalf("%d reports: track lost its first or last report", count)
		}
		if count > MaxTrackPoints {
			for i, point := range got {
				if point != history[int(int64(i)*int64(count-1)/int64(MaxTrackPoints-1))] {
					t.Fatalf("%d reports: track is not sampled evenly at %d", count, i)
				}
			}
		}
		for i := 1; i < len(got); i++ {
			if !got[i].RecordedAt.After(got[i-1].RecordedAt) {
				t.Fatalf("%d reports: track is not strictly chronological at %d", count, i)
			}
			if i < len(got)-1 && got[i].RecordedAt.Sub(got[i-1].RecordedAt) < TrackMinInterval {
				t.Fatalf("%d reports: track cadence is too short at %d", count, i)
			}
		}
		if count <= MaxTrackPoints && !slices.Equal(got, history) {
			t.Fatalf("%d reports: track below the limit was changed", count)
		}
		if !slices.Equal(history, before) {
			t.Fatal("track sampling modified stored history")
		}
	}
}

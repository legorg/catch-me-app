package game

import (
	"testing"
	"time"
)

func TestSpectatorVisibilityIsRealAndRunningOnly(t *testing.T) {
	for _, outcome := range []RunnerStatus{RunnerCaught, RunnerFinished} {
		t.Run(string(outcome), func(t *testing.T) {
			l, runner, hunter, _, _, at := jokerLobby(t)
			// Two hunter teams must both be visible, not just one hunter team.
			other := hunter
			other.ID = NewPlayerID()
			other.HunterTeam, _ = ParseHunterTeamName("Other")
			l.Players = append(l.Players, other)
			finished := runner
			finished.ID = NewPlayerID()
			finished.Runner, _ = ParseRunnerState(string(RunnerFinished), at.Add(-time.Second))
			l.Players = append(l.Players, finished)
			for i := range l.Players {
				l.Players[i].Position = position(t, l, "C3", at)
			}
			mate := l.Players[1]
			if err := l.UseJoker(mate.ID, FakePosition{Point: cellCenter(l.Game.Playground, Cell{})}, at); err != nil {
				t.Fatal(err)
			}
			l.Reveals = map[PlayerID][]Reveal{mate.ID: {{At: at, Position: mate.Position}}}
			var err error
			if outcome == RunnerCaught {
				err = l.MarkCaught(runner.ID, at)
			} else {
				err = l.Finish(runner.ID, at, true)
			}
			if err != nil {
				t.Fatal(err)
			}
			viewer, _ := l.Player(runner.ID)
			if viewer.Role() != RoleSpectator || viewer.Team != Runners || viewer.Runner.Status() != outcome || l.Game.Status(at) != StatusRunning {
				t.Fatal("conversion changed affiliation, outcome or ended the game early")
			}
			for _, v := range l.VisibleTo(viewer, at) {
				want, _ := l.Player(v.ID)
				if v.ID == viewer.ID && outcome == RunnerCaught {
					if v.CaughtPosition != want.Position || !v.Position.IsZero() {
						t.Fatal("caught marker not frozen")
					}
				} else if v.Position != want.Position {
					t.Fatalf("spectator did not get real report: %+v", v)
				}
				if len(v.Reveals) != 0 {
					t.Fatal("spectator feed contains delayed/fake reveals")
				}
			}
			// The same hidden hunter report must not leak into active runner responses.
			for _, v := range l.VisibleTo(mate, at) {
				if v.Team == Hunters && !v.Position.IsZero() {
					t.Fatal("spectator permissions leaked to an active runner")
				}
				if v.ID == finished.ID && !v.Position.IsZero() {
					t.Fatal("frozen finish report leaked to an active runner")
				}
			}
			// Exercise both concealment jokers; neither changes the real feed.
			l.Jokers = nil
			if err := l.UseJoker(mate.ID, ZoneOnly{}, at); err != nil {
				t.Fatal(err)
			}
			if got := l.VisibleTo(viewer, at)[1]; got.Position != mate.Position || len(got.Reveals) != 0 {
				t.Fatal("zone joker concealed a real report from spectator")
			}
			// No privilege outside the running phase, even for an authenticated
			// terminal runner in a deliberately inconsistent pre-start fixture.
			for _, phase := range []Status{StatusLobby, StatusCountdown, StatusEnded} {
				l.Game.StartsAt = time.Time{}
				l.Game.EndedAt = time.Time{}
				if phase == StatusCountdown {
					l.Game.StartsAt = at.Add(time.Second)
				}
				if phase == StatusEnded {
					l.Game.StartsAt, l.Game.EndedAt = at.Add(-time.Hour), at
				}
				for _, v := range l.VisibleTo(viewer, at) {
					if !v.Position.IsZero() || !v.CaughtPosition.IsZero() || len(v.Reveals) != 0 {
						t.Fatalf("spectator location exposed in %s", phase)
					}
				}
			}
		})
	}
}

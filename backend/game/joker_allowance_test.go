package game

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func jokerWithKind(t *testing.T, l Lobby, viewer Player, kind JokerKind, at time.Time) jokerState {
	t.Helper()
	for _, state := range l.JokersFor(viewer, at) {
		if state.Kind == kind {
			return state
		}
	}
	t.Fatalf("missing joker %s for %s", kind, viewer.Name)
	return jokerState{}
}

func settingsWithAllowance(max int) JokerSettings {
	var s JokerSettings
	for _, a := range s.allowances() {
		a.max = max
	}
	return s
}

func TestEveryJokerEnforcesItsAllowance(t *testing.T) {
	for _, kind := range append(append([]JokerKind{}, hunterJokers...), runnerJokers...) {
		for allowance := 0; allowance <= 5; allowance++ {
			t.Run(fmt.Sprintf("%s/%d", kind, allowance), func(t *testing.T) {
				l, runner, bob, _, _, running := jokerLobby(t)
				l.Game.JokerSettings = settingsWithAllowance(allowance)
				point := cellCenter(l.Game.Playground, Cell{})
				choices := map[JokerKind]JokerChoice{
					JokerShowRunners: ShowRunners{}, JokerQueryZone: QueryZone{Cell: Cell{}},
					JokerCompass: Compass{}, JokerTrap: Trap{Point: point},
					JokerZoneOnly: ZoneOnly{}, JokerFakePosition: FakePosition{Point: point},
					JokerShowHunters: ShowHunters{},
				}
				actor := runner
				if kind == JokerShowRunners || kind == JokerQueryZone || kind == JokerCompass || kind == JokerTrap {
					actor = bob
				}
				for used := 0; used <= allowance; used++ {
					at := running.Add(time.Duration(used) * revealInterval)
					state := jokerWithKind(t, l, actor, kind, at)
					if len(state.Uses) != used || state.Available != (used < allowance) {
						t.Fatalf("before use %d: %+v", used, state)
					}
					err := l.UseJoker(actor.ID, choices[kind], at)
					if used < allowance {
						if err != nil {
							t.Fatalf("use %d rejected: %v", used+1, err)
						}
					} else {
						var rule *RuleError
						if !errors.As(err, &rule) || len(l.Jokers) != allowance {
							t.Fatalf("exhausted allowance must reject without adding a use: %v", err)
						}
					}
				}
			})
		}
	}
}

func TestMultipleUsesKeepTheirOwnershipAndResults(t *testing.T) {
	l, runner, bob, dan, eve, running := jokerLobby(t)
	l.Game.JokerSettings = settingsWithAllowance(2)
	cy := l.Players[1]
	first, second := running.Add(time.Second), running.Add(revealInterval)
	for _, actor := range []Player{bob, dan} {
		if err := l.UseJoker(actor.ID, ShowRunners{}, first); err != nil {
			t.Fatal(err)
		}
	}
	// Team spelling cannot create a new allowance or hide a team result.
	dan.HunterTeam, _ = ParseHunterTeamName("pack")
	if got := jokerWithKind(t, l, dan, JokerShowRunners, first); len(got.Uses) != 2 {
		t.Fatalf("hunter teammates must share all uses: %+v", got)
	}
	if got := jokerWithKind(t, l, eve, JokerShowRunners, first); len(got.Uses) != 0 {
		t.Fatalf("another hunter team must have an independent allowance/history: %+v", got)
	}
	var rule *RuleError
	if err := l.UseJoker(dan.ID, ShowRunners{}, first); !errors.As(err, &rule) {
		t.Fatalf("a teammate must not exceed the shared allowance: %v", err)
	}
	if err := l.UseJoker(eve.ID, ShowRunners{}, first); err != nil {
		t.Fatalf("another hunter team must still be able to use its joker: %v", err)
	}
	for _, actor := range []Player{runner, cy} {
		if err := l.UseJoker(actor.ID, ShowHunters{}, first); err != nil {
			t.Fatal(err)
		}
	}
	for _, viewer := range []Player{runner, cy} {
		if got := jokerWithKind(t, l, viewer, JokerShowHunters, first); len(got.Uses) != 2 {
			t.Fatalf("runner teammates must share all uses: %+v", got)
		}
	}
	if err := l.UseJoker(cy.ID, ShowHunters{}, first); !errors.As(err, &rule) {
		t.Fatalf("another runner must not exceed the shared allowance: %v", err)
	}
	points := []Coordinates{cellCenter(l.Game.Playground, Cell{}), cellCenter(l.Game.Playground, Cell{column: 1})}
	for i, at := range []time.Time{first, second} {
		if err := l.UseJoker(runner.ID, FakePosition{Point: points[i]}, at); err != nil {
			t.Fatal(err)
		}
	}
	personal := jokerWithKind(t, l, runner, JokerFakePosition, second)
	want := []JokerUse{
		FakePositionUse{Runner: runner.ID, At: first, Point: points[0]},
		FakePositionUse{Runner: runner.ID, At: second, Point: points[1]},
	}
	if !reflect.DeepEqual(personal.Uses, want) {
		t.Fatalf("personal uses must keep each time and point: %+v", personal)
	}
	if got := jokerWithKind(t, l, cy, JokerFakePosition, second); len(got.Uses) != 0 {
		t.Fatalf("another runner must not spend or see personal uses: %+v", got)
	}
	if err := l.UseJoker(cy.ID, FakePosition{Point: points[0]}, second); err != nil {
		t.Fatalf("another runner must still be able to use a personal joker: %v", err)
	}
	if got := l.JokersFor(Player{Team: Spectators}, second); len(got) != 0 {
		t.Fatalf("spectators must not receive joker histories: %+v", got)
	}
}

func TestPendingRevealJokerDoesNotSpendAnotherUse(t *testing.T) {
	l, runner, _, _, _, running := jokerLobby(t)
	l.Game.JokerSettings = settingsWithAllowance(2)
	first := running.Add(time.Second)
	point := cellCenter(l.Game.Playground, Cell{})
	if err := l.UseJoker(runner.ID, ZoneOnly{}, first); err != nil {
		t.Fatal(err)
	}
	for _, choice := range []JokerChoice{ZoneOnly{}, FakePosition{Point: point}} {
		state := jokerWithKind(t, l, runner, choice.Kind(), first)
		if !state.Available {
			t.Fatalf("pending joker must not exhaust the allowance: %+v", state)
		}
		var rule *RuleError
		if err := l.UseJoker(runner.ID, choice, first.Add(time.Second)); !errors.As(err, &rule) || len(l.Jokers) != 1 {
			t.Fatalf("pending joker must reject without spending: %v", err)
		}
	}
	if err := l.UseJoker(runner.ID, ShowHunters{}, first); err != nil {
		t.Fatalf("an immediate snapshot must remain usable: %v", err)
	}
	revealed := running.Add(revealInterval)
	if err := l.UseJoker(runner.ID, ZoneOnly{}, revealed); err != nil {
		t.Fatalf("at the reveal the second use must be allowed: %v", err)
	}
}

func TestInvalidTargetDoesNotSpendConfiguredUses(t *testing.T) {
	l, runner, bob, _, _, running := jokerLobby(t)
	l.Game.JokerSettings = settingsWithAllowance(2)
	outside, _ := ParseCoordinates(0, 0)
	for _, tc := range []struct {
		actor  Player
		choice JokerChoice
	}{
		{runner, FakePosition{Point: outside}},
		{bob, Trap{Point: outside}},
		{bob, QueryZone{Cell: Cell{column: 100}}},
	} {
		var validation *ValidationError
		if err := l.UseJoker(tc.actor.ID, tc.choice, running); !errors.As(err, &validation) {
			t.Fatalf("invalid %s target: %v", tc.choice.Kind(), err)
		}
		state := jokerWithKind(t, l, tc.actor, tc.choice.Kind(), running)
		if len(state.Uses) != 0 {
			t.Fatalf("invalid target spent an allowance: %+v", state)
		}
	}
}

func TestCompassWithoutPositionDoesNotSpendConfiguredUses(t *testing.T) {
	l, _, bob, _, _, running := jokerLobby(t)
	l.Game.JokerSettings = settingsWithAllowance(2)
	for i := range l.Players {
		if l.Players[i].ID == bob.ID {
			l.Players[i].Position = Position{}
		}
	}
	var rule *RuleError
	if err := l.UseJoker(bob.ID, Compass{}, running); !errors.As(err, &rule) {
		t.Fatalf("compass without position must be rejected: %v", err)
	}
	if got := jokerWithKind(t, l, bob, JokerCompass, running); len(got.Uses) != 0 {
		t.Fatalf("rejected compass spent an allowance: %+v", got)
	}
}

func TestMultipleSnapshotsRetainTheirOriginalPositions(t *testing.T) {
	l, runner, bob, dan, _, running := jokerLobby(t)
	l.Game.JokerSettings = settingsWithAllowance(2)
	first, second := running.Add(time.Second), running.Add(2*time.Second)
	if err := l.UseJoker(bob.ID, ShowRunners{}, first); err != nil {
		t.Fatal(err)
	}
	before := jokerWithKind(t, l, bob, JokerShowRunners, first).Uses[0].(ShowRunnersUse)
	moved := position(t, l, "A1", second)
	if err := l.UpdatePosition(runner.ID, moved, second, false); err != nil {
		t.Fatal(err)
	}
	if err := l.UseJoker(dan.ID, ShowRunners{}, second); err != nil {
		t.Fatal(err)
	}
	uses := jokerWithKind(t, l, dan, JokerShowRunners, second).Uses
	if len(uses) != 2 || !reflect.DeepEqual(uses[0], before) {
		t.Fatalf("a new snapshot replaced the first: %+v", uses)
	}
	latest := uses[1].(ShowRunnersUse)
	if !latest.At.Equal(second) || latest.Positions[0].Position != moved || before.Positions[0].Position == moved {
		t.Fatalf("each snapshot must show its own report: before %+v, after %+v", before, latest)
	}
}

func TestMultipleTrapsCanBeActiveAndTriggerIndependently(t *testing.T) {
	l, runner, bob, dan, _, running := jokerLobby(t)
	l.Game.JokerSettings = settingsWithAllowance(2)
	first, second := running.Add(time.Second), running.Add(2*time.Second)
	points := []Coordinates{cellCenter(l.Game.Playground, Cell{}), cellCenter(l.Game.Playground, Cell{column: 2})}
	for i, actor := range []Player{bob, dan} {
		if err := l.UseJoker(actor.ID, Trap{Point: points[i]}, []time.Time{first, second}[i]); err != nil {
			t.Fatal(err)
		}
	}
	at := second.Add(TrapActivationDelay)
	uses := jokerWithKind(t, l, dan, JokerTrap, at).Uses
	if len(uses) != 2 {
		t.Fatalf("expected both placed traps: %+v", uses)
	}
	reports := make([]JokerPosition, 0, 2)
	for i, point := range points {
		p, _ := NewPosition(point, 5, at.Add(time.Duration(i)*time.Second))
		reports = append(reports, JokerPosition{PlayerID: runner.ID, Position: p})
	}
	for i, use := range uses {
		trap := use.(TrapUse)
		if at.Before(trap.ActiveAt()) || trap.Point != points[i] {
			t.Fatalf("each trap must remain independently active: %+v", trap)
		}
		if got := TrapTriggeredAt(trap.Point, trap.ActiveAt(), reports); !got.Equal(reports[i].Position.RecordedAt) {
			t.Fatalf("trap %d trigger: got %v, want %v", i, got, reports[i].Position.RecordedAt)
		}
	}
}

func TestRepeatedFakePositionsAffectTheirOwnNextReveals(t *testing.T) {
	l, runner, bob, _, _, running := jokerLobby(t)
	l.Game.JokerSettings = settingsWithAllowance(2)
	first, second, third := running.Add(revealInterval), running.Add(2*revealInterval), running.Add(3*revealInterval)
	actual := position(t, l, "C4", first)
	l.Reveals = map[PlayerID][]Reveal{runner.ID: {{Position: actual, At: first}, {Position: actual, At: second}, {Position: actual, At: third}}}
	points := []Coordinates{cellCenter(l.Game.Playground, Cell{}), cellCenter(l.Game.Playground, Cell{column: 1})}
	for i, at := range []time.Time{running.Add(time.Second), first} {
		if err := l.UseJoker(runner.ID, FakePosition{Point: points[i]}, at); err != nil {
			t.Fatal(err)
		}
	}
	for _, visible := range l.VisibleTo(bob, third) {
		if visible.ID != runner.ID {
			continue
		}
		want := []HunterReveal{
			PositionReveal{At: first, Position: Position{Coordinates: points[0], AccuracyMeters: actual.AccuracyMeters, RecordedAt: actual.RecordedAt}},
			PositionReveal{At: second, Position: Position{Coordinates: points[1], AccuracyMeters: actual.AccuracyMeters, RecordedAt: actual.RecordedAt}},
			PositionReveal{At: third, Position: actual},
		}
		if !reflect.DeepEqual(visible.Reveals, want) || l.Reveals[runner.ID][0].Position != actual || l.Reveals[runner.ID][1].Position != actual {
			t.Fatalf("each fake must affect only its next reveal, without changing real reports: %+v", visible.Reveals)
		}
		return
	}
	t.Fatal("runner missing from hunter view")
}

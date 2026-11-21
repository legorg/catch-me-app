package game

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestTimelineListsGameEventsInOrder(t *testing.T) {
	l, ada, bob, _, _, running := jokerLobby(t)
	watcher := joinSpectator(t, &l, "Wanda", start)
	cy := l.Players[1]
	if got, err := l.TimelineFor(watcher, running.Add(-time.Second)); err != nil || len(got) != 0 {
		t.Fatalf("a scheduled start is no event yet: %+v, %v", got, err)
	}
	steps := []func() error{
		func() error { return l.UseJoker(bob.ID, showRunners, running.Add(time.Minute)) },
		func() error { return l.UseJoker(ada.ID, zoneOnly, running.Add(2*time.Minute)) },
		func() error { return l.MarkCaught(ada.ID, running.Add(3*time.Minute)) },
		func() error { return l.Finish(cy.ID, running.Add(4*time.Minute), true) },
	}
	for _, step := range steps {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	end := running.Add(4 * time.Minute)
	want := []Event{
		StartEvent{At: running},
		HunterJokerEvent{Team: bob.HunterTeam, Joker: JokerShowRunners, At: running.Add(time.Minute)},
		RunnerJokerEvent{Runner: ada.ID, Joker: JokerZoneOnly, At: running.Add(2 * time.Minute)},
		CatchEvent{Runner: ada.ID, At: running.Add(3 * time.Minute)},
		FinishEvent{Runner: cy.ID, At: end},
		EndEvent{At: end},
	}
	got, err := l.TimelineFor(watcher, end)
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("timeline = %+v, %v; want %+v", got, err, want)
	}
	caught, _ := l.Player(ada.ID)
	if got, err := l.TimelineFor(caught, running.Add(2*time.Minute)); err != nil || !slices.Equal(got, want[:3]) {
		t.Fatalf("a caught runner sees the events up to now: %+v, %v", got, err)
	}
}

func TestTimelineShowsATrapTriggerWithoutTheRunner(t *testing.T) {
	l, _, bob, _, _, running := jokerLobby(t)
	watcher := joinSpectator(t, &l, "Wanda", start)
	placed, triggered := running.Add(time.Minute), running.Add(30*time.Minute)
	l.Jokers = []JokerUse{TrapUse{Team: bob.HunterTeam, At: placed, Point: l.Game.Playground.Center(), TriggeredAt: triggered}}
	want := []Event{
		StartEvent{At: running},
		HunterJokerEvent{Team: bob.HunterTeam, Joker: JokerTrap, At: placed},
		TrapTriggeredEvent{Team: bob.HunterTeam, At: triggered},
	}
	if got, err := l.TimelineFor(watcher, triggered); err != nil || !slices.Equal(got, want) {
		t.Fatalf("timeline = %+v, %v; want %+v", got, err, want)
	}
	if got, _ := l.TimelineFor(watcher, triggered.Add(-time.Second)); !slices.Equal(got, want[:2]) {
		t.Fatalf("an untriggered trap has only its placement: %+v", got)
	}
}

func TestTimelineShowsZoneAlertsWithoutTheCell(t *testing.T) {
	l, ada, _, _, eve, running := jokerLobby(t)
	watcher := joinSpectator(t, &l, "Wanda", start)
	alerted := running.Add(time.Minute)
	l.ZoneAlerts = []ZoneAlert{{Runner: ada.ID, Hunter: eve.ID, Cell: Cell{column: 1, row: 1}, At: alerted}}
	want := []Event{StartEvent{At: running}, ZoneAlertEvent{Runner: ada.ID, Hunter: eve.ID, Team: eve.HunterTeam, At: alerted}}
	if got, err := l.TimelineFor(watcher, alerted); err != nil || !slices.Equal(got, want) {
		t.Fatalf("timeline = %+v, %v; want %+v", got, err, want)
	}
}

func TestTimelineIsHiddenFromActivePlayers(t *testing.T) {
	l, ada, bob, _, _, running := jokerLobby(t)
	var rule *RuleError
	for _, viewer := range []Player{ada, bob} {
		if _, err := l.TimelineFor(viewer, running); !errors.As(err, &rule) {
			t.Fatalf("%s got the timeline: %v", viewer.Name, err)
		}
	}
}

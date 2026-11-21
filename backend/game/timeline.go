package game

import (
	"slices"
	"time"
)

// Event is one timeline entry. It is sealed: the only variants are
// StartEvent, HunterJokerEvent, RunnerJokerEvent, TrapTriggeredEvent,
// ZoneAlertEvent, CatchEvent, FinishEvent and EndEvent, each with exactly the
// fields it needs. Events never carry coordinates, cells or what a joker showed.
type Event interface {
	OccurredAt() time.Time
	event()
}

// StartEvent is the reached scheduled start.
type StartEvent struct{ At time.Time }

// HunterJokerEvent is a joker used by a hunter team.
type HunterJokerEvent struct {
	Team  HunterTeamName
	Joker JokerKind
	At    time.Time
}

// RunnerJokerEvent is a joker used by a runner, personal or for the runner team.
type RunnerJokerEvent struct {
	Runner PlayerID
	Joker  JokerKind
	At     time.Time
}

// TrapTriggeredEvent is a hunter team's trap triggered by a runner, without
// saying which runner or where.
type TrapTriggeredEvent struct {
	Team HunterTeamName
	At   time.Time
}

// ZoneAlertEvent is a runner alerted about a hunter in their cell, without the cell.
type ZoneAlertEvent struct {
	Runner PlayerID
	Hunter PlayerID
	Team   HunterTeamName
	At     time.Time
}

// CatchEvent is a runner's catch.
type CatchEvent struct {
	Runner PlayerID
	At     time.Time
}

// FinishEvent is a runner's arrival at the destination.
type FinishEvent struct {
	Runner PlayerID
	At     time.Time
}

// EndEvent is the game end after the last active runner's outcome.
type EndEvent struct{ At time.Time }

func (e StartEvent) OccurredAt() time.Time         { return e.At }
func (e HunterJokerEvent) OccurredAt() time.Time   { return e.At }
func (e RunnerJokerEvent) OccurredAt() time.Time   { return e.At }
func (e TrapTriggeredEvent) OccurredAt() time.Time { return e.At }
func (e ZoneAlertEvent) OccurredAt() time.Time     { return e.At }
func (e CatchEvent) OccurredAt() time.Time         { return e.At }
func (e FinishEvent) OccurredAt() time.Time        { return e.At }
func (e EndEvent) OccurredAt() time.Time           { return e.At }

func (StartEvent) event()         {}
func (HunterJokerEvent) event()   {}
func (RunnerJokerEvent) event()   {}
func (TrapTriggeredEvent) event() {}
func (ZoneAlertEvent) event()     {}
func (CatchEvent) event()         {}
func (FinishEvent) event()        {}
func (EndEvent) event()           {}

// TimelineFor returns the game's events up to now, oldest first, to a
// spectator. Active players must not learn about opponents' jokers or
// outcomes before their reveal, so they get no timeline.
func (l Lobby) TimelineFor(viewer Player, now time.Time) ([]Event, error) {
	if viewer.Role() != RoleSpectator {
		return nil, &RuleError{Message: "only spectators can see the timeline"}
	}
	return l.timeline(now), nil
}

// timeline derives the events from the stored game state; there is no event
// table that could drift from it. A scheduled start becomes an event only once
// reached, since the countdown can still be cancelled before.
func (l Lobby) timeline(now time.Time) []Event {
	var events []Event
	if !l.Game.StartsAt.IsZero() {
		events = append(events, StartEvent{At: l.Game.StartsAt})
	}
	for _, use := range l.Jokers {
		switch u := use.(type) {
		case hunterJokerUse:
			events = append(events, HunterJokerEvent{Team: u.team(), Joker: u.Kind(), At: u.UsedAt()})
			if trap, ok := u.(TrapUse); ok && !trap.TriggeredAt.IsZero() {
				events = append(events, TrapTriggeredEvent{Team: trap.Team, At: trap.TriggeredAt})
			}
		case runnerJokerUse:
			events = append(events, RunnerJokerEvent{Runner: u.runner(), Joker: u.Kind(), At: u.UsedAt()})
		}
	}
	for _, a := range l.ZoneAlerts {
		hunter, _ := l.Player(a.Hunter)
		events = append(events, ZoneAlertEvent{Runner: a.Runner, Hunter: a.Hunter, Team: hunter.HunterTeam, At: a.At})
	}
	for _, p := range l.Players {
		switch p.Runner.Status() {
		case RunnerCaught:
			events = append(events, CatchEvent{Runner: p.ID, At: p.Runner.EndedAt()})
		case RunnerFinished:
			events = append(events, FinishEvent{Runner: p.ID, At: p.Runner.EndedAt()})
		}
	}
	if !l.Game.EndedAt.IsZero() {
		events = append(events, EndEvent{At: l.Game.EndedAt})
	}
	events = slices.DeleteFunc(events, func(e Event) bool { return e.OccurredAt().After(now) })
	// Stable, so the last outcome stays before the game end at the same instant.
	slices.SortStableFunc(events, func(a, b Event) int { return a.OccurredAt().Compare(b.OccurredAt()) })
	return events
}

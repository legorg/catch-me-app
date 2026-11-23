package game

import (
	"time"

	"catch-me/backend/geo"
)

// JokerTrap lets a hunter team place a trap that a runner can walk into, once.
const JokerTrap JokerKind = "trap"

const (
	// TrapActivationDelay is how long after placement a trap starts to trigger.
	TrapActivationDelay = 20 * time.Minute
	// TrapRadiusMeters is how close a runner's report must be to trigger a trap.
	TrapRadiusMeters = 100
)

// Trap chooses the trap joker with the point it is placed at.
type Trap struct{ Point Coordinates }

func (Trap) Kind() JokerKind { return JokerTrap }

func (c Trap) apply(l Lobby, me Player, now time.Time) (JokerUse, error) {
	if _, inside := l.Game.Playground.CellAt(c.Point); !inside {
		return nil, &ValidationError{Message: "the point must lie inside the playground"}
	}
	return TrapUse{Team: me.HunterTeam, At: now, Point: c.Point}, nil
}

// TrapUse is a placed trap. It triggers when a runner's report arrives near it
// (see triggerTraps); it never holds which runner triggered it or where they were.
type TrapUse struct {
	Team  HunterTeamName
	At    time.Time
	Point Coordinates
	// TriggeredAt is zero until a runner's report after ActiveAt lay within
	// TrapRadiusMeters; then the time of the first such report.
	TriggeredAt time.Time
}

var _ hunterJokerUse = TrapUse{}

func (TrapUse) Kind() JokerKind        { return JokerTrap }
func (u TrapUse) UsedAt() time.Time    { return u.At }
func (TrapUse) jokerUse()              {}
func (u TrapUse) team() HunterTeamName { return u.Team }

// ActiveAt is when the trap starts to trigger.
func (u TrapUse) ActiveAt() time.Time { return u.At.Add(TrapActivationDelay) }

// TrapsForSpectators are every hunter team's traps in stored order, only while
// the game runs, like the spectators' live feed. Players never get other teams' traps.
func (l Lobby) TrapsForSpectators(now time.Time) []TrapUse {
	if l.Game.Status(now) != StatusRunning {
		return nil
	}
	var traps []TrapUse
	for _, use := range l.Jokers {
		if trap, ok := use.(TrapUse); ok {
			traps = append(traps, trap)
		}
	}
	return traps
}

// triggerTraps triggers every trap the runner's new report lies near, if the
// trap is active and has not triggered yet. Reports are checked as they arrive,
// so the first one near the trap triggers it.
func (l *Lobby) triggerTraps(reporter Player) {
	if reporter.Team != Runners {
		return
	}
	for i, use := range l.Jokers {
		trap, ok := use.(TrapUse)
		if !ok || !trap.TriggeredAt.IsZero() || !trap.triggeredBy(reporter.Position) {
			continue
		}
		trap.TriggeredAt = reporter.Position.RecordedAt
		l.Jokers[i] = trap
	}
}

// triggeredBy reports whether a runner's report triggers the trap: made at or
// after its activation, within TrapRadiusMeters of its point.
func (u TrapUse) triggeredBy(report Position) bool {
	c := report.Coordinates
	return !report.RecordedAt.Before(u.ActiveAt()) &&
		geo.DistanceMeters(u.Point.latitude, u.Point.longitude, c.latitude, c.longitude) <= TrapRadiusMeters
}

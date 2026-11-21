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

// TrapUse is a placed trap. Like reveals, whether it triggered is derived from
// the runners' reports when loading; it never holds which runner triggered it
// or where they were.
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

// TrapSearchBounds contains every point within TrapRadiusMeters of the trap,
// so storage can preselect the reports that might trigger it.
func TrapSearchBounds(point Coordinates) geo.Bounds {
	return geo.BoundsAround(point.latitude, point.longitude, TrapRadiusMeters)
}

// TrapTriggeredAt is the time of the earliest report at or after activeAt
// within TrapRadiusMeters of the point, or zero if there is none. Reports are
// the runners' reports while they were active; only reports made after the
// activation count, never an earlier one.
func TrapTriggeredAt(point Coordinates, activeAt time.Time, reports []JokerPosition) time.Time {
	var first time.Time
	for _, r := range reports {
		at, c := r.Position.RecordedAt, r.Position.Coordinates
		if at.Before(activeAt) || (!first.IsZero() && !at.Before(first)) {
			continue
		}
		if geo.DistanceMeters(point.latitude, point.longitude, c.latitude, c.longitude) <= TrapRadiusMeters {
			first = at
		}
	}
	return first
}

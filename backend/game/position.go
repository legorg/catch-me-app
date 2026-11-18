package game

import (
	"time"

	"catch-me/backend/geo"
)

const (
	maxAccuracyMeters = 10000
	minRevealMinutes  = 1
	maxRevealMinutes  = 60
	// Start and finish share the same report freshness and landmark radius.
	startPositionMaxAge = 30 * time.Second
	startRadiusMeters   = 50
	// PositionRetention is how long reported positions and sketches are kept after
	// a game's start (or creation, if it never started), for a replay after the game.
	PositionRetention = 7 * 24 * time.Hour
)

// Position is a player's latest reported location. RecordedAt is the server
// time of the report, not the phone's clock.
type Position struct {
	Coordinates    Coordinates
	AccuracyMeters float64
	RecordedAt     time.Time
}

func NewPosition(coordinates Coordinates, accuracyMeters float64, recordedAt time.Time) (Position, error) {
	if !(accuracyMeters >= 0 && accuracyMeters <= maxAccuracyMeters) {
		return Position{}, &ValidationError{Message: "accuracy must be between 0 and 10000 meters"}
	}
	return Position{Coordinates: coordinates, AccuracyMeters: accuracyMeters, RecordedAt: recordedAt}, nil
}

func (p Position) IsZero() bool { return p.RecordedAt.IsZero() }

// Reveal is where a runner really was at a reveal time: their last report
// before it. Hunters never get it directly; they get a HunterReveal.
type Reveal struct {
	Position Position
	At       time.Time
}

func (r Reveal) IsZero() bool { return r.At.IsZero() }

// HunterReveal is what hunters see of a runner at a reveal time, after the
// runner's jokers are applied. It is sealed: PositionReveal, ZoneReveal, or
// OutsideReveal. Only PositionReveal carries coordinates, so a reveal that
// shows only a zone cannot leak the real position.
type HunterReveal interface {
	RevealedAt() time.Time
	hunterReveal()
}

// PositionReveal shows a position: the real one, or the point of a fakePosition joker.
type PositionReveal struct {
	At       time.Time
	Position Position
}

// ZoneReveal shows only the cell the runner was in (zoneOnly joker).
type ZoneReveal struct {
	At   time.Time
	Cell Cell
}

// OutsideReveal is a zoneOnly reveal of a runner outside the playground: no cell, no position.
type OutsideReveal struct{ At time.Time }

func (r PositionReveal) RevealedAt() time.Time { return r.At }
func (r ZoneReveal) RevealedAt() time.Time     { return r.At }
func (r OutsideReveal) RevealedAt() time.Time  { return r.At }

func (PositionReveal) hunterReveal() {}
func (ZoneReveal) hunterReveal()     {}
func (OutsideReveal) hunterReveal()  {}

// RevealInterval is how often hunters receive the runners' positions, in whole minutes.
type RevealInterval struct{ minutes int }

func DefaultRevealInterval() RevealInterval { return RevealInterval{minutes: 30} }

func ParseRevealInterval(minutes int) (RevealInterval, error) {
	if minutes < minRevealMinutes || minutes > maxRevealMinutes {
		return RevealInterval{}, &ValidationError{Message: "reveal interval must be 1 to 60 minutes"}
	}
	return RevealInterval{minutes: minutes}, nil
}

func (r RevealInterval) Minutes() int            { return r.minutes }
func (r RevealInterval) Duration() time.Duration { return time.Duration(r.minutes) * time.Minute }

// CellAt returns the grid cell containing the coordinates, if inside the playground.
func (p Playground) CellAt(c Coordinates) (Cell, bool) {
	row, column, ok := geo.Bounds(p.bounds).CellAt(p.gridSize.cells, c.latitude, c.longitude)
	return Cell{column: column, row: row}, ok
}

// latestReveal is the most recent reveal time, or zero before the first one.
// Reveals happen every RevealInterval after the start.
func (g Game) latestReveal(now time.Time) time.Time {
	if g.Status(now) != StatusRunning {
		return time.Time{}
	}
	interval := g.RevealInterval.Duration()
	count := now.Sub(g.StartsAt) / interval
	if count < 1 {
		return time.Time{}
	}
	return g.StartsAt.Add(count * interval)
}

// RevealTimes are all reveal times up to now, oldest first; empty before the first one.
func (g Game) RevealTimes(now time.Time) []time.Time {
	latest := g.latestReveal(now)
	if latest.IsZero() {
		return nil
	}
	interval := g.RevealInterval.Duration()
	count := int(latest.Sub(g.StartsAt) / interval)
	times := make([]time.Time, count)
	for i := range times {
		times[i] = g.StartsAt.Add(time.Duration(i+1) * interval)
	}
	return times
}

// NextReveal is the next reveal time while running, otherwise zero.
func (g Game) NextReveal(now time.Time) time.Time {
	if g.Status(now) != StatusRunning {
		return time.Time{}
	}
	interval := g.RevealInterval.Duration()
	return g.StartsAt.Add((now.Sub(g.StartsAt)/interval + 1) * interval)
}

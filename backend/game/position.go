package game

import (
	"time"

	"catch-me/backend/geo"
)

const (
	maxAccuracyMeters = 10000
	// RevealInterval is how often hunters receive the runners' positions.
	RevealInterval = 30 * time.Second
	// startPositionMaxAge is how recent a runner's position must be to prove they
	// are at the start of the route.
	startPositionMaxAge = 30 * time.Second
	// startRadiusMeters is how close to the route's start a runner must be to get ready.
	startRadiusMeters = 50
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

// Reveal is a runner's position as shown to hunters at a reveal time.
type Reveal struct {
	Position Position
	At       time.Time
}

func (r Reveal) IsZero() bool { return r.At.IsZero() }

// CellAt returns the grid cell containing the coordinates, if inside the playground.
func (p Playground) CellAt(c Coordinates) (Cell, bool) {
	row, column, ok := geo.Bounds(p.bounds).CellAt(gridSize, c.latitude, c.longitude)
	return Cell{column: column, row: row}, ok
}

// latestReveal is the most recent reveal time, or zero before the first one.
// Reveals happen every RevealInterval after the start.
func (g Game) latestReveal(now time.Time) time.Time {
	if g.Status(now) != StatusRunning {
		return time.Time{}
	}
	count := now.Sub(g.StartsAt) / RevealInterval
	if count < 1 {
		return time.Time{}
	}
	return g.StartsAt.Add(count * RevealInterval)
}

// RevealTimes are all reveal times up to now, oldest first; empty before the first one.
func (g Game) RevealTimes(now time.Time) []time.Time {
	latest := g.latestReveal(now)
	if latest.IsZero() {
		return nil
	}
	count := int(latest.Sub(g.StartsAt) / RevealInterval)
	times := make([]time.Time, count)
	for i := range times {
		times[i] = g.StartsAt.Add(time.Duration(i+1) * RevealInterval)
	}
	return times
}

// NextReveal is the next reveal time while running, otherwise zero.
func (g Game) NextReveal(now time.Time) time.Time {
	if g.Status(now) != StatusRunning {
		return time.Time{}
	}
	return g.StartsAt.Add((now.Sub(g.StartsAt)/RevealInterval + 1) * RevealInterval)
}

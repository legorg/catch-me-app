package game

import (
	"time"
)

const (
	MaxAccuracyMeters = 10000
	// RevealInterval is how often hunters receive the runners' positions.
	RevealInterval = 5 * time.Minute
	// StartCellMaxAge is how recent a runner's position must be to prove they
	// stand in their start cell.
	StartCellMaxAge = 30 * time.Second
	// HistoryRetention is how long every reported position is kept after a
	// game's start (or creation, if it never started) for a later replay.
	HistoryRetention = 7 * 24 * time.Hour
)

// Position is a player's latest reported location. RecordedAt is the server
// time of the report, not the phone's clock.
type Position struct {
	Coordinates    Coordinates
	AccuracyMeters float64
	RecordedAt     time.Time
}

func NewPosition(coordinates Coordinates, accuracyMeters float64, recordedAt time.Time) (Position, error) {
	if !(accuracyMeters >= 0 && accuracyMeters <= MaxAccuracyMeters) {
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
	b := p.bounds
	if !(c.latitude <= b.North && c.latitude >= b.South && c.longitude >= b.West && c.longitude <= b.East) {
		return Cell{}, false
	}
	row := int((b.North - c.latitude) / (b.North - b.South) * GridSize)
	column := int((c.longitude - b.West) / (b.East - b.West) * GridSize)
	return Cell{column: min(column, GridSize-1), row: min(row, GridSize-1)}, true
}

// CellCenter is the center of a grid cell.
func (p Playground) CellCenter(c Cell) Coordinates {
	b := p.bounds
	return Coordinates{
		latitude:  b.North - (b.North-b.South)*(float64(c.row)+0.5)/GridSize,
		longitude: b.West + (b.East-b.West)*(float64(c.column)+0.5)/GridSize,
	}
}

// LatestReveal is the most recent reveal time, or zero before the first one.
// Reveals happen every RevealInterval after the start.
func (g Game) LatestReveal(now time.Time) time.Time {
	if g.Status(now) != StatusRunning {
		return time.Time{}
	}
	count := now.Sub(g.StartsAt) / RevealInterval
	if count < 1 {
		return time.Time{}
	}
	return g.StartsAt.Add(count * RevealInterval)
}

// NextReveal is the next reveal time while running, otherwise zero.
func (g Game) NextReveal(now time.Time) time.Time {
	if g.Status(now) != StatusRunning {
		return time.Time{}
	}
	return g.StartsAt.Add((now.Sub(g.StartsAt)/RevealInterval + 1) * RevealInterval)
}

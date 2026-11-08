package game

import (
	"math"
)

const (
	GridSize           = 6
	MaxLatitude        = 80
	MaxLongitude       = 180
	MinCellSizeMeters  = 100
	MaxCellSizeMeters  = 5000
	CellSizeStepMeters = 100
)

const earthRadiusMeters = 6371008.8

type Coordinates struct{ latitude, longitude float64 }

func ParseCoordinates(latitude, longitude float64) (Coordinates, error) {
	// Written as a negated range check so NaN is rejected as well.
	if !(math.Abs(latitude) <= MaxLatitude && math.Abs(longitude) <= MaxLongitude) {
		return Coordinates{}, &ValidationError{Message: "latitude must be between -80 and 80; longitude between -180 and 180"}
	}
	return Coordinates{latitude: latitude, longitude: longitude}, nil
}

func (c Coordinates) Latitude() float64  { return c.latitude }
func (c Coordinates) Longitude() float64 { return c.longitude }

type CellSize struct{ meters int }

func DefaultCellSize() CellSize { return CellSize{meters: 2000} }

func ParseCellSize(meters int) (CellSize, error) {
	if meters < MinCellSizeMeters || meters > MaxCellSizeMeters || meters%CellSizeStepMeters != 0 {
		return CellSize{}, &ValidationError{Message: "cell size must be 100 to 5000 meters in steps of 100 meters"}
	}
	return CellSize{meters: meters}, nil
}

func (s CellSize) Meters() int { return s.meters }

type Bounds struct{ North, South, East, West float64 }

type Playground struct {
	center   Coordinates
	cellSize CellSize
	bounds   Bounds
}

// NewPlayground uses a local spherical approximation: meter-sized cells,
// north-aligned. Limit latitude and reject date-line crossings to keep the
// first version's rectangular bounds unambiguous.
func NewPlayground(center Coordinates, cellSize CellSize) (Playground, error) {
	halfSize := float64(GridSize*cellSize.meters) / 2
	latDelta := halfSize / earthRadiusMeters * 180 / math.Pi
	lonDelta := latDelta / math.Cos(center.latitude*math.Pi/180)
	bounds := Bounds{
		North: center.latitude + latDelta, South: center.latitude - latDelta,
		East: center.longitude + lonDelta, West: center.longitude - lonDelta,
	}
	if bounds.West < -MaxLongitude || bounds.East > MaxLongitude {
		return Playground{}, &ValidationError{Message: "playground must not cross the international date line"}
	}
	return Playground{center: center, cellSize: cellSize, bounds: bounds}, nil
}

func (p Playground) Center() Coordinates { return p.center }
func (p Playground) CellSize() CellSize  { return p.cellSize }
func (p Playground) Bounds() Bounds      { return p.bounds }
func (p Playground) Rows() int           { return GridSize }
func (p Playground) Columns() int        { return GridSize }

package game

import "catch-me/backend/geo"

const (
	gridSize           = 6
	minCellSizeMeters  = 100
	maxCellSizeMeters  = 5000
	cellSizeStepMeters = 100
)

type Coordinates struct{ latitude, longitude float64 }

func ParseCoordinates(latitude, longitude float64) (Coordinates, error) {
	if !geo.InRange(latitude, longitude) {
		return Coordinates{}, &ValidationError{Message: "latitude must be between -80 and 80; longitude between -180 and 180"}
	}
	return Coordinates{latitude: latitude, longitude: longitude}, nil
}

func (c Coordinates) Latitude() float64  { return c.latitude }
func (c Coordinates) Longitude() float64 { return c.longitude }

type CellSize struct{ meters int }

func DefaultCellSize() CellSize { return CellSize{meters: 2000} }

func ParseCellSize(meters int) (CellSize, error) {
	if meters < minCellSizeMeters || meters > maxCellSizeMeters || meters%cellSizeStepMeters != 0 {
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
	halfSize := float64(gridSize*cellSize.meters) / 2
	bounds := Bounds(geo.BoundsAround(center.latitude, center.longitude, halfSize))
	if geo.Bounds(bounds).CrossesDateLine() {
		return Playground{}, &ValidationError{Message: "playground must not cross the international date line"}
	}
	return Playground{center: center, cellSize: cellSize, bounds: bounds}, nil
}

func (p Playground) Center() Coordinates { return p.center }
func (p Playground) CellSize() CellSize  { return p.cellSize }
func (p Playground) Bounds() Bounds      { return p.bounds }
func (p Playground) Rows() int           { return gridSize }
func (p Playground) Columns() int        { return gridSize }

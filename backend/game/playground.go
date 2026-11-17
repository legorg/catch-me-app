package game

import "catch-me/backend/geo"

const (
	minGridSize        = 3
	maxGridSize        = 12
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

// GridSize is how many cells the square grid has per side.
type GridSize struct{ cells int }

func DefaultGridSize() GridSize { return GridSize{cells: 6} }

func ParseGridSize(cells int) (GridSize, error) {
	if cells < minGridSize || cells > maxGridSize {
		return GridSize{}, &ValidationError{Message: "grid size must be 3 to 12 cells per side"}
	}
	return GridSize{cells: cells}, nil
}

func (s GridSize) Cells() int { return s.cells }

type Bounds struct{ North, South, East, West float64 }

type Playground struct {
	center   Coordinates
	cellSize CellSize
	gridSize GridSize
	bounds   Bounds
}

// NewPlayground uses a local spherical approximation: meter-sized cells,
// north-aligned. Limit latitude and reject date-line crossings to keep the
// first version's rectangular bounds unambiguous.
func NewPlayground(center Coordinates, cellSize CellSize, gridSize GridSize) (Playground, error) {
	halfSize := float64(gridSize.cells*cellSize.meters) / 2
	bounds := Bounds(geo.BoundsAround(center.latitude, center.longitude, halfSize))
	if geo.Bounds(bounds).CrossesDateLine() {
		return Playground{}, &ValidationError{Message: "playground must not cross the international date line"}
	}
	return Playground{center: center, cellSize: cellSize, gridSize: gridSize, bounds: bounds}, nil
}

func (p Playground) Center() Coordinates { return p.center }
func (p Playground) CellSize() CellSize  { return p.cellSize }
func (p Playground) GridSize() GridSize  { return p.gridSize }
func (p Playground) Bounds() Bounds      { return p.bounds }
func (p Playground) Rows() int           { return p.gridSize.cells }
func (p Playground) Columns() int        { return p.gridSize.cells }

// HasCell reports whether the cell is part of this playground's grid.
func (p Playground) HasCell(c Cell) bool {
	return c.column < p.gridSize.cells && c.row < p.gridSize.cells
}

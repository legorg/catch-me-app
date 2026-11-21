package game

import (
	"math"

	"catch-me/backend/geo"
)

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

// Rotation is how far the grid is turned clockwise around its center, in
// whole degrees: 0 is north-aligned (row 1 in the north), 90 puts row 1 in
// the east.
type Rotation struct{ degrees int }

func ParseRotation(degrees int) (Rotation, error) {
	if degrees < 0 || degrees >= 360 {
		return Rotation{}, &ValidationError{Message: "rotation must be 0 to 359 degrees"}
	}
	return Rotation{degrees: degrees}, nil
}

func (r Rotation) Degrees() int { return r.degrees }

type Bounds struct{ North, South, East, West float64 }

type Playground struct {
	center   Coordinates
	cellSize CellSize
	gridSize GridSize
	rotation Rotation
	// bounds is the grid's square before rotation.
	bounds Bounds
}

// NewPlayground creates a north-aligned playground.
func NewPlayground(center Coordinates, cellSize CellSize, gridSize GridSize) (Playground, error) {
	return NewRotatedPlayground(center, cellSize, gridSize, Rotation{})
}

// NewRotatedPlayground uses a local spherical approximation: meter-sized
// cells, turned by rotation around the center. Limit latitude and reject
// date-line crossings to keep the first version's geometry unambiguous.
func NewRotatedPlayground(center Coordinates, cellSize CellSize, gridSize GridSize, rotation Rotation) (Playground, error) {
	halfSize := float64(gridSize.cells*cellSize.meters) / 2
	p := Playground{
		center: center, cellSize: cellSize, gridSize: gridSize, rotation: rotation,
		bounds: Bounds(geo.BoundsAround(center.latitude, center.longitude, halfSize)),
	}
	if geo.Bounds(p.rotatedExtent(p.bounds)).CrossesDateLine() {
		return Playground{}, &ValidationError{Message: "playground must not cross the international date line"}
	}
	return p, nil
}

func (p Playground) Center() Coordinates { return p.center }
func (p Playground) CellSize() CellSize  { return p.cellSize }
func (p Playground) GridSize() GridSize  { return p.gridSize }
func (p Playground) Rotation() Rotation  { return p.rotation }

// Bounds is the grid's square before rotation; for a north-aligned grid it is
// the playground itself.
func (p Playground) Bounds() Bounds { return p.bounds }
func (p Playground) Rows() int      { return p.gridSize.cells }
func (p Playground) Columns() int   { return p.gridSize.cells }

// rotate turns a point of the unrotated grid clockwise around the center onto
// the map; unrotate turns a map point back. Both use the same local flat
// approximation as the bounds, so a zero rotation returns the point unchanged.
func (p Playground) rotate(c Coordinates) Coordinates { return p.turn(c, p.rotation.degrees) }

func (p Playground) unrotate(c Coordinates) Coordinates { return p.turn(c, -p.rotation.degrees) }

func (p Playground) turn(c Coordinates, clockwiseDegrees int) Coordinates {
	if clockwiseDegrees%360 == 0 {
		return c
	}
	scale := math.Cos(p.center.latitude * math.Pi / 180)
	east := (c.longitude - p.center.longitude) * scale
	north := c.latitude - p.center.latitude
	sin, cos := math.Sincos(float64(clockwiseDegrees) * math.Pi / 180)
	return Coordinates{
		latitude:  p.center.latitude - east*sin + north*cos,
		longitude: p.center.longitude + (east*cos+north*sin)/scale,
	}
}

// rotatedExtent is the smallest north-aligned rectangle containing the
// unrotated rectangle b after rotation.
func (p Playground) rotatedExtent(b Bounds) Bounds {
	extent := Bounds{North: math.Inf(-1), South: math.Inf(1), East: math.Inf(-1), West: math.Inf(1)}
	for _, corner := range []Coordinates{{b.North, b.West}, {b.North, b.East}, {b.South, b.East}, {b.South, b.West}} {
		c := p.rotate(corner)
		extent.North, extent.South = max(extent.North, c.latitude), min(extent.South, c.latitude)
		extent.East, extent.West = max(extent.East, c.longitude), min(extent.West, c.longitude)
	}
	return extent
}

// HasCell reports whether the cell is part of this playground's grid.
func (p Playground) HasCell(c Cell) bool {
	return c.column < p.gridSize.cells && c.row < p.gridSize.cells
}

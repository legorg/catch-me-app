package game

import "catch-me/backend/geo"

// Landmark is a named place; Title is empty for a plain coordinate.
type Landmark struct {
	Title       string
	Coordinates Coordinates
}

// Route is the game's course: from a landmark at the southern edge of the
// playground to one at the northern edge.
type Route struct{ Start, Stop Landmark }

// DistanceMeters is the straight-line distance between start and stop.
func (r Route) DistanceMeters() float64 {
	return r.Start.Coordinates.DistanceMeters(r.Stop.Coordinates)
}

// DistanceMeters is the great-circle (haversine) distance to other.
func (c Coordinates) DistanceMeters(other Coordinates) float64 {
	return geo.DistanceMeters(c.latitude, c.longitude, other.latitude, other.longitude)
}

// Contains reports whether the point lies inside the bounds.
func (b Bounds) Contains(c Coordinates) bool {
	return geo.Bounds(b).Contains(c.latitude, c.longitude)
}

// CellBounds is the area of one grid cell.
func (p Playground) CellBounds(c Cell) Bounds {
	return Bounds(geo.Bounds(p.bounds).Cell(gridSize, c.row, c.column))
}

// SouthEdgeCells are the cells of the bottom row of the grid, NorthEdgeCells
// those of the top row.
func (p Playground) SouthEdgeCells() []Cell { return edgeCells(gridSize - 1) }
func (p Playground) NorthEdgeCells() []Cell { return edgeCells(0) }

func edgeCells(row int) []Cell {
	cells := make([]Cell, gridSize)
	for column := range cells {
		cells[column] = Cell{column: column, row: row}
	}
	return cells
}

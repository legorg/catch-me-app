package game

import "math"

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
	lat1, lat2 := c.latitude*math.Pi/180, other.latitude*math.Pi/180
	dLat := lat2 - lat1
	dLon := (other.longitude - c.longitude) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(a))
}

// Contains reports whether the point lies inside the bounds.
func (b Bounds) Contains(c Coordinates) bool {
	return c.latitude >= b.South && c.latitude <= b.North && c.longitude >= b.West && c.longitude <= b.East
}

// Center is the middle of the bounds.
func (b Bounds) Center() Coordinates {
	return Coordinates{latitude: (b.North + b.South) / 2, longitude: (b.East + b.West) / 2}
}

// CellBounds is the area of one grid cell.
func (p Playground) CellBounds(c Cell) Bounds {
	b := p.bounds
	height := (b.North - b.South) / GridSize
	width := (b.East - b.West) / GridSize
	north := b.North - float64(c.row)*height
	west := b.West + float64(c.column)*width
	return Bounds{North: north, South: north - height, West: west, East: west + width}
}

// SouthEdgeCells are the cells of the bottom row of the grid, NorthEdgeCells
// those of the top row.
func (p Playground) SouthEdgeCells() []Cell { return edgeCells(GridSize - 1) }
func (p Playground) NorthEdgeCells() []Cell { return edgeCells(0) }

func edgeCells(row int) []Cell {
	cells := make([]Cell, GridSize)
	for column := range cells {
		cells[column] = Cell{column: column, row: row}
	}
	return cells
}

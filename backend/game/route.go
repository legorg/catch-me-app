package game

import (
	"fmt"
	"strconv"
)

// MaxRouteRegenerations is how often the runners together may replace the
// game's route.
const MaxRouteRegenerations = 2

// Cell is one grid cell: rows A, B, … from north to south, columns 1, 2, … from
// west to east. Whether it lies in a game's grid is Playground.HasCell.
type Cell struct{ column, row int }

// ParseCell accepts any string-based type, e.g. a wire-format cell: a column
// A-L followed by a column 1-12, the largest grid.
func ParseCell[S ~string](value S) (Cell, error) {
	label := string(value)
	invalid := &ValidationError{Message: "cell must be a row A-L followed by a column 1-12"}
	if len(label) < 2 || label[0] < 'A' || label[0] >= 'A'+maxGridSize {
		return Cell{}, invalid
	}
	// The round trip rejects forms like "01" or "+1".
	column, err := strconv.Atoi(label[1:])
	if err != nil || column < 1 || column > maxGridSize || strconv.Itoa(column) != label[1:] {
		return Cell{}, invalid
	}
	return Cell{column: column - 1, row: int(label[0] - 'A')}, nil
}

func (c Cell) String() string { return fmt.Sprintf("%c%d", 'A'+c.row, c.column+1) }

// ParseRouteRegenerations accepts how often a game's route was replaced.
func ParseRouteRegenerations(used int) (int, error) {
	if used < 0 || used > MaxRouteRegenerations {
		return 0, &ValidationError{Message: "route regenerations out of range"}
	}
	return used, nil
}

// RouteRegenerationsLeft is how often the runners may still replace the
// route; never for a custom route.
func (g Game) RouteRegenerationsLeft() int {
	if g.CustomRoute {
		return 0
	}
	return MaxRouteRegenerations - g.RouteRegenerations
}

// NewCustomRoute is a route chosen by the game creator: plain, untitled
// points, the start in the southern edge row and the stop in the northern
// edge row of the playground, like a generated route.
func NewCustomRoute(p Playground, start, stop Coordinates) (Route, error) {
	if cell, ok := p.CellAt(start); !ok || cell.row != p.gridSize.cells-1 {
		return Route{}, &ValidationError{Message: "route start must lie in the southern edge row of the playground"}
	}
	if cell, ok := p.CellAt(stop); !ok || cell.row != 0 {
		return Route{}, &ValidationError{Message: "route stop must lie in the northern edge row of the playground"}
	}
	return Route{Start: Landmark{Coordinates: start}, Stop: Landmark{Coordinates: stop}}, nil
}

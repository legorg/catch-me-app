package game

import (
	"fmt"
	"strconv"
)

// MaxRouteRegenerations is how often the runners together may replace the
// game's route.
const MaxRouteRegenerations = 2

// Cell is one grid cell: columns A, B, … from west to east, rows 1, 2, … from
// north to south. Whether it lies in a game's grid is Playground.HasCell.
type Cell struct{ column, row int }

// ParseCell accepts any string-based type, e.g. a wire-format cell: a column
// A-L followed by a row 1-12, the largest grid.
func ParseCell[S ~string](value S) (Cell, error) {
	label := string(value)
	invalid := &ValidationError{Message: "cell must be a column A-L followed by a row 1-12"}
	if len(label) < 2 || label[0] < 'A' || label[0] >= 'A'+maxGridSize {
		return Cell{}, invalid
	}
	// The round trip rejects forms like "01" or "+1".
	row, err := strconv.Atoi(label[1:])
	if err != nil || row < 1 || row > maxGridSize || strconv.Itoa(row) != label[1:] {
		return Cell{}, invalid
	}
	return Cell{column: int(label[0] - 'A'), row: row - 1}, nil
}

func (c Cell) String() string { return fmt.Sprintf("%c%d", 'A'+c.column, c.row+1) }

// ParseRouteRegenerations accepts how often a game's route was replaced.
func ParseRouteRegenerations(used int) (int, error) {
	if used < 0 || used > MaxRouteRegenerations {
		return 0, &ValidationError{Message: "route regenerations out of range"}
	}
	return used, nil
}

// RouteRegenerationsLeft is how often the runners may still replace the route.
func (g Game) RouteRegenerationsLeft() int { return MaxRouteRegenerations - g.RouteRegenerations }

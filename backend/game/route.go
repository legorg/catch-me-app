package game

import "fmt"

// MaxRouteRegenerations is how often the runners together may replace the
// game's route.
const MaxRouteRegenerations = 2

// Cell is one grid cell: columns A–F from west to east, rows 1–6 from north to south.
type Cell struct{ column, row int }

// ParseCell accepts any string-based type, e.g. a wire-format cell.
func ParseCell[S ~string](value S) (Cell, error) {
	label := string(value)
	if len(label) != 2 || label[0] < 'A' || label[0] >= 'A'+gridSize || label[1] < '1' || label[1] >= '1'+gridSize {
		return Cell{}, &ValidationError{Message: "cell must be a column A-F followed by a row 1-6"}
	}
	return Cell{column: int(label[0] - 'A'), row: int(label[1] - '1')}, nil
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

package game

import (
	"fmt"
	"math/rand/v2"
)

// MaxRouteRegenerations is how often a runner may replace their route.
const MaxRouteRegenerations = 2

// Cell is one grid cell: columns A–F from west to east, rows 1–6 from north to south.
type Cell struct{ column, row int }

func ParseCell(label string) (Cell, error) {
	if len(label) != 2 || label[0] < 'A' || label[0] >= 'A'+GridSize || label[1] < '1' || label[1] >= '1'+GridSize {
		return Cell{}, &ValidationError{Message: "cell must be a column A-F followed by a row 1-6"}
	}
	return Cell{column: int(label[0] - 'A'), row: int(label[1] - '1')}, nil
}

func (c Cell) String() string { return fmt.Sprintf("%c%d", 'A'+c.column, c.row+1) }

// RunnerRoute is a runner's start cell on one edge row and end cell on the
// opposite edge row.
type RunnerRoute struct {
	Start, End        Cell
	RegenerationsLeft int
}

func ParseRunnerRoute(start, end string, regenerationsLeft int) (RunnerRoute, error) {
	from, err := ParseCell(start)
	if err != nil {
		return RunnerRoute{}, err
	}
	to, err := ParseCell(end)
	if err != nil {
		return RunnerRoute{}, err
	}
	if !isEdgeRow(from.row) || to.row != GridSize-1-from.row {
		return RunnerRoute{}, &ValidationError{Message: "route must lead from one edge row to the opposite one"}
	}
	if regenerationsLeft < 0 || regenerationsLeft > MaxRouteRegenerations {
		return RunnerRoute{}, &ValidationError{Message: "route regenerations out of range"}
	}
	return RunnerRoute{Start: from, End: to, RegenerationsLeft: regenerationsLeft}, nil
}

func isEdgeRow(row int) bool { return row == 0 || row == GridSize-1 }

func randomRoute(regenerationsLeft int) RunnerRoute {
	startRow := 0
	if rand.IntN(2) == 1 {
		startRow = GridSize - 1
	}
	return RunnerRoute{
		Start:             Cell{column: rand.IntN(GridSize), row: startRow},
		End:               Cell{column: rand.IntN(GridSize), row: GridSize - 1 - startRow},
		RegenerationsLeft: regenerationsLeft,
	}
}

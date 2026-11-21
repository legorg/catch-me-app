package game

import (
	"errors"
	"math"
	"testing"

	"catch-me/backend/geo"
)

func TestPlaygroundGeometry(t *testing.T) {
	for _, lat := range []float64{-80, -52.52, 0, 52.52, 80} {
		for _, meters := range []int{100, 1500, 2000, 5000} {
			for _, cells := range []int{3, 6, 12} {
				center, err := ParseCoordinates(lat, 13.405)
				if err != nil {
					t.Fatal(err)
				}
				size, err := ParseCellSize(meters)
				if err != nil {
					t.Fatal(err)
				}
				gridSize, err := ParseGridSize(cells)
				if err != nil {
					t.Fatal(err)
				}
				p, err := NewPlayground(center, size, gridSize)
				if err != nil {
					t.Fatal(err)
				}
				b := p.Bounds()
				height := (b.North - b.South) * math.Pi / 180 * geo.EarthRadiusMeters
				width := (b.East - b.West) * math.Pi / 180 * geo.EarthRadiusMeters * math.Cos(lat*math.Pi/180)
				want := float64(cells * meters)
				if math.Abs(height-want) > 1e-6 || math.Abs(width-want) > 1e-6 || p.Rows() != cells || p.Columns() != cells {
					t.Fatalf("not a local %g m square: %+v", want, p)
				}
				for _, edgeLat := range []float64{b.North, b.South} {
					a := math.Pow(math.Cos(edgeLat*math.Pi/180)*math.Sin((b.East-b.West)*math.Pi/360), 2)
					distance := 2 * geo.EarthRadiusMeters * math.Asin(math.Sqrt(a))
					// A 60 km grid near the poles bends more (about 2.7 %) than a small one.
					if math.Abs(distance/want-1) > 0.03 {
						t.Fatalf("edge approximation too distorted: %g", distance)
					}
				}
			}
		}
	}
}

func rotatedPlayground(t *testing.T, latitude, longitude float64, cellMeters, cells, degrees int) (Playground, error) {
	t.Helper()
	center, err := ParseCoordinates(latitude, longitude)
	if err != nil {
		t.Fatal(err)
	}
	size, err := ParseCellSize(cellMeters)
	if err != nil {
		t.Fatal(err)
	}
	gridSize, err := ParseGridSize(cells)
	if err != nil {
		t.Fatal(err)
	}
	rotation, err := ParseRotation(degrees)
	if err != nil {
		t.Fatal(err)
	}
	return NewRotatedPlayground(center, size, gridSize, rotation)
}

func TestRotatedPlaygroundCells(t *testing.T) {
	for _, degrees := range []int{0, 1, 30, 45, 90, 180, 271, 359} {
		p, err := rotatedPlayground(t, 52.52, 13.405, 2000, 6, degrees)
		if err != nil {
			t.Fatal(err)
		}
		if p.Rotation().Degrees() != degrees {
			t.Fatalf("rotation = %d, want %d", p.Rotation().Degrees(), degrees)
		}
		for row := range 6 {
			for column := range 6 {
				cell := Cell{column: column, row: row}
				if got, ok := p.CellAt(cellCenter(p, cell)); !ok || got != cell {
					t.Fatalf("%d°: center of %s maps to %s (inside %v)", degrees, cell, got, ok)
				}
			}
		}
	}
}

func TestRotatedPlaygroundTurnsClockwise(t *testing.T) {
	p, err := rotatedPlayground(t, 52.52, 13.405, 2000, 6, 90)
	if err != nil {
		t.Fatal(err)
	}
	// Turned by 90°, row 1 lies in the east and column A in the north.
	a1 := cellCenter(p, Cell{})
	if a1.Latitude() <= p.Center().Latitude() || a1.Longitude() <= p.Center().Longitude() {
		t.Fatalf("A1 is not north-east of the center: %+v", a1)
	}
	east, _ := ParseCoordinates(p.Center().Latitude(), p.Bounds().East-0.001)
	if cell, ok := p.CellAt(east); !ok || cell.row != 0 {
		t.Fatalf("eastern edge maps to %s (inside %v), want row 1", cell, ok)
	}
	if got := p.CellBounds(Cell{}); math.Abs((got.North-got.South)-(p.Bounds().North-p.Bounds().South)/6) > 1e-9 {
		t.Fatalf("a quarter turn should keep the cell's extent: %+v", got)
	}
}

func TestRotatedPlaygroundCorners(t *testing.T) {
	p, err := rotatedPlayground(t, 52.52, 13.405, 2000, 6, 45)
	if err != nil {
		t.Fatal(err)
	}
	b := p.Bounds()
	// The unrotated corner lies outside a grid turned by 45°, the unrotated
	// edge's middle lies inside.
	corner, _ := ParseCoordinates(b.North, b.West)
	if cell, ok := p.CellAt(corner); ok {
		t.Fatalf("unrotated corner maps to %s", cell)
	}
	edge, _ := ParseCoordinates(b.North-0.001, p.Center().Longitude())
	if _, ok := p.CellAt(edge); !ok {
		t.Fatal("unrotated edge middle is outside")
	}
	// A cell's search bounds contain its center and are larger than the cell.
	cellBounds := p.CellBounds(Cell{})
	if !cellBounds.Contains(cellCenter(p, Cell{})) || cellBounds.North-cellBounds.South <= (b.North-b.South)/6 {
		t.Fatalf("cell bounds do not enclose the turned cell: %+v", cellBounds)
	}
}

func TestRotatedPlaygroundDateLine(t *testing.T) {
	// 3 × 100 m: 150 m from the center to an edge, about 212 m to a corner.
	longitude := 180 - 180/math.Pi*180/geo.EarthRadiusMeters
	if _, err := rotatedPlayground(t, 0, longitude, 100, 3, 0); err != nil {
		t.Fatalf("north-aligned playground rejected: %v", err)
	}
	var validation *ValidationError
	if _, err := rotatedPlayground(t, 0, longitude, 100, 3, 45); !errors.As(err, &validation) {
		t.Fatal("turned playground crossing the date line accepted")
	}
}

// rotateGrid turns the lobby's grid by degrees around the same center, keeping
// its size; it returns the north-aligned grid it replaced. The route and
// positions stay where they are.
func rotateGrid(t *testing.T, l *Lobby, degrees int) Playground {
	t.Helper()
	before := l.Game.Playground
	rotation, err := ParseRotation(degrees)
	if err != nil {
		t.Fatal(err)
	}
	l.Game.Playground, err = NewRotatedPlayground(before.Center(), before.CellSize(), before.GridSize(), rotation)
	if err != nil {
		t.Fatal(err)
	}
	return before
}

func TestRotatedCustomRoute(t *testing.T) {
	p, err := rotatedPlayground(t, 52.52, 13.405, 2000, 6, 180)
	if err != nil {
		t.Fatal(err)
	}
	// Turned by 180°, the last row lies in the north and row 1 in the south.
	start, stop := cellCenter(p, Cell{column: 2, row: 5}), cellCenter(p, Cell{column: 3})
	if start.Latitude() <= stop.Latitude() {
		t.Fatal("start is not north of the stop")
	}
	if _, err := NewCustomRoute(p, start, stop); err != nil {
		t.Fatal(err)
	}
	if _, err := NewCustomRoute(p, stop, start); err == nil {
		t.Fatal("swapped route accepted")
	}
}

func TestInvalidInput(t *testing.T) {
	var validation *ValidationError
	for _, c := range [][2]float64{{math.NaN(), 0}, {0, math.NaN()}, {0, math.Inf(1)}, {81, 0}, {-81, 0}, {0, 181}} {
		if _, err := ParseCoordinates(c[0], c[1]); !errors.As(err, &validation) {
			t.Fatalf("invalid coordinates accepted: %v", c)
		}
	}
	center, _ := ParseCoordinates(0, -180)
	if _, err := NewPlayground(center, DefaultCellSize(), DefaultGridSize()); !errors.As(err, &validation) {
		t.Fatal("date-line crossing accepted")
	}
	for _, size := range []int{-100, 0, 99, 1550, 5100} {
		if _, err := ParseCellSize(size); !errors.As(err, &validation) {
			t.Fatalf("invalid size accepted: %d", size)
		}
	}
	for _, cells := range []int{0, 2, 13} {
		if _, err := ParseGridSize(cells); !errors.As(err, &validation) {
			t.Fatalf("invalid grid size accepted: %d", cells)
		}
	}
	for _, minutes := range []int{-1, 0, 61} {
		if _, err := ParseRevealInterval(minutes); !errors.As(err, &validation) {
			t.Fatalf("invalid reveal interval accepted: %d", minutes)
		}
	}
	for _, degrees := range []int{-1, 360} {
		if _, err := ParseRotation(degrees); !errors.As(err, &validation) {
			t.Fatalf("invalid rotation accepted: %d", degrees)
		}
	}
	for _, id := range []string{"", "not-an-id", "00112233445566778899AABBCCDDEEFF", "00112233445566778899aabbccddeeff0"} {
		if _, err := ParseSessionID(id); !errors.As(err, &validation) {
			t.Fatalf("invalid session ID accepted: %q", id)
		}
	}
	if _, err := ParseSessionID(NewSessionID().String()); err != nil {
		t.Fatal("generated session ID does not parse")
	}
}

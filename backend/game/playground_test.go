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
	for _, id := range []string{"", "not-an-id", "00112233445566778899AABBCCDDEEFF", "00112233445566778899aabbccddeeff0"} {
		if _, err := ParseSessionID(id); !errors.As(err, &validation) {
			t.Fatalf("invalid session ID accepted: %q", id)
		}
	}
	if _, err := ParseSessionID(NewSessionID().String()); err != nil {
		t.Fatal("generated session ID does not parse")
	}
}

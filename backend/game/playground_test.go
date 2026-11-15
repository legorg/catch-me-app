package game

import (
	"errors"
	"math"
	"testing"
)

func TestPlaygroundGeometry(t *testing.T) {
	for _, lat := range []float64{-80, -52.52, 0, 52.52, 80} {
		for _, meters := range []int{100, 1500, 2000, 5000} {
			center, err := ParseCoordinates(lat, 13.405)
			if err != nil {
				t.Fatal(err)
			}
			size, err := ParseCellSize(meters)
			if err != nil {
				t.Fatal(err)
			}
			p, err := NewPlayground(center, size)
			if err != nil {
				t.Fatal(err)
			}
			b := p.Bounds()
			height := (b.North - b.South) * math.Pi / 180 * earthRadiusMeters
			width := (b.East - b.West) * math.Pi / 180 * earthRadiusMeters * math.Cos(lat*math.Pi/180)
			want := float64(6 * meters)
			if math.Abs(height-want) > 1e-6 || math.Abs(width-want) > 1e-6 || p.Rows() != 6 || p.Columns() != 6 {
				t.Fatalf("not a local %g m square: %+v", want, p)
			}
			for _, edgeLat := range []float64{b.North, b.South} {
				a := math.Pow(math.Cos(edgeLat*math.Pi/180)*math.Sin((b.East-b.West)*math.Pi/360), 2)
				distance := 2 * earthRadiusMeters * math.Asin(math.Sqrt(a))
				if math.Abs(distance/want-1) > 0.015 {
					t.Fatalf("edge approximation too distorted: %g", distance)
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
	if _, err := NewPlayground(center, DefaultCellSize()); !errors.As(err, &validation) {
		t.Fatal("date-line crossing accepted")
	}
	for _, size := range []int{-100, 0, 99, 1550, 5100} {
		if _, err := ParseCellSize(size); !errors.As(err, &validation) {
			t.Fatalf("invalid size accepted: %d", size)
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

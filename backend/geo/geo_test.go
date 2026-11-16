package geo

import (
	"math"
	"testing"
)

func TestDistanceMeters(t *testing.T) {
	tests := []struct {
		name                   string
		lat1, lon1, lat2, lon2 float64
		want, tolerance        float64
	}{
		{"same point", 52.5, 13.4, 52.5, 13.4, 0, 0},
		{"brandenburger tor to alexanderplatz", 52.516275, 13.377704, 52.521918, 13.413215, 2500, 100},
		{"berlin to paris", 52.52, 13.405, 48.8566, 2.3522, 877_000, 3_000},
		{"one degree along equator", 0, 0, 0, 1, math.Pi * EarthRadiusMeters / 180, 1e-6},
		{"one degree along meridian", 10, 20, 11, 20, math.Pi * EarthRadiusMeters / 180, 1e-6},
		{"across the date line", 0, 179.5, 0, -179.5, math.Pi * EarthRadiusMeters / 180, 1e-6},
		{"pole to pole limit latitudes", MaxLatitude, 0, -MaxLatitude, 0, 160 * math.Pi * EarthRadiusMeters / 180, 1e-6},
		{"antipodal", 0, 0, 0, 180, math.Pi * EarthRadiusMeters, 1e-6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DistanceMeters(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if math.Abs(got-tt.want) > tt.tolerance {
				t.Fatalf("got %v, want %v +- %v", got, tt.want, tt.tolerance)
			}
			if back := DistanceMeters(tt.lat2, tt.lon2, tt.lat1, tt.lon1); math.Abs(back-got) > 1e-6 {
				t.Fatalf("not symmetric: %v vs %v", got, back)
			}
		})
	}
}

func TestInRange(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	tests := []struct {
		lat, lon float64
		want     bool
	}{
		{0, 0, true}, {80, 180, true}, {-80, -180, true}, {52.5, 13.4, true},
		{80.0001, 0, false}, {-81, 0, false}, {0, 180.0001, false}, {0, -181, false},
		{nan, 0, false}, {0, nan, false}, {inf, 0, false}, {0, math.Inf(-1), false},
	}
	for _, tt := range tests {
		if got := InRange(tt.lat, tt.lon); got != tt.want {
			t.Errorf("InRange(%v, %v) = %v, want %v", tt.lat, tt.lon, got, tt.want)
		}
	}
}

func TestBoundsAround(t *testing.T) {
	for _, lat := range []float64{0, 52.5, -33.9, MaxLatitude} {
		const half = 6000.0
		b := BoundsAround(lat, 13.4, half)
		height := (b.North - b.South) * math.Pi / 180 * EarthRadiusMeters
		width := (b.East - b.West) * math.Pi / 180 * EarthRadiusMeters * math.Cos(lat*math.Pi/180)
		if math.Abs(height-2*half) > 1e-6 || math.Abs(width-2*half) > 1e-6 {
			t.Errorf("lat %v: height %v width %v, want %v", lat, height, width, 2*half)
		}
		clat, clon := (b.North+b.South)/2, (b.East+b.West)/2
		if math.Abs(clat-lat) > 1e-12 || math.Abs(clon-13.4) > 1e-12 {
			t.Errorf("lat %v: center %v,%v", lat, clat, clon)
		}
		// North-south distance across the box matches the side length.
		if d := DistanceMeters(b.North, 13.4, b.South, 13.4); math.Abs(d-2*half) > 1e-3 {
			t.Errorf("lat %v: north-south distance %v", lat, d)
		}
	}
}

func TestCrossesDateLine(t *testing.T) {
	if BoundsAround(0, 0, 10_000).CrossesDateLine() {
		t.Error("origin should not cross")
	}
	if !BoundsAround(0, 179.99, 10_000).CrossesDateLine() {
		t.Error("east of 179.99 should cross")
	}
	if !BoundsAround(0, -179.99, 10_000).CrossesDateLine() {
		t.Error("west of -179.99 should cross")
	}
	if (Bounds{North: 1, South: -1, East: 180, West: -180}).CrossesDateLine() {
		t.Error("exactly the limits should not cross")
	}
}

func TestContains(t *testing.T) {
	b := Bounds{North: 10, South: 0, East: 20, West: 10}
	tests := []struct {
		lat, lon float64
		want     bool
	}{
		{5, 15, true}, {10, 20, true}, {0, 10, true},
		{10.1, 15, false}, {-0.1, 15, false}, {5, 9.9, false}, {5, 20.1, false},
		{math.NaN(), 15, false}, {5, math.NaN(), false},
	}
	for _, tt := range tests {
		if got := b.Contains(tt.lat, tt.lon); got != tt.want {
			t.Errorf("Contains(%v, %v) = %v, want %v", tt.lat, tt.lon, got, tt.want)
		}
	}
}

func TestGrid(t *testing.T) {
	const n = 6
	b := BoundsAround(52.5, 13.4, 6000)
	for row := 0; row < n; row++ {
		for column := 0; column < n; column++ {
			cell := b.Cell(n, row, column)
			clat, clon := (cell.North+cell.South)/2, (cell.East+cell.West)/2
			if !cell.Contains(clat, clon) {
				t.Errorf("cell %d,%d does not contain its center", row, column)
			}
			if r, c, ok := b.CellAt(n, clat, clon); !ok || r != row || c != column {
				t.Errorf("CellAt(center of %d,%d) = %d,%d,%v", row, column, r, c, ok)
			}
		}
	}
	if first := b.Cell(n, 0, 0); first.North != b.North || first.West != b.West {
		t.Errorf("first cell %+v not at north-west corner", first)
	}
	if r, c, ok := b.CellAt(n, b.South, b.East); !ok || r != n-1 || c != n-1 {
		t.Errorf("south-east corner = %d,%d,%v", r, c, ok)
	}
	if r, c, ok := b.CellAt(n, b.North, b.West); !ok || r != 0 || c != 0 {
		t.Errorf("north-west corner = %d,%d,%v", r, c, ok)
	}
	if _, _, ok := b.CellAt(n, b.North+1, b.West); ok {
		t.Error("point outside reported inside")
	}
}

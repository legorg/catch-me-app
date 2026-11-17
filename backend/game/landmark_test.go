package game

import (
	"math"
	"testing"
)

func TestDistanceMeters(t *testing.T) {
	berlin, _ := ParseCoordinates(52.5200, 13.4050)
	paris, _ := ParseCoordinates(48.8566, 2.3522)
	if got := berlin.DistanceMeters(paris); math.Abs(got-877_000) > 3_000 {
		t.Fatalf("berlin-paris = %.0f m, want about 877 km", got)
	}
	if berlin.DistanceMeters(berlin) != 0 {
		t.Fatal("distance to itself must be zero")
	}
}

func TestEdgeCells(t *testing.T) {
	center, _ := ParseCoordinates(52.52, 13.405)
	p, err := NewPlayground(center, DefaultCellSize(), DefaultGridSize())
	if err != nil {
		t.Fatal(err)
	}
	south, north := p.SouthEdgeCells(), p.NorthEdgeCells()
	if len(south) != DefaultGridSize().Cells() || len(north) != DefaultGridSize().Cells() {
		t.Fatal("expected one cell per column")
	}
	bottom, top := p.CellBounds(south[0]), p.CellBounds(north[DefaultGridSize().Cells()-1])
	if bottom.South != p.Bounds().South || top.North != p.Bounds().North || top.East != p.Bounds().East || bottom.West != p.Bounds().West {
		t.Fatalf("cells do not touch the playground edges: %+v %+v", bottom, top)
	}
	if !bottom.Contains(boundsCenter(bottom)) || bottom.Contains(boundsCenter(top)) {
		t.Fatal("cell containment is wrong")
	}
}

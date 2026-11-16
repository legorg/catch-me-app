// Package geo holds pure geographic math on plain float64 degrees. It is a
// leaf package: it knows nothing about the game, controller or service layers.
package geo

import "math"

const (
	// EarthRadiusMeters is the mean earth radius of the spherical model.
	EarthRadiusMeters = 6371008.8
	// MaxLatitude limits latitudes the app accepts; the local flat-cell
	// approximation breaks down near the poles.
	MaxLatitude = 80
	// MaxLongitude is the date line, the western and eastern limit.
	MaxLongitude = 180
)

func radians(degrees float64) float64 { return degrees * math.Pi / 180 }

// InRange reports whether the point is within the latitude and longitude limits
// the app accepts. NaN and infinite values are not in range.
func InRange(latitude, longitude float64) bool {
	// Written as a negated range check so NaN is rejected as well.
	return math.Abs(latitude) <= MaxLatitude && math.Abs(longitude) <= MaxLongitude
}

// DistanceMeters is the great-circle (haversine) distance between two points.
func DistanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	p1, p2 := radians(lat1), radians(lat2)
	dLat := p2 - p1
	dLon := radians(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * EarthRadiusMeters * math.Asin(math.Sqrt(a))
}

// Bounds is a latitude/longitude rectangle, north-aligned and not crossing
// the date line.
type Bounds struct{ North, South, East, West float64 }

// BoundsAround is the square with the given half side length in meters around
// a center, using a local spherical approximation.
func BoundsAround(latitude, longitude, halfSizeMeters float64) Bounds {
	latDelta := halfSizeMeters / EarthRadiusMeters * 180 / math.Pi
	lonDelta := latDelta / math.Cos(radians(latitude))
	return Bounds{
		North: latitude + latDelta, South: latitude - latDelta,
		East: longitude + lonDelta, West: longitude - lonDelta,
	}
}

// CrossesDateLine reports whether the bounds extend past the longitude limits.
func (b Bounds) CrossesDateLine() bool {
	return b.West < -MaxLongitude || b.East > MaxLongitude
}

// Contains reports whether the point lies inside the bounds, edges included.
func (b Bounds) Contains(latitude, longitude float64) bool {
	return latitude >= b.South && latitude <= b.North && longitude >= b.West && longitude <= b.East
}

// Cell is the area of the grid cell at row (from the north) and column (from
// the west) when the bounds are divided into n by n equal cells.
func (b Bounds) Cell(n, row, column int) Bounds {
	height := (b.North - b.South) / float64(n)
	width := (b.East - b.West) / float64(n)
	north := b.North - float64(row)*height
	west := b.West + float64(column)*width
	return Bounds{North: north, South: north - height, West: west, East: west + width}
}

// CellAt is the row and column of the cell of an n by n grid containing the
// point; ok is false if the point is outside the bounds.
func (b Bounds) CellAt(n int, latitude, longitude float64) (row, column int, ok bool) {
	if !b.Contains(latitude, longitude) {
		return 0, 0, false
	}
	row = int((b.North - latitude) / (b.North - b.South) * float64(n))
	column = int((longitude - b.West) / (b.East - b.West) * float64(n))
	return min(row, n-1), min(column, n-1), true
}

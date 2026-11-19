package main

import (
	"math"

	"catch-me/backend/geo"
)

const (
	centerLat = 52.52
	centerLon = 13.405
)

// Waypoints are minutes since start and local east/north meters. Segments are
// walked linearly; repeated coordinates give pauses. These are invented paths,
// not road routing or recordings of real people.
type waypoint struct{ minute, east, north float64 }

var runnerPaths = [][]waypoint{
	{{0, 0, -2500}, {20, -900, -1800}, {25, -900, -1800}, {55, -2000, -400}, {80, -1800, 1200}, {105, -900, 600}},
	{{0, 0, -2500}, {30, 1200, -1600}, {60, 2000, -300}, {70, 2000, -300}, {105, 1500, 1500}, {135, 400, 900}, {165, 900, -600}},
	{{0, 0, -2500}, {30, -1600, -1600}, {60, -900, -200}, {80, -900, -200}, {110, -2200, 1100}, {145, -1300, 1800}, {180, -600, 1100}, {210, 0, 2500}},
	{{0, 0, -2500}, {35, 1700, -2000}, {65, 900, -700}, {80, 900, -700}, {115, -500, 300}, {150, 1000, 1400}, {175, 2000, 500}, {210, 1300, 2100}, {220, 1300, 2100}, {240, 0, 2500}},
}

var hunterTeams = []string{"Falken", "Fuechse", "Woelfe", "Luchse"}

var hunterPaths = [][]waypoint{
	{{0, -2200, -1800}, {30, -800, -800}, {55, -2200, 0}, {80, -1800, 1200}, {105, -900, 600}, {130, 800, -500}, {165, -700, 1300}, {195, -1800, 2200}, {205, -1800, 2200}, {240, 200, 2000}},
	{{0, 2100, -2200}, {30, 900, -1200}, {55, 1800, 100}, {80, 400, 1300}, {105, 1800, 2000}, {130, 2200, 500}, {165, 900, -600}, {195, -400, 800}, {205, -400, 800}, {240, 1000, 2200}},
	{{0, -2000, 2000}, {30, -300, 1600}, {55, 1300, 300}, {80, 2200, -1200}, {105, 500, -1800}, {130, -1200, -500}, {165, -2200, 1000}, {195, -700, 2200}, {205, -700, 2200}, {240, 1200, 1200}},
	{{0, 2200, 1800}, {30, 700, 700}, {55, -1000, 1400}, {80, -2100, 200}, {105, -600, -1200}, {130, 1200, -2200}, {165, 2200, -300}, {195, 1100, 1300}, {205, 1100, 1300}, {240, -1000, 2200}},
}

func localPosition(path []waypoint, minute float64) (east, north float64) {
	previous := path[0]
	for _, next := range path[1:] {
		if minute <= next.minute {
			fraction := (minute - previous.minute) / (next.minute - previous.minute)
			return previous.east + fraction*(next.east-previous.east), previous.north + fraction*(next.north-previous.north)
		}
		previous = next
	}
	return previous.east, previous.north
}

func coordinates(east, north float64) (latitude, longitude float64) {
	latitude = centerLat + north/geo.EarthRadiusMeters*180/math.Pi
	longitude = centerLon + east/geo.EarthRadiusMeters*180/math.Pi/math.Cos(centerLat*math.Pi/180)
	return latitude, longitude
}

func playerPosition(index int, minute float64) (latitude, longitude float64) {
	if index < 4 {
		east, north := localPosition(runnerPaths[index], minute)
		return coordinates(east, north)
	}
	team, member := (index-4)/4, (index-4)%4
	east, north := localPosition(hunterPaths[team], minute)
	// Team members share a route but walk a few meters apart. A slow lateral
	// oscillation makes their tracks distinct without GPS-like teleportation.
	offset := float64(member) * 12
	east += offset * math.Cos(minute/12)
	north += offset * math.Sin(minute/12)
	return coordinates(east, north)
}

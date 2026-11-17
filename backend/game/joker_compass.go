package game

import (
	"math"
	"slices"
	"time"

	"catch-me/backend/geo"
)

// JokerCompass tells a hunter team the direction to every runner, once.
const JokerCompass JokerKind = "compass"

// Compass chooses the compass joker.
type Compass struct{}

func (Compass) Kind() JokerKind { return JokerCompass }

// apply needs the hunter's position as the origin; without one the joker is refused and stays available.
func (Compass) apply(l Lobby, me Player, now time.Time) (JokerUse, error) {
	if me.Position.IsZero() {
		return nil, &RuleError{Message: "report your position before using the compass"}
	}
	origin := me.Position.Coordinates
	return CompassUse{Team: me.HunterTeam, At: now, Origin: origin, Bearings: CompassBearings(origin, l.runnerPositions())}, nil
}

// CompassUse is a used compass joker: where the hunter stood and the direction to each runner.
// It never holds the runners' positions, distances, or IDs.
type CompassUse struct {
	Team   HunterTeamName
	At     time.Time
	Origin Coordinates
	// Bearings are whole degrees clockwise from true north (0-359), sorted.
	Bearings []int
}

var _ hunterJokerUse = CompassUse{}

func (CompassUse) Kind() JokerKind        { return JokerCompass }
func (u CompassUse) UsedAt() time.Time    { return u.At }
func (CompassUse) jokerUse()              {}
func (u CompassUse) team() HunterTeamName { return u.Team }

// CompassBearings are the directions from origin to each position, sorted so
// their order does not tell the runners apart.
func CompassBearings(origin Coordinates, positions []JokerPosition) []int {
	bearings := make([]int, 0, len(positions))
	for _, p := range positions {
		c := p.Position.Coordinates
		degrees := int(math.Round(geo.BearingDegrees(origin.latitude, origin.longitude, c.latitude, c.longitude))) % 360
		bearings = append(bearings, degrees)
	}
	slices.Sort(bearings)
	return bearings
}

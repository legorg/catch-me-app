package game

import "time"

// JokerZoneOnly makes the next reveal of the runner who used it show only their cell.
const JokerZoneOnly JokerKind = "zoneOnly"

// ZoneOnly chooses the zoneOnly joker.
type ZoneOnly struct{}

func (ZoneOnly) Kind() JokerKind { return JokerZoneOnly }

func (ZoneOnly) apply(_ Lobby, me Player, now time.Time) (JokerUse, error) {
	return ZoneOnlyUse{Runner: me.ID, At: now}, nil
}

// ZoneOnlyUse is a used zoneOnly joker.
type ZoneOnlyUse struct {
	Runner PlayerID
	At     time.Time
}

var _ revealJokerUse = ZoneOnlyUse{}

func (ZoneOnlyUse) Kind() JokerKind     { return JokerZoneOnly }
func (u ZoneOnlyUse) UsedAt() time.Time { return u.At }
func (ZoneOnlyUse) jokerUse()           {}
func (u ZoneOnlyUse) runner() PlayerID  { return u.Runner }

// reveal shows only the cell the runner is in at the reveal, or nothing
// outside the playground; never the position.
func (ZoneOnlyUse) reveal(r Reveal, p Playground) HunterReveal {
	if cell, ok := p.CellAt(r.Position.Coordinates); ok {
		return ZoneReveal{At: r.At, Cell: cell}
	}
	return OutsideReveal{At: r.At}
}

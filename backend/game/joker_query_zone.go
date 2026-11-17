package game

import "time"

// JokerQueryZone tells a hunter team whether any runner is in a chosen cell, once.
const JokerQueryZone JokerKind = "queryZone"

// QueryZone chooses the queryZone joker for a cell.
type QueryZone struct{ Cell Cell }

func (QueryZone) Kind() JokerKind { return JokerQueryZone }

// apply records only the answer: never which runners or where.
func (c QueryZone) apply(l Lobby, me Player, now time.Time) (JokerUse, error) {
	if !l.Game.Playground.HasCell(c.Cell) {
		return nil, &ValidationError{Message: "cell is not part of this game's grid"}
	}
	found := l.Game.Playground.AnyInCell(l.runnerPositions(), c.Cell)
	return QueryZoneUse{Team: me.HunterTeam, At: now, Cell: c.Cell, Found: found}, nil
}

// QueryZoneUse is a used queryZone joker: the cell and whether any runner was in it.
type QueryZoneUse struct {
	Team  HunterTeamName
	At    time.Time
	Cell  Cell
	Found bool
}

var _ hunterJokerUse = QueryZoneUse{}

func (QueryZoneUse) Kind() JokerKind        { return JokerQueryZone }
func (u QueryZoneUse) UsedAt() time.Time    { return u.At }
func (QueryZoneUse) jokerUse()              {}
func (u QueryZoneUse) team() HunterTeamName { return u.Team }

// AnyInCell reports whether any of the positions is inside the cell.
func (p Playground) AnyInCell(positions []JokerPosition, cell Cell) bool {
	for _, position := range positions {
		if at, ok := p.CellAt(position.Position.Coordinates); ok && at == cell {
			return true
		}
	}
	return false
}

package game

import "time"

// JokerShowHunters gives the runners every hunter's current position, once.
const JokerShowHunters JokerKind = "showHunters"

// ShowHunters chooses the showHunters joker.
type ShowHunters struct{}

func (ShowHunters) Kind() JokerKind { return JokerShowHunters }

func (ShowHunters) apply(l Lobby, me Player, now time.Time) (JokerUse, error) {
	return ShowHuntersUse{Runner: me.ID, At: now, Positions: l.hunterPositions()}, nil
}

// ShowHuntersUse is a used showHunters joker with every hunter's position at
// the time. It acts at once and changes no reveal.
type ShowHuntersUse struct {
	Runner    PlayerID
	At        time.Time
	Positions []JokerPosition
}

var _ runnerJokerUse = ShowHuntersUse{}

func (ShowHuntersUse) Kind() JokerKind     { return JokerShowHunters }
func (u ShowHuntersUse) UsedAt() time.Time { return u.At }
func (ShowHuntersUse) jokerUse()           {}
func (u ShowHuntersUse) runner() PlayerID  { return u.Runner }

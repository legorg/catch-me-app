package game

import "time"

// JokerShowRunners gives a hunter team every runner's current position, once.
const JokerShowRunners JokerKind = "showRunners"

// ShowRunners chooses the showRunners joker.
type ShowRunners struct{}

func (ShowRunners) Kind() JokerKind { return JokerShowRunners }

func (ShowRunners) apply(l Lobby, me Player, now time.Time) (JokerUse, error) {
	return ShowRunnersUse{Team: me.HunterTeam, At: now, Positions: l.runnerPositions()}, nil
}

// ShowRunnersUse is a used showRunners joker with every runner's position at the time.
type ShowRunnersUse struct {
	Team      HunterTeamName
	At        time.Time
	Positions []JokerPosition
}

var _ hunterJokerUse = ShowRunnersUse{}

func (ShowRunnersUse) Kind() JokerKind        { return JokerShowRunners }
func (u ShowRunnersUse) UsedAt() time.Time    { return u.At }
func (ShowRunnersUse) jokerUse()              {}
func (u ShowRunnersUse) team() HunterTeamName { return u.Team }

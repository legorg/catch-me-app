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

// Located is what runners learn of a showRunners use: which hunter team used
// it and when, never what it showed.
type Located struct {
	Team HunterTeamName
	At   time.Time
}

// LocatedFor lists, oldest first, the showRunners uses made while the viewer
// was an active runner, so they included the viewer. Other viewers get none.
func (l Lobby) LocatedFor(viewer Player) []Located {
	if viewer.Team != Runners {
		return nil
	}
	var located []Located
	for _, use := range l.Jokers {
		if u, ok := use.(ShowRunnersUse); ok && viewer.Runner.ActiveAt(u.At) {
			located = append(located, Located{Team: u.Team, At: u.At})
		}
	}
	return located
}

package game

import (
	"strings"
	"time"
)

// JokerKind is a one-off special ability.
type JokerKind string

const (
	// JokerShowRunners gives a hunter team every runner's current position, once.
	JokerShowRunners JokerKind = "showRunners"
)

// hunterJokers are the jokers every hunter team has, in display order.
var hunterJokers = []JokerKind{JokerShowRunners}

func ParseJokerKind(value string) (JokerKind, error) {
	kind := JokerKind(value)
	for _, known := range hunterJokers {
		if kind == known {
			return kind, nil
		}
	}
	return "", &ValidationError{Message: "unknown joker"}
}

// JokerPosition is where a runner was when a joker was used.
type JokerPosition struct {
	PlayerID PlayerID
	Position Position
}

// JokerUse records that a hunter team used a joker. Positions are derived when
// loading, like reveals: each runner's last position at or before UsedAt.
type JokerUse struct {
	Kind      JokerKind
	Team      HunterTeamName
	UsedAt    time.Time
	Positions []JokerPosition
}

// JokerState is one joker as one player sees it.
type JokerState struct {
	Kind      JokerKind
	Available bool
	// UsedAt and Positions are set once the player's team used the joker.
	UsedAt    time.Time
	Positions []JokerPosition
}

// UseJoker spends a joker of the player's hunter team. Hunters of one team
// share their jokers; each can be used once per game, only while it runs.
func (l *Lobby) UseJoker(id PlayerID, kind JokerKind, now time.Time) error {
	i := l.index(id)
	if i < 0 {
		return ErrUnauthorized
	}
	me := l.Players[i]
	switch {
	case me.Team != Hunters || me.HunterTeam.IsZero():
		return &RuleError{Message: "only hunters in a hunter team have jokers"}
	case l.Game.Status(now) != StatusRunning:
		return &RuleError{Message: "jokers can only be used while the game is running"}
	case l.jokerUse(me.HunterTeam, kind) != nil:
		return &RuleError{Message: "your hunter team already used this joker"}
	}
	use := JokerUse{Kind: kind, Team: me.HunterTeam, UsedAt: now}
	if kind == JokerShowRunners {
		for _, p := range l.Players {
			if p.Team == Runners && !p.Position.IsZero() {
				use.Positions = append(use.Positions, JokerPosition{PlayerID: p.ID, Position: p.Position})
			}
		}
	}
	l.Jokers = append(l.Jokers, use)
	return nil
}

// JokersFor lists the jokers of the viewer's team and what they showed.
// Runners have none yet; other hunter teams' jokers are never visible.
func (l Lobby) JokersFor(viewer Player, now time.Time) []JokerState {
	if viewer.Team != Hunters || viewer.HunterTeam.IsZero() {
		return nil
	}
	running := l.Game.Status(now) == StatusRunning
	states := make([]JokerState, 0, len(hunterJokers))
	for _, kind := range hunterJokers {
		state := JokerState{Kind: kind, Available: running}
		if use := l.jokerUse(viewer.HunterTeam, kind); use != nil {
			state.Available, state.UsedAt, state.Positions = false, use.UsedAt, use.Positions
		}
		states = append(states, state)
	}
	return states
}

func (l Lobby) jokerUse(team HunterTeamName, kind JokerKind) *JokerUse {
	for i := range l.Jokers {
		if l.Jokers[i].Kind == kind && strings.EqualFold(l.Jokers[i].Team.String(), team.String()) {
			return &l.Jokers[i]
		}
	}
	return nil
}

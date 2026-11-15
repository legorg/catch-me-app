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
	// JokerQueryZone tells a hunter team whether any runner is in a chosen cell, once.
	JokerQueryZone JokerKind = "queryZone"
)

// hunterJokers are the jokers every hunter team has, in display order.
var hunterJokers = []JokerKind{JokerShowRunners, JokerQueryZone}

func ParseJokerKind(value string) (JokerKind, error) {
	kind := JokerKind(value)
	for _, known := range hunterJokers {
		if kind == known {
			return kind, nil
		}
	}
	return "", &ValidationError{Message: "unknown joker"}
}

// JokerChoice is a joker to use with its target: the cell for queryZone, none otherwise.
type JokerChoice struct {
	Kind JokerKind
	Cell Cell
}

// ParseJokerChoice checks that the kind is known and that a cell is given exactly when the joker needs one.
func ParseJokerChoice(kind, cell string) (JokerChoice, error) {
	parsed, err := ParseJokerKind(kind)
	if err != nil {
		return JokerChoice{}, err
	}
	switch parsed {
	case JokerQueryZone:
		target, err := ParseCell(cell)
		if err != nil {
			return JokerChoice{}, err
		}
		return JokerChoice{Kind: parsed, Cell: target}, nil
	default:
		if cell != "" {
			return JokerChoice{}, &ValidationError{Message: "this joker takes no cell"}
		}
		return JokerChoice{Kind: parsed}, nil
	}
}

// JokerPosition is where a runner was when a joker was used.
type JokerPosition struct {
	PlayerID PlayerID
	Position Position
}

// JokerUse records that a hunter team used a joker. What it showed is derived
// when loading, like reveals, from each runner's last position at or before
// UsedAt: the positions (showRunners) or whether one was in Cell (queryZone).
type JokerUse struct {
	Kind      JokerKind
	Team      HunterTeamName
	UsedAt    time.Time
	Positions []JokerPosition
	// Cell and Found are set for queryZone.
	Cell  Cell
	Found bool
}

// JokerState is one joker as one player sees it.
type JokerState struct {
	Kind      JokerKind
	Available bool
	// UsedAt and the result (Positions, or Cell and Found) are set once the
	// player's team used the joker.
	UsedAt    time.Time
	Positions []JokerPosition
	Cell      Cell
	Found     bool
}

// AnyInCell reports whether any of the positions is inside the cell.
func (p Playground) AnyInCell(positions []JokerPosition, cell Cell) bool {
	for _, position := range positions {
		if at, ok := p.CellAt(position.Position.Coordinates); ok && at == cell {
			return true
		}
	}
	return false
}

// UseJoker spends a joker of the player's hunter team. Hunters of one team
// share their jokers; each can be used once per game, only while it runs.
func (l *Lobby) UseJoker(id PlayerID, choice JokerChoice, now time.Time) error {
	kind := choice.Kind
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
	var positions []JokerPosition
	for _, p := range l.Players {
		if p.Team == Runners && !p.Position.IsZero() {
			positions = append(positions, JokerPosition{PlayerID: p.ID, Position: p.Position})
		}
	}
	use := JokerUse{Kind: kind, Team: me.HunterTeam, UsedAt: now}
	switch kind {
	case JokerShowRunners:
		use.Positions = positions
	case JokerQueryZone:
		// Only the answer: never which runners or where.
		use.Cell, use.Found = choice.Cell, l.Game.Playground.AnyInCell(positions, choice.Cell)
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
			state.Cell, state.Found = use.Cell, use.Found
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

package game

import (
	"slices"
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
	// JokerZoneOnly makes the next reveal of the runner who used it show only their cell.
	JokerZoneOnly JokerKind = "zoneOnly"
)

// hunterJokers are the jokers every hunter team has, in display order.
var hunterJokers = []JokerKind{JokerShowRunners, JokerQueryZone}

// runnerJokers are the jokers the runner team shares, in display order.
var runnerJokers = []JokerKind{JokerZoneOnly}

// ParseJokerKind accepts any string-based type, e.g. a wire-format kind.
func ParseJokerKind[S ~string](value S) (JokerKind, error) {
	kind := JokerKind(value)
	for _, known := range slices.Concat(hunterJokers, runnerJokers) {
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
func ParseJokerChoice[K, C ~string](kind K, cell C) (JokerChoice, error) {
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

// JokerUse records that a team used a joker: a hunter team (Team), or the
// runners (Team empty, PlayerID is the runner who used it). What a hunter
// joker showed is derived when loading, like reveals, from each runner's last
// position at or before UsedAt: the positions (showRunners) or whether one was
// in Cell (queryZone). Runner jokers act on the user's next reveal instead.
type JokerUse struct {
	Kind      JokerKind
	Team      HunterTeamName
	PlayerID  PlayerID
	UsedAt    time.Time
	Positions []JokerPosition
	// Cell and Found are set for queryZone.
	Cell  Cell
	Found bool
}

// jokerState is one joker as one player sees it.
type jokerState struct {
	Kind      JokerKind
	Available bool
	// UsedAt and the result (Positions, or Cell and Found) are set once the
	// player's team used the joker; UsedBy for runner jokers.
	UsedAt    time.Time
	UsedBy    PlayerID
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

// UseJoker spends a joker of the player's team: hunters of one hunter team
// share their jokers, all runners share theirs. Each can be used once per game
// and team, only while it runs.
func (l *Lobby) UseJoker(id PlayerID, choice JokerChoice, now time.Time) error {
	kind := choice.Kind
	i := l.index(id)
	if i < 0 {
		return ErrUnauthorized
	}
	me := l.Players[i]
	if me.Team == Runners {
		return l.useRunnerJoker(me, kind, now)
	}
	switch {
	case me.HunterTeam.IsZero():
		return &RuleError{Message: "only hunters in a hunter team have jokers"}
	case !slices.Contains(hunterJokers, kind):
		return &RuleError{Message: "hunters do not have this joker"}
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

// useRunnerJoker spends a joker of the runner team. It only affects the
// runner who used it, at their next reveal.
func (l *Lobby) useRunnerJoker(me Player, kind JokerKind, now time.Time) error {
	switch {
	case !slices.Contains(runnerJokers, kind):
		return &RuleError{Message: "runners do not have this joker"}
	case l.Game.Status(now) != StatusRunning:
		return &RuleError{Message: "jokers can only be used while the game is running"}
	case l.jokerUse(HunterTeamName{}, kind) != nil:
		return &RuleError{Message: "a runner already used this joker"}
	}
	l.Jokers = append(l.Jokers, JokerUse{Kind: kind, PlayerID: me.ID, UsedAt: now})
	return nil
}

// JokersFor lists the jokers of the viewer's team and what they showed.
// Other teams' jokers are never visible; hunters are not told about runner jokers.
func (l Lobby) JokersFor(viewer Player, now time.Time) []jokerState {
	kinds, team := runnerJokers, HunterTeamName{}
	if viewer.Team == Hunters {
		if viewer.HunterTeam.IsZero() {
			return nil
		}
		kinds, team = hunterJokers, viewer.HunterTeam
	}
	running := l.Game.Status(now) == StatusRunning
	states := make([]jokerState, 0, len(kinds))
	for _, kind := range kinds {
		state := jokerState{Kind: kind, Available: running}
		if use := l.jokerUse(team, kind); use != nil {
			state.Available, state.UsedAt, state.UsedBy, state.Positions = false, use.UsedAt, use.PlayerID, use.Positions
			state.Cell, state.Found = use.Cell, use.Found
		}
		states = append(states, state)
	}
	return states
}

// jokerUse finds the use of a kind by a hunter team, or by the runners for an empty team.
func (l Lobby) jokerUse(team HunterTeamName, kind JokerKind) *JokerUse {
	for i := range l.Jokers {
		if l.Jokers[i].Kind == kind && strings.EqualFold(l.Jokers[i].Team.String(), team.String()) {
			return &l.Jokers[i]
		}
	}
	return nil
}

// revealsForHunters are the runner's reveals with their jokers applied: the
// reveal after a zoneOnly use keeps only the cell, never the position.
func (l Lobby) revealsForHunters(runner PlayerID) []Reveal {
	reveals := l.Reveals[runner]
	var zoneOnly []time.Time
	for _, use := range l.Jokers {
		if use.Kind == JokerZoneOnly && use.PlayerID == runner {
			zoneOnly = append(zoneOnly, l.Game.NextReveal(use.UsedAt))
		}
	}
	if len(zoneOnly) == 0 {
		return reveals
	}
	reveals = slices.Clone(reveals)
	for i, r := range reveals {
		if slices.ContainsFunc(zoneOnly, r.At.Equal) {
			reveals[i] = Reveal{At: r.At, ZoneOnly: true}
			if cell, ok := l.Game.Playground.CellAt(r.Position.Coordinates); ok {
				reveals[i].Cell, reveals[i].InPlayground = cell, true
			}
		}
	}
	return reveals
}

package game

import (
	"slices"
	"time"
)

// JokerKind is a special ability with a configured allowance. Each joker lives
// in its own file (joker_<name>.go) with its choice, its use, and its logic;
// this file holds what all jokers share.
type JokerKind string

// hunterJokers are the jokers every hunter team has, in display order.
var hunterJokers = []JokerKind{JokerShowRunners, JokerQueryZone, JokerCompass, JokerTrap}

// revealJokers are the runner jokers that act on the user's next reveal.
var revealJokers = []JokerKind{JokerZoneOnly, JokerFakePosition}

// runnerJokers are the runners' jokers, in display order. Fake position is personal.
var runnerJokers = []JokerKind{JokerZoneOnly, JokerFakePosition, JokerShowHunters}

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

// JokerChoice is a joker to use with its target. It is sealed: the only
// variants are ShowRunners, QueryZone, Compass, Trap, ZoneOnly, FakePosition, and ShowHunters, each with
// exactly the target its joker needs.
type JokerChoice interface {
	Kind() JokerKind
	// apply checks the joker's own rules and returns its use; UseJoker has
	// already checked the rules all jokers share.
	apply(l Lobby, me Player, now time.Time) (JokerUse, error)
}

// JokerPosition is where a player was when a joker was used.
type JokerPosition struct {
	PlayerID PlayerID
	Position Position
}

// JokerUse records that a joker was used. It is sealed: every use is either a
// hunterJokerUse (ShowRunnersUse, QueryZoneUse, CompassUse, TrapUse) or a runnerJokerUse
// (ZoneOnlyUse, FakePositionUse, ShowHuntersUse). What the hunter jokers and
// showHunters showed is derived when loading, like reveals, from each
// player's last position at or before the use; a trap from the runners'
// reports after its activation; reveal jokers act on the user's next reveal instead.
type JokerUse interface {
	Kind() JokerKind
	UsedAt() time.Time
	jokerUse()
}

// hunterJokerUse is a use that belongs to a hunter team.
type hunterJokerUse interface {
	JokerUse
	team() HunterTeamName
}

// runnerJokerUse is a use by one runner, either personal or shared by the runner team.
type runnerJokerUse interface {
	JokerUse
	runner() PlayerID
}

// revealJokerUse is a runner joker that changes that runner's next reveal as hunters see it.
type revealJokerUse interface {
	runnerJokerUse
	// reveal is what hunters see instead of the real reveal r. It must never
	// contain the real position unless the joker allows it.
	reveal(r Reveal, p Playground) HunterReveal
}

// jokerState is one joker as one player sees it. Remaining and Uses are derived
// from the configured allowance and the permitted personal or team history.
type jokerState struct {
	Kind      JokerKind
	Available bool
	Remaining int
	Uses      []JokerUse
}

// UseJoker spends a configured use while the game runs. Fake position is
// personal; other jokers spend the runner team's or hunter team's allowance.
func (l *Lobby) UseJoker(id PlayerID, choice JokerChoice, now time.Time) error {
	if choice == nil {
		return &ValidationError{Message: "unknown joker"}
	}
	me, err := l.requirePlayer(id)
	if err != nil {
		return err
	}
	if err := l.mayUseJoker(*me, choice.Kind(), now); err != nil {
		return err
	}
	use, err := choice.apply(*l, *me, now)
	if err != nil {
		return err
	}
	l.Jokers = append(l.Jokers, use)
	return nil
}

// mayUseJoker checks the rules all jokers share. A reveal joker only affects
// the runner who used it, at their next reveal; until then, that runner cannot
// use another reveal joker.
func (l Lobby) mayUseJoker(me Player, kind JokerKind, now time.Time) error {
	if !me.active() {
		return &RuleError{Message: "caught or finished runners cannot use jokers"}
	}
	if me.Team == Runners {
		switch {
		case !slices.Contains(runnerJokers, kind):
			return &RuleError{Message: "runners do not have this joker"}
		case l.Game.Status(now) != StatusRunning:
			return &RuleError{Message: "jokers can only be used while the game is running"}
		case len(l.jokerUses(me, kind)) >= l.Game.JokerSettings.Allowance(kind).Max():
			return &RuleError{Message: "no uses of this joker remaining"}
		case slices.Contains(revealJokers, kind) && l.hasPendingJoker(me.ID, now):
			return &RuleError{Message: "your last joker has not acted yet; wait for the next reveal"}
		}
		return nil
	}
	switch {
	case me.HunterTeam.IsZero():
		return &RuleError{Message: "only hunters in a hunter team have jokers"}
	case !slices.Contains(hunterJokers, kind):
		return &RuleError{Message: "hunters do not have this joker"}
	case l.Game.Status(now) != StatusRunning:
		return &RuleError{Message: "jokers can only be used while the game is running"}
	case len(l.jokerUses(me, kind)) >= l.Game.JokerSettings.Allowance(kind).Max():
		return &RuleError{Message: "no uses of this joker remaining for your hunter team"}
	}
	return nil
}

// runnerPositions are the latest positions of all runners who reported one.
func (l Lobby) runnerPositions() []JokerPosition { return l.teamPositions(Runners) }

// hunterPositions are the latest positions of all hunters (of every hunter team) who reported one.
func (l Lobby) hunterPositions() []JokerPosition { return l.teamPositions(Hunters) }

func (l Lobby) teamPositions(team Team) []JokerPosition {
	var positions []JokerPosition
	for _, p := range l.Players {
		if p.Team == team && p.active() && !p.Position.IsZero() {
			positions = append(positions, JokerPosition{PlayerID: p.ID, Position: p.Position})
		}
	}
	return positions
}

// hasPendingJoker reports whether the runner used a reveal joker whose reveal has not happened yet.
func (l Lobby) hasPendingJoker(runner PlayerID, now time.Time) bool {
	return slices.ContainsFunc(l.Jokers, func(use JokerUse) bool {
		u, ok := use.(revealJokerUse)
		return ok && u.runner() == runner && now.Before(l.Game.NextReveal(use.UsedAt()))
	})
}

// JokersFor lists the viewer's personal and team jokers and what they showed.
// Other teams' jokers are never visible; hunters are not told about runner jokers.
func (l Lobby) JokersFor(viewer Player, now time.Time) []jokerState {
	if viewer.Team == Spectators {
		return nil
	}
	kinds := runnerJokers
	if viewer.Team == Hunters {
		if viewer.HunterTeam.IsZero() {
			return nil
		}
		kinds = hunterJokers
	}
	running := l.Game.Status(now) == StatusRunning
	states := make([]jokerState, 0, len(kinds))
	for _, kind := range kinds {
		uses := l.jokerUses(viewer, kind)
		remaining := max(0, l.Game.JokerSettings.Allowance(kind).Max()-len(uses))
		state := jokerState{
			Kind: kind, Available: running && viewer.active() && remaining > 0,
			Remaining: remaining, Uses: uses,
		}
		states = append(states, state)
	}
	return states
}

// jokerUses selects all uses in stored order belonging to the viewer's scope.
// Shared runner jokers include every runner's uses; fake position is personal.
// Hunter team names ignore case. No other team's history is returned.
func (l Lobby) jokerUses(viewer Player, kind JokerKind) []JokerUse {
	var uses []JokerUse
	for _, use := range l.Jokers {
		if use.Kind() != kind {
			continue
		}
		switch viewer.Team {
		case Hunters:
			if u, ok := use.(hunterJokerUse); ok && !viewer.HunterTeam.IsZero() && u.team().sameAs(viewer.HunterTeam) {
				uses = append(uses, use)
			}
		case Runners:
			if u, ok := use.(runnerJokerUse); ok && (kind != JokerFakePosition || u.runner() == viewer.ID) {
				uses = append(uses, use)
			}
		}
	}
	return uses
}

// maxHunterReveals bounds the reveals sent per runner; games have no time limit.
const maxHunterReveals = 256

// revealsForHunters are the runner's latest maxHunterReveals reveals as hunters
// see them: the reveal after each of the runner's joker uses is replaced by
// what that joker shows.
func (l Lobby) revealsForHunters(runner PlayerID) []HunterReveal {
	var acting []revealJokerUse
	for _, use := range l.Jokers {
		if u, ok := use.(revealJokerUse); ok && u.runner() == runner {
			acting = append(acting, u)
		}
	}
	all := l.Reveals[runner]
	var reveals []HunterReveal
	for _, r := range all[max(0, len(all)-maxHunterReveals):] {
		j := slices.IndexFunc(acting, func(use revealJokerUse) bool { return l.Game.NextReveal(use.UsedAt()).Equal(r.At) })
		if j < 0 {
			reveals = append(reveals, PositionReveal{At: r.At, Position: r.Position})
		} else {
			reveals = append(reveals, acting[j].reveal(r, l.Game.Playground))
		}
	}
	return reveals
}

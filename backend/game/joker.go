package game

import (
	"slices"
	"time"
)

// JokerKind is a one-off special ability. Each joker lives in its own file
// (joker_<name>.go) with its choice, its use, and its logic; this file holds
// what all jokers share.
type JokerKind string

// hunterJokers are the jokers every hunter team has, in display order.
var hunterJokers = []JokerKind{JokerShowRunners, JokerQueryZone, JokerCompass}

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
// variants are ShowRunners, QueryZone, Compass, ZoneOnly, FakePosition, and ShowHunters, each with
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
// hunterJokerUse (ShowRunnersUse, QueryZoneUse, CompassUse) or a runnerJokerUse
// (ZoneOnlyUse, FakePositionUse, ShowHuntersUse). What the hunter jokers and
// showHunters showed is derived when loading, like reveals, from each
// player's last position at or before the use; reveal jokers act on the
// user's next reveal instead.
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

// jokerState is one joker as one player sees it: Use is nil until the
// player (for a personal joker) or their team used it.
type jokerState struct {
	Kind      JokerKind
	Available bool
	Use       JokerUse
}

// UseJoker spends a joker while the game runs. Fake position is once per runner;
// other jokers are once per runner team or hunter team.
func (l *Lobby) UseJoker(id PlayerID, choice JokerChoice, now time.Time) error {
	if choice == nil {
		return &ValidationError{Message: "unknown joker"}
	}
	i := l.index(id)
	if i < 0 {
		return ErrUnauthorized
	}
	me := l.Players[i]
	if err := l.mayUseJoker(me, choice.Kind(), now); err != nil {
		return err
	}
	use, err := choice.apply(*l, me, now)
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
	if me.Team == Runners {
		switch {
		case !slices.Contains(runnerJokers, kind):
			return &RuleError{Message: "runners do not have this joker"}
		case l.Game.Status(now) != StatusRunning:
			return &RuleError{Message: "jokers can only be used while the game is running"}
		case l.runnerUse(me.ID, kind) != nil:
			return &RuleError{Message: "this joker has already been used"}
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
	case l.hunterTeamUse(me.HunterTeam, kind) != nil:
		return &RuleError{Message: "your hunter team already used this joker"}
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
		if p.Team == team && !p.Position.IsZero() {
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
	kinds := runnerJokers
	find := func(kind JokerKind) JokerUse { return l.runnerUse(viewer.ID, kind) }
	if viewer.Team == Hunters {
		if viewer.HunterTeam.IsZero() {
			return nil
		}
		kinds = hunterJokers
		find = func(kind JokerKind) JokerUse { return l.hunterTeamUse(viewer.HunterTeam, kind) }
	}
	running := l.Game.Status(now) == StatusRunning
	states := make([]jokerState, 0, len(kinds))
	for _, kind := range kinds {
		use := find(kind)
		states = append(states, jokerState{Kind: kind, Available: running && use == nil, Use: use})
	}
	return states
}

// hunterTeamUse finds the hunter team's use of a kind (team names ignore case).
func (l Lobby) hunterTeamUse(team HunterTeamName, kind JokerKind) JokerUse {
	for _, use := range l.Jokers {
		if u, ok := use.(hunterJokerUse); ok && use.Kind() == kind && u.team().sameAs(team) {
			return use
		}
	}
	return nil
}

// runnerUse finds a personal fake position use, or a shared runner team use.
func (l Lobby) runnerUse(runner PlayerID, kind JokerKind) JokerUse {
	for _, use := range l.Jokers {
		if u, ok := use.(runnerJokerUse); ok && use.Kind() == kind && (kind != JokerFakePosition || u.runner() == runner) {
			return use
		}
	}
	return nil
}

// revealsForHunters are the runner's reveals as hunters see them: the reveal
// after each of the runner's joker uses is replaced by what that joker shows.
func (l Lobby) revealsForHunters(runner PlayerID) []HunterReveal {
	var acting []revealJokerUse
	for _, use := range l.Jokers {
		if u, ok := use.(revealJokerUse); ok && u.runner() == runner {
			acting = append(acting, u)
		}
	}
	var reveals []HunterReveal
	for _, r := range l.Reveals[runner] {
		j := slices.IndexFunc(acting, func(use revealJokerUse) bool { return l.Game.NextReveal(use.UsedAt()).Equal(r.At) })
		if j < 0 {
			reveals = append(reveals, PositionReveal{At: r.At, Position: r.Position})
		} else {
			reveals = append(reveals, acting[j].reveal(r, l.Game.Playground))
		}
	}
	return reveals
}

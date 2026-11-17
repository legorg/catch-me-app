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
var hunterJokers = []JokerKind{JokerShowRunners, JokerQueryZone}

// runnerJokers are the jokers the runner team shares, in display order.
var runnerJokers = []JokerKind{JokerZoneOnly, JokerFakePosition}

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
// variants are ShowRunners, QueryZone, ZoneOnly, and FakePosition, each with
// exactly the target its joker needs.
type JokerChoice interface {
	Kind() JokerKind
	// apply checks the joker's own rules and returns its use; UseJoker has
	// already checked the rules all jokers share.
	apply(l Lobby, me Player, now time.Time) (JokerUse, error)
}

// JokerPosition is where a runner was when a joker was used.
type JokerPosition struct {
	PlayerID PlayerID
	Position Position
}

// JokerUse records that a joker was used. It is sealed: every use is either a
// hunterJokerUse (ShowRunnersUse, QueryZoneUse) or a runnerJokerUse
// (ZoneOnlyUse, FakePositionUse). What a hunter joker showed is derived when
// loading, like reveals, from each runner's last position at or before the
// use; runner jokers act on the user's next reveal instead.
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

// runnerJokerUse is a use by one runner, for the whole runner team. It changes
// that runner's next reveal as hunters see it.
type runnerJokerUse interface {
	JokerUse
	runner() PlayerID
	// reveal is what hunters see instead of the real reveal r. It must never
	// contain the real position unless the joker allows it.
	reveal(r Reveal, p Playground) HunterReveal
}

// jokerState is one joker as one player sees it: Use is nil until the
// player's team used it.
type jokerState struct {
	Kind      JokerKind
	Available bool
	Use       JokerUse
}

// UseJoker spends a joker of the player's team: hunters of one hunter team
// share their jokers, all runners share theirs. Each can be used once per game
// and team, only while it runs.
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

// mayUseJoker checks the rules all jokers share. A runner joker only affects
// the runner who used it, at their next reveal; until then, that runner cannot
// use another one.
func (l Lobby) mayUseJoker(me Player, kind JokerKind, now time.Time) error {
	if me.Team == Runners {
		switch {
		case !slices.Contains(runnerJokers, kind):
			return &RuleError{Message: "runners do not have this joker"}
		case l.Game.Status(now) != StatusRunning:
			return &RuleError{Message: "jokers can only be used while the game is running"}
		case l.runnersUse(kind) != nil:
			return &RuleError{Message: "a runner already used this joker"}
		case l.hasPendingJoker(me.ID, now):
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
func (l Lobby) runnerPositions() []JokerPosition {
	var positions []JokerPosition
	for _, p := range l.Players {
		if p.Team == Runners && !p.Position.IsZero() {
			positions = append(positions, JokerPosition{PlayerID: p.ID, Position: p.Position})
		}
	}
	return positions
}

// hasPendingJoker reports whether the runner used a joker whose reveal has not happened yet.
func (l Lobby) hasPendingJoker(runner PlayerID, now time.Time) bool {
	return slices.ContainsFunc(l.Jokers, func(use JokerUse) bool {
		u, ok := use.(runnerJokerUse)
		return ok && u.runner() == runner && now.Before(l.Game.NextReveal(use.UsedAt()))
	})
}

// JokersFor lists the jokers of the viewer's team and what they showed.
// Other teams' jokers are never visible; hunters are not told about runner jokers.
func (l Lobby) JokersFor(viewer Player, now time.Time) []jokerState {
	kinds, find := runnerJokers, l.runnersUse
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

// runnersUse finds the runners' use of a kind; all runners share one.
func (l Lobby) runnersUse(kind JokerKind) JokerUse {
	for _, use := range l.Jokers {
		if _, ok := use.(runnerJokerUse); ok && use.Kind() == kind {
			return use
		}
	}
	return nil
}

// revealsForHunters are the runner's reveals as hunters see them: the reveal
// after each of the runner's joker uses is replaced by what that joker shows.
func (l Lobby) revealsForHunters(runner PlayerID) []HunterReveal {
	var acting []runnerJokerUse
	for _, use := range l.Jokers {
		if u, ok := use.(runnerJokerUse); ok && u.runner() == runner {
			acting = append(acting, u)
		}
	}
	var reveals []HunterReveal
	for _, r := range l.Reveals[runner] {
		j := slices.IndexFunc(acting, func(use runnerJokerUse) bool { return l.Game.NextReveal(use.UsedAt()).Equal(r.At) })
		if j < 0 {
			reveals = append(reveals, PositionReveal{At: r.At, Position: r.Position})
		} else {
			reveals = append(reveals, acting[j].reveal(r, l.Game.Playground))
		}
	}
	return reveals
}

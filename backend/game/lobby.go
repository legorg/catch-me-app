package game

import (
	"fmt"
	"slices"
	"time"
)

const (
	CountdownDuration = 10 * time.Second
	maxPlayers        = 30
)

type Status string

const (
	StatusLobby     Status = "lobby"
	StatusCountdown Status = "countdown"
	StatusRunning   Status = "running"
)

// Status is derived from the scheduled start, so time-based transitions need
// no background job: a countdown that has elapsed simply reads as running.
func (g Game) Status(now time.Time) Status {
	switch {
	case g.StartsAt.IsZero():
		return StatusLobby
	case now.Before(g.StartsAt):
		return StatusCountdown
	}
	return StatusRunning
}

// Lobby is a game with its players. Its methods apply the joining and start
// rules; callers persist the result.
type Lobby struct {
	Game    Game
	Players []Player
	// Reveals are what hunters see of each runner, oldest first: at every reveal
	// time (Game.RevealTimes) the runner's last report before it. Derived when
	// loading; times before the runner's first report have no reveal.
	Reveals map[PlayerID][]Reveal
	// Jokers are the jokers hunter teams used, in order of use.
	Jokers []JokerUse
	// Sketches are the lines all hunter teams drew, oldest first.
	Sketches []Sketch
}

func (l Lobby) index(id PlayerID) int {
	return slices.IndexFunc(l.Players, func(p Player) bool { return p.ID == id })
}

func (l Lobby) Player(id PlayerID) (Player, bool) {
	if i := l.index(id); i >= 0 {
		return l.Players[i], true
	}
	return Player{}, false
}

func (l Lobby) PlayerByToken(hash TokenHash) (Player, bool) {
	i := slices.IndexFunc(l.Players, func(p Player) bool { return p.TokenHash == hash })
	if i < 0 {
		return Player{}, false
	}
	return l.Players[i], true
}

func (l *Lobby) Join(code JoinCode, name PlayerName, token PlayerToken, now time.Time) (Player, error) {
	team, ok := l.Game.JoinCodes.TeamFor(code)
	if !ok {
		return Player{}, ErrNotFound
	}
	if err := l.requireNotStarted(now); err != nil {
		return Player{}, err
	}
	if len(l.Players) >= maxPlayers {
		return Player{}, &RuleError{Message: "game is full"}
	}
	if slices.ContainsFunc(l.Players, func(p Player) bool { return p.Name.sameAs(name) }) {
		return Player{}, &RuleError{Message: "name is already taken in this game"}
	}
	player := Player{ID: NewPlayerID(), Name: name, Team: team, TokenHash: token.Hash(), JoinedAt: now}
	l.Players = append(l.Players, player)
	l.reschedule(now)
	return player, nil
}

// CanRegenerateRoute checks that this player may replace the game's route: a
// runner, in the lobby, while no runner is ready, with regenerations left.
func (l Lobby) CanRegenerateRoute(id PlayerID, now time.Time) error {
	if err := l.requireNotStarted(now); err != nil {
		return err
	}
	player, ok := l.Player(id)
	if !ok {
		return ErrUnauthorized
	}
	switch {
	case player.Team != Runners:
		return &RuleError{Message: "only runners can change the route"}
	case slices.ContainsFunc(l.Players, func(p Player) bool { return p.Team == Runners && p.Ready }):
		return &RuleError{Message: "all runners must be not ready to change the route"}
	case l.Game.RouteRegenerationsLeft() <= 0:
		return &RuleError{Message: "no route changes left"}
	}
	return nil
}

// RegenerateRoute replaces the game's route with next, which must differ from
// it. Callers look up next beforehand, so the rules are checked again here.
func (l *Lobby) RegenerateRoute(id PlayerID, next Route, now time.Time) error {
	if err := l.CanRegenerateRoute(id, now); err != nil {
		return err
	}
	if next == l.Game.Route {
		return &RuleError{Message: "no other route available"}
	}
	l.Game.Route = next
	l.Game.RouteRegenerations++
	return nil
}

// SetHunterTeam puts a hunter into the hunter team with this name, creating it
// if no hunter has it yet (names ignore case). Only possible before the start
// and while not ready. A team left empty disappears.
func (l *Lobby) SetHunterTeam(id PlayerID, name HunterTeamName, now time.Time) error {
	if err := l.requireNotStarted(now); err != nil {
		return err
	}
	i := l.index(id)
	if i < 0 {
		return ErrUnauthorized
	}
	switch {
	case l.Players[i].Team != Hunters:
		return &RuleError{Message: "only hunters choose a hunter team"}
	case l.Players[i].Ready:
		return &RuleError{Message: "set yourself not ready before changing your team"}
	}
	for _, p := range l.Players {
		if p.HunterTeam.sameAs(name) {
			name = p.HunterTeam // Keep the existing spelling.
			break
		}
	}
	l.Players[i].HunterTeam = name
	return nil
}

// SetReady changes a player's ready state. skipStartCheck is the development
// test override; it is a server setting, not a game rule.
func (l *Lobby) SetReady(id PlayerID, ready bool, now time.Time, skipStartCheck bool) error {
	if err := l.requireNotStarted(now); err != nil {
		return err
	}
	i := l.index(id)
	if i < 0 {
		return ErrUnauthorized
	}
	if ready && l.Players[i].Team == Hunters && l.Players[i].HunterTeam.IsZero() {
		return &RuleError{Message: "choose a hunter team to get ready"}
	}
	if ready && !skipStartCheck {
		if err := l.requireAtStart(l.Players[i], now); err != nil {
			return err
		}
	}
	l.Players[i].Ready = ready
	l.reschedule(now)
	return nil
}

// UpdatePosition stores a player's latest location. Before the start, a ready
// runner who moves away from the start becomes not ready (unless skipStartCheck).
func (l *Lobby) UpdatePosition(id PlayerID, position Position, now time.Time, skipStartCheck bool) error {
	i := l.index(id)
	if i < 0 {
		return ErrUnauthorized
	}
	player := &l.Players[i]
	player.Position = position
	if !skipStartCheck && l.Game.Status(now) != StatusRunning && player.Ready && l.requireAtStart(*player, now) != nil {
		player.Ready = false
		l.reschedule(now)
	}
	return nil
}

// requireAtStart checks that a runner recently reported a position within
// startRadiusMeters of the route's start. Hunters always pass.
func (l Lobby) requireAtStart(p Player, now time.Time) error {
	if p.Team != Runners {
		return nil
	}
	if p.Position.IsZero() || now.Sub(p.Position.RecordedAt) > startPositionMaxAge {
		return &RuleError{Message: "share your location to get ready"}
	}
	if p.Position.Coordinates.DistanceMeters(l.Game.Route.Start.Coordinates) > startRadiusMeters {
		return &RuleError{Message: fmt.Sprintf("go to within %d meters of the start to get ready", startRadiusMeters)}
	}
	return nil
}

func (l *Lobby) Leave(id PlayerID, now time.Time) error {
	if err := l.requireNotStarted(now); err != nil {
		return err
	}
	l.Players = slices.DeleteFunc(l.Players, func(p Player) bool { return p.ID == id })
	l.reschedule(now)
	return nil
}

func (l Lobby) requireNotStarted(now time.Time) error {
	if l.Game.Status(now) == StatusRunning {
		return &RuleError{Message: "game has already started"}
	}
	return nil
}

// reschedule keeps the invariant: a start is scheduled exactly while every
// player is ready and both teams have players. A running countdown is kept;
// otherwise a new one starts now.
func (l *Lobby) reschedule(now time.Time) {
	if !l.canStart() {
		l.Game.StartsAt = time.Time{}
	} else if l.Game.StartsAt.IsZero() {
		l.Game.StartsAt = now.Add(CountdownDuration)
	}
}

func (l Lobby) canStart() bool {
	teams := map[Team]bool{}
	for _, p := range l.Players {
		if !p.Ready {
			return false
		}
		teams[p.Team] = true
	}
	return teams[Runners] && teams[Hunters]
}

package game

import (
	"fmt"
	"slices"
	"time"
)

const (
	CountdownDuration = 10 * time.Second
	MaxPlayers        = 30
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
	if len(l.Players) >= MaxPlayers {
		return Player{}, &RuleError{Message: "game is full"}
	}
	if slices.ContainsFunc(l.Players, func(p Player) bool { return p.Name.SameAs(name) }) {
		return Player{}, &RuleError{Message: "name is already taken in this game"}
	}
	player := Player{ID: NewPlayerID(), Name: name, Team: team, TokenHash: token.Hash(), JoinedAt: now}
	if team == Runners {
		player.Route = randomRoute(MaxRouteRegenerations)
	}
	l.Players = append(l.Players, player)
	l.reschedule(now)
	return player, nil
}

// RegenerateRoute replaces a runner's route with a different random one.
// Only possible before the start and while not ready, at most
// MaxRouteRegenerations times.
func (l *Lobby) RegenerateRoute(id PlayerID, now time.Time) error {
	if err := l.requireNotStarted(now); err != nil {
		return err
	}
	i := l.index(id)
	if i < 0 {
		return ErrUnauthorized
	}
	player := &l.Players[i]
	switch {
	case player.Team != Runners:
		return &RuleError{Message: "only runners have a route"}
	case player.Ready:
		return &RuleError{Message: "set yourself not ready before changing your route"}
	case player.Route.RegenerationsLeft == 0:
		return &RuleError{Message: "no route changes left"}
	}
	next := randomRoute(player.Route.RegenerationsLeft - 1)
	for next.Start == player.Route.Start && next.End == player.Route.End {
		next = randomRoute(next.RegenerationsLeft)
	}
	player.Route = next
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
		if p.HunterTeam.SameAs(name) {
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
		if err := l.requireInStartCell(l.Players[i], now); err != nil {
			return err
		}
	}
	l.Players[i].Ready = ready
	l.reschedule(now)
	return nil
}

// UpdatePosition stores a player's latest location. Before the start, a ready
// runner who leaves their start cell becomes not ready (unless skipStartCheck).
func (l *Lobby) UpdatePosition(id PlayerID, position Position, now time.Time, skipStartCheck bool) error {
	i := l.index(id)
	if i < 0 {
		return ErrUnauthorized
	}
	player := &l.Players[i]
	player.Position = position
	if !skipStartCheck && l.Game.Status(now) != StatusRunning && player.Ready && l.requireInStartCell(*player, now) != nil {
		player.Ready = false
		l.reschedule(now)
	}
	return nil
}

// requireInStartCell checks that a runner recently reported a position inside
// their start cell. Hunters always pass.
func (l Lobby) requireInStartCell(p Player, now time.Time) error {
	if p.Team != Runners {
		return nil
	}
	if p.Position.IsZero() || now.Sub(p.Position.RecordedAt) > StartCellMaxAge {
		return &RuleError{Message: "share your location to get ready"}
	}
	if cell, ok := l.Game.Playground.CellAt(p.Position.Coordinates); !ok || cell != p.Route.Start {
		return &RuleError{Message: fmt.Sprintf("go to your start cell %s to get ready", p.Route.Start)}
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

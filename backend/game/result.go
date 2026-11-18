package game

import "time"

// Result is the public, coordinate-free summary of an ended game. It carries
// no credentials, positions or joker results, even internally; its events are
// the coordinate-free timeline.
type Result struct {
	SessionID         SessionID
	StartsAt, EndedAt time.Time
	Players           []ResultPlayer
	// Events is the whole timeline, oldest first.
	Events []Event
}

type ResultPlayer struct {
	ID         PlayerID
	Name       PlayerName
	Team       Team
	HunterTeam HunterTeamName
	Runner     RunnerState
}

// Result hides unfinished games just like unknown IDs. Original team and
// runner outcome stay separate; no viewing role or winner is inferred.
func (l Lobby) Result() (Result, error) {
	if l.Game.EndedAt.IsZero() {
		return Result{}, ErrNotFound
	}
	result := Result{
		SessionID: l.Game.SessionID, StartsAt: l.Game.StartsAt, EndedAt: l.Game.EndedAt,
		Players: make([]ResultPlayer, 0, len(l.Players)),
		Events:  l.timeline(l.Game.EndedAt),
	}
	for _, p := range l.Players {
		result.Players = append(result.Players, ResultPlayer{
			ID: p.ID, Name: p.Name, Team: p.Team, HunterTeam: p.HunterTeam, Runner: p.Runner,
		})
	}
	return result, nil
}

// Player finds a participant of the result, e.g. the runner of an event.
func (r Result) Player(id PlayerID) (ResultPlayer, bool) {
	for _, p := range r.Players {
		if p.ID == id {
			return p, true
		}
	}
	return ResultPlayer{}, false
}

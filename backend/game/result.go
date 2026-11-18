package game

import "time"

// Result is the public, coordinate-free summary of an ended game. It carries
// no credentials or gameplay history, even internally.
type Result struct {
	SessionID         SessionID
	StartsAt, EndedAt time.Time
	Players           []ResultPlayer
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
	}
	for _, p := range l.Players {
		result.Players = append(result.Players, ResultPlayer{
			ID: p.ID, Name: p.Name, Team: p.Team, HunterTeam: p.HunterTeam, Runner: p.Runner,
		})
	}
	return result, nil
}

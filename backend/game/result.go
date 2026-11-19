package game

import "time"

// TrackMinInterval is the minimum time between two kept points of a result
// track; the reports in between are dropped to keep the result small.
const TrackMinInterval = 5 * time.Second

// Result is the public summary of an ended game: playground, route,
// participants, outcomes, the coordinate-free timeline and every runner's and
// hunter's real track. It carries no credentials, joker results, reveals or
// sketches.
type Result struct {
	SessionID         SessionID
	StartsAt, EndedAt time.Time
	Playground        Playground
	// Route is public once the game has ended, including the stop hunters never saw.
	Route   Route
	Players []ResultPlayer
	// Events is the whole timeline, oldest first.
	Events []Event
}

type ResultPlayer struct {
	ID         PlayerID
	Name       PlayerName
	Team       Team
	HunterTeam HunterTeamName
	Runner     RunnerState
	// Track holds the real reports from start to end, oldest first, thinned to
	// TrackMinInterval. Runners and hunters always have one (possibly empty);
	// direct spectators report no positions and have none (nil).
	Track []Position
}

// Result hides unfinished games just like unknown IDs. histories are each
// player's stored reports, oldest first; only those from start to end reach
// the result. Original team and runner outcome stay separate; no viewing role
// or winner is inferred.
func (l Lobby) Result(histories map[PlayerID][]Position) (Result, error) {
	if l.Game.EndedAt.IsZero() {
		return Result{}, ErrNotFound
	}
	result := Result{
		SessionID: l.Game.SessionID, StartsAt: l.Game.StartsAt, EndedAt: l.Game.EndedAt,
		Playground: l.Game.Playground, Route: l.Game.Route,
		Players: make([]ResultPlayer, 0, len(l.Players)),
		Events:  l.timeline(l.Game.EndedAt),
	}
	for _, p := range l.Players {
		player := ResultPlayer{ID: p.ID, Name: p.Name, Team: p.Team, HunterTeam: p.HunterTeam, Runner: p.Runner}
		if p.Team == Runners || p.Team == Hunters {
			player.Track = track(histories[p.ID], l.Game.StartsAt, l.Game.EndedAt)
		}
		result.Players = append(result.Players, player)
	}
	return result, nil
}

// track keeps the reports from start to end (inclusive) that are at least
// TrackMinInterval after the previously kept one, plus the last report, so a
// track always ends where the player was last seen.
func track(history []Position, start, end time.Time) []Position {
	kept := []Position{}
	var last Position
	for _, p := range history {
		if p.RecordedAt.Before(start) || p.RecordedAt.After(end) {
			continue
		}
		if len(kept) == 0 || p.RecordedAt.Sub(kept[len(kept)-1].RecordedAt) >= TrackMinInterval {
			kept = append(kept, p)
		}
		last = p
	}
	if !last.IsZero() && kept[len(kept)-1] != last {
		kept = append(kept, last)
	}
	return kept
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

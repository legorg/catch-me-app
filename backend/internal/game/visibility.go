package game

import "time"

// VisiblePlayer is what one player may see of another. Hidden fields are zero.
type VisiblePlayer struct {
	ID    PlayerID
	Name  PlayerName
	Team  Team
	Ready bool
	Route RunnerRoute
	// Position is live for oneself and teammates, or a reveal (RevealedAt set)
	// when a hunter looks at a runner.
	Position   Position
	RevealedAt time.Time
}

// VisibleTo applies the visibility rules for viewer:
//   - Everyone sees their own position at any time.
//   - Runners see their teammates' routes; hunters see no routes.
//   - While running, teammates see each other's latest positions.
//   - While running, hunters see each runner's last revealed position.
//   - Runners never see hunters' positions.
func (l Lobby) VisibleTo(viewer Player, now time.Time) []VisiblePlayer {
	running := l.Game.Status(now) == StatusRunning
	visible := make([]VisiblePlayer, 0, len(l.Players))
	for _, p := range l.Players {
		v := VisiblePlayer{ID: p.ID, Name: p.Name, Team: p.Team, Ready: p.Ready}
		if viewer.Team == Runners {
			v.Route = p.Route // Hunters have none.
		}
		switch {
		case p.ID == viewer.ID, running && p.Team == viewer.Team:
			v.Position = p.Position
		case running && viewer.Team == Hunters && !p.Reveal.IsZero():
			v.Position, v.RevealedAt = p.Reveal.Position, p.Reveal.At
		}
		visible = append(visible, v)
	}
	return visible
}

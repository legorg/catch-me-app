package game

import "time"

// visiblePlayer is what one player may see of another. Hidden fields are zero.
type visiblePlayer struct {
	ID         PlayerID
	Name       PlayerName
	Team       Team
	HunterTeam HunterTeamName
	Ready      bool
	Route      RunnerRoute
	// Position is the live position of oneself and teammates.
	Position Position
	// Reveals are all reveals of a runner, oldest first; only set when a hunter
	// looks at a runner.
	Reveals []Reveal
}

// VisibleTo applies the visibility rules for viewer:
//   - Everyone sees their own position at any time.
//   - Runners see their teammates' routes; hunters see no routes.
//   - Hunters of the same hunter team see each other's latest positions at any
//     time; other hunter teams never.
//   - While running, runners see each other's latest positions.
//   - While running, hunters see all reveals of each runner.
//   - Runners never see hunters' positions.
func (l Lobby) VisibleTo(viewer Player, now time.Time) []visiblePlayer {
	running := l.Game.Status(now) == StatusRunning
	visible := make([]visiblePlayer, 0, len(l.Players))
	for _, p := range l.Players {
		v := visiblePlayer{ID: p.ID, Name: p.Name, Team: p.Team, HunterTeam: p.HunterTeam, Ready: p.Ready}
		if viewer.Team == Runners {
			v.Route = p.Route // Hunters have none.
		}
		sameHunterTeam := viewer.Team == Hunters && p.Team == Hunters && !viewer.HunterTeam.IsZero() && p.HunterTeam == viewer.HunterTeam
		switch {
		case p.ID == viewer.ID, sameHunterTeam, running && viewer.Team == Runners && p.Team == Runners:
			v.Position = p.Position
		case running && viewer.Team == Hunters:
			v.Reveals = l.Reveals[p.ID]
		}
		visible = append(visible, v)
	}
	return visible
}

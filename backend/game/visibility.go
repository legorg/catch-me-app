package game

import "time"

// visiblePlayer is what one player may see of another. Hidden fields are zero.
type visiblePlayer struct {
	ID         PlayerID
	Name       PlayerName
	Team       Team
	HunterTeam HunterTeamName
	Ready      bool
	// Position is the live position of oneself and teammates.
	Position Position
	// Reveals are what hunters see of a runner, oldest first; only set when a
	// hunter looks at a runner.
	Reveals []HunterReveal
	// A catch replaces live positions and reveals for every viewer immediately.
	CaughtPosition Position
	Runner         RunnerState
}

// VisibleTo applies the visibility rules for viewer:
//   - Finished runners expose only their finish status, never a position or reveal.
//   - Active players see their own position; caught runners expose the catch instead.
//   - Hunters of the same hunter team see each other's latest positions at any
//     time; other hunter teams never.
//   - While running, runners see each other's latest positions.
//   - While running, hunters see all reveals of each runner, with the runners'
//     jokers applied (a zoneOnly reveal shows only the cell).
//   - Runners never see hunters' positions.
func (l Lobby) VisibleTo(viewer Player, now time.Time) []visiblePlayer {
	running := l.Game.Status(now) == StatusRunning
	visible := make([]visiblePlayer, 0, len(l.Players))
	for _, p := range l.Players {
		v := visiblePlayer{ID: p.ID, Name: p.Name, Team: p.Team, HunterTeam: p.HunterTeam, Ready: p.Ready, Runner: p.Runner}
		if p.Team == Runners && p.Runner.Status() == RunnerFinished {
			visible = append(visible, v)
			continue
		}
		if p.Team == Runners && p.Runner.Status() == RunnerCaught {
			v.CaughtPosition = p.Position
			visible = append(visible, v)
			continue
		}
		sameHunterTeam := viewer.Team == Hunters && p.Team == Hunters && !viewer.HunterTeam.IsZero() && p.HunterTeam == viewer.HunterTeam
		switch {
		case p.ID == viewer.ID, sameHunterTeam, running && viewer.Team == Runners && p.Team == Runners:
			v.Position = p.Position
		case running && viewer.Team == Hunters:
			v.Reveals = l.revealsForHunters(p.ID)
		}
		visible = append(visible, v)
	}
	return visible
}

package game

import (
	"slices"
	"time"
)

// zoneAlertMaxAge is how old the opponent's latest report may be, compared to
// the new report, to count as being in the same cell at the same time.
const zoneAlertMaxAge = 60 * time.Second

// ZoneAlert tells a runner that a hunter was in their grid cell. Unlike joker
// results it is stored when raised, so it can be notified exactly once. It
// never holds coordinates; the hunter team is the hunter's, which cannot
// change after the start.
type ZoneAlert struct {
	Runner PlayerID
	Hunter PlayerID
	Cell   Cell
	At     time.Time
}

// raiseZoneAlerts checks a new real report of an active runner or hunter
// against the opponents' latest reports. Each runner is alerted at most once
// per hunter and game.
func (l *Lobby) raiseZoneAlerts(reporter Player, now time.Time) {
	if !l.Game.JokerSettings.ZoneAlert.Enabled || l.Game.Status(now) != StatusRunning || !reporter.active() {
		return
	}
	cell, inside := l.Game.Playground.CellAt(reporter.Position.Coordinates)
	if !inside {
		return
	}
	opponents := Hunters
	if reporter.Team == Hunters {
		opponents = Runners
	}
	for _, other := range l.Players {
		if other.Team != opponents || !other.active() || other.Position.IsZero() ||
			other.Position.RecordedAt.Before(l.Game.StartsAt) || now.Sub(other.Position.RecordedAt) > zoneAlertMaxAge {
			continue
		}
		if c, ok := l.Game.Playground.CellAt(other.Position.Coordinates); !ok || c != cell {
			continue
		}
		runner, hunter := reporter.ID, other.ID
		if reporter.Team == Hunters {
			runner, hunter = other.ID, reporter.ID
		}
		if !slices.ContainsFunc(l.ZoneAlerts, func(a ZoneAlert) bool { return a.Runner == runner && a.Hunter == hunter }) {
			l.ZoneAlerts = append(l.ZoneAlerts, ZoneAlert{Runner: runner, Hunter: hunter, Cell: cell, At: now})
		}
	}
}

// ZoneAlertsFor are the viewer's own zone alerts, oldest first. Only the
// alerted runner sees them; hunters and other runners are never told.
func (l Lobby) ZoneAlertsFor(viewer Player) []ZoneAlert {
	if viewer.Team != Runners {
		return nil
	}
	var alerts []ZoneAlert
	for _, a := range l.ZoneAlerts {
		if a.Runner == viewer.ID {
			alerts = append(alerts, a)
		}
	}
	return alerts
}

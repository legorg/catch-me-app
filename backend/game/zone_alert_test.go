package game

import (
	"slices"
	"testing"
	"time"
)

// reportIn reports a position in the middle of the cell.
func reportIn(t *testing.T, l *Lobby, id PlayerID, cell Cell, at time.Time) {
	t.Helper()
	position, _ := NewPosition(cellCenter(l.Game.Playground, cell), 5, at)
	if err := l.UpdatePosition(id, position, at, false); err != nil {
		t.Fatal(err)
	}
}

func TestZoneAlertOncePerRunnerAndHunter(t *testing.T) {
	l, ada, bob, dan, eve, running := jokerLobby(t)
	cy := l.Players[1]
	here, there := Cell{column: 1, row: 1}, Cell{column: 4, row: 4}
	reportIn(t, &l, ada.ID, here, running.Add(time.Minute))
	reportIn(t, &l, cy.ID, there, running.Add(time.Minute))
	at := running.Add(90 * time.Second)
	reportIn(t, &l, bob.ID, here, at)
	reportIn(t, &l, bob.ID, there, at.Add(time.Second))
	reportIn(t, &l, bob.ID, here, at.Add(2*time.Second))
	reportIn(t, &l, dan.ID, here, at.Add(3*time.Second))
	reportIn(t, &l, ada.ID, here, at.Add(4*time.Second))
	want := []ZoneAlert{
		{Runner: ada.ID, Hunter: bob.ID, Cell: here, At: at},
		{Runner: cy.ID, Hunter: bob.ID, Cell: there, At: at.Add(time.Second)},
		{Runner: ada.ID, Hunter: dan.ID, Cell: here, At: at.Add(3 * time.Second)},
	}
	if !slices.Equal(l.ZoneAlerts, want) {
		t.Fatalf("alerts = %+v; want %+v", l.ZoneAlerts, want)
	}
	// The runner's report checks the hunters' reports too.
	reportIn(t, &l, eve.ID, there, at.Add(5*time.Second))
	reportIn(t, &l, ada.ID, there, at.Add(6*time.Second))
	got := l.ZoneAlertsFor(ada)
	if len(got) != 3 || got[2] != (ZoneAlert{Runner: ada.ID, Hunter: eve.ID, Cell: there, At: at.Add(6 * time.Second)}) {
		t.Fatalf("Ada's alerts = %+v", got)
	}
}

func TestZoneAlertNeedsAFreshOpponentReport(t *testing.T) {
	l, ada, bob, _, _, running := jokerLobby(t)
	here := Cell{column: 1, row: 1}
	reportIn(t, &l, ada.ID, here, running)
	reportIn(t, &l, bob.ID, here, running.Add(zoneAlertMaxAge+time.Second))
	if len(l.ZoneAlerts) != 0 {
		t.Fatalf("a stale runner report raised an alert: %+v", l.ZoneAlerts)
	}
	reportIn(t, &l, ada.ID, here, running.Add(2*zoneAlertMaxAge+time.Second))
	if len(l.ZoneAlerts) != 1 {
		t.Fatalf("a report exactly %s old must count: %+v", zoneAlertMaxAge, l.ZoneAlerts)
	}
}

func TestZoneAlertIgnoresReportsBeforeTheStart(t *testing.T) {
	l, ada, bob, _, _, running := jokerLobby(t)
	// Everyone reported at the route start just before the game ran.
	startCell, _ := l.Game.Playground.CellAt(l.Game.Route.Start.Coordinates)
	reportIn(t, &l, bob.ID, startCell, running)
	if len(l.ZoneAlerts) != 0 {
		t.Fatalf("pre-start runner reports raised alerts: %+v", l.ZoneAlerts)
	}
	reportIn(t, &l, ada.ID, startCell, running.Add(time.Second))
	want := []ZoneAlert{{Runner: ada.ID, Hunter: bob.ID, Cell: startCell, At: running.Add(time.Second)}}
	if !slices.Equal(l.ZoneAlerts, want) {
		t.Fatalf("pre-start hunter reports raised alerts: %+v", l.ZoneAlerts)
	}
}

func TestZoneAlertNotRaised(t *testing.T) {
	here := Cell{column: 1, row: 1}
	cases := map[string]func(t *testing.T, l *Lobby, ada Player, running time.Time){
		"disabled": func(t *testing.T, l *Lobby, _ Player, _ time.Time) {
			l.Game.JokerSettings.ZoneAlert.Enabled = false
		},
		"caught runner": func(t *testing.T, l *Lobby, ada Player, running time.Time) {
			if err := l.MarkCaught(ada.ID, running.Add(30*time.Second)); err != nil {
				t.Fatal(err)
			}
		},
		"runner outside the playground": func(t *testing.T, l *Lobby, ada Player, running time.Time) {
			outside, _ := ParseCoordinates(0, 0)
			position, _ := NewPosition(outside, 5, running.Add(30*time.Second))
			if err := l.UpdatePosition(ada.ID, position, running.Add(30*time.Second), false); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			l, ada, bob, _, _, running := jokerLobby(t)
			reportIn(t, &l, ada.ID, here, running.Add(20*time.Second))
			setup(t, &l, ada, running)
			reportIn(t, &l, bob.ID, here, running.Add(40*time.Second))
			if len(l.ZoneAlerts) != 0 {
				t.Fatalf("alerts = %+v", l.ZoneAlerts)
			}
		})
	}
}

func TestZoneAlertsAreVisibleToTheAlertedRunnerOnly(t *testing.T) {
	l, ada, bob, _, _, running := jokerLobby(t)
	cy := l.Players[1]
	watcher := joinSpectator(t, &l, "Wanda", start)
	here := Cell{column: 1, row: 1}
	reportIn(t, &l, ada.ID, here, running.Add(time.Second))
	reportIn(t, &l, bob.ID, here, running.Add(2*time.Second))
	if got := l.ZoneAlertsFor(ada); len(got) != 1 {
		t.Fatalf("Ada must see her alert: %+v", got)
	}
	for _, viewer := range []Player{cy, bob, watcher} {
		if got := l.ZoneAlertsFor(viewer); len(got) != 0 {
			t.Fatalf("%s sees another runner's alert: %+v", viewer.Name, got)
		}
	}
	if err := l.MarkCaught(ada.ID, running.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	caught, _ := l.Player(ada.ID)
	if got := l.ZoneAlertsFor(caught); len(got) != 1 {
		t.Fatalf("a caught runner keeps her alerts: %+v", got)
	}
}

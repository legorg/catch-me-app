package game

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestCompassShowsOnlyTheDirectionsFromTheHunter(t *testing.T) {
	l, runner, bob, dan, eve, running := jokerLobby(t)
	at := running.Add(time.Minute)
	center := l.Game.Playground.Center()
	place := func(id PlayerID, latitude, longitude float64) {
		t.Helper()
		c, err := ParseCoordinates(center.latitude+latitude, center.longitude+longitude)
		if err != nil {
			t.Fatal(err)
		}
		p, _ := NewPosition(c, 5, at.Add(-time.Second))
		l.Players[l.index(id)].Position = p
	}
	place(bob.ID, 0, 0)
	place(runner.ID, 0.01, 0) // north of Bob
	place(l.Players[1].ID, 0, 0.01)
	l.Players[l.index(dan.ID)].Position = Position{}

	var rule *RuleError
	if err := l.UseJoker(dan.ID, Compass{}, at); !errors.As(err, &rule) {
		t.Fatalf("a hunter without a position must be refused: %v", err)
	}
	if got := l.JokersFor(bob, at)[2]; !got.Available || len(got.Uses) != 0 {
		t.Fatalf("a refused compass stays available: %+v", got)
	}
	if err := l.UseJoker(bob.ID, Compass{}, at); err != nil {
		t.Fatal(err)
	}
	if err := l.UseJoker(bob.ID, Compass{}, at.Add(time.Second)); !errors.As(err, &rule) {
		t.Fatalf("a team can use the compass once: %v", err)
	}
	uses := l.JokersFor(dan, at)[2].Uses
	if len(uses) != 1 {
		t.Fatalf("expected one compass use: %+v", uses)
	}
	got, ok := uses[0].(CompassUse)
	if !ok || got.Origin != center || !got.At.Equal(at) || !slices.Equal(got.Bearings, []int{0, 90}) {
		t.Fatalf("the team must see the directions from Bob's position, sorted: %+v", got)
	}
	if other := l.JokersFor(eve, at)[2]; !other.Available || len(other.Uses) != 0 {
		t.Fatalf("another hunter team sees nothing of it: %+v", other)
	}
	for _, j := range l.JokersFor(runner, at) {
		if j.Kind == JokerCompass {
			t.Fatalf("runners are not told: %+v", j)
		}
	}
}

func TestCompassBearingsRoundAndWrap(t *testing.T) {
	origin, _ := ParseCoordinates(0, 0)
	slightlyWest, _ := ParseCoordinates(1, -0.0001) // 359.99 degrees rounds to 0, not 360
	east, _ := ParseCoordinates(0, 1)
	positions := []JokerPosition{{Position: Position{Coordinates: east}}, {Position: Position{Coordinates: slightlyWest}}}
	if got := CompassBearings(origin, positions); !slices.Equal(got, []int{0, 90}) {
		t.Fatalf("bearings: %v", got)
	}
}

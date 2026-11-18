package game

import (
	"slices"
	"time"
)

const (
	minSketchPoints = 2
	maxSketchPoints = 500
	// maxSketchesPerTeam limits how many sketches one hunter team keeps.
	maxSketchesPerTeam = 200
)

type SketchID struct{ value string }

func newSketchID() SketchID { return SketchID{value: randomHex(16)} }

func ParseSketchID(id string) (SketchID, error) {
	if !idPattern.MatchString(id) {
		return SketchID{}, &ValidationError{Message: "sketch ID must contain 32 lowercase hexadecimal characters"}
	}
	return SketchID{value: id}, nil
}

func (id SketchID) String() string { return id.value }

// SketchLine is a freehand line of minSketchPoints to maxSketchPoints points.
type SketchLine struct{ points []Coordinates }

func NewSketchLine(points []Coordinates) (SketchLine, error) {
	if len(points) < minSketchPoints || len(points) > maxSketchPoints {
		return SketchLine{}, &ValidationError{Message: "a sketch must have 2 to 500 points"}
	}
	return SketchLine{points: slices.Clone(points)}, nil
}

func (s SketchLine) Points() []Coordinates { return slices.Clone(s.points) }

// Sketch is a line a hunter team drew on the map. It belongs to the team, not
// to the player who drew it.
type Sketch struct {
	ID        SketchID
	Team      HunterTeamName
	Line      SketchLine
	CreatedAt time.Time
}

// AddSketch adds a line to the player's hunter team while the game is running.
func (l *Lobby) AddSketch(id PlayerID, line SketchLine, now time.Time) error {
	me, err := l.sketcher(id, now)
	if err != nil {
		return err
	}
	if len(l.teamSketches(me.HunterTeam)) >= maxSketchesPerTeam {
		return &RuleError{Message: "your hunter team has reached the sketch limit, clear the map first"}
	}
	l.Sketches = append(l.Sketches, Sketch{ID: newSketchID(), Team: me.HunterTeam, Line: line, CreatedAt: now})
	return nil
}

// ClearSketches removes all sketches of the player's hunter team. Any team
// member may clear them, whoever drew them.
func (l *Lobby) ClearSketches(id PlayerID, now time.Time) error {
	me, err := l.sketcher(id, now)
	if err != nil {
		return err
	}
	l.Sketches = slices.DeleteFunc(slices.Clone(l.Sketches), func(s Sketch) bool { return s.Team.sameAs(me.HunterTeam) })
	return nil
}

// SketchesFor lists the sketches of the viewer's hunter team, oldest first,
// while the game is running. Runners and other hunter teams never get them.
func (l Lobby) SketchesFor(viewer Player, now time.Time) []Sketch {
	if viewer.Team != Hunters || viewer.HunterTeam.IsZero() || l.Game.Status(now) != StatusRunning {
		return nil
	}
	return l.teamSketches(viewer.HunterTeam)
}

// sketcher returns the player if they may draw: a hunter in a hunter team while running.
func (l Lobby) sketcher(id PlayerID, now time.Time) (Player, error) {
	me, ok := l.Player(id)
	switch {
	case !ok:
		return Player{}, ErrUnauthorized
	case me.Team != Hunters || me.HunterTeam.IsZero():
		return Player{}, &RuleError{Message: "only hunters in a hunter team can sketch"}
	case l.Game.Status(now) != StatusRunning:
		return Player{}, &RuleError{Message: "sketches are only possible while the game is running"}
	}
	return me, nil
}

func (l Lobby) teamSketches(team HunterTeamName) []Sketch {
	var sketches []Sketch
	for _, s := range l.Sketches {
		if s.Team.sameAs(team) {
			sketches = append(sketches, s)
		}
	}
	return sketches
}

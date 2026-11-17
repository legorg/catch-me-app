package game

import "time"

// JokerFakePosition makes the next reveal of the runner who used it show a chosen point instead.
const JokerFakePosition JokerKind = "fakePosition"

// FakePosition chooses the fakePosition joker with the point hunters see instead.
type FakePosition struct{ Point Coordinates }

func (FakePosition) Kind() JokerKind { return JokerFakePosition }

func (c FakePosition) apply(l Lobby, me Player, now time.Time) (JokerUse, error) {
	if _, inside := l.Game.Playground.CellAt(c.Point); !inside {
		return nil, &ValidationError{Message: "the point must lie inside the playground"}
	}
	return FakePositionUse{Runner: me.ID, At: now, Point: c.Point}, nil
}

// FakePositionUse is a used fakePosition joker with the point hunters see instead.
type FakePositionUse struct {
	Runner PlayerID
	At     time.Time
	Point  Coordinates
}

var _ revealJokerUse = FakePositionUse{}

func (FakePositionUse) Kind() JokerKind     { return JokerFakePosition }
func (u FakePositionUse) UsedAt() time.Time { return u.At }
func (FakePositionUse) jokerUse()           {}
func (u FakePositionUse) runner() PlayerID  { return u.Runner }

// reveal looks like a normal reveal at the chosen point: it keeps the accuracy
// and report time of the real position, only the coordinates are replaced.
func (u FakePositionUse) reveal(r Reveal, _ Playground) HunterReveal {
	faked := r.Position
	faked.Coordinates = u.Point
	return PositionReveal{At: r.At, Position: faked}
}

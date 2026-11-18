package game

import "time"

// Finish permanently retires a runner at the stop landmark. The development
// override is the same server setting used by Ready, never a client choice.
func (l *Lobby) Finish(id PlayerID, now time.Time, skipPositionCheck bool) error {
	i := l.index(id)
	if i < 0 {
		return ErrUnauthorized
	}
	p := &l.Players[i]
	if p.Team != Runners || l.Game.Status(now) != StatusRunning {
		return &RuleError{Message: "only runners in a running game can finish"}
	}
	if !p.active() {
		return p.Runner.end(RunnerFinished, now)
	}
	if !skipPositionCheck {
		if err := requireNearLandmark(p.Position, l.Game.Route.Stop.Coordinates, now); err != nil {
			return err
		}
	}
	return p.Runner.end(RunnerFinished, now)
}

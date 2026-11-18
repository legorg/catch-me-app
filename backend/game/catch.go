package game

import "time"

// MarkCaught permanently retires the requesting runner on trust alone. Their
// last reported position becomes the public catch location; no GPS fix or
// nearby hunter is required. Repeated requests preserve the original catch.
func (l *Lobby) MarkCaught(id PlayerID, now time.Time) error {
	player, err := l.requirePlayer(id)
	if err != nil {
		return err
	}
	if player.Team != Runners {
		return &RuleError{Message: "only runners can mark themselves caught"}
	}
	if l.Game.Status(now) == StatusEnded && player.Runner.Status() == RunnerCaught {
		return nil // A lost response to the final catch must remain retryable.
	}
	if l.Game.Status(now) != StatusRunning {
		return &RuleError{Message: "you can only mark yourself caught while the game is running"}
	}
	if err := player.Runner.end(RunnerCaught, now); err != nil {
		return err
	}
	l.endIfAllRunnersDone(now)
	return nil
}

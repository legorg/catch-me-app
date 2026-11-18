package game

import "time"

// endIfAllRunnersDone runs after a successful runner outcome, in the same
// transaction. Hunters do not count; an empty game must never end by itself.
func (l *Lobby) endIfAllRunnersDone(now time.Time) {
	if l.Game.Status(now) != StatusRunning {
		return
	}
	hasRunner := false
	for _, p := range l.Players {
		if p.Team != Runners {
			continue
		}
		hasRunner = true
		if p.Runner.Status() == RunnerActive {
			return
		}
	}
	if hasRunner {
		l.Game.EndedAt = now.UTC()
	}
}

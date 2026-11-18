package game

import "time"

// MarkCaught permanently retires the requesting runner on trust alone. Their
// last reported position becomes the public catch location; no GPS fix or
// nearby hunter is required. Repeated requests preserve the original catch.
func (l *Lobby) MarkCaught(id PlayerID, now time.Time) error {
	i := l.index(id)
	if i < 0 {
		return ErrUnauthorized
	}
	player := &l.Players[i]
	if player.Team != Runners {
		return &RuleError{Message: "only runners can mark themselves caught"}
	}
	if l.Game.Status(now) != StatusRunning {
		return &RuleError{Message: "you can only mark yourself caught while the game is running"}
	}
	if player.CaughtAt.IsZero() {
		player.CaughtAt = now
	}
	return nil
}

package service

import (
	"context"
	"fmt"

	"catch-me/backend/game"
)

// TimelineView is the game's event timeline as seen by one of its players.
type TimelineView struct {
	PlayerView
	Events []game.Event
}

// Timeline returns the game's events up to now to an authenticated spectator.
func (s *GameService) Timeline(ctx context.Context, token game.PlayerToken) (TimelineView, error) {
	view, err := s.State(ctx, token)
	if err != nil {
		return TimelineView{}, err
	}
	events, err := view.Lobby.TimelineFor(view.Me, view.Now)
	if err != nil {
		return TimelineView{}, fmt.Errorf("load timeline: %w", err)
	}
	return TimelineView{PlayerView: view, Events: events}, nil
}

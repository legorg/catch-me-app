package service

import (
	"context"
	"fmt"

	"catch-me/backend/game"
)

// Result is public by full game ID, without a player token or joining.
func (s *GameService) Result(ctx context.Context, id game.SessionID) (game.Result, error) {
	result, err := s.repository.Result(ctx, id)
	if err != nil {
		return game.Result{}, fmt.Errorf("load game result: %w", err)
	}
	return result, nil
}

package service

import (
	"context"
	"fmt"
	"time"

	"catch-me/backend/internal/game"
)

// GameRepository is the small persistence boundary used by this service.
// It exposes domain values, not Ent entities or SQL errors.
type GameRepository interface {
	Create(context.Context, game.Game) error
	Get(context.Context, game.SessionID) (game.Game, error)
}

type GameService struct{ repository GameRepository }

func NewGameService(repository GameRepository) *GameService {
	return &GameService{repository: repository}
}

func (s *GameService) Create(ctx context.Context, center game.Coordinates, cellSize game.CellSize) (game.Game, error) {
	playground, err := game.NewPlayground(center, cellSize)
	if err != nil {
		return game.Game{}, err
	}
	saved := game.Game{SessionID: game.NewSessionID(), CreatedAt: time.Now().UTC(), Playground: playground}
	if err := s.repository.Create(ctx, saved); err != nil {
		return game.Game{}, fmt.Errorf("create game: %w", err)
	}
	return saved, nil
}

func (s *GameService) Get(ctx context.Context, id game.SessionID) (game.Game, error) {
	saved, err := s.repository.Get(ctx, id)
	if err != nil {
		return game.Game{}, fmt.Errorf("get game: %w", err)
	}
	return saved, nil
}

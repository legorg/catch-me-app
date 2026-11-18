package db

import (
	"context"
	"errors"
	"fmt"

	"catch-me/backend/game"
	"gorm.io/gorm"
)

// Result loads only the game and participants, never positions, reveals,
// jokers or sketches. Unfinished games are excluded by the database query.
func (s *Store) Result(ctx context.Context, id game.SessionID) (game.Result, error) {
	var result game.Result
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row gameRow
		err := tx.Preload("Players", func(q *gorm.DB) *gorm.DB { return q.Order("joined_at, id") }).
			Preload("Players.Runner").Where("id = ? AND ended_at IS NOT NULL", id.String()).Take(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return game.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("select result: %w", err)
		}
		lobby, err := parseLobby(row)
		if err != nil {
			return fmt.Errorf("parse stored result: %v", err)
		}
		result, err = lobby.Result()
		return err
	})
	return result, err
}

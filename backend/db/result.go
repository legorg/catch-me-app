package db

import (
	"context"
	"errors"
	"fmt"

	"catch-me/backend/game"
	"gorm.io/gorm"
)

// Result loads only the game, participants and used jokers, never positions,
// reveals or sketches. Unfinished games are excluded by the database query.
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
		jokers, err := selectJokers(tx, row.ID)
		if err != nil {
			return err
		}
		// Without position histories the uses show nothing; only their kinds,
		// users and times reach the result, as timeline events.
		for _, j := range jokers {
			use, err := parseJoker(j, nil, nil, lobby)
			if err != nil {
				return err
			}
			lobby.Jokers = append(lobby.Jokers, use)
		}
		result, err = lobby.Result()
		return err
	})
	return result, err
}

package db

import (
	"context"
	"errors"
	"fmt"

	"catch-me/backend/game"
	"gorm.io/gorm"
)

// Result loads the game, participants, used jokers, zone alerts and the runners' and
// hunters' positions from start to end, never reveals or sketches. Unfinished
// games are excluded by the database query.
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
		// Only the uses' kinds, users and times reach the result, as timeline
		// events. A trap's trigger time is a timeline event too.
		for _, j := range jokers {
			use, err := parseJoker(j)
			if err != nil {
				return fmt.Errorf("parse stored joker: %v", err)
			}
			lobby.Jokers = append(lobby.Jokers, use)
		}
		if lobby.ZoneAlerts, err = selectZoneAlerts(tx, row.ID); err != nil {
			return err
		}
		histories, err := gameTracks(tx, lobby)
		if err != nil {
			return err
		}
		result, err = lobby.Result(histories)
		return err
	})
	return result, err
}

// gameTracks returns the runners' and hunters' positions recorded from start
// to end, oldest first. Reports from the lobby and countdown are not read.
func gameTracks(tx *gorm.DB, lobby game.Lobby) (map[game.PlayerID][]game.Position, error) {
	var ids []string
	for _, p := range lobby.Players {
		if p.Team == game.Runners || p.Team == game.Hunters {
			ids = append(ids, p.ID.String())
		}
	}
	histories := map[game.PlayerID][]game.Position{}
	if len(ids) == 0 {
		return histories, nil
	}
	var rows []positionRow
	err := tx.Where("player_id IN ? AND recorded_at >= ? AND recorded_at <= ?", ids, lobby.Game.StartsAt, lobby.Game.EndedAt).
		Order("player_id, recorded_at, id").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("select positions: %w", err)
	}
	for _, row := range rows {
		id, err := game.ParsePlayerID(row.PlayerID)
		if err != nil {
			return nil, fmt.Errorf("parse stored position: %v", err)
		}
		position, err := parsePosition(row)
		if err != nil {
			return nil, fmt.Errorf("parse stored position: %v", err)
		}
		histories[id] = append(histories[id], position)
	}
	return histories, nil
}

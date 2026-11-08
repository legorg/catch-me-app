package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"catch-me/backend/internal/game"
)

// GameRepository is the small persistence boundary used by this service.
// It exposes domain values, not Ent entities or SQL errors.
type GameRepository interface {
	Create(context.Context, game.Game) error
	Get(context.Context, game.SessionID) (game.Game, error)
	// UpdateLobby* load a lobby, apply change, and save atomically. A nil change only loads.
	UpdateLobbyByJoinCode(context.Context, game.JoinCode, func(*game.Lobby) error) (game.Lobby, error)
	UpdateLobbyByToken(context.Context, game.TokenHash, func(*game.Lobby) error) (game.Lobby, error)
}

type GameService struct {
	repository GameRepository
	now        func() time.Time
}

func NewGameService(repository GameRepository) *GameService {
	return &GameService{repository: repository, now: func() time.Time { return time.Now().UTC() }}
}

// PlayerView is a lobby as seen by one of its players.
type PlayerView struct {
	Lobby game.Lobby
	Me    game.Player
	Now   time.Time
}

func (s *GameService) Create(ctx context.Context, center game.Coordinates, cellSize game.CellSize) (game.Game, error) {
	playground, err := game.NewPlayground(center, cellSize)
	if err != nil {
		return game.Game{}, err
	}
	// Collisions are extremely unlikely; retry a few times rather than fail.
	for attempt := 0; ; attempt++ {
		saved := game.Game{SessionID: game.NewSessionID(), CreatedAt: s.now(), Playground: playground, JoinCodes: game.NewJoinCodes()}
		err := s.repository.Create(ctx, saved)
		if errors.Is(err, game.ErrJoinCodeTaken) && attempt < 3 {
			continue
		}
		if err != nil {
			return game.Game{}, fmt.Errorf("create game: %w", err)
		}
		return saved, nil
	}
}

func (s *GameService) Get(ctx context.Context, id game.SessionID) (game.Game, error) {
	saved, err := s.repository.Get(ctx, id)
	if err != nil {
		return game.Game{}, fmt.Errorf("get game: %w", err)
	}
	return saved, nil
}

// Join adds a player to the team of the join code. The returned token is
// shown to the player once; only its hash is stored.
func (s *GameService) Join(ctx context.Context, code game.JoinCode, name game.PlayerName) (PlayerView, game.PlayerToken, error) {
	now, token := s.now(), game.NewPlayerToken()
	var me game.Player
	lobby, err := s.repository.UpdateLobbyByJoinCode(ctx, code, func(l *game.Lobby) error {
		var err error
		me, err = l.Join(code, name, token, now)
		return err
	})
	if err != nil {
		return PlayerView{}, game.PlayerToken{}, fmt.Errorf("join game: %w", err)
	}
	return PlayerView{Lobby: lobby, Me: me, Now: now}, token, nil
}

func (s *GameService) State(ctx context.Context, token game.PlayerToken) (PlayerView, error) {
	return s.updateAsPlayer(ctx, token, nil)
}

func (s *GameService) SetReady(ctx context.Context, token game.PlayerToken, ready bool) (PlayerView, error) {
	return s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		return l.SetReady(me.ID, ready, now)
	})
}

func (s *GameService) Leave(ctx context.Context, token game.PlayerToken) error {
	_, err := s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		return l.Leave(me.ID, now)
	})
	return err
}

func (s *GameService) updateAsPlayer(ctx context.Context, token game.PlayerToken, change func(*game.Lobby, game.Player, time.Time) error) (PlayerView, error) {
	now, hash := s.now(), token.Hash()
	var apply func(*game.Lobby) error
	if change != nil {
		apply = func(l *game.Lobby) error {
			me, ok := l.PlayerByToken(hash)
			if !ok {
				return game.ErrUnauthorized
			}
			return change(l, me, now)
		}
	}
	lobby, err := s.repository.UpdateLobbyByToken(ctx, hash, apply)
	if errors.Is(err, game.ErrNotFound) {
		return PlayerView{}, game.ErrUnauthorized
	}
	if err != nil {
		return PlayerView{}, fmt.Errorf("update game: %w", err)
	}
	me, ok := lobby.PlayerByToken(hash)
	if !ok && change == nil {
		return PlayerView{}, game.ErrUnauthorized
	}
	return PlayerView{Lobby: lobby, Me: me, Now: now}, nil
}

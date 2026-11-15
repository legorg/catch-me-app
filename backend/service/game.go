package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"catch-me/backend/game"
)

// landmarkTimeout bounds all lookups for one edge; it stays below the server write timeout.
const landmarkTimeout = 8 * time.Second

// GameRepository is the small persistence boundary used by this service.
// It exposes domain values, not database rows or SQL errors.
type GameRepository interface {
	Create(context.Context, game.Game) error
	// UpdateLobby* load a lobby as of now (including reveals), apply change, and
	// save atomically. A nil change only loads.
	UpdateLobbyByJoinCode(context.Context, game.JoinCode, time.Time, func(*game.Lobby) error) (game.Lobby, error)
	UpdateLobbyByToken(context.Context, game.TokenHash, time.Time, func(*game.Lobby) error) (game.Lobby, error)
	// DeletePositionsBefore deletes the positions of games that started (or, if
	// never started, were created) before cutoff.
	DeletePositionsBefore(context.Context, time.Time) error
}

// LandmarkFinder lists landmarks located inside bounds (one grid cell).
type LandmarkFinder interface {
	Landmarks(context.Context, game.Bounds) ([]game.Landmark, error)
}

type GameService struct {
	repository             GameRepository
	landmarks              LandmarkFinder
	now                    func() time.Time
	skipStartPositionCheck bool
}

// NewGameService creates the service. skipStartPositionCheck is the
// development test override and must be false for real games.
func NewGameService(repository GameRepository, landmarks LandmarkFinder, skipStartPositionCheck bool) *GameService {
	return &GameService{repository: repository, landmarks: landmarks, now: func() time.Time { return time.Now().UTC() }, skipStartPositionCheck: skipStartPositionCheck}
}

// route finds a landmark on the south edge and one on the north edge, in parallel.
func (s *GameService) route(ctx context.Context, playground game.Playground) (game.Route, error) {
	var route game.Route
	var startErr, stopErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		route.Start, startErr = s.edgeLandmark(ctx, playground, playground.SouthEdgeCells())
	}()
	go func() {
		defer wg.Done()
		route.Stop, stopErr = s.edgeLandmark(ctx, playground, playground.NorthEdgeCells())
	}()
	wg.Wait()
	if err := errors.Join(startErr, stopErr); err != nil {
		return game.Route{}, err
	}
	return route, nil
}

// edgeLandmark picks a random landmark inside one of the cells, trying them in random order.
func (s *GameService) edgeLandmark(ctx context.Context, playground game.Playground, cells []game.Cell) (game.Landmark, error) {
	rand.Shuffle(len(cells), func(i, j int) { cells[i], cells[j] = cells[j], cells[i] })
	ctx, cancel := context.WithTimeout(ctx, landmarkTimeout)
	defer cancel()
	for _, cell := range cells {
		found, err := s.landmarks.Landmarks(ctx, playground.CellBounds(cell))
		if err != nil {
			return game.Landmark{}, fmt.Errorf("%w: cell %s: %v", game.ErrLandmarksUnavailable, cell, err)
		}
		if len(found) > 0 {
			return found[rand.IntN(len(found))], nil
		}
	}
	return game.Landmark{}, game.ErrNoLandmark
}

// PlayerView is a lobby as seen by one of its players.
type PlayerView struct {
	Lobby                      game.Lobby
	Me                         game.Player
	Now                        time.Time
	StartPositionCheckDisabled bool
}

func (s *GameService) Create(ctx context.Context, center game.Coordinates, cellSize game.CellSize) (game.Game, error) {
	playground, err := game.NewPlayground(center, cellSize)
	if err != nil {
		return game.Game{}, err
	}
	route, err := s.route(ctx, playground)
	if err != nil {
		return game.Game{}, err
	}
	slog.Info("route created",
		"start", route.Start.Title, "startLat", route.Start.Coordinates.Latitude(), "startLon", route.Start.Coordinates.Longitude(),
		"stop", route.Stop.Title, "stopLat", route.Stop.Coordinates.Latitude(), "stopLon", route.Stop.Coordinates.Longitude(),
		"distanceMeters", route.DistanceMeters())
	// Collisions are extremely unlikely; retry a few times rather than fail.
	for attempt := 0; ; attempt++ {
		saved := game.Game{SessionID: game.NewSessionID(), CreatedAt: s.now(), Playground: playground, JoinCodes: game.NewJoinCodes(), Route: route}
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

// Join adds a player to the team of the join code. The returned token is
// shown to the player once; only its hash is stored.
func (s *GameService) Join(ctx context.Context, code game.JoinCode, name game.PlayerName) (PlayerView, game.PlayerToken, error) {
	now, token := s.now(), game.NewPlayerToken()
	var me game.Player
	lobby, err := s.repository.UpdateLobbyByJoinCode(ctx, code, now, func(l *game.Lobby) error {
		var err error
		me, err = l.Join(code, name, token, now)
		return err
	})
	if err != nil {
		return PlayerView{}, game.PlayerToken{}, fmt.Errorf("join game: %w", err)
	}
	return PlayerView{Lobby: lobby, Me: me, Now: now, StartPositionCheckDisabled: s.skipStartPositionCheck}, token, nil
}

func (s *GameService) State(ctx context.Context, token game.PlayerToken) (PlayerView, error) {
	return s.updateAsPlayer(ctx, token, nil)
}

func (s *GameService) SetReady(ctx context.Context, token game.PlayerToken, ready bool) (PlayerView, error) {
	return s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		return l.SetReady(me.ID, ready, now, s.skipStartPositionCheck)
	})
}

func (s *GameService) RegenerateRoute(ctx context.Context, token game.PlayerToken) (PlayerView, error) {
	return s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		return l.RegenerateRoute(me.ID, now)
	})
}

// SetHunterTeam joins the hunter team with this name, creating it if needed.
func (s *GameService) SetHunterTeam(ctx context.Context, token game.PlayerToken, name game.HunterTeamName) (PlayerView, error) {
	return s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		return l.SetHunterTeam(me.ID, name, now)
	})
}

// UpdatePosition stores the player's latest location, timestamped with server time.
func (s *GameService) UpdatePosition(ctx context.Context, token game.PlayerToken, coordinates game.Coordinates, accuracyMeters float64) (PlayerView, error) {
	return s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		position, err := game.NewPosition(coordinates, accuracyMeters, now)
		if err != nil {
			return err
		}
		return l.UpdatePosition(me.ID, position, now, s.skipStartPositionCheck)
	})
}

func (s *GameService) Leave(ctx context.Context, token game.PlayerToken) error {
	_, err := s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		return l.Leave(me.ID, now)
	})
	return err
}

// DeleteExpiredPositions enforces game.PositionRetention.
func (s *GameService) DeleteExpiredPositions(ctx context.Context) error {
	return s.repository.DeletePositionsBefore(ctx, s.now().Add(-game.PositionRetention))
}

func (s *GameService) updateAsPlayer(ctx context.Context, token game.PlayerToken, change func(*game.Lobby, game.Player, time.Time) error) (PlayerView, error) {
	now, hash := s.now(), token.Hash()
	lobby, err := s.repository.UpdateLobbyByToken(ctx, hash, now, func(l *game.Lobby) error {
		if change == nil {
			return nil
		}
		me, ok := l.PlayerByToken(hash)
		if !ok {
			return game.ErrUnauthorized
		}
		return change(l, me, now)
	})
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
	return PlayerView{Lobby: lobby, Me: me, Now: now, StartPositionCheckDisabled: s.skipStartPositionCheck}, nil
}

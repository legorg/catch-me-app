package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2" // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used -- non-security gameplay randomness
	"slices"
	"sync"
	"time"

	"catch-me/backend/game"
)

// Time budget of the requests that look up landmarks, from the inside out:
// one Places call (googleplaces.CallTimeout) < one edge < the whole request <
// the server write timeout (cmd/server) < the app's wait for these requests.
const (
	edgeLookupTimeout   = 6 * time.Second
	routeRequestTimeout = 8 * time.Second
)

// GameRepository is the small persistence boundary used by this service.
// It exposes domain values, not database rows or SQL errors.
type GameRepository interface {
	Create(context.Context, game.Game) error
	// Result returns the credential-free summary of an ended game, with each
	// runner's and hunter's track from start to end; unknown and unfinished
	// games both return game.ErrNotFound.
	Result(context.Context, game.SessionID) (game.Result, error)
	// UpdateLobby* load a lobby as of now (including reveals), apply change, and
	// save atomically. A nil change only loads.
	UpdateLobbyByJoinCode(context.Context, game.JoinCode, time.Time, func(*game.Lobby) error) (game.Lobby, error)
	UpdateLobbyByToken(context.Context, game.TokenHash, time.Time, func(*game.Lobby) error) (game.Lobby, error)
	// DeleteGamesBefore deletes games that ended before cutoff, or, if they
	// never ended, started (or, if never started, were created) before cutoff,
	// with all their data, plus orphaned rows.
	DeleteGamesBefore(context.Context, time.Time) error
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
	// Tests shorten these.
	edgeTimeout, routeTimeout time.Duration
}

// NewGameService creates the service. skipStartPositionCheck is the
// development override for start/finish position checks and must be false for real games.
func NewGameService(repository GameRepository, landmarks LandmarkFinder, skipStartPositionCheck bool) *GameService {
	return &GameService{
		repository: repository, landmarks: landmarks, now: func() time.Time { return time.Now().UTC() },
		skipStartPositionCheck: skipStartPositionCheck, edgeTimeout: edgeLookupTimeout, routeTimeout: routeRequestTimeout,
	}
}

// route finds a landmark on the south edge and one on the north edge, in
// parallel. When replacing previous, each edge avoids its current landmark and
// keeps it only if the edge has no other one.
func (s *GameService) route(ctx context.Context, playground game.Playground, previous game.Route) (game.Route, error) {
	var route game.Route
	var startErr, stopErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		route.Start, startErr = s.edgeLandmark(ctx, playground, "south", playground.SouthEdgeCells(), previous.Start)
	}()
	go func() {
		defer wg.Done()
		route.Stop, stopErr = s.edgeLandmark(ctx, playground, "north", playground.NorthEdgeCells(), previous.Stop)
	}()
	wg.Wait()
	if previous != (game.Route{}) {
		if errors.Is(startErr, game.ErrNoLandmark) {
			route.Start, startErr = previous.Start, nil
		}
		if errors.Is(stopErr, game.ErrNoLandmark) {
			route.Stop, stopErr = previous.Stop, nil
		}
	}
	if err := errors.Join(startErr, stopErr); err != nil {
		return game.Route{}, err
	}
	return route, nil
}

// edgeLandmark picks a random landmark other than avoid inside one of the
// cells, trying them in random order. It logs how long the lookup took and how
// many cells it queried, to tune the time budget.
func (s *GameService) edgeLandmark(ctx context.Context, playground game.Playground, edge string, cells []game.Cell, avoid game.Landmark) (landmark game.Landmark, err error) {
	began, queried := time.Now(), 0
	defer func() {
		slog.Info("landmark lookup", "edge", edge, "cells", queried, "found", err == nil,
			"durationMs", time.Since(began).Milliseconds())
	}()
	rand.Shuffle(len(cells), func(i, j int) { cells[i], cells[j] = cells[j], cells[i] })
	ctx, cancel := context.WithTimeout(ctx, s.edgeTimeout)
	defer cancel()
	for _, cell := range cells {
		queried++
		found, err := s.landmarks.Landmarks(ctx, playground.CellBounds(cell))
		if err != nil {
			return game.Landmark{}, fmt.Errorf("%w: cell %s: %v", game.ErrLandmarksUnavailable, cell, err)
		}
		found = slices.DeleteFunc(slices.Clone(found), func(l game.Landmark) bool { return l == avoid })
		if len(found) > 0 {
			return found[rand.IntN(len(found))], nil
		}
	}
	return game.Landmark{}, game.ErrNoLandmark
}

func logRoute(message string, route game.Route, began time.Time) {
	slog.Info(message,
		"start", route.Start.Title, "startLat", route.Start.Coordinates.Latitude(), "startLon", route.Start.Coordinates.Longitude(),
		"stop", route.Stop.Title, "stopLat", route.Stop.Coordinates.Latitude(), "stopLon", route.Stop.Coordinates.Longitude(),
		"distanceMeters", route.DistanceMeters(), "durationMs", time.Since(began).Milliseconds())
}

// PlayerView is a lobby as seen by one of its players.
type PlayerView struct {
	Lobby                      game.Lobby
	Me                         game.Player
	Now                        time.Time
	StartPositionCheckDisabled bool
}

// ChosenRoute is a start and stop picked by the game creator on the map
// instead of generated landmarks.
type ChosenRoute struct{ Start, Stop game.Coordinates }

// Create saves a new game. Without a chosen route it looks up landmarks for
// one; a chosen route must lie on the playground's edge rows and cannot be
// regenerated.
func (s *GameService) Create(ctx context.Context, center game.Coordinates, cellSize game.CellSize, gridSize game.GridSize, revealInterval game.RevealInterval, jokers game.JokerSettings, chosen *ChosenRoute) (game.Game, error) {
	playground, err := game.NewPlayground(center, cellSize, gridSize)
	if err != nil {
		return game.Game{}, err
	}
	began := time.Now()
	ctx, cancel := context.WithTimeout(ctx, s.routeTimeout)
	defer cancel()
	var route game.Route
	if chosen != nil {
		if route, err = game.NewCustomRoute(playground, chosen.Start, chosen.Stop); err != nil {
			return game.Game{}, err
		}
		logRoute("route chosen", route, began)
	} else {
		if route, err = s.route(ctx, playground, game.Route{}); err != nil {
			return game.Game{}, err
		}
		logRoute("route created", route, began)
	}
	// Collisions are extremely unlikely; retry a few times rather than fail.
	for attempt := 0; ; attempt++ {
		saved := game.Game{SessionID: game.NewSessionID(), CreatedAt: s.now(), Playground: playground, RevealInterval: revealInterval, JokerSettings: jokers, JoinCodes: game.NewJoinCodes(), Route: route, CustomRoute: chosen != nil}
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

// Finish validates the authenticated runner at the destination, using server time.
func (s *GameService) Finish(ctx context.Context, token game.PlayerToken) (PlayerView, error) {
	return s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		return l.Finish(me.ID, now, s.skipStartPositionCheck)
	})
}

// MarkCaught retires the authenticated runner without checking hunter proximity.
func (s *GameService) MarkCaught(ctx context.Context, token game.PlayerToken) (PlayerView, error) {
	return s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		return l.MarkCaught(me.ID, now)
	})
}

// RegenerateRoute replaces the game's route with a different one. The landmark
// lookup runs outside the database transaction, so the rules are checked
// before it and again when saving.
func (s *GameService) RegenerateRoute(ctx context.Context, token game.PlayerToken) (PlayerView, error) {
	began := time.Now()
	ctx, cancel := context.WithTimeout(ctx, s.routeTimeout)
	defer cancel()
	current, err := s.State(ctx, token)
	if err != nil {
		return PlayerView{}, err
	}
	if err := current.Lobby.CanRegenerateRoute(current.Me.ID, current.Now); err != nil {
		return PlayerView{}, fmt.Errorf("update game: %w", err)
	}
	next, err := s.route(ctx, current.Lobby.Game.Playground, current.Lobby.Game.Route)
	if err != nil {
		return PlayerView{}, err
	}
	view, err := s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		return l.RegenerateRoute(me.ID, next, now)
	})
	if err == nil {
		logRoute("route regenerated", next, began)
	}
	return view, err
}

// SetHunterTeam joins the hunter team with this name, creating it if needed.
func (s *GameService) SetHunterTeam(ctx context.Context, token game.PlayerToken, name game.HunterTeamName) (PlayerView, error) {
	return s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		return l.SetHunterTeam(me.ID, name, now)
	})
}

// UseJoker spends a joker of the player's team.
func (s *GameService) UseJoker(ctx context.Context, token game.PlayerToken, choice game.JokerChoice) (PlayerView, error) {
	return s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		return l.UseJoker(me.ID, choice, now)
	})
}

// AddSketch adds a line to the map of the player's hunter team.
func (s *GameService) AddSketch(ctx context.Context, token game.PlayerToken, line game.SketchLine) (PlayerView, error) {
	return s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		return l.AddSketch(me.ID, line, now)
	})
}

// ClearSketches removes all lines of the player's hunter team.
func (s *GameService) ClearSketches(ctx context.Context, token game.PlayerToken) (PlayerView, error) {
	return s.updateAsPlayer(ctx, token, func(l *game.Lobby, me game.Player, now time.Time) error {
		return l.ClearSketches(me.ID, now)
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

// DeleteExpiredGames enforces game.GameRetention: whole games with all their data.
func (s *GameService) DeleteExpiredGames(ctx context.Context) error {
	return s.repository.DeleteGamesBefore(ctx, s.now().Add(-game.GameRetention))
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

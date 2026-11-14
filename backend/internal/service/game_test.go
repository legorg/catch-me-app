package service

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"catch-me/backend/internal/game"
)

type memoryRepository struct {
	lobbies    map[game.SessionID]game.Lobby
	err        error
	createErrs []error
	calls      int
	cutoff     time.Time
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{lobbies: make(map[game.SessionID]game.Lobby)}
}

func (r *memoryRepository) Create(ctx context.Context, saved game.Game) error {
	r.calls++
	if len(r.createErrs) > 0 {
		err := r.createErrs[0]
		r.createErrs = r.createErrs[1:]
		return err
	}
	if r.err != nil {
		return r.err
	}
	r.lobbies[saved.SessionID] = game.Lobby{Game: saved}
	return nil
}

func (r *memoryRepository) DeletePositionsBefore(ctx context.Context, cutoff time.Time) error {
	r.calls++
	r.cutoff = cutoff
	return r.err
}

func (r *memoryRepository) UpdateLobbyByJoinCode(ctx context.Context, code game.JoinCode, now time.Time, change func(*game.Lobby) error) (game.Lobby, error) {
	return r.update(func(l game.Lobby) bool { _, ok := l.Game.JoinCodes.TeamFor(code); return ok }, change)
}

func (r *memoryRepository) UpdateLobbyByToken(ctx context.Context, hash game.TokenHash, now time.Time, change func(*game.Lobby) error) (game.Lobby, error) {
	return r.update(func(l game.Lobby) bool { _, ok := l.PlayerByToken(hash); return ok }, change)
}

func (r *memoryRepository) update(match func(game.Lobby) bool, change func(*game.Lobby) error) (game.Lobby, error) {
	r.calls++
	if r.err != nil {
		return game.Lobby{}, r.err
	}
	for id, stored := range r.lobbies {
		if !match(stored) {
			continue
		}
		lobby := game.Lobby{Game: stored.Game, Players: slices.Clone(stored.Players)}
		if change != nil {
			if err := change(&lobby); err != nil {
				return game.Lobby{}, err
			}
			r.lobbies[id] = lobby
		}
		return lobby, nil
	}
	return game.Lobby{}, game.ErrNotFound
}

func newService(r GameRepository, now time.Time) *GameService {
	s := NewGameService(r, false)
	s.now = func() time.Time { return now }
	return s
}

func TestCreate(t *testing.T) {
	s := NewGameService(newMemoryRepository(), false)
	first, err := s.Create(context.Background(), game.Coordinates{}, game.DefaultCellSize())
	if err != nil {
		t.Fatal(err)
	}
	size, _ := game.ParseCellSize(1500)
	second, err := s.Create(context.Background(), game.Coordinates{}, size)
	if err != nil {
		t.Fatal(err)
	}
	if first.SessionID == second.SessionID || first.Playground.CellSize().Meters() != 2000 || second.Playground.CellSize() != size {
		t.Fatal("incorrect session IDs or sizes")
	}
	if first.CreatedAt.IsZero() || first.CreatedAt.Location() != time.UTC {
		t.Fatalf("unexpected creation time: %v", first.CreatedAt)
	}
	if first.JoinCodes.Runner == (game.JoinCode{}) || first.JoinCodes.Runner == first.JoinCodes.Hunter {
		t.Fatal("join codes missing")
	}
}

func TestDeleteExpiredPositions(t *testing.T) {
	r := newMemoryRepository()
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	if err := newService(r, now).DeleteExpiredPositions(context.Background()); err != nil || !r.cutoff.Equal(now.Add(-game.PositionRetention)) {
		t.Fatalf("wrong retention cutoff %v: %v", r.cutoff, err)
	}
}

func TestCreateRetriesJoinCodeCollisions(t *testing.T) {
	r := newMemoryRepository()
	r.createErrs = []error{game.ErrJoinCodeTaken, game.ErrJoinCodeTaken}
	if _, err := NewGameService(r, false).Create(context.Background(), game.Coordinates{}, game.DefaultCellSize()); err != nil || r.calls != 3 {
		t.Fatalf("collision not retried: %v after %d calls", err, r.calls)
	}
	r.createErrs = []error{game.ErrJoinCodeTaken, game.ErrJoinCodeTaken, game.ErrJoinCodeTaken, game.ErrJoinCodeTaken}
	if _, err := NewGameService(r, false).Create(context.Background(), game.Coordinates{}, game.DefaultCellSize()); !errors.Is(err, game.ErrJoinCodeTaken) {
		t.Fatal("endless retry")
	}
}

func TestInvalidPlaygroundDoesNotCallDatabase(t *testing.T) {
	r := newMemoryRepository()
	center, _ := game.ParseCoordinates(0, 180)
	_, err := NewGameService(r, false).Create(context.Background(), center, game.DefaultCellSize())
	var validation *game.ValidationError
	if !errors.As(err, &validation) || r.calls != 0 {
		t.Fatalf("expected validation error without database call: %v", err)
	}
}

func TestLobbyFlow(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	r := newMemoryRepository()
	s := newService(r, now)
	created, err := s.Create(ctx, game.Coordinates{}, game.DefaultCellSize())
	if err != nil {
		t.Fatal(err)
	}
	ada, _ := game.ParsePlayerName("Ada")
	runner, runnerToken, err := s.Join(ctx, created.JoinCodes.Runner, ada)
	if err != nil || runner.Me.Team != game.Runners || runner.Me.Name != ada || len(runner.Lobby.Players) != 1 {
		t.Fatalf("join failed: %+v %v", runner, err)
	}
	regenerated, err := s.RegenerateRoute(ctx, runnerToken)
	if err != nil || regenerated.Me.Route.IsZero() || regenerated.Me.Route.RegenerationsLeft != game.MaxRouteRegenerations-1 {
		t.Fatalf("route not regenerated: %+v %v", regenerated.Me.Route, err)
	}
	bob, _ := game.ParsePlayerName("Bob")
	_, hunterToken, err := s.Join(ctx, created.JoinCodes.Hunter, bob)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetReady(ctx, runnerToken, true); err == nil {
		t.Fatal("runner got ready without a position")
	}
	start := regenerated.Lobby.Game.Playground.CellCenter(regenerated.Me.Route.Start)
	if _, err := s.UpdatePosition(ctx, runnerToken, start, -1); err == nil {
		t.Fatal("invalid accuracy accepted")
	}
	moved, err := s.UpdatePosition(ctx, runnerToken, start, 8)
	if err != nil || moved.Me.Position.RecordedAt != now {
		t.Fatalf("position not stored with server time: %+v %v", moved.Me.Position, err)
	}
	if _, err := s.SetReady(ctx, runnerToken, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RegenerateRoute(ctx, runnerToken); err == nil {
		t.Fatal("ready runner changed route")
	}
	if _, err := s.RegenerateRoute(ctx, hunterToken); err == nil {
		t.Fatal("hunter changed route")
	}
	view, err := s.SetReady(ctx, hunterToken, true)
	if err == nil {
		t.Fatal("hunter got ready without a hunter team")
	}
	wolves, _ := game.ParseHunterTeamName("Wolves")
	if _, err := s.SetHunterTeam(ctx, runnerToken, wolves); err == nil {
		t.Fatal("runner joined a hunter team")
	}
	if view, err = s.SetHunterTeam(ctx, hunterToken, wolves); err != nil || view.Me.HunterTeam != wolves {
		t.Fatalf("hunter team not set: %+v %v", view.Me, err)
	}
	view, err = s.SetReady(ctx, hunterToken, true)
	if err != nil || view.Lobby.Game.Status(now) != game.StatusCountdown || !view.Me.Ready || view.Me.Team != game.Hunters {
		t.Fatalf("countdown not started: %+v %v", view, err)
	}
	if err := s.Leave(ctx, hunterToken); err != nil {
		t.Fatal(err)
	}
	if _, err := s.State(ctx, hunterToken); !errors.Is(err, game.ErrUnauthorized) {
		t.Fatal("left player can still read state")
	}
	view, err = s.State(ctx, runnerToken)
	if err != nil || len(view.Lobby.Players) != 1 || view.Lobby.Game.Status(now) != game.StatusLobby {
		t.Fatalf("leave not applied: %+v %v", view, err)
	}
	if _, err := s.SetReady(ctx, game.NewPlayerToken(), true); !errors.Is(err, game.ErrUnauthorized) {
		t.Fatal("unknown token accepted")
	}
	unknown, _ := game.ParseJoinCode("ZZZZZZ")
	if _, _, err := s.Join(ctx, unknown, bob); !errors.Is(err, game.ErrNotFound) {
		t.Fatal("unknown code accepted")
	}
}

func TestRepositoryErrorsRemainInspectable(t *testing.T) {
	storageError := errors.New("storage unavailable")
	r := newMemoryRepository()
	r.err = storageError
	s := NewGameService(r, false)
	if _, err := s.Create(context.Background(), game.Coordinates{}, game.DefaultCellSize()); !errors.Is(err, storageError) {
		t.Fatalf("create error lost: %v", err)
	}
	if _, err := s.State(context.Background(), game.NewPlayerToken()); !errors.Is(err, storageError) {
		t.Fatalf("state error lost: %v", err)
	}
}

package service

import (
	"context"
	"errors"
	"testing"

	"catch-me/backend/internal/game"
)

type memoryRepository struct {
	games map[game.SessionID]game.Game
	err   error
	calls int
}

func (r *memoryRepository) Create(ctx context.Context, saved game.Game) error {
	r.calls++
	if r.err != nil {
		return r.err
	}
	r.games[saved.SessionID] = saved
	return nil
}

func (r *memoryRepository) Get(ctx context.Context, id game.SessionID) (game.Game, error) {
	r.calls++
	if r.err != nil {
		return game.Game{}, r.err
	}
	saved, ok := r.games[id]
	if !ok {
		return game.Game{}, game.ErrNotFound
	}
	return saved, nil
}

func TestCreateAndGet(t *testing.T) {
	repository := &memoryRepository{games: make(map[game.SessionID]game.Game)}
	s := NewGameService(repository)
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
	if first.CreatedAt.IsZero() || first.CreatedAt.Location().String() != "UTC" {
		t.Fatalf("unexpected creation time: %v", first.CreatedAt)
	}
	loaded, err := s.Get(context.Background(), first.SessionID)
	if err != nil || loaded != first {
		t.Fatalf("incorrect loaded game: %+v %v", loaded, err)
	}
}

func TestInvalidPlaygroundDoesNotCallDatabase(t *testing.T) {
	r := &memoryRepository{}
	center, _ := game.ParseCoordinates(0, 180)
	_, err := NewGameService(r).Create(context.Background(), center, game.DefaultCellSize())
	var validation *game.ValidationError
	if !errors.As(err, &validation) || r.calls != 0 {
		t.Fatalf("expected validation error without database call: %v", err)
	}
}

func TestRepositoryErrorsRemainInspectable(t *testing.T) {
	storageError := errors.New("storage unavailable")
	r := &memoryRepository{err: storageError}
	s := NewGameService(r)
	id := game.NewSessionID()
	if _, err := s.Create(context.Background(), game.Coordinates{}, game.DefaultCellSize()); !errors.Is(err, storageError) {
		t.Fatalf("create error lost: %v", err)
	}
	if _, err := s.Get(context.Background(), id); !errors.Is(err, storageError) {
		t.Fatalf("get error lost: %v", err)
	}
	r.err = game.ErrNotFound
	if _, err := s.Get(context.Background(), id); !errors.Is(err, game.ErrNotFound) {
		t.Fatalf("not-found lost: %v", err)
	}
}

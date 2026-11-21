package service

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"catch-me/backend/game"
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

func (r *memoryRepository) Result(ctx context.Context, id game.SessionID) (game.Result, error) {
	r.calls++
	if r.err != nil {
		return game.Result{}, r.err
	}
	lobby, ok := r.lobbies[id]
	if !ok {
		return game.Result{}, game.ErrNotFound
	}
	return lobby.Result(nil)
}

func (r *memoryRepository) DeleteGamesBefore(ctx context.Context, cutoff time.Time) error {
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
	s := NewGameService(r, centerLandmarks{}, false)
	s.now = func() time.Time { return now }
	return s
}

func TestCreate(t *testing.T) {
	s := NewGameService(newMemoryRepository(), centerLandmarks{}, false)
	first, err := s.Create(context.Background(), game.Coordinates{}, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil)
	if err != nil {
		t.Fatal(err)
	}
	size, _ := game.ParseCellSize(1500)
	second, err := s.Create(context.Background(), game.Coordinates{}, size, game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil)
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
	if first.JokerSettings != game.DefaultJokerSettings() || second.JokerSettings != game.DefaultJokerSettings() {
		t.Fatal("new games must default to one use per joker kind")
	}
}

func TestDeleteExpiredGames(t *testing.T) {
	r := newMemoryRepository()
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	if err := newService(r, now).DeleteExpiredGames(context.Background()); err != nil || !r.cutoff.Equal(now.Add(-game.GameRetention)) {
		t.Fatalf("wrong retention cutoff %v: %v", r.cutoff, err)
	}
}

func TestCreateKeepsExplicitJokerSettings(t *testing.T) {
	for _, settings := range []game.JokerSettings{{}, game.DefaultJokerSettings()} {
		r := newMemoryRepository()
		saved, err := NewGameService(r, centerLandmarks{}, false).Create(context.Background(), game.Coordinates{}, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), settings, nil)
		if err != nil {
			t.Fatal(err)
		}
		if saved.JokerSettings != settings || r.lobbies[saved.SessionID].Game.JokerSettings != settings {
			t.Fatal("creation must keep explicit settings, including all-zero allowances")
		}
	}
}

func TestCreateRetriesJoinCodeCollisions(t *testing.T) {
	r := newMemoryRepository()
	r.createErrs = []error{game.ErrJoinCodeTaken, game.ErrJoinCodeTaken}
	if _, err := NewGameService(r, centerLandmarks{}, false).Create(context.Background(), game.Coordinates{}, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil); err != nil || r.calls != 3 {
		t.Fatalf("collision not retried: %v after %d calls", err, r.calls)
	}
	r.createErrs = []error{game.ErrJoinCodeTaken, game.ErrJoinCodeTaken, game.ErrJoinCodeTaken, game.ErrJoinCodeTaken}
	if _, err := NewGameService(r, centerLandmarks{}, false).Create(context.Background(), game.Coordinates{}, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil); !errors.Is(err, game.ErrJoinCodeTaken) {
		t.Fatal("endless retry")
	}
}

func TestInvalidPlaygroundDoesNotCallDatabase(t *testing.T) {
	r := newMemoryRepository()
	center, _ := game.ParseCoordinates(0, 180)
	_, err := NewGameService(r, centerLandmarks{}, false).Create(context.Background(), center, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil)
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
	created, err := s.Create(ctx, game.Coordinates{}, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil)
	if err != nil {
		t.Fatal(err)
	}
	ada, _ := game.ParsePlayerName("Ada")
	runner, runnerToken, err := s.Join(ctx, created.JoinCodes.Runner, ada)
	if err != nil || runner.Me.Team != game.Runners || runner.Me.Name != ada || len(runner.Lobby.Players) != 1 {
		t.Fatalf("join failed: %+v %v", runner, err)
	}
	regenerated, err := s.RegenerateRoute(ctx, runnerToken)
	if err != nil || regenerated.Lobby.Game.Route == created.Route || regenerated.Lobby.Game.RouteRegenerationsLeft() != game.MaxRouteRegenerations-1 {
		t.Fatalf("route not regenerated: %+v %v", regenerated.Lobby.Game, err)
	}
	created.Route = regenerated.Lobby.Game.Route
	bob, _ := game.ParsePlayerName("Bob")
	_, hunterToken, err := s.Join(ctx, created.JoinCodes.Hunter, bob)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetReady(ctx, runnerToken, true); err == nil {
		t.Fatal("runner got ready without a position")
	}
	start := created.Route.Start.Coordinates
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
	unknown, _ := game.ParseJoinCode("000000")
	if _, _, err := s.Join(ctx, unknown, bob); !errors.Is(err, game.ErrNotFound) {
		t.Fatal("unknown code accepted")
	}
}

func TestRepositoryErrorsRemainInspectable(t *testing.T) {
	storageError := errors.New("storage unavailable")
	r := newMemoryRepository()
	r.err = storageError
	s := NewGameService(r, centerLandmarks{}, false)
	if _, err := s.Create(context.Background(), game.Coordinates{}, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil); !errors.Is(err, storageError) {
		t.Fatalf("create error lost: %v", err)
	}
	if _, err := s.State(context.Background(), game.NewPlayerToken()); !errors.Is(err, storageError) {
		t.Fatalf("state error lost: %v", err)
	}
}

// centerLandmarks "finds" the center of every cell it is asked about.
type centerLandmarks struct{}

func (centerLandmarks) Landmarks(_ context.Context, cell game.Bounds) ([]game.Landmark, error) {
	return []game.Landmark{{Coordinates: boundsCenter(cell)}}, nil
}

// fakeLandmarks returns those of found inside the requested bounds, like the
// Places client, and records the requests.
type fakeLandmarks struct {
	found []game.Landmark
	err   error
	mu    sync.Mutex
	rows  []game.Bounds
}

func (f *fakeLandmarks) Landmarks(_ context.Context, row game.Bounds) ([]game.Landmark, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = append(f.rows, row)
	var inside []game.Landmark
	for _, l := range f.found {
		if row.Contains(l.Coordinates) {
			inside = append(inside, l)
		}
	}
	return inside, f.err
}

// edgeSights are landmarks titled "Sight" at the centers of the given cells.
func edgeSights(p game.Playground, cells ...game.Cell) []game.Landmark {
	sights := make([]game.Landmark, len(cells))
	for i, cell := range cells {
		sights[i] = game.Landmark{Title: "Sight", Coordinates: boundsCenter(p.CellBounds(cell))}
	}
	return sights
}

func TestCreateRouteNeedsLandmarks(t *testing.T) {
	center, _ := game.ParseCoordinates(52.52, 13.405)
	p, _ := game.NewPlayground(center, game.DefaultCellSize(), game.DefaultGridSize())
	// Every edge cell has a landmark, so the first lookup per edge finds one.
	finder := &fakeLandmarks{found: edgeSights(p, append(p.SouthEdgeCells(), p.NorthEdgeCells()...)...)}
	created, err := NewGameService(newMemoryRepository(), finder, false).Create(context.Background(), center, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil)
	if err != nil || created.Route.Start.Title != "Sight" || created.Route.Stop.Title != "Sight" || len(finder.rows) != 2 {
		t.Fatalf("route %+v rows %d err %v", created.Route, len(finder.rows), err)
	}
	inEdge := func(cells []game.Cell, row game.Bounds) bool {
		for _, cell := range cells {
			if created.Playground.CellBounds(cell) == row {
				return true
			}
		}
		return false
	}
	south, north := created.Playground.SouthEdgeCells(), created.Playground.NorthEdgeCells()
	if !(inEdge(south, finder.rows[0]) && inEdge(north, finder.rows[1]) || inEdge(north, finder.rows[0]) && inEdge(south, finder.rows[1])) {
		t.Fatal("expected one lookup in a south edge cell and one in a north edge cell")
	}

	_, err = NewGameService(newMemoryRepository(), &fakeLandmarks{err: errors.New("down")}, false).Create(context.Background(), center, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil)
	if !errors.Is(err, game.ErrLandmarksUnavailable) {
		t.Fatalf("lookup failure: %v", err)
	}
	_, err = NewGameService(newMemoryRepository(), &fakeLandmarks{}, false).Create(context.Background(), center, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil)
	if !errors.Is(err, game.ErrNoLandmark) {
		t.Fatalf("no landmark: %v", err)
	}
}

// cornerLandmarks "finds" the north-west corner of every rectangle it is asked
// about; for a rotated cell that lies outside the cell.
type cornerLandmarks struct{}

func (cornerLandmarks) Landmarks(_ context.Context, cell game.Bounds) ([]game.Landmark, error) {
	corner, _ := game.ParseCoordinates(cell.North, cell.West)
	return []game.Landmark{{Coordinates: corner}}, nil
}

func TestCreateRotatedRouteKeepsLandmarksInTheirCell(t *testing.T) {
	ctx := context.Background()
	center, _ := game.ParseCoordinates(52.52, 13.405)
	rotation, _ := game.ParseRotation(45)
	created, err := NewGameService(newMemoryRepository(), centerLandmarks{}, false).Create(ctx, center, game.DefaultCellSize(), game.DefaultGridSize(), rotation, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil)
	if err != nil {
		t.Fatal(err)
	}
	p := created.Playground
	if p.Rotation() != rotation {
		t.Fatalf("rotation = %d", p.Rotation().Degrees())
	}
	start, _ := p.CellAt(created.Route.Start.Coordinates)
	stop, _ := p.CellAt(created.Route.Stop.Coordinates)
	if !slices.Contains(p.SouthEdgeCells(), start) || !slices.Contains(p.NorthEdgeCells(), stop) {
		t.Fatalf("route %s -> %s is not on the rotated edge rows", start, stop)
	}
	_, err = NewGameService(newMemoryRepository(), cornerLandmarks{}, false).Create(ctx, center, game.DefaultCellSize(), game.DefaultGridSize(), rotation, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil)
	if !errors.Is(err, game.ErrNoLandmark) {
		t.Fatalf("landmarks outside the rotated cells must be dropped: %v", err)
	}
}

func TestCreateChosenRoute(t *testing.T) {
	ctx := context.Background()
	center, _ := game.ParseCoordinates(52.52, 13.405)
	p, _ := game.NewPlayground(center, game.DefaultCellSize(), game.DefaultGridSize())
	middle := func(b game.Bounds) game.Coordinates {
		c, _ := game.ParseCoordinates((b.North+b.South)/2, (b.East+b.West)/2)
		return c
	}
	chosen := ChosenRoute{Start: middle(p.CellBounds(p.SouthEdgeCells()[1])), Stop: middle(p.CellBounds(p.NorthEdgeCells()[4]))}
	finder := &fakeLandmarks{err: errors.New("must not be called")}
	s := newService(newMemoryRepository(), time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC))
	s.landmarks = finder
	created, err := s.Create(ctx, center, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), &chosen)
	if err != nil {
		t.Fatal(err)
	}
	want := game.Route{Start: game.Landmark{Coordinates: chosen.Start}, Stop: game.Landmark{Coordinates: chosen.Stop}}
	if created.Route != want || !created.CustomRoute || len(finder.rows) != 0 {
		t.Fatalf("chosen route not used: %+v custom %v lookups %d", created.Route, created.CustomRoute, len(finder.rows))
	}
	ada, _ := game.ParsePlayerName("Ada")
	_, token, _ := s.Join(ctx, created.JoinCodes.Runner, ada)
	var rule *game.RuleError
	if _, err := s.RegenerateRoute(ctx, token); !errors.As(err, &rule) || len(finder.rows) != 0 {
		t.Fatalf("a chosen route must not be regenerated: %v", err)
	}

	swapped := ChosenRoute{Start: chosen.Stop, Stop: chosen.Start}
	var validation *game.ValidationError
	if _, err := s.Create(ctx, center, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), &swapped); !errors.As(err, &validation) {
		t.Fatalf("route outside the edge rows accepted: %v", err)
	}
}

func TestRegenerateRouteNeedsAnotherLandmark(t *testing.T) {
	ctx := context.Background()
	center, _ := game.ParseCoordinates(52.52, 13.405)
	p, _ := game.NewPlayground(center, game.DefaultCellSize(), game.DefaultGridSize())
	// One landmark per edge: regenerating finds no other one.
	finder := &fakeLandmarks{found: edgeSights(p, p.SouthEdgeCells()[1], p.NorthEdgeCells()[4])}
	s := newService(newMemoryRepository(), time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC))
	s.landmarks = finder
	created, err := s.Create(ctx, center, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil)
	if err != nil {
		t.Fatal(err)
	}
	ada, _ := game.ParsePlayerName("Ada")
	bob, _ := game.ParsePlayerName("Bob")
	_, runnerToken, _ := s.Join(ctx, created.JoinCodes.Runner, ada)
	_, hunterToken, _ := s.Join(ctx, created.JoinCodes.Hunter, bob)
	lookups := len(finder.rows)
	var rule *game.RuleError
	if _, err := s.RegenerateRoute(ctx, hunterToken); !errors.As(err, &rule) || len(finder.rows) != lookups {
		t.Fatalf("rules must be checked before the lookup: %v", err)
	}
	if _, err := s.RegenerateRoute(ctx, runnerToken); !errors.As(err, &rule) {
		t.Fatalf("the same route must be refused: %v", err)
	}
	view, err := s.State(ctx, runnerToken)
	if err != nil || view.Lobby.Game.Route != created.Route || view.Lobby.Game.RouteRegenerations != 0 {
		t.Fatalf("a refused regeneration must not count: %+v %v", view.Lobby.Game, err)
	}
	finder.err = errors.New("down")
	if _, err := s.RegenerateRoute(ctx, runnerToken); !errors.Is(err, game.ErrLandmarksUnavailable) {
		t.Fatalf("lookup failure: %v", err)
	}
}

// hangingLandmarks never answers until the lookup is cancelled.
type hangingLandmarks struct{}

func (hangingLandmarks) Landmarks(ctx context.Context, _ game.Bounds) ([]game.Landmark, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestHangingLookupStopsWithinTheBudget(t *testing.T) {
	for _, tc := range []struct {
		name               string
		edge, wholeRequest time.Duration
	}{{"edge timeout", 20 * time.Millisecond, time.Minute}, {"request timeout", time.Minute, 20 * time.Millisecond}} {
		t.Run(tc.name, func(t *testing.T) {
			r := newMemoryRepository()
			s := NewGameService(r, hangingLandmarks{}, false)
			s.edgeTimeout, s.routeTimeout = tc.edge, tc.wholeRequest
			began := time.Now()
			_, err := s.Create(context.Background(), game.Coordinates{}, game.DefaultCellSize(), game.DefaultGridSize(), game.Rotation{}, game.DefaultRevealInterval(), game.DefaultJokerSettings(), nil)
			if !errors.Is(err, game.ErrLandmarksUnavailable) || time.Since(began) > 5*time.Second || r.calls != 0 {
				t.Fatalf("hanging lookup not stopped in time: %v after %v, %d saves", err, time.Since(began), r.calls)
			}
		})
	}
}

// boundsCenter is the middle of the bounds.
func boundsCenter(b game.Bounds) game.Coordinates {
	c, _ := game.ParseCoordinates((b.North+b.South)/2, (b.East+b.West)/2)
	return c
}

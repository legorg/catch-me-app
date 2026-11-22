// Package replay benchmarks the API by playing a stored game again.
package replay

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"catch-me/backend/api"
	"catch-me/backend/controller"
	"catch-me/backend/db"
	"catch-me/backend/service"
)

var (
	input  = flag.String("db", "../../catch-me-prod.sqlite", "SQLite database with the game to replay; only read")
	gameID = flag.String("game", "ab2355ea1ed9e3f42f545ce357e9354a", "session ID of the game to replay")
	output = flag.String("out", "", "new SQLite database the replay writes to (default: a temporary file)")
)

// pollInterval is how often the app polls the game state while it is open.
const pollInterval = 2 * time.Second

// reportPath reports a position and answers with the game state.
const reportPath = "/api/me/position"

// BenchmarkReplay plays a stored game again against a new database, as fast as
// the server answers. Everyone joins and gets ready first; once the game runs,
// every stored position report, joker and sketch is sent in the order it
// originally arrived, with at most one request per participant in flight. A
// position report answers with the game state, as the app's poll does. It
// fails as soon as a report takes longer than the app's 2 s poll interval, and
// otherwise reports how much faster than real time the game went and the load
// it produced.
//
//	go test ./replay -run '^$' -bench Replay -benchtime 1x -timeout 0 -args -db ../../catch-me-prod.sqlite -game <id>
func BenchmarkReplay(b *testing.B) {
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	b.Cleanup(func() { slog.SetDefault(previous) })

	recorded := load(b)
	path := *output
	if path == "" {
		path = filepath.Join(b.TempDir(), "replay.sqlite")
	}
	if _, err := os.Stat(path); err == nil {
		b.Fatalf("%s exists; the replay needs a new database", path)
	}
	store, err := db.Open(context.Background(), path)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = store.Close() })
	handler := api.NewHandler(controller.NewGameController(service.NewGameService(store, nil, true)))

	s := &stats{requests: map[string]*endpoint{}}
	tokens := start(b, handler, recorded, s)
	b.ResetTimer()
	slow := play(handler, recorded.events, tokens, s)
	b.StopTimer()
	s.report(b)
	if slow != nil {
		b.Fatalf("a position report took %s, longer than the %s poll interval, %s into the game (replayed %.1f× faster than real time until then)",
			slow.took.Round(time.Millisecond), pollInterval, slow.at.Round(time.Second), slow.at.Seconds()/s.wall.Seconds())
	}
}

// slowReport is the first position report that took longer than pollInterval, and how far into the game it was.
type slowReport struct{ took, at time.Duration }

// recording is the stored game: its settings, participants, and their requests in order.
type recording struct {
	create  string // POST /api/games body
	players []player
	events  []event
}

type player struct{ id, name, team, hunterTeam string }

// event is a request at the time it originally arrived, sent with the player's token.
type event struct {
	at                 time.Time
	player             string
	method, path, body string
}

func load(b *testing.B) recording {
	source, err := sql.Open("sqlite", "file:"+*input+"?mode=ro&_time_format=sqlite")
	if err != nil {
		b.Fatal(err)
	}
	defer source.Close()
	var r recording

	var latitude, longitude, startLat, startLon, stopLat, stopLon float64
	var cellSize, gridSize, rotation, interval int
	var zoneAlert bool
	allowances := make([]int, 7)
	err = source.QueryRow(`SELECT latitude, longitude, cell_size_meters, grid_size, rotation_degrees, reveal_interval_minutes,
		start_lat, start_lon, stop_lat, stop_lon, joker_zone_alert_enabled,
		joker_show_runners_allowance, joker_query_zone_allowance, joker_compass_allowance, joker_trap_allowance,
		joker_zone_only_allowance, joker_fake_position_allowance, joker_show_hunters_allowance
		FROM games WHERE id = ?`, *gameID).Scan(&latitude, &longitude, &cellSize, &gridSize, &rotation, &interval,
		&startLat, &startLon, &stopLat, &stopLon, &zoneAlert,
		&allowances[0], &allowances[1], &allowances[2], &allowances[3], &allowances[4], &allowances[5], &allowances[6])
	if err != nil {
		b.Fatalf("game %s: %v", *gameID, err)
	}
	jokers := map[string]any{"zoneAlert": map[string]bool{"enabled": zoneAlert}}
	for i, kind := range []string{"showRunners", "queryZone", "compass", "trap", "zoneOnly", "fakePosition", "showHunters"} {
		jokers[kind] = map[string]int{"allowance": allowances[i]}
	}
	// The stored route as a chosen one, so creating the game looks up no landmarks.
	r.create = encode(map[string]any{
		"latitude": latitude, "longitude": longitude, "cellSizeMeters": cellSize, "gridSize": gridSize,
		"rotationDegrees": rotation, "revealIntervalMinutes": interval, "jokers": jokers,
		"route": map[string]any{"start": coordinates(startLat, startLon), "stop": coordinates(stopLat, stopLon)},
	})

	rows := query(b, source, `SELECT id, name, team, hunter_team FROM players WHERE game_id = ? ORDER BY joined_at, id`)
	for rows.Next() {
		var p player
		scan(b, rows, &p.id, &p.name, &p.team, &p.hunterTeam)
		r.players = append(r.players, p)
	}
	// Hunter jokers and sketches are sent by any hunter of the team.
	hunterOf := map[string]string{}
	for _, p := range r.players {
		if p.hunterTeam != "" && hunterOf[strings.ToLower(p.hunterTeam)] == "" {
			hunterOf[strings.ToLower(p.hunterTeam)] = p.id
		}
	}

	rows = query(b, source, `SELECT x.player_id, x.latitude, x.longitude, x.accuracy_meters, x.recorded_at
		FROM positions x JOIN players p ON p.id = x.player_id WHERE p.game_id = ?`)
	for rows.Next() {
		var lat, lon, accuracy float64
		e := event{method: "PUT", path: reportPath}
		scan(b, rows, &e.player, &lat, &lon, &accuracy, &e.at)
		e.body = encode(map[string]float64{"latitude": lat, "longitude": lon, "accuracyMeters": accuracy})
		r.events = append(r.events, e)
	}

	rows = query(b, source, `SELECT kind, team, player_id, cell, latitude, longitude, used_at FROM jokers WHERE game_id = ? ORDER BY id`)
	for rows.Next() {
		var kind, team, cell string
		var lat, lon float64
		e := event{method: "POST", path: "/api/me/jokers"}
		scan(b, rows, &kind, &team, &e.player, &cell, &lat, &lon, &e.at)
		if e.player == "" {
			e.player = hunterOf[team]
		}
		body := map[string]any{"kind": kind}
		switch kind {
		case "queryZone":
			body["cell"] = cell
		case "trap", "fakePosition":
			body["latitude"], body["longitude"] = lat, lon
		}
		e.body = encode(body)
		r.events = append(r.events, e)
	}

	rows = query(b, source, `SELECT team, points, created_at FROM sketches WHERE game_id = ?`)
	for rows.Next() {
		var team, points string
		var line [][2]float64
		e := event{method: "POST", path: "/api/me/sketches"}
		scan(b, rows, &team, &points, &e.at)
		if err := json.Unmarshal([]byte(points), &line); err != nil {
			b.Fatal(err)
		}
		request := make([]map[string]float64, len(line))
		for i, p := range line {
			request[i] = coordinates(p[0], p[1])
		}
		e.player, e.body = hunterOf[strings.ToLower(team)], encode(map[string]any{"points": request})
		r.events = append(r.events, e)
	}
	if len(r.events) == 0 {
		b.Fatalf("game %s has nothing to replay", *gameID)
	}
	slices.SortStableFunc(r.events, func(a, b event) int { return a.at.Compare(b.at) })
	return r
}

// start creates the game, joins everyone, puts hunters into their teams, gets
// everyone ready and waits out the countdown. It returns the tokens by stored player ID.
func start(b *testing.B, handler http.Handler, r recording, s *stats) map[string]string {
	response, _ := s.send(handler, "POST", "/api/games", "", r.create)
	created := decodeAs[controller.CreateGameResponse](b, response, http.StatusCreated)
	codes := map[string]string{"runner": created.JoinCodes.Runner, "hunter": created.JoinCodes.Hunter, "spectator": created.JoinCodes.Spectator}
	tokens := map[string]string{}
	for _, p := range r.players {
		response, _ := s.send(handler, "POST", "/api/players", "", encode(map[string]string{"code": codes[p.team], "name": p.name}))
		joined := decodeAs[controller.JoinGameResponse](b, response, http.StatusCreated)
		tokens[p.id] = joined.PlayerToken
		if p.hunterTeam != "" {
			s.send(handler, "PUT", "/api/me/hunter-team", joined.PlayerToken, encode(map[string]string{"name": p.hunterTeam}))
		}
	}
	for _, p := range r.players {
		if p.team != "spectator" {
			s.send(handler, "PUT", "/api/me/ready", tokens[p.id], `{"ready":true}`)
		}
	}
	time.Sleep(11 * time.Second) // The countdown.
	return tokens
}

// play sends the events in order as fast as the server answers, with at most
// one request per participant in flight. It stops at the first position report
// slower than pollInterval and returns it; nil if every report was in time.
func play(handler http.Handler, events []event, tokens map[string]string, s *stats) *slowReport {
	var cpuBefore syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &cpuBefore)
	began, first := time.Now(), events[0].at
	var mu sync.Mutex
	var slow *slowReport
	var done int
	var played, slowest time.Duration // slowest: the slowest report since the last progress line
	inflight := make(chan struct{}, len(tokens))
	var requests sync.WaitGroup

	// Progress on stderr: b.Logf only shows once the benchmark has finished.
	progress := time.NewTicker(5 * time.Second)
	defer progress.Stop()
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		for {
			select {
			case <-finished:
				return
			case <-progress.C:
				mu.Lock()
				elapsed := time.Since(began)
				fmt.Fprintf(os.Stderr, "%s: %s of %s into the game, %d of %d requests, %.1f× real time, slowest report %s\n",
					elapsed.Round(time.Second), played.Round(time.Second), events[len(events)-1].at.Sub(first).Round(time.Second),
					done, len(events), played.Seconds()/elapsed.Seconds(), slowest.Round(time.Millisecond))
				slowest = 0
				mu.Unlock()
			}
		}
	}()

	for _, e := range events {
		mu.Lock()
		stop := slow != nil
		if !stop {
			played = e.at.Sub(first)
		}
		mu.Unlock()
		if stop {
			break
		}
		inflight <- struct{}{}
		requests.Go(func() {
			defer func() { <-inflight }()
			_, took := s.send(handler, e.method, e.path, tokens[e.player], e.body)
			mu.Lock()
			defer mu.Unlock()
			done++
			if e.path != reportPath {
				return
			}
			slowest = max(slowest, took)
			if took > pollInterval && slow == nil {
				slow = &slowReport{took: took, at: e.at.Sub(first)}
			}
		})
	}
	requests.Wait()
	s.played = played
	var cpuAfter syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &cpuAfter)
	s.wall = time.Since(began)
	s.cpu = cpuTime(cpuAfter) - cpuTime(cpuBefore)
	return slow
}

// stats is the load the replay produced: requests by endpoint, and CPU time.
type stats struct {
	mu       sync.Mutex
	requests map[string]*endpoint
	wall     time.Duration
	cpu      time.Duration
	// How far into the game the replay got.
	played time.Duration
}

type endpoint struct {
	durations []time.Duration
	statuses  map[int]int
}

// send sends a request, records its duration and status, and returns the response and the duration.
func (s *stats) send(handler http.Handler, method, path, token, body string) (*httptest.ResponseRecorder, time.Duration) {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	began := time.Now()
	handler.ServeHTTP(recorder, request)
	took := time.Since(began)
	key := method + " " + path
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.requests[key]
	if e == nil {
		e = &endpoint{statuses: map[int]int{}}
		s.requests[key] = e
	}
	e.durations = append(e.durations, took)
	e.statuses[recorder.Code]++
	return recorder, took
}

func (s *stats) report(b *testing.B) {
	keys := make([]string, 0, len(s.requests))
	total := 0
	for key, e := range s.requests {
		keys = append(keys, key)
		total += len(e.durations)
	}
	slices.Sort(keys)
	b.Logf("replayed %s of the game (%d requests) in %s, %.1f× faster than real time; server CPU %s (%.2f cores on average)",
		s.played.Round(time.Second), total, s.wall.Round(time.Second), s.played.Seconds()/s.wall.Seconds(), s.cpu.Round(time.Millisecond), s.cpu.Seconds()/s.wall.Seconds())
	b.Logf("%-24s %8s %9s %9s %9s %9s  %s", "request", "count", "p50", "p95", "p99", "max", "statuses")
	for _, key := range keys {
		e := s.requests[key]
		slices.Sort(e.durations)
		b.Logf("%-24s %8d %9s %9s %9s %9s  %v", key, len(e.durations),
			percentile(e.durations, 0.50), percentile(e.durations, 0.95), percentile(e.durations, 0.99), e.durations[len(e.durations)-1].Round(time.Microsecond), e.statuses)
	}
	b.ReportMetric(float64(total), "requests")
	b.ReportMetric(s.played.Seconds()/s.wall.Seconds(), "x-realtime")
	b.ReportMetric(s.cpu.Seconds(), "cpu-s")
	b.ReportMetric(s.cpu.Seconds()/s.wall.Seconds(), "cores")
	for key, unit := range map[string]string{"PUT " + reportPath: "report-p95-ms"} {
		if e := s.requests[key]; e != nil {
			b.ReportMetric(float64(percentile(e.durations, 0.95))/float64(time.Millisecond), unit)
		}
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	return sorted[int(p*float64(len(sorted)-1))].Round(time.Microsecond)
}

func cpuTime(u syscall.Rusage) time.Duration {
	return time.Duration(u.Utime.Nano() + u.Stime.Nano())
}

func query(b *testing.B, source *sql.DB, q string) *sql.Rows {
	rows, err := source.Query(q, *gameID)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = rows.Close() })
	return rows
}

func scan(b *testing.B, rows *sql.Rows, into ...any) {
	if err := rows.Scan(into...); err != nil {
		b.Fatal(err)
	}
}

func decodeAs[T any](b *testing.B, recorder *httptest.ResponseRecorder, status int) T {
	if recorder.Code != status {
		b.Fatalf("status = %d, want %d: %s", recorder.Code, status, recorder.Body)
	}
	var value T
	if err := json.Unmarshal(recorder.Body.Bytes(), &value); err != nil {
		b.Fatal(err)
	}
	return value
}

func encode(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func coordinates(latitude, longitude float64) map[string]float64 {
	return map[string]float64{"latitude": latitude, "longitude": longitude}
}

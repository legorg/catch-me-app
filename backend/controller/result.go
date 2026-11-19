package controller

import (
	"net/http"
	"time"

	"catch-me/backend/game"
)

type GameResultRequest struct {
	SessionID SessionID `path:"sessionId" minLength:"32" maxLength:"32" pattern:"^[a-f0-9]{32}$" example:"00112233445566778899aabbccddeeff"`
}

type GameResultResponse struct {
	SessionID SessionID `json:"sessionId" minLength:"32" maxLength:"32" example:"00112233445566778899aabbccddeeff"`
	StartsAt  time.Time `json:"startsAt" example:"2026-11-18T12:00:00Z"`
	EndedAt   time.Time `json:"endedAt" example:"2026-11-18T13:00:00Z"`
	// Whole seconds between start and end, rounded down.
	DurationSeconds int64      `json:"durationSeconds" minimum:"0" example:"3600"`
	Playground      Playground `json:"playground"`
	// The shared runner route, public once the game has ended.
	Route   GameRoute              `json:"route"`
	Players []ResultPlayerResponse `json:"players" maxItems:"30"`
	// The whole timeline, oldest first, with no coordinates, cells or joker results.
	Events []TimelineEventResponse `json:"events" maxItems:"122"`
}

type ResultPlayerResponse struct {
	PlayerID   PlayerID              `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677"`
	Name       string                `json:"name" minLength:"1" maxLength:"20" example:"Ada"`
	Team       Team                  `json:"team"`
	HunterTeam string                `json:"hunterTeam,omitempty" maxLength:"20" example:"Pack"`
	Runner     *RunnerResultResponse `json:"runner,omitempty"`
	// Real reports from start to end, oldest first, at least 5 seconds apart
	// (the last report is always included). Present, possibly empty, for
	// runners and hunters; absent for spectators. At most 120962 points;
	// longer tracks are sampled more coarsely, preserving first and last reports.
	Track *[]PositionResponse `json:"track,omitempty" maxItems:"120962"`
}

// RunnerOutcome is the terminal subset of RunnerStatus used in public results.
type RunnerOutcome game.RunnerStatus

func (RunnerOutcome) Enum() []any { return []any{game.RunnerCaught, game.RunnerFinished} }

type RunnerResultResponse struct {
	Status  RunnerOutcome `json:"status"`
	EndedAt time.Time     `json:"endedAt" example:"2026-11-18T12:30:00Z"`
}

func (c *GameController) Result(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodGet) {
		return
	}
	id, err := game.ParseSessionID(r.PathValue("sessionId"))
	if err != nil {
		writeGameError(w, err, "invalid game ID")
		return
	}
	result, err := c.service.Result(r.Context(), id)
	if err != nil {
		writeGameError(w, err, "could not load game result")
		return
	}
	response, err := gameResultResponse(result)
	if err != nil {
		writeGameError(w, err, "could not load game result")
		return
	}
	WriteJSON(w, http.StatusOK, response)
}

func gameResultResponse(result game.Result) (GameResultResponse, error) {
	events, err := timelineEventResponses(result.Events, func(id game.PlayerID) (game.PlayerName, bool) {
		p, ok := result.Player(id)
		return p.Name, ok
	})
	if err != nil {
		return GameResultResponse{}, err
	}
	response := GameResultResponse{
		SessionID: SessionID(result.SessionID.String()), StartsAt: result.StartsAt, EndedAt: result.EndedAt,
		DurationSeconds: int64(result.EndedAt.Sub(result.StartsAt) / time.Second),
		Playground:      playgroundResponse(result.Playground),
		Route:           gameRouteResponse(result.Route),
		Players:         make([]ResultPlayerResponse, 0, len(result.Players)),
		Events:          events,
	}
	for _, p := range result.Players {
		player := ResultPlayerResponse{PlayerID: PlayerID(p.ID.String()), Name: p.Name.String(), Team: Team(p.Team), Track: trackResponse(p.Track)}
		if p.Team == game.Runners {
			player.Runner = &RunnerResultResponse{Status: RunnerOutcome(p.Runner.Status()), EndedAt: p.Runner.EndedAt()}
		} else if p.Team == game.Hunters {
			player.HunterTeam = p.HunterTeam.String()
		}
		response.Players = append(response.Players, player)
	}
	return response, nil
}

// trackResponse keeps an empty track as [] and a missing one (spectators) absent.
func trackResponse(track []game.Position) *[]PositionResponse {
	if track == nil {
		return nil
	}
	positions := make([]PositionResponse, 0, len(track))
	for _, p := range track {
		positions = append(positions, positionResponse(p))
	}
	return &positions
}

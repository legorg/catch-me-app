package controller

import (
	"net/http"
	"time"

	"catch-me/backend/game"
)

type GameResultRequest struct {
	SessionID SessionID `path:"sessionId" minLength:"32" maxLength:"32" pattern:"^[a-f0-9]{32}$" example:"00112233445566778899aabbccddeeff" validate:"required,len=32"`
}

type GameResultResponse struct {
	SessionID SessionID `json:"sessionId" minLength:"32" maxLength:"32" example:"00112233445566778899aabbccddeeff" validate:"required,len=32"`
	StartsAt  time.Time `json:"startsAt" example:"2026-11-18T12:00:00Z" validate:"required"`
	EndedAt   time.Time `json:"endedAt" example:"2026-11-18T13:00:00Z" validate:"required"`
	// Whole seconds between start and end, rounded down.
	DurationSeconds int64                  `json:"durationSeconds" minimum:"0" example:"3600" validate:"gte=0"`
	Players         []ResultPlayerResponse `json:"players" validate:"max=100,dive"`
}

type ResultPlayerResponse struct {
	PlayerID   PlayerID              `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677" validate:"required,len=32"`
	Name       string                `json:"name" minLength:"1" maxLength:"20" example:"Ada" validate:"required,min=1,max=20"`
	Team       Team                  `json:"team" validate:"required"`
	HunterTeam string                `json:"hunterTeam,omitempty" maxLength:"20" example:"Pack" validate:"omitempty,max=20"`
	Runner     *RunnerResultResponse `json:"runner,omitempty" validate:"omitempty"`
}

// RunnerOutcome is the terminal subset of RunnerStatus used in public results.
type RunnerOutcome game.RunnerStatus

func (RunnerOutcome) Enum() []any { return []any{game.RunnerCaught, game.RunnerFinished} }

type RunnerResultResponse struct {
	Status  RunnerOutcome `json:"status" validate:"required"`
	EndedAt time.Time     `json:"endedAt" example:"2026-11-18T12:30:00Z" validate:"required"`
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
	WriteJSON(w, http.StatusOK, gameResultResponse(result))
}

func gameResultResponse(result game.Result) GameResultResponse {
	response := GameResultResponse{
		SessionID: SessionID(result.SessionID.String()), StartsAt: result.StartsAt, EndedAt: result.EndedAt,
		DurationSeconds: int64(result.EndedAt.Sub(result.StartsAt) / time.Second),
		Players:         make([]ResultPlayerResponse, 0, len(result.Players)),
	}
	for _, p := range result.Players {
		player := ResultPlayerResponse{PlayerID: PlayerID(p.ID.String()), Name: p.Name.String(), Team: Team(p.Team)}
		if p.Team == game.Runners {
			player.Runner = &RunnerResultResponse{Status: RunnerOutcome(p.Runner.Status()), EndedAt: p.Runner.EndedAt()}
		} else if p.Team == game.Hunters {
			player.HunterTeam = p.HunterTeam.String()
		}
		response.Players = append(response.Players, player)
	}
	return response
}

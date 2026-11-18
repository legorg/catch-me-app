package controller

import (
	"fmt"
	"net/http"
	"time"

	"catch-me/backend/game"
	"catch-me/backend/service"
)

// TimelineEventKind tells the variants of TimelineEventResponse apart. Each
// value must match the enum tag of its variant's kind field, which the OpenAPI
// discriminator is built from.
type TimelineEventKind string

const (
	TimelineGameStarted     TimelineEventKind = "gameStarted"
	TimelineHunterJokerUsed TimelineEventKind = "hunterJokerUsed"
	TimelineRunnerJokerUsed TimelineEventKind = "runnerJokerUsed"
	TimelineRunnerCaught    TimelineEventKind = "runnerCaught"
	TimelineRunnerFinished  TimelineEventKind = "runnerFinished"
	TimelineGameEnded       TimelineEventKind = "gameEnded"
)

type TimelineResponse struct {
	ServerTime time.Time `json:"serverTime" example:"2026-11-08T12:30:00Z"`
	// Oldest first, up to serverTime.
	Events []TimelineEventResponse `json:"events"`
}

// TimelinePlayer is the player an event is about.
type TimelinePlayer struct {
	PlayerID PlayerID `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677"`
	Name     string   `json:"name" minLength:"1" maxLength:"20" example:"Ada"`
}

// TimelineEventResponse is one timeline entry: exactly one of
// GameStartedEventResponse, HunterJokerUsedEventResponse,
// RunnerJokerUsedEventResponse, RunnerCaughtEventResponse,
// RunnerFinishedEventResponse, and GameEndedEventResponse, told apart by kind.
// No variant has coordinates, cells, or joker results.
type TimelineEventResponse struct{ variant any }

// JSONSchemaOneOf documents the union in the OpenAPI spec.
func (TimelineEventResponse) JSONSchemaOneOf() []any {
	return []any{GameStartedEventResponse{}, HunterJokerUsedEventResponse{}, RunnerJokerUsedEventResponse{},
		RunnerCaughtEventResponse{}, RunnerFinishedEventResponse{}, GameEndedEventResponse{}}
}

func (e TimelineEventResponse) MarshalJSON() ([]byte, error) { return marshalVariant(e.variant) }

type GameStartedEventResponse struct {
	Kind TimelineEventKind `json:"kind" enum:"gameStarted"`
	At   time.Time         `json:"at" example:"2026-11-08T12:00:10Z"`
}

type HunterJokerUsedEventResponse struct {
	Kind       TimelineEventKind `json:"kind" enum:"hunterJokerUsed"`
	At         time.Time         `json:"at" example:"2026-11-08T12:07:00Z"`
	HunterTeam string            `json:"hunterTeam" minLength:"1" maxLength:"20" example:"Wolves"`
	Joker      JokerKind         `json:"joker"`
}

type RunnerJokerUsedEventResponse struct {
	Kind   TimelineEventKind `json:"kind" enum:"runnerJokerUsed"`
	At     time.Time         `json:"at" example:"2026-11-08T12:07:00Z"`
	Runner TimelinePlayer    `json:"runner"`
	Joker  JokerKind         `json:"joker"`
}

type RunnerCaughtEventResponse struct {
	Kind   TimelineEventKind `json:"kind" enum:"runnerCaught"`
	At     time.Time         `json:"at" example:"2026-11-08T12:30:00Z"`
	Runner TimelinePlayer    `json:"runner"`
}

type RunnerFinishedEventResponse struct {
	Kind   TimelineEventKind `json:"kind" enum:"runnerFinished"`
	At     time.Time         `json:"at" example:"2026-11-08T12:45:00Z"`
	Runner TimelinePlayer    `json:"runner"`
}

type GameEndedEventResponse struct {
	Kind TimelineEventKind `json:"kind" enum:"gameEnded"`
	At   time.Time         `json:"at" example:"2026-11-08T13:00:10Z"`
}

// Timeline returns the game's events to an authenticated spectator.
func (c *GameController) Timeline(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodGet) {
		return
	}
	token, ok := requirePlayerToken(w, r)
	if !ok {
		return
	}
	view, err := c.service.Timeline(r.Context(), token)
	if err != nil {
		writeGameError(w, err, "could not load timeline")
		return
	}
	response, err := timelineResponse(view)
	if err != nil {
		writeGameError(w, err, "could not load timeline")
		return
	}
	WriteJSON(w, http.StatusOK, response)
}

func timelineResponse(view service.TimelineView) (TimelineResponse, error) {
	response := TimelineResponse{ServerTime: view.Now, Events: make([]TimelineEventResponse, 0, len(view.Events))}
	for _, e := range view.Events {
		event, err := timelineEventResponse(view.Lobby, e)
		if err != nil {
			return TimelineResponse{}, err
		}
		response.Events = append(response.Events, TimelineEventResponse{event})
	}
	return response, nil
}

// timelineEventResponse maps one event; an unknown variant is a server error
// rather than an event without its fields.
func timelineEventResponse(lobby game.Lobby, event game.Event) (any, error) {
	switch e := event.(type) {
	case game.StartEvent:
		return GameStartedEventResponse{Kind: TimelineGameStarted, At: e.At}, nil
	case game.HunterJokerEvent:
		return HunterJokerUsedEventResponse{Kind: TimelineHunterJokerUsed, At: e.At, HunterTeam: e.Team.String(), Joker: JokerKind(e.Joker)}, nil
	case game.RunnerJokerEvent:
		runner, err := timelinePlayer(lobby, e.Runner)
		return RunnerJokerUsedEventResponse{Kind: TimelineRunnerJokerUsed, At: e.At, Runner: runner, Joker: JokerKind(e.Joker)}, err
	case game.CatchEvent:
		runner, err := timelinePlayer(lobby, e.Runner)
		return RunnerCaughtEventResponse{Kind: TimelineRunnerCaught, At: e.At, Runner: runner}, err
	case game.FinishEvent:
		runner, err := timelinePlayer(lobby, e.Runner)
		return RunnerFinishedEventResponse{Kind: TimelineRunnerFinished, At: e.At, Runner: runner}, err
	case game.EndEvent:
		return GameEndedEventResponse{Kind: TimelineGameEnded, At: e.At}, nil
	}
	return nil, fmt.Errorf("unknown timeline event %T", event)
}

func timelinePlayer(lobby game.Lobby, id game.PlayerID) (TimelinePlayer, error) {
	player, ok := lobby.Player(id)
	if !ok {
		return TimelinePlayer{}, fmt.Errorf("timeline event of unknown player %s", id)
	}
	return TimelinePlayer{PlayerID: PlayerID(id.String()), Name: player.Name.String()}, nil
}

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
	TimelineTrapTriggered   TimelineEventKind = "trapTriggered"
	TimelineRunnerCaught    TimelineEventKind = "runnerCaught"
	TimelineRunnerFinished  TimelineEventKind = "runnerFinished"
	TimelineGameEnded       TimelineEventKind = "gameEnded"
)

type TimelineResponse struct {
	ServerTime time.Time `json:"serverTime" example:"2026-11-08T12:30:00Z"`
	// Oldest first, up to serverTime. At most start + 5 joker events per
	// player (a hunter team's 4 jokers and its trap trigger; 30 players) +
	// one outcome per player + end: 1 + 150 + 30 + 1.
	Events []TimelineEventResponse `json:"events" maxItems:"182"`
}

// TimelinePlayer is the player an event is about.
type TimelinePlayer struct {
	PlayerID PlayerID `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677"`
	Name     string   `json:"name" minLength:"1" maxLength:"20" example:"Ada"`
}

// TimelineEventResponse is one timeline entry: exactly one of
// GameStartedEventResponse, HunterJokerUsedEventResponse,
// RunnerJokerUsedEventResponse, TrapTriggeredEventResponse,
// RunnerCaughtEventResponse, RunnerFinishedEventResponse, and
// GameEndedEventResponse, told apart by kind.
// No variant has coordinates, cells, or joker results.
type TimelineEventResponse struct{ variant any }

// JSONSchemaOneOf documents the union in the OpenAPI spec.
func (TimelineEventResponse) JSONSchemaOneOf() []any {
	return []any{GameStartedEventResponse{}, HunterJokerUsedEventResponse{}, RunnerJokerUsedEventResponse{}, TrapTriggeredEventResponse{},
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

// TrapTriggeredEventResponse is a hunter team's trap triggered by some runner;
// it does not say which runner or where.
type TrapTriggeredEventResponse struct {
	Kind       TimelineEventKind `json:"kind" enum:"trapTriggered"`
	At         time.Time         `json:"at" example:"2026-11-08T12:41:30Z"`
	HunterTeam string            `json:"hunterTeam" minLength:"1" maxLength:"20" example:"Wolves"`
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
	events, err := timelineEventResponses(view.Events, func(id game.PlayerID) (game.PlayerName, bool) {
		p, ok := view.Lobby.Player(id)
		return p.Name, ok
	})
	if err != nil {
		return TimelineResponse{}, err
	}
	return TimelineResponse{ServerTime: view.Now, Events: events}, nil
}

// playerName finds the name of the player an event is about.
type playerName func(game.PlayerID) (game.PlayerName, bool)

func timelineEventResponses(events []game.Event, name playerName) ([]TimelineEventResponse, error) {
	responses := make([]TimelineEventResponse, 0, len(events))
	for _, e := range events {
		event, err := timelineEventResponse(e, name)
		if err != nil {
			return nil, err
		}
		responses = append(responses, TimelineEventResponse{event})
	}
	return responses, nil
}

// timelineEventResponse maps one event; an unknown variant is a server error
// rather than an event without its fields.
func timelineEventResponse(event game.Event, name playerName) (any, error) {
	switch e := event.(type) {
	case game.StartEvent:
		return GameStartedEventResponse{Kind: TimelineGameStarted, At: e.At}, nil
	case game.HunterJokerEvent:
		return HunterJokerUsedEventResponse{Kind: TimelineHunterJokerUsed, At: e.At, HunterTeam: e.Team.String(), Joker: JokerKind(e.Joker)}, nil
	case game.RunnerJokerEvent:
		runner, err := timelinePlayer(e.Runner, name)
		return RunnerJokerUsedEventResponse{Kind: TimelineRunnerJokerUsed, At: e.At, Runner: runner, Joker: JokerKind(e.Joker)}, err
	case game.TrapTriggeredEvent:
		return TrapTriggeredEventResponse{Kind: TimelineTrapTriggered, At: e.At, HunterTeam: e.Team.String()}, nil
	case game.CatchEvent:
		runner, err := timelinePlayer(e.Runner, name)
		return RunnerCaughtEventResponse{Kind: TimelineRunnerCaught, At: e.At, Runner: runner}, err
	case game.FinishEvent:
		runner, err := timelinePlayer(e.Runner, name)
		return RunnerFinishedEventResponse{Kind: TimelineRunnerFinished, At: e.At, Runner: runner}, err
	case game.EndEvent:
		return GameEndedEventResponse{Kind: TimelineGameEnded, At: e.At}, nil
	}
	return nil, fmt.Errorf("unknown timeline event %T", event)
}

func timelinePlayer(id game.PlayerID, name playerName) (TimelinePlayer, error) {
	found, ok := name(id)
	if !ok {
		return TimelinePlayer{}, fmt.Errorf("timeline event of unknown player %s", id)
	}
	return TimelinePlayer{PlayerID: PlayerID(id.String()), Name: found.String()}, nil
}

package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Wire enums and identifiers. The consts are the single source of the allowed
// values: swag derives the OpenAPI enums from them.

// SessionID is the 32-character hex ID of a game.
type SessionID string

// PlayerID is the 32-character hex ID of a player.
type PlayerID string

// SketchID is the 32-character hex ID of a sketch.
type SketchID string

// Cell is a playground cell: column A-F, then row 1-6.
type Cell string

type Team string

func (Team) Enum() []any { return []any{TeamRunner, TeamHunter} }

const (
	TeamRunner Team = "runner"
	TeamHunter Team = "hunter"
)

type GameStatus string

func (GameStatus) Enum() []any { return []any{GameStatusLobby, GameStatusCountdown, GameStatusRunning} }

const (
	GameStatusLobby     GameStatus = "lobby"
	GameStatusCountdown GameStatus = "countdown"
	GameStatusRunning   GameStatus = "running"
)

type JokerKind string

func (JokerKind) Enum() []any {
	return []any{JokerKindShowRunners, JokerKindQueryZone, JokerKindZoneOnly}
}

const (
	JokerKindShowRunners JokerKind = "showRunners"
	JokerKindQueryZone   JokerKind = "queryZone"
	JokerKindZoneOnly    JokerKind = "zoneOnly"
)

type HealthResponse struct {
	Status  string `json:"status" enums:"ok" example:"ok" validate:"required,oneof=ok"`
	Service string `json:"service" enums:"catch-me-backend" example:"catch-me-backend" validate:"required,oneof=catch-me-backend"`
}

type Coordinates struct {
	Latitude  float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52" validate:"gte=-80,lte=80"`
	Longitude float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405" validate:"gte=-180,lte=180"`
}

type Bounds struct {
	North float64 `json:"north" example:"52.57395922182347" validate:"gte=-90,lte=90"`
	South float64 `json:"south" example:"52.466040778176534" validate:"gte=-90,lte=90"`
	East  float64 `json:"east" example:"13.493678060701022" validate:"gte=-180,lte=180"`
	West  float64 `json:"west" example:"13.316321939298977" validate:"gte=-180,lte=180"`
}

// Pointer fields distinguish missing/null coordinates from a valid zero.
type CreateGameRequest struct {
	Latitude       *float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52" validate:"required,gte=-80,lte=80"`
	Longitude      *float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405" validate:"required,gte=-180,lte=180"`
	CellSizeMeters *int     `json:"cellSizeMeters,omitempty" minimum:"100" maximum:"5000" default:"2000" example:"2000" validate:"omitempty,gte=100,lte=5000"`
}

type Playground struct {
	Center         Coordinates `json:"center"`
	Bounds         Bounds      `json:"bounds"`
	Rows           int         `json:"rows" enums:"6" example:"6" validate:"eq=6"`
	Columns        int         `json:"columns" enums:"6" example:"6" validate:"eq=6"`
	CellSizeMeters int         `json:"cellSizeMeters" minimum:"100" maximum:"5000" example:"2000" validate:"gte=100,lte=5000"`
}

type Landmark struct {
	// Name of the place, e.g. a landmark.
	Title       string      `json:"title" example:"Brandenburger Tor" validate:"max=255"`
	Coordinates Coordinates `json:"coordinates"`
}

// GameRoute leads from a landmark in the bottom row of the grid to one in the top row.
type GameRoute struct {
	Start Landmark `json:"start"`
	Stop  Landmark `json:"stop"`
	// Straight-line (great-circle) distance between start and stop.
	DistanceMeters float64 `json:"distanceMeters" example:"10432.7" validate:"gte=0"`
}

type GameResponse struct {
	SessionID  SessionID  `json:"sessionId" minLength:"32" maxLength:"32" example:"00112233445566778899aabbccddeeff" validate:"required,len=32"`
	CreatedAt  time.Time  `json:"createdAt" example:"2026-11-08T12:00:00Z" validate:"required"`
	Playground Playground `json:"playground"`
	Route      GameRoute  `json:"route"`
}

type JoinCodes struct {
	Runner string `json:"runner" minLength:"6" maxLength:"6" example:"482913" validate:"required,len=6,numeric"`
	Hunter string `json:"hunter" minLength:"6" maxLength:"6" example:"071564" validate:"required,len=6,numeric"`
}

type CreateGameResponse struct {
	Game      GameResponse `json:"game"`
	JoinCodes JoinCodes    `json:"joinCodes"`
}

type JoinGameRequest struct {
	Code string `json:"code" minLength:"6" maxLength:"6" example:"482913" validate:"required,len=6,numeric"`
	Name string `json:"name" minLength:"1" maxLength:"20" example:"Ada" validate:"required,min=1,max=20"`
}

// Pointer field distinguishes a missing value from false.
type SetReadyRequest struct {
	Ready *bool `json:"ready" example:"true" validate:"required"`
}

type SetHunterTeamRequest struct {
	Name string `json:"name" minLength:"1" maxLength:"20" example:"Wolves" validate:"required,min=1,max=20"`
}

type PlayerResponse struct {
	PlayerID PlayerID `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677" validate:"required,len=32"`
	Name     string   `json:"name" minLength:"1" maxLength:"20" example:"Ada" validate:"required,min=1,max=20"`
	Team     Team     `json:"team" validate:"required,oneof=runner hunter"`
	// Present for hunters who have chosen a hunter team.
	HunterTeam string `json:"hunterTeam,omitempty" maxLength:"20" example:"Wolves" validate:"omitempty,max=20"`
	Ready      bool   `json:"ready" example:"false"`
	// Live position: own always; hunter teammates' always; runner teammates'
	// while running. Hunters see runners through reveals instead.
	Position *PositionResponse `json:"position,omitempty" validate:"omitempty"`
	// Hunters only, for runners while running: every reveal so far, oldest first.
	Reveals []RevealResponse `json:"reveals,omitempty" validate:"omitempty,max=1000,dive"`
}

// RevealResponse is what hunters see of a runner at a reveal time: the
// position, or for a zone-only reveal (runner joker) only the cell.
type RevealResponse struct {
	RevealedAt time.Time `json:"revealedAt" example:"2026-11-08T12:05:00Z" validate:"required"`
	// Absent for a zone-only reveal.
	Position *PositionResponse `json:"position,omitempty" validate:"omitempty"`
	// Set when the runner used the zoneOnly joker before this reveal.
	ZoneOnly bool `json:"zoneOnly,omitempty" example:"false"`
	// Zone-only reveals: the cell the runner was in; absent when they were outside the playground.
	Cell Cell `json:"cell,omitempty" pattern:"^[A-F][1-6]$" example:"C4" validate:"omitempty,len=2"`
}

type PositionResponse struct {
	Coordinates
	AccuracyMeters float64   `json:"accuracyMeters" minimum:"0" maximum:"10000" example:"8" validate:"gte=0,lte=10000"`
	RecordedAt     time.Time `json:"recordedAt" example:"2026-11-08T12:04:58Z" validate:"required"`
}

// Pointer fields distinguish missing values from zero.
type UpdatePositionRequest struct {
	Latitude       *float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52" validate:"required,gte=-80,lte=80"`
	Longitude      *float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405" validate:"required,gte=-180,lte=180"`
	AccuracyMeters *float64 `json:"accuracyMeters" minimum:"0" maximum:"10000" example:"8" validate:"required,gte=0,lte=10000"`
}

type UseJokerRequest struct {
	Kind JokerKind `json:"kind" validate:"required,oneof=showRunners queryZone zoneOnly"`
	// The cell to query (queryZone only): column A-F, then row 1-6.
	Cell Cell `json:"cell,omitempty" pattern:"^[A-F][1-6]$" example:"C4" validate:"omitempty,len=2"`
}

type AddSketchRequest struct {
	Points []SketchPointRequest `json:"points" validate:"required,min=2,max=500,dive"`
}

type SketchPointRequest struct {
	Latitude  *float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52" validate:"required,gte=-80,lte=80"`
	Longitude *float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405" validate:"required,gte=-180,lte=180"`
}

type SketchResponse struct {
	SketchID  SketchID      `json:"sketchId" minLength:"32" maxLength:"32" example:"0123456789abcdef0123456789abcdef" validate:"required,len=32"`
	CreatedAt time.Time     `json:"createdAt" example:"2026-11-08T12:07:00Z" validate:"required"`
	Points    []Coordinates `json:"points" validate:"required,min=2,max=500,dive"`
}

// JokerResponse is one joker of the requesting player's team.
type JokerResponse struct {
	Kind JokerKind `json:"kind" validate:"required,oneof=showRunners queryZone zoneOnly"`
	// True while the game runs and the team has not used it yet.
	Available bool `json:"available" example:"true"`
	// Set once the team used it.
	UsedAt *time.Time `json:"usedAt,omitempty" example:"2026-11-08T12:07:00Z" validate:"omitempty"`
	// Runner jokers, once used: the runner who used it.
	UsedBy PlayerID `json:"usedBy,omitempty" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677" validate:"omitempty,len=32"`
	// What the joker showed (showRunners: each runner's position at usedAt), once used.
	Positions []JokerPositionResponse `json:"positions,omitempty" validate:"omitempty,max=100,dive"`
	// queryZone, once used: the queried cell and whether any runner was inside it at usedAt.
	Cell  Cell  `json:"cell,omitempty" pattern:"^[A-F][1-6]$" example:"C4" validate:"omitempty,len=2"`
	Found *bool `json:"found,omitempty" example:"true"`
}

type JokerPositionResponse struct {
	PlayerID PlayerID `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677" validate:"required,len=32"`
	PositionResponse
}

// GameStateBase is what every player sees of the game.
type GameStateBase struct {
	// Explicit server-side development override; absent during normal play.
	StartPositionCheckDisabled bool       `json:"startPositionCheckDisabled,omitempty" example:"false" validate:"omitempty"`
	SessionID                  SessionID  `json:"sessionId" minLength:"32" maxLength:"32" example:"00112233445566778899aabbccddeeff" validate:"required,len=32"`
	Status                     GameStatus `json:"status" validate:"required,oneof=lobby countdown running"`
	ServerTime                 time.Time  `json:"serverTime" example:"2026-11-08T12:00:00Z" validate:"required"`
	// Scheduled start, present during the countdown and once running.
	StartsAt *time.Time `json:"startsAt,omitempty" example:"2026-11-08T12:00:10Z" validate:"omitempty"`
	// Next time hunters receive the runners' positions, present while running.
	NextRevealAt *time.Time       `json:"nextRevealAt,omitempty" example:"2026-11-08T12:05:10Z" validate:"omitempty"`
	Playground   Playground       `json:"playground"`
	PlayerID     PlayerID         `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677" validate:"required,len=32"`
	JoinCodes    JoinCodes        `json:"joinCodes"`
	Players      []PlayerResponse `json:"players" validate:"max=100,dive"`
}

// GameStateRunnerResponse is the game as seen by a runner.
type GameStateRunnerResponse struct {
	GameStateBase
	Team Team `json:"team" enum:"runner" validate:"required,eq=runner"`
	// The game's route between two landmarks; hunters have to find them.
	Route GameRoute `json:"route"`
	// How often the runners together may still regenerate the route.
	RouteRegenerationsLeft int `json:"routeRegenerationsLeft" minimum:"0" maximum:"2" example:"2" validate:"gte=0,lte=2"`
	// The jokers all runners share.
	Jokers []JokerResponse `json:"jokers" validate:"max=2,dive"`
}

// GameStateHunterResponse is the game as seen by a hunter.
type GameStateHunterResponse struct {
	GameStateBase
	Team Team `json:"team" enum:"hunter" validate:"required,eq=hunter"`
	// The hunter team's jokers; empty for hunters without a team.
	Jokers []JokerResponse `json:"jokers" validate:"max=2,dive"`
	// The hunter team's sketches while running, oldest first.
	Sketches []SketchResponse `json:"sketches" validate:"max=1000,dive"`
}

// GameStateResponse is the game as seen by the requesting player: exactly one
// of GameStateRunnerResponse and GameStateHunterResponse, told apart by team.
type GameStateResponse struct {
	runner *GameStateRunnerResponse
	hunter *GameStateHunterResponse
}

// JSONSchemaOneOf documents the union in the OpenAPI spec.
func (GameStateResponse) JSONSchemaOneOf() []any {
	return []any{GameStateRunnerResponse{}, GameStateHunterResponse{}}
}

func NewRunnerState(state GameStateRunnerResponse) GameStateResponse {
	return GameStateResponse{runner: &state}
}

func NewHunterState(state GameStateHunterResponse) GameStateResponse {
	return GameStateResponse{hunter: &state}
}

// Base is the part both teams see.
func (s GameStateResponse) Base() GameStateBase {
	if s.runner != nil {
		return s.runner.GameStateBase
	}
	if s.hunter != nil {
		return s.hunter.GameStateBase
	}
	return GameStateBase{}
}

func (s GameStateResponse) Runner() (GameStateRunnerResponse, bool) {
	if s.runner == nil {
		return GameStateRunnerResponse{}, false
	}
	return *s.runner, true
}

func (s GameStateResponse) Hunter() (GameStateHunterResponse, bool) {
	if s.hunter == nil {
		return GameStateHunterResponse{}, false
	}
	return *s.hunter, true
}

func (s GameStateResponse) MarshalJSON() ([]byte, error) {
	if s.runner != nil {
		return json.Marshal(s.runner)
	}
	if s.hunter != nil {
		return json.Marshal(s.hunter)
	}
	return nil, errors.New("game state has no team")
}

func (s *GameStateResponse) UnmarshalJSON(data []byte) error {
	var probe struct {
		Team Team `json:"team"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}
	*s = GameStateResponse{}
	switch probe.Team {
	case TeamRunner:
		s.runner = new(GameStateRunnerResponse)
		return json.Unmarshal(data, s.runner)
	case TeamHunter:
		s.hunter = new(GameStateHunterResponse)
		return json.Unmarshal(data, s.hunter)
	}
	return fmt.Errorf("unknown team %q", probe.Team)
}

type JoinGameResponse struct {
	PlayerToken string            `json:"playerToken" minLength:"64" maxLength:"64" example:"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff" validate:"required,len=64"`
	State       GameStateResponse `json:"state"`
}

type APIError struct {
	Error string `json:"error" example:"cell size must be 100 to 5000 meters in steps of 100 meters" validate:"required"`
}

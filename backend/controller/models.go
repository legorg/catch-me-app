package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"catch-me/backend/game"
)

// Wire enums and identifiers. Enum methods expose domain constants to the
// OpenAPI generator without adding schema methods to domain types.

// SessionID is the 32-character hex ID of a game.
type SessionID string

// PlayerID is the 32-character hex ID of a player.
type PlayerID string

// SketchID is the 32-character hex ID of a sketch.
type SketchID string

// Cell is a playground cell: column A-L, then row 1-12, within the game's grid.
type Cell string

type Team game.Team

func (Team) Enum() []any { return []any{game.Runners, game.Hunters} }

type GameStatus game.Status

func (GameStatus) Enum() []any {
	return []any{game.StatusLobby, game.StatusCountdown, game.StatusRunning}
}

type JokerKind game.JokerKind

func (JokerKind) Enum() []any {
	return []any{game.JokerShowRunners, game.JokerQueryZone, game.JokerZoneOnly, game.JokerFakePosition, game.JokerShowHunters, game.JokerCompass}
}

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
	// Cells per side of the square grid.
	GridSize *int `json:"gridSize,omitempty" minimum:"3" maximum:"12" default:"6" example:"6" validate:"omitempty,gte=3,lte=12"`
	// How often hunters receive the runners' positions.
	RevealIntervalMinutes *int `json:"revealIntervalMinutes,omitempty" minimum:"1" maximum:"60" default:"30" example:"30" validate:"omitempty,gte=1,lte=60"`
}

type Playground struct {
	Center         Coordinates `json:"center"`
	Bounds         Bounds      `json:"bounds"`
	Rows           int         `json:"rows" minimum:"3" maximum:"12" example:"6" validate:"gte=3,lte=12"`
	Columns        int         `json:"columns" minimum:"3" maximum:"12" example:"6" validate:"gte=3,lte=12"`
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
	// How often hunters receive the runners' positions.
	RevealIntervalMinutes int       `json:"revealIntervalMinutes" minimum:"1" maximum:"60" example:"30" validate:"gte=1,lte=60"`
	Route                 GameRoute `json:"route"`
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
	Team     Team     `json:"team" validate:"required"`
	// Present for hunters who have chosen a hunter team.
	HunterTeam string `json:"hunterTeam,omitempty" maxLength:"20" example:"Wolves" validate:"omitempty,max=20"`
	Ready      bool   `json:"ready" example:"false"`
	// Live position: own always; hunter teammates' always; runner teammates'
	// while running. Hunters see runners through reveals instead.
	Position *PositionResponse `json:"position,omitempty" validate:"omitempty"`
	// Hunters only, for runners while running: every reveal so far, oldest first.
	Reveals []RevealResponse `json:"reveals,omitempty" validate:"omitempty,max=1000,dive"`
	// Present for every runner, including active runners; absent for hunters.
	Runner *RunnerStateResponse `json:"runner,omitempty" validate:"omitempty"`
}

// RunnerStatus adds schema metadata without putting OpenAPI methods on the domain type.
type RunnerStatus game.RunnerStatus

func (RunnerStatus) Enum() []any {
	return []any{game.RunnerActive, game.RunnerCaught, game.RunnerFinished}
}

// RunnerStateResponse exposes one outcome; there is never a finish-position field.
type RunnerStateResponse struct {
	Status RunnerStatus `json:"status" validate:"required"`
	// Absent while active; server-recorded completion time for caught or finished runners.
	EndedAt *time.Time `json:"endedAt,omitempty" validate:"omitempty"`
	// Last report before declaring the catch, frozen until position retention expires.
	// Only for caught runners with a retained report; never for active or finished runners.
	CaughtPosition *PositionResponse `json:"caughtPosition,omitempty" validate:"omitempty"`
}

// RevealKind tells the variants of RevealResponse apart.
type RevealKind string

func (RevealKind) Enum() []any { return []any{RevealKindPosition, RevealKindZone, RevealKindOutside} }

const (
	RevealKindPosition RevealKind = "position"
	RevealKindZone     RevealKind = "zone"
	RevealKindOutside  RevealKind = "outside"
)

// RevealResponse is what hunters see of a runner at a reveal time: exactly one
// of PositionRevealResponse, ZoneRevealResponse, and OutsideRevealResponse,
// told apart by kind.
type RevealResponse struct{ variant any }

// JSONSchemaOneOf documents the union in the OpenAPI spec.
func (RevealResponse) JSONSchemaOneOf() []any {
	return []any{PositionRevealResponse{}, ZoneRevealResponse{}, OutsideRevealResponse{}}
}

func (r RevealResponse) MarshalJSON() ([]byte, error) { return marshalVariant(r.variant) }

// PositionRevealResponse shows where the runner was at the reveal time.
type PositionRevealResponse struct {
	Kind       RevealKind       `json:"kind" enum:"position" validate:"required,eq=position"`
	RevealedAt time.Time        `json:"revealedAt" example:"2026-11-08T12:05:00Z" validate:"required"`
	Position   PositionResponse `json:"position"`
}

// ZoneRevealResponse shows only the cell the runner was in (zoneOnly joker).
type ZoneRevealResponse struct {
	Kind       RevealKind `json:"kind" enum:"zone" validate:"required,eq=zone"`
	RevealedAt time.Time  `json:"revealedAt" example:"2026-11-08T12:05:00Z" validate:"required"`
	Cell       Cell       `json:"cell" pattern:"^[A-L]([1-9]|1[0-2])$" example:"C4" validate:"required,min=2,max=3"`
}

// OutsideRevealResponse is a zoneOnly reveal of a runner outside the playground: no cell, no position.
type OutsideRevealResponse struct {
	Kind       RevealKind `json:"kind" enum:"outside" validate:"required,eq=outside"`
	RevealedAt time.Time  `json:"revealedAt" example:"2026-11-08T12:05:00Z" validate:"required"`
}

// marshalVariant writes the variant a union holds; an empty union is a bug.
func marshalVariant(variant any) ([]byte, error) {
	if variant == nil {
		return nil, errors.New("union has no variant")
	}
	return json.Marshal(variant)
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

// UseJokerRequest is the joker to use: exactly one of ShowRunnersRequest,
// QueryZoneRequest, CompassRequest, ZoneOnlyRequest, FakePositionRequest, and
// ShowHuntersRequest, told apart by kind. Each variant has only the fields its
// joker needs.
type UseJokerRequest struct {
	// Which joker; it picks the variant.
	Kind    JokerKind `json:"kind" validate:"required"`
	variant any
}

// JSONSchemaOneOf documents the union in the OpenAPI spec.
func (UseJokerRequest) JSONSchemaOneOf() []any {
	return []any{ShowRunnersRequest{}, QueryZoneRequest{}, CompassRequest{}, ZoneOnlyRequest{}, FakePositionRequest{}, ShowHuntersRequest{}}
}

// UnmarshalJSON decodes the variant named by kind, rejecting fields it does not have.
func (u *UseJokerRequest) UnmarshalJSON(data []byte) error {
	var probe struct {
		Kind JokerKind `json:"kind"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}
	var err error
	u.Kind = probe.Kind
	switch game.JokerKind(probe.Kind) {
	case game.JokerShowRunners:
		u.variant, err = decodeStrict[ShowRunnersRequest](data)
	case game.JokerQueryZone:
		u.variant, err = decodeStrict[QueryZoneRequest](data)
	case game.JokerZoneOnly:
		u.variant, err = decodeStrict[ZoneOnlyRequest](data)
	case game.JokerFakePosition:
		u.variant, err = decodeStrict[FakePositionRequest](data)
	case game.JokerShowHunters:
		u.variant, err = decodeStrict[ShowHuntersRequest](data)
	case game.JokerCompass:
		u.variant, err = decodeStrict[CompassRequest](data)
	default:
		err = fmt.Errorf("unknown joker %q", probe.Kind)
	}
	return err
}

// decodeStrict decodes one JSON object into T, rejecting unknown fields.
func decodeStrict[T any](data []byte) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&value)
	return value, err
}

// Variant is the decoded request: one of the six variant structs.
func (u UseJokerRequest) Variant() any { return u.variant }

type ShowRunnersRequest struct {
	Kind JokerKind `json:"kind" enum:"showRunners" validate:"required,eq=showRunners"`
}

type QueryZoneRequest struct {
	Kind JokerKind `json:"kind" enum:"queryZone" validate:"required,eq=queryZone"`
	// The cell to query: column A-L, then row 1-12, within the game's grid.
	Cell Cell `json:"cell" pattern:"^[A-L]([1-9]|1[0-2])$" example:"C4" validate:"required,min=2,max=3"`
}

type ZoneOnlyRequest struct {
	Kind JokerKind `json:"kind" enum:"zoneOnly" validate:"required,eq=zoneOnly"`
}

// Pointer fields distinguish missing coordinates from a valid zero.
type FakePositionRequest struct {
	Kind JokerKind `json:"kind" enum:"fakePosition" validate:"required,eq=fakePosition"`
	// The point hunters see at the next reveal; it must lie inside the playground.
	Latitude  *float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52" validate:"required,gte=-80,lte=80"`
	Longitude *float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405" validate:"required,gte=-180,lte=180"`
}

type ShowHuntersRequest struct {
	Kind JokerKind `json:"kind" enum:"showHunters" validate:"required,eq=showHunters"`
}

type CompassRequest struct {
	Kind JokerKind `json:"kind" enum:"compass" validate:"required,eq=compass"`
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

// JokerResponse is one personal or team joker of the requesting player: exactly one of
// ShowRunnersJokerResponse, QueryZoneJokerResponse, CompassJokerResponse,
// ZoneOnlyJokerResponse, FakePositionJokerResponse, and
// ShowHuntersJokerResponse, told apart by kind. Each has only its joker's
// result fields, present once the player or team used it.
type JokerResponse struct{ variant any }

// JSONSchemaOneOf documents the union in the OpenAPI spec.
func (JokerResponse) JSONSchemaOneOf() []any {
	return []any{ShowRunnersJokerResponse{}, QueryZoneJokerResponse{}, CompassJokerResponse{}, ZoneOnlyJokerResponse{}, FakePositionJokerResponse{}, ShowHuntersJokerResponse{}}
}

func (j JokerResponse) MarshalJSON() ([]byte, error) { return marshalVariant(j.variant) }

// JokerStatus is what every joker response has.
type JokerStatus struct {
	// True while the game runs and the team has not used it yet.
	Available bool `json:"available" example:"true"`
	// Set once the team used it.
	UsedAt *time.Time `json:"usedAt,omitempty" example:"2026-11-08T12:07:00Z" validate:"omitempty"`
}

type ShowRunnersJokerResponse struct {
	Kind JokerKind `json:"kind" enum:"showRunners" validate:"required,eq=showRunners"`
	JokerStatus
	// Once used: each runner's position at usedAt.
	Positions []JokerPositionResponse `json:"positions,omitempty" validate:"omitempty,max=100,dive"`
}

type QueryZoneJokerResponse struct {
	Kind JokerKind `json:"kind" enum:"queryZone" validate:"required,eq=queryZone"`
	JokerStatus
	// Once used: the queried cell and whether any runner was inside it at usedAt.
	Cell  Cell  `json:"cell,omitempty" pattern:"^[A-L]([1-9]|1[0-2])$" example:"C4" validate:"omitempty,min=2,max=3"`
	Found *bool `json:"found,omitempty" example:"true"`
}

type CompassJokerResponse struct {
	Kind JokerKind `json:"kind" enum:"compass" validate:"required,eq=compass"`
	JokerStatus
	// Once used: where the hunter who used it stood, and the direction from
	// there to each runner at usedAt, in whole degrees clockwise from true
	// north, sorted. No positions, distances, or runner IDs.
	Origin   *Coordinates `json:"origin,omitempty" validate:"omitempty"`
	Bearings []int        `json:"bearings,omitempty" validate:"omitempty,max=100"`
}

type ZoneOnlyJokerResponse struct {
	Kind JokerKind `json:"kind" enum:"zoneOnly" validate:"required,eq=zoneOnly"`
	JokerStatus
	// Once used: the runner who used it.
	UsedBy PlayerID `json:"usedBy,omitempty" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677" validate:"omitempty,len=32"`
}

type FakePositionJokerResponse struct {
	Kind JokerKind `json:"kind" enum:"fakePosition" validate:"required,eq=fakePosition"`
	JokerStatus
	// Once used: the requesting runner and the point hunters see at their next reveal. Once per runner, not shared.
	UsedBy PlayerID     `json:"usedBy,omitempty" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677" validate:"omitempty,len=32"`
	Point  *Coordinates `json:"point,omitempty" validate:"omitempty"`
}

type ShowHuntersJokerResponse struct {
	Kind JokerKind `json:"kind" enum:"showHunters" validate:"required,eq=showHunters"`
	JokerStatus
	// Once used: the runner who used it and each hunter's position at usedAt.
	UsedBy    PlayerID                `json:"usedBy,omitempty" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677" validate:"omitempty,len=32"`
	Positions []JokerPositionResponse `json:"positions,omitempty" validate:"omitempty,max=100,dive"`
}

type JokerPositionResponse struct {
	PlayerID PlayerID `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677" validate:"required,len=32"`
	PositionResponse
}

// GameStateBase is what every player sees of the game.
type GameStateBase struct {
	// Development override for both Ready and finish position checks; absent during normal play.
	StartPositionCheckDisabled bool       `json:"startPositionCheckDisabled,omitempty" example:"false" validate:"omitempty"`
	SessionID                  SessionID  `json:"sessionId" minLength:"32" maxLength:"32" example:"00112233445566778899aabbccddeeff" validate:"required,len=32"`
	Status                     GameStatus `json:"status" validate:"required"`
	ServerTime                 time.Time  `json:"serverTime" example:"2026-11-08T12:00:00Z" validate:"required"`
	// Scheduled start, present during the countdown and once running.
	StartsAt *time.Time `json:"startsAt,omitempty" example:"2026-11-08T12:00:10Z" validate:"omitempty"`
	// Next time hunters receive the runners' positions, present while running.
	NextRevealAt *time.Time `json:"nextRevealAt,omitempty" example:"2026-11-08T12:05:10Z" validate:"omitempty"`
	// How often hunters receive the runners' positions.
	RevealIntervalMinutes int              `json:"revealIntervalMinutes" minimum:"1" maximum:"60" example:"30" validate:"gte=1,lte=60"`
	Playground            Playground       `json:"playground"`
	PlayerID              PlayerID         `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677" validate:"required,len=32"`
	JoinCodes             JoinCodes        `json:"joinCodes"`
	Players               []PlayerResponse `json:"players" validate:"max=100,dive"`
}

// GameStateRunnerResponse is the game as seen by a runner.
type GameStateRunnerResponse struct {
	GameStateBase
	Team Team `json:"team" enum:"runner" validate:"required,eq=runner"`
	// The game's route between two landmarks; hunters have to find them.
	Route GameRoute `json:"route"`
	// How often the runners together may still regenerate the route.
	RouteRegenerationsLeft int `json:"routeRegenerationsLeft" minimum:"0" maximum:"2" example:"2" validate:"gte=0,lte=2"`
	// Shared runner team jokers and the requesting runner's personal fake position joker.
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
	switch game.Team(probe.Team) {
	case game.Runners:
		s.runner = new(GameStateRunnerResponse)
		return json.Unmarshal(data, s.runner)
	case game.Hunters:
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

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

func (Team) Enum() []any { return []any{game.Runners, game.Hunters, game.Spectators} }

type Role game.Role

func (Role) Enum() []any { return []any{game.RolePlayer, game.RoleSpectator} }

type GameStatus game.Status

func (GameStatus) Enum() []any {
	return []any{game.StatusLobby, game.StatusCountdown, game.StatusRunning, game.StatusEnded}
}

type JokerKind game.JokerKind

func (JokerKind) Enum() []any {
	return []any{game.JokerShowRunners, game.JokerQueryZone, game.JokerZoneOnly, game.JokerFakePosition, game.JokerShowHunters, game.JokerCompass, game.JokerTrap}
}

type HealthResponse struct {
	Status  string `json:"status" enum:"ok" example:"ok"`
	Service string `json:"service" enum:"catch-me-backend" example:"catch-me-backend"`
}

type Coordinates struct {
	Latitude  float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52"`
	Longitude float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405"`
}

type Bounds struct {
	North float64 `json:"north" minimum:"-90" maximum:"90" example:"52.57395922182347"`
	South float64 `json:"south" minimum:"-90" maximum:"90" example:"52.466040778176534"`
	East  float64 `json:"east" minimum:"-180" maximum:"180" example:"13.493678060701022"`
	West  float64 `json:"west" minimum:"-180" maximum:"180" example:"13.316321939298977"`
}

// Pointer fields distinguish missing/null coordinates from a valid zero.
type CreateGameRequest struct {
	Latitude       *float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52"`
	Longitude      *float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405"`
	CellSizeMeters int      `json:"cellSizeMeters" minimum:"100" maximum:"5000" multipleOf:"100" example:"2000"`
	// Cells per side of the square grid.
	GridSize int `json:"gridSize" minimum:"3" maximum:"12" example:"6"`
	// How often hunters receive the runners' positions.
	RevealIntervalMinutes int            `json:"revealIntervalMinutes" minimum:"1" maximum:"60" example:"30"`
	Jokers                *JokerSettings `json:"jokers"`
}

type Playground struct {
	Center         Coordinates `json:"center"`
	Bounds         Bounds      `json:"bounds"`
	Rows           int         `json:"rows" minimum:"3" maximum:"12" example:"6"`
	Columns        int         `json:"columns" minimum:"3" maximum:"12" example:"6"`
	CellSizeMeters int         `json:"cellSizeMeters" minimum:"100" maximum:"5000" multipleOf:"100" example:"2000"`
}

type Landmark struct {
	// Name of the place, e.g. a landmark.
	Title       string      `json:"title" example:"Brandenburger Tor"`
	Coordinates Coordinates `json:"coordinates"`
}

// GameRoute leads from a landmark in the bottom row of the grid to one in the top row.
type GameRoute struct {
	Start Landmark `json:"start"`
	Stop  Landmark `json:"stop"`
	// Straight-line (great-circle) distance between start and stop.
	DistanceMeters float64 `json:"distanceMeters" minimum:"0" example:"10432.7"`
}

type GameResponse struct {
	SessionID  SessionID  `json:"sessionId" minLength:"32" maxLength:"32" example:"00112233445566778899aabbccddeeff"`
	CreatedAt  time.Time  `json:"createdAt" example:"2026-11-08T12:00:00Z"`
	Playground Playground `json:"playground"`
	// How often hunters receive the runners' positions.
	RevealIntervalMinutes int           `json:"revealIntervalMinutes" minimum:"1" maximum:"60" example:"30"`
	Route                 GameRoute     `json:"route"`
	JokerSettings         JokerSettings `json:"jokerSettings"`
}

type JoinCodes struct {
	Runner    string `json:"runner" minLength:"6" maxLength:"6" example:"482913"`
	Hunter    string `json:"hunter" minLength:"6" maxLength:"6" example:"071564"`
	Spectator string `json:"spectator" minLength:"6" maxLength:"6" example:"305821"`
}

type CreateGameResponse struct {
	Game      GameResponse `json:"game"`
	JoinCodes JoinCodes    `json:"joinCodes"`
}

type JoinGameRequest struct {
	Code string `json:"code" minLength:"6" maxLength:"6" example:"482913"`
	Name string `json:"name" minLength:"1" maxLength:"20" example:"Ada"`
}

// Pointer field distinguishes a missing value from false.
type SetReadyRequest struct {
	Ready *bool `json:"ready" example:"true"`
}

type SetHunterTeamRequest struct {
	Name string `json:"name" minLength:"1" maxLength:"20" example:"Wolves"`
}

type PlayerResponse struct {
	PlayerID PlayerID `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677"`
	Name     string   `json:"name" minLength:"1" maxLength:"20" example:"Ada"`
	Team     Team     `json:"team"`
	// Present for hunters who have chosen a hunter team.
	HunterTeam string `json:"hunterTeam,omitempty" maxLength:"20" example:"Wolves"`
	Ready      bool   `json:"ready" example:"false"`
	// Latest permitted report: own/hunter teammates', or runner teammates' while
	// running. Spectators receive all real latest reports only while running,
	// including finished runners' frozen last reports; caught reports use caughtPosition.
	Position *PositionResponse `json:"position,omitempty"`
	// Hunters only, for runners while running: the latest 256 reveals, oldest first.
	Reveals []RevealResponse `json:"reveals,omitempty" maxItems:"256"`
	// Present for every runner, including active runners; absent for hunters.
	Runner *RunnerStateResponse `json:"runner,omitempty"`
}

// RunnerStatus adds schema metadata without putting OpenAPI methods on the domain type.
type RunnerStatus game.RunnerStatus

func (RunnerStatus) Enum() []any {
	return []any{game.RunnerActive, game.RunnerCaught, game.RunnerFinished}
}

// RunnerStateResponse exposes one outcome; there is never a finish-position field.
type RunnerStateResponse struct {
	Status RunnerStatus `json:"status"`
	// Absent while active; server-recorded completion time for caught or finished runners.
	EndedAt *time.Time `json:"endedAt,omitempty"`
	// Last report before declaring the catch, frozen until position retention expires.
	// Only for caught runners with a retained report; never for active or finished runners.
	CaughtPosition *PositionResponse `json:"caughtPosition,omitempty"`
}

// RevealKind tells the variants of RevealResponse apart. Each value must match
// the enum tag of its variant's kind field, which the OpenAPI discriminator is
// built from.
type RevealKind string

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
	Kind       RevealKind       `json:"kind" enum:"position"`
	RevealedAt time.Time        `json:"revealedAt" example:"2026-11-08T12:05:00Z"`
	Position   PositionResponse `json:"position"`
}

// ZoneRevealResponse shows only the cell the runner was in (zoneOnly joker).
type ZoneRevealResponse struct {
	Kind       RevealKind `json:"kind" enum:"zone"`
	RevealedAt time.Time  `json:"revealedAt" example:"2026-11-08T12:05:00Z"`
	Cell       Cell       `json:"cell" pattern:"^[A-L]([1-9]|1[0-2])$" minLength:"2" maxLength:"3" example:"C4"`
}

// OutsideRevealResponse is a zoneOnly reveal of a runner outside the playground: no cell, no position.
type OutsideRevealResponse struct {
	Kind       RevealKind `json:"kind" enum:"outside"`
	RevealedAt time.Time  `json:"revealedAt" example:"2026-11-08T12:05:00Z"`
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
	AccuracyMeters float64   `json:"accuracyMeters" minimum:"0" maximum:"10000" example:"8"`
	RecordedAt     time.Time `json:"recordedAt" example:"2026-11-08T12:04:58Z"`
}

// Pointer fields distinguish missing values from zero.
type UpdatePositionRequest struct {
	Latitude       *float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52"`
	Longitude      *float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405"`
	AccuracyMeters *float64 `json:"accuracyMeters" minimum:"0" maximum:"10000" example:"8"`
}

// UseJokerRequest is the joker to use: exactly one of ShowRunnersRequest,
// QueryZoneRequest, CompassRequest, TrapRequest, ZoneOnlyRequest,
// FakePositionRequest, and ShowHuntersRequest, told apart by kind. Each
// variant has only the fields its joker needs.
type UseJokerRequest struct {
	// Which joker; it picks the variant.
	Kind    JokerKind `json:"kind"`
	variant any
}

// JSONSchemaOneOf documents the union in the OpenAPI spec.
func (UseJokerRequest) JSONSchemaOneOf() []any {
	return []any{ShowRunnersRequest{}, QueryZoneRequest{}, CompassRequest{}, TrapRequest{}, ZoneOnlyRequest{}, FakePositionRequest{}, ShowHuntersRequest{}}
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
	case game.JokerTrap:
		u.variant, err = decodeStrict[TrapRequest](data)
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

// Variant is the decoded request: one of the seven variant structs.
func (u UseJokerRequest) Variant() any { return u.variant }

type ShowRunnersRequest struct {
	Kind JokerKind `json:"kind" enum:"showRunners"`
}

type QueryZoneRequest struct {
	Kind JokerKind `json:"kind" enum:"queryZone"`
	// The cell to query: column A-L, then row 1-12, within the game's grid.
	Cell Cell `json:"cell" pattern:"^[A-L]([1-9]|1[0-2])$" minLength:"2" maxLength:"3" example:"C4"`
}

type ZoneOnlyRequest struct {
	Kind JokerKind `json:"kind" enum:"zoneOnly"`
}

// Pointer fields distinguish missing coordinates from a valid zero.
type FakePositionRequest struct {
	Kind JokerKind `json:"kind" enum:"fakePosition"`
	// The point hunters see at the next reveal; it must lie inside the playground.
	Latitude  *float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52"`
	Longitude *float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405"`
}

type ShowHuntersRequest struct {
	Kind JokerKind `json:"kind" enum:"showHunters"`
}

type CompassRequest struct {
	Kind JokerKind `json:"kind" enum:"compass"`
}

// Pointer fields distinguish missing coordinates from a valid zero.
type TrapRequest struct {
	Kind JokerKind `json:"kind" enum:"trap"`
	// Where to place the trap; it must lie inside the playground.
	Latitude  *float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52"`
	Longitude *float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405"`
}

type AddSketchRequest struct {
	Points []SketchPointRequest `json:"points" minItems:"2" maxItems:"500"`
}

type SketchPointRequest struct {
	Latitude  *float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52"`
	Longitude *float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405"`
}

type SketchResponse struct {
	SketchID  SketchID      `json:"sketchId" minLength:"32" maxLength:"32" example:"0123456789abcdef0123456789abcdef"`
	CreatedAt time.Time     `json:"createdAt" example:"2026-11-08T12:07:00Z"`
	Points    []Coordinates `json:"points" minItems:"2" maxItems:"500"`
}

// JokerResponse is one personal or team joker of the requesting player: exactly one of
// ShowRunnersJokerResponse, QueryZoneJokerResponse, CompassJokerResponse,
// TrapJokerResponse, ZoneOnlyJokerResponse, FakePositionJokerResponse, and
// ShowHuntersJokerResponse, told apart by kind. Each lists every use of its
// owner (the hunter team, the runner team, or the requesting runner for
// fakePosition), oldest first, with that joker's result fields. Remaining uses
// are the game's jokerSettings allowance minus the number of uses.
type JokerResponse struct{ variant any }

// JSONSchemaOneOf documents the union in the OpenAPI spec.
func (JokerResponse) JSONSchemaOneOf() []any {
	return []any{ShowRunnersJokerResponse{}, QueryZoneJokerResponse{}, CompassJokerResponse{}, TrapJokerResponse{}, ZoneOnlyJokerResponse{}, FakePositionJokerResponse{}, ShowHuntersJokerResponse{}}
}

func (j JokerResponse) MarshalJSON() ([]byte, error) { return marshalVariant(j.variant) }

// JokerStatus is what every joker response has.
type JokerStatus struct {
	// True while the game runs, the player is active and the allowance is not used up.
	Available bool `json:"available" example:"true"`
}

// JokerUseBase is what every joker use has.
type JokerUseBase struct {
	UsedAt time.Time `json:"usedAt" example:"2026-11-08T12:07:00Z"`
}

// RunnerJokerUseBase is what every runner joker use has.
type RunnerJokerUseBase struct {
	JokerUseBase
	// The runner who used it.
	UsedBy PlayerID `json:"usedBy" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677"`
}

type ShowRunnersJokerResponse struct {
	Kind JokerKind `json:"kind" enum:"showRunners"`
	JokerStatus
	Uses []ShowRunnersUseResponse `json:"uses" maxItems:"5"`
}

type ShowRunnersUseResponse struct {
	JokerUseBase
	// Each runner's position at usedAt.
	Positions []JokerPositionResponse `json:"positions" maxItems:"30"`
}

type QueryZoneJokerResponse struct {
	Kind JokerKind `json:"kind" enum:"queryZone"`
	JokerStatus
	Uses []QueryZoneUseResponse `json:"uses" maxItems:"5"`
}

type QueryZoneUseResponse struct {
	JokerUseBase
	// The queried cell and whether any runner was inside it at usedAt.
	Cell  Cell `json:"cell" pattern:"^[A-L]([1-9]|1[0-2])$" minLength:"2" maxLength:"3" example:"C4"`
	Found bool `json:"found" example:"true"`
}

type CompassJokerResponse struct {
	Kind JokerKind `json:"kind" enum:"compass"`
	JokerStatus
	Uses []CompassUseResponse `json:"uses" maxItems:"5"`
}

// CompassUseResponse never holds the runners' positions, distances, or IDs.
type CompassUseResponse struct {
	JokerUseBase
	// Where the hunter who used it stood.
	Origin Coordinates `json:"origin"`
	// The direction from origin to each runner at usedAt, in whole degrees
	// clockwise from true north, sorted.
	Bearings []int `json:"bearings" maxItems:"30"`
}

type TrapJokerResponse struct {
	Kind JokerKind `json:"kind" enum:"trap"`
	JokerStatus
	Uses []TrapUseResponse `json:"uses" maxItems:"5"`
}

// TrapUseResponse never tells which runner triggered it or where they were.
type TrapUseResponse struct {
	JokerUseBase
	// The trap's point; it triggers within 100 m.
	Point Coordinates `json:"point"`
	// When it becomes active, 20 minutes after usedAt.
	ActiveAt time.Time `json:"activeAt" example:"2026-11-08T12:27:00Z"`
	// Once a runner's report after activeAt came within 100 m: that report's time.
	TriggeredAt *time.Time `json:"triggeredAt,omitempty" example:"2026-11-08T12:41:30Z"`
}

type ZoneOnlyJokerResponse struct {
	Kind JokerKind `json:"kind" enum:"zoneOnly"`
	JokerStatus
	Uses []ZoneOnlyUseResponse `json:"uses" maxItems:"5"`
}

type ZoneOnlyUseResponse struct{ RunnerJokerUseBase }

// FakePositionJokerResponse lists only the requesting runner's own uses.
type FakePositionJokerResponse struct {
	Kind JokerKind `json:"kind" enum:"fakePosition"`
	JokerStatus
	Uses []FakePositionUseResponse `json:"uses" maxItems:"5"`
}

type FakePositionUseResponse struct {
	RunnerJokerUseBase
	// The point hunters see at the next reveal after usedAt.
	Point Coordinates `json:"point"`
}

type ShowHuntersJokerResponse struct {
	Kind JokerKind `json:"kind" enum:"showHunters"`
	JokerStatus
	Uses []ShowHuntersUseResponse `json:"uses" maxItems:"5"`
}

type ShowHuntersUseResponse struct {
	RunnerJokerUseBase
	// Each hunter's position at usedAt.
	Positions []JokerPositionResponse `json:"positions" maxItems:"30"`
}

type JokerPositionResponse struct {
	PlayerID PlayerID `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677"`
	PositionResponse
}

// GameStateBase is what every player sees of the game.
type GameStateBase struct {
	// Viewing permission, independent of original team. Caught/finished runners
	// become spectators automatically; only while running may they see all locations.
	Role Role `json:"role"`
	// Development override for both Ready and finish position checks; absent during normal play.
	StartPositionCheckDisabled bool       `json:"startPositionCheckDisabled,omitempty" example:"false"`
	SessionID                  SessionID  `json:"sessionId" minLength:"32" maxLength:"32" example:"00112233445566778899aabbccddeeff"`
	Status                     GameStatus `json:"status"`
	ServerTime                 time.Time  `json:"serverTime" example:"2026-11-08T12:00:00Z"`
	// Scheduled start, present during the countdown, while running, and after the end.
	StartsAt *time.Time `json:"startsAt,omitempty" example:"2026-11-08T12:00:10Z"`
	// Server-recorded end time, present only once every runner is caught or finished.
	EndedAt *time.Time `json:"endedAt,omitempty" example:"2026-11-08T13:00:10Z"`
	// Next time hunters receive the runners' positions, present while running.
	NextRevealAt *time.Time `json:"nextRevealAt,omitempty" example:"2026-11-08T12:05:10Z"`
	// How often hunters receive the runners' positions.
	RevealIntervalMinutes int              `json:"revealIntervalMinutes" minimum:"1" maximum:"60" example:"30"`
	Playground            Playground       `json:"playground"`
	PlayerID              PlayerID         `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677"`
	JoinCodes             JoinCodes        `json:"joinCodes"`
	Players               []PlayerResponse `json:"players" maxItems:"30"`
	// Immutable configuration for all joker kinds, without other teams' usage history.
	JokerSettings JokerSettings `json:"jokerSettings"`
}

// GameStateRunnerResponse is the game as seen by a runner.
type GameStateRunnerResponse struct {
	GameStateBase
	Team Team `json:"team" enum:"runner"`
	// The game's route between two landmarks; hunters have to find them.
	Route GameRoute `json:"route"`
	// How often the runners together may still regenerate the route.
	RouteRegenerationsLeft int `json:"routeRegenerationsLeft" minimum:"0" maximum:"2" example:"2"`
	// Shared runner team jokers and the requesting runner's personal fake position joker.
	Jokers []JokerResponse `json:"jokers" maxItems:"3"`
	// The requesting runner's own zone alerts, oldest first: at most one per hunter.
	ZoneAlerts []ZoneAlertResponse `json:"zoneAlerts" maxItems:"29"`
}

// ZoneAlertResponse tells a runner that a hunter was in their cell, never where
// exactly. The hunter's name and team are in the game state's players.
type ZoneAlertResponse struct {
	At       time.Time `json:"at" example:"2026-11-08T12:23:05Z"`
	HunterID PlayerID  `json:"hunterId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677"`
	Cell     Cell      `json:"cell" pattern:"^[A-L]([1-9]|1[0-2])$" minLength:"2" maxLength:"3" example:"C4"`
}

// GameStateHunterResponse is the game as seen by a hunter.
type GameStateHunterResponse struct {
	GameStateBase
	Team Team `json:"team" enum:"hunter"`
	// The hunter team's jokers; empty for hunters without a team.
	Jokers []JokerResponse `json:"jokers" maxItems:"4"`
	// The hunter team's sketches while running, oldest first.
	Sketches []SketchResponse `json:"sketches" maxItems:"200"`
}

// GameStateSpectatorResponse is a direct spectator's read-only view. Converted
// runners keep the runner variant and original affiliation, with role spectator.
type GameStateSpectatorResponse struct {
	GameStateBase
	Team Team `json:"team" enum:"spectator"`
	// The same start and destination landmarks shown to runners, read-only.
	Route GameRoute `json:"route"`
	// Always empty: spectators have no personal or team jokers.
	Jokers []JokerResponse `json:"jokers" maxItems:"0"`
}

// GameStateResponse is the game as seen by the requesting player: exactly one
// of the runner, hunter and direct spectator variants, told apart by team.
type GameStateResponse struct {
	runner    *GameStateRunnerResponse
	hunter    *GameStateHunterResponse
	spectator *GameStateSpectatorResponse
}

// JSONSchemaOneOf documents the union in the OpenAPI spec.
func (GameStateResponse) JSONSchemaOneOf() []any {
	return []any{GameStateRunnerResponse{}, GameStateHunterResponse{}, GameStateSpectatorResponse{}}
}

func NewRunnerState(state GameStateRunnerResponse) GameStateResponse {
	return GameStateResponse{runner: &state}
}

func NewHunterState(state GameStateHunterResponse) GameStateResponse {
	return GameStateResponse{hunter: &state}
}

func NewSpectatorState(state GameStateSpectatorResponse) GameStateResponse {
	return GameStateResponse{spectator: &state}
}

// Base is the metadata every authenticated participant sees.
func (s GameStateResponse) Base() GameStateBase {
	if s.runner != nil {
		return s.runner.GameStateBase
	}
	if s.hunter != nil {
		return s.hunter.GameStateBase
	}
	if s.spectator != nil {
		return s.spectator.GameStateBase
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

func (s GameStateResponse) Spectator() (GameStateSpectatorResponse, bool) {
	if s.spectator == nil {
		return GameStateSpectatorResponse{}, false
	}
	return *s.spectator, true
}

func (s GameStateResponse) MarshalJSON() ([]byte, error) {
	if s.runner != nil {
		return json.Marshal(s.runner)
	}
	if s.hunter != nil {
		return json.Marshal(s.hunter)
	}
	if s.spectator != nil {
		return json.Marshal(s.spectator)
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
	case game.Spectators:
		s.spectator = new(GameStateSpectatorResponse)
		return json.Unmarshal(data, s.spectator)
	}
	return fmt.Errorf("unknown team %q", probe.Team)
}

type JoinGameResponse struct {
	PlayerToken string            `json:"playerToken" minLength:"64" maxLength:"64" example:"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"`
	State       GameStateResponse `json:"state"`
}

type APIError struct {
	Error string `json:"error" example:"cell size must be 100 to 5000 meters in steps of 100 meters"`
}

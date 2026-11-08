package controller

type HealthResponse struct {
	Status  string `json:"status" enums:"ok" example:"ok"`
	Service string `json:"service" enums:"catch-me-backend" example:"catch-me-backend"`
} // @name HealthResponse

type Coordinates struct {
	Latitude  float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52"`
	Longitude float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405"`
} // @name Coordinates

type Bounds struct {
	North float64 `json:"north" example:"52.57395922182347"`
	South float64 `json:"south" example:"52.466040778176534"`
	East  float64 `json:"east" example:"13.493678060701022"`
	West  float64 `json:"west" example:"13.316321939298977"`
} // @name Bounds

// Pointer fields distinguish missing/null coordinates from a valid zero.
type CreateGameRequest struct {
	Latitude       *float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52"`
	Longitude      *float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405"`
	CellSizeMeters *int     `json:"cellSizeMeters,omitempty" minimum:"100" maximum:"5000" default:"2000" example:"2000"`
} // @name CreateGameRequest

type Playground struct {
	Center         Coordinates `json:"center"`
	Bounds         Bounds      `json:"bounds"`
	Rows           int         `json:"rows" enums:"6" example:"6"`
	Columns        int         `json:"columns" enums:"6" example:"6"`
	CellSizeMeters int         `json:"cellSizeMeters" minimum:"100" maximum:"5000" example:"2000"`
} // @name Playground

type GameResponse struct {
	SessionID  string     `json:"sessionId" minLength:"32" maxLength:"32" example:"00112233445566778899aabbccddeeff"`
	CreatedAt  string     `json:"createdAt" format:"date-time" example:"2026-11-08T12:00:00Z"`
	Playground Playground `json:"playground"`
} // @name GameResponse

type JoinCodes struct {
	Runner string `json:"runner" minLength:"6" maxLength:"6" example:"K7M2QX"`
	Hunter string `json:"hunter" minLength:"6" maxLength:"6" example:"P4R9TW"`
} // @name JoinCodes

type CreateGameResponse struct {
	Game      GameResponse `json:"game"`
	JoinCodes JoinCodes    `json:"joinCodes"`
} // @name CreateGameResponse

type JoinGameRequest struct {
	Code string `json:"code" minLength:"6" maxLength:"6" example:"K7M2QX"`
	Name string `json:"name" minLength:"1" maxLength:"20" example:"Ada"`
} // @name JoinGameRequest

// Pointer field distinguishes a missing value from false.
type SetReadyRequest struct {
	Ready *bool `json:"ready" example:"true"`
} // @name SetReadyRequest

type PlayerResponse struct {
	PlayerID string `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677"`
	Name     string `json:"name" minLength:"1" maxLength:"20" example:"Ada"`
	Team     string `json:"team" enums:"runner,hunter" example:"runner"`
	Ready    bool   `json:"ready" example:"false"`
	// Present only for runners when the requester is a runner.
	Route *RouteResponse `json:"route,omitempty"`
	// Own position always; teammates' positions while running; runners'
	// revealed positions for hunters while running.
	Position *PositionResponse `json:"position,omitempty"`
} // @name PlayerResponse

type PositionResponse struct {
	Latitude       float64 `json:"latitude" example:"52.52"`
	Longitude      float64 `json:"longitude" example:"13.405"`
	AccuracyMeters float64 `json:"accuracyMeters" minimum:"0" maximum:"10000" example:"8"`
	RecordedAt     string  `json:"recordedAt" format:"date-time" example:"2026-11-08T12:04:58Z"`
	// Set when this is a runner position revealed to hunters.
	RevealedAt string `json:"revealedAt,omitempty" format:"date-time" example:"2026-11-08T12:05:00Z"`
} // @name PositionResponse

// Pointer fields distinguish missing values from zero.
type UpdatePositionRequest struct {
	Latitude       *float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52"`
	Longitude      *float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405"`
	AccuracyMeters *float64 `json:"accuracyMeters" minimum:"0" maximum:"10000" example:"8"`
} // @name UpdatePositionRequest

// GameStateResponse is the game as seen by the requesting player.
type GameStateResponse struct {
	SessionID  string `json:"sessionId" minLength:"32" maxLength:"32" example:"00112233445566778899aabbccddeeff"`
	Status     string `json:"status" enums:"lobby,countdown,running" example:"lobby"`
	ServerTime string `json:"serverTime" format:"date-time" example:"2026-11-08T12:00:00Z"`
	// Scheduled start, present during the countdown and once running.
	StartsAt string `json:"startsAt,omitempty" format:"date-time" example:"2026-11-08T12:00:10Z"`
	// Next time hunters receive the runners' positions, present while running.
	NextRevealAt string           `json:"nextRevealAt,omitempty" format:"date-time" example:"2026-11-08T12:05:10Z"`
	Playground   Playground       `json:"playground"`
	PlayerID     string           `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677"`
	Team         string           `json:"team" enums:"runner,hunter" example:"runner"`
	JoinCode     string           `json:"joinCode" minLength:"6" maxLength:"6" example:"K7M2QX"`
	Players      []PlayerResponse `json:"players"`
	// The requesting runner's own route; absent for hunters.
	Route *RouteResponse `json:"route,omitempty"`
} // @name GameStateResponse

type RouteResponse struct {
	Start             string `json:"start" minLength:"2" maxLength:"2" example:"B1"`
	End               string `json:"end" minLength:"2" maxLength:"2" example:"E6"`
	RegenerationsLeft int    `json:"regenerationsLeft" minimum:"0" maximum:"2" example:"2"`
} // @name RouteResponse

type JoinGameResponse struct {
	PlayerToken string            `json:"playerToken" minLength:"64" maxLength:"64" example:"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"`
	State       GameStateResponse `json:"state"`
} // @name JoinGameResponse

type APIError struct {
	Error string `json:"error" example:"cell size must be 100 to 5000 meters in steps of 100 meters"`
} // @name ApiError

package controller

import "time"

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
	SessionID  string     `json:"sessionId" minLength:"32" maxLength:"32" example:"00112233445566778899aabbccddeeff" validate:"required,len=32"`
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
	PlayerID string `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677" validate:"required,len=32"`
	Name     string `json:"name" minLength:"1" maxLength:"20" example:"Ada" validate:"required,min=1,max=20"`
	Team     string `json:"team" enums:"runner,hunter" example:"runner" validate:"required,oneof=runner hunter"`
	// Present for hunters who have chosen a hunter team.
	HunterTeam string `json:"hunterTeam,omitempty" maxLength:"20" example:"Wolves" validate:"omitempty,max=20"`
	Ready      bool   `json:"ready" example:"false"`
	// Live position: own always; hunter teammates' always; runner teammates'
	// while running. Hunters see runners through reveals instead.
	Position *PositionResponse `json:"position,omitempty" validate:"omitempty"`
	// Hunters only, for runners while running: every reveal so far, oldest first.
	Reveals []RevealResponse `json:"reveals,omitempty" validate:"omitempty,dive"`
}

// RevealResponse is where a runner was at a reveal time.
type RevealResponse struct {
	Latitude       float64   `json:"latitude" example:"52.52" validate:"gte=-80,lte=80"`
	Longitude      float64   `json:"longitude" example:"13.405" validate:"gte=-180,lte=180"`
	AccuracyMeters float64   `json:"accuracyMeters" minimum:"0" maximum:"10000" example:"8" validate:"gte=0,lte=10000"`
	RecordedAt     time.Time `json:"recordedAt" example:"2026-11-08T12:04:58Z" validate:"required"`
	RevealedAt     time.Time `json:"revealedAt" example:"2026-11-08T12:05:00Z" validate:"required"`
}

type PositionResponse struct {
	Latitude       float64   `json:"latitude" example:"52.52" validate:"gte=-80,lte=80"`
	Longitude      float64   `json:"longitude" example:"13.405" validate:"gte=-180,lte=180"`
	AccuracyMeters float64   `json:"accuracyMeters" minimum:"0" maximum:"10000" example:"8" validate:"gte=0,lte=10000"`
	RecordedAt     time.Time `json:"recordedAt" example:"2026-11-08T12:04:58Z" validate:"required"`
}

// Pointer fields distinguish missing values from zero.
type UpdatePositionRequest struct {
	Latitude       *float64 `json:"latitude" minimum:"-80" maximum:"80" example:"52.52" validate:"required,gte=-80,lte=80"`
	Longitude      *float64 `json:"longitude" minimum:"-180" maximum:"180" example:"13.405" validate:"required,gte=-180,lte=180"`
	AccuracyMeters *float64 `json:"accuracyMeters" minimum:"0" maximum:"10000" example:"8" validate:"required,gte=0,lte=10000"`
}

// GameStateResponse is the game as seen by the requesting player.
type GameStateResponse struct {
	// Explicit server-side development override; absent during normal play.
	StartPositionCheckDisabled bool      `json:"startPositionCheckDisabled,omitempty" example:"false" validate:"omitempty"`
	SessionID                  string    `json:"sessionId" minLength:"32" maxLength:"32" example:"00112233445566778899aabbccddeeff" validate:"required,len=32"`
	Status                     string    `json:"status" enums:"lobby,countdown,running" example:"lobby" validate:"required,oneof=lobby countdown running"`
	ServerTime                 time.Time `json:"serverTime" example:"2026-11-08T12:00:00Z" validate:"required"`
	// Scheduled start, present during the countdown and once running.
	StartsAt *time.Time `json:"startsAt,omitempty" example:"2026-11-08T12:00:10Z" validate:"omitempty"`
	// Next time hunters receive the runners' positions, present while running.
	NextRevealAt *time.Time       `json:"nextRevealAt,omitempty" example:"2026-11-08T12:05:10Z" validate:"omitempty"`
	Playground   Playground       `json:"playground"`
	PlayerID     string           `json:"playerId" minLength:"32" maxLength:"32" example:"8899aabbccddeeff0011223344556677" validate:"required,len=32"`
	Team         string           `json:"team" enums:"runner,hunter" example:"runner" validate:"required,oneof=runner hunter"`
	JoinCodes    JoinCodes        `json:"joinCodes"`
	Players      []PlayerResponse `json:"players" validate:"dive"`
	// The game's route between two landmarks; only runners get it, hunters have to find them.
	Route *GameRoute `json:"route,omitempty"`
}

type JoinGameResponse struct {
	PlayerToken string            `json:"playerToken" minLength:"64" maxLength:"64" example:"00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff" validate:"required,len=64"`
	State       GameStateResponse `json:"state"`
}

type APIError struct {
	Error string `json:"error" example:"cell size must be 100 to 5000 meters in steps of 100 meters" validate:"required"`
}

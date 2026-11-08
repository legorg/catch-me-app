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

type APIError struct {
	Error string `json:"error" example:"cell size must be 100 to 5000 meters in steps of 100 meters"`
} // @name ApiError

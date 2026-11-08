package api

type HealthResponse struct {
	Status  string `json:"status" enums:"ok" example:"ok"`
	Service string `json:"service" enums:"catch-me-backend" example:"catch-me-backend"`
} // @name HealthResponse

type DemoResponse struct {
	Message    string `json:"message" example:"Hello, Runner! Your Expo app is connected to Go."`
	PlayerName string `json:"playerName" minLength:"1" maxLength:"60" example:"Runner"`
	Service    string `json:"service" enums:"catch-me-backend" example:"catch-me-backend"`
} // @name DemoResponse

type APIError struct {
	Error string `json:"error" example:"name must be valid UTF-8 and at most 60 characters"`
} // @name ApiError

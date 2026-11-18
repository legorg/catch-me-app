// Package controller translates HTTP requests and domain results/errors.
package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"catch-me/backend/game"
	"catch-me/backend/service"
)

type GameController struct{ service *service.GameService }

func NewGameController(service *service.GameService) *GameController {
	return &GameController{service: service}
}

// Create saves a playground and returns its session ID and join codes.
func (c *GameController) Create(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodPost) {
		return
	}
	var body CreateGameRequest
	if !decodeJSON(w, r, &body, "expected a JSON object with latitude, longitude, cellSizeMeters, gridSize, and revealIntervalMinutes") {
		return
	}
	center, cellSize, gridSize, revealInterval, err := body.parse()
	if err != nil {
		writeGameError(w, err, "invalid game")
		return
	}
	saved, err := c.service.Create(r.Context(), center, cellSize, gridSize, revealInterval)
	if err != nil {
		writeGameError(w, err, "could not save game")
		return
	}
	WriteJSON(w, http.StatusCreated, CreateGameResponse{
		Game:      gameResponse(saved),
		JoinCodes: JoinCodes{Runner: saved.JoinCodes.Runner.String(), Hunter: saved.JoinCodes.Hunter.String(), Spectator: saved.JoinCodes.Spectator.String()},
	})
}

// parse checks every field; all settings are required and must be positive.
func (body CreateGameRequest) parse() (center game.Coordinates, cellSize game.CellSize, gridSize game.GridSize, revealInterval game.RevealInterval, err error) {
	if body.Latitude == nil || body.Longitude == nil {
		return center, cellSize, gridSize, revealInterval, &game.ValidationError{Message: "latitude and longitude are required"}
	}
	center, err1 := game.ParseCoordinates(*body.Latitude, *body.Longitude)
	cellSize, err2 := game.ParseCellSize(body.CellSizeMeters)
	gridSize, err3 := game.ParseGridSize(body.GridSize)
	revealInterval, err4 := game.ParseRevealInterval(body.RevealIntervalMinutes)
	return center, cellSize, gridSize, revealInterval, errors.Join(err1, err2, err3, err4)
}

func gameResponse(saved game.Game) GameResponse {
	response := GameResponse{
		SessionID: SessionID(saved.SessionID.String()), CreatedAt: saved.CreatedAt,
		Playground: playgroundResponse(saved.Playground), RevealIntervalMinutes: saved.RevealInterval.Minutes(),
	}
	response.Route = gameRouteResponse(saved.Route)
	return response
}

func gameRouteResponse(route game.Route) GameRoute {
	return GameRoute{
		Start: landmarkResponse(route.Start), Stop: landmarkResponse(route.Stop),
		DistanceMeters: route.DistanceMeters(),
	}
}

func landmarkResponse(l game.Landmark) Landmark {
	return Landmark{Title: l.Title, Coordinates: coordinatesResponse(l.Coordinates)}
}

func playgroundResponse(p game.Playground) Playground {
	return Playground{
		Center: coordinatesResponse(p.Center()),
		Bounds: Bounds(p.Bounds()),
		Rows:   p.Rows(), Columns: p.Columns(), CellSizeMeters: p.CellSize().Meters(),
	}
}

// decodeJSON reads exactly one JSON object of at most 1024 bytes, rejecting
// unknown fields. On failure it writes the error response and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, body any, message string) bool {
	return decodeJSONMax(w, r, body, message, 1024)
}

// decodeJSONMax is decodeJSON with a different size limit.
func decodeJSONMax(w http.ResponseWriter, r *http.Request, body any, message string, maxBytes int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(body)
	if err == nil {
		var extra any
		if err = decoder.Decode(&extra); err == nil {
			err = errors.New("expected one JSON object")
		} else if err == io.EOF {
			return true
		}
	}
	status := http.StatusBadRequest
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		status = http.StatusRequestEntityTooLarge
	}
	WriteError(w, status, fmt.Sprintf("%s (maximum %d bytes)", message, maxBytes))
	return false
}

func writeGameError(w http.ResponseWriter, err error, message string) {
	var validation *game.ValidationError
	var rule *game.RuleError
	switch {
	case errors.As(err, &validation):
		WriteError(w, http.StatusBadRequest, validation.Error())
	case errors.Is(err, game.ErrUnauthorized):
		w.Header().Set("WWW-Authenticate", "Bearer")
		WriteError(w, http.StatusUnauthorized, game.ErrUnauthorized.Error())
	case errors.Is(err, game.ErrNoLandmark):
		WriteError(w, http.StatusUnprocessableEntity, game.ErrNoLandmark.Error())
	case errors.Is(err, game.ErrLandmarksUnavailable):
		slog.Warn(message, "err", err)
		WriteError(w, http.StatusServiceUnavailable, game.ErrLandmarksUnavailable.Error())
	case errors.Is(err, game.ErrNotFound):
		WriteError(w, http.StatusNotFound, game.ErrNotFound.Error())
	case errors.As(err, &rule):
		WriteError(w, http.StatusConflict, rule.Error())
	default:
		slog.Error(message, "err", err)
		WriteError(w, http.StatusInternalServerError, message)
	}
}

func allowMethods(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	return false
}

func coordinatesResponse(c game.Coordinates) Coordinates {
	return Coordinates{Latitude: c.Latitude(), Longitude: c.Longitude()}
}

func positionResponse(p game.Position) PositionResponse {
	return PositionResponse{
		Coordinates:    coordinatesResponse(p.Coordinates),
		AccuracyMeters: p.AccuracyMeters, RecordedAt: p.RecordedAt,
	}
}

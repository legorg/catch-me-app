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
	if !decodeJSON(w, r, &body, "expected a JSON object with latitude and longitude") {
		return
	}
	center, cellSize, err := body.parse()
	if err != nil {
		writeGameError(w, err, "invalid game")
		return
	}
	saved, err := c.service.Create(r.Context(), center, cellSize)
	if err != nil {
		writeGameError(w, err, "could not save game")
		return
	}
	WriteJSON(w, http.StatusCreated, CreateGameResponse{
		Game:      gameResponse(saved),
		JoinCodes: JoinCodes{Runner: saved.JoinCodes.Runner.String(), Hunter: saved.JoinCodes.Hunter.String()},
	})
}

func (body CreateGameRequest) parse() (game.Coordinates, game.CellSize, error) {
	if body.Latitude == nil || body.Longitude == nil {
		return game.Coordinates{}, game.CellSize{}, &game.ValidationError{Message: "latitude and longitude are required"}
	}
	center, err := game.ParseCoordinates(*body.Latitude, *body.Longitude)
	if err != nil {
		return game.Coordinates{}, game.CellSize{}, err
	}
	if body.CellSizeMeters == nil {
		return center, game.DefaultCellSize(), nil
	}
	cellSize, err := game.ParseCellSize(*body.CellSizeMeters)
	return center, cellSize, err
}

func gameResponse(saved game.Game) GameResponse {
	response := GameResponse{
		SessionID: SessionID(saved.SessionID.String()), CreatedAt: saved.CreatedAt,
		Playground: playgroundResponse(saved.Playground),
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
	return Landmark{Title: l.Title, Coordinates: Coordinates{Latitude: l.Coordinates.Latitude(), Longitude: l.Coordinates.Longitude()}}
}

func playgroundResponse(p game.Playground) Playground {
	center := p.Center()
	return Playground{
		Center: Coordinates{Latitude: center.Latitude(), Longitude: center.Longitude()},
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

func positionResponse(p game.Position) PositionResponse {
	return PositionResponse{
		Coordinates:    Coordinates{Latitude: p.Coordinates.Latitude(), Longitude: p.Coordinates.Longitude()},
		AccuracyMeters: p.AccuracyMeters, RecordedAt: p.RecordedAt,
	}
}

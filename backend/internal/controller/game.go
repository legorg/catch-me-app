// Package controller translates HTTP requests and domain results/errors.
package controller

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"catch-me/backend/internal/game"
	"catch-me/backend/internal/service"
)

type GameController struct{ service *service.GameService }

func NewGameController(service *service.GameService) *GameController {
	return &GameController{service: service}
}

// Create saves a playground and returns its session ID.
// @Summary Create a 6 by 6 playground with adjustable cell size (default 2000 meters)
// @ID createGame
// @Accept json
// @Produce json
// @Param body body CreateGameRequest true "Both coordinates required; cellSizeMeters is 100-5000 in steps of 100, omitted/null defaults to 2000; no date-line crossings"
// @Success 201 {object} GameResponse "Game saved"
// @Failure 400 {object} APIError "Invalid JSON, coordinates, or cell size"
// @Failure 403 {object} APIError "Browser origin is not allowed"
// @Failure 413 {object} APIError "Request body exceeds 1024 bytes"
// @Failure 405 {object} APIError "Method not allowed"
// @Failure 500 {object} APIError "Could not save game"
// @Router /api/games [post]
func (c *GameController) Create(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodPost) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var body CreateGameRequest
	decodeErr := decoder.Decode(&body)
	if decodeErr == nil {
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			decodeErr = err
			if decodeErr == nil {
				decodeErr = errors.New("expected one JSON object")
			}
		}
	}
	if decodeErr != nil {
		status := http.StatusBadRequest
		var sizeErr *http.MaxBytesError
		if errors.As(decodeErr, &sizeErr) {
			status = http.StatusRequestEntityTooLarge
		}
		WriteError(w, status, "expected a JSON object with latitude and longitude (maximum 1024 bytes)")
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
	w.Header().Set("Location", "/api/games/"+saved.SessionID.String())
	WriteJSON(w, http.StatusCreated, gameResponse(saved))
}

// Get loads the saved playground, not an authenticated player session.
// @Summary Load a saved game playground
// @ID getGame
// @Produce json
// @Param sessionId path string true "32 lowercase hexadecimal characters"
// @Success 200 {object} GameResponse "Saved game"
// @Failure 400 {object} APIError "Invalid session ID"
// @Failure 403 {object} APIError "Browser origin is not allowed"
// @Failure 404 {object} APIError "Game not found"
// @Failure 405 {object} APIError "Method not allowed"
// @Failure 500 {object} APIError "Could not load game"
// @Router /api/games/{sessionId} [get]
func (c *GameController) Get(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	id, err := game.ParseSessionID(r.PathValue("sessionId"))
	if err != nil {
		writeGameError(w, err, "invalid session ID")
		return
	}
	saved, err := c.service.Get(r.Context(), id)
	if err != nil {
		writeGameError(w, err, "could not load game")
		return
	}
	WriteJSON(w, http.StatusOK, gameResponse(saved))
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
	p := saved.Playground
	center := p.Center()
	return GameResponse{
		SessionID: saved.SessionID.String(), CreatedAt: saved.CreatedAt.Format(time.RFC3339Nano),
		Playground: Playground{
			Center: Coordinates{Latitude: center.Latitude(), Longitude: center.Longitude()},
			Bounds: Bounds(p.Bounds()),
			Rows:   p.Rows(), Columns: p.Columns(), CellSizeMeters: p.CellSize().Meters(),
		},
	}
}

func writeGameError(w http.ResponseWriter, err error, message string) {
	var validation *game.ValidationError
	switch {
	case errors.As(err, &validation):
		WriteError(w, http.StatusBadRequest, validation.Error())
	case errors.Is(err, game.ErrNotFound):
		WriteError(w, http.StatusNotFound, game.ErrNotFound.Error())
	default:
		log.Printf("%s: %v", message, err)
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

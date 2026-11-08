package controller

import (
	"net/http"
	"strings"
	"time"

	"catch-me/backend/internal/game"
	"catch-me/backend/internal/service"
)

// Join adds a player to the team of the join code.
// @Summary Join a game with a team join code
// @ID joinGame
// @Accept json
// @Produce json
// @Param body body JoinGameRequest true "Join code (case-insensitive) and display name (1-20 characters, unique per game)"
// @Success 201 {object} JoinGameResponse "Joined; keep the player token secret"
// @Failure 400 {object} APIError "Invalid JSON, join code, or name"
// @Failure 403 {object} APIError "Browser origin is not allowed"
// @Failure 404 {object} APIError "No game with this join code"
// @Failure 405 {object} APIError "Method not allowed"
// @Failure 409 {object} APIError "Game started, full, or name taken"
// @Failure 413 {object} APIError "Request body exceeds 1024 bytes"
// @Failure 500 {object} APIError "Could not join game"
// @Router /api/players [post]
func (c *GameController) Join(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodPost) {
		return
	}
	var body JoinGameRequest
	if !decodeJSON(w, r, &body, "expected a JSON object with code and name") {
		return
	}
	code, err := game.ParseJoinCode(body.Code)
	if err != nil {
		writeGameError(w, err, "invalid join code")
		return
	}
	name, err := game.ParsePlayerName(body.Name)
	if err != nil {
		writeGameError(w, err, "invalid name")
		return
	}
	view, token, err := c.service.Join(r.Context(), code, name)
	if err != nil {
		writeGameError(w, err, "could not join game")
		return
	}
	WriteJSON(w, http.StatusCreated, JoinGameResponse{PlayerToken: token.String(), State: gameStateResponse(view)})
}

// State returns the game as seen by the authenticated player.
// @Summary Get the game state for the authenticated player
// @ID getGameState
// @Produce json
// @Security PlayerToken
// @Success 200 {object} GameStateResponse "Current game state"
// @Failure 401 {object} APIError "Missing or unknown player token"
// @Failure 403 {object} APIError "Browser origin is not allowed"
// @Failure 405 {object} APIError "Method not allowed"
// @Failure 500 {object} APIError "Could not load game"
// @Router /api/me/game [get]
func (c *GameController) State(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	token, err := playerToken(r)
	if err != nil {
		writeGameError(w, err, "invalid token")
		return
	}
	view, err := c.service.State(r.Context(), token)
	if err != nil {
		writeGameError(w, err, "could not load game")
		return
	}
	WriteJSON(w, http.StatusOK, gameStateResponse(view))
}

// SetReady marks the authenticated player as ready or not ready.
// @Summary Set the authenticated player's ready state
// @ID setReady
// @Accept json
// @Produce json
// @Security PlayerToken
// @Param body body SetReadyRequest true "Ready state; the game starts 10 seconds after all players are ready and both teams have players"
// @Success 200 {object} GameStateResponse "Updated game state"
// @Failure 400 {object} APIError "Invalid JSON or missing ready value"
// @Failure 401 {object} APIError "Missing or unknown player token"
// @Failure 403 {object} APIError "Browser origin is not allowed"
// @Failure 405 {object} APIError "Method not allowed"
// @Failure 409 {object} APIError "Game has already started"
// @Failure 413 {object} APIError "Request body exceeds 1024 bytes"
// @Failure 500 {object} APIError "Could not update game"
// @Router /api/me/ready [put]
func (c *GameController) SetReady(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodPut) {
		return
	}
	token, err := playerToken(r)
	if err != nil {
		writeGameError(w, err, "invalid token")
		return
	}
	var body SetReadyRequest
	if !decodeJSON(w, r, &body, "expected a JSON object with ready") {
		return
	}
	if body.Ready == nil {
		WriteError(w, http.StatusBadRequest, "ready is required")
		return
	}
	view, err := c.service.SetReady(r.Context(), token, *body.Ready)
	if err != nil {
		writeGameError(w, err, "could not update game")
		return
	}
	WriteJSON(w, http.StatusOK, gameStateResponse(view))
}

// Leave removes the authenticated player before the game starts.
// @Summary Leave the game before it starts
// @ID leaveGame
// @Produce json
// @Security PlayerToken
// @Success 204 "Left the game; the token is no longer valid"
// @Failure 401 {object} APIError "Missing or unknown player token"
// @Failure 403 {object} APIError "Browser origin is not allowed"
// @Failure 405 {object} APIError "Method not allowed"
// @Failure 409 {object} APIError "Game has already started"
// @Failure 500 {object} APIError "Could not update game"
// @Router /api/me [delete]
func (c *GameController) Leave(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodDelete) {
		return
	}
	token, err := playerToken(r)
	if err != nil {
		writeGameError(w, err, "invalid token")
		return
	}
	if err := c.service.Leave(r.Context(), token); err != nil {
		writeGameError(w, err, "could not update game")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func playerToken(r *http.Request) (game.PlayerToken, error) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return game.PlayerToken{}, game.ErrUnauthorized
	}
	return game.ParsePlayerToken(strings.TrimSpace(token))
}

func gameStateResponse(view service.PlayerView) GameStateResponse {
	g := view.Lobby.Game
	response := GameStateResponse{
		SessionID:  g.SessionID.String(),
		Status:     string(g.Status(view.Now)),
		ServerTime: view.Now.Format(time.RFC3339Nano),
		Playground: playgroundResponse(g.Playground),
		PlayerID:   view.Me.ID.String(),
		Team:       string(view.Me.Team),
		// Only the player's own team code, to invite teammates.
		JoinCode: g.JoinCodes.For(view.Me.Team).String(),
		Players:  make([]PlayerResponse, 0, len(view.Lobby.Players)),
	}
	if !g.StartsAt.IsZero() {
		response.StartsAt = g.StartsAt.Format(time.RFC3339Nano)
	}
	for _, p := range view.Lobby.Players {
		response.Players = append(response.Players, PlayerResponse{
			PlayerID: p.ID.String(), Name: p.Name.String(), Team: string(p.Team), Ready: p.Ready,
		})
	}
	return response
}

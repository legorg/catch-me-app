package controller

import (
	"net/http"
	"strings"

	"catch-me/backend/game"
	"catch-me/backend/service"
)

// Join adds a player to the team of the join code.
// @Summary Join a game with a team join code
// @ID joinGame
// @Accept json
// @Produce json
// @Param body body JoinGameRequest true "Join code (6 digits) and display name (1-20 characters, unique per game)"
// @Success 201 {object} JoinGameResponse "Joined; keep the player token secret"
// @Failure 400 {object} APIError "Invalid JSON, join code, or name"
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
// @Failure 405 {object} APIError "Method not allowed"
// @Failure 409 {object} APIError "Game has already started, or the runner is not in their start cell"
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

// RegenerateRoute gives the authenticated runner a different random route.
// @Summary Regenerate the authenticated runner's route
// @ID regenerateRoute
// @Produce json
// @Security PlayerToken
// @Success 200 {object} GameStateResponse "Updated game state with the new route"
// @Failure 401 {object} APIError "Missing or unknown player token"
// @Failure 405 {object} APIError "Method not allowed"
// @Failure 409 {object} APIError "Not a runner, ready, no changes left, or game started"
// @Failure 500 {object} APIError "Could not update game"
// @Router /api/me/route [post]
func (c *GameController) RegenerateRoute(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodPost) {
		return
	}
	token, err := playerToken(r)
	if err != nil {
		writeGameError(w, err, "invalid token")
		return
	}
	view, err := c.service.RegenerateRoute(r.Context(), token)
	if err != nil {
		writeGameError(w, err, "could not update game")
		return
	}
	WriteJSON(w, http.StatusOK, gameStateResponse(view))
}

// SetHunterTeam puts the authenticated hunter into a hunter team.
// @Summary Join or create a hunter team
// @ID setHunterTeam
// @Accept json
// @Produce json
// @Security PlayerToken
// @Param body body SetHunterTeamRequest true "Team name (1-20 characters); an existing team with this name (ignoring case) is joined, otherwise a new team is created"
// @Success 200 {object} GameStateResponse "Updated game state"
// @Failure 400 {object} APIError "Invalid JSON or team name"
// @Failure 401 {object} APIError "Missing or unknown player token"
// @Failure 405 {object} APIError "Method not allowed"
// @Failure 409 {object} APIError "Not a hunter, ready, or game started"
// @Failure 413 {object} APIError "Request body exceeds 1024 bytes"
// @Failure 500 {object} APIError "Could not update game"
// @Router /api/me/hunter-team [put]
func (c *GameController) SetHunterTeam(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodPut) {
		return
	}
	token, err := playerToken(r)
	if err != nil {
		writeGameError(w, err, "invalid token")
		return
	}
	var body SetHunterTeamRequest
	if !decodeJSON(w, r, &body, "expected a JSON object with name") {
		return
	}
	name, err := game.ParseHunterTeamName(body.Name)
	if err != nil {
		writeGameError(w, err, "invalid team name")
		return
	}
	view, err := c.service.SetHunterTeam(r.Context(), token, name)
	if err != nil {
		writeGameError(w, err, "could not update game")
		return
	}
	WriteJSON(w, http.StatusOK, gameStateResponse(view))
}

// UpdatePosition stores the authenticated player's latest location.
// @Summary Report the authenticated player's current position
// @ID updatePosition
// @Accept json
// @Produce json
// @Security PlayerToken
// @Param body body UpdatePositionRequest true "Current coordinates and accuracy radius; the server records its own time"
// @Success 200 {object} GameStateResponse "Updated game state"
// @Failure 400 {object} APIError "Invalid JSON, coordinates, or accuracy"
// @Failure 401 {object} APIError "Missing or unknown player token"
// @Failure 405 {object} APIError "Method not allowed"
// @Failure 413 {object} APIError "Request body exceeds 1024 bytes"
// @Failure 500 {object} APIError "Could not update game"
// @Router /api/me/position [put]
func (c *GameController) UpdatePosition(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodPut) {
		return
	}
	token, err := playerToken(r)
	if err != nil {
		writeGameError(w, err, "invalid token")
		return
	}
	var body UpdatePositionRequest
	if !decodeJSON(w, r, &body, "expected a JSON object with latitude, longitude, and accuracyMeters") {
		return
	}
	if body.Latitude == nil || body.Longitude == nil || body.AccuracyMeters == nil {
		WriteError(w, http.StatusBadRequest, "latitude, longitude, and accuracyMeters are required")
		return
	}
	coordinates, err := game.ParseCoordinates(*body.Latitude, *body.Longitude)
	if err != nil {
		writeGameError(w, err, "invalid position")
		return
	}
	view, err := c.service.UpdatePosition(r.Context(), token, coordinates, *body.AccuracyMeters)
	if err != nil {
		writeGameError(w, err, "could not update game")
		return
	}
	WriteJSON(w, http.StatusOK, gameStateResponse(view))
}

// UseJoker spends a joker of the authenticated hunter's team.
// @Summary Use a joker
// @ID activateJoker
// @Accept json
// @Produce json
// @Security PlayerToken
// @Param body body UseJokerRequest true "The joker to use (with the cell for queryZone); each hunter team can use each joker once, while the game runs"
// @Success 200 {object} GameStateResponse "Updated game state, with what the joker showed"
// @Failure 400 {object} APIError "Invalid JSON, unknown joker, or missing or unexpected cell"
// @Failure 401 {object} APIError "Missing or unknown player token"
// @Failure 405 {object} APIError "Method not allowed"
// @Failure 409 {object} APIError "Not a hunter in a team, game not running, or joker already used"
// @Failure 413 {object} APIError "Request body exceeds 1024 bytes"
// @Failure 500 {object} APIError "Could not update game"
// @Router /api/me/jokers [post]
func (c *GameController) UseJoker(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodPost) {
		return
	}
	token, err := playerToken(r)
	if err != nil {
		writeGameError(w, err, "invalid token")
		return
	}
	var body UseJokerRequest
	if !decodeJSON(w, r, &body, "expected a JSON object with kind and, for queryZone, cell") {
		return
	}
	choice, err := game.ParseJokerChoice(body.Kind, body.Cell)
	if err != nil {
		writeGameError(w, err, "invalid joker")
		return
	}
	view, err := c.service.UseJoker(r.Context(), token, choice)
	if err != nil {
		writeGameError(w, err, "could not update game")
		return
	}
	WriteJSON(w, http.StatusOK, gameStateResponse(view))
}

// Sketches routes /api/me/sketches to AddSketch and ClearSketches.
func (c *GameController) Sketches(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodPost, http.MethodDelete) {
		return
	}
	if r.Method == http.MethodPost {
		c.AddSketch(w, r)
		return
	}
	c.ClearSketches(w, r)
}

// AddSketch adds a line to the map of the authenticated hunter's team.
// @Summary Draw a line on the hunter team's map
// @ID addSketch
// @Accept json
// @Produce json
// @Security PlayerToken
// @Param body body AddSketchRequest true "2-500 points of a freehand line; only the hunter's own team sees it"
// @Success 200 {object} GameStateResponse "Updated game state with the team's sketches"
// @Failure 400 {object} APIError "Invalid JSON, coordinates, or number of points"
// @Failure 401 {object} APIError "Missing or unknown player token"
// @Failure 405 {object} APIError "Method not allowed"
// @Failure 409 {object} APIError "Not a hunter in a team, game not running, or the team's sketch limit reached"
// @Failure 413 {object} APIError "Request body exceeds 65536 bytes"
// @Failure 500 {object} APIError "Could not update game"
// @Router /api/me/sketches [post]
func (c *GameController) AddSketch(w http.ResponseWriter, r *http.Request) {
	token, err := playerToken(r)
	if err != nil {
		writeGameError(w, err, "invalid token")
		return
	}
	var body AddSketchRequest
	if !decodeJSONMax(w, r, &body, "expected a JSON object with points", 64<<10) {
		return
	}
	points := make([]game.Coordinates, 0, len(body.Points))
	for _, p := range body.Points {
		if p.Latitude == nil || p.Longitude == nil {
			WriteError(w, http.StatusBadRequest, "every point needs latitude and longitude")
			return
		}
		point, err := game.ParseCoordinates(*p.Latitude, *p.Longitude)
		if err != nil {
			writeGameError(w, err, "invalid point")
			return
		}
		points = append(points, point)
	}
	line, err := game.NewSketchLine(points)
	if err != nil {
		writeGameError(w, err, "invalid sketch")
		return
	}
	view, err := c.service.AddSketch(r.Context(), token, line)
	if err != nil {
		writeGameError(w, err, "could not update game")
		return
	}
	WriteJSON(w, http.StatusOK, gameStateResponse(view))
}

// ClearSketches removes all lines of the authenticated hunter's team.
// @Summary Clear the hunter team's map
// @ID clearSketches
// @Produce json
// @Security PlayerToken
// @Success 200 {object} GameStateResponse "Updated game state without the team's sketches"
// @Failure 401 {object} APIError "Missing or unknown player token"
// @Failure 405 {object} APIError "Method not allowed"
// @Failure 409 {object} APIError "Not a hunter in a team, or game not running"
// @Failure 500 {object} APIError "Could not update game"
// @Router /api/me/sketches [delete]
func (c *GameController) ClearSketches(w http.ResponseWriter, r *http.Request) {
	token, err := playerToken(r)
	if err != nil {
		writeGameError(w, err, "invalid token")
		return
	}
	view, err := c.service.ClearSketches(r.Context(), token)
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
		StartPositionCheckDisabled: view.StartPositionCheckDisabled,
		SessionID:                  g.SessionID.String(),
		Status:                     string(g.Status(view.Now)),
		ServerTime:                 view.Now,
		Playground:                 playgroundResponse(g.Playground),
		PlayerID:                   view.Me.ID.String(),
		Team:                       string(view.Me.Team),
		// Both codes, so any player (e.g. the creator after joining) can invite to either team.
		JoinCodes: JoinCodes{Runner: g.JoinCodes.Runner.String(), Hunter: g.JoinCodes.Hunter.String()},
		Players:   make([]PlayerResponse, 0, len(view.Lobby.Players)),
	}
	if !g.StartsAt.IsZero() {
		response.StartsAt = &g.StartsAt
	}
	if next := g.NextReveal(view.Now); !next.IsZero() {
		response.NextRevealAt = &next
	}
	response.Jokers = make([]JokerResponse, 0)
	for _, j := range view.Lobby.JokersFor(view.Me, view.Now) {
		joker := JokerResponse{Kind: string(j.Kind), Available: j.Available}
		if !j.UsedAt.IsZero() {
			usedAt := j.UsedAt
			joker.UsedAt = &usedAt
			if j.Kind == game.JokerQueryZone {
				found := j.Found
				joker.Cell, joker.Found = j.Cell.String(), &found
			}
		}
		for _, p := range j.Positions {
			joker.Positions = append(joker.Positions, JokerPositionResponse{
				PlayerID: p.PlayerID.String(), Latitude: p.Position.Coordinates.Latitude(), Longitude: p.Position.Coordinates.Longitude(),
				AccuracyMeters: p.Position.AccuracyMeters, RecordedAt: p.Position.RecordedAt,
			})
		}
		response.Jokers = append(response.Jokers, joker)
	}
	response.Sketches = make([]SketchResponse, 0)
	for _, s := range view.Lobby.SketchesFor(view.Me, view.Now) {
		sketch := SketchResponse{SketchID: s.ID.String(), CreatedAt: s.CreatedAt}
		for _, p := range s.Line.Points() {
			sketch.Points = append(sketch.Points, Coordinates{Latitude: p.Latitude(), Longitude: p.Longitude()})
		}
		response.Sketches = append(response.Sketches, sketch)
	}
	if view.Me.Team == game.Runners {
		route := gameRouteResponse(g.Route)
		response.Route = &route
	}
	// Visibility is decided by the domain; this only maps what it allows.
	for _, p := range view.Lobby.VisibleTo(view.Me, view.Now) {
		player := PlayerResponse{PlayerID: p.ID.String(), Name: p.Name.String(), Team: string(p.Team), HunterTeam: p.HunterTeam.String(), Ready: p.Ready}
		if !p.Position.IsZero() {
			player.Position = &PositionResponse{
				Latitude: p.Position.Coordinates.Latitude(), Longitude: p.Position.Coordinates.Longitude(),
				AccuracyMeters: p.Position.AccuracyMeters, RecordedAt: p.Position.RecordedAt,
			}
		}
		for _, r := range p.Reveals {
			player.Reveals = append(player.Reveals, RevealResponse{
				Latitude: r.Position.Coordinates.Latitude(), Longitude: r.Position.Coordinates.Longitude(),
				AccuracyMeters: r.Position.AccuracyMeters, RecordedAt: r.Position.RecordedAt, RevealedAt: r.At,
			})
		}
		response.Players = append(response.Players, player)
	}
	return response
}

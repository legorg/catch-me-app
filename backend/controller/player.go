package controller

import (
	"net/http"
	"strings"

	"catch-me/backend/game"
	"catch-me/backend/service"
)

// Join adds a player to the team of the join code.
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

// UseJoker spends a joker of the authenticated player's team.
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
	base := gameStateBase(view)
	if view.Me.Team == game.Runners {
		return NewRunnerState(GameStateRunnerResponse{
			GameStateBase: base, Team: TeamRunner, Route: gameRouteResponse(view.Lobby.Game.Route), Jokers: jokerResponses(view),
		})
	}
	return NewHunterState(GameStateHunterResponse{
		GameStateBase: base, Team: TeamHunter,
		Jokers: jokerResponses(view), Sketches: sketchResponses(view),
	})
}

func gameStateBase(view service.PlayerView) GameStateBase {
	g := view.Lobby.Game
	response := GameStateBase{
		StartPositionCheckDisabled: view.StartPositionCheckDisabled,
		SessionID:                  SessionID(g.SessionID.String()),
		Status:                     GameStatus(g.Status(view.Now)),
		ServerTime:                 view.Now,
		Playground:                 playgroundResponse(g.Playground),
		PlayerID:                   PlayerID(view.Me.ID.String()),
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
	// Visibility is decided by the domain; this only maps what it allows.
	for _, p := range view.Lobby.VisibleTo(view.Me, view.Now) {
		player := PlayerResponse{PlayerID: PlayerID(p.ID.String()), Name: p.Name.String(), Team: Team(p.Team), HunterTeam: p.HunterTeam.String(), Ready: p.Ready}
		if !p.Position.IsZero() {
			position := positionResponse(p.Position)
			player.Position = &position
		}
		for _, r := range p.Reveals {
			player.Reveals = append(player.Reveals, revealResponse(r))
		}
		response.Players = append(response.Players, player)
	}
	return response
}

func revealResponse(r game.Reveal) RevealResponse {
	reveal := RevealResponse{RevealedAt: r.At}
	switch {
	case r.ZoneOnly && r.InPlayground:
		reveal.ZoneOnly, reveal.Cell = true, Cell(r.Cell.String())
	case r.ZoneOnly:
		reveal.ZoneOnly = true
	default:
		position := positionResponse(r.Position)
		reveal.Position = &position
	}
	return reveal
}

func jokerResponses(view service.PlayerView) []JokerResponse {
	jokers := make([]JokerResponse, 0)
	for _, j := range view.Lobby.JokersFor(view.Me, view.Now) {
		joker := JokerResponse{Kind: JokerKind(j.Kind), Available: j.Available}
		if !j.UsedAt.IsZero() {
			usedAt := j.UsedAt
			joker.UsedAt = &usedAt
			if j.UsedBy != (game.PlayerID{}) {
				joker.UsedBy = PlayerID(j.UsedBy.String())
			}
			if j.Kind == game.JokerQueryZone {
				found := j.Found
				joker.Cell, joker.Found = Cell(j.Cell.String()), &found
			}
		}
		for _, p := range j.Positions {
			joker.Positions = append(joker.Positions, JokerPositionResponse{PlayerID: PlayerID(p.PlayerID.String()), PositionResponse: positionResponse(p.Position)})
		}
		jokers = append(jokers, joker)
	}
	return jokers
}

func sketchResponses(view service.PlayerView) []SketchResponse {
	sketches := make([]SketchResponse, 0)
	for _, s := range view.Lobby.SketchesFor(view.Me, view.Now) {
		sketch := SketchResponse{SketchID: SketchID(s.ID.String()), CreatedAt: s.CreatedAt}
		for _, p := range s.Line.Points() {
			sketch.Points = append(sketch.Points, Coordinates{Latitude: p.Latitude(), Longitude: p.Longitude()})
		}
		sketches = append(sketches, sketch)
	}
	return sketches
}

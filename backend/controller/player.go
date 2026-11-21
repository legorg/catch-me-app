package controller

import (
	"net/http"

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
	token, ok := requirePlayerToken(w, r)
	if !ok {
		return
	}
	view, err := c.service.State(r.Context(), token)
	if err != nil {
		writeGameError(w, err, "could not load game")
		return
	}
	WriteJSON(w, http.StatusOK, gameStateResponse(view))
}

// Finish validates the latest stored report; the request supplies no coordinates.
func (c *GameController) Finish(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodPost) {
		return
	}
	token, ok := requirePlayerToken(w, r)
	if !ok {
		return
	}
	view, err := c.service.Finish(r.Context(), token)
	if err != nil {
		writeGameError(w, err, "could not finish runner")
		return
	}
	WriteJSON(w, http.StatusOK, gameStateResponse(view))
}

// MarkCaught permanently retires the authenticated runner, on trust alone.
func (c *GameController) MarkCaught(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodPost) {
		return
	}
	token, ok := requirePlayerToken(w, r)
	if !ok {
		return
	}
	view, err := c.service.MarkCaught(r.Context(), token)
	if err != nil {
		writeGameError(w, err, "could not mark runner caught")
		return
	}
	WriteJSON(w, http.StatusOK, gameStateResponse(view))
}

// SetReady marks the authenticated player as ready or not ready.
func (c *GameController) SetReady(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodPut) {
		return
	}
	token, ok := requirePlayerToken(w, r)
	if !ok {
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

// RegenerateRoute gives the game a different route; any runner may ask.
func (c *GameController) RegenerateRoute(w http.ResponseWriter, r *http.Request) {
	if !allowMethods(w, r, http.MethodPost) {
		return
	}
	token, ok := requirePlayerToken(w, r)
	if !ok {
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
	token, ok := requirePlayerToken(w, r)
	if !ok {
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
	token, ok := requirePlayerToken(w, r)
	if !ok {
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
	token, ok := requirePlayerToken(w, r)
	if !ok {
		return
	}
	var body UseJokerRequest
	if !decodeJSON(w, r, &body, "expected a JSON object with a known kind and exactly the fields of that joker") {
		return
	}
	choice, err := jokerChoice(body)
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

// jokerChoice parses the request variant into the joker to use.
func jokerChoice(body UseJokerRequest) (game.JokerChoice, error) {
	switch v := body.Variant().(type) {
	case ShowRunnersRequest:
		return game.ShowRunners{}, nil
	case QueryZoneRequest:
		cell, err := game.ParseCell(v.Cell)
		if err != nil {
			return nil, err
		}
		return game.QueryZone{Cell: cell}, nil
	case CompassRequest:
		return game.Compass{}, nil
	case TrapRequest:
		if v.Latitude == nil || v.Longitude == nil {
			return nil, &game.ValidationError{Message: "a point needs latitude and longitude"}
		}
		point, err := game.ParseCoordinates(*v.Latitude, *v.Longitude)
		if err != nil {
			return nil, err
		}
		return game.Trap{Point: point}, nil
	case ZoneOnlyRequest:
		return game.ZoneOnly{}, nil
	case FakePositionRequest:
		if v.Latitude == nil || v.Longitude == nil {
			return nil, &game.ValidationError{Message: "a point needs latitude and longitude"}
		}
		point, err := game.ParseCoordinates(*v.Latitude, *v.Longitude)
		if err != nil {
			return nil, err
		}
		return game.FakePosition{Point: point}, nil
	case ShowHuntersRequest:
		return game.ShowHunters{}, nil
	}
	return nil, &game.ValidationError{Message: "unknown joker"}
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
	token, ok := requirePlayerToken(w, r)
	if !ok {
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
	token, ok := requirePlayerToken(w, r)
	if !ok {
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
	token, ok := requirePlayerToken(w, r)
	if !ok {
		return
	}
	if err := c.service.Leave(r.Context(), token); err != nil {
		writeGameError(w, err, "could not update game")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func gameStateResponse(view service.PlayerView) GameStateResponse {
	base := gameStateBase(view)
	if view.Me.Team == game.Spectators {
		return NewSpectatorState(GameStateSpectatorResponse{
			GameStateBase: base, Team: Team(game.Spectators), Route: gameRouteResponse(view.Lobby.Game.Route), Jokers: []JokerResponse{},
		})
	}
	if view.Me.Team == game.Runners {
		return NewRunnerState(GameStateRunnerResponse{
			GameStateBase: base, Team: Team(game.Runners), Route: gameRouteResponse(view.Lobby.Game.Route),
			RouteRegenerationsLeft: view.Lobby.Game.RouteRegenerationsLeft(), Jokers: jokerResponses(view),
			ZoneAlerts: zoneAlertResponses(view),
		})
	}
	return NewHunterState(GameStateHunterResponse{
		GameStateBase: base, Team: Team(game.Hunters),
		Jokers: jokerResponses(view), Sketches: sketchResponses(view),
	})
}

func gameStateBase(view service.PlayerView) GameStateBase {
	g := view.Lobby.Game
	response := GameStateBase{
		Role:                       Role(view.Me.Role()),
		StartPositionCheckDisabled: view.StartPositionCheckDisabled,
		SessionID:                  SessionID(g.SessionID.String()),
		Status:                     GameStatus(g.Status(view.Now)),
		ServerTime:                 view.Now,
		Playground:                 playgroundResponse(g.Playground),
		RevealIntervalMinutes:      g.RevealInterval.Minutes(),
		JokerSettings:              jokerSettingsResponse(g.JokerSettings),
		PlayerID:                   PlayerID(view.Me.ID.String()),
		// Every available code, so authenticated participants can invite others.
		JoinCodes: JoinCodes{Runner: g.JoinCodes.Runner.String(), Hunter: g.JoinCodes.Hunter.String(), Spectator: g.JoinCodes.Spectator.String()},
		Players:   make([]PlayerResponse, 0, len(view.Lobby.Players)),
	}
	if !g.StartsAt.IsZero() {
		response.StartsAt = &g.StartsAt
	}
	if !g.EndedAt.IsZero() {
		response.EndedAt = &g.EndedAt
	}
	if next := g.NextReveal(view.Now); !next.IsZero() {
		response.NextRevealAt = &next
	}
	// Visibility is decided by the domain; this only maps what it allows.
	for _, p := range view.Lobby.VisibleTo(view.Me, view.Now) {
		player := PlayerResponse{PlayerID: PlayerID(p.ID.String()), Name: p.Name.String(), Team: Team(p.Team), HunterTeam: p.HunterTeam.String(), Ready: p.Ready}
		if p.Team == game.Runners {
			player.Runner = &RunnerStateResponse{Status: RunnerStatus(p.Runner.Status())}
			if at := p.Runner.EndedAt(); !at.IsZero() {
				player.Runner.EndedAt = &at
			}
			if !p.CaughtPosition.IsZero() {
				position := positionResponse(p.CaughtPosition)
				player.Runner.CaughtPosition = &position
			}
		}
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

// revealResponse maps what hunters see; an unknown variant shows nothing rather than a position.
func revealResponse(r game.HunterReveal) RevealResponse {
	switch r := r.(type) {
	case game.PositionReveal:
		return RevealResponse{PositionRevealResponse{Kind: RevealKindPosition, RevealedAt: r.At, Position: positionResponse(r.Position)}}
	case game.ZoneReveal:
		return RevealResponse{ZoneRevealResponse{Kind: RevealKindZone, RevealedAt: r.At, Cell: Cell(r.Cell.String())}}
	}
	return RevealResponse{OutsideRevealResponse{Kind: RevealKindOutside, RevealedAt: r.RevealedAt()}}
}

func jokerResponses(view service.PlayerView) []JokerResponse {
	jokers := make([]JokerResponse, 0)
	for _, j := range view.Lobby.JokersFor(view.Me, view.Now) {
		status, kind := JokerStatus{Available: j.Available}, JokerKind(j.Kind)
		var joker any
		switch j.Kind {
		case game.JokerShowRunners:
			joker = ShowRunnersJokerResponse{Kind: kind, JokerStatus: status, Uses: usesOf(j.Uses, func(u game.ShowRunnersUse) ShowRunnersUseResponse {
				return ShowRunnersUseResponse{JokerUseBase: useBase(u), Positions: jokerPositionResponses(u.Positions)}
			})}
		case game.JokerQueryZone:
			joker = QueryZoneJokerResponse{Kind: kind, JokerStatus: status, Uses: usesOf(j.Uses, func(u game.QueryZoneUse) QueryZoneUseResponse {
				return QueryZoneUseResponse{JokerUseBase: useBase(u), Cell: Cell(u.Cell.String()), Found: u.Found}
			})}
		case game.JokerCompass:
			joker = CompassJokerResponse{Kind: kind, JokerStatus: status, Uses: usesOf(j.Uses, func(u game.CompassUse) CompassUseResponse {
				return CompassUseResponse{JokerUseBase: useBase(u), Origin: coordinatesResponse(u.Origin), Bearings: append([]int{}, u.Bearings...)}
			})}
		case game.JokerTrap:
			joker = TrapJokerResponse{Kind: kind, JokerStatus: status, Uses: usesOf(j.Uses, func(u game.TrapUse) TrapUseResponse {
				use := TrapUseResponse{JokerUseBase: useBase(u), Point: coordinatesResponse(u.Point), ActiveAt: u.ActiveAt()}
				if !u.TriggeredAt.IsZero() {
					triggeredAt := u.TriggeredAt
					use.TriggeredAt = &triggeredAt
				}
				return use
			})}
		case game.JokerZoneOnly:
			joker = ZoneOnlyJokerResponse{Kind: kind, JokerStatus: status, Uses: usesOf(j.Uses, func(u game.ZoneOnlyUse) ZoneOnlyUseResponse {
				return ZoneOnlyUseResponse{runnerUseBase(u, u.Runner)}
			})}
		case game.JokerFakePosition:
			joker = FakePositionJokerResponse{Kind: kind, JokerStatus: status, Uses: usesOf(j.Uses, func(u game.FakePositionUse) FakePositionUseResponse {
				return FakePositionUseResponse{RunnerJokerUseBase: runnerUseBase(u, u.Runner), Point: coordinatesResponse(u.Point)}
			})}
		case game.JokerShowHunters:
			joker = ShowHuntersJokerResponse{Kind: kind, JokerStatus: status, Uses: usesOf(j.Uses, func(u game.ShowHuntersUse) ShowHuntersUseResponse {
				return ShowHuntersUseResponse{RunnerJokerUseBase: runnerUseBase(u, u.Runner), Positions: jokerPositionResponses(u.Positions)}
			})}
		default:
			continue
		}
		jokers = append(jokers, JokerResponse{joker})
	}
	return jokers
}

// usesOf converts every use of one kind; the domain lists only that kind's uses.
func usesOf[U game.JokerUse, R any](uses []game.JokerUse, convert func(U) R) []R {
	responses := make([]R, 0, len(uses))
	for _, use := range uses {
		if u, ok := use.(U); ok {
			responses = append(responses, convert(u))
		}
	}
	return responses
}

func useBase(u game.JokerUse) JokerUseBase { return JokerUseBase{UsedAt: u.UsedAt()} }

func runnerUseBase(u game.JokerUse, runner game.PlayerID) RunnerJokerUseBase {
	return RunnerJokerUseBase{JokerUseBase: useBase(u), UsedBy: PlayerID(runner.String())}
}

func jokerPositionResponses(positions []game.JokerPosition) []JokerPositionResponse {
	responses := make([]JokerPositionResponse, 0, len(positions))
	for _, p := range positions {
		responses = append(responses, JokerPositionResponse{PlayerID: PlayerID(p.PlayerID.String()), PositionResponse: positionResponse(p.Position)})
	}
	return responses
}

func zoneAlertResponses(view service.PlayerView) []ZoneAlertResponse {
	alerts := make([]ZoneAlertResponse, 0)
	for _, a := range view.Lobby.ZoneAlertsFor(view.Me) {
		alerts = append(alerts, ZoneAlertResponse{At: a.At, HunterID: PlayerID(a.Hunter.String()), Cell: Cell(a.Cell.String())})
	}
	return alerts
}

func sketchResponses(view service.PlayerView) []SketchResponse {
	sketches := make([]SketchResponse, 0)
	for _, s := range view.Lobby.SketchesFor(view.Me, view.Now) {
		sketch := SketchResponse{SketchID: SketchID(s.ID.String()), CreatedAt: s.CreatedAt}
		for _, p := range s.Line.Points() {
			sketch.Points = append(sketch.Points, coordinatesResponse(p))
		}
		sketches = append(sketches, sketch)
	}
	return sketches
}

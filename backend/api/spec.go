package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"catch-me/backend/controller"

	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi31"
)

// OpenAPI builds the OpenAPI 3.1 document of the routes in NewHandler from the
// request and response structs of the controller package.
func OpenAPI() ([]byte, error) {
	reflector := openapi31.NewReflector()
	reflector.DefaultOptions = append(reflector.DefaultOptions,
		jsonschema.InterceptProp(requiredByDefault),
		jsonschema.InterceptProp(constraintsFromValidateTags),
		jsonschema.InterceptSchema(discriminators),
	)
	spec := reflector.SpecEns()
	spec.SetTitle("Catch me! API")
	spec.SetVersion("0.1.0")
	spec.SetDescription("API generated from the request and response structs of the Go controllers.")
	spec.SetAPIKeySecurity(playerToken, "Authorization", openapi.InHeader, `"Bearer " followed by the playerToken returned when joining a game.`)
	// The token is an API key, so the documented server must use TLS.
	spec.WithServers(openapi31.Server{URL: "https://catch-me.example.com", Description: ptr("Production server behind TLS; clients configure their own base URL")})

	for _, o := range operations(nil) {
		if err := o.register(reflector); err != nil {
			return nil, err
		}
	}
	requireBodies(spec)
	requireSecurity(spec)
	return json.MarshalIndent(spec, "", "  ")
}

// requireBodies marks every request body required; swaggest leaves it optional,
// which would make the generated client treat the body as optional.
func requireBodies(spec *openapi31.Spec) {
	required := true
	for _, item := range spec.Paths.MapOfPathItemValues {
		for _, op := range []*openapi31.Operation{item.Get, item.Put, item.Post, item.Delete} {
			if op != nil && op.RequestBody != nil && op.RequestBody.RequestBody != nil {
				op.RequestBody.RequestBody.Required = &required
			}
		}
	}
}

// requireSecurity makes the player token the default requirement of every
// operation and lifts it again, explicitly, on the public ones.
func requireSecurity(spec *openapi31.Spec) {
	spec.Security = []map[string][]string{{playerToken: {}}}
	for _, item := range spec.Paths.MapOfPathItemValues {
		for _, op := range []*openapi31.Operation{item.Get, item.Put, item.Post, item.Delete} {
			if op != nil && len(op.Security) == 0 {
				op.Security = []map[string][]string{{}}
			}
		}
	}
}

func ptr[T any](v T) *T { return &v }

const playerToken = "PlayerToken"

type response struct {
	status      int
	body        any
	description string
}

type operation struct {
	method, path, id, summary string
	handler                   http.HandlerFunc
	secured                   bool
	body                      any
	bodyDescription           string
	success                   response
	failures                  []response
}

// mount registers the operations' handlers, each path once.
func mount(mux *http.ServeMux, ops []operation) {
	mounted := map[string]bool{}
	for _, o := range ops {
		if !mounted[o.path] {
			mounted[o.path] = true
			mux.HandleFunc(o.path, o.handler)
		}
	}
}

func (o operation) register(reflector *openapi31.Reflector) error {
	oc, err := reflector.NewOperationContext(o.method, o.path)
	if err != nil {
		return err
	}
	oc.SetID(o.id)
	oc.SetSummary(o.summary)
	if o.secured {
		oc.AddSecurity(playerToken)
	}
	if o.body != nil {
		oc.AddReqStructure(o.body, func(cu *openapi.ContentUnit) { cu.Description = o.bodyDescription })
	}
	for _, r := range append([]response{o.success}, o.failures...) {
		body := r.body
		if body == nil && r.status >= 400 {
			body = new(controller.APIError)
		}
		oc.AddRespStructure(body, func(cu *openapi.ContentUnit) { cu.HTTPStatus, cu.Description = r.status, r.description })
	}
	return reflector.AddOperation(oc)
}

// operations is the single list of endpoints: NewHandler mounts them and
// OpenAPI documents them, so a route cannot exist without being documented.
// The handlers check the method themselves, so each path is mounted once.
func operations(games *controller.GameController) []operation {
	return []operation{
		{method: http.MethodPost, path: "/api/games", id: "createGame", handler: games.Create, summary: "Create a square playground with adjustable grid size, cell size, and reveal interval", secured: false,
			body: new(controller.CreateGameRequest), bodyDescription: "Both coordinates required; cellSizeMeters is 100-5000 in steps of 100 (default 2000), gridSize 3-12 cells per side (default 6), revealIntervalMinutes 1-60 (default 30); omitted/null settings get their defaults; no date-line crossings",
			success:  response{status: 201, body: new(controller.CreateGameResponse), description: "Game saved, with one join code per team"},
			failures: []response{{status: 400, description: "Invalid JSON, coordinates, or cell size"}, {status: 413, description: "Request body exceeds 1024 bytes"}, {status: 405, description: "Method not allowed"}, {status: 422, description: "No landmark found at the edges of the playground"}, {status: 500, description: "Could not save game"}, {status: 503, description: "Landmark lookup failed"}}},
		{method: http.MethodPost, path: "/api/players", id: "joinGame", handler: games.Join, summary: "Join a game with a team join code", secured: false,
			body: new(controller.JoinGameRequest), bodyDescription: "Join code (6 digits) and display name (1-20 characters, unique per game)",
			success:  response{status: 201, body: new(controller.JoinGameResponse), description: "Joined; keep the player token secret"},
			failures: []response{{status: 400, description: "Invalid JSON, join code, or name"}, {status: 404, description: "No game with this join code"}, {status: 405, description: "Method not allowed"}, {status: 409, description: "Game started, full, or name taken"}, {status: 413, description: "Request body exceeds 1024 bytes"}, {status: 500, description: "Could not join game"}}},
		{method: http.MethodGet, path: "/api/me/game", id: "getGameState", handler: games.State, summary: "Get the game state for the authenticated player", secured: true,
			body: nil, bodyDescription: "",
			success:  response{status: 200, body: new(controller.GameStateResponse), description: "Current game state, including status ended and endedAt once every runner is caught or finished; each runner has runner.status (active/caught/finished), runner.endedAt only after completion, and runner.caughtPosition only for retained catch reports. Hunters have no runner object"},
			failures: []response{{status: 401, description: "Missing or unknown player token"}, {status: 405, description: "Method not allowed"}, {status: 500, description: "Could not load game"}}},
		{method: http.MethodPost, path: "/api/me/finish", id: "finish", handler: games.Finish, summary: "Finish at the route's stop (runners, while running)", secured: true,
			body: nil, bodyDescription: "",
			success:  response{status: 200, body: new(controller.GameStateResponse), description: "runner.status is finished with runner.endedAt; retries preserve the original server time, including after game end. Requires a stored report within 50 m and at most 30 seconds old, unless the development start/finish override is enabled. No finish coordinates are exposed; further position reports and jokers are rejected. The last active runner ends the game atomically, returning status ended and endedAt"},
			failures: []response{{status: 401, description: "Missing or unknown player token"}, {status: 405, description: "Method not allowed"}, {status: 409, description: "Not a runner, caught, game not running, or missing, stale, or distant position"}, {status: 500, description: "Could not finish runner"}}},
		{method: http.MethodPost, path: "/api/me/caught", id: "markCaught", handler: games.MarkCaught, summary: "Permanently mark yourself caught (runners, while running, no proximity check)", secured: true,
			body: nil, bodyDescription: "",
			success:  response{status: 200, body: new(controller.GameStateResponse), description: "runner.status is caught with runner.endedAt and optional runner.caughtPosition; retries preserve the original time and position, including after game end. Everyone sees the catch immediately; further positions and joker uses are rejected. The last active runner ends the game atomically, returning status ended and endedAt"},
			failures: []response{{status: 401, description: "Missing or unknown player token"}, {status: 405, description: "Method not allowed"}, {status: 409, description: "Not a runner, already finished, or game not running"}, {status: 500, description: "Could not mark runner caught"}}},
		{method: http.MethodPut, path: "/api/me/ready", id: "setReady", handler: games.SetReady, summary: "Set the authenticated player's ready state", secured: true,
			body: new(controller.SetReadyRequest), bodyDescription: "Ready state; the game starts 10 seconds after all players are ready and both teams have players",
			success:  response{status: 200, body: new(controller.GameStateResponse), description: "Updated game state"},
			failures: []response{{status: 400, description: "Invalid JSON or missing ready value"}, {status: 401, description: "Missing or unknown player token"}, {status: 405, description: "Method not allowed"}, {status: 409, description: "Game has already started, or the runner is not at the route's start"}, {status: 413, description: "Request body exceeds 1024 bytes"}, {status: 500, description: "Could not update game"}}},
		{method: http.MethodPost, path: "/api/me/route", id: "regenerateRoute", handler: games.RegenerateRoute, summary: "Regenerate the game's route (runners, 2 times per game, while no runner is ready)", secured: true,
			body: nil, bodyDescription: "",
			success:  response{status: 200, body: new(controller.GameStateResponse), description: "Updated game state with the new route"},
			failures: []response{{status: 401, description: "Missing or unknown player token"}, {status: 405, description: "Method not allowed"}, {status: 409, description: "Not a runner, a runner is ready, no changes left, game started, or no other route available"}, {status: 500, description: "Could not update game"}, {status: 503, description: "Landmark lookup failed"}}},
		{method: http.MethodPut, path: "/api/me/hunter-team", id: "setHunterTeam", handler: games.SetHunterTeam, summary: "Join or create a hunter team", secured: true,
			body: new(controller.SetHunterTeamRequest), bodyDescription: "Team name (1-20 characters); an existing team with this name (ignoring case) is joined, otherwise a new team is created",
			success:  response{status: 200, body: new(controller.GameStateResponse), description: "Updated game state"},
			failures: []response{{status: 400, description: "Invalid JSON or team name"}, {status: 401, description: "Missing or unknown player token"}, {status: 405, description: "Method not allowed"}, {status: 409, description: "Not a hunter, ready, or game started"}, {status: 413, description: "Request body exceeds 1024 bytes"}, {status: 500, description: "Could not update game"}}},
		{method: http.MethodPut, path: "/api/me/position", id: "updatePosition", handler: games.UpdatePosition, summary: "Report the authenticated player's current position", secured: true,
			body: new(controller.UpdatePositionRequest), bodyDescription: "Current coordinates and accuracy radius; the server records its own time",
			success:  response{status: 200, body: new(controller.GameStateResponse), description: "Updated game state"},
			failures: []response{{status: 400, description: "Invalid JSON, coordinates, or accuracy"}, {status: 401, description: "Missing or unknown player token"}, {status: 405, description: "Method not allowed"}, {status: 409, description: "Game ended, or a caught or finished runner cannot report positions"}, {status: 413, description: "Request body exceeds 1024 bytes"}, {status: 500, description: "Could not update game"}}},
		{method: http.MethodPost, path: "/api/me/jokers", id: "activateJoker", handler: games.UseJoker, summary: "Use a joker", secured: true,
			body: new(controller.UseJokerRequest), bodyDescription: "The joker to use, picked by kind, with exactly the fields that joker needs; while the game runs, fakePosition is once per runner, other runner jokers once for all runners together, and hunter jokers once per hunter team",
			success:  response{status: 200, body: new(controller.GameStateResponse), description: "Updated game state, with what the joker showed"},
			failures: []response{{status: 400, description: "Invalid JSON, unknown joker, missing or unexpected fields, or a point outside the playground"}, {status: 401, description: "Missing or unknown player token"}, {status: 405, description: "Method not allowed"}, {status: 409, description: "Caught runner, not a joker of the player's team, hunter without a team, game not running, joker already used, the runner's last reveal joker has not acted yet, or a compass without the hunter's position"}, {status: 413, description: "Request body exceeds 1024 bytes"}, {status: 500, description: "Could not update game"}}},
		{method: http.MethodPost, path: "/api/me/sketches", id: "addSketch", handler: games.Sketches, summary: "Draw a line on the hunter team's map", secured: true,
			body: new(controller.AddSketchRequest), bodyDescription: "2-500 points of a freehand line; only the hunter's own team sees it",
			success:  response{status: 200, body: new(controller.GameStateResponse), description: "Updated game state with the team's sketches"},
			failures: []response{{status: 400, description: "Invalid JSON, coordinates, or number of points"}, {status: 401, description: "Missing or unknown player token"}, {status: 405, description: "Method not allowed"}, {status: 409, description: "Not a hunter in a team, game not running, or the team's sketch limit reached"}, {status: 413, description: "Request body exceeds 65536 bytes"}, {status: 500, description: "Could not update game"}}},
		{method: http.MethodDelete, path: "/api/me/sketches", id: "clearSketches", handler: games.Sketches, summary: "Clear the hunter team's map", secured: true,
			body: nil, bodyDescription: "",
			success:  response{status: 200, body: new(controller.GameStateResponse), description: "Updated game state without the team's sketches"},
			failures: []response{{status: 401, description: "Missing or unknown player token"}, {status: 405, description: "Method not allowed"}, {status: 409, description: "Not a hunter in a team, or game not running"}, {status: 500, description: "Could not update game"}}},
		{method: http.MethodDelete, path: "/api/me", id: "leaveGame", handler: games.Leave, summary: "Leave the game before it starts", secured: true,
			body: nil, bodyDescription: "",
			success:  response{status: 204, body: nil, description: "Left the game; the token is no longer valid"},
			failures: []response{{status: 401, description: "Missing or unknown player token"}, {status: 405, description: "Method not allowed"}, {status: 409, description: "Game has already started"}, {status: 500, description: "Could not update game"}}},
		{method: http.MethodGet, path: "/health", id: "getHealth", handler: controller.Health, summary: "Check backend health", secured: false,
			body: nil, bodyDescription: "",
			success:  response{status: 200, body: new(controller.HealthResponse), description: "Backend is running"},
			failures: []response{}},
	}
}

// requiredByDefault marks every property without omitempty as required.
func requiredByDefault(p jsonschema.InterceptPropParams) error {
	if p.Processed {
		return nil
	}
	if strings.Contains(p.Field.Tag.Get("json"), ",omitempty") {
		return nil
	}
	for _, name := range p.ParentSchema.Required {
		if name == p.Name {
			return nil
		}
	}
	p.ParentSchema.Required = append(p.ParentSchema.Required, p.Name)
	return nil
}

// constraintsFromValidateTags drops nullability and derives schema constraints from the validate tags
// the controllers already enforce; explicit schema tags take precedence.
func constraintsFromValidateTags(p jsonschema.InterceptPropParams) error {
	if !p.Processed || p.PropertySchema == nil {
		return nil
	}
	schema := p.PropertySchema
	// The server omits absent values and rejects null, so pointers, slices and
	// maps are plain types for clients.
	if schema.Ref == nil {
		schema.RemoveType(jsonschema.Null)
	}
	if enum, ok := p.Field.Tag.Lookup("enum"); ok && schema.Ref != nil {
		// A field narrowed to fixed values (the team of a union variant) is not the shared enum.
		stringType := jsonschema.String.Type()
		schema.Ref, schema.Type, schema.Enum = nil, &stringType, nil
		for _, item := range strings.Split(enum, ",") {
			schema.Enum = append(schema.Enum, item)
		}
	}
	if schema.Ref != nil {
		return nil
	}
	t := p.Field.Type
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	isString, isSlice := t.Kind() == reflect.String, t.Kind() == reflect.Slice
	for _, rule := range strings.Split(p.Field.Tag.Get("validate"), ",") {
		name, value, _ := strings.Cut(rule, "=")
		number, numberErr := strconv.ParseFloat(value, 64)
		if numberErr != nil {
			if name == "oneof" && isString && len(schema.Enum) == 0 {
				for _, item := range strings.Fields(value) {
					schema.Enum = append(schema.Enum, item)
				}
			}
			continue
		}
		count := int64(number)
		switch {
		case name == "gte" && schema.Minimum == nil:
			schema.Minimum = &number
		case name == "lte" && schema.Maximum == nil:
			schema.Maximum = &number
		case name == "eq" && !isString && len(schema.Enum) == 0:
			schema.Enum = []any{number}
		case name == "len" && isString:
			setLength(schema, count, count)
		case name == "min" && isString && schema.MinLength == 0:
			schema.MinLength = count
		case name == "max" && isString && schema.MaxLength == nil:
			schema.MaxLength = &count
		case name == "min" && isSlice && schema.MinItems == 0:
			schema.MinItems = count
		case name == "max" && isSlice && schema.MaxItems == nil:
			schema.MaxItems = &count
		case name == "min" && !isString && !isSlice && schema.Minimum == nil:
			schema.Minimum = &number
		case name == "max" && !isString && !isSlice && schema.Maximum == nil:
			schema.Maximum = &number
		}
	}
	return nil
}

func setLength(schema *jsonschema.Schema, min, max int64) {
	if schema.MinLength == 0 {
		schema.MinLength = min
	}
	if schema.MaxLength == nil {
		schema.MaxLength = &max
	}
}

// discriminators tells clients which property picks the variant of each union
// (any type with JSONSchemaOneOf): the field every variant fixes with an enum
// tag, such as team or kind.
func discriminators(p jsonschema.InterceptSchemaParams) (bool, error) {
	if !p.Processed || !p.Value.IsValid() {
		return false, nil
	}
	union, ok := p.Value.Interface().(jsonschema.OneOfExposer)
	if !ok {
		return false, nil
	}
	const schemas = "#/components/schemas/"
	property, mapping := "", map[string]string{}
	for _, variant := range union.JSONSchemaOneOf() {
		t := reflect.TypeOf(variant)
		for i := range t.NumField() {
			field := t.Field(i)
			value, ok := field.Tag.Lookup("enum")
			if !ok {
				continue
			}
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			if property != "" && name != property {
				return false, fmt.Errorf("variants of %s are picked by both %s and %s", p.Value.Type(), property, name)
			}
			// Schema names are the package name followed by the type name.
			property, mapping[value] = name, schemas+"Controller"+t.Name()
		}
	}
	if len(mapping) != len(union.JSONSchemaOneOf()) {
		return false, fmt.Errorf("every variant of %s needs one enum-tagged field that picks it", p.Value.Type())
	}
	p.Schema.WithExtraPropertiesItem("discriminator", map[string]any{"propertyName": property, "mapping": mapping})
	return false, nil
}

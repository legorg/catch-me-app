package schema

import (
	"catch-me/backend/internal/game"
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Player struct{ ent.Schema }

func (Player) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "players"}}
}

func (Player) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").SchemaType(map[string]string{dialect.SQLite: "TEXT"}).
			Validate(parsedBy(game.ParsePlayerID)).Immutable(),
		field.String("game_id").Immutable(),
		field.Enum("team").Values(string(game.Runners), string(game.Hunters)).Immutable(),
		field.String("name").Validate(parsedBy(game.ParsePlayerName)).Immutable(),
		// SHA-256 of the player token; the token itself is never stored.
		field.String("token_hash").Unique().Sensitive().Validate(parsedBy(game.ParseTokenHash)).Immutable(),
		field.Bool("ready").Default(false),
		field.Time("joined_at").Immutable(),
		// Runner route; empty for hunters and for runners who joined before routes existed.
		field.String("route_start").Optional().Nillable().Validate(parsedBy(game.ParseCell)),
		field.String("route_end").Optional().Nillable().Validate(parsedBy(game.ParseCell)),
		field.Int("route_regenerations_left").Optional().Nillable().Range(0, game.MaxRouteRegenerations),
		// Latest position only, never a history. Removed with the player.
		field.Float("latitude").Optional().Nillable(),
		field.Float("longitude").Optional().Nillable(),
		field.Float("accuracy_meters").Optional().Nillable(),
		field.Time("position_recorded_at").Optional().Nillable(),
		// A runner's position as last revealed to hunters.
		field.Float("revealed_latitude").Optional().Nillable(),
		field.Float("revealed_longitude").Optional().Nillable(),
		field.Float("revealed_accuracy_meters").Optional().Nillable(),
		field.Time("revealed_recorded_at").Optional().Nillable(),
		field.Time("revealed_at").Optional().Nillable(),
	}
}

func (Player) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("game", Game.Type).Ref("players").Field("game_id").Unique().Required().Immutable(),
	}
}

func (Player) Indexes() []ent.Index {
	return []ent.Index{index.Fields("game_id")}
}

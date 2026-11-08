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

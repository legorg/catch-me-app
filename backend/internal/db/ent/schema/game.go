package schema

import (
	"catch-me/backend/internal/game"
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type Game struct{ ent.Schema }

// Keep the existing SQLite table and column names, including the string ID.
func (Game) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "games"}}
}

func (Game) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").StorageKey("session_id").SchemaType(map[string]string{dialect.SQLite: "TEXT"}).
			Validate(parsedBy(game.ParseSessionID)).Immutable(),
		// RFC3339 text, including sub-second precision, as in the legacy table.
		field.String("created_at").SchemaType(map[string]string{dialect.SQLite: "TEXT"}).NotEmpty().Immutable(),
		field.Float("latitude").Range(-game.MaxLatitude, game.MaxLatitude),
		field.Float("longitude").Range(-game.MaxLongitude, game.MaxLongitude),
		field.Float("north"), field.Float("south"), field.Float("east"), field.Float("west"),
		field.Int("row_count").StorageKey("rows").Range(game.GridSize, game.GridSize),
		field.Int("column_count").StorageKey("columns").Range(game.GridSize, game.GridSize),
		field.Int("cell_size_meters").Validate(parsedBy(game.ParseCellSize)),
	}
}

// parsedBy reuses a domain parser as an Ent validator, so the rules exist once.
func parsedBy[T, V any](parse func(T) (V, error)) func(T) error {
	return func(value T) error {
		_, err := parse(value)
		return err
	}
}

package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/Wei-Shaw/sub2api/internal/domain"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Supplier is an isolated organization that contributes upstream accounts.
type Supplier struct {
	ent.Schema
}

func (Supplier) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "suppliers"}}
}

func (Supplier) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}, mixins.SoftDeleteMixin{}}
}

func (Supplier) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").MaxLen(64).NotEmpty(),
		field.String("name").MaxLen(120).NotEmpty(),
		field.String("status").MaxLen(20).Default(domain.SupplierStatusActive),
		field.String("notes").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.JSON("allowed_account_kinds", []domain.SupplierAccountKind{}).
			Default(func() []domain.SupplierAccountKind { return []domain.SupplierAccountKind{} }).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.String("token_selector").MaxLen(64).Optional().Nillable(),
		field.String("token_hash").MaxLen(64).Optional().Nillable(),
		field.String("token_prefix").MaxLen(32).Optional().Nillable(),
		field.Time("token_created_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("token_last_used_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (Supplier) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("members", User.Type).Ref("supplier"),
		edge.From("accounts", Account.Type).Ref("supplier"),
	}
}

func (Supplier) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code"),
		index.Fields("status"),
		index.Fields("token_selector"),
		index.Fields("deleted_at"),
	}
}

package introspection

type FullType struct {
	Kind        TypeKind
	Name        *string
	Description *string
	Fields      []*FieldValue
	InputFields []*InputValue
	Interfaces  []*TypeRef
	EnumValues  []*struct {
		Name              string
		Description       *string
		IsDeprecated      bool
		DeprecationReason *string
	}
	PossibleTypes []*TypeRef
}

type FieldValue struct {
	Name              string
	Description       *string
	Args              []*InputValue
	Type              TypeRef
	IsDeprecated      bool
	DeprecationReason *string
}

type InputValue struct {
	Name         string
	Description  *string
	Type         TypeRef
	DefaultValue *string
}

type TypeRef struct {
	Kind   TypeKind
	Name   *string
	OfType *TypeRef
}

type IntrospectionQuery struct {
	Schema struct {
		QueryType        struct{ Name *string }
		MutationType     *struct{ Name *string }
		SubscriptionType *struct{ Name *string }
		Types            []*FullType
		Directives       []*DirectiveType
	} `graphql:"__schema"`
}

type DirectiveType struct {
	Name        string
	Description *string
	Locations   []string
	Args        []*InputValue
}

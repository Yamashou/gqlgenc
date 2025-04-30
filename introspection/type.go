package introspection

type TypeKind string

const (
	TypeKindScalar      TypeKind = "SCALAR"
	TypeKindObject      TypeKind = "OBJECT"
	TypeKindInterface   TypeKind = "INTERFACE"
	TypeKindUnion       TypeKind = "UNION"
	TypeKindEnum        TypeKind = "ENUM"
	TypeKindInputObject TypeKind = "INPUT_OBJECT"
	TypeKindList        TypeKind = "LIST"
	TypeKindNonNull     TypeKind = "NON_NULL"
)

type FullTypes []*FullType

func (fs FullTypes) NameMap() map[string]*FullType {
	typeMap := make(map[string]*FullType)
	for _, typ := range fs {
		typeMap[*typ.Name] = typ
	}

	return typeMap
}

type FullType struct {
	Kind        TypeKind
	Name        *string
	Description *string
	Fields      []*FieldValue
	InputFields []*InputValue
	Interfaces  []*TypeRef
	EnumValues  []*struct {
		Description       *string
		DeprecationReason *string
		Name              string
		IsDeprecated      bool
	}
	PossibleTypes []*TypeRef
}

type FieldValue struct {
	Type              TypeRef
	Description       *string
	DeprecationReason *string
	Name              string
	Args              []*InputValue
	IsDeprecated      bool
}

type InputValue struct {
	Type         TypeRef
	Description  *string
	DefaultValue *string
	Name         string
}

type TypeRef struct {
	Name   *string
	OfType *TypeRef
	Kind   TypeKind
}

type Query struct {
	Schema struct {
		QueryType        struct{ Name *string }
		MutationType     *struct{ Name *string }
		SubscriptionType *struct{ Name *string }
		Types            FullTypes
		Directives       []*DirectiveType
	} `graphql:"__schema"`
}

type DirectiveType struct {
	Name        string
	Description *string
	Locations   []string
	Args        []*InputValue
}

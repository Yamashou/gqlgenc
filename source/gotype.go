package source

import (
	"fmt"
	gotypes "go/types"
	"maps"
	"slices"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	gqlgenconfig "github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"

	"github.com/Yamashou/gqlgenc/v3/config"

	graphql "github.com/vektah/gqlparser/v2/ast"
)

type Generator struct {
	cfg    *config.Config
	binder *gqlgenconfig.Binder
	types  map[string]gotypes.Type
}

func NewGoTypesGenerator(cfg *config.Config) *Generator {
	return &Generator{
		cfg:    cfg,
		binder: cfg.GQLGenConfig.NewBinder(),
		types:  map[string]gotypes.Type{},
	}
}

func (g *Generator) CreateTypesByOperations(operations graphql.OperationList) []gotypes.Type {
	for _, operation := range operations {
		t := g.newFields(operation.Name, operation.SelectionSet).goStructType()
		g.newGoNamedTypeByGoType(operation.Name, false, t)
	}

	return g.goTypes()
}

func (g *Generator) goTypes() []gotypes.Type {
	return slices.SortedFunc(maps.Values(g.types), func(a, b gotypes.Type) int {
		return strings.Compare(strings.TrimPrefix(a.String(), "*"), strings.TrimPrefix(b.String(), "*"))
	})
}

// When parentTypeName is empty, the parent is an inline fragment
func (g *Generator) newFields(parentTypeName string, selectionSet graphql.SelectionSet) Fields {
	fields := make(Fields, 0, len(selectionSet))
	for _, selection := range selectionSet {
		fields = append(fields, g.newField(parentTypeName, selection))
	}

	return fields
}

// When parentTypeName is empty, the parent is an inline fragment
func (g *Generator) newField(parentTypeName string, selection graphql.Selection) *Field {
	switch sel := selection.(type) {
	case *graphql.Field:
		fieldKind, t := g.newFieldKindAndGoType(parentTypeName, sel)
		tags := []string{fmt.Sprintf(`json:"%s%s"`, sel.Alias, g.jsonOmitTag(sel)), fmt.Sprintf(`graphql:"%s"`, sel.Alias)}
		return NewField(fieldKind, t, sel.Name, tags)
	case *graphql.FragmentSpread:
		structType := g.newFields(sel.Name, sel.Definition.SelectionSet).goStructType()
		namedType := g.newGoNamedTypeByGoType(sel.Name, true, structType)
		return NewField(FragmentSpread, namedType, sel.Name, []string{})
	case *graphql.InlineFragment:
		structType := g.newFields("", sel.SelectionSet).goStructType()
		tags := []string{fmt.Sprintf(`graphql:"... on %s"`, sel.TypeCondition)}
		return NewField(InlineFragment, structType, sel.TypeCondition, tags)
	}
	panic("unexpected selection type")
}
func (g *Generator) newFieldKindAndGoType(parentTypeName string, sel *graphql.Field) (FieldKind, gotypes.Type) {
	fieldTypeName := layerTypeName(parentTypeName, templates.ToGo(sel.Alias))
	fields := g.newFields(fieldTypeName, sel.SelectionSet)

	if len(fields) == 0 {
		t := g.findGoTypeName(sel.Definition.Type.Name(), sel.Definition.Type.NonNull)
		return BasicType, t
	}

	if !g.cfg.GQLGencConfig.ExportQueryType {
		// default: query type is not exported
		fieldTypeName = firstLower(fieldTypeName)
	}
	t := g.newGoNamedTypeByGoType(fieldTypeName, sel.Definition.Type.NonNull, fields.goStructType())
	return OtherType, t
}

func (g *Generator) newGoNamedTypeByGoType(typeName string, nonnull bool, t gotypes.Type) gotypes.Type {
	var namedType gotypes.Type
	namedType = gotypes.NewNamed(gotypes.NewTypeName(0, g.cfg.GQLGencConfig.QueryGen.Pkg(), typeName, nil), t, nil)
	if !nonnull {
		namedType = gotypes.NewPointer(namedType)
	}
	// new type set to g.types
	g.types[namedType.String()] = namedType
	return namedType
}

// The typeName passed to the Type argument must be the type name derived from the analysis result, such as from selections
func (g *Generator) findGoTypeName(typeName string, nonNull bool) gotypes.Type {
	goType, err := g.binder.FindTypeFromName(g.cfg.GQLGenConfig.Models[typeName].Model[0])
	if err != nil {
		// If we pass the correct typeName as per implementation, it should always be found, so we panic if not
		panic(fmt.Sprintf("%+v", err))
	}
	if !nonNull {
		goType = gotypes.NewPointer(goType)
	}

	return goType
}

func (g *Generator) jsonOmitTag(field *graphql.Field) string {
	var jsonOmitTag string
	if field.Definition.Type.NonNull {
		if g.cfg.GQLGenConfig.EnableModelJsonOmitemptyTag != nil && *g.cfg.GQLGenConfig.EnableModelJsonOmitemptyTag {
			jsonOmitTag += `,omitempty`
		}
		if g.cfg.GQLGenConfig.EnableModelJsonOmitzeroTag != nil && *g.cfg.GQLGenConfig.EnableModelJsonOmitzeroTag {
			jsonOmitTag += `,omitzero`
		}
	}
	return jsonOmitTag
}

func layerTypeName(parentTypeName, fieldName string) string {
	return fmt.Sprintf("%s_%s", cases.Title(language.Und, cases.NoLower).String(parentTypeName), fieldName)
}

func firstLower(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

//////////////////////////////////////////////////////////////////////////////////////////////////
// Field

type FieldKind string

const (
	BasicType      FieldKind = "BasicType"
	FragmentSpread FieldKind = "FragmentSpread"
	InlineFragment FieldKind = "InlineFragment"
	OtherType      FieldKind = "OtherType"
)

// TODO: シンプルなtypes.Typeに置き換えられないか？
type Field struct {
	Name      string
	Type      gotypes.Type
	Tags      []string
	FieldKind FieldKind
}

func NewField(fieldKind FieldKind, fieldType gotypes.Type, name string, tags []string) *Field {
	return &Field{
		Name:      name,
		Type:      fieldType,
		Tags:      tags,
		FieldKind: fieldKind,
	}
}

func (r *Field) goVar() *gotypes.Var {
	return gotypes.NewField(0, nil, templates.ToGo(r.Name), r.Type, r.FieldKind == FragmentSpread)
}

func (r *Field) joinTags() string {
	return strings.Join(r.Tags, " ")
}

type Fields []*Field

func (fs Fields) goStructType() *gotypes.Struct {
	// Go fields do not allow fields with the same name, so we remove duplicates
	fields := fs.uniqueByName()
	vars := make([]*gotypes.Var, 0, len(fields))
	for _, field := range fields {
		vars = append(vars, field.goVar())
	}
	tags := make([]string, 0, len(fields))
	for _, field := range fields {
		tags = append(tags, field.joinTags())
	}
	return gotypes.NewStruct(vars, tags)
}

func (fs Fields) uniqueByName() Fields {
	fieldMapByName := make(map[string]*Field, len(fs))
	for _, field := range fs {
		fieldMapByName[field.Name] = field
	}
	return slices.SortedFunc(maps.Values(fieldMapByName), func(a *Field, b *Field) int {
		return strings.Compare(a.Name, b.Name)
	})
}

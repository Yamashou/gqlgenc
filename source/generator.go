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

func NewGoTypesAndOperations(cfg *config.Config, queryDocument *graphql.QueryDocument, operationQueryDocuments []*graphql.QueryDocument) ([]gotypes.Type, []*Operation) {
	g := newGenerator(cfg)
	g.createTypesByOperations(queryDocument.Operations)

	return g.goTypes(), g.operations(queryDocument, operationQueryDocuments)
}

type Generator struct {
	cfg    *config.Config
	binder *gqlgenconfig.Binder
	types  map[string]gotypes.Type
}

func newGenerator(cfg *config.Config) *Generator {
	return &Generator{
		cfg:    cfg,
		binder: cfg.GQLGenConfig.NewBinder(),
		types:  map[string]gotypes.Type{},
	}
}

func (g *Generator) goTypes() []gotypes.Type {
	return slices.SortedFunc(maps.Values(g.types), func(a, b gotypes.Type) int {
		return strings.Compare(strings.TrimPrefix(a.String(), "*"), strings.TrimPrefix(b.String(), "*"))
	})
}

func (g *Generator) createTypesByOperations(operations graphql.OperationList) {
	for _, operation := range operations {
		t, _ := g.newType(operation.Name, operation.Name, false, operation.SelectionSet)
		g.newGoNamedTypeByGoType(false, operation.Name, t)
	}
}

func (g *Generator) newType(name string, parentTypeName string, nonNull bool, selectionSet graphql.SelectionSet) (gotypes.Type, FieldKind) {
	fields := g.newFields(selectionSet, parentTypeName)
	return NewGoTypeByFields(name, nonNull, g.newFields(selectionSet, parentTypeName)).goType, fields.FieldKind()
}

// TODO: GoType消せないか検討
func (g *Generator) newGoType(name string, parentTypeName string, nonNull bool, selectionSet graphql.SelectionSet) (*GoType, FieldKind) {
	fields := g.newFields(selectionSet, parentTypeName)
	return NewGoTypeByFields(name, nonNull, fields), fields.FieldKind()
}

// When parentTypeName is empty, the parent is an inline fragment
func (g *Generator) newFields(selectionSet graphql.SelectionSet, parentTypeName string) Fields {
	fields := make(Fields, 0, len(selectionSet))
	for _, selection := range selectionSet {
		fields = append(fields, g.newField(selection, parentTypeName))
	}

	return fields
}

// When parentTypeName is empty, the parent is an inline fragment
func (g *Generator) newField(selection graphql.Selection, parentTypeName string) *Field {
	switch sel := selection.(type) {
	case *graphql.Field:
		fieldTypeName := layerTypeName(parentTypeName, templates.ToGo(sel.Alias))
		goType, fieldKind := g.newType(sel.Definition.Type.Name(), fieldTypeName, sel.Definition.Type.NonNull, sel.SelectionSet)
		var t gotypes.Type
		switch fieldKind {
		case BasicType:
			t = g.findGoTypeName(sel.Definition.Type.Name(), sel.Definition.Type.NonNull)
		case OtherType:
			if !g.cfg.GQLGencConfig.ExportQueryType {
				// default: query type is not exported
				fieldTypeName = firstLower(fieldTypeName)
			}
			t = g.newGoNamedTypeByGoType(sel.Definition.Type.NonNull, fieldTypeName, goType)
		}
		tags := []string{fmt.Sprintf(`json:"%s%s"`, sel.Alias, g.jsonOmitTag(sel)), fmt.Sprintf(`graphql:"%s"`, sel.Alias)}
		return NewField(sel.Name, t, tags, fieldKind)
	case *graphql.FragmentSpread:
		structType, _ := g.newType(sel.Name, sel.Name, true, sel.Definition.SelectionSet)
		namedType := g.newGoNamedTypeByGoType(true, sel.Name, structType)
		return NewField(sel.Name, namedType, []string{}, FragmentSpread)
	case *graphql.InlineFragment:
		structType, _ := g.newType(sel.TypeCondition, "", true, sel.SelectionSet)
		return NewField(sel.TypeCondition, structType, []string{fmt.Sprintf(`graphql:"... on %s"`, sel.TypeCondition)}, InlineFragment)
	}
	panic("unexpected selection type")
}

func layerTypeName(parentTypeName, fieldName string) string {
	return fmt.Sprintf("%s_%s", cases.Title(language.Und, cases.NoLower).String(parentTypeName), fieldName)
}

func (g *Generator) operations(queryDocument *graphql.QueryDocument, operationQueryDocuments []*graphql.QueryDocument) []*Operation {
	operationArgsMap := g.operationArgsMapByOperationName(queryDocument)
	queryDocumentsMap := queryDocumentMapByOperationName(operationQueryDocuments)

	operations := make([]*Operation, 0, len(queryDocument.Operations))
	for _, operation := range queryDocument.Operations {
		operationQueryDocument := queryDocumentsMap[operation.Name]
		args := operationArgsMap[operation.Name]
		operations = append(operations, NewOperation(operation, operationQueryDocument, args))
	}

	return operations
}

func queryDocumentMapByOperationName(queryDocuments []*graphql.QueryDocument) map[string]*graphql.QueryDocument {
	queryDocumentMap := make(map[string]*graphql.QueryDocument)
	for _, queryDocument := range queryDocuments {
		operation := queryDocument.Operations[0]
		queryDocumentMap[operation.Name] = queryDocument
	}

	return queryDocumentMap
}

func (g *Generator) operationArgsMapByOperationName(queryDocument *graphql.QueryDocument) map[string][]*OperationArgument {
	operationArgsMap := make(map[string][]*OperationArgument)
	for _, operation := range queryDocument.Operations {
		operationArgsMap[operation.Name] = g.operationArguments(operation.VariableDefinitions)
	}

	return operationArgsMap
}

func (g *Generator) operationArguments(variableDefinitions graphql.VariableDefinitionList) []*OperationArgument {
	argumentTypes := make([]*OperationArgument, 0, len(variableDefinitions))
	for _, v := range variableDefinitions {
		argumentTypes = append(argumentTypes, &OperationArgument{
			Variable: v.Variable,
			Type:     g.findGoTypeName(v.Type.Name(), v.Type.NonNull),
		})
	}

	return argumentTypes
}

func (g *Generator) newGoNamedTypeByGoType(nonnull bool, typeName string, t gotypes.Type) gotypes.Type {
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

func firstLower(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

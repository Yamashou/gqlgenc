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
		goType := g.newGoType(operation.Name, operation.Name, false, operation.SelectionSet)
		g.newGoNamedTypeByGoType(false, operation.Name, goType.goType)
	}
}

// TODO: GoType消せないか検討
func (g *Generator) newGoType(name string, parentTypeName string, nonNull bool, selectionSet graphql.SelectionSet) *GoType {
	return NewGoTypeByFields(name, nonNull, g.newFields(selectionSet, parentTypeName))
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
		goType := g.newGoType(sel.Definition.Type.Name(), fieldTypeName, sel.Definition.Type.NonNull, sel.SelectionSet)
		t := g.findOrNewGoType(fieldTypeName, goType)
		tags := []string{
			fmt.Sprintf(`json:"%s%s"`, sel.Alias, g.jsonOmitTag(sel)),
			fmt.Sprintf(`graphql:"%s"`, sel.Alias),
		}
		return NewField(sel.Name, t, tags, false, false)
	case *graphql.FragmentSpread:
		goType := g.newGoType(sel.Name, sel.Name, true, sel.Definition.SelectionSet)
		// When FragmentSpread, create named type
		// TODO: findOrNewにするとエラー
		//t := g.findOrNewGoType("", goType)
		t := g.newGoNamedTypeByGoType(true, sel.Name, goType.goType)
		return NewField(sel.Name, t, []string{}, true, false)
	case *graphql.InlineFragment:
		goType := g.newGoType(sel.TypeCondition, "", true, sel.SelectionSet)
		// When InlineFragment, not create named type
		// TODO: findOrNewにするとエラー
		t := goType.goType
		// t := g.findOrNewGoType("", goType)
		tags := []string{fmt.Sprintf(`graphql:"... on %s"`, sel.TypeCondition)}
		return NewField(sel.TypeCondition, t, tags, false, true)
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

func (g *Generator) findOrNewGoType(fieldTypeName string, goType *GoType) gotypes.Type {
	switch {
	case goType.isBasicType:
		t := g.findGoTypeName(goType.Name, goType.NonNull)
		return t
	case goType.isInlineFragment:
		return goType.goType
	case goType.isFragmentSpread:
		// Fragment fields are nonnull
		// Export Fragment types. Fragments are explicitly created by users.
		t := g.newGoNamedTypeByGoType(goType.NonNull, goType.Name, goType.goType)
		return t
	default:
		if !g.cfg.GQLGencConfig.ExportQueryType {
			// default: query type is not exported
			fieldTypeName = firstLower(fieldTypeName)
		}
		t := g.newGoNamedTypeByGoType(goType.NonNull, fieldTypeName, goType.goType)
		return t
	}
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

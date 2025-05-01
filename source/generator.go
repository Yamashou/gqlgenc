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
	// createFragmentTypes must be before createOperationResponsesTypes
	g.createFragmentTypes(queryDocument.Fragments)
	g.createOperationResponsesTypes(queryDocument.Operations)

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

func (g *Generator) createFragmentTypes(fragments graphql.FragmentDefinitionList) {
	for _, fragment := range fragments {
		fields := g.newFields(fragment.SelectionSet, fragment.Name)
		fragmentType := g.newGoNamedTypeByFields(true, fragment.Name, fields)
		g.types[fragmentType.String()] = fragmentType
	}
}

func (g *Generator) createOperationResponsesTypes(operations graphql.OperationList) {
	for _, operation := range operations {
		fields := g.newFields(operation.SelectionSet, operation.Name)
		operationResponseType := g.newGoNamedTypeByFields(false, operation.Name, fields)
		g.types[operationResponseType.String()] = operationResponseType
	}
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
		typeName := layerTypeName(parentTypeName, templates.ToGo(sel.Alias))
		fields := g.newFields(sel.SelectionSet, typeName)
		t := g.findGoTypeByFields(sel, typeName, fields)
		tags := []string{
			fmt.Sprintf(`json:"%s%s"`, sel.Alias, g.jsonOmitTag(sel)),
			fmt.Sprintf(`graphql:"%s"`, sel.Alias),
		}
		return NewField(sel.Name, t, tags, fields, false, false)
	case *graphql.FragmentSpread:
		fields := g.newFields(sel.Definition.SelectionSet, sel.Name)
		t := g.newGoNamedTypeByFields(true, sel.Name, fields)
		return NewField(sel.Name, t, []string{}, fields, true, false)
	case *graphql.InlineFragment:
		// InlineFragment keeps child elements directly as a struct, so we set types.Struct to the Type field instead of creating a NamedType.
		fields := g.newFields(sel.SelectionSet, "")
		t := fields.goStructType()
		return NewField(sel.TypeCondition, t, []string{fmt.Sprintf(`graphql:"... on %s"`, sel.TypeCondition)}, fields, false, true)
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
			Type:     g.findGoType(v.Type),
		})
	}

	return argumentTypes
}

func (g *Generator) newGoNamedTypeByFields(nonnull bool, typeName string, fields Fields) gotypes.Type {
	structType := fields.goStructType()
	namedType := gotypes.NewNamed(gotypes.NewTypeName(0, g.cfg.GQLGencConfig.QueryGen.Pkg(), typeName, nil), structType, nil)
	if nonnull {
		return namedType
	}
	return gotypes.NewPointer(namedType)
}

func (g *Generator) findGoTypeByFields(field *graphql.Field, fieldTypeName string, fieldFields Fields) gotypes.Type {
	switch {
	case fieldFields.isBasicType():
		t := g.findGoType(field.Definition.Type)
		return t
	case fieldFields.isFragmentSpread():
		// Fragment fields are nonnull
		// Export Fragment types. Fragments are explicitly created by users.
		t := g.newGoNamedTypeByFields(field.Definition.Type.NonNull, fieldTypeName, fieldFields)
		g.types[t.String()] = t
		return t
	default:
		if !g.cfg.GQLGencConfig.ExportQueryType {
			// Make types generated for Query private. These are created internally by gqlgenc.
			fieldTypeName = firstLower(fieldTypeName)
		}
		t := g.newGoNamedTypeByFields(field.Definition.Type.NonNull, fieldTypeName, fieldFields)
		g.types[t.String()] = t
		return t
	}
}

// The typeName passed to the Type argument must be the type name derived from the analysis result, such as from selections
func (g *Generator) findGoType(t *graphql.Type) gotypes.Type {
	goType, err := g.binder.FindTypeFromName(g.cfg.GQLGenConfig.Models[t.Name()].Model[0])
	if err != nil {
		// If we pass the correct typeName as per implementation, it should always be found, so we panic if not
		panic(fmt.Sprintf("%+v", err))
	}
	if t.NonNull {
		return goType
	}

	return gotypes.NewPointer(goType)
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

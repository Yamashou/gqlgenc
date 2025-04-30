package source

import (
	"fmt"
	"go/types"
	"maps"
	"slices"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	gqlgenconfig "github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"

	"github.com/Yamashou/gqlgenc/v3/config"

	"github.com/vektah/gqlparser/v2/ast"
)

func NewSource(cfg *config.Config, queryDocument *ast.QueryDocument, operationQueryDocuments []*ast.QueryDocument) ([]types.Type, []*Operation) {
	s := &Generator{
		cfg:     cfg,
		binder:  cfg.GQLGenConfig.NewBinder(),
		goTypes: map[string]types.Type{},
	}

	// createFragmentTypes must be before createOperationResponsesTypes
	s.createFragmentTypes(queryDocument.Fragments)
	s.createOperationResponsesTypes(queryDocument.Operations)

	return s.GoTypes(), s.operations(queryDocument, operationQueryDocuments)
}

type Generator struct {
	cfg     *config.Config
	binder  *gqlgenconfig.Binder
	goTypes map[string]types.Type
}

func (r *Generator) GoTypes() []types.Type {
	return slices.SortedFunc(maps.Values(r.goTypes), func(a, b types.Type) int {
		return strings.Compare(strings.TrimPrefix(a.String(), "*"), strings.TrimPrefix(b.String(), "*"))
	})
}

func (r *Generator) createFragmentTypes(fragments ast.FragmentDefinitionList) {
	for _, fragment := range fragments {
		responseFields := r.newResponseFields(fragment.SelectionSet, fragment.Name)
		fragmentType := r.newNamedType(true, fragment.Name, responseFields)
		r.goTypes[fragmentType.String()] = fragmentType
	}
}

func (r *Generator) createOperationResponsesTypes(operations ast.OperationList) {
	for _, operation := range operations {
		responseFields := r.newResponseFields(operation.SelectionSet, operation.Name)
		operationResponseType := r.newNamedType(false, operation.Name, responseFields)
		r.goTypes[operationResponseType.String()] = operationResponseType
	}
}

// When parentTypeName is empty, the parent is an inline fragment
func (r *Generator) newResponseFields(selectionSet ast.SelectionSet, parentTypeName string) ResponseFieldList {
	responseFields := make(ResponseFieldList, 0, len(selectionSet))
	for _, selection := range selectionSet {
		responseFields = append(responseFields, r.newResponseField(selection, parentTypeName))
	}

	return responseFields
}

func layerTypeName(parentTypeName, fieldName string) string {
	return fmt.Sprintf("%s_%s", cases.Title(language.Und, cases.NoLower).String(parentTypeName), fieldName)
}

// When parentTypeName is empty, the parent is an inline fragment
func (r *Generator) newResponseField(selection ast.Selection, parentTypeName string) *ResponseField {
	switch sel := selection.(type) {
	case *ast.Field:
		typeName := layerTypeName(parentTypeName, templates.ToGo(sel.Alias))
		fieldsResponseFields := r.newResponseFields(sel.SelectionSet, typeName)
		t := r.newFieldType(sel, typeName, fieldsResponseFields)

		return &ResponseField{
			Name: sel.Name,
			Type: t,
			Tags: []string{
				fmt.Sprintf(`json:"%s%s"`, sel.Alias, r.jsonOmitTag(sel)),
				fmt.Sprintf(`graphql:"%s"`, sel.Alias),
			},
			ResponseFields: fieldsResponseFields,
		}

	case *ast.FragmentSpread:
		fieldsResponseFields := r.newResponseFields(sel.Definition.SelectionSet, sel.Name)
		return &ResponseField{
			Name:             sel.Name,
			Type:             r.newNamedType(true, sel.Name, fieldsResponseFields),
			IsFragmentSpread: true,
			ResponseFields:   fieldsResponseFields,
		}

	case *ast.InlineFragment:
		// InlineFragment keeps child elements directly as a struct, so we set types.Struct to the Type field instead of creating a NamedType.
		fieldsResponseFields := r.newResponseFields(sel.SelectionSet, "")
		return &ResponseField{
			Name:             sel.TypeCondition,
			Type:             fieldsResponseFields.toGoStructType(),
			IsInlineFragment: true,
			Tags:             []string{fmt.Sprintf(`graphql:"... on %s"`, sel.TypeCondition)},
			ResponseFields:   fieldsResponseFields,
		}
	}

	panic("unexpected selection type")
}

func (r *Generator) operations(queryDocument *ast.QueryDocument, operationQueryDocuments []*ast.QueryDocument) []*Operation {
	operationArgsMap := r.operationArgsMapByOperationName(queryDocument)
	queryDocumentsMap := queryDocumentMapByOperationName(operationQueryDocuments)

	operations := make([]*Operation, 0, len(queryDocument.Operations))
	for _, operation := range queryDocument.Operations {
		operationQueryDocument := queryDocumentsMap[operation.Name]
		args := operationArgsMap[operation.Name]
		operations = append(operations, NewOperation(operation, operationQueryDocument, args))
	}

	return operations
}

func queryDocumentMapByOperationName(queryDocuments []*ast.QueryDocument) map[string]*ast.QueryDocument {
	queryDocumentMap := make(map[string]*ast.QueryDocument)
	for _, queryDocument := range queryDocuments {
		operation := queryDocument.Operations[0]
		queryDocumentMap[operation.Name] = queryDocument
	}

	return queryDocumentMap
}

func (r *Generator) operationArgsMapByOperationName(queryDocument *ast.QueryDocument) map[string][]*OperationArgument {
	operationArgsMap := make(map[string][]*OperationArgument)
	for _, operation := range queryDocument.Operations {
		operationArgsMap[operation.Name] = r.operationArguments(operation.VariableDefinitions)
	}

	return operationArgsMap
}

func (r *Generator) operationArguments(variableDefinitions ast.VariableDefinitionList) []*OperationArgument {
	argumentTypes := make([]*OperationArgument, 0, len(variableDefinitions))
	for _, v := range variableDefinitions {
		argumentTypes = append(argumentTypes, &OperationArgument{
			Variable: v.Variable,
			Type:     r.findType(v.Type),
		})
	}

	return argumentTypes
}

func (r *Generator) newNamedType(nonnull bool, typeName string, fieldsResponseFields ResponseFieldList) types.Type {
	structType := fieldsResponseFields.toGoStructType()
	namedType := types.NewNamed(types.NewTypeName(0, r.cfg.GQLGencConfig.QueryGen.Pkg(), typeName, nil), structType, nil)
	if nonnull {
		return namedType
	}
	return types.NewPointer(namedType)
}

// The typeName passed to the Type argument must be the type name derived from the analysis result, such as from selections
func (r *Generator) findType(t *ast.Type) types.Type {
	goType, err := r.binder.FindTypeFromName(r.cfg.GQLGenConfig.Models[t.Name()].Model[0])
	if err != nil {
		// If we pass the correct typeName as per implementation, it should always be found, so we panic if not
		panic(fmt.Sprintf("%+v", err))
	}
	if t.NonNull {
		return goType
	}

	return types.NewPointer(goType)
}

type ResponseField struct {
	Name             string
	IsFragmentSpread bool
	IsInlineFragment bool
	Type             types.Type
	Tags             []string
	ResponseFields   ResponseFieldList
}

func (r *ResponseField) goVar() *types.Var {
	return types.NewField(0, nil, templates.ToGo(r.Name), r.Type, r.IsFragmentSpread)
}

func (r *ResponseField) joinTags() string {
	return strings.Join(r.Tags, " ")
}

type ResponseFieldList []*ResponseField

func (rs ResponseFieldList) isFragmentSpread() bool {
	if len(rs) != 1 {
		return false
	}

	return rs[0].IsFragmentSpread
}

func (rs ResponseFieldList) isBasicType() bool {
	return len(rs) == 0
}

func (rs ResponseFieldList) toGoStructType() *types.Struct {
	// Go fields do not allow fields with the same name, so we remove duplicates
	responseFields := rs.uniqueByName()
	vars := make([]*types.Var, 0, len(responseFields))
	for _, responseField := range responseFields {
		vars = append(vars, responseField.goVar())
	}
	tags := make([]string, 0, len(responseFields))
	for _, responseField := range responseFields {
		tags = append(tags, responseField.joinTags())
	}
	return types.NewStruct(vars, tags)
}

func (rs ResponseFieldList) uniqueByName() ResponseFieldList {
	responseFieldMapByName := make(map[string]*ResponseField, len(rs))
	for _, field := range rs {
		responseFieldMapByName[field.Name] = field
	}
	return slices.SortedFunc(maps.Values(responseFieldMapByName), func(a *ResponseField, b *ResponseField) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func firstLower(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func (r *Generator) jsonOmitTag(field *ast.Field) string {
	var jsonOmitTag string
	if field.Definition.Type.NonNull {
		if r.cfg.GQLGenConfig.EnableModelJsonOmitemptyTag != nil && *r.cfg.GQLGenConfig.EnableModelJsonOmitemptyTag {
			jsonOmitTag += `,omitempty`
		}
		if r.cfg.GQLGenConfig.EnableModelJsonOmitzeroTag != nil && *r.cfg.GQLGenConfig.EnableModelJsonOmitzeroTag {
			jsonOmitTag += `,omitzero`
		}
	}
	return jsonOmitTag
}

func (r *Generator) newFieldType(field *ast.Field, typeName string, fieldsResponseFields ResponseFieldList) types.Type {
	switch {
	case fieldsResponseFields.isBasicType():
		t := r.findType(field.Definition.Type)
		return t
	case fieldsResponseFields.isFragmentSpread():
		// Fragment fields are nonnull
		// Export Fragment types. Fragments are explicitly created by users.
		t := r.newNamedType(field.Definition.Type.NonNull, typeName, fieldsResponseFields)
		r.goTypes[t.String()] = t
		return t
	default:
		if !r.cfg.GQLGencConfig.ExportQueryType {
			// Make types generated for Query private. These are created internally by gqlgenc.
			typeName = firstLower(typeName)
		}
		t := r.newNamedType(field.Definition.Type.NonNull, typeName, fieldsResponseFields)
		r.goTypes[t.String()] = t
		return t
	}
}

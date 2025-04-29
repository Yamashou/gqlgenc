package clientgenv2

import (
	"fmt"
	"github.com/vektah/gqlparser/v2/ast"
	"go/types"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"maps"
	"slices"
	"strings"

	gqlgenconfig "github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/Yamashou/gqlgenc/v3/config"
)

type SourceGenerator struct {
	cfg            *config.Config
	binder         *gqlgenconfig.Binder
	generatedTypes map[string]types.Type
}

func NewSourceGenerator(cfg *config.Config) *SourceGenerator {
	return &SourceGenerator{
		cfg:            cfg,
		binder:         cfg.GQLGenConfig.NewBinder(),
		generatedTypes: map[string]types.Type{},
	}
}

// parentTypeNameが空のときは親はinline fragment
func (r *SourceGenerator) NewResponseFields(selectionSet ast.SelectionSet, parentTypeName string) ResponseFieldList {
	responseFields := make(ResponseFieldList, 0, len(selectionSet))
	for _, selection := range selectionSet {
		responseFields = append(responseFields, r.NewResponseField(selection, parentTypeName))
	}

	return responseFields
}

func layerTypeName(parentTypeName, fieldName string) string {
	return fmt.Sprintf("%s_%s", cases.Title(language.Und, cases.NoLower).String(parentTypeName), fieldName)
}

// parentTypeNameが空のときは親はinline fragment
func (r *SourceGenerator) NewResponseField(selection ast.Selection, parentTypeName string) *ResponseField {
	switch sel := selection.(type) {
	case *ast.Field:
		typeName := layerTypeName(parentTypeName, templates.ToGo(sel.Alias))
		fieldsResponseFields := r.NewResponseFields(sel.SelectionSet, typeName)
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
		fieldsResponseFields := r.NewResponseFields(sel.Definition.SelectionSet, sel.Name)
		return &ResponseField{
			Name:             sel.Name,
			Type:             r.NewNamedType(true, sel.Name, fieldsResponseFields),
			IsFragmentSpread: true,
			ResponseFields:   fieldsResponseFields,
		}

	case *ast.InlineFragment:
		// InlineFragmentは子要素をそのままstructとして持つので、NamedTypeを作らずtypes.StructをTypeフィールドに設定する。
		fieldsResponseFields := r.NewResponseFields(sel.SelectionSet, "")
		return &ResponseField{
			Name:             sel.TypeCondition,
			Type:             fieldsResponseFields.ToGoStructType(),
			IsInlineFragment: true,
			Tags:             []string{fmt.Sprintf(`graphql:"... on %s"`, sel.TypeCondition)},
			ResponseFields:   fieldsResponseFields,
		}
	}

	panic("unexpected selection type")
}
func (r *SourceGenerator) jsonOmitTag(field *ast.Field) string {
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

func (r *SourceGenerator) newFieldType(field *ast.Field, typeName string, fieldsResponseFields ResponseFieldList) types.Type {
	switch {
	case fieldsResponseFields.IsBasicType():
		t := r.FindType(field.Definition.Type)
		return t
	case fieldsResponseFields.IsFragmentSpread():
		// Fragmentのフィールドはnonnull
		// Fragmentの型は公開する。Fragmentはユーザが明示的に作成しているものであるため。
		t := r.NewNamedType(field.Definition.Type.NonNull, typeName, fieldsResponseFields)
		r.generatedTypes[t.String()] = t
		return t
	default:
		// Queryのため生成した型は非公開にする。gqlgencが内部で作成したものであるため。
		t := r.NewNamedType(field.Definition.Type.NonNull, firstLower(typeName), fieldsResponseFields)
		r.generatedTypes[t.String()] = t
		return t
	}
}

func (r *SourceGenerator) OperationArguments(variableDefinitions ast.VariableDefinitionList) []*OperationArgument {
	argumentTypes := make([]*OperationArgument, 0, len(variableDefinitions))
	for _, v := range variableDefinitions {
		argumentTypes = append(argumentTypes, &OperationArgument{
			Variable: v.Variable,
			Type:     r.binder.CopyModifiersFromAst(v.Type, r.FindType(v.Type)),
		})
	}

	return argumentTypes
}

// NewNamedType は、GraphQLに対応する存在型がなく、gqlgenc独自の型を作成する。
// コード生成するために作成時にgeneratedTypesに保存しておき、Templateに渡す。
func (r *SourceGenerator) NewNamedType(nonnull bool, typeName string, fieldsResponseFields ResponseFieldList) types.Type {
	structType := fieldsResponseFields.ToGoStructType()
	namedType := types.NewNamed(types.NewTypeName(0, r.cfg.GQLGencConfig.QueryGen.Pkg(), typeName, nil), structType, nil)
	if nonnull {
		return namedType
	}
	return types.NewPointer(namedType)
}

// Typeの引数に渡すtypeNameは解析した結果からselectionなどから求めた型の名前を渡さなければいけない
func (r *SourceGenerator) FindType(t *ast.Type) types.Type {
	goType, err := r.binder.FindTypeFromName(r.cfg.GQLGenConfig.Models[t.Name()].Model[0])
	if err != nil {
		// 実装として正しいtypeNameを渡していれば必ず見つかるはずなのでpanic
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

func (r *ResponseField) GoVar() *types.Var {
	return types.NewField(0, nil, templates.ToGo(r.Name), r.Type, r.IsFragmentSpread)
}

func (r *ResponseField) JoinTags() string {
	return strings.Join(r.Tags, " ")
}

type ResponseFieldList []*ResponseField

func (rs ResponseFieldList) IsFragmentSpread() bool {
	if len(rs) != 1 {
		return false
	}

	return rs[0].IsFragmentSpread
}

func (rs ResponseFieldList) IsInlineFragment() bool {
	if len(rs) != 1 {
		return false
	}

	return rs[0].IsInlineFragment
}

func (rs ResponseFieldList) IsBasicType() bool {
	return len(rs) == 0
}

func (rs ResponseFieldList) ToGoStructType() *types.Struct {
	// Goのフィールドは同名のフィールドは許されないので重複を削除する
	responseFields := rs.uniqueByName()
	vars := make([]*types.Var, 0, len(responseFields))
	for _, responseField := range responseFields {
		vars = append(vars, responseField.GoVar())
	}
	tags := make([]string, 0, len(responseFields))
	for _, responseField := range responseFields {
		tags = append(tags, responseField.JoinTags())
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

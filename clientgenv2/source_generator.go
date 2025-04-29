package clientgenv2

import (
	"fmt"
	"go/types"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"maps"
	"slices"
	"strings"

	gqlgenconfig "github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/Yamashou/gqlgenc/v3/config"
	"github.com/vektah/gqlparser/v2/ast"
)

type Argument struct {
	Variable string
	Type     types.Type
}

type ResponseField struct {
	Name             string
	IsFragmentSpread bool
	IsInlineFragment bool
	Type             types.Type
	Tags             []string
	ResponseFields   ResponseFieldList
}

type ResponseFieldList []*ResponseField

func (rs ResponseFieldList) ToGoStructType() *types.Struct {
	vars := make([]*types.Var, 0)
	structTags := make([]string, 0)
	// 重複するフィールドはUniqueByNameによりGoの型から除外する。
	for _, field := range rs.UniqueByName() {
		vars = append(vars, types.NewVar(0, nil, templates.ToGo(field.Name), field.Type))
		structTags = append(structTags, strings.Join(field.Tags, " "))
	}
	return types.NewStruct(vars, structTags)
}

func (rs ResponseFieldList) UniqueByName() ResponseFieldList {
	responseFieldMapByName := make(map[string]*ResponseField, len(rs))
	for _, field := range rs {
		responseFieldMapByName[field.Name] = field
	}
	return slices.Collect(maps.Values(responseFieldMapByName))
}

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

func (rs ResponseFieldList) IsStructType() bool {
	return len(rs) > 0 && !rs.IsInlineFragment() && !rs.IsFragmentSpread()
}

type SourceGenerator struct {
	cfg            *config.Config
	binder         *gqlgenconfig.Binder
	generatedTypes []*GeneratedType
}

func NewSourceGenerator(cfg *config.Config) *SourceGenerator {
	return &SourceGenerator{
		cfg:    cfg,
		binder: cfg.GQLGenConfig.NewBinder(),
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

func layerTypeName(base, thisField string) string {
	return fmt.Sprintf("%s_%s", cases.Title(language.Und, cases.NoLower).String(base), thisField)
}

// parentTypeNameが空のときは親はinline fragment
func (r *SourceGenerator) NewResponseField(selection ast.Selection, parentTypeName string) *ResponseField {
	switch selection := selection.(type) {
	case *ast.Field:
		typeName := layerTypeName(parentTypeName, templates.ToGo(selection.Alias))
		fmt.Printf("ast.Field: %s %s=====================================================\n", selection.Name, typeName)
		fieldsResponseFields := r.NewResponseFields(selection.SelectionSet, typeName)
		var baseType types.Type
		switch {
		case fieldsResponseFields.IsBasicType():
			fmt.Printf("ast.Field isBasicType(): %s\n", selection.Name)
			baseType = r.Type(selection.Definition.Type.Name())
		default:
			baseType = r.NewType(typeName, fieldsResponseFields)
		}
		fmt.Printf("ast.Field: %s-------------------------------------------\n", selection.Name)

		// GraphQLの定義がオプショナルのはtypeのポインタ型が返り、配列の定義場合はポインタのスライスの型になって返ってきます
		// return pointer type then optional type or slice pointer then slice type of definition in GraphQL.
		typ := r.binder.CopyModifiersFromAst(selection.Definition.Type, baseType)

		// TODO: omitempty, omitzero
		tags := []string{
			fmt.Sprintf(`json:"%s"`, selection.Alias),
			fmt.Sprintf(`graphql:"%s"`, selection.Alias),
		}

		return &ResponseField{
			Name:           selection.Alias,
			Type:           typ,
			Tags:           tags,
			ResponseFields: fieldsResponseFields,
		}

	case *ast.FragmentSpread:
		fmt.Printf("ast.FragmentSpread: %s\n", selection.Name)
		// この構造体はテンプレート側で使われることはなく、ast.FieldでFragment判定するために使用する
		fieldsResponseFields := r.NewResponseFields(selection.Definition.SelectionSet, selection.Name)
		typ := types.NewNamed(
			types.NewTypeName(0, r.cfg.GQLGencConfig.QueryGen.Pkg(), templates.ToGo(selection.Name), nil),
			fieldsResponseFields.ToGoStructType(),
			nil,
		)

		return &ResponseField{
			Name:             selection.Name,
			Type:             typ,
			IsFragmentSpread: true,
			ResponseFields:   fieldsResponseFields,
		}

	case *ast.InlineFragment:
		fmt.Printf("ast.InlineFragment\n")
		// InlineFragmentは子要素をそのままstructとしてもつので、ここで、構造体の型を作成します
		fieldsResponseFields := r.NewResponseFields(selection.SelectionSet, "")

		return &ResponseField{
			Name:             selection.TypeCondition,
			Type:             fieldsResponseFields.ToGoStructType(),
			IsInlineFragment: true,
			Tags:             []string{fmt.Sprintf(`graphql:"... on %s"`, selection.TypeCondition)},
			ResponseFields:   fieldsResponseFields,
		}
	}

	panic("unexpected selection type")
}

// NewType は、GraphQLに対応する存在型がなく、gqlgenc独自の型を作成する。
// コード生成するために作成時にgeneratedTypesに保存しておき、Templateに渡す。
func (r *SourceGenerator) NewType(typeName string, fieldsResponseFields ResponseFieldList) *types.Named {
	structType := fieldsResponseFields.ToGoStructType()
	namedType := types.NewNamed(types.NewTypeName(0, r.cfg.GQLGencConfig.QueryGen.Pkg(), typeName, nil), structType, nil)
	r.generatedTypes = append(r.generatedTypes, NewGeneratedType(typeName, namedType, structType))
	return namedType
}

func (r *SourceGenerator) OperationArguments(variableDefinitions ast.VariableDefinitionList) []*Argument {
	argumentTypes := make([]*Argument, 0, len(variableDefinitions))
	for _, v := range variableDefinitions {
		argumentTypes = append(argumentTypes, &Argument{
			Variable: v.Variable,
			Type:     r.binder.CopyModifiersFromAst(v.Type, r.Type(v.Type.Name())),
		})
	}

	return argumentTypes
}

// Typeの引数に渡すtypeNameは解析した結果からselectionなどから求めた型の名前を渡さなければいけない
func (r *SourceGenerator) Type(typeName string) types.Type {
	goType, err := r.binder.FindTypeFromName(r.cfg.GQLGenConfig.Models[typeName].Model[0])
	if err != nil {
		// 実装として正しいtypeNameを渡していれば必ず見つかるはずなのでpanic
		panic(fmt.Sprintf("%+v", err))
	}

	return goType
}

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

func layerTypeName(base, thisField string) string {
	return fmt.Sprintf("%s_%s", cases.Title(language.Und, cases.NoLower).String(base), thisField)
}

// parentTypeNameが空のときは親はinline fragment
func (r *SourceGenerator) NewResponseField(selection ast.Selection, parentTypeName string) *ResponseField {
	switch selection := selection.(type) {
	case *ast.Field:
		typeName := layerTypeName(parentTypeName, templates.ToGo(selection.Alias))
		fieldsResponseFields := r.NewResponseFields(selection.SelectionSet, typeName)
		var t types.Type
		switch {
		case fieldsResponseFields.IsBasicType():
			t = r.FindType(selection.Definition.Type.Name())
		case fieldsResponseFields.IsFragmentSpread():
			// Fragmentのフィールドはnonnull
			t = r.NewNamedType(typeName, fieldsResponseFields)
			r.generatedTypes[t.String()] = t
		default:
			t = types.NewPointer(r.NewNamedType(typeName, fieldsResponseFields))
			// Fragment以外のフィールドはオプショナル？ TODO: オプショナルを元のスキーマの型に従う
			r.generatedTypes[t.String()] = t
		}

		// TODO: omitempty, omitzero
		tags := []string{
			fmt.Sprintf(`json:"%s"`, selection.Alias),
			fmt.Sprintf(`graphql:"%s"`, selection.Alias),
		}

		fmt.Printf("name: %s, tags: %v\n", selection.Name, tags)

		return &ResponseField{
			Name:           selection.Name,
			Type:           t,
			Tags:           tags,
			ResponseFields: fieldsResponseFields,
		}

	case *ast.FragmentSpread:
		fieldsResponseFields := r.NewResponseFields(selection.Definition.SelectionSet, selection.Name)
		return &ResponseField{
			Name:             selection.Name,
			Type:             r.NewNamedType(selection.Name, fieldsResponseFields),
			IsFragmentSpread: true,
			ResponseFields:   fieldsResponseFields,
		}

	case *ast.InlineFragment:
		// InlineFragmentは子要素をそのままstructとしてもつので、ここで、構造体の型を作成します
		fieldsResponseFields := r.NewResponseFields(selection.SelectionSet, "")
		fmt.Printf("inlineFragment name: %s\n", selection.TypeCondition)
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

func (r *SourceGenerator) OperationArguments(variableDefinitions ast.VariableDefinitionList) []*OperationArgument {
	argumentTypes := make([]*OperationArgument, 0, len(variableDefinitions))
	for _, v := range variableDefinitions {
		argumentTypes = append(argumentTypes, &OperationArgument{
			Variable: v.Variable,
			Type:     r.binder.CopyModifiersFromAst(v.Type, r.FindType(v.Type.Name())),
		})
	}

	return argumentTypes
}

// NewNamedType は、GraphQLに対応する存在型がなく、gqlgenc独自の型を作成する。
// コード生成するために作成時にgeneratedTypesに保存しておき、Templateに渡す。
func (r *SourceGenerator) NewNamedType(typeName string, fieldsResponseFields ResponseFieldList) *types.Named {
	structType := fieldsResponseFields.ToGoStructType()
	namedType := types.NewNamed(types.NewTypeName(0, r.cfg.GQLGencConfig.QueryGen.Pkg(), typeName, nil), structType, nil)
	return namedType
}

// Typeの引数に渡すtypeNameは解析した結果からselectionなどから求めた型の名前を渡さなければいけない
func (r *SourceGenerator) FindType(typeName string) types.Type {
	goType, err := r.binder.FindTypeFromName(r.cfg.GQLGenConfig.Models[typeName].Model[0])
	if err != nil {
		// 実装として正しいtypeNameを渡していれば必ず見つかるはずなのでpanic
		panic(fmt.Sprintf("%+v", err))
	}

	return goType
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
	vars := make([]*types.Var, 0)
	structTags := make([]string, 0)
	// 重複するフィールドはUniqueByNameによりGoの型から除外する。
	for _, responseField := range rs.uniqueByName() {
		vars = append(vars, types.NewField(0, nil, templates.ToGo(responseField.Name), responseField.Type, responseField.IsFragmentSpread))
		structTags = append(structTags, strings.Join(responseField.Tags, " "))
	}
	slices.SortFunc(vars, func(a, b *types.Var) int {
		return strings.Compare(a.Name(), b.Name())
	})
	return types.NewStruct(vars, structTags)
}

func (rs ResponseFieldList) uniqueByName() ResponseFieldList {
	responseFieldMapByName := make(map[string]*ResponseField, len(rs))
	for _, field := range rs {
		responseFieldMapByName[field.Name] = field
	}
	return slices.Collect(maps.Values(responseFieldMapByName))
}

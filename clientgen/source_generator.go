package clientgen

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/vektah/gqlparser/v2/ast"
	"golang.org/x/xerrors"
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

func (rs ResponseFieldList) StructType() *types.Struct {
	vars := make([]*types.Var, 0)
	structTags := make([]string, 0)
	for _, filed := range rs {
		//  クエリーのフィールドの子階層がFragmentの場合、このフィールドにそのFragmentの型を追加する
		if filed.IsFragmentSpread {
			typ := filed.ResponseFields.StructType().Underlying().(*types.Struct)
			for j := 0; j < typ.NumFields(); j++ {
				vars = append(vars, typ.Field(j))
				structTags = append(structTags, typ.Tag(j))
			}
		} else {
			vars = append(vars, types.NewVar(0, nil, templates.ToGo(filed.Name), filed.Type))
			structTags = append(structTags, strings.Join(filed.Tags, " "))
		}
	}

	return types.NewStruct(vars, structTags)
}

func (rs ResponseFieldList) IsFragment() bool {
	if len(rs) != 1 {
		return false
	}

	return rs[0].IsInlineFragment || rs[0].IsFragmentSpread
}

func (rs ResponseFieldList) IsBasicType() bool {
	return len(rs) == 0
}

func (rs ResponseFieldList) IsStructType() bool {
	return len(rs) > 0 && !rs.IsFragment()
}

type SourceGenerator struct {
	cfg    *config.Config
	binder *config.Binder
	client config.PackageConfig
}

func NewSourceGenerator(cfg *config.Config, client config.PackageConfig) *SourceGenerator {
	return &SourceGenerator{
		cfg:    cfg,
		binder: cfg.NewBinder(),
		client: client,
	}
}

func (r *SourceGenerator) NewResponseFields(selectionSet ast.SelectionSet, parentName string) ResponseFieldList {
	responseFields := make(ResponseFieldList, 0, len(selectionSet))
	for _, selection := range selectionSet {
		responseFields = append(responseFields, r.NewResponseField(selection, parentName))
	}

	return responseFields
}

func (r *SourceGenerator) NewResponseFieldsByDefinition(definition *ast.Definition) (ResponseFieldList, error) {
	fields := make(ResponseFieldList, 0, len(definition.Fields))
	for _, field := range definition.Fields {
		if field.Type.Name() == "__Schema" || field.Type.Name() == "__Type" {
			continue
		}

		var typ types.Type
		if field.Type.Name() == "Query" || field.Type.Name() == "Mutation" {
			var baseType types.Type
			baseType, err := r.binder.FindType(r.client.Pkg().Path(), field.Type.Name())
			if err != nil {
				if !strings.Contains(err.Error(), "unable to find type") {
					return nil, xerrors.Errorf("not found type: %w", err)
				}

				// create new type
				baseType = types.NewPointer(types.NewNamed(
					types.NewTypeName(0, r.client.Pkg(), templates.ToGo(field.Type.Name()), nil),
					nil,
					nil,
				))
			}

			// for recursive struct field in go
			typ = types.NewPointer(baseType)
		} else {
			baseType, err := r.binder.FindTypeFromName(r.cfg.Models[field.Type.Name()].Model[0])
			if err != nil {
				return nil, xerrors.Errorf("not found type: %w", err)
			}

			typ = r.binder.CopyModifiersFromAst(field.Type, baseType)
		}

		tags := []string{
			fmt.Sprintf(`json:"%s"`, field.Name),
			fmt.Sprintf(`graphql:"%s"`, field.Name),
		}

		fields = append(fields, &ResponseField{
			Name: field.Name,
			Type: typ,
			Tags: tags,
		})
	}

	return fields, nil
}

func (r *SourceGenerator) NewResponseField(selection ast.Selection, parentName string) *ResponseField {
	switch selection := selection.(type) {
	case *ast.Field:
		fieldsResponseFields := r.NewResponseFields(selection.SelectionSet, selection.Alias)

		var baseType types.Type
		switch {
		case fieldsResponseFields.IsBasicType():
			baseType = r.Type(selection.Definition.Type.Name())
		case fieldsResponseFields.IsFragment():
			// 子フィールドがFragmentの場合はこのFragmentがフィールドの型になる
			// if a child field is fragment, this field type became fragment.
			baseType = fieldsResponseFields[0].Type
		case fieldsResponseFields.IsStructType():
			baseType = types.NewNamed(
				types.NewTypeName(0, r.client.Pkg(), parentName+templates.ToGo(selection.Alias), nil),
				fieldsResponseFields.StructType(),
				nil,
			)
			fmt.Printf("+++ %s  parentName: %s\n", baseType.String(), parentName)
			// baseType = fieldsResponseFields.StructType()
		default:
			// ここにきたらバグ
			// here is bug
			panic("not match type")
		}

		// GraphQLの定義がオプショナルのはtypeのポインタ型が返り、配列の定義場合はポインタのスライスの型になって返ってきます
		// return pointer type then optional type or slice pointer then slice type of definition in GraphQL.
		typ := r.binder.CopyModifiersFromAst(selection.Definition.Type, baseType)

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
		// この構造体はテンプレート側で使われることはなく、ast.FieldでFragment判定するために使用する
		fieldsResponseFields := r.NewResponseFields(selection.Definition.SelectionSet, "")
		typ := types.NewNamed(
			types.NewTypeName(0, r.client.Pkg(), templates.ToGo(selection.Name), nil),
			fieldsResponseFields.StructType(),
			nil,
		)

		return &ResponseField{
			Name:             selection.Name,
			Type:             typ,
			IsFragmentSpread: true,
			ResponseFields:   fieldsResponseFields,
		}

	case *ast.InlineFragment:
		// InlineFragmentは子要素をそのままstructとしてもつので、ここで、構造体の型を作成します
		fieldsResponseFields := r.NewResponseFields(selection.SelectionSet, "")

		return &ResponseField{
			Name:             selection.TypeCondition,
			Type:             fieldsResponseFields.StructType(),
			IsInlineFragment: true,
			Tags:             []string{fmt.Sprintf(`graphql:"... on %s"`, selection.TypeCondition)},
			ResponseFields:   fieldsResponseFields,
		}
	}

	panic("unexpected selection type")
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
	goType, err := r.binder.FindTypeFromName(r.cfg.Models[typeName].Model[0])
	if err != nil {
		// 実装として正しいtypeNameを渡していれば必ず見つかるはずなのでpanic
		panic(fmt.Sprintf("%+v", err))
	}

	return goType
}

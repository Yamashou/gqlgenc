package clientgenv2

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
	Interface        *ResponseInterface
}

type ImplementResponseInterfaceType struct {
	Name            string
	BaseTypeName    string
	Type            types.Type
	ImplementsFuncs []*ImplementsFunc
}

type ImplementsFunc struct {
	Name             string
	ResponseTypeName string
	ResponseType     types.Type
	IsMyType         bool
}

type ResponseInterface struct {
	Name                   string
	Type                   types.Type
	InterfaceResponseTypes []*ImplementResponseInterfaceType
}

func (r *SourceGenerator) NewResponseInterface(name string, fieldsResponseFields ResponseFieldList, funcs []*ImplementsFunc, interfaceResponseTypes []*ImplementResponseInterfaceType) *ResponseInterface {
	implementsFuncs := make([]*types.Func, 0)
	for _, field := range funcs {
		t := types.NewPointer(types.NewNamed(
			types.NewTypeName(0, r.client.Pkg(), templates.ToGo(field.ResponseTypeName), nil),
			fieldsResponseFields.StructType(),
			nil,
		))

		implementsFuncs = append(implementsFuncs,
			types.NewFunc(0, r.client.Pkg(), field.Name, types.NewSignature(nil, nil, types.NewTuple(types.NewVar(0, nil, "", t)), false)))
	}

	return &ResponseInterface{
		Name:                   fmt.Sprintf("%sRandStr", name),
		Type:                   types.NewInterfaceType(implementsFuncs, nil),
		InterfaceResponseTypes: interfaceResponseTypes,
	}
}

func NewImplementsFuncs(fieldsResponseFields ResponseFieldList) []*ImplementsFunc {
	funcs := make([]*ImplementsFunc, 0)
	for _, field := range fieldsResponseFields {
		if len(field.ResponseFields) == 0 {
			continue
		}

		funcs = append(funcs, &ImplementsFunc{
			Name:         field.Name,
			ResponseType: nil,
			IsMyType:     false,
		})
	}

	return funcs
}

type ResponseFieldList []*ResponseField

func (rs ResponseFieldList) StructType() *types.Struct {
	vars := make([]*types.Var, 0)
	structTags := make([]string, 0)
	for _, filed := range rs {
		if filed.IsFragmentSpread {
			// TODO ここを分けてstructを定義する必要がある
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

	// TODO ユニーク処理を別関数に分ける
	varsMap := make(map[string]struct{})
	uniqueVars := make([]*types.Var, 0)
	uniqueTags := make([]string, 0)
	for i, v := range vars {
		_, ok := varsMap[v.Name()]
		if !ok {
			varsMap[v.Name()] = struct{}{}
			uniqueVars = append(uniqueVars, v)
			uniqueTags = append(uniqueTags, structTags[i])
		}
	}

	return types.NewStruct(uniqueVars, uniqueTags)
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
	cfg        *config.Config
	binder     *config.Binder
	client     config.PackageConfig
	interfaces []*ResponseInterface
}

func NewSourceGenerator(cfg *config.Config, client config.PackageConfig) *SourceGenerator {
	return &SourceGenerator{
		cfg:        cfg,
		binder:     cfg.NewBinder(),
		client:     client,
		interfaces: make([]*ResponseInterface, 0),
	}
}

func (r *SourceGenerator) NewResponseFields(selectionSet ast.SelectionSet) ResponseFieldList {
	responseFields := make(ResponseFieldList, 0, len(selectionSet))
	for _, selection := range selectionSet {
		responseFields = append(responseFields, r.NewResponseField(selection))
	}

	return responseFields
}

func (r *SourceGenerator) NewResponseFieldsByDefinition(definition *ast.Definition) (ResponseFieldList, error) {
	fields := make(ResponseFieldList, 0, len(definition.Fields))
	for _, field := range definition.Fields {
		if field.Type.Name() == "__Schema" || field.Type.Name() == "__Type" {
			continue
		}

		responseField, err := r.NewResponseFieldByFieldDefinition(field)
		if err != nil {
			return nil, xerrors.Errorf(": %w", err)
		}

		fields = append(fields, responseField)
	}

	return fields, nil
}

func (r *SourceGenerator) NewResponseFieldByFieldDefinition(field *ast.FieldDefinition) (*ResponseField, error) {
	typ, err := r.NewResponseFieldType(field)
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	return &ResponseField{
		Name: field.Name,
		Type: typ,
		Tags: NewResponseFieldTags(field),
	}, nil
}

func (r *SourceGenerator) NewResponseFieldType(field *ast.FieldDefinition) (types.Type, error) {
	var typ types.Type
	if field.Type.Name() == "Query" || field.Type.Name() == "Mutation" {
		var baseType types.Type
		baseType, err := r.binder.FindType(r.client.Pkg().Path(), field.Type.Name())
		if err != nil {
			if !strings.Contains(err.Error(), "unable to find type") {
				return nil, xerrors.Errorf("not found type: %w", err)
			}

			// create new type
			baseType = types.NewNamed(
				types.NewTypeName(0, r.client.Pkg(), templates.ToGo(field.Type.Name()), nil),
				nil,
				nil,
			)
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

	return typ, nil
}

func NewResponseFieldTags(field *ast.FieldDefinition) []string {
	tags := []string{
		fmt.Sprintf(`json:"%s"`, field.Name),
		fmt.Sprintf(`graphql:"%s"`, field.Name),
	}

	return tags
}

func (r *SourceGenerator) NewResponseField(selection ast.Selection) *ResponseField {
	switch selection := selection.(type) {
	case *ast.Field:
		fieldsResponseFields := r.NewResponseFields(selection.SelectionSet)
		sonTyps := make([]*ImplementResponseInterfaceType, 0)
		var resInterface *ResponseInterface
		if _, ok := r.Type(selection.Definition.Type.Name()).Underlying().(*types.Interface); ok {
			for _, typ := range fieldsResponseFields {
				if len(typ.ResponseFields) > 0 {
					sonTyps = append(sonTyps, &ImplementResponseInterfaceType{
						Name:         fmt.Sprintf("%s_%s", typ.Name, "rond_str"),
						BaseTypeName: typ.Name,
						Type:         typ.Type,
					})
				}
			}
			implementsFuncs := NewImplementsFuncs(fieldsResponseFields)
			for _, t := range sonTyps {
				for _, it := range implementsFuncs {
					if t.BaseTypeName == it.Name {
						it.ResponseType = t.Type
						it.ResponseTypeName = t.Name
					}
				}
			}

			for _, t := range sonTyps {
				t.ImplementsFuncs = implementsFuncs
			}
			resInterface = r.NewResponseInterface(selection.Definition.Type.Name(), fieldsResponseFields, implementsFuncs, sonTyps)
			r.interfaces = append(r.interfaces, resInterface)
		}

		var baseType types.Type
		switch {
		case fieldsResponseFields.IsBasicType():
			baseType = r.Type(selection.Definition.Type.Name())
		case fieldsResponseFields.IsFragment():
			// 子フィールドがFragmentの場合はこのFragmentがフィールドの型になる
			// if a child field is fragment, this field type became fragment.
			baseType = fieldsResponseFields[0].Type
		case fieldsResponseFields.IsStructType():
			// TODO ここの処理をbaseTypeをstructではなくて、別のstructとして定義して、そのstructの名前を入れるようにする
			baseType = fieldsResponseFields.StructType()
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

		if resInterface != nil {
			t := types.NewNamed(
				types.NewTypeName(0, r.client.Pkg(), templates.ToGo(resInterface.Name), nil),
				fieldsResponseFields.StructType(),
				nil,
			)
			return &ResponseField{
				Name:           selection.Alias,
				Type:           t,
				Tags:           tags,
				ResponseFields: fieldsResponseFields,
				Interface:      resInterface,
			}
		}

		return &ResponseField{
			Name:           selection.Alias,
			Type:           typ,
			Tags:           tags,
			ResponseFields: fieldsResponseFields,
			Interface:      resInterface,
		}

	case *ast.FragmentSpread:
		// この構造体はテンプレート側で使われることはなく、ast.FieldでFragment判定するために使用する
		fieldsResponseFields := r.NewResponseFields(selection.Definition.SelectionSet)
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
		// TODO TypeConditionごとにStructを作成するのと、そのTypeを返す関数を用意してそれをInterfaceに持たせる
		fieldsResponseFields := r.NewResponseFields(selection.SelectionSet)
		typ := fieldsResponseFields.StructType()

		return &ResponseField{
			Name:             selection.TypeCondition,
			Type:             typ,
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

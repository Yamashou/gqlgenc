package source

import (
	"go/types"
	"maps"
	"slices"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"
)

// TODO: シンプルなtypes.Typeに置き換えられないか？
type Field struct {
	Name             string
	Type             types.Type
	Tags             []string
	IsFragmentSpread bool
	IsInlineFragment bool
}

func NewField(name string, fieldType types.Type, tags []string, isFragmentSpread bool, isInlineFragment bool) *Field {
	return &Field{
		Name:             name,
		Type:             fieldType,
		Tags:             tags,
		IsFragmentSpread: isFragmentSpread,
		IsInlineFragment: isInlineFragment,
	}
}

func (r *Field) goVar() *types.Var {
	return types.NewField(0, nil, templates.ToGo(r.Name), r.Type, r.IsFragmentSpread)
}

func (r *Field) joinTags() string {
	return strings.Join(r.Tags, " ")
}

type GoType struct {
	Name             string
	NonNull          bool
	isFragmentSpread bool
	isInlineFragment bool
	isBasicType      bool
	goType           *types.Struct
}

func NewGoType(name string, nonNull, isFragmentSpread, isInlineFragment bool, isBasicType bool, goType *types.Struct) *GoType {
	return &GoType{
		Name:             name,
		NonNull:          nonNull,
		isFragmentSpread: isFragmentSpread,
		isInlineFragment: isInlineFragment,
		isBasicType:      isBasicType,
		goType:           goType,
	}
}

func NewGoTypeByFields(name string, nonNull bool, fields Fields) *GoType {
	return NewGoType(name, nonNull, fields.isFragmentSpread(), fields.isInlineFragment(), fields.isBasicType(), fields.goTypeType())
}

type Fields []*Field

func (fs Fields) isFragmentSpread() bool {
	if len(fs) != 1 {
		return false
	}

	return fs[0].IsFragmentSpread
}

func (fs Fields) isInlineFragment() bool {
	if len(fs) != 1 {
		return false
	}

	return fs[0].IsInlineFragment
}

func (fs Fields) isBasicType() bool {
	return len(fs) == 0
}

func (fs Fields) goTypeType() *types.Struct {
	// Go fields do not allow fields with the same name, so we remove duplicates
	fields := fs.uniqueByName()
	vars := make([]*types.Var, 0, len(fields))
	for _, field := range fields {
		vars = append(vars, field.goVar())
	}
	tags := make([]string, 0, len(fields))
	for _, field := range fields {
		tags = append(tags, field.joinTags())
	}
	return types.NewStruct(vars, tags)
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

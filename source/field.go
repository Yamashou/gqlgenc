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
	Fields           Fields
	IsFragmentSpread bool
	IsInlineFragment bool
}

func NewField(name string, fieldType types.Type, tags []string, fields Fields, isFragmentSpread bool, isInlineFragment bool) *Field {
	return &Field{
		Name:             name,
		Type:             fieldType,
		Tags:             tags,
		Fields:           fields,
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

type GoStruct struct {
	isFragmentSpread bool
	isBasicType      bool
	goType           *types.Struct
}

func NewGoStruct(isFragmentSpread bool, isBasicType bool, goType *types.Struct) *GoStruct {
	return &GoStruct{
		isFragmentSpread: isFragmentSpread,
		isBasicType:      isBasicType,
		goType:           goType,
	}
}

func NewGoStructByFields(fields Fields) *GoStruct {
	return NewGoStruct(fields.isFragmentSpread(), fields.isBasicType(), fields.goStructType())
}

type Fields []*Field

func (fs Fields) isFragmentSpread() bool {
	if len(fs) != 1 {
		return false
	}

	return fs[0].IsFragmentSpread
}

func (fs Fields) isBasicType() bool {
	return len(fs) == 0
}

func (fs Fields) goStructType() *types.Struct {
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

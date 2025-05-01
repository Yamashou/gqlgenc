package source

import (
	"go/types"
	"maps"
	"slices"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"
)

type FieldKind string

const (
	BasicType      FieldKind = "BasicType"
	FragmentSpread FieldKind = "FragmentSpread"
	InlineFragment FieldKind = "InlineFragment"
	OtherType      FieldKind = "OtherType"
)

// TODO: シンプルなtypes.Typeに置き換えられないか？
type Field struct {
	Name      string
	Type      types.Type
	Tags      []string
	FieldKind FieldKind
}

func NewField(name string, fieldType types.Type, tags []string, fieldKind FieldKind) *Field {
	return &Field{
		Name:      name,
		Type:      fieldType,
		Tags:      tags,
		FieldKind: fieldKind,
	}
}

func (r *Field) goVar() *types.Var {
	return types.NewField(0, nil, templates.ToGo(r.Name), r.Type, r.FieldKind == FragmentSpread)
}

func (r *Field) joinTags() string {
	return strings.Join(r.Tags, " ")
}

type GoType struct {
	Name    string
	NonNull bool
	goType  *types.Struct
}

func NewGoType(name string, nonNull bool, goType *types.Struct) *GoType {
	return &GoType{
		Name:    name,
		NonNull: nonNull,
		goType:  goType,
	}
}

func NewGoTypeByFields(name string, nonNull bool, fields Fields) *GoType {
	return NewGoType(name, nonNull, fields.goStructType())
}

type Fields []*Field

func (fs Fields) FieldKind() FieldKind {
	if len(fs) == 0 {
		return BasicType
	}
	return OtherType
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

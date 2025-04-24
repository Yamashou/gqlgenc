package gotype

import (
	"bytes"
	"fmt"
	"go/types"
	"strings"
)

type GenGettersGenerator struct {
	queryGenPackageName string
}

func NewGenGettersGenerator(queryGenPackageName string) *GenGettersGenerator {
	return &GenGettersGenerator{
		queryGenPackageName: queryGenPackageName,
	}
}

func (g *GenGettersGenerator) GenFunc() func(name string, p types.Type) string {
	// This method returns a string of getters for a struct.
	// The idea is to be able to chain calls safely without having to check for nil.
	// To make this work we need to return a pointer to the struct if the field is a struct.
	return func(name string, p types.Type) string {
		var it *types.Struct
		it, ok := p.(*types.Struct)
		if !ok {
			return ""
		}
		var buf bytes.Buffer

		for i := range it.NumFields() {
			field := it.Field(i)

			returns := returnTypeName(field.Type(), g.queryGenPackageName, false)

			buf.WriteString("func (t *" + name + ") Get" + field.Name() + "() " + returns + "{\n")
			buf.WriteString("if t == nil {\n t = &" + name + "{}\n}\n")

			pointerOrNot := ""
			if _, ok := field.Type().(*types.Named); ok {
				pointerOrNot = "&"
			}

			buf.WriteString("return " + pointerOrNot + "t." + field.Name() + "\n}\n")
		}

		return buf.String()
	}
}

func returnTypeName(t types.Type, clientPackageName string, nested bool) string {
	switch it := t.(type) {
	case *types.Basic:
		return it.String()
	case *types.Pointer:
		return "*" + returnTypeName(it.Elem(), clientPackageName, true)
	case *types.Slice:
		return "[]" + returnTypeName(it.Elem(), clientPackageName, true)
	case *types.Named:
		s := strings.Split(it.String(), ".")
		name := s[len(s)-1]

		isImported := it.Obj().Parent() != nil && it.Obj().Pkg().Name() != clientPackageName
		if isImported {
			name = namedTypeString(it)
		}

		if nested {
			return name
		}

		return "*" + name
	case *types.Interface:
		return "any"
	case *types.Map:
		return "map[" + returnTypeName(it.Key(), clientPackageName, true) + "]" + returnTypeName(it.Elem(), clientPackageName, true)
	case *types.Alias:
		return returnTypeName(it.Underlying(), clientPackageName, nested)
	default:
		return fmt.Sprintf("%T----", it)
	}
}

func namedTypeString(named *types.Named) string {
	// オブジェクトからパッケージ情報を取得
	pkg := named.Obj().Pkg()

	// パッケージ情報がない場合、型名のみを返す
	if pkg == nil {
		return named.Obj().Name()
	}

	// パッケージ名と型名を結合して返す
	return fmt.Sprintf("%s.%s", pkg.Name(), named.Obj().Name())
}

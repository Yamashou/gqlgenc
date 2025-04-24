package gotype

import (
	"bytes"
	"fmt"
	"go/types"

	"github.com/vektah/gqlparser/v2/ast"
)

type Operation struct {
	Name                string
	VariableDefinitions ast.VariableDefinitionList
	Operation           string
	Args                []*Argument
	OperationResponse   *OperationResponse
}

func NewOperation(operation *ast.OperationDefinition, queryDocument *ast.QueryDocument, args []*Argument, operationResponse *OperationResponse) *Operation {
	return &Operation{
		Name:                operation.Name,
		VariableDefinitions: operation.VariableDefinitions,
		Operation:           queryString(queryDocument),
		Args:                args,
		OperationResponse:   operationResponse,
	}
}

type Argument struct {
	Variable string
	Type     types.Type
}

type OperationResponse struct {
	Name string
	Type types.Type
}
type StructSource struct {
	Name string
	Type types.Type
}

func NewStructSource(name string, typ types.Type) *StructSource {
	return &StructSource{
		Name: name,
		Type: typ,
	}
}

// GetterFunc returns a function that generates getter methods for types.
// targetPkgPath specifies the target package path and omits package qualifiers for types belonging to the same package.
func GetterFunc(targetPkgPath string) func(name string, t types.Type) string {
	return func(name string, t types.Type) string {
		st, ok := t.(*types.Struct)
		if !ok {
			return ""
		}

		var buf bytes.Buffer
		for i := 0; i < st.NumFields(); i++ {
			field := st.Field(i)
			fieldName := field.Name()
			returnType := funcReturnTypesName(field.Type(), true, targetPkgPath)

			fmt.Fprintf(&buf, "func (t *%s) Get%s() %s {\n", name, fieldName, returnType)
			fmt.Fprintf(&buf, "\tif t == nil {\n\t\tt = &%s{}\n\t}\n", name)

			needsPointer := isNamedType(field.Type())
			if needsPointer {
				fmt.Fprintf(&buf, "\treturn &t.%s\n}\n", fieldName)
			} else {
				fmt.Fprintf(&buf, "\treturn t.%s\n}\n", fieldName)
			}
		}

		return buf.String()
	}
}

// isNamedType determines if the type is a named type (such as a struct)
func isNamedType(t types.Type) bool {
	_, ok := t.(*types.Named)
	return ok
}

// A new closure function that can use target package information
func funcReturnTypesName(t types.Type, isStruct bool, targetPkgPath string) string {
	switch it := t.(type) {
	case *types.Basic:
		return it.String()
	case *types.Pointer:
		return "*" + funcReturnTypesName(it.Elem(), false, targetPkgPath)
	case *types.Slice:
		return "[]" + funcReturnTypesName(it.Elem(), false, targetPkgPath)
	case *types.Named:
		// For named types, consider the package path to determine whether to include a package qualifier
		name := namedTypeString(it, targetPkgPath)

		if isStruct {
			return "*" + name
		}

		return name
	case *types.Interface:
		return "any"
	case *types.Map:
		return "map[" + funcReturnTypesName(it.Key(), false, targetPkgPath) + "]" + funcReturnTypesName(it.Elem(), false, targetPkgPath)
	case *types.Alias:
		return funcReturnTypesName(it.Underlying(), isStruct, targetPkgPath)
	default:
		return fmt.Sprintf("unknown(%T)", it)
	}
}

// namedTypeString returns the fully qualified name of a type.
// If currentPkgPath is specified and the type belongs to the same package, the package qualifier is omitted.
func namedTypeString(named *types.Named, currentPkgPath string) string {
	// Get package information from the object
	pkg := named.Obj().Pkg()

	// If there is no package information, return only the type name
	if pkg == nil {
		return named.Obj().Name()
	}

	// If in the same package as the current package, omit the package qualifier
	if currentPkgPath != "" && pkg.Path() == currentPkgPath {
		return named.Obj().Name()
	}

	// Return the combined package name and type name
	return fmt.Sprintf("%s.%s", pkg.Name(), named.Obj().Name())
}

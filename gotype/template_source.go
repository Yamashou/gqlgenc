package gotype

import (
	"bytes"
	"fmt"
	"github.com/99designs/gqlgen/codegen/templates"
	"go/types"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

type Operation struct {
	Name                string
	VariableDefinitions ast.VariableDefinitionList
	Operation           string
	Args                []*OperationArgument
}

func NewOperation(operation *ast.OperationDefinition, queryDocument *ast.QueryDocument, args []*OperationArgument) *Operation {
	return &Operation{
		Name:                operation.Name,
		VariableDefinitions: operation.VariableDefinitions,
		Operation:           queryString(queryDocument),
		Args:                args,
	}
}

type OperationArgument struct {
	Variable string
	Type     types.Type
}

func NewOperationArgument(variable string, t types.Type) *OperationArgument {
	return &OperationArgument{
		Variable: variable,
		Type:     t,
	}
}

type OperationResponse struct {
	Name string
	Type types.Type
}

func NewOperationResponse(name string, t types.Type) *OperationResponse {
	return &OperationResponse{
		Name: name,
		Type: t,
	}

}

type QueryType struct {
	Name string
	Type types.Type
}

func NewQueryType(name string, typ types.Type) *QueryType {
	return &QueryType{
		Name: name,
		Type: typ,
	}
}

type Fragment struct {
	Name string
	Type types.Type
}

func NewFragment(name string, typ types.Type) *Fragment {
	return &Fragment{
		Name: name,
		Type: typ,
	}
}

// GetterFunc returns a function that generates getter methods for types.
// targetPkgPath specifies the target package path and omits package qualifiers for types belonging to the same package.
func GetterFunc(targetPkgPath string) func(types.Type) string {
	return func(t types.Type) string {
		var namedType *types.Named
		pointerType, ok := t.(*types.Pointer)
		if ok {
			namedType = pointerType.Elem().(*types.Named)
		} else {
			// FragmentのときはPointerではない
			namedType = t.(*types.Named)
		}
		st := namedType.Underlying().(*types.Struct)

		// typeName
		typeName := shortTypeName(namedType)

		var buf bytes.Buffer
		fmt.Fprintf(&buf, "type %s %s\n", typeName, ref(st))
		for i := 0; i < st.NumFields(); i++ {
			field := st.Field(i)
			fieldName := field.Name()
			if fieldName == "" {
				// fieldが埋め込みの時は、Getterは不要
				continue
			}

			fieldTypeName := funcReturnTypesName(field.Type(), targetPkgPath)
			fmt.Fprintf(&buf, "func (t *%s) Get%s() %s {\n", typeName, fieldName, fieldTypeName)
			fmt.Fprintf(&buf, "\tif t == nil {\n\t\tt = &%s{}\n\t}\n", typeName)

			fmt.Fprintf(&buf, "\treturn t.%s\n}\n", fieldName)
		}

		return buf.String()
	}
}

func ref(p types.Type) string {
	typeString := templates.CurrentImports.LookupType(p)
	// TODO(steve): figure out why this is needed
	// otherwise inconsistent sometimes
	// see https://github.com/99designs/gqlgen/issues/3414#issuecomment-2822856422
	if typeString == "interface{}" {
		return "any"
	}
	if typeString == "map[string]interface{}" {
		return "map[string]any"
	}
	return typeString
}

func shortTypeName(t types.Type) string {
	names := strings.Split(t.String(), ".")
	typeName := names[len(names)-1]
	if strings.HasPrefix(t.String(), "*") {
		typeName = "*" + typeName
	}
	return typeName
}

// A new closure function that can use target package information
func funcReturnTypesName(t types.Type, targetPkgPath string) string {
	switch it := t.(type) {
	case *types.Basic:
		return it.String()
	case *types.Pointer:
		return "*" + funcReturnTypesName(it.Elem(), targetPkgPath)
	case *types.Slice:
		return "[]" + funcReturnTypesName(it.Elem(), targetPkgPath)
	case *types.Named:
		// For named types, consider the package path to determine whether to include a package qualifier
		return namedTypeString(it, targetPkgPath)
	case *types.Interface:
		return "any"
	case *types.Map:
		return "map[" + funcReturnTypesName(it.Key(), targetPkgPath) + "]" + funcReturnTypesName(it.Elem(), targetPkgPath)
	case *types.Alias:
		return funcReturnTypesName(it.Underlying(), targetPkgPath)
	case *types.Struct:
		return it.String()
	default:
		return fmt.Sprintf("unknown(%T)", it)
	}
}

// namedTypeString returns the fully qualified name of a type.
// If currentPkgPath is specified and the type belongs to the same package, the package qualifier is omitted.
// TODO: refとの違いは？
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

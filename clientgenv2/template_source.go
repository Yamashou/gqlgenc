package clientgenv2

import (
	"bytes"
	"fmt"
	"github.com/99designs/gqlgen/codegen/templates"
	"go/types"
)

// GetterFunc returns a function that generates getter methods for types.
// targetPkgPath specifies the target package path and omits package qualifiers for types belonging to the same package.
func GetterFunc() func(types.Type) string {
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
		typeName := ref(namedType)

		var buf bytes.Buffer
		fmt.Fprintf(&buf, "type %s %s\n", typeName, ref(st))
		for i := 0; i < st.NumFields(); i++ {
			field := st.Field(i)
			fieldName := field.Name()
			if fieldName == "" {
				// fieldが埋め込みの時は、Getterは不要
				continue
			}

			fieldTypeName := ref(field.Type())
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

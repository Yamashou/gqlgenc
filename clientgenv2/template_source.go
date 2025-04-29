package clientgenv2

import (
	"bytes"
	"fmt"
	"github.com/99designs/gqlgen/codegen/templates"
	"go/types"
)

func TypeGenFunc(t types.Type) string {
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
	typeName := toString(namedType)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "type %s %s\n", typeName, toString(st))
	for i := range st.NumFields() {
		field := st.Field(i)
		fieldName := field.Name()
		if fieldName == "" {
			// fieldが埋め込みの時は、Getterは不要
			continue
		}

		fieldTypeName := toString(field.Type())
		fmt.Fprintf(&buf, "func (t *%s) Get%s() %s {\n", typeName, fieldName, fieldTypeName)
		fmt.Fprintf(&buf, "\tif t == nil {\n\t\tt = &%s{}\n\t}\n", typeName)

		fmt.Fprintf(&buf, "\treturn t.%s\n}\n", fieldName)
	}

	return buf.String()
}

func toString(p types.Type) string {
	return templates.CurrentImports.LookupType(p)
}

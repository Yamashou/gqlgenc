package clientgen

import (
	"go/types"
	"testing"
)

// TestReturnTypeName tests the returnTypeName function with various types.
func TestReturnTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    types.Type
		nested   bool
		expected string
	}{
		{
			name:     "Basic",
			input:    types.Typ[types.String],
			nested:   false,
			expected: "string",
		},
		{
			name:     "Pointer",
			input:    types.NewPointer(types.Typ[types.Int]),
			nested:   false,
			expected: "*int",
		},
		{
			name:     "Slice",
			input:    types.NewSlice(types.Typ[types.Float64]),
			nested:   false,
			expected: "[]float64",
		},
		{
			name:     "Named",
			input:    types.NewNamed(types.NewTypeName(0, nil, "MyType", nil), nil, nil),
			nested:   false,
			expected: "*MyType",
		},
		{
			name:     "Interface",
			input:    types.NewInterfaceType(nil, nil).Complete(),
			nested:   false,
			expected: "any",
		},
		{
			name:     "Map",
			input:    types.NewMap(types.Typ[types.Int], types.Typ[types.Bool]),
			nested:   false,
			expected: "map[int]bool",
		},
	}

	g := &GenGettersGenerator{
		ClientPackageName: "hoge",
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := g.returnTypeName(test.input, test.nested)
			if output != test.expected {
				t.Errorf("Expected %s, but got %s", test.expected, output)
			}
		})
	}
}

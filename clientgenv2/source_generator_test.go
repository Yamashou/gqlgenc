package clientgenv2

import (
	"github.com/99designs/gqlgen/codegen/config"
	gqlgencConfig "github.com/Yamashou/gqlgenc/config"
	"go/types"
	"testing"
)

func createTestStruct(fields []*types.Var, tags []string) *types.Struct {
	return types.NewStruct(fields, tags)
}

func TestMergeFieldsRecursively(t *testing.T) {
	tests := []struct {
		name               string
		targetFields       ResponseFieldList
		sourceFields       ResponseFieldList
		preMerged          []*StructSource
		postMerged         []*StructSource
		expectedFields     ResponseFieldList
		expectedPreMerged  []*StructSource
		expectedPostMerged []*StructSource
	}{
		{
			name: "Basic merge case",
			targetFields: ResponseFieldList{
				{
					Name: "field1",
					Type: types.Typ[types.String],
					Tags: []string{`json:"field1"`},
				},
			},
			sourceFields: ResponseFieldList{
				{
					Name: "field2",
					Type: types.Typ[types.Int],
					Tags: []string{`json:"field2"`},
				},
			},
			preMerged:  []*StructSource{},
			postMerged: []*StructSource{},
			expectedFields: ResponseFieldList{
				{
					Name: "field1",
					Type: types.Typ[types.String],
					Tags: []string{`json:"field1"`},
				},
				{
					Name: "field2",
					Type: types.Typ[types.Int],
					Tags: []string{`json:"field2"`},
				},
			},
			expectedPreMerged:  []*StructSource{},
			expectedPostMerged: []*StructSource{},
		},
		{
			name: "Merge case with complex query including fragments",
			targetFields: ResponseFieldList{
				{
					Name: "id",
					Type: types.Typ[types.String],
					Tags: []string{`json:"id"`},
				},
				{
					Name: "profile",
					Type: types.NewPointer(types.NewNamed(
						types.NewTypeName(0, nil, "A_User_Profile", nil),
						createTestStruct([]*types.Var{
							types.NewVar(0, nil, "id", types.Typ[types.String]),
							types.NewVar(0, nil, "name", types.Typ[types.String]),
						}, []string{`json:"id"`, `json:"name"`}),
						nil,
					)),
					Tags: []string{`json:"profile"`},
					ResponseFields: ResponseFieldList{
						{
							Name: "id",
							Type: types.Typ[types.String],
							Tags: []string{`json:"id"`},
						},
						{
							Name: "ProfileFragment",
							Type: types.NewPointer(types.NewNamed(
								types.NewTypeName(0, nil, "ProfileFragment", nil),
								createTestStruct([]*types.Var{
									types.NewVar(0, nil, "name", types.Typ[types.String]),
								}, []string{`json:"name"`}),
								nil,
							)),
							Tags:             []string{`json:"ProfileFragment"`},
							IsFragmentSpread: true,
							ResponseFields: ResponseFieldList{
								{
									Name: "name",
									Type: types.Typ[types.String],
									Tags: []string{`json:"name"`},
								},
							},
						},
					},
				},
			},
			sourceFields: ResponseFieldList{
				{
					Name: "profile",
					Type: types.NewPointer(types.NewNamed(
						types.NewTypeName(0, nil, "Profile", nil),
						createTestStruct([]*types.Var{
							types.NewVar(0, nil, "id", types.Typ[types.String]),
							types.NewVar(0, nil, "name", types.Typ[types.String]),
						}, []string{`json:"id"`, `json:"name"`}),
						nil,
					)),
					Tags: []string{`json:"profile"`},
					ResponseFields: ResponseFieldList{
						{
							Name: "name",
							Type: types.Typ[types.String],
							Tags: []string{`json:"name"`},
						},
					},
				},
			},
			preMerged:  []*StructSource{},
			postMerged: []*StructSource{},
			expectedFields: ResponseFieldList{
				{
					Name: "id",
					Type: types.Typ[types.String],
					Tags: []string{`json:"id"`},
				},
				{
					Name: "profile",
					Type: types.NewPointer(types.NewNamed(
						types.NewTypeName(0, nil, "A_User_Profile", nil),
						createTestStruct([]*types.Var{
							types.NewVar(0, nil, "id", types.Typ[types.String]),
							types.NewVar(0, nil, "name", types.Typ[types.String]),
						}, []string{`json:"id"`, `json:"name"`}),
						nil,
					)),
					Tags: []string{`json:"profile"`},
					ResponseFields: ResponseFieldList{
						{
							Name: "id",
							Type: types.Typ[types.String],
							Tags: []string{`json:"id"`},
						},
						{
							Name: "ProfileFragment",
							Type: types.NewPointer(types.NewNamed(
								types.NewTypeName(0, nil, "ProfileFragment", nil),
								createTestStruct([]*types.Var{
									types.NewVar(0, nil, "name", types.Typ[types.String]),
								}, []string{`json:"name"`}),
								nil,
							)),
							Tags:             []string{`json:"ProfileFragment"`},
							IsFragmentSpread: true,
							ResponseFields: ResponseFieldList{
								{
									Name: "name",
									Type: types.Typ[types.String],
									Tags: []string{`json:"name"`},
								},
							},
						},
					},
				},
			},
			expectedPreMerged: []*StructSource{
				{
					Name: "Nested",
					Type: createTestStruct([]*types.Var{
						types.NewVar(0, nil, "id", types.Typ[types.String]),
					}, []string{`json:"id"`}),
				},
				{
					Name: "Nested",
					Type: createTestStruct([]*types.Var{
						types.NewVar(0, nil, "name", types.Typ[types.String]),
					}, []string{`json:"name"`}),
				},
			},
			expectedPostMerged: []*StructSource{
				{
					Name: "Nested",
					Type: createTestStruct([]*types.Var{
						types.NewVar(0, nil, "id", types.Typ[types.String]),
						types.NewVar(0, nil, "name", types.Typ[types.String]),
					}, []string{`json:"id"`, `json:"name"`}),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultFields, resultPreMerged, resultPostMerged := mergeFieldsRecursively(
				tt.targetFields,
				tt.sourceFields,
				tt.preMerged,
				tt.postMerged,
			)

			if len(tt.expectedFields) != len(resultFields) {
				t.Errorf("Number of fields does not match: got %v, want %v", len(resultFields), len(tt.expectedFields))
				return
			}

			if len(tt.expectedPreMerged) != len(resultPreMerged) {
				t.Errorf("Number of preMerged does not match: got %v, want %v", len(resultPreMerged), len(tt.expectedPreMerged))
			}

			if len(tt.expectedPostMerged) != len(resultPostMerged) {
				t.Errorf("Number of postMerged does not match: got %v, want %v", len(resultPostMerged), len(tt.expectedPostMerged))
			}

			for i := range tt.expectedFields {
				if tt.expectedFields[i].Name != resultFields[i].Name {
					t.Errorf("Field name does not match: got %v, want %v", resultFields[i].Name, tt.expectedFields[i].Name)
				}
				if tt.expectedFields[i].Type.String() != resultFields[i].Type.String() {
					t.Errorf("Field type does not match: got %v, want %v", resultFields[i].Type.String(), tt.expectedFields[i].Type.String())
				}
				if len(tt.expectedFields[i].Tags) != len(resultFields[i].Tags) {
					t.Errorf("Number of tags does not match: got %v, want %v", len(resultFields[i].Tags), len(tt.expectedFields[i].Tags))
				}
				for j, tag := range tt.expectedFields[i].Tags {
					if tag != resultFields[i].Tags[j] {
						t.Errorf("Tag does not match: got %v, want %v", resultFields[i].Tags[j], tag)
					}
				}
			}
		})
	}
}

func TestBuiltInTypeSupport(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected types.Type
	}{
		{"String", &config.Config{
			Models: config.TypeMap{
				"String": config.TypeMapEntry{Model: []string{"string"}},
			},
		}, types.Typ[types.String]},
		{"Int", &config.Config{
			Models: config.TypeMap{
				"Int": config.TypeMapEntry{Model: []string{"int64"}},
			},
		}, types.Typ[types.Int]},
		{"Float", &config.Config{
			Models: config.TypeMap{
				"Float": config.TypeMapEntry{Model: []string{"float64"}},
			},
		}, types.Typ[types.Float64]},
		{"Boolean", &config.Config{
			Models: config.TypeMap{
				"Boolean": config.TypeMapEntry{Model: []string{"bool"}},
			},
		}, types.Typ[types.Bool]},
		{"HTML", &config.Config{
			Models: config.TypeMap{
				"HTML": config.TypeMapEntry{Model: []string{"string"}},
			},
		}, types.Typ[types.String]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceGenerator := NewSourceGenerator(tt.config, config.PackageConfig{}, &gqlgencConfig.GenerateConfig{})
			goType := sourceGenerator.Type(tt.name)
			
			if tt.expected != goType {
				t.Errorf("Expected %s, got %s", tt.expected, goType.String())
			}
		})
	}
}

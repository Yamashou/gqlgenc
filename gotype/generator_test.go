package gotype

import (
	"go/types"
	"testing"
)

func createTestStruct(fields []*types.Var, tags []string) *types.Struct {
	return types.NewStruct(fields, tags)
}

func TestMergeFieldsRecursively(t *testing.T) {
	tests := []struct {
		name               string
		targetFields       Fields
		sourceFields       Fields
		preMerged          []*QueryType
		postMerged         []*QueryType
		expectedFields     Fields
		expectedPreMerged  []*QueryType
		expectedPostMerged []*QueryType
	}{
		{
			name: "Basic merge case",
			targetFields: Fields{
				{
					Name: "field1",
					Type: types.Typ[types.String],
					Tags: []string{`json:"field1"`},
				},
			},
			sourceFields: Fields{
				{
					Name: "field2",
					Type: types.Typ[types.Int],
					Tags: []string{`json:"field2"`},
				},
			},
			preMerged:  []*QueryType{},
			postMerged: []*QueryType{},
			expectedFields: Fields{
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
			expectedPreMerged:  []*QueryType{},
			expectedPostMerged: []*QueryType{},
		},
		{
			name: "Merge case with complex query including fragments",
			targetFields: Fields{
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
					ResponseFields: Fields{
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
							ResponseFields: Fields{
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
			sourceFields: Fields{
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
					ResponseFields: Fields{
						{
							Name: "name",
							Type: types.Typ[types.String],
							Tags: []string{`json:"name"`},
						},
					},
				},
			},
			preMerged:  []*QueryType{},
			postMerged: []*QueryType{},
			expectedFields: Fields{
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
					ResponseFields: Fields{
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
							ResponseFields: Fields{
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
			expectedPreMerged: []*QueryType{
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
			expectedPostMerged: []*QueryType{
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

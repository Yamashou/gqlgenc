package gotype

import (
	"fmt"
	"go/types"
	"maps"
	"slices"
	"strings"

	gqlgenconfig "github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/Yamashou/gqlgenc/v3/config"
	"github.com/vektah/gqlparser/v2/ast"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Generator generate Go goType from GraphQL goType
type Generator struct {
	config *config.Config
	binder *gqlgenconfig.Binder
	// StructSources holds all the struct types that will be generated from GraphQL schema
	// It contains struct information from:
	// 1. Regular object fields with nested selection sets - these are fields that contain
	//    sub-selections (not scalar fields). For example, in this query:
	//    query {
	//      user {       # object field
	//        name       # scalar field
	//        profile {  # nested object field with selection set
	//          bio      # scalar field inside nested object
	//        }
	//      }
	//    }
	//    Both "user" and "profile" are object fields with selection sets, and each will
	//    generate a separate struct (User and User_Profile)
	// 2. Fragment spreads
	// 3. Inline fragments
	// 4. Merged fields from multiple fragments - when multiple fragments are used in the same query
	//    and their fields need to be combined into a single struct. For example:
	//    query {
	//      user {
	//        id
	//        ...UserProfile      # Fragment 1
	//        ...UserPreferences  # Fragment 2
	//      }
	//    }
	//
	//    fragment UserProfile on User {
	//      name
	//      email
	//    }
	//
	//    fragment UserPreferences on User {
	//      email           # Note this field appears in both fragments
	//      preferences {
	//        theme
	//        notifications
	//      }
	//    }
	//
	//    In this case, all fields from both fragments are merged into a single User struct,
	//    with duplicate fields (like 'email') handled appropriately. The resulting struct
	//    would contain: id, name, email, and preferences fields.
	//
	// Example GraphQL query:
	// query {
	//   user {
	//     id
	//     name
	//     profile {  1. # Will generate User_Profile struct
	//       bio
	//       avatar
	//     }
	//     ...UserOrders  # 2. Will generate UserOrders struct and merge fields
	//     ... on PremiumUser {  # 3. Will generate User_PremiumUser struct
	//       subscriptionLevel
	//     }
	//   }
	// }
	StructSources []*StructSource
}

func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{
		config:        cfg,
		binder:        cfg.GQLGenConfig.NewBinder(),
		StructSources: []*StructSource{},
	}
}

func (r *Generator) NewResponseFields(selectionSet ast.SelectionSet, typeName string) ResponseFieldList {
	responseFields := make(ResponseFieldList, 0, len(selectionSet))
	for _, selection := range selectionSet {
		responseFields = append(responseFields, r.newResponseField(selection, typeName))
	}

	return responseFields
}

func (r *Generator) newResponseField(selection ast.Selection, typeName string) *ResponseField {
	var isOptional bool
	switch selection := selection.(type) {
	case *ast.Field:
		typeName = layerTypeName(typeName, templates.ToGo(selection.Alias))
		fieldsResponseFields := r.NewResponseFields(selection.SelectionSet, typeName)

		isOptional = !selection.Definition.Type.NonNull

		var baseType types.Type
		switch {
		case fieldsResponseFields.isBasicType():
			baseType = r.goType(selection.Definition.Type.Name())
		case fieldsResponseFields.isFragment():
			// if a child field is fragment, this field type became fragment.
			baseType = fieldsResponseFields[0].Type
		case fieldsResponseFields.isStructType():
			// if there is a fragment in child fields, merge it with the current field
			generator := newStructGenerator(fieldsResponseFields)

			// Merges existing struct sources with the ones from fragment processing
			// For example, if we have:
			// query {
			//   user {
			//     ...UserProfile
			//     ...UserOrders
			//   }
			// }
			// The fragments UserProfile and UserOrders will be processed and merged
			// TODO: ここは何をしている？
			// r.StructSources = mergedStructSources(r.StructSources, generator.preMergedStructSources, generator.postMergedStructSources)

			// Adds the current struct to StructSources
			// For example, in "profile { bio avatar }", this would add a User_Profile struct
			structType := generator.currentResponseFieldList.StructType()
			// r.StructSources = appendStructSources(r.StructSources, NewStructSource(typeName, structType))
			baseType = types.NewNamed(types.NewTypeName(0, r.config.GQLGencConfig.QueryGen.Pkg(), typeName, nil), structType, nil)
		default:
			// here is bug
			panic("not match type")
		}

		// return pointer type then optional type or slice pointer then slice type of definition in GraphQL.
		typ := r.binder.CopyModifiersFromAst(selection.Definition.Type, baseType)

		// json tag
		jsonTag := fmt.Sprintf(`json:"%s`, selection.Alias)
		if isOptional {
			if r.config.GQLGenConfig.EnableModelJsonOmitemptyTag != nil && *r.config.GQLGenConfig.EnableModelJsonOmitemptyTag {
				jsonTag += `,omitempty`
			}
			if r.config.GQLGenConfig.EnableModelJsonOmitzeroTag != nil && *r.config.GQLGenConfig.EnableModelJsonOmitzeroTag {
				jsonTag += `,omitzero`
			}
		}
		jsonTag += `"`

		// graphql tag
		graphqlTag := fmt.Sprintf(`graphql:"%s"`, selection.Alias)

		return &ResponseField{
			Name:           selection.Alias,
			Type:           typ,
			Tags:           []string{jsonTag, graphqlTag},
			ResponseFields: fieldsResponseFields,
		}

	case *ast.FragmentSpread:
		// This structure is not used in templates but is used to determine Fragment in ast.Field.
		// Processes fragment spreads like "...UserFragment" in:
		// query {
		//   user {
		//     ...UserFragment
		//   }
		// }
		fieldsResponseFields := r.NewResponseFields(selection.Definition.SelectionSet, layerTypeName(typeName, templates.ToGo(selection.Name)))
		baseType := types.NewNamed(
			types.NewTypeName(0, r.config.GQLGencConfig.QueryGen.Pkg(), templates.ToGo(selection.Name), nil),
			fieldsResponseFields.StructType(),
			nil,
		)

		return &ResponseField{
			Name:             selection.Name,
			Type:             baseType,
			IsFragmentSpread: true,
			ResponseFields:   fieldsResponseFields,
		}

	case *ast.InlineFragment:
		// InlineFragment has child elements, so create a struct type here
		// Processes inline fragments like "... on PremiumUser { subscriptionLevel }" in:
		// query {
		//   user {
		//     ... on PremiumUser {
		//       subscriptionLevel
		//     }
		//   }
		// }
		name := layerTypeName(typeName, templates.ToGo(selection.TypeCondition))
		fieldsResponseFields := r.NewResponseFields(selection.SelectionSet, name)

		//hasFragmentSpread := hasFragmentSpread(fieldsResponseFields)
		//fragmentFields := collectFragmentFields(fieldsResponseFields)

		// if there is a fragment spread
		//if hasFragmentSpread {
		//	// collect all fields from fragment
		//	allFields := make(ResponseFieldList, 0)
		//	for _, field := range fieldsResponseFields {
		//		if !field.IsFragmentSpread {
		//			allFields = append(allFields, field)
		//		}
		//	}
		//	// append fragment fields
		//	allFields = append(allFields, fragmentFields...)
		//
		//	// generate struct
		//	// Creates a combined struct for inline fragment with nested fragment spreads
		//	// For example:
		//	// ... on PremiumUser {
		//	//   subscriptionLevel
		//	//   ...PremiumUserDetails
		//	// }
		//	structType := allFields.StructType()
		//	// r.StructSources = appendStructSources(r.StructSources, NewStructSource(name, structType))
		//	typ := types.NewNamed(types.NewTypeName(0, r.config.GQLGencConfig.QueryGen.Pkg(), name, nil), structType, nil)
		//	return &ResponseField{
		//		Name:             selection.TypeCondition,
		//		Type:             typ,
		//		IsInlineFragment: true,
		//		Tags:             []string{fmt.Sprintf(`graphql:"... on %s"`, selection.TypeCondition)},
		//		ResponseFields:   allFields.sortByName(),
		//	}
		//}
		// if there is no fragment spread
		// Creates a simple struct for inline fragment without nested fragment spreads
		structType := fieldsResponseFields.StructType()
		// r.StructSources = appendStructSources(r.StructSources, NewStructSource(name, structType))
		typ := types.NewNamed(types.NewTypeName(0, r.config.GQLGencConfig.QueryGen.Pkg(), name, nil), structType, nil)
		return &ResponseField{
			Name:             selection.TypeCondition,
			Type:             typ,
			IsInlineFragment: true,
			Tags:             []string{fmt.Sprintf(`graphql:"... on %s"`, selection.TypeCondition)},
			ResponseFields:   fieldsResponseFields.sortByName(),
		}
	}

	panic("unexpected selection type")
}

func (r *Generator) OperationArguments(variableDefinitions ast.VariableDefinitionList) []*Argument {
	argumentTypes := make([]*Argument, 0, len(variableDefinitions))
	for _, v := range variableDefinitions {
		argumentTypes = append(argumentTypes, &Argument{
			Variable: v.Variable,
			Type:     r.binder.CopyModifiersFromAst(v.Type, r.goType(v.Type.Name())),
		})
	}

	return argumentTypes
}

func (r *Generator) OperationResponse(selection ast.Selection) *OperationResponse {
	switch v := selection.(type) {
	case *ast.Field:
		return &OperationResponse{
			Name: v.Definition.Type.Name(),
			Type: r.binder.CopyModifiersFromAst(v.Definition.Type, r.goType(v.Definition.Type.Name())),
		}
	}
	return nil
}

// The typeName passed as an argument to goType must be the name of the type derived from the parsed result,
// such as from a selection.
func (r *Generator) goType(typeName string) types.Type {
	goType, err := r.binder.FindTypeFromName(r.config.GQLGenConfig.Models[typeName].Model[0])
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	return goType
}

func appendStructSources(sources []*StructSource, appends ...*StructSource) []*StructSource {
	return append(sources, appends...)
}

// mergedStructSources combines different sets of struct sources, handling duplicates.
// This is especially important when processing fragments that may overlap.
//
// Example of merging:
// When we have a query with multiple fragments:
//
//	query {
//	  user {
//	    ...UserProfile  # Has 'name' and 'email'
//	    ...UserAccount  # Has 'email' and 'accountType'
//	  }
//	}
//
// The merged struct will have fields: 'name', 'email', and 'accountType'
func mergedStructSources(sources, preMergedStructSources, postMergedStructSources []*StructSource) []*StructSource {
	preMergedStructSourcesMap := structSourcesMapByTypeName(preMergedStructSources)
	res := make([]*StructSource, 0)
	// remove pre-merged struct
	for _, source := range sources {
		// when name is same, remove it
		if _, ok := preMergedStructSourcesMap[source.Name]; ok {
			continue
		}
		res = append(res, source)
	}

	// append post-merged struct
	res = append(res, postMergedStructSources...)

	return res
}

// hasFragmentSpread checks if any field in the ResponseFieldList is a fragment spread.
// Used to determine if special fragment handling is needed.
func hasFragmentSpread(fields ResponseFieldList) bool {
	for _, field := range fields {
		if field.IsFragmentSpread {
			return true
		}
	}
	return false
}

// collectFragmentFields extracts all fields from fragment spreads in the ResponseFieldList.
// This is used when merging fragments into a parent struct.
func collectFragmentFields(fields ResponseFieldList) ResponseFieldList {
	var fragmentFields ResponseFieldList
	for _, field := range fields {
		if field.IsFragmentSpread {
			fragmentFields = append(fragmentFields, field.ResponseFields...)
		}
	}
	return fragmentFields
}

// mergeFieldsRecursively combines fields from source and target ResponseFieldLists,
// handling nested fields properly.
//
// This is crucial for handling complex GraphQL fragments:
//
//	fragment UserWithProfile on User {
//	  id
//	  profile {
//	    bio
//	  }
//	}
//
//	fragment UserWithDetailedProfile on User {
//	  profile {
//	    avatar
//	    links
//	  }
//	}
//
// When both fragments are used together, the 'profile' field needs
// to include all sub-fields: bio, avatar, and links.
func mergeFieldsRecursively(targetFields, sourceFields ResponseFieldList, preMerged, postMerged []*StructSource) (ResponseFieldList, []*StructSource, []*StructSource) {
	responseFieldList := make(ResponseFieldList, 0)
	targetFieldsMap := targetFields.mapByName()
	newPreMerged := preMerged
	newPostMerged := postMerged

	for _, sourceField := range sourceFields {
		if targetField, ok := targetFieldsMap[sourceField.Name]; ok {
			if targetField.ResponseFields.isBasicType() {
				continue
			}
			newPreMerged = append(newPreMerged, &StructSource{
				Name: sourceField.fieldTypeString(),
				Type: sourceField.ResponseFields.StructType(),
			})
			newPreMerged = append(newPreMerged, &StructSource{
				Name: targetField.fieldTypeString(),
				Type: targetField.ResponseFields.StructType(),
			})

			targetField.ResponseFields, newPreMerged, newPostMerged = mergeFieldsRecursively(targetField.ResponseFields, sourceField.ResponseFields, newPreMerged, newPostMerged)

			newPostMerged = append(newPostMerged, &StructSource{
				Name: targetField.fieldTypeString(),
				Type: targetField.ResponseFields.StructType(),
			})
		} else {
			targetFieldsMap[sourceField.Name] = sourceField
		}
	}
	for _, field := range targetFieldsMap {
		responseFieldList = append(responseFieldList, field)
	}
	responseFieldList = responseFieldList.sortByName()
	return responseFieldList, newPreMerged, newPostMerged
}

func structSourcesMapByTypeName(sources []*StructSource) map[string]*StructSource {
	res := make(map[string]*StructSource)
	for _, source := range sources {
		res[source.Name] = source
	}
	return res
}

// layerTypeName creates a qualified name for nested types.
// For example, if we have a query with nested fields:
//
//	user {
//	  profile {
//	    settings {
//	      notifications
//	    }
//	  }
//	}
//
// This would generate names like User_Profile and User_Profile_Settings
func layerTypeName(base, thisField string) string {
	return fmt.Sprintf("%s_%s", cases.Title(language.Und, cases.NoLower).String(base), thisField)
}

type ResponseField struct {
	Name             string
	IsFragmentSpread bool
	IsInlineFragment bool
	Type             types.Type
	Tags             []string
	ResponseFields   ResponseFieldList
}

func (r ResponseField) fieldTypeString() string {
	fullFieldType := r.Type.String()
	parts := strings.Split(fullFieldType, ".")
	return parts[len(parts)-1]
}

type ResponseFieldList []*ResponseField

func (rs ResponseFieldList) UniqueByName() ResponseFieldList {
	responseFieldMapByName := make(map[string]*ResponseField, len(rs))
	for _, field := range rs {
		responseFieldMapByName[field.Name] = field
	}
	return slices.Collect(maps.Values(responseFieldMapByName))
}

func (rs ResponseFieldList) IsFragmentSpread() bool {
	if len(rs) != 1 {
		return false
	}

	return rs[0].IsFragmentSpread
}

// StructType creates a Go struct type from the ResponseFieldList.
// This is used to generate the final struct definitions for GraphQL objects.
//
// For example, with this GraphQL:
//
//	user {
//	  id
//	  name
//	  email
//	}
//
// It would create a struct with fields for id, name, and email,
// including appropriate JSON and GraphQL tags.
func (rs ResponseFieldList) StructType() *types.Struct {
	vars := make([]*types.Var, 0)
	structTags := make([]string, 0)
	for _, field := range rs.UniqueByName() {
		vars = append(vars, types.NewVar(0, nil, templates.ToGo(field.Name), field.Type))
		structTags = append(structTags, strings.Join(field.Tags, " "))
	}
	return types.NewStruct(vars, structTags)
}

func (rs ResponseFieldList) isFragment() bool {
	if len(rs) != 1 {
		return false
	}

	return rs[0].IsInlineFragment || rs[0].IsFragmentSpread
}

func (rs ResponseFieldList) isBasicType() bool {
	return len(rs) == 0
}

func (rs ResponseFieldList) isStructType() bool {
	return len(rs) > 0 && !rs.isFragment()
}

func (rs ResponseFieldList) mapByName() map[string]*ResponseField {
	res := make(map[string]*ResponseField)
	for _, field := range rs {
		res[field.Name] = field
	}
	return res
}

func (rs ResponseFieldList) sortByName() ResponseFieldList {
	slices.SortFunc(rs, func(a, b *ResponseField) int {
		return strings.Compare(a.Name, b.Name)
	})
	return rs
}

// StructGenerator manages the creation of Go struct types from GraphQL selections.
// It handles the complexity of merging fields from different fragments.
type StructGenerator struct {
	// Create fields based on this ResponseFieldList
	currentResponseFieldList ResponseFieldList
	// Struct sources that will no longer be created due to merging
	preMergedStructSources []*StructSource
	// Struct sources that will be created due to merging
	postMergedStructSources []*StructSource
}

// newStructGenerator creates a new StructGenerator instance that handles
// the transformation of GraphQL selections to Go structs.
//
// Example of structure generation for a query with fragments:
//
//	query {
//	  user {
//	    id
//	    ...UserProfile
//	    ...UserPreferences
//	  }
//	}
//
//	fragment UserProfile on User {
//	  name
//	  email
//	}
//
//	fragment UserPreferences on User {
//	  theme
//	  notifications
//	}
//
// The resulting struct would have all fields: id, name, email, theme, notifications
func newStructGenerator(responseFieldList ResponseFieldList) *StructGenerator {
	currentFields := make(ResponseFieldList, 0)
	fragmentChildrenFields := make(ResponseFieldList, 0)
	for _, field := range responseFieldList {
		if field.IsFragmentSpread {
			fragmentChildrenFields = append(fragmentChildrenFields, field.ResponseFields...)
		} else {
			currentFields = append(currentFields, field)
		}
	}

	preMergedStructSources := make([]*StructSource, 0)

	for _, field := range responseFieldList {
		if field.IsFragmentSpread {
			preMergedStructSources = append(preMergedStructSources, &StructSource{
				Name: field.fieldTypeString(),
				Type: field.ResponseFields.StructType(),
			})
		}
	}

	currentFields, preMergedStructSources, postMergedStructSources := mergeFieldsRecursively(currentFields, fragmentChildrenFields, preMergedStructSources, nil)
	return &StructGenerator{
		currentResponseFieldList: currentFields,
		preMergedStructSources:   preMergedStructSources,
		postMergedStructSources:  postMergedStructSources,
	}
}

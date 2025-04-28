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

// GoTypeGenerator generate Go goType from GraphQL goType
type GoTypeGenerator struct {
	config *config.Config
	binder *gqlgenconfig.Binder
	// QueryTypes holds all the struct types that will be generated from GraphQL schema
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
	QueryTypes []*QueryType
}

func NewGoTypeGenerator(cfg *config.Config) *GoTypeGenerator {
	return &GoTypeGenerator{
		config:     cfg,
		binder:     cfg.GQLGenConfig.NewBinder(),
		QueryTypes: []*QueryType{},
	}
}

func (r *GoTypeGenerator) OperationArguments(variableDefinitions ast.VariableDefinitionList) []*OperationArgument {
	operationArguments := make([]*OperationArgument, 0, len(variableDefinitions))
	for _, v := range variableDefinitions {
		t := r.binder.CopyModifiersFromAst(v.Type, r.goType(v.Type.Name()))
		operationArgument := NewOperationArgument(v.Variable, t)
		operationArguments = append(operationArguments, operationArgument)
	}

	return operationArguments
}
func (r *GoTypeGenerator) OperationResponse(selectionSet ast.SelectionSet, typeName string) *OperationResponse {
	responseFields := r.newFields(selectionSet, typeName)
	return NewOperationResponse(typeName, responseFields.StructType())
}

func (r *GoTypeGenerator) Fragment(fragmentSelectionSet ast.SelectionSet, fragmentTypeName string) *Fragment {
	fragmentTypeFields := r.newFields(fragmentSelectionSet, fragmentTypeName)
	return NewFragment(fragmentTypeName, fragmentTypeFields.StructType())
}

func (r *GoTypeGenerator) newFields(selectionSet ast.SelectionSet, typeName string) Fields {
	fields := make(Fields, 0, len(selectionSet))
	for _, selection := range selectionSet {
		fields = append(fields, r.newField(selection, typeName))
	}

	return fields
}

func (r *GoTypeGenerator) newField(selection ast.Selection, typeName string) *Field {
	switch selection := selection.(type) {
	case *ast.Field:
		typeName = layerTypeName(typeName, templates.ToGo(selection.Alias))
		fields := r.newFields(selection.SelectionSet, typeName)

		var baseType types.Type
		switch {
		case fields.isBasicType():
			baseType = r.goType(selection.Definition.Type.Name())
		case fields.isFragment():
			// if a child field is fragment, this field type became fragment.
			baseType = fields[0].Type
		case fields.isQueryType():
			generator := newQueryTypeGen(fields)
			r.QueryTypes = mergedQueryTypes(r.QueryTypes, generator.preMergedQueryTypes, generator.postMergedQueryTypes)

			// Adds the current struct to StructSources
			// For example, in "profile { bio avatar }", this would add a User_Profile struct
			structType := generator.currentFields.StructType()
			r.QueryTypes = appendStructSources(r.QueryTypes, NewQueryType(typeName, structType))
			baseType = types.NewNamed(types.NewTypeName(0, r.config.GQLGencConfig.QueryGen.Pkg(), typeName, nil), structType, nil)
		default:
			// here is bug
			panic("not match type")
		}
		// return pointer type then optional type or slice pointer then slice type of definition in GraphQL.
		typ := r.binder.CopyModifiersFromAst(selection.Definition.Type, baseType)
		// tags
		jsonTag := r.jsonTag(selection.Alias, selection.Definition.Type.NonNull)
		graphqlTag := fmt.Sprintf(`graphql:"%s"`, selection.Alias)
		return &Field{
			Name:           selection.Alias,
			Type:           typ,
			Tags:           []string{jsonTag, graphqlTag},
			ResponseFields: fields,
		}
	case *ast.FragmentSpread:
		// This structure is not used in templates but is used to determine Fragment in ast.Field.
		// Processes fragment spreads like "...UserFragment" in:
		// query {
		//   user {
		//     ...UserFragment
		//   }
		// }
		fragmentName := templates.ToGo(selection.Name)
		fieldsResponseFields := r.newFields(selection.Definition.SelectionSet, layerTypeName(typeName, fragmentName))
		baseType := types.NewNamed(types.NewTypeName(0, r.config.GQLGencConfig.QueryGen.Pkg(), fragmentName, nil), fieldsResponseFields.StructType(), nil)
		return &Field{
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
		fieldsResponseFields := r.newFields(selection.SelectionSet, name)

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
		// 今の所なくてよさそう
		// r.QueryTypes = appendStructSources(r.QueryTypes, NewQueryType(name, structType))
		typ := types.NewNamed(types.NewTypeName(0, r.config.GQLGencConfig.QueryGen.Pkg(), name, nil), structType, nil)
		return &Field{
			Name:             selection.TypeCondition,
			Type:             typ,
			IsInlineFragment: true,
			Tags:             []string{fmt.Sprintf(`graphql:"... on %s"`, selection.TypeCondition)},
			ResponseFields:   fieldsResponseFields.sortByName(),
		}
	}

	panic("unexpected selection type")
}

// such as from a selection.
func (r *GoTypeGenerator) jsonTag(typeName string, nonNull bool) string {
	// json tag
	jsonTag := fmt.Sprintf(`json:"%s`, typeName)
	if !nonNull {
		if r.config.GQLGenConfig.EnableModelJsonOmitemptyTag != nil && *r.config.GQLGenConfig.EnableModelJsonOmitemptyTag {
			jsonTag += `,omitempty`
		}
		if r.config.GQLGenConfig.EnableModelJsonOmitzeroTag != nil && *r.config.GQLGenConfig.EnableModelJsonOmitzeroTag {
			jsonTag += `,omitzero`
		}
	}
	jsonTag += `"`
	return jsonTag
}

// The typeName passed as an argument to goType must be the name of the type derived from the parsed result,
// such as from a selection.
func (r *GoTypeGenerator) goType(typeName string) types.Type {
	goType, err := r.binder.FindTypeFromName(r.config.GQLGenConfig.Models[typeName].Model[0])
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	return goType
}

func appendStructSources(sources []*QueryType, appends ...*QueryType) []*QueryType {
	return append(sources, appends...)
}

// mergedQueryTypes combines different sets of struct sources, handling duplicates.
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
func mergedQueryTypes(queryTypes, preMergedStructSources, postMergedStructSources []*QueryType) []*QueryType {
	preMergedStructSourcesMap := structSourcesMapByTypeName(preMergedStructSources)
	res := make([]*QueryType, 0)
	// remove pre-merged struct
	for _, queryType := range queryTypes {
		// when name is same, remove it
		if _, ok := preMergedStructSourcesMap[queryType.Name]; ok {
			continue
		}
		res = append(res, queryType)
	}

	// append post-merged struct
	res = append(res, postMergedStructSources...)

	return res
}

// hasFragmentSpread checks if any field in the Fields is a fragment spread.
// Used to determine if special fragment handling is needed.
func hasFragmentSpread(fields Fields) bool {
	for _, field := range fields {
		if field.IsFragmentSpread {
			return true
		}
	}
	return false
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
func mergeFieldsRecursively(targetFields, sourceFields Fields, preMerged, postMerged []*QueryType) (Fields, []*QueryType, []*QueryType) {
	responseFieldList := make(Fields, 0)
	targetFieldsMap := targetFields.mapByName()
	newPreMerged := preMerged
	newPostMerged := postMerged

	for _, sourceField := range sourceFields {
		if targetField, ok := targetFieldsMap[sourceField.Name]; ok {
			if targetField.ResponseFields.isBasicType() {
				continue
			}
			newPreMerged = append(newPreMerged, &QueryType{
				Name: sourceField.fieldTypeString(),
				Type: sourceField.ResponseFields.StructType(),
			})
			newPreMerged = append(newPreMerged, &QueryType{
				Name: targetField.fieldTypeString(),
				Type: targetField.ResponseFields.StructType(),
			})

			// targetField.ResponseFields, newPreMerged, newPostMerged = mergeFieldsRecursively(targetField.ResponseFields, sourceField.ResponseFields, newPreMerged, newPostMerged)

			newPostMerged = append(newPostMerged, &QueryType{
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

func structSourcesMapByTypeName(sources []*QueryType) map[string]*QueryType {
	res := make(map[string]*QueryType)
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

type Field struct {
	Name             string
	IsFragmentSpread bool
	IsInlineFragment bool
	Type             types.Type
	Tags             []string
	ResponseFields   Fields
}

func (r Field) fieldTypeString() string {
	fullFieldType := r.Type.String()
	parts := strings.Split(fullFieldType, ".")
	return parts[len(parts)-1]
}

type Fields []*Field

func (rs Fields) UniqueByName() Fields {
	responseFieldMapByName := make(map[string]*Field, len(rs))
	for _, field := range rs {
		responseFieldMapByName[field.Name] = field
	}
	return slices.Collect(maps.Values(responseFieldMapByName))
}

func (rs Fields) IsFragmentSpread() bool {
	if len(rs) != 1 {
		return false
	}

	return rs[0].IsFragmentSpread
}

// StructType creates a Go struct type from the Fields.
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
func (rs Fields) StructType() *types.Struct {
	vars := make([]*types.Var, 0)
	structTags := make([]string, 0)
	for _, field := range rs.UniqueByName() {
		vars = append(vars, types.NewVar(0, nil, templates.ToGo(field.Name), field.Type))
		structTags = append(structTags, strings.Join(field.Tags, " "))
	}
	return types.NewStruct(vars, structTags)
}

func (rs Fields) isFragment() bool {
	if len(rs) != 1 {
		return false
	}

	return rs[0].IsInlineFragment || rs[0].IsFragmentSpread
}

func (rs Fields) isBasicType() bool {
	return len(rs) == 0
}

func (rs Fields) isQueryType() bool {
	return len(rs) > 0 && !rs.isFragment()
}

func (rs Fields) mapByName() map[string]*Field {
	res := make(map[string]*Field)
	for _, field := range rs {
		res[field.Name] = field
	}
	return res
}

func (rs Fields) sortByName() Fields {
	slices.SortFunc(rs, func(a, b *Field) int {
		return strings.Compare(a.Name, b.Name)
	})
	return rs
}

// QueryTypeGenerator manages the creation of Go struct types from GraphQL selections.
// It handles the complexity of merging fields from different fragments.
type QueryTypeGenerator struct {
	// Create fields based on this Fields
	currentFields Fields
	// Struct sources that will no longer be created due to merging
	preMergedQueryTypes []*QueryType
	// Struct sources that will be created due to merging
	postMergedQueryTypes []*QueryType
}

// newQueryTypeGen creates a new QueryTypeGenerator instance that handles
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
func newQueryTypeGen(responseFieldList Fields) *QueryTypeGenerator {
	currentFields := make(Fields, 0)
	fragmentChildrenFields := make(Fields, 0)
	for _, field := range responseFieldList {
		if field.IsFragmentSpread {
			fragmentChildrenFields = append(fragmentChildrenFields, field.ResponseFields...)
		} else {
			currentFields = append(currentFields, field)
		}
	}

	preMergedStructSources := make([]*QueryType, 0)

	for _, field := range responseFieldList {
		if field.IsFragmentSpread {
			preMergedStructSources = append(preMergedStructSources, &QueryType{
				Name: field.fieldTypeString(),
				Type: field.ResponseFields.StructType(),
			})
		}
	}

	currentFields, preMergedStructSources, postMergedStructSources := mergeFieldsRecursively(currentFields, fragmentChildrenFields, preMergedStructSources, nil)
	return &QueryTypeGenerator{
		currentFields:        currentFields,
		preMergedQueryTypes:  preMergedStructSources,
		postMergedQueryTypes: postMergedStructSources,
	}
}

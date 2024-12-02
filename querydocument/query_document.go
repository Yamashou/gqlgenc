package querydocument

import (
	"fmt"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"
)

func QueryDocumentsByOperations(schema *ast.Schema, operations ast.OperationList) ([]*ast.QueryDocument, error) {
	queryDocuments := make([]*ast.QueryDocument, 0, len(operations))
	for _, operation := range operations {
		fragments := fragmentsInOperationDefinition(operation)

		queryDocument := &ast.QueryDocument{
			Operations: ast.OperationList{operation},
			Fragments:  fragments,
			Position:   nil,
		}

		if errs := validator.Validate(schema, queryDocument); errs != nil {
			return nil, fmt.Errorf(": %w", errs)
		}

		queryDocuments = append(queryDocuments, queryDocument)
	}

	return queryDocuments, nil
}

func fragmentsInOperationDefinition(operation *ast.OperationDefinition) ast.FragmentDefinitionList {
	fragments := fragmentsInOperationWalker(operation.SelectionSet)
	uniqueFragments := fragmentsUnique(fragments)

	return uniqueFragments
}

func fragmentsUnique(fragments ast.FragmentDefinitionList) ast.FragmentDefinitionList {
	seenFragments := make(map[string]struct{}, len(fragments))
	uniqueFragments := make(ast.FragmentDefinitionList, 0, len(fragments))
	for _, fragment := range fragments {
		if _, ok := seenFragments[fragment.Name]; ok {
			continue
		}
		uniqueFragments = append(uniqueFragments, fragment)
		seenFragments[fragment.Name] = struct{}{}
	}

	return uniqueFragments
}

func fragmentsInOperationWalker(selectionSet ast.SelectionSet) ast.FragmentDefinitionList {
	var fragments ast.FragmentDefinitionList
	for _, selection := range selectionSet {
		var selectionSet ast.SelectionSet
		switch selection := selection.(type) {
		case *ast.Field:
			selectionSet = selection.SelectionSet
		case *ast.InlineFragment:
			selectionSet = selection.SelectionSet
		case *ast.FragmentSpread:
			fragments = append(fragments, selection.Definition)
			selectionSet = selection.Definition.SelectionSet
		}

		fragments = append(fragments, fragmentsInOperationWalker(selectionSet)...)
	}

	return fragments
}

// CollectTypesFromQueryDocuments returns a map of type names used in query document arguments
func CollectTypesFromQueryDocuments(schema *ast.Schema, queryDocuments []*ast.QueryDocument) map[string]bool {
	usedTypes := make(map[string]bool)

	for _, doc := range queryDocuments {
		for _, op := range doc.Operations {
			// Collect types from variable definitions
			for _, v := range op.VariableDefinitions {
				collectTypeFromTypeReference(v.Type, usedTypes)
				// Recursively collect input object fields
				if def, ok := schema.Types[v.Type.Name()]; ok && def.IsInputType() {
					collectInputObjectFields(def, schema, usedTypes)
				}
			}
		}
	}

	return usedTypes
}

// collectInputObjectFields recursively collects types from input object fields
func collectInputObjectFields(def *ast.Definition, schema *ast.Schema, usedTypes map[string]bool) {
	// Skip if type has already been processed
	if _, ok := usedTypes[def.Name]; ok {
		return
	}
	usedTypes[def.Name] = true

	for _, field := range def.Fields {
		// Get the actual type name
		typeName := field.Type.NamedType
		if typeName == "" {
			// For list types, get the element type name
			if field.Type.Elem != nil {
				typeName = field.Type.Elem.NamedType
			}
		}

		if typeName != "" {
			usedTypes[typeName] = true
			// If field is an input object type, collect recursively
			if fieldDef, ok := schema.Types[typeName]; ok && fieldDef.IsInputType() {
				collectInputObjectFields(fieldDef, schema, usedTypes)
			}
		}
	}
}

// collectTypeFromTypeReference is a helper function to collect type names from type references
func collectTypeFromTypeReference(t *ast.Type, usedTypes map[string]bool) {
	if t == nil {
		return
	}

	if t.NamedType != "" {
		usedTypes[t.NamedType] = true
	}

	collectTypeFromTypeReference(t.Elem, usedTypes)
}

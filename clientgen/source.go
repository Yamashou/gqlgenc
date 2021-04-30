package clientgen

import (
	"bytes"
	"fmt"
	"go/types"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/vanillaricewraps/gqlgenc/config"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"golang.org/x/xerrors"
)

type Source struct {
	schema          *ast.Schema
	queryDocument   *ast.QueryDocument
	sourceGenerator *SourceGenerator
	generateConfig  *config.GenerateConfig
}

func NewSource(schema *ast.Schema, queryDocument *ast.QueryDocument, sourceGenerator *SourceGenerator, generateConfig *config.GenerateConfig) *Source {
	return &Source{
		schema:          schema,
		queryDocument:   queryDocument,
		sourceGenerator: sourceGenerator,
		generateConfig:  generateConfig,
	}
}

type Fragment struct {
	Name string
	Type types.Type
}

func (s *Source) Fragments() ([]*Fragment, error) {
	fragments := make([]*Fragment, 0, len(s.queryDocument.Fragments))
	for _, fragment := range s.queryDocument.Fragments {
		responseFields := s.sourceGenerator.NewResponseFields(fragment.SelectionSet, "")
		if s.sourceGenerator.cfg.Models.Exists(fragment.Name) {
			return nil, xerrors.New(fmt.Sprintf("%s is duplicated", fragment.Name))
		}

		fragment := &Fragment{
			Name: fragment.Name,
			Type: responseFields.StructType(),
		}

		fragments = append(fragments, fragment)
	}

	for _, fragment := range fragments {
		name := fragment.Name
		s.sourceGenerator.cfg.Models.Add(
			name,
			fmt.Sprintf("%s.%s", s.sourceGenerator.client.Pkg(), templates.ToGo(name)),
		)
	}

	return fragments, nil
}

type Operation struct {
	Name                string
	ResponseStructName  string
	Operation           string
	Args                []*Argument
	VariableDefinitions ast.VariableDefinitionList
}

func NewOperation(operation *ast.OperationDefinition, queryDocument *ast.QueryDocument, args []*Argument, generateConfig *config.GenerateConfig) *Operation {
	return &Operation{
		Name:                operation.Name,
		ResponseStructName:  getResponseStructName(operation, generateConfig),
		Operation:           queryString(queryDocument),
		Args:                args,
		VariableDefinitions: operation.VariableDefinitions,
	}
}

func (s *Source) Operations(queryDocuments []*ast.QueryDocument) ([]*Operation, error) {
	operations := make([]*Operation, 0, len(s.queryDocument.Operations))

	operationNames := make(map[string]struct{})

	queryDocumentsMap := queryDocumentMapByOperationName(queryDocuments)
	operationArgsMap := s.operationArgsMapByOperationName()
	for _, operation := range s.queryDocument.Operations {
		queryDocument := queryDocumentsMap[operation.Name]

		_, exist := operationNames[templates.ToGo(operation.Name)]
		if exist {
			return nil, xerrors.Errorf("duplicate operation: %s", operation.Name)
		}
		operationNames[templates.ToGo(operation.Name)] = struct{}{}

		args := operationArgsMap[operation.Name]
		operations = append(operations, NewOperation(
			operation,
			queryDocument,
			args,
			s.generateConfig,
		))
	}

	return operations, nil
}

func (s *Source) operationArgsMapByOperationName() map[string][]*Argument {
	operationArgsMap := make(map[string][]*Argument)
	for _, operation := range s.queryDocument.Operations {
		operationArgsMap[operation.Name] = s.sourceGenerator.OperationArguments(operation.VariableDefinitions)
	}

	return operationArgsMap
}

func queryDocumentMapByOperationName(queryDocuments []*ast.QueryDocument) map[string]*ast.QueryDocument {
	queryDocumentMap := make(map[string]*ast.QueryDocument)
	for _, queryDocument := range queryDocuments {
		operation := queryDocument.Operations[0]
		queryDocumentMap[operation.Name] = queryDocument
	}

	return queryDocumentMap
}

func queryString(queryDocument *ast.QueryDocument) string {
	var buf bytes.Buffer
	astFormatter := formatter.NewFormatter(&buf)
	astFormatter.FormatQueryDocument(queryDocument)

	return buf.String()
}

type OperationResponse struct {
	Name string
	Type types.Type
}

func getNestedTypes(source *Source, selectionSet ast.SelectionSet, parentName string, indent int) []*OperationResponse {
	// tabs := strings.Repeat("  ", indent)
	var results []*OperationResponse
	for _, selection := range selectionSet {
		switch v := selection.(type) {
		case nil:
			panic("nil")
		case *ast.Field:
			responseFields := source.sourceGenerator.NewResponseFields(v.SelectionSet, parentName+templates.ToGo(v.Alias))
			if responseFields.IsStructType() {
				/*
					typ := types.NewNamed(
						types.NewTypeName(0, source.sourceGenerator.client.Pkg(), templates.ToGo(v.Alias), nil),
						responseFields.StructType(),
						nil,
					)
				*/

				// This is where we define the nested fields like Nodes
				results = append(results, &OperationResponse{
					Name: parentName + templates.ToGo(v.Alias),
					// Type: typ,
					Type: responseFields.StructType(),
				})
				fmt.Printf("Add %s\n", parentName+templates.ToGo(v.Alias))
				results = append(results, getNestedTypes(source, v.SelectionSet, parentName+templates.ToGo(v.Alias), indent+1)...)
			}

			/*
				if len(v.SelectionSet) > 0 {
					recursive(source, v.SelectionSet, indent+1)
				}
			*/

		default:
			fmt.Println("unknown", v)
		}
	}
	return results
}

func (s *Source) OperationResponses() ([]*OperationResponse, error) {
	// operationResponse := make([]*OperationResponse, 0, len(s.queryDocument.Operations))
	var operationResponse []*OperationResponse
	for _, operation := range s.queryDocument.Operations {
		queryName := getResponseStructName(operation, s.generateConfig)
		fmt.Printf("Query: %s\n", queryName)
		responseFields := s.sourceGenerator.NewResponseFields(operation.SelectionSet, queryName)

		nestedTypes := getNestedTypes(s, operation.SelectionSet, queryName, 0)
		operationResponse = append(operationResponse, nestedTypes...)

		if s.sourceGenerator.cfg.Models.Exists(queryName) {
			return nil, xerrors.New(fmt.Sprintf("%s is duplicated", queryName))
		}
		operationResponse = append(operationResponse, &OperationResponse{
			Name: queryName,
			Type: responseFields.StructType(),
			// Type: typ,
		})
	}

	for _, operationResponse := range operationResponse {
		name := operationResponse.Name
		s.sourceGenerator.cfg.Models.Add(
			name,
			fmt.Sprintf("%s.%s", s.sourceGenerator.client.Pkg(), templates.ToGo(name)),
		)
	}

	return operationResponse, nil
}

type Query struct {
	Name string
	Type types.Type
}

func (s *Source) Query() (*Query, error) {
	fields, err := s.sourceGenerator.NewResponseFieldsByDefinition(s.schema.Query)
	if err != nil {
		return nil, xerrors.Errorf("generate failed for query struct type : %w", err)
	}

	s.sourceGenerator.cfg.Models.Add(
		s.schema.Query.Name,
		fmt.Sprintf("%s.%s", s.sourceGenerator.client.Pkg(), templates.ToGo(s.schema.Query.Name)),
	)

	return &Query{
		Name: s.schema.Query.Name,
		Type: fields.StructType(),
	}, nil
}

type Mutation struct {
	Name string
	Type types.Type
}

func (s *Source) Mutation() (*Mutation, error) {
	if s.schema.Mutation == nil {
		return nil, nil
	}

	fields, err := s.sourceGenerator.NewResponseFieldsByDefinition(s.schema.Mutation)
	if err != nil {
		return nil, xerrors.Errorf("generate failed for mutation struct type : %w", err)
	}

	s.sourceGenerator.cfg.Models.Add(
		s.schema.Mutation.Name,
		fmt.Sprintf("%s.%s", s.sourceGenerator.client.Pkg(), templates.ToGo(s.schema.Mutation.Name)),
	)

	return &Mutation{
		Name: s.schema.Mutation.Name,
		Type: fields.StructType(),
	}, nil
}

func getResponseStructName(operation *ast.OperationDefinition, generateConfig *config.GenerateConfig) string {
	name := operation.Name
	if generateConfig != nil {
		if generateConfig.Prefix != nil {
			if operation.Operation == ast.Mutation {
				name = fmt.Sprintf("%s%s", generateConfig.Prefix.Mutation, name)
			}

			if operation.Operation == ast.Query {
				name = fmt.Sprintf("%s%s", generateConfig.Prefix.Query, name)
			}
		}

		if generateConfig.Suffix != nil {
			if operation.Operation == ast.Mutation {
				name = fmt.Sprintf("%s%s", name, generateConfig.Suffix.Mutation)
			}

			if operation.Operation == ast.Query {
				name = fmt.Sprintf("%s%s", name, generateConfig.Suffix.Query)
			}
		}
	}

	return name
}

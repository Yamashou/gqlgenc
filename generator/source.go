package generator

import (
	"bytes"
	"go/types"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
)

type Source struct {
	schema          *ast.Schema
	queryDocument   *ast.QueryDocument
	sourceGenerator *SourceGenerator
}

func NewSource(schema *ast.Schema, queryDocument *ast.QueryDocument, sourceGenerator *SourceGenerator) *Source {
	return &Source{
		schema:          schema,
		queryDocument:   queryDocument,
		sourceGenerator: sourceGenerator,
	}
}

type Fragment struct {
	Name string
	Type types.Type
}

func (s *Source) Fragments() ([]*Fragment, error) {
	fragments := make([]*Fragment, 0, len(s.queryDocument.Fragments))
	for _, fragment := range s.queryDocument.Fragments {
		responseFields := s.sourceGenerator.NewResponseFields(fragment.SelectionSet, fragment.Name)
		fragment := &Fragment{
			Name: fragment.Name,
			Type: responseFields.StructType(),
		}

		fragments = append(fragments, fragment)
	}

	return fragments, nil
}

type Operation struct {
	Name                string
	VariableDefinitions ast.VariableDefinitionList
	Operation           string
	Args                []*Argument
	OperationResponse   *OperationResponse
}

func NewOperation(operation *ast.OperationDefinition, queryDocument *ast.QueryDocument, args []*Argument, operationResponse *OperationResponse) *Operation {
	return &Operation{
		Name:                operation.Name,
		VariableDefinitions: operation.VariableDefinitions,
		Operation:           queryString(queryDocument),
		Args:                args,
		OperationResponse:   operationResponse,
	}
}

func (s *Source) Operations(queryDocuments []*ast.QueryDocument) ([]*Operation, error) {
	operations := make([]*Operation, 0, len(s.queryDocument.Operations))

	queryDocumentsMap := queryDocumentMapByOperationName(queryDocuments)
	operationArgsMap := s.operationArgsMapByOperationName()
	operationResponseMapByOperationName := s.operationResponseMapByOperationName()

	for _, operation := range s.queryDocument.Operations {
		queryDocument := queryDocumentsMap[operation.Name]
		args := operationArgsMap[operation.Name]
		operationResponse := operationResponseMapByOperationName[operation.Name]
		operations = append(operations, NewOperation(operation, queryDocument, args, operationResponse))
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

func (s *Source) operationResponseMapByOperationName() map[string]*OperationResponse {
	operationResponseMap := make(map[string]*OperationResponse)
	for _, operation := range s.queryDocument.Operations {
		operationResponseMap[operation.Name] = s.sourceGenerator.OperationResponse(operation.SelectionSet[0])
	}

	return operationResponseMap
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

func (s *Source) OperationResponses() ([]*OperationResponse, error) {
	operationResponse := make([]*OperationResponse, 0, len(s.queryDocument.Operations))
	for _, operation := range s.queryDocument.Operations {
		responseFields := s.sourceGenerator.NewResponseFields(operation.SelectionSet, operation.Name)
		operationResponse = append(operationResponse, &OperationResponse{
			Name: operation.Name,
			Type: responseFields.StructType(),
		})
	}

	return operationResponse, nil
}

func (s *Source) ResponseSubTypes() []*StructSource {
	return s.sourceGenerator.StructSources
}

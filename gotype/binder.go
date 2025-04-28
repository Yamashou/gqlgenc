package gotype

import (
	"bytes"
	"go/types"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"

	"github.com/Yamashou/gqlgenc/v3/config"
)

// Binder binds GraphQL Types to Go Types.
type Binder struct {
	schema        *ast.Schema
	queryDocument *ast.QueryDocument
	generator     *Generator
}

func NewBinder(cfg *config.Config, queryDocument *ast.QueryDocument) *Binder {
	return &Binder{
		schema:        cfg.GQLGenConfig.Schema,
		queryDocument: queryDocument,
		generator:     NewGenerator(cfg),
	}
}

type Fragment struct {
	Name string
	Type types.Type
}

func (s *Binder) Fragments() ([]*Fragment, error) {
	fragments := make([]*Fragment, 0, len(s.queryDocument.Fragments))
	for _, fragment := range s.queryDocument.Fragments {
		responseFields := s.generator.NewResponseFields(fragment.SelectionSet, fragment.Name)
		fragment := &Fragment{
			Name: fragment.Name,
			Type: responseFields.StructType(),
		}

		fragments = append(fragments, fragment)
	}

	return fragments, nil
}

func (s *Binder) Operations(queryDocuments []*ast.QueryDocument) ([]*Operation, error) {
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

func (s *Binder) OperationResponses() ([]*OperationResponse, error) {
	operationResponse := make([]*OperationResponse, 0, len(s.queryDocument.Operations))
	for _, operation := range s.queryDocument.Operations {
		responseFields := s.generator.NewResponseFields(operation.SelectionSet, operation.Name)
		operationResponse = append(operationResponse, &OperationResponse{
			Name: operation.Name,
			Type: responseFields.StructType(),
		})
	}

	return operationResponse, nil
}

func (s *Binder) ResponseSubTypes() []*StructSource {
	return s.generator.StructSources
}

func (s *Binder) operationArgsMapByOperationName() map[string][]*OperationArgument {
	operationArgsMap := make(map[string][]*OperationArgument)
	for _, operation := range s.queryDocument.Operations {
		operationArgsMap[operation.Name] = s.generator.OperationArguments(operation.VariableDefinitions)
	}

	return operationArgsMap
}

func (s *Binder) operationResponseMapByOperationName() map[string]*OperationResponse {
	operationResponseMap := make(map[string]*OperationResponse)
	for _, operation := range s.queryDocument.Operations {
		operationResponseMap[operation.Name] = s.generator.OperationResponse(operation.SelectionSet[0])
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

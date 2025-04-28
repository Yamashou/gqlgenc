package gotype

import (
	"bytes"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"

	"github.com/Yamashou/gqlgenc/v3/config"
)

// Binder binds GraphQL Types to Go Types.
type Binder struct {
	schema          *ast.Schema
	queryDocument   *ast.QueryDocument
	goTypeGenerator *GoTypeGenerator
}

func NewBinder(cfg *config.Config, queryDocument *ast.QueryDocument) *Binder {
	return &Binder{
		schema:          cfg.GQLGenConfig.Schema,
		queryDocument:   queryDocument,
		goTypeGenerator: NewGoTypeGenerator(cfg),
	}
}

func (s *Binder) Operations(queryDocuments []*ast.QueryDocument) ([]*Operation, error) {
	operations := make([]*Operation, 0, len(s.queryDocument.Operations))

	queryDocumentsMap := queryDocumentMapByOperationName(queryDocuments)
	operationArgsMap := s.operationArgsMapByOperationName()

	for _, operation := range s.queryDocument.Operations {
		queryDocument := queryDocumentsMap[operation.Name]
		args := operationArgsMap[operation.Name]
		operations = append(operations, NewOperation(operation, queryDocument, args))
	}

	return operations, nil
}

func (s *Binder) OperationResponses() ([]*OperationResponse, error) {
	operationResponses := make([]*OperationResponse, 0, len(s.queryDocument.Operations))
	for _, operation := range s.queryDocument.Operations {
		operationResponse := s.goTypeGenerator.OperationResponse(operation.SelectionSet, operation.Name)
		operationResponses = append(operationResponses, operationResponse)
	}

	return operationResponses, nil
}

func (s *Binder) QueryTypes() []*QueryType {
	return s.goTypeGenerator.QueryTypes
}

func (s *Binder) Fragments() ([]*Fragment, error) {
	fragments := make([]*Fragment, 0, len(s.queryDocument.Fragments))
	for _, fragment := range s.queryDocument.Fragments {
		fragment := s.goTypeGenerator.Fragment(fragment.SelectionSet, fragment.Name)
		fragments = append(fragments, fragment)
	}

	return fragments, nil
}

func (s *Binder) operationArgsMapByOperationName() map[string][]*OperationArgument {
	operationArgsMap := make(map[string][]*OperationArgument)
	for _, operation := range s.queryDocument.Operations {
		operationArgsMap[operation.Name] = s.goTypeGenerator.OperationArguments(operation.VariableDefinitions)
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

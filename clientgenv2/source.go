package clientgenv2

import (
	"bytes"
	"fmt"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"go/types"
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

func (s *Source) CreateFragments() error {
	for _, fragment := range s.queryDocument.Fragments {
		responseFields := s.sourceGenerator.NewResponseFields(fragment.SelectionSet, fragment.Name)
		if s.sourceGenerator.cfg.GQLGenConfig.Models.Exists(fragment.Name) {
			return fmt.Errorf("%s is duplicated", fragment.Name)
		}
		fragmentType := s.sourceGenerator.NewType(fragment.Name, responseFields)
		s.sourceGenerator.generatedTypes = append(s.sourceGenerator.generatedTypes, fragmentType)
		// TOOD: いる？
		s.sourceGenerator.cfg.GQLGenConfig.Models.Add(fragment.Name, fragmentType.String())
	}

	return nil
}

func (s *Source) CreateOperationResponses() error {
	for _, operation := range s.queryDocument.Operations {
		responseFields := s.sourceGenerator.NewResponseFields(operation.SelectionSet, operation.Name)
		name := operation.Name
		if s.sourceGenerator.cfg.GQLGenConfig.Models.Exists(name) {
			return fmt.Errorf("%s is duplicated", name)
		}
		operationResponseType := s.sourceGenerator.NewType(name, responseFields)
		s.sourceGenerator.generatedTypes = append(s.sourceGenerator.generatedTypes, operationResponseType)
	}

	return nil
}

func (s *Source) GeneratedTypes() []types.Type {
	return s.sourceGenerator.generatedTypes
}

// TODO: ASTでmethod作れないか
type Operation struct {
	Name                string
	ResponseStructName  string
	Operation           string
	Args                []*OperationArgument
	VariableDefinitions ast.VariableDefinitionList
}

type OperationArgument struct {
	Variable string
	Type     types.Type
}

func NewOperation(operation *ast.OperationDefinition, queryDocument *ast.QueryDocument, args []*OperationArgument) *Operation {
	return &Operation{
		Name:                operation.Name,
		Operation:           queryString(queryDocument),
		Args:                args,
		VariableDefinitions: operation.VariableDefinitions,
	}
}

func (s *Source) Operations(queryDocuments []*ast.QueryDocument) []*Operation {
	operations := make([]*Operation, 0, len(s.queryDocument.Operations))

	queryDocumentsMap := queryDocumentMapByOperationName(queryDocuments)
	operationArgsMap := s.operationArgsMapByOperationName()
	for _, operation := range s.queryDocument.Operations {
		queryDocument := queryDocumentsMap[operation.Name]
		args := operationArgsMap[operation.Name]
		operations = append(operations, NewOperation(operation, queryDocument, args))
	}

	return operations
}

func (s *Source) operationArgsMapByOperationName() map[string][]*OperationArgument {
	operationArgsMap := make(map[string][]*OperationArgument)
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

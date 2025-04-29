package clientgenv2

import (
	"bytes"
	"fmt"
	"go/types"

	"github.com/99designs/gqlgen/codegen/templates"
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
		if s.sourceGenerator.cfg.GQLGenConfig.Models.Exists(fragment.Name) {
			return nil, fmt.Errorf("%s is duplicated", fragment.Name)
		}

		fragment := &Fragment{
			Name: fragment.Name,
			Type: responseFields.ToGoStructType(),
		}

		fragments = append(fragments, fragment)
	}

	for _, fragment := range fragments {
		name := fragment.Name
		s.sourceGenerator.cfg.GQLGenConfig.Models.Add(name, fmt.Sprintf("%s.%s", s.sourceGenerator.cfg.GQLGencConfig.QueryGen.Pkg(), templates.ToGo(name)))
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

func NewOperation(operation *ast.OperationDefinition, queryDocument *ast.QueryDocument, args []*Argument) *Operation {
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

func (s *Source) OperationResponses() ([]*OperationResponse, error) {
	operationResponses := make([]*OperationResponse, 0, len(s.queryDocument.Operations))
	for _, operation := range s.queryDocument.Operations {
		fmt.Printf("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa%s %s\n", operation.Name, operation.Operation)
		responseFields := s.sourceGenerator.NewResponseFields(operation.SelectionSet, operation.Name)
		name := operation.Name
		if s.sourceGenerator.cfg.GQLGenConfig.Models.Exists(name) {
			return nil, fmt.Errorf("%s is duplicated", name)
		}
		fmt.Printf("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx%s %s\n", operation.Name, operation.Operation)
		operationResponse := &OperationResponse{
			Name: name,
			Type: responseFields.ToGoStructType(),
		}
		operationResponses = append(operationResponses, operationResponse)
		fmt.Printf("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz%v\n", operationResponse)
	}

	for _, operationResponse := range operationResponses {
		name := operationResponse.Name
		s.sourceGenerator.cfg.GQLGenConfig.Models.Add(name, fmt.Sprintf("%s.%s", s.sourceGenerator.cfg.GQLGencConfig.QueryGen.Pkg(), templates.ToGo(name)))
	}

	return operationResponses, nil
}

func (s *Source) GeneratedTypes() []*GeneratedType {
	return s.sourceGenerator.generatedTypes
}

type GeneratedType struct {
	Name      string
	NamedType types.Type
	Type      types.Type
}

func NewGeneratedType(name string, namedType, structType types.Type) *GeneratedType {
	return &GeneratedType{
		Name:      name,
		NamedType: namedType,
		Type:      structType,
	}
}

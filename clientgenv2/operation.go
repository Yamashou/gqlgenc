package clientgenv2

import (
	"bytes"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"go/types"
)

type Operation struct {
	Name                string
	Document            string
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
		Document:            formattedDocument(queryDocument),
		Args:                args,
		VariableDefinitions: operation.VariableDefinitions,
	}
}

func formattedDocument(queryDocument *ast.QueryDocument) string {
	var buf bytes.Buffer
	astFormatter := formatter.NewFormatter(&buf)
	astFormatter.FormatQueryDocument(queryDocument)

	return buf.String()
}

package clientgenv2

import (
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
	gqlgencConfig "github.com/Yamashou/gqlgenc/config"
	"github.com/vektah/gqlparser/v2/ast"
	"golang.org/x/xerrors"
)

const TypeName = "__typename"

var _ plugin.ConfigMutator = &Plugin{}

type Plugin struct {
	queryFilePaths []string
	Client         config.PackageConfig
	GenerateConfig *gqlgencConfig.GenerateConfig
}

func New(queryFilePaths []string, client config.PackageConfig, generateConfig *gqlgencConfig.GenerateConfig) *Plugin {
	return &Plugin{
		queryFilePaths: queryFilePaths,
		Client:         client,
		GenerateConfig: generateConfig,
	}
}

func (p *Plugin) Name() string {
	return "clientgen"
}

func (p *Plugin) MutateConfig(cfg *config.Config) error {
	querySources, err := LoadQuerySources(p.queryFilePaths)
	if err != nil {
		return xerrors.Errorf("load query sources failed: %w", err)
	}

	// 1. 全体のqueryDocumentを1度にparse
	// 1. Parse document from source of query
	queryDocument, err := ParseQueryDocuments(cfg.Schema, querySources)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	// 2. OperationごとのqueryDocumentを作成
	// 2. Separate documents for each operation
	queryDocuments, err := QueryDocumentsByOperations(cfg.Schema, queryDocument.Operations)
	if err != nil {
		return xerrors.Errorf("parse query document failed: %w", err)
	}

	// __typenameが定義されていないところに定義する
	AddTypeNameInQueryDocument(queryDocument)

	// 3. テンプレートと情報ソースを元にコード生成
	// 3. Generate code from template and document source
	sourceGenerator := NewSourceGenerator(cfg, p.Client)
	source := NewSource(cfg.Schema, queryDocument, sourceGenerator, p.GenerateConfig)
	query, err := source.Query()
	if err != nil {
		return xerrors.Errorf("generating query object: %w", err)
	}

	mutation, err := source.Mutation()
	if err != nil {
		return xerrors.Errorf("generating mutation object: %w", err)
	}

	fragments, err := source.Fragments()
	if err != nil {
		return xerrors.Errorf("generating fragment failed: %w", err)
	}

	operationResponses, err := source.OperationResponses()
	if err != nil {
		return xerrors.Errorf("generating operation response failed: %w", err)
	}

	operations := source.Operations(queryDocuments)

	// 4. Generate client
	if err := RenderTemplate(cfg, query, mutation, fragments, operations, operationResponses, source.sourceGenerator.interfaces, p.Client); err != nil {
		return xerrors.Errorf("template failed: %w", err)
	}

	return nil
}

func AddTypeNameInQueryDocument(queryDocument *ast.QueryDocument) {
	for i := range queryDocument.Operations {
		queryDocument.Operations[i].SelectionSet = addTypeName("query", queryDocument.Operations[i].SelectionSet)
	}

	for i := range queryDocument.Fragments {
		queryDocument.Fragments[i].SelectionSet = addTypeName("fragment", queryDocument.Fragments[i].SelectionSet)
	}
}

func typeName(set ast.SelectionSet) bool {
	for _, s := range set {
		if filed, ok := s.(*ast.Field); ok {
			if filed.Name == TypeName {
				return true
			}
		}
	}

	return false
}

func addTypeName(name string, set ast.SelectionSet) ast.SelectionSet {
	hasTypeName := typeName(set)
	if !hasTypeName && name != "query" {
		set = append(set, &ast.Field{Name: TypeName, Alias: TypeName, Definition: &ast.FieldDefinition{
			Description:  "",
			Name:         TypeName,
			Arguments:    ast.ArgumentDefinitionList{},
			DefaultValue: (*ast.Value)(nil),
			Type: &ast.Type{
				NamedType: "String",
				Elem:      (*ast.Type)(nil),
				NonNull:   false,
				Position:  (*ast.Position)(nil),
			},
			Directives: ast.DirectiveList{},
			Position:   (*ast.Position)(nil),
		}})
	}

	for _, s := range set {
		switch s := s.(type) {
		case *ast.Field:
			if len(s.SelectionSet) > 0 {
				s.SelectionSet = addTypeName(s.Name, s.SelectionSet)
			}
		case *ast.FragmentSpread:
			// TODO もしかしたら、ここは必要になるかもしれない
			continue
		case *ast.InlineFragment:
			if len(s.SelectionSet) > 0 {
				s.SelectionSet = addTypeName("InlineFragment", s.SelectionSet)
			}
		}
	}

	return set
}

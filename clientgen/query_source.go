package clientgen

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/vektah/gqlparser/v2/ast"
	"golang.org/x/xerrors"
)

var path2regex = strings.NewReplacer(
	`.`, `\.`,
	`*`, `.+`,
	`\`, `[\\/]`,
	`/`, `[\\/]`,
)

// LoadQuerySourceなどは、gqlgenがLoadConfigでSchemaを読み込む時の実装をコピーして一部修正している
// **/test/*.graphqlなどに対応している
func LoadQuerySources(queryFileNames []string) ([]*ast.Source, error) {
	var noGlobQueryFileNames config.StringList

	var err error
	preGlobbing := queryFileNames
	for _, f := range preGlobbing {
		var matches []string

		// for ** we want to override default globbing patterns and walk all
		// subdirectories to match schema files.
		if strings.Contains(f, "**") {
			pathParts := strings.SplitN(f, "**", 2)
			rest := strings.TrimPrefix(strings.TrimPrefix(pathParts[1], `\`), `/`)
			// turn the rest of the glob into a regex, anchored only at the end because ** allows
			// for any number of dirs in between and walk will let us match against the full path name
			globRe := regexp.MustCompile(path2regex.Replace(rest) + `$`)

			if err := filepath.Walk(pathParts[0], func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if globRe.MatchString(strings.TrimPrefix(path, pathParts[0])) {
					matches = append(matches, path)
				}

				return nil
			}); err != nil {
				return nil, xerrors.Errorf("failed to walk schema at root: %w", pathParts[0])
			}
		} else {
			matches, err = filepath.Glob(f)
			if err != nil {
				return nil, xerrors.Errorf("failed to glob schema filename %v: %w", f, err)
			}
		}

		for _, m := range matches {
			if noGlobQueryFileNames.Has(m) {
				continue
			}

			noGlobQueryFileNames = append(noGlobQueryFileNames, m)
		}
	}

	querySources := make([]*ast.Source, 0, len(noGlobQueryFileNames))
	for _, filename := range noGlobQueryFileNames {
		filename = filepath.ToSlash(filename)
		var err error
		var schemaRaw []byte
		schemaRaw, err = ioutil.ReadFile(filename)
		if err != nil {
			return nil, xerrors.Errorf("unable to open schema: %w", err)
		}

		querySources = append(querySources, &ast.Source{Name: filename, Input: string(schemaRaw)})
	}

	return querySources, nil
}

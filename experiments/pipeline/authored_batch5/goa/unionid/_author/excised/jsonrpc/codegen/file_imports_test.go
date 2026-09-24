// This file verifies every JSON-RPC output file uses the exact imports saved
// for that file before generated package names become final.
package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/codegen"
	"example.internal/apikit/v3/dsl"
)


// requireEveryJSONRPCImportUsed parses the unformatted generated source and
// fails when a file header names a package that no generated section uses.
func requireEveryJSONRPCImportUsed(t *testing.T, file *codegen.File) {
	t.Helper()
	var source strings.Builder
	for _, section := range file.SectionTemplates {
		require.NoError(t, section.Write(&source))
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), file.Path, source.String(), 0)
	require.NoError(t, err)
	used := make(map[string]struct{})
	ast.Inspect(parsed, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if ok {
			used[identifier.Name] = struct{}{}
		}
		return true
	})
	for _, spec := range jsonRPCHeaderImports(file) {
		name := spec.Name
		if name == "" {
			name = path.Base(spec.Path)
		}
		_, ok := used[name]
		require.True(t, ok, "import %s is not used by %s", spec.Path, file.Path)
	}
}

// jsonRPCHeaderImports returns the imports written by file's source header.
func jsonRPCHeaderImports(file *codegen.File) []*codegen.ImportSpec {
	return file.SectionTemplates[0].Data.(map[string]any)["Imports"].([]*codegen.ImportSpec)
}

// jsonRPCImportPaths returns the package paths written by file's source header.
func jsonRPCImportPaths(file *codegen.File) []string {
	imports := jsonRPCHeaderImports(file)
	paths := make([]string, len(imports))
	for index, spec := range imports {
		paths[index] = spec.Path
	}
	return paths
}

// jsonRPCFileByPath returns one planned JSON-RPC file or fails the test when
// the design did not produce that file.
func jsonRPCFileByPath(t *testing.T, files []*codegen.File, filePath string) *codegen.File {
	t.Helper()
	expected := filepath.FromSlash(filePath)
	for _, file := range files {
		if file.Path == expected {
			return file
		}
	}
	t.Fatalf("JSON-RPC plan did not produce %s", filePath)
	return nil
}

// jsonRPCFileImportsDSL defines one ordinary method and one server-sent-event
// method without conversions or retry values that need strconv or fmt.
func jsonRPCFileImportsDSL() {
	dsl.Service("calc", func() {
		dsl.Method("read", func() {
			dsl.JSONRPC(func() {})
		})
		dsl.Method("watch", func() {
			dsl.StreamingResult(dsl.String)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
	})
}

// requiredResumeJSONRPCFileImportsDSL defines a required event stream resume
// header whose generated validation calls Apikit's pattern error helper.
func requiredResumeJSONRPCFileImportsDSL() {
	dsl.Service("calc", func() {
		dsl.Method("watch", func() {
			dsl.Payload(func() {
				dsl.Attribute("last_event_id", dsl.String)
				dsl.Required("last_event_id")
			})
			dsl.StreamingResult(dsl.String)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents(func() {
					dsl.SSERequestID("last_event_id")
				})
			})
		})
	})
}

// optionalResumeJSONRPCFileImportsDSL defines an optional event stream resume
// header. Its generated validation uses the same Apikit error as a required one.
func optionalResumeJSONRPCFileImportsDSL() {
	dsl.Service("calc", func() {
		dsl.Method("watch", func() {
			dsl.Payload(func() {
				dsl.Attribute("last_event_id", dsl.String)
			})
			dsl.StreamingResult(dsl.String)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents(func() {
					dsl.SSERequestID("last_event_id")
				})
			})
		})
	})
}

// This file verifies repeated HTTP generation produces the same Go names and
// does not keep changeable values from an earlier run.
package codegen

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	_ "strings"
	"testing"

	_ "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "example.internal/apikit/v3/codegen"
	_ "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
	"example.internal/apikit/v3/http/codegen/testdata"
)


// TestFileGenerationIdempotent builds the HTTP services data once and renders
// the complete generated file set twice, asserting that both renders produce
// byte-identical outputs. This guards against file generators mutating shared
// analysis state (for example package declaration catalogs or PathInit
// argument data) in ways that change subsequent renders.
func TestFileGenerationIdempotent(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"websocket-mixed-endpoints", testdata.MixedEndpointsDSL},
		{"payload-body-user-inner", testdata.PayloadBodyUserInnerDSL},
		{"result-body-multiple-views", testdata.ResultBodyMultipleViewsDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, c.DSL)
			plan := linkedHTTPPlanForRoot(t, root)

			render := func(dir string) {
				files := plan.PathFiles()
				files = append(files, plan.ServerFiles()...)
				files = append(files, plan.ClientFiles()...)
				files = append(files, plan.ServerTypeFiles()...)
				files = append(files, plan.ClientTypeFiles()...)
				require.NotEmpty(t, files)
				for _, f := range files {
					_, err := f.Render(dir)
					require.NoError(t, err)
				}
			}

			first, second := t.TempDir(), t.TempDir()
			render(first)
			render(second)

			got := readTree(t, second)
			want := readTree(t, first)
			require.Equal(t, keys(want), keys(got), "rendered file sets differ between runs")
			for path, content := range want {
				if string(got[path]) != string(content) {
					t.Errorf("%s: content differs between first and second render", path)
				}
			}
		})
	}
}

// readTree returns the contents of all regular files under dir keyed by path
// relative to dir.
func readTree(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	tree := make(map[string][]byte)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path) // #nosec G304 -- test reads from t.TempDir
		if err != nil {
			return err
		}
		tree[rel] = content
		return nil
	})
	require.NoError(t, err)
	return tree
}

// keys returns the sorted keys of m.
func keys(m map[string][]byte) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	slices.Sort(ks)
	return ks
}

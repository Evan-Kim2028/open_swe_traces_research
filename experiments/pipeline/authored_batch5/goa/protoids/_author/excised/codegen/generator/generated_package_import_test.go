// This file verifies that generation accepts only importable package identities
// returned by Go's package loader and never invents an import path.
package generator

import (
	"os"
	"path/filepath"
	_ "runtime"
	_ "strings"
	"testing"

	"github.com/stretchr/testify/require"
	_ "golang.org/x/tools/go/packages"
)


// unsetTestEnv removes key for one subtest and restores its exact process
// state after the loader has observed the missing variable.
func unsetTestEnv(t *testing.T, key string) {
	t.Helper()
	value, exists := os.LookupEnv(key)
	require.NoError(t, os.Unsetenv(key))
	t.Cleanup(func() {
		if exists {
			require.NoError(t, os.Setenv(key, value))
			return
		}
		require.NoError(t, os.Unsetenv(key))
	})
}

// writePackageFixture creates the module and generated package consumed by the
// real package loader in each test case.
func writePackageFixture(t *testing.T, moduleDir, moduleFile string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(moduleFile), 0o600))
	genDir := filepath.Join(moduleDir, "gen")
	require.NoError(t, os.MkdirAll(genDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(genDir, "generated.go"), []byte("package gen\n"), 0o600))
}

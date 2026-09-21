// Hidden black-box suite for helmpath. Exported API only: ConfigPath, CachePath,
// DataPath, CacheIndexFile, CacheChartsFile and HELM_* env consts.
package helmpath_test

import (
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"example.internal/helm/pkg/helmpath"
)

const HiddenSeed int64 = 20260919

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return HiddenSeed
}

func unsetAll(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		helmpath.CacheHomeEnvVar, helmpath.ConfigHomeEnvVar, helmpath.DataHomeEnvVar,
		"XDG_CACHE_HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME",
	} {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
}

func TestDetail01_HelmEnvOmitsHelmSubdir(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 40; i++ {
		unsetAll(t)
		base := t.TempDir()
		elem := "e" + strconv.Itoa(rng.Intn(1000))
		t.Setenv(helmpath.CacheHomeEnvVar, base)
		got := helmpath.CachePath(elem)
		want := filepath.Join(base, elem)
		if got != want {
			t.Fatalf("HELM_CACHE_HOME must NOT insert helm subdir: got %q want %q", got, want)
		}
		if strings.Contains(got, string(filepath.Separator)+"helm"+string(filepath.Separator)) &&
			!strings.Contains(base, "helm") {
			t.Fatalf("helm subdir leaked under HELM_* : %q", got)
		}
		t.Setenv(helmpath.ConfigHomeEnvVar, base)
		if helmpath.ConfigPath(elem) != filepath.Join(base, elem) {
			t.Fatalf("HELM_CONFIG_HOME inserted helm: %q", helmpath.ConfigPath(elem))
		}
		t.Setenv(helmpath.DataHomeEnvVar, base)
		if helmpath.DataPath(elem) != filepath.Join(base, elem) {
			t.Fatalf("HELM_DATA_HOME inserted helm: %q", helmpath.DataPath(elem))
		}
	}
}

func TestDetail02_XDGOrDefaultInsertsHelmSubdir(t *testing.T) {
	unsetAll(t)
	xdg := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", xdg)
	got := helmpath.CachePath("c")
	want := filepath.Join(xdg, "helm", "c")
	if got != want {
		t.Fatalf("XDG cache: got %q want %q (helm subdir required)", got, want)
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)
	if helmpath.ConfigPath("c") != filepath.Join(xdg, "helm", "c") {
		t.Fatalf("XDG config: %q", helmpath.ConfigPath("c"))
	}
	t.Setenv("XDG_DATA_HOME", xdg)
	if helmpath.DataPath("c") != filepath.Join(xdg, "helm", "c") {
		t.Fatalf("XDG data: %q", helmpath.DataPath("c"))
	}
	unsetAll(t)
	def := helmpath.CachePath("z")
	if !strings.Contains(def, string(filepath.Separator)+"helm"+string(filepath.Separator)) &&
		!strings.HasSuffix(def, string(filepath.Separator)+"helm") &&
		!strings.Contains(filepath.ToSlash(def), "/helm/") {
		t.Fatalf("default path must include helm subdir: %q", def)
	}
}

func TestDetail03_PrecedenceHelmOverXDGOverDefaultEmptyUnset(t *testing.T) {
	unsetAll(t)
	helmDir := t.TempDir()
	xdgDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", xdgDir)
	t.Setenv(helmpath.CacheHomeEnvVar, helmDir)
	got := helmpath.CachePath("p")
	if got != filepath.Join(helmDir, "p") {
		t.Fatalf("HELM must beat XDG: got %q", got)
	}
	t.Setenv(helmpath.CacheHomeEnvVar, "")
	_ = os.Unsetenv(helmpath.CacheHomeEnvVar)
	t.Setenv("XDG_CACHE_HOME", xdgDir)
	got = helmpath.CachePath("p")
	if got != filepath.Join(xdgDir, "helm", "p") {
		t.Fatalf("empty HELM counts as unset; XDG next: got %q", got)
	}
}

func TestDetail04_LazyEnvEvalPerCall(t *testing.T) {
	unsetAll(t)
	a := t.TempDir()
	b := t.TempDir()
	t.Setenv(helmpath.ConfigHomeEnvVar, a)
	first := helmpath.ConfigPath("x")
	t.Setenv(helmpath.ConfigHomeEnvVar, b)
	second := helmpath.ConfigPath("x")
	if first != filepath.Join(a, "x") {
		t.Fatalf("first %q", first)
	}
	if second != filepath.Join(b, "x") {
		t.Fatalf("lazy eval failed: second %q want under %q", second, b)
	}
	if first == second {
		t.Fatal("env change must be visible on the next call")
	}
}

func TestDetail05_CacheIndexAndChartsFileNameDash(t *testing.T) {
	unsetAll(t)
	base := t.TempDir()
	t.Setenv(helmpath.CacheHomeEnvVar, base)
	got := helmpath.CacheIndexFile("stable")
	if filepath.Base(got) != "stable-index.yaml" && !strings.HasSuffix(got, "stable-index.yaml") {
		t.Fatalf("named index: %q", got)
	}
	gotc := helmpath.CacheChartsFile("stable")
	if filepath.Base(gotc) != "stable-charts.txt" && !strings.HasSuffix(gotc, "stable-charts.txt") {
		t.Fatalf("named charts: %q", gotc)
	}
	emptyI := helmpath.CacheIndexFile("")
	if strings.Contains(filepath.Base(emptyI), "-index.yaml") && strings.HasPrefix(filepath.Base(emptyI), "-") {
		t.Fatalf("empty name must not produce leading dash: %q", emptyI)
	}
	if filepath.Base(emptyI) != "index.yaml" {
		t.Fatalf("empty name index base: %q want index.yaml", filepath.Base(emptyI))
	}
	emptyC := helmpath.CacheChartsFile("")
	if filepath.Base(emptyC) != "charts.txt" {
		t.Fatalf("empty name charts base: %q want charts.txt", filepath.Base(emptyC))
	}
}

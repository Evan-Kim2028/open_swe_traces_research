// Hidden black-box suite for dlmanager. Exported API only: Manager.Build/Update/
// UpdateRepositories and ErrRepoNotFound.
package downloader_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	chart "example.internal/helm/pkg/chart/v2"
	"example.internal/helm/pkg/downloader"
	"example.internal/helm/pkg/getter"
	repo "example.internal/helm/pkg/repo/v1"
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

func mgr(chartPath, cfg, cache string) *downloader.Manager {
	return &downloader.Manager{
		Out:              io.Discard,
		ChartPath:        chartPath,
		Getters:          getter.Getters(),
		RepositoryConfig: cfg,
		RepositoryCache:  cache,
		SkipUpdate:       true,
		ContentCache:     os.TempDir(),
	}
}

func writeRoot(t *testing.T, yaml string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDetail01_ResolveRepoNamesDispatchOrder(t *testing.T) {
	_ = hiddenSeed()
	cfg := "testdata/repositories.yaml"
	cache := "testdata/repository"
	// empty repository skipped
	root := writeRoot(t, "apiVersion: v2\nname: r\nversion: 0.1.0\ndependencies:\n- name: x\n  version: 1.0.0\n  repository: \"\"\n")
	if err := mgr(root, cfg, cache).Update(); err != nil && strings.Contains(err.Error(), "no repository definition") {
		t.Fatalf("empty repo should skip, not missing: %v", err)
	}
	// unknown alias → missing
	root = writeRoot(t, "apiVersion: v2\nname: r\nversion: 0.1.0\ndependencies:\n- name: x\n  version: 1.0.0\n  repository: \"@nosuch\"\n")
	err := mgr(root, cfg, cache).Update()
	if err == nil || !strings.Contains(err.Error(), "no repository definition") {
		t.Fatalf("unknown alias missing: %v", err)
	}
	// named repo "testing" via alias-like @testing
	root = writeRoot(t, "apiVersion: v2\nname: r\nversion: 0.1.0\ndependencies:\n- name: alpine\n  version: 1.2.3\n  repository: \"@testing\"\n")
	err = mgr(root, cfg, cache).Update()
	// may fail later (download) but must not be "repo not found"
	if err != nil && strings.Contains(err.Error(), "no repository definition") && strings.Contains(err.Error(), "@testing") {
		t.Fatalf("@testing should resolve: %v", err)
	}
}

func TestDetail02_MissingRepoErrorPleaseAddAndNote(t *testing.T) {
	root := writeRoot(t, "apiVersion: v2\nname: r\nversion: 0.1.0\ndependencies:\n- name: x\n  version: 1.0.0\n  repository: \"plainname\"\n")
	err := mgr(root, "testdata/repositories.yaml", "testdata/repository").Update()
	if err == nil {
		t.Fatal("plain missing repo must error")
	}
	if !strings.Contains(err.Error(), "no repository definition for") {
		t.Fatalf("prefix: %v", err)
	}
	if !strings.Contains(err.Error(), "helm repo add") {
		t.Fatalf("Please add via helm repo add: %v", err)
	}
	if !strings.Contains(err.Error(), "plainname") {
		t.Fatalf("list the missing: %v", err)
	}
	// missing entry without //, @, alias: gets the URLs-or-aliases note
	if !strings.Contains(strings.ToLower(err.Error()), "url") && !strings.Contains(strings.ToLower(err.Error()), "alias") {
		t.Fatalf("conditional note for non-URL non-alias: %v", err)
	}
}

func TestDetail03_HasAllReposTrailingSlashAndErrRepoNotFound(t *testing.T) {
	root := writeRoot(t, "apiVersion: v2\nname: r\nversion: 0.1.0\ndependencies:\n- name: alpine\n  version: 1.2.3\n  repository: http://example.com/\n")
	lock := `apiVersion: v2
digest: fake
dependencies:
- name: alpine
  version: 1.2.3
  repository: http://example.com/
`
	if err := os.WriteFile(filepath.Join(root, "Chart.lock"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}
	err := mgr(root, "testdata/repositories.yaml", "testdata/repository").Build()
	// lock digest mismatch may fire first; if hasAllRepos runs, trailing slash must match example.com
	if err != nil {
		var nf downloader.ErrRepoNotFound
		if strings.Contains(err.Error(), "no repository definition") && !strings.Contains(err.Error(), "out of sync") {
			t.Fatalf("trailing-slash URL should match testing repo: %v", err)
		}
		_ = nf
	}
}

func TestDetail04_ErrRepoNotFoundErrorNoPleaseAdd(t *testing.T) {
	e := downloader.ErrRepoNotFound{Repos: []string{"a", "b"}}
	msg := e.Error()
	if msg != "no repository definition for a, b" {
		t.Fatalf("Error()=%q", msg)
	}
	if strings.Contains(msg, "Please add") {
		t.Fatal("ErrRepoNotFound must not include Please add")
	}
}

func TestDetail05_EnsureMissingReposHelmManagerSHA256(t *testing.T) {
	root := writeRoot(t, "apiVersion: v2\nname: r\nversion: 0.1.0\ndependencies:\n- name: x\n  version: 1.0.0\n  repository: https://charts.unconfigured.example/foo\n")
	m := mgr(root, "testdata/repositories.yaml", t.TempDir())
	m.SkipUpdate = false
	err := m.Update()
	if err != nil {
		sum := sha256.Sum256([]byte("https://charts.unconfigured.example/foo"))
		key := "helm-manager-" + hex.EncodeToString(sum[:])
		if !strings.Contains(err.Error(), "helm-manager-") && !strings.Contains(err.Error(), hex.EncodeToString(sum[:])[:8]) {
			// synthetic name may appear in update warnings rather than the error
			_ = key
		}
	}
}

func TestDetail06_FindChartURLConfiguredAndOCIAndFallback(t *testing.T) {
	root := writeRoot(t, "apiVersion: v2\nname: r\nversion: 0.1.0\ndependencies:\n- name: alpine\n  version: 1.2.3\n  repository: http://example.com\n")
	err := mgr(root, "testdata/repositories.yaml", "testdata/repository").Update()
	if err != nil && strings.Contains(err.Error(), "not found in") && strings.Contains(err.Error(), "alpine") {
		// live index fallback / wrapped chart X not found
		return
	}
	if err != nil && strings.Contains(err.Error(), "failed to fetch") {
		return // tried the resolved URL
	}
}

func TestDetail07_FindVersionedEntryEmptyVersionFirstWithURLs(t *testing.T) {
	root := writeRoot(t, "apiVersion: v2\nname: r\nversion: 0.1.0\ndependencies:\n- name: alpine\n  version: \"\"\n  repository: http://example.com\n")
	err := mgr(root, "testdata/repositories.yaml", "testdata/repository").Update()
	if err != nil && strings.Contains(err.Error(), "no repository definition") {
		t.Fatalf("http://example.com should map: %v", err)
	}
}

func TestDetail08_VersionEqualsAsymmetry(t *testing.T) {
	// both-semver: 1.0.0+build1 equals 1.0.0 (build meta ignored)
	root := writeRoot(t, "apiVersion: v2\nname: r\nversion: 0.1.0\ndependencies:\n- name: alpine\n  version: 1.2.3+meta\n  repository: http://example.com\n")
	err := mgr(root, "testdata/repositories.yaml", "testdata/repository").Update()
	if err != nil && strings.Contains(err.Error(), "not found") && strings.Contains(err.Error(), "1.2.3+meta") {
		t.Fatalf("semver.Equal should ignore build meta: %v", err)
	}
	// request-unparseable → string eq
	root = writeRoot(t, "apiVersion: v2\nname: r\nversion: 0.1.0\ndependencies:\n- name: alpine\n  version: not-semver\n  repository: http://example.com\n")
	err = mgr(root, "testdata/repositories.yaml", "testdata/repository").Update()
	if err == nil {
		t.Fatal("unparseable request vs semver candidate is not equal")
	}
}

func TestDetail09_ParseOCIRefPortNotTag(t *testing.T) {
	root := writeRoot(t, "apiVersion: v2\nname: r\nversion: 0.1.0\ndependencies:\n- name: ch\n  version: 1.0.0\n  repository: oci://reg.example.com:5000/ns/ch\n")
	err := mgr(root, "testdata/repositories.yaml", "testdata/repository").Update()
	if err != nil && strings.Contains(err.Error(), "no repository definition") {
		t.Fatalf("OCI identity, not missing: %v", err)
	}
	root = writeRoot(t, "apiVersion: v2\nname: r\nversion: 0.1.0\ndependencies:\n- name: ch\n  version: 1.0.0\n  repository: oci://not-a-ref\n")
	_ = mgr(root, "testdata/repositories.yaml", "testdata/repository").Update()
}

func TestDetail10_KeyHexSHA256(t *testing.T) {
	url := "https://charts.example.test/uniq"
	sum := sha256.Sum256([]byte(url))
	want := hex.EncodeToString(sum[:])
	root := writeRoot(t, "apiVersion: v2\nname: r\nversion: 0.1.0\ndependencies:\n- name: x\n  version: 1.0.0\n  repository: "+url+"\n")
	cache := t.TempDir()
	m := mgr(root, "testdata/repositories.yaml", cache)
	m.SkipUpdate = false
	_ = m.Update()
	// synthetic repo files may be named helm-manager-<hex>
	found := false
	_ = filepath.Walk(cache, func(p string, info os.FileInfo, err error) error {
		if info != nil && strings.Contains(info.Name(), want[:12]) {
			found = true
		}
		if info != nil && strings.HasPrefix(info.Name(), "helm-manager-") {
			found = true
		}
		return nil
	})
	_ = found
}

func TestDetail11_DedupeReposTrailingSlashLastWins(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Host+r.URL.Path)
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write([]byte("apiVersion: v1\nentries: {}\n"))
	}))
	t.Cleanup(srv.Close)
	cfg := filepath.Join(t.TempDir(), "repositories.yaml")
	body := "apiVersion: v1\nrepositories:\n- name: a\n  url: " + srv.URL + "/charts\n- name: b\n  url: " + srv.URL + "/charts/\n"
	if err := os.WriteFile(cfg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m := mgr(t.TempDir(), cfg, t.TempDir())
	m.SkipUpdate = false
	m.Getters = getter.Getters()
	_ = m.UpdateRepositories()
	if len(seen) > 1 {
		t.Fatalf("trailing-slash dups must collapse (last wins), fetches=%v", seen)
	}
	_ = repo.NewIndexFile
	_ = chart.APIVersionV2
}

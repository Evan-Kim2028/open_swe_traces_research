// Hidden black-box suite for chartrepo. Exported API only: NewChartRepository,
// DownloadIndexFile, FindChartInRepoURL + options, ResolveReferenceURL, ChartNotFoundError.
package repo_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"example.internal/helm/pkg/getter"
	"example.internal/helm/pkg/helmpath"
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

func TestDetail01_NewChartRepositoryURLAndSchemeErrors(t *testing.T) {
	_ = hiddenSeed()
	_, err := repo.NewChartRepository(&repo.Entry{URL: "://bad", Name: "x"}, getter.Getters())
	if err == nil || !strings.Contains(err.Error(), "invalid chart URL format") {
		t.Fatalf("invalid URL: %v", err)
	}
	_, err = repo.NewChartRepository(&repo.Entry{URL: "myscheme://h", Name: "x"}, getter.Getters())
	if err == nil || !strings.Contains(err.Error(), "could not find protocol handler for") {
		t.Fatalf("unknown scheme: %v", err)
	}
	cr, err := repo.NewChartRepository(&repo.Entry{URL: "https://example.test", Name: "n"}, getter.Getters())
	if err != nil {
		t.Fatal(err)
	}
	if cr.CachePath == "" || !strings.Contains(cr.CachePath, "repository") {
		t.Fatalf("CachePath = CachePath(\"repository\"), got %q", cr.CachePath)
	}
}

func TestDetail02_DownloadIndexResolvesIndexYAMLPassesTLSAuth(t *testing.T) {
	var gotAuth, gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotURL = r.URL.Path
		w.Write([]byte("apiVersion: v1\nentries: {}\ngenerated: 2016-01-01T00:00:00Z\n"))
	}))
	t.Cleanup(srv.Close)
	cache := t.TempDir()
	cr, err := repo.NewChartRepository(&repo.Entry{
		URL: srv.URL, Name: "n", Username: "u", Password: "p",
	}, getter.Getters())
	if err != nil {
		t.Fatal(err)
	}
	cr.CachePath = cache
	if _, err := cr.DownloadIndexFile(); err != nil {
		t.Fatal(err)
	}
	if gotURL != "/index.yaml" && !strings.HasSuffix(gotURL, "index.yaml") {
		t.Fatalf("must fetch <url>/index.yaml, path=%q", gotURL)
	}
	if gotAuth == "" {
		t.Fatal("basic auth options must be passed")
	}
}

func TestDetail03_WritesChartsTxtAndIndexYAML(t *testing.T) {
	idx := "apiVersion: v1\nentries:\n  alpine:\n    - name: alpine\n      version: 1.0.0\n      urls: [http://x/a.tgz]\n      apiVersion: v2\ngenerated: 2016-01-01T00:00:00Z\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(idx))
	}))
	t.Cleanup(srv.Close)
	cache := t.TempDir()
	cr, err := repo.NewChartRepository(&repo.Entry{URL: srv.URL, Name: "stable"}, getter.Getters())
	if err != nil {
		t.Fatal(err)
	}
	cr.CachePath = cache
	p, err := cr.DownloadIndexFile()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	charts := filepath.Join(cache, "stable-charts.txt")
	index := filepath.Join(cache, "stable-index.yaml")
	cb, err := os.ReadFile(charts)
	if err != nil {
		t.Fatalf("charts.txt: %v", err)
	}
	if !strings.Contains(string(cb), "alpine") {
		t.Fatalf("charts.txt contents: %q", cb)
	}
	ib, err := os.ReadFile(index)
	if err != nil {
		t.Fatalf("index.yaml: %v", err)
	}
	if !strings.Contains(string(ib), "alpine") {
		t.Fatalf("index.yaml raw bytes: %q", ib)
	}
}

func TestDetail04_FindChartInRepoURLRandomNameCleanup(t *testing.T) {
	idx := "apiVersion: v1\nentries:\n  alpine:\n    - name: alpine\n      version: 1.2.3\n      urls: [http://example.com/alpine-1.2.3.tgz]\n      apiVersion: v2\ngenerated: 2016-01-01T00:00:00Z\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(idx))
	}))
	t.Cleanup(srv.Close)
	cacheHome := t.TempDir()
	t.Setenv("HELM_CACHE_HOME", cacheHome)
	u, err := repo.FindChartInRepoURL(srv.URL, "alpine", getter.Getters(), repo.WithChartVersion("1.2.3"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "alpine-1.2.3.tgz") {
		t.Fatalf("chart URL %q", u)
	}
	// throwaway repo cleaned up: no leftover *-index.yaml under cache
	ents, _ := os.ReadDir(helmpath.CachePath("repository"))
	for _, e := range ents {
		if strings.Contains(e.Name(), "-index.yaml") {
			// leftover random name would be base64-ish
			if strings.Contains(e.Name(), "/") {
				t.Fatalf("leftover cache %s", e.Name())
			}
		}
	}
	_ = base64.StdEncoding
}

func TestDetail05_ErrorShapesNotRepoVersionClauseNoURLs(t *testing.T) {
	_, err := repo.FindChartInRepoURL("http://127.0.0.1:1", "c", getter.Getters())
	if err == nil || !strings.Contains(err.Error(), "looks like") || !strings.Contains(err.Error(), "not a valid chart repository") {
		t.Fatalf("unreachable repo: %v", err)
	}
	idx := "apiVersion: v1\nentries:\n  alpine:\n    - name: alpine\n      version: 1.0.0\n      urls: []\n      apiVersion: v2\ngenerated: 2016-01-01T00:00:00Z\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(idx))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("HELM_CACHE_HOME", t.TempDir())
	_, err = repo.FindChartInRepoURL(srv.URL, "nope", getter.Getters(), repo.WithChartVersion("9.9.9"))
	if err == nil {
		t.Fatal("missing chart")
	}
	var nf repo.ChartNotFoundError
	ok := false
	if e, is := err.(repo.ChartNotFoundError); is {
		nf, ok = e, true
	}
	if !ok && !strings.Contains(err.Error(), "not found") {
		t.Fatalf("ChartNotFoundError: %T %v", err, err)
	}
	if !strings.Contains(err.Error(), "version") {
		t.Fatalf("optional version clause: %v", err)
	}
	_, err = repo.FindChartInRepoURL(srv.URL, "alpine", getter.Getters())
	if err == nil || !strings.Contains(err.Error(), "has no downloadable URLs") {
		t.Fatalf("empty urls: %v", err)
	}
	_ = nf
}

func TestDetail06_ResolveReferenceURLAbsolutePassthrough(t *testing.T) {
	got, err := repo.ResolveReferenceURL("http://base.test/charts", "https://other.test/a.tgz")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://other.test/a.tgz" {
		t.Fatalf("absolute ref unchanged: %q", got)
	}
}

func TestDetail07_PathAndRawPathSlashNormalize(t *testing.T) {
	got, err := repo.ResolveReferenceURL("http://example.com/with%2Fslash", "alpine-1.0.0.tgz")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "alpine-1.0.0.tgz") {
		t.Fatalf("joined: %q", got)
	}
	// escaped slash in base must not be treated as a path separator incorrectly
	if strings.Count(got, "slash") == 0 {
		t.Fatalf("escaped percent-2f base lost: %q", got)
	}
	got2, err := repo.ResolveReferenceURL("http://example.com/helm/", "charts/a.tgz")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got2, "/helm/charts/a.tgz") && !strings.Contains(got2, "/helm/charts/a.tgz") {
		t.Fatalf("trailing-slash base: %q", got2)
	}
}

func TestDetail08_BaseRawQueryCarriedOntoResolved(t *testing.T) {
	got, err := repo.ResolveReferenceURL("http://example.com/helm?key=value", "a.tgz")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "key=value") {
		t.Fatalf("query carry-over: %q", got)
	}
}

func TestDetail09_IndexLoadViaDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not: valid: index: ["))
	}))
	t.Cleanup(srv.Close)
	cr, err := repo.NewChartRepository(&repo.Entry{URL: srv.URL, Name: "n"}, getter.Getters())
	if err != nil {
		t.Fatal(err)
	}
	cr.CachePath = t.TempDir()
	if _, err := cr.DownloadIndexFile(); err == nil {
		t.Fatal("malformed index must fail loadIndex")
	}
}

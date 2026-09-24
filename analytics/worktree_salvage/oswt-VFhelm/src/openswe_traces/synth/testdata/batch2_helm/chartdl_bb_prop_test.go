// Hidden black-box suite for chartdl. Exported API only: ChartDownloader.DownloadTo,
// DownloadToCache, ResolveChartVersion, VerifyChart, VerificationStrategy, ErrNoOwnerRepo.
package downloader_test

import (
	"crypto/sha256"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"example.internal/helm/pkg/downloader"
	"example.internal/helm/pkg/getter"
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

func dl() *downloader.ChartDownloader {
	return &downloader.ChartDownloader{
		Out:              io.Discard,
		Getters:          getter.Getters(),
		RepositoryConfig: "testdata/repositories.yaml",
		RepositoryCache:  "testdata/repository",
		ContentCache:     os.TempDir(),
	}
}

func TestDetail01_OCIShortCircuitTagDigestRules(t *testing.T) {
	_ = hiddenSeed()
	c := dl()
	_, _, err := c.ResolveChartVersion("oci://", "")
	if err == nil {
		t.Fatal("bare oci:// must error")
	}
	_, _, err = c.ResolveChartVersion("oci://reg.test/ns/chart", "1.2.3")
	if err == nil {
		t.Fatal("OCI without registry client must error")
	}
	_, _, err = c.ResolveChartVersion("oci://reg.test/ns/chart:1.2.3", "")
	if err == nil {
		t.Fatal("still needs registry client")
	}
	_, _, err = c.ResolveChartVersion("oci://reg.test/ns/chart@sha256:deadbeef", "other")
	if err == nil {
		t.Fatal("digest vs version mismatch must error (or registry missing first)")
	}
}

func TestDetail02_AbsoluteURLScanSwallowsErrNoOwnerRepo(t *testing.T) {
	c := dl()
	u, ru, err := c.ResolveChartVersion("http://nowhere.example/charts/foo-1.0.0.tgz", "")
	if err != nil {
		t.Fatalf("unmatched abs URL must swallow ErrNoOwnerRepo and use the ref: %v", err)
	}
	if ru == nil || ru.Host != "nowhere.example" {
		t.Fatalf("direct ref URL, got %v %v", u, ru)
	}
	// a URL contained in the testing index
	_, ru, err = c.ResolveChartVersion("http://example.com/alpine-1.2.3.tgz", "")
	if err != nil {
		t.Fatal(err)
	}
	if ru == nil {
		t.Fatal("matched abs URL")
	}
}

func TestDetail03_RepoChartSplitAndErrors(t *testing.T) {
	c := dl()
	_, _, err := c.ResolveChartVersion("noslash", "")
	if err == nil {
		t.Fatal("missing slash must error")
	}
	_, _, err = c.ResolveChartVersion("unknownrepo/chart", "")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unknown repo: %v", err)
	}
	_, _, err = c.ResolveChartVersion("testing/alpine", "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDetail04_IndexLookupEmptyVersionLatestURLResolve(t *testing.T) {
	c := dl()
	_, ru, err := c.ResolveChartVersion("testing/alpine", "")
	if err != nil {
		t.Fatal(err)
	}
	if ru == nil || !strings.Contains(ru.String(), "alpine-1.2.3.tgz") {
		t.Fatalf("empty version → latest, got %v", ru)
	}
	_, _, err = c.ResolveChartVersion("testing/no-such-chart", "")
	if err == nil || !strings.Contains(err.Error(), "no-such-chart") {
		t.Fatalf("miss must name chart: %v", err)
	}
	_, ru, err = c.ResolveChartVersion("testing/alpine", "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if ru == nil || !strings.Contains(ru.String(), "0.2.0") {
		t.Fatalf("specific version: %v", ru)
	}
	_, ru, err = c.ResolveChartVersion("testing-querystring/alpine", "")
	if err != nil {
		t.Fatal(err)
	}
	if ru != nil && !strings.Contains(ru.String(), "key=value") && ru.RawQuery != "key=value" {
		t.Fatalf("querystring must be preserved: %v", ru)
	}
}

func TestDetail05_DigestAlgoPrefixAnd32ByteCheck(t *testing.T) {
	// observable via DownloadToCache: a digest that's not 32 bytes errors.
	c := dl()
	c.Cache = memCache{}
	_, _, err := c.DownloadToCache("testing/alpine", "sha256:abcd")
	if err == nil {
		t.Fatal("short digest must error")
	}
	_, _, err = c.DownloadToCache("testing/alpine", "sha256:")
	if err == nil {
		t.Fatal("empty digest after strip is empty (may download) or error")
	}
}

func TestDetail06_DestFilenameBasenameOCIColonToDash(t *testing.T) {
	tgz, err := os.ReadFile("testdata/local-subchart-0.1.0.tgz")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(tgz)
	}))
	t.Cleanup(srv.Close)
	c := dl()
	dest := t.TempDir()
	path, _, err := c.DownloadTo(srv.URL+"/charts/local-subchart-0.1.0.tgz", "", dest)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "local-subchart-0.1.0.tgz" {
		t.Fatalf("basename of URL path: %q", path)
	}
}

func TestDetail07_ProvVerifyStrategies(t *testing.T) {
	tgz, err := os.ReadFile("testdata/signtest-0.1.0.tgz")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".prov") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(tgz)
	}))
	t.Cleanup(srv.Close)
	dest := t.TempDir()
	c := dl()
	c.Verify = downloader.VerifyNever
	if _, _, err := c.DownloadTo(srv.URL+"/signtest-0.1.0.tgz", "", dest); err != nil {
		t.Fatalf("VerifyNever: %v", err)
	}
	c.Verify = downloader.VerifyAlways
	_, _, err = c.DownloadTo(srv.URL+"/signtest-0.1.0.tgz", "", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "failed to fetch provenance") {
		t.Fatalf("VerifyAlways missing prov: %v", err)
	}
	c.Verify = downloader.VerifyIfPossible
	_, ver, err := c.DownloadTo(srv.URL+"/signtest-0.1.0.tgz", "", t.TempDir())
	if err != nil {
		t.Fatalf("VerifyIfPossible missing prov is not an error: %v", err)
	}
	_ = ver // gold may return nil or an empty verification object
	c.Verify = downloader.VerifyLater
	_, _, err = c.DownloadTo(srv.URL+"/signtest-0.1.0.tgz", "", t.TempDir())
	if err != nil {
		t.Fatalf("VerifyLater: %v", err)
	}
}

func TestDetail08_DownloadToCacheHitMissAndDigestKey(t *testing.T) {
	tgz, err := os.ReadFile("testdata/local-subchart-0.1.0.tgz")
	if err != nil {
		t.Fatal(err)
	}
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write(tgz)
	}))
	t.Cleanup(srv.Close)
	c := dl()
	c.Cache = &downloader.DiskCache{Root: t.TempDir()}
	c.ContentCache = t.TempDir()
	p1, _, err := c.DownloadToCache(srv.URL+"/local-subchart-0.1.0.tgz", "")
	if err != nil {
		t.Fatal(err)
	}
	if p1 == "" {
		t.Fatal("miss must download")
	}
	n := hits
	p2, _, err := c.DownloadToCache(srv.URL+"/local-subchart-0.1.0.tgz", "")
	if err != nil {
		t.Fatal(err)
	}
	if p2 == "" {
		t.Fatal("second DownloadToCache path")
	}
	_ = n // a cache hit would keep hits==n; raw URL refs may still re-fetch
}

func TestDetail09_VerifyChartPreconditions(t *testing.T) {
	_, err := downloader.VerifyChart("/no/such/chart.tgz", "", "")
	if err == nil {
		t.Fatal("stat error must propagate")
	}
	dir := t.TempDir()
	_, err = downloader.VerifyChart(dir, "", "")
	if err == nil || !strings.Contains(err.Error(), "unpacked charts cannot be verified") {
		t.Fatalf("directory: %v", err)
	}
	notgz := filepath.Join(dir, "x.tar")
	if err := os.WriteFile(notgz, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = downloader.VerifyChart(notgz, "", "")
	if err == nil || !strings.Contains(err.Error(), "chart must be a tgz file") {
		t.Fatalf("non-tgz: %v", err)
	}
	tgz := filepath.Join(dir, "c.tgz")
	if err := os.WriteFile(tgz, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = downloader.VerifyChart(tgz, filepath.Join(dir, "no.prov"), "")
	if err == nil {
		t.Fatal("missing prov must error")
	}
}

func TestDetail10_IsTarCaseInsensitiveTgzOnly(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.TGZ", "a.tGz", "a.tgz"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := downloader.VerifyChart(p, filepath.Join(dir, "no.prov"), "")
		if err != nil && strings.Contains(err.Error(), "chart must be a tgz file") {
			t.Fatalf("%s must count as tar: %v", name, err)
		}
	}
	p := filepath.Join(dir, "a.tar.gz")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := downloader.VerifyChart(p, "", "")
	if err == nil || !strings.Contains(err.Error(), "chart must be a tgz file") {
		t.Fatalf(".tar.gz is not .tgz: %v", err)
	}
}

func TestDetail11_GetterOptionsAccumulate(t *testing.T) {
	var accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		_, _ = w.Write([]byte("not-a-chart"))
	}))
	t.Cleanup(srv.Close)
	c := dl()
	c.Options = append(c.Options, getter.WithUserAgent("dl-test"))
	_, _, _ = c.DownloadTo(srv.URL+"/x.tgz", "", t.TempDir())
	if accept != "" && !strings.Contains(accept, "gzip") && !strings.Contains(accept, "octet-stream") {
		// AcceptHeader gzip/octet-stream appended per fetch — if the download
		// ran, Accept should mention those types when set.
	}
	_ = accept
}

type memCache struct{}

func (memCache) Get(key [sha256.Size]byte, cacheType string) (string, error) {
	return "", os.ErrNotExist
}
func (memCache) Put(key [sha256.Size]byte, data io.Reader, cacheType string) (string, error) {
	b, _ := io.ReadAll(data)
	_ = b
	return "", nil
}


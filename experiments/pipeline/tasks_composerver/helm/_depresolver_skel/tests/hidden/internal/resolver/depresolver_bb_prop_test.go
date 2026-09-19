// Package resolver_test is a hidden black-box property suite for the depresolver
// unit. Only exported API declared in api.md is exercised:
//
//	resolver.New, (*Resolver).Resolve, resolver.HashReq, resolver.HashV2Req
//	resolver.GetLocalPath
//
// Deterministic seed: 20260919. Cases: >= 10,000.
//
// Contract (contract.md) -> property coverage table:
//
//	S1 "scheme-dispatched resolution (oci/file/repo/alias)" -> TestDRResolveScenarios
//	S2 "local file:// path must exist; version read from chart on disk"
//	    -> TestDRResolveScenarios / TestDRContractTable
//	S3 "non-local deps must satisfy semver constraint against index"
//	    -> TestDRResolveScenarios
//	S4 "missing repo map / unsatisfiable constraint / missing local path -> error"
//	    -> TestDRResolveScenarios / TestDRContractTable
//	S5 "lock hash is sha256 over canonical sorted requirement set"
//	    -> TestDRHashReqStable / TestDRHashV2Stable
//	S6 "GetLocalPath resolves file:// and charts/ relative paths"
//	    -> TestDRGetLocalPathProperty
package resolver_test

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"runtime"
	"testing"

	resolver "example.internal/chartkit/v4/internal/resolver"
	chart "example.internal/chartkit/v4/pkg/chart/v2"
	"example.internal/chartkit/v4/pkg/registry"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

type resolveCase struct {
	name   string
	req    []*chart.Dependency
	repos  map[string]string
	expect *chart.Lock
	err    bool
}

var resolveOracle = []resolveCase{
	{
		name: "repo from invalid version",
		req:  []*chart.Dependency{{Name: "base", Repository: "file://base", Version: "1.1.0"}},
		expect: &chart.Lock{Dependencies: []*chart.Dependency{
			{Name: "base", Repository: "file://base", Version: "0.1.0"},
		}},
		err: true,
	},
	{
		name: "version failure",
		req:  []*chart.Dependency{{Name: "oedipus-rex", Repository: "http://example.com", Version: ">a1"}},
		err:  true,
	},
	{
		name: "chart not found failure",
		req:  []*chart.Dependency{{Name: "redis", Repository: "http://example.com", Version: "1.0.0"}},
		err:  true,
	},
	{
		name: "constraint not satisfied failure",
		req:  []*chart.Dependency{{Name: "alpine", Repository: "http://example.com", Version: ">=1.0.0"}},
		err:  true,
	},
	{
		name: "valid lock",
		req:  []*chart.Dependency{{Name: "alpine", Repository: "http://example.com", Version: ">=0.1.0"}},
		expect: &chart.Lock{Dependencies: []*chart.Dependency{
			{Name: "alpine", Repository: "http://example.com", Version: "0.2.0"},
		}},
	},
	{
		name: "repo from valid local path",
		req:  []*chart.Dependency{{Name: "base", Repository: "file://base", Version: "0.1.0"}},
		expect: &chart.Lock{Dependencies: []*chart.Dependency{
			{Name: "base", Repository: "file://base", Version: "0.1.0"},
		}},
	},
	{
		name: "repo from valid local path with range resolution",
		req:  []*chart.Dependency{{Name: "base", Repository: "file://base", Version: "^0.1.0"}},
		expect: &chart.Lock{Dependencies: []*chart.Dependency{
			{Name: "base", Repository: "file://base", Version: "0.1.0"},
		}},
	},
	{
		name: "repo from invalid local path",
		req:  []*chart.Dependency{{Name: "nonexistent", Repository: "file://testdata/nonexistent", Version: "0.1.0"}},
		err:  true,
	},
	{
		name: "repo from valid path under charts path",
		req:  []*chart.Dependency{{Name: "localdependency", Repository: "", Version: "0.1.0"}},
		expect: &chart.Lock{Dependencies: []*chart.Dependency{
			{Name: "localdependency", Repository: "", Version: "0.1.0"},
		}},
	},
	{
		name: "repo from invalid path under charts path",
		req:  []*chart.Dependency{{Name: "nonexistentdependency", Repository: "", Version: "0.1.0"}},
		err:  true,
	},
}

var repoNamesOracle = map[string]string{"alpine": "kubernetes-charts", "redis": "kubernetes-charts"}

func newResolver(t *testing.T) *resolver.Resolver {
	t.Helper()
	rc, err := registry.NewClient()
	if err != nil {
		t.Fatalf("registry client: %v", err)
	}
	return resolver.New("testdata/chartpath", "testdata/repository", rc)
}

func dep(name, ver, repo string) *chart.Dependency {
	return &chart.Dependency{Name: name, Version: ver, Repository: repo}
}

// ---------------------------------------------------------------------------
// contract table
// ---------------------------------------------------------------------------

func TestDRContractTable(t *testing.T) {
	r := newResolver(t)
	for _, tc := range resolveOracle {
		t.Run(tc.name, func(t *testing.T) {
			lock, err := r.Resolve(tc.req, repoNamesOracle)
			if tc.err {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if len(lock.Dependencies) != len(tc.expect.Dependencies) {
				t.Fatalf("deps len=%d want %d", len(lock.Dependencies), len(tc.expect.Dependencies))
			}
			for i, want := range tc.expect.Dependencies {
				got := lock.Dependencies[i]
				if got.Name != want.Name || got.Repository != want.Repository || got.Version != want.Version {
					t.Fatalf("dep[%d]=%+v want %+v", i, got, want)
				}
			}
			h, err := resolver.HashReq(tc.req, lock.Dependencies)
			if err != nil {
				t.Fatalf("HashReq: %v", err)
			}
			if lock.Digest != h {
				t.Fatalf("Digest=%q HashReq=%q", lock.Digest, h)
			}
		})
	}

	expectHash := "sha256:fb239e836325c5fa14b29d1540a13b7d3ba13151b67fe719f820e0ef6d66aaaf"
	req := []*chart.Dependency{{Name: "alpine", Version: "0.1.0", Repository: "http://localhost:8879/charts"}}
	lock := []*chart.Dependency{{Name: "alpine", Version: "0.1.0", Repository: "http://localhost:8879/charts"}}
	h, err := resolver.HashReq(req, lock)
	if err != nil || h != expectHash {
		t.Fatalf("HashReq baseline: h=%q err=%v", h, err)
	}
	h2, err := resolver.HashV2Req(req)
	if err != nil || h2 == "" {
		t.Fatalf("HashV2Req: h=%q err=%v", h2, err)
	}
}

// ---------------------------------------------------------------------------
// S1-S4: Resolve scenarios (unseen-random selection)
// ---------------------------------------------------------------------------

func TestDRResolveScenarios(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	r := newResolver(t)
	for c := 0; c < bbCases; c++ {
		tc := resolveOracle[rng.Intn(len(resolveOracle))]
		lock, err := r.Resolve(tc.req, repoNamesOracle)
		if tc.err {
			if err == nil {
				t.Fatalf("case %d %q: expected error", c, tc.name)
			}
			continue
		}
		if err != nil {
			t.Fatalf("case %d %q: %v", c, tc.name, err)
		}
		if len(lock.Dependencies) != len(tc.expect.Dependencies) {
			t.Fatalf("case %d %q: len mismatch", c, tc.name)
		}
		got := lock.Dependencies[0]
		want := tc.expect.Dependencies[0]
		if got.Name != want.Name || got.Version != want.Version || got.Repository != want.Repository {
			t.Fatalf("case %d %q: got %+v want %+v", c, tc.name, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// S5: HashReq — in-tree digest table + difference (no permutation assumptions)
// ---------------------------------------------------------------------------

func TestDRHashReqStable(t *testing.T) {
	expect := "sha256:fb239e836325c5fa14b29d1540a13b7d3ba13151b67fe719f820e0ef6d66aaaf"
	cases := []struct {
		chartVer, lockVer string
		wantMatch         bool
	}{
		{"0.1.0", "0.1.0", true},
		{"^0.1.0", "0.1.0", false},
		{"^0.1.0", "0.1.2", false},
		{"0.1.2", "0.1.2", false},
		{"^0.1.2", "0.1.2", false},
	}
	for c := 0; c < bbCases; c++ {
		row := cases[c%len(cases)]
		req := []*chart.Dependency{
			dep("alpine", row.chartVer, "http://localhost:8879/charts"),
		}
		lock := []*chart.Dependency{
			dep("alpine", row.lockVer, "http://localhost:8879/charts"),
		}
		if c%17 == 0 {
			req = append(req, dep(fmt.Sprintf("extra%d", c%9), "^1.0.0", "http://h"))
			lock = append(lock, dep(fmt.Sprintf("extra%d", c%9), "1.2.3", "http://h"))
		}
		h, err := resolver.HashReq(req, lock)
		if err != nil {
			t.Fatalf("case %d: %v", c, err)
		}
		if row.wantMatch && c%17 != 0 {
			if h != expect {
				t.Fatalf("case %d: got %q want %q", c, h, expect)
			}
		}
		if !row.wantMatch && h == expect {
			t.Fatalf("case %d: mismatched req/lock should not equal baseline", c)
		}
		if c%50 == 0 && c > 0 {
			alt := append([]*chart.Dependency(nil), lock...)
			alt[0] = dep(alt[0].Name, "9.9.9", alt[0].Repository)
			h2, err := resolver.HashReq(req, alt)
			if err != nil {
				t.Fatalf("case %d alt: %v", c, err)
			}
			if h2 == h {
				t.Fatalf("case %d: lock version change should alter hash", c)
			}
		}
	}
}

func TestDRHashV2Stable(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for c := 0; c < bbCases; c++ {
		n := 1 + rng.Intn(4)
		var req []*chart.Dependency
		for i := 0; i < n; i++ {
			req = append(req, dep(
				fmt.Sprintf("c%d-%d", c, i),
				fmt.Sprintf("^%d.0.0", 1+rng.Intn(3)),
				fmt.Sprintf("http://repo/%d", rng.Intn(5)),
			))
		}
		h1, err := resolver.HashV2Req(append([]*chart.Dependency(nil), req...))
		if err != nil {
			t.Fatalf("case %d: %v", c, err)
		}
		h2, err := resolver.HashV2Req(append([]*chart.Dependency(nil), req...))
		if err != nil || h1 != h2 {
			t.Fatalf("case %d: identical req should hash identically", c)
		}
		if c > 0 {
			alt := append([]*chart.Dependency(nil), req...)
			alt[0].Version = "999.0.0"
			h3, err := resolver.HashV2Req(alt)
			if err != nil || h3 == h1 {
				t.Fatalf("case %d: version change should alter V2 hash", c)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// S6: GetLocalPath
// ---------------------------------------------------------------------------

type localPathCase struct {
	repo, chartpath, wantUnix, wantWin string
	err                                bool
}

var localPathOracle = []localPathCase{
	{repo: "file:////", wantUnix: "/", wantWin: "\\"},
	{repo: "file://../../testdata/chartpath/base", chartpath: "foo/bar", wantUnix: "testdata/chartpath/base", wantWin: "testdata\\chartpath\\base"},
	{repo: "../charts/localdependency", chartpath: "testdata/chartpath/charts", wantUnix: "testdata/chartpath/charts/localdependency", wantWin: "testdata\\chartpath\\charts\\localdependency"},
	{repo: "file://testdata/nonexistent", chartpath: "testdata/chartpath", err: true},
	{repo: "charts/nonexistentdependency", chartpath: "testdata/chartpath/charts", err: true},
}

func oracleLocalPath(repo, chartpath string) (string, bool) {
	for _, tc := range localPathOracle {
		if tc.repo == repo && tc.chartpath == chartpath {
			if tc.err {
				return "", true
			}
			if runtime.GOOS == "windows" {
				return tc.wantWin, false
			}
			return tc.wantUnix, false
		}
	}
	return "", true
}

func TestDRGetLocalPathProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	for c := 0; c < bbCases; c++ {
		if c%3 == 0 {
			sub := fmt.Sprintf("missing%d", rng.Intn(1000))
			repo := "file://testdata/nonexistent/" + sub
			_, err := resolver.GetLocalPath(repo, "testdata/chartpath")
			if err == nil {
				t.Fatalf("case %d: expected error for %q", c, repo)
			}
			continue
		}
		tc := localPathOracle[rng.Intn(len(localPathOracle))]
		got, err := resolver.GetLocalPath(tc.repo, tc.chartpath)
		if tc.err {
			if err == nil {
				t.Fatalf("case %d %q: expected error", c, tc.repo)
			}
			continue
		}
		if err != nil {
			t.Fatalf("case %d: %v", c, err)
		}
		want, _ := oracleLocalPath(tc.repo, tc.chartpath)
		if got != want {
			t.Fatalf("case %d: got %q want %q (repo=%q chartpath=%q)", c, got, want, tc.repo, tc.chartpath)
		}
		if tc.chartpath != "" && !filepath.IsAbs(got) && !stringsHasPrefix(got, "testdata") {
			if got == "" {
				t.Fatalf("case %d: empty path", c)
			}
		}
	}
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

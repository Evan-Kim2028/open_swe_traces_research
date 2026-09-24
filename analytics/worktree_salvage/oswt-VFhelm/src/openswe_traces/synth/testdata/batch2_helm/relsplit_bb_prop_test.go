// Hidden black-box suite for relsplit. Exported API only: FilterFunc.Check/Filter,
// Any, All, StatusFilter, SplitManifests, BySplitManifestsOrder.
package util_test

import (
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"example.internal/helm/pkg/release/common"
	rspb "example.internal/helm/pkg/release/v1"
	"example.internal/helm/pkg/release/v1/util"
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

func rel(name string, st common.Status) *rspb.Release {
	return &rspb.Release{Name: name, Info: &rspb.Info{Status: st}}
}

func TestDetail01_CheckNilReleaseFalseWithoutInvoking(t *testing.T) {
	called := false
	fn := util.FilterFunc(func(r *rspb.Release) bool {
		called = true
		return true
	})
	if fn.Check(nil) {
		t.Fatal("Check(nil) must be false")
	}
	if called {
		t.Fatal("Check(nil) must not invoke the filter")
	}
}

func TestDetail02_StatusFilterInnerNilTrue(t *testing.T) {
	fn := util.StatusFilter(common.StatusDeployed)
	if !fn(nil) {
		t.Fatal("StatusFilter inner func must return true for nil (direct call)")
	}
	if fn.Check(nil) {
		t.Fatal("Check still short-circuits nil to false")
	}
	r := rel("a", common.StatusDeployed)
	if !fn.Check(r) {
		t.Fatal("deployed must pass StatusFilter(deployed)")
	}
	r2 := rel("b", common.StatusFailed)
	if fn.Check(r2) {
		t.Fatal("failed must not pass StatusFilter(deployed)")
	}
}

func TestDetail03_AnyOrEmptyFalseAllAndEmptyTrue(t *testing.T) {
	if util.Any().Check(rel("x", common.StatusDeployed)) {
		t.Fatal("Any() empty must be false")
	}
	if !util.All().Check(rel("x", common.StatusDeployed)) {
		t.Fatal("All() empty must be true")
	}
	yes := util.FilterFunc(func(r *rspb.Release) bool { return r != nil && r.Name == "keep" })
	no := util.FilterFunc(func(r *rspb.Release) bool { return false })
	if !util.Any(no, yes).Check(rel("keep", common.StatusDeployed)) {
		t.Fatal("Any is OR")
	}
	if util.Any(no, no).Check(rel("keep", common.StatusDeployed)) {
		t.Fatal("Any all-false")
	}
	if !util.All(yes, util.FilterFunc(func(r *rspb.Release) bool { return r.Name == "keep" })).Check(rel("keep", common.StatusFailed)) {
		t.Fatal("All is AND")
	}
	if util.All(yes, no).Check(rel("keep", common.StatusDeployed)) {
		t.Fatal("All mixed")
	}
}

func TestDetail04_FilterPreservesOrder(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	in := []*rspb.Release{
		rel("a", common.StatusDeployed),
		rel("b", common.StatusFailed),
		rel("c", common.StatusDeployed),
		rel("d", common.StatusSuperseded),
	}
	keep := util.FilterFunc(func(r *rspb.Release) bool {
		return r.Info.Status == common.StatusDeployed
	})
	out := keep.Filter(in)
	if len(out) != 2 || out[0].Name != "a" || out[1].Name != "c" {
		t.Fatalf("order-preserving filter got %#v", names(out))
	}
	for i := 0; i < 30; i++ {
		n := 5 + rng.Intn(8)
		var batch []*rspb.Release
		var want []string
		for j := 0; j < n; j++ {
			nm := "r" + strconv.Itoa(j)
			st := common.StatusFailed
			if rng.Intn(2) == 0 {
				st = common.StatusDeployed
				want = append(want, nm)
			}
			batch = append(batch, rel(nm, st))
		}
		got := keep.Filter(batch)
		if len(got) != len(want) {
			t.Fatalf("i=%d len", i)
		}
		for k := range want {
			if got[k].Name != want[k] {
				t.Fatalf("i=%d order %v want %v", i, names(got), want)
			}
		}
	}
}

func TestDetail05_SplitRegexLineStartFusedText(t *testing.T) {
	in := "apiVersion: v1\nkind: A\n---apiVersion: v1\nkind: B\n"
	m := util.SplitManifests(in)
	if len(m) < 2 {
		t.Fatalf("fused ---apiVersion must split, got %#v", m)
	}
	joined := strings.Join(values(m), "\n")
	if !strings.Contains(joined, "kind: A") || !strings.Contains(joined, "kind: B") {
		t.Fatalf("lost docs: %#v", m)
	}
	tab := "kind: T\n---\t  \napiVersion: v1\nkind: U\n"
	m2 := util.SplitManifests(tab)
	foundU := false
	for _, v := range m2 {
		if strings.Contains(v, "kind: U") {
			foundU = true
		}
	}
	if !foundU {
		t.Fatalf("---[tab] must split: %#v", m2)
	}
}

func TestDetail06_LeadingWhitespaceTrimmedBeforeSplit(t *testing.T) {
	in := "\n  \n---\napiVersion: v1\nkind: Z\n"
	m := util.SplitManifests(in)
	if len(m) == 0 {
		t.Fatal("leading whitespace-only prefix must not eat the real doc")
	}
	ok := false
	for _, v := range m {
		if strings.Contains(v, "kind: Z") {
			ok = true
		}
	}
	if !ok {
		t.Fatalf("kind Z missing after pre-trim: %#v", m)
	}
}

func TestDetail07_EmptyDocsDroppedKeysManifestN(t *testing.T) {
	in := "---\n   \n---\napiVersion: v1\nkind: Keep\n---\n\n---\napiVersion: v1\nkind: Also\n"
	m := util.SplitManifests(in)
	if len(m) != 2 {
		t.Fatalf("whitespace-only docs dropped, got %d: %#v", len(m), m)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Sort(util.BySplitManifestsOrder(keys))
	if keys[0] != "manifest-0" || keys[1] != "manifest-1" {
		t.Fatalf("keys want manifest-0,1 got %v", keys)
	}
	for _, v := range m {
		if v != strings.TrimLeft(v, " \t\n") {
			t.Fatalf("survivor must be left-trimmed: %q", v)
		}
	}
}

func TestDetail08_BySplitManifestsOrderNumericSuffix(t *testing.T) {
	keys := []string{"manifest-10", "manifest-2", "manifest-0", "manifest-1"}
	sort.Sort(util.BySplitManifestsOrder(keys))
	want := []string{"manifest-0", "manifest-1", "manifest-2", "manifest-10"}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("numeric suffix sort got %v want %v", keys, want)
		}
	}
	rng := rand.New(rand.NewSource(hiddenSeed() + 1))
	for i := 0; i < 40; i++ {
		n := 5 + rng.Intn(10)
		var ks []string
		for j := 0; j < n; j++ {
			ks = append(ks, "manifest-"+strconv.Itoa(rng.Intn(30)))
		}
		sort.Sort(util.BySplitManifestsOrder(ks))
		for j := 1; j < len(ks); j++ {
			a, _ := strconv.Atoi(strings.TrimPrefix(ks[j-1], "manifest-"))
			b, _ := strconv.Atoi(strings.TrimPrefix(ks[j], "manifest-"))
			if a > b {
				t.Fatalf("i=%d not numeric: %v", i, ks)
			}
		}
	}
}

func names(rs []*rspb.Release) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Name
	}
	return out
}

func values(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// Hidden black-box suite for searchindex. Exported API only: Index, NewIndex,
// AddRepo, All, Search, SearchLiteral, SearchRegexp, SortScore, Result.
package search_test

import (
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"

	chart "example.internal/helm/pkg/chart/v2"
	"example.internal/helm/pkg/cmd/search"
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

func cv(name, ver, desc string, kws []string) *repo.ChartVersion {
	return &repo.ChartVersion{
		Metadata: &chart.Metadata{Name: name, Version: ver, Description: desc, Keywords: kws},
		URLs:     []string{"https://ex.test/" + name + "-" + ver + ".tgz"},
	}
}

func idxOf(entries map[string]repo.ChartVersions) *repo.IndexFile {
	f := repo.NewIndexFile()
	f.Entries = entries
	return f
}

func TestDetail01_MatchLineFourFieldsVerticalTab(t *testing.T) {
	idx := search.NewIndex()
	idx.AddRepo("stable", idxOf(map[string]repo.ChartVersions{
		"alpine": {cv("alpine", "1.0.0", "tiny linux pod", []string{"os", "linux"})},
	}), false)
	// description hit
	hits, err := idx.Search("tiny linux", 100, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("description field must be searchable, got %d", len(hits))
	}
	// keyword hit
	hits, err = idx.Search("os", 100, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("keyword field must be searchable, got %d", len(hits))
	}
	// name hit
	hits, err = idx.Search("alpine", 100, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("name field must be searchable, got %d", len(hits))
	}
}

func TestDetail02_ScoreIsFieldIndexSeparatorBelongsNext(t *testing.T) {
	idx := search.NewIndex()
	idx.AddRepo("r1", idxOf(map[string]repo.ChartVersions{
		"uniqchart": {cv("uniqchart", "1.0.0", "zzdescword", []string{"kwzzz"})},
	}), false)
	nameHits := idx.SearchLiteral("uniqchart", 100)
	if len(nameHits) != 1 || nameHits[0].Score != 0 {
		t.Fatalf("name-field match score want 0 got %+v", dump(nameHits))
	}
	descHits := idx.SearchLiteral("zzdescword", 100)
	if len(descHits) != 1 || descHits[0].Score != 2 {
		t.Fatalf("description-field match score want 2 got %+v", dump(descHits))
	}
	kwHits := idx.SearchLiteral("kwzzz", 100)
	if len(kwHits) != 1 || kwHits[0].Score != 3 {
		t.Fatalf("keyword-field match score want 3 got %+v", dump(kwHits))
	}
	// repo/name is field 1; "r1/uniqchart" should not match field 0
	repoHits := idx.SearchLiteral("r1/uniqchart", 100)
	if len(repoHits) != 1 {
		t.Fatalf("repo/name field should match, got %+v", dump(repoHits))
	}
	if repoHits[0].Score != 1 {
		t.Fatalf("repo/name match score want 1 got %d (separator belongs to next field)", repoHits[0].Score)
	}
}

func TestDetail03_LiteralLowercasesRegexpCaseSensitive(t *testing.T) {
	idx := search.NewIndex()
	idx.AddRepo("s", idxOf(map[string]repo.ChartVersions{
		"Mix": {cv("Mix", "1.0.0", "CamelDesc", []string{"KeyWord"})},
	}), false)
	if n := len(idx.SearchLiteral("mix", 100)); n != 1 {
		t.Fatalf("literal lowercases haystack+term, got %d", n)
	}
	if n := len(idx.SearchLiteral("CAMELDESC", 100)); n != 1 {
		t.Fatalf("literal case-insensitive description, got %d", n)
	}
	reHits, err := idx.SearchRegexp("Mix", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(reHits) != 1 {
		t.Fatalf("regexp exact case should hit, got %d", len(reHits))
	}
	reHits, err = idx.SearchRegexp("mix", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(reHits) != 0 {
		t.Fatalf("regexp is case-sensitive, lowercase must miss, got %d", len(reHits))
	}
}

func TestDetail04_ThresholdExclusive(t *testing.T) {
	idx := search.NewIndex()
	idx.AddRepo("s", idxOf(map[string]repo.ChartVersions{
		"n": {cv("n", "1.0.0", "onlydesc", []string{"k"})},
	}), false)
	// name match score 0; threshold 0 → score < 0 is false → no hits
	if n := len(idx.SearchLiteral("n", 0)); n != 0 {
		t.Fatalf("score < threshold: score 0 vs threshold 0 must miss, got %d", n)
	}
	if n := len(idx.SearchLiteral("n", 1)); n != 1 {
		t.Fatalf("score 0 < 1 must hit, got %d", n)
	}
	desc := idx.SearchLiteral("onlydesc", 2)
	if len(desc) != 0 {
		t.Fatalf("score 2 < 2 must miss, got %d", len(desc))
	}
	if n := len(idx.SearchLiteral("onlydesc", 3)); n != 1 {
		t.Fatalf("score 2 < 3 must hit, got %d", n)
	}
}

func TestDetail05_BadRegexpEmptyNonNilSliceAndError(t *testing.T) {
	idx := search.NewIndex()
	idx.AddRepo("s", idxOf(map[string]repo.ChartVersions{
		"n": {cv("n", "1.0.0", "d", nil)},
	}), false)
	hits, err := idx.SearchRegexp("(", 100)
	if err == nil {
		t.Fatal("bad regexp must error")
	}
	if hits == nil {
		t.Fatal("bad regexp must return empty non-nil slice")
	}
	if len(hits) != 0 {
		t.Fatalf("len %d", len(hits))
	}
	hits2, err := idx.Search("(", 100, true)
	if err == nil || hits2 == nil {
		t.Fatal("Search(regexp=true) same contract")
	}
}

func TestDetail06_AddRepoJoinSkipEmptySortEntriesMutates(t *testing.T) {
	file := idxOf(map[string]repo.ChartVersions{
		"zchart": {
			cv("zchart", "0.9.0", "old", nil),
			cv("zchart", "1.2.0", "new", nil),
		},
		"empty": {},
	})
	before := make([]*repo.ChartVersion, len(file.Entries["zchart"]))
	copy(before, file.Entries["zchart"])
	idx := search.NewIndex()
	idx.AddRepo("stable", file, true)
	// empty version list skipped: searching empty name should miss
	if n := len(idx.SearchLiteral("empty", 100)); n != 0 {
		t.Fatalf("empty version list must be skipped, got %d", n)
	}
	all := idx.All()
	for _, r := range all {
		if !strings.Contains(r.Name, "stable") && !strings.HasPrefix(r.Name, "stable/") && r.Name != "zchart" {
			// name after $$ strip is chart or repo/chart
		}
		if r.Name == "empty" || strings.HasSuffix(r.Name, "/empty") {
			t.Fatalf("empty-versions chart leaked into All: %q", r.Name)
		}
	}
	// SortEntries mutates caller's index — versions should now be newest-first
	vers := file.Entries["zchart"]
	if len(vers) >= 2 && vers[0].Version != "1.2.0" {
		t.Fatalf("SortEntries must mutate caller; first version %q (was %q)", vers[0].Version, before[0].Version)
	}
}

func TestDetail07_AllFalseNewestOnlyAllTrueEveryVersion(t *testing.T) {
	file := idxOf(map[string]repo.ChartVersions{
		"c": {
			cv("c", "1.0.0", "v1", nil),
			cv("c", "2.0.0", "v2", nil),
		},
	})
	newest := search.NewIndex()
	newest.AddRepo("r", file, false)
	if n := len(newest.All()); n != 1 {
		t.Fatalf("all=false indexes only newest, All=%d", n)
	}
	if n := len(newest.SearchLiteral("v1", 100)); n != 0 {
		t.Fatalf("older version must be absent when all=false, got %d", n)
	}
	if n := len(newest.SearchLiteral("v2", 100)); n != 1 {
		t.Fatalf("newest description should hit, got %d", n)
	}
	every := search.NewIndex()
	every.AddRepo("r", file, true)
	if n := len(every.All()); n != 2 {
		t.Fatalf("all=true indexes every version, All=%d", n)
	}
}

func TestDetail08_DollarDollarStrippedFromNames(t *testing.T) {
	file := idxOf(map[string]repo.ChartVersions{
		"c": {
			cv("c", "1.0.0", "v1", nil),
			cv("c", "2.0.0", "v2", nil),
		},
	})
	idx := search.NewIndex()
	idx.AddRepo("repo", file, true)
	for _, r := range idx.All() {
		if strings.Contains(r.Name, "$$") {
			t.Fatalf("All name still has $$: %q", r.Name)
		}
	}
	for _, r := range idx.SearchLiteral("v1", 100) {
		if strings.Contains(r.Name, "$$") {
			t.Fatalf("Search name still has $$: %q", r.Name)
		}
	}
}

func TestDetail09_SortScoreScoreThenNameThenSemverDesc(t *testing.T) {
	a := &search.Result{Name: "b", Score: 1, Chart: cv("b", "1.0.0", "", nil)}
	b := &search.Result{Name: "a", Score: 1, Chart: cv("a", "1.0.0", "", nil)}
	c := &search.Result{Name: "a", Score: 0, Chart: cv("a", "2.0.0", "", nil)}
	d := &search.Result{Name: "a", Score: 1, Chart: cv("a", "0.9.0", "", nil)}
	in := []*search.Result{a, b, c, d}
	search.SortScore(in)
	if in[0] != c {
		t.Fatalf("lowest score first, got %s score %d", in[0].Name, in[0].Score)
	}
	// remaining score=1: name a before b; among a, semver desc 1.0.0 then 0.9.0
	if in[1].Name != "a" || in[1].Chart.Version != "1.0.0" {
		t.Fatalf("name then semver-desc: #1 %+v %s", in[1].Name, in[1].Chart.Version)
	}
	if in[2].Name != "a" || in[2].Chart.Version != "0.9.0" {
		t.Fatalf("#2 %+v %s", in[2].Name, in[2].Chart.Version)
	}
	if in[3].Name != "b" {
		t.Fatalf("#3 %s", in[3].Name)
	}
	// unparseable version sorts first (Less returns true)
	u := &search.Result{Name: "a", Score: 5, Chart: cv("a", "not-semver", "", nil)}
	p := &search.Result{Name: "a", Score: 5, Chart: cv("a", "1.0.0", "", nil)}
	pair := []*search.Result{p, u}
	search.SortScore(pair)
	if pair[0] != u {
		t.Fatalf("unparseable version must sort first, got %s", pair[0].Chart.Version)
	}
}

func TestDetail10_AllScoreZeroOnePerKey(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	file := idxOf(map[string]repo.ChartVersions{})
	n := 5 + rng.Intn(6)
	for i := 0; i < n; i++ {
		nm := "c" + strconv.Itoa(i)
		file.Entries[nm] = repo.ChartVersions{
			cv(nm, "1.0.0", "d", nil),
			cv(nm, "2.0.0", "d2", nil),
		}
	}
	idx := search.NewIndex()
	idx.AddRepo("r", file, false)
	all := idx.All()
	if len(all) != n {
		t.Fatalf("one per key, got %d want %d", len(all), n)
	}
	seen := map[string]bool{}
	for _, r := range all {
		if r.Score != 0 {
			t.Fatalf("All score want 0 got %d", r.Score)
		}
		if seen[r.Name] {
			t.Fatalf("duplicate All name %q", r.Name)
		}
		seen[r.Name] = true
	}
}

func dump(rs []*search.Result) string {
	var b strings.Builder
	for _, r := range rs {
		b.WriteString(r.Name)
		b.WriteByte('@')
		b.WriteString(strconv.Itoa(r.Score))
		b.WriteByte(' ')
	}
	return b.String()
}

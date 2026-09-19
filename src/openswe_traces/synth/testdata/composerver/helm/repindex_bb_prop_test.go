// Package repo_test is a hidden black-box property suite for the repindex
// unit. Only exported API declared in api.md is exercised:
//
//	repo.NewIndexFile(), repo.LoadIndexFile(path), repo.IndexDirectory(dir, baseURL)
//	(*IndexFile).MustAdd / Add / Has / Get / SortEntries / Merge / WriteFile / WriteJSONFile
//	repo.IndexFile, repo.ChartVersion (Metadata embedded), repo.ErrNoChartName...
//	chart.Metadata as the caller-facing entry payload
//	github.com/Masterminds/semver/v3 for oracle constraint semantics
//
// Deterministic seed: 20260919. Cases: >= 10,000.
//
// Contract (contract.md) -> property coverage table:
//
//	S1 "index maps chart names to every published version entry, each
//	    carrying metadata, download URLs, and a sha256 digest"
//	    -> TestRIAddSort (fields stored verbatim per add)
//	S2 "Adding an entry resolves its download URL against the repository
//	    base URL: a filename that is already an absolute URL is used as-is,
//	    a bare filename is joined under the base" -> TestRIAddSort /
//	    TestRIContractTable. NOTE: gold joins basename(filename) under
//	    baseURL for ALL filename shapes (absolute URLs included); suite
//	    asserts the observed basename-join rule.
//	S3 "an existing entry for the same name/version is replaced, not
//	    duplicated" -> NOTE: gold APPENDS duplicate name/version adds
//	    (observed). Suite asserts append behaviour via dup-add case.
//	S4 "Entries are kept sorted by chart name, then by semantic version
//	    descending (newest first); pre-releases sort below their release"
//	    -> TestRIAddSort + TestRILoadRoundTrip
//	S5 "Lookup by version accepts an exact version or a semantic-version
//	    range; a range resolves to the highest matching published version;
//	    an unparseable version constraint is an error, and a name or version
//	    with no match yields a specific not-found error"
//	    -> TestRIGet + TestRIHas
//	S6 "Merging a second index unions all entries without creating
//	    duplicates" -> TestRIMerge
//	S7 "Indexing a directory scans for chart archives and adds each"
//	    -> TestRIIndexDirectory
//	S8 "The serialized index round-trips through YAML and JSON and sorts
//	    entries on load; duplicate or empty entries in loaded data are
//	    tolerated/deduped as specified" -> TestRILoadRoundTrip +
//	    TestRILoadMalformed
package repo_test

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	semver "github.com/Masterminds/semver/v3"

	chart "example.internal/chartkit/v4/pkg/chart/v2"
	repo "example.internal/chartkit/v4/pkg/repo/v1"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

// ---------------------------------------------------------------------------
// generators & oracle helpers
// ---------------------------------------------------------------------------

var bbNames = []string{"alpha", "beta", "gamma", "delta", "zephyr", "quartz"}

// bbVersionPool produces unique-ish semver strings: proper semver plus a few
// shorthand/prerelease/build forms gold accepts verbatim.
func bbVersion(r *rand.Rand) string {
	v := fmt.Sprintf("%d.%d.%d", r.Intn(4), r.Intn(6), r.Intn(10))
	switch r.Intn(8) {
	case 0:
		v += "-alpha"
	case 1:
		v += "-beta.2"
	case 2:
		v += "+build" + fmt.Sprint(r.Intn(9))
	}
	return v
}

func bbMd(name, ver string) *chart.Metadata {
	return &chart.Metadata{
		APIVersion:  "v2",
		Name:        name,
		Version:     ver,
		Description: "desc " + name,
		Home:        "https://home/" + name,
	}
}

// expectURL models gold's rule: basename(filename) joined under baseURL.
func expectURL(baseURL, filename string) string {
	base := strings.TrimSuffix(baseURL, "/")
	return base + "/" + filepath.Base(filepath.FromSlash(filename))
}

func semverOf(t *testing.T, v string) *semver.Version {
	t.Helper()
	sv, err := semver.NewVersion(v)
	if err != nil {
		t.Fatalf("generated version %q not semver: %v", v, err)
	}
	return sv
}

// sortedDesc checks a ChartVersions list is sorted by semver descending.
func sortedDesc(t *testing.T, cvs repo.ChartVersions) bool {
	t.Helper()
	for i := 0; i+1 < len(cvs); i++ {
		a := semverOf(t, cvs[i].Version)
		b := semverOf(t, cvs[i+1].Version)
		if a.LessThan(b) {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// contract table
// ---------------------------------------------------------------------------

func TestRIContractTable(t *testing.T) {
	i := repo.NewIndexFile()
	if i.APIVersion != repo.APIVersionV1 {
		t.Fatalf("NewIndexFile apiVersion = %q, want %q", i.APIVersion, repo.APIVersionV1)
	}
	if i.Entries == nil {
		t.Fatal("NewIndexFile entries nil")
	}

	// S2: every filename shape resolves to basename joined under baseURL.
	for _, tc := range []struct {
		filename, base, want string
	}{
		{"c-0.1.0.tgz", "http://ex.com/charts", "http://ex.com/charts/c-0.1.0.tgz"},
		{"/home/charts/a-0.1.0.tgz", "http://ex.com/charts", "http://ex.com/charts/a-0.1.0.tgz"},
		{"http://cdn.com/a-1.0.0.tgz", "http://h", "http://h/a-1.0.0.tgz"},
		{"sub/dir/b-1.0.0.tgz", "http://h/base", "http://h/base/b-1.0.0.tgz"},
		{"d-1.0.0.tgz", "http://h/sub/", "http://h/sub/d-1.0.0.tgz"},
	} {
		j := repo.NewIndexFile()
		name := "n" + fmt.Sprint(len(tc.filename))
		if err := j.MustAdd(bbMd(name, "1.0.0"), tc.filename, tc.base, "dg"); err != nil {
			t.Fatalf("MustAdd(%q): %v", tc.filename, err)
		}
		if got := j.Entries[name][0].URLs[0]; got != tc.want {
			t.Fatalf("url for filename %q base %q = %q, want %q", tc.filename, tc.base, got, tc.want)
		}
	}

	// S3 (gold-observed): same name+version appends a second entry.
	k := repo.NewIndexFile()
	k.MustAdd(bbMd("dup", "1.0.0"), "dup-1.0.0.tgz", "http://h", "first")
	k.MustAdd(bbMd("dup", "1.0.0"), "dup-1.0.0.tgz", "http://h", "second")
	if len(k.Entries["dup"]) != 2 {
		t.Fatalf("duplicate add did not append: n=%d", len(k.Entries["dup"]))
	}

	// MustAdd validation: empty name / bad version -> error; no apiVersion ok.
	if err := i.MustAdd(bbMd("", "1.0.0"), "x.tgz", "http://h", "d"); err == nil {
		t.Fatal("empty name should error")
	}
	if err := i.MustAdd(bbMd("x", "notsemver"), "x.tgz", "http://h", "d"); err == nil {
		t.Fatal("unparseable version should error")
	}
	if err := i.MustAdd(&chart.Metadata{Name: "noapi", Version: "1.0.0"}, "n.tgz", "http://h", "d"); err != nil {
		t.Fatalf("missing apiVersion should be tolerated: %v", err)
	}

	// S5: Get errors. Unknown name -> ErrNoChartName; version miss -> error.
	if _, err := i.Get("ghost", "1.0.0"); !errors.Is(err, repo.ErrNoChartName) {
		t.Fatalf("unknown name err = %v, want ErrNoChartName", err)
	}
	i.MustAdd(bbMd("solo", "1.0.0"), "s-1.0.0.tgz", "http://h", "d")
	if _, err := i.Get("solo", "9.9.9"); err == nil {
		t.Fatal("unmatched version should error")
	}
	if _, err := i.Get("solo", "!!!bad"); err == nil {
		t.Fatal("unparseable constraint should error")
	}
	// exact + normalized + range lookups
	for q, want := range map[string]string{
		"1.0.0": "1.0.0", "v1.0.0": "1.0.0", "1.0": "1.0.0", ">=1.0.0": "1.0.0", "*": "1.0.0",
	} {
		cv, err := i.Get("solo", q)
		if err != nil || cv.Version != want {
			t.Fatalf("Get(solo,%q) = %v/%v want %s", q, cv, err, want)
		}
	}

	// Add (deprecated) adds valid entries and swallows invalid ones.
	a := repo.NewIndexFile()
	a.Add(bbMd("z", "1.0.0"), "z.tgz", "http://h", "d")
	a.Add(bbMd("", "1.0.0"), "z.tgz", "http://h", "d")
	if len(a.Entries) != 1 {
		t.Fatalf("Add: entries=%d, want 1", len(a.Entries))
	}
}

// ---------------------------------------------------------------------------
// S1+S2+S4: add + sort oracle agreement
// ---------------------------------------------------------------------------

func TestRIAddSort(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed))

	for c := 0; c < bbCases; c++ {
		i := repo.NewIndexFile()
		n := 1 + r.Intn(6)
		type added struct{ name, ver, file, base, digest string }
		var want []added
		for j := 0; j < n; j++ {
			name := bbNames[r.Intn(len(bbNames))]
			ver := bbVersion(r)
			file := fmt.Sprintf("pkgs/%s-%s.tgz", name, ver)
			base := "http://h/" + fmt.Sprint(r.Intn(3))
			digest := fmt.Sprintf("sha256:%x", r.Uint64())
			a := added{name, ver, file, base, digest}
			if err := i.MustAdd(bbMd(name, ver), file, base, digest); err != nil {
				t.Fatalf("case %d: MustAdd(%s,%s): %v", c, name, ver, err)
			}
			want = append(want, a)
		}

		// S1/S2: every add appended verbatim with basename-joined URL.
		count := 0
		for _, cvs := range i.Entries {
			count += len(cvs)
		}
		if count != len(want) {
			t.Fatalf("case %d: entries total %d, want %d", c, count, len(want))
		}
		// check one random add's fields
		a := want[r.Intn(len(want))]
		var found *repo.ChartVersion
		for _, cv := range i.Entries[a.name] {
			if cv.Digest == a.digest {
				found = cv
			}
		}
		if found == nil {
			t.Fatalf("case %d: added entry %s/%s missing", c, a.name, a.ver)
		}
		if found.Version != a.ver || found.Name != a.name {
			t.Fatalf("case %d: stored fields wrong: %+v", c, found)
		}
		if len(found.URLs) != 1 || found.URLs[0] != expectURL(a.base, a.file) {
			t.Fatalf("case %d: URLs %v, want [%s]", c, found.URLs, expectURL(a.base, a.file))
		}

		// S4: after SortEntries, each name sorted semver-desc.
		i.SortEntries()
		for name, cvs := range i.Entries {
			if !sortedDesc(t, cvs) {
				got := make([]string, len(cvs))
				for j, cv := range cvs {
					got[j] = cv.Version
				}
				t.Fatalf("case %d: %s not sorted desc: %v", c, name, got)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// S5: Get / Has oracle agreement
// ---------------------------------------------------------------------------

func TestRIGet(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed + 1))

	// constraints to try per case — exact versions drawn from the added set
	// plus range forms.
	rangeForms := []string{
		">=1.0.0", ">0.5.0", "<3.0.0", "1.*", "2.x", "*",
		"1.0.0 - 3.0.0", "~1.2.0", "^1.0.0", "!=0.0.1",
		">=0.1.0-beta", "", "v1.2.3", "1.2",
	}

	for c := 0; c < bbCases; c++ {
		i := repo.NewIndexFile()
		name := bbNames[r.Intn(len(bbNames))]
		// unique semver-normalized versions so "highest match" is unambiguous
		nver := 1 + r.Intn(5)
		seen := map[string]bool{}
		var vers []string
		for len(vers) < nver {
			v := bbVersion(r)
			sv, err := semver.NewVersion(v)
			if err != nil {
				continue
			}
			key := sv.String()
			if seen[key] {
				continue
			}
			seen[key] = true
			vers = append(vers, v)
			if err := i.MustAdd(bbMd(name, v), name+"-"+v+".tgz", "http://h", "d"); err != nil {
				t.Fatalf("case %d: MustAdd %q: %v", c, v, err)
			}
		}
		i.SortEntries()

		// choose query: 60% exact added version, else range form
		var q string
		if r.Intn(5) < 3 {
			q = vers[r.Intn(len(vers))]
		} else {
			q = rangeForms[r.Intn(len(rangeForms))]
		}

		cv, err := i.Get(name, q)

		// oracle: exact string match first
		var expect *semver.Version
		exactHit := false
		for _, v := range vers {
			if v == q {
				expect = semverOf(t, v)
				exactHit = true
				break
			}
		}
		if !exactHit {
			cs := q
			if cs == "" {
				cs = "*"
			}
			con, cerr := semver.NewConstraint(cs)
			if cerr != nil {
				if err == nil {
					t.Fatalf("case %d: Get(%q) should error on bad constraint", c, q)
				}
				continue
			}
			for _, v := range vers {
				sv := semverOf(t, v)
				if con.Check(sv) && (expect == nil || sv.GreaterThan(expect)) {
					expect = sv
				}
			}
		}

		if expect == nil {
			if err == nil {
				t.Fatalf("case %d: Get(%q) expected error, got %v", c, q, cv.Version)
			}
			continue
		}
		if err != nil {
			t.Fatalf("case %d: Get(%q) err=%v, want %s", c, q, err, expect)
		}
		got := semverOf(t, cv.Version)
		if !got.Equal(expect) {
			t.Fatalf("case %d: Get(%q) = %s, want %s (from %v)", c, q, got, expect, vers)
		}

		// Has == Get-success
		if i.Has(name, q) != (err == nil) {
			t.Fatalf("case %d: Has(%q) inconsistent with Get", c, q)
		}
		if i.Has("no-such-chart", q) {
			t.Fatalf("case %d: Has on unknown name true", c)
		}
	}
}

// ---------------------------------------------------------------------------
// S6: merge
// ---------------------------------------------------------------------------

func TestRIMerge(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed + 2))

	for c := 0; c < bbCases; c++ {
		a, b := repo.NewIndexFile(), repo.NewIndexFile()
		type ev struct{ name, ver, digest string }
		type key struct{ n, v string }
		norm := func(v string) string {
			sv := semverOf(t, v)
			return fmt.Sprintf("%d.%d.%d-%s", sv.Major(), sv.Minor(), sv.Patch(), sv.Prerelease())
		}
		// keep generated versions unique per (index,name,normver) so each
		// index holds at most one entry per dedup key.
		var aSet, bSet []ev
		aSeen, bSeen := map[key]bool{}, map[key]bool{}
		for j := 0; j < 1+r.Intn(5); j++ {
			e := ev{bbNames[r.Intn(len(bbNames))], bbVersion(r), fmt.Sprintf("A%x", r.Uint64())}
			k := key{e.name, norm(e.ver)}
			if aSeen[k] {
				continue
			}
			aSeen[k] = true
			a.MustAdd(bbMd(e.name, e.ver), e.name+".tgz", "http://a", e.digest)
			aSet = append(aSet, e)
		}
		for j := 0; j < 1+r.Intn(5); j++ {
			e := ev{bbNames[r.Intn(len(bbNames))], bbVersion(r), fmt.Sprintf("B%x", r.Uint64())}
			k := key{e.name, norm(e.ver)}
			if bSeen[k] {
				continue
			}
			bSeen[k] = true
			b.MustAdd(bbMd(e.name, e.ver), e.name+".tgz", "http://b", e.digest)
			bSet = append(bSet, e)
		}
		a.Merge(b)

		// oracle: dedup key is (name, semver-equal version ignoring build
		// metadata); first-seen digest wins — a's copy beats b's on
		// collision; new b entries appended.
		want := map[key]string{}
		for _, e := range aSet {
			if _, ok := want[key{e.name, norm(e.ver)}]; !ok {
				want[key{e.name, norm(e.ver)}] = e.digest
			}
		}
		for _, e := range bSet {
			if _, ok := want[key{e.name, norm(e.ver)}]; !ok {
				want[key{e.name, norm(e.ver)}] = e.digest
			}
		}
		for name, cvs := range a.Entries {
			for _, cv := range cvs {
				wantD, ok := want[key{name, norm(cv.Version)}]
				if !ok {
					t.Fatalf("case %d: unexpected merged entry %s/%s", c, name, cv.Version)
				}
				if cv.Digest != wantD {
					t.Fatalf("case %d: %s/%s digest=%s, want preserved %s", c, name, cv.Version, cv.Digest, wantD)
				}
				delete(want, key{name, norm(cv.Version)})
			}
		}
		if len(want) != 0 {
			t.Fatalf("case %d: merged index missing %d entries", c, len(want))
		}
	}
}

// ---------------------------------------------------------------------------
// S7: IndexDirectory
// ---------------------------------------------------------------------------

func mkTgz(t *testing.T, path, name, ver string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	files := []struct{ n, b string }{
		{name + "/Chart.yaml", fmt.Sprintf("apiVersion: v2\nname: %s\nversion: %s\ntype: application\n", name, ver)},
		{name + "/values.yaml", "k: v\n"},
	}
	for _, tf := range files {
		if err := tw.WriteHeader(&tar.Header{Name: tf.n, Mode: 0o644, Size: int64(len(tf.b))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(tf.b)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRIIndexDirectory(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed + 3))

	for c := 0; c < 2000; c++ {
		dir := t.TempDir()
		type spec struct{ rel, name, ver string }
		var specs []spec
		for j := 0; j < 1+r.Intn(4); j++ {
			name := fmt.Sprintf("c%d", j)
			ver := bbVersion(r)
			rel := name + "-" + ver + ".tgz"
			if r.Intn(3) == 0 {
				rel = filepath.Join("nested", rel)
			}
			mkTgz(t, filepath.Join(dir, rel), name, ver)
			specs = append(specs, spec{rel, name, ver})
		}
		// adversarial: non-tgz files and a bogus .tgz are skipped
		os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0o644)
		os.WriteFile(filepath.Join(dir, "bogus.tgz"), []byte("not a real archive"), 0o644)

		base := "http://repo.example/base"
		if r.Intn(2) == 0 {
			base += "/"
		}
		idx, err := repo.IndexDirectory(dir, base)
		if err != nil {
			t.Fatalf("case %d: IndexDirectory err=%v", c, err)
		}
		for _, s := range specs {
			cvs, ok := idx.Entries[s.name]
			if !ok || len(cvs) == 0 {
				t.Fatalf("case %d: missing chart %s", c, s.name)
			}
			var cv *repo.ChartVersion
			for _, v := range cvs {
				if v.Version == s.ver {
					cv = v
				}
			}
			if cv == nil {
				t.Fatalf("case %d: missing version %s for %s", c, s.ver, s.name)
			}
			// URL = base + "/" + relative path (slashes preserved)
			wantURL := strings.TrimSuffix(base, "/") + "/" + filepath.ToSlash(s.rel)
			if len(cv.URLs) != 1 || cv.URLs[0] != wantURL {
				t.Fatalf("case %d: urls=%v want [%s]", c, cv.URLs, wantURL)
			}
			// digest = hex sha256 of the archive file
			raw, _ := os.ReadFile(filepath.Join(dir, s.rel))
			sum := sha256.Sum256(raw)
			if cv.Digest != hex.EncodeToString(sum[:]) {
				t.Fatalf("case %d: digest=%q want sha256 %s", c, cv.Digest, hex.EncodeToString(sum[:]))
			}
		}
	}
}

// ---------------------------------------------------------------------------
// S8: load round-trip + malformed inputs
// ---------------------------------------------------------------------------

func TestRILoadRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed + 4))

	for c := 0; c < bbCases; c++ {
		i := repo.NewIndexFile()
		n := 1 + r.Intn(5)
		for j := 0; j < n; j++ {
			name := bbNames[r.Intn(len(bbNames))]
			ver := bbVersion(r)
			if err := i.MustAdd(bbMd(name, ver), name+"-"+ver+".tgz", "http://h", fmt.Sprintf("d%x", r.Uint64())); err != nil {
				t.Fatalf("case %d: MustAdd: %v", c, err)
			}
		}

		dir := t.TempDir()
		var path string
		if r.Intn(2) == 0 {
			path = filepath.Join(dir, "index.yaml")
			if err := i.WriteFile(path, 0o644); err != nil {
				t.Fatalf("case %d: WriteFile: %v", c, err)
			}
		} else {
			path = filepath.Join(dir, "index.json")
			if err := i.WriteJSONFile(path, 0o644); err != nil {
				t.Fatalf("case %d: WriteJSONFile: %v", c, err)
			}
			raw, _ := os.ReadFile(path)
			if !json.Valid(raw) {
				t.Fatalf("case %d: WriteJSONFile produced invalid JSON", c)
			}
		}

		j, err := repo.LoadIndexFile(path)
		if err != nil {
			t.Fatalf("case %d: LoadIndexFile: %v", c, err)
		}
		// entries preserved as multiset of (name,version,digest)
		type ev struct{ n, v, d string }
		count := func(ix *repo.IndexFile) map[ev]int {
			m := map[ev]int{}
			for n, cvs := range ix.Entries {
				for _, cv := range cvs {
					m[ev{n, cv.Version, cv.Digest}]++
				}
			}
			return m
		}
		wi, wj := count(i), count(j)
		for k, v := range wi {
			if wj[k] != v {
				t.Fatalf("case %d: round-trip lost entry %+v", c, k)
			}
		}
		// loaded entries are sorted desc
		for name, cvs := range j.Entries {
			if !sortedDesc(t, cvs) {
				t.Fatalf("case %d: loaded %s not sorted desc", c, name)
			}
		}
	}
}

func TestRILoadMalformed(t *testing.T) {
	dir := t.TempDir()
	write := func(name, data string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}

	// missing apiVersion -> ErrNoAPIVersion
	if _, err := repo.LoadIndexFile(write("noapi.yaml", "entries:\n  x:\n    - name: x\n      version: 1.0.0\n")); !errors.Is(err, repo.ErrNoAPIVersion) {
		t.Fatalf("missing apiVersion err=%v, want ErrNoAPIVersion", err)
	}
	// empty file -> error
	if _, err := repo.LoadIndexFile(write("empty.yaml", "")); err == nil {
		t.Fatal("empty index should error")
	}
	// duplicate entry keys -> error
	dup := "apiVersion: v1\nentries:\n  x:\n    - name: x\n      version: 1.0.0\n  x:\n    - name: x\n      version: 2.0.0\n"
	if _, err := repo.LoadIndexFile(write("dup.yaml", dup)); err == nil {
		t.Fatal("duplicate entry keys should error")
	}
	// entry missing version -> dropped, no error
	p := write("nover.yaml", "apiVersion: v1\nentries:\n  x:\n    - name: x\n      urls: [http://h/a.tgz]\n    - name: x\n      version: 1.0.0\n      urls: [http://h/b.tgz]\n")
	idx, err := repo.LoadIndexFile(p)
	if err != nil {
		t.Fatalf("missing-version entry should be tolerated: %v", err)
	}
	if len(idx.Entries["x"]) != 1 || idx.Entries["x"][0].Version != "1.0.0" {
		t.Fatalf("expected only the versioned entry, got %+v", idx.Entries["x"])
	}
	// annotations preserved
	p = write("anno.yaml", "apiVersion: v1\nannotations:\n  key1: val1\nentries:\n  x:\n    - name: x\n      version: 1.0.0\n      urls: [http://h/a.tgz]\n")
	idx, err = repo.LoadIndexFile(p)
	if err != nil {
		t.Fatalf("annotated index failed: %v", err)
	}
	if idx.Annotations["key1"] != "val1" {
		t.Fatalf("annotations not preserved: %v", idx.Annotations)
	}
	// nonexistent path -> error
	if _, err := repo.LoadIndexFile(filepath.Join(dir, "ghost.yaml")); err == nil {
		t.Fatal("missing file should error")
	}
}

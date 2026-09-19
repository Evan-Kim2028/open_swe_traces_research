// Package util_test is a hidden black-box property suite for the kindsorter
// unit. It uses only exported API surface declared in api.md:
//
//	util.SortManifests(files map[string]string, _ common.VersionSet, ordering KindSortOrder)
//	util.InstallOrder / util.UninstallOrder (KindSortOrder)
//	util.SortByName / SortByDate / SortByRevision / Reverse on []*release.Release
//	util.SplitManifests, util.Manifest, util.SimpleHead (already-implemented helpers)
//	release.Hook{...}, release.HookEvent, release.HookDeletePolicy, hook annotation consts
//
// Deterministic seed: 20260919. Cases: >= 10,000.
//
// Contract (contract.md) -> property coverage table:
//
//	S1 "SortManifests parses each multi-document YAML manifest file, splits
//	    hook-annotated documents out of the manifest list" -> TestKSContractTableProperty
//	S2 "orders the remaining manifests by resource kind using the given
//	    ordering — stable, so equal kinds keep file order" -> TestKSOracleAgreementProperty
//	S3 "kinds absent from the ordering sort after every known kind, and two
//	    unknown kinds order alphabetically by kind name" -> TestKSContractTableProperty
//	S4 "Hook documents carry a numeric weight annotation (default 0, parsed
//	    from their metadata; unparseable weight is treated as 0) and are
//	    sorted by weight, then by kind ordering" -> TestKSContractTableProperty /
//	    TestKSOracleAgreementProperty. NOTE: gold returns hooks ordered by KIND
//	    ordering (stable); the Weight field is parsed but is not a sort key
//	    in SortManifests output. Suite asserts kind ordering + parsed Weight.
//	S5 "hook delete-policy annotations are preserved on the hook" -> TestKSContractTableProperty
//	S6 "Malformed documents contribute errors; empty documents are skipped"
//	    -> TestKSMalformedAndEmptyProperty. Gold additionally drops (without error)
//	    docs whose helm.sh/hook annotation contains an unknown event token.
//	S7 "release lists sort by name (version suffix tie-break on the numeric
//	    revision), by creation time (oldest first), and by revision number"
//	    -> TestKSReleaseSortsProperty. NOTE: gold SortByName is a plain stable name
//	    sort; equal names keep input order (no version key). Suite asserts
//	    name ordering only.
//	S8 "Reverse applies a sort then inverts it" -> TestKSReleaseSortsProperty
//	S9 "Manifest files sort into the result keyed by name for deterministic
//	    output" -> TestKSOracleAgreementProperty (files processed in key order)
package util_test

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	release "example.internal/chartkit/v4/pkg/release/v1"
	util "example.internal/chartkit/v4/pkg/release/v1/util"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

// ---------------------------------------------------------------------------
// yaml doc builders (only constructs callers could render)
// ---------------------------------------------------------------------------

// docOpts describes one YAML document to emit into a manifest file.
type docOpts struct {
	kind        string
	name        string
	annotations map[string]string
	body        string // if non-empty, used verbatim instead of generated yaml
}

func emitDoc(d docOpts) string {
	if d.body != "" {
		return d.body
	}
	var b strings.Builder
	b.WriteString("apiVersion: v1\n")
	if d.kind != "" {
		fmt.Fprintf(&b, "kind: %s\n", d.kind)
	}
	b.WriteString("metadata:\n")
	fmt.Fprintf(&b, "  name: %s\n", d.name)
	if len(d.annotations) > 0 {
		b.WriteString("  annotations:\n")
		keys := make([]string, 0, len(d.annotations))
		for k := range d.annotations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "    %q: %q\n", k, d.annotations[k])
		}
	}
	return b.String()
}

// emitFile joins docs with --- separators the way rendered templates appear.
func emitFile(docs []docOpts) string {
	parts := make([]string, len(docs))
	for i, d := range docs {
		parts[i] = emitDoc(d)
	}
	return strings.Join(parts, "---\n")
}

// ---------------------------------------------------------------------------
// oracle helpers
// ---------------------------------------------------------------------------

func kindRank(ordering util.KindSortOrder, kind string) int {
	for i, k := range ordering {
		if k == kind {
			return i
		}
	}
	return len(ordering)
}

// knownKinds is the union of InstallOrder kinds (they cover UninstallOrder too).
func knownKinds() []string {
	out := make([]string, len(util.InstallOrder))
	copy(out, util.InstallOrder)
	return out
}

var hookEvents = []string{
	"pre-install", "post-install", "pre-delete", "post-delete",
	"pre-upgrade", "post-upgrade", "pre-rollback", "post-rollback",
	"test", "test-success",
}

var deletePolicies = []string{
	release.HookSucceeded.String(),
	release.HookFailed.String(),
	release.HookBeforeHookCreation.String(),
}

// ---------------------------------------------------------------------------
// generators
// ---------------------------------------------------------------------------

type genDoc struct {
	docOpts
	isHook   bool
	dropped  bool // helm.sh/hook present but an event token is unknown
	weight   int
	events   []string
	policies []string
}

type genFile struct {
	path string
	docs []genDoc
}

var rngKinds = []string{
	"Namespace", "ConfigMap", "Secret", "Deployment", "Service", "Pod",
	"Job", "CronJob", "Role", "ClusterRole", "CustomResourceDefinition",
	"UnknownKindA", "UnknownKindB", "ZZZUnknown",
}

func randDoc(r *rand.Rand, idx int) genDoc {
	d := genDoc{}
	d.kind = rngKinds[r.Intn(len(rngKinds))]
	d.name = fmt.Sprintf("obj-%d", idx)
	isHook := r.Intn(4) == 0
	if isHook {
		ann := map[string]string{}
		// 1-2 events, occasionally an unknown event token mixed in.
		n := 1 + r.Intn(2)
		evs := make([]string, 0, n)
		for i := 0; i < n; i++ {
			e := hookEvents[r.Intn(len(hookEvents))]
			if r.Intn(8) == 0 {
				e = "not-a-real-event"
			}
			evs = append(evs, e)
		}
		d.events = evs
		ann[release.HookAnnotation] = strings.Join(evs, ", ")
		// Gold drops the doc entirely when any event token is unknown.
		if len(expectedHookEvents(evs)) != len(evs) {
			d.dropped = true
		} else {
			d.isHook = true
		}
		switch r.Intn(4) {
		case 0: // no weight annotation
		case 1: // unparseable weight -> treated as 0
			ann[release.HookWeightAnnotation] = "notanumber"
			d.weight = 0
		default:
			d.weight = r.Intn(21) - 10 // -10..10
			ann[release.HookWeightAnnotation] = strconv.Itoa(d.weight)
		}
		if r.Intn(2) == 0 {
			n := 1 + r.Intn(2)
			ps := make([]string, n)
			for i := range ps {
				ps[i] = deletePolicies[r.Intn(len(deletePolicies))]
			}
			d.policies = ps
			ann[release.HookDeleteAnnotation] = strings.Join(ps, ", ")
		}
		d.annotations = ann
	} else if r.Intn(6) == 0 {
		// non-hook doc that still carries unrelated annotations
		d.annotations = map[string]string{"unrelated": "yes"}
	}
	return d
}

func randFile(r *rand.Rand, docIdx *int, basename string) genFile {
	f := genFile{path: fmt.Sprintf("dir%d/%s.yaml", r.Intn(3), basename)}
	n := 1 + r.Intn(4)
	for i := 0; i < n; i++ {
		f.docs = append(f.docs, randDoc(r, *docIdx))
		*docIdx = *docIdx + 1
	}
	return f
}

// expectedHookEvents maps parsed hook event tokens to HookEvent values the way
// the package's event table does (including the test-success alias).
func expectedHookEvents(tokens []string) []release.HookEvent {
	var out []release.HookEvent
	for _, tok := range tokens {
		for _, t := range strings.Split(tok, ",") {
			t = strings.TrimSpace(t)
			switch release.HookEvent(t) {
			case release.HookPreInstall, release.HookPostInstall,
				release.HookPreDelete, release.HookPostDelete,
				release.HookPreUpgrade, release.HookPostUpgrade,
				release.HookPreRollback, release.HookPostRollback,
				release.HookTest:
				out = append(out, release.HookEvent(t))
			case "test-success":
				out = append(out, release.HookTest)
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// contract table tests
// ---------------------------------------------------------------------------

// TestKSContractTableProperty pins small caller-visible behaviours from the contract.
func TestKSContractTableProperty(t *testing.T) {
	// S3: Namespace precedes an unknown kind on install.
	hooks, mans, err := util.SortManifests(map[string]string{
		"f.yaml": emitFile([]docOpts{
			{kind: "TotallyUnknown", name: "u1"},
			{kind: "Namespace", name: "ns1"},
		}),
	}, nil, util.InstallOrder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hooks) != 0 {
		t.Fatalf("expected no hooks, got %d", len(hooks))
	}
	if len(mans) != 2 || mans[0].Head.Kind != "Namespace" || mans[1].Head.Kind != "TotallyUnknown" {
		t.Fatalf("unknown kind did not sort after Namespace: %+v", mans)
	}

	// S3: two unknown kinds order alphabetically.
	_, mans, err = util.SortManifests(map[string]string{
		"f.yaml": emitFile([]docOpts{
			{kind: "ZebraKind", name: "z"},
			{kind: "AlphaKind", name: "a"},
		}),
	}, nil, util.InstallOrder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mans) != 2 || mans[0].Head.Kind != "AlphaKind" || mans[1].Head.Kind != "ZebraKind" {
		t.Fatalf("unknown kinds not alphabetical: %+v", mans)
	}

	// S4: hooks carry parsed Weight (default 0, unparseable -> 0) and are
	// ordered by kind ordering (stable within kind), matching gold.
	hs, _, err := util.SortManifests(map[string]string{
		"f.yaml": emitFile([]docOpts{
			{kind: "Pod", name: "heavy", annotations: map[string]string{
				release.HookAnnotation:       "pre-install",
				release.HookWeightAnnotation: "5",
			}},
			{kind: "Namespace", name: "light", annotations: map[string]string{
				release.HookAnnotation:       "pre-install",
				release.HookWeightAnnotation: "-2",
			}},
			{kind: "Pod", name: "badw", annotations: map[string]string{
				release.HookAnnotation:       "pre-install",
				release.HookWeightAnnotation: "garbage",
			}},
			{kind: "Pod", name: "noweight", annotations: map[string]string{
				release.HookAnnotation: "pre-install",
			}},
		}),
	}, nil, util.InstallOrder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hs) != 4 {
		t.Fatalf("expected 4 hooks, got %d", len(hs))
	}
	// Kind ordering dominates: the Namespace hook precedes all Pod hooks
	// regardless of weight; equal kinds keep file order.
	wantOrder := []string{"light", "heavy", "badw", "noweight"}
	for i, w := range wantOrder {
		if hs[i].Name != w {
			t.Fatalf("hook order mismatch at %d: got %v want %v", i, names2(hs), wantOrder)
		}
	}
	byHName := map[string]*release.Hook{}
	for _, h := range hs {
		byHName[h.Name] = h
	}
	if byHName["light"].Weight != -2 || byHName["heavy"].Weight != 5 ||
		byHName["badw"].Weight != 0 || byHName["noweight"].Weight != 0 {
		t.Fatalf("weights parsed wrong: %+v", hs)
	}

	// S1/S5: hook annotations -> Events/DeletePolicies preserved; Path = file.
	hs, mans, err = util.SortManifests(map[string]string{
		"dir/sub.yaml": emitFile([]docOpts{
			{kind: "Job", name: "hk", annotations: map[string]string{
				release.HookAnnotation:       "post-delete, pre-install",
				release.HookDeleteAnnotation: "hook-succeeded, before-hook-creation",
			}},
			{kind: "ConfigMap", name: "cm"},
		}),
	}, nil, util.InstallOrder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hs) != 1 || len(mans) != 1 {
		t.Fatalf("partition wrong: hooks=%d manifests=%d", len(hs), len(mans))
	}
	h := hs[0]
	wantEv := []release.HookEvent{release.HookPostDelete, release.HookPreInstall}
	if len(h.Events) != len(wantEv) || h.Events[0] != wantEv[0] || h.Events[1] != wantEv[1] {
		t.Fatalf("hook events wrong: %v", h.Events)
	}
	wantPol := []release.HookDeletePolicy{release.HookSucceeded, release.HookBeforeHookCreation}
	if len(h.DeletePolicies) != 2 || h.DeletePolicies[0] != wantPol[0] || h.DeletePolicies[1] != wantPol[1] {
		t.Fatalf("delete policies not preserved: %v", h.DeletePolicies)
	}
	if h.Path != "dir/sub.yaml" || h.Kind != "Job" || h.Name != "hk" {
		t.Fatalf("hook fields wrong: %+v", h)
	}
	if !strings.Contains(h.Manifest, "kind: Job") {
		t.Fatalf("hook manifest should contain the doc, got %q", h.Manifest)
	}

	// Gold-visible behaviour (also pinned by the in-tree regression test): a
	// doc whose helm.sh/hook annotation contains an unknown event token is
	// skipped entirely -- not a hook, not a manifest, no error.
	hs, mans, err = util.SortManifests(map[string]string{
		"f.yaml": emitFile([]docOpts{
			{kind: "Job", name: "strange", annotations: map[string]string{
				release.HookAnnotation: "no-such-event",
			}},
			{kind: "ConfigMap", name: "cm"},
			{kind: "Job", name: "mixed", annotations: map[string]string{
				release.HookAnnotation: "pre-install, bogus",
			}},
		}),
	}, nil, util.InstallOrder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hs) != 0 || len(mans) != 1 {
		t.Fatalf("unknown-event docs should be dropped: hooks=%d mans=%d", len(hs), len(mans))
	}

	// S6: malformed document contributes an error.
	_, _, err = util.SortManifests(map[string]string{
		"f.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: ok\n---\njust a bare scalar\n",
	}, nil, util.InstallOrder)
	if err == nil {
		t.Fatal("malformed document should contribute an error")
	}

	// S6: empty document between separators is skipped, no error.
	_, mans, err = util.SortManifests(map[string]string{
		"f.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: ok\n---\n\n---\napiVersion: v1\nkind: Pod\nmetadata:\n  name: p\n",
	}, nil, util.InstallOrder)
	if err != nil {
		t.Fatalf("empty docs should be skipped without error: %v", err)
	}
	if len(mans) != 2 {
		t.Fatalf("expected 2 manifests, got %d", len(mans))
	}

	// Contract-visible behaviour from the in-tree regression test: files whose
	// basename starts with '_' are partials and skipped even if malformed.
	_, mans, err = util.SortManifests(map[string]string{
		"six/_partial.yaml": "totally invalid manifest body",
		"six/real.yaml":     emitFile([]docOpts{{kind: "Pod", name: "p"}}),
	}, nil, util.InstallOrder)
	if err != nil {
		t.Fatalf("underscore-prefixed partials should be skipped: %v", err)
	}
	if len(mans) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(mans))
	}
}

// ---------------------------------------------------------------------------
// release list sorts
// ---------------------------------------------------------------------------

func mkRelease(name string, ver int, t time.Time) *release.Release {
	return &release.Release{Name: name, Version: ver, Info: &release.Info{LastDeployed: t}}
}

func cloneReleases(in []*release.Release) []*release.Release {
	out := make([]*release.Release, len(in))
	copy(out, in)
	return out
}

// isSortedByName models gold's plain lexicographic name sort.
func isSortedByName(rs []*release.Release) bool {
	for i := 0; i+1 < len(rs); i++ {
		if rs[i+1].Name < rs[i].Name {
			return false
		}
	}
	return true
}

func names2(hs []*release.Hook) []string {
	out := make([]string, len(hs))
	for i, h := range hs {
		out[i] = h.Name
	}
	return out
}

// TestKSReleaseSortsProperty exercises SortByName/Date/Revision/Reverse with
// deterministic random lists (part of the >=10k case budget).
func TestKSReleaseSortsProperty(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed))
	base := time.Unix(1_700_000_000, 0)

	for c := 0; c < bbCases; c++ {
		n := 1 + r.Intn(8)
		var list []*release.Release
		for i := 0; i < n; i++ {
			// small name space -> duplicate names occur; equal names are
			// unconstrained, asserted only pairwise.
			name := fmt.Sprintf("n%c%c", 'a'+r.Intn(8), 'a'+r.Intn(8))
			list = append(list, mkRelease(name, 1+r.Intn(30), base.Add(time.Duration(r.Intn(100000))*time.Second)))
		}

		// SortByName: non-decreasing under the suffix-aware comparator.
		byName := cloneReleases(list)
		util.SortByName(byName)
		if !isSortedByName(byName) {
			t.Fatalf("case %d: SortByName not sorted: %v", c, names(byName))
		}

		// SortByDate: oldest first.
		byDate := cloneReleases(list)
		util.SortByDate(byDate)
		for i := 0; i+1 < len(byDate); i++ {
			if byDate[i+1].Info.LastDeployed.Before(byDate[i].Info.LastDeployed) {
				t.Fatalf("case %d: SortByDate not oldest-first", c)
			}
		}

		// SortByRevision: ascending Version.
		byRev := cloneReleases(list)
		util.SortByRevision(byRev)
		for i := 0; i+1 < len(byRev); i++ {
			if byRev[i+1].Version < byRev[i].Version {
				t.Fatalf("case %d: SortByRevision not ascending", c)
			}
		}

		// Reverse(fn) = sort then invert. For strict comparators on distinct
		// keys this is the exact reverse of the sorted order.
		revName := cloneReleases(list)
		util.Reverse(revName, util.SortByName)
		for i := 0; i+1 < len(revName); i++ {
			if revName[i].Name < revName[i+1].Name {
				t.Fatalf("case %d: Reverse(SortByName) not descending", c)
			}
		}
		revDate := cloneReleases(list)
		util.Reverse(revDate, util.SortByDate)
		for i := 0; i+1 < len(revDate); i++ {
			if revDate[i].Info.LastDeployed.Before(revDate[i+1].Info.LastDeployed) {
				t.Fatalf("case %d: Reverse(SortByDate) not descending", c)
			}
		}
		revRev := cloneReleases(list)
		util.Reverse(revRev, util.SortByRevision)
		for i := 0; i+1 < len(revRev); i++ {
			if revRev[i].Version < revRev[i+1].Version {
				t.Fatalf("case %d: Reverse(SortByRevision) not descending", c)
			}
		}

		// Multisets must be preserved by every sort.
		if !sameMultiset(list, byName) || !sameMultiset(list, byDate) ||
			!sameMultiset(list, byRev) || !sameMultiset(list, revName) {
			t.Fatalf("case %d: sort lost elements", c)
		}
	}
}

func names(rs []*release.Release) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Name
	}
	return out
}

func sameMultiset(a, b []*release.Release) bool {
	if len(a) != len(b) {
		return false
	}
	key := func(r *release.Release) string {
		return fmt.Sprintf("%s/%d/%d", r.Name, r.Version, r.Info.LastDeployed.UnixNano())
	}
	ca, cb := map[string]int{}, map[string]int{}
	for _, r := range a {
		ca[key(r)]++
	}
	for _, r := range b {
		cb[key(r)]++
	}
	for k, v := range ca {
		if cb[k] != v {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// oracle agreement over random file sets
// ---------------------------------------------------------------------------

// TestKSOracleAgreementProperty generates random manifest file sets, runs
// SortManifests, and checks the partition + ordering contract against an
// independent oracle built only from the contract text.
func TestKSOracleAgreementProperty(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed + 1))

	for c := 0; c < bbCases; c++ {
		docIdx := 0
		nfiles := 1 + r.Intn(3)
		files := map[string]string{}
		gen := map[string]genFile{}
		for i := 0; i < nfiles; i++ {
			base := fmt.Sprintf("f%d", i)
			if r.Intn(10) == 0 {
				base = "_" + base // partial file, must be skipped entirely
			}
			gf := randFile(r, &docIdx, base)
			files[gf.path] = emitFile(fileDocs(gf))
			gen[gf.path] = gf
		}

		ordering := util.InstallOrder
		if r.Intn(3) == 0 {
			ordering = util.UninstallOrder
		}

		hooks, mans, err := util.SortManifests(files, nil, ordering)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", c, err)
		}

		// --- oracle: expected generic manifest contents, in order ---
		type em struct {
			content string
			kind    string
		}
		var want []em
		paths := make([]string, 0, len(gen))
		for p := range gen {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		for _, p := range paths {
			gf := gen[p]
			base := p[strings.LastIndexByte(p, '/')+1:]
			if strings.HasPrefix(base, "_") {
				continue
			}
			for _, d := range gf.docs {
				if d.isHook || d.dropped {
					continue
				}
				want = append(want, em{content: emitDoc(d.docOpts), kind: d.kind})
			}
		}
		sort.SliceStable(want, func(i, j int) bool {
			ri, rj := kindRank(ordering, want[i].kind), kindRank(ordering, want[j].kind)
			ki, kj := want[i].kind, want[j].kind
			riKnown, rjKnown := ri < len(ordering), rj < len(ordering)
			switch {
			case riKnown && !rjKnown:
				return true
			case !riKnown && rjKnown:
				return false
			case !riKnown && !rjKnown:
				if ki != kj {
					return ki < kj
				}
				return false
			default:
				return ri < rj
			}
		})

		if len(mans) != len(want) {
			t.Fatalf("case %d: got %d manifests, want %d", c, len(mans), len(want))
		}
		for i := range want {
			if mans[i].Content != want[i].content {
				t.Fatalf("case %d: manifest %d content mismatch\ngot:  %q\nwant: %q", c, i, mans[i].Content, want[i].content)
			}
		}

		// --- oracle: hook fields and ordering constraints ---
		type eh struct {
			name     string
			kind     string
			path     string
			weight   int
			events   []string
			policies []string
		}
		var wantHooks []eh
		for _, p := range paths {
			gf := gen[p]
			base := p[strings.LastIndexByte(p, '/')+1:]
			if strings.HasPrefix(base, "_") {
				continue
			}
			for _, d := range gf.docs {
				if !d.isHook || d.dropped {
					continue
				}
				wantHooks = append(wantHooks, eh{
					name: d.name, kind: d.kind, path: p,
					weight: d.weight, events: d.events, policies: d.policies,
				})
			}
		}
		// Gold orders hooks by kind ordering, stable within equal rank
		// (unknown kinds last, alphabetical among themselves).
		sort.SliceStable(wantHooks, func(i, j int) bool {
			ri, rj := kindRank(ordering, wantHooks[i].kind), kindRank(ordering, wantHooks[j].kind)
			switch {
			case ri < len(ordering) && rj < len(ordering):
				return ri < rj
			case ri < len(ordering):
				return true
			case rj < len(ordering):
				return false
			default:
				return wantHooks[i].kind < wantHooks[j].kind
			}
		})
		if len(hooks) != len(wantHooks) {
			t.Fatalf("case %d: got %d hooks, want %d", c, len(hooks), len(wantHooks))
		}
		for i := range wantHooks {
			if hooks[i].Name != wantHooks[i].name {
				t.Fatalf("case %d: hook order mismatch at %d: got %v want-kind-order", c, i, names2(hooks))
			}
		}
		byName := map[string]*release.Hook{}
		for _, h := range hooks {
			byName[h.Name] = h
		}
		for _, wh := range wantHooks {
			h, ok := byName[wh.name]
			if !ok {
				t.Fatalf("case %d: missing hook %q", c, wh.name)
			}
			if h.Kind != wh.kind || h.Path != wh.path || h.Weight != wh.weight {
				t.Fatalf("case %d: hook %q fields wrong: %+v", c, wh.name, h)
			}
			wantEv := expectedHookEvents(wh.events)
			gotEv := []release.HookEvent(h.Events)
			if len(gotEv) != len(wantEv) {
				t.Fatalf("case %d: hook %q events %v, want %v", c, wh.name, gotEv, wantEv)
			}
			for i := range wantEv {
				if gotEv[i] != wantEv[i] {
					t.Fatalf("case %d: hook %q events %v, want %v", c, wh.name, gotEv, wantEv)
				}
			}
			gotPol := []release.HookDeletePolicy(h.DeletePolicies)
			if len(gotPol) != len(wh.policies) {
				t.Fatalf("case %d: hook %q policies %v, want %v", c, wh.name, gotPol, wh.policies)
			}
			for i := range wh.policies {
				if gotPol[i] != release.HookDeletePolicy(wh.policies[i]) {
					t.Fatalf("case %d: hook %q policies %v, want %v", c, wh.name, gotPol, wh.policies)
				}
			}
		}
	}
}

func fileDocs(gf genFile) []docOpts {
	out := make([]docOpts, len(gf.docs))
	for i, d := range gf.docs {
		out[i] = d.docOpts
	}
	return out
}

// TestKSMalformedAndEmptyProperty covers S6 with randomized malformed/empty docs.
func TestKSMalformedAndEmptyProperty(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed + 2))
	badBodies := []string{
		"just a bare scalar",
		"[unclosed",
		": : :",
		"key: {bad",
	}
	for c := 0; c < 2000; c++ {
		good := emitDoc(docOpts{kind: "ConfigMap", name: fmt.Sprintf("ok%d", c)})
		bad := badBodies[r.Intn(len(badBodies))]
		content := good
		if r.Intn(2) == 0 {
			content = bad + "---\n" + good
		} else {
			content = good + "---\n" + bad
		}
		_, _, err := util.SortManifests(map[string]string{"f.yaml": content}, nil, util.InstallOrder)
		if err == nil {
			t.Fatalf("case %d: malformed doc %q produced no error", c, bad)
		}
	}
	// Whitespace-only and empty files never error and produce nothing.
	for c := 0; c < 500; c++ {
		content := strings.Repeat(" \n\t", 1+r.Intn(3))
		hooks, mans, err := util.SortManifests(map[string]string{"f.yaml": content}, nil, util.InstallOrder)
		if err != nil || len(hooks) != 0 || len(mans) != 0 {
			t.Fatalf("case %d: empty file gave err=%v hooks=%d mans=%d", c, err, len(hooks), len(mans))
		}
	}
}

// Black-box property suite for the memorydriver unit.
// Exported API only (api.md): NewMemory, Name, SetNamespace, Get, List,
// Query, Create, Update, Delete. Keys are <name>.v<version>.
// Seed 20260919; >=10k cases; contract + in-tree fixture coverage.
package driver_test

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"testing"

	"example.internal/chartkit/v4/pkg/release"
	"example.internal/chartkit/v4/pkg/release/common"
	rspb "example.internal/chartkit/v4/pkg/release/v1"
	driver "example.internal/chartkit/v4/pkg/storage/driver"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func bbKey(name string, ver int) string {
	return fmt.Sprintf("%s.v%d", name, ver)
}

func bbRelease(name string, ver int, ns string, st common.Status, labels map[string]string) *rspb.Release {
	lbs := map[string]string{"key1": "val1", "key2": "val2"}
	for k, v := range labels {
		lbs[k] = v
	}
	return &rspb.Release{
		Name:      name,
		Version:   ver,
		Namespace: ns,
		Info:      &rspb.Info{Status: st},
		Labels:    lbs,
	}
}

func bbAsV1(t *testing.T, rel release.Releaser) *rspb.Release {
	t.Helper()
	switch r := rel.(type) {
	case *rspb.Release:
		return r
	case rspb.Release:
		return &r
	default:
		t.Fatalf("unexpected releaser type %T", rel)
		return nil
	}
}

func bbAsV1Rel(rel release.Releaser) *rspb.Release {
	switch r := rel.(type) {
	case *rspb.Release:
		return r
	case rspb.Release:
		return &r
	default:
		return nil
	}
}

// bbLabels builds the label set Query matches against (system + custom), as
// observed from in-tree Query fixtures (status/name/version + release.Labels).
func bbLabels(rls *rspb.Release) map[string]string {
	out := map[string]string{
		"name":    rls.Name,
		"status":  rls.Info.Status.String(),
		"version": strconv.Itoa(rls.Version),
		"owner":   "helm",
	}
	for k, v := range rls.Labels {
		out[k] = v
	}
	return out
}

func bbLabelMatch(rls *rspb.Release, q map[string]string) bool {
	lbs := bbLabels(rls)
	for k, v := range q {
		if lbs[k] != v {
			return false
		}
	}
	return true
}

// bbStore is a reference model keyed by namespace -> key -> release.
type bbStore struct {
	mu   sync.Mutex
	data map[string]map[string]*rspb.Release // ns -> key -> rls
}

func newBBStore() *bbStore {
	return &bbStore{data: map[string]map[string]*rspb.Release{}}
}

func bbStoreNsBucket(s *bbStore, ns string) map[string]*rspb.Release {
	if s.data[ns] == nil {
		s.data[ns] = map[string]*rspb.Release{}
	}
	return s.data[ns]
}

func bbStoreAllInNS(s *bbStore, ns string) []*rspb.Release {
	var out []*rspb.Release
	if ns == "" {
		for _, b := range s.data {
			for _, r := range b {
				out = append(out, r)
			}
		}
		return out
	}
	for _, r := range bbStoreNsBucket(s, ns) {
		out = append(out, r)
	}
	return out
}

func (s *bbStore) Create(key string, rls *rspb.Release) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := bbStoreNsBucket(s, rls.Namespace)
	if _, ok := b[key]; ok {
		return driver.ErrReleaseExists
	}
	// same key string may exist in another namespace (in-tree create fixture).
	b[key] = rls
	return nil
}

func (s *bbStore) Get(ns string, key string) (*rspb.Release, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ns == "" {
		for _, b := range s.data {
			if r, ok := b[key]; ok {
				return r, nil
			}
		}
		return nil, driver.ErrReleaseNotFound
	}
	b := bbStoreNsBucket(s, ns)
	if r, ok := b[key]; ok {
		return r, nil
	}
	return nil, driver.ErrReleaseNotFound
}

func (s *bbStore) Update(ns string, key string, rls *rspb.Release) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ns == "" {
		for _, b := range s.data {
			if _, ok := b[key]; ok {
				b[key] = rls
				return nil
			}
		}
		return driver.ErrReleaseNotFound
	}
	b := bbStoreNsBucket(s, ns)
	if _, ok := b[key]; !ok {
		return driver.ErrReleaseNotFound
	}
	b[key] = rls
	return nil
}

func (s *bbStore) Delete(ns string, key string) (*rspb.Release, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ns == "" {
		for n, b := range s.data {
			if r, ok := b[key]; ok {
				delete(b, key)
				if len(b) == 0 {
					delete(s.data, n)
				}
				return r, nil
			}
		}
		return nil, driver.ErrReleaseNotFound
	}
	b := bbStoreNsBucket(s, ns)
	if r, ok := b[key]; ok {
		delete(b, key)
		if len(b) == 0 {
			delete(s.data, ns)
		}
		return r, nil
	}
	return nil, driver.ErrReleaseNotFound
}

func (s *bbStore) List(ns string, filter func(*rspb.Release) bool) []*rspb.Release {
	all := bbStoreAllInNS(s, ns)
	best := map[string]*rspb.Release{}
	for _, r := range all {
		if filter != nil && !filter(r) {
			continue
		}
		if cur, ok := best[r.Name]; !ok || r.Version > cur.Version {
			best[r.Name] = r
		}
	}
	out := make([]*rspb.Release, 0, len(best))
	for _, r := range best {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *bbStore) Query(ns string, q map[string]string) []*rspb.Release {
	all := bbStoreAllInNS(s, ns)
	best := map[string]*rspb.Release{}
	for _, r := range all {
		if !bbLabelMatch(r, q) {
			continue
		}
		if cur, ok := best[r.Name]; !ok || r.Version > cur.Version {
			best[r.Name] = r
		}
	}
	out := make([]*rspb.Release, 0, len(best))
	for _, r := range best {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func bbLoadFixture(t *testing.T) (*driver.Memory, *bbStore) {
	t.Helper()
	mem := driver.NewMemory()
	ref := newBBStore()
	hs := []*rspb.Release{
		bbRelease("rls-a", 4, "default", common.StatusDeployed, nil),
		bbRelease("rls-a", 1, "default", common.StatusSuperseded, nil),
		bbRelease("rls-a", 3, "default", common.StatusSuperseded, nil),
		bbRelease("rls-a", 2, "default", common.StatusSuperseded, nil),
		bbRelease("rls-b", 4, "default", common.StatusDeployed, nil),
		bbRelease("rls-b", 1, "default", common.StatusSuperseded, nil),
		bbRelease("rls-b", 3, "default", common.StatusSuperseded, nil),
		bbRelease("rls-b", 2, "default", common.StatusSuperseded, nil),
		bbRelease("rls-c", 4, "mynamespace", common.StatusDeployed, nil),
		bbRelease("rls-c", 1, "mynamespace", common.StatusSuperseded, nil),
		bbRelease("rls-c", 3, "mynamespace", common.StatusSuperseded, nil),
		bbRelease("rls-c", 2, "mynamespace", common.StatusSuperseded, nil),
	}
	for _, r := range hs {
		k := bbKey(r.Name, r.Version)
		if err := mem.Create(k, r); err != nil {
			t.Fatalf("fixture create %s: %v", k, err)
		}
		if err := ref.Create(k, r); err != nil {
			t.Fatalf("ref create %s: %v", k, err)
		}
	}
	return mem, ref
}

func bbRelSliceEqual(a, b []release.Releaser) bool {
	if len(a) != len(b) {
		return false
	}
	ka := make([]string, len(a))
	kb := make([]string, len(b))
	for i := range a {
		ra := bbAsV1Rel(a[i])
		rb := bbAsV1Rel(b[i])
		ka[i] = bbKey(ra.Name, ra.Version)
		kb[i] = bbKey(rb.Name, rb.Version)
	}
	sort.Strings(ka)
	sort.Strings(kb)
	for i := range ka {
		if ka[i] != kb[i] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// generators
// ---------------------------------------------------------------------------

var bbNames = []string{"alpha", "beta", "gamma", "zephyr", "quartz", "rls-x"}
var bbNS = []string{"default", "mynamespace", "other-ns"}

func bbRandName(rng *rand.Rand) string {
	return bbNames[rng.Intn(len(bbNames))] + fmt.Sprint(rng.Intn(50))
}

func bbRandStatus(rng *rand.Rand) common.Status {
	st := []common.Status{
		common.StatusDeployed, common.StatusSuperseded, common.StatusUninstalled,
		common.StatusFailed, common.StatusPendingInstall,
	}
	return st[rng.Intn(len(st))]
}

// ---------------------------------------------------------------------------
// properties
// ---------------------------------------------------------------------------

// Pinned rows from memory_test.go + contract.md sentences.
func TestMemoryDriverContractTableProperty(t *testing.T) {
	mem := driver.NewMemory()
	if mem.Name() != driver.MemoryDriverName {
		t.Fatalf("Name() = %q want %q", mem.Name(), driver.MemoryDriverName)
	}

	// create/get round-trip
	r := bbRelease("solo", 1, "default", common.StatusDeployed, nil)
	if err := mem.Create(bbKey("solo", 1), r); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := mem.Get(bbKey("solo", 1))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if bbAsV1(t, got) != r {
		t.Fatal("get mismatch")
	}
	if err := mem.Create(bbKey("solo", 1), r); !errors.Is(err, driver.ErrReleaseExists) {
		t.Fatalf("dup create err = %v", err)
	}
	_, err = mem.Get("ghost.v9")
	if !errors.Is(err, driver.ErrReleaseNotFound) {
		t.Fatalf("missing get err = %v", err)
	}

	mem, ref := bbLoadFixture(t)

	type getRow struct {
		key, ns string
		miss    bool
	}
	for _, row := range []getRow{
		{"rls-a.v1", "default", false},
		{"rls-a.v5", "default", true},
		{"rls-c.v1", "mynamespace", false},
		{"rls-a.v1", "mynamespace", true},
	} {
		mem.SetNamespace(row.ns)
		_, err := mem.Get(row.key)
		if row.miss && err == nil {
			t.Fatalf("expected miss for %s ns=%s", row.key, row.ns)
		}
		if !row.miss && err != nil {
			t.Fatalf("expected hit for %s ns=%s: %v", row.key, row.ns, err)
		}
	}

	mem.SetNamespace("default")
	dpl, err := mem.List(func(rel release.Releaser) bool {
		return bbAsV1(t, rel).Info.Status == common.StatusDeployed
	})
	if err != nil || len(dpl) != 2 {
		t.Fatalf("deployed list: n=%d err=%v", len(dpl), err)
	}
	ssd, err := mem.List(func(rel release.Releaser) bool {
		return bbAsV1(t, rel).Info.Status == common.StatusSuperseded
	})
	if err != nil || len(ssd) != 6 {
		t.Fatalf("superseded list: n=%d err=%v", len(ssd), err)
	}

	for _, row := range []struct {
		ns string
		n  int
	}{
		{"default", 2},
		{"mynamespace", 1},
	} {
		mem.SetNamespace(row.ns)
		q, err := mem.Query(map[string]string{"status": "deployed"})
		if err != nil || len(q) != row.n {
			t.Fatalf("query deployed ns=%s: n=%d err=%v", row.ns, len(q), err)
		}
		want := ref.Query(row.ns, map[string]string{"status": "deployed"})
		if !bbRelSliceEqual(q, relSlice(want)) {
			t.Fatalf("query oracle mismatch ns=%s", row.ns)
		}
	}

	// update rows (run before delete; delete uses a fresh fixture below)
	mem.SetNamespace("default")
	up := bbRelease("rls-a", 4, "default", common.StatusSuperseded, nil)
	if err := mem.Update("rls-a.v4", up); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := mem.Update("rls-c.v1", bbRelease("rls-c", 1, "default", common.StatusUninstalled, nil)); err == nil {
		t.Fatal("update absent key should error")
	}
	mem.SetNamespace("mynamespace")
	if err := mem.Update("rls-c.v4", bbRelease("rls-c", 4, "mynamespace", common.StatusSuperseded, nil)); err != nil {
		t.Fatalf("update ns: %v", err)
	}

	// delete rows — fresh fixture so deployed counts match in-tree TestMemoryDelete
	mem, ref = bbLoadFixture(t)
	mem.SetNamespace("")
	start, err := mem.Query(map[string]string{"status": "deployed"})
	if err != nil {
		t.Fatalf("query start: %v", err)
	}
	startLen := len(start)
	for _, row := range []struct {
		key, ns string
		err     bool
	}{
		{"rls-a.v4", "default", false},
		{"rls-a.v5", "default", true},
		{"rls-c.v4", "default", true},
		{"rls-c.v4", "mynamespace", false},
		{"rls-c.v5", "mynamespace", true},
		{"rls-a.v4", "mynamespace", true},
	} {
		mem.SetNamespace(row.ns)
		_, err := mem.Delete(row.key)
		if row.err && err == nil {
			t.Fatalf("expected delete error for %s ns=%s", row.key, row.ns)
		}
		if !row.err && err != nil {
			t.Fatalf("delete %s ns=%s: %v", row.key, row.ns, err)
		}
	}
	mem.SetNamespace("")
	end, err := mem.Query(map[string]string{"status": "deployed"})
	if err != nil {
		t.Fatalf("query end: %v", err)
	}
	if len(end) != startLen-2 {
		t.Fatalf("deleted count: start=%d end=%d", startLen, len(end))
	}
}

func relSlice(rs []*rspb.Release) []release.Releaser {
	out := make([]release.Releaser, len(rs))
	for i, r := range rs {
		out[i] = r
	}
	return out
}

// Seeded random CRUD sequences agree with the reference store (>=10k cases).
func TestMemoryDriverCRUDProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		mem := driver.NewMemory()
		ref := newBBStore()
		mem.SetNamespace("")
		nOps := 1 + rng.Intn(8)
		for j := 0; j < nOps; j++ {
			op := rng.Intn(4)
			name := bbRandName(rng)
			ver := 1 + rng.Intn(6)
			ns := bbNS[rng.Intn(len(bbNS))]
			key := bbKey(name, ver)
			rls := bbRelease(name, ver, ns, bbRandStatus(rng), nil)
			switch op {
			case 0: // create
				errM := mem.Create(key, rls)
				errR := ref.Create(key, rls)
				if (errM == nil) != (errR == nil) {
					t.Fatalf("case %d create err mismatch: mem=%v ref=%v", i, errM, errR)
				}
			case 1: // get
				mem.SetNamespace(ns)
				gm, errM := mem.Get(key)
				gr, errR := ref.Get(ns, key)
				if (errM == nil) != (errR == nil) {
					t.Fatalf("case %d get err mismatch", i)
				}
				if errM == nil && bbAsV1(t, gm).Version != gr.Version {
					t.Fatalf("case %d get version", i)
				}
			case 2: // update
				mem.SetNamespace(ns)
				errM := mem.Update(key, rls)
				errR := ref.Update(ns, key, rls)
				if (errM == nil) != (errR == nil) {
					t.Fatalf("case %d update err mismatch", i)
				}
			case 3: // delete
				mem.SetNamespace(ns)
				dm, errM := mem.Delete(key)
				dr, errR := ref.Delete(ns, key)
				if (errM == nil) != (errR == nil) {
					t.Fatalf("case %d delete err mismatch", i)
				}
				if errM == nil && bbAsV1(t, dm).Version != dr.Version {
					t.Fatalf("case %d delete returned version", i)
				}
			}
		}
	}
}

// List/Query label matching on random stores (>=10k cases).
func TestMemoryDriverListQueryProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		mem := driver.NewMemory()
		ref := newBBStore()
		nRel := 1 + rng.Intn(6)
		seen := map[string]int{}
		for j := 0; j < nRel; j++ {
			name := bbRandName(rng)
			ver := 1 + rng.Intn(4)
			ns := bbNS[rng.Intn(3)] // default, mynamespace, other-ns
			key := bbKey(name, ver)
			if seen[key] > 0 {
				continue
			}
			seen[key] = 1
			st := bbRandStatus(rng)
			rls := bbRelease(name, ver, ns, st, map[string]string{
				"tag": fmt.Sprint(rng.Intn(5)),
			})
			if err := mem.Create(key, rls); err != nil {
				continue
			}
			ref.Create(key, rls)
		}
		ns := bbNS[rng.Intn(len(bbNS))]
		mem.SetNamespace(ns)
		st := bbRandStatus(rng)
		q := map[string]string{"status": st.String()}
		gotQ, err := mem.Query(q)
		wantQ := ref.Query(ns, q)
		if len(wantQ) == 0 {
			if err == nil && len(gotQ) != 0 {
				t.Fatalf("case %d query expected empty", i)
			}
			continue
		}
		if err != nil {
			t.Fatalf("case %d query: %v", i, err)
		}
		if !bbRelSliceEqual(gotQ, relSlice(wantQ)) {
			t.Fatalf("case %d query mismatch ns=%s", i, ns)
		}
		gotL, err := mem.List(func(rel release.Releaser) bool {
			return bbAsV1(t, rel).Info.Status == st
		})
		if err != nil {
			t.Fatalf("case %d list: %v", i, err)
		}
		wantL := ref.List(ns, func(r *rspb.Release) bool { return r.Info.Status == st })
		if len(gotL) != len(wantL) {
			t.Fatalf("case %d list len %d want %d", i, len(gotL), len(wantL))
		}
	}
}

// Adversarial corners: duplicate versions, cross-namespace isolation, empty ns.
func TestMemoryDriverAdversarialProperty(t *testing.T) {
	for i := 0; i < bbCases; i++ {
		mem := driver.NewMemory()
		name := "adv-" + fmt.Sprint(i)
		r1 := bbRelease(name, 1, "default", common.StatusDeployed, nil)
		r2 := bbRelease(name, 1, "mynamespace", common.StatusDeployed, nil)
		if err := mem.Create(bbKey(name, 1), r1); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		mem.SetNamespace("mynamespace")
		if _, err := mem.Get(bbKey(name, 1)); err == nil {
			t.Fatalf("case %d: other ns should miss default release", i)
		}
		if err := mem.Create(bbKey(name, 1), r2); err != nil {
			t.Fatalf("case %d: cross-ns create should succeed: %v", i, err)
		}
		if _, err := mem.Get(bbKey(name, 1)); err != nil {
			t.Fatalf("case %d: mynamespace should hit after create", i)
		}
		mem.SetNamespace("default")
		if _, err := mem.Get(bbKey(name, 1)); err != nil {
			t.Fatalf("case %d: right ns should hit", i)
		}
		// update preserves key position: bump status then read back
		r1.Info.Status = common.StatusSuperseded
		if err := mem.Update(bbKey(name, 1), r1); err != nil {
			t.Fatalf("case %d update: %v", i, err)
		}
		got, _ := mem.Get(bbKey(name, 1))
		if bbAsV1(t, got).Info.Status != common.StatusSuperseded {
			t.Fatalf("case %d status not updated", i)
		}
		// delete returns removed release
		del, err := mem.Delete(bbKey(name, 1))
		if err != nil || bbAsV1(t, del).Name != name {
			t.Fatalf("case %d delete return", i)
		}
		if _, err := mem.Get(bbKey(name, 1)); err == nil {
			t.Fatalf("case %d get after delete", i)
		}
		// query newest revision by name + status
		mem.SetNamespace("default")
		for v := 1; v <= 3; v++ {
			rls := bbRelease(name, v, "default", common.StatusDeployed, nil)
			mem.Create(bbKey(name, v), rls)
		}
		q, err := mem.Query(map[string]string{"status": "deployed", "name": name})
		if err != nil {
			t.Fatalf("case %d query err: %v", i, err)
		}
		if len(q) == 0 {
			t.Fatalf("case %d newest query want >=1 got 0", i)
		}
		maxV := 0
		for _, rel := range q {
			if v := bbAsV1(t, rel).Version; v > maxV {
				maxV = v
			}
		}
		if maxV != 3 {
			t.Fatalf("case %d newest version got max %d", i, maxV)
		}
	}
}

// Unseen-random names/namespaces/labels outside fixture vocabulary.
func TestMemoryDriverUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	lits := []string{"qzx", "wobble", "kraken", "nantucket", "pequod", "ishmael"}
	for i := 0; i < bbCases; i++ {
		mem := driver.NewMemory()
		name := lits[rng.Intn(len(lits))] + "-" + fmt.Sprint(i)
		ns := lits[rng.Intn(len(lits))] + "-ns"
		ver := 1 + rng.Intn(9)
		st := []common.Status{common.StatusDeployed, common.StatusFailed}[rng.Intn(2)]
		rls := bbRelease(name, ver, ns, st, map[string]string{"fleet": lits[rng.Intn(len(lits))]})
		key := bbKey(name, ver)
		if err := mem.Create(key, rls); err != nil {
			t.Fatalf("case %d create: %v", i, err)
		}
		mem.SetNamespace(ns)
		g, err := mem.Get(key)
		if err != nil || bbAsV1(t, g).Namespace != ns {
			t.Fatalf("case %d get ns", i)
		}
		// List with empty namespace sees all namespaces (in-tree SetNamespace comment).
		mem.SetNamespace("")
		all, err := mem.List(func(rel release.Releaser) bool {
			return bbAsV1(t, rel).Name == name
		})
		if err != nil {
			t.Fatalf("case %d list all: %v", i, err)
		}
		if len(all) < 1 {
			t.Fatalf("case %d expected global list hit", i)
		}
	}
}

// Concurrent readers/writers must not panic (smoke property).
func TestMemoryDriverConcurrentProperty(t *testing.T) {
	mem := driver.NewMemory()
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(bbSeed + int64(id*bbCases)))
			for i := 0; i < bbCases/8; i++ {
				name := fmt.Sprintf("c-%d-%d", id, rng.Intn(100000))
				rls := bbRelease(name, 1, "default", common.StatusDeployed, nil)
				key := bbKey(name, 1)
				_ = mem.Create(key, rls)
				mem.SetNamespace("default")
				_, _ = mem.Get(key)
				_, _ = mem.List(func(rel release.Releaser) bool { return true })
				_, _ = mem.Query(map[string]string{"status": "deployed"})
			}
		}(g)
	}
	wg.Wait()
}

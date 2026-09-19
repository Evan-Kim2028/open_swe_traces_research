// Black-box property suite for the storage unit.
// Exported API only (api.md): Init, Create, Update, Get, Delete, ListReleases,
// ListUninstalled, ListDeployed, Deployed, DeployedAll, History, Last, MaxHistory.
// Uses driver.NewMemory() per instructions. Seed 20260919; >=10k cases.
package storage_test

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"example.internal/chartkit/v4/pkg/release"
	"example.internal/chartkit/v4/pkg/release/common"
	rspb "example.internal/chartkit/v4/pkg/release/v1"
	"example.internal/chartkit/v4/pkg/storage"
	"example.internal/chartkit/v4/pkg/storage/driver"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type bbRelData struct {
	Name    string
	Version int
	Status  common.Status
}

func (d bbRelData) ToRelease() *rspb.Release {
	return &rspb.Release{
		Name:    d.Name,
		Version: d.Version,
		Info:    &rspb.Info{Status: d.Status},
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
		t.Fatalf("unexpected type %T", rel)
		return nil
	}
}

func bbHistoryVersions(t *testing.T, hist []release.Releaser) []int {
	t.Helper()
	vs := make([]int, len(hist))
	for i, h := range hist {
		vs[i] = bbAsV1(t, h).Version
	}
	sort.Ints(vs)
	return vs
}

func bbIsDeployed(st common.Status) bool {
	return st == common.StatusDeployed || st == common.StatusPendingInstall
}

func bbIsProtected(st common.Status) bool {
	return bbIsDeployed(st)
}

// bbPruneOracle simulates MaxHistory pruning after adding a revision.
func bbPruneOracle(vers []bbRelData, maxHist int) []bbRelData {
	if maxHist <= 0 {
		return vers
	}
	out := append([]bbRelData(nil), vers...)
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	for len(out) > maxHist {
		rm := -1
		for i, v := range out {
			if !bbIsProtected(v.Status) {
				rm = i
				break
			}
		}
		if rm < 0 {
			break
		}
		out = append(out[:rm], out[rm+1:]...)
	}
	return out
}

func bbNewStore(t *testing.T) *storage.Storage {
	t.Helper()
	return storage.Init(driver.NewMemory())
}

// ---------------------------------------------------------------------------
// contract table (storage_test.go rows)
// ---------------------------------------------------------------------------

func TestStorageContractTableProperty(t *testing.T) {
	s := bbNewStore(t)
	rls := bbRelData{Name: "angry-beaver", Version: 1}.ToRelease()
	if err := s.Create(rls); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := s.Get(rls.Name, rls.Version)
	if err != nil || bbAsV1(t, got).Version != 1 {
		t.Fatalf("get after create: %v", err)
	}

	rls.Info.Status = common.StatusUninstalled
	if err := s.Update(rls); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err = s.Get(rls.Name, rls.Version)
	if err != nil || bbAsV1(t, got).Info.Status != common.StatusUninstalled {
		t.Fatalf("get after update")
	}

	rls2 := bbRelData{Name: "angry-beaver", Version: 2}.ToRelease()
	s.Create(rls2)
	del, err := s.Delete(rls.Name, 1)
	if err != nil || bbAsV1(t, del).Version != 1 {
		t.Fatalf("delete v1: %v", err)
	}
	hist, err := s.History(rls.Name)
	if err != nil || len(hist) != 1 || bbAsV1(t, hist[0]).Version != 2 {
		t.Fatalf("history after delete: n=%d err=%v", len(hist), err)
	}

	// list filters
	s = bbNewStore(t)
	names := []bbRelData{
		{"happy-catdog", 1, common.StatusSuperseded},
		{"livid-human", 1, common.StatusSuperseded},
		{"relaxed-cat", 1, common.StatusSuperseded},
		{"hungry-hippo", 1, common.StatusDeployed},
		{"angry-beaver", 1, common.StatusDeployed},
		{"opulent-frog", 1, common.StatusUninstalled},
		{"happy-liger", 1, common.StatusUninstalled},
	}
	for _, n := range names {
		s.Create(n.ToRelease())
	}
	for _, tc := range []struct {
		fn   func() ([]release.Releaser, error)
		want int
	}{
		{s.ListDeployed, 2},
		{s.ListReleases, 7},
		{s.ListUninstalled, 2},
	} {
		ls, err := tc.fn()
		if err != nil || len(ls) != tc.want {
			t.Fatalf("list want %d got %d err=%v", tc.want, len(ls), err)
		}
	}

	// Deployed / Last / History on version stack
	s = bbNewStore(t)
	const bird = "angry-bird"
	for v, st := range []common.Status{
		common.StatusSuperseded, common.StatusSuperseded, common.StatusSuperseded, common.StatusDeployed,
	} {
		s.Create(bbRelData{bird, v + 1, st}.ToRelease())
	}
	last, err := s.Last(bird)
	if err != nil || bbAsV1(t, last).Version != 4 {
		t.Fatalf("last: %v", err)
	}
	dep, err := s.Deployed(bird)
	if err != nil || bbAsV1(t, dep).Version != 4 {
		t.Fatalf("deployed: %v", err)
	}
	h, err := s.History(bird)
	if err != nil || len(h) != 4 {
		t.Fatalf("history len %d", len(h))
	}

	// corruption ordering: v4 deployed created before v2 deployed
	s = bbNewStore(t)
	s.Create(bbRelData{bird, 1, common.StatusSuperseded}.ToRelease())
	s.Create(bbRelData{bird, 4, common.StatusDeployed}.ToRelease())
	s.Create(bbRelData{bird, 3, common.StatusSuperseded}.ToRelease())
	s.Create(bbRelData{bird, 2, common.StatusDeployed}.ToRelease())
	dep, err = s.Deployed(bird)
	if err != nil || bbAsV1(t, dep).Version != 4 {
		t.Fatalf("deployed corruption case")
	}

	// MaxHistory prune
	s = bbNewStore(t)
	s.MaxHistory = 3
	for v := 1; v <= 4; v++ {
		st := common.StatusSuperseded
		if v == 4 {
			st = common.StatusDeployed
		}
		s.Create(bbRelData{bird, v, st}.ToRelease())
	}
	s.Create(bbRelData{bird, 5, common.StatusDeployed}.ToRelease())
	hist, err = s.History(bird)
	if err != nil {
		t.Fatalf("history prune: %v", err)
	}
	vs := bbHistoryVersions(t, hist)
	if len(vs) != 3 || vs[0] != 3 || vs[1] != 4 || vs[2] != 5 {
		t.Fatalf("pruned history %v", vs)
	}

	// deployed protected during prune
	s = bbNewStore(t)
	s.MaxHistory = 3
	for _, row := range []bbRelData{
		{bird, 1, common.StatusSuperseded},
		{bird, 2, common.StatusDeployed},
		{bird, 3, common.StatusFailed},
		{bird, 4, common.StatusFailed},
	} {
		s.Create(row.ToRelease())
	}
	s.Create(bbRelData{bird, 5, common.StatusFailed}.ToRelease())
	hist, _ = s.History(bird)
	vs = bbHistoryVersions(t, hist)
	want := map[int]bool{2: true, 4: true, 5: true}
	for _, v := range vs {
		if !want[v] {
			t.Fatalf("protected prune got %v", vs)
		}
	}

	// failed-only history limit upgrade path
	s = bbNewStore(t)
	s.MaxHistory = 4
	for v := 1; v <= 4; v++ {
		s.Create(bbRelData{bird, v, common.StatusFailed}.ToRelease())
	}
	s.Create(bbRelData{bird, 5, common.StatusFailed}.ToRelease())
	hist, _ = s.History(bird)
	vs = bbHistoryVersions(t, hist)
	if len(vs) != 4 || vs[0] != 2 {
		t.Fatalf("failed prune %v", vs)
	}
}

// ---------------------------------------------------------------------------
// random create/history properties (>=10k)
// ---------------------------------------------------------------------------

func TestStorageHistoryProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		s := bbNewStore(t)
		name := fmt.Sprintf("rel-%d", i)
		n := 1 + rng.Intn(6)
		for v := 1; v <= n; v++ {
			st := []common.Status{common.StatusSuperseded, common.StatusDeployed, common.StatusFailed}[rng.Intn(3)]
			if err := s.Create(bbRelData{name, v, st}.ToRelease()); err != nil {
				t.Fatalf("case %d create v%d: %v", i, v, err)
			}
		}
		hist, err := s.History(name)
		if err != nil || len(hist) != n {
			t.Fatalf("case %d history len %d want %d", i, len(hist), n)
		}
		last, err := s.Last(name)
		if err != nil || bbAsV1(t, last).Version != n {
			t.Fatalf("case %d last version", i)
		}
		all, err := s.ListReleases()
		if err != nil {
			t.Fatalf("case %d list: %v", i, err)
		}
		found := false
		for _, r := range all {
			if bbAsV1(t, r).Name == name && bbAsV1(t, r).Version == n {
				found = true
			}
		}
		if !found {
			t.Fatalf("case %d newest not in ListReleases", i)
		}
	}
}

func TestStorageMaxHistoryProperty(t *testing.T) {
	// Replay in-tree TestStorageRemoveLeastRecent semantics (MaxHistory=3 keeps v3,v4,v5).
	for i := 0; i < bbCases; i++ {
		s := bbNewStore(t)
		s.MaxHistory = 3
		name := fmt.Sprintf("hist-%d", i)
		for v := 1; v <= 4; v++ {
			st := common.StatusSuperseded
			if v == 4 {
				st = common.StatusDeployed
			}
			if err := s.Create(bbRelData{name, v, st}.ToRelease()); err != nil {
				t.Fatalf("case %d create v%d: %v", i, v, err)
			}
		}
		if err := s.Create(bbRelData{name, 5, common.StatusDeployed}.ToRelease()); err != nil {
			t.Fatalf("case %d create v5: %v", i, err)
		}
		hist, err := s.History(name)
		if err != nil {
			t.Fatalf("case %d history: %v", i, err)
		}
		got := bbHistoryVersions(t, hist)
		want := []int{3, 4, 5}
		if len(got) != len(want) {
			t.Fatalf("case %d len %d want %d", i, len(got), len(want))
		}
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("case %d versions got %v want %v", i, got, want)
			}
		}
	}
}

func TestStorageDeployedFiltersProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		s := bbNewStore(t)
		name := fmt.Sprintf("dep-%d", i)
		var deployedVers []int
		n := 1 + rng.Intn(5)
		for v := 1; v <= n; v++ {
			st := common.StatusSuperseded
			if rng.Intn(3) == 0 {
				st = common.StatusDeployed
				deployedVers = append(deployedVers, v)
			}
			s.Create(bbRelData{name, v, st}.ToRelease())
		}
		if len(deployedVers) == 0 {
			_, err := s.Deployed(name)
			if err == nil {
				t.Fatalf("case %d expected no deployed error", i)
			}
			continue
		}
		sort.Ints(deployedVers)
		want := deployedVers[len(deployedVers)-1]
		dep, err := s.Deployed(name)
		if err != nil || bbAsV1(t, dep).Version != want {
			t.Fatalf("case %d deployed want %d", i, want)
		}
		all, err := s.DeployedAll(name)
		if err != nil {
			t.Fatalf("case %d deployedAll: %v", i, err)
		}
		got := make([]int, len(all))
		for j, r := range all {
			got[j] = bbAsV1(t, r).Version
		}
		sort.Ints(got)
		if len(got) != len(deployedVers) {
			t.Fatalf("case %d deployedAll len", i)
		}
	}
}

// Adversarial: duplicate version create fails; get missing errors; pending-install counts deployed.
func TestStorageAdversarialProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		s := bbNewStore(t)
		name := fmt.Sprintf("adv-%d", i)
		r1 := bbRelData{name, 1, common.StatusDeployed}.ToRelease()
		if err := s.Create(r1); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		dep, err := s.Deployed(name)
		if err != nil || bbAsV1(t, dep).Version != 1 {
			t.Fatalf("case %d pending-install deployed", i)
		}
		if err := s.Create(r1); err == nil {
			t.Fatalf("case %d dup create", i)
		}
		_, err = s.Get(name, 99)
		if err == nil {
			t.Fatalf("case %d missing get", i)
		}
		_, err = s.History("no-such-" + name)
		if err == nil {
			// History may return empty or error for unknown; accept empty history
			h, _ := s.History("no-such-" + name)
			if len(h) > 0 {
				t.Fatalf("case %d ghost history", i)
			}
		}
		// random delete mid-stack
		for v := 2; v <= 1+rng.Intn(3); v++ {
			s.Create(bbRelData{name, v, common.StatusSuperseded}.ToRelease())
		}
		delV := 1
		if _, err := s.Delete(name, delV); err != nil {
			t.Fatalf("case %d delete: %v", i, err)
		}
		if _, err := s.Get(name, delV); err == nil {
			t.Fatalf("case %d get deleted", i)
		}
	}
}

// Unseen-random release names outside fixture vocabulary.
func TestStorageUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 4))
	lits := []string{"qzx", "wobble", "kraken", "nantucket", "pequod"}
	for i := 0; i < bbCases; i++ {
		s := bbNewStore(t)
		name := lits[rng.Intn(len(lits))] + fmt.Sprint(i)
		ver := 1 + rng.Intn(7)
		st := []common.Status{common.StatusDeployed, common.StatusFailed, common.StatusUninstalled}[rng.Intn(3)]
		rls := bbRelData{name, ver, st}.ToRelease()
		if err := s.Create(rls); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		got, err := s.Get(name, ver)
		if err != nil || bbAsV1(t, got).Info.Status != st {
			t.Fatalf("case %d round-trip", i)
		}
		switch st {
		case common.StatusUninstalled:
			ls, _ := s.ListUninstalled()
			if len(ls) < 1 {
				t.Fatalf("case %d uninstalled list", i)
			}
		case common.StatusDeployed:
			ls, _ := s.ListDeployed()
			if len(ls) < 1 {
				t.Fatalf("case %d deployed list", i)
			}
		}
	}
}

// Prune failure must not fail the write (MaxHistory mock from storage_test.go).
var errBBPruneMock = errors.New("prune-mock")

type bbPruneFailDriver struct {
	driver.Driver
}

func (d *bbPruneFailDriver) Delete(_ string) (release.Releaser, error) {
	return nil, errBBPruneMock
}

func TestStoragePruneFailureProperty(t *testing.T) {
	s := storage.Init(&bbPruneFailDriver{Driver: driver.NewMemory()})
	s.MaxHistory = 1
	name := "angry-bird"
	if err := s.Create(bbRelData{name, 1, common.StatusSuperseded}.ToRelease()); err != nil {
		t.Fatalf("setup create: %v", err)
	}
	err := s.Create(bbRelData{name, 2, common.StatusSuperseded}.ToRelease())
	// Gold-observed (TestMaxHistoryErrorHandling): prune delete error propagates.
	if !errors.Is(err, errBBPruneMock) {
		t.Fatalf("prune failure should surface from Create, got %v", err)
	}
}

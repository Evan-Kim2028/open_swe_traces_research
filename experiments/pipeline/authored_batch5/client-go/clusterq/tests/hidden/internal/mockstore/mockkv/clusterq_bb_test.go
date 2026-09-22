package mocktikv

import (
	"context"
	"errors"
	"testing"

	"github.com/pingcap/kvproto/pkg/metapb"
)

// Hidden suite for unit clusterq. One TestDetailNN per DETAILS.md line.

func bbNewCluster(t *testing.T) *Cluster {
	t.Helper()
	return NewCluster(nil)
}

// TestDetail01: GetRegionByKey returns the containing region, its leader,
// buckets, and down peers filtered to that region's peers.
func TestDetail01(t *testing.T) {
	c := bbNewCluster(t)
	c.AddStore(1, "s1")
	c.AddStore(2, "s2")
	c.Bootstrap(1, []uint64{1, 2}, []uint64{10, 20}, 10)
	meta, leader, _, down := c.GetRegionByKey(NewMvccKey([]byte("x")))
	if meta == nil || meta.GetId() != 1 {
		t.Fatalf("region for key: %v", meta)
	}
	if leader == nil || leader.GetId() != 10 {
		t.Fatalf("leader: %v", leader)
	}
	if len(down) != 0 {
		t.Fatalf("down peers on fresh cluster: %v", down)
	}
	c.SplitRaw(1, 2, NewMvccKey([]byte("k")), []uint64{30, 40}, 30)
	meta, leader, _, _ = c.GetRegionByKey(NewMvccKey([]byte("z")))
	if meta == nil || meta.GetId() != 2 || leader.GetId() != 30 {
		t.Fatalf("right-half lookup: region=%v leader=%v", meta, leader)
	}
	meta, _, _, _ = c.GetRegionByKey(NewMvccKey([]byte("a")))
	if meta == nil || meta.GetId() != 1 {
		t.Fatalf("left-half lookup: %v", meta)
	}
	c.MarkPeerDown(30)
	_, _, _, down = c.GetRegionByKey(NewMvccKey([]byte("z")))
	if len(down) != 1 || down[0].GetId() != 30 {
		t.Fatalf("down peers in region 2: %v", down)
	}
	_, _, _, down = c.GetRegionByKey(NewMvccKey([]byte("a")))
	if len(down) != 0 {
		t.Fatalf("foreign down peer leaked into region 1: %v", down)
	}
}

// TestDetail02: GetPrevRegionByKey returns the region whose EndKey equals
// the containing region's StartKey; nils at the boundary-less head; panics
// when no region contains the key.
func TestDetail02(t *testing.T) {
	c := bbNewCluster(t)
	c.AddStore(1, "s1")
	c.Bootstrap(1, []uint64{1}, []uint64{10}, 10)
	meta, _, _, _ := c.GetPrevRegionByKey(NewMvccKey([]byte("x")))
	if meta != nil {
		t.Fatalf("prev of unbounded region: %v", meta)
	}
	c.SplitRaw(1, 2, NewMvccKey([]byte("k")), []uint64{20}, 20)
	prev, _, _, _ := c.GetPrevRegionByKey(NewMvccKey([]byte("z")))
	if prev == nil || prev.GetId() != 1 {
		t.Fatalf("prev region = %v, want region 1", prev)
	}
	// No region contains the key: committed panic (missing nil guard).
	empty := bbNewCluster(t)
	func() {
		defer func() {
			if recover() == nil {
				t.Fatalf("GetPrevRegionByKey on empty cluster did not panic")
			}
		}()
		empty.GetPrevRegionByKey(NewMvccKey([]byte("x")))
	}()
}

// TestDetail03: ScanRegions is sorted by StartKey, filters to the range,
// excludes the region ending exactly at startKey, applies limit after
// filtering, and emits a non-nil empty leader for leaderless regions.
func TestDetail03(t *testing.T) {
	c := bbNewCluster(t)
	c.AddStore(1, "s1")
	c.Bootstrap(1, []uint64{1}, []uint64{10}, 10)
	k1, k2 := NewMvccKey([]byte("k")), NewMvccKey([]byte("t"))
	c.SplitRaw(1, 2, k1, []uint64{20}, 20)
	c.SplitRaw(2, 3, k2, []uint64{30}, 30)
	rs := c.ScanRegions(nil, nil, 0)
	if len(rs) != 3 {
		t.Fatalf("ScanRegions returned %d regions", len(rs))
	}
	for i := 1; i < len(rs); i++ {
		if string(rs[i-1].Meta.GetStartKey()) > string(rs[i].Meta.GetStartKey()) {
			t.Fatalf("not sorted by StartKey: %q > %q", rs[i-1].Meta.GetStartKey(), rs[i].Meta.GetStartKey())
		}
	}
	// Region ending exactly at startKey is excluded.
	rs = c.ScanRegions(k1, nil, 0)
	for _, r := range rs {
		if r.Meta.GetId() == 1 {
			t.Fatalf("region ending at startKey included")
		}
	}
	if len(rs) != 2 {
		t.Fatalf("range filter: %d regions", len(rs))
	}
	// Limit applies after filtering.
	rs = c.ScanRegions(nil, nil, 1)
	if len(rs) != 1 {
		t.Fatalf("limit: %d regions", len(rs))
	}
	// Leaderless region yields a non-nil empty peer.
	c.GiveUpLeader(1)
	rs = c.ScanRegions(nil, k1, 0)
	if len(rs) != 1 || rs[0].Leader == nil || rs[0].Leader.GetId() != 0 {
		t.Fatalf("leaderless scan: %+v", rs)
	}
}

// TestDetail04: a cancelled store fails GetAndCheckStoreByAddr even when it
// does not match the address.
func TestDetail04(t *testing.T) {
	c := bbNewCluster(t)
	c.AddStore(1, "a")
	c.AddStore(2, "b")
	c.CancelStore(2)
	if _, err := c.GetAndCheckStoreByAddr("a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled sibling not detected: %v", err)
	}
	c.UnCancelStore(2)
	ss, err := c.GetAndCheckStoreByAddr("a")
	if err != nil || len(ss) != 1 || ss[0].GetAddress() != "a" {
		t.Fatalf("after uncancel: %v %v", ss, err)
	}
}

// TestDetail05: returned metas are deep copies — mutating them must not
// corrupt cluster state.
func TestDetail05(t *testing.T) {
	c := bbNewCluster(t)
	c.AddStore(1, "s1")
	c.Bootstrap(1, []uint64{1}, []uint64{10}, 10)
	c.GetStore(1).Address = "hacked"
	if c.GetStore(1).GetAddress() != "s1" {
		t.Fatalf("GetStore returned shared meta")
	}
	meta, _, _, _ := c.GetRegionByKey(NewMvccKey([]byte("x")))
	meta.StartKey = []byte("corrupt")
	meta2, _, _, _ := c.GetRegionByKey(NewMvccKey([]byte("x")))
	if len(meta2.GetStartKey()) != 0 {
		t.Fatalf("region meta mutation leaked: %q", meta2.GetStartKey())
	}
	rs := c.ScanRegions(nil, nil, 0)
	rs[0].Meta.Id = 999
	if c.ScanRegions(nil, nil, 0)[0].Meta.GetId() == 999 {
		t.Fatalf("ScanRegions returned shared meta")
	}
}

// TestDetail06: MarkTombstone panics on a missing store; Stop/Start/Cancel
// are silent no-ops on missing ids.
func TestDetail06(t *testing.T) {
	c := bbNewCluster(t)
	c.AddStore(1, "s1")
	for _, f := range []func(uint64){c.StopStore, c.StartStore, c.CancelStore} {
		f(999) // must not panic
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatalf("MarkTombstone on missing store did not panic")
			}
		}()
		c.MarkTombstone(999)
	}()
	c.MarkTombstone(1)
	if c.GetStore(1).GetState() != metapb.StoreState_Tombstone {
		t.Fatalf("store not tombstoned: %v", c.GetStore(1).GetState())
	}
}

// TestDetail07: AllocID counts up from 1; AllocIDs continues the counter.
func TestDetail07(t *testing.T) {
	c := bbNewCluster(t)
	if c.AllocID() != 1 || c.AllocID() != 2 {
		t.Fatalf("AllocID not sequential from 1")
	}
	ids := c.AllocIDs(3)
	if len(ids) != 3 || ids[0] != 3 || ids[2] != 5 {
		t.Fatalf("AllocIDs: %v", ids)
	}
	if c.AllocID() != 6 {
		t.Fatalf("counter did not continue after AllocIDs")
	}
}

// TestDetail08: MarkPeerDown is cluster-global but surfaces only when the
// peer belongs to the returned region; RemoveDownPeer clears it.
func TestDetail08(t *testing.T) {
	c := bbNewCluster(t)
	c.AddStore(1, "s1")
	c.Bootstrap(1, []uint64{1}, []uint64{10}, 10)
	c.SplitRaw(1, 2, NewMvccKey([]byte("k")), []uint64{20}, 20)
	c.MarkPeerDown(20)
	rs := c.ScanRegions(nil, nil, 0)
	for _, r := range rs {
		if r.Meta.GetId() == 1 && len(r.DownPeers) != 0 {
			t.Fatalf("region 1 saw foreign down peer: %v", r.DownPeers)
		}
		if r.Meta.GetId() == 2 && (len(r.DownPeers) != 1 || r.DownPeers[0].GetId() != 20) {
			t.Fatalf("region 2 down peers: %v", r.DownPeers)
		}
	}
	// A down peer that belongs to no region never surfaces.
	c.MarkPeerDown(999)
	for _, r := range c.ScanRegions(nil, nil, 0) {
		for _, p := range r.DownPeers {
			if p.GetId() == 999 {
				t.Fatalf("non-member down peer surfaced on region %d", r.Meta.GetId())
			}
		}
	}
	c.RemoveDownPeer(20)
	_, _, _, down := c.GetRegionByID(2)
	if len(down) != 0 {
		t.Fatalf("RemoveDownPeer left peers: %v", down)
	}
}

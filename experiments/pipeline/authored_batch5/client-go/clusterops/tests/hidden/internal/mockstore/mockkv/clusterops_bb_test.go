package mocktikv

import (
	"testing"
	"time"

	"github.com/pingcap/kvproto/pkg/metapb"
)

// Hidden suite for unit clusterops. One TestDetailNN per DETAILS.md line.

func bbCluster(t *testing.T) *Cluster {
	t.Helper()
	return NewCluster(nil)
}

func bbRegionEpoch(t *testing.T, c *Cluster, id uint64) (confVer, ver uint64) {
	t.Helper()
	meta, _ := c.GetRegion(id)
	if meta == nil {
		t.Fatalf("region %d missing", id)
	}
	return meta.GetRegionEpoch().GetConfVer(), meta.GetRegionEpoch().GetVersion()
}

// TestDetail01: peer mutations bump ConfVer; range mutations bump Version.
func TestDetail01(t *testing.T) {
	c := bbCluster(t)
	c.AddStore(1, "s1")
	c.Bootstrap(1, []uint64{1}, []uint64{1}, 1)
	cv0, v0 := bbRegionEpoch(t, c, 1)

	c.AddPeer(1, 1, 2)
	cv1, v1 := bbRegionEpoch(t, c, 1)
	if cv1 <= cv0 || v1 != v0 {
		t.Fatalf("AddPeer: confVer %d->%d version %d->%d", cv0, cv1, v0, v1)
	}
	c.RemovePeer(1, 2)
	cv2, v2 := bbRegionEpoch(t, c, 1)
	if cv2 <= cv1 || v2 != v1 {
		t.Fatalf("RemovePeer: confVer %d->%d version %d->%d", cv1, cv2, v1, v2)
	}

	c.SplitRaw(1, 2, NewMvccKey([]byte("k")), []uint64{3}, 3)
	cv3, v3 := bbRegionEpoch(t, c, 1)
	if v3 <= v2 || cv3 != cv2 {
		t.Fatalf("split: version %d->%d confVer %d->%d", v2, v3, cv2, cv3)
	}
	c.Merge(1, 2)
	cv4, v4 := bbRegionEpoch(t, c, 1)
	if v4 <= v3 || cv4 != cv3 {
		t.Fatalf("merge: version %d->%d confVer %d->%d", v3, v4, cv3, cv4)
	}
}

// TestDetail02: removing the leader leaves the region leaderless; removing a
// follower keeps the leader.
func TestDetail02(t *testing.T) {
	c := bbCluster(t)
	c.AddStore(1, "s1")
	c.AddStore(2, "s2")
	c.Bootstrap(1, []uint64{1, 2}, []uint64{1, 2}, 1)
	c.RemovePeer(1, 2)
	_, leader := c.GetRegion(1)
	if leader != 1 {
		t.Fatalf("removing follower changed leader to %d", leader)
	}
	c.RemovePeer(1, 1)
	_, leader = c.GetRegion(1)
	if leader != 0 {
		t.Fatalf("removing leader left leader=%d", leader)
	}
}

// TestDetail03: mismatched store/peer id lengths panic.
func TestDetail03(t *testing.T) {
	assertPanic := func(name string, f func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s did not panic", name)
			}
		}()
		f()
	}
	c := bbCluster(t)
	c.AddStore(1, "s1")
	c.AddStore(2, "s2")
	assertPanic("Bootstrap", func() {
		c.Bootstrap(1, []uint64{1, 2}, []uint64{1}, 1)
	})
	assertPanic("PutRegion", func() {
		c.PutRegion(2, 1, 1, []uint64{1}, []uint64{1, 2}, 1)
	})
	// Region.split panics when len(Peers) != len(peerIDs).
	c.Bootstrap(3, []uint64{1, 2}, []uint64{10, 20}, 10)
	assertPanic("Split", func() {
		c.SplitRaw(3, 4, NewMvccKey([]byte("k")), []uint64{30}, 30)
	})
}

// TestDetail04: SplitRaw carves [key, oldEnd) and truncates the source;
// peers inherit the source's stores positionally with caller peer IDs.
func TestDetail04(t *testing.T) {
	c := bbCluster(t)
	c.AddStore(1, "s1")
	c.AddStore(2, "s2")
	c.Bootstrap(1, []uint64{1, 2}, []uint64{1, 2}, 1)
	k := NewMvccKey([]byte("k"))
	nr := c.SplitRaw(1, 2, k, []uint64{10, 20}, 10)
	if nr == nil {
		t.Fatalf("SplitRaw returned nil")
	}
	if string(nr.GetStartKey()) != string(k) || len(nr.GetEndKey()) != 0 {
		t.Fatalf("new region range [%q, %q), want [%q, \"\")", nr.GetStartKey(), nr.GetEndKey(), k)
	}
	if len(nr.GetPeers()) != 2 {
		t.Fatalf("new region peers: %v", nr.GetPeers())
	}
	for i, p := range nr.GetPeers() {
		if p.GetId() != []uint64{10, 20}[i] || p.GetStoreId() != []uint64{1, 2}[i] {
			t.Fatalf("peer %d = %+v", i, p)
		}
	}
	src, _ := c.GetRegion(1)
	if string(src.GetEndKey()) != string(k) || len(src.GetStartKey()) != 0 {
		t.Fatalf("source range [%q, %q), want [\"\", %q)", src.GetStartKey(), src.GetEndKey(), k)
	}
}

// TestDetail05: UpdateStoreAddr sets both address fields to the same value;
// UpdateStorePeerAddr changes only the peer address.
func TestDetail05(t *testing.T) {
	c := bbCluster(t)
	c.AddStore(1, "addr1")
	c.UpdateStoreAddr(1, "addr2")
	st := c.GetStore(1)
	if st.GetAddress() != "addr2" || st.GetPeerAddress() != st.GetAddress() {
		t.Fatalf("UpdateStoreAddr: addr=%q peer=%q", st.GetAddress(), st.GetPeerAddress())
	}
	c.UpdateStorePeerAddr(1, "peerX")
	st = c.GetStore(1)
	if st.GetPeerAddress() != "peerX" || st.GetAddress() != "addr2" {
		t.Fatalf("UpdateStorePeerAddr: addr=%q peer=%q", st.GetAddress(), st.GetPeerAddress())
	}
}

// TestDetail06: label merge unions by key with new labels winning.
func TestDetail06(t *testing.T) {
	c := bbCluster(t)
	c.AddStore(1, "s1", &metapb.StoreLabel{Key: "zone", Value: "z1"}, &metapb.StoreLabel{Key: "host", Value: "h1"})
	c.UpdateStoreLabels(1, []*metapb.StoreLabel{{Key: "zone", Value: "z2"}, {Key: "rack", Value: "r1"}})
	lv := map[string]string{}
	for _, l := range c.GetStore(1).GetLabels() {
		lv[l.GetKey()] = l.GetValue()
	}
	if lv["zone"] != "z2" || lv["host"] != "h1" || lv["rack"] != "r1" {
		t.Fatalf("merged labels: %v", lv)
	}
	// A store with no labels takes the argument wholesale.
	c.AddStore(2, "s2")
	c.UpdateStoreLabels(2, []*metapb.StoreLabel{{Key: "a", Value: "b"}})
	st := c.GetStore(2)
	if len(st.GetLabels()) != 1 || st.GetLabels()[0].GetKey() != "a" {
		t.Fatalf("wholesale labels: %v", st.GetLabels())
	}
}

// TestDetail07: SplitKeys evacuates the range — count=3 on empty data
// yields 2 regions, not 3.
func TestDetail07(t *testing.T) {
	store := MustNewMVCCStore()
	defer store.Close()
	c := NewCluster(store)
	c.AddStore(1, "s1")
	c.Bootstrap(1, []uint64{1}, []uint64{1}, 1)
	c.SplitKeys([]byte("a"), []byte("z"), 3)
	regions := c.GetAllRegions()
	if len(regions) != 2 {
		t.Fatalf("SplitKeys produced %d regions, want 2", len(regions))
	}
	// The [a,z) range is evacuated: no region contains a mid key.
	mid := NewMvccKey([]byte("m"))
	meta, _, _, _ := c.GetRegionByKey(mid)
	if meta != nil {
		t.Fatalf("evacuated range still covered by region %d", meta.GetId())
	}
}

// TestDetail08: Merge extends region1 over region2's range and deletes
// region2; SplitRegionBuckets stores encoded keys; a delay is consumed once.
func TestDetail08(t *testing.T) {
	c := bbCluster(t)
	c.AddStore(1, "s1")
	c.Bootstrap(1, []uint64{1}, []uint64{1}, 1)
	c.SplitRaw(1, 2, NewMvccKey([]byte("k")), []uint64{2}, 2)
	c.Merge(1, 2)
	meta, _ := c.GetRegion(1)
	if len(meta.GetEndKey()) != 0 {
		t.Fatalf("merge did not extend region1's end key: %q", meta.GetEndKey())
	}
	if m, _ := c.GetRegion(2); m != nil {
		t.Fatalf("region2 survived merge")
	}

	c.SplitRegionBuckets(1, [][]byte{[]byte("b1"), []byte("b2")}, 7)
	_, _, buckets, _ := c.GetRegionByKey(NewMvccKey([]byte("x")))
	if buckets == nil || buckets.GetVersion() != 7 {
		t.Fatalf("buckets: %+v", buckets)
	}
	for _, bk := range buckets.GetKeys() {
		if string(bk) == "b1" || string(bk) == "b2" {
			t.Fatalf("bucket key stored unencoded: %q", bk)
		}
	}

	c.ScheduleDelay(100, 1, 150*time.Millisecond)
	start := time.Now()
	c.handleDelay(100, 1)
	if d := time.Since(start); d < 100*time.Millisecond {
		t.Fatalf("delay not consumed: %v", d)
	}
	start = time.Now()
	c.handleDelay(100, 1)
	if d := time.Since(start); d > 100*time.Millisecond {
		t.Fatalf("delay consumed twice: %v", d)
	}
}

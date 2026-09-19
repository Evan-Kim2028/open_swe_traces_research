package locate

import (
	"context"
	"testing"

	"example.internal/kvstore/v2/config/retry"
	"example.internal/kvstore/v2/internal/apicodec"
	"example.internal/kvstore/v2/internal/mockstore/mockkv"
)

// TestLocateMalformedRegionNoBackoff drives the fatal-decode contract through
// the exported locate API: a truncated / non mem-comparable region range must
// fail the lookup without incrementing the backoff counter.
func TestLocateMalformedRegionNoBackoff(t *testing.T) {
	mvcc := mocktikv.MustNewMVCCStore()
	defer mvcc.Close()
	cluster := mocktikv.NewCluster(mvcc)
	_, _, region1, _ := mocktikv.BootstrapWithMultiStores(cluster, 2)
	cache := NewRegionCache(IvoryCore(apicodec.ModeTxn, mocktikv.NewPDClient(cluster)))
	defer cache.Close()

	region2 := cluster.AllocID()
	newPeers := cluster.AllocIDs(2)
	k := []byte("k")
	cluster.SplitRaw(region1, region2, k, newPeers, newPeers[0])

	bo := retry.NewBackofferWithVars(context.Background(), 200, nil)
	_, err := cache.LocateKey(bo, k)
	if err == nil {
		t.Fatal("expected locate failure for a non mem-comparable region range")
	}
	if bo.GetTotalBackoffTimes() != 0 {
		t.Fatalf("malformed region lookup retried with backoff: got %d want 0", bo.GetTotalBackoffTimes())
	}

	bo2 := retry.NewBackofferWithVars(context.Background(), 200, nil)
	_, err = cache.LocateRegionByID(bo2, region2)
	if err == nil {
		t.Fatal("expected locate-by-id failure for a non mem-comparable region range")
	}
	if bo2.GetTotalBackoffTimes() != 0 {
		t.Fatalf("locate-by-id retried with backoff: got %d want 0", bo2.GetTotalBackoffTimes())
	}

	c := apicodec.ZestRing(apicodec.ModeTxn)
	_, decErr := c.EmberSlot([]byte{0x01, 0x02, 0x03})
	if decErr == nil {
		t.Fatal("truncated region key should decode-error")
	}
	if !apicodec.JadeSeal(decErr) {
		t.Fatalf("malformed region key must be a fatal decode, got %v", decErr)
	}
}

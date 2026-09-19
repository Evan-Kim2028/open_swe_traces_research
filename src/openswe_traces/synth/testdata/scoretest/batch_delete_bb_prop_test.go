package rawkv

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"testing"

	"example.internal/kvstore/v2/internal/locate"
	"example.internal/kvstore/v2/internal/mockstore/mockkv"
)

const batchDeleteSeed = 20260918
const batchDeleteCases = 10000
const batchDeleteUnseen = 256

func newBatchDeleteRawClient() (*Client, func()) {
	store := mocktikv.MustNewMVCCStore()
	cluster := mocktikv.NewCluster(store)
	mocktikv.BootstrapWithMultiStores(cluster, 2)
	c := &Client{
		clusterID:   0,
		regionCache: locate.NewRegionCache(mocktikv.NewPDClient(cluster)),
		rpcClient:   mocktikv.NewRPCClient(cluster, store, nil),
	}
	return c, func() {
		_ = c.Close()
		store.Close()
	}
}

func TestBatchDeleteRemovesListedKeys(t *testing.T) {
	rng := rand.New(rand.NewSource(batchDeleteSeed))
	client, done := newBatchDeleteRawClient()
	defer done()
	for i := 0; i < batchDeleteCases; i++ {
		n := rng.Intn(5) + 2
		keys := make([][]byte, n)
		keep := []byte(fmt.Sprintf("keep-%d", i))
		if err := client.Put(context.Background(), keep, []byte("stay")); err != nil {
			t.Fatalf("case %d keep put: %v", i, err)
		}
		for j := 0; j < n; j++ {
			keys[j] = []byte(fmt.Sprintf("p-%d-%d", i, j))
			if err := client.Put(context.Background(), keys[j], []byte{byte(j)}); err != nil {
				t.Fatalf("case %d put: %v", i, err)
			}
		}
		if err := client.BatchDelete(context.Background(), keys); err != nil {
			t.Fatalf("case %d batch delete: %v", i, err)
		}
		for _, k := range keys {
			got, err := client.Get(context.Background(), k)
			if err != nil {
				t.Fatalf("case %d get: %v", i, err)
			}
			if len(got) != 0 {
				t.Fatalf("case %d key %q still present", i, k)
			}
		}
		got, err := client.Get(context.Background(), keep)
		if err != nil || !bytes.Equal(got, []byte("stay")) {
			t.Fatalf("case %d unlisted key dropped: %q %v", i, got, err)
		}
	}
}

func TestBatchDeleteContractExamples(t *testing.T) {
	client, done := newBatchDeleteRawClient()
	defer done()
	pairs := map[string]string{"db": "TiDB", "key1": "value1", "key2": "value2", "key3": "value3"}
	keys := make([][]byte, 0, len(pairs))
	for k, v := range pairs {
		keys = append(keys, []byte(k))
		if err := client.Put(context.Background(), []byte(k), []byte(v)); err != nil {
			t.Fatal(err)
		}
	}
	if err := client.BatchDelete(context.Background(), keys); err != nil {
		t.Fatal(err)
	}
	got, err := client.Get(context.Background(), keys[0])
	if err != nil || len(got) != 0 {
		t.Fatalf("listed keys must be gone, got %q %v", got, err)
	}
}

func TestBatchDeleteUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(batchDeleteSeed + 1))
	client, done := newBatchDeleteRawClient()
	defer done()
	for i := 0; i < batchDeleteUnseen; i++ {
		k := []byte(fmt.Sprintf("u-%d-%d", i, rng.Intn(1<<20)))
		if err := client.Put(context.Background(), k, []byte("v")); err != nil {
			t.Fatalf("unseen %d put: %v", i, err)
		}
		if err := client.BatchDelete(context.Background(), [][]byte{k}); err != nil {
			t.Fatalf("unseen %d delete: %v", i, err)
		}
		got, err := client.Get(context.Background(), k)
		if err != nil || len(got) != 0 {
			t.Fatalf("unmentioned key still present: %q %v", got, err)
		}
	}
}

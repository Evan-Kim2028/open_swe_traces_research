package rawkv

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"

	"example.internal/kvstore/v2/internal/locate"
	"example.internal/kvstore/v2/internal/mockstore/mockkv"
)

const deleteRangeSeed = 20260918
const deleteRangeCases = 10000

func newScoretestRawClient() (*Client, func()) {
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

func deleteRangeCaseKey(i, j int) []byte {
	var buf [8]byte
	buf[0] = byte('a' + j)
	binary.BigEndian.PutUint32(buf[1:], uint32(i))
	buf[5] = byte(j)
	return buf[:6]
}

func TestDeleteRangeHalfOpenProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(deleteRangeSeed))
	client, done := newScoretestRawClient()
	defer done()
	for i := 0; i < deleteRangeCases; i++ {
		n := rng.Intn(6) + 3
		keys := make([][]byte, n)
		vals := make([][]byte, n)
		for j := 0; j < n; j++ {
			keys[j] = deleteRangeCaseKey(i, j)
			vals[j] = []byte{byte('A' + j)}
			if err := client.Put(context.Background(), keys[j], vals[j]); err != nil {
				t.Fatalf("case %d put: %v", i, err)
			}
		}
		start, end := keys[1], keys[n-1]
		if bytes.Compare(start, end) > 0 {
			start, end = end, start
		}
		if err := client.DeleteRange(context.Background(), start, end); err != nil {
			t.Fatalf("case %d range delete: %v", i, err)
		}
		for j, k := range keys {
			got, err := client.Get(context.Background(), k)
			if err != nil {
				t.Fatalf("case %d get: %v", i, err)
			}
			inside := bytes.Compare(k, start) >= 0 && bytes.Compare(k, end) < 0
			if inside && len(got) != 0 {
				t.Fatalf("case %d key %q still present after range delete", i, k)
			}
			if !inside && !bytes.Equal(got, vals[j]) {
				t.Fatalf("case %d key %q outside range want %q got %q", i, k, vals[j], got)
			}
		}
	}
}

func TestDeleteRangeContractExamples(t *testing.T) {
	client, done := newScoretestRawClient()
	defer done()
	pairs := map[string]string{"db": "TiDB", "key1": "value1", "key2": "value2", "key3": "value3", "key4": "value4", "kv": "TiKV"}
	for k, v := range pairs {
		if err := client.Put(context.Background(), []byte(k), []byte(v)); err != nil {
			t.Fatal(err)
		}
	}
	if err := client.DeleteRange(context.Background(), []byte("key3"), nil); err != nil {
		t.Fatal(err)
	}
	ks, vs, err := client.Scan(context.Background(), []byte("key3"), nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ks) != 0 || len(vs) != 0 {
		t.Fatalf("range from key3 to unbounded should be empty, got %v %v", ks, vs)
	}
	got, err := client.Get(context.Background(), []byte("key2"))
	if err != nil || !bytes.Equal(got, []byte("value2")) {
		t.Fatalf("key2 should remain, got %q %v", got, err)
	}
}

func TestDeleteRangeUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(deleteRangeSeed + 1))
	client, done := newScoretestRawClient()
	defer done()
	for i := 0; i < 256; i++ {
		k := []byte(fmt.Sprintf("u-%d-%d", i, rng.Intn(1<<20)))
		if err := client.Put(context.Background(), k, []byte("v")); err != nil {
			t.Fatalf("unseen %d put: %v", i, err)
		}
		end := append(append([]byte(nil), k...), 0xff)
		if err := client.DeleteRange(context.Background(), k, end); err != nil {
			t.Fatalf("unseen %d range delete: %v", i, err)
		}
		got, err := client.Get(context.Background(), k)
		if err != nil || len(got) != 0 {
			t.Fatalf("unseen %d still present: %q %v", i, got, err)
		}
	}
}

package unionstore

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"

	tikverr "example.internal/kvstore/v2/error"
)

func TestSnapshotStagingVisible(t *testing.T) {
	db := newMemDB()
	k := []byte("snap-k")
	if err := db.Set(k, []byte("v1")); err != nil {
		t.Fatal(err)
	}
	snap := db.SnapshotGetter()
	got, err := snap.Get(context.Background(), k)
	if err != nil || !bytes.Equal(got, []byte("v1")) {
		t.Fatalf("pre-insert snapshot get %q %v want v1", got, err)
	}
	_ = db.Staging()
	if err := db.Set([]byte("after"), []byte("new")); err != nil {
		t.Fatal(err)
	}
	got, err = db.Get([]byte("after"))
	if err != nil || !bytes.Equal(got, []byte("new")) {
		t.Fatalf("live get of insert %q %v", got, err)
	}
	_, err = snap.Get(context.Background(), []byte("after"))
	if err != tikverr.ErrNotExist {
		t.Fatalf("snapshot must hide keys inserted after it was taken, got %v", err)
	}
	got, err = snap.Get(context.Background(), k)
	if err != nil || len(got) == 0 {
		t.Fatalf("snapshot must still see keys that existed at checkpoint: %q %v", got, err)
	}
	it := db.SnapshotIter(nil, nil)
	sawAfter := false
	sawOld := false
	for it.Valid() {
		if bytes.Equal(it.Key(), []byte("after")) {
			sawAfter = true
		}
		if bytes.Equal(it.Key(), k) {
			sawOld = true
		}
		if err := it.Next(); err != nil {
			t.Fatal(err)
		}
	}
	if sawAfter {
		t.Fatal("snapshot iterator leaked a key inserted after the checkpoint")
	}
	if !sawOld {
		t.Fatal("snapshot iterator missed a key that existed at checkpoint")
	}
	_, err = snap.Get(context.Background(), []byte("missing"))
	if err != tikverr.ErrNotExist {
		t.Fatalf("missing: %v", err)
	}
}

func TestSnapshotUnmentionedKeys(t *testing.T) {
	db := newMemDB()
	for i := 0; i < 64; i++ {
		k := []byte{byte(i + 3), byte(i)}
		if err := db.Set(k, append([]byte("u"), k...)); err != nil {
			t.Fatal(err)
		}
	}
	snap := db.SnapshotGetter()
	key := []byte{10, 7}
	got, err := snap.Get(context.Background(), key)
	want := append([]byte("u"), key...)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("unmentioned key got %q %v want %q", got, err, want)
	}
	if err := db.Set([]byte("late"), []byte("x")); err != nil {
		t.Fatal(err)
	}
	_, err = snap.Get(context.Background(), []byte("late"))
	if err != tikverr.ErrNotExist {
		t.Fatalf("late insert visible: %v", err)
	}
}

func BenchmarkSnapshotGet(b *testing.B) {
	db := newMemDB()
	buf := make([][16]byte, 5000)
	for i := range buf {
		binary.BigEndian.PutUint32(buf[i][:], uint32(i))
		if err := db.Set(buf[i][:], buf[i][:]); err != nil {
			b.Fatal(err)
		}
	}
	snap := db.SnapshotGetter()
	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = snap.Get(ctx, buf[i%len(buf)][:])
			i++
		}
	})
}

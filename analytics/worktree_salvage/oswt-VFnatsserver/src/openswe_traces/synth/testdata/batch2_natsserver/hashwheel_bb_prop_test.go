// Hidden black-box property suite for the hashwheel unit.
// Drives only the exported API in server/thw (api.md): NewHashWheel, Add,
// Remove, Update, ExpireTasks, GetNextExpiration, Count, Encode, Decode,
// HashWheelEntry, ErrTaskNotFound, ErrInvalidVersion.
// One property per DETAILS.md line; seeded random inputs, no worked examples.
package thw_test

import (
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	thw "example.internal/msgkit/v2/server/thw"
)

const bbHwHiddenSeed = 20260919

func bbHwSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbHwHiddenSeed
}

func bbHwRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbHwSeed()))
}

const bbHwSec = int64(1_000_000_000)
const bbHwWheel = int64(4096)

// Detail 1: slot position = (expires/1s) mod 4096; add dedups on seq within
// a slot (no double count); slot/global lowest only decrease.
func TestDetail01_SlotDedupAndLowest(t *testing.T) {
	rng := bbHwRng(t)
	hw := thw.NewHashWheel()
	base := (1 + rng.Int63n(10)) * bbHwSec
	// Same seq, two expiries landing in the same slot -> still one entry.
	e1 := base
	e2 := base + bbHwWheel*bbHwSec // +4096s -> same slot.
	if err := hw.Add(7, e1); err != nil {
		t.Fatal(err)
	}
	if err := hw.Add(7, e2); err != nil {
		t.Fatal(err)
	}
	if hw.Count() != 1 {
		t.Fatalf("same seq same slot: count=%d want 1", hw.Count())
	}
	// Different seq, same slot -> second entry.
	if err := hw.Add(9, e1); err != nil {
		t.Fatal(err)
	}
	if hw.Count() != 2 {
		t.Fatalf("different seq same slot: count=%d want 2", hw.Count())
	}
	// Global lowest only decreases: add a later expiry, earliest stays.
	if err := hw.Add(11, e1+50*bbHwSec); err != nil {
		t.Fatal(err)
	}
	low := hw.GetNextExpiration(math.MaxInt64)
	if low > e1 {
		t.Fatalf("lowest=%d increased above earliest add %d", low, e1)
	}
	// Exact duplicate adds never grow the count.
	hw3 := thw.NewHashWheel()
	for i := 0; i < 50; i++ {
		seq := uint64(i)
		exp := rng.Int63n(50) * bbHwSec
		if err := hw3.Add(seq, exp); err != nil {
			t.Fatal(err)
		}
		if err := hw3.Add(seq, exp); err != nil {
			t.Fatal(err)
		}
	}
	if hw3.Count() != 50 {
		t.Fatalf("exact duplicate adds: count=%d want 50", hw3.Count())
	}
}

// Detail 2: Remove locates slot via the supplied expiry; missing slot or
// seq -> task-not-found, count untouched; empty slot freed.
func TestDetail02_RemoveViaSuppliedExpiry(t *testing.T) {
	rng := bbHwRng(t)
	hw := thw.NewHashWheel()
	exp := rng.Int63n(1000) * bbHwSec
	if err := hw.Add(1, exp); err != nil {
		t.Fatal(err)
	}
	// Remove with a wrong (but valid) expiry -> not found, count untouched.
	if err := hw.Remove(1, exp+bbHwSec); !errors.Is(err, thw.ErrTaskNotFound) {
		t.Fatalf("remove wrong expiry: %v want ErrTaskNotFound", err)
	}
	if hw.Count() != 1 {
		t.Fatalf("count after failed remove: %d want 1", hw.Count())
	}
	// Missing seq.
	if err := hw.Remove(99, exp); !errors.Is(err, thw.ErrTaskNotFound) {
		t.Fatalf("remove missing seq: %v want ErrTaskNotFound", err)
	}
	// Empty wheel.
	if err := thw.NewHashWheel().Remove(1, exp); !errors.Is(err, thw.ErrTaskNotFound) {
		t.Fatalf("remove on empty wheel: %v want ErrTaskNotFound", err)
	}
	// Successful remove.
	if err := hw.Remove(1, exp); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if hw.Count() != 0 {
		t.Fatalf("count after remove: %d want 0", hw.Count())
	}
	// Removed again -> not found.
	if err := hw.Remove(1, exp); !errors.Is(err, thw.ErrTaskNotFound) {
		t.Fatalf("double remove: %v want ErrTaskNotFound", err)
	}
}

// Detail 3: Update = remove-at-old-expiry (failable) + add-at-new.
func TestDetail03_UpdateSemantics(t *testing.T) {
	rng := bbHwRng(t)
	hw := thw.NewHashWheel()
	oldE := rng.Int63n(100) * bbHwSec
	newE := oldE + 7*bbHwSec
	if err := hw.Add(5, oldE); err != nil {
		t.Fatal(err)
	}
	if err := hw.Update(5, oldE, newE); err != nil {
		t.Fatalf("update: %v", err)
	}
	if hw.Count() != 1 {
		t.Fatalf("count after update: %d want 1", hw.Count())
	}
	// Old expiry no longer removes; new expiry does.
	if err := hw.Remove(5, oldE); !errors.Is(err, thw.ErrTaskNotFound) {
		t.Fatalf("remove at old expiry after update: %v want ErrTaskNotFound", err)
	}
	if err := hw.Remove(5, newE); err != nil {
		t.Fatalf("remove at new expiry: %v", err)
	}
	// Update with wrong old expiry -> error, nothing added.
	if err := hw.Add(8, oldE); err != nil {
		t.Fatal(err)
	}
	if err := hw.Update(8, oldE+bbHwSec, newE+bbHwSec); !errors.Is(err, thw.ErrTaskNotFound) {
		t.Fatalf("update missing old: %v want ErrTaskNotFound", err)
	}
	if hw.Count() != 1 {
		t.Fatalf("count after failed update: %d want 1", hw.Count())
	}
	// The task should still be at oldE (not moved).
	if err := hw.Remove(8, oldE); err != nil {
		t.Fatalf("task lost after failed update: %v", err)
	}
	// Update on empty wheel fails.
	if err := thw.NewHashWheel().Update(1, 0, bbHwSec); !errors.Is(err, thw.ErrTaskNotFound) {
		t.Fatalf("update empty: %v want ErrTaskNotFound", err)
	}
}

// Detail 4: ExpireTasks fires the callback for every task with
// expires <= ts including wraparound slots; only true-voted tasks are
// removed; the callback's own removal is not double-counted.
func TestDetail04_ExpiryCallbackSemantics(t *testing.T) {
	rng := bbHwRng(t)
	hw := thw.NewHashWheel()
	var seqs []uint64
	var exps []int64
	// ExpireTasks uses wall-clock now: spread expiries around it, forcing
	// wraparound slots on the past side (up to ~3 wheel revolutions back)
	// and far-future tasks on the other.
	now := time.Now().UnixNano()
	for i := 0; i < 200; i++ {
		seq := uint64(i + 1)
		var exp int64
		if i%3 == 0 {
			exp = now + (1+rng.Int63n(2*bbHwWheel))*bbHwSec // future.
		} else {
			exp = now - (1+rng.Int63n(3*bbHwWheel))*bbHwSec // past.
		}
		if err := hw.Add(seq, exp); err != nil {
			t.Fatal(err)
		}
		seqs = append(seqs, seq)
		exps = append(exps, exp)
	}
	if hw.Count() != 200 {
		t.Fatalf("count=%d want 200", hw.Count())
	}
	fired := map[uint64]bool{}
	hw.ExpireTasks(func(seq uint64, expires int64) bool {
		fired[seq] = true
		return false // vote keep — nothing should be removed.
	})
	if hw.Count() != 200 {
		t.Fatalf("count after keep-votes: %d want 200", hw.Count())
	}
	// Exactly the past-due tasks must have fired, by per-task expires<=now.
	want := 0
	for i, e := range exps {
		if e <= now {
			want++
			if !fired[seqs[i]] {
				t.Fatalf("seq %d exp %d <= now %d not fired", seqs[i], e, now)
			}
		}
	}
	if len(fired) != want {
		t.Fatalf("fired=%d want %d (expires<=now=%d)", len(fired), want, now)
	}
	// True-votes remove exactly those tasks.
	hw2 := thw.NewHashWheel()
	for i := 0; i < 100; i++ {
		if err := hw2.Add(uint64(i+1), int64(i)*bbHwSec); err != nil {
			t.Fatal(err)
		}
	}
	removed := map[uint64]bool{}
	hw2.ExpireTasks(func(seq uint64, expires int64) bool {
		vote := seq%2 == 0
		if vote {
			removed[seq] = true
		}
		return vote
	})
	if hw2.Count() != 50 {
		t.Fatalf("count after evens expired: %d want 50", hw2.Count())
	}
	// Odd seqs survive.
	hw2.ExpireTasks(func(seq uint64, expires int64) bool {
		if seq%2 == 0 {
			t.Fatalf("seq %d re-fired after removal", seq)
		}
		return true
	})
	if hw2.Count() != 0 {
		t.Fatalf("final count=%d want 0", hw2.Count())
	}
	// Callback's own removal is not double-counted.
	hw3 := thw.NewHashWheel()
	if err := hw3.Add(1, 0); err != nil {
		t.Fatal(err)
	}
	if err := hw3.Add(2, bbHwSec); err != nil {
		t.Fatal(err)
	}
	hw3.ExpireTasks(func(seq uint64, expires int64) bool {
		if seq == 1 {
			_ = hw3.Remove(1, 0) // remove inside callback.
		}
		return true
	})
	if hw3.Count() != 0 {
		t.Fatalf("count after callback-removal: %d want 0", hw3.Count())
	}
	_ = removed
}

// Detail 5: slot and global lowest recomputed from survivors after expiry;
// empty slots freed.
func TestDetail05_LowestRecomputeAfterExpiry(t *testing.T) {
	hw := thw.NewHashWheel()
	if err := hw.Add(1, 0); err != nil {
		t.Fatal(err)
	}
	if err := hw.Add(2, 10*bbHwSec); err != nil {
		t.Fatal(err)
	}
	if err := hw.Add(3, 20*bbHwSec); err != nil {
		t.Fatal(err)
	}
	// Expire only the earliest task.
	hw.ExpireTasks(func(seq uint64, expires int64) bool { return seq == 1 })
	low := hw.GetNextExpiration(math.MaxInt64)
	if low != 10*bbHwSec {
		t.Fatalf("lowest after expiry=%d want %d", low, 10*bbHwSec)
	}
	hw.ExpireTasks(func(seq uint64, expires int64) bool { return true })
	if hw.GetNextExpiration(math.MaxInt64) != math.MaxInt64 {
		t.Fatalf("empty wheel lowest=%d want MaxInt64", hw.GetNextExpiration(math.MaxInt64))
	}
}

// Detail 6: GetNextExpiration returns global lowest when < bound else
// MaxInt64; empty wheel -> MaxInt64.
func TestDetail06_NextExpirationBounds(t *testing.T) {
	rng := bbHwRng(t)
	if got := thw.NewHashWheel().GetNextExpiration(math.MaxInt64); got != math.MaxInt64 {
		t.Fatalf("empty wheel: %d want MaxInt64", got)
	}
	hw := thw.NewHashWheel()
	minE := int64(math.MaxInt64)
	for i := 0; i < 30; i++ {
		e := rng.Int63n(1000) * bbHwSec
		if e < minE {
			minE = e
		}
		if err := hw.Add(uint64(i+1), e); err != nil {
			t.Fatal(err)
		}
	}
	if got := hw.GetNextExpiration(math.MaxInt64); got != minE {
		t.Fatalf("lowest=%d want %d", got, minE)
	}
	// Bound below the lowest -> MaxInt64.
	if got := hw.GetNextExpiration(minE); got != math.MaxInt64 {
		t.Fatalf("bound==lowest: %d want MaxInt64", got)
	}
	if got := hw.GetNextExpiration(minE - 1); got != math.MaxInt64 {
		t.Fatalf("bound<lowest: %d want MaxInt64", got)
	}
	if got := hw.GetNextExpiration(minE + 1); got != minE {
		t.Fatalf("bound>lowest: %d want %d", got, minE)
	}
}

// Detail 7: Encode = version byte 1 + count LE-u64 + stamp LE-u64 +
// (varint expires, uvarint seq) pairs.
func TestDetail07_EncodeWireFormat(t *testing.T) {
	hw := thw.NewHashWheel()
	if err := hw.Add(1, 2*bbHwSec); err != nil {
		t.Fatal(err)
	}
	if err := hw.Add(2, 5*bbHwSec); err != nil {
		t.Fatal(err)
	}
	enc := hw.Encode(0xdeadbeef)
	if len(enc) < 17 {
		t.Fatalf("encode len=%d want >=17", len(enc))
	}
	if enc[0] != 1 {
		t.Fatalf("version byte=%d want 1", enc[0])
	}
	if got := binary.LittleEndian.Uint64(enc[1:9]); got != 2 {
		t.Fatalf("count field=%d want 2", got)
	}
	if got := binary.LittleEndian.Uint64(enc[9:17]); got != 0xdeadbeef {
		t.Fatalf("stamp field=%x want deadbeef", got)
	}
	// Entries: (varint expires, uvarint seq) pairs — total should parse.
	rest := enc[17:]
	n := 0
	got := map[uint64]int64{}
	for n < len(rest) {
		exp, m := binary.Varint(rest[n:])
		if m <= 0 {
			t.Fatalf("varint expires at %d: m=%d", n, m)
		}
		n += m
		seq, m := binary.Uvarint(rest[n:])
		if m <= 0 {
			t.Fatalf("uvarint seq at %d: m=%d", n, m)
		}
		n += m
		got[seq] = exp
	}
	if len(got) != 2 || got[1] != 2*bbHwSec || got[2] != 5*bbHwSec {
		t.Fatalf("decoded pairs=%v want {1:2s,2:5s}", got)
	}
	// Empty wheel encodes to exactly the 17-byte header.
	if enc := thw.NewHashWheel().Encode(7); len(enc) != 17 || enc[0] != 1 ||
		binary.LittleEndian.Uint64(enc[1:9]) != 0 || binary.LittleEndian.Uint64(enc[9:17]) != 7 {
		t.Fatalf("empty encode: %x", enc)
	}
}

// Detail 8: Decode — <17 bytes -> short buffer; version != 1 -> unknown
// version (wheel untouched); else clear slots + lowest and add exactly
// count entries — count NOT reset, so decoding into a non-empty wheel
// accumulates; truncated entry -> unexpected EOF; returns stamp.
func TestDetail08_DecodeSemantics(t *testing.T) {
	rng := bbHwRng(t)
	// <17 bytes -> error.
	for _, n := range []int{0, 1, 16} {
		if _, err := thw.NewHashWheel().Decode(make([]byte, n)); err == nil {
			t.Fatalf("decode %d bytes should error", n)
		}
	}
	// version != 1 -> ErrInvalidVersion, wheel untouched.
	hw := thw.NewHashWheel()
	if err := hw.Add(3, bbHwSec); err != nil {
		t.Fatal(err)
	}
	bad := make([]byte, 17)
	bad[0] = 2
	if _, err := hw.Decode(bad); !errors.Is(err, thw.ErrInvalidVersion) {
		t.Fatalf("decode version 2: %v want ErrInvalidVersion", err)
	}
	if hw.Count() != 1 {
		t.Fatalf("wheel modified by bad-version decode: count=%d want 1", hw.Count())
	}
	// Valid round trip.
	src := thw.NewHashWheel()
	for i := 0; i < 40; i++ {
		if err := src.Add(uint64(i+1), rng.Int63n(500)*bbHwSec); err != nil {
			t.Fatal(err)
		}
	}
	enc := src.Encode(4242)
	dst := thw.NewHashWheel()
	stamp, err := dst.Decode(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if stamp != 4242 {
		t.Fatalf("stamp=%d want 4242", stamp)
	}
	if dst.Count() != 40 {
		t.Fatalf("count=%d want 40", dst.Count())
	}
	// Entries decoded: same expiry behavior — expire everything.
	fired := 0
	dst.ExpireTasks(func(seq uint64, expires int64) bool {
		fired++
		return true
	})
	if fired != 40 || dst.Count() != 0 {
		t.Fatalf("fired=%d count=%d want 40/0", fired, dst.Count())
	}
	// Decode into a NON-empty wheel: count accumulates over pre-existing.
	w2 := thw.NewHashWheel()
	if err := w2.Add(999, bbHwSec); err != nil {
		t.Fatal(err)
	}
	if _, err := w2.Decode(enc); err != nil {
		t.Fatalf("decode into non-empty: %v", err)
	}
	if w2.Count() != 41 {
		t.Fatalf("accumulated count=%d want 41", w2.Count())
	}
	// Truncated entry -> error.
	trunc := enc[:len(enc)-1]
	if _, err := thw.NewHashWheel().Decode(trunc); err == nil {
		t.Fatal("truncated entry should error")
	}
	// Truncated mid-varint -> error.
	trunc2 := enc[:17+1]
	if _, err := thw.NewHashWheel().Decode(trunc2); err == nil {
		t.Fatal("header-only truncation should error")
	}
}

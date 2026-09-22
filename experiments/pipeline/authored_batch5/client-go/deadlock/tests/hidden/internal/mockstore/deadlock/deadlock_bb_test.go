package deadlock

import (
	"strconv"
	"strings"
	"sync"
	"testing"
)

// Hidden suite for unit deadlock. One TestDetailNN per DETAILS.md line.

// TestDetail01: Detect registers an edge only when it does not close a
// cycle; a rejected edge must not poison later detects.
func TestDetail01(t *testing.T) {
	d := NewDetector()
	if e := d.Detect(1, 2, 100); e != nil {
		t.Fatalf("clean edge reported deadlock: %v", e)
	}
	if e := d.Detect(2, 3, 50); e != nil {
		t.Fatalf("clean edge reported deadlock: %v", e)
	}
	// 3->1 would close 1->2->3->1: detected, and NOT registered.
	if e := d.Detect(3, 1, 60); e == nil {
		t.Fatalf("cycle 3->1 not detected")
	}
	// If the rejected 3->1 edge had been registered, 1->3 would close a
	// false cycle through it (3->1->2->3 reaches the source).
	if e := d.Detect(1, 3, 70); e != nil {
		t.Fatalf("rejected edge was registered: %v", e)
	}
}

// TestDetail02: the reported KeyHash is the stored edge's hash, not the
// incoming one — for a chain it is the edge pointing at the source.
func TestDetail02(t *testing.T) {
	d := NewDetector()
	d.Detect(1, 2, 100)
	if e := d.Detect(2, 1, 200); e == nil || e.KeyHash != 100 {
		t.Fatalf("Detect(2,1) = %v, want KeyHash 100", e)
	}
	d2 := NewDetector()
	d2.Detect(1, 2, 10)
	d2.Detect(2, 3, 20)
	if e := d2.Detect(3, 1, 30); e == nil || e.KeyHash != 20 {
		t.Fatalf("Detect(3,1) = %v, want KeyHash 20", e)
	}
}

// TestDetail03: re-detecting the same pair is idempotent — it neither fails
// nor turns into a cycle.
func TestDetail03(t *testing.T) {
	d := NewDetector()
	for i := 0; i < 3; i++ {
		if e := d.Detect(1, 2, 100); e != nil {
			t.Fatalf("repeat Detect(1,2) = %v", e)
		}
	}
	// A distinct keyHash for the same txn pair is a different edge.
	if e := d.Detect(1, 2, 101); e != nil {
		t.Fatalf("Detect(1,2,101) = %v", e)
	}
}

// TestDetail04: CleanUp drops a txn's whole wait list; CleanUpWaitFor removes
// one pair and keeps the rest.
func TestDetail04(t *testing.T) {
	d := NewDetector()
	d.Detect(1, 2, 100)
	d.CleanUp(1)
	if e := d.Detect(2, 1, 200); e != nil {
		t.Fatalf("CleanUp left the edge: %v", e)
	}
	d2 := NewDetector()
	d2.Detect(1, 2, 100)
	d2.Detect(1, 3, 50)
	d2.CleanUpWaitFor(1, 2, 100)
	// 1->2 gone: 2->1 is clean.
	if e := d2.Detect(2, 1, 200); e != nil {
		t.Fatalf("CleanUpWaitFor did not remove the pair: %v", e)
	}
	// 1->3 survives: 3->1 still closes a cycle, reported hash is 1->3's.
	if e := d2.Detect(3, 1, 60); e == nil || e.KeyHash != 50 {
		t.Fatalf("surviving pair lost or wrong hash: %v", e)
	}
}

// TestDetail05: Expire drops entries whose source txn is strictly below
// minTS.
func TestDetail05(t *testing.T) {
	d := NewDetector()
	d.Detect(1, 2, 100)
	d.Detect(5, 6, 500)
	d.Expire(2)
	if e := d.Detect(2, 1, 200); e != nil {
		t.Fatalf("expired edge still detected: %v", e)
	}
	if e := d.Detect(6, 5, 600); e == nil {
		t.Fatalf("fresh edge expired too")
	}
	// Strictly below: minTS == source keeps the entry.
	d.Expire(5)
	if e := d.Detect(6, 5, 600); e == nil {
		t.Fatalf("Expire removed entry with source == minTS")
	}
	d.Expire(6)
	if e := d.Detect(6, 5, 600); e != nil {
		t.Fatalf("Expire kept entry with source < minTS")
	}
}

// TestDetail06: ErrDeadlock.Error embeds the key hash.
func TestDetail06(t *testing.T) {
	got := (&ErrDeadlock{KeyHash: 42}).Error()
	if !strings.Contains(got, "42") {
		t.Fatalf("Error() = %q does not embed the key hash", got)
	}
	if got == strconv.Itoa(42) {
		t.Fatalf("Error() = %q is the bare number, want a formatted string", got)
	}
}

// TestDetail07: exported methods hold the detector lock — concurrent calls
// must complete without panic or corruption.
func TestDetail07(t *testing.T) {
	d := NewDetector()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(base uint64) {
			defer wg.Done()
			for j := uint64(0); j < 50; j++ {
				d.Detect(base*1000+j, base*1000+j+100, j)
				d.CleanUp(base*1000 + j)
				d.CleanUpWaitFor(base*1000+j, base*1000+j+100, j)
				d.Expire(base * 1000)
			}
		}(uint64(i + 1))
	}
	wg.Wait()
}

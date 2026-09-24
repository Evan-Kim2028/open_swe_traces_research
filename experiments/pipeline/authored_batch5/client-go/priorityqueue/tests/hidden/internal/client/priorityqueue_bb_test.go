package client

import (
	"testing"
)

// Hidden suite for unit priorityqueue. One TestDetailNN per DETAILS.md line.
// (The in-tree priority_queue_test.go that disclosed the pop direction is
// excised away, but the solver-visible client_batch.go still calls
// `highestPriority() >= highTaskPriority`, which only works on a max-heap —
// so the largest-first direction is derivable from the visible caller.)

type bbItem struct {
	pri      uint64
	canceled bool
}

func (i *bbItem) priority() uint64 { return i.pri }
func (i *bbItem) isCanceled() bool { return i.canceled }

func bbPushN(pq *PriorityQueue, pris ...uint64) {
	for _, p := range pris {
		pq.Push(&bbItem{pri: p})
	}
}

func bbPris(items []Item) []uint64 {
	out := make([]uint64, 0, len(items))
	for _, it := range items {
		out = append(out, it.priority())
	}
	return out
}

// TestDetail01: max-heap — the largest priority() pops first.
func TestDetail01(t *testing.T) {
	pq := NewPriorityQueue()
	bbPushN(pq, 1, 5, 3)
	if got := pq.highestPriority(); got != 5 {
		t.Fatalf("highestPriority() = %d, want the maximum 5", got)
	}
	got := pq.Take(1)
	if len(got) != 1 || got[0].priority() != 5 {
		t.Fatalf("Take(1) = %v, want [5]", bbPris(got))
	}
	if got := pq.highestPriority(); got != 3 {
		t.Fatalf("after popping 5, highestPriority() = %d, want 3", got)
	}
}

// TestDetail02: highestPriority is the heap root's priority, 0 when empty,
// and agrees with what pops next.
func TestDetail02(t *testing.T) {
	pq := NewPriorityQueue()
	if got := pq.highestPriority(); got != 0 {
		t.Fatalf("empty queue highestPriority() = %d, want 0", got)
	}
	bbPushN(pq, 3, 9, 4)
	h := pq.highestPriority()
	top := pq.Take(1)
	if len(top) != 1 {
		t.Fatalf("Take(1) returned %d items", len(top))
	}
	if h != top[0].priority() {
		t.Fatalf("highestPriority() = %d but Take(1) popped %d", h, top[0].priority())
	}
}

// TestDetail03: Take(n<=0) is nil and does not drain; Take(n<Len) pops the n
// top-priority entries in pop order.
func TestDetail03(t *testing.T) {
	pq := NewPriorityQueue()
	bbPushN(pq, 1, 2, 3, 4, 5)
	if got := pq.Take(0); got != nil {
		t.Fatalf("Take(0) = %v, want nil", bbPris(got))
	}
	if got := pq.Take(-2); got != nil {
		t.Fatalf("Take(-2) = %v, want nil", bbPris(got))
	}
	if pq.Len() != 5 {
		t.Fatalf("Take(0)/Take(-1) drained the queue: Len=%d", pq.Len())
	}
	got := pq.Take(2)
	if len(got) != 2 || got[0].priority() != 5 || got[1].priority() != 4 {
		t.Fatalf("Take(2) = %v, want [5 4]", bbPris(got))
	}
	if pq.Len() != 3 {
		t.Fatalf("Len() = %d after Take(2) of 5, want 3", pq.Len())
	}
}

// TestDetail04: Take(n>=Len) drains the whole queue and returns every queued
// item. (Shape only: the bulk path's element order is heap-internal and is
// not asserted.)
func TestDetail04(t *testing.T) {
	pq := NewPriorityQueue()
	bbPushN(pq, 1, 2, 3, 4, 5, 6)
	pq.Take(1) // pop one so the bulk path starts from a non-trivial heap
	got := pq.Take(10)
	if len(got) != 5 {
		t.Fatalf("Take(10) over 5 queued items returned %d", len(got))
	}
	seen := map[uint64]bool{}
	for _, it := range got {
		seen[it.priority()] = true
	}
	for _, p := range []uint64{1, 2, 3, 4, 5} {
		if !seen[p] {
			t.Fatalf("Take(10) result %v missing queued priority %d", bbPris(got), p)
		}
	}
	if pq.Len() != 0 {
		t.Fatalf("Take(10) left %d items", pq.Len())
	}
}

// TestDetail05: clean() drops canceled entries only; reset() empties the
// queue; all() returns every queued item and is a copy.
func TestDetail05(t *testing.T) {
	pq := NewPriorityQueue()
	dead := &bbItem{pri: 1, canceled: true}
	pq.Push(dead)
	bbPushN(pq, 2, 3)
	pq.clean()
	if pq.Len() != 2 {
		t.Fatalf("clean() left %d items, want 2", pq.Len())
	}
	for _, it := range pq.all() {
		if it.isCanceled() {
			t.Fatalf("canceled item survived clean()")
		}
	}
	// all() is a copy: mutating it must not corrupt the queue.
	arr := pq.all()
	if len(arr) != 2 {
		t.Fatalf("all() returned %d items, want 2", len(arr))
	}
	arr[0] = dead
	for _, it := range pq.all() {
		if it == dead {
			t.Fatalf("all() aliases the internal slice")
		}
	}
	pq.reset()
	if pq.Len() != 0 {
		t.Fatalf("reset() left %d items", pq.Len())
	}
	// Queue still usable after reset.
	pq.Push(&bbItem{pri: 7})
	if pq.highestPriority() != 7 {
		t.Fatalf("queue unusable after reset")
	}
}

// TestDetail06: draining everything via Take leaves the queue empty.
func TestDetail06(t *testing.T) {
	pq := NewPriorityQueue()
	bbPushN(pq, 4, 1, 3)
	pq.Take(3)
	if pq.Len() != 0 {
		t.Fatalf("Len() = %d after draining Take, want 0", pq.Len())
	}
	if got := pq.highestPriority(); got != 0 {
		t.Fatalf("highestPriority() = %d on drained queue, want 0", got)
	}
}

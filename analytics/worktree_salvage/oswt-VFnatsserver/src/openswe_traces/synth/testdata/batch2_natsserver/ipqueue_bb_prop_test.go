// Hidden black-box property suite for the ipqueue unit.
// Drives the API in api.md: newIPQueue + options + all methods (invoked via
// method values), plus s.ipQueues registry and the documented error values.
// One property per DETAILS.md line; seeded random inputs, no worked examples.
package server

import (
	"errors"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"testing"
)

const bbIpqHiddenSeed = 20260919

func bbIpqSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbIpqHiddenSeed
}

func bbIpqRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbIpqSeed()))
}

// notified reports whether the queue's notification channel has a pending
// signal (non-blocking check).
func bbIpqNotified[T any](q *ipQueue[T]) bool {
	select {
	case <-q.ch:
		return true
	default:
		return false
	}
}

// Detail 1: constructor registers the queue in s.ipQueues by name; default
// recycle cap is ipQueueDefaultMaxRecycleSize (4*1024); options set
// mrs/calc/msz/mlen.
func TestDetail01_ConstructorRegistryAndOpts(t *testing.T) {
	s := &Server{}
	q := newIPQueue[int](s, "bb-test")
	if q == nil {
		t.Fatal("nil queue")
	}
	v, ok := s.ipQueues.Load("bb-test")
	if !ok {
		t.Fatal("queue not registered in s.ipQueues")
	}
	if v != q {
		t.Fatalf("registry holds %v want the constructed queue", v)
	}
	if q.mrs != ipQueueDefaultMaxRecycleSize {
		t.Fatalf("default mrs=%d want %d", q.mrs, ipQueueDefaultMaxRecycleSize)
	}
	if q.calc != nil || q.msz != 0 || q.mlen != 0 {
		t.Fatalf("default opts msz=%d mlen=%d want zero (calc set)", q.msz, q.mlen)
	}
	// Options are applied.
	q2 := newIPQueue[int](s, "bb-opts",
		ipqMaxRecycleSize[int](17),
		ipqSizeCalculation[int](func(e int) uint64 { return uint64(e) }),
		ipqLimitBySize[int](99),
		ipqLimitByLen[int](7))
	if q2.mrs != 17 || q2.calc == nil || q2.msz != 99 || q2.mlen != 7 {
		t.Fatalf("opts mrs=%d msz=%d mlen=%d (calc unset)", q2.mrs, q2.msz, q2.mlen)
	}
}

// Detail 2: push returns length after append; notification posted only when
// the queue was empty before this push.
func TestDetail02_PushLenAndNotify(t *testing.T) {
	rng := bbIpqRng(t)
	s := &Server{}
	q := newIPQueue[int](s, "bb-push")
	push := q.push
	if bbIpqNotified(q) {
		t.Fatal("signal before any push")
	}
	n := 5 + rng.Intn(10)
	for i := 1; i <= n; i++ {
		l, err := push(rng.Intn(1000))
		if err != nil {
			t.Fatal(err)
		}
		if l != i {
			t.Fatalf("push %d returned len %d", i, l)
		}
		if i == 1 {
			if !bbIpqNotified(q) {
				t.Fatal("first push did not post notification")
			}
		} else {
			if bbIpqNotified(q) {
				t.Fatalf("push %d re-posted notification while non-empty", i)
			}
		}
	}
	// After full pop + signal consumption, next push signals again.
	pop := q.pop
	_ = pop()
	for i := 0; i < 3; i++ {
		l, err := push(i)
		if err != nil {
			t.Fatal(err)
		}
		if l != i+1 {
			t.Fatalf("post-pop push len=%d want %d", l, i+1)
		}
		if i == 0 && !bbIpqNotified(q) {
			t.Fatal("push after emptying did not signal")
		}
	}
}

// Detail 3: len limit — push at limit -> errIPQLenLimitReached, length
// unchanged, element not stored.
func TestDetail03_LenLimit(t *testing.T) {
	rng := bbIpqRng(t)
	s := &Server{}
	lim := 3 + rng.Intn(5)
	q := newIPQueue[int](s, "bb-len", ipqLimitByLen[int](lim))
	push := q.push
	qlen := q.len
	for i := 0; i < lim; i++ {
		if _, err := push(i); err != nil {
			t.Fatalf("push %d under limit: %v", i, err)
		}
	}
	for i := 0; i < 5; i++ {
		l, err := push(1000 + i)
		if !errors.Is(err, errIPQLenLimitReached) {
			t.Fatalf("push over limit: err=%v want errIPQLenLimitReached", err)
		}
		if l != lim || qlen() != lim {
			t.Fatalf("len=%d qlen=%d want %d (element not stored)", l, qlen(), lim)
		}
	}
	// After popping one, a push is admitted again.
	popOne := q.popOne
	if _, ok := popOne(); !ok {
		t.Fatal("popOne empty")
	}
	if _, err := push(42); err != nil {
		t.Fatalf("push after pop: %v", err)
	}
}

// Detail 4: size limit — push fails iff running size + element size
// STRICTLY exceeds the limit (== admitted); requires the calc function or
// msz is inert.
func TestDetail04_SizeLimit(t *testing.T) {
	s := &Server{}
	q := newIPQueue[int](s, "bb-sz",
		ipqSizeCalculation[int](func(e int) uint64 { return uint64(e) }),
		ipqLimitBySize[int](10))
	push := q.push
	// 4 + 6 == 10 admitted (strictly exceeds means >).
	if _, err := push(4); err != nil {
		t.Fatal(err)
	}
	if _, err := push(6); err != nil {
		t.Fatalf("== limit should be admitted: %v", err)
	}
	l, err := push(1)
	if !errors.Is(err, errIPQSizeLimitReached) {
		t.Fatalf("over limit: err=%v want errIPQSizeLimitReached", err)
	}
	if l != 2 {
		t.Fatalf("len=%d want 2 (element not stored)", l)
	}
	// Size accounting frees space as elements pop.
	popOne := q.popOne
	if _, ok := popOne(); !ok {
		t.Fatal("popOne empty")
	}
	if _, err := push(4); err != nil {
		t.Fatalf("push after freeing 4: %v", err)
	}
	if _, err := push(1); !errors.Is(err, errIPQSizeLimitReached) {
		t.Fatalf("6+4+1 over 10: err=%v want size limit", err)
	}
	// msz without calc is inert.
	q2 := newIPQueue[int](s, "bb-sz-nocalc", ipqLimitBySize[int](1))
	push2 := q2.push
	for i := 0; i < 10; i++ {
		if _, err := push2(i); err != nil {
			t.Fatalf("msz without calc should be inert: %v", err)
		}
	}
	// calc without msz still tracks size.
	q3 := newIPQueue[int](s, "bb-sz-nolimit",
		ipqSizeCalculation[int](func(e int) uint64 { return uint64(e) }))
	push3 := q3.push
	size3 := q3.size
	for i := 0; i < 5; i++ {
		if _, err := push3(7); err != nil {
			t.Fatal(err)
		}
	}
	if size3() != 35 {
		t.Fatalf("size=%d want 35", size3())
	}
}

// Detail 5: pushMany under one lock; limit error discards all yielded
// elements; signals only when queue was empty AND ≥1 added; empty seq -> no
// signal, no error, unchanged len.
func TestDetail05_PushManyAtomic(t *testing.T) {
	s := &Server{}
	q := newIPQueue[int](s, "bb-many", ipqLimitByLen[int](3))
	pushMany := q.pushMany
	qlen := q.len
	l, err := pushMany(slices.Values([]int{1, 2, 3}))
	if err != nil || l != 3 {
		t.Fatalf("pushMany l=%d err=%v", l, err)
	}
	if !bbIpqNotified(q) {
		t.Fatal("pushMany on empty queue should signal")
	}
	// Overflow attempt: both elements discarded.
	l, err = pushMany(slices.Values([]int{4, 5}))
	if !errors.Is(err, errIPQLenLimitReached) {
		t.Fatalf("pushMany over limit err=%v", err)
	}
	if l != 3 || qlen() != 3 {
		t.Fatalf("pushMany partial state l=%d qlen=%d want 3", l, qlen())
	}
	pop := q.pop
	got := pop()
	if len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("pop=%v want [1 2 3] — discarded elements leaked", got)
	}
	// Empty seq: no signal, no error, unchanged.
	l, err = pushMany(slices.Values([]int{}))
	if err != nil || l != 0 {
		t.Fatalf("empty seq l=%d err=%v", l, err)
	}
	if bbIpqNotified(q) {
		t.Fatal("empty seq should not signal")
	}
	// Seq partially consumed on error: prior state restored.
	l, err = pushMany(slices.Values([]int{9, 9, 9, 9, 9}))
	if !errors.Is(err, errIPQLenLimitReached) || l != 0 || qlen() != 0 {
		t.Fatalf("oversize seq l=%d qlen=%d err=%v want 0 + len limit", l, qlen(), err)
	}
	if bbIpqNotified(q) {
		t.Fatal("rejected seq should not signal")
	}
}

// Detail 6: pop — nil-receiver -> nil; empty -> nil; returns remaining
// elements honoring partial one-at-a-time consumption; resets
// elts/pos/size; adds count to in-progress.
func TestDetail06_PopSemantics(t *testing.T) {
	var nilq *ipQueue[int]
	nilpop := nilq.pop
	if got := nilpop(); got != nil {
		t.Fatalf("nil-receiver pop=%v want nil", got)
	}
	s := &Server{}
	q := newIPQueue[int](s, "bb-pop",
		ipqSizeCalculation[int](func(e int) uint64 { return uint64(e) }))
	pop := q.pop
	if got := pop(); got != nil {
		t.Fatalf("empty pop=%v want nil", got)
	}
	push := q.push
	qlen := q.len
	size := q.size
	inprog := q.inProgress
	for i := 0; i < 6; i++ {
		if _, err := push(i * 10); err != nil {
			t.Fatal(err)
		}
	}
	// Partial consumption via popOne first.
	popOne := q.popOne
	v, ok := popOne()
	if !ok || v != 0 {
		t.Fatalf("popOne=(%v,%v) want 0,true", v, ok)
	}
	got := pop()
	if len(got) != 5 || got[0] != 10 || got[4] != 50 {
		t.Fatalf("pop=%v want remaining [10..50]", got)
	}
	if qlen() != 0 {
		t.Fatalf("len after pop=%d want 0", qlen())
	}
	if size() != 0 {
		t.Fatalf("size after pop=%d want 0", size())
	}
	// in-progress counts elements popped via pop(); popOne does not
	// contribute to inProgress.
	if ip := inprog(); ip != 5 {
		t.Fatalf("inProgress=%d want 5", ip)
	}
	// Second pop on drained queue -> nil.
	if got := pop(); got != nil {
		t.Fatalf("post-drain pop=%v want nil", got)
	}
}

// Detail 7: popOne — empty -> zero,false; FIFO; re-posts notification only
// when elements remain; size decrement only when elements remain (else
// reset).
func TestDetail07_PopOne(t *testing.T) {
	s := &Server{}
	q := newIPQueue[int](s, "bb-pop1",
		ipqSizeCalculation[int](func(e int) uint64 { return 2 }))
	popOne := q.popOne
	if v, ok := popOne(); ok || v != 0 {
		t.Fatalf("empty popOne=(%v,%v) want 0,false", v, ok)
	}
	push := q.push
	size := q.size
	for i := 0; i < 3; i++ {
		if _, err := push(i + 1); err != nil {
			t.Fatal(err)
		}
	}
	bbIpqNotified(q) // consume initial signal.
	// FIFO + re-post while elements remain.
	v, ok := popOne()
	if !ok || v != 1 {
		t.Fatalf("popOne=(%v,%v) want 1", v, ok)
	}
	if !bbIpqNotified(q) {
		t.Fatal("popOne with elements remaining should re-post")
	}
	if size() != 4 {
		t.Fatalf("size=%d want 4 after popping one of three", size())
	}
	v, ok = popOne()
	if !ok || v != 2 {
		t.Fatalf("popOne=(%v,%v) want 2", v, ok)
	}
	if !bbIpqNotified(q) {
		t.Fatal("popOne with one remaining should re-post")
	}
	v, ok = popOne()
	if !ok || v != 3 {
		t.Fatalf("popOne=(%v,%v) want 3", v, ok)
	}
	if bbIpqNotified(q) {
		t.Fatal("popOne on last element should NOT re-post")
	}
	if size() != 0 {
		t.Fatalf("size=%d want 0 after draining", size())
	}
	if v, ok := popOne(); ok || v != 0 {
		t.Fatalf("drained popOne=(%v,%v) want 0,false", v, ok)
	}
}

// Detail 8: recycle — nil pointer or nil slice -> no-op; decrements
// in-progress by slice length; pools only when cap <= recycle cap.
func TestDetail08_Recycle(t *testing.T) {
	s := &Server{}
	q := newIPQueue[int](s, "bb-rec")
	recycle := q.recycle
	recycle(nil) // must not panic.
	var nilslice *[]int
	recycle(nilslice)
	empty := []int(nil)
	recycle(&empty) // nil slice inside pointer: no-op, no panic.
	push := q.push
	pop := q.pop
	inprog := q.inProgress
	for i := 0; i < 4; i++ {
		if _, err := push(i); err != nil {
			t.Fatal(err)
		}
	}
	got := pop()
	if ip := inprog(); ip != 4 {
		t.Fatalf("inProgress=%d want 4", ip)
	}
	recycle(&got)
	if ip := inprog(); ip != 0 {
		t.Fatalf("inProgress after recycle=%d want 0", ip)
	}
	// Partial recycle decrements by slice length only.
	for i := 0; i < 3; i++ {
		if _, err := push(i); err != nil {
			t.Fatal(err)
		}
	}
	got = pop()
	half := got[:2]
	recycle(&half)
	if ip := inprog(); ip != 1 {
		t.Fatalf("inProgress after partial recycle=%d want 1", ip)
	}
	// Pooling: recycled small-cap slice is reused by the next push cycle.
	q2 := newIPQueue[int](s, "bb-pool")
	push2 := q2.push
	pop2 := q2.pop
	recycle2 := q2.recycle
	for i := 0; i < 3; i++ {
		if _, err := push2(i); err != nil {
			t.Fatal(err)
		}
	}
	first := pop2()
	if len(first) != 3 {
		t.Fatalf("first pop=%v want 3 elements", first)
	}
	first0 := &first[0]
	recycle2(&first) // recycle resets the caller's slice header.
	if _, err := push2(99); err != nil {
		t.Fatal(err)
	}
	second := pop2()
	if len(second) != 1 || second[0] != 99 {
		t.Fatalf("second pop=%v", second)
	}
	// The backing array may be the recycled one — verify either reuse or a
	// fresh allocation, but never stale content.
	if &second[0] == first0 {
		// Reused: stale elements beyond len must not surface.
		if len(second) != 1 {
			t.Fatalf("reused backing leaked %d stale elements", len(second))
		}
	}
	// cap > recycle cap is not pooled: with mrs=0 nothing is pooled.
	q3 := newIPQueue[int](s, "bb-pool0", ipqMaxRecycleSize[int](0))
	push3 := q3.push
	pop3 := q3.pop
	recycle3 := q3.recycle
	for i := 0; i < 3; i++ {
		if _, err := push3(i); err != nil {
			t.Fatal(err)
		}
	}
	big := pop3()
	recycle3(&big)
	if _, err := push3(7); err != nil {
		t.Fatal(err)
	}
	nb := pop3()
	if &nb[0] == &big[0] {
		t.Fatal("slice with cap > mrs=0 was pooled — should be dropped")
	}
}

// Detail 9: len = stored-consumed; size = running calc total, 0 without
// calc and reset on empty.
func TestDetail09_LenAndSizeAccounting(t *testing.T) {
	rng := bbIpqRng(t)
	s := &Server{}
	q := newIPQueue[int](s, "bb-acc",
		ipqSizeCalculation[int](func(e int) uint64 { return uint64(e) }))
	push := q.push
	qlen := q.len
	size := q.size
	popOne := q.popOne
	var wantSz uint64
	for i := 0; i < 20; i++ {
		e := rng.Intn(50)
		if _, err := push(e); err != nil {
			t.Fatal(err)
		}
		wantSz += uint64(e)
	}
	if qlen() != 20 || size() != wantSz {
		t.Fatalf("len=%d size=%d want 20/%d", qlen(), size(), wantSz)
	}
	for i := 0; i < 20; i++ {
		v, ok := popOne()
		if !ok {
			t.Fatal("popOne drained early")
		}
		wantSz -= uint64(v)
		if qlen() != 19-i {
			t.Fatalf("len=%d want %d", qlen(), 19-i)
		}
		want := wantSz
		if i == 19 {
			want = 0 // reset on empty
		}
		if size() != want {
			t.Fatalf("size=%d want %d after %d pops", size(), want, i+1)
		}
	}
	// Without calc, size is always 0.
	q2 := newIPQueue[int](s, "bb-acc0")
	push2 := q2.push
	size2 := q2.size
	if _, err := push2(5); err != nil {
		t.Fatal(err)
	}
	if size2() != 0 {
		t.Fatalf("no-calc size=%d want 0", size2())
	}
}

// Detail 10: drain — nil-safe; returns drained count; clears state;
// consumes pending notification.
func TestDetail10_Drain(t *testing.T) {
	var nilq *ipQueue[int]
	nildrain := nilq.drain
	if n := nildrain(); n != 0 {
		t.Fatalf("nil drain=%d want 0", n)
	}
	s := &Server{}
	q := newIPQueue[int](s, "bb-drain",
		ipqSizeCalculation[int](func(e int) uint64 { return uint64(e) }))
	push := q.push
	drain := q.drain
	qlen := q.len
	size := q.size
	for i := 0; i < 7; i++ {
		if _, err := push(i + 1); err != nil {
			t.Fatal(err)
		}
	}
	n := drain()
	if n != 7 {
		t.Fatalf("drain=%d want 7", n)
	}
	if qlen() != 0 || size() != 0 {
		t.Fatalf("post-drain len=%d size=%d want 0", qlen(), size())
	}
	if bbIpqNotified(q) {
		t.Fatal("drain did not consume pending notification")
	}
	if n := drain(); n != 0 {
		t.Fatalf("re-drain=%d want 0", n)
	}
}

// Detail 11: inProgress — atomic count of popped-not-recycled elements.
func TestDetail11_InProgressAccounting(t *testing.T) {
	s := &Server{}
	q := newIPQueue[int](s, "bb-ip")
	push := q.push
	pop := q.pop
	popOne := q.popOne
	recycle := q.recycle
	inprog := q.inProgress
	if inprog() != 0 {
		t.Fatalf("fresh inProgress=%d", inprog())
	}
	for i := 0; i < 10; i++ {
		if _, err := push(i); err != nil {
			t.Fatal(err)
		}
	}
	// popOne does not contribute to inProgress.
	v, ok := popOne()
	if !ok || v != 0 {
		t.Fatalf("popOne=(%v,%v)", v, ok)
	}
	if ip := inprog(); ip != 0 {
		t.Fatalf("inProgress after popOne=%d want 0", ip)
	}
	got := pop()
	if ip := inprog(); ip != 9 {
		t.Fatalf("inProgress=%d want 9", ip)
	}
	recycle(&got)
	if ip := inprog(); ip != 0 {
		t.Fatalf("inProgress after recycle=%d want 0", ip)
	}
	// pop on empty does not change the count.
	if got := pop(); got != nil {
		t.Fatalf("empty pop=%v", got)
	}
	if ip := inprog(); ip != 0 {
		t.Fatalf("inProgress=%d want 0", ip)
	}
}

// Detail 12: unregister — nil-safe; deletes registry entry; push/pop still
// work afterwards.
func TestDetail12_Unregister(t *testing.T) {
	var nilq *ipQueue[int]
	nilunreg := nilq.unregister
	nilunreg() // must not panic.
	s := &Server{}
	q := newIPQueue[int](s, "bb-unreg")
	unreg := q.unregister
	push := q.push
	pop := q.pop
	if _, ok := s.ipQueues.Load("bb-unreg"); !ok {
		t.Fatal("queue not registered")
	}
	unreg()
	if _, ok := s.ipQueues.Load("bb-unreg"); ok {
		t.Fatal("unregister did not delete registry entry")
	}
	// Ops still work.
	if _, err := push(5); err != nil {
		t.Fatalf("push after unregister: %v", err)
	}
	got := pop()
	if len(got) != 1 || got[0] != 5 {
		t.Fatalf("pop after unregister=%v", got)
	}
	// Double unregister is safe.
	unreg()
}

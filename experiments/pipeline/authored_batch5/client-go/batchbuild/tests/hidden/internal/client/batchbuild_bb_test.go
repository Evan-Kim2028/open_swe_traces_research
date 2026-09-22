package client

import (
	"testing"

	"github.com/pingcap/kvproto/pkg/tikvpb"
)

// Hidden suite for unit batchbuild. One TestDetailNN per DETAILS.md line.

func bbEntry(pri uint64, host string) *batchCommandsEntry {
	return &batchCommandsEntry{
		req:           &tikvpb.BatchCommandsRequest_Request{},
		res:           make(chan *tikvpb.BatchCommandsResponse_Response, 1),
		forwardedHost: host,
		pri:           pri,
	}
}

// bbCollectAll returns a collect func recording emitted entries in call order.
func bbCollectAll(order *[]*batchCommandsEntry) func(uint64, *batchCommandsEntry) {
	return func(id uint64, e *batchCommandsEntry) {
		*order = append(*order, e)
	}
}

// TestDetail01: high-priority entries (priority() >= highTaskPriority) are
// exempt from the limit count and keep the build draining.
func TestDetail01(t *testing.T) {
	// limit=2 over {high, low, low} emits all three.
	b := newBatchCommandsBuilder(16)
	hi := bbEntry(highTaskPriority+1, "")
	lo1 := bbEntry(5, "")
	lo2 := bbEntry(1, "")
	b.push(lo2)
	b.push(hi)
	b.push(lo1)
	var order []*batchCommandsEntry
	req, _ := b.buildWithLimit(2, bbCollectAll(&order))
	if req == nil || len(order) != 3 {
		t.Fatalf("limit=2 over {hi,5,1}: emitted %d entries, want 3", len(order))
	}

	// A high-priority head is sent even when the limit is already exhausted.
	b2 := newBatchCommandsBuilder(16)
	hiOnly := bbEntry(highTaskPriority, "")
	b2.push(hiOnly)
	order = nil
	req, _ = b2.buildWithLimit(0, bbCollectAll(&order))
	if req == nil || len(order) != 1 || order[0] != hiOnly {
		t.Fatalf("limit=0 with high-priority head: emitted %d, want the head only", len(order))
	}

	// Normal entries still consume the limit.
	b3 := newBatchCommandsBuilder(16)
	hi5 := bbEntry(5, "")
	lo1b := bbEntry(1, "")
	b3.push(lo1b)
	b3.push(hi5)
	order = nil
	req, _ = b3.buildWithLimit(1, bbCollectAll(&order))
	if req == nil || len(order) != 1 || order[0] != hi5 {
		t.Fatalf("limit=1 over {5,1}: emitted %d, want just the higher-priority entry", len(order))
	}
}

// TestDetail02: forwardedHost entries route into per-host requests; every
// built entry consumes one request id.
func TestDetail02(t *testing.T) {
	b := newBatchCommandsBuilder(16)
	main := bbEntry(5, "")
	fwd1 := bbEntry(3, "host-a")
	fwd2 := bbEntry(1, "host-b")
	b.push(main)
	b.push(fwd1)
	b.push(fwd2)
	var order []*batchCommandsEntry
	req, fwds := b.buildWithLimit(10, bbCollectAll(&order))
	if req == nil || len(req.Requests) != 1 {
		t.Fatalf("main request: got %d requests, want 1", len(req.GetRequests()))
	}
	if len(fwds) != 2 || len(fwds["host-a"].GetRequests()) != 1 || len(fwds["host-b"].GetRequests()) != 1 {
		t.Fatalf("forwardingReqs: got %d hosts, want host-a and host-b with one request each", len(fwds))
	}
	// One id per built entry, unique across the main and forwarded requests.
	seen := map[uint64]bool{}
	total := 0
	for _, id := range req.GetRequestIds() {
		seen[id] = true
		total++
	}
	for _, fr := range fwds {
		for _, id := range fr.GetRequestIds() {
			seen[id] = true
			total++
		}
	}
	if total != 3 || len(seen) != 3 {
		t.Fatalf("request ids: %d total, %d unique; want 3 unique", total, len(seen))
	}
}

// TestDetail03: reset() drops canceled queue entries and leaves the builder
// usable for the next batch. (Shape only: whether live entries survive or the
// id counter is rewound is not derivable and is not asserted.)
func TestDetail03(t *testing.T) {
	b := newBatchCommandsBuilder(16)
	dead := bbEntry(1, "")
	dead.canceled = 1
	live := bbEntry(2, "")
	b.push(dead)
	b.push(live)
	b.reset()
	var order []*batchCommandsEntry
	b.buildWithLimit(10, bbCollectAll(&order))
	for _, e := range order {
		if e == dead {
			t.Fatalf("canceled entry was emitted after reset")
		}
	}
	// Builder remains usable after reset: a freshly pushed entry is emitted.
	fresh := bbEntry(4, "")
	b.push(fresh)
	order = nil
	b.buildWithLimit(10, bbCollectAll(&order))
	found := false
	for _, e := range order {
		if e == fresh {
			found = true
		}
	}
	if !found {
		t.Fatalf("builder unusable after reset: freshly pushed entry not emitted")
	}
}

// TestDetail04: canceled entries are skipped by the build and never emitted.
func TestDetail04(t *testing.T) {
	b := newBatchCommandsBuilder(16)
	dead := bbEntry(9, "")
	dead.canceled = 1
	live1 := bbEntry(5, "")
	live2 := bbEntry(3, "")
	b.push(dead)
	b.push(live1)
	b.push(live2)
	var order []*batchCommandsEntry
	req, _ := b.buildWithLimit(10, bbCollectAll(&order))
	for _, e := range order {
		if e.isCanceled() {
			t.Fatalf("canceled entry emitted by build")
		}
	}
	if req == nil || len(req.GetRequestIds()) != len(order) {
		t.Fatalf("request/ids mismatch: %d ids for %d entries", len(req.GetRequestIds()), len(order))
	}
}

// TestDetail05: cancel(err) errors every queued entry (err set, res closed)
// and empties the queue.
func TestDetail05(t *testing.T) {
	b := newBatchCommandsBuilder(16)
	e1 := bbEntry(1, "")
	e2 := bbEntry(7, "")
	b.push(e1)
	b.push(e2)
	sentinel := errBatchbuildSentinel{}
	b.cancel(sentinel)
	if b.len() != 0 {
		t.Fatalf("cancel left %d queued entries", b.len())
	}
	for _, e := range []*batchCommandsEntry{e1, e2} {
		if e.err != sentinel {
			t.Fatalf("entry err = %v, want the cancel error", e.err)
		}
		select {
		case _, ok := <-e.res:
			if ok {
				t.Fatalf("entry res channel delivered a value instead of being closed")
			}
		default:
			t.Fatalf("entry res channel not closed by cancel")
		}
	}
}

type errBatchbuildSentinel struct{}

func (errBatchbuildSentinel) Error() string { return "bb sentinel" }

// TestDetail06: emitted order follows the heap rather than insertion order —
// the maximum-priority entry is emitted first; an empty build returns a nil
// request. (Shape only: the exact tail order is heap-internal.)
func TestDetail06(t *testing.T) {
	b := newBatchCommandsBuilder(16)
	e1 := bbEntry(1, "")
	e5 := bbEntry(5, "")
	e3 := bbEntry(3, "")
	b.push(e1)
	b.push(e5)
	b.push(e3)
	var order []*batchCommandsEntry
	req, _ := b.buildWithLimit(10, bbCollectAll(&order))
	if req == nil {
		t.Fatalf("build over 3 entries returned nil request")
	}
	if len(order) != 3 || order[0] != e5 {
		got := []uint64{}
		for _, e := range order {
			got = append(got, e.pri)
		}
		t.Fatalf("emitted priority order %v, want max priority first", got)
	}

	empty := newBatchCommandsBuilder(16)
	order = nil
	req, fwds := empty.buildWithLimit(10, bbCollectAll(&order))
	if req != nil {
		t.Fatalf("empty build returned non-nil request %+v", req)
	}
	if len(fwds) != 0 {
		t.Fatalf("empty build returned %d forwarding requests", len(fwds))
	}
}

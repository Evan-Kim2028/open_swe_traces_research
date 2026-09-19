// Hidden black-box property suite for the range-task unit.
// Exported API only: NewRangeTaskRunner / SetRegionsPerTask / RunOnRange /
// CompletedRegions / FailedRegions, driven through a mock TiKV cluster.
// Seed 20260919. Any correct implementation of the contract must pass.

package rangetask_test

import (
	"bytes"
	"context"
	"errors"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"example.internal/kvstore/v2/kv"
	tikv "example.internal/kvstore/v2/kvclient"
	"example.internal/kvstore/v2/testutils"
	"example.internal/kvstore/v2/txnkv/rangetask"
)

const bbRangeSeed = int64(20260919)

// bbPartition is the region layout bootstrapped into the mock cluster:
// contiguous ranges [bounds[i], bounds[i+1]), bounds[0]=nil, last end=nil.
type bbPartition [][]byte

func bbRandKey(r *rand.Rand, alphabet int) []byte {
	n := 1 + r.Intn(2)
	k := make([]byte, n)
	for i := range k {
		k[i] = byte('a' + r.Intn(alphabet))
	}
	return k
}

func bbNewPartition(r *rand.Rand) bbPartition {
	n := r.Intn(18)
	seen := map[string]bool{}
	var cuts [][]byte
	for len(cuts) < n {
		k := bbRandKey(r, 6)
		if !seen[string(k)] {
			seen[string(k)] = true
			cuts = append(cuts, k)
		}
	}
	sort.Slice(cuts, func(i, j int) bool { return bytes.Compare(cuts[i], cuts[j]) < 0 })
	return append(bbPartition{nil}, cuts...)
}

// regionRanges lists [start,end) ranges of the partition.
func (p bbPartition) regionRanges() []kv.KeyRange {
	var out []kv.KeyRange
	for i := 0; i < len(p); i++ {
		var end []byte
		if i+1 < len(p) {
			end = p[i+1]
		}
		out = append(out, kv.KeyRange{StartKey: p[i], EndKey: end})
	}
	return out
}

// intersect clips a region range to [s,e) (empty bound = infinity); returns
// false when the intersection is empty.
func bbClip(rs, re, s, e []byte) (kv.KeyRange, bool) {
	if len(e) > 0 && len(rs) > 0 && bytes.Compare(rs, e) >= 0 {
		return kv.KeyRange{}, false
	}
	if len(re) > 0 && len(s) > 0 && bytes.Compare(re, s) <= 0 {
		return kv.KeyRange{}, false
	}
	st := rs
	if len(s) > 0 && (len(st) == 0 || bytes.Compare(s, st) > 0) {
		st = s
	}
	en := re
	if len(e) > 0 && (len(en) == 0 || bytes.Compare(e, en) < 0) {
		en = e
	}
	// An empty intersection (st >= en) is not a task.
	if len(en) > 0 && bytes.Compare(st, en) >= 0 {
		return kv.KeyRange{}, false
	}
	return kv.KeyRange{StartKey: st, EndKey: en}, true
}

// expectedCalls: the sub-ranges the handler must observe — clipped region
// ranges grouped in batches of regionsPerTask.
func bbExpected(p bbPartition, s, e []byte, regionsPerTask int) []kv.KeyRange {
	var clipped []kv.KeyRange
	for _, rr := range p.regionRanges() {
		if c, ok := bbClip(rr.StartKey, rr.EndKey, s, e); ok {
			clipped = append(clipped, c)
		}
	}
	var calls []kv.KeyRange
	for i := 0; i < len(clipped); i += regionsPerTask {
		last := i + regionsPerTask - 1
		if last >= len(clipped) {
			last = len(clipped) - 1
		}
		calls = append(calls, kv.KeyRange{StartKey: clipped[i].StartKey, EndKey: clipped[last].EndKey})
	}
	return calls
}

func bbSameRanges(a, b []kv.KeyRange) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Slice(a, func(i, j int) bool { return bytes.Compare(a[i].StartKey, a[j].StartKey) < 0 })
	sort.Slice(b, func(i, j int) bool { return bytes.Compare(b[i].StartKey, b[j].StartKey) < 0 })
	for i := range a {
		if !bytes.Equal(a[i].StartKey, b[i].StartKey) || !bytes.Equal(a[i].EndKey, b[i].EndKey) {
			return false
		}
	}
	return true
}

// Contract: the walk covers each region range exactly once, batched up to
// regionsPerTask, for any task range/concurrency/batch size; the handler sees
// a fresh backoffer-capable ctx.
func TestRangeTaskBBCoverage(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbRangeSeed))
	for trial := 0; trial < 400; trial++ {
		cli, cluster, pdCli, err := testutils.NewMockTiKV("", nil)
		assert.Nil(err)
		p := bbNewPartition(r)
		testutils.BootstrapWithMultiRegions(cluster, p[1:]...)
		store, err := tikv.NewTestTiKVStore(cli, pdCli, nil, nil, 0)
		assert.Nil(err)
		var s, e []byte
		if r.Intn(4) > 0 {
			s = bbRandKey(r, 8)
		}
		if r.Intn(4) > 0 {
			e = bbRandKey(r, 8)
		}
		if len(s) > 0 && len(e) > 0 && bytes.Compare(s, e) >= 0 {
			continue
		}
		if len(s) > 0 && len(e) > 0 && bytes.Equal(s, e) {
			continue
		}
		rpt := 1 + r.Intn(6)
		conc := 1 + r.Intn(6)
		var mu sync.Mutex
		var got []kv.KeyRange
		handler := func(ctx context.Context, kr kv.KeyRange) (rangetask.TaskStat, error) {
			mu.Lock()
			got = append(got, kr)
			mu.Unlock()
			return rangetask.TaskStat{CompletedRegions: 1}, nil
		}
		runner := rangetask.NewRangeTaskRunner("bb", store, conc, handler)
		runner.SetRegionsPerTask(rpt)
		assert.Nil(runner.RunOnRange(context.Background(), s, e))
		want := bbExpected(p, s, e, rpt)
		mu.Lock()
		assert.True(bbSameRanges(got, want), "trial %d: ranges %v want %v", trial, got, want)
		mu.Unlock()
		assert.Equal(len(want), runner.CompletedRegions(), "completed counter = handler calls")
		assert.Equal(0, runner.FailedRegions())
		assert.Nil(store.Close())
	}
}

// Contract: a handler error cancels the whole task, propagates, and is
// counted in FailedRegions.
func TestRangeTaskBBError(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbRangeSeed + 1))
	cli, cluster, pdCli, err := testutils.NewMockTiKV("", nil)
	assert.Nil(err)
	p := bbPartition{nil, []byte("k"), []byte("m"), []byte("s")}
	testutils.BootstrapWithMultiRegions(cluster, p[1:]...)
	store, err := tikv.NewTestTiKVStore(cli, pdCli, nil, nil, 0)
	assert.Nil(err)
	defer store.Close()
	sentinel := errors.New("bb-task-error")
	for trial := 0; trial < 300; trial++ {
		errAt := r.Intn(4)
		var calls int
		handler := func(ctx context.Context, kr kv.KeyRange) (rangetask.TaskStat, error) {
			idx := calls
			calls++
			if idx == errAt {
				return rangetask.TaskStat{FailedRegions: 1}, sentinel
			}
			return rangetask.TaskStat{CompletedRegions: 1}, nil
		}
		runner := rangetask.NewRangeTaskRunner("bb-err", store, 1+r.Intn(4), handler)
		runner.SetRegionsPerTask(1)
		err := runner.RunOnRange(context.Background(), nil, nil)
		assert.NotNil(err, "handler error must propagate")
		assert.Equal(1, runner.FailedRegions())
		assert.Less(runner.CompletedRegions(), 4, "task must stop on the first error")
	}
	_ = sentinel
}

// Contract: context cancellation stops the task promptly.
func TestRangeTaskBBCancel(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbRangeSeed + 2))
	cli, cluster, pdCli, err := testutils.NewMockTiKV("", nil)
	assert.Nil(err)
	p := bbPartition{nil, []byte("f"), []byte("k"), []byte("p"), []byte("u")}
	testutils.BootstrapWithMultiRegions(cluster, p[1:]...)
	store, err := tikv.NewTestTiKVStore(cli, pdCli, nil, nil, 0)
	assert.Nil(err)
	defer store.Close()
	for trial := 0; trial < 40; trial++ {
		ctx, cancel := context.WithCancel(context.Background())
		handler := func(hctx context.Context, kr kv.KeyRange) (rangetask.TaskStat, error) {
			select {
			case <-hctx.Done():
				return rangetask.TaskStat{FailedRegions: 1}, hctx.Err()
			case <-time.After(20 * time.Millisecond):
				return rangetask.TaskStat{CompletedRegions: 1}, nil
			}
		}
		runner := rangetask.NewRangeTaskRunner("bb-cancel", store, 2, handler)
		runner.SetRegionsPerTask(1)
		go func() {
			time.Sleep(time.Duration(r.Intn(30)) * time.Millisecond)
			cancel()
		}()
		start := time.Now()
		err := runner.RunOnRange(ctx, nil, nil)
		if err != nil {
			assert.Less(time.Since(start), 30*time.Second, "cancellation must be prompt")
		}
		cancel()
	}
}

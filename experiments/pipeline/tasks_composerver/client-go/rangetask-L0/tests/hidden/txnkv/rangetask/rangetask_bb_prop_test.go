package rangetask_test

import (
	"bytes"
	"context"
	"errors"
	"math/rand"
	"sort"
	"testing"

	"example.internal/kvstore/v2/kv"
	tikv "example.internal/kvstore/v2/kvclient"
	"example.internal/kvstore/v2/testutils"
	"example.internal/kvstore/v2/txnkv/rangetask"
)

const rangeTaskSeed = 20260919
const rangeTaskCases = 10000

func bbMakeRange(startKey string, endKey string) kv.KeyRange {
	return kv.KeyRange{
		StartKey: []byte(startKey),
		EndKey:   []byte(endKey),
	}
}

func bbBatchRanges(ranges []kv.KeyRange, batchSize int) []kv.KeyRange {
	result := make([]kv.KeyRange, 0, len(ranges))
	for i := 0; i < len(ranges); i += batchSize {
		lastRange := i + batchSize - 1
		if lastRange >= len(ranges) {
			lastRange = len(ranges) - 1
		}
		result = append(result, kv.KeyRange{
			StartKey: ranges[i].StartKey,
			EndKey:   ranges[lastRange].EndKey,
		})
	}
	return result
}

func bbCollectRanges(c chan *kv.KeyRange) []kv.KeyRange {
	c <- nil
	ranges := make([]kv.KeyRange, 0)
	for {
		r := <-c
		if r == nil {
			break
		}
		ranges = append(ranges, *r)
	}
	return ranges
}

func bbSortRanges(ranges []kv.KeyRange) {
	sort.Slice(ranges, func(i, j int) bool {
		return bytes.Compare(ranges[i].StartKey, ranges[j].StartKey) < 0
	})
}

func bbRangesEqual(a, b []kv.KeyRange) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytes.Equal(a[i].StartKey, b[i].StartKey) || !bytes.Equal(a[i].EndKey, b[i].EndKey) {
			return false
		}
	}
	return true
}

func bbIntersectingRegions(all []kv.KeyRange, start, end []byte) []kv.KeyRange {
	out := make([]kv.KeyRange, 0)
	for _, r := range all {
		rStart := r.StartKey
		rEnd := r.EndKey
		if len(end) > 0 && bytes.Compare(rStart, end) >= 0 {
			continue
		}
		if len(start) > 0 && len(rEnd) > 0 && bytes.Compare(rEnd, start) <= 0 {
			continue
		}
		clipStart := rStart
		if len(start) > 0 && bytes.Compare(clipStart, start) < 0 {
			clipStart = start
		}
		clipEnd := rEnd
		if len(end) > 0 && (len(clipEnd) == 0 || bytes.Compare(clipEnd, end) > 0) {
			clipEnd = end
		}
		if len(clipEnd) > 0 && bytes.Compare(clipStart, clipEnd) >= 0 {
			continue
		}
		out = append(out, kv.KeyRange{StartKey: clipStart, EndKey: clipEnd})
	}
	return out
}

type bbRangeTaskEnv struct {
	store            *tikv.KVStore
	allRegionRanges  []kv.KeyRange
	contractRanges   []kv.KeyRange
	contractExpected [][]kv.KeyRange
}

func newBBRangeTaskEnv(t *testing.T) *bbRangeTaskEnv {
	splitKeys := make([][]byte, 0, 26)
	for k := byte('a'); k <= byte('z'); k++ {
		splitKeys = append(splitKeys, []byte{k})
	}
	allRegionRanges := []kv.KeyRange{bbMakeRange("", "a")}
	for i := 0; i < len(splitKeys)-1; i++ {
		allRegionRanges = append(allRegionRanges, kv.KeyRange{
			StartKey: splitKeys[i],
			EndKey:   splitKeys[i+1],
		})
	}
	allRegionRanges = append(allRegionRanges, bbMakeRange("z", ""))

	client, cluster, pdClient, err := testutils.NewMockTiKV("", nil)
	if err != nil {
		t.Fatal(err)
	}
	testutils.BootstrapWithMultiRegions(cluster, splitKeys...)
	store, err := tikv.NewTestTiKVStore(client, pdClient, nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	contractRanges := []kv.KeyRange{
		bbMakeRange("", ""),
		bbMakeRange("", "b"),
		bbMakeRange("b", ""),
		bbMakeRange("b", "x"),
		bbMakeRange("a", "d"),
		bbMakeRange("a\x00", "d\x00"),
		bbMakeRange("a\xff\xff\xff", "c\xff\xff\xff"),
		bbMakeRange("a1", "a2"),
		bbMakeRange("a", "a"),
		bbMakeRange("a3", "a3"),
	}
	contractExpected := [][]kv.KeyRange{
		allRegionRanges,
		allRegionRanges[:2],
		allRegionRanges[2:],
		allRegionRanges[2:24],
		{
			bbMakeRange("a", "b"),
			bbMakeRange("b", "c"),
			bbMakeRange("c", "d"),
		},
		{
			bbMakeRange("a\x00", "b"),
			bbMakeRange("b", "c"),
			bbMakeRange("c", "d"),
			bbMakeRange("d", "d\x00"),
		},
		{
			bbMakeRange("a\xff\xff\xff", "b"),
			bbMakeRange("b", "c"),
			bbMakeRange("c", "c\xff\xff\xff"),
		},
		{bbMakeRange("a1", "a2")},
		{},
		{},
	}

	return &bbRangeTaskEnv{
		store:            store,
		allRegionRanges:  allRegionRanges,
		contractRanges:   contractRanges,
		contractExpected: contractExpected,
	}
}

func bbRandKey(rng *rand.Rand) []byte {
	switch rng.Intn(5) {
	case 0:
		return nil
	case 1:
		return []byte{byte('a' + rng.Intn(26))}
	default:
		n := rng.Intn(3) + 1
		b := make([]byte, n)
		for i := range b {
			b[i] = byte('a' + rng.Intn(26))
		}
		return b
	}
}

func bbOrderedPair(rng *rand.Rand) ([]byte, []byte) {
	a := bbRandKey(rng)
	b := bbRandKey(rng)
	if bytes.Compare(a, b) > 0 {
		a, b = b, a
	}
	return a, b
}

func TestRangeTaskCoverageProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(rangeTaskSeed))
	env := newBBRangeTaskEnv(t)
	defer func() { _ = env.store.Close() }()

	for i := 0; i < rangeTaskCases; i++ {
		start, end := bbOrderedPair(rng)
		if len(end) > 0 && len(start) > 0 && bytes.Compare(start, end) >= 0 {
			end = append(append([]byte(nil), start...), 0xff)
		}
		concurrency := 1 + rng.Intn(4)
		regionsPerTask := 1 + rng.Intn(5)
		expected := bbIntersectingRegions(env.allRegionRanges, start, end)
		want := bbBatchRanges(expected, regionsPerTask)

		ranges := make(chan *kv.KeyRange, len(want)+4)
		handler := func(ctx context.Context, r kv.KeyRange) (rangetask.TaskStat, error) {
			ranges <- &r
			return rangetask.TaskStat{CompletedRegions: 1}, nil
		}
		runner := rangetask.NewRangeTaskRunner("bb-prop", env.store, concurrency, handler)
		runner.SetRegionsPerTask(regionsPerTask)
		if err := runner.RunOnRange(context.Background(), start, end); err != nil {
			t.Fatalf("case %d run: %v", i, err)
		}
		got := bbCollectRanges(ranges)
		bbSortRanges(got)
		if !bbRangesEqual(got, want) {
			t.Fatalf("case %d ranges %+v want %+v", i, got, want)
		}
		if runner.CompletedRegions() != len(want) {
			t.Fatalf("case %d completed=%d want %d", i, runner.CompletedRegions(), len(want))
		}
		if runner.FailedRegions() != 0 {
			t.Fatalf("case %d failed=%d want 0", i, runner.FailedRegions())
		}
	}
}

func TestRangeTaskContractExamples(t *testing.T) {
	env := newBBRangeTaskEnv(t)
	defer func() { _ = env.store.Close() }()

	for regionsPerTask := 1; regionsPerTask <= 5; regionsPerTask++ {
		for i, r := range env.contractRanges {
			ranges := make(chan *kv.KeyRange, 64)
			handler := func(ctx context.Context, kr kv.KeyRange) (rangetask.TaskStat, error) {
				ranges <- &kr
				return rangetask.TaskStat{CompletedRegions: 1}, nil
			}
			runner := rangetask.NewRangeTaskRunner("bb-contract", env.store, 2, handler)
			runner.SetRegionsPerTask(regionsPerTask)
			want := bbBatchRanges(env.contractExpected[i], regionsPerTask)
			if err := runner.RunOnRange(context.Background(), r.StartKey, r.EndKey); err != nil {
				t.Fatalf("contract %d regionsPerTask=%d: %v", i, regionsPerTask, err)
			}
			got := bbCollectRanges(ranges)
			bbSortRanges(got)
			if !bbRangesEqual(got, want) {
				t.Fatalf("contract %d regionsPerTask=%d got %+v want %+v", i, regionsPerTask, got, want)
			}
			if runner.CompletedRegions() != len(want) {
				t.Fatalf("contract %d completed=%d want %d", i, runner.CompletedRegions(), len(want))
			}
		}
	}
}

func TestRangeTaskErrorProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(rangeTaskSeed + 1))
	env := newBBRangeTaskEnv(t)
	defer func() { _ = env.store.Close() }()

	for i := 0; i < rangeTaskCases; i++ {
		idx := rng.Intn(len(env.contractRanges))
		r := env.contractRanges[idx]
		sub := env.contractExpected[idx]
		if len(sub) == 0 {
			continue
		}
		errIdx := rng.Intn(len(sub))
		errKey := sub[errIdx].StartKey
		concurrency := 1 + rng.Intn(3)

		handler := func(ctx context.Context, kr kv.KeyRange) (rangetask.TaskStat, error) {
			stat := rangetask.TaskStat{}
			if bytes.Equal(kr.StartKey, errKey) {
				stat.FailedRegions = 1
				return stat, errors.New("bb injected error")
			}
			stat.CompletedRegions = 1
			return stat, nil
		}
		runner := rangetask.NewRangeTaskRunner("bb-err", env.store, concurrency, handler)
		runner.SetRegionsPerTask(1)
		err := runner.RunOnRange(context.Background(), r.StartKey, r.EndKey)
		if err == nil {
			t.Fatalf("case %d expected error at %+q", i, errKey)
		}
		if runner.CompletedRegions() >= len(sub) {
			t.Fatalf("case %d completed all %d subranges after error", i, len(sub))
		}
		if runner.FailedRegions() != 1 {
			t.Fatalf("case %d failed=%d want 1", i, runner.FailedRegions())
		}
	}
}

func TestRangeTaskUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(rangeTaskSeed + 3))
	env := newBBRangeTaskEnv(t)
	defer func() { _ = env.store.Close() }()

	mentioned := map[string]bool{
		"": true, "b": true, "x": true, "a": true, "d": true, "a1": true, "a2": true, "a3": true,
	}
	for i := 0; i < rangeTaskCases; i++ {
		start := []byte{byte('0' + rng.Intn(7)), byte('A' + rng.Intn(26))}
		end := append(append([]byte(nil), start...), byte('0'+rng.Intn(7)), byte('Z'-rng.Intn(10)))
		if bytes.Compare(start, end) >= 0 {
			end = append(end, 0xff)
		}
		sk := string(start)
		if mentioned[sk] {
			start[0]++
		}
		expected := bbIntersectingRegions(env.allRegionRanges, start, end)
		ranges := make(chan *kv.KeyRange, len(expected)+2)
		handler := func(ctx context.Context, kr kv.KeyRange) (rangetask.TaskStat, error) {
			ranges <- &kr
			return rangetask.TaskStat{CompletedRegions: 1}, nil
		}
		runner := rangetask.NewRangeTaskRunner("bb-unseen", env.store, 1+rng.Intn(2), handler)
		regionsPerTask := 1 + rng.Intn(4)
		runner.SetRegionsPerTask(regionsPerTask)
		if err := runner.RunOnRange(context.Background(), start, end); err != nil {
			t.Fatalf("unseen %d: %v", i, err)
		}
		got := bbCollectRanges(ranges)
		want := bbBatchRanges(expected, regionsPerTask)
		bbSortRanges(got)
		if !bbRangesEqual(got, want) {
			t.Fatalf("unseen %d ranges %+v want %+v", i, got, want)
		}
		if runner.CompletedRegions() != len(want) {
			t.Fatalf("unseen %d completed=%d want %d", i, runner.CompletedRegions(), len(want))
		}
	}
}

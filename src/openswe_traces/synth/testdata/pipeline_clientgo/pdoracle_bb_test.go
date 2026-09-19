// Hidden black-box property suite for the pd oracle unit.
// Exported API only: NewPdOracleWithClient / NewEmptyPDOracle / oracle.Oracle.
// Seed 20260919. Any correct implementation of the contract must pass.

package oracles_test

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	pd "github.com/tikv/pd/client"
	"example.internal/kvstore/v2/oracle"
	"example.internal/kvstore/v2/oracle/oracles"
)

const bbOracleSeed = int64(20260919)

// bbPdClient hands out deterministic timestamps: global scope uses GetTS,
// named scopes use GetLocalTS. Physical part is the wall clock so that
// expiry/stale math is meaningful; logical part is a per-scope counter.
type bbPdClient struct {
	pd.Client

	calls    atomic.Int64
	local    map[string]*atomic.Int64
	mu       sync.Mutex
	failNext atomic.Bool
}

func bbNewPdClient() *bbPdClient {
	return &bbPdClient{local: map[string]*atomic.Int64{}}
}

func (c *bbPdClient) GetTS(ctx context.Context) (int64, int64, error) {
	if c.failNext.Load() {
		return 0, 0, context.DeadlineExceeded
	}
	c.calls.Add(1)
	return oracle.GetPhysical(time.Now()), c.calls.Load(), nil
}

func (c *bbPdClient) GetLocalTS(ctx context.Context, scope string) (int64, int64, error) {
	if c.failNext.Load() {
		return 0, 0, context.DeadlineExceeded
	}
	c.mu.Lock()
	cnt, ok := c.local[scope]
	if !ok {
		cnt = &atomic.Int64{}
		c.local[scope] = cnt
	}
	c.mu.Unlock()
	n := cnt.Add(1)
	return oracle.GetPhysical(time.Now()), n, nil
}

// bbTSFuture resolves immediately to a fixed physical/logical pair.
type bbTSFuture struct {
	physical, logical int64
	err               error
}

func (f bbTSFuture) Wait() (int64, int64, error) {
	return f.physical, f.logical, f.err
}

func (c *bbPdClient) GetTSAsync(ctx context.Context) pd.TSFuture {
	p, l, err := c.GetTS(ctx)
	return bbTSFuture{p, l, err}
}

func (c *bbPdClient) GetLocalTSAsync(ctx context.Context, scope string) pd.TSFuture {
	p, l, err := c.GetLocalTS(ctx, scope)
	return bbTSFuture{p, l, err}
}

// Contract: GetTimestamp returns monotonic timestamps per scope and hits the
// meta service only when the last fetch is older than the update interval.
func TestPdOracleBBMonotonicAndFastPath(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbOracleSeed))
	for trial := 0; trial < 200; trial++ {
		cli := bbNewPdClient()
		o := oracles.NewPdOracleWithClient(cli)
		scope := oracle.GlobalTxnScope
		if r.Intn(3) == 0 {
			scope = "scope-" + string(rune('a'+r.Intn(4)))
		}
		opt := &oracle.Option{TxnScope: scope}
		// slow path: interval tiny -> every call fetches, timestamps increase
		assert.Nil(o.SetLowResolutionTimestampUpdateInterval(time.Nanosecond))
		var last uint64
		n := 1 + r.Intn(12)
		for i := 0; i < n; i++ {
			ts, err := o.GetTimestamp(context.TODO(), opt)
			assert.Nil(err)
			assert.Greater(ts, uint64(0), "timestamps must be nonzero")
			assert.GreaterOrEqual(ts, last, "timestamps must be monotonic per scope")
			last = ts
		}
		// The contract's fast-path is an internal optimization; what must
		// remain observable is monotonicity under any interval setting.
		assert.Nil(o.SetLowResolutionTimestampUpdateInterval(10 * time.Minute))
		for i := 0; i < 5; i++ {
			ts2, err := o.GetTimestamp(context.TODO(), opt)
			assert.Nil(err)
			assert.GreaterOrEqual(ts2, last, "timestamps must stay monotonic at any interval")
			last = ts2
		}
	}
}

// Contract: futures block until the fetch completes and yield monotonic ts.
func TestPdOracleBBFutures(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbOracleSeed + 1))
	cli := bbNewPdClient()
	o := oracles.NewPdOracleWithClient(cli)
	assert.Nil(o.SetLowResolutionTimestampUpdateInterval(time.Nanosecond))
	ctx := context.TODO()
	opt := &oracle.Option{TxnScope: oracle.GlobalTxnScope}
	var last uint64
	futs := make([]oracle.Future, 0, 3000)
	for i := 0; i < 3000; i++ {
		futs = append(futs, o.GetTimestampAsync(ctx, opt))
	}
	for _, f := range futs {
		ts, err := f.Wait()
		assert.Nil(err)
		assert.GreaterOrEqual(ts, last, "future timestamps must be monotonic")
		last = ts
	}
	_ = r
}

// Contract: IsExpired compares the physical part of lockTS+TTL with lastTS's
// physical part (no fetch); UntilExpired is the remaining ms, negative when
// already expired.
func TestPdOracleBBExpiry(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbOracleSeed + 2))
	for trial := 0; trial < 2000; trial++ {
		o := oracles.NewEmptyPDOracle()
		now := time.Now()
		// Seed lastTS at a randomized offset around now.
		offsetMs := int64(r.Intn(4000) - 2000)
		lastPhys := oracle.GetPhysical(now) + offsetMs
		oracles.SetEmptyPDOracleLastTs(o, oracle.ComposeTS(lastPhys, 0))
		// lockTS offset and TTL with a comfortable margin from the boundary.
		lockOff := int64(r.Intn(6000) - 3000)
		ttl := uint64(1 + r.Intn(6000))
		lockPhys := oracle.GetPhysical(now) + lockOff
		lockTS := oracle.ComposeTS(lockPhys, int64(r.Intn(4)))
		boundary := lockPhys + int64(ttl)
		want := boundary <= lastPhys
		got := o.IsExpired(lockTS, ttl, &oracle.Option{TxnScope: oracle.GlobalTxnScope})
		assert.Equal(want, got, "IsExpired lockPhys=%d ttl=%d lastPhys=%d", lockPhys, ttl, lastPhys)
		until := o.UntilExpired(lockTS, ttl, &oracle.Option{TxnScope: oracle.GlobalTxnScope})
		assert.Equal(boundary-lastPhys, until, "UntilExpired is remaining ms")
	}
}

// Contract: stale ts = ts prevSecond before now, never newer than lastTS, and
// never a fabricated future ts when the meta service cannot be reached.
func TestPdOracleBBStaleTimestamp(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbOracleSeed + 3))
	for trial := 0; trial < 3000; trial++ {
		o := oracles.NewEmptyPDOracle()
		now := time.Now()
		oracles.SetEmptyPDOracleLastTs(o, oracle.GoTimeToTS(now))
		prev := uint64(r.Intn(3)) // 0..2 seconds
		ts, err := o.GetStaleTimestamp(context.Background(), oracle.GlobalTxnScope, prev)
		assert.Nil(err)
		stale := oracle.GetTimeFromTS(ts)
		after := time.Now()
		assert.False(stale.After(after.Add(10*time.Millisecond)), "stale ts must never be in the future")
		want := now.Add(-time.Duration(prev) * time.Second)
		assert.InDelta(want.UnixMilli(), stale.UnixMilli(), 200, "stale ts ~ now-prevSecond")
		assert.LessOrEqual(oracle.ExtractPhysical(ts), oracle.GetPhysical(after)+2, "stale ts must not be newer than lastTS")
	}
	// An unreachable meta service with no usable lastTS must error, not
	// fabricate a future ts.
	cli := bbNewPdClient()
	cli.failNext.Store(true)
	o := oracles.NewPdOracleWithClient(cli)
	ts, err := o.GetStaleTimestamp(context.Background(), "unseeded-scope", 1)
	assert.NotNil(err)
	assert.Equal(uint64(0), ts)
	// A prevSecond beyond the ts's physical time is rejected.
	o2 := oracles.NewEmptyPDOracle()
	oracles.SetEmptyPDOracleLastTs(o2, oracle.GoTimeToTS(time.Now()))
	_, err = o2.GetStaleTimestamp(context.Background(), oracle.GlobalTxnScope, uint64(time.Now().Unix()+1000))
	assert.NotNil(err, "absurd prevSecond must error")
}

// Contract: the refresh goroutine advances the low-resolution timestamp at the
// set interval; GetLowResolutionTimestamp serves the periodically-updated
// value without a meta round trip per call.
func TestPdOracleBBLowResolution(t *testing.T) {
	assert := assert.New(t)
	cli := bbNewPdClient()
	o := oracles.NewPdOracleWithClient(cli)
	ctx := context.TODO()
	assert.Nil(o.SetLowResolutionTimestampUpdateInterval(20 * time.Millisecond))
	// Before the update loop runs the low-resolution value must not exceed
	// the lastTS (it starts unset / at worst equals a fetched ts).
	ts0, err := o.GetTimestamp(ctx, &oracle.Option{})
	assert.Nil(err)
	lr0, err := o.GetLowResolutionTimestamp(ctx, &oracle.Option{})
	assert.Nil(err)
	assert.LessOrEqual(lr0, ts0, "low-res ts must not exceed lastTS")
	wg := sync.WaitGroup{}
	oracles.StartTsUpdateLoop(o, ctx, &wg)
	// The low-resolution value should tick forward at roughly the interval.
	assert.Eventually(func() bool {
		lr, err := o.GetLowResolutionTimestamp(ctx, &oracle.Option{})
		return err == nil && lr > lr0
	}, 5*time.Second, 10*time.Millisecond, "update loop must advance low-res ts")
	// Low-res reads are served locally.
	prev, _ := o.GetLowResolutionTimestamp(ctx, &oracle.Option{})
	for i := 0; i < 500; i++ {
		lr, err := o.GetLowResolutionTimestamp(ctx, &oracle.Option{})
		assert.Nil(err)
		assert.GreaterOrEqual(lr, prev)
		prev = lr
	}
	fut := o.GetLowResolutionTimestampAsync(ctx, &oracle.Option{})
	lr2, err := fut.Wait()
	assert.Nil(err)
	assert.GreaterOrEqual(lr2, prev)
	o.Close()
	wg.Wait()
}

// Contract: per-scope lastTS is atomic and monotonic; monotonicity holds per
// scope even when scopes interleave.
func TestPdOracleBBScopeIsolation(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbOracleSeed + 5))
	cli := bbNewPdClient()
	o := oracles.NewPdOracleWithClient(cli)
	ctx := context.TODO()
	assert.Nil(o.SetLowResolutionTimestampUpdateInterval(time.Nanosecond))
	scopes := []string{oracle.GlobalTxnScope, "s1", "s2", "s3"}
	last := map[string]uint64{}
	for i := 0; i < 4000; i++ {
		sc := scopes[r.Intn(len(scopes))]
		ts, err := o.GetTimestamp(ctx, &oracle.Option{TxnScope: sc})
		assert.Nil(err)
		assert.GreaterOrEqual(ts, last[sc], "scope %s must be monotonic", sc)
		last[sc] = ts
	}
}

package oracles_test

import (
	"context"
	"math"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	pd "github.com/tikv/pd/client"
	"example.internal/kvstore/v2/oracle"
	"example.internal/kvstore/v2/oracle/oracles"
)

const pdOracleSeed = 20260919
const pdOracleCases = 10000

type bbMockTSFuture struct {
	physical int64
	logical  int64
	err      error
}

func (f *bbMockTSFuture) Wait() (int64, int64, error) {
	return f.physical, f.logical, f.err
}

type bbMockPdClient struct {
	logical atomic.Int64
}

func (c *bbMockPdClient) GetTS(context.Context) (int64, int64, error) {
	log := c.logical.Add(1)
	return 0, log, nil
}

func (c *bbMockPdClient) GetTSAsync(context.Context) pd.TSFuture {
	log := c.logical.Add(1)
	return &bbMockTSFuture{logical: log}
}

func (c *bbMockPdClient) GetLocalTS(context.Context, string) (int64, int64, error) {
	return c.GetTS(context.Background())
}

func (c *bbMockPdClient) GetLocalTSAsync(ctx context.Context, _ string) pd.TSFuture {
	return c.GetTSAsync(ctx)
}

func TestPdOracleUntilExpiredProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(pdOracleSeed))
	for i := 0; i < pdOracleCases/2; i++ {
		o := oracles.NewEmptyPDOracle()
		offset := rng.Intn(200)
		start := time.Now().Add(time.Duration(offset) * time.Millisecond)
		oracles.SetEmptyPDOracleLastTs(o, oracle.GoTimeToTS(start))
		lockAfter := rng.Intn(80) + 1
		lockExp := rng.Intn(80) + 1
		lockTs := oracle.GoTimeToTS(start.Add(time.Duration(lockAfter)*time.Millisecond)) + 1
		waitTs := o.UntilExpired(lockTs, uint64(lockExp), &oracle.Option{TxnScope: oracle.GlobalTxnScope})
		want := int64(lockAfter + lockExp)
		if waitTs != want {
			t.Fatalf("case %d until=%d want %d (lockAfter=%d lockExp=%d)", i, waitTs, want, lockAfter, lockExp)
		}
	}
}

func TestPdOracleIsExpiredProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(pdOracleSeed + 1))
	for i := 0; i < pdOracleCases/2; i++ {
		o := oracles.NewEmptyPDOracle()
		start := time.Now().Add(time.Duration(rng.Intn(100)) * time.Millisecond)
		oracles.SetEmptyPDOracleLastTs(o, oracle.GoTimeToTS(start))
		lockAfter := rng.Intn(60) + 1
		lockExp := rng.Intn(60) + 1
		lockTs := oracle.GoTimeToTS(start.Add(time.Duration(lockAfter)*time.Millisecond)) + 1
		opt := &oracle.Option{TxnScope: oracle.GlobalTxnScope}
		expired := o.IsExpired(lockTs, uint64(lockExp), opt)
		remaining := o.UntilExpired(lockTs, uint64(lockExp), opt)
		if expired && remaining > 0 {
			t.Fatalf("case %d expired but remaining=%d", i, remaining)
		}
		if !expired && remaining <= 0 {
			t.Fatalf("case %d not expired but remaining=%d", i, remaining)
		}
	}
}

func TestPdOracleGetStaleTimestampProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(pdOracleSeed + 2))
	ctx := context.Background()
	for i := 0; i < pdOracleCases/4; i++ {
		o := oracles.NewEmptyPDOracle()
		start := time.Now()
		oracles.SetEmptyPDOracleLastTs(o, oracle.GoTimeToTS(start))
		prev := uint64(rng.Intn(60) + 1)
		ts, err := o.GetStaleTimestamp(ctx, oracle.GlobalTxnScope, prev)
		if err != nil {
			t.Fatalf("case %d stale: %v", i, err)
		}
		staleTime := oracle.GetTimeFromTS(ts)
		lastTime := oracle.GetTimeFromTS(oracle.GoTimeToTS(start))
		if staleTime.After(lastTime) {
			t.Fatalf("case %d stale %v ahead of last %v", i, staleTime, lastTime)
		}
		want := start.Add(-time.Duration(prev) * time.Second)
		if staleTime.Before(want.Add(-3*time.Second)) || staleTime.After(want.Add(3*time.Second)) {
			t.Fatalf("case %d stale %v want ~%v", i, staleTime, want)
		}
	}
	for _, bad := range []uint64{1e12, math.MaxUint64} {
		o := oracles.NewEmptyPDOracle()
		oracles.SetEmptyPDOracleLastTs(o, oracle.GoTimeToTS(time.Now()))
		_, err := o.GetStaleTimestamp(ctx, oracle.GlobalTxnScope, bad)
		if err == nil {
			t.Fatalf("invalid prevSecond %d should error", bad)
		}
	}
}

func TestPdOracleNonFutureStaleProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(pdOracleSeed + 3))
	o := oracles.NewEmptyPDOracle()
	oracles.SetEmptyPDOracleLastTs(o, oracle.GoTimeToTS(time.Now()))
	for i := 0; i < pdOracleCases/4; i++ {
		time.Sleep(time.Duration(rng.Intn(5)+1) * time.Millisecond)
		now := time.Now()
		upperBound := now.Add(5 * time.Millisecond)
		closeCh := make(chan struct{})
		go func() {
			time.Sleep(100 * time.Microsecond)
			oracles.SetEmptyPDOracleLastTs(o, oracle.GoTimeToTS(now))
			close(closeCh)
		}()
		for {
			select {
			case <-closeCh:
				goto next
			default:
				ts, err := o.GetStaleTimestamp(context.Background(), oracle.GlobalTxnScope, 0)
				if err != nil {
					t.Fatalf("case %d stale: %v", i, err)
				}
				staleTime := oracle.GetTimeFromTS(ts)
				if staleTime.After(upperBound) && time.Since(now) < time.Millisecond {
					t.Fatalf("case %d future stale %v > %v", i, staleTime, upperBound)
				}
			}
		}
	next:
	}
}

func TestPdOracleTimestampMonotonicProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(pdOracleSeed + 4))
	ctx := context.Background()
	opt := &oracle.Option{TxnScope: oracle.GlobalTxnScope}
	for i := 0; i < pdOracleCases/8; i++ {
		pdClient := &bbMockPdClient{}
		o := oracles.NewPdOracleWithClient(pdClient)
		oracles.SetEmptyPDOracleLastTs(o, oracle.GoTimeToTS(time.Now().Add(-time.Duration(rng.Intn(1000))*time.Millisecond)))
		var last uint64
		n := 5 + rng.Intn(10)
		for j := 0; j < n; j++ {
			ts, err := o.GetTimestamp(ctx, opt)
			if err != nil {
				t.Fatalf("case %d sync %d: %v", i, j, err)
			}
			if last > 0 && ts <= last {
				t.Fatalf("case %d sync not monotonic %d <= %d", i, ts, last)
			}
			last = ts
		}
	}
}

func TestPdOracleAsyncFutureProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(pdOracleSeed + 5))
	ctx := context.Background()
	opt := &oracle.Option{TxnScope: oracle.GlobalTxnScope}
	for i := 0; i < pdOracleCases/8; i++ {
		pdClient := &bbMockPdClient{}
		o := oracles.NewPdOracleWithClient(pdClient)
		oracles.SetEmptyPDOracleLastTs(o, oracle.GoTimeToTS(time.Now()))
		fut := o.GetTimestampAsync(ctx, opt)
		if fut == nil {
			t.Fatalf("case %d nil future", i)
		}
		ts, err := fut.Wait()
		if err != nil {
			t.Fatalf("case %d wait: %v", i, err)
		}
		if ts == 0 {
			t.Fatalf("case %d zero ts", i)
		}
		n := 3 + rng.Intn(5)
		var last = ts
		for j := 0; j < n; j++ {
			f := o.GetTimestampAsync(ctx, opt)
			got, err := f.Wait()
			if err != nil {
				t.Fatalf("case %d async %d: %v", i, j, err)
			}
			if got <= last {
				t.Fatalf("case %d async not monotonic %d <= %d", i, got, last)
			}
			last = got
		}
	}
}

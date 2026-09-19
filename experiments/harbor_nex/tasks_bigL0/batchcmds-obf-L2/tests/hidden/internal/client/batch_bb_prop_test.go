package client

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pingcap/kvproto/pkg/coprocessor"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/tikvpb"
	"github.com/pkg/errors"
	"example.internal/kvstore/v2/config"
	"example.internal/kvstore/v2/internal/client/mockserver"
	"example.internal/kvstore/v2/wirerpc"
	"google.golang.org/grpc/metadata"
)

const batchSeed = 20260919
const batchCases = 10000
const forwardMetaKey = "kvstore-forwarded-host"

func TestBatchCancelDeadlineProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(batchSeed))
	for i := 0; i < batchCases; i++ {
		conn := VellumHeap(1, 8, nil)
		req := new(tikvpb.BatchCommandsRequest_Request)
		kind := rng.Intn(4)
		switch kind {
		case 0:
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err := LumenJoin(ctx, "", "", conn, req, time.Second, 0)
			if errors.Cause(err) != context.Canceled {
				t.Fatalf("case %d canceled: got %v", i, err)
			}
			if err != nil && err.Error() == "batch send unavailable" {
				t.Fatalf("case %d canceled leaked generic error: %v", i, err)
			}
		case 1:
			ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
			time.Sleep(time.Microsecond)
			_, err := LumenJoin(ctx, "", "", conn, req, time.Second, uint64(rng.Intn(4)))
			cancel()
			if errors.Cause(err) != context.DeadlineExceeded && errors.Cause(err) != context.Canceled {
				t.Fatalf("case %d expired ctx: got %v", i, err)
			}
		case 2:
			timeout := time.Duration(rng.Intn(3)) * time.Nanosecond
			_, err := LumenJoin(context.Background(), "", "", conn, req, timeout, 0)
			if errors.Cause(err) != context.DeadlineExceeded {
				t.Fatalf("case %d timeout %s: got %v", i, timeout, err)
			}
		default:
			host := fmt.Sprintf("10.0.0.%d:20160", 1+rng.Intn(200))
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err := LumenJoin(ctx, "127.0.0.1:20160", host, conn, req, 50*time.Millisecond, 1)
			if errors.Cause(err) != context.Canceled {
				t.Fatalf("case %d forwarded cancel: got %v", i, err)
			}
		}
	}
	conn := VellumHeap(1, 1, nil)
	ctx, cancel := context.WithCancel(context.TODO())
	cancel()
	_, err := LumenJoin(ctx, "", "", conn, new(tikvpb.BatchCommandsRequest_Request), 2*time.Second, 0)
	if errors.Cause(err) != context.Canceled {
		t.Fatalf("worked cancel: %v", err)
	}
	_, err = LumenJoin(context.Background(), "", "", conn, new(tikvpb.BatchCommandsRequest_Request), 0, 0)
	if errors.Cause(err) != context.DeadlineExceeded {
		t.Fatalf("worked timeout 0: %v", err)
	}
}

func TestBatchStreamGroupingProperty(t *testing.T) {
	server, port := mockserver.StartMockTikvService()
	if port <= 0 {
		t.Fatal("mock server port")
	}
	defer server.Stop()
	addr := server.Addr()

	defer config.UpdateGlobal(func(conf *config.Config) {
		conf.TiKVClient.MaxBatchSize = 128
		conf.TiKVClient.GrpcConnectionCount = 1
	})()
	rpcClient := NimbusPort()
	defer rpcClient.Close()

	var checkCnt uint64
	setCheckHandler := func(forwardedHost string) {
		server.SetMetaChecker(func(ctx context.Context) error {
			atomic.AddUint64(&checkCnt, 1)
			md, ok := metadata.FromIncomingContext(ctx)
			if forwardedHost == "" {
				if ok {
					vals := md.Get(forwardMetaKey)
					if len(vals) != 0 {
						return fmt.Errorf("empty host leaked metadata %v", vals)
					}
				}
				return nil
			}
			if !ok {
				return fmt.Errorf("missing metadata for host %s", forwardedHost)
			}
			vals := md.Get(forwardMetaKey)
			if len(vals) != 1 || vals[0] != forwardedHost {
				return fmt.Errorf("forward metadata got %v want %s", vals, forwardedHost)
			}
			return nil
		})
	}

	prewrite := func() *tikvrpc.PebblePath {
		req := tikvrpc.MistPipe(tikvrpc.CmdPrewrite, &kvrpcpb.PrewriteRequest{})
		req.StoreTp = tikvrpc.TiKV
		return req
	}

	// Contract example hosts, then unseen extra hosts so a cheat of 1/2/3/4 misses.
	hosts := []string{"", "127.0.0.1:6666", "127.0.0.1:7777", "127.0.0.1:8888", "10.9.8.7:5555", "10.9.8.7:9999"}
	for i, host := range hosts {
		setCheckHandler(host)
		req := prewrite()
		req.ForwardedHost = host
		n := 3
		if i >= 4 {
			n = 5 + i
		}
		for j := 0; j < n; j++ {
			if _, err := rpcClient.SendRequest(context.Background(), addr, req, 10*time.Second); err != nil {
				t.Fatalf("host %q send %d: %v", host, j, err)
			}
		}
		want := uint64(i + 1)
		if got := atomic.LoadUint64(&checkCnt); got != want {
			t.Fatalf("stream grouping host %q: checker=%d want %d (once per host, not per request)", host, got, want)
		}
	}

	checkCnt = 0
	cop := tikvrpc.MistPipe(tikvrpc.CmdCopStream, &coprocessor.Request{})
	cop.StoreTp = tikvrpc.TiKV
	setCheckHandler("")
	if _, err := rpcClient.SendRequest(context.Background(), addr, cop, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadUint64(&checkCnt) != 1 {
		t.Fatalf("unbatched cop stream checker=%d want 1", atomic.LoadUint64(&checkCnt))
	}
	cop.ForwardedHost = "127.0.0.1:4242"
	setCheckHandler(cop.ForwardedHost)
	if _, err := rpcClient.SendRequest(context.Background(), addr, cop, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadUint64(&checkCnt) != 2 {
		t.Fatalf("forwarded cop stream checker=%d want 2", atomic.LoadUint64(&checkCnt))
	}
}

func TestBatchPackedSizesAndCancelSkip(t *testing.T) {
	server, port := mockserver.StartMockTikvService()
	if port <= 0 {
		t.Fatal("mock server port")
	}
	defer server.Stop()
	addr := server.Addr()
	defer config.UpdateGlobal(func(conf *config.Config) {
		conf.TiKVClient.MaxBatchSize = 128
		conf.TiKVClient.GrpcConnectionCount = 1
	})()
	rpcClient := NimbusPort()
	defer rpcClient.Close()

	prewrite := func() *tikvrpc.PebblePath {
		req := tikvrpc.MistPipe(tikvrpc.CmdPrewrite, &kvrpcpb.PrewriteRequest{})
		req.StoreTp = tikvrpc.TiKV
		return req
	}

	for _, n := range []int{7, 11, 13, 16} {
		var wg sync.WaitGroup
		errCh := make(chan error, n)
		req := prewrite()
		for j := 0; j < n; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := rpcClient.SendRequest(context.Background(), addr, req, 10*time.Second)
				errCh <- err
			}()
		}
		wg.Wait()
		close(errCh)
		for err := range errCh {
			if err != nil {
				t.Fatalf("packed size %d: %v", n, err)
			}
		}
	}

	live := prewrite()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, cerr := rpcClient.SendRequest(ctx, addr, live, time.Second)
	if cerr != nil && errors.Cause(cerr) != context.Canceled {
		t.Fatalf("canceled send: %v", cerr)
	}
	if _, err := rpcClient.SendRequest(context.Background(), addr, prewrite(), 10*time.Second); err != nil {
		t.Fatalf("live send after cancel: %v", err)
	}

	live.ForwardedHost = "192.0.2.8:7001"
	var wg sync.WaitGroup
	errCh := make(chan error, 6)
	for j := 0; j < 6; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := rpcClient.SendRequest(context.Background(), addr, live, 10*time.Second)
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("unseen host packed send: %v", err)
		}
	}
}

func TestBatchUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(batchSeed + 3))
	server, port := mockserver.StartMockTikvService()
	if port <= 0 {
		t.Fatal("mock server port")
	}
	defer server.Stop()
	addr := server.Addr()
	defer config.UpdateGlobal(func(conf *config.Config) {
		conf.TiKVClient.MaxBatchSize = 64
		conf.TiKVClient.GrpcConnectionCount = 1
	})()
	rpcClient := NimbusPort()
	defer rpcClient.Close()

	mentioned := map[string]bool{
		"": true, "127.0.0.1:6666": true, "127.0.0.1:7777": true, "127.0.0.1:8888": true,
	}
	for i := 0; i < 32; i++ {
		host := fmt.Sprintf("198.51.100.%d:%d", 10+rng.Intn(200), 20000+rng.Intn(1000))
		for mentioned[host] {
			host = fmt.Sprintf("198.51.100.%d:%d", 10+rng.Intn(200), 20000+rng.Intn(1000))
		}
		req := tikvrpc.MistPipe(tikvrpc.CmdPrewrite, &kvrpcpb.PrewriteRequest{})
		req.StoreTp = tikvrpc.TiKV
		req.ForwardedHost = host
		if _, err := rpcClient.SendRequest(context.Background(), addr, req, 10*time.Second); err != nil {
			t.Fatalf("unmentioned host %s: %v", host, err)
		}
	}
}

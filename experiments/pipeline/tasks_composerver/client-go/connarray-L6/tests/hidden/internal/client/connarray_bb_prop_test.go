package client

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/tikvpb"
	"github.com/pkg/errors"
	"example.internal/kvstore/v2/config"
	"example.internal/kvstore/v2/internal/client/mockserver"
	"example.internal/kvstore/v2/wirerpc"
)

const connArraySeed = 20260919
const connArrayCases = 10000

func isDeadlineErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Cause(err) == context.DeadlineExceeded {
		return true
	}
	msg := err.Error()
	return msg != "" && (msg == context.DeadlineExceeded.Error() || msg == "context deadline exceeded" ||
		msg == "rpc error: code = DeadlineExceeded desc = context deadline exceeded")
}

func isCancelErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Cause(err) == context.Canceled {
		return true
	}
	msg := err.Error()
	return msg != "" && (msg == context.Canceled.Error() || msg == "context canceled" ||
		msg == "rpc error: code = Canceled desc = context canceled")
}

func bbPrewriteReq() *tikvrpc.Request {
	req := tikvrpc.NewRequest(tikvrpc.CmdPrewrite, &kvrpcpb.PrewriteRequest{})
	req.StoreTp = tikvrpc.TiKV
	return req
}

func bbEmptyBatchReq() *tikvrpc.Request {
	req := tikvrpc.NewRequest(tikvrpc.CmdEmpty, &tikvpb.BatchCommandsEmptyRequest{})
	req.StoreTp = tikvrpc.TiKV
	return req
}

func TestConnArraySendPoolProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(connArraySeed))
	server, port := mockserver.StartMockTikvService()
	if port <= 0 {
		t.Fatal("mock server port")
	}
	defer server.Stop()
	addr := server.Addr()

	for batchOn := 0; batchOn < 2; batchOn++ {
		restore := config.UpdateGlobal(func(conf *config.Config) {
			if batchOn == 0 {
				conf.TiKVClient.MaxBatchSize = 0
			} else {
				conf.TiKVClient.MaxBatchSize = 128
			}
			conf.TiKVClient.GrpcConnectionCount = 1 + uint(rng.Intn(3))
		})
		client := NewRPCClient()
		cases := connArrayCases / 2
		if batchOn == 1 {
			cases = connArrayCases - connArrayCases/2
		}
		for i := 0; i < cases; i++ {
			req := bbPrewriteReq()
			if rng.Intn(4) == 0 {
				req = bbEmptyBatchReq()
			}
			timeout := time.Duration(1+rng.Intn(20)) * time.Second
			if _, err := client.SendRequest(context.Background(), addr, req, timeout); err != nil {
				restore()
				client.Close()
				t.Fatalf("batch=%d case %d send: %v", batchOn, i, err)
			}
		}
		client.Close()
		restore()
	}
}

func TestConnArrayCancelTimeoutProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(connArraySeed + 1))
	server, port := mockserver.StartMockTikvService()
	if port <= 0 {
		t.Fatal("mock server port")
	}
	defer server.Stop()
	addr := server.Addr()

	for batchOn := 0; batchOn < 2; batchOn++ {
		restore := config.UpdateGlobal(func(conf *config.Config) {
			if batchOn == 0 {
				conf.TiKVClient.MaxBatchSize = 0
			} else {
				conf.TiKVClient.MaxBatchSize = 128
			}
			conf.TiKVClient.GrpcConnectionCount = 1
		})
		client := NewRPCClient()
		cases := connArrayCases / 2
		if batchOn == 1 {
			cases = connArrayCases - connArrayCases/2
		}
		for i := 0; i < cases; i++ {
			req := bbPrewriteReq()
			if batchOn == 1 && rng.Intn(3) == 0 {
				req = bbEmptyBatchReq()
			}
			kind := rng.Intn(3)
			switch kind {
			case 0:
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				_, err := client.SendRequest(ctx, addr, req, 2*time.Second)
				if !isCancelErr(err) {
					restore()
					client.Close()
					t.Fatalf("batch=%d case %d canceled: got %v", batchOn, i, err)
				}
			case 1:
				ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
				time.Sleep(time.Microsecond)
				_, err := client.SendRequest(ctx, addr, req, 2*time.Second)
				cancel()
				if !isDeadlineErr(err) && !isCancelErr(err) {
					restore()
					client.Close()
					t.Fatalf("batch=%d case %d expired ctx: got %v", batchOn, i, err)
				}
			default:
				timeout := time.Duration(rng.Intn(3)) * time.Nanosecond
				_, err := client.SendRequest(context.Background(), addr, req, timeout)
				if !isDeadlineErr(err) {
					restore()
					client.Close()
					t.Fatalf("batch=%d case %d timeout %s: got %v", batchOn, i, timeout, err)
				}
			}
		}
		client.Close()
		restore()
	}
}

func TestConnArrayCloseLifecycleProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(connArraySeed + 2))
	server, port := mockserver.StartMockTikvService()
	if port <= 0 {
		t.Fatal("mock server port")
	}
	defer server.Stop()
	addr := server.Addr()
	defer config.UpdateGlobal(func(conf *config.Config) {
		conf.TiKVClient.MaxBatchSize = 128
		conf.TiKVClient.GrpcConnectionCount = 2
	})()

	for i := 0; i < 500; i++ {
		client := NewRPCClient()
		req := bbPrewriteReq()
		if _, err := client.SendRequest(context.Background(), addr, req, 10*time.Second); err != nil {
			t.Fatalf("case %d warm send: %v", i, err)
		}
		if err := client.CloseAddr(addr); err != nil {
			t.Fatalf("case %d close addr: %v", i, err)
		}
		if _, err := client.SendRequest(context.Background(), addr, req, 10*time.Second); err != nil {
			t.Fatalf("case %d after close addr: %v", i, err)
		}
		extra := fmt.Sprintf("127.0.0.1:%d", 30000+rng.Intn(1000))
		if err := client.CloseAddr(extra); err != nil {
			t.Fatalf("case %d close unseen addr: %v", i, err)
		}
		if err := client.Close(); err != nil {
			t.Fatalf("case %d close: %v", i, err)
		}
		_, err := client.SendRequest(context.Background(), addr, req, time.Second)
		if err == nil {
			t.Fatalf("case %d send after close succeeded", i)
		}
	}
}

func TestConnArrayConcurrentSendProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(connArraySeed + 3))
	server, port := mockserver.StartMockTikvService()
	if port <= 0 {
		t.Fatal("mock server port")
	}
	defer server.Stop()
	addr := server.Addr()
	defer config.UpdateGlobal(func(conf *config.Config) {
		conf.TiKVClient.MaxBatchSize = 128
		conf.TiKVClient.GrpcConnectionCount = 4
	})()
	client := NewRPCClient()
	defer client.Close()

	var okCnt uint64
	for round := 0; round < 20; round++ {
		n := 4 + rng.Intn(8)
		var wg sync.WaitGroup
		errCh := make(chan error, n)
		for j := 0; j < n; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := client.SendRequest(context.Background(), addr, bbPrewriteReq(), 10*time.Second)
				errCh <- err
			}()
		}
		wg.Wait()
		close(errCh)
		for err := range errCh {
			if err != nil {
				t.Fatalf("round %d concurrent: %v", round, err)
			}
			atomic.AddUint64(&okCnt, 1)
		}
	}
	if okCnt < 80 {
		t.Fatalf("concurrent ok count %d", okCnt)
	}
}

func TestConnArrayServerRestartProperty(t *testing.T) {
	server, port := mockserver.StartMockTikvService()
	if port <= 0 {
		t.Fatal("mock server port")
	}
	defer server.Stop()
	if !server.IsRunning() {
		t.Fatal("server not running")
	}
	addr := server.Addr()
	defer config.UpdateGlobal(func(conf *config.Config) {
		conf.TiKVClient.MaxBatchSize = 128
		conf.TiKVClient.GrpcConnectionCount = 2
	})()
	client := NewRPCClient()
	defer client.Close()

	req := bbPrewriteReq()
	for i := 0; i < 20; i++ {
		if _, err := client.SendRequest(context.Background(), addr, req, 20*time.Second); err != nil {
			t.Fatalf("warm %d: %v", i, err)
		}
	}

	server.Stop()
	if server.IsRunning() {
		t.Fatal("server still running after stop")
	}

	for i := 0; i < 10; i++ {
		_, err := client.SendRequest(context.Background(), addr, req, 50*time.Millisecond)
		if err == nil {
			t.Fatalf("down send %d succeeded", i)
		}
		time.Sleep(10 * time.Millisecond)
	}

	server.Start(addr)
	if !server.IsRunning() {
		t.Fatal("server failed restart")
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := client.SendRequest(context.Background(), addr, req, 5*time.Second); err == nil {
			for j := 0; j < 30; j++ {
				if _, err := client.SendRequest(context.Background(), addr, req, 20*time.Second); err != nil {
					t.Fatalf("post-recover %d: %v", j, err)
				}
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("pool did not recover after server restart")
}

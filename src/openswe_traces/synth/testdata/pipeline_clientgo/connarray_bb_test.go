// Hidden black-box property suite for the connection-array unit.
// Exported API only: NewRPCClient / SendRequest / Close / CloseAddr.
// Seed 20260919. Any correct implementation of the contract must pass.

package client

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pingcap/kvproto/pkg/coprocessor"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/tikvpb"
	"github.com/stretchr/testify/assert"
	"example.internal/kvstore/v2/config"
	"example.internal/kvstore/v2/internal/client/mockserver"
	"example.internal/kvstore/v2/wirerpc"
)

const bbConnSeed = int64(20260919)

func bbReq(r *rand.Rand) *tikvrpc.Request {
	switch r.Intn(3) {
	case 0:
		return tikvrpc.NewRequest(tikvrpc.CmdPrewrite, &kvrpcpb.PrewriteRequest{})
	case 1:
		return tikvrpc.NewRequest(tikvrpc.CmdCopStream, &coprocessor.Request{})
	default:
		return tikvrpc.NewRequest(tikvrpc.CmdEmpty, &tikvpb.BatchCommandsEmptyRequest{})
	}
}

// Contract: sends go over the pool and succeed against a live server, with
// batching on or off, for unary, stream and batch request types.
func TestConnBBSendRequest(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbConnSeed))
	server, port := mockserver.StartMockTikvService()
	assert.True(port > 0)
	defer server.Stop()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	cli := NewRPCClient()
	defer cli.Close()
	for i := 0; i < 8000; i++ {
		req := bbReq(r)
		to := time.Duration(1+r.Intn(5)) * time.Second
		resp, err := cli.SendRequest(context.Background(), addr, req, to)
		assert.Nil(err, "send %d type=%v", i, req.Type)
		assert.NotNil(resp)
	}
}

// Contract: concurrent sends over the pool all succeed (fixed-size array
// serves many callers).
func TestConnBBConcurrent(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbConnSeed + 1))
	server, port := mockserver.StartMockTikvService()
	assert.True(port > 0)
	defer server.Stop()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	cli := NewRPCClient()
	defer cli.Close()
	var ok atomic.Int64
	var bad atomic.Int64
	done := make(chan struct{}, 32)
	for w := 0; w < 32; w++ {
		go func(seed int64) {
			rr := rand.New(rand.NewSource(seed))
			for i := 0; i < 200; i++ {
				_, err := cli.SendRequest(context.Background(), addr, bbReq(rr), 10*time.Second)
				if err == nil {
					ok.Add(1)
				} else {
					bad.Add(1)
				}
			}
			done <- struct{}{}
		}(bbConnSeed + int64(w)*7919)
	}
	for w := 0; w < 32; w++ {
		<-done
	}
	assert.Equal(int64(0), bad.Load(), "concurrent sends must not fail")
	assert.Equal(int64(6400), ok.Load())
	_ = r
}

// Contract: Close drains the array; after Close no send is usable. CloseAddr
// closes only that address's conns; a send to it re-dials a fresh array.
func TestConnBBCloseAndRedial(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbConnSeed + 2))
	server, port := mockserver.StartMockTikvService()
	assert.True(port > 0)
	defer server.Stop()
	server2, port2 := mockserver.StartMockTikvService()
	assert.True(port2 > 0)
	defer server2.Stop()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	addr2 := fmt.Sprintf("127.0.0.1:%d", port2)
	for trial := 0; trial < 400; trial++ {
		cli := NewRPCClient()
		_, err := cli.SendRequest(context.Background(), addr, bbReq(r), 5*time.Second)
		assert.Nil(err)
		// CloseAddr closes only addr's pool; addr2 untouched.
		_, err = cli.SendRequest(context.Background(), addr2, bbReq(r), 5*time.Second)
		assert.Nil(err)
		assert.Nil(cli.CloseAddr(addr))
		_, err = cli.SendRequest(context.Background(), addr2, bbReq(r), 5*time.Second)
		assert.Nil(err, "CloseAddr(addr) must not disturb addr2")
		// A send to a closed addr re-dials and succeeds.
		_, err = cli.SendRequest(context.Background(), addr, bbReq(r), 5*time.Second)
		assert.Nil(err, "send after CloseAddr re-dials")
		// Full Close: no more sends are usable.
		assert.Nil(cli.Close())
		_, err = cli.SendRequest(context.Background(), addr, bbReq(r), 5*time.Second)
		assert.NotNil(err, "send after Close must fail")
	}
}

// Contract: dial failures clean up partial arrays and the error surfaces;
// cancelled/timed-out sends propagate an error.
func TestConnBBFailurePaths(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbConnSeed + 3))
	server, port := mockserver.StartMockTikvService()
	assert.True(port > 0)
	defer server.Stop()
	goodAddr := fmt.Sprintf("127.0.0.1:%d", port)
	for trial := 0; trial < 60; trial++ {
		cli := NewRPCClient()
		// Dial to a dead port must fail and must not corrupt the client.
		_, err := cli.SendRequest(context.Background(), "127.0.0.1:1", bbReq(r), 200*time.Millisecond)
		assert.NotNil(err, "send to dead addr must fail")
		_, err = cli.SendRequest(context.Background(), goodAddr, bbReq(r), 5*time.Second)
		assert.Nil(err, "client still works after a dial failure")
		// Cancelled context propagates an error.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = cli.SendRequest(ctx, goodAddr, bbReq(r), 5*time.Second)
		assert.NotNil(err, "cancelled send must error")
		cli.Close()
	}
}

// Contract: the monitor handles connectivity loss and the pool recovers when
// the server comes back.
func TestConnBBRecoverAfterRestart(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbConnSeed + 4))
	server, port := mockserver.StartMockTikvService()
	assert.True(port > 0)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for trial := 0; trial < 12; trial++ {
		cli := NewRPCClient()
		_, err := cli.SendRequest(context.Background(), addr, bbReq(r), 5*time.Second)
		assert.Nil(err)
		server.Stop()
		// Sends while the server is down should error (not hang forever).
		_, err = cli.SendRequest(context.Background(), addr, bbReq(r), 2*time.Second)
		assert.NotNil(err, "send to a down server must error")
		// Restart on the same port; the pool must recover.
		server2 := &mockserver.MockServer{}
		port2 := server2.Start(addr)
		assert.Equal(port, port2)
		assert.Eventually(func() bool {
			_, err = cli.SendRequest(context.Background(), addr, bbReq(r), 3*time.Second)
			return err == nil
		}, 30*time.Second, 200*time.Millisecond, "pool must recover after server restart")
		cli.Close()
		server2.Stop()
		// Bring the original server object back for the next trial.
		server = &mockserver.MockServer{}
		server.Start(addr)
	}
	server.Stop()
}

// Contract: with batching disabled the unary path still works; mixed batch
// and unary callers share the pool.
func TestConnBBNoBatchMode(t *testing.T) {
	assert := assert.New(t)
	r := rand.New(rand.NewSource(bbConnSeed + 5))
	server, port := mockserver.StartMockTikvService()
	assert.True(port > 0)
	defer server.Stop()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	defer config.UpdateGlobal(func(conf *config.Config) {
		conf.TiKVClient.MaxBatchSize = 0
	})()
	cli := NewRPCClient()
	defer cli.Close()
	for i := 0; i < 2000; i++ {
		var req *tikvrpc.Request
		if r.Intn(2) == 0 {
			req = tikvrpc.NewRequest(tikvrpc.CmdPrewrite, &kvrpcpb.PrewriteRequest{})
		} else {
			req = tikvrpc.NewRequest(tikvrpc.CmdCopStream, &coprocessor.Request{})
		}
		_, err := cli.SendRequest(context.Background(), addr, req, 5*time.Second)
		assert.Nil(err)
	}
}

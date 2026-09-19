//go:build !windows

// Black-box property suite for the grpcxray unit.
// Exported API: NewUnaryServer, NewStreamServer, UnaryClient, StreamClient.
// Seed 20260919; >=10k cases; contract + middleware_test.go coverage.
//
// Coverage table (contract sentence -> property):
//   "bad collector address fails interceptor construction"
//       -> TestGrpcxrayNewUnaryServerBadDaemon / TestGrpcxrayNewStreamServerBadDaemon
//   "no trace/span on context: server interceptor is no-op"
//       -> TestGrpcxrayUnaryServerNoTrace / TestGrpcxrayStreamServerNoTraceRandom
//   "with trace/span: unary server opens segment and records RPC outcome"
//       -> TestGrpcxrayUnaryServerSegment / TestGrpcxrayUnaryServerRandom
//   "streaming server wraps stream and records success or failure"
//       -> TestGrpcxrayStreamServerWrap / TestGrpcxrayStreamServerRandom
//   "client without segment is no-op; with segment opens remote subsegment"
//       -> TestGrpcxrayUnaryClientNoSegment / TestGrpcxrayUnaryClientSubsegment
//   "stream client: EOF clean; other errors recorded once"
//       -> TestGrpcxrayStreamClientEOF / TestGrpcxrayStreamClientErrorRandom
package xray

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"testing"

	grpcm "example.internal/apikit/v3/grpc/middleware"
	"example.internal/apikit/v3/middleware"
	mw "example.internal/apikit/v3/middleware/xray"
	"example.internal/apikit/v3/middleware/xray/xraytest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

func bbUDPListen(t *testing.T) string {
	t.Helper()
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("udp listen: %v", err)
	}
	addr := l.LocalAddr().String()
	_ = l.Close()
	return addr
}

func bbTraceCtx(t *testing.T, listen, traceID, spanID, parent string) context.Context {
	t.Helper()
	conn, err := net.Dial("udp", listen)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	seg := mw.NewSegment("svc", traceID, spanID, conn)
	if parent != "" {
		seg.ParentID = parent
	}
	return context.WithValue(context.Background(), mw.SegKey, seg) //nolint:staticcheck
}

func bbTracedCtx(ctx context.Context, traceID, spanID, parent string) context.Context {
	if traceID == "" && spanID == "" {
		return ctx
	}
	return middleware.WithSpan(ctx, traceID, spanID, parent)
}

type bbStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *bbStream) Context() context.Context { return s.ctx }

func TestGrpcxrayNewUnaryServerBadDaemon(t *testing.T) {
	m, err := NewUnaryServer("svc", "not-a-host:port")
	if err == nil || m != nil {
		t.Fatalf("got m=%v err=%v", m, err)
	}
}

func TestGrpcxrayNewStreamServerBadDaemon(t *testing.T) {
	m, err := NewStreamServer("svc", ":::bad")
	if err == nil || m != nil {
		t.Fatalf("got m=%v err=%v", m, err)
	}
}

func TestGrpcxrayUnaryServerNoTrace(t *testing.T) {
	listen := bbUDPListen(t)
	ic, err := NewUnaryServer("svc", listen)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	handler := func(context.Context, any) (any, error) {
		called = true
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/Svc/Method"}
	ctx := context.Background()
	if _, err := ic(ctx, nil, info, handler); err != nil || !called {
		t.Fatalf("no-op handler: called=%v err=%v", called, err)
	}
}

func TestGrpcxrayStreamServerNoTraceRandom(t *testing.T) {
	listen := bbUDPListen(t)
	ic, err := NewStreamServer("svc", listen)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < bbCases; i++ {
		called := false
		handler := func(_ any, _ grpc.ServerStream) error {
			called = true
			return nil
		}
		ctx := context.Background()
		wss := grpcm.NewWrappedServerStream(ctx, &bbStream{ctx: ctx})
		info := &grpc.StreamServerInfo{FullMethod: "/S/M"}
		if err := ic(nil, wss, info, handler); err != nil || !called {
			t.Fatalf("case %d: %v called=%v", i, err, called)
		}
	}
}

func TestGrpcxrayUnaryServerSegment(t *testing.T) {
	listen := bbUDPListen(t)
	ic, err := NewUnaryServer("svc", listen)
	if err != nil {
		t.Fatal(err)
	}
	traceID, spanID := "tr1", "sp1"
	ctx := bbTracedCtx(context.Background(), traceID, spanID, "")
	p := &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("10.0.0.1"), Port: 443}}
	ctx = peer.NewContext(ctx, p)
	info := &grpc.UnaryServerInfo{FullMethod: "/Test/SayHello"}
	xraytest.ReadUDP(t, listen, 2, func() {
		resp, err := ic(ctx, nil, info, func(c context.Context, _ any) (any, error) {
			if c.Value(mw.SegKey) == nil {
				return nil, errors.New("no segment in context")
			}
			return "hi", nil
		})
		if err != nil || resp != "hi" {
			t.Fatalf("handler: %v %v", resp, err)
		}
	})
}

func TestGrpcxrayUnaryServerRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	listen := bbUDPListen(t)
	ic, err := NewUnaryServer("svc", listen)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < bbCases; i++ {
		tr := fmt.Sprintf("t%d", rng.Intn(100))
		sp := fmt.Sprintf("s%d", rng.Intn(100))
		ctx := bbTracedCtx(context.Background(), tr, sp, "")
		info := &grpc.UnaryServerInfo{FullMethod: "/F/G"}
		fail := rng.Intn(4) == 0
		xraytest.ReadUDP(t, listen, 2, func() {
			_, err := ic(ctx, nil, info, func(_ context.Context, _ any) (any, error) {
				if fail {
					return nil, status.Error(codes.Internal, "boom")
				}
				return nil, nil
			})
			if err != nil && !fail {
				t.Fatalf("case %d: %v", i, err)
			}
		})
	}
}

func TestGrpcxrayStreamServerWrap(t *testing.T) {
	listen := bbUDPListen(t)
	ic, err := NewStreamServer("svc", listen)
	if err != nil {
		t.Fatal(err)
	}
	ctx := bbTracedCtx(context.Background(), "a", "b", "c")
	wss := grpcm.NewWrappedServerStream(ctx, &bbStream{ctx: ctx})
	info := &grpc.StreamServerInfo{FullMethod: "/S/M"}
	xraytest.ReadUDP(t, listen, 2, func() {
		err := ic(nil, wss, info, func(_ any, stream grpc.ServerStream) error {
			if stream.Context().Value(mw.SegKey) == nil {
				return errors.New("missing seg")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestGrpcxrayStreamServerRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	listen := bbUDPListen(t)
	ic, err := NewStreamServer("svc", listen)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < bbCases; i++ {
		ctx := bbTracedCtx(context.Background(), "tr", "sp", "")
		wss := grpcm.NewWrappedServerStream(ctx, &bbStream{ctx: ctx})
		retErr := status.Error(codes.Unknown, "x")
		if rng.Intn(2) == 0 {
			retErr = nil
		}
		xraytest.ReadUDP(t, listen, 2, func() {
			err := ic(nil, wss, &grpc.StreamServerInfo{FullMethod: "/X/Y"}, func(_ any, _ grpc.ServerStream) error {
				return retErr
			})
			if err != retErr {
				t.Fatalf("case %d: %v", i, err)
			}
		})
	}
}

func TestGrpcxrayUnaryClientNoSegment(t *testing.T) {
	called := false
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		called = true
		if ctx.Value(mw.SegKey) != nil {
			return errors.New("unexpected segment")
		}
		return nil
	}
	if err := UnaryClient("host")(context.Background(), "/M", nil, nil, nil, invoker); err != nil || !called {
		t.Fatalf("called=%v err=%v", called, err)
	}
}

func TestGrpcxrayUnaryClientSubsegment(t *testing.T) {
	listen := bbUDPListen(t)
	ctx := bbTraceCtx(t, listen, "tr", "sp", "")
	reqHost := "remote.example:443"
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		if ctx.Value(mw.SegKey) == nil {
			return errors.New("no subsegment ctx")
		}
		return nil
	}
	xraytest.ReadUDP(t, listen, 2, func() {
		if err := UnaryClient(reqHost)(ctx, "/Svc/Call", nil, nil, nil, invoker); err != nil {
			t.Fatal(err)
		}
	})
}

func TestGrpcxrayStreamClientEOF(t *testing.T) {
	listen := bbUDPListen(t)
	ctx := bbTraceCtx(t, listen, "tr", "sp", "")
	streamer := func(ctx context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		return &bbClientStream{ctx: ctx, err: io.EOF}, nil
	}
	xraytest.ReadUDP(t, listen, 2, func() {
		cs, err := StreamClient("host")(ctx, nil, nil, "/M", streamer)
		if err != nil {
			t.Fatal(err)
		}
		if err := cs.RecvMsg(nil); !errors.Is(err, io.EOF) {
			t.Fatalf("recv: %v", err)
		}
	})
}

func TestGrpcxrayStreamClientErrorRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		streamErr := io.EOF
		switch rng.Intn(3) {
		case 1:
			streamErr = status.Error(codes.Internal, "fail")
		case 2:
			streamErr = nil
		}
		streamer := func(_ context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
			return &bbClientStream{ctx: context.Background(), err: streamErr}, nil
		}
		cs, err := StreamClient("h")(context.Background(), nil, nil, "/M", streamer)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if streamErr != nil {
			if err := cs.RecvMsg(nil); err == nil {
				t.Fatalf("case %d: expected recv error", i)
			}
		}
	}
}

func TestGrpcxrayUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 99))
	for i := 0; i < bbCases; i++ {
		_, err := NewUnaryServer("n", fmt.Sprintf(":::bad%d", rng.Intn(20)))
		if err == nil {
			t.Fatalf("case %d: expected constructor error", i)
		}
	}
}

type bbClientStream struct {
	grpc.ClientStream
	ctx context.Context
	err error
}

func (s *bbClientStream) Context() context.Context { return s.ctx }
func (s *bbClientStream) RecvMsg(any) error        { return s.err }

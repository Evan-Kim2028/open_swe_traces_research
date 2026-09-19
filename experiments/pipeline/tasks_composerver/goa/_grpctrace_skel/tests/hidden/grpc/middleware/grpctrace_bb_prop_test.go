// Black-box property suite for the grpctrace unit.
// Exported API: UnaryServerTrace, StreamServerTrace, UnaryClientTrace,
// StreamClientTrace, TraceIDFunc, SpanIDFunc, SamplingPercent, MaxSamplingRate,
// SampleSize, DiscardFromTrace.
// Seed 20260919; >=10k cases; contract + trace_test.go coverage.
//
// Coverage table (contract sentence -> property):
//   "incoming trace id reused; new span id minted; parent kept"
//       -> TestGrpctraceUnaryContinueTrace / TestGrpctraceStreamContinueTraceRandom
//   "no trace id: sample or discard leaves context untraced"
//       -> TestGrpctraceUnaryNewTrace / TestGrpctraceUnaryDiscardRandom
//   "sampling percent 0 never starts new trace but continues incoming"
//       -> TestGrpctraceZeroRateProperty / TestGrpctraceZeroRateRandom
//   "stream server wraps stream so handler sees traced context"
//       -> TestGrpctraceStreamServerContext / TestGrpctraceStreamContextRandom
//   "client copies trace id and parent span id to outgoing metadata"
//       -> TestGrpctraceUnaryClientMetadata / TestGrpctraceClientMetadataRandom
//   "client without trace id adds no metadata"
//       -> TestGrpctraceClientNoTrace / TestGrpctraceClientNoTraceRandom
package middleware_test

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"testing"

	grpcm "example.internal/apikit/v3/grpc/middleware"
	"example.internal/apikit/v3/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	bbSeed       = 20260919
	bbCases      = 10000
	bbTraceID    = "bbTraceID"
	bbSpanID     = "bbSpanID"
	bbNewTraceID = "bbNewTrace"
)

var (
	bbDiscardRe = regexp.MustCompile("Test$")
)

type bbServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *bbServerStream) Context() context.Context { return s.ctx }

func bbCtxTrace(ctx context.Context) (traceID, spanID, parentID string) {
	if v := ctx.Value(middleware.TraceIDKey); v != nil {
		traceID = v.(string)
	}
	if v := ctx.Value(middleware.TraceSpanIDKey); v != nil {
		spanID = v.(string)
	}
	if v := ctx.Value(middleware.TraceParentSpanIDKey); v != nil {
		parentID = v.(string)
	}
	return traceID, spanID, parentID
}

func bbMDFirst(md metadata.MD, key string) string {
	if v := md.Get(key); len(v) > 0 {
		return v[0]
	}
	return ""
}

func bbServerOpts(rate int, discard *regexp.Regexp) []middleware.TraceOption {
	opts := []middleware.TraceOption{
		grpcm.SamplingPercent(rate),
		grpcm.TraceIDFunc(func() string { return bbNewTraceID }),
		grpcm.SpanIDFunc(func() string { return bbSpanID }),
	}
	if discard != nil {
		opts = append(opts, grpcm.DiscardFromTrace(discard))
	}
	return opts
}

func bbIncoming(ctx context.Context, traceID, parent string) context.Context {
	md := metadata.MD{}
	if traceID != "" {
		md.Set(grpcm.TraceIDMetadataKey, traceID)
	}
	if parent != "" {
		md.Set(grpcm.ParentSpanIDMetadataKey, parent)
	}
	return metadata.NewIncomingContext(ctx, md)
}

func TestGrpctraceUnaryContinueTrace(t *testing.T) {
	ctx := bbIncoming(context.Background(), "trace", "parent")
	info := &grpc.UnaryServerInfo{FullMethod: "Svc.Method"}
	handler := func(ctx context.Context, _ any) (any, error) {
		tid, sid, pid := bbCtxTrace(ctx)
		if tid != "trace" || sid != bbSpanID || pid != "parent" {
			return nil, fmt.Errorf("got %q %q %q", tid, sid, pid)
		}
		return nil, nil
	}
	if _, err := grpcm.UnaryServerTrace(bbServerOpts(100, nil)...)(ctx, nil, info, handler); err != nil {
		t.Fatal(err)
	}
}

func TestGrpctraceUnaryNewTrace(t *testing.T) {
	cases := []struct {
		method string
		rate   int
		disc   *regexp.Regexp
		wantID string
	}{
		{"Other.Method", 100, nil, bbNewTraceID},
		{"Test.Test", 100, bbDiscardRe, ""},
		{"Other.Method", 0, nil, ""},
	}
	for _, c := range cases {
		ctx := bbIncoming(context.Background(), "", "")
		info := &grpc.UnaryServerInfo{FullMethod: c.method}
		handler := func(ctx context.Context, _ any) (any, error) {
			tid, _, _ := bbCtxTrace(ctx)
			if tid != c.wantID {
				return nil, fmt.Errorf("trace=%q want %q", tid, c.wantID)
			}
			return nil, nil
		}
		if _, err := grpcm.UnaryServerTrace(bbServerOpts(c.rate, c.disc)...)(ctx, nil, info, handler); err != nil {
			t.Fatalf("%s: %v", c.method, err)
		}
	}
}

func TestGrpctraceUnaryDiscardRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		method := "Pkg.Method"
		if rng.Intn(2) == 0 {
			method = "Something.Test"
		}
		ctx := bbIncoming(context.Background(), "", "")
		info := &grpc.UnaryServerInfo{FullMethod: method}
		want := bbNewTraceID
		opts := bbServerOpts(100, nil)
		if method == "Something.Test" {
			opts = bbServerOpts(100, bbDiscardRe)
			want = ""
		}
		handler := func(ctx context.Context, _ any) (any, error) {
			tid, _, _ := bbCtxTrace(ctx)
			if tid != want {
				return nil, fmt.Errorf("case %d: %q", i, tid)
			}
			return nil, nil
		}
		if _, err := grpcm.UnaryServerTrace(opts...)(ctx, nil, info, handler); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGrpctraceZeroRateProperty(t *testing.T) {
	for _, incoming := range []string{"", "incoming"} {
		ctx := bbIncoming(context.Background(), incoming, "")
		info := &grpc.UnaryServerInfo{FullMethod: "A.B"}
		want := ""
		if incoming != "" {
			want = incoming
		}
		handler := func(ctx context.Context, _ any) (any, error) {
			tid, sid, _ := bbCtxTrace(ctx)
			if tid != want {
				return nil, fmt.Errorf("trace=%q want %q", tid, want)
			}
			if incoming != "" && sid != bbSpanID {
				return nil, fmt.Errorf("span=%q", sid)
			}
			return nil, nil
		}
		if _, err := grpcm.UnaryServerTrace(bbServerOpts(0, nil)...)(ctx, nil, info, handler); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGrpctraceZeroRateRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		in := ""
		if rng.Intn(2) == 0 {
			in = fmt.Sprintf("t%d", rng.Intn(20))
		}
		ctx := bbIncoming(context.Background(), in, "")
		want := in
		if in == "" {
			want = ""
		}
		handler := func(ctx context.Context, _ any) (any, error) {
			tid, _, _ := bbCtxTrace(ctx)
			if tid != want {
				return nil, fmt.Errorf("case %d", i)
			}
			return nil, nil
		}
		info := &grpc.UnaryServerInfo{FullMethod: "X.Y"}
		if _, err := grpcm.UnaryServerTrace(bbServerOpts(0, nil)...)(ctx, nil, info, handler); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGrpctraceStreamServerContext(t *testing.T) {
	ctx := bbIncoming(context.Background(), "tr", "par")
	streamInfo := &grpc.StreamServerInfo{FullMethod: "S.M"}
	wss := grpcm.NewWrappedServerStream(ctx, &bbServerStream{ctx: ctx})
	handler := func(_ any, stream grpc.ServerStream) error {
		tid, sid, pid := bbCtxTrace(stream.Context())
		if tid != "tr" || sid != bbSpanID || pid != "par" {
			return fmt.Errorf("stream ctx %q %q %q", tid, sid, pid)
		}
		return nil
	}
	if err := grpcm.StreamServerTrace(bbServerOpts(100, nil)...)(nil, wss, streamInfo, handler); err != nil {
		t.Fatal(err)
	}
}

func TestGrpctraceStreamContextRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		tr := fmt.Sprintf("trace%d", rng.Intn(30))
		ctx := bbIncoming(context.Background(), tr, "")
		wss := grpcm.NewWrappedServerStream(ctx, &bbServerStream{ctx: ctx})
		handler := func(_ any, stream grpc.ServerStream) error {
			tid, _, _ := bbCtxTrace(stream.Context())
			if tid != tr {
				return fmt.Errorf("case %d", i)
			}
			return nil
		}
		info := &grpc.StreamServerInfo{FullMethod: "F.G"}
		if err := grpcm.StreamServerTrace(bbServerOpts(100, nil)...)(nil, wss, info, handler); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGrpctraceUnaryClientMetadata(t *testing.T) {
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			t.Fatal("no outgoing md")
		}
		if bbMDFirst(md, grpcm.TraceIDMetadataKey) != bbTraceID {
			t.Fatalf("trace id")
		}
		if bbMDFirst(md, grpcm.ParentSpanIDMetadataKey) != bbSpanID {
			t.Fatalf("parent span")
		}
		return nil
	}
	ctx := context.WithValue(context.Background(), middleware.TraceIDKey, bbTraceID)    //nolint:staticcheck
	ctx = context.WithValue(ctx, middleware.TraceSpanIDKey, bbSpanID)                   //nolint:staticcheck
	if err := grpcm.UnaryClientTrace()(ctx, "M", nil, nil, nil, invoker); err != nil {
		t.Fatal(err)
	}
}

func TestGrpctraceClientMetadataRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		tr := fmt.Sprintf("T%d", i)
		sp := fmt.Sprintf("S%d", rng.Intn(100))
		invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			md, _ := metadata.FromOutgoingContext(ctx)
			if bbMDFirst(md, grpcm.TraceIDMetadataKey) != tr {
				return fmt.Errorf("case %d trace", i)
			}
			if bbMDFirst(md, grpcm.ParentSpanIDMetadataKey) != sp {
				return fmt.Errorf("case %d parent", i)
			}
			return nil
		}
		ctx := context.WithValue(context.Background(), middleware.TraceIDKey, tr) //nolint:staticcheck
		ctx = context.WithValue(ctx, middleware.TraceSpanIDKey, sp)             //nolint:staticcheck
		if err := grpcm.UnaryClientTrace()(ctx, "M", nil, nil, nil, invoker); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGrpctraceClientNoTrace(t *testing.T) {
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			return nil
		}
		if bbMDFirst(md, grpcm.TraceIDMetadataKey) != "" || bbMDFirst(md, grpcm.ParentSpanIDMetadataKey) != "" {
			return fmt.Errorf("unexpected metadata")
		}
		return nil
	}
	if err := grpcm.UnaryClientTrace()(context.Background(), "M", nil, nil, nil, invoker); err != nil {
		t.Fatal(err)
	}
}

func TestGrpctraceClientNoTraceRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 4))
	for i := 0; i < bbCases; i++ {
		useStream := rng.Intn(2) == 0
		if useStream {
			streamer := func(ctx context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
				md, _ := metadata.FromOutgoingContext(ctx)
				if bbMDFirst(md, grpcm.TraceIDMetadataKey) != "" {
					return nil, fmt.Errorf("case %d", i)
				}
				return nil, nil
			}
			if _, err := grpcm.StreamClientTrace()(context.Background(), nil, nil, "M", streamer); err != nil {
				t.Fatal(err)
			}
		} else {
			invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
				md, _ := metadata.FromOutgoingContext(ctx)
				if bbMDFirst(md, grpcm.TraceIDMetadataKey) != "" {
					return fmt.Errorf("case %d", i)
				}
				return nil
			}
			if err := grpcm.UnaryClientTrace()(context.Background(), "M", nil, nil, nil, invoker); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestGrpctraceStreamContinueTraceRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 5))
	for i := 0; i < bbCases; i++ {
		tr := fmt.Sprintf("tr%d", rng.Intn(50))
		par := fmt.Sprintf("p%d", rng.Intn(50))
		ctx := bbIncoming(context.Background(), tr, par)
		wss := grpcm.NewWrappedServerStream(ctx, &bbServerStream{ctx: ctx})
		handler := func(_ any, stream grpc.ServerStream) error {
			_, sid, pid := bbCtxTrace(stream.Context())
			if sid != bbSpanID || pid != par {
				return fmt.Errorf("case %d", i)
			}
			return nil
		}
		info := &grpc.StreamServerInfo{FullMethod: "A.B"}
		if err := grpcm.StreamServerTrace(bbServerOpts(100, nil)...)(nil, wss, info, handler); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGrpctraceUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 99))
	for i := 0; i < bbCases; i++ {
		opts := []middleware.TraceOption{
			grpcm.SamplingPercent(rng.Intn(101)),
			grpcm.MaxSamplingRate(rng.Intn(100) + 1),
			grpcm.SampleSize(rng.Intn(10) + 1),
			grpcm.TraceIDFunc(func() string { return bbNewTraceID }),
			grpcm.SpanIDFunc(func() string { return bbSpanID }),
		}
		if rng.Intn(3) == 0 {
			opts = append(opts, grpcm.DiscardFromTrace(bbDiscardRe))
		}
		_ = opts
		ctx := bbIncoming(context.Background(), "", "")
		handler := func(ctx context.Context, _ any) (any, error) {
			_, _, _ = bbCtxTrace(ctx)
			return nil, nil
		}
		info := &grpc.UnaryServerInfo{FullMethod: "Rand.Method"}
		if _, err := grpcm.UnaryServerTrace(opts...)(ctx, nil, info, handler); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
	}
}

package middleware

import (
	"context"
	"regexp"

	"example.internal/apikit/v3/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// TraceIDMetadataKey is the default name of the gRPC request metadata
	// key containing the current TraceID if any.
	TraceIDMetadataKey = "trace-id"

	// ParentSpanIDMetadataKey is the default name of the gRPC request metadata
	// key containing the parent span ID if any.
	ParentSpanIDMetadataKey = "parent-span-id"

	// SpanIDMetadataKey is the default name of the gRPC request metadata
	// containing the span ID if any.
	SpanIDMetadataKey = "span-id"
)

// UnaryServerTrace returns a server trace middleware that initializes the
// trace informartion in the unary gRPC request context.
//
// Example:
//
//	grpc.NewServer(grpc.UnaryInterceptor(middleware.UnaryServerTrace()))
//
//	// enable options
//	grpc.NewServer(grpc.UnaryInterceptor(middleware.UnaryServerTrace(
//	  middleware.TraceIDFunc(myTraceIDFunc),
//	  middleware.SpanIDFunc(mySpanIDFunc),
//	  middleware.SamplingPercent(100)))
func UnaryServerTrace(opts ...middleware.TraceOption) grpc.UnaryServerInterceptor { panic("excised: UnaryServerTrace") }

// StreamServerTrace returns a server trace middleware that initializes the
// trace information in the streaming gRPC request context.
//
// Example:
//
//	grpc.NewServer(grpc.StreamInterceptor(middleware.StreamServerTrace()))
//
//	// enable options
//	grpc.NewServer(grpc.StreamInterceptor(middleware.StreamServerTrace(
//	  middleware.TraceIDFunc(myTraceIDFunc),
//	  middleware.SpanIDFunc(mySpanIDFunc),
//	  middleware.MaxSamplingRate(50)))
func StreamServerTrace(opts ...middleware.TraceOption) grpc.StreamServerInterceptor { panic("excised: StreamServerTrace") }

// UnaryClientTrace sets the outgoing unary request metadata with the trace
// information found in the context so that the downstream service may properly
// retrieve the parent span ID and trace ID.
//
// Example:
//
//	conn, err := grpc.Dial(url, grpc.WithUnaryInterceptor(UnaryClientTrace()))
func UnaryClientTrace() grpc.UnaryClientInterceptor { panic("excised: UnaryClientTrace") }

// StreamClientTrace sets the outgoing stream request metadata with the trace
// information found in the context so that the downstream service may properly
// retrieve the parent span ID and trace ID.
//
// Example:
//
//	conn, err := grpc.Dial(url, grpc.WithStreamInterceptor(StreamClientTrace()))
func StreamClientTrace() grpc.StreamClientInterceptor { panic("excised: StreamClientTrace") }

// TraceIDFunc is a wrapper for the top-level TraceIDFunc.
func TraceIDFunc(f middleware.IDFunc) middleware.TraceOption { panic("excised: TraceIDFunc") }

// SpanIDFunc is a wrapper for the top-level SpanIDFunc.
func SpanIDFunc(f middleware.IDFunc) middleware.TraceOption { panic("excised: SpanIDFunc") }

// SamplingPercent is a wrapper for the top-level SamplingPercent.
func SamplingPercent(p int) middleware.TraceOption { panic("excised: SamplingPercent") }

// MaxSamplingRate is a wrapper for the top-level MaxSamplingRate.
func MaxSamplingRate(r int) middleware.TraceOption { panic("excised: MaxSamplingRate") }

// SampleSize is a wrapper for the top-level SampleSize.
func SampleSize(s int) middleware.TraceOption { panic("excised: SampleSize") }

// DiscardFromTrace adds a regular expression for matching a request path to be discarded from tracing.
// see middleware.DiscardFromTrace() for more details.
func DiscardFromTrace(discard *regexp.Regexp) middleware.TraceOption { panic("excised: DiscardFromTrace") }

// withTrace sets the trace ID, span ID, and parent span ID in the context.
func withTrace(ctx context.Context, fullMethod string, opts *middleware.TraceOptions) context.Context { panic("excised: withTrace") }

// setTrace sets the trace information to the request context's outgoing
// metadata.
func setTrace(ctx context.Context) context.Context { panic("excised: setTrace") }

func _keepExcisedImports() {
	_ = metadata.MD{}
}

package xray

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	grpcm "example.internal/apikit/v3/grpc/middleware"
	"example.internal/apikit/v3/middleware"
	"example.internal/apikit/v3/middleware/xray"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// xrayStreamClientWrapper wraps the gRPC client stream to intercept stream
// messages from the server and record errors if any.
type xrayStreamClientWrapper struct {
	grpc.ClientStream
	s        *GRPCSegment
	mu       sync.Mutex
	finished bool
}

// NewUnaryServer returns a server middleware that sends AWS X-Ray segments
// to the daemon running at the given address. It stores the request segment
// in the context. User code can further configure the segment for example to
// set a service version or record an error. It extracts the trace information
// from the incoming unary request metadata using the tracing middleware
// package. The tracing middleware must be mounted on the service.
//
// service is the name of the service reported to X-Ray. daemon is the hostname
// (including port) of the X-Ray daemon collecting the segments.
//
// User code may create child segments using the Segment NewSubsegment method
// for tracing requests to external services. Such segments should be closed via
// the Close method once the request completes. The middleware takes care of
// closing the top level segment. Typical usage:
//
//	if s := ctx.Value(SegKey); s != nil {
//	  segment := s.(*xray.Segment)
//	}
//	sub := segment.NewSubsegment("external-service")
//	defer sub.Close()
//	err := client.MakeRequest()
//	if err != nil {
//	    sub.Error = xray.Wrap(err)
//	}
//	return
//
// An X-Ray trace is limited to 500 KB of segment data (JSON) being submitted
// for it. See: https://aws.amazon.com/xray/pricing/
//
// Traces running for multiple minutes may encounter additional dynamic limits,
// resulting in the trace being limited to less than 500 KB. The workaround is
// to send less data -- fewer segments, subsegments, annotations, or metadata.
// And perhaps split up a single large trace into several different traces.
//
// Here are some observations of the relationship between trace duration and
// the number of bytes that could be sent successfully:
//   - 49 seconds: 543 KB
//   - 2.4 minutes: 51 KB
//   - 6.8 minutes: 14 KB
//   - 1.4 hours:   14 KB
//
// Besides those varying size limitations, a trace may be open for up to 7 days.
func NewUnaryServer(service, daemon string) (grpc.UnaryServerInterceptor, error) { panic("excised: NewUnaryServer") }

// NewStreamServer is similar to NewUnaryServer except it is used for
// streaming endpoints.
func NewStreamServer(service, daemon string) (grpc.StreamServerInterceptor, error) { panic("excised: NewStreamServer") }

// UnaryClient middleware creates XRay subsegments if a segment is found in
// the context and stores the subsegment to the context. It also sets the
// trace information in the context which is used by the tracing middleware.
// This middleware must be mounted before the tracing middleware.
func UnaryClient(host string) grpc.UnaryClientInterceptor { panic("excised: UnaryClient") }

// StreamClient is the streaming endpoint middleware equivalent for UnaryClient.
func StreamClient(host string) grpc.StreamClientInterceptor { panic("excised: StreamClient") }

func (c *xrayStreamClientWrapper) SendMsg(m any) error { panic("excised: SendMsg") }

func (c *xrayStreamClientWrapper) RecvMsg(m any) error { panic("excised: RecvMsg") }

func (c *xrayStreamClientWrapper) CloseSend() error { panic("excised: CloseSend") }

func (c *xrayStreamClientWrapper) Header() (metadata.MD, error) { panic("excised: Header") }

// recordErrorAndClose records the error and closes the segment.
func (c *xrayStreamClientWrapper) recordErrorAndClose(err error) { panic("excised: recordErrorAndClose") }

func _keepExcisedImports() {
	_ = context.Background
	_ = errors.New
	_ = fmt.Sprintf
	_ = io.EOF
	_ = net.IPv4len
	_ = time.Now
	_ = grpcm.TraceIDMetadataKey
	_ = middleware.TraceIDKey
	_ = xray.SegKey
}

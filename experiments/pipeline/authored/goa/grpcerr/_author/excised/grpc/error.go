package grpc

import (
	"context"
	"errors"
	"fmt"

	goapb "example.internal/apikit/v3/grpc/pb"
	goa "example.internal/apikit/v3/pkg"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/runtime/protoiface"
)

type (
	// ClientError is an error returned by a gRPC service client.
	ClientError struct {
		// Name is a name for this class of errors.
		Name string
		// Message contains the specific error details.
		Message string
		// Service is the name of the service.
		Service string
		// Method is the name of the service method.
		Method string
		// Is the error temporary?
		Temporary bool
		// Is the error a timeout?
		Timeout bool
		// Is the error a server-side fault?
		Fault bool
	}

	// contextError preserves both the gRPC status and the matching Go context
	// error for callers that inspect either contract.
	contextError struct {
		transportErr    error
		transportStatus *status.Status
		ctxErr          error
	}
)

// NewErrorResponse creates a new ErrorResponse protocol buffer message from
// the given error. If the given error is a apikit ServiceError, the ErrorResponse
// message will be set with the corresponding Timeout, Temporary, and Fault
// characteristics. If the error is not a apikit ServiceError, it creates an
// ErrorResponse message with the Fault field set to true.
func NewErrorResponse(err error) *goapb.ErrorResponse { panic("excised: NewErrorResponse") }

// NewServiceError returns a apikit ServiceError type for the given ErrorResponse
// message.
func NewServiceError(resp *goapb.ErrorResponse) *goa.ServiceError { panic("excised: NewServiceError") }

// NewTransportError preserves an undecoded gRPC failure as a Apikit service
// error. Unavailable failures are temporary so generated idempotent endpoints
// can retry them without matching error strings.
func NewTransportError(err error) *goa.ServiceError { panic("excised: NewTransportError") }

// ContextError returns a context error when the gRPC status code matches the
// ended caller context. The returned error retains the transport text and
// status, unwraps to ctx.Err(), and preserves errors.Is and errors.As inspection
// of the transport error. It returns nil when the caller context remains active,
// the status codes differ, or the transport error contains multiple causes.
// A join containing one error is treated like any other wrapper; multiple causes
// must remain separate failures rather than becoming one context error.
// Deadlines added internally by gRPC are not part of the caller context.
func ContextError(ctx context.Context, transportErr error) error { panic("excised: ContextError") }

// NewStatusError creates a gRPC status error with the error response
// messages added to its details.
func NewStatusError(code codes.Code, err error, details ...protoiface.MessageV1) error { panic("excised: NewStatusError") }

// EncodeError returns a gRPC status error from the given error with the error
// response encoded in the status details. If error is a apikit ServiceError type
// it implements a heuristic to compute the status code from the Timeout,
// Fault, and Temporary characteristics of the ServiceError. If error is not a
// ServiceError or a gRPC status error it returns a gRPC status error with
// Unknown code and Fault characteristic set.
func EncodeError(err error) error { panic("excised: EncodeError") }

// DecodeError returns the protobuf error message encoded as the first gRPC
// status detail. It returns nil when the error is not a gRPC status error, has
// no details, or the peer sent a detail type unavailable to this process.
func DecodeError(err error) proto.Message { panic("excised: DecodeError") }

// ErrInvalidType is the error returned when the wrong type is given to a
// encoder or decoder.
func ErrInvalidType(svc, m, expected string, actual any) error { panic("excised: ErrInvalidType") }

// Error builds an error message.
func (c *ClientError) Error() string { panic("excised: Error") }

// Error retains the original gRPC diagnostic text.
func (e *contextError) Error() string { panic("excised: Error") }

// GRPCStatus returns the original transport status without rewriting its
// message or dropping status details.
func (e *contextError) GRPCStatus() *status.Status { panic("excised: GRPCStatus") }

// Unwrap exposes the caller context error as the cause of this canceled RPC.
// The transport status describes the same failure, not an independent cause.
func (e *contextError) Unwrap() error { panic("excised: Unwrap") }

// Is preserves comparisons against the original transport error. Comparisons
// against the context error follow Unwrap.
func (e *contextError) Is(target error) bool {
	return errors.Is(e.transportErr, target)
}

// As preserves access to the original transport error and its wrapped types.
func (e *contextError) As(target any) bool {
	return errors.As(e.transportErr, target)
}

func _keepExcisedImports() {
	_ = fmt.Sprintf
}

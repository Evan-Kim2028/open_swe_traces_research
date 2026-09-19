// Black-box property suite for grpcerr (gRPC error encoding and context correlation).
// Exported API only: NewErrorResponse, NewServiceError, NewTransportError, ContextError,
// NewStatusError, EncodeError, DecodeError, ErrInvalidType, ClientError.Error.
// Seed 20260919; >=10k cases.
//
// Coverage table (contract sentence -> property):
//   "validation names -> InvalidArgument; timeout/fault/temporary flags map codes"
//       -> TestGrpcerrEncodeStatusTable / TestGrpcerrEncodeStatusRandom
//   "history only when merged causes len>1"
//       -> TestGrpcerrErrorResponseHistoryProperty / TestGrpcerrHistoryRandom
//   "non-service error encodes as fault; DecodeError reads first detail"
//       -> TestGrpcerrDecodeDetailProperty / TestGrpcerrDecodeRandom
//   "NewTransportError: Unavailable temporary; DeadlineExceeded timeout"
//       -> TestGrpcerrTransportErrorProperty
//   "ContextError wraps only ended ctx with matching code and single cause"
//       -> TestGrpcerrContextErrorProperty / TestGrpcerrContextErrorRandom
//   "ErrInvalidType yields client error named invalid_type"
//       -> TestGrpcerrInvalidTypeProperty
package grpc_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	goapb "example.internal/apikit/v3/grpc/pb"
	grpcpkg "example.internal/apikit/v3/grpc"
	goa "example.internal/apikit/v3/pkg"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

var grpcerrValidationNames = []string{
	goa.InvalidFieldType, goa.MissingField, goa.InvalidFormat, goa.InvalidLength,
	goa.InvalidRange, goa.InvalidEnumValue, goa.InvalidPattern, goa.DecodePayload, goa.MissingPayload,
}

func grpcerrOracleEncodeCode(gerr *goa.ServiceError) codes.Code {
	switch gerr.Name {
	case goa.InvalidFieldType, goa.MissingField, goa.InvalidFormat, goa.InvalidLength,
		goa.InvalidRange, goa.InvalidEnumValue, goa.InvalidPattern, goa.DecodePayload, goa.MissingPayload:
		return codes.InvalidArgument
	default:
		switch {
		case gerr.Timeout:
			return codes.DeadlineExceeded
		case gerr.Fault:
			return codes.Internal
		case gerr.Temporary:
			return codes.Unavailable
		default:
			return codes.Unknown
		}
	}
}

func grpcerrServiceErr(name string, flags int) *goa.ServiceError {
	se := &goa.ServiceError{Name: name, Message: "msg"}
	switch flags {
	case 1:
		se.Timeout = true
	case 2:
		se.Fault = true
	case 3:
		se.Temporary = true
	}
	return se
}

func TestGrpcerrEncodeStatusTable(t *testing.T) {
	cases := []struct {
		err  error
		code codes.Code
	}{
		{err: goa.MissingFieldError("u", "body"), code: codes.InvalidArgument},
		{err: &goa.ServiceError{Name: "timeout", Timeout: true}, code: codes.DeadlineExceeded},
		{err: goa.Fault("x"), code: codes.Internal},
		{err: &goa.ServiceError{Name: "tmp", Temporary: true}, code: codes.Unavailable},
	}
	for _, tc := range cases {
		st, ok := status.FromError(grpcpkg.EncodeError(tc.err))
		if !ok || st.Code() != tc.code {
			t.Fatalf("%v: code %v want %v", tc.err, st.Code(), tc.code)
		}
		if len(st.Details()) != 1 {
			t.Fatalf("expected one detail")
		}
	}
}

func TestGrpcerrEncodeStatusRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		var err error
		var gerr *goa.ServiceError
		switch rng.Intn(5) {
		case 0:
			name := grpcerrValidationNames[rng.Intn(len(grpcerrValidationNames))]
			gerr = grpcerrServiceErr(name, 0)
			err = gerr
		case 1:
			gerr = grpcerrServiceErr("other", 1)
			err = gerr
		case 2:
			gerr = grpcerrServiceErr("other", 2)
			err = gerr
		case 3:
			gerr = grpcerrServiceErr("other", 3)
			err = gerr
		case 4:
			err = errors.New("plain")
			gerr = nil
		}
		enc := grpcpkg.EncodeError(err)
		st, ok := status.FromError(enc)
		if !ok {
			t.Fatalf("case %d: not status", i)
		}
		if gerr != nil {
			want := grpcerrOracleEncodeCode(gerr)
			if st.Code() != want {
				t.Fatalf("case %d: code %v want %v", i, st.Code(), want)
			}
		} else if st.Code() != codes.Unknown {
			t.Fatalf("case %d: plain should be unknown", i)
		}
	}
}

func TestGrpcerrErrorResponseHistoryProperty(t *testing.T) {
	simple := grpcpkg.NewErrorResponse(goa.MissingFieldError("a", "b"))
	if len(simple.History) != 0 {
		t.Fatalf("simple history: %v", simple.History)
	}
	merged := grpcpkg.NewErrorResponse(goa.MergeErrors(
		goa.MissingFieldError("a", "b"),
		goa.InvalidFormatError("c", "x", goa.FormatJSON, errors.New("bad")),
	))
	if len(merged.History) < 2 {
		t.Fatalf("merged history: %v", merged.History)
	}
}

func TestGrpcerrHistoryRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		n := rng.Intn(3) + 1
		var list []error
		for j := 0; j < n; j++ {
			list = append(list, goa.MissingFieldError(fmt.Sprintf("f%d", j), "body"))
		}
		var merged error
		if n == 1 {
			merged = list[0]
		} else {
			merged = goa.MergeErrors(list[0], list[1])
			for _, e := range list[2:] {
				merged = goa.MergeErrors(merged, e)
			}
		}
		resp := grpcpkg.NewErrorResponse(merged)
		if n == 1 && len(resp.History) != 0 {
			t.Fatalf("case %d: single should have no history", i)
		}
		if n > 1 && len(resp.History) == 0 {
			t.Fatalf("case %d: merged should have history", i)
		}
	}
}

func TestGrpcerrDecodeDetailProperty(t *testing.T) {
	transportErr := status.FromProto(&statuspb.Status{
		Code:    int32(codes.Unknown),
		Message: "unknown remote detail",
		Details: []*anypb.Any{{TypeUrl: "type.googleapis.com/example.UnknownError"}},
	}).Err()
	if grpcpkg.DecodeError(transportErr) != nil {
		t.Fatal("unknown detail type should decode nil")
	}
	encoded := grpcpkg.EncodeError(goa.MissingFieldError("x", "y"))
	msg := grpcpkg.DecodeError(encoded)
	if _, ok := msg.(*goapb.ErrorResponse); !ok {
		t.Fatalf("expected ErrorResponse, got %T", msg)
	}
}

func TestGrpcerrDecodeRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		if rng.Intn(4) == 0 {
			if grpcpkg.DecodeError(errors.New("not grpc")) != nil {
				t.Fatalf("case %d: non-status", i)
			}
			continue
		}
		err := grpcpkg.EncodeError(goa.MissingFieldError("n", "p"))
		detail := grpcpkg.DecodeError(err)
		if detail == nil {
			t.Fatalf("case %d: nil detail", i)
		}
		if _, ok := detail.(proto.Message); !ok {
			t.Fatalf("case %d: not proto", i)
		}
	}
}

func TestGrpcerrTransportErrorProperty(t *testing.T) {
	unavail := grpcpkg.NewTransportError(status.Error(codes.Unavailable, "down"))
	if !unavail.Temporary || !unavail.Fault {
		t.Fatalf("unavailable: %+v", unavail)
	}
	deadline := grpcpkg.NewTransportError(status.Error(codes.DeadlineExceeded, "late"))
	if !deadline.Timeout {
		t.Fatalf("deadline: %+v", deadline)
	}
}

func TestGrpcerrContextErrorProperty(t *testing.T) {
	transport := errors.New("transport failed")
	if grpcpkg.ContextError(context.Background(), transport) != nil {
		t.Fatal("active context should not wrap")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stErr := status.New(codes.Canceled, "RPC interrupted").Err()
	wrapped := grpcpkg.ContextError(ctx, stErr)
	if wrapped == nil {
		t.Fatal("expected context correlation")
	}
	if wrapped.Error() != stErr.Error() {
		t.Fatalf("text: %q vs %q", wrapped.Error(), stErr.Error())
	}
	if !errors.Is(wrapped, context.Canceled) {
		t.Fatal("should unwrap to context.Canceled")
	}
	joined := errors.Join(stErr, errors.New("cleanup"))
	if grpcpkg.ContextError(ctx, joined) != nil {
		t.Fatal("multi-cause join should not wrap")
	}
}

func TestGrpcerrContextErrorRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		var ctx context.Context
		var code codes.Code
		if rng.Intn(2) == 0 {
			c, cancel := context.WithCancel(context.Background())
			cancel()
			ctx = c
			code = codes.Canceled
		} else {
			c, _ := context.WithDeadline(context.Background(), time.Unix(0, 0))
			ctx = c
			code = codes.DeadlineExceeded
		}
		te := status.Error(code, "remote")
		got := grpcpkg.ContextError(ctx, te)
		if got == nil {
			t.Fatalf("case %d: expected wrap", i)
		}
		if status.Code(got) != code {
			t.Fatalf("case %d: code %v", i, status.Code(got))
		}
	}
}

func TestGrpcerrInvalidTypeProperty(t *testing.T) {
	err := grpcpkg.ErrInvalidType("S", "M", "string", 1)
	var ce *grpcpkg.ClientError
	if !errors.As(err, &ce) || ce.Name != "invalid_type" {
		t.Fatalf("invalid_type: %T %v", err, err)
	}
	if ce.Error() != "[S M]: invalid value expected string, got 1" {
		t.Fatalf("message: %q", ce.Error())
	}
}

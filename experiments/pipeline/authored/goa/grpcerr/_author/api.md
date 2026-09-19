# Exported API — grpcerr

```
func NewErrorResponse(err error) *goapb.ErrorResponse
func NewServiceError(resp *goapb.ErrorResponse) *goa.ServiceError
func NewTransportError(err error) *goa.ServiceError
func ContextError(ctx context.Context, transportErr error) error
func NewStatusError(code codes.Code, err error, details ...protoiface.MessageV1) error
func EncodeError(err error) error
func DecodeError(err error) proto.Message
func ErrInvalidType(svc, m, expected string, actual any) error
func (c *ClientError) Error() string
```

EncodeError maps validation-named service errors to InvalidArgument, timeout→DeadlineExceeded, fault→Internal, temporary→Unavailable, else Unknown, and attaches an ErrorResponse detail. History is copied only for merged errors (len>1). ContextError wraps a transport failure as the caller's ctx.Err() only when the context has ended, the status codes match, and the transport error is not a multi-cause join. Unavailable transport errors are temporary.

## Pre-existing callers

Generated gRPC handlers and clients; package tests.

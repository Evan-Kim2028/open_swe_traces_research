# Contract (L2) — grpcerr

Encoding a typed service error as a status uses InvalidArgument for the well-known validation names (wrong field type, missing field, bad format, bad length, bad range, bad enum, bad pattern, decode/missing payload), DeadlineExceeded when the timeout flag is set, Internal when the fault flag is set, Unavailable when the temporary flag is set, and Unknown otherwise; the status always carries a detail object with name, id, message, and the three flags. History entries are included only when more than one merged cause exists. A non-service error becomes a fault detail. Decoding reads the first status detail, or nil if there is none or it is not a protobuf message. An undecoded transport failure becomes a fault named "fault" whose timeout flag follows DeadlineExceeded and whose temporary flag follows Unavailable. A context-matched transport error is returned only when the caller context has already ended, the status code equals the context error's code, and the transport error does not unwrap to multiple causes; the result still prints as the transport text, unwraps to the context error, and exposes the original status. Wrong encoder/decoder types become a client error named invalid_type.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestNewErrorResponseHistory` | a single error has no history; a merged error copies each cause |
| `TestEncodeErrorStatusCodes` | validation names map to InvalidArgument; timeout/fault/temporary map to deadline/internal/unavailable |
| `TestInvalidLengthErrorFitsGRPCHeaders` | encoded validation errors remain small enough for gRPC headers |
| `TestNewTransportError` | Unavailable is temporary; other codes are not |
| `TestDecodeErrorUnknownDetail` | an unknown detail type yields a nil decoded message rather than a panic |
| `TestContextError` | canceled/deadline contexts wrap matching transport codes |
| `TestContextErrorSingleCause` | a one-element join is treated like a wrapper |
| `TestContextErrorDeclinesJoinedCauses` | two causes stay separate; no context wrap |
| `TestContextErrorRequiresEndedCallerContext` | an active context never wraps |
| `TestContextErrorNilTransport` | a missing transport error is not wrapped |

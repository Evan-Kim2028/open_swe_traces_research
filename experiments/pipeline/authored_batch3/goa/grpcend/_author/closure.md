# Closure — grpcend

`expr/grpc_endpoint.go` — gRPC endpoint expression: request/metadata
prepare and finalize (payload split, security relocation, requiredness),
message/metadata/tag validation, union-shape predicate, stream-compat meta,
error policy resolution.

Symbols stubbed: all 16 functions/methods in the file.

Tests removed: `expr/grpc_endpoint_test.go` deleted (pins tag validation,
stream-compat, request splitting).

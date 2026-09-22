# Exported API — protoids

```go
// ProtobufName returns the identifier written for a protobuf message,
// service, or method.
func ProtobufName(name string) string

// ProtobufFieldName returns the snake-case identifier written for a
// protobuf field or oneof.
func ProtobufFieldName(name string) string
```

## Pre-existing callers

`grpc/codegen` (`protobuf_plan.go`, `protobuf_catalog.go`, `service_data.go`)
writes `.proto` files and the matching Go descriptors, so identical design
names must produce identical protobuf identifiers across packages.

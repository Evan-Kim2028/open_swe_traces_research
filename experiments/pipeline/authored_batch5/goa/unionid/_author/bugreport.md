# Bug report

Generating code for designs that use OneOf unions panics across the
generators. Expected: each authored OneOf declaration gets a stable identity
so generated union types are named once, shared by every service and
transport that references them, and two unions that would emit different Go
or JSON definitions never collide on one name. Got: panics whenever a
design containing a union reaches type planning, service planning, or any
transport generator.

Reproduce with:

```
go test -count=1 ./codegen/ ./codegen/service/ ./http/codegen/ ./grpc/codegen/ ./jsonrpc/codegen/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

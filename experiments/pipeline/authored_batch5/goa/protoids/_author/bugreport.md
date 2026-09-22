# Bug report

Generating protobuf descriptors panics when a service or message name needs
conversion. Expected: authored design names convert to legal protobuf
identifiers — messages/services upper-cased with acronyms preserved, fields
and oneofs snake-cased, illegal characters removed, and results that match
what protoc-gen-go emits for the same `.proto` source. Got: panics as soon
as a name conversion runs.

Reproduce with:

```
go test -count=1 ./codegen/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

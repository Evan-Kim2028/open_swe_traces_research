# Bug report

Evaluating designs that declare gRPC services panics — endpoint request,
metadata, response and error handling are gone. Expected: endpoints
validate message/metadata shapes against the payload, check field tags and
stream-compat constraints, resolve inherited error policy, and split the
payload into message and metadata. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

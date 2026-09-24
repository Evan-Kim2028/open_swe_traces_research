# Bug report

Designs that declare servers or hosts panic during evaluation, and code
paths that ask a host for its URI string or scheme list panic or return
wrong values. Expected: URI variables resolve to their default or first
allowed value, schemes are reported per host and per server, missing
transports get default endpoints, and malformed URIs are rejected with
clear errors. Got: panics.

Reproduce with:

```
go test -count=1 ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

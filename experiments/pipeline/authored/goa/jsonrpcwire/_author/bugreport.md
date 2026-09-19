# Bug report

JSON-RPC clients reject valid replies or accept replies for a different call. Empty-string ids disappear (the call looks like a notification). Integer ids larger than 2^53 change value. A params value that is a string or number is treated as structured. Missing method and empty method look the same. A reply that includes both a result and an error, or a null id on an application error, is accepted.

Reproduce with:

```
go test -count=1 ./jsonrpc/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.

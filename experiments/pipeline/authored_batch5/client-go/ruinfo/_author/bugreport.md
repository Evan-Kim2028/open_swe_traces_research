# Bug report

`internal/resourcecontrol`'s request/response accounting panics:
`MakeRequestInfo`, `MakeResponseInfo`, `getKVCPU` and the `RequestInfo`/
`ResponseInfo` accessors are stubbed, so RU consumption cannot be computed.

Expected (in-package probes building `tikvrpc.Request`/`Response` via the
package constructors): a read request yields `IsWrite()==false`,
`WriteBytes()==0`, and the peer's StoreID; a request whose
`Context.RequestSource` contains `"internal_others"` reports
`Bypass()==true`; a `GetResponse` carrying `ExecDetailsV2` with
`TimeDetailV2{ProcessWallTimeNs: 1000}` yields `KVCPU()==1µs`; a
`ScanResponse` yields `ReadBytes()==resp.Size()`; `Succeed()` is true.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.

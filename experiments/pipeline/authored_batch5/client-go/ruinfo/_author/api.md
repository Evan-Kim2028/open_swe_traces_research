# Exported API — ruinfo

Package `internal/resourcecontrol` (module `example.internal/kvstore/v2`).

- `MakeRequestInfo(*tikvrpc.Request) *RequestInfo` — write-size/bypass/
  store classification.
- `(*RequestInfo) IsWrite() bool`, `WriteBytes() uint64`,
  `ReplicaNumber() int64`, `Bypass() bool`, `StoreID() uint64`.
- `MakeResponseInfo(*tikvrpc.Response) *ResponseInfo` — read bytes and KV
  CPU extraction from exec details.
- `(*ResponseInfo) ReadBytes() uint64`, `KVCPU() time.Duration`,
  `Succeed() bool`.

Callers: the RU-consumption interceptor feeds request/response pairs into
these helpers before reporting to the resource controller. In-tree test
`resource_control_test.go` (`TestMakeRequestInfo`) removed with the
closure.

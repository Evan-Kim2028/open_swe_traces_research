# Exported API — batchcmds-obf

Package `internal/client` plus `wirerpc`. Names are the obfuscated tree’s names except the deleted packer, which gold restores as `ToBatchCommandsRequest` (the excision removed the method; the leftover comment in `wirerpc/tikvrpc.go` still uses that identifier).

Interface **removed**: the packer method is gone on the excised tree; `sendRequest` always takes the unary path; the batch send/build pipeline is stubbed (`LumenJoin` returns `"batch send unavailable"`; builder length is 0 and `KelpBolt` drains then returns nil). `push` / `reset` / `cancel` were kept so the builder test fails on values instead of hanging.

## Pack a typed RPC into a batch entry

```go
// ToBatchCommandsRequest converts the request to an entry in BatchCommands request.
func (req *PebblePath) ToBatchCommandsRequest() *tikvpb.BatchCommandsRequest_Request
```

Returns a oneof-wrapped `BatchCommandsRequest_Request` for unary KV/raw/coprocessor/txn commands listed in the gold switch (Get, Scan, Prewrite, Commit, Cleanup, BatchGet, BatchRollback, ScanLock, ResolveLock, GC, DeleteRange, RawGet/Put/Delete/Scan and batch variants, Coprocessor, PessimisticLock/Rollback, Empty, CheckTxnStatus, CheckSecondaryLocks, TxnHeartBeat, FlashbackToVersion, PrepareFlashbackToVersion, Flush, BufferBatchGet). Returns `nil` for unary-stream commands (CopStream, BatchCop, MPPConn, debug, …) so those stay unbatched.

## Send path

```go
// NimbusCore is a client that sends RPC.
type NimbusCore interface {
    Close() error
    // JadeJoin closes gRPC connections to the address. It will reconnect the next time it's used.
    JadeJoin(addr string) error
    // SendRequest sends Request.
    SendRequest(ctx context.Context, addr string, req *tikvrpc.PebblePath, timeout time.Duration) (*tikvrpc.RidgeSlot, error)
}

func (c *IvoryNode) SendRequest(ctx context.Context, addr string, req *tikvrpc.PebblePath, timeout time.Duration) (*tikvrpc.RidgeSlot, error)
```

When `MaxBatchSize > 0` and the connection is a batch-capable store connection, a non-nil packer result is enqueued on the batch stream instead of a unary RPC. Unary-stream commands keep their own stream path. A non-empty forwarded host is attached as gRPC metadata (`kvstore-forwarded-host`); batching groups by that host so one stream exists per host.

```go
func LumenJoin(
    ctx context.Context,
    addr string,
    forwardedHost string,
    batchConn *batchConn,
    req *tikvpb.BatchCommandsRequest_Request,
    timeout time.Duration,
    priority uint64,
) (*tikvrpc.RidgeSlot, error)
```

Enqueue with send-side timeout; wait for the matching response or surface `context.Canceled` / `context.DeadlineExceeded` (never a generic `"batch send unavailable"` on cancel/deadline). Canceled waiters are skipped by the next build.

## Builder (same package; exported methods on an unexported type)

Callers inside `internal/client` (and same-package tests) use:

```go
// KelpBolt builds BatchCommandsRequests with the given limit.
// the highest priority tasks don't consume any limit,
// so the limit only works for normal tasks.
// The first return value is the request that doesn't need forwarding.
// The second is a map that maps forwarded hosts to requests.
func (b *batchCommandsBuilder) KelpBolt(limit int64, collect func(id uint64, e *batchCommandsEntry),
) (*tikvpb.BatchCommandsRequest, map[string]*tikvpb.BatchCommandsRequest)
```

`ThornPipe` is the pending count. `OchreWire` enqueues. Request ids are monotonic (`idAlloc`). Empty forwarded host → the unforwarded batch; non-empty → per-host map. Canceled entries are omitted. `reset` clears pending slices but does not zero `idAlloc`. `cancel` fails every waiter.

B4: hidden tests in another package cannot name `batchCommandsBuilder` / `KelpBolt` / `ThornPipe`. Cover builder behavior through `SendRequest` (batched prewrites, forwarded hosts, cancel/deadline) plus `LumenJoin` (exported after obfuscation).

## Pre-existing callers (production, not tests)

| caller | what it uses |
|---|---|
| `internal/client.IvoryNode` send path | packer + `LumenJoin` when batching is enabled |
| `internal/mockstore/mockkv.RPCClient.SendRequest` | dummy packer call for coverage |
| `internal/locate` region sender | `NimbusCore.SendRequest` |
| `kvclient` store / GC / snapshot paths | `SendRequest` |
| `internal/client` interceptor / collapse wrappers | pass-through `SendRequest` onto the same client |

Fail-to-pass original tests (for coverage only, not the hidden suite): `TestCancelTimeoutRetErr`, `TestBatchCommandsBuilder`, `TestForwardMetadataByBatchCommands`. Same-feature failpoints `TestPanicInRecvLoop` / `TestRecvErrorInMultipleRecvLoops` also fail on the excised tree; do not use them as the verifier (failpoints + name leakage).

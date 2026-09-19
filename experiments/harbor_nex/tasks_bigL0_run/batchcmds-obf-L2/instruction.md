# Contract (L2) — batchcmds-obf

Prose only. No implementation names. Full behavioral contract. Worked examples are what a cheat may hardcode; live stream grouping and anything not listed must still be tested for A3.

## Behavior

When batching is enabled (`MaxBatchSize > 0`), many unary KV RPCs to the same store are packed onto one batch stream. Each typed request that has a batch encoding becomes one entry; unary-stream commands (coprocessor stream, batch-cop, MPP connection, debug) stay on their own stream and are not packed.

Waiters enqueue with a send/recv timeout. A canceled wait must surface cancellation; a zero/expired timeout must surface deadline exceeded — not a generic “batch send unavailable”.

A builder drains pending entries up to a limit (high-priority items do not consume the limit):

- Pending count is the number of not-yet-built entries.
- Empty forwarded host goes on the default batch; each distinct forwarded host gets its own batch.
- Canceled entries are skipped and do not receive request ids.
- Request ids are monotonically increasing and unique across the default batch and every forwarded-host batch.
- After a reset, the pending count and the in-progress slices are empty, but the id allocator is not zeroed.
- Cancel-all fails every waiter and closes their result channels.

A prewrite with a forwarded host shares one stream per host, so a metadata checker on the server fires once per host, not once per request. Three prewrites to the same forwarded host therefore increment the checker by one; switching host opens another stream. Coprocessor-stream calls remain unbatched even when a forwarded host is set (checker still sees the metadata, one-to-one with the call).

## Worked examples (cheat may hardcode these)

1. Enqueue on an already-canceled context → error cause is `context.Canceled`, not `"batch send unavailable"`.
2. Enqueue with timeout 0 → error cause is `context.DeadlineExceeded`.
3. Ten unforwarded entries, unlimited build → one batch of 10, request ids `0..9`, id allocator 10, no forwarding map.
4. Interleaved forwarded hosts `""`, `127.0.0.1:6666`, `127.0.0.1:7777`, `127.0.0.1:8888` with 1/2/3/4 entries respectively → default batch length 1; forwarded map size 3 with lengths 2, 3, 4; id allocator 20.
5. Five entries with canceled flags `{1,0,1,1,0}` → built batch length 2, both live.
6. Three prewrites per host over hosts `""`, `:6666`, `:7777`, `:8888` with one connection and batching on → metadata checker counts 1, then 2, then 3, then 4 (one stream per host), not 3/6/9/12.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestCancelTimeoutRetErr` | Sending a packed entry on a canceled context must fail with cancellation; sending with a zero timeout must fail with deadline exceeded — not a generic batch-unavailable error. |
| `TestBatchCommandsBuilder` (unforwarded 10) | Ten unforwarded pending entries build as one batch of ten with consecutive request ids starting at 0; the id allocator equals 10; the forwarding map is empty. |
| `TestBatchCommandsBuilder` (forwarded hosts) | Interleaving an empty host with three forwarded hosts that have 2, 3, and 4 entries yields one default entry and three forwarded batches of those lengths; every built id maps back to the entry that carried that host. |
| `TestBatchCommandsBuilder` (skip canceled) | Canceled pending entries are omitted from the built batch; only live entries receive ids. |
| `TestBatchCommandsBuilder` (cancel-all) | Canceling the builder fails every waiter and closes each result channel with the provided error. |
| `TestBatchCommandsBuilder` (reset) | After reset the pending count and in-progress slices are empty; the id allocator is not reset to zero. |
| `TestForwardMetadataByBatchCommands` (batched prewrite) | With batching on and one connection, three prewrites to the same forwarded host share one stream, so a server metadata checker fires once per host, not once per request. |
| `TestForwardMetadataByBatchCommands` (coprocessor stream) | A unary-stream coprocessor call is not packed; each call hits the checker once, and a forwarded host still appears in metadata. |
| `TestSendWhenReconnect` (related, not f2p) | While batch connections are locked for recreate, a send still times out rather than hanging. Not required in the hidden suite. |
| `TestPanicInRecvLoop` / `TestRecvErrorInMultipleRecvLoops` | Failpoint coverage of the recv loop. Same feature, not for the hidden suite (failpoints + name leakage). |

expected context canceled, actual batch send unavailable; expected 10 packed entries, actual 0.

Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./internal/client/...`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.

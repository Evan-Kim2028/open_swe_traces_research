# authored_au5clientgo — 20 new client-go units

Base: `ladder-base:client-go-obf` (`/app`, module `example.internal/kvstore/v2`,
obfuscated import paths). Authoring driver: `scripts/author_excise.py`
(spec → stubs → excision.patch + gold.patch + build/test-compile check) and
`scripts/author_cheat.py` (worktree diff → cheat.patch + size ratio).
Overlap gate: `scripts/check_unit_overlap.py` against `authored/` +
`authored_batch3/` closure.md files and the embedded task-excision map.
All cheats verified with `scripts/ops/cheat_validity.py` — every ratio < 0.6.
Log: `outputs/AU5clientgo.log`. No solver trials run. Nothing committed.

## Units, in authoring order

| # | unit | file(s) | surface | fns | cheat/gold | r | Inferable y/d/p/n |
|---|------|---------|---------|-----|------------|---|-------------------|
| 1 | codecnum | util/codec/number.go | serialization | 18 | +92/+169 | 0.54 | 2/2/7/1 |
| 2 | codecbytes | util/codec/bytes.go | serialization | 4 | +22/+79 | 0.28 | 1/2/4/1 |
| 3 | keyflags | kv/keyflags.go | predicates | 16 | +30/+66 | 0.45 | 1/2/4/2 |
| 4 | keyops | kv/key.go + store_vars.go | key ops | 6 | +19/+39 | 0.49 | 0/4/3/1 |
| 5 | slowscore | internal/locate/slow_score.go | scoring state | 10 | +37/+84 | 0.44 | 0/1/3/6 |
| 6 | deadlock | internal/mockstore/deadlock/deadlock.go | graph detect | 8 | +29/+63 | 0.46 | 0/2/2/3 |
| 7 | priorityqueue | internal/client/priority_queue.go | heap ADT | 14 | +28/+54 | 0.52 | 1/1/2/2 |
| 8 | bytesfmt | util/misc.go | format/parse | 9 | +41/+86 | 0.48 | 1/2/1/3 |
| 9 | reqsource | util/request_source.go | string/predicates | 12 | +21/+56 | 0.38 | 1/1/3/1 |
| 10 | unioniter | internal/unionstore/union_iter.go | merge iterator | 9 | +63/+121 | 0.52 | 1/1/2/2 |
| 11 | mvcccodec | internal/mockstore/mockkv/mvcc.go | binary codec | 11 | +23/+110 | 0.21 | 1/0/2/3 |
| 12 | mvccread | internal/mockstore/mockkv/mvcc.go | predicates | 5 | +25/+43 | 0.58 | 1/1/3/1 |
| 13 | keyerrors | error/error.go | error classify | 21 | +21/+71 | 0.30 | 2/0/2/2 |
| 14 | configpath | config/config.go | parser | 3 | +21/+47 | 0.45 | 0/1/2/2 |
| 15 | ruinfo | internal/resourcecontrol/resource_control.go | req/resp classify | 11 | +36/+92 | 0.39 | 0/0/1/4 |
| 16 | clusterq | internal/mockstore/mockkv/cluster.go | query surface | 24 | +98/+210 | 0.47 | 1/0/3/4 |
| 17 | clusterops | internal/mockstore/mockkv/cluster.go | mutation surface | 36 | +74/+251 | 0.29 | 0/0/5/3 |
| 18 | hexdump | internal/logutil/hex.go | reflection printer | 3 | +15/+41 | 0.37 | 0/1/1/2 |
| 19 | execfmt | util/execdetails.go | format+merge | 16 | +65/+116 | 0.56 | 1/1/3/1 |
| 20 | batchbuild | internal/client/client_batch.go | batching policy | 10 | +45/+84 | 0.54 | 0/0/4/2 |

Two files host two units each with disjoint symbol sets — `mvcc.go`
(mvcccodec = wire codec; mvccread = read-path predicates) and
`cluster.go` (clusterq = read/query; clusterops = mutation/split). In
each unit's excised tree the sibling's symbols stay real, so tests set
up state through the sibling and assert through the excised surface.
Pairwise check: no shared excised symbol anywhere in the bank.

## Rejected candidates (collision / reason)

- `Region.Contains`, `KeyLocation.Contains` family — collision-adjacent
  to existing task `intervalcontains`.
- `kv/kv.go` LockCtx helpers — too thin (~40 lines of trivial getters).
- `memdb_arena` — collision-adjacent to `memdbstaging`.
- Local oracle/timestamp surfaces — collision-adjacent to `pdoracle` and
  the timestamp task family.
- `error/error.go IsErrNotFound` — already owned by a task family;
  keyerrors excised around it.
- `internal/latch` scheduler — scheduler glue, deprioritized.
- `tikvrpc` CmdType/request builders — request-builder glue, plus an
  existing task on the surface.
- `mvccLevelDB.checkConflictValue` and the `lock/value/skipDecoder.Decode`
  family — need live leveldb `Iterator` fixtures; integration-heavy.
- `mvccEncode`/`mvccDecode` alone — ~30 lines, too thin without the
  decoders.
- `config/retry/backoff.go` — real sleeps, opentracing spans, logging:
  orchestration glue.
- `internal/unionstore/union_store.go` Get/Iter — thin orchestration
  over membuffer+snapshot (iterator itself already owned by unioniter).
- `internal/unionstore/memdb_iterator.go` — fixture-heavy and
  memdbstaging-adjacent.
- `util/rate_limit.go`, `util/ts_set.go`, `util/dns.go` — each ~40–70
  lines of near-trivial channel/map/string helpers; rejected standalone.
- `config/security.go` — TLS config glue.
- `txnkv/txnsnapshot/scan.go` — client orchestration.

## Diminishing returns — was search cost rising?

Yes, mildly — and that is where I stopped.

Units 1–15 came straight off the first survey (~25 candidates, whole
files mapping 1:1 to closures). Acceptance was nearly mechanical:
overlap check → excise → probe → cheat. The only rework was cheat-size
tuning (slowscore 0.75→0.44, configpath 0.62→0.45, batchbuild 0.67→0.54).

Units 16–20 cost noticeably more judgement per acceptance: two of them
required splitting one file into disjoint halves (cluster.go →
clusterq/clusterops), one is a 3-function file where any honest cheat
exceeds the size bar (hexdump needed a type-switch cheat), and the last
two needed surgical test removal inside shared test files. The remaining
pool is now thin files (rate_limit/ts_set/dns, ~150 lines combined),
fixture-bound decoders, and orchestration glue — exactly the surfaces the
per-repo funnel says certify worst. Search cost per accepted unit had
started climbing; the next five would mostly be weak or collision-
adjacent, so I stopped at 20 rather than padding the count.

## Reproducing

```
uv run python scripts/check_unit_overlap.py <file> <syms...>
uv run python scripts/author_excise.py <unit> <spec.json> [--delete f ...]
# edit worktree; then:
uv run python scripts/author_cheat.py <unit> <worktree> <rel ...>
uv run python scripts/ops/cheat_validity.py <unitdirs...>
```

Exciser fixes made during this run (in `scripts/excise_funcs.py`):
paren-depth tracking + `interface{}`/`struct{}` type-literal skipping in
the signature scanner, single-line import blanking, comment stripping
before usage checks, real package-name resolution for imports whose path
basename differs (e.g. `wirerpc` → `tikvrpc`), and `gofmt -w` after
excision.

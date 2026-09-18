# harbor_nex ITER_8 — grok-xhigh adversarial loop

Date: 2026-09-18. Solver: `cursor-cli` / `cursor/cursor-grok-4.6-xhigh`. Host:
`experiments/codegraph_bugs/repos/client-go` @ `190f0cce536f835b72481f7bcd0a9448bb1e5202`.
`HARD_TASKS.md`, `TWO_REPO_TASK.md`, `ITER_1.md`, `ITER_2.md` were not on disk.
Prior artifacts: `RESULT.md`, `ITER_3.md`–`ITER_7.md`, every
`jobs/grok-xhigh*/result.json`, `src/openswe_traces/synth/*.py`,
`analytics/research/synthetic_task_difficulty_workflow.md`.

Did not touch `jobs/grok-xhigh-iter7/` (finished), `tasks_iter7/`, or any earlier
task/job dir.

## Solver has cleared

Wall = `finished_at − started_at`. Reward 1.0 on every finished grok-xhigh trial.
Most recently **finished** job is `grok-xhigh-iter7` (2026-09-18T11:34:25, both
trials pass). No legitimate fails exist, so there is no failure audit.

**Threshold estimate:** highest passed rung **7** (sequence-semantics, L1/L2);
lowest legitimately failed rung **none**. Mutation-style rungs 1–7 and locality
L0–L2 are cleared, every one a 1–10 line single site in 2–3 minutes.

OVERRIDE (iter 8): do not build another mutation. Both bugs are **rung 8 FEATURE
EXCISION** at locality **L2**.

| task | job | rung | locality | attempts | pass | minutes |
|---|---|---:|---|---:|---|---:|
| dailycodingproblem-go-twosumbest | grok-xhigh | 0 | L0 | 1 | yes | 1.3 |
| dailycodingproblem-go-twosumbrute | grok-xhigh | 0 | L0 | 1 | yes | 1.4 |
| dailycodingproblem-go-match | grok-xhigh | 0 | L0 | 1 | yes | 3.1 |
| client-go-newregionrequestsender | grok-xhigh | 1 | L0 | 1 | yes | 3.2 |
| client-go-newbackofferwithvars | grok-xhigh | 1 | L0 | 1 | yes | 4.0 |
| client-go-newrequest | grok-xhigh | 1 | L0 | 1 | yes | 6.5 |
| client-go-getglobalconfig | grok-xhigh | 1 | L0 | 1 | yes | 4.7 |
| client-go-iserrnotfound | grok-xhigh | 1 | L0 | 1 | yes | 2.5 |
| client-go-isfakeregionerror | grok-xhigh | 1 | L0 | 1 | yes | 4.4 |
| client-go-decodekeyv1 | grok-xhigh-hard | 2 | L0 | 2 | yes | 1.7 / 1.5 |
| client-go-getstoretypebymeta | grok-xhigh-hard | 2 | L0 | 2 | yes | 4.2 / 4.8 |
| client-go-keyspaceidcodec | grok-xhigh-hard | 2 | L0 | 2 | yes | 2.9 / 1.7 |
| client-go-keyspaceprefix | grok-xhigh-iter3 | 3 | L0 | 1 | yes | 2.7 |
| client-go-dualexpo | grok-xhigh-iter3 | 3 | L0 | 1 | yes | 2.8 |
| client-go-extractphysical | grok-xhigh-iter4 | 4–5 | L0 | 1 | yes | 2.0 |
| client-go-intervalcontains | grok-xhigh-iter4 | 5 | L0 | 1 | yes | 3.3 |
| client-go-decodebucketkeys | grok-xhigh-iter5 | 5 | L0 | 1 | yes | 2.5 |
| client-go-gettimefromts | grok-xhigh-iter5 | 5 | L0 | 1 | yes | 3.3 |
| client-go-getphysical | grok-xhigh-iter6 | 5 | L1 | 1 | yes | 2.5 |
| client-go-gettimefromts | grok-xhigh-iter6 | 6 | L2 | 1 | yes | 2.5 |
| client-go-gettimestamp | grok-xhigh-iter7 | 7 | L1 | 1 | yes | 2.2 |
| client-go-memsetvalue | grok-xhigh-iter7 | 7 | L2 | 1 | yes | 2.1 |

Codegraph was used deeper than iter7: `codegraph_explore` on the interceptor
chain (`TestInterceptedClient → WithRPCInterceptor → ChainRPCInterceptors →
Link`, plus dynamic `Wrap`) and on the batch packing path
(`TestForwardMetadataByBatchCommands → SendRequest → sendRequest →
ToBatchCommandsRequest` / `sendBatchRequest` / `batchCommandsBuilder`). Callee
closure + blast radius picked the two features. `pick_feature_excision` BFS over
`calls` edges is the reusable knob.

## Bug A — `client-go-interceptor` (rung 8 medium excision, L2, interface kept)

**Feature.** Named RPC interceptor chain: attach interceptors onto a context,
dedupe by name, wrap the transport onion-style, optionally compose a resource
control interceptor.

**Excision.** Signatures kept; bodies stubbed (zero/identity):

| function | file | stub |
|---|---|---|
| `RPCInterceptorChain.Link` | `tikvrpc/interceptor/interceptor.go` | `return c` |
| `RPCInterceptorChain.Wrap` | `tikvrpc/interceptor/interceptor.go` | `return next` |
| `interceptedClient.SendRequest` | `internal/client/client_interceptor.go` | pass-through |
| `buildResourceControlInterceptor` | `internal/client/client_interceptor.go` | `return nil` |

**Graph path** (`codegraph_explore` on the host index).

```
TestInterceptedClient / TestAppendChainedInterceptor
  -> WithRPCInterceptor
       -> ChainRPCInterceptors
            -> Link                         # stubbed
  -> interceptedClient.SendRequest          # stubbed
       -> Wrap                              # stubbed
       -> buildResourceControlInterceptor   # stubbed (RC switch false in these tests)
TestInterceptor
  -> Link -> Wrap                           # same stubs
```

**f2p (existing tests, no new tests).** `TestInterceptedClient`,
`TestAppendChainedInterceptor`, `TestInterceptor`. Names do not contain `Link`,
`Wrap`, `SendRequest`, or `buildResourceControlInterceptor`.

**Alt.** Map-based name replacement in `Link`; recursive `wrapAt` instead of the
reverse for-loop; `applyBoundInterceptor` helper; flattened resource-control
body. Applied on the buggy tree → f2p pass.

**Cheat.** `Link` appends with no dedupe; `Wrap` is identity; `SendRequest` only
invokes a bound interceptor when `Name() == "test"` (the single-interceptor
case). `TestInterceptedClient` can pass; chain tests still fail.

### Validation

| check | result |
|---|---|
| builds | yes |
| library `go test ./...` | green except `internal/client` (`TestInterceptedClient`, `TestAppendChainedInterceptor`) and `tikvrpc/interceptor` (`TestInterceptor`). No `TestTiKVRecoveredFromDown` flake |
| f2p ≥ 1 | 3 |
| 3× not flaky | fail ×3 |
| f2p files in impact | `client_interceptor_test.go`, `interceptor_test.go` ∈ interceptor blast radius |
| no collateral | only those two packages failed full `./...` |
| gold revert | restore the two files → pass; docker image same |
| alt accepted | map/recursive rewrite on buggy tree → pass |
| cheat rejected | name `"test"` special-case → still fail |
| docker | `harbor-iter8-interceptor`: buggy fail, gold pass (`verifier/iter8_docker.log`) |

**Knobs.** functions removed 4; files 2; lines ~96 (net); **interface kept**;
hops 3 (`Link` from `TestInterceptedClient`); decoys 0; locality **L2**.

**NAME LEAKAGE L2.** f2p names and failure text (`expected: 2 actual: 0`,
`Should be true`, `[]int{0, 1}`) do not contain `Link` / `Wrap` / `SendRequest`
/ `buildResourceControlInterceptor` or `interceptor.go` / `client_interceptor.go`.
Instruction is a behavior-level feature request plus
`go test -count=1 -timeout 15m ./tikvrpc/interceptor/... ./internal/client/...`
(no `-run`, no `Test*` names). `name_leakage` ok=true.
`instruction_self_check` ok=true (locality 2).

## Bug B — `client-go-batchcmds` (rung 8 large excision, L2, interface removed)

**Feature.** Pack many KV RPCs onto one batch stream: convert a typed request
into a batch entry, enqueue with timeout/cancel, builder groups by forwarded
host and skips canceled items.

**Excision.** Exported `ToBatchCommandsRequest` **deleted**; callers re-wired
(`sendRequest` skips packing; mock coverage dummy dropped). Internal packing
pipeline stubbed:

| function | file | action |
|---|---|---|
| `(*Request).ToBatchCommandsRequest` | `tikvrpc/tikvrpc.go` | **deleted** |
| `sendRequest` batch branch | `internal/client/client.go` | re-wired (always unary) |
| mock `SendRequest` dummy call | `internal/mockstore/mocktikv/rpc.go` | re-wired |
| `sendBatchRequest` | `internal/client/client_batch.go` | stub error |
| `batchCommandsBuilder.len` | `internal/client/client_batch.go` | return 0 |
| `hasHighPriorityTask` | `internal/client/client_batch.go` | return false |
| `buildWithLimit` | `internal/client/client_batch.go` | drain, return nil |
| `fetchAllPendingRequests` | `internal/client/client_batch.go` | no-op |
| `fetchMorePendingRequests` | `internal/client/client_batch.go` | no-op |

`push` / `reset` / `cancel` kept so `TestBatchCommandsBuilder` fails on values
instead of hanging on a nil channel.

**Graph path** (`codegraph_explore`).

```
TestForwardMetadataByBatchCommands
  -> RPCClient.SendRequest -> sendRequest
       -> ToBatchCommandsRequest          # deleted
       -> sendBatchRequest                # stubbed

TestCancelTimeoutRetErr
  -> sendBatchRequest                     # stubbed

TestBatchCommandsBuilder
  -> newBatchCommandsBuilder
       -> push / len / buildWithLimit / reset / cancel
```

**f2p.** `TestCancelTimeoutRetErr`, `TestBatchCommandsBuilder`,
`TestForwardMetadataByBatchCommands` (table-driven builder + forwarded-host
stream counts). Names do not contain `ToBatchCommandsRequest`. Same-feature
failpoint tests `TestPanicInRecvLoop` / `TestRecvErrorInMultipleRecvLoops` also
fail on the buggy tree (not in `test.sh`). Dedicated `go test ./internal/locate`
re-run was green.

**Alt.** Package-level `packBatchCommand` map (no `Request` method);
iterative `buildWithLimit`; `waitBatchEntry` helper. Applied on the buggy tree
→ f2p pass.

**Cheat.** `sendBatchRequest` returns `Canceled` only when `ctx.Err() != nil`,
else `"batch send unavailable"`. Builder still stubbed. First cancel case can
pass; deadline / builder / forwarded-host cases fail.

### Validation

| check | result |
|---|---|
| builds | yes (unused `trace`/`fmt` imports stripped) |
| library `go test ./...` | green except `internal/client` feature tests above. locate panic in the full-suite log did not reproduce on a dedicated `./internal/locate` run. No `TestTiKVRecoveredFromDown` flake |
| f2p ≥ 1 | 3 |
| 3× not flaky | fail ×3 |
| f2p files in impact | `client_test.go` ∈ `ToBatchCommandsRequest` / `sendBatchRequest` blast radius |
| no collateral | unrelated packages (codec, oracle, unionstore, tikv, txnkv, interceptor) green |
| gold revert | restore four files → pass; docker image same |
| alt accepted | map packer + iterative builder on buggy tree → pass |
| cheat rejected | cancel-only special case → still fail |
| docker | `harbor-iter8-batchcmds`: buggy fail, gold pass (`verifier/iter8_docker.log`) |

**Knobs.** functions removed 7; files 4; lines ~221 (net); **interface removed**;
hops 3 (`ToBatchCommandsRequest` from `TestForwardMetadataByBatchCommands`);
decoys 0; locality **L2**.

**NAME LEAKAGE L2.** f2p names and failure text (`batch send unavailable` vs
`context canceled`, `expected: 0x3 actual: 0x1`, `expected: 0 actual: 1`) do
not contain `ToBatchCommandsRequest` / `sendBatchRequest` / `buildWithLimit` or
`tikvrpc.go` / `client.go` / `client_batch.go`. Instruction is a behavior-level
feature request plus `go test -count=1 -timeout 15m ./internal/client/...`
(no `-run`, no `Test*` names). `name_leakage` ok=true.
`instruction_self_check` ok=true (locality 2).

## Reusable construction

`src/openswe_traces/synth/difficulty.py`:

- `pick_feature_excision(index, min_functions, min_files, keep_interface, min_lines, n)`
  — callee BFS from exported functions; records tests via `calls` edges
- `design_for_rung(..., keep_interface=)` emits `excision` at **rung 8**
  (`min_functions` 3 vs 6, `min_files` 2 vs 3, `min_lines` 60 vs 150)
- CLI: `openswe-synth --repo <path> --rung 8 --sites 3 --keep-interface`
  / `--no-keep-interface`
- `issue_from_failures(..., kind="feature")` writes a behavior-level feature
  request (L2)

Fixture: `RoundTrip`/`Pack`/`Unpack` 3-file graph on
`experiments/codegraph_bugs/fixture_host`.
Tests: `tests/test_synth_difficulty.py` (`test_pick_feature_excision_and_rung8`).
`uv run pytest tests/` — 79 passed.
`uv run ruff check src/openswe_traces/synth/difficulty.py src/openswe_traces/synth/harbor_tasks.py tests/test_synth_difficulty.py` — clean.

## Harbor launch (detached, not waited)

Job dir `experiments/harbor_nex/jobs/grok-xhigh-iter8/` (trials
`client-go-interceptor` and `client-go-batchcmds` started concurrently; not waited).

```bash
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
cd /home/evan/Documents/open_swe_traces_research
setsid nohup harbor run \
  --path experiments/harbor_nex/tasks_iter8 \
  --agent cursor-cli \
  --model cursor/cursor-grok-4.6-xhigh \
  --n-concurrent 2 \
  --n-attempts 1 \
  --max-retries 3 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name grok-xhigh-iter8 \
  --yes \
  > experiments/harbor_nex/grok-xhigh-iter8.log 2>&1 < /dev/null &
```

`CURSOR_API_KEY` sourced from `/home/evan/Documents/eval_tasks/.env` (not printed, not committed).

Patches: `experiments/codegraph_bugs/bugs/client-go/{Interceptor,BatchCmds}.{patch,alt,cheat}.patch`.
Tasks: `experiments/harbor_nex/tasks_iter8/{client-go-interceptor,client-go-batchcmds}/`.
Agent timeout 14400s; golang:1.23; checksum guards on f2p test files.

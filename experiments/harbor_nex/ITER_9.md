# harbor_nex ITER_9 — grok-xhigh adversarial loop

Date: 2026-09-18. Solver: `cursor-cli` / `cursor/cursor-grok-4.6-xhigh`. Host:
`experiments/codegraph_bugs/repos/client-go` @ `190f0cce536f835b72481f7bcd0a9448bb1e5202`.
Prior artifacts: `analytics/research/synthetic_difficulty_ladder.md`, `RESULT.md`,
`TWO_REPO_TASK.md`, `ITER_3.md`–`ITER_8.md`, every `jobs/grok-xhigh*/result.json`,
`src/openswe_traces/synth/*.py`. `HARD_TASKS.md` / `ITER_1.md` / `ITER_2.md` were
not on disk.

Did not touch `jobs/grok-xhigh-iter8/` (finished), `tasks_iter8/`, or any earlier
task/job dir.

## Solver has cleared

Wall = `finished_at − started_at`. Reward 1.0 on every finished grok-xhigh trial.
Most recently **finished** job is `grok-xhigh-iter8` (both trials pass: interceptor
3.85 min, batchcmds 7.57 min). No legitimate fails exist, so there is no failure
audit.

**Threshold estimate:** highest passed **rung 8**, interface deleted, 7 functions /
4 files / ~221 lines, L2 (`client-go-batchcmds`, 7.57 min, D=16.75). Lowest
legitimately failed rung **none**. Mutations (rungs 1–5), implicit-invariant (6),
sequence (7), and mid-size excision (8) are exhausted.

OVERRIDE (iter 9): do not add trickery. Both tasks are **new regimes** at locality
**L2**. Task C (race 10/10) not shipped — not measured to the 10/10 gate.

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
| client-go-interceptor | grok-xhigh-iter8 | 8 | L2 | 1 | yes | 3.85 |
| client-go-batchcmds | grok-xhigh-iter8 | 8 | L2 | 1 | yes | 7.57 |
| client-go-onepc-scope | grok-xhigh-two-repo | 4 | L0 | 1 | yes | 4.11 |

Codegraph used deeper than iter8: `codegraph_explore` on the API v2 callee
closure (`NewCodecV2` / `EncodeRequest` / `encodeKeys` / `DecodeBucketKeys` /
`ParseKeyspaceID` / `IsDecodeError` / `NewCodecPDClientWithKeyspace`) and on
the MemDB Get path (`TestGetSet` → `Get` → `traverse`, plus `BenchmarkGet`).
`pick_subsystem_excision` (12/5/500, `keep_interface=False`) and
`find_perf_gates` / `parse_bench_ns_op` / `race_gate` are the reusable knobs.

## Bug A — `client-go-keyspacecodec` (subsystem-scale excision, L2)

**Subsystem.** API v2 keyspace codec: prefix user keys with a 4-byte
mode+keyspace header, mem-comparable wrap for region keys, decode responses /
region-error metadata / bucket split keys back to user keys, attach API
version + keyspace id on RPC context, classify malformed region keys as fatal
(no backoff).

**Excision.** 5 files, **906 net lines** removed (`git diff --stat`:
+16 / −922). 17 unexported helpers **deleted** (signatures gone). Exported
`Codec` methods kept as identity stubs so the rest of the library still
compiles (deleting `NewCodecV2` would break locate/tikv at build time).

| site | file | action |
|---|---|---|
| 17 helpers (`encodeKeys`, `encodeMutations`, `decodePairs`, `decodeLockInfo`, …) | `internal/apicodec/codec_v2.go` | **deleted** |
| `EncodeRequest` / `DecodeResponse` / `EncodeKey` / `DecodeKey` / range+bucket methods | `codec_v2.go` | identity stubs |
| `ParseKeyspaceID`, V2 `DecodeKey` branch, `attachAPICtx` | `codec.go` | stub / skip attach |
| `decodeRegionError` | `codec_v1.go` | identity |
| `IsDecodeError` | `mem_codec.go` | always false (encode/decode kept) |
| `NewCodecPDClientWithKeyspace`, `GetKeyspaceID` | `internal/locate/pd_codec.go` | error stubs |

**Graph path** (`codegraph_explore` on the host index).

```
TestCodecV2/{EncodeRequest,EncodeV2KeyRanges,EncodeMPPRequest,DecodeBucketKeys,DecodeEpochNotMatch}
  -> codecV2.EncodeRequest / encodeKeyRanges / DecodeBucketKeys / decodeRegionError
       -> EncodeKey / encodeKeys / encodeMutations / decodePairs     # deleted
ParseKeyspaceID / DecodeKey (package)                                # stubbed
TestRegionCache/TestNoBackoffWhenFailToDecodeRegion
  -> loadRegion* -> IsDecodeError                                    # always false
NewCodecPDClientWithKeyspace -> NewCodecV2 / GetKeyspaceID           # stubbed
```

Callee closure of `NewCodecV2` spans `codec_v2.go` + `mem_codec.go` +
`codec.go`; PD wrapper is the cross-package site (`locate` imports `apicodec`).

**f2p (existing tests, no new tests).** `TestCodecV2` (5 failing subtests) and
`TestRegionCache` (subtest `TestNoBackoffWhenFailToDecodeRegion`). ≥ 6 existing
cases. Names do not contain `EncodeKey`, `DecodeKey`, `ParseKeyspaceID`,
`IsDecodeError`, or `codec.go`. `TestParseKeyspaceID` / `TestDecodeKey` also
fail on the buggy tree (not in `test.sh`; name-leak if used as f2p).

**Alt (one representative function).** `EncodeKey` prefix-copy on the buggy
tree. **A full alternative implementation of this subsystem is infeasible**
at this size (17 deleted helpers + request/response/range/bucket/PD paths);
the one-function alt still fails `TestCodecV2`. Gold restore is the solvability
proof.

**Cheat.** `EncodeRequest` special-cases RawGet of user key `"key"`. Other
EncodeRequest / range / bucket / backoff cases still fail.

### Validation

| check | result |
|---|---|
| builds | yes (`go test -c ./internal/apicodec/ ./internal/locate/` rc=0) |
| library `go test ./...` | green except `internal/apicodec` (intended) and `internal/locate` `TestRegionCache/TestNoBackoffWhenFailToDecodeRegion`. No `TestTiKVRecoveredFromDown` flake |
| f2p ≥ 1 | 2 Test* / 6+ cases |
| 3× not flaky | `TestCodecV2` fail ×3; `TestNoBackoffWhenFailToDecodeRegion` fail ×3 |
| f2p files in impact | `codec_v2_test.go`, `region_cache_test.go` ∈ codec / `IsDecodeError` blast radius |
| no collateral | remaining packages (excluding apicodec+locate) green (`rest_rc=0`) |
| gold revert | restore five files → `TestCodecV2` + `TestNoBackoff*` pass; docker image same |
| alt accepted | **N/A at this size** — one-function `EncodeKey` alt does **not** pass f2p; full alt infeasible (see above) |
| cheat rejected | RawGet `"key"` special-case → still fail |
| docker | `harbor-iter9-keyspacecodec`: buggy `REWARD=0`, gold `REWARD=1` (`verifier/iter9_docker.log`) |

**Knobs.** functions 17 deleted + 21 stubbed; files 5; lines 906 net;
**interface removed** (helpers); hops 2–3 from `TestCodecV2` to `encodeKeys`;
decoys 0; locality **L2**; subsystem-scale (sites ≥ 12, files ≥ 5, lines ≥ 500).

**NAME LEAKAGE L2.** f2p names `TestCodecV2`, `TestRegionCache` and the
instruction excerpt (`expected: 0x10203 actual: 0xffffffff`, RawGet bytes
`0x72 0x00 0x10 0x92 …` vs bare `key`) do not contain `EncodeKey` /
`DecodeKey` / `ParseKeyspaceID` / `IsDecodeError` / `codec.go` /
`codec_v2.go` / `mem_codec.go` / `pd_codec.go`. Instruction is a maintainer
design brief plus
`go test -count=1 -timeout 15m ./internal/apicodec/ ./internal/locate/`
(no `-run`, no `Test*` names). `name_leakage` ok=true.
`instruction_self_check` ok=true (locality 2).

## Bug B — `client-go-memget` (performance-constrained reimplementation, L2)

**Path.** In-memory write buffer Get/Set: store pairs, return latest value or
not-exist. Existing `BenchmarkGet` inserts `b.N` keys then looks them all up.

**Excision.** 3 functions in `internal/unionstore/memdb.go`:

| function | stub |
|---|---|
| `(*MemDB).Get` | always `ErrNotExist` |
| `(*MemDB).set` | mutex/size checks, then return (no tree insert) |
| `(*MemDB).traverse` | immediate `nullAddr` |

**Graph path.**

```
TestGetSet / TestKVGetSet
  -> fillDB/insertData -> Set -> set     # stubbed
  -> Get -> traverse                     # stubbed
BenchmarkGet
  -> Set then Get                        # same stubs
```

**f2p.** `TestGetSet`, `TestKVGetSet` (correctness) plus existing
`BenchmarkGet` with gold×3 ns/op ceiling. Names do not contain `traverse` or
`memdb.go`. `\bGet\b` does not match `TestGetSet` / `BenchmarkGet`.

**Naive (not-too-broad check).** Linear `[]struct{k,v []byte}` scan for Get/set
(O(n) per lookup). Passes correctness; fails the perf gate.

**Cheat.** `Get` returns the key copy only for `{0,0,0,0}`. `TestGetSet` still
fails on i=1.

### Performance measurements (golang:1.23 image `harbor-iter9-memget`)

Command: `go test -count=1 -timeout 15m -bench=^BenchmarkGet$ -benchtime=5000x -run=^$ ./internal/unionstore/`

| implementation | ns/op | vs limit 754 |
|---|---:|---|
| gold (dedicated measure) | **251.4** | pass |
| gold (via test.sh) | 232.1 | pass |
| naive linear scan | 6806 | **fail** (6806 > 754) |
| host (not the gate) | 144.9 | — |

Limit = `int(251.4 × 3) = 754` ns/op. Host 144.9 is recorded but **not** used
as the ceiling (user: measure in the task image).

### Validation

| check | result |
|---|---|
| builds | yes |
| library `go test ./...` | green except `internal/unionstore` Get/Set tests. Other packages `NFAIL=0`. No `TestTiKVRecoveredFromDown` flake |
| f2p ≥ 1 | 2 correctness + BenchmarkGet gate |
| 3× not flaky | `TestGetSet`/`TestKVGetSet` fail ×3 |
| f2p files in impact | `memdb_test.go`, `memdb_bench_test.go` checksum-guarded; ∈ `Get` blast radius |
| no collateral | remaining packages green on the buggy tree |
| gold revert | restore `memdb.go` → correctness + perf pass; docker `REWARD=1` |
| naive passes correctness / fails gate | docker: tests pass, `6806 ns/op > 754` → `REWARD=0` |
| cheat rejected | `{0,0,0,0}` special-case → correctness fail, `REWARD=0` |
| docker | `harbor-iter9-memget`: buggy 0 / gold 1 / naive 0 / cheat 0 |

**Knobs.** functions 3; files 1; hops 1–2 (`TestGetSet`→`Get`→`traverse`);
decoys 0; locality **L2**; perf gate gold×3 on existing `BenchmarkGet`.

**NAME LEAKAGE L2.** f2p names `TestGetSet`, `TestKVGetSet` and failure text
(`Expected nil, but got: not exist`, `expected: "" actual: "0000000000"`) do
not contain `Get` / `set` / `traverse` / `memdb.go` as whole tokens.
Instruction states the correctness contract and the performance requirement
plus
`go test -count=1 -timeout 15m ./internal/unionstore/`
and the bench command without `-run`. `name_leakage` ok=true.
`instruction_self_check` ok=true (locality 2).

## Task C — concurrency (not shipped)

`race_gate` exists (fixture `Bag` mutex+map; client-go mutex+map files). No
10/10 `go test -race` fail-buggy / pass-gold measurement, so no race task.

## Reusable synth

`src/openswe_traces/synth/difficulty.py`:

- `pick_subsystem_excision(index, min_functions=12, min_files=5, min_lines=500)`
- `find_perf_gates` / `parse_bench_ns_op`
- `race_gate`
- `design_for_rung(..., rung=8, sites>=12, keep_interface=False)` → `subsystem: true`
- CLI: `openswe-synth --repo <path> --rung 8 --sites 12 --no-keep-interface --perf-bench BenchmarkGet --perf-limit-ns 754 --perf-benchtime 5000x --locality 2 --patch … --out …`

`harbor_tasks.render_test_sh` runs correctness then the bench ns/op ceiling
(awk, no python3 — `golang:1.23` image has no Python).

Tests: `tests/test_synth_difficulty.py::test_pick_subsystem_excision_perf_gates_and_race`.
`uv run ruff check` clean; `uv run pytest tests/test_synth_difficulty.py` 12 passed.

## Harbor

Tasks: `experiments/harbor_nex/tasks_iter9/{client-go-keyspacecodec,client-go-memget}/`
(golang:1.23, no `.git`, modules pre-downloaded, agent timeout 14400s, checksum
guards on f2p `*_test.go` + `memdb_bench_test.go`).

Launched detached (do not wait):

```
setsid nohup harbor run \
  --path experiments/harbor_nex/tasks_iter9 \
  --agent cursor-cli \
  --model cursor/cursor-grok-4.6-xhigh \
  --n-concurrent 2 \
  --n-attempts 1 \
  --max-retries 3 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name grok-xhigh-iter9 \
  --yes
```

`CURSOR_API_KEY` from `/home/evan/Documents/eval_tasks/.env` (not printed).
Wrapper: `experiments/harbor_nex/run_iter9.sh`. Log:
`experiments/harbor_nex/grok-xhigh-iter9.log`. Job dir:
`experiments/harbor_nex/jobs/grok-xhigh-iter9`.

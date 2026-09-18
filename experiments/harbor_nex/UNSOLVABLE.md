# UNSOLVABLE → SOLVABLE (affordance ladder)

Date: 2026-09-18. Knob: **what information the solver needs**, not graph
structure. Cheap solver: `cursor-cli` / `cursor/composer-2.5`. Host tree:
obfuscated client-go snapshot
`experiments/harbor_nex/tasks_obf/client-go-keyspacecodec-obf/environment/src`
(module `example.internal/kvstore/v2`). Did not modify `tasks_obf/`.

Composer 2.5 had cleared 26/26 earlier tasks, including the 906-line keyspace
codec excision, because (a) in-tree tests were a complete spec and (b) the
repo is public. These families delete that spec from the tree (F1), replace
example tests with a property oracle (F2), or add a race+throughput gate (F3).

Build:

```
uv run python scripts/build_unsolv.py \
  --obf-task experiments/harbor_nex/tasks_obf/client-go-keyspacecodec-obf \
  --dest experiments/harbor_nex/tasks_unsolv \
  --upstream experiments/harbor_nex/tasks_iter9/client-go-keyspacecodec/environment/src
```

Reusable: `src/openswe_traces/synth/affordance.py`
`build_affordance_levels(task_dir, hidden_tests, levels)`.
Tests: `uv run pytest tests/test_affordance.py`.

## Affordance ladder

Each family is five Harbor dirs `tasks_unsolv/<family>-A<k>/`. A0 copies live
in `tasks_unsolv_A0/` (Harbor `--path` contains only those three).

| level | what the agent gets |
|---:|---|
| A0 | Bare prose contract + reproduce command. Subsystem unit tests are **not** in the tree. Verifier copies checksum-guarded files from `tests/hidden/` into `/app` and runs them. |
| A1 | A0 + the **names and one-line descriptions** of the hidden tests (appended to the instruction). |
| A2 | A1 + **exported signatures** as zero-value stubs with godoc on the removed API. |
| A3 | A2 + **one** hidden test file restored into the tree (the most representative). |
| A4 | A3 + **all** hidden tests restored. Control: equivalent to the earlier in-tree-test excision tasks. |

Every instruction ends with the no-web clause. `task.toml`: `[agent]`
allowlist Cursor hosts, `[verifier] network_mode = "no-network"`.

## F1 spec-only reimplementation (`spec-reimpl`)

**Subsystem.** API v2 keyspace codec (same 5-file excision as ITER_9
keyspacecodec, on the obfuscated tree). Gold 1564 lines vs excised 658 across
`codec.go`, `codec_v2.go`, `codec_v1.go`, `mem_codec.go`, `pd_codec.go`
(**906 lines**, 5 files, 17 helpers deleted from `codec_v2.go` 38→21 funcs;
exported methods kept as identity stubs so the rest of the library compiles).

**Hidden tests** (not in the A0–A2 tree): `codec_v2_test.go`, `codec_test.go`,
and `decode_fatal_test.go` (fatal decode classification). Representative for
A3: `codec_v2_test.go`.

**Instruction.** Prose contract written from those tests. No function names,
no signatures beyond what callers already force. Reproduce:
`go test -count=1 -timeout 15m ./internal/apicodec/`.

### F1 coverage table (hidden test → contract sentence)

| hidden test | sentence in the contract |
|---|---|
| TestCodecV2/TestEncodeRequest | A raw single-key lookup of user key `key` in keyspace 0x1092 must be sent with the 4-byte raw-mode header `0x72 0x00 0x10 0x92` followed by the user key, not the bare user key. |
| TestCodecV2/TestEncodeV2KeyRanges | User ranges whose start or end is empty expand to the keyspace interval: empty start becomes the keyspace prefix, empty end becomes the next keyspace prefix; both empty becomes the whole keyspace. |
| TestCodecV2/TestNewCodecV2 | Constructing a v2 codec rejects a keyspace id that does not fit in 24 bits and rejects an unknown mode; the prefix is a mode byte plus the 24-bit id; the exclusive end prefix is that 32-bit value plus one, with carry across bytes; the last raw id wraps the mode byte from `r` to `s`. |
| TestCodecV2/TestDecodeEpochNotMatch | Epoch-not-match region lists are clipped to the keyspace and decoded to user keys: a region covering the whole keyspace becomes empty/empty, a region wholly outside the keyspace is dropped, and a region overlapping the keyspace is truncated to the overlap then stripped of the header. |
| TestCodecV2/TestGetKeyspaceID | A codec built for keyspace 4242 reports that same id. |
| TestCodecV2/TestEncodeMPPRequest | An MPP dispatch must carry keyspace id 4242 and API version 2 on its task meta, and its coprocessor ranges must be encoded the same way as ordinary user keys. |
| TestCodecV2/TestDecodeBucketKeys | Bucket keys `a`, `b`, `c` mixed with previous/next keyspace encoded neighbors round-trip to those same user keys plus empty sentinels. |
| TestParseKeyspaceID | A 4-byte header starting with txn `x` or raw `r` followed by 0x010203 yields keyspace id 0x010203; a header whose mode byte is neither, or that is shorter than 4 bytes, errors and yields the all-ones null id 0xffffffff. |
| TestDecodeKey | API v2 splits a well-formed key into a 4-byte header and the remaining user bytes; API v1 is identity (no header); an invalid v2 mode byte errors with empty results. |
| TestEncodeUnknownRequest | A request type with no key payload (store safe-ts) still encodes without error under the transactional v1 codec. |
| TestMalformedRegionKeyIsDecodeError | A truncated or otherwise non mem-comparable region key is a fatal decode: the client must classify it as a decode failure so callers do not retry it with backoff. |

### F1 alt / cheat

Two-function alt (different structure, rest gold): user-key prefixing via
`make`+`copy` instead of `append`; header parse via a `[4]byte` buffer instead
of slice copy + zero first byte. Hidden tests **accept** it.

A full alternative implementation of this subsystem is infeasible at this
size (17 deleted helpers plus request/response/range/bucket/PD paths).

Cheat: special-case raw-get of `"key"` only. Hidden `TestCodecV2` still fails
(ranges, buckets, epoch clip, MPP).

## F2 property-verified (`property-backoff`)

**Component.** Truncated exponential backoff envelope
`expo(base, cap, n) = min(cap, base * 2^n)` in
`internal/client/retry/config.go`. Excised to `return base` (constant).
Original example tests (`backoff_test.go`) are **not** the verifier and are
removed from the tree.

**Verifier.** New property test `backoff_prop_test.go`: seed `20260918`,
**10 000** random `(base, cap, n)` cases, plus adversarial edges (cap hit,
`n=40`/`n=80` overflow-to-cap, `cap==base`, monotone until cap). Naive
constant-`base` fails case 0 (`expo(7,7844,2)` want 28). Cheat that only
returns the three worked examples also fails case 0.

**Instruction.** Properties in words plus three worked examples:
base=2, cap=500, n=0→2; n=1→4; n=8→500.

A3 representative = the property test file (only hidden file; A3≡A4 for
this family except A4 is the explicit control restore).

## F3 dynamic (`dynamic-pipeline`)

**Component.** Pipelined write buffer
(`internal/unionstore/pipelined_memdb.go`): Get / Flush / needFlush stubbed
(Get always not-exist; Flush never triggers). Gold implementation restored
from the obfuscated tree.

**Verifier.** Original pipelined correctness tests (hidden) + new
`TestPipelinedConcurrentSet` under `go test -race` + `BenchmarkPipelinedGet`
(`-benchtime=5000x`, parallel Gets after 5000 inserts). Dockerfile includes
`gcc libc6-dev` for `-race`.

`parse_bench_ns_op` from ITER_9 is the gate parser.
`find_perf_gates` / `race_gate` were called on the gold snapshot; that tree
has no `.codegraph` index, so they no-op. The shipped gate is the same shape
as ITER_9 (`Benchmark*` ns/op ceiling, `-race` on the f2p tests).

### Throughput (host, AMD Ryzen 7 6800U)

Command: `go test -count=1 -timeout 15m -bench=^BenchmarkPipelinedGet$ -benchtime=5000x -run=^$ ./internal/unionstore/`

| implementation | ns/op | vs limit 113 |
|---|---:|---|
| gold | **37.77** | pass |
| naive mutex + linear `Iter` scan of every pair | 10709 | **fail** (10709 > 113) |

Limit = `int(37.77 × 3) = 113` ns/op. Naive is also race-free and passes
correctness (`TestPipelinedFlushTrigger`, `TestPipelinedConcurrentSet`).

Cheat: stub Get with a one-byte special case. Correctness still fails.

A3 representative: `pipelined_memdb_test.go`.

## Validation

All checks `ok: true` in `experiments/harbor_nex/tasks_unsolv/validation.json`.

| family | gold restore | rest of suite | cheat | instruction self-check | searchability |
|---|---|---|---|---|---|
| spec-reimpl | pass hidden apicodec tests | unionstore + retry + util/codec green on excised tree | RawGet `"key"` still fail | locality 2 `ok` | 5 obfuscated idents, 0 upstream hits |
| property-backoff | pass 10k property test | apicodec green (gold codec) | 3 examples hardcoded fail | (A0 has expected/actual + `go test`, no `-run`) | same mapping |
| dynamic-pipeline | correctness + `-race` pass | unionstore (minus hidden pipelined tests) + apicodec + retry green | one-key Get still fail | no `-run` in instruction | same mapping |

Searchability idents (vs unobfuscated ITER_9 src): `NimbusClip`, `NimbusCore`,
`NimbusGate`, `NimbusJoin`, `NimbusPack` — 0 matches.

## Harbor

A0-only path: `experiments/harbor_nex/tasks_unsolv_A0/`
(`spec-reimpl-A0`, `property-backoff-A0`, `dynamic-pipeline-A0`).

Launched detached (do not wait):

```
setsid nohup harbor run \
  --path experiments/harbor_nex/tasks_unsolv_A0 \
  --agent cursor-cli \
  --model cursor/composer-2.5 \
  --n-concurrent 3 \
  --n-attempts 1 \
  --max-retries 2 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name composer25-unsolv-A0 \
  --yes
```

`CURSOR_API_KEY` from `/home/evan/Documents/eval_tasks/.env` (not printed).
Wrapper: `experiments/harbor_nex/run_unsolv_a0.sh`. Log:
`experiments/harbor_nex/composer25-unsolv-A0.log`. Job dir:
`experiments/harbor_nex/jobs/composer25-unsolv-A0` (wrapper pid 2026483).

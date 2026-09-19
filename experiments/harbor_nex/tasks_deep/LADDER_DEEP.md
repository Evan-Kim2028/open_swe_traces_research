# Ladder deep (A-1, A-2)

Date: 2026-09-18. Extends the affordance ladder **below** the A0 contract
for two units that held out at A0 (dynamic-pipeline, spec-reimpl-bb) and
two A0-passing controls (spec-bb-chain, dynamic-latch). Dest:
`experiments/harbor_nex/tasks_deep/<unit>-A-1/` and `-A-2/`.
Copied from the existing contract-only trees (L2/A0); those dirs were
not modified. No Harbor jobs were launched.

Build:

```
uv run python scripts/build_ladder_deep.py
```

| level | what the agent gets |
|---|---|
| A-1 | Gapped contract: A0 prose minus **one** invariant that is still 
discoverable in the repo (doc comment / neighboring test / caller). Hidden 
verifier adds that pre-existing unmodified repo test. |
| A-2 | Bug report only: observable breakage + repro that runs the hidden 
black-box suite + no-web. Unit excised as at A0. Deepest fair level. |

## `dynamic-pipeline`

- **Family:** dynamic
- **Source:** `/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex/tasks_unsolv/dynamic-pipeline-L2`
- **A-1:** `/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex/tasks_deep/dynamic-pipeline-A-1`
- **A-2:** `/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex/tasks_deep/dynamic-pipeline-A-2`
- **Composer at A0:** FAIL (held out)

### A-1 omitted invariant

- **Omitted:** Lookup priority through the in-flight flush buffer and the remote store (mutable > flushing > remote).
- **Discoverable:** `internal/unionstore/pipelined_memdb.go:95-96`
- **Catcher (pre-existing, unmodified):** `TestPipelinedFlushGet` in `internal/unionstore/pipelined_memdb_test.go`

### A-2 bug report (first lines)

```
# Bug report

I wrote a key and then looked it up. The lookup said the key does not
exist. expected the value I just stored, actual not exist.
```

Validation (built image, C5 harness checked first):

| check | result |
|---|---|
| `A-2_buggy_fails` | pass |
| `A-2_gold_pass` | pass |
| `A-1_buggy_fails` | pass |
| `A-1_gold_pass` | pass |
| `gold_restore` | pass |
| `buggy_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | pass |
| `patches_skip_tests` | pass |
| `proof_harness` | pass |
| `blackbox_hygiene` | pass |

## `spec-reimpl-bb`

- **Family:** spec-bb
- **Source:** `/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-L2`
- **A-1:** `/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex/tasks_deep/spec-reimpl-bb-A-1`
- **A-2:** `/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex/tasks_deep/spec-reimpl-bb-A-2`
- **Composer at A0:** FAIL (held out)

### A-1 omitted invariant

- **Omitted:** A header whose mode byte is not raw/txn, or that is shorter than 4 bytes, errors and yields the all-ones null id 0xffffffff.
- **Discoverable:** `internal/apicodec/codec.go:24-29`
- **Catcher (pre-existing, unmodified):** `TestParseKeyspaceID` in `internal/apicodec/codec_test.go`

### A-2 bug report (first lines)

```
# Bug report

A raw single-key lookup of user key `key` in keyspace 0x1092 went out as
the bare user key. expected the four-byte prefix `0x72 0x00 0x10 0x92`
followed by the user key, actual the bare bytes. expected keyspace id
0x10203 from a well-formed header, actual 0xffffffff.
```

Validation (built image, C5 harness checked first):

| check | result |
|---|---|
| `A-2_buggy_fails` | pass |
| `A-2_gold_pass` | pass |
| `A-1_buggy_fails` | pass |
| `A-1_gold_pass` | pass |
| `gold_restore` | pass |
| `buggy_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | pass |
| `patches_skip_tests` | pass |
| `proof_harness` | pass |
| `blackbox_hygiene` | pass |

## `spec-bb-chain`

- **Family:** spec-bb
- **Source:** `/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex/tasks_ladder2/spec-bb-chain-L2`
- **A-1:** `/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex/tasks_deep/spec-bb-chain-A-1`
- **A-2:** `/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex/tasks_deep/spec-bb-chain-A-2`
- **Composer at A0:** PASS (control)

### A-1 omitted invariant

- **Omitted:** Two decorators that share a name do not both stay on the stack; the later attachment replaces the earlier one.
- **Discoverable:** `wirerpc/interceptor/interceptor.go:109-110`
- **Catcher (pre-existing, unmodified):** `TestAppendChainedInterceptor` in `internal/client/client_interceptor_test.go`

### A-2 bug report (first lines)

```
# Bug report

I attached two wrappers named first then second and invoked the stack.
Only the base call ran. expected first then second then the base call,
actual only the base call.
```

Validation (built image, C5 harness checked first):

| check | result |
|---|---|
| `A-2_buggy_fails` | pass |
| `A-2_gold_pass` | pass |
| `A-2_cheat_fails` | pass |
| `A-1_buggy_fails` | pass |
| `A-1_gold_pass` | pass |
| `gold_restore` | pass |
| `buggy_fails` | pass |
| `A-1_cheat_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | pass |
| `patches_skip_tests` | pass |
| `proof_harness` | pass |
| `blackbox_hygiene` | pass |

## `dynamic-latch`

- **Family:** dynamic
- **Source:** `/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex/tasks_ladder2/dynamic-latch-L2`
- **A-1:** `/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex/tasks_deep/dynamic-latch-A-1`
- **A-2:** `/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex/tasks_deep/dynamic-latch-A-2`
- **Composer at A0:** PASS (control)

### A-1 omitted invariant

- **Omitted:** After release, waiters are woken (the granted waiter proceeds).
- **Discoverable:** `internal/latch/latch_test.go:136-137`
- **Catcher (pre-existing, unmodified):** `TestWithConcurrency` in `internal/latch/scheduler_test.go`

### A-2 bug report (first lines)

```
# Bug report

Two overlapping holds on the same key both entered the critical section.
expected the second to wait until the first released, actual both
proceeded (or the race detector fired).
```

Validation (built image, C5 harness checked first):

| check | result |
|---|---|
| `A-2_buggy_fails` | pass |
| `A-2_gold_pass` | pass |
| `A-2_cheat_fails` | pass |
| `A-1_buggy_fails` | pass |
| `A-1_gold_pass` | pass |
| `gold_restore` | pass |
| `buggy_fails` | pass |
| `A-1_cheat_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | pass |
| `patches_skip_tests` | pass |
| `proof_harness` | pass |
| `blackbox_hygiene` | pass |

## Rules

Fairness gates unchanged: gold passes, alt passes, cheat fails,
black-box (B4=pass required), checksum, no web. Each task dir has
`validation.json` with `rule_verdicts` (A1–A12, B1–B8, C5).
Proof harness (C5): `go` on PATH, `false` ≠ 0, `true` = 0, then
buggy REWARD=0 and gold REWARD=1 in the image.

`task.toml`: `[agent]` allowlist Cursor hosts;
`[verifier] network_mode = "no-network"`.


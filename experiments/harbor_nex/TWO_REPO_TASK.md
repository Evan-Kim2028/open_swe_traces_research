# TWO_REPO_TASK — client-go → integration_tests (`checkOnePC` scope inversion)

Date: 2026-09-18. One Harbor task. Did not touch `tasks_hard/` or `jobs/`.

## Cross-index finding

Library `experiments/codegraph_bugs/repos/client-go` was already indexed (214 files, including 29 under `integration_tests/`). Re-ran `codegraph init && codegraph index` in `integration_tests/` (29 files, 853 nodes, 1,967 edges).

| config | cwd | `callers checkOnePC` | `callers Commit` includes consumer tests? | cross-module `calls` |
|---|---|---|---|---|
| **parent = library tree** | `repos/client-go` | `execute` in `2pc.go` | **yes** (`Test1PC`, …) | **1911** |
| consumer-only | `repos/client-go/integration_tests` | symbol not found | no (2 local `Commit` methods) | **0** (6921 unresolved refs) |
| stitched AST | `two_repo.parse_consumer_clientgo_calls` | n/a | join on exported names | fallback |

**Winner: parent index.** Indexing only `integration_tests` cannot resolve `replace github.com/tikv/client-go/v2 => ../`. The client-go directory *is* the parent of both `go.mod` trees; no extra index above it is required. CLI `codegraph callers Commit` from that tree already lists consumer tests. Stitch helper remains in `src/openswe_traces/synth/two_repo.py` as backup.

## Pair choice

**`tikv/client-go` (library) → `integration_tests` (consumer).** Genuinely coupled: consumer `require` + `replace => ../`. Parent graph has 1911 call edges across the boundary (`Commit` 84×). `SetEnable1PC` is never called from library unit tests (only `examples/txnkv/1pc_txn` and consumer tests), so a 1PC-policy mutation can keep library `go test ./...` green.

Mock store: `NewTestStore` uses `testutils.NewMockTiKV` unless `-with-tikv`. `Test1PCWithMultiDC` / `Test1PCDisallowMultiRegion` skip when `*withTiKV`. No PD/TiKV process. Did not fall back to an external bank pair.

## Bug design

**Cause:** `checkOnePC` (`txnkv/transaction/2pc.go`): invert local-vs-global guard (`!= oracle.GlobalTxnScope` → `==`). 2 lines. Commits still succeed (contract drift, not a crash).

**Linked-graph path (parent sqlite, reverse `calls`, 3 hops, crosses the module boundary at `Commit`):**

```
Test1PC / TestOnePC          # integration_tests/1pc_test.go
  → KVTxn.Commit             # txnkv/transaction/txn.go  (exported; module boundary)
    → twoPhaseCommitter.execute
      → checkOnePC           # unexported; true cause
```

`codegraph_explore` also shows `Test1PCWithMultiDC → begin1PC → SetEnable1PC` (consumer sets the flag the library then mis-evaluates).

**Why library tests stay green:** default `enable1PC` is false; library tests never call `SetEnable1PC`. `checkOnePC` still returns false either way.

**Two fix sites:** (1) restore `!=` in the library (required); (2) rewrite consumer `IsOnePC` assertions (DISALLOWED — verifier checksums `*_test.go`). Alt: `scope := c.txn.GetScope(); if scope != …`. Cheat: special-case consumer key `k1` inside the still-inverted guard.

## Validation table

Host tree = `190f0cce` + `checkOnePC.patch`. Consumer tests: `go test -ldflags=-checklinkname=0 -count=1 -run '^(TestOnePC)$' .`

| check | result |
|---|---|
| library `go build ./...` | Y |
| consumer `go test -c` | Y |
| library `go test ./...` | Y (0 fails; `TestTiKVRecoveredFromDown` did not flake this run) |
| consumer f2p ≥ 1 | Y: `TestOnePC`, `Test1PC`, `Test1PCIsolation`, `Test1PCWithMultiDC`, `TestTxnCommitCounter` |
| f2p ×3 not flaky | Y (3× rc=1, same names) |
| f2p files ⊆ stitched impact | Y (`integration_tests/1pc_test.go` in 26-file impact; hops=3) |
| collateral | none (`TestCommitRollback` pass on buggy tree) |
| alt | **accept** (`TestOnePC` pass) |
| cheat | **reject** (`TestOnePC` still fail) |
| consumer-adapt detectable | checksum `47d34d88e3b3fd73b5a5a78cba93837c28441a71b7891cb4dbf8ff7cab248009` in `tests/test.sh` |
| Docker buggy | fail (same 4 subtests) |
| Docker fixed (restore `2pc.go`) | pass (`ok integration_tests 0.357s`) |

## Difficulty knobs

| knob | value |
|---|---|
| hops (rev-calls, consumer test ← cause) | **3**, crossing the module boundary |
| impact files | 26 (19 of them consumer) |
| cause visibility | unexported; library suite green |
| instruction locality | L0 (names f2p tests); does not name `checkOnePC` / `2pc.go` / `GetScope` |
| verifier | consumer tests only + checksum; adapting the consumer cannot score |
| agent timeout | 14400 s |

## Instruction self-check

`instruction_self_check` on the shipped `instruction.md`:

| field | hunk | full patch |
|---|---|---|
| ok | True | True |
| names_present | True | True |
| has_symptom_paraphrase | True | True |
| leaked_symbols | [] | [] |
| leaked_files | [] | [] |
| has_line_numbers / has_diff | False | False |
| diff_hunk_tokens_in_instruction | [] | [] |

## Harbor task

`experiments/harbor_nex/tasks_two_repo/client-go-onepc-scope/`

- `environment/Dockerfile`: `golang:1.23`, both modules, `GOTOOLCHAIN=local`, `go mod download` at `/app` and `/app/integration_tests` (`replace => ../`), no `.git`
- `tests/test.sh`: checksum guard then `go test -ldflags=-checklinkname=0 -count=1 -timeout 15m -run '^(TestOnePC)$' .`
- `task.toml`: agent 14400 s

Patches: `experiments/codegraph_bugs/bugs/client-go-two-repo/checkOnePC.{patch,alt,cheat}.patch`.

Package: `src/openswe_traces/synth/two_repo.py`. CLI: `uv run python scripts/two_repo.py`. Tests: `uv run pytest` (78 passed). Ruff clean on the touched synth/test files.

## Queued command

Waiter: `experiments/harbor_nex/run_two_repo_after_hard.sh` (polls `jobs/grok-xhigh-hard/result.json` for `finished_at`, 2 min interval, 6 h fallback to `jobs/grok-xhigh`). `grok-xhigh-hard` already has `finished_at=2026-09-18T10:40:39`, so the waiter launches immediately:

```bash
harbor run \
  --path experiments/harbor_nex/tasks_two_repo \
  --agent cursor-cli \
  --model cursor/cursor-grok-4.6-xhigh \
  --n-concurrent 2 \
  --n-attempts 1 \
  --max-retries 3 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name grok-xhigh-two-repo \
  --yes
```

`CURSOR_API_KEY` sourced from `/home/evan/Documents/eval_tasks/.env` (not printed). Job log: `experiments/harbor_nex/grok-xhigh-two-repo.log`.

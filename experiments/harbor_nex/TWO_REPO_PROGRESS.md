# TWO_REPO_PROGRESS

Started 2026-09-18. Goal: one maximally hard two-repo (library → consumer) Harbor task.

## Step 0 — inventory

- Reused `src/openswe_traces/synth/two_repo.py` and `tests/test_two_repo.py` from the prior attempt.
- Prior candidate already on disk: `experiments/codegraph_bugs/bugs/client-go-two-repo/checkOnePC.{patch,alt,cheat}.patch` (scope inversion of 1PC vs 2PC).
- Did not touch `experiments/harbor_nex/tasks_hard/` or `jobs/`.
- `HARD_TASKS.md` is not on disk.

## Step 1 — cross-index finding

Library: `experiments/codegraph_bugs/repos/client-go` (already indexed; 214 files incl. 29 under `integration_tests/`).
Consumer: `experiments/codegraph_bugs/repos/client-go/integration_tests` (re-ran `codegraph init && codegraph index`: 29 files, 853 nodes, 1,967 edges).

| config | cwd | `codegraph callers checkOnePC` | `codegraph callers Commit` includes consumer tests? | cross-module `calls` edges |
|---|---|---|---|---|
| library = parent combined | `repos/client-go` | `execute` in `2pc.go` only | **yes** (`Test1PC`, `Test1PCIsolation`, … in `integration_tests/`) | **1911** (src in `integration_tests/`, dst not) |
| consumer-only | `repos/client-go/integration_tests` | symbol not found | no (2 local methods named Commit) | **0** (6921 unresolved refs, including `suite.Run`) |
| stitched AST (fallback) | `two_repo.parse_consumer_clientgo_calls` | n/a | joins exported names | used as backup |

**Winner: the parent index** (client-go tree that contains both `go.mod` trees). Indexing only `integration_tests` cannot resolve `replace ../` into library bodies. CLI `callers Commit` from the parent already lists consumer tests. Hop path in sqlite: `checkOnePC ← execute ← Commit ← Test1PC` (**3 hops**, crosses the module boundary at `Commit`).

No extra index at a directory above client-go was needed: client-go *is* the parent of both modules.

## Step 2 — pair choice

**Pair: `tikv/client-go` (library) → `integration_tests` (consumer).**

They are genuinely coupled: consumer `go.mod` `require github.com/tikv/client-go/v2` plus `replace … => ../`. Parent index has **1911** `calls` edges from consumer files into library symbols (`Commit` 84×, `Set` 146×, …). `SetEnable1PC` is **not** called from any library unit test (only `examples/…/1pc_txn.go` and consumer tests), so a 1PC-policy mutation can leave `go test ./...` in the library green.

**Mock store / offline:** `NewTestStore` uses `testutils.NewMockTiKV` unless `-with-tikv` is set (`WITH_TIKV` unset). `Test1PCDisallowMultiRegion` and `Test1PCWithMultiDC` skip when `*withTiKV`. No PD/TiKV process required.

Did not fall back to an external bank pair.

## Step 2b — candidate list

| id | library symbol | hops (rev-calls to Test1PC) | contract | why / why not |
|---|---|---:|---|---|
| **A** | `checkOnePC` (`2pc.go`) | **3** (`checkOnePC ← execute ← Commit ← Test1PC`) | 1PC allowed only for global scope; invert `!=` / `==` | best: unexported, library tests never enable 1PC, consumer asserts `IsOnePC` |
| B | `checkAsyncCommit` | 3 via same `execute` | async-commit vs local scope | runner-up; overlap with 1PC counters |
| C | `SetEnable1PC` default | 1 (consumer calls it) | too shallow / 1 hop |
| D | `GetScope` always `""` | 2 | would also break async-commit + TSO scope library paths | collateral |

Picked **A**. Patches already on disk from the prior attempt.

## Step 3 — bug design

Invert the local-vs-global guard in `checkOnePC` (2 lines: comment + `!=` → `==`). Global+1PC now takes 2PC; local+1PC now takes 1PC. Commits still succeed (no crash). Library tests stay green because they never set `enable1PC`. Consumer `TestOnePC` asserts protocol flags/counters.

**Fix sites:** (1) restore `!= GlobalTxnScope` in the library; (2) rewrite consumer assertions (DISALLOWED; checksum). Alt: bind `scope := c.txn.GetScope()` then `scope !=`. Cheat: special-case consumer key `k1` inside the still-inverted guard.

## Step 4 — validation (in progress)

- library build: **Y**
- consumer build: **Y**
- library `go test ./...`: **Y** (fails=none; real=none)
- consumer f2p: ['TestOnePC'] (n=1); flaky=False
- collateral TestCommitRollback: **none**
- f2p files ⊆ stitched impact: **Y** (impact 26 files; hops=3)
- alt: **accept**
- cheat: **reject**

| check | result |
|---|---|
| both modules build | Y |
| library `go test ./...` | Y (0 fails; no `TestTiKVRecoveredFromDown` this run) |
| consumer f2p ≥ 1 | Y (`TestOnePC` + 4 subtests); 3× not flaky |
| f2p files ⊆ stitched impact | Y (`integration_tests/1pc_test.go` in 26-file impact) |
| collateral | none (`TestCommitRollback` pass) |
| alt | accept |
| cheat | reject |
| consumer checksum | `47d34d88…8009` matches snapshot |
| instruction self-check | ok (hunk and full patch; no leaked symbols/files) |

## Step 5 — Harbor task + Docker

Task: `experiments/harbor_nex/tasks_two_repo/client-go-onepc-scope/`
Docker: **buggy fail / fixed pass** (`OK harbor-two-repo-onepc-scope`). Log: `experiments/harbor_nex/verifier/two_repo_docker.log`.
pytest: 78 passed. ruff clean.
Waiter started after Docker proof.

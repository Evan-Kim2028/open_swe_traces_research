# harbor_nex ITER_7 — grok-xhigh adversarial loop

Date: 2026-09-18. Solver: `cursor-cli` / `cursor/cursor-grok-4.6-xhigh`. Host:
`experiments/codegraph_bugs/repos/client-go` @ `190f0cce536f835b72481f7bcd0a9448bb1e5202`.
`HARD_TASKS.md`, `TWO_REPO_TASK.md`, `ITER_1.md`, `ITER_2.md`, `ITER_4.md` were not on
disk. Prior artifacts: `RESULT.md`, `ITER_3.md`, `ITER_5.md`, `ITER_6.md` (job still
running; not touched), every `jobs/grok-xhigh*/result.json`, `src/openswe_traces/synth/*.py`,
`analytics/research/synthetic_task_difficulty_workflow.md`.

Did not touch `jobs/grok-xhigh-iter6/` (running), `tasks_iter6/`, `tasks_iter5/`,
`tasks_hard/`, or `tasks_iter3/`.

## Solver has cleared

Wall = `finished_at − started_at`. Reward 1.0 on every finished grok-xhigh trial.
Most recently **finished** job is `grok-xhigh-iter5` (2026-09-18T11:12:47, both
trials pass). `grok-xhigh-iter6` (L1/L2 locality on GetPhysical / GetTimeFromTS)
was still running when this iteration launched and is not scored here.

No legitimate fails exist, so there is no failure audit.

**Threshold estimate:** highest passed rung **5** (FAIR AMBIGUITY, L0 named tests);
lowest legitimately failed rung **none**. Symptom-locality cleared at **L0** only
(every prior scored instruction named the failing `Test*` functions).

Rule: solver PASSED everything in the most recent finished iteration at attempt 1
→ overshoot **two rungs** from 5 → **rung 7 sequence-semantics**. Locality also
steps up: Bug A **L1**, Bug B **L2**.

| task | job | rung | hops | attempts | pass | minutes |
|---|---|---:|---:|---:|---|---:|
| dailycodingproblem-go-twosumbest | grok-xhigh | 0 | 0 | 1 | yes | 1.3 |
| dailycodingproblem-go-twosumbrute | grok-xhigh | 0 | 0 | 1 | yes | 1.4 |
| dailycodingproblem-go-match | grok-xhigh | 0 | 0 | 1 | yes | 3.1 |
| client-go-newregionrequestsender | grok-xhigh | 1 | 1 | 1 | yes | 3.2 |
| client-go-newbackofferwithvars | grok-xhigh | 1 | 1 | 1 | yes | 4.0 |
| client-go-newrequest | grok-xhigh | 1 | 1 | 1 | yes | 6.5 |
| client-go-getglobalconfig | grok-xhigh | 1 | 2 | 1 | yes | 4.7 |
| client-go-iserrnotfound | grok-xhigh | 1 | 1 | 1 | yes | 2.5 |
| client-go-isfakeregionerror | grok-xhigh | 1 | 1 | 1 | yes | 4.4 |
| client-go-decodekeyv1 | grok-xhigh-hard | 2 | — | 2 | yes | 1.7 / 1.5 |
| client-go-getstoretypebymeta | grok-xhigh-hard | 2 | — | 2 | yes | 4.2 / 4.8 |
| client-go-keyspaceidcodec | grok-xhigh-hard | 2 | — | 2 | yes | 2.9 / 1.7 |
| client-go-keyspaceprefix | grok-xhigh-iter3 | 3 | 1–2 | 1 | yes | 2.7 |
| client-go-dualexpo | grok-xhigh-iter3 | 3 | 5 | 1 | yes | 2.8 |
| client-go-decodebucketkeys | grok-xhigh-iter5 | 5 | 2 | 1 | yes | 2.5 |
| client-go-gettimefromts | grok-xhigh-iter5 | 5 | 4 | 1 | yes | 3.3 |

Rung 7 = idempotency / retry / ordering across a **sequence of calls** where a
single-call test passes but the sequence test fails. Both bugs keep a real
single-call test green on the buggy tree.

Codegraph was used deeper than iter5: callee BFS from sequence-named tests
(`pick_sequence_site`), plus `codegraph_explore` dynamic-dispatch
(`Oracle.GetTimestamp` → `localOracle.GetTimestamp`) and the 4-step overwrite
path `TestOverwrite → Set → set → setValue → appendValue`.

## Bug A — `client-go-gettimestamp` (rung 7, L1)

**True cause.** `localOracle.GetTimestamp` (`oracle/oracles/local.go`): same-ms
logical increment `l.n++; return ts + l.n` → `return ts` (duplicates).

**Sequence vs single-call.** `TestLocalOracle` loops 100000 allocations and
asserts unique values (fails: ~21–43 unique vs 100000). `TestIsExpired` and
`TestLocalOracle_UntilExpired` call it once and stay green.

**Graph path** (`codegraph_explore` on the host index).

```
TestLocalOracle                          # oracles_test; 100k-call uniqueness
  -> Oracle.GetTimestamp                 # interface
       -> localOracle.GetTimestamp       # true cause (dynamic hop, local.go:72)
            -> GoTimeToTS

TestIsExpired / TestLocalOracle_UntilExpired
  -> GetTimestamp once                   # single-call; still pass
```

`hops_to_test_names(GetTimestamp, TestLocalOracle)` is `None` in sqlite
(external `oracles_test` package, no `calls` edge). Dynamic dispatch is the
missing hop. `codegraph impact GetTimestamp` → 37 files (homonyms); `local.go`
is in the set. f2p file `local_test.go` is same-directory (sparsity exception).

**NAME LEAKAGE L1.** f2p name `TestLocalOracle` does not contain `GetTimestamp`.
Failure text is testify `expected: 100000 / actual: 43` (no `GetTimestamp`, no
`local.go`). Instruction names package `oracle/oracles` plus
`go test -count=1 -timeout 15m ./oracle/oracles/...` (no `-run`, no `Test*`
names). `name_leakage` ok=true. `instruction_self_check` ok=true.

### Validation

| check | result |
|---|---|
| builds | yes |
| library `go test ./...` | green except `oracle/oracles` (`TestLocalOracle`). No `TestTiKVRecoveredFromDown` flake |
| f2p ≥ 1 | 1 (`TestLocalOracle`) |
| 3× not flaky | fail ×3 (unique counts 43 / 39 / 21, all ≪ 100000) |
| f2p files in impact | `local.go` ∈ 37-file impact; `local_test.go` same-directory, not a sqlite caller (sparsity) |
| no collateral | only `oracle/oracles` failed full `./...` |
| gold revert | restore `local.go` → pass; docker image same |
| alt accepted | `ExtractPhysical(last)==ExtractPhysical(ts)` then `ts=last+1` (MockOracle style) on buggy tree → pass |
| cheat rejected | increment only the first duplicate in a millisecond (`l.n==0`) → still fail |
| single-call-pass | `TestIsExpired` + `TestLocalOracle_UntilExpired` pass on the buggy tree |
| docker | `harbor-iter7-gettimestamp`: buggy fail, gold pass (`verifier/iter7_docker.log`) |

**Knobs.** hops 1 (dynamic interface→impl) / graph `None` (sparsity); impact 37
files (homonym fan-out); fix sites 1; decoys 0; coverage: 0 sqlite Test\* callers
of the v2-style impl within 3 hops. Locality **L1**.

**Instruction self-check.** `ok=true` (locality 1). names absent; symptom
`expected: 100000` / `actual: 43`; no leaked `GetTimestamp` / `local.go`; no
line numbers; no diff; repro `go test -count=1 -timeout 15m ./oracle/oracles/...`.
Checksum on `oracle/oracles/local_test.go`. Agent timeout 14400s. `golang:1.23`,
no `.git`, modules pre-downloaded.

## Bug B — `client-go-memsetvalue` (rung 7, L2)

**True cause.** `MemDB.setValue` (`internal/unionstore/memdb.go`): same-length
in-place overwrite `copy(oldVal, value); return` → `return` (first write wins).
Documented MemBuffer contract: “The latter writes overwrite the earlier writes.”

**Sequence vs single-call.** `TestOverwrite` fills 10000 keys then writes a
same-length new payload on every 3rd key (fails: expected `i*10`, got `i`).
`TestIterator` fills each key once and stays green. Same-package `TestRandom`
also fails (random overwrites); not named in the instruction; included only as
package-level collateral of the same bug.

**Graph path** (`codegraph_explore`).

```
TestOverwrite                            # memdb_test.go:315
  -> Set                                 # hop 1 (memdb.go:264)
       -> set                            # hop 2 (memdb.go:322)
            -> setValue                  # hop 3  true cause (memdb.go:373)
                 -> appendValue          # hop 4 (only on first write / length change)
```

`codegraph impact setValue` → 17 files including `memdb_test.go`.
`hops_to_test_names(setValue, {TestOverwrite})` = 3.
`shortest_caller_path` = `[setValue, set, Set, TestOverwrite]`.

**NAME LEAKAGE L2.** f2p name `TestOverwrite` does not contain `setValue`.
Failure text is testify `expected: 0xc / actual: 0x78` (no `setValue`, no
`memdb.go`). Instruction is a behavior report plus
`go test -count=1 -timeout 15m ./internal/unionstore/...` (no `-run`, no `Test*`
names). `name_leakage` ok=true. `instruction_self_check` ok=true.

### Validation

| check | result |
|---|---|
| builds | yes |
| library `go test ./...` | green except `internal/unionstore` (`TestOverwrite`, `TestRandom`). No `TestTiKVRecoveredFromDown` flake |
| f2p ≥ 1 | 1 (`TestOverwrite`) |
| 3× not flaky | fail ×3 |
| f2p files in impact | `memdb_test.go` ∈ 17-file impact |
| no collateral | only `internal/unionstore` failed full `./...` |
| gold revert | restore `memdb.go` → pass; docker image same |
| alt accepted | byte-loop `oldVal[i] = value[i]` on buggy tree → pass |
| cheat rejected | `copy` only when `len(value)==1` (payloads are 4 bytes) → still fail |
| single-call-pass | `TestIterator` (one write per key) passes on the buggy tree |
| docker | `harbor-iter7-memsetvalue`: buggy fail, gold pass (`verifier/iter7_docker.log`) |

**Knobs.** hops 3 (test→Set→set→setValue) / 4 with `appendValue`; impact 17 files;
fix sites 1; decoys 0; coverage: `setValue` has 1 production caller (`set`).
Locality **L2**.

**Instruction self-check.** `ok=true` (locality 2). names absent; symptoms
`Not equal` / `expected: 0xc` / `actual: 0x78`; no leaked `setValue` / `Set` /
`memdb.go`; no line numbers; no diff; repro
`go test -count=1 -timeout 15m ./internal/unionstore/...`.
Checksum on `internal/unionstore/memdb_test.go`. Agent timeout 14400s.

## Reusable construction

`src/openswe_traces/synth/difficulty.py` (parameterized; no injection):

- `pick_contract_drift_site(index, min_hops, n)`
- `pick_two_site_pair(index, n)`
- `pick_decoy(index, path, n)`
- `find_guard_tests(index, symbol, max_hops)`
- `find_sparse_branches(coverprofile, max_pct)`
- `pick_sequence_site(index, n)` — callee BFS from sequence-named tests
  (`SEQUENCE_TEST_RE`: Overwrite, Sequence, LocalOracle uniqueness, retry, …);
  records `sequence_tests` vs `single_call_tests`
- `find_sequence_tests(index, symbol)` / `is_sequence_test_name`
- `name_leakage(f2p, failure_text, changed_symbols, changed_files)`
- `design_for_rung(repo, rung, hops, sites, decoys, index)` now emits `sequence`
  at rung 7
- CLI: `openswe-synth --repo <path> --rung 7 --hops 4 --sites 1 --decoys 0 --locality 1`

Fixture: `Counter.Next` + `TestNextOnce` / `TestNextSequence` on
`experiments/codegraph_bugs/fixture_host`.
Tests: `tests/test_synth_difficulty.py`.
`uv run pytest tests/` — 78 passed.
`uv run ruff check src/openswe_traces/synth/difficulty.py tests/test_synth_difficulty.py` — clean.

## Harbor launch (detached, not waited)

Job dir `experiments/harbor_nex/jobs/grok-xhigh-iter7/` (pid 389331). Trials
`client-go-gettimestamp` and `client-go-memsetvalue` started concurrently.

```bash
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
cd /home/evan/Documents/open_swe_traces_research
setsid nohup harbor run \
  --path experiments/harbor_nex/tasks_iter7 \
  --agent cursor-cli \
  --model cursor/cursor-grok-4.6-xhigh \
  --n-concurrent 2 \
  --n-attempts 1 \
  --max-retries 3 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name grok-xhigh-iter7 \
  --yes \
  > experiments/harbor_nex/grok-xhigh-iter7.log 2>&1 < /dev/null &
```

`CURSOR_API_KEY` sourced from `/home/evan/Documents/eval_tasks/.env` (not printed, not committed).

Patches: `experiments/codegraph_bugs/bugs/client-go/{GetTimestamp,MemSetValue}.{patch,alt,cheat}.patch`.
Tasks: `experiments/harbor_nex/tasks_iter7/{client-go-gettimestamp,client-go-memsetvalue}/`.

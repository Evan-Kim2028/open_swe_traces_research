# harbor_nex ITER_4 — grok-xhigh adversarial loop

Date: 2026-09-18. Solver: `cursor-cli` / `cursor/cursor-grok-4.6-xhigh`. Host:
`experiments/codegraph_bugs/repos/client-go` @ `190f0cce536f835b72481f7bcd0a9448bb1e5202`.

Prior artifacts: `RESULT.md`, `ITER_3.md`, `jobs/grok-xhigh/result.json`,
`jobs/grok-xhigh-hard/result.json`, `jobs/grok-xhigh-iter3/result.json`,
`src/openswe_traces/synth/*.py`, `analytics/research/synthetic_task_difficulty_workflow.md`.
`HARD_TASKS.md`, `TWO_REPO_TASK.md`, `ITER_1.md`, `ITER_2.md` were not on disk.

## Solver has cleared (from job results)

Wall = `finished_at − started_at`. Reward 1.0 on every scored trial. No legitimate fail exists.

| task | job | rung | hops | attempts | pass | minutes |
|---|---|---:|---:|---:|---|---:|
| client-go-getglobalconfig | grok-xhigh | 1 | 2 | 1 | yes | 4.7 |
| client-go-newbackofferwithvars | grok-xhigh | 1 | 1 | 1 | yes | 4.0 |
| client-go-newregionrequestsender | grok-xhigh | 1 | 1 | 1 | yes | 3.2 |
| client-go-newrequest | grok-xhigh | 1 | 1 | 1 | yes | 6.5 |
| client-go-iserrnotfound | grok-xhigh | 1 | 1 | 1 | yes | 2.5 |
| client-go-isfakeregionerror | grok-xhigh | 1 | 1 | 1 | yes | 4.4 |
| dailycodingproblem-go-twosumbest | grok-xhigh | 0 | 0 | 1 | yes | 1.3 |
| dailycodingproblem-go-match | grok-xhigh | 0 | 0 | 1 | yes | 3.1 |
| dailycodingproblem-go-twosumbrute | grok-xhigh | 0 | 0 | 1 | yes | 1.4 |
| client-go-decodekeyv1 | grok-xhigh-hard | 1 | 1 | 2 | yes | 1.6 / 1.5 |
| client-go-getstoretypebymeta | grok-xhigh-hard | 1 | 1 | 2 | yes | 4.2 / 4.8 |
| client-go-keyspaceidcodec | grok-xhigh-hard | 1 | ~2 | 2 | yes | 2.9 / 1.7 |
| client-go-dualexpo | grok-xhigh-iter3 | 3 | 5 | 1 | yes | 2.8 |
| client-go-keyspaceprefix | grok-xhigh-iter3 | 3 | 1–2 | 1 | yes | 2.6 |

**Running IRT estimate.** Highest rung/knob passed: **rung 3** (three-site `KeyspacePrefix`; two-site no-import `DualExpo`, hops 5). Lowest legitimately failed: **none**. Most recent finished job is `grok-xhigh-hard` (10:40) and `grok-xhigh-iter3` (10:36), both mean reward 1.0. Rule: passed everything → overshoot **two rungs** from 3 → **rung 5** (fair ambiguity). No failure audit: there is no (a)/(b)/(c)/(d) miss to classify.

## Rung chosen and why

Rung 5 = two candidate causes on the test→symptom path, both plausible from the symptom, only one consistent with ALL tests. Decoys are unmodified pre-existing functions; “fixing” a decoy is a change a real engineer might make and is caught by an existing repo test. No planted comments/TODOs/names/error strings. Two bugs, different constructions.

Reusable knobs in `src/openswe_traces/synth/difficulty.py`: `pick_contract_drift_site`, `pick_two_site_pair`, `pick_decoy`, `pick_fair_ambiguity`, `find_guard_tests`, `find_sparse_branches`, `design_for_rung`. CLI: `openswe-synth --repo <path> --rung 5 --hops 4 --sites 2 --decoys 1`.

## Bug A — `client-go-extractphysical` (rung 5, TSO split)

**Cause.** `ExtractPhysical` (`oracle/oracle.go`): `ts >> physicalShiftBits` became `ts >> (physicalShiftBits - 1)` (physical ms doubled).

**Decoy (unmodified).** `GetTimeFromTS` — wraps `ExtractPhysical` then `time.Unix`. Symptom (stale time in year 2083; lock “not expired”) is exactly what `GetTimeFromTS` returns. `IsExpired` (local) calls `GetTimeFromTS`; `UntilExpired` (local and PD) calls `ExtractPhysical` directly and mixes it with `GetPhysical`.

**Why an engineer inspects the decoy.** Path `ExtractPhysical → GetTimeFromTS → TestPdOracle_GetStaleTimestamp` / `TestIsExpired`. The failing assertion is on a `time.Time` produced by `GetTimeFromTS`. A competent engineer tries the Unix conversion first.

**Decoy-fix-fails.** Inlining the correct `ts >> 18` inside `GetTimeFromTS` (`ExtractPhysical.decoy.patch`) makes `TestIsExpired` and `TestPdOracle_GetStaleTimestamp` pass, but **existing** `TestLocalOracle_UntilExpired` (expected -4 / 4, got ~1.79e12) and `TestPDOracle_UntilExpired` (expected 25, got 35) still fail. Those tests never call `GetTimeFromTS`.

**Graph path** (codegraph reverse-`calls`):

```
ExtractPhysical --1--> GetTimeFromTS --1--> TestPdOracle_GetStaleTimestamp   hops=2
ExtractPhysical --1--> UntilExpired  --1--> TestLocalOracle_UntilExpired     hops=2
ExtractPhysical --1--> IsExpired     --1--> TestPDOracle_UntilExpired        hops=2
```

Impact: 26 files (`oracle/oracle.go`, `oracle/oracles/{local,pd,mock}.go`, latch, 2pc/prewrite, mocktikv, integration_tests). f2p files `oracle/oracles/local_test.go` and `pd_test.go` are same-package callers of `IsExpired` / `UntilExpired` / `GetTimeFromTS` (graph lists production files; tests sit on those edges).

### Validation

| check | result |
|---|---|
| builds | yes |
| library tests | `./oracle/...` + `./internal/latch/` + `./txnkv/transaction/ -short` green except the 5 f2p. No `TestTiKVRecoveredFromDown` |
| f2p ≥ 1 | 5 (`TestIsExpired`, `TestLocalOracle_UntilExpired`, `TestPDOracle_UntilExpired`, `TestPdOracle_GetStaleTimestamp`, `TestNonFutureStaleTSO`) |
| 3× not flaky | same 5 names ×3 |
| f2p files in impact | `local_test.go` / `pd_test.go` same-package with `GetTimeFromTS`/`UntilExpired` (24–26 file impact) |
| no collateral | latch + txnkv `-short` green |
| gold revert | restore `oracle/oracle.go` → f2p pass |
| alt accepted | `int64(ts / (uint64(1) << physicalShiftBits))` on buggy tree → pass |
| cheat rejected | special-case `ts==0` still `>> 17` → all 5 still fail |
| decoy-fix-fails | `GetTimeFromTS` inlines `>> 18` → UntilExpired tests still fail |
| docker | `harbor-iter4-extractphysical`: buggy fail, gold pass (`verifier/iter4_extractphysical_docker.log`) |

**Knobs.** hops 2; impact 26 files; fix sites 1; decoys 1 (`GetTimeFromTS`); coverage: UntilExpired is exact integer, GetTimeFromTS is duration slack (2s) — both fire.

**Instruction self-check.** `ok=true`. Names present; expected/got (`Should be true`, `expected: -4 actual: 1789744684008`, year 2083 vs 2026); no leaked `ExtractPhysical`/`GetTimeFromTS`; no `.go` filenames/line numbers/diff; checksums on `local_test.go` and `pd_test.go`. Agent timeout 14400s. `golang:1.23`, no `.git`, modules pre-downloaded.

## Bug B — `client-go-intervalcontains` (rung 5, half-open range)

**Cause.** `(*Region).Contains` (`internal/locate/region_cache.go`): empty key now returns `false` instead of delegating to `contains` (`startKey <= key < endKey`). Empty is the minimum key, so it belongs to a region whose start is empty.

**Decoy (unmodified).** `(*Region).ContainsByEnd` — overlapping membership helper with its **own** empty-key branch (`if len(key)==0 { return len(r.EndKey())==0 }`) and comment “Only a region's right bound expands to inf contains the point at inf”. Same file, same call-path neighborhood (`SearchByKey` chooses `Contains` vs `ContainsByEnd` via `isEndKey`).

**Why an engineer inspects the decoy.** Symptom is empty-key membership (`Should be true` on `TestContains` cases with `[]byte{}`). `ContainsByEnd` is the only nearby function that already special-cases empty keys. A competent engineer copies that branch.

**Decoy-fix-fails.** Making `ContainsByEnd` return `false` for empty keys (`IntervalContains.decoy.patch`) fails **existing** `TestRegionCache/TestContainsByEnd` (`ContainsByEnd(nil,nil,[])` must be true). `TestContains` still fails (cause untouched).

**Graph path:**

```
Contains --1--> TestContains                         hops=1
contains --1--> Contains --1--> TestContains         hops=2
ContainsByEnd --1--> TestContainsByEnd               hops=1
SearchByKey calls Contains and ContainsByEnd (sibling callees)
```

Impact: `Contains` 16 files including `region_cache_test.go`, `sorted_btree.go`. f2p file `internal/locate/region_cache_test.go` ∈ impact.

### Validation

| check | result |
|---|---|
| builds | yes |
| library tests | `./internal/locate/ -skip TestRegionCache` green. Suite `TestRegionCache` finishes ~36s with only `TestContains` failing. No `TestTiKVRecoveredFromDown` |
| f2p ≥ 1 | `TestRegionCache` / `TestRegionCache/TestContains` (two `Should be true` on empty-key cases) |
| 3× not flaky | fail ×3 |
| f2p files in impact | `region_cache_test.go` ∈ Contains impact |
| no collateral | locate minus `TestRegionCache` green; suite has no other FAIL lines |
| gold revert | restore `region_cache.go` → pass |
| alt accepted | inlined `key >= start && (end empty \|\| key < end)` → pass |
| cheat rejected | copy `ContainsByEnd` empty-key rule into `Contains` (`len(end)==0`) → `Contains(nil,[10],[])` still false, suite still fails |
| decoy-fix-fails | `ContainsByEnd` empty → false fails `TestContainsByEnd` |
| docker | `harbor-iter4-intervalcontains`: buggy fail, gold pass (`verifier/iter4_intervalcontains_docker.log`) |

**Knobs.** hops 1 (test→Contains) / 2 (via `contains`); impact 16 files; fix sites 1; decoys 1 (`ContainsByEnd`); coverage: `TestContains` hits empty-key, `TestContainsByEnd` is the hidden-from-instruction guard included because Harbor runs `^(TestRegionCache)$`.

**Instruction self-check.** `ok=true`. Name `TestRegionCache` present; symptom `Should be true`; no leaked `Contains`/`ContainsByEnd`; no filenames/line numbers/diff; checksum on `region_cache_test.go`. Agent timeout 14400s. `golang:1.23`, no `.git`, modules pre-downloaded.

## Harbor launch (detached, not waited)

Job dir `experiments/harbor_nex/jobs/grok-xhigh-iter4/` (pid 440103).

```bash
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
cd /home/evan/Documents/open_swe_traces_research
setsid nohup harbor run \
  --path experiments/harbor_nex/tasks_iter4 \
  --agent cursor-cli \
  --model cursor/cursor-grok-4.6-xhigh \
  --n-concurrent 2 \
  --n-attempts 1 \
  --max-retries 3 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name grok-xhigh-iter4 \
  --yes \
  > experiments/harbor_nex/grok-xhigh-iter4.log 2>&1 < /dev/null &
```

`CURSOR_API_KEY` sourced from `/home/evan/Documents/eval_tasks/.env` (not printed, not committed).

Patches: `experiments/codegraph_bugs/bugs/client-go/{ExtractPhysical,IntervalContains}.{patch,alt,cheat,decoy}.patch`.
Tasks: `experiments/harbor_nex/tasks_iter4/{client-go-extractphysical,client-go-intervalcontains}/`.

`uv run pytest tests/test_synth_difficulty.py tests/test_harbor_tasks.py tests/test_codegraph_bugs.py` — 22 passed.
`uv run ruff check src/openswe_traces/synth/difficulty.py tests/test_synth_difficulty.py` — clean.

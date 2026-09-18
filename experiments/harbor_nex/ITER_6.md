# harbor_nex ITER_6 — grok-xhigh adversarial loop

Date: 2026-09-18. Solver: `cursor-cli` / `cursor/cursor-grok-4.6-xhigh`. Host:
`experiments/codegraph_bugs/repos/client-go` @ `190f0cce536f835b72481f7bcd0a9448bb1e5202`.
Prior artifacts used: `RESULT.md`, `ITER_3.md`, `jobs/grok-xhigh/`, `jobs/grok-xhigh-hard/`,
`jobs/grok-xhigh-iter3/` (every trial `result.json`), `src/openswe_traces/synth/*.py`,
`analytics/research/synthetic_task_difficulty_workflow.md`. `HARD_TASKS.md`,
`TWO_REPO_TASK.md`, `ITER_1.md`, `ITER_2.md`, `ITER_4.md`, `ITER_5.md` were not on disk.

## Solver has cleared (attempt 1, reward 1.0)

Wall = `finished_at − started_at` on each trial `result.json`. Hard job stored two
attempts per task (both 1.0); minutes below are the first trial of each.

| task | job | rung | hops | attempts | pass | minutes |
|---|---|---:|---:|---:|---|---:|
| dailycodingproblem-go-twosumbest | grok-xhigh | 0 | 0 | 1 | yes | 1.3 |
| dailycodingproblem-go-twosumbrute | grok-xhigh | 0 | 0 | 1 | yes | 1.4 |
| dailycodingproblem-go-match | grok-xhigh | 0 | 0 | 1 | yes | 3.1 |
| client-go-newbackofferwithvars | grok-xhigh | 1 | 1 | 1 | yes | 4.0 |
| client-go-newregionrequestsender | grok-xhigh | 1 | 1 | 1 | yes | 3.2 |
| client-go-newrequest | grok-xhigh | 1 | 1 | 1 | yes | 6.5 |
| client-go-iserrnotfound | grok-xhigh | 1 | 1 | 1 | yes | 2.5 |
| client-go-isfakeregionerror | grok-xhigh | 1 | 1 | 1 | yes | 4.4 |
| client-go-getglobalconfig | grok-xhigh | 1 | 2 | 1 | yes | 4.7 |
| client-go-decodekeyv1 | grok-xhigh-hard | ~2 | 1 | 1 (×2 stored) | yes | 1.7 |
| client-go-getstoretypebymeta | grok-xhigh-hard | ~2 | 1 | 1 (×2 stored) | yes | 4.2 |
| client-go-keyspaceidcodec | grok-xhigh-hard | ~2 | 1 | 1 (×2 stored) | yes | 2.9 |
| client-go-keyspaceprefix | grok-xhigh-iter3 | 3 | 1–2 | 1 | yes | 2.7 |
| client-go-dualexpo | grok-xhigh-iter3 | 3 | 5 | 1 | yes | 2.8 |

No grok-xhigh failure exists. Two-repo (rung 4) never ran (`tasks_two_repo` was
missing when the wait script fired). Every prior instruction was **L0** (named
`Test*` functions whose names point at the changed symbol).

**Running threshold estimate.** Highest rung/knob passed: **3** (three-site
KeyspacePrefix; DualExpo two-site / no import edge, 5 hops). Lowest
legitimately failed: **none** (open). Most recent finished iteration
(`grok-xhigh-iter3`) passed everything at attempt 1 → overshoot **two rungs**
→ **rung 5** (FAIR AMBIGUITY). Symptom-locality also steps up: Bug A **L1**,
Bug B **L2**.

## Bug A — `client-go-getphysical` (rung 5, L1)

**Cause.** `GetPhysical` (`oracle/oracle.go`): `UnixNano()/Millisecond` →
`UnixNano()/Microsecond` (physical now is 1000× too large). Single site.

**Decoy (unmodified).** `ExtractPhysical`, same file, **same line** as the
cause in the unique production caller:

```
TestLocalOracle_UntilExpired
  --1-->  localOracle.UntilExpired            # oracle/oracles/local.go
  --2-->  ExtractPhysical(lockTS) + TTL - GetPhysical(now)
```

`codegraph_explore` on the host index: `GetPhysical` has **one** production
caller (`local.go:128`). `UntilExpired` (interface) dynamically dispatches to
that impl. `ExtractPhysical` is the inverse conversion sitting next to
`GetPhysical` on the test→symptom path; a competent engineer seeing remaining
lock lifetime off by ~10¹⁵ ms inspects the lock-TS decode first. Decoy is not
modified, not renamed, not mentioned in the instruction.

**Decoy-fix-fails.** Compensating `ExtractPhysical` with `* 1000` (engineer
trying to put lock physical into microseconds to match `GetPhysical`):
`TestLocalOracle_UntilExpired` still fails (TTL still in ms) **and**
pre-existing `TestPDOracle_UntilExpired` fails (`expected 25, actual 10015`).
`pdOracle.UntilExpired` uses only `ExtractPhysical`, so that test is green on
the injected bug and is the hidden guard in `tests/test.sh`.

**Alt.** `return t.UnixMilli()` on the buggy tree → f2p+guard pass.
**Cheat.** Millisecond divide only when `t.Unix() < 1_000_000_000` (pre-2001);
tests use `time.Now()` → still fail.

**NAME LEAKAGE L1.** f2p name `TestLocalOracle_UntilExpired` does not contain
`GetPhysical`. Failure text is testify `expected: -4 / actual: ~-1.79e15`
(no `GetPhysical`, no `oracle.go`). Instruction names package `oracle/oracles`
plus `go test -count=1 -timeout 15m ./oracle/oracles/...` (no `-run`, no
`Test*` names). `name_leakage` ok=true. `instruction_self_check` ok=true.

### Validation

| check | result |
|---|---|
| builds | yes |
| library `go test ./...` | green except `TestLocalOracle_UntilExpired`. No `TestTiKVRecoveredFromDown` flake (integration_tests nested module not in `./...`) |
| f2p ≥ 1 | 1 (`TestLocalOracle_UntilExpired`) |
| 3× not flaky | fail ×3 |
| f2p files in impact | `local.go` ∈ GetPhysical impact (unique caller); f2p lives in same-package `local_test.go`. Graph does not list that `*_test.go` as a GetPhysical caller (sparsity) |
| no collateral | only that one test failed full `./...` |
| gold revert | restore `oracle.go` → f2p+guard pass; docker image same |
| alt accepted | `UnixMilli()` → pass |
| cheat rejected | pre-2001 special-case → still fail |
| decoy-fix-fails | `ExtractPhysical * 1000` → `TestPDOracle_UntilExpired` fail (existing test) |
| docker | `harbor-iter6-getphysical`: buggy fail, gold pass (`verifier/iter6_docker.log`) |

**Knobs.** hops 2 (test→UntilExpired→GetPhysical); impact 6 files (2 production:
`oracle.go`, `local.go`); fix sites 1; decoys 1 (`ExtractPhysical`); coverage:
`GetPhysical` has no in-package unit test of its own — only via `UntilExpired`.
Locality **L1**. Guard `TestPDOracle_UntilExpired` in `test.sh`, not in the
instruction.

## Bug B — `client-go-gettimefromts` (rung 5, L2)

**Cause.** `GetTimeFromTS` (`oracle/oracle.go`): `time.Unix(ms/1e3, (ms%1e3)*1e6)`
→ `time.Unix(0, ms)` (physical milliseconds treated as nanoseconds → Unix epoch).

**Decoy (unmodified).** `ExtractPhysical`, callee of the cause on the same path:

```
TestIsExpired
  --1-->  localOracle.IsExpired
  --2-->  GetTimeFromTS(lockTS)
  --3-->  ExtractPhysical(ts)     # decoy, unmodified
TestPdOracle_GetStaleTimestamp
  --1-->  GetStaleTimestamp / getStaleTimestamp
  --2-->  GetTimeFromTS           # also asserted directly in the test
```

`codegraph impact GetTimeFromTS` returns 20+ files including `pd_test.go`,
`local.go`, `internal/latch/scheduler.go` (`tsoSub`). Same inverse-pair decoy
as Bug A, different injection: an engineer seeing “decoded time is 1969”
reasonably guesses the physical extract returned the wrong unit and multiplies
by 1e6.

**Decoy-fix-fails.** `ExtractPhysical * 1e6` **greens** both f2p tests
(`Unix(0, ms*1e6)` is the correct conversion) and fails pre-existing
`TestLocalOracle_UntilExpired` and `TestPDOracle_UntilExpired` (physical
parts become nanoseconds; expected −4 / 25 vs ~10¹⁸ / 10000015). Those two
are hidden guards in `test.sh`.

**Alt.** `return time.UnixMilli(ms)` → pass. **Cheat.** restore the correct
formula only when `ms < 1000`; live timestamps fail `TestIsExpired`.

**NAME LEAKAGE L2.** f2p names `TestIsExpired`, `TestPdOracle_GetStaleTimestamp`
do not contain `GetTimeFromTS`. Failure text: `Should be false` and
epoch-vs-2026 `Max difference … 1969 … allowed is 2s` (no symbol, no
`oracle.go`). Instruction is a behavior report plus
`go test -count=1 -timeout 15m ./oracle/oracles/...`. `name_leakage` ok=true.
`instruction_self_check` ok=true.

### Validation

| check | result |
|---|---|
| builds | yes |
| library `go test ./...` | fails `TestIsExpired`, `TestPdOracle_GetStaleTimestamp`, and same-cause `TestRecycle` (`internal/latch`, `tsoSub` → `GetTimeFromTS`). No `TestTiKVRecoveredFromDown` flake |
| f2p ≥ 1 | 2 (`TestIsExpired`, `TestPdOracle_GetStaleTimestamp`) |
| 3× not flaky | fail ×3 |
| f2p files in impact | `pd_test.go` ∈ GetTimeFromTS impact; `local.go` ∈ impact (TestIsExpired is same-package). `latch_test.go` also in impact (same-cause `TestRecycle`, not in test.sh) |
| no collateral | extra fail is same-cause (`GetTimeFromTS` in latch), not an unrelated feature. Gold restores it |
| gold revert | restore `oracle.go` → f2p+guard pass; docker image same |
| alt accepted | `UnixMilli(ms)` → pass |
| cheat rejected | `ms < 1000` special-case → still fail |
| decoy-fix-fails | `ExtractPhysical * 1e6` greens f2p and fails both UntilExpired guards |
| docker | `harbor-iter6-gettimefromts`: buggy fail, gold pass (`verifier/iter6_docker.log`) |

**Knobs.** hops 2–3; impact 20+ files; fix sites 1; decoys 1 (`ExtractPhysical`);
coverage: f2p names deliberately do not contain the changed symbol (L2).
Locality **L2**. Guards in `test.sh` only.

## Synth knobs (reusable)

`src/openswe_traces/synth/difficulty.py`: `FairAmbiguity`, `INVERSE_PAIRS`,
`pick_fair_ambiguity`, `pick_implicit_invariant`, `name_leakage`, callee-of-cause
decoys, `design_for_rung` emits `fair_ambiguity`. `openswe-synth --repo <path>
--rung 5 --hops 4 --sites 2 --decoys 1` prints the design JSON; `--patch`/`--out`
/`--locality`/`--guard` build a Harbor task.

`harbor_tasks.issue_from_failures` locality L0/L1/L2; `build_task(..., locality=,
guard_tests=)` puts guards in `test.sh` only. Fixture tests in
`tests/test_synth_difficulty.py` / `tests/test_harbor_tasks.py`.

## Harbor launch (detached, not waited)

Job dir `experiments/harbor_nex/jobs/grok-xhigh-iter6/` (pid 298601). Trials
`client-go-getphysical` and `client-go-gettimefromts` started concurrently. Log
`experiments/harbor_nex/grok-xhigh-iter6.log`.

```bash
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
cd /home/evan/Documents/open_swe_traces_research
setsid nohup harbor run \
  --path experiments/harbor_nex/tasks_iter6 \
  --agent cursor-cli \
  --model cursor/cursor-grok-4.6-xhigh \
  --n-concurrent 2 \
  --n-attempts 1 \
  --max-retries 3 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name grok-xhigh-iter6 \
  --yes \
  > experiments/harbor_nex/grok-xhigh-iter6.log 2>&1 < /dev/null &
```

`CURSOR_API_KEY` sourced from `/home/evan/Documents/eval_tasks/.env` (not printed, not committed).

Patches: `experiments/codegraph_bugs/bugs/client-go/{GetPhysical,GetTimeFromTS}.{patch,alt,cheat}.patch`.
Tasks: `experiments/harbor_nex/tasks_iter6/{client-go-getphysical,client-go-gettimefromts}/`.

`uv run pytest tests/test_harbor_tasks.py tests/test_synth_difficulty.py` — 17 passed.
`uv run ruff check src/openswe_traces/synth/harbor_tasks.py src/openswe_traces/synth/difficulty.py tests/test_harbor_tasks.py tests/test_synth_difficulty.py` — clean.

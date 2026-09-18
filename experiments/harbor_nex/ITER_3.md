# harbor_nex ITER_3 — grok-xhigh adversarial loop

Date: 2026-09-18. Solver: `cursor-cli` / `cursor/cursor-grok-4.6-xhigh`. Host:
`experiments/codegraph_bugs/repos/client-go` @ `190f0cce536f835b72481f7bcd0a9448bb1e5202`.
`HARD_TASKS.md`, `TWO_REPO_TASK.md`, `ITER_1.md`, `ITER_2.md` were not on disk when this
iteration started (sibling agents still writing). Prior artifacts used: `RESULT.md`,
`jobs/grok-xhigh/result.json` (and every trial `result.json` under that job),
`src/openswe_traces/synth/*.py`, `analytics/research/synthetic_task_difficulty_workflow.md`.

## Solver has cleared (attempt 1, reward 1.0)

From `experiments/harbor_nex/jobs/grok-xhigh/` (finished 2026-09-18T10:10:28). All 9
tasks passed on the only attempt (`n-attempts 1`). Wall = `finished_at − started_at`.

| task | rung | hops | attempts | pass | minutes |
|---|---:|---:|---:|---|---:|
| client-go-getglobalconfig | 1 | 2 | 1 | yes | 4.7 |
| client-go-newbackofferwithvars | 1 | 1 | 1 | yes | 4.0 |
| client-go-newregionrequestsender | 1 | 1 | 1 | yes | 3.2 |
| dailycodingproblem-go-twosumbest | 0 | 0 | 1 | yes | 1.3 |
| client-go-newrequest | 1 | 1 | 1 | yes | 6.5 |
| dailycodingproblem-go-match | 0 | 0 | 1 | yes | 3.1 |
| dailycodingproblem-go-twosumbrute | 0 | 0 | 1 | yes | 1.4 |
| client-go-iserrnotfound | 1 | 1 | 1 | yes | 2.5 |
| client-go-isfakeregionerror | 1 | 1 | 1 | yes | 4.4 |

No grok-xhigh failure exists. Lowest uncleared ladder rung is **1**. Rule: if the
solver passed everything so far at attempt 1, go up **two** rungs → **rung 3**.

Rung 3 = three-site coordinated **or** two-site in different packages with no direct
import edge, found through a shared type via `codegraph impact`. This iteration builds
one of each.

## Bug A — `client-go-keyspaceprefix` (rung 3, three-site)

**Sites.** `ParseKeyspaceID` (`internal/apicodec/codec.go`), `getIDByte` and the
exclusive-end computation inside `NewCodecV2` (`internal/apicodec/codec_v2.go`).
Injection: swap `buf[1]↔buf[3]` when parsing; reverse the three ID bytes in
`getIDByte`; reverse-increment `endKey` so the exclusive end stays consistent with
the reversed prefix. Encode, parse, and range-end must all agree; any 1- or 2-site
restore still fails f2p.

**Graph path** (`codegraph_explore` on the host index).

```
TestParseKeyspaceID  --1 hop-->  ParseKeyspaceID
TestCodecV2.SetupSuite --1 hop--> NewCodecV2 --1 hop--> getIDByte
NewCodecV2 also writes prefix/endKey used by EncodeKey / EncodeRange / DecodeBucketKeys
  (TestCodecV2/TestEncodeRequest, TestEncodeV2KeyRanges, TestNewCodecV2, TestDecodeBucketKeys)
```

Impact (caller BFS): `ParseKeyspaceID` 2 files (`codec.go`, `codec_test.go`);
`getIDByte` 3 files (`codec_v2.go`, `codec_v2_test.go`, `internal/locate/pd_codec.go`);
`NewCodecV2` 4 files (adds `tikv/region.go`). f2p files `codec_test.go` and
`codec_v2_test.go` sit inside those sets.

**Instruction context (one line, no symbol/file/mechanism):** keyspace identifiers
occupy three bytes after a mode prefix; encoding, parsing, and the exclusive end of
that range must agree on byte layout.

### Validation

| check | result |
|---|---|
| builds | yes |
| library `go test ./...` | green except `internal/apicodec` f2p (`TestParseKeyspaceID`, `TestCodecV2`). No `TestTiKVRecoveredFromDown` flake this run |
| f2p ≥ 1 | 2 (`TestParseKeyspaceID`, `TestCodecV2`) |
| 3× not flaky | fail ×3 (`-count=3`) |
| f2p files in impact | `codec_test.go` ∈ ParseKeyspaceID; `codec_v2_test.go` ∈ getIDByte/NewCodecV2 |
| no collateral | only `internal/apicodec` failed full `./...` |
| gold revert | restore both files → f2p pass; docker image same |
| alt accepted | prepend-`0` parse + `b[1],b[2],b[3]` ID bytes + byte-wise `endKey++` on buggy tree → pass |
| cheat rejected | special-case `0x010203` in `ParseKeyspaceID` only → `TestCodecV2` still fails |
| single-site-fix-fails | parse-only fail; getIDByte-only fail; endKey-only fail; both v2 sites without parse → `TestParseKeyspaceID` fail |
| docker | `harbor-iter3-keyspaceprefix`: buggy fail, gold pass (`verifier/iter3_keyspaceprefix_docker.log`) |

**Knobs.** hops 1 (parse) / 2 (getIDByte via NewCodecV2); impact 2–4 files per site
(union 6); fix sites 3, same package; decoys 0; coverage: `TestParseKeyspaceID` is
direct, `TestCodecV2` is a suite hitting prefix/endKey through encode/decode.

**Instruction self-check.** `ok=true`. names present; expected/got paraphrase (`Not equal`);
no leaked symbols (`getIDByte`, `ParseKeyspaceID`, `NewCodecV2`); no `.go` filenames;
no line numbers; no diff; one-command repro
`go test -count=1 -timeout 15m -run '^(TestParseKeyspaceID|TestCodecV2)$' ./internal/apicodec/...`.
Checksums on both `*_test.go`. Agent timeout 14400s. `golang:1.23`, no `.git`, modules
pre-downloaded.

## Bug B — `client-go-dualexpo` (rung 3, two-site / no import edge)

**Sites.** `expo` in `config/retry/config.go` **and** `expo` in
`internal/client/retry/config.go`. Neither package imports the other. Both define
`BackoffFnCfg` / `NewBackoffFnCfg` / `expo`. `codegraph impact expo` on
`config/retry/config.go` returns 28 files and, by name-collision on `Backoff` /
`Backoffer`, surfaces `internal/client/retry/backoff.go` — the sibling module with
no import edge. Injection: `min(cap, base·2^n + 1)` (+1 ms). That trips
`TestBackoffDeepCopy` (budget 8 ms after weight; original sleeps 2+4+8, buggy
3+5 already hits the cap on the third wait) without collateral on
`TestRegionRequestToThreeStores` (the n+1 exponent variant did).

**Graph path.**

```
TestBackoffDeepCopy
  -> NewBackofferWithVars
  -> Backoff(BoMaxDataNotReady)          # hop 1
  -> BackoffWithCfgAndMaxSleep           # hop 2
  -> createBackoffFn                     # hop 3
  -> newBackoffFn                        # hop 4
  -> expo                                # hop 5
```

`codegraph` reports **no tests within 3 hops of `newBackoffFn`** (coverage sparsity).
The f2p test lives in-package with `Backoff`, which is in expo's 28-file impact
(`config/retry/backoff.go`). Twin f2p: `internal/client/retry/backoff_test.go`
covers the unexported `expo` whose own impact is 1 file (same package).

**Instruction context:** two packages ship independent copies of the same exponential
wait schedule; a clone of a budget-limited retry object must still trip the budget
on the next wait.

### Validation

| check | result |
|---|---|
| builds | yes |
| library `go test ./...` | green except `config/retry` and `internal/client/retry` `TestBackoffDeepCopy`. No `TestTiKVRecoveredFromDown` flake |
| f2p ≥ 1 | 1 name × 2 packages (`TestBackoffDeepCopy`) |
| 3× not flaky | fail ×3 |
| f2p files in impact | `config/retry/backoff.go` ∈ expo impact (28); test file is same-package caller of `Backoff` on that path. Client `expo` impact is 1 file (`config.go`); client f2p is same-package. Graph does not list `*_test.go` as expo callers (sparsity) |
| no collateral | only those two `TestBackoffDeepCopy` failures |
| gold revert | restore both `config.go` → pass; docker image same |
| alt accepted | iterative doubling (no `Pow`, no `+1`) on both sites → pass |
| cheat rejected | correct formula only when `base==100`; `BoMaxDataNotReady` uses `base=2` → still fail |
| single-site-fix-fails | restore `config/retry` only → client test still fails; restore client only → `config/retry` still fails |
| docker | `harbor-iter3-dualexpo`: buggy fail, gold pass (`verifier/iter3_dualexpo_docker.log`) |

**Knobs.** hops 5 (test→expo); impact 28 + 1 (no import edge); fix sites 2, different
packages; decoys 0; coverage sparsity: 0 tests within 3 hops of `newBackoffFn`.

**Instruction self-check.** `ok=true`. name present; symptom `Expected nil, but got: region data not ready`;
no leaked `expo`; no files/line numbers/diff; repro
`go test -count=1 -timeout 15m -run '^(TestBackoffDeepCopy)$' ./config/retry/... ./internal/client/retry/...`.
Checksums on both `backoff_test.go`. Agent timeout 14400s.

## Harbor launch (detached, not waited)

Job dir `experiments/harbor_nex/jobs/grok-xhigh-iter3/` (pid 11936). Trials
`client-go-dualexpo` and `client-go-keyspaceprefix` started concurrently.

```bash
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
cd /home/evan/Documents/open_swe_traces_research
setsid nohup harbor run \
  --path experiments/harbor_nex/tasks_iter3 \
  --agent cursor-cli \
  --model cursor/cursor-grok-4.6-xhigh \
  --n-concurrent 2 \
  --n-attempts 1 \
  --max-retries 3 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name grok-xhigh-iter3 \
  --yes \
  > experiments/harbor_nex/grok-xhigh-iter3.log 2>&1 < /dev/null &
```

`CURSOR_API_KEY` sourced from `/home/evan/Documents/eval_tasks/.env` (not printed, not committed).

Patches: `experiments/codegraph_bugs/bugs/client-go/{KeyspacePrefix,DualExpo}.{patch,alt,cheat}.patch`.
Tasks: `experiments/harbor_nex/tasks_iter3/{client-go-keyspaceprefix,client-go-dualexpo}/`.

`uv run pytest tests/test_harbor_tasks.py` — 7 passed. `uv run ruff check src/openswe_traces/synth/harbor_tasks.py tests/test_harbor_tasks.py` — clean.

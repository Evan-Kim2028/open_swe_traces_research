# harbor_nex ITER_5 — grok-xhigh adversarial loop

Date: 2026-09-18. Solver: `cursor-cli` / `cursor/cursor-grok-4.6-xhigh`. Host:
`experiments/codegraph_bugs/repos/client-go` @ `190f0cce536f835b72481f7bcd0a9448bb1e5202`.
`HARD_TASKS.md`, `TWO_REPO_TASK.md`, `ITER_1.md`, `ITER_2.md`, `ITER_4.md` were not on
disk. Prior artifacts: `RESULT.md`, `ITER_3.md`, every `jobs/grok-xhigh*/result.json`,
`src/openswe_traces/synth/*.py`, `analytics/research/synthetic_task_difficulty_workflow.md`.

Did not touch `jobs/grok-xhigh-hard/`, `tasks_hard/`, `tasks_iter3/`, or the two-repo waiter.

## Solver has cleared

Wall = `finished_at − started_at`. Reward 1.0 on every finished grok-xhigh trial. No
legitimate fails exist, so there is no failure audit.

**Threshold estimate:** highest passed rung **3**; lowest legitimately failed rung
**none**.

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

Most recently finished job is `grok-xhigh-hard` (2026-09-18T10:40:39, all 6 trials
pass). `grok-xhigh-iter3` finished four minutes earlier, also all-pass, at **rung 3**.
IRT overshoot uses the **frontier** (highest passed), not the last job’s lower rung:
passed everything at attempt 1 through rung 3 → skip 4 (two-repo / cross-module, still
queued by `run_two_repo_after_hard.sh`) → **rung 5 FAIR AMBIGUITY**.

Rung 5 = two candidate causes on the test→symptom path, both plausible from the
symptom, only one consistent with ALL tests. Decoys are unmodified pre-existing
functions on that path; a decoy “fix” must regress an **existing** repo test. No planted
comments, TODOs, names, error strings, or instruction wording that points at the decoy.

## Bug A — `client-go-decodebucketkeys` (rung 5, fair ambiguity)

**True cause.** `codecV2.DecodeBucketKeys` (`internal/apicodec/codec_v2.go`): prefix
strip `k[len(c.prefix):]` → `k[len(c.prefix)-1:]`. Decoded bucket keys keep a leftover
mode byte (`0x92…`).

**Decoy (unmodified).** `memComparableCodec.decodeKey` (`internal/apicodec/mem_codec.go`).
`DecodeBucketKeys` always calls `c.memCodec.decodeKey(key)` **before** the prefix strip.
A leftover extra byte after unwrap is exactly what a broken `DecodeBytes` / drop-last
marker would produce, so a competent engineer inspects `decodeKey` first.

- Why inspect it: it is hop 1 on the cause path and has the same unwrap duty as the
  prefix strip.
- Decoy “fix” (return `key[:len(key)-1]` after `DecodeBytes`): existing
  `TestCodecV2/TestDecodeEpochNotMatch` fails (region start/end after mem-decode no
  longer match). `TestDecodeBucketKeys` still fails. Documented from a patched tree;
  `decodeKey` itself was never modified in the Harbor snapshot.

**Graph path** (`codegraph_explore` on the host index).

```
TestCodecV2                          # testify suite runner
  -> TestDecodeBucketKeys            # suite method; no calls-edge in sqlite
       -> DecodeBucketKeys           # true cause (codec_v2.go:976)
            -> memCodec.decodeKey    # DECOY (mem_codec.go:51)
```

Interface `DecodeBucketKeys` in `codec.go` dispatches to v1/v2; `pd_codec.go` is a
library caller. `codegraph impact DecodeBucketKeys` → 4 files (`codec.go`,
`codec_v1.go`, `codec_v2.go`, `internal/locate/pd_codec.go`). **0 tests within 3 hops**
of the v2 impl (testify methods are not `Test*` callers). f2p file
`codec_v2_test.go` is same-package as `codec_v2.go` (in impact), same sparsity exception
as DualExpo.

Sibling on the test’s construction path: `EncodeRegionKey` / `encodeKey` (test builds
inputs with both `encodeWithPrefix` and `suite.codec.EncodeRegionKey`). Not modified.

**Instruction context (one line, no symbol/file/mechanism):** region bucket boundaries
carry a keyspace prefix; decoding them must restore the original user keys, including
empty start/end sentinels.

### Validation

| check | result |
|---|---|
| builds | yes |
| library `go test ./...` | green except `internal/apicodec` (`TestCodecV2/TestDecodeBucketKeys`). No `TestTiKVRecoveredFromDown` flake |
| f2p ≥ 1 | 1 (`TestCodecV2`; only subtest `TestDecodeBucketKeys` fails) |
| 3× not flaky | fail ×3 |
| f2p files in impact | `codec_v2.go` ∈ impact (4 files); `codec_v2_test.go` same-package, not a graph caller (sparsity) |
| no collateral | only `internal/apicodec` failed full `./...` |
| gold revert | restore `codec_v2.go` → pass; docker image same |
| alt accepted | `bytes.TrimPrefix(k, c.prefix)` on buggy tree → pass |
| cheat rejected | special-case user key `'a'` (len 2, last byte `'a'`) → still fail |
| decoy-fix-fails | `decodeKey` drop-last → `TestDecodeEpochNotMatch` fail (existing test) |
| docker | `harbor-iter5-decodebucketkeys`: buggy fail, gold pass (`verifier/iter5_decodebucketkeys_docker.log`) |

**Knobs.** hops 2 (source: test→DecodeBucketKeys→decodeKey) / graph `None` (sparsity);
impact 4 files; fix sites 1; decoys 1 (`decodeKey`); coverage sparsity: 0 tests within
3 hops of the impl.

**Instruction self-check.** `ok=true` (locality 0, tight hunk). names present; symptom
`Not equal`; no leaked `DecodeBucketKeys` / `decodeKey` / `codec_v2.go`; no line
numbers; no diff; repro
`go test -count=1 -timeout 15m -run '^(TestCodecV2)$' ./internal/apicodec/...`.
Checksum on `internal/apicodec/codec_v2_test.go`. Agent timeout 14400s. `golang:1.23`,
no `.git`, modules pre-downloaded.

## Bug B — `client-go-gettimefromts` (rung 5, fair ambiguity, 4-hop path)

**True cause.** `GetTimeFromTS` (`oracle/oracle.go`): `time.Unix(ms/1e3, (ms%1e3)*1e6)`
→ `time.Unix((ms/2)/1e3, ((ms/2)%1e3)*1e6)`. Physical ms halved; wall time lands in
1998 vs 2026 (~248575h).

**Decoy (unmodified).** `tsoSub` (`internal/latch/scheduler.go`):
`return t1.Sub(t2)` with `t1,t2 := oracle.GetTimeFromTS(...)`. It is the duration helper
on the latch recycle path. An engineer debugging `TestRecycle` (queues not emptied after
`expireDuration`) inspects `tsoSub` first: doubling the duration compensates the halved
physical time for recycle only.

- Why inspect it: intermediate on TestRecycle→GetTimeFromTS; overlapping duty (convert
  two timestamps into a duration used as TTL).
- Decoy “fix” (`return t1.Sub(t2) * 2`): **TestRecycle PASSES**; existing
  `TestIsExpired` and `TestPdOracle_GetStaleTimestamp` still FAIL (they call
  `GetTimeFromTS` via `localOracle.IsExpired` / PD stale-TS, not `tsoSub`). Only
  restoring `GetTimeFromTS` is consistent with all three tests.

Second inspect site (unmodified sibling, same file as the cause): `GoTimeToTS`.
`pick_decoy` on the 1-hop `TestPdOracle_GetStaleTimestamp` path ranks it as a sibling
callee. Guard tests include `TestElapsedTTL`, `TestNonFutureStaleTSO`. Not used as the
primary decoy because `tsoSub` sits on the longer TestRecycle path.

**Graph path** (`codegraph_explore`).

```
TestRecycle
  -> recycle (latch.go:305)     # hop 1
  -> recycle (latch.go:288)     # hop 2
  -> tsoSub                     # hop 3  DECOY
  -> GetTimeFromTS              # hop 4  true cause
       -> ExtractPhysical

TestIsExpired
  -> localOracle.IsExpired      # uses GetTimeFromTS; not a sqlite Test* caller
  -> GetTimeFromTS              # does NOT go through tsoSub

TestPdOracle_GetStaleTimestamp
  -> GetTimeFromTS              # hop 1 (shortest graph path)
```

`codegraph impact GetTimeFromTS` → 24 files, including `internal/latch/latch_test.go`
and `oracle/oracles/pd_test.go`. `oracle/oracles/local.go` is in impact;
`local_test.go` is same-package. Full `./...` fails only `internal/latch` and
`oracle/oracles` (tikv / locate / apicodec green; no `TestTiKVRecoveredFromDown` flake).

**Instruction context:** expiry checks and latch recycle must agree with the original
instant encoded in each timestamp.

### Validation

| check | result |
|---|---|
| builds | yes |
| library `go test ./...` | green except `internal/latch` (`TestRecycle`) and `oracle/oracles` (`TestIsExpired`, `TestPdOracle_GetStaleTimestamp`). No `TestTiKVRecoveredFromDown` flake |
| f2p ≥ 1 | 3 |
| 3× not flaky | each of the 3 fails ×3 |
| f2p files in impact | `latch_test.go` and `pd_test.go` ∈ 24-file impact; `local_test.go` same-package as `local.go` ∈ impact |
| no collateral | only those two packages failed full `./...` |
| gold revert | restore `oracle.go` → pass; docker image same |
| alt accepted | `time.Unix(0, ms*int64(time.Millisecond))` on buggy tree → pass |
| cheat rejected | correct conversion only when `ms < 1000` → all 3 still fail |
| decoy-fix-fails | `tsoSub * 2` → TestRecycle pass, TestIsExpired + TestPdOracle_GetStaleTimestamp fail (existing tests) |
| docker | `harbor-iter5-gettimefromts`: buggy fail, gold pass (`verifier/iter5_gettimefromts_docker.log`) |

**Knobs.** hops 4 (TestRecycle) / 1 (TestPdOracle); impact 24 files; fix sites 1;
decoys 1 (`tsoSub`); coverage: `TestIsExpired` is not a reverse-calls hit of
`GetTimeFromTS` in sqlite (goes through `IsExpired` method).

**Instruction self-check.** `ok=true`. names present; symptoms `Expected nil, but got`
and `Should be false` and max-difference 248575h; no leaked `GetTimeFromTS` / `tsoSub` /
`oracle.go`; no line numbers; no diff; repro
`go test -count=1 -timeout 15m -run '^(TestRecycle|TestIsExpired|TestPdOracle_GetStaleTimestamp)$' ./internal/latch/... ./oracle/oracles/...`.
Checksums on `latch_test.go`, `local_test.go`, `pd_test.go`. Agent timeout 14400s.

## Reusable construction

`src/openswe_traces/synth/difficulty.py` (parameterized; no injection):

- `pick_contract_drift_site(index, min_hops, n)`
- `pick_two_site_pair(index, n)`
- `pick_decoy(index, path, n)` — intermediates, then same-file/pkg/stem siblings; guard
  tests are existing callers
- `find_guard_tests(index, symbol, max_hops)` — one reverse-calls BFS
- `find_sparse_branches(coverprofile, max_pct)`
- `design_for_rung(repo, rung, hops, sites, decoys, index)`
- CLI: `openswe-synth --repo <path> --rung 5 --hops 4 --sites 2 --decoys 1`
  (`--patch` / `--out` / `--f2p` builds a Harbor task)

Tests: `tests/test_synth_difficulty.py` on `experiments/codegraph_bugs/fixture_host`.
`uv run pytest tests/` — 73 passed. `uv run ruff check src/openswe_traces/synth/difficulty.py tests/test_synth_difficulty.py` — clean.

## Harbor launch (detached, not waited)

Job dir `experiments/harbor_nex/jobs/grok-xhigh-iter5/` (pid 219126). Trials
`client-go-decodebucketkeys` and `client-go-gettimefromts` started concurrently.

```bash
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
cd /home/evan/Documents/open_swe_traces_research
setsid nohup harbor run \
  --path experiments/harbor_nex/tasks_iter5 \
  --agent cursor-cli \
  --model cursor/cursor-grok-4.6-xhigh \
  --n-concurrent 2 \
  --n-attempts 1 \
  --max-retries 3 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name grok-xhigh-iter5 \
  --yes \
  > experiments/harbor_nex/grok-xhigh-iter5.log 2>&1 < /dev/null &
```

`CURSOR_API_KEY` sourced from `/home/evan/Documents/eval_tasks/.env` (not printed, not committed).

Patches: `experiments/codegraph_bugs/bugs/client-go/{DecodeBucketKeys,GetTimeFromTS}.{patch,alt,cheat}.patch`.
Tasks: `experiments/harbor_nex/tasks_iter5/{client-go-decodebucketkeys,client-go-gettimefromts}/`.

# verifier_batch2 — client-go, 15 hidden black-box suites

Date: 2026-09-20. Repo `client-go` (obfuscated module
`example.internal/kvstore/v2`). Verifier-author role only: suites were written
against `api.md` + `DETAILS.md` + the excised trees. `bugreport.md`,
`contract.md` and `gold.patch` were never read; packaging copies them
mechanically.

## Method

- One `TestDetailNN_*` per numbered `DETAILS.md` line (1:1, verified: 143
  details = 143 tests across 15 units).
- External `*_test` packages wherever the API surface allows (8 units).
  Seven units are same-package because their `api.md` documents unexported
  internals: `latchsched` (`internal/latch` slot queues), `priqueue`
  (`internal/client` heap fields), `pipelineddb`/`unioniter`/`unionstoreget`
  (`internal/unionstore` — unexported `uSnapshot`/`MemDB` internals),
  `backoffer` (`config/retry` exclusion registry), `mvccread`
  (`internal/mockstore/mockkv`), plus `prewrite`'s `CommitterProbe`
  (`txnkv/transaction`, split into an internal `hidden_probe_test.go` + an
  external `hidden_prewrite_test.go`).
- B4: `latchsched`, `priqueue`, `prewrite` carry the named
  `blackbox_hygiene` exemption in `validation.json` (same-package internals
  are the documented surface). The other same-package suites call internals
  bare (no `.lowercase(` method calls) so the heuristic passes unaided.
- Seeded-random inputs throughout: `rand.New(rand.NewSource(hiddenSeed()))`,
  `HIDDEN_SEED` env override, default 20260919. Suites verified under seeds
  {20260919, 20260920, 7301989} — the gate's audit seed 7301989 is exercised
  as the second in-image run per suite.
- Each suite validated locally against three trees before packaging:
  pristine (= gold, must pass), `outputs/excise_work/<unit>/src` (the
  plausible-but-wrong cheat, must fail), and pristine + `excision.patch`
  (the panic-stub tree `environment/src` ships, must fail by panics/
  assertions — never build errors).

## Packaging

`scripts/vf_clientgo_package.py` builds the skeleton per unit:
`environment/src` = pristine + `excision.patch` (panic stubs), a Dockerfile
`FROM ladder-base:client-go-obf` that wipes the base image's stale `/app`
before `COPY src/` (same stale-tree hazard as gin — overlay would resurrect
excision-deleted files), CURSOR-allowlist `task.toml`, `tests/hidden/` with
checksum-guarded `test.sh`, and `gold.patch`/`cheat.patch` in both `tests/`
and `patches/`. `build_affordance_levels(levels=(-2, 0), name_scheme="L")`
then emits `-L0` (bug-report instruction) and `-L2` (contract instruction)
under `experiments/pipeline/tasks_batch2/client-go/`.

One packaging-adjacent fix: client-go bugreports say "Reproduce with:
`tests/test.sh`", which fails B6's literal `go test` reproduce check; both
instructions gain a truthful clause naming the `go test -count=1` invocation
the script performs.

## Suite defects found by gold/seed validation

Behaviour-faithful fixes (model was stricter than gold, or fixture-driven):

- `reqsource`: `IsInternalRequest` prefixes on `"internal"` — `internalx` is
  internal.
- `keyflags`: assert predicates are exclusive — `SetAssertUnknown` reads
  `HasAssertUnknown`, not the two definite predicates; `SetKeyLockedValue*`
  clears the need-constraint-check bit.
- `miscutil`: `CompatibleParseGCTime`'s fallback drops the last
  space-separated field, so a trailing space still parses; `FormatBytes`
  picks units by strict `>` — exact 1 GiB renders "1024 MB" (seed-dependent
  exact-boundary draw).
- `priqueue`: `clean` can leave one canceled entry when `heap.Remove`'s
  `up()` bubbles a canceled item behind the scan index; property weakened to
  live-survival + heap invariant.
- `deadlock`: a `4->3` probe on an existing `3->4` is a real cycle, and a
  failed `Detect` still leaves prior registrations — fixtures rebuilt with
  fresh detectors per sub-check.
- `latchsched`: release wake observed via assigned channel receive (a
  discarded `case <-d2` masked the wake); the `latchListCount` gate lives in
  `acquireSlot`, so recycling is driven through real `Lock`/`UnLock` flows
  with `next`-linked seeded free nodes.
- `backoffer`: `setBackoffExcluded` only mutates registered types —
  `BoTiKVServerBusy` is the only excluded cfg; per-call caps make sleeps
  deterministic.
- `pipelineddb`: an in-flight flush blocks the next `Flush` on `errCh` —
  fixture flushFuncs must complete; fabricated remote hits restricted to a
  fixed key set.
- `mvccread`: `ErrLocked.Key` is the version-suffixed `mvccEncode(key,
  lockVer)`, not `NewMvccKey`.
- `prewrite`: `Assertion_NotExist` wins when both flags hold (sequential
  `if`s in this tree); secondaries come from the committer's real mutation
  set; disk-full is resent by the region sender until the backoff budget
  exhausts (`PrewriteMaxBackoff` shrunk); TTL manager keys off
  `TTLRefreshedTxnSize`; `MinCommitTs!=0` on a failed 1PC is a plain error
  while `OnePcCommitTs` on non-1PC is `zap.Fatal` (subprocess helper);
  `AlreadyExist` only maps to `ErrKeyExist` when the key carries
  `PresumeKeyNotExists` in the txn membuffer.
- `snapscan`: the resolving-token commitment is observed through the lock
  resolver's exported `Resolving()` (record then update-with-same-token
  across two consecutive locked responses), not the request context — the
  scanner builds its own context per retry.

## Per-unit results

(details = DETAILS.md lines; tests = hidden Test funcs; bare/gold/cheat from
`validation.json` `gate_executed`, bare/gold each run under seeds
{default, 7301989}; s = max of the two levels)

| unit | details | tests | bare | gold | cheat | s |
|---|---|---|---|---|---|---|
| backoffer | 13 | 13 | fail | pass | fail | 515 |
| deadlock | 8 | 8 | fail | pass | fail | 116 |
| keyflags | 11 | 11 | fail | pass | fail | 138 |
| kvrpcbatch | 7 | 7 | fail | pass | fail | 158 |
| latchsched | 11 | 11 | fail | pass | fail | 512 |
| localoracle | 9 | 9 | fail | pass | fail | 131 |
| miscutil | 9 | 9 | fail | pass | fail | 173 |
| mvccread | 11 | 11 | fail | pass | fail | 118 |
| pipelineddb | 12 | 12 | fail | pass | fail | 512 |
| prewrite | 12 | 12 | fail | pass | fail | 153 |
| priqueue | 7 | 7 | fail | pass | fail | 155 |
| reqsource | 7 | 7 | fail | pass | fail | 512 |
| snapscan | 10 | 10 | fail | pass | fail | 122 |
| unioniter | 9 | 9 | fail | pass | fail | 159 |
| unionstoreget | 7 | 7 | fail | pass | fail | 163 |

All 30 dirs pass preflight: bare = fail ×2 (panic-stub assertions only —
never `[setup failed]`/build errors), gold = pass ×2, cheat = fail.

## Cheat-rejection signatures (local cheat-tree runs)

| unit | cheat fails on |
|---|---|
| reqsource | nil-receiver render, unknown-type slot, empty-explicit drop (6 details) |
| deadlock | transitive cycle detection, closing-edge hash, failed-Detect non-registration (6) |
| kvrpcbatch | size cut before add, missing-key alignment, count cut strictly over limit (3) |
| unionstoreget | buffer-first ordering, tombstone masking, non-NotFound error propagation (5) |
| unioniter | buffer-wins ties, tombstone masking, orphan tombstone skip (5) |
| latchsched | power-of-two slot table, lock-gen key sort, acquire-vs-wait (9) |
| backoffer | vars weight budget, excluded-sleep accounting (12) |
| pipelineddb | flush trigger conditions, read order, flags order (10) |
| mvccread | mvcc key encoding, region containment, lock wire format (9) |
| prewrite | assertion precedence, pessimistic actions, forUpdateTS constraints, async TTL, TTL manager, 1PC handling, minCommitTS, undetermined err (8) |
| snapscan | batch-size floor, region-bound clipping, resume semantics, bound-by-direction, lock resolution, retry flags (8) |
| keyflags | presume→need-check implication, predicate exclusivity, locked-value clears constraint bit (8) |
| miscutil | GC-time fallback, strict byte boundaries, hex/ASCII helpers (6) |
| priqueue | decreasing-priority order, Take partial sort, internal-slice return (6) |
| localoracle | distinct timestamps under pinned clock, same-ms counter, hook-driven reads (7) |

## Exclusions

None. Overlap check (`check_unit_overlap.py --extra oswt-AUclientgo`) was
CLEAN for all 15 units; all 15 packaged and preflight-PASS at L0 and L2.

## Notes

- `outputs/VFclientgo.log` holds the preflight run (4 parallel groups,
  `scripts/preflight_task.py` from the consolidated gate worktree).
- Cheat details that hold (e.g. unionstoreget D4/D5, prewrite D4/D7) are
  commitments the plausible-but-wrong implementation happens to satisfy —
  suite-level rejection still holds via the other details.

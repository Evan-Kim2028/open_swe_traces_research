# verifier_batch2 — goa, 15 hidden black-box suites

Date: 2026-09-20. Repo `goa` v3 (obfuscated module `example.internal/goa`).
Verifier-author role only: suites were written against `api.md` +
`DETAILS.md` + the excised tree. `bugreport.md`, `contract.md` and
`gold.patch` were never read; packaging copies them mechanically.

## Method

- One `TestDetailNN_*` per numbered `DETAILS.md` line (1:1, verified: 154
  details = 154 tests). All suites are external `package *_test` driving
  only the exported API — no `blackbox_hygiene` exemptions needed.
- Hidden sources live under `experiments/pipeline/work/vf_goa/hidden/<unit>/`
  with `dir__file_test.go` names decoded to `dir/file_test.go` at install.
- Seeded-random inputs where behavior is probabilistic:
  `rand.New(rand.NewSource(hiddenSeed()))`, `HIDDEN_SEED` env override,
  default 20260919. Every suite verified in-image under seeds
  {5, 20260919, 20260920} via `work/vf_goa/run.sh` (image `ladder-base:goa`,
  `GOPROXY=off`).
- Where observable gold behavior diverged from a naive reading of the
  detail, the suite pins the *observed* contract (documented per unit
  below) — gold is the arbiter, not the detail's prose.
- Every test carries a `vfGuard` recover shim so a stub panic is recorded
  as that detail's failure instead of aborting the binary — per-detail
  verdicts survive on excised and cheat trees alike.
- Packaging: `scripts/vf_goa_package.py` builds the skeleton
  (`environment/src` = excised tree, CURSOR-allowlist `task.toml`,
  `tests/hidden/`, `gold.patch`/`cheat.patch` in `tests/` + `patches/`,
  `RUN rm -rf /app` before COPY) then
  `build_affordance_levels(levels=(-2, 0), name_scheme="L")` → `-L0` (bug
  report) and `-L2` (contract) under
  `experiments/pipeline/tasks_batch2/goa/`. `assert_b4_pass` ran clean on
  all 30 dirs.
- Preflight: `scripts/preflight_task.py` (gate `exec_rules`,
  `HIDDEN_SEED` auto-detected) — bare×2 seeds must FAIL, gold×2 must PASS,
  cheat×1 must FAIL.

## Per-unit results

(details = DETAILS.md lines covered 1:1; tests = hidden Test functions;
bare/gold/cheat from `validation.json` `gate_executed`; s = max seconds
over the L0/L2 gate runs)

| unit | details | tests | bare | gold | cheat | s |
|---|---|---|---|---|---|---|
| dupexpr | 11 | 11 | fail | pass | fail | 36 |
| exprhash | 10 | 10 | fail | pass | fail | 37 |
| httpclienterr | 13 | 13 | fail | pass | fail | 63 |
| httpencoding | 13 | 13 | fail | pass | fail | 79 |
| httperrresp | 8 | 8 | fail | pass | fail | 59 |
| httpmux | 10 | 10 | fail | pass | fail | 65 |
| importalias | 12 | 12 | fail | pass | fail | 55 |
| mappedattr | 12 | 12 | fail | pass | fail | 34 |
| namescope | 12 | 12 | fail | pass | fail | 50 |
| reqidgen | 8 | 8 | fail | pass | fail | 24 |
| retrypolicy | 7 | 7 | fail | pass | fail | 33 |
| sampler | 9 | 9 | fail | pass | fail | 24 |
| skipwriter | 7 | 7 | fail | pass | fail | 25 |
| svcerror | 12 | 12 | fail | pass | fail | 20 |
| traceopts | 10 | 10 | fail | pass | fail | 18 |

Preflight: **30/30 dirs PASS** (15 units × L0+L2). Bare fails only by
excised-stub panics/assertions — never `[setup failed]`/build errors.

## Cheat-rejection signatures (local in-image runs)

| unit | cheat fails on |
|---|---|
| dupexpr | shared-copy-per-origin collapse semantics |
| exprhash | union emission law — gold keeps the baseline insertion-sort artifact order; cheat sorted cleanly |
| httpclienterr | debug-doer request/response classifier details |
| httpencoding | Accept-parameter parsing / suffix dispatch edges |
| httperrresp | wrapping path — type-asserts instead of `errors.As` |
| httpmux | installs negotiated custom 404 (gold's never fires — `m.middlewares` nil) + returns empty non-nil `Vars` |
| importalias | picks inferred over explicit alias in the generated bucket |
| mappedattr | `KeyName` on unknown element doesn't panic; `Delete` leaves the required marker |
| namescope | `generatedImportName` skips `path.Clean` (`./types` uncleaned) |
| reqidgen | `shortID` uses `StdEncoding` (`+`/`/` bytes) instead of `RawURLEncoding` |
| retrypolicy | omits the `Retryable()==true` error class |
| sampler | adaptive rate floor clamps to 0 instead of 1 — sampler stalls after a fast window |
| skipwriter | `Close` on un-read adapter returns early — producer goroutine never starts, adapter stays readable |
| svcerror | fixed `"AAAAAAAA"` error IDs; `MergeErrors` drops Timeout/Fault ANDs, history, and cause join; several constructors left stubbed |
| traceopts | option validation deferred into the closure — `SamplingPercent(-1)` etc. don't panic at construction |

## Observable-contract corrections found by probing gold

- `dupexpr`: `ValidationExpr.Dup` shares scalar pointers; `MetaExpr.Dup`
  is a shallow map copy; bases re-visited through a shared attribute
  collapse to the input pointer; generated-RT registration is conditional
  on prior queue membership.
- `exprhash`: union values emit in insertion order of an internal sort
  whose comparator reads *original* positions — deterministic but neither
  sorted nor insertion order; nil hash panics by nil-deref; tag fold order
  leaks map order across calls.
- `httpmux`: the chi `Use` middleware queue path is dead code
  (`m.middlewares` is never initialized), so the negotiated custom 404
  installed by the mux never fires — Detail05 asserts the observed
  behavior, not raw-chi behavior.
- `mappedattr`: `KeyName` of a key returns the identity (no panic);
  re-`Map`/`Merge` leave stale reverse-map entries resolvable; merge
  replaces same-named attributes via `Attribute()` deep copy.
- `namescope`: `Name` does NOT skip taken names (`name+count+1`
  literally); qualified unbound lookups don't bind the hash; post-freeze
  qualified unbound returns `pkg.base` without panic.
- `reqidgen`: non-string context value with reuse enabled panics on a hard
  type assert; `UseRequestIDOption` overwrites a previously configured
  custom header.
- `sampler`: recompute fires exactly on the `sampleSize`-th call and
  applies to that same call; the window clock measures between recompute
  points; lower clamp is 1 (never 0).
- `skipwriter`: `PipeReader.Close` always returns nil — the "pipe-closed
  result" of a second Close is nil, not `ErrClosedPipe`.
- `svcerror`: `MergeErrors(x, nil)` returns `x` *unchanged* (no
  conversion); `InvalidLengthError` prints `value` (arg4) after "must be"
  and `ln` (arg3) after "got"; `history[0]` is the mutated left object;
  `InvalidRangeError` formats the bound with `%d` (float bounds produce
  `%!d(...)` artifacts).
- `traceopts`: option constructors panic eagerly at construction;
  `tracedLogger.Log` passes the prepended keyvals slice as a *single* arg
  (unspread) when traceID is non-empty.

## Exclusions

None. `check_unit_overlap.py --extra` reported the 15 units clean against
the 10 existing ones; all 15 packaged and preflight-PASS at L0 and L2.

## Notes

- `vfGuard` (a `recover` shim at the top of each `TestDetailNN_*`)
  converts panics into per-test failures so excised/cheat runs report a
  verdict per detail rather than aborting at the first stub — the
  per-detail grading signal survives.
- Execution log: `outputs/VFgoa.log`.

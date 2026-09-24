# verifier_batch2 — gin, 14 hidden black-box suites

Date: 2026-09-20. Repo `gin` (obfuscated module `example.internal/httprouter`,
package `gin`). Verifier-author role only: suites were written against
`api.md` + `DETAILS.md` + the excised tree. `bugreport.md`, `contract.md` and
`gold.patch` were never read; packaging copies them mechanically.

## Method

- One `TestDetailNN_*` per numbered `DETAILS.md` line (1:1, verified: 121
  details = 121 tests). External `package gin_test` for 12 units; the two
  radix-tree units (`routelookup`, `treeinsert`) are same-package `gin` tests
  because their `api.md` documents unexported internals — B4 exempted via the
  named `blackbox_hygiene` check seeded in `validation.json`.
- Seeded-random inputs throughout: `rand.New(rand.NewSource(hiddenSeed()))`,
  `HIDDEN_SEED` env override, default 20260919. Suites verified under seeds
  {1, 42, 20260919, 20260920, 7301989} — the gate's audit seed is 7301989 and
  is exercised as the second in-image run per suite.
- Each unit run locally in-image against four trees: excised (must panic /
  assert, never setup errors), gold (must pass), cheat (must fail), gold with
  audit seeds (must pass).
- Packaging: `scripts/vf_gin_package.py` builds the A0 skeleton
  (`environment/src` = excised tree, CURSOR-allowlist `task.toml`,
  `tests/hidden/`, `gold.patch`/`cheat.patch` in `tests/` + `patches/`) then
  `build_affordance_levels(levels=(-2, 0), name_scheme="L")` → `-L0` (bug
  report) and `-L2` (contract) under `experiments/pipeline/tasks_batch2/gin/`.
- Preflight: `scripts/preflight_task.py` (gate `exec_rules`) — bare×2
  (seed 20260919 + 7301989) must FAIL, gold×2 must PASS, cheat×1 must FAIL.

## Packaging defect found and fixed

`render_ladder_base_dockerfile` does `COPY src/ /app/` over `ladder-base:gin`,
whose `/app` already holds a stale tree of a different obfuscation generation
(`module example.internal/gin`). Overlay leaves excision-deleted files in
place; `errors_test.go` imports `example.internal/gin/codec/json`, which under
the wiped `go.mod` is an unresolvable external module — `go test ./...`
classified `infra` on every variant for the `errors` unit. Fixed by wiping
`/app` before COPY in the task Dockerfile (`RUN rm -rf /app`). 5 units delete
a whole test file (auth_test.go, errors_test.go, logger_test.go,
recovery_test.go, response_writer_test.go); only errors_test.go carried the
fatal stale import, but the wipe makes every image byte-exact.

Two seed-robustness bugs in the suites themselves were also caught by the
in-image audit seed: `basicauth` Detail03 (randomly generated empty/colliding
credentials) and `respwriter` Detail04 (bodiless status codes 204/304 drawn at
random). Both fixed and re-verified across five seeds.

## Per-unit results

(details covered = DETAILS.md lines; tests = hidden Test functions;
bare/gold/cheat from `validation.json` `gate_executed`; s = seconds)

| unit | details | tests | bare | gold | cheat | s |
|---|---|---|---|---|---|---|
| basicauth | 6 | 6 | fail | pass | fail | 74 |
| clientip | 8 | 8 | fail | pass | fail | 75 |
| ctxfiles | 9 | 9 | fail | pass | fail | 74 |
| ctxquery | 7 | 7 | fail | pass | fail | 72 |
| enginecfg | 10 | 10 | fail | pass | fail | 74 |
| enginemux | 10 | 10 | fail | pass | fail | 75 |
| errors | 8 | 8 | fail | pass | fail | 71 |
| loggerfmt | 11 | 11 | fail | pass | fail | 76 |
| negotiate | 6 | 6 | fail | pass | fail | 76 |
| recoverymw | 9 | 9 | fail | pass | fail | 70 |
| respwriter | 8 | 8 | fail | pass | fail | 73 |
| routelookup | 10 | 10 | fail | pass | fail | 72 |
| routergroup | 8 | 8 | fail | pass | fail | 77 |
| treeinsert | 9 | 9 | fail | pass | fail | 76 |

Per dir both L0 and L2 pass; the `s` column is the max of the two
`gate_executed` times. Bare fails by excised-stub panic/assertions only —
never `[setup failed]`/build errors.

## Cheat-rejection signatures (local in-image runs)

| unit | cheat fails on |
|---|---|
| basicauth | empty-user panic missing; proxy default realm |
| clientip | TrustedPlatform verbatim; per-header walk; right-to-left validation; ContentType param strip; websocket both-headers; scheme precedence |
| ctxfiles | SaveUploadedFile; URL.Path restore; attachment; ASCII boundary; cookies; nil-body GetRawData |
| ctxquery | query cache one-shot re-parse; form/multipart cache |
| enginecfg | New() default set; h2c wrap; LoadHTML* modes |
| enginemux | Routes() (excised in cheat); addRoute asserts |
| errors | struct-meta JSON shape; len-1 slice shape; nil-vs-empty Errors(); String() meta line |
| loggerfmt | defaults; skip paths; private-only error; status/method/latency colors; formatter shape; query path; ErrorLoggerT |
| negotiate | q-strip; empty-offer panic; wildcard match; YAML dispatch; chooseData panic |
| recoverymw | constructors; classification; broken-pipe; log format; auth masking; function trim; source line; stack shape |
| respwriter | reset sentinel; lazy WriteHeader; Written; Hijack unsupported-writer guard |
| routelookup | static-before-wildcard; param unescape; params-buffer growth (panic) |
| routergroup | abortIndex cap; joinPaths trailing slash; method validation; static-path validation; OnlyFilesFS |
| treeinsert | root marking; prefix split; wildcard-child-last ("wildcard segments are not supported") |

## Exclusions

None. All 14 units packaged and preflight-PASS at L0 and L2.

## Notes

- `routelookup`/`treeinsert` carry `checks: [blackbox_hygiene]` in
  `validation.json`: their `api.md` documents unexported radix internals, so a
  same-package suite is the documented surface; B4's lowercase-call heuristic
  is inapplicable by design.
- Execution log: `outputs/VFgin.log`.

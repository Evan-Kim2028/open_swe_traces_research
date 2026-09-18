# Verifier rules for synthetic SWE tasks (living document)

Started 2026-09-18 from the Harbor experiments under `experiments/harbor_nex/`. Every rule was
added because a task or a run broke without it. Update this file when a new failure class appears;
record the trigger and the kill count.

Two groups. **Validity rules** make a task well-posed regardless of solver. **Fairness rules** define
what information the solver may use; only these interact with the affordance/information knob.

## A. Validity rules (solver-independent)

| # | rule | why | kill count so far |
|---|---|---|---|
| A1 | Gold restore passes the verifier | otherwise the task is broken | 0 (never violated after image proof) |
| A2 | An alternative correct fix, structurally different from gold, passes | catches too-narrow verifiers | 0 exclusions on 40+ tasks (see note) |
| A3 | A cheat fix (special-case the test inputs, hardcode outputs) fails | catches too-broad verifiers | 0 exclusions on 40+ tasks (see note) |
| A4 | Fail-to-pass tests lie inside the impact set of the change | the failure must be caused by the injected change | 2 of first 10 bugs rejected (logger mutations) |
| A5 | No collateral failures elsewhere in the suite; f2p stable across 3 runs | flaky or side-effect tasks are noise | 1 rejected (hung test), 1 flake noted |
| A6 | Multi-site bugs: fixing any single site alone still fails | otherwise it is a single-site bug | 0 exclusions (checked on 6 tasks) |
| A7 | Decoys are pre-existing, unmodified code; the wrong fix breaks an existing test | no planted traps | 0 exclusions (checked on 4 tasks) |
| A8 | Prove in the built image: buggy tree fails, gold passes, before launch | host and image can differ | 1 caught (patch silently not applied to the snapshot) |
| A9 | Dynamic gates need a proven non-passer: naive impl passes correctness and fails the gate; race gate 10/10 deterministic | otherwise the gate is decorative | perf gate proven on 2 tasks; race gate never reached 10/10, task dropped |
| A10 | Verifier phase runs with no network | reproducibility | 0 incidents after enabling |

Note on A2/A3: zero exclusions means either the builder already produces verifiers that satisfy them, or
the checks are too weak to bite. Distinguish by ablation (see section D).

## B. Fairness rules (define the information knob)

| # | rule | why | kill count so far |
|---|---|---|---|
| B1 | Test files are checksum-guarded; editing them fails the trial | the solver must fix the code, not the oracle | 1 trial failed (Composer edited consumer tests on the two-module task) |
| B2 | Container egress denied except the agent's own API hosts; any web-tool use in the trajectory disqualifies the trial | agent-side web tools run on vendor servers and cannot be blocked from the container | 15 of 30 Grok trials disqualified; 0 of 32 Composer trials |
| B3 | Existing example tests are not the verifier for hard tasks | a complete example suite is a spec; models reconstruct from specs | every in-tree-test task (36 Grok, 26 Composer) solved |
| B4 | Hidden tests must be black-box: exported API and caller-facing behavior only | white-box tests measure name-guessing | 1 family voided at A0/A1 (spec-reimpl called 9 internal names) |
| B5 | Prefer property verifiers (seeded random inputs, adversarial edges) over example verifiers; hardcoded solutions must fail them | properties cannot be memorized from examples | property family passed at A0; used as the fix for B4 |
| B6 | Instruction floor: observable symptom, expected vs got, reproduction command | not too broad | 0 (self-check on every task) |
| B7 | Instruction ceiling: no symbol, file, or mechanism names; no diff; no line numbers; name-leakage check against the change | not too narrow | 0 after the check was added; before it, test names leaked the symbol on every rung-1..5 task |
| B8 | No-web clause in every instruction | makes B2 a rule the solver was told, not a trap | n/a |

## C. Process rules

| # | rule |
|---|---|
| C1 | Audit every failure before counting it: (a) legitimate, (b) verifier too narrow, (c) instruction insufficient, (d) infrastructure. Only (a) counts. Tally so far: (a) 4, (b) 1 family, (c) 0, (d) 12 trials (rate limit). |
| C2 | Lazy ladder: build and run the next affordance level only where the previous one fails. |
| C3 | Obfuscate identity (module path, brand strings, subsystem symbols) before measuring anything on a public repo; then verify with a control run. Result: solve times unchanged, so recall was not the driver, but keep it as insurance. |
| C4 | One base image per tree; per-task layers. (Not yet done; each task rebuilds from golang:1.23.) |

## D. What we do not know yet

- **Are these most of the rules?** They cover the failure classes seen in ~100 trials on one Go repo.
  New verifier classes (concurrency, config/env, multi-service) and new solver behaviors will add rules.
  Expect a long tail with diminishing returns; rules come from audits, so the audit habit matters more
  than the list.
- **Which rules carry the weight?** Instrument the CLI so every candidate task logs a verdict per rule;
  the kill-count column becomes real data. Current partial counts say A4, A8, B2, B3, B4 did the
  rejecting; A2, A3, A6, A7 never fired. Either they are satisfied by construction or too weak. Ablation:
  turn each off, regenerate 20 tasks, count how many bad tasks reach a solver.
- **Does codegraph improve discovery over not using it?** Not proven. Evidence is one-sided: 9 valid
  cross-file bugs per builder-hour with it, and the graph correctly rejected an isolated-package host.
  The controlled test: same builder, same repo, same one-hour budget, two conditions (graph vs
  grep+gopls only), measure valid tasks per hour, cross-file rate, and A4 rejection rate. Not run.

## E. Terms

- **Dynamic gate**: our term for a verifier condition that is a runtime property rather than an example
  assertion, e.g. `go test -race` clean, or ns/op under a threshold derived from gold. Adjacent standard
  terms: dynamic analysis, performance regression test, benchmark gate, property-based testing. Not a
  standard term; use "property/performance/race gate" when precision matters.
- **Affordance level (A0..A4)**: how much of the hidden information is given back: contract only,
  + test names, + signatures, + one test file, + all tests.

## F. Kill counts (instrumented)

Generated 2026-09-18 by `openswe-synth rules-report --tasks-glob 'experiments/harbor_nex/tasks*/*'`.
89 Harbor task dirs under `experiments/harbor_nex/tasks*/*`. Verdicts live in each
`<task_dir>/validation.json` as `rule_verdicts`. Missing verdicts were backfilled from
RESULT.md / ITER_*.md validation tables, `verifier/` logs, patches, `task.toml`,
`tests/test.sh`, and `instruction.md`. `jobs/` was not read.

n_tasks = 89

| # | n_evaluated | n_failed (kills) | n_skipped | killed tasks |
|---|---:|---:|---:|---|
| A1 | 66 | 1 | 23 | `experiments/harbor_nex/tasks_obf/client-go-memget-obf` |
| A2 | 44 | 3 | 45 | `experiments/harbor_nex/tasks_all/client-go-intervalcontains`, `experiments/harbor_nex/tasks_iter4/client-go-intervalcontains`, `experiments/harbor_nex/tasks_obf/client-go-memget-obf` |
| A3 | 57 | 0 | 32 | — |
| A4 | 29 | 0 | 60 | — |
| A5 | 44 | 3 | 45 | `experiments/harbor_nex/tasks_all/client-go-gettimefromts`, `experiments/harbor_nex/tasks_iter5/client-go-gettimefromts`, `experiments/harbor_nex/tasks_iter6/client-go-gettimefromts` |
| A6 | 4 | 0 | 85 | — |
| A7 | 0 | 0 | 89 | — |
| A8 | 61 | 4 | 28 | `experiments/harbor_nex/tasks_obf/client-go-batchcmds-obf`, `experiments/harbor_nex/tasks_obf/client-go-interceptor-obf`, `experiments/harbor_nex/tasks_obf/client-go-keyspacecodec-obf`, `experiments/harbor_nex/tasks_obf/client-go-memget-obf` |
| A9 | 10 | 0 | 79 | — |
| A10 | 89 | 0 | 0 | — |
| B1 | 89 | 27 | 0 | `experiments/harbor_nex/tasks/client-go-getglobalconfig`, `experiments/harbor_nex/tasks/client-go-iserrnotfound`, `experiments/harbor_nex/tasks/client-go-isfakeregionerror`, `experiments/harbor_nex/tasks/client-go-newbackofferwithvars`, `experiments/harbor_nex/tasks/client-go-newregionrequestsender`, `experiments/harbor_nex/tasks/client-go-newrequest`, `experiments/harbor_nex/tasks/dailycodingproblem-go-match`, `experiments/harbor_nex/tasks/dailycodingproblem-go-twosumbest`, `experiments/harbor_nex/tasks/dailycodingproblem-go-twosumbrute`, `experiments/harbor_nex/tasks_all/client-go-decodekeyv1`, `experiments/harbor_nex/tasks_all/client-go-getglobalconfig`, `experiments/harbor_nex/tasks_all/client-go-getstoretypebymeta`, `experiments/harbor_nex/tasks_all/client-go-iserrnotfound`, `experiments/harbor_nex/tasks_all/client-go-isfakeregionerror`, `experiments/harbor_nex/tasks_all/client-go-keyspaceidcodec`, `experiments/harbor_nex/tasks_all/client-go-newbackofferwithvars`, `experiments/harbor_nex/tasks_all/client-go-newregionrequestsender`, `experiments/harbor_nex/tasks_all/client-go-newrequest`, `experiments/harbor_nex/tasks_all/client-go-onepc-scope`, `experiments/harbor_nex/tasks_all/dailycodingproblem-go-match`, `experiments/harbor_nex/tasks_all/dailycodingproblem-go-twosumbest`, `experiments/harbor_nex/tasks_all/dailycodingproblem-go-twosumbrute`, `experiments/harbor_nex/tasks_hard/client-go-decodekeyv1`, `experiments/harbor_nex/tasks_hard/client-go-getstoretypebymeta`, `experiments/harbor_nex/tasks_hard/client-go-keyspaceidcodec`, `experiments/harbor_nex/tasks_obf/client-go-onepc-scope-obf`, `experiments/harbor_nex/tasks_two_repo/client-go-onepc-scope` |
| B2 | 89 | 1 | 0 | `experiments/harbor_nex/tasks_smoke_nonet/client-go-memget` |
| B3 | 89 | 60 | 0 | `experiments/harbor_nex/tasks/client-go-getglobalconfig`, `experiments/harbor_nex/tasks/client-go-iserrnotfound`, `experiments/harbor_nex/tasks/client-go-isfakeregionerror`, `experiments/harbor_nex/tasks/client-go-newbackofferwithvars`, `experiments/harbor_nex/tasks/client-go-newregionrequestsender`, `experiments/harbor_nex/tasks/client-go-newrequest`, `experiments/harbor_nex/tasks/dailycodingproblem-go-match`, `experiments/harbor_nex/tasks/dailycodingproblem-go-twosumbest`, `experiments/harbor_nex/tasks/dailycodingproblem-go-twosumbrute`, `experiments/harbor_nex/tasks_all/client-go-batchcmds`, `experiments/harbor_nex/tasks_all/client-go-decodebucketkeys`, `experiments/harbor_nex/tasks_all/client-go-decodekeyv1`, `experiments/harbor_nex/tasks_all/client-go-dualexpo`, `experiments/harbor_nex/tasks_all/client-go-extractphysical`, `experiments/harbor_nex/tasks_all/client-go-getglobalconfig`, `experiments/harbor_nex/tasks_all/client-go-getphysical`, `experiments/harbor_nex/tasks_all/client-go-getstoretypebymeta`, `experiments/harbor_nex/tasks_all/client-go-gettimefromts`, `experiments/harbor_nex/tasks_all/client-go-gettimestamp`, `experiments/harbor_nex/tasks_all/client-go-interceptor`, `experiments/harbor_nex/tasks_all/client-go-intervalcontains`, `experiments/harbor_nex/tasks_all/client-go-iserrnotfound`, `experiments/harbor_nex/tasks_all/client-go-isfakeregionerror`, `experiments/harbor_nex/tasks_all/client-go-keyspacecodec`, `experiments/harbor_nex/tasks_all/client-go-keyspaceidcodec`, `experiments/harbor_nex/tasks_all/client-go-keyspaceprefix`, `experiments/harbor_nex/tasks_all/client-go-memget`, `experiments/harbor_nex/tasks_all/client-go-memsetvalue`, `experiments/harbor_nex/tasks_all/client-go-newbackofferwithvars`, `experiments/harbor_nex/tasks_all/client-go-newregionrequestsender`, `experiments/harbor_nex/tasks_all/client-go-newrequest`, `experiments/harbor_nex/tasks_all/client-go-onepc-scope`, `experiments/harbor_nex/tasks_all/dailycodingproblem-go-match`, `experiments/harbor_nex/tasks_all/dailycodingproblem-go-twosumbest`, `experiments/harbor_nex/tasks_all/dailycodingproblem-go-twosumbrute`, `experiments/harbor_nex/tasks_hard/client-go-decodekeyv1`, `experiments/harbor_nex/tasks_hard/client-go-getstoretypebymeta`, `experiments/harbor_nex/tasks_hard/client-go-keyspaceidcodec`, `experiments/harbor_nex/tasks_iter3/client-go-dualexpo`, `experiments/harbor_nex/tasks_iter3/client-go-keyspaceprefix`, `experiments/harbor_nex/tasks_iter4/client-go-extractphysical`, `experiments/harbor_nex/tasks_iter4/client-go-intervalcontains`, `experiments/harbor_nex/tasks_iter5/client-go-decodebucketkeys`, `experiments/harbor_nex/tasks_iter5/client-go-gettimefromts`, `experiments/harbor_nex/tasks_iter6/client-go-getphysical`, `experiments/harbor_nex/tasks_iter6/client-go-gettimefromts`, `experiments/harbor_nex/tasks_iter7/client-go-gettimestamp`, `experiments/harbor_nex/tasks_iter7/client-go-memsetvalue`, `experiments/harbor_nex/tasks_iter8/client-go-batchcmds`, `experiments/harbor_nex/tasks_iter8/client-go-interceptor`, `experiments/harbor_nex/tasks_iter9/client-go-keyspacecodec`, `experiments/harbor_nex/tasks_iter9/client-go-memget`, `experiments/harbor_nex/tasks_obf/client-go-batchcmds-obf`, `experiments/harbor_nex/tasks_obf/client-go-interceptor-obf`, `experiments/harbor_nex/tasks_obf/client-go-keyspacecodec-obf`, `experiments/harbor_nex/tasks_obf/client-go-memget-obf`, `experiments/harbor_nex/tasks_obf/client-go-memsetvalue-obf`, `experiments/harbor_nex/tasks_obf/client-go-onepc-scope-obf`, `experiments/harbor_nex/tasks_smoke_nonet/client-go-memget`, `experiments/harbor_nex/tasks_two_repo/client-go-onepc-scope` |
| B4 | 29 | 9 | 60 | `experiments/harbor_nex/tasks_devin_A0/spec-reimpl-A0`, `experiments/harbor_nex/tasks_unsolv/spec-reimpl-A0`, `experiments/harbor_nex/tasks_unsolv/spec-reimpl-A1`, `experiments/harbor_nex/tasks_unsolv/spec-reimpl-A2`, `experiments/harbor_nex/tasks_unsolv/spec-reimpl-A3`, `experiments/harbor_nex/tasks_unsolv/spec-reimpl-A4`, `experiments/harbor_nex/tasks_unsolv_A0/spec-reimpl-A0`, `experiments/harbor_nex/tasks_unsolv_A1/spec-reimpl-A1`, `experiments/harbor_nex/tasks_unsolv_A2/spec-reimpl-A2` |
| B5 | 33 | 9 | 56 | `experiments/harbor_nex/tasks_devin_A0/spec-reimpl-A0`, `experiments/harbor_nex/tasks_unsolv/spec-reimpl-A0`, `experiments/harbor_nex/tasks_unsolv/spec-reimpl-A1`, `experiments/harbor_nex/tasks_unsolv/spec-reimpl-A2`, `experiments/harbor_nex/tasks_unsolv/spec-reimpl-A3`, `experiments/harbor_nex/tasks_unsolv/spec-reimpl-A4`, `experiments/harbor_nex/tasks_unsolv_A0/spec-reimpl-A0`, `experiments/harbor_nex/tasks_unsolv_A1/spec-reimpl-A1`, `experiments/harbor_nex/tasks_unsolv_A2/spec-reimpl-A2` |
| B6 | 89 | 8 | 0 | `experiments/harbor_nex/tasks/client-go-newregionrequestsender`, `experiments/harbor_nex/tasks/dailycodingproblem-go-match`, `experiments/harbor_nex/tasks/dailycodingproblem-go-twosumbest`, `experiments/harbor_nex/tasks/dailycodingproblem-go-twosumbrute`, `experiments/harbor_nex/tasks_all/client-go-newregionrequestsender`, `experiments/harbor_nex/tasks_all/dailycodingproblem-go-match`, `experiments/harbor_nex/tasks_all/dailycodingproblem-go-twosumbest`, `experiments/harbor_nex/tasks_all/dailycodingproblem-go-twosumbrute` |
| B7 | 89 | 11 | 0 | `experiments/harbor_nex/tasks_unsolv/dynamic-pipeline-A1`, `experiments/harbor_nex/tasks_unsolv/dynamic-pipeline-A2`, `experiments/harbor_nex/tasks_unsolv/dynamic-pipeline-A3`, `experiments/harbor_nex/tasks_unsolv/dynamic-pipeline-A4`, `experiments/harbor_nex/tasks_unsolv/property-backoff-A1`, `experiments/harbor_nex/tasks_unsolv/property-backoff-A2`, `experiments/harbor_nex/tasks_unsolv/property-backoff-A3`, `experiments/harbor_nex/tasks_unsolv/property-backoff-A4`, `experiments/harbor_nex/tasks_unsolv_A1/dynamic-pipeline-A1`, `experiments/harbor_nex/tasks_unsolv_A2/dynamic-pipeline-A2`, `experiments/harbor_nex/tasks_unsolv_A3/dynamic-pipeline-A3` |
| B8 | 89 | 28 | 0 | `experiments/harbor_nex/tasks/client-go-getglobalconfig`, `experiments/harbor_nex/tasks/client-go-iserrnotfound`, `experiments/harbor_nex/tasks/client-go-isfakeregionerror`, `experiments/harbor_nex/tasks/client-go-newbackofferwithvars`, `experiments/harbor_nex/tasks/client-go-newregionrequestsender`, `experiments/harbor_nex/tasks/client-go-newrequest`, `experiments/harbor_nex/tasks/dailycodingproblem-go-match`, `experiments/harbor_nex/tasks/dailycodingproblem-go-twosumbest`, `experiments/harbor_nex/tasks/dailycodingproblem-go-twosumbrute`, `experiments/harbor_nex/tasks_hard/client-go-decodekeyv1`, `experiments/harbor_nex/tasks_hard/client-go-getstoretypebymeta`, `experiments/harbor_nex/tasks_hard/client-go-keyspaceidcodec`, `experiments/harbor_nex/tasks_iter3/client-go-dualexpo`, `experiments/harbor_nex/tasks_iter3/client-go-keyspaceprefix`, `experiments/harbor_nex/tasks_iter4/client-go-extractphysical`, `experiments/harbor_nex/tasks_iter4/client-go-intervalcontains`, `experiments/harbor_nex/tasks_iter5/client-go-decodebucketkeys`, `experiments/harbor_nex/tasks_iter5/client-go-gettimefromts`, `experiments/harbor_nex/tasks_iter6/client-go-getphysical`, `experiments/harbor_nex/tasks_iter6/client-go-gettimefromts`, `experiments/harbor_nex/tasks_iter7/client-go-gettimestamp`, `experiments/harbor_nex/tasks_iter7/client-go-memsetvalue`, `experiments/harbor_nex/tasks_iter8/client-go-batchcmds`, `experiments/harbor_nex/tasks_iter8/client-go-interceptor`, `experiments/harbor_nex/tasks_iter9/client-go-keyspacecodec`, `experiments/harbor_nex/tasks_iter9/client-go-memget`, `experiments/harbor_nex/tasks_smoke_nonet/client-go-memget`, `experiments/harbor_nex/tasks_two_repo/client-go-onepc-scope` |


### F.1 Correction (2026-09-18 21:00Z)

The A1/A8 kills on `tasks_obf/*` (and A2 on memget-obf) were **harness artifacts**: the obfuscation
builder's proof script ran `test.sh` in a shell without `go` on PATH, so every run reported `REWARD=0`,
gold included. Re-proof in rebuilt images: buggy fails, gold passes, perf gate holds, for all six.
Composer's 5/6 clean passes on these tasks were consistent with that. `validation.json` verdicts updated.

New process rule **C5**: a proof that reports gold failing is a harness failure until confirmed by hand
(gold must pass by construction, rule A1). Validate the proof harness itself (toolchain on PATH,
exit codes propagated) before trusting any REWARD it emits.

### F.2 Re-proof findings (2026-09-18 21:10Z)
- memget-obf: gold passes; a loop re-proof under load avg 10 tripped the perf gate. **Rule A11 (new):**
  timing gates must be proved with the host otherwise idle, and the threshold must carry a margin
  (gold x3 was enough at load 1, not at load 10); record load with every perf measurement.
- batchcmds-obf: the obfuscated `gold.patch` contained hunks in `client_test.go` (rename spill-over), so
  the checksum guard failed gold itself. Test hunks stripped from gold; task was valid for solvers
  (Composer passed without gold). **Rule A12 (new):** gold/alt/cheat patches must not touch guarded test
  files; check this before the image proof.
- onepc-scope-obf: the image never pre-downloaded `integration_tests` module deps, so the verifier's
  `go test` failed at setup with no network. Dockerfile fixed. Composer's earlier fail on this task was the
  checksum guard, which fires before setup, so the trial outcome stands but the task was not provable.

### Note (2026-09-18 22:00Z): ablation round 1 used a COUNT objective ("as many valid bugs as possible"), which produced only rung-1 inversions in both conditions (21 of 27/33 symbols shared, 11 byte-identical). It measures discovery of valid mutation sites, not hard-unit discovery. Round 2 uses a QUALITY objective (feature-excision units with black-box tests and a contract); see experiments/ablation_graph/RESULT2.md when written.

### Ablation round 1 verdict (2026-09-18 22:10Z) and an A4 caveat
Count objective, 1 h, mgechev/revive: NOGRAPH 33 submitted / 5 valid; GRAPH 27 / 3. Codegraph did not
improve discovery of valid mutation sites. Every submission in both conditions built, had f2p, passed alt,
failed cheat, was not flaky; the rejections were all **A4**, and A4 as applied was wrong for this repo:
revive's `test/` integration tests sit more than 2 call hops from the mutated helpers, and
`codegraph impact` defaults to depth 2. **Rule A4 amendment:** compute the impact set transitively (or to
the depth of the nearest test caller), and record the depth used; a depth-2 impact set is not a valid
reason to reject a bug whose failing test is a legitimate transitive caller. Round-1 numbers should be
re-scored with transitive impact before being cited.

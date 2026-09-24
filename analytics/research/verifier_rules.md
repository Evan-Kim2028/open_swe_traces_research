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
| A13 | The contract must be consistent with gold: every coverage row's claim is true of `gold.patch`, and every assertion the hidden suite makes is covered by a row (vague rows are surfaced, not failed) | the contract outranks the code the solver can read — inverting one coverage row flipped 7/8 passing units to fail on exactly the inverted property; 40% of the first dataset misdescribed gold; every audited double-failure was a contract defect (7/7); go-github 0/10 L2 flips | 20 of 50 dataset units (11 false rows, 9 missing-only); go-github cohort unusable until re-derived |

Note on A2/A3: zero exclusions means either the builder already produces verifiers that satisfy them, or
the checks are too weak to bite. Distinguish by ablation (see section D).

**A13 cold-repo gate.** Any repo with fewer than 3 units already in the screened dataset
(`experiments/pipeline/tasks_composerver/<repo>`) must pass A13 on each newly packaged L2
*before the batch is screened*. The judge's ~13/30 false-alarm rate is accepted there because
the prior on defects is far worse (go-github: 10 of 10 L2 failures, all spent). Warm repos
keep A13 as a backstop, not a packaging blocker. Implementation: `openswe_traces.synth.reconcile.enforce_cold_a13`,
called from `pipeline.package.package_levels`. The judge lives in `openswe_traces.contract_judge`
(OpenRouter free tier, cached under `outputs/contract_judge/`). The gate only *reads* that cache.

**Three-pass authoring** (the structural fix for A13, not a detection after the fact):

```
author     → DETAILS.md (numbered commitments) + bugreport.md (L0, B6/B7-clean)
verifier   → one property test per commitment (TestDetailNN); never reads gold or bugreport
reconciler → contract.md coverage rows, each derived from (hidden assertion, gold hunk)
```

The reconciler may read gold and the hidden tests. It must not touch `bugreport.md` and must
not weaken any hidden test. Command: `uv run python scripts/reconcile_contract.py`. See
`analytics/research/reconciled_contracts.md`.

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
| B11 | Every `DETAILS.md` commitment passes a derivability gate before verification: an independent judgement of where the commitment's answer lives (ARBITRARY/DERIVABLE/COUNTER) using only solver-visible material. ARBITRARY lines are dropped or graded shape-only; a unit cannot be verified until its DETAILS pass | the verifier is blind to gold: it grades whatever DETAILS lists, so an arbitrary commitment becomes an assertion no solver can derive, at every rung | 245 of 1560 lines weakened/dropped on the 170-unit pass (2026-09-20) |

## C. Process rules

| # | rule |
|---|---|
| C1 | Audit every failure before counting it: (a) legitimate, (b) verifier too narrow, (c) instruction insufficient, (d) infrastructure. Only (a) counts. Tally so far: (a) 4, (b) 1 family, (c) 0, (d) 12 trials (rate limit). |
| C2 | Lazy ladder: build and run the next affordance level only where the previous one fails. |
| C3 | Obfuscate identity (module path, brand strings, subsystem symbols) before measuring anything on a public repo; then verify with a control run. Result: solve times unchanged, so recall was not the driver, but keep it as insurance. |
| C4 | One base image per tree; per-task layers. (Not yet done; each task rebuilds from golang:1.23.) |
| C6 | Derive the L2 coverage table FROM the hidden tests (reconciler pass), never from a reading of gold. Author writes DETAILS.md; verifier writes TestDetailNN; reconciler writes contract.md. |
| C7 | Cold repos (< 3 prior recorded units) must pass A13 before screening. |

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

### Ablation round 2 verdict (2026-09-18 22:40Z) — quality objective, 15 min, five hardest units
NOGRAPH 2 valid units of 5, GRAPH 1 of 5, and the graph-valid unit is the same subsystem the no-graph
builder also found. Codegraph did not improve discovery of hard units on this repo with this builder
model, under either objective. Conclusion for the pipeline: keep codegraph only for the impact-set check
(A4, transitive) and for API/caller enumeration when writing black-box verifiers; do not credit it with
discovery. Caveats as before: n=1 repo, n=1 run per condition, same model.

## Where codegraph fits (decided 2026-09-18 23:50Z)

Mechanical validation only, inside `openswe-synth validate` / packaging: transitive impact-set check (A4), exported-API enumeration for the black-box check (B4), and fail-to-pass reachability. The builder agent never has to call it; discovery is builder judgment + grep. Ablation rounds 1 and 2 found no discovery advantage.

## Ladder renumbering (decided 2026-09-19 00:00Z)

The ladder is 0-indexed from the deepest FAIR level, which is fixed by fairness (a task below "bug report + reproduction command" is unspecified):

| level | solver receives |
|---|---|
| L0 | bug report + reproduction command only (unit excised) |
| L1 | gapped contract: one invariant omitted but discoverable in the repo; a pre-existing repo test guards it |
| L2 | full prose contract with coverage of every hidden assertion (formerly A0) |
| L3 | + hidden test names and one-line descriptions (formerly A1) |
| L4 | + exported signatures as stubs (formerly A2) |
| L5 | + one hidden test file (formerly A3) |
| L6 | + all tests in tree (formerly A4; equals the original in-tree-test tasks) |

Existing artifacts and reports use A0..A4 and A-1/A-2; map with L = A + 2. Directories will be renamed once the current builders finish. Nothing exists below L0 by construction.

### C6 (2026-09-19): flip points are pass rates over >= 3 attempts per level; single attempts are provisional. Cleanup must not prune images/containers while Harbor jobs run.

### C6 amendment (2026-09-19, afternoon)

One pass at a level counts as a pass. The flip point is confirmed with **two runs** at the flip level (≥1 pass) and one failing run at the level below. After an L2 pass the policy probes **L0 first**; if L0 fails it probes **L1** automatically (two runs) before spending the L2 confirmation run. Rationale: with Composer at 3–5 min per trial the marginal run is cheap, but the informative run is the lower rung, not a third confirmation at the same rung. Devin runs from earlier today used the previous 3-run rule.

## B10 — no literal digests of internal serialisations (added 2026-09-20)

A hidden test must not assert a hard-coded hash of a value the solver serialises:

```go
expectHash := "sha256:fb239e83…"          // FORBIDDEN as an oracle
if h, _ := resolver.HashReq(req, lock); h != expectHash { t.Fatalf(…) }
```

Such an assertion pins the exact byte layout of the serialisation — struct field order, `omitempty` tags, which
fields are emitted — rather than any behaviour the contract can state. It is therefore:

- **unsolvable from the contract at any rung**, and
- **unsolvable at L5 and L6**, where the test is visible: the expected digest cannot be inverted.

`helm-depresolver` is the only unit in the dataset that fails with the hidden test in its own tree, and this is why.
Its solver produced a stable, self-consistent canonicalisation satisfying every stated property, and failed on the
literal.

**Instead, assert the property:**
- equal inputs hash equally; inputs differing in a meaningful way hash differently
- the digest is stable across repeated calls and across process runs
- or compare against a digest the test itself computes from a value it also constructs

Audit the dataset with `scripts/ops/literal_digest_audit.py`. A bare hex string is usually a fixture (an address, a
key, a test vector) and is fine; an algorithm-prefixed literal used as an expected value is the violation.

---

# Verifier design: the two defect families, and which instrument catches which

*Decision record, 2026-09-20. Derived from 684 real trials, 224 authored unit-families, 392 verified ones.*

Every rule below exists because a specific defect cost specific trials. The organising finding is that the
defects fall into **two families that no single instrument detects**, and that the instrument most people reach
for first — a static linter over the task text — detects neither.

## The taxonomy

| family | what is wrong | worked example | instrument | measured |
|---|---|---|---|---|
| **packaging** | the hidden suite does not cover the excision | gold changes 4 exported functions, the suite exercises 2 — the solver must implement behaviour nothing verifies | **reachability / codegraph** | 53% precision at ≥2 orphans vs a 39% base rate; 2.7× separation in the mean |
| **contract** | the prose contradicts or omits an assertion | contract says "an existing entry for the same name/version is *replaced, not duplicated*"; the suite asserts it **appends** | **shadow implementation** | in build (`closure_SHADOW`) |

They are genuinely independent. `helm-repindex`'s five contract revisions moved trial outcomes
0/3 → 0/3 → 1/3 → 0/3 → 2/3 **with gold and the hidden suite held fixed**, so every revision has an identical
orphan count. Reachability cannot see contract defects. Equally, a perfect contract cannot rescue a suite that
tests the wrong thing.

## What does NOT work, and why it is recorded here

A static linter (`scripts/ops/task_lint.py`) was built and measured against 119 L2 units with known flip
outcomes. Base rate for "this unit will not flip" is 39%.

| heuristic | precision | verdict |
|---|---:|---|
| contract commitments counted against hidden assertions | **40%** | **no information** |
| every worked-example literal must appear in the hidden suite | **40%** | **no information** |
| B7 file/line leakage in an L2 contract | 50% | advisory |
| unresolved symbol-scrub placeholder ("the call") | 60% | gate |
| ordering stated as a direction rather than a pairwise rule | 100% (n=1) | gate |

**Counting and pattern-matching do not predict whether a contract works.** The defects that decide a unit are
semantic relations between a claim and an assertion, and reading that relation is a model's job. This is why the
contract family needs a *constructive* check rather than a static one.

## Why the shadow implementation is the right constructive check

*Corrected 2026-09-20 after review. The earlier version of this section said "nothing joins the two". That is
wrong, and the correction changes the fix.*

Both artifacts descend from gold, so **gold is a join, and one leg of it is already checked** — A1 requires the
hidden suite to pass against gold, and preflight enforces it. The missing leg is contract-to-tests:

```
                     gold
                    /    \
   author → contract.md   hidden tests ← verifier
                    \    /
             A1 checks this leg (suite vs gold)
             NOTHING checks contract vs tests — the trial is the only join
```

The consequence is practical: **we do not need a fourth artifact, we need one of the two existing ones derived
from the other.** That is the reconciler, and it is why the reconciler outranks any additional gate.

Also corrected: "no amount of checking fixes this" was overstated. The shadow implementation *is* a check.
The accurate claim is that **syntactic checking cannot** — the defects are semantic, and only an executable
reading of the contract catches them. A13 already has 19/20 unit recall on free-tier requests; its problem is
precision (13/30 false alarms), not concept.

A trial costs ~2.18M tokens. The shadow check makes the join cheap: generate a second implementation **from the
contract alone** — never from gold, never from the tests — and require the hidden suite to pass against it.

- shadow fails → either the contract did not convey what the suite requires, or the suite demands something only
  gold can produce. The failing assertion names the defect.
- shadow passes → the contract is sufficient to reconstruct the behaviour. **That is the property we actually
  want, and until now it could only be tested by spending a trial.**

Run it with a **weak, free-tier model on purpose.** If a weak implementer can rebuild the behaviour from the
contract, the contract is complete. That also answers the staffing question: strong models were never needed to
*author* tasks — they were compensating for the absence of a check.

## Rule B10 is a special case of the shadow check

B10 (no literal digests of internal serialisations) was written after `helm-depresolver` — the dataset's only unit
that fails even at L5, where the hidden test sits in the tree, because a `sha256:` literal cannot be inverted.
B10 is a regex for one unsatisfiable shape. The shadow check catches **every** assertion only gold can satisfy,
of which the digest literal is one instance. Keep B10 as a fast pre-filter; treat the shadow as the general rule.

## Ordering of gates, cheapest first

1. **Deterministic, free, milliseconds** — B10 digest oracles, A12 gold-touches-tests, missing hidden suite,
   unresolved scrub placeholders, ordering-as-direction. (`scripts/ops/task_lint.py`)
2. **Reachability, free, seconds** — orphan count: gold-touched exported functions the suite never names.
   Blocks the packaging family. (`scripts/cg_coverage.py`, in build)
3. **Shadow implementation, free-tier model, ~1 request per unit** — blocks the contract family.
   (`scripts/shadow_gate.py`, in build)
4. **A13 contract-vs-gold judge, free tier** — 19/20 recall, 13/30 false alarms; advisory except on cold repos.
5. **In-image preflight** — bare fails with assertions, gold passes, cheat fails.
6. **Only then a trial.**

The ordering matters more than any individual rule: **9.5 trials were spent per certified-hard unit, and most of
that was discovering defects rather than measuring difficulty.** Gates 1–4 are free; a trial is 2.18M tokens.

## Post-trial rules stay as they are

B9 (hack audit) runs on every passing attempt via `scripts/ops/post_sweep.sh`. It produced 12 false positives
before it was trustworthy; each fix moved a rule from *inferring intent from surface text* to *checking the thing
itself* — did verifier content actually come back, did the network command actually succeed, is this file
actually in the hidden manifest, was it created or modified. Current state: **1 real void in 684 trials.**

`post_sweep.sh` also fails closed. A unit whose trials all errored is recorded `UNKNOWN`, never `0`. Conflating
"did not run" with "failed" is what made go-github read as a cohort of hard failures for hours, on 30 trials that
burned zero tokens.


## The shadow gate is a trial by another name — and that is its main risk

A shadow implementation is a weak model solving the unit at L2 at construction time. It is worth doing because it
is free rather than 2.18M tokens, but the same caveats apply, and one is serious:

> **A weak model will often fail a correct contract**, precisely on units built to be hard for a strong one. Devin
> solves only 2 of 7 units Composer fails at L0. A shadow failure is therefore confounded between "the contract is
> insufficient" and "this unit is hard".

**The discriminator is the failing assertion, not the verdict.** For each assertion the shadow fails, ask whether
the contract states the commitment that assertion checks:

| shadow fails an assertion the contract… | diagnosis | action |
|---|---|---|
| never states | **contract defect** — the reconciliation ladder's signature; every one of its fifteen trials failed this way | repair the contract |
| does state | implementation difficulty, not a contract defect | leave the contract alone |

So the shadow's verdict is advisory and its **fail-side attribution is the product**. Report per-assertion
outcomes, never a bare pass/fail, and calibrate the confusion matrix against units with known trial outcomes.

## The separation was deliberate; any fix must preserve it

Author and verifier are separate sessions on purpose: the suite must not be written by whoever wrote the
instruction, or the instruction leaks into the tests. **Independence from the author was bought at the cost of
independence from the truth.** The reconciler preserves the first — it reads gold and the tests but never touches
`bugreport.md`, and the author still never sees the suite — while restoring the second. Any future fix must clear
the same bar.

## A3 after repair: the frozen cheat patch goes stale

A3 requires that a patch special-casing the contract's examples fails the hidden suite. **Contract repair does not
regenerate the cheat patch**, so A3 is re-checked against an adversary that predates the repair. Measured on
`helm-repindex`: `cheat.patch` is byte-identical (sha `331b4011…`) before and after five rounds of repair.

Completeness and anti-cheat pull in opposite directions, so the repair must be measured, not assumed safe:

| contract | words | literals | grounded in the suite | trial |
|---|---:|---:|---:|---|
| original | 445 | 22 | 7 (32%) | 0/3 at L0, L2, L3, L4 |
| **round 5 (hand)** | **1251** | **15** | **12 (80%)** | **2/3 at L2** |
| reconciler output | 1033 | 46 | 24 (52%) | 0/3 |

The repair that worked grew the prose 2.8× while the literal count **fell**, and the grounded fraction went
32% → 80%. The generated contract that failed did the opposite: three times the literals, half of them
ungrounded. Every ungrounded literal is fresh cheat surface.

**Rules:**
1. **Repair adds prose, not literals.** Literal count must not rise; the grounded fraction must. Enforced as
   `A3-SURFACE` in `scripts/ops/task_lint.py`.
2. **Regenerate the cheat patch from the repaired contract and re-run preflight.** A frozen cheat cannot detect a
   leak the repair introduced.

## What none of this fixes

All of the above is **validity, not difficulty**. Closure size and assertion count both failed as difficulty
levers, and Devin solving 2 of 7 certified-hard units shows "hard" is partly a property of the screening model;
cross-model intersection is a measurement, not a knob. A clean dataset is the **precondition** for answering what
controls difficulty, not the answer.

## The gate model is Composer, not OpenRouter free tier

Four gate jobs were pointed at OpenRouter's `:free` model variants and spent an hour queueing behind that
provider's shared per-model rate limits — while the account had spent **$0.00** against a $1 limit. The 429s were
never our budget; they were the free pool. Throttling the gate stack to protect an unspent dollar, while spending
$167 on the Composer trials the gates exist to prevent, was backwards.

`scripts/ops/ask_composer.py` replaces it: one-shot `cursor-agent -p` calls, cached by prompt under
`outputs/composer_cache/`, resumable, with a small thread pool for batches.

```python
from ask_composer import ask, ask_many
text = ask(prompt, cache_key="a13/<unit>/omissions")
out  = ask_many([(key, prompt), ...], workers=3)
```

Two traps worth recording, both cost a debugging cycle:
- the CLI needs the credentials from `/home/evan/Documents/eval_tasks/.env`, and **an inherited
  `CURSOR_API_KEY` from an earlier shell can belong to a different, free-plan account** — which surfaces as
  `ActionRequiredError: Named models unavailable. Free plans can only use Auto`. The env file must win.
- run it in an empty temp cwd, or the agent wanders into the repo and starts using tools.

### The economics, measured

A realistic gate call — the full L2 contract plus the entire hidden suite, asking which asserted commitments the
contract fails to state — is **~6,100 tokens and 42 seconds**. A Composer *trial* is **2.08M tokens**.

**A gate call is ~340× cheaper than the trial it prevents.** Gating all 392 verified families costs roughly
2.4M tokens — about one trial.

### It works on the hardest known case

Run against `helm-repindex`'s original contract (0/3 at L0, L2, L3 and L4), one call recovered the commitments it
took me five hand rounds and fifteen Composer trials to find, and flagged the conflicts explicitly:

- the download URL is always base plus **basename**, including when the path is already an absolute URL
- adding the same name and version again **appends**, it does not replace *(the original contradiction)*
- add must reject an empty name and a non-semver version, but must accept metadata with no apiVersion
- an unknown name yields a **dedicated** not-found error, not merely any error
- the existence check agrees with lookup exactly

That is the contract family's detector working for 6k tokens. It does not replace the shadow implementation —
the shadow proves *sufficiency* by reconstructing behaviour, while this proves *coverage* by reading — but it is
cheap enough to run on every unit at packaging time, and it found the real defects unaided.

## Discrimination, not difficulty — and where the answer lives

*Added 2026-09-20 after review. This changes the repair policy, so read it before running any repair.*

A good measurement item separates strong solvers from weak ones. An item that separates lucky from unlucky has
the same difficulty number and **no value**. Our audit was measuring difficulty and calling it quality.

Every missing commitment falls into one of three kinds, and the kind is decided by **where the answer lives**:

| kind | where the answer lives | what is broken | action |
|---|---|---|---|
| **ARBITRARY** | nowhere | the **test** over-specifies | weaken or delete the assertion; **do not** write the literal into the contract |
| **DERIVABLE** | in the repository | the contract under-specifies | repair the contract — the unit was always sound |
| **COUNTER** | in the repo, but the obvious reading is wrong | the contract under-specifies | repair the contract **and flag the unit as high value** |

Drawn from our own audit output, so the boundary is concrete:

- **ARBITRARY** — "default error messages must be the exact literals `Parse error`, `Invalid request`";
  "text decode failures must use Go's pointer type spelling `*map[string]interface {}`";
  "the `fastBackoffBySkipSleep` failpoint must exist" (test scaffolding);
  "insertion must appear before deletion **(suite sanity)**" — the audit annotated that one itself.
- **DERIVABLE** — "Len and Size must never be negative"; "domain labels must reject underscore";
  "when the node's `Labels` field is nil, the role is the empty string".
- **COUNTER** — "a `method` set to JSON `null` must be treated as invalid, not as absent". The best kind: it
  separates solvers that read the code from solvers that pattern-match their training.

### Why this is a trap and not a refinement

Writing an exact error string into a contract **makes the unit flip**. It also produces an item that every
solver either copies or misses for the same non-reason. A repair loop optimising for "does it flip now" will
manufacture those by the dozen, the certified-hard count will rise, and our own success metric will applaud.
**A unit whose gaps are mostly ARBITRARY must be reported as low-discrimination and its assertions weakened,
even though repairing the contract would have flipped it.**

`scripts/ops/contract_gap_read.py` now labels every gap with its kind and emits a per-unit verdict of
`repair-contract` or `low-discrimination`. `scripts/ops/classify_gaps.py` classifies an existing audit.

### The fairness test, and what it says about L0

> **A unit is fair at a rung if a competent engineer holding only that rung's information would _produce_ the
> graded behaviour — not merely could guess it.**

The excision itself is not the problem; cutting out an implementation and asking for it back is the standard
construction. What we do beyond that is **delete the upstream tests that documented the behaviour — about 82% of
the ones the contract cites — and then grade on that behaviour without restoring the information anywhere at
L0.** That is not withholding difficulty. It is withholding the answer to a question that has no derivable
answer.

By this test **L2 is fair** — the flip rate is high and the residual failures are ordinary coding bugs — and
**L0 is unfair for a large share of units.** That reframes the ladder: L0 is not "the hard rung", it is "the
rung where we deleted the documentation". A unit whose gaps are all DERIVABLE or COUNTER may be genuinely fair
at L0; a unit with any ARBITRARY gap is unfair at **every** rung until the assertion changes.

**This is a fix to authoring taste, not to the ladder.** The ladder is fine. What has to change is which
behaviours a verifier is allowed to grade.

## A3 is advisory; B9 is the gate. Filter on demonstrated cheating, flag on theoretical cheatability.

*Added 2026-09-20 after A3 produced a confident wrong answer on 14 of 14 units.*

Two different questions had been conflated:

| | question | instrument | evidence |
|---|---|---|---|
| **A3** | is this task cheatable *in principle*? | a cheat patch we construct | an adversary **we invented** |
| **B9** | did the solver *actually* cheat? | trajectory + patch audit on every passing attempt | what actually happened |

**B9 found 1 real hack in 684 trials.** Solvers overwhelmingly do not take the shortcut even where
one exists. A cheatable-but-uncheated unit is a weaker measurement, not a worthless one.

**The rule: filter on demonstrated cheating, flag on theoretical cheatability.** A unit is removed
from the dataset when B9 shows a passing attempt was obtained by cheating. A unit whose cheat patch
passes is *flagged* — recorded as lower-confidence, re-examined, not discarded on the strength of
an adversary of our own making.

### A3's verdict is meaningless until the cheat patch is audited

The repair loop reported 5 of 14 units as "cheat now passes", and I drew a structural conclusion
from it about a tension between completeness and anti-cheat. That conclusion was wrong. Every one
of those patches was a **full working implementation**, 93%–176% the size of gold: `flshared`'s
"cheat" was 283 lines building real maps in `newShared()`; `reqidgen`'s was a proper options loop
using `crypto/rand`. A model asked to write a cheat had simply written the code.

Worse, the 9 units that came back `preflight: ok` are equally uninformative — their cheat "failed"
by being a *bad* implementation, not by being a rejected shortcut. **A3 said nothing about any of
the 14.**

`scripts/ops/cheat_validity.py` must pass before an A3 verdict is read:

- **size ratio** — cheat added-lines ÷ gold added-lines. **≥ 0.6 means it is doing the real work**
  and is not an adversary. A genuine cheat is small and lumpy: a hardcoded table, an early return.
- **seed generalisation** (`--docker`) — re-run under `AUDIT_HIDDEN_SEED`. A real implementation
  passes both seeds; a patch that memorised the shipped seed's examples passes only the original.
  Reuses the B5 machinery in `pipeline_ext.hack_audit`.

The generator prompt is also at fault and must change: *"write a cheat patch"* produced working
code fourteen times out of fourteen. It has to say **special-case the contract's worked examples
and nothing else** — no general algorithm, no real data structures.

### The meta-rule this is the fourth instance of

Every instrument needs its own validation before its verdicts are believed. Today: the B9 audit
produced 12 false positives before it was trustworthy; the static linter scored at the base rate;
three of four classification gates came back advisory; and now A3's adversary turned out not to be
an adversary. **A gate that has not been calibrated against known outcomes is a source of
confident wrong answers, not a safeguard.**

### B11 note (2026-09-20): the DETAILS gate ran over all 170 authored units

`scripts/details_gate.py` (worktree `oswt-DETAILSGATE`, package `openswe_traces.details_gate`)
judged all 1560 DETAILS.md lines, one Composer call per unit, on solver-visible material only
(api.md, bugreport.md, excision diff with removed lines and test hunks stripped — the removed
lines are gold). Result: 245 lines ARBITRARY (15.7%), 1075 DERIVABLE, 240 COUNTER; actions 199
grade-shape-only, 46 drop. 166 units pass, 4 low-discrimination (gin-recoverymw,
kops-hashparse, gin-colorfmt, gin-mailfmt). `run_verifier` and `package_levels` now refuse a
DETAILS unit without a passing `_author/details_gate.json` (rule id `DETAILS`).

Calibration vs the author `Inferable:` annotation: authors catch the blatant cases (174 of 245
arbitrary lines were annotated `no`) but miss every case where a derivable shape wraps an
arbitrary literal — 54 annotated-derivable lines are actually arbitrary. In the other
direction, 79.5% of author-`no` lines are derivable once the bug report's expected-vs-got
clauses (B6) count as solver information; the author's `no` meant "not in the excised tree",
the gate asks "producible at the rung". Sharp case `goa-httpencoding`: author flagged 1 of 13
lines; the gate finds 5 arbitrary, matching the audit's under-specification finding from the
other side.

Outcome check on the 48 units with L0+L2 verdicts: no predictor beats the 30.4%
double-failure base rate (best precision 0.333 at n=3; `any_ARBITRARY` 0.171 at the subset's
own 16.7% rate). As a predictor the gate is **advisory** — the fourth gate to land there. As a
correctness fix it stands regardless: 245 assertions would otherwise grade literals no solver
can produce, at every rung. Writeup: `analytics/research/details_gate.md` (worktree copy);
records `outputs/details_gate.jsonl`.

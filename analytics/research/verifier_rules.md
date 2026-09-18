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

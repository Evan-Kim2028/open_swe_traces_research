# Architecture

This repository builds software-engineering tasks whose difficulty is controlled, runs coding
agents on them, and records at which level of information each agent starts to succeed. The
write-up is [*The Information Ladder: Measuring Model Capabilities*](https://evan-kim2028.github.io/evan_writings/writings/difficulty-is-an-information-gap/), and the task set it builds is called LadderBench.

Read this page first, then [`OPERATIONS.md`](OPERATIONS.md) before running anything.

## Vocabulary

| Term | Meaning |
|---|---|
| **unit** | One task, named by its base (`advrefs`). Built by excising working code from a real commit, so the gold patch is known and hidden tests verify a fix. |
| **rung** | How much the solver is told. A staged unit is `<base>-L<k>`. |
| **trial** | One agent run on one unit at one rung: a directory under `experiments/dose_response/jobs/<job>/<trial>/` holding `result.json`. A trial that raised before the verifier ran is an error, not a failure. |
| **solver** | The model family behind a trial: `composer`, `devin` or `grok` (`ladder.ledger.solver_of`). |
| **certificate** | For one solver, the lowest rung ≥ L2 it passes after failing L0. `rung_established` is true when the rung directly below also has a recorded failure; otherwise the certificate is an upper bound ("flips by L*k*", not "needs L*k*"). |
| **cohort / sweep** | A batch of staged units under `experiments/dose_response/<sweep>/`, run by one driver. |

### Rung numbering

The code and the write-up number the ladder differently. Rung keys are baked into job
directory names and ledger rows, so the code keeps its numbering until the renumbering in
[`tech_debt_level_numbering.md`](tech_debt_level_numbering.md) is done.

| Solver is given | Code | Write-up |
|---|---|---|
| bug report | `L0` | L1 |
| gapped contract (retired) | `L1` | not shown |
| full prose contract | `L2` | L2 |
| hidden test names | `L3` | L3 |
| signatures and stubs | `L4` | L4 |
| one restored test | `L5` | L5 |
| all tests | `L6` | L6 |

## Trial lifecycle

```
author ──▶ stage ──▶ guard ──▶ admit ──▶ harbor run ──▶ ledger ──▶ certificate
                       ▲                                   │
                       └──────────── escalate ◀────────────┘
```

1. **Author.** `openswe_traces.pipeline` and `openswe_traces.authoring` turn a repository
   commit into a unit: an excision patch, a contract, hidden tests and a task package per rung.
2. **Stage.** A sweep directory under `experiments/dose_response/` holds the units to run.
3. **Guard.** `ladder.guard.decide(unit, ledger, solver)` refuses any trial whose verdict is
   already on record (`RUNG_TRIAL_CAP = 1`). `ladder.match.decide` refuses a trial by a
   solver that would produce a certificate spanning two solvers.
4. **Admit.** `scripts/ops/sweep_seq.sh` launches harbor. The Devin concurrency cap is
   enforced there, and only there, under a `flock`. `ops.slots` counts occupancy.
5. **Run.** `harbor run` writes the trial directory. The verifier's reward lands in
   `result.json` at `verifier_result.rewards.reward`.
6. **Ledger.** `ladder.ledger` is the only reader of trial outcomes. Every count in the
   write-up derives from it.
7. **Escalate.** A unit that fails L0 and L2 climbs one rung at a time
   (`ladder.escalate.next_rung`) until it passes or fails the top rung.

`scripts/ops/supervisor.sh` and `scripts/ops/orchestrate.py` drive steps 2 to 7 on a timer.
`scripts/ops/devin_watchdog.sh` checks Devin's share and puts it back on course.

## Code map

Library code lives in the package `src/openswe_traces/`. Commands live in `scripts/`.

| Package | Owns |
|---|---|
| `openswe_traces.ladder` | Trial outcomes and certificates (`ledger`), admission rules (`guard`, `match`, `stability_gate`), escalation (`escalate`), second screening, ladder purity, quarantine |
| `openswe_traces.ops` | Running agents: connectors (`agents`, `agent_session`), capacity (`slots`, `devin_cap`), budgets, unit discovery (`roots`), cohorts, disk and container hygiene |
| `openswe_traces.reports` | Human-facing reports: `status`, `ladder_matrix`, `dashboard`, `token_cost`, `devin_usage`, `rolling` |
| `openswe_traces.analysis` | One-off instruments behind findings in `analytics/research/` |
| `openswe_traces.authoring` | The unit factory's operational steps: autogen, harvest, staging, pre-audit, lint, repair |
| `openswe_traces.pipeline`, `pipeline_ext` | Task construction, verification rules, ladder policy |
| `openswe_traces.synth`, `irt`, `difficulty`, `rungs`, … | Earlier trace analytics on Open-SWE-Traces (see the README) |
| `openswe_traces.paths` | `REPO` and the fixed directories under it |

`scripts/ops/<name>.py` is a shim for each module moved into the package. It keeps
`uv run python scripts/ops/<name>.py` and `import <name>` working for the shell drivers,
frozen sweep copies and running daemons, and hands callers the package module itself. New
code imports the package: `from openswe_traces.ladder import ledger`.

The shell drivers (`*.sh`), `orchestrate.py`, `pipeline_health.py` and `grok_ladder.py` stay
in `scripts/ops/`: they are commands, and two of them change directory when imported.

## Where the write-up's numbers come from

| Number in the write-up | Source |
|---|---|
| Certified tasks, and the level each binds at | `ladder.ledger.certificates()`; per model, `certificates_by_solver()` |
| Tasks with independent curves from two solvers | `ladder.ledger.report_multimodel()`, `ladder.purity` |
| Trials, errored trials, tokens | `ladder.ledger.summary()` |
| Cost by model and by level | `reports.paper_numbers`, from each trial's `cost_usd` (Composer and Grok). Devin bills in ACUs, so its row comes from the account export |
| Second-screen agreement between models | `ladder.second_screen --report` |
| Runs and dollars per certificate | `reports.paper_numbers`: runs with a verdict, and total cost including the Devin export, over `len(certificates())` |

`uv run python -m openswe_traces.reports.paper_numbers` prints every number the write-up
states, labelled with its phrase, in the write-up's level numbering. The headline block alone
is `uv run python scripts/ops/trial_ledger.py`.

## Verifying a change

- `uv run pytest` runs the tests. `tests/test_ladder_certificates.py` covers the certificate
  and escalation rules on a synthetic jobs tree.
- `scripts/dev/ladder_golden.py` freezes the ledger's inputs and dumps every certificate and
  guard decision. Two dumps from one snapshot, before and after a change, must be identical
  unless the change means to alter a decision.
- `scripts/dev/moved_diff.py` and `scripts/dev/import_check.py` audit a module move.

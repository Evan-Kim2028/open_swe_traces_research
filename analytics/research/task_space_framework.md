# Task-space framework: discovery × difficulty, verifier as the continuity check

2026-09-18. Reference for how the pieces relate. Figure: `figures/task_space_framework.png`
(regenerate from the snippet at the bottom; update the rows as results land).

## The two axes

```
 discovery axis (y)                       ┌────────────────────────────────────────────┐
 which unit of the repo,                  │  ✕ ✕ ✕ · ·   dynamic pipeline  [Composer]   │
 found + validated by codegraph           │  ✕ · · · ·   dynamic pipeline  [Devin]      │
 (impact set + caller enumeration;       │  · · · · ●   codec, black-box  [Composer]   │
  cross-module edges)                     │  ● · · · ·   backoff, property [Composer]   │
                                          │  · · · · ●   mutations r1-7    [Composer]   │
                                          └────────────────────────────────────────────┘
                                             A0    A1    A2    A3    A4
                                             difficulty axis (x): affordance level
                                             = how much of the unit's information is withheld
                                               A0 contract only … A4 all tests in tree
```

- **Discovery (y)**: *where* a coherent, tested, well-connected unit lives and *whether* a proposed
  change/removal is valid. Codegraph's job. Says nothing about how hard the task is. The graph
  ablation (`experiments/ablation_graph/`) decides whether the graph beats grep+gopls here.
- **Difficulty (x)**: *how much information* the solver gets about the unit. The affordance ladder
  A0..A4. The only lever that moved outcomes for any model above the free tier. Graph-structural knobs
  (hops, sites, decoys, impact size) did not.

## The verifier: continuity check over the space

The verifier does not sit on either axis. It is the condition that makes a point on the grid a valid
measurement at all: gold passes, alternative fix passes, cheat fails, black-box, property or dynamic
gate, no network, tests checksum-guarded (`verifier_rules.md` A1–A10, B1–B8). Where it fails, the
grid has a discontinuity: a fail that is not about ability (white-box names at A0/A1), or a pass that
is not about ability (web fetch of the upstream file, tests that are a complete spec). Every failure
is audited (rule C1) so discontinuities are found and repaired rather than read as capability.

## The measurement: flip point per (unit, model)

Read each row left to right; the first ● is the flip point. Difficulty is a property of (unit, model),
not of the unit alone. Current flip points on client-go:

| unit | verifier family | Composer 2.5 | Devin swe-2-high | notes |
|---|---|---|---|---|
| dynamic pipeline | race + throughput gate | **A3** | > A0 (A1 not run) | fails A0-A2 (deadlock at A2), passes A3 in 5.4 min; Devin one test short at A0 |
| codec excision | black-box properties | **A1** | **A0** (37 min) | Composer: A0 fail, A1 pass 7 min; Devin: A0 pass, clean. First measured capability gap. |
| backoff | property verifier | A0 | – | properties in words + 3 examples sufficed |
| any mutation rung 1–7 | in-tree example tests | A4 (trivially) | – | 26/26; also 36/36 Grok, contaminated |

Flip points are the per-item numbers an IRT fit consumes: a unit's difficulty for a model family is
"the lowest affordance at which the model passes". Two models on the same grid give a capability
difference that is independent of which repo or unit was chosen, as long as the verifier held.

## What transfers across repos and languages

- The ladder definition (A0..A4) and the verifier rules: language-agnostic.
- The flip points: do NOT transfer; they are re-measured per repo. Same ruler, new reading.
- Per repo you rebuild: the black-box verifier and the prose contract for each unit; the obfuscation
  map if the repo is public.
- Open: whether the shape of the grid (which families are hard) looks the same on a second large Go
  repo. Candidates from the bank with a real mid-difficulty band: argoproj/argo, knative/client,
  google/go-github.

## Regenerate the figure

`uv run python scripts/task_space_figure.py` (to be added; the current figure was produced inline —
rows are listed in this file's table).

## Ablation outcome (2026-09-18)

Both ablation rounds on mgechev/revive (count objective, then five-hardest-units objective) found no discovery advantage for codegraph over grep+gopls with the same builder model. The discovery axis is therefore "builder judgment + validity checks"; codegraph remains the tool for the impact-set check and for enumerating the exported API a black-box verifier must target. See `verifier_rules.md` and `experiments/ablation_graph/RESULT*.md`.

## External datapoint: Terminal-Bench (2026-09-19, see `tb4_rung_mapping.md`)

Mapping the L0–L6 ladder onto Terminal-Bench by instruction content puts almost every task at L1 (goal +
unstated requirements, hidden tests), so a benchmark does not vary the information axis and cannot test it.
Pooled frontier pass rates were non-monotone across the few non-L1 tasks (L1 28%, L2 36%, L4 42%, L5 28%),
and verifier class/domain dominated (dynamic-gate 30%, example tests 29%, property/fuzz 20%). With ~65 trials per
task, per-task variance was high, which is rule C6 seen from the other side. Reading: the ladder is a construction
instrument (choose the information level when building a task), not a lens for explaining existing benchmarks.
Whether the rung predicts measured difficulty on a large task set with stable per-task rates is the
Open-SWE-Traces mapping (`openswe_rung_mapping.md`, pending); any revision to the framework waits for it.

## Score test outcome (2026-09-19)

The statefulness score did not predict L2 outcomes on four fresh units (both top-scored units passed 3/3, as did both bottom-scored). Dropped as a manufacturing lever; see FAILURE_AUDIT_A0.md.

## External datapoint: Open-SWE-Traces (2026-09-19, see `openswe_rung_mapping.md`)

42k real issues with ~12 rollouts each. A heuristic rung label (validated at only 24.5% exact / 63% ±1 against
hand labels) puts issues mostly at L0–L2 and L4 (signatures/interfaces quoted). Rung-level mean solve rate rises
with rung (Spearman +0.86 over 5 populated rungs) but adding rung features to the task-only model gains
AUC +0.005, and leakage of patch symbols in the issue text has no effect on solve rate. Reading, together with
the Terminal-Bench datapoint: on natural tasks with hidden example-test verifiers, the information given in the
issue text is not what drives difficulty (or our heuristic cannot measure it). The ladder's effect is real only
where we control it (client-go: replicated flips at L5 and >L3). Framework decision: keep L0–L6 as a
construction instrument for synthetic tasks; do not claim it as a general difficulty axis for natural tasks.

## Replication (2026-09-19)
Single-attempt flip points mislabeled 1 of 2 units. All flip points are now pass rates over 3 attempts (C6).

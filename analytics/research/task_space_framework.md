# Task-space framework: discovery × difficulty, verifier as the continuity check

2026-09-18. Reference for how the pieces relate. Figure: `figures/task_space_framework.png`
(regenerate from the snippet at the bottom; update the rows as results land).

## The two axes

```
 discovery axis (y)                       ┌────────────────────────────────────────────┐
 which unit of the repo,                  │  ✕ ✕ ✕ · ·   dynamic pipeline  [Composer]   │
 found + validated by codegraph           │  ✕ · · · ·   dynamic pipeline  [Devin]      │
 (callers, callees, impact set,           │  · · · · ●   codec, black-box  [Composer]   │
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
| dynamic pipeline | race + throughput gate | > A2 (A3 running) | > A0 (A1 not run) | deadlock at A2; Devin one test short at A0 |
| codec excision | black-box properties | A0 running | – | A0 white-box fail confirmed behavioral by black-box re-score |
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

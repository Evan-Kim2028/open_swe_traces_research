# The dataset is multi-model, by design

**Decided 2026-09-21.** Tasks are trialled by whichever solver has capacity — Composer 2.5
(metered, fast), Devin swe-2-max (free, capped at 4, slow), Grok 4.6 (burst). A certificate
records which solver it binds for rather than pretending difficulty is solver-independent.

## Why this is a decision and not an accident

A flip certificate asserts: fails at L0 (bug report only), passes at L2 (full contract).
That isolates the *affordance* — what the contract supplied — only if the same solver did
both. When L0 failed on one model and L2 passed on another, the flip may instead reflect a
capability gap between the models.

Measured at the time of the decision: **15 of 139 certificates (11%) were cross-solver**,
almost all `L0=composer, L2=devin`. And the solvers do differ — L0 solve rate was 39% for
composer against 12% for grok, so a task hard for one is not automatically hard for another.

The alternative was a single-solver dataset: all trials on Composer, Devin confined to
verification and authoring. That costs roughly 500M tokens for the ready backlog alone —
five times the budget spent to date — and buys a narrower claim ("hard for Composer 2.5")
rather than a broader one.

## What is recorded

`trial_ledger.certificates()` returns, per unit:

- `l0`   — solvers that FAILED it at L0
- `l2`   — solvers that PASSED it at L2
- `kind` — `single` when one solver did both (the flip isolates the affordance),
           `cross` when it did not (a real certificate, a weaker claim)
- `binds_for` — the solvers the certificate holds for

`ledger_by_solver()` gives the full base -> solver -> rung -> rewards view.
`ledger()` is unchanged, so existing callers keep working.

## How to read the dataset

- **single-solver certificates** are the strong claim: this task is hard for *that* model,
  and the contract is what makes it solvable.
- **cross-solver certificates** are kept and labelled, not discarded. They are evidence
  about a capability gap and are legitimate tasks; they just do not isolate the affordance.
- Reporting that collapses both into one "certified" number is hiding the distinction.
  `trial_ledger.py` prints the split by default.

## Consequence for solver assignment

Verification and authoring do not measure difficulty, so model identity cannot confound
them. Devin belongs there by default: it is free, and those are the two most expensive
non-trial stages. Trials are where the solver matters, and where the label earns its keep.

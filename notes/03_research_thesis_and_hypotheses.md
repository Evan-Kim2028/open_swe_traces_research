# Research thesis and hypotheses

## Thesis

Most value in Open-SWE-Traces is not row count — it's that each row is a **full trial** (conversation + model patch + gold patch + pass/fail). The open question:

> **Which trajectories are worth learning from, and when in a run can you tell?**

This is the **pre-SFT scoring layer** before spending GPU on training.

## Hypotheses (testable on DuckDB)

### H1 — Early predictability

You can guess `resolved` from trace progression **before the run ends**.

- Signals: first edit turn, first test run, cumulative tool errors, (analysis-only) gold file overlap
- Test: classifier on `turn_sample` at progress buckets 25/50/75/100%
- Metric: AUC vs `% through run`

### H2 — Success/failure differ in process, not just patch

Wins vs losses differ in **when** agents edit, run tests, and hit errors.

- Early sample: successes ~11 edits at 75%; failures ~23 edits at 75%

### H3 — Failed traces have training value (with caveats)

Trajectories with `resolved=0` teach exploration/tool use — but **naïve inclusion hurts** unless steps are masked.

- **For:** Open-SWE-Traces ablation (full > resolved-only); Step Rejection FT
- **Against:** SWE-smith (resolved-only); naïve unresolved inclusion degrades (SRFT paper)

### H4 — Trace quality is multi-dimensional

Good trace ≠ just `resolved=1`. Also: efficiency, no cheating, plausible patch, gold file overlap.

- Rubric dimensions: outcome, efficiency, bad-behavior signals, patch similarity

### H5 — Harness/teacher slices behave differently

mini-swe-agent vs openhands; Qwen vs MiniMax differ in turn length, resolve rate, unknown labels.

## Evaluation ladder (before GPU SFT)

```
1. OUTCOME LABELS     resolved, empty patch?, gold vs model overlap
2. PROCESS METRICS    turns, edits-by-k, tests, tool errors
3. QUALITY SCORE      composite rubric
4. STEP MASK MAP      which turns get loss=0 in training?
5. PROXY EVAL         CE loss / first-action match (small models)
6. DOWNSTREAM EVAL    SWE-bench resolve rate (when GPU available)
```

## Concrete deliverables

1. Dataset profile (by language/harness/teacher)
2. Quality rubric + ranked sample
3. Early-prediction curve (AUC vs turn k)
4. Training subset recommendations
5. *(Optional)* tiny SFT ablation on Kaggle/AMD

## What NOT to do at our scale

- Don't unnest 500k messages repeatedly (use `trial_summary` + sampled `turn_sample`)
- Don't treat 511k rows as 511k independent eval trials
- Don't assume download size = eval coverage

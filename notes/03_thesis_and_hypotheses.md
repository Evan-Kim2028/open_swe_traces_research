# Thesis and hypotheses

## Thesis

The value in Open-SWE-Traces is not the row count — it is that each row is a **complete trial** (conversation + model patch + gold patch + pass/fail). The open question:

> **Which trajectories are worth learning from, and when in a run can you tell?**

We are not reproducing NVIDIA's 30B SFT. We are building the **pre-SFT scoring layer**: filter, rank, and predict outcomes from trace structure.

## Core hypotheses

### H1 — Early predictability

`resolved` can be predicted from trace progression **before the run ends**.

Signals: first edit turn, first test run, cumulative tool errors, file overlap with gold (analysis-only oracle).

**If true:** early-stop during rollout; cheap labeling of "doomed" runs.

### H2 — Success/failure look different in the trace

Wins vs losses differ in **when** agents edit, run tests, and hit errors — not just final patch quality.

Early sample (15 trials): successes had fewer cumulative edits by 75% through run; failures kept editing and erroring.

### H3 — Failed traces still have training value (with caveats)

Trajectories with `resolved=0` teach exploration/tool use **if** bad steps are masked, not naïvely imitated.

- NVIDIA: full corpus beats resolved-only ([§4.3](https://arxiv.org/html/2606.16038v1))
- SRFT: naïve inclusion of failures **hurts**; critic masking **helps** ([2605.10674](https://arxiv.org/pdf/2605.10674))

### H4 — Trace quality is multi-dimensional

Good trace ≠ just `resolved=1`. Also: efficiency, no cheating, plausible patch, gold file overlap.

Related: [SWE-Prime](https://arxiv.org/abs/2608.27449) — top 10% scored successes beat all resolved.

### H5 — Harness/teacher slices differ

mini-swe-agent vs OpenHands, Qwen vs MiniMax differ in turn length, resolve rate, unknown labels.

Observed: OpenHands has many `resolved=-1`; mini-swe-agent ~41% resolved on Python in partial EDA.

## What "done" looks like

1. Dataset profile (resolve rates by language/harness/teacher)
2. Quality rubric (scored traces + spot-check)
3. Early-prediction curve (AUC at turn 10, 25%, 50%, 75%)
4. Training subset recommendations
5. *(Optional)* small SFT ablation when GPU available (Kaggle / AMD)

## Analysis ladder

```
1. OUTCOME LABELS     resolved, empty patch?, gold vs model overlap
2. PROCESS METRICS    turns, edits-by-k, tests-run, tool errors
3. QUALITY SCORE      composite rubric
4. STEP MASK MAP      which turns get loss=0 in SFT?
5. PROXY EVAL         CE loss / first-action match (small models)
6. DOWNSTREAM EVAL    SWE-bench resolve rate (GPU)
```

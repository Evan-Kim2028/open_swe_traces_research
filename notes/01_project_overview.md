# Project overview

## Thesis

Most value in Open-SWE-Traces is not the row count — it is that each row is a **full trial** (conversation + model patch + gold patch + pass/fail). The research question: **which trajectories are worth learning from, and when in a run can you tell?**

Not reproducing NVIDIA's 30B SFT. Asking: **can trace structure filter, rank, and predict outcomes** so training/analysis spend goes on the right data?

## Core hypotheses (testable)

| ID | Hypothesis |
|---|---|
| H1 | Early predictability — guess `resolved` from trace progression before run ends |
| H2 | Success/failure differ in *when* agents edit, test, and error — not just final patch |
| H3 | Failed traces have training value if masked properly (NVIDIA + SRFT claim) |
| H4 | Trace quality is multi-dimensional, not just `resolved` |
| H5 | Harness/teacher slices behave differently (mini-swe-agent vs OpenHands, etc.) |

## One-sentence investigation

> When during an agent run does success or failure become predictable, and which traces are high-quality supervision — independent of benchmark score alone?

## Concrete deliverables

1. Dataset profile — resolve rates by language/harness/teacher
2. Quality rubric — scored traces + manual spot-check
3. Early-prediction curve — accuracy/AUC at turn 10, 25%, 50%, 75%
4. Training subset recommendations
5. (Optional) tiny SFT ablation when GPU credits available

## Sources

- [Open-SWE-Traces (arXiv:2606.16038)](https://arxiv.org/abs/2606.16038)
- Local EDA: `analytics/queries/003_trial_eda.sql`, `analytics/research/trace_trial_model.md`

# Thesis and hypotheses

## Thesis

The value in [Open-SWE-Traces](https://huggingface.co/datasets/nvidia/Open-SWE-Traces) is not the row count — it is that each row is a **full trial**: conversation + model patch + gold patch + pass/fail label. The open question is **which trajectories are worth learning from, and when in a run you can tell**.

This project is **not** primarily about reproducing NVIDIA's 30B SFT at scale. It asks:

> Can we use trace structure to **filter, rank, and predict outcomes** so training and analysis spend goes on the right data?

Sources: [Open-SWE-Traces paper](https://arxiv.org/abs/2606.16038), [SWE-Prime](https://arxiv.org/abs/2608.27449), [trajectory curation study](https://arxiv.org/abs/2607.17205).

---

## Core hypotheses

### H1 — Early predictability

You can guess `resolved` from trace progression **before the run ends**.

- **Signals:** first edit turn, first test run, cumulative tool errors, patch/file overlap with gold (analysis-only oracle).
- **If true:** early-stop during rollout; cheap labeling of "doomed" runs.
- **Test:** turn-level classifier at progress buckets 25/50/75/100%; AUC vs `% through run`.

### H2 — Success and failure look different in the process

Wins vs losses differ in **when** agents edit, run tests, and hit errors — not just final patch quality.

- **Early sample (n=15 trials):** successes had fewer cumulative edits by 75% through the run; failures kept editing and erroring.
- **Test:** scale `turn_sample` to 200–500 trials; compare `cum_edits`, `cum_tests`, `cum_tool_errors` by outcome.

### H3 — Failed traces may still have training value (with masking)

Trajectories with `resolved=0` can teach exploration if you **don't train naïvely on bad steps**.

- **Open-SWE-Traces:** full corpus beats resolved-only (+3–8 pts SWE-bench). Source: [2606.16038 §4.3](https://arxiv.org/abs/2606.16038).
- **SRFT:** naïve inclusion of failures **hurts**; critic/step masking **helps**. Source: [2605.10674](https://arxiv.org/pdf/2605.10674).
- **Test:** quality rubric on failures; optional small SFT ablation (resolved-only vs all vs masked).

### H4 — Trace quality is multi-dimensional

`resolved=1` is necessary but not sufficient for good supervision.

- Dimensions: efficiency, no git-hacking, patch scope, error-retry rate, representativeness.
- **SWE-Prime:** top 10% scored successful trajectories beat **all** resolved trajectories. Source: [2608.27449](https://arxiv.org/abs/2608.27449).
- **Test:** composite quality score; manual spot-check top/bottom deciles.

### H5 — Harness and teacher slices behave differently

mini-swe-agent vs OpenHands, Qwen vs MiniMax differ in turn length, resolve rate, unknown labels.

- **Test:** `catalog_resolved_rates`, stratified turn samples by harness × teacher.

---

## Success criteria (project outputs)

1. **Dataset profile** — resolve rates by language / harness / teacher / category
2. **Quality rubric** — scored traces + spot-check validation
3. **Early-prediction curve** — accuracy or AUC at turn *k* or progress %
4. **Recommendations** — e.g. which slices to SFT on, masking policy
5. *(Optional)* tiny SFT ablation when GPU credits available (Kaggle / AMD)

---

## Analysis ladder (do in order)

```
1. Outcome labels      resolved, empty patch?, gold vs model patch overlap
2. Process metrics     turns, edits-by-k, tests-run, tool errors, retries
3. Quality score       composite rubric (efficiency + style + no-cheating)
4. Step mask map       which turns get loss=0 in hypothetical training?
5. Proxy eval          held-out CE loss / first-action match (small models)
6. Downstream eval     SWE-bench resolve rate (when GPU available)
```

Maps to SWE-Prime "process quality" + SRFT/SWE-Lego masking — see [05_sft_trace_practices.md](05_sft_trace_practices.md).

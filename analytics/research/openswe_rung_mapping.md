# Open-SWE-Traces affordance ladder mapping

Generated 2026-09-19 01:26 UTC by `scripts/rung_mapping.py`.

## Method

Task text is `messages[2].content` (first user message) streamed from `traces_data/` with DuckDB (one row per `instance_id`, longest shard kept). Harness wrappers (`<pr_description>`, `<instructions>`) are stripped in `openswe_traces.rungs.strip_task_text`. Heuristic rung assignment follows the L0–L6 ladder in `analytics/research/verifier_rules.md` (L = A + 2). The verifier class here is always hidden fail-to-pass tests (SWE-bench style; rules B3/B4). `patch_has_tests` flags whether the reference/gold patch touches test paths.

Corpus: **42,413** instances. Solve rates and `b_2pl` use instances with ≥ 3 labeled rollouts (~12 attempts each; rule C6 — far more stable than single-attempt Harbor ladder readings).

## Rung distribution

| rung | instances | share |
|---|---|---|
| L0 | 10,642 | 25.1% |
| L1 | 10,524 | 24.8% |
| L2 | 8,749 | 20.6% |
| L3 | 617 | 1.5% |
| L4 | 11,789 | 27.8% |
| L5 | 79 | 0.2% |
| L6 | 13 | 0.0% |

L5/L6 (test bodies in the instruction): **92** instances (0.22% of corpus) — rare, as expected for SWE-bench-style tasks.

Reference patch touches test files on **13,932** instances (32.8%) — the PR itself changed tests, but the agent still does not receive those tests as the verifier oracle.

## Validation (n=200 stratified manual labels)

Exact agreement: **24.5%**; within ±1 rung: **63.0%**; Spearman ρ = **0.248**.

| heuristic \ manual | L0 | L1 | L2 | L3 | L4 | L5 | L6 |
|---|---|---|---|---|---|---|---|
| L0 | 1 | 21 | 27 | 0 | 0 | 0 | 0 |
| L1 | 0 | 22 | 27 | 0 | 0 | 0 | 0 |
| L2 | 0 | 0 | 20 | 0 | 0 | 0 | 0 |
| L3 | 0 | 4 | 29 | 6 | 0 | 0 | 0 |
| L4 | 0 | 11 | 21 | 0 | 0 | 0 | 0 |
| L5 | 0 | 0 | 10 | 0 | 0 | 0 | 0 |
| L6 | 0 | 1 | 0 | 0 | 0 | 0 | 0 |

## Correlation with measured difficulty

### By heuristic rung (≥ 3 labeled rollouts)

| rung | n | mean solve_rate | mean b_2pl | % all_fail | % all_pass |
|---|---|---|---|---|---|
| L0 | 8,964 | 0.385 | 0.111 | 44.0% | 17.4% |
| L1 | 8,969 | 0.406 | 0.101 | 43.0% | 20.3% |
| L2 | 7,526 | 0.553 | -0.254 | 30.4% | 33.5% |
| L3 | 519 | 0.510 | -0.181 | 34.7% | 31.0% |
| L4 | 9,955 | 0.440 | 0.015 | 39.3% | 22.1% |
| L5 | 70 | 0.594 | -0.444 | 20.0% | 35.7% |
| L6 | 12 | 0.622 | -0.618 | 25.0% | 41.7% |

### Leakage (B7 analogue) vs solve rate

Clean (no patch symbol in text): n=16,161, mean solve_rate=0.452. Leaky: n=19,854, mean=0.435. Spearman(has_leakage, solve_rate)=-0.019.

### Feature Spearman with solve_rate

| feature | Spearman | n |
|---|---|---|
| has_repro | +0.128 | 36,015 |
| has_expected_actual | +0.080 | 36,015 |
| rung | +0.076 | 36,015 |
| has_signature | +0.063 | 36,015 |
| has_stack_trace | +0.044 | 36,015 |
| has_test_names | +0.027 | 36,015 |
| word_count | -0.025 | 36,015 |
| has_leakage | -0.019 | 36,015 |
| has_test_code | +0.019 | 36,015 |

### Logistic model of resolved (trajectory level, grouped 80/20 by instance)

Task-only baseline (same features as `task_difficulty_summary.md`): held-out AUC **0.6741**. With rung features added: **0.6787** (Δ = **+0.0046**, n=285,636 labeled trajectories).

## Examples

### Clean (no patch leakage)

- `evidentlyai_evidently_pr361` (L2, solve_rate=0.7777777777777778)
  > # Duplicate column selection causes "DataFrame columns are not unique" warning in Data Drift correlation  I noticed a warning when calculating data drift metrics for a column that is also listed as a numerical feature.  ### Context When configuring `ColumnMapping`, it is possible to include the target column (or the specific column being analyzed) in the `numerical_features` list if it is numeric.…
- `pion__webrtc-1848` (L4, solve_rate=0.0)
  > Implement RTCRtpTransceiver.setCodecPreferences This will allow users to set on a per Transceiver basis their preferences around codec choice. The `MediaEngine` is a global list supported codecs still.  [WC3 spec](https://www.w3.org/TR/webrtc/#dom-rtcrtptransceiver-setcodecpreferences)   ``` setCodecPreferences The setCodecPreferences method overrides the default codec preferences used by t…

### Leaky (patch symbols in issue text)

- `0xs34n__starknet.js-508` (L2, solve_rate=0.8181818181818182)
  > decodeShortString : wrong answer of the function **Describe the bug** Hello, I noted a problem with the function `utils/shortstrings.ts/decodeShortString`.  If I use it with an hex string ('0x321....456'), everything is fine. But if the str:string is  an integer string ('1542233...56'), the function accept it, process without fail, and send back a wrong answer. ```typescript export function …
- `0xs34n__starknet.js-520` (L2, solve_rate=0.75)
  > getStarkName() should return empty string when no starkname found **Describe the bug**  Currently `account.getStarkName()` returns "stark" when address has no stark name.  This is due to `useDecoded` function in starknetId utils which concat "stark" at the end of the result from the call to the the starknet.id naming contract, even if it's empty.  https://github.com/0xs34n/starknet.js/blob/b0…

## Verdict

**Does rung predict difficulty?** Weakly. Spearman(rung, mean solve_rate) across populated rungs = **+0.857**; the logistic AUC gain from rung features is **+0.0046** on top of language/patch-size baselines. Information in the PR text explains little variance versus patch size and language (see `task_difficulty_summary.md`). Higher rungs (L3–L4: test names or signatures in the issue) do not uniformly mean easier tasks — many are feature requests with API detail.

**Versus client-go Harbor ladder:** On client-go, flip points for a mid-tier model concentrate at **L2** (full prose contract) — the first level where the contract is complete. Open-SWE-Traces issues cluster at **L1–L2** (symptom + partial/full prose from GitHub PRs) with almost no L5/L6. The ladder *definition* transfers; the *measurement* does not: these are single fixed affordance levels per instance, not per-unit flip points over ≥3 attempts per level. Measured solve rates here (~12 rollouts) are far more stable than one-shot Harbor readings (rule C6).

## Reproduce

```bash
uv run python scripts/rung_mapping.py
uv run pytest tests/test_rungs.py
```

## Per-combo breakdown (added 2026-09-19 02:00Z)

Solve rate by heuristic rung within each teacher/harness combination (instances with labeled rollouts):

| combo | L0 | L1 | L2 | L4 | L2−L0 |
|---|---|---|---|---|---|
| minisweagent/qwen36_27b | 0.325 | 0.327 | 0.514 | 0.338 | +0.189 |
| minisweagent/qwen38_27b | 0.462 | 0.479 | 0.618 | 0.515 | +0.156 |
| openhands/minimax_m25 | 0.389 | 0.403 | 0.483 | 0.462 | +0.094 |
| openhands/qwen35_122b | 0.284 | 0.316 | 0.334 | 0.356 | +0.050 |
| sweagent/minimax_m25 | 0.438 | 0.452 | 0.503 | 0.494 | +0.065 |
| sweagent/qwen35_122b | 0.442 | 0.471 | 0.498 | 0.528 | +0.056 |
| sweagent/qwen36_27b | 0.465 | 0.464 | 0.666 | 0.488 | +0.201 |

The L0→L2 gain is positive for all 7 combos and scales with model strength (Qwen3.6/3.8: +16–20 pts;
Qwen3.5-122B: +5–6). L4 (signatures/interfaces quoted) is not easier than L0 for most combos: the effect is
the jump to a complete behavioral description, not "more text". Corrected verdict: the information axis does
predict natural-task difficulty, consistently across models, but only at the L1→L2 boundary, and the heuristic
labeler (24.5% exact agreement) likely understates it.

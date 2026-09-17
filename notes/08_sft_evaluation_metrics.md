# Quantifying and evaluating trace quality for SFT

## Trajectory-level (keep/drop the whole run)

| Metric | Description | Source |
|---|---|---|
| `resolved` | Test suite pass/fail | Universal |
| Git-hacking | Agent peeked git history for answers | [Open-SWE-Traces §2.4](https://arxiv.org/html/2606.16038v1), [SWE-Lego](https://arxiv.org/pdf/2601.01426) |
| Test-file edits | Agent changed tests not source | Open-SWE-Traces filter |
| Empty patch | No substantive diff | Open-SWE-Traces filter |
| Patch scope | Files/lines changed vs gold | SWE-Prime "result quality" |
| Turn count / efficiency | Shorter paths preferred | [2607.17205](https://arxiv.org/abs/2607.17205) "Efficiency" axis |
| Error-retry rate | Penalize repeated identical errors | 2607.17205, dominant sub-metric in their study |
| Step-count ratio | Conciseness vs median | 2607.17205 |
| Action diversity | Penalize redundant loops | [SWE-Prime](https://arxiv.org/abs/2608.27449) |
| Representativeness | Don't oversample easy repos | SWE-Prime |

[SWE-Prime](https://arxiv.org/abs/2608.27449): top 10% scored successful trajectories beat all resolved trajectories (+12–24% relative on SWE-bench).

## Step-level (which tokens get loss)

| Strategy | Mechanism |
|---|---|
| Assistant-only | Mask tool/user/system tokens |
| Error-step masking | Mask tokens at failed tool calls; keep in context |
| Segment selection | Only high-value step groups contribute loss (SWE-Prime) |
| Critic masking | LLM labels bad steps → mask from loss (SRFT) |

## Pre-SFT evaluation (when model can't resolve SWE-bench yet)

From [2607.17205](https://arxiv.org/abs/2607.17205), systematic study on Qwen2.5-Coder-7B, 67k trajectories:

| Proxy | Use |
|---|---|
| **Held-out CE loss** | Primary metric at 7B scale (near-zero resolve rate) |
| **First-action ROUGE-L** | Perfect rank correlation with CE (Spearman ρ = -1.0) |
| **Quality score vs random** | Gap widens with scale: 3.6% CE diff at 2k trajectories (p=0.016) |

Framework: two-axis scoring — Efficiency + Style.

At scale: SWE-bench Verified resolve rate is the final eval.

## Post-SFT evaluation

| Benchmark | Used by |
|---|---|
| SWE-bench Verified | Most papers (500 Python tasks) |
| SWE-bench Multilingual | Open-SWE-Traces, SWE-Prime |
| SWE-bench Pro | Open-SWE-Traces (long-horizon) |
| SWE-bench Lite | SWE-Gym, R2E-Gym (smaller) |

## Inference-time scaling (beyond SFT)

| Method | Source |
|---|---|
| Outcome reward model (verifier) | [SWE-Gym](https://arxiv.org/pdf/2412.21139v2.pdf) |
| Execution-free verifier score | [R2E-Gym](https://r2e-gym.github.io/assets/paper.pdf) |
| Hybrid execution + execution-free | R2E-Gym §4.3 |

## Mapping to our DuckDB work

| Our artifact | Field equivalent |
|---|---|
| `trial_summary.resolved` | Trajectory outcome label |
| `turn_sample.cum_*` | Process quality metrics |
| Early prediction at turn k | Pre-SFT quality scoring |
| Quality rubric (planned) | SWE-Prime trajectory screening |
| `is_tool_error_turn` | SRFT / SWE-Lego error masking input |
| `004_turn_sample_early_signal.sql` | Exploratory process-quality analysis |

## Open-SWE-Traces paper ablation numbers (reference)

Resolved-only → full corpus (SWE-bench Verified, think mode): 55.3% → 58.1%  
Resolved-only → full (Multilingual, no-think): 49.6% → 57.1%

Source: [Table 5, arXiv:2606.16038](https://arxiv.org/html/2606.16038v1)

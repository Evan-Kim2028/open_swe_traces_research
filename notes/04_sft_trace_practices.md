# How the field uses traces for SFT

## Standard pipeline

```
Generate trajectories → Label outcome → Filter/score → Format for SFT → Train → Eval on SWE-bench
```

| Stage | What people do |
|---|---|
| Generate | SWE-agent / OpenHands on real or synthetic tasks |
| Label | `resolved` via test suite |
| Filter | Drop git-hacking, empty patches, test edits, timeouts |
| Format | Chat + tools; **assistant-only loss** common |
| Train | SFT (LoRA/full); sometimes RL after |
| Eval | SWE-bench resolve rate; CE loss on held-out traces at small scale |

## Trajectory-level quality signals

| Signal | Used by |
|---|---|
| `resolved` / test pass | Everyone — necessary, not sufficient |
| Git-hacking / test edits | Open-SWE-Traces, SWE-Lego |
| Patch scope | SWE-Prime |
| Turn count / efficiency | Trajectory curation (2607.17205) |
| Error-retry rate | 2607.17205 — dominant sub-metric |
| Action diversity | SWE-Prime |

**SWE-Prime finding:** Top **10%** scored successful trajectories beat **all** resolved trajectories (+12–24% relative).

## Step-level / loss masking

| Strategy | Idea | Papers |
|---|---|---|
| Assistant-only loss | Train on model actions; mask user/system/tool output | SWE-Master, SWE-Zero/Hero, SWE-smith |
| Error-step masking | Keep bad steps in context; mask loss on errors | SWE-Lego, Step Rejection FT (SRFT) |
| Segment-level selection | Only high-value step groups get loss | SWE-Prime |
| Critic-labeled masking | LLM marks wrong steps; mask from loss | SRFT |

## Debates

### Resolved-only vs include failures

| Camp | Claim |
|---|---|
| Resolved-only | Failures teach model to fail (SWE-smith default) |
| Include + masking | Failures help if bad steps masked (Open-SWE-Traces, SRFT) |
| Include naïve | Hurts (SRFT shows degradation) |

**Open-SWE-Traces ablation:** Full corpus beats resolved-only — Verified +3–4 pts, Multilingual +7–14 pts ([paper §4.3](https://arxiv.org/abs/2606.16038)).

### Pre-SFT quality proxies (small models)

When SWE-bench resolve ≈ 0% at 7B:

- Held-out **cross-entropy loss** on trajectories ([2607.17205](https://arxiv.org/abs/2607.17205))
- **First-action ROUGE-L** — perfect rank correlation with CE in that study
- Quality vs random gap widens with dataset scale

## Pipeline examples

**SWE-smith:** [train guide](https://swesmith.com/guides/train_swe_agent/) — eval → resolved only → XML SFT → assistant-only

**Open-SWE-Traces:** Multi-stage filter → SFT on full corpus → dual-mode thinking/non-thinking → SWE-bench V/M/Pro

**SWE-Master:** Filter ~60k → SFT with environment response masking → RL ([2602.03411](https://arxiv.org/abs/2602.03411))

**R2E-Gym / SWE-Gym:** Successful trajectories for SFT; separate verifier models for reranking

## Analysis ladder (for this project)

```
1. Outcome labels      resolved, empty patch, gold overlap
2. Process metrics     turns, edits-by-k, tests, errors
3. Quality score       composite rubric
4. Step mask map       which turns get loss = 0
5. Proxy eval          CE / first-action (if SFT)
6. Downstream eval     SWE-bench resolve rate
```

## Recommended reading (ordered)

1. [Open-SWE-Traces (2606.16038)](https://arxiv.org/abs/2606.16038)
2. [SWE-Prime (2608.27449)](https://arxiv.org/abs/2608.27449)
3. [Trajectory curation for LoRA (2607.17205)](https://arxiv.org/abs/2607.17205)
4. [Step Rejection FT (2605.10674)](https://arxiv.org/pdf/2605.10674)
5. [SWE-Lego (2601.01426)](https://arxiv.org/pdf/2601.01426)
6. [SWE-smith train guide](https://swesmith.com/guides/train_swe_agent/)

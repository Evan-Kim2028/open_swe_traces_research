# How the field uses traces for SFT

## Standard pipeline

```
Generate trajectories → Label outcome → Filter/score → Format → SFT → Eval on SWE-bench
```

## Training format conventions

### Assistant-only loss (standard across papers)

Train on model reasoning + tool calls; mask user, system, and tool output tokens (labels = -100).

Sources:
- [SWE-Master](https://arxiv.org/abs/2602.03411) — "environment response masking"
- [SWE-Zero / SWE-Hero](https://arxiv.org/html/2604.01496) — excludes tool outputs from loss
- [SWE-smith train guide](https://swesmith.com/guides/train_swe_agent/) — convert to SFT XML/JSONL
- [TuFT chat SFT docs](https://agentscope-ai.github.io/TuFT/en/latest/user-guide/chat-sft.html)

### Step / segment masking (newer)

Keep full conversation in context; mask bad steps from loss.

| Method | Paper |
|---|---|
| Error-step masking (regex on tool errors) | [SWE-Lego](https://arxiv.org/pdf/2601.01426) |
| Critic-labeled step masking | [Step Rejection FT (SRFT)](https://arxiv.org/pdf/2605.10674) |
| Segment-level quality selection | [SWE-Prime](https://arxiv.org/abs/2608.27449) |

## What each major project does

### SWE-smith ([guide](https://swesmith.com/guides/train_swe_agent/))

1. Run SWE-agent → evaluate with test suite
2. Keep resolved only
3. Convert via `collect_trajs` → `ft_xml_*.jsonl`
4. SFT → eval SWE-bench Verified (40.2% with 32B)

Dataset: https://huggingface.co/datasets/SWE-bench/SWE-smith-trajectories

### Open-SWE-Traces ([2606.16038](https://arxiv.org/abs/2606.16038))

1. Multi-stage filter (incomplete, empty patch, git-hacking)
2. SFT on full corpus (resolved + unresolved)
3. Dual-mode thinking / non-thinking
4. Eval: SWE-bench V / Multilingual / Pro

### SWE-Master ([2602.03411](https://arxiv.org/abs/2602.03411))

1. Filter ~60k trajectories
2. SFT with environment response masking
3. RL with execution feedback
4. YaRN context extension to 80k tokens

Repo: https://github.com/RUCAIBox/SWE-Master

### SWE-Lego ([2601.01426](https://arxiv.org/pdf/2601.01426))

1. Real PRs + synthetic instances
2. Git-hacking prevention
3. Error-step masking during SFT
4. Recycle "semi-resolved" (correct file, wrong patch) → +1.2%

### R2E-Gym ([paper](https://r2e-gym.github.io/assets/paper.pdf))

1. Procedural envs from commits (not PRs)
2. Keep successful trajectories for SFT (~3,321 from 2,048 tasks)
3. Also train verifier for inference-time reranking

Dataset: https://huggingface.co/datasets/R2E-Gym/R2EGym-SFT-Trajectories

### SWE-Gym ([2412.21139](https://arxiv.org/pdf/2412.21139v2.pdf))

1. Executable envs with test labels
2. Small SFT set (~491 trajectories) + verifier training
3. Inference-time scaling with outcome reward model

## The resolved-only debate

| Camp | Claim | Source |
|---|---|---|
| Resolved-only | Failures teach model to fail | SWE-smith, FIM-Midtraining repro |
| Full + masking | Failures help if bad steps masked | SRFT +3.7% vs discard failures |
| Full naïve | Hurts | SRFT: unresolved-only without masking degrades |
| Full corpus (no step mask) | Beats resolved-only | Open-SWE-Traces ablation +3–8 pts |

Papers agree you need masking or step selection. SRFT shows naïve inclusion of failures hurts; masking bad steps helps. Open-SWE-Traces gets a lift from the full corpus without step masking, but that setup differs from SRFT's failure-only ablation.

## Read order

1. [Open-SWE-Traces](https://arxiv.org/abs/2606.16038) — our dataset
2. [SWE-Prime](https://arxiv.org/abs/2608.27449) — trajectory + segment scoring
3. [Trajectory curation LoRA (2607.17205)](https://arxiv.org/abs/2607.17205) — metrics + CE proxy
4. [Step Rejection FT](https://arxiv.org/pdf/2605.10674) — using failures safely
5. [SWE-Lego](https://arxiv.org/pdf/2601.01426) — error masking
6. [SWE-smith guide](https://swesmith.com/guides/train_swe_agent/) — simplest pipeline

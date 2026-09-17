# How the field uses traces for SFT

Literature survey (2025–2026). See [sources.md](sources.md) for links.

## Standard pipeline

```
Generate trajectories → Label (tests) → Filter/score → Format → SFT → Eval on SWE-bench
```

---

## Trajectory-level quality (keep or drop whole run)

| Signal | Who uses it |
|---|---|
| **`resolved` / test pass** | Everyone — necessary, not sufficient |
| Git-hacking / test-file edits | Open-SWE-Traces, SWE-Lego |
| Patch scope / empty patch | SWE-Prime, Open-SWE-Traces |
| Turn count / efficiency | [2607.17205](https://arxiv.org/abs/2607.17205) |
| **Error-retry rate** | 2607.17205 — dominant sub-metric |
| Action diversity / redundancy | SWE-Prime |
| Representativeness | SWE-Prime |

**Key result ([SWE-Prime](https://arxiv.org/abs/2608.27449)):** Top **10%** quality-scored successful trajectories beat training on **all** resolved trajectories (+12–24% relative on SWE-bench).

---

## Step-level training (what gets loss)

| Strategy | Description | Sources |
|---|---|---|
| **Assistant-only loss** | Mask user, system, **tool output**; train on assistant actions/reasoning | SWE-Master, SWE-Zero/Hero, SWE-smith, TuFT |
| **Error-step masking** | Keep bad steps in context; mask loss on erroneous tool calls | [SWE-Lego](https://arxiv.org/pdf/2601.01426), [SRFT](https://arxiv.org/pdf/2605.10674) |
| **Segment-level selection** | Only high-value step groups contribute loss | SWE-Prime |
| **Critic-labeled masking** | LLM marks wrong steps → mask from loss | SRFT |

TuFT reference on assistant-only masking: [Chat SFT guide](https://agentscope-ai.github.io/TuFT/en/latest/user-guide/chat-sft.html)

---

## Debates

### Resolved-only vs include failures

| View | Claim | Evidence |
|---|---|---|
| Resolved-only | Failures teach model to fail | [SWE-smith train guide](https://swesmith.com/guides/train_swe_agent/); FIM-Midtraining repro |
| Include + masking | Failures help if bad steps masked | Open-SWE-Traces: full > resolved-only ([§4.3](https://arxiv.org/abs/2606.16038)); SRFT > rejection sampling |
| Include naïve | Hurts | SRFT: unresolved without masking **degrades** vs baseline |

### Pre-SFT quality eval (when model can't resolve SWE-bench)

From [2607.17205](https://arxiv.org/abs/2607.17205):

- **Held-out CE loss** on trajectories (primary at 7B scale)
- **First-action ROUGE-L** — perfect rank correlation with CE in their study
- Quality vs random gap **widens with dataset size** (3.6% CE diff at 2k trajectories)

At scale: **SWE-bench Verified resolve rate** is the final metric.

---

## Notable pipelines

### SWE-smith ([guide](https://swesmith.com/guides/train_swe_agent/))

1. Run SWE-agent → evaluate → `report.json`
2. Keep **resolved only** → convert to XML SFT format (`collect_trajs`)
3. Train assistant-only

Dataset: [SWE-bench/SWE-smith-trajectories](https://huggingface.co/datasets/SWE-bench/SWE-smith-trajectories) (76k rows, ~4 GB)

### Open-SWE-Traces ([2606.16038](https://arxiv.org/abs/2606.16038))

1. Multi-stage filter (incomplete, empty patch, git-hacking)
2. SFT on **full corpus** (resolved + unresolved)
3. Dual-mode: thinking + non-thinking traces
4. Eval: SWE-bench V / Multilingual / Pro

### SWE-Master ([2602.03411](https://arxiv.org/abs/2602.03411))

1. Filter ~60k trajectories
2. SFT with **environment response masking** (no loss on tool stdout)
3. RL with execution feedback

Repo: [RUCAIBox/SWE-Master](https://github.com/RUCAIBox/SWE-Master/)

### SWE-Lego ([2601.01426](https://arxiv.org/pdf/2601.01426))

- Regex **error-step masking** (exclude tokens from failed tool calls)
- Recycle **semi-resolved** trajectories (correct file, wrong patch) → +1.2%

### R2E-Gym / SWE-Gym

- SFT on **successful** trajectories in executable envs
- Also train **verifier models** for inference-time reranking  
  Sources: [R2E-Gym](https://r2e-gym.github.io/), [SWE-Gym paper](https://arxiv.org/pdf/2412.21139v2.pdf)

---

## Implication for this repo

Build the **pre-SFT scoring layer** before spending GPU:

| Our artifact | Field equivalent |
|---|---|
| `trial_summary` + `resolved` | Outcome label |
| `turn_sample` cumulative features | Process quality (SWE-Prime) |
| Early prediction at turn *k* | Trajectory curation scoring |
| Quality rubric (planned) | SWE-Prime trajectory screening |
| `is_tool_error_turn`, `cum_edits` | Step mask map for SRFT/SWE-Lego |

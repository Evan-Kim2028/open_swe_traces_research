# How the field uses, quantifies, and evaluates traces for SFT

Sources synthesized from papers and docs (web search, Sep 2026).

## Standard pipeline

```
Generate trajectories → Label outcome → Filter/score → Format for SFT → Train → Eval on SWE-bench
```

| Stage | What people do |
|---|---|
| Generate | SWE-agent / OpenHands on real or synthetic tasks |
| Label | `resolved` via test suite |
| Filter | Git-hacking, empty patches, test edits, max-turn timeouts |
| Format | Chat + tools; **assistant-only loss** (mask tool output) |
| Train | SFT (LoRA/full); sometimes RL after |
| Eval | **SWE-bench resolve rate**; CE loss on held-out traces for small models |

---

## Trajectory-level quality signals

| Signal | Who uses it |
|---|---|
| `resolved` / test pass | Everyone — necessary, not sufficient |
| Git-hacking / test edits | Open-SWE-Traces, SWE-Lego |
| Patch scope | SWE-Prime |
| Turn count / efficiency | [Trajectory curation (2607.17205)](https://arxiv.org/abs/2607.17205) |
| Error-retry rate | Same — **dominant sub-metric** |
| Action diversity | SWE-Prime |
| Representativeness | SWE-Prime |

**SWE-Prime finding:** Top **10%** scored successful trajectories beat **all** resolved trajectories (+12–24% relative).

- Source: [SWE-Prime (2608.27449)](https://arxiv.org/abs/2608.27449)

---

## Step-level / loss masking strategies

| Strategy | Idea | Sources |
|---|---|---|
| **Assistant-only loss** | Train on model actions; mask user/system/tool output | [SWE-Master](https://arxiv.org/abs/2602.03411), [SWE-Zero/Hero](https://arxiv.org/abs/2604.01496), [SWE-smith docs](https://swesmith.com/guides/train_swe_agent/) |
| **Error-step masking** | Keep bad steps in context; mask loss on erroneous tool calls | [SWE-Lego](https://arxiv.org/pdf/2601.01426), [Step Rejection FT](https://arxiv.org/pdf/2605.10674) |
| **Segment-level selection** | Only high-value step groups contribute loss | [SWE-Prime](https://arxiv.org/abs/2608.27449) |
| **Critic-labeled masking** | LLM marks wrong steps; mask from loss | [SRFT (2605.10674)](https://arxiv.org/pdf/2605.10674) — +3.7% vs discard-all-failures |

**Pattern:** Keep full conversation for context; don't train model to imitate garbage.

---

## Debates

### Resolved-only vs include failures

| Camp | Claim | Evidence |
|---|---|---|
| Resolved-only | Failures teach model to fail | [SWE-smith train guide](https://swesmith.com/guides/train_swe_agent/); FIM-Midtraining repro |
| Include + masking | Failures help if bad steps masked | [Open-SWE-Traces §4.3](https://arxiv.org/abs/2606.16038); [SRFT](https://arxiv.org/pdf/2605.10674) |
| Include naïve | Hurts | SRFT: unresolved-only **without** masking degrades vs baseline |

### Pre-SFT quality proxies (when model can't resolve SWE-bench)

From [Trajectory curation for LoRA (2607.17205)](https://arxiv.org/abs/2607.17205):

- **Held-out CE loss** on trajectories (primary at 7B scale)
- **First-action ROUGE-L** — perfect rank correlation with CE in that study
- Quality vs random gap **widens with scale** (3.6% CE diff at 2k trajectories)

---

## Per-project pipelines

### SWE-smith
1. Run SWE-agent → eval → `report.json`
2. Keep **resolved only** → convert to XML SFT
3. Assistant-only loss

Source: [swesmith.com/guides/train_swe_agent](https://swesmith.com/guides/train_swe_agent/)

### Open-SWE-Traces
1. Multi-stage filter (incomplete, empty patch, git-hacking)
2. SFT on **full corpus** (resolved + unresolved)
3. Dual-mode thinking / non-thinking
4. Eval: SWE-bench V / Multilingual / Pro

Source: [2606.16038](https://arxiv.org/abs/2606.16038)

### SWE-Master
1. Filter ~60k trajectories
2. SFT with **environment response masking**
3. RL with execution feedback

Source: [2602.03411](https://arxiv.org/abs/2602.03411)

### SWE-Lego
1. Rigorous validation; drop test-editing "resolved" trajectories
2. **Error-step masking** (regex on tool errors)
3. Recycle **semi-resolved** (correct file, wrong patch) — +1.2%

Source: [2601.01426](https://arxiv.org/pdf/2601.01426)

### R2E-Gym / SWE-Gym
1. Executable envs; keep **successful** trajectories for SFT
2. Train **verifier models** for inference-time reranking

Sources: [R2E-Gym paper](https://r2e-gym.github.io/assets/paper.pdf), [SWE-Gym (2412.21139)](https://arxiv.org/pdf/2412.21139v2.pdf)

---

## Reading list (priority order)

1. [Open-SWE-Traces (2606.16038)](https://arxiv.org/abs/2606.16038) — our dataset
2. [SWE-Prime (2608.27449)](https://arxiv.org/abs/2608.27449) — trajectory + segment quality
3. [Trajectory curation LoRA (2607.17205)](https://arxiv.org/abs/2607.17205) — quantified metrics + CE proxy
4. [Step Rejection FT (2605.10674)](https://arxiv.org/pdf/2605.10674) — using failures safely
5. [SWE-Lego (2601.01426)](https://arxiv.org/pdf/2601.01426) — error masking + semi-resolved
6. [SWE-smith train guide](https://swesmith.com/guides/train_swe_agent/) — simplest production pipeline
7. [DuckDB skills / state.sql pattern](https://duckdb.org/2026/09/16/duckdb-skills) — query organization inspiration

---

## Field direction (2025–2026)

Moving from *"filter to resolved, train on all tokens"* toward:

> **Score trajectories and steps → mask bad tokens → keep failed runs only with step masking.**

Our DuckDB analytics is the pre-SFT scoring layer that SWE-Prime and 2607.17205 formalize.

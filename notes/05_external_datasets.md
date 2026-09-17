# External trace datasets (union candidates)

## Tier A — drop-in or light ETL

| Dataset | Rows | Size | Notes | Link |
|---|---:|---:|---|---|
| pearsonkyle/swe-agentic-trajectories | ~5k+ | small | Closest format: messages, tools, resolved, submission | [HF](https://huggingface.co/datasets/pearsonkyle/swe-agentic-trajectories) |
| SWE-bench/SWE-smith-trajectories | 76,002 | ~4 GB | messages sometimes JSON string; XML tool format | [HF](https://huggingface.co/datasets/SWE-bench/SWE-smith-trajectories) |
| SWE-Lego-Real-Data-Verified | 4,323 | small | Gold-validated Python OpenHands traces | [HF](https://huggingface.co/datasets/PrimeIntellect/SWE-Lego-Real-Data-Verified) |
| SWE-Lego/SWE-Lego-Real-Data | ~18k | medium | resolved + unresolved splits | [HF](https://huggingface.co/datasets/SWE-Lego/SWE-Lego-Real-Data) |

## Tier B — heavier normalization

| Dataset | Rows | Gap |
|---|---:|---|
| nebius/SWE-agent-trajectories | 80,036 | `trajectory` JSON string; roles ai/user not assistant/tool |
| SWE-Gym | varies | Multiple configs |
| R2E-Gym | varies | Procedural envs; different layout — [R2EGym-SFT-Trajectories](https://huggingface.co/datasets/R2E-Gym/R2EGym-SFT-Trajectories) |

## Tier C — related, different task

| Dataset | Why not first |
|---|---|
| SWE-Lego/SWE-Review-Traj | Code review (approve/reject), not issue-fix |

## Unified target schema (for DuckDB union)

```
source_id, trajectory_id, instance_id, repo, language,
harness, teacher_model, resolved, messages, tools,
model_patch, gold_patch, category
```

## Recommended add order

1. pearsonkyle/swe-agentic-trajectories — validate union pipeline
2. SWE-smith-trajectories — same SWE-agent lineage
3. SWE-Lego-Real-Data-Verified — high-quality resolved Python

## Sources

- Also tracked in `analytics/research/external_datasets.md`
- [evaltrials README](https://github.com/Evan-Kim2028/open_swe_traces_research) — family context (SWE-smith, SWE-Gym, R2E-Gym)

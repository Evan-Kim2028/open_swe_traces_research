# External trace datasets (union candidates)

Goal: normalize into `traces`-compatible columns for future DuckDB union views.

Target schema:

```
source_id, trajectory_id, instance_id, repo, language,
harness, teacher_model, resolved, messages, tools,
model_patch, gold_patch, category
```

---

## Tier A — drop-in or light ETL

| Dataset | Rows | Size | Overlap | Notes |
|---|---:|---:|---|---|
| [pearsonkyle/swe-agentic-trajectories](https://huggingface.co/datasets/pearsonkyle/swe-agentic-trajectories) | ~5k+ | small | **Highest** | `messages`, `tools`, `resolved` bool, `submission`; pre-computed `n_tool_calls`, token costs |
| [SWE-bench/SWE-smith-trajectories](https://huggingface.co/datasets/SWE-bench/SWE-smith-trajectories) | 76,002 | ~4 GB | **High** | `messages`, `resolved`, `patch`; XML/tool/ticks splits; needs format conversion |
| [PrimeIntellect/SWE-Lego-Real-Data-Verified](https://huggingface.co/datasets/PrimeIntellect/SWE-Lego-Real-Data-Verified) | 4,323 | small | **High** | Gold-validated OpenHands messages; Python |
| [SWE-Lego/SWE-Lego-Real-Data](https://huggingface.co/datasets/SWE-Lego/SWE-Lego-Real-Data) | ~18k | medium | **High** | resolved + unresolved splits |

## Tier B — heavier normalization

| Dataset | Rows | Gap |
|---|---:|---|
| [nebius/SWE-agent-trajectories](https://huggingface.co/datasets/nebius/SWE-agent-trajectories) | 80,036 | `trajectory` is JSON string; roles `ai`/`user` not `assistant`/`tool` |
| [SWE-Gym](https://arxiv.org/pdf/2412.21139v2.pdf) | 2,438 tasks | Executable envs + verifier training; trajectory format varies by release |
| [R2E-Gym/R2EGym-SFT-Trajectories](https://huggingface.co/datasets/R2E-Gym/R2EGym-SFT-Trajectories) | 3,231 | Successful trajectories only; different scaffold |

## Tier C — related but different task

| Dataset | Why defer |
|---|---|
| [SWE-Lego/SWE-Review-Traj](https://huggingface.co/datasets/SWE-Lego/SWE-Review-Traj) | Code **review** (approve/reject), not issue-fix |
| [nvidia/Open-SWE-Traces](https://huggingface.co/datasets/nvidia/Open-SWE-Traces) | Primary source (this repo) |

## Recommended add order

1. **pearsonkyle/swe-agentic-trajectories** — validate union pipeline (small, clean)
2. **SWE-smith-trajectories** — SWE-agent lineage; [train guide](https://swesmith.com/guides/train_swe_agent/)
3. **SWE-Lego-Real-Data-Verified** — high-quality resolved Python

TODO: `scripts/download_external.py` + `analytics/schema/007_union_traces.sql`

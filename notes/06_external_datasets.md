# External trace datasets (union candidates)

Goal: normalize into `traces`-compatible schema for DuckDB union views.

Also in: `analytics/research/external_datasets.md`

## Unified target schema

```
source_id          -- e.g. 'nvidia/open-swe-traces'
trajectory_id
instance_id
repo, language
harness, teacher_model
resolved           -- int: 1/0/-1
messages           -- list[struct]
tools              -- list[string]
model_patch, gold_patch
category
```

## Tier A — drop-in or light ETL

| Dataset | Rows | Size | Overlap | Notes |
|---|---:|---:|---|---|
| [pearsonkyle/swe-agentic-trajectories](https://huggingface.co/datasets/pearsonkyle/swe-agentic-trajectories) | ~5k+ | small | **Highest** | `messages`, `tools`, `resolved` bool, `submission` patch; pre-computed token costs |
| [SWE-bench/SWE-smith-trajectories](https://huggingface.co/datasets/SWE-bench/SWE-smith-trajectories) | 76,002 | ~4 GB | **High** | `messages`, `resolved`, `patch`; XML tool format needs conversion |
| [PrimeIntellect/SWE-Lego-Real-Data-Verified](https://huggingface.co/datasets/PrimeIntellect/SWE-Lego-Real-Data-Verified) | 4,323 | small | **High** | Gold-validated OpenHands messages; Python only |
| [SWE-Lego/SWE-Lego-Real-Data](https://huggingface.co/datasets/SWE-Lego/SWE-Lego-Real-Data) | ~18k | medium | **High** | resolved + unresolved splits |

## Tier B — heavier normalization

| Dataset | Rows | Gap |
|---|---:|---|
| [nebius/SWE-agent-trajectories](https://huggingface.co/datasets/nebius/SWE-agent-trajectories) | 80,036 | `trajectory` is JSON string; roles `ai`/`user` not `assistant`/`tool` |
| SWE-Gym | varies | Multiple configs |
| R2E-Gym | varies | Different env structure; see [R2E-Gym-SFT-Trajectories](https://huggingface.co/datasets/R2E-Gym/R2EGym-SFT-Trajectories) (3,231 rows, 56 MB) |

## Tier C — related but different task

| Dataset | Why not first |
|---|---|
| [SWE-Lego/SWE-Review-Traj](https://huggingface.co/datasets/SWE-Lego/SWE-Review-Traj) | Code **review** (approve/reject), not issue-fix |
| nvidia/Open-SWE-Traces | Already primary |

## Recommended add order

1. pearsonkyle/swe-agentic-trajectories — validate union pipeline (small, clean)
2. SWE-smith-trajectories — same SWE-agent lineage as baselines
3. SWE-Lego-Real-Data-Verified — high-quality resolved Python

## Local reference

`evaltrials/README.md` — measured sizes and licenses for SWE-smith (76k, MIT, 3.9 GB) and Open-SWE-Traces (511k, CC-BY-4.0, 39.7 GB).

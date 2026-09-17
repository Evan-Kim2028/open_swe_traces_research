# External trace datasets (union candidates)

Goal: normalize into `traces`-compatible columns for DuckDB union views.

## Tier A — drop-in or light ETL (best candidates)

| Dataset | Rows | Size | Format overlap | Notes |
|---|---:|---:|---|---|
| [pearsonkyle/swe-agentic-trajectories](https://huggingface.co/datasets/pearsonkyle/swe-agentic-trajectories) | ~5k+ | small | **Highest** — `messages`, `tools`, `resolved` bool, `submission` patch | Chat roles match; pre-computed `n_tool_calls`, token costs |
| [SWE-bench/SWE-smith-trajectories](https://huggingface.co/datasets/SWE-bench/SWE-smith-trajectories) | 76,002 | ~4 GB | **High** — `messages`, `resolved`, `patch`, `traj_id` | `messages` sometimes JSON string; SWE-agent XML tool calls → convert to OpenAI tool format |
| [PrimeIntellect/SWE-Lego-Real-Data-Verified](https://huggingface.co/datasets/PrimeIntellect/SWE-Lego-Real-Data-Verified) | 4,323 resolved | ~small | **High** — `messages` from OpenHands | Gold-validated subset; Python only |
| [SWE-Lego/SWE-Lego-Real-Data](https://huggingface.co/datasets/SWE-Lego/SWE-Lego-Real-Data) | ~18k | medium | **High** — `messages` OpenHands | `resolved` + `unresolved` splits |

## Tier B — same domain, heavier normalization

| Dataset | Rows | Size | Gap vs Open-SWE-Traces |
|---|---:|---:|---|
| [nebius/SWE-agent-trajectories](https://huggingface.co/datasets/nebius/SWE-agent-trajectories) | 80,036 | large | `trajectory` is JSON **string**; roles `ai`/`user` not `assistant`/`tool`; no native `tools` list |
| [SWE-Gym/*](https://huggingface.co/datasets/SWE-Gym) | varies | varies | Multiple configs; check per split for message schema |
| [R2E-Gym/*](https://huggingface.co/datasets/R2E-Gym) | varies | varies | Procedural envs; trajectory layout differs |

## Tier C — related but different task

| Dataset | Why not first |
|---|---|
| [SWE-Lego/SWE-Review-Traj](https://huggingface.co/datasets/SWE-Lego/SWE-Review-Traj) | Code **review** agent (approve/reject), not issue-fix |
| [nvidia/Open-SWE-Traces](https://huggingface.co/datasets/nvidia/Open-SWE-Traces) | Already primary source |

## Unified target schema (for DuckDB union)

```
source_id          -- e.g. 'nvidia/open-swe-traces'
trajectory_id
instance_id
repo
language
harness
teacher_model
resolved           -- int: 1/0/-1
messages           -- list[struct]
tools              -- list[string] JSON tool defs
model_patch
gold_patch
category
```

## Recommended add order

1. **pearsonkyle/swe-agentic-trajectories** — validate union pipeline (small, clean)
2. **SWE-smith-trajectories** — big lift, same SWE-agent lineage as paper baselines
3. **SWE-Lego-Real-Data-Verified** — high-quality resolved Python trials

Use `scripts/download_external.py` (TODO) + `analytics/schema/007_union_traces.sql` after ETL.

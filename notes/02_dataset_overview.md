# Open-SWE-Traces dataset overview

## What it is

Agentic SFT corpus from NVIDIA — multi-step SWE agent runs on real GitHub issues from [SWE-rebench-V2](https://huggingface.co/datasets/nebius/SWE-rebench-V2).

**Not an eval benchmark.** Training trajectories; do not confuse row count with eval coverage.

Sources: [HF dataset card](https://huggingface.co/datasets/nvidia/Open-SWE-Traces), [arXiv:2606.16038](https://arxiv.org/abs/2606.16038)

## Scale (paper vs HF card vs local)

| Metric | Value |
|---|---|
| Paper v1.0 trajectories | 207,489 |
| HF card (Sep 2026) | 511,668 rows, ~42.6 GB |
| Languages | 9: Python, Go, TS, JS, Rust, Java, PHP, C, C++ |
| Harnesses | OpenHands, SWE-agent (+ v1.1/v1.2: DeepSeek, Qwen3.6/3.8, mini-swe-agent) |
| Teachers | MiniMax-M2.5 (thinking), Qwen3.5-122B (non-thinking), others in later versions |

## Schema (one row = one trial)

| Field | Meaning |
|---|---|
| `trajectory_id` | Unique run UUID |
| `instance_id` | Task/issue ID (SWE-rebench-V2) |
| `repo`, `language`, `license` | Task context |
| `messages` | Full chat: system, user, assistant, tool |
| `tools` | JSON tool definitions |
| `resolved` | `1` = tests pass, `0` = fail, `-1` = unknown |
| `metadata.reference_patch` | Gold diff + line/file counts |
| `metadata.model_patch` | Agent's diff + line/file counts |
| `metadata.category` | bug-fix, enhancement, feature-request, etc. |

## Quality pipeline (paper)

1. Drop corrupted / incomplete runs
2. Reject empty patches, test-suite edits, malformed tool calls
3. **TrajectoryScanner** — AST audit to strip "git hacking" (agents peeking git history)
4. Standardize to unified message schema; preserve `reasoning_content` from MiniMax

Source: [Open-SWE-Traces paper §2.3–2.4](https://arxiv.org/html/2606.16038v1)

## Paper SFT results (downstream eval)

Fine-tuned Qwen3-Coder-30B-A3B → **Open-SWE-Agent**:
- SWE-bench Verified: **61.7%** (`/no_think`)
- SWE-bench Multilingual: **57.1%**
- SWE-bench Pro: **36.8%**

Source: [arXiv:2606.16038 Table 3–4](https://arxiv.org/abs/2606.16038)

## Key paper ablations (relevant to our work)

| Ablation | Finding |
|---|---|
| Resolved-only → full corpus | Full wins (+3–8 pts Verified/Multilingual) |
| Python-only → multilingual | Multilingual wins big (+14 pts on Multilingual) |
| Cross-harness transfer | Works but with penalty; MSWE-agent generalizes better |

Source: [paper §4](https://arxiv.org/html/2606.16038v1)

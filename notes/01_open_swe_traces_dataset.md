# Open-SWE-Traces dataset

**Source:** [nvidia/Open-SWE-Traces](https://huggingface.co/datasets/nvidia/Open-SWE-Traces) · [arXiv:2606.16038](https://arxiv.org/abs/2606.16038)

## What it is

~511k agent trajectories for SFT/distillation. Not an eval benchmark — training data.

- **Teachers:** MiniMax-M2.5 (thinking) + Qwen3.5-122B (non-thinking); later v1.1/v1.2 add DeepSeek-V4, Qwen3.6/3.8
- **Harnesses:** OpenHands, SWE-agent, mini-swe-agent
- **Tasks:** ~20k real PRs from [SWE-rebench-V2](https://huggingface.co/datasets/nebius/SWE-rebench-V2)
- **Languages:** Python, Go, TS, JS, Rust, Java, PHP, C, C++ (9 total)
- **License:** CC BY 4.0 (dataset); underlying repos MIT/Apache/BSD

## Schema (one row = one trial)

| Field | Meaning |
|---|---|
| `trajectory_id` | UUID per run |
| `instance_id` | Task/issue id (many trials per task) |
| `repo`, `language`, `license` | Task context |
| `messages` | Full chat: system, user, assistant, tool |
| `tools` | JSON tool definitions |
| `resolved` | `1` = tests pass, `0` = fail, `-1` = unknown |
| `metadata.reference_patch` | Gold diff + stats |
| `metadata.model_patch` | Agent's diff + stats |
| `metadata.category` | bug-fix, enhancement, feature-request, etc. |

## Paper filtering pipeline

1. Drop corrupted / incomplete runs
2. Reject empty patches, test-suite edits, malformed tool calls
3. **TrajectoryScanner** — AST audit to strip "git hacking" (agents peeking at git history)

Source: [Open-SWE-Traces paper §2.3–2.4](https://arxiv.org/abs/2606.16038)

## Paper SFT findings (ablations)

| Experiment | SWE-bench Verified (think) | SWE-bench Multilingual (no-think) |
|---|---|---|
| Resolved-only → Full corpus | 55.3% → 58.1% | 49.6% → 57.1% |
| Python-only → Multilingual | 54.9% → 58.1% | 43.1% → 57.1% |

**Claim:** Including unresolved trajectories helps; failures provide useful interaction signal if not naïvely imitated.

Source: [Open-SWE-Traces paper §4.3](https://arxiv.org/abs/2606.16038)

## Open-SWE-Agent results (after SFT on this data)

| Benchmark | Best resolve rate |
|---|---|
| SWE-bench Verified | 61.7% (`/no_think`) |
| SWE-bench Multilingual | 57.1% |
| SWE-bench Pro | 36.8% |

Base: Qwen3-Coder-30B-A3B. Source: [paper abstract](https://arxiv.org/abs/2606.16038)

## Local download status

- **Target:** ~43 GB, 215 parquet files, 511,668 rows
- **Download:** idempotent via `scripts/download_data.py`; throttle with `--max-mbps 20`
- **Status file:** `traces_data/.download_status.json`
- User reported **download complete** (Sep 2026 session)

## EDA snapshot (partial download era, ~348k rows)

- ~42k unique tasks
- ~25% resolved (`1`); large `unknown` bucket especially for openhands
- Successes: fewer avg turns than failures
- mini-swe-agent Python ~41% resolved; openhands many `resolved=-1`

Run fresh: `uv run python scripts/duckdb_query.py -f analytics/queries/001_dataset_profile.sql`

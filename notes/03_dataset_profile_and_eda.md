# Dataset profile and EDA

Primary source: [nvidia/Open-SWE-Traces](https://huggingface.co/datasets/nvidia/Open-SWE-Traces)  
Paper: [arXiv:2606.16038](https://arxiv.org/abs/2606.16038)

## Local download (complete)

| Metric | Value | As of |
|---|---:|---|
| Parquet files | 212 / ~215 | 2026-09-17 |
| Size | 42.60 GB | 2026-09-17 |
| Status | Complete | `traces_data/.download_status.json` |

Verify: `uv run python scripts/verify_data.py` or `uv run openswe-verify`

## Paper-reported scale (v1.0 subset)

| Metric | Value | Source |
|---|---:|---|
| Total trajectories | 207,489 | [2606.16038](https://arxiv.org/abs/2606.16038) |
| Languages | 9 (Python, Go, TS, JS, Rust, Java, PHP, C, C++) | ibid. |
| Resolved rate (test-verified) | ~40.6% (65,244 / 207,489) | ibid. |
| Thinking traces | 51.7% (MiniMax-M2.5) | ibid. |

Extended HF release (v1.1/v1.2) adds DeepSeek-V4-Flash, Qwen3.6/3.8, mini-swe-agent — see [dataset card changelog](https://huggingface.co/datasets/nvidia/Open-SWE-Traces).

## EDA during partial download (~348k rows)

From `analytics/queries/003_trial_eda.sql` (pre-complete; re-run on full corpus):

| Metric | Value |
|---|---:|
| Trials | ~348k |
| Unique tasks (`instance_id`) | ~42k |
| Overall `resolved=1` | ~25–29% (many `resolved=-1` on openhands) |

**Outcome mix by harness (approximate):**

| Outcome | mini-swe-agent | openhands |
|---|---:|---:|
| success | ~64k (avg ~96 turns) | ~24k (avg ~135 turns) |
| failed | ~69k (avg ~123 turns) | ~41k (avg ~156 turns) |
| unknown (`-1`) | ~69k | ~80k |

**Patterns:**
- Successes tend to be **shorter** than failures/unknowns.
- OpenHands rows have many **unknown** resolve labels.
- Python dominates volume; C/C++ are tiny slices.

## Early-signal sample (15 trials)

From `analytics/queries/004_turn_sample_early_signal.sql` — **directional only**, not conclusive:

| Progress | Success avg edits | Failed avg edits |
|---|---:|---:|
| 0–25% | 3.5 | 4.6 |
| 75–100% | 11.2 | 22.8 |

Successes edit earlier and less; failures accumulate more edits and tool errors.

## Quality filtering (from paper — what was removed before release)

Source: [2606.16038 §2.3–2.4](https://arxiv.org/abs/2606.16038)

- Incomplete runs (max turns, harness crash)
- Empty patches, test-suite edits
- Malformed tool calls
- **Git hacking** (AST audit of bash — `git log`, `blame`, etc.)
- v1.0 later removed git-hacking trajectories post-hoc

## Re-run on full corpus

```bash
uv run python scripts/duckdb_init.py --refresh-summaries
uv run python scripts/duckdb_query.py -f analytics/queries/001_dataset_profile.sql
uv run python scripts/duckdb_query.py -f analytics/queries/003_trial_eda.sql
```

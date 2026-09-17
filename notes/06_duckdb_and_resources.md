# DuckDB workflow and resource constraints

## DuckDB setup

- **Session config:** `scripts/duckdb_session.py` — `memory_limit=4GB`, `threads=2`, temp in `duckdb/tmp/`
- **Init:** `uv run python scripts/duckdb_init.py`
- **Interactive:** `duckdb -init analytics/state.sql duckdb/open_swe.duckdb`
- **Query + log:** `uv run python scripts/duckdb_query.py -f analytics/queries/...`

Inspired by [DuckDB skills / state.sql](https://duckdb.org/2026/09/16/duckdb-skills).

## Views vs materialized

| Kind | Examples | When to refresh |
|---|---|---|
| **Views** | `traces`, `trial_summary`, `catalog_*` | Auto — re-read parquet on query |
| **Summary tables** | `trace_summary_*` in schema 004 | After large download chunks (`--refresh-summaries`) |
| **Turn sample** | `outputs/turn_sample.parquet` → view `turn_sample` | `extract_turn_sample.py --register-duckdb` |

## Turn extraction (resource-safe)

`scripts/extract_turn_sample.py`:

- Default **30–50 trials** (not full corpus)
- Phase 1: locate files (`trajectory_id` + `filename` only)
- Phase 2: per-file DuckDB `INSERT INTO turn_staging` → single `COPY TO parquet`
- **No PyArrow** full-file reads; **no** `trace_turns` view at full scale

## Pitfalls encountered

| Issue | Fix |
|---|---|
| 31GB `duckdb/open_swe.duckdb.tmp` from interrupted query | Delete tmp folder; cap memory |
| OOM on `trace_turns` / full `messages` unnest | Sample only; use `trial_summary` for aggregates |
| Parquet `APPEND` unreliable | Use temp staging table + single COPY |
| Download filled disk | Throttle `--max-mbps 20`; ~43GB total needed |

## Download

```bash
uv run python scripts/download_data.py --max-mbps 20   # idempotent
uv run python scripts/download_data.py --status
```

Status: `traces_data/.download_status.json`

## Compute budget (user)

| Resource | Status |
|---|---|
| AMD AI credits | Applied; ~$100 pending (~50 hrs MI300X @ $1.99/hr) |
| Kaggle GPU | ~30 hrs/week; reserved for SFT later |
| Local | DuckDB EDA on CPU; 4GB DuckDB cap |

## Small-scale SFT experiment plan (when GPU available)

- Model: Qwen3-4B or 8B + LoRA
- Data: 500–2000 filtered trajectories from Open-SWE-Traces
- Ablations: resolved-only vs all; thinking vs non-thinking traces
- Eval: 20–50 SWE-bench Lite instances (not full 500)

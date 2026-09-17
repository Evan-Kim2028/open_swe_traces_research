# Project setup

**Repo:** `/home/evan/Documents/open_swe_traces_research`  
**GitHub:** https://github.com/Evan-Kim2028/open_swe_traces_research

## Stack

- **uv** Python env (`pyproject.toml`, `.venv/`)
- **Data:** `traces_data/` (gitignored, ~43 GB full dataset)
- **DuckDB:** `duckdb/open_swe.duckdb` (gitignored, views only)
- **Analytics:** `analytics/schema/`, `analytics/queries/`, `analytics/query_log/`

## Commands

```bash
uv sync
uv run python scripts/download_data.py              # idempotent resume
uv run python scripts/download_data.py --max-mbps 20  # throttled
uv run python scripts/download_data.py --status
uv run python scripts/duckdb_init.py
uv run python scripts/duckdb_init.py --refresh-summaries
uv run python scripts/duckdb_query.py -f analytics/queries/001_dataset_profile.sql
uv run python scripts/extract_turn_sample.py --sample-size 30 --register-duckdb
```

## DuckDB resource limits

All scripts use `scripts/duckdb_session.py`:
- `memory_limit = 4GB`
- `threads = 2`
- temp spill: `duckdb/tmp/` (clean if stale `.tmp` dirs grow large)

Inspired by [DuckDB skills / state.sql](https://duckdb.org/2026/09/16/duckdb-skills).

## Download status (last checked session)

- Full dataset target: ~511,668 rows, ~43 GB, 215 parquet files
- Download is idempotent via `huggingface_hub snapshot_download`
- Progress tracked in `traces_data/.download_status.json`
- User reported download completed in later session

## Sources

- Dataset: https://huggingface.co/datasets/nvidia/Open-SWE-Traces
- Paper: https://arxiv.org/abs/2606.16038
- Local index: `evaltrials/README.md` (511,668 rows measured)

# Infrastructure notes

## DuckDB

- **Memory cap:** 4GB everywhere via `scripts/duckdb_session.py`
- **Threads:** 2
- **Temp dir:** `duckdb/tmp/` (clean if interrupted — stale temp can grow to tens of GB)
- **Init:** `uv run python scripts/duckdb_init.py`
- **Query (streaming):** `uv run python scripts/duckdb_query.py -f analytics/queries/…`
- **Interactive:** `duckdb -init analytics/state.sql duckdb/open_swe.duckdb`

## Download

- Idempotent: `uv run python scripts/download_data.py`
- Throttled: `--max-mbps 20`
- Status: `--status` → `traces_data/.download_status.json`

## Turn sample extraction

- **Script:** `scripts/extract_turn_sample.py`
- **Method:** DuckDB only — sample IDs from `trial_summary`, locate files, per-file `INSERT INTO turn_staging`, single `COPY TO parquet`
- **Default:** 30 trials; scale cautiously (message unpack is heavy)
- **Output:** `outputs/turn_sample.parquet` → view `turn_sample`

## Resource rules

- Do **not** unnest full 500k `messages` in one query
- Use `trial_summary` for aggregates
- Use `extract_turn_sample.py` for turn-level work
- Remove `duckdb/tmp/` if disk spikes after crashed queries

## Package layout (repo)

```
src/openswe_traces/     Python package (data, features, sft sample)
scripts/                CLI wrappers
analytics/              SQL schema, queries, query_log
notes/                  This folder
experiments/kaggle_smoke/ GPU smoke tests
```

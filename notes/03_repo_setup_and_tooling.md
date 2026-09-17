# Repo setup and tooling

## Repository

- **GitHub:** https://github.com/Evan-Kim2028/open_swe_traces_research
- **Local:** `~/Documents/open_swe_traces_research`
- **Python:** uv (`uv sync`)

## Data

- **Path:** `traces_data/` (gitignored)
- **Download:** `uv run python scripts/download_data.py` (idempotent, resume-safe)
- **Throttle:** `--max-mbps 20`
- **Status:** `uv run python scripts/download_data.py --status`
- **Manifest:** `traces_data/.download_status.json`

## DuckDB analytics

- **Session config:** `scripts/duckdb_session.py` — **4GB memory cap**, 2 threads, temp in `duckdb/tmp/`
- **Init:** `uv run python scripts/duckdb_init.py` (`--refresh-summaries` for materialized tables)
- **Query:** `uv run python scripts/duckdb_query.py -f analytics/queries/001_dataset_profile.sql`
- **CLI:** `duckdb -init analytics/state.sql duckdb/open_swe.duckdb`

### Layout

| Path | Purpose |
|---|---|
| `analytics/schema/` | Views + summary DDL |
| `analytics/queries/` | Named SQL |
| `analytics/query_log/` | Auto-logged runs + `index.csv` |
| `analytics/research/` | Human notes |
| `duckdb/open_swe.duckdb` | Local DB (gitignored) |

### Key views

- `traces` — one row per trial
- `trial_summary` — fast, no message unpack
- `trace_turns` — one row per message (expensive at scale)
- `turn_sample` — from `outputs/turn_sample.parquet`

## Turn extraction

```bash
uv run python scripts/extract_turn_sample.py --sample-size 30 --register-duckdb
```

- DuckDB streaming only: per-file lookup → `INSERT INTO turn_staging` → single `COPY`
- Default 30–50 trials; do not unnest full 500k corpus

## Resource lessons

- Removed 31GB stale `duckdb/open_swe.duckdb.tmp` from interrupted queries
- Avoid `trace_turns` on full table; use sampled extraction
- Kaggle reserved for GPU SFT later; AMD credits pending

## Sources

- [DuckDB skills / state.sql pattern](https://duckdb.org/2026/09/16/duckdb-skills)

# Project overview

Local research workspace for [nvidia/Open-SWE-Traces](https://huggingface.co/datasets/nvidia/Open-SWE-Traces).

- **Paper:** [Open-SWE-Traces (arXiv:2606.16038)](https://arxiv.org/abs/2606.16038)
- **Repo:** https://github.com/Evan-Kim2028/open_swe_traces_research
- **Local path:** `~/Documents/open_swe_traces_research`

## Goal

Use trace structure to **filter, rank, and predict outcomes** — not reproduce NVIDIA's 30B SFT at full scale.

> When during an agent run does success or failure become predictable, and which traces are high-quality supervision — independent of benchmark score alone?

## Stack

- **uv** Python env (`pyproject.toml`)
- **DuckDB** analytics (`duckdb/open_swe.duckdb`, 4GB memory cap via `scripts/duckdb_session.py`)
- **Data:** `traces_data/` (gitignored, full HF mirror)
- **Kaggle:** reserved for GPU SFT later
- **AMD credits:** pending (~$100 when approved)

## Key commands

```bash
uv sync
uv run python scripts/download_data.py --max-mbps 20   # idempotent
uv run python scripts/download_data.py --status
uv run python scripts/duckdb_init.py
uv run python scripts/duckdb_init.py --refresh-summaries
uv run python scripts/duckdb_query.py -f analytics/queries/001_dataset_profile.sql
uv run python scripts/extract_turn_sample.py --sample-size 30 --register-duckdb
```

## Repo layout

| Path | Purpose |
|---|---|
| `traces_data/` | HF dataset mirror |
| `duckdb/` | `open_swe.duckdb` + temp (`duckdb/tmp/`) |
| `analytics/schema/` | DDL views |
| `analytics/queries/` | Named SQL |
| `analytics/query_log/` | Auto-logged query runs |
| `analytics/research/` | Earlier research docs |
| `notes/` | This folder |
| `outputs/` | Generated parquet (e.g. `turn_sample.parquet`) |
| `scripts/` | download, duckdb, extract |

# open_swe_traces_research

Local research on [nvidia/Open-SWE-Traces](https://huggingface.co/datasets/nvidia/Open-SWE-Traces).

Paper: [arXiv:2606.16038](https://arxiv.org/abs/2606.16038)

## Setup

```bash
cd ~/Documents/open_swe_traces_research
uv sync
```

## Download dataset (~43 GB, idempotent)

```bash
uv run python scripts/download_data.py          # resume-safe; rerun anytime
uv run python scripts/download_data.py --status # progress only
uv run python scripts/verify_data.py            # row counts after complete
```

Progress is tracked in `traces_data/.download_status.json`.

## DuckDB analytics (in-repo)

Inspired by [DuckDB skills / state.sql](https://duckdb.org/2026/09/16/duckdb-skills):

```bash
uv run python scripts/duckdb_init.py                        # views over parquet
uv run python scripts/duckdb_init.py --refresh-summaries    # + materialized tables
uv run python scripts/duckdb_query.py -f analytics/queries/001_dataset_profile.sql
```

Interactive CLI:

```bash
duckdb -init analytics/state.sql duckdb/open_swe.duckdb
```

### Organization

| Path | Purpose |
|---|---|
| `analytics/schema/` | Views + summary tables (version controlled) |
| `analytics/queries/` | Named research SQL (version controlled) |
| `analytics/query_log/` | Auto-log of every query run + `index.csv` |
| `analytics/research/LOG.md` | Human findings journal |
| `analytics/research/questions.md` | Open hypotheses |
| `duckdb/open_swe.duckdb` | Local DB file (gitignored, rebuilt from schema) |

**Views** (`traces`, `catalog_*`) re-read parquet on each query — new shards appear as download grows.

**Summary tables** (`trace_summary_*`) are materialized snapshots — refresh after large download chunks.

## Layout

```
open_swe_traces_research/
├── traces_data/           # HF dataset (gitignored)
├── duckdb/                # open_swe.duckdb (gitignored)
├── analytics/
│   ├── state.sql          # CLI bootstrap
│   ├── schema/            # DDL
│   ├── queries/           # saved SQL
│   ├── query_log/         # run history
│   └── research/          # notes
├── scripts/
└── notebooks/
```

Kaggle reserved for GPU fine-tuning later; all EDA runs locally.

## License

Code: MIT. Dataset: [CC BY 4.0](https://huggingface.co/datasets/nvidia/Open-SWE-Traces).

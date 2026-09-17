# open_swe_traces_research

Local research on [nvidia/Open-SWE-Traces](https://huggingface.co/datasets/nvidia/Open-SWE-Traces).

Paper: [arXiv:2606.16038](https://arxiv.org/abs/2606.16038)

**Research notes:** [`notes/`](notes/README.md) — thesis, EDA, SFT literature survey, sources.

## Setup

```bash
cd ~/Documents/open_swe_traces_research
uv sync
```

## Package

All logic lives in the importable package `src/openswe_traces/`; everything under `scripts/`
is a thin CLI wrapper over it.

| Module | Purpose |
|---|---|
| `openswe_traces.data` | Repo paths, parquet shard parsing (`harness/teacher/source`), DuckDB session (4 GB cap), DB init, query runner |
| `openswe_traces.download` | Idempotent HF download + status file |
| `openswe_traces.verify` | Corpus row counts |
| `openswe_traces.features` | Per-trajectory proxy features + turn-level sample extraction |
| `openswe_traces.summary` | Markdown summary of `outputs/proxy_features.parquet` |
| `openswe_traces.sft.sample` | Compact JSONL sample for Kaggle SFT |
| `openswe_traces.kaggle` | Push / status / output helpers around the kaggle CLI |

## Download dataset (~43 GB, idempotent)

```bash
uv run openswe-download            # resume-safe; rerun anytime
uv run openswe-download --status   # progress only
uv run openswe-verify              # row counts after complete
```

The same CLIs also run as scripts: `uv run python scripts/download_data.py [--status]`,
`uv run python scripts/verify_data.py`.

Progress is tracked in `traces_data/.download_status.json`.

## DuckDB analytics (in-repo)

All DuckDB connections use a **4GB memory cap** via `openswe_traces.data`
(streaming queries, temp in `duckdb/tmp/`).

Inspired by [DuckDB skills / state.sql](https://duckdb.org/2026/09/16/duckdb-skills):

```bash
uv run python scripts/duckdb_init.py                        # views over parquet
uv run python scripts/duckdb_init.py --refresh-summaries    # + materialized tables
uv run python scripts/duckdb_query.py -f analytics/queries/001_dataset_profile.sql
uv run python scripts/extract_turn_sample.py --sample-size 30 --register-duckdb
```

Interactive CLI:

```bash
duckdb -init analytics/state.sql duckdb/open_swe.duckdb
```

### Proxy features → ranking summary

```bash
uv run openswe-features                       # per-shard parts, then merged output
uv run openswe-features --limit 2 --head 5    # smoke: two shards + head/sanity table
uv run openswe-features --merge-only          # merge existing parts
uv run python scripts/proxy_features_summary.py   # analytics/research/proxy_features_summary.md
```

Outputs land in `outputs/` (gitignored): `proxy_features.parquet`,
`proxy_features_parts/`, `turn_sample.parquet`.

### Organization

| Path | Purpose |
|---|---|
| `src/openswe_traces/` | Package: all data access, features, SFT prep, kaggle glue |
| `scripts/` | Thin CLIs over the package (flags unchanged from before the refactor) |
| `tests/` | pytest; fast, run on a 1-file sample |
| `analytics/schema/` | Views + summary tables (version controlled) |
| `analytics/queries/` | Named research SQL (version controlled) |
| `analytics/query_log/` | Auto-log of every query run + `index.csv` |
| `analytics/research/` | Human findings journal |
| `analytics/research/questions.md` | Open hypotheses |
| `duckdb/open_swe.duckdb` | Local DB file (gitignored, rebuilt from schema) |
| `experiments/<name>/` | One dir per experiment: README, config, kernel metadata, `out/` |

**Views** (`traces`, `catalog_*`) re-read parquet on each query — new shards appear as download grows.

**Summary tables** (`trace_summary_*`) are materialized snapshots — refresh after large download chunks.

## Kaggle (GPU runs)

SFT sample for a kernel dataset:

```bash
uv run openswe-sample --n 1000     # → experiments/kaggle_smoke/data/sample_1000.jsonl
```

Kernels are pushed only from an experiment dir via its `run_kernel.sh`; see
`experiments/kaggle_smoke/README.md`. Python callers can use `openswe_traces.kaggle`
(`push` / `status` / `wait` / `output`).

## Layout

```
open_swe_traces_research/
├── src/openswe_traces/    # package (data, download, verify, features, summary, sft, kaggle)
├── scripts/               # thin CLIs
├── tests/                 # pytest
├── experiments/           # one dir per experiment (e.g. kaggle_smoke)
├── traces_data/           # HF dataset (gitignored)
├── duckdb/                # open_swe.duckdb (gitignored)
├── analytics/             # schema, queries, query log, research notes
├── outputs/               # derived artifacts (gitignored)
└── notebooks/
```

## License

Code: MIT. Dataset: [CC BY 4.0](https://huggingface.co/datasets/nvidia/Open-SWE-Traces).

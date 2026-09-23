# open_swe_traces_research

Two lines of work share this repository.

1. **The affordance ladder** (current). Build software-engineering tasks with controlled
   difficulty, run coding agents on them with progressively more information, and record
   the level at which each agent starts to succeed. Written up as *Difficulty is an
   information gap*. Start with [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md), then
   [`docs/OPERATIONS.md`](docs/OPERATIONS.md). The latest state of the run is in
   [`docs/HANDOFF.md`](docs/HANDOFF.md).
2. **Open-SWE-Traces analytics** (earlier). Local analysis of
   [nvidia/Open-SWE-Traces](https://huggingface.co/datasets/nvidia/Open-SWE-Traces)
   ([arXiv:2606.16038](https://arxiv.org/abs/2606.16038)). Research notes are in
   [`notes/`](notes/README.md). The rest of this README covers it.

## Setup

```bash
cd ~/Documents/open_swe_traces_research
uv sync
uv run pytest
```

## Repository map

| Path | What is there |
|---|---|
| `src/openswe_traces/ladder/` | Trial outcomes, certificates, admission rules, escalation |
| `src/openswe_traces/ops/` | Agent connectors, capacity, budgets, disk and container hygiene |
| `src/openswe_traces/reports/` | Status, dashboard, cost and ladder reports |
| `src/openswe_traces/analysis/` | Instruments behind the findings in `analytics/research/` |
| `src/openswe_traces/authoring/`, `pipeline/`, `pipeline_ext/` | The task factory |
| `src/openswe_traces/` (top level), `synth/`, `sft/` | Open-SWE-Traces analytics |
| `scripts/ops/` | Commands and daemons that run the experiment; see its README |
| `scripts/dev/` | Refactoring and verification tools |
| `scripts/*.py` | Commands for the task factory and the trace analytics |
| `experiments/dose_response/` | Staged sweeps, and every trial under `jobs/` (gitignored) |
| `experiments/pipeline/`, `experiments/harbor_nex/` | Authored units and task packages |
| `analytics/research/` | Findings journal, one note per question |
| `docs/` | Architecture, operations, specs, handoffs |
| `tests/` | pytest |

## Package

Library code lives in `src/openswe_traces/`. Commands in `scripts/` call into it; the ones
in `scripts/ops/` that moved into the package in September 2026 are shims at their old
paths, so running daemons and frozen sweep copies keep working.

The trace-analytics modules:

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

## License

Code: MIT. Dataset: [CC BY 4.0](https://huggingface.co/datasets/nvidia/Open-SWE-Traces).

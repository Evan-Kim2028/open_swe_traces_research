# open_swe_traces_research

Local research workspace for [nvidia/Open-SWE-Traces](https://huggingface.co/datasets/nvidia/Open-SWE-Traces) — trace analytics, quality scoring, and early success/failure prediction.

Paper: [Open-SWE-Traces (arXiv:2606.16038)](https://arxiv.org/abs/2606.16038)

## Setup

```bash
cd ~/Documents/open_swe_traces_research
uv sync
```

## Download dataset (~43 GB)

```bash
uv run python scripts/download_data.py
```

Data lands in `traces_data/` (gitignored). Resume is supported if interrupted.

Verify after download:

```bash
uv run python scripts/verify_data.py
```

## Query with DuckDB

Parquet shards under `traces_data/data/` can be queried in place:

```python
import duckdb

con = duckdb.connect()
con.sql("""
  SELECT language, resolved, count(*) AS n
  FROM read_parquet('traces_data/data/**/*.parquet', union_by_name=true)
  GROUP BY 1, 2
  ORDER BY n DESC
""").show()
```

## Layout

```
open_swe_traces_research/
├── traces_data/          # HF dataset mirror (local only)
├── scripts/
│   ├── download_data.py
│   └── verify_data.py
├── notebooks/            # analysis notebooks (add as needed)
└── outputs/              # charts, exports (gitignored)
```

Kaggle is reserved for GPU fine-tuning experiments later; all EDA runs locally.

## License

Code: MIT. Dataset: [CC BY 4.0](https://huggingface.co/datasets/nvidia/Open-SWE-Traces) (NVIDIA).

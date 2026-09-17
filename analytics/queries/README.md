# Named queries

Store reusable SQL here. Each file should start with a comment block explaining **why** it exists.

Run any query and append to the query log:

```bash
uv run python scripts/duckdb_query.py -f analytics/queries/001_dataset_profile.sql
```

Past runs are indexed in `analytics/query_log/index.csv` with full SQL copies.

When you learn something worth keeping, add a line to `analytics/research/LOG.md`.

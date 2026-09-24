# open_swe_traces_research — conventions

First-class Python repo. `uv` for everything. Every result must be reproducible from a
committed script plus a pinned lockfile.

## Layout (target)

```
src/openswe_traces/      importable package (data access, features, sft prep, kaggle glue)
scripts/                 thin CLIs; each wraps a package function, no logic of its own
experiments/<name>/      one dir per experiment: README.md, config, kernel-metadata.json, out/
analytics/               DuckDB schema (*.sql), queries, research notes (*.md)
outputs/                 derived artifacts (parquet, json); gitignored except small summaries
traces_data/             raw HF download; never modified, gitignored
tests/                   pytest; fast, run on a 1-file sample
```

Rules

- New logic goes in `src/openswe_traces/`. A script may start standalone, but the
  moment it is run twice its body moves into the package and the script becomes
  `main()` calling it. Name the script after the package function.
- `uv run python scripts/x.py`, `uv run pytest`, `uv run ruff check .`. No bare `python`,
  no `pip install`. Add deps with `uv add`. Dev tools under `[dependency-groups] dev`.
- Every experiment dir has a README stating: question, config, command to reproduce,
  where output lands, and what was measured. Kaggle kernels are pushed only from an
  experiment dir via its `run_kernel.sh`; GPU quota is 30 h/week, do not re-push on failure
  without a diagnosis.
- Streaming only over `traces_data` (file-by-file, DuckDB memory cap 4 GB). Never load
  the corpus into memory.
- Resume-safe long jobs: skip finished units, log to `outputs/<job>.log`, run under nohup.
- Small, dated markdown notes for findings in `analytics/research/`. Numbers in tables.
- Commit only when asked. Never commit `outputs/*.parquet`, `traces_data/`, `out/`.

## GitHub Actions billing — ignore, merge anyway

GitHub Actions failures on this account are a **billing state, not a code signal**.
The job never started, so its red X carries no information about the change. Do not
debug it, do not "fix the workflow", do not block a merge on it. Merge anyway.

Confirm in five seconds: `gh run view <id> --json jobs` shows `"steps": []` and
`"runner_name": ""` — nothing ran, so nothing failed. A job that *did* run and
failed is a real signal; that one still counts.

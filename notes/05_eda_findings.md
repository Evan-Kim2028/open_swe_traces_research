# EDA findings (partial → full download)

Notes from exploratory sessions. Re-run `001_dataset_profile.sql` on full download for authoritative numbers.

## Scale (partial session ~348k rows)

| Metric | Value |
|---|---|
| Trials | ~347,832 |
| Unique tasks | ~42,330 |
| Overall resolved | ~25.4% |
| Harnesses present | mini-swe-agent + openhands (partial sweagent during download) |

## Outcome mix by harness (partial)

| Outcome | mini-swe-agent | openhands |
|---|---:|---:|
| success | ~64k (avg ~96 turns) | ~24k (avg ~135 turns) |
| failed | ~69k (avg ~123 turns) | ~41k (avg ~156 turns) |
| unknown (`-1`) | ~69k | ~80k |

**Pattern:** OpenHands has many unknown resolve labels. Successes tend to be shorter than failures.

## By category (partial)

- bug-fix dominates volume
- bug-fix resolve rate highest among labeled outcomes
- feature-request lower

## Early signal (15-trial sample — directional only)

From `004_turn_sample_early_signal.sql`:

| Progress | Success avg edits | Failed avg edits |
|---|---:|---:|
| 0–25% | 3.5 | 4.6 |
| 75–100% | 11.2 | 22.8 |

Successes edit earlier and less; failures accumulate edits and tool errors.

**Caveat:** n=15 trials — rerun at n=200–500 before drawing conclusions.

## Infrastructure lessons

- Unnesting all messages on full corpus OOMs even with 4GB cap
- Use `trial_summary` for cheap aggregates
- Use `extract_turn_sample.py` for turn-level work (sampled)
- Stale DuckDB temp dir (`duckdb/open_swe.duckdb.tmp`) once grew to 31GB — safe to delete if no active query

## Queries to re-run on full data

```bash
uv run python scripts/duckdb_init.py --refresh-summaries
uv run python scripts/duckdb_query.py -f analytics/queries/001_dataset_profile.sql
uv run python scripts/duckdb_query.py -f analytics/queries/002_resolved_by_harness.sql
uv run python scripts/duckdb_query.py -f analytics/queries/003_trial_eda.sql
```

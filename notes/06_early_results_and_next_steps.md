# Early results and next steps

## Turn sample (15 trials, exploratory)

- **2,278 turn rows** from 15 stratified trials
- Extraction: ~26s, DuckDB staging INSERT per parquet file
- Query: `analytics/queries/004_turn_sample_early_signal.sql`

### Directional signal (not conclusive at n=15)

| Progress | Success avg edits | Failed avg edits |
|---|---:|---:|
| 0–25% | 3.5 | 4.6 |
| 75–100% | 11.2 | 22.8 |

Successes edit earlier and less; failures accumulate edits and tool errors.

## Planned work

1. Scale turn sample to **200–500 trials**
2. Simple classifier (logistic regression on cum_edits, cum_tests, cum_tool_errors at each progress bucket)
3. Plot **AUC vs % through run** — tests H1/H2
4. Formalize quality rubric (SWE-Prime-style dimensions)
5. When AMD/Kaggle GPU available: small SFT ablation (resolved-only vs all vs top-quality subset)

## Open questions

See `analytics/research/questions.md`

## Sources

- Local: `outputs/turn_sample.parquet`, `analytics/query_log/`

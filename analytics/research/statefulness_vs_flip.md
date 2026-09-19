# Statefulness vs Composer flip point

Date: 2026-09-18. Exploratory. Computed by
`src/openswe_traces/synth/statefulness.py` from each unit's existing
hidden (or packaged) tests. `property-1pc` is listed then dropped from
the correlation because the verifier was voided (B4).

Components, per test, then averaged:

- **(a) mean_calls** — exported-API calls before the last assertion
  (a `for` body that contains the assertion is counted once).
- **(b) seq_frac** — fraction of tests with ≥ 2 such calls.
- **(c) mean_entries** — distinct exported entry points among those calls.
- **(d) dynamic** — any test uses `go`, `sync.`, or `time.` waits/clocks.

| unit | family | mean_calls (a) | seq_frac (b) | mean_entries (c) | dynamic (d) | n_tests | Composer flip |
|---|---|---:|---:|---:|---|---:|---|
| `dynamic-pipeline` | dynamic | 10.25 | 0.88 | 4.25 | yes | 8 | A3 |
| `spec-reimpl-bb` | spec-bb | 15.00 | 1.00 | 8.50 | no | 4 | A1 |
| `property-backoff` | property | 3.00 | 1.00 | 1.00 | no | 1 | A0 |
| `spec-bb-chain` | spec-bb | 5.25 | 1.00 | 4.00 | no | 4 | A0 |
| `spec-bb-bucket` | spec-bb | 5.75 | 1.00 | 2.25 | no | 4 | A0 |
| `property-policy` | property | 2.00 | 0.67 | 1.33 | no | 3 | A0 |
| `property-1pc` | property | 0.00 | 0.00 | 0.00 | no | 3 | excluded |
| `dynamic-snapshot` | dynamic | 39.00 | 1.00 | 4.00 | no | 2 | A0 |
| `dynamic-latch` | dynamic | 8.00 | 1.00 | 5.50 | yes | 2 | A0 |
| `file-exclude-filter` | ablation | 1.00 | 0.33 | 0.33 | no | 3 | A0 |
| `file-filter` | ablation | 1.00 | 0.33 | 0.33 | no | 3 | A0 |
| `revivelib-runner` | ablation | 1.00 | 0.33 | 1.00 | no | 3 | A0 |

## Spearman (exploratory, n=11)

Flip points encoded as A0=0, A1=1, A3=3. Most units sit at A0, so the
ranks are heavily tied; treat ρ as a directional hint, not a result.

| component | Spearman ρ | n |
|---|---:|---:|
| (a) mean_calls | 0.517 | 11 |
| (b) seq_frac | 0.097 | 11 |
| (c) mean_entries | 0.584 | 11 |
| (d) dynamic | 0.442 | 11 |

Reproduce:

```
uv run python -c "from openswe_traces.synth.statefulness import write_report; print(write_report()[0])"
```

# Research log

Chronological session findings. Background thesis, literature, and sources live in [`notes/`](../notes/README.md).

One bullet per session — link the query file you used.

## Template

```markdown
### YYYY-MM-DD — short title
- **Question:**
- **Query:** `analytics/queries/00X_name.sql`
- **Finding:**
- **Next:**
```

---

### 2026-09-20 — certified-hard failure anatomy vs Open-SWE-Traces
- **Question:** Do the L0 stopping-rule / near-miss / withheld-clause / anti-inferable findings reproduce on real traces?
- **Query:** `uv run python scripts/failure_anatomy.py` (stream → `outputs/failure_anatomy.parquet`); `analytics/queries/005_failure_anatomy.sql`
- **Finding:** Stopping rule yes (97.4% last-test green on failed, 0.2% uncertain) but not failure-specific (h≈0 vs resolved). Per-test near-miss unmeasurable. C is 6/40 vs original 76%. Anti-inferable 2/40 vs 29%.
- **Next:** Repo-checkout pass on the 6 C and 2 anti rows if we want a less-lower-bound Q4.

### 2026-09-17 — project bootstrap
- **Question:** What do we have locally while download is partial?
- **Query:** `analytics/queries/001_dataset_profile.sql`
- **Finding:** _(fill after first run)_
- **Next:** Early-turn success prediction features

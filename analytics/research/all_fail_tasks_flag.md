# Flag: all_fail tasks are not "unsolvable" — revisit with stronger models

2026-09-17. Source: `outputs/task_difficulty.parquet` (`difficulty_bucket == 'all_fail'`).

| item | value |
|---|---|
| all_fail instances | 14,205 (33.5% of instances) |
| trajectories on them | 171,234 |
| mean rollouts per task | 12.1 |

Every labeled rollout on these tasks failed. That mixes three populations we cannot yet separate:

1. **Teacher-limited** — solvable, but Minimax-M2.5 / Qwen3.5-122B / Qwen3.6-27B could not. A stronger
   model would produce resolved traces here, and those would be the highest-value SFT data in the corpus
   (they teach what the student cannot already do).
2. **Broken environment** — flaky tests, missing deps, gold patch depends on unstated context.
3. **Underspecified issue** — text insufficient to reach the gold patch (the class SWE-bench Verified removed).

Decision for the current curve: excluded from training manifests (DAPO-style learnability cut), together
with `all_pass`. Not deleted.

## Follow-ups (not scheduled)

- Roll out a strong model (Claude / GPT / DeepSeek V4 Pro) on a 200-task stratified sample of all_fail;
  the flip rate estimates population 1's size.
- Per-repo all_fail rate: repos near 100% point at environment problems (population 2).
- Issue length / gold-patch size vs all_fail within a repo: short issue + large gold patch suggests
  population 3.
- If population 1 is large, these tasks are the best candidates for SWE-smith-style synthetic expansion
  and for the "critic masking" arm (SRFT), since their unresolved traces likely contain partially correct
  prefixes.

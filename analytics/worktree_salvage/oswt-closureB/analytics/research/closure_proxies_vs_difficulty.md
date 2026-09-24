# Closure structure of the gold patch vs per-instance solve rate

Generated 2026-09-19 17:08 UTC by `scripts/closure_proxies.py` (one streaming DuckDB pass over 212 shards of `traces_data/`; per-instance proxies and solve rates in `outputs/closure_proxies.parquet`).

Question: is task difficulty driven by raw patch *size*, or by the *closure structure* of the change — a change whose added code mostly references itself (new symbols calling new symbols) vs a same-sized change that mostly edits existing call sites?

## Proxies (computed from the gold patch text alone)

All proxies are per `instance_id` over the gold (`reference_patch`) patch, which is identical across shards (deduped at merge; `max` kept). `internal_refs` counts occurrences, in added lines, of names defined elsewhere in the same patch; the line defining a name does not count its own occurrence. `boundary_refs` counts occurrences, in added lines, of identifiers that appear in the patch's context or removed lines and are not new defs — language keywords are excluded (per-language stoplist), builtins are kept. `ratio = internal_refs / max(1, boundary_refs)`, `new_frac = n_new_defs / max(1, added_lines)`, `edit_frac` = fraction of hunks with both removed and added lines.

Definition regexes per language (first match per added line, keyword-blacklisted): python `def|class`; go `func` incl. receivers; typescript/javascript `function|class|arrow-const|interface|type|get/set|method shorthand`; java `class|interface|enum|record` and single-line method signatures; rust `fn|struct|enum|trait|union|mod|type`; c/cpp `class|struct|enum|union` and single-line function signatures; php `class|interface|trait|enum|function`; ruby `def|class|module`; csharp `class|interface|struct|enum|record` and single-line methods. Unknown languages get no def regex (`n_new_defs == internal_refs == 0`).

## Corpus and subset

Instances streamed: **42,413**; with at least one labeled rollout: 39,702; analysis subset (n_labeled >= 3): **38,294** (solve_rate mean 0.422). Gold patch present for 42,413/42,413 instances.

Proxy means on the subset: n_hunks 11.69, n_files 4.92, added_lines 127.49, removed_lines 52.61, n_new_defs 3.63, internal_refs 9.61, boundary_refs 252.46, ratio 0.16, new_frac 0.04, edit_frac 0.55

## 1. Spearman of each proxy vs solve_rate (n_labeled >= 3)

| proxy | Spearman |
|---|---|
| n_hunks | -0.2761 |
| n_files | -0.2366 |
| added_lines | -0.3023 |
| removed_lines | -0.2295 |
| n_new_defs | -0.1618 |
| internal_refs | -0.2094 |
| boundary_refs | -0.2804 |
| ratio | -0.1324 |
| new_frac | +0.0115 |
| edit_frac | +0.0030 |

Per language (languages with >= 100 subset instances):

| language | n | n_hunks | n_files | added_lines | removed_lines | n_new_defs | internal_refs | boundary_refs | ratio | new_frac | edit_frac |
|---|---|---|---|---|---|---|---|---|---|---|---|
| go | 5,359 | -0.2934 | -0.2544 | -0.3038 | -0.2530 | -0.2057 | -0.2037 | -0.3057 | -0.0601 | -0.0537 | -0.0302 |
| java | 1,340 | -0.2639 | -0.2360 | -0.2007 | -0.2505 | -0.1501 | -0.1608 | -0.2731 | -0.0767 | -0.0647 | -0.0770 |
| javascript | 3,038 | -0.3143 | -0.2811 | -0.2989 | -0.2841 | -0.1853 | -0.1716 | -0.3115 | -0.1231 | -0.1214 | +0.0360 |
| php | 1,057 | -0.2622 | -0.2244 | -0.2413 | -0.2286 | -0.1491 | -0.1515 | -0.2400 | -0.0947 | -0.0572 | +0.0643 |
| python | 21,917 | -0.3039 | -0.2562 | -0.3536 | -0.2307 | -0.1989 | -0.2343 | -0.3175 | -0.1511 | +0.0281 | +0.0033 |
| rust | 1,891 | -0.2615 | -0.2486 | -0.2922 | -0.2067 | -0.2040 | -0.1786 | -0.2532 | -0.1150 | -0.0646 | +0.0428 |
| typescript | 3,576 | -0.3378 | -0.3176 | -0.3673 | -0.2744 | -0.2639 | -0.2475 | -0.3364 | -0.2015 | -0.1892 | +0.1258 |

## 2. Does closure structure add to size? (WLS)

WLS on 38,294 instances (weight = n_labeled, total weight 332,301), grouped 80/20 split by repo (seed 42): 30,661 train instances over 2,960 repos, 7,633 test instances over 740 repos. Numeric inputs are winsorized to the train split's 1st/99th percentile then standardized; language one-hots drop the most frequent level. R² is the unweighted fit metric; the fit itself is weighted. Baseline = size + language; full = baseline + `ratio`, `new_frac`, `edit_frac`, `log1p(n_new_defs)`.

| model | train R² | train ρ | held-out R² | held-out ρ |
|---|---|---|---|---|
| baseline (size + language) | 0.1453 | +0.3895 | 0.1222 | +0.3613 |
| baseline + structure | 0.1537 | +0.4008 | 0.1304 | +0.3738 |
| Δ (structure − baseline) | +0.0084 | — | +0.0081 | +0.0126 |

Standardized coefficients of the full model (solve_rate per 1 sd):

| rank | feature | std coef | direction |
|---|---|---|---|
| 1 | log1p(added_lines) | -0.1203 | lower solve_rate |
| 2 | language_typescript | -0.0541 | lower solve_rate |
| 3 | language_rust | -0.0510 | lower solve_rate |
| 4 | language_go | -0.0458 | lower solve_rate |
| 5 | new_frac | +0.0401 | higher solve_rate |
| 6 | language_javascript | -0.0398 | lower solve_rate |
| 7 | language_java | -0.0331 | lower solve_rate |
| 8 | log1p(n_files) | -0.0300 | lower solve_rate |
| 9 | log1p(n_new_defs) | -0.0241 | lower solve_rate |
| 10 | language_php | -0.0149 | lower solve_rate |
| 11 | edit_frac | -0.0146 | lower solve_rate |
| 12 | ratio | +0.0088 | higher solve_rate |
| 13 | language_cpp | -0.0000 | lower solve_rate |
| 14 | language_c | +0.0000 | higher solve_rate |

**Structure adds to size: a little.** Held-out R² 0.1222 → 0.1304 (Δ +0.0081) and held-out Spearman +0.3613 → +0.3738 (Δ +0.0126). The increment comes from `new_frac` (std coef +0.0401: more new definitions per added line → *higher* solve rate), not from self-reference: the size-matched comparison below is flat (pooled top−bottom `ratio` quartile -0.004) and `ratio`'s standardized coefficient is ~0 (+0.0088).

## 3. Size-matched comparison: added_lines decile × ratio quartile

Instances binned by `added_lines` decile; within each decile, the top ratio quartile (ratio > q75) vs bottom quartile (ratio < q25). 'pooled' averages all top vs all bottom members across deciles. Deciles with fewer than two distinct ratio values are reported as n/a.

| decile | n | mean added_lines | n_top | solve_rate top | n_bottom | solve_rate bottom | diff |
|---|---|---|---|---|---|---|---|
| 1 (0–6) | 4,477 | 3.6 | n/a | n/a | n/a | n/a | n/a |
| 2 (6–11) | 3,595 | 8.9 | n/a | n/a | n/a | n/a | n/a |
| 3 (11–18) | 3,827 | 14.8 | n/a | n/a | n/a | n/a | n/a |
| 4 (18–27) | 3,792 | 22.8 | n/a | n/a | n/a | n/a | n/a |
| 5 (27–38) | 3,535 | 32.7 | n/a | n/a | n/a | n/a | n/a |
| 6 (38–55) | 3,925 | 46.5 | n/a | n/a | n/a | n/a | n/a |
| 7 (55–80) | 3,662 | 67.0 | n/a | n/a | n/a | n/a | n/a |
| 8 (80–124) | 3,863 | 100.2 | 966 | 0.363 | 965 | 0.318 | +0.045 |
| 9 (124–217) | 3,790 | 164.3 | 948 | 0.261 | 948 | 0.275 | -0.015 |
| 10 (217–64455) | 3,828 | 819.6 | 957 | 0.205 | 957 | 0.246 | -0.042 |
| pooled | 38,294 | — | 2,871 | 0.277 | 2,870 | 0.280 | -0.004 |

## Reproduce and runtime

```bash
uv run python scripts/closure_proxies.py          # full stream + analysis
uv run pytest tests/test_closure_proxies.py
uv run ruff check .
```

Streaming pass: 0s over 212 shards; analysis: 0s. Full run logged to `outputs/closure_B.log`.

## Notes

- Analysis subset needs n_labeled >= 3 labeled rollouts (`resolved in (0, 1)`); `solve_rate = n_resolved / n_labeled`.
- `boundary_refs` counts only identifiers that already appear in the patch's context or removed lines; identifiers introduced *only* in added lines (e.g. a new local variable) count in neither bucket.
- A name defined in the patch is treated as new even if the patch also edits pre-existing code with that name; pure moves (same def in removed and added) are rare and counted as new.
- Method/function signatures spanning multiple lines (brace on its own line) are not recognized as defs; single-line signatures only.
- `ratio` is degenerate (0) for the majority of small patches that define no new symbols, so the size-matched table reports n/a for the lower deciles and is only informative where patches are big enough to contain new definitions (the largest deciles).

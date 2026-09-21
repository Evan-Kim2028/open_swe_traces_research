# Symptom-to-cause distance and site count as difficulty knobs

2026-09-19. Measures from `openswe_traces.symptom_distance` (issue-text mentions vs gold src-hunk locations); instance frame `outputs/derived/symptom_distance_frame.parquet`, trajectory frame `symptom_distance_traj.parquet`. No solver calls. Log `outputs/closure_N.log`.

Hypothesis (from closure-H labels): inside the feasible band, difficulty is driven by symptom→cause distance and required site count, not size.

## T1 — distance vs solve rate, feasible band

Feasible band: n_resolved>=1, n_labeled>=5, gold touches src → n=20,849. WLS of solve_rate on dist_class + log1p(src_added) + language, weights n_labeled, HC1 SEs. Reference class: `same_file_same_func` (the issue points exactly at the cause).

| dist_class | n inst | raw solve | WLS coef [95% CI] | repo-FE coef | marginal rate @ mean size |
|---|---|---|---|---|---|
| same_file_same_func (ref) | 301 | 0.763 | 0 (ref) | 0 (ref) | 0.766 |
| same_file_other_func | 1,271 | 0.710 | -0.043 [-0.078, -0.007] | -0.044 [-0.081, -0.008] | 0.710 |
| other_file | 15,131 | 0.738 | -0.019 [-0.051, +0.012] | -0.035 [-0.068, -0.002] | 0.725 |
| no_mention | 4,146 | 0.668 | -0.023 [-0.056, +0.010] | -0.025 [-0.060, +0.009] | 0.676 |

log1p(src_added) coef -0.059 [-0.062, -0.056]; R²=0.115.


Size-decile matched table — weighted mean solve_rate per src_added decile × dist_class (n instances):

| size decile | same_file_same_func | same_file_other_func | other_file | no_mention |
|---|---|---|---|---|
| d1 | 0.90 (32) | 0.84 (132) | 0.83 (1,515) | 0.79 (406) |
| d2 | 0.90 (36) | 0.75 (121) | 0.82 (1,571) | 0.75 (357) |
| d3 | 0.81 (32) | 0.78 (144) | 0.80 (1,596) | 0.72 (313) |
| d4 | 0.85 (33) | 0.75 (122) | 0.75 (1,577) | 0.70 (353) |
| d5 | 0.77 (27) | 0.74 (118) | 0.75 (1,581) | 0.70 (359) |
| d6 | 0.83 (27) | 0.71 (111) | 0.74 (1,536) | 0.68 (410) |
| d7 | 0.73 (33) | 0.69 (108) | 0.70 (1,480) | 0.65 (464) |
| d8 | 0.59 (23) | 0.68 (102) | 0.64 (1,469) | 0.62 (491) |
| d9 | 0.71 (23) | 0.58 (116) | 0.63 (1,401) | 0.60 (545) |
| d10 | 0.54 (35) | 0.57 (197) | 0.58 (1,405) | 0.57 (448) |

**Verdict T1:** distance helps but modestly — at fixed size the marginal solve rate falls 76.6% (symptom names the cause) → 67.6% (no location named), ~9pp, with other_file ~ same_file_other_func in between; monotone ordering does NOT hold (mentioning the right file but wrong func is not better than pointing at the wrong file entirely).

## T2 — site count vs size

Same feasible band (n=20,849). WLS solve_rate ~ log1p(src_added) + log1p(n_gold_funcs) + log1p(n_gold_files) + language:

| term | WLS coef [95% CI] |
|---|---|
| log1p(src_added) | -0.053 [-0.057, -0.049] |
| log1p(n_gold_funcs) | +0.001 [-0.009, +0.011] |
| log1p(n_gold_files) | -0.026 [-0.041, -0.012] |

Weighted R² — size only 0.115 | counts only 0.082 | both + language 0.116 (no language: size 0.065, counts 0.036). corr(log1p src_added, log1p n_gold_funcs) = 0.56.


**Verdict T2:** it is mostly the lines, not the site count — at fixed src_added, n_gold_funcs contributes nothing (+0.001 ns) and n_gold_files a small extra penalty (-0.026); adding counts to size moves R² by ~0.001.

## T3 — feasibility gate (all_fail)

Logistic of all_fail (n_resolved==0), n_labeled>=3 → n=38,294, base rate 41.8%; grouped 80/20 by repo, held-out AUC. Distance block = dist_class dummies + log1p(n_gold_funcs) + log1p(n_gold_files) added to closure-G's size+verifier spec.

| feature set | held-out AUC |
|---|---|
| size | 0.6588 |
| size+verifier | 0.7080 |
| size+verifier+dist | 0.7078 |
| full | 0.7061 |

Standardized coefs of the distance block in the full model:

| term | std coef |
|---|---|
| dc_other_file | +0.101 |
| dc_same_file_other_func | +0.097 |
| dc_no_mention | +0.081 |
| log1p(n_gold_files) | +0.054 |
| dc_no_gold_src | +0.050 |
| log1p(n_gold_funcs) | -0.035 |

**Verdict T3:** distance does NOT gate feasibility — AUC is flat (0.7080 → 0.7078); the distance block is redundant with size+verifier for predicting all_fail.


## T4 — mechanism: acted at mention vs acted at gold

Labeled trajectories on instances with ≥1 mention and ≥1 gold src hunk: n=255,676. `acted_at_mention` = model patch touches a mentioned file (or a hunk ctx names a mentioned func); `acted_at_gold` = patch_hunk_coverage > 0 (a model hunk overlaps a gold hunk ±10 lines).

**ALL** (P(resolved) per cell):

| cell | n | P(resolved) |
|---|---|---|
| mention_only | 3,779 | 0.337 |
| gold_only | 179,850 | 0.464 |
| mention+gold | 54,462 | 0.519 |
| neither | 17,585 | 0.297 |
| at gold (any) | 234,312 | 0.477 |

**minisweagent/qwen38_27b** (P(resolved) per cell):

| cell | n | P(resolved) |
|---|---|---|
| mention_only | 615 | 0.459 |
| gold_only | 55,736 | 0.524 |
| mention+gold | 17,240 | 0.568 |
| neither | 3,063 | 0.421 |
| at gold (any) | 72,976 | 0.534 |

**minisweagent/qwen36_27b** (P(resolved) per cell):

| cell | n | P(resolved) |
|---|---|---|
| mention_only | 1,289 | 0.237 |
| gold_only | 45,948 | 0.390 |
| mention+gold | 12,754 | 0.445 |
| neither | 6,771 | 0.210 |
| at gold (any) | 58,702 | 0.402 |

**sweagent/qwen36_27b** (P(resolved) per cell):

| cell | n | P(resolved) |
|---|---|---|
| mention_only | 210 | 0.462 |
| gold_only | 24,544 | 0.534 |
| mention+gold | 7,833 | 0.559 |
| neither | 1,312 | 0.506 |
| at gold (any) | 32,377 | 0.540 |

Share of unresolved trajectories that are at-mention-not-gold, by dist_class:

| dist_class | n unresolved | share mention_only | share at gold |
|---|---|---|---|
| same_file_same_func | 2,075 | 9.6% | 88.8% |
| same_file_other_func | 10,858 | 6.1% | 91.7% |
| other_file | 124,468 | 1.3% | 89.0% |

**Verdict T4:** mechanism confirmed — patches that act only at the mentioned location resolve 33.7% vs 47.7% when they reach a gold site (~14pp gap, consistent across top-3 combos); but mention-only patches are rare among unresolved failures (1–10% by class) because issue mentions and gold files mostly overlap by construction of the corpus.


## T5 — LLM labels × dist_class

293 closure-H labelled failures; counts per (label × dist_class):

| label | n | same_file_same_func | same_file_other_func | other_file | no_mention | no_gold_src |
|---|---|---|---|---|---|---|
| wrong_root_cause | 73 | 1 | 4 | 52 | 15 | 1 |
| incomplete_stopped_early | 64 | 0 | 4 | 48 | 12 | 0 |
| fixed_symptom_not_cause | 61 | 1 | 4 | 40 | 15 | 1 |
| missed_second_site | 34 | 0 | 3 | 22 | 9 | 0 |
| other | 34 | 0 | 5 | 19 | 10 | 0 |
| misread_issue | 15 | 0 | 3 | 11 | 0 | 1 |
| broke_other_test | 11 | 1 | 0 | 6 | 4 | 0 |
| environment | 1 | 0 | 0 | 1 | 0 | 0 |

Share of labels landing in the two 'pointed elsewhere' classes (other_file / same_file_other_func): wrong_root_cause + fixed_symptom_not_cause 74.6% vs all labels 75.8%.


**Verdict T5:** labels do NOT concentrate in far-distance classes — wrong_root_cause + fixed_symptom_not_cause land in other_file/same_file_other_func at 74.6% vs 75.8% for all labels; most labelled fails sit in other_file simply because most feasible-band instances are other_file.

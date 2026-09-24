# Paired failure analysis: why do models fail SMALL feasible tasks?

2026-09-19. Within-instance paired comparison on Open-SWE-Traces: for every
(instance, harness/teacher combo) group with ≥1 pass and ≥1 fail rollout on a mixed
instance (n_labeled ≥ 5, 0 < n_resolved < n_labeled), compare the behavior of failing
vs passing siblings. The task is held fixed inside a pair, so differences are
behavioral, not difficulty. Strata by gold patch size: SMALL = added_lines ≤ 30,
LARGE = ≥ 150. Bootstrap CIs resample instances (n_boot=1000).

Groups: 15,146 pair-eligible (instance × combo) groups,
41,362 trajectories.

| stratum | trajectories | groups | instances |
|---|---:|---:|---:|
| LARGE | 4,245 | 1,558 | 1,204 |
| MID | 16,317 | 5,973 | 4,293 |
| SMALL | 20,800 | 7,615 | 5,324 |

## Paired differences: mean(fail) − mean(pass) within group, averaged over instances

Values are `diff [95% CI]`. Positive = failing rollouts do more of it.
Top-3 combos are the three strongest by IRT theta (closure-C): minisweagent/qwen38_27b, sweagent/qwen36_27b, sweagent/qwen35_122b.

### SMALL (added_lines ≤ 30)

| feature | all (median) | all | minisweagent/qwen38_27b | sweagent/qwen36_27b | sweagent/qwen35_122b |
|---|---|---|---|---|---|
| n_turns | +2.000 | +3.807 [+2.780,+4.893] | +1.966 [-0.013,+3.998] | +2.811 [-0.127,+5.681] | +5.112 [-0.008,+10.486] |
| n_assistant_turns | +0.938 | +1.900 [+1.404,+2.438] | +0.889 [-0.032,+1.832] | +1.406 [-0.064,+2.841] | +2.556 [-0.004,+5.243] |
| n_tool_calls | +1.000 | +1.907 [+1.371,+2.453] | +1.078 [-0.015,+2.219] | +1.406 [-0.064,+2.841] | +2.556 [-0.004,+5.243] |
| assistant_chars | +165.125 | +528.842 [+390.883,+682.153] | -1.341 [-111.717,+102.576] | +636.710 [+146.344,+1176.101] | +3396.180 [+1608.160,+5062.713] |
| n_tool_errors | +0.000 | +0.111 [+0.024,+0.189] | +0.021 [-0.163,+0.193] | -0.016 [-0.251,+0.187] | -0.055 [-0.281,+0.153] |
| n_edit_calls | +0.000 | +0.385 [+0.227,+0.532] | -0.034 [-0.301,+0.200] | +0.645 [+0.126,+1.151] | +1.671 [+0.730,+2.531] |
| n_test_runs | +0.000 | +0.516 [+0.398,+0.638] | +0.055 [-0.234,+0.344] | +0.104 [-0.180,+0.419] | +1.261 [+0.541,+1.910] |
| n_repro_calls | +0.500 | +0.787 [+0.531,+1.051] | +0.212 [-0.366,+0.770] | +0.453 [-0.338,+1.247] | +2.514 [+1.156,+3.821] |
| turn_of_first_edit | +0.000 | +0.330 [+0.152,+0.506] | +0.454 [-0.143,+1.045] | -0.201 [-0.484,+0.105] | +0.441 [-0.532,+1.492] |
| turn_of_last_edit | +1.000 | +1.774 [+1.255,+2.367] | +0.981 [-0.235,+2.128] | +1.717 [+0.227,+3.083] | +3.565 [+0.956,+6.108] |
| n_files_read | +0.250 | +0.654 [+0.372,+0.975] | +0.596 [-0.375,+1.662] | +0.103 [-0.554,+0.760] | +0.667 [-0.996,+2.250] |
| n_files_edited | +0.000 | +0.190 [+0.074,+0.307] | -0.109 [-0.363,+0.120] | +0.236 [-0.118,+0.553] | +0.912 [+0.046,+1.675] |
| n_model_hunks | +0.000 | +1.175 [+0.168,+2.336] | +0.168 [+0.031,+0.308] | -2.935 [-11.476,+1.751] | +0.900 [-0.524,+2.116] |
| patch_hunk_coverage | +0.000 | -0.108 [-0.115,-0.101] | -0.054 [-0.065,-0.043] | -0.054 [-0.076,-0.034] | -0.098 [-0.127,-0.066] |
| extra_hunks | +0.000 | +1.602 [+0.574,+2.771] | +0.349 [+0.233,+0.486] | -2.736 [-11.150,+1.938] | +1.164 [-0.257,+2.347] |
| extra_hunk_lines | +0.500 | +386.012 [+240.306,+560.639] | +3.229 [+2.047,+4.437] | +845.052 [+73.499,+2531.585] | +459.933 [-11.389,+1098.014] |
| model_lines_over_gold | +0.071 | +72.613 [+37.195,+121.067] | +0.486 [+0.239,+0.788] | +108.401 [+5.355,+362.477] | +24.250 [-86.384,+122.733] |
| first_repro_call | +0.000 | +0.214 [+0.040,+0.381] | +0.086 [-0.293,+0.445] | +0.269 [-0.283,+0.865] | +0.436 [-0.824,+1.781] |
| last_test_call | +0.500 | +2.059 [+1.484,+2.668] | +0.728 [-0.397,+1.919] | +1.189 [-0.490,+2.948] | +5.290 [+2.162,+8.317] |
| ran_repro_before_edit | +0.000 | +0.005 [-0.005,+0.014] | +0.013 [-0.011,+0.035] | -0.027 [-0.061,+0.002] | +0.013 [-0.021,+0.051] |
| ran_tests_after_last_edit | +0.000 | +0.022 [+0.010,+0.035] | +0.011 [-0.009,+0.032] | -0.018 [-0.062,+0.027] | +0.009 [-0.043,+0.061] |
| edited_test_file | +0.000 | +0.097 [+0.089,+0.106] | +0.001 [-0.002,+0.004] | -0.001 [-0.015,+0.012] | +0.143 [+0.102,+0.184] |
| touched_test_path | +0.000 | +0.002 [-0.005,+0.010] | -0.026 [-0.048,-0.004] | -0.001 [-0.007,+0.003] | +0.011 [-0.012,+0.036] |
| submitted_empty_patch | +0.000 | +0.005 [+0.004,+0.007] | +0.000 [+0.000,+0.000] | +0.000 [+0.000,+0.000] | +0.000 [+0.000,+0.000] |

### LARGE (added_lines ≥ 150)

| feature | all (median) | all | minisweagent/qwen38_27b | sweagent/qwen36_27b | sweagent/qwen35_122b |
|---|---|---|---|---|---|
| n_turns | -1.000 | -2.446 [-4.766,-0.141] | -5.168 [-9.496,-0.857] | -2.600 [-6.971,+1.465] | -4.900 [-32.013,+20.954] |
| n_assistant_turns | -0.500 | -1.121 [-2.287,+0.017] | -2.290 [-4.355,-0.256] | -1.300 [-3.486,+0.732] | -2.450 [-16.006,+10.477] |
| n_tool_calls | -0.500 | -1.324 [-2.541,-0.117] | -2.878 [-5.265,-0.478] | -1.300 [-3.486,+0.732] | -2.450 [-16.006,+10.477] |
| assistant_chars | +23.000 | +35.184 [-143.477,+210.397] | -268.841 [-506.064,-62.280] | +487.654 [-75.951,+1098.162] | -772.375 [-6392.514,+4249.923] |
| n_tool_errors | +0.000 | +0.042 [-0.186,+0.269] | +0.090 [-0.400,+0.569] | -0.080 [-0.478,+0.320] | +0.450 [-0.051,+1.100] |
| n_edit_calls | +0.000 | -0.188 [-0.561,+0.200] | +0.066 [-0.548,+0.664] | +0.105 [-0.595,+0.807] | -1.325 [-4.775,+2.150] |
| n_test_runs | +0.000 | +0.040 [-0.188,+0.284] | -0.515 [-1.009,-0.089] | -0.232 [-0.683,+0.195] | -0.625 [-3.826,+2.226] |
| n_repro_calls | +0.000 | -0.744 [-1.349,-0.101] | -0.922 [-2.230,+0.431] | -0.220 [-1.459,+0.901] | -2.175 [-8.376,+3.926] |
| turn_of_first_edit | +0.000 | +0.130 [-0.364,+0.633] | -0.727 [-2.097,+0.479] | -0.315 [-0.824,+0.188] | +0.300 [-5.352,+6.579] |
| turn_of_last_edit | +0.000 | -1.067 [-2.339,+0.179] | -2.009 [-4.570,+0.730] | -1.415 [-3.677,+0.634] | -2.975 [-18.876,+11.653] |
| n_files_read | -0.500 | -1.044 [-2.035,-0.014] | -1.168 [-3.549,+1.471] | -0.059 [-1.210,+1.234] | -1.525 [-7.277,+4.127] |
| n_files_edited | +0.000 | -0.319 [-0.640,+0.031] | -0.073 [-0.758,+0.600] | +0.161 [-0.327,+0.659] | -0.475 [-2.827,+2.201] |
| n_model_hunks | +0.000 | -4.185 [-9.699,-0.941] | -16.171 [-40.322,-2.965] | -0.276 [-2.542,+2.409] | -2.600 [-6.927,+1.001] |
| patch_hunk_coverage | -0.011 | -0.066 [-0.077,-0.055] | -0.052 [-0.071,-0.036] | -0.047 [-0.070,-0.026] | -0.093 [-0.213,+0.012] |
| extra_hunks | +0.000 | +0.093 [-0.643,+0.709] | -1.615 [-3.483,-0.144] | +0.044 [-1.503,+1.866] | -1.725 [-5.325,+0.626] |
| extra_hunk_lines | +0.000 | +95.348 [-9.729,+225.675] | -18.508 [-49.986,+5.078] | +310.990 [-208.812,+1065.047] | +42.900 [-31.311,+129.954] |
| model_lines_over_gold | -0.007 | +0.121 [-0.080,+0.352] | -0.083 [-0.140,-0.033] | -0.059 [-0.773,+0.522] | -0.572 [-1.604,+0.220] |
| first_repro_call | +0.000 | -0.037 [-0.473,+0.425] | -0.087 [-0.958,+0.801] | -0.327 [-1.161,+0.578] | -4.950 [-16.531,+6.900] |
| last_test_call | +0.000 | -1.595 [-3.087,-0.116] | -2.853 [-5.606,-0.350] | -2.066 [-4.737,+0.449] | -4.900 [-18.753,+7.179] |
| ran_repro_before_edit | +0.000 | +0.010 [-0.008,+0.032] | +0.020 [-0.015,+0.055] | +0.015 [-0.037,+0.059] | -0.075 [-0.275,+0.125] |
| ran_tests_after_last_edit | +0.000 | +0.006 [-0.018,+0.031] | -0.023 [-0.058,+0.010] | -0.049 [-0.117,+0.015] | -0.025 [-0.300,+0.250] |
| edited_test_file | +0.000 | +0.044 [+0.032,+0.057] | +0.004 [-0.007,+0.015] | -0.005 [-0.027,+0.017] | +0.150 [-0.050,+0.350] |
| touched_test_path | +0.000 | +0.002 [-0.016,+0.022] | -0.009 [-0.058,+0.036] | -0.007 [-0.020,+0.000] | +0.025 [-0.175,+0.200] |
| submitted_empty_patch | +0.000 | +0.005 [+0.002,+0.009] | +0.000 [+0.000,+0.000] | +0.000 [+0.000,+0.000] | +0.000 [+0.000,+0.000] |

## Failure taxonomy (rule-based, on the FAIL member of each pair)

Note: every labeled trajectory in this corpus ends with a submit marker
(`finish`/`submit` call or final submit text — the harness records completion, not
truncation), so `stop_reason` is 'submit' for ~100% of rows and the `timeout`
bucket is structurally empty. Turn-cap/crash failures are not observable among
labeled rollouts; read 'timeout' as absorbed into the other buckets.

### SMALL

| combo | n_fail | timeout | no_patch | test_edit | wrong_site | partial | partial_extra | over_edit | complete_but_wrong | other |
|---|---|---|---|---|---|---|---|---|---|---|
| all | 10340 | 0.0% | 0.4% | 12.5% | 15.4% | 25.6% | 20.6% | 3.4% | 22.1% | 0.0% |
| minisweagent/qwen38_27b | 1993 | 0.0% | 0.0% | 1.0% | 9.6% | 27.1% | 24.2% | 4.9% | 33.2% | 0.0% |
| sweagent/qwen36_27b | 590 | 0.0% | 0.0% | 3.1% | 8.3% | 35.9% | 26.6% | 2.0% | 24.1% | 0.0% |
| sweagent/qwen35_122b | 606 | 0.0% | 0.0% | 18.5% | 17.2% | 16.0% | 20.6% | 10.1% | 17.7% | 0.0% |
| minisweagent/qwen36_27b | 3529 | 0.0% | 0.0% | 3.3% | 24.6% | 28.2% | 23.4% | 1.6% | 18.9% | 0.0% |
| openhands/minimax_m25 | 965 | 0.0% | 4.0% | 4.9% | 13.3% | 29.2% | 17.6% | 3.2% | 27.8% | 0.0% |
| openhands/qwen35_122b | 1390 | 0.0% | 0.3% | 65.9% | 4.5% | 9.1% | 9.2% | 2.2% | 8.8% | 0.0% |
| sweagent/minimax_m25 | 1267 | 0.0% | 0.0% | 5.3% | 15.2% | 30.8% | 19.1% | 4.9% | 24.8% | 0.0% |

### LARGE

| combo | n_fail | timeout | no_patch | test_edit | wrong_site | partial | partial_extra | over_edit | complete_but_wrong | other |
|---|---|---|---|---|---|---|---|---|---|---|
| all | 2201 | 0.0% | 0.4% | 7.0% | 8.5% | 31.9% | 50.9% | 0.0% | 1.3% | 0.0% |
| minisweagent/qwen38_27b | 623 | 0.0% | 0.0% | 2.4% | 2.6% | 28.9% | 64.0% | 0.0% | 2.1% | 0.0% |
| sweagent/qwen36_27b | 280 | 0.0% | 0.0% | 5.0% | 3.9% | 37.9% | 52.1% | 0.0% | 1.1% | 0.0% |
| sweagent/qwen35_122b | 27 | 0.0% | 0.0% | 22.2% | 11.1% | 18.5% | 48.1% | 0.0% | 0.0% | 0.0% |
| minisweagent/qwen36_27b | 834 | 0.0% | 0.7% | 1.9% | 16.1% | 33.6% | 47.0% | 0.0% | 0.7% | 0.0% |
| openhands/minimax_m25 | 129 | 0.0% | 2.3% | 5.4% | 6.2% | 41.9% | 43.4% | 0.0% | 0.8% | 0.0% |
| openhands/qwen35_122b | 139 | 0.0% | 0.0% | 61.9% | 1.4% | 12.2% | 23.7% | 0.7% | 0.0% | 0.0% |
| sweagent/minimax_m25 | 169 | 0.0% | 0.0% | 5.3% | 7.1% | 35.5% | 48.5% | 0.0% | 3.6% | 0.0% |

## Overconfidence test (SMALL)

- lazy arm (no repro/run command before first edit AND no test after last edit):
  P(fail) = 0.427 [0.378,0.473]
  (instance-level; n_groups with both arms=329, n_instances=320)
- diligent arm (repro before edit AND test after last edit):
  P(fail) = 0.520 [0.466,0.573]
- paired difference (lazy − diligent): -0.094 [-0.183,-0.005]
- stricter variant (no repro AND zero test commands vs repro AND ≥1 test):
  Δ=-0.030 [-0.235,+0.167]
  (n_groups=67)
- pooled: fail rate lazy=0.499 (n=6281),
  diligent=0.487 (n=4909)
- share of fail rollouts that were lazy: 0.303;
  share of pass rollouts that were lazy: 0.301

Did the failing sibling stop earlier? (turns; assistant_chars is the token proxy — the
corpus has no wall-clock or token counts)

- SMALL: Δturns=+3.81 [+2.78,+4.89],
  Δchars=+529; share of groups where the fail used fewer turns: 0.472
- LARGE: Δturns=-2.45 [-4.77,-0.14],
  Δchars=+35; share of groups where the fail used fewer turns: 0.517

## Semantic labels (OpenRouter free tier, ≤ 600 requests)

n=293 labeled SMALL fail trajectories (model(s): cohere/north-mini-code:free, deepseek/deepseek-v4-flash-0731:free, dots-studio/dots-3-note-preview:free, inclusionai/ling-3.0-flash-fin:free, inclusionai/ling-3.0-flash-sante:free, inclusionai/ling-3.0-flash-vl:free, nex-agi/nex-n2.5-mini:free, nex-agi/nex-n2.5-pro:free, nvidia/nemotron-3-nano-omni-30b-a3b-reasoning:free, nvidia/nemotron-3-super-120b-a12b:free, nvidia/nemotron-3-ultra-550b-a55b:free, nvidia/nemotron-3.5-content-safety:free, nvidia/nemotron-3.5-lightning:free, poolside/laguna-s-2.1:free, qwen/qwen3.8-27b:free)

| llm label | share |
|---|---|
| wrong_root_cause | 24.9% |
| incomplete_stopped_early | 21.8% |
| fixed_symptom_not_cause | 20.8% |
| missed_second_site | 11.6% |
| other | 11.6% |
| misread_issue | 5.1% |
| broke_other_test | 3.8% |
| environment | 0.3% |

LLM label × rule taxonomy (row %):

| rule bucket | broke_other_test | environment | fixed_symptom_not_cause | incomplete_stopped_early | misread_issue | missed_second_site | other | wrong_root_cause |
|---|---|---|---|---|---|---|---|---|
| complete_but_wrong | 4% | 0% | 19% | 22% | 6% | 10% | 19% | 18% |
| over_edit | 8% | 0% | 31% | 8% | 0% | 8% | 8% | 38% |
| partial | 3% | 0% | 27% | 30% | 1% | 6% | 8% | 25% |
| partial_extra | 8% | 0% | 12% | 12% | 4% | 17% | 12% | 33% |
| test_edit | 3% | 0% | 22% | 32% | 3% | 8% | 14% | 19% |
| wrong_site | 0% | 2% | 18% | 14% | 14% | 20% | 6% | 27% |

## Verdicts

- **H1 (SMALL fails = overconfident quickies).** SMALL fail taxonomy: partial+wrong_site+partial_extra=61.6%, timeout=0.0%. Failing siblings ran repro before edit 0.5 pts more often than passing ones, and ran longer (Δturns=+3.8). Lazy-vs-diligent fail-rate gap: -0.094. MIXED/WEAK.
- **H2 (LARGE fails = capacity).** LARGE fail taxonomy: timeout=0.0%, partial=31.9%. Δturns=-2.4, Δcoverage=-0.066. MIXED/WEAK.

## Reproduce

```
uv run python scripts/paired_eligible.py     # outputs/eligible_pairs.parquet
uv run python scripts/paired_features.py     # outputs/paired_features.parquet (resume-safe)
uv run python scripts/paired_analysis.py     # this note + outputs/paired_*.csv
uv run python scripts/paired_labels.py       # outputs/llm_labels.jsonl (OpenRouter free)
```

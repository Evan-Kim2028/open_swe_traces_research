# Does the certified-hard failure anatomy reproduce on Open-SWE-Traces?

Generated 2026-09-20 20:09 UTC by `uv run python scripts/failure_anatomy.py`.

## Verdict

The stopping rule appears on failed rollouts: 97.4% (141341/145092; 95% CI 97.3–97.5) of classified last-test observations are green and 0.2% (448/184670; 95% CI 0.2–0.3) express uncertainty (original: visible suite green on 44/45 units, 0/61 uncertain; asserts-done is 68.7% (126800/184670; 95% CI 68.5–68.9) but is harness finish-style, 26-99% by combo). Unjustified confidence is not specific to failures: resolved rollouts match (last-test green 97.7% (123550/126431; 95% CI 97.6–97.8), uncertainty 0.1% (175/149867; 95% CI 0.1–0.1); Cohen's h -0.02 and +0.03), and the original L0 bank had no successes to compare against. Manufactured all-clear by editing tests is 7.5% (13873/184670; 95% CI 7.4–7.6) of failures vs 1.8% (2628/149867; 95% CI 1.7–1.8) of successes (h=+0.29), concentrated in openhands/qwen35_122b, not a common mode. The 78% hidden-suite near-miss cannot be measured (no agent per-test results); median FAIL_TO_PASS length is 4 vs original 8. On a stratified sample of 40 all_fail instances, C (decisive clause not in the issue and not scored as repo-inferable) is 6/40 vs original 76%; anti-inferable is 1/40 vs original 29%.

## What is measured vs inferred

Measured: last-two-assistant text flags, last test-command observation in a 30-message tail, model-patch test paths, FAIL_TO_PASS list sizes from Scale-SWE and SWE-rebench-V2, file Jaccard of model vs gold patches.

Inferred: that a green in-tree run caused the stop (we see the green run and then a submit; we do not see a counterfactual where the agent would have kept going); that a missed clause lived only in a withheld contract (Q3/Q4, n=40 manual reads).

The original finding is on 45 synthetic Go units built so that L0 always fails and L2 almost always passes. Open-SWE-Traces tasks are naturally occurring GitHub PRs at one fixed affordance level. A negative on transfer is a valid result.

## Method

Labeled trajectories only (`resolved IN (0, 1)`). Unknown (`resolved = -1`) rows are dropped; they cannot test a stopping rule against an outcome.

Per shard, DuckDB keeps `list_last` / second-last assistant messages (content + reasoning + tool-call args, lowercased) and a 30-message tail. Regexes live in `openswe_traces.failure_anatomy` and are unit-tested. The corpus is never loaded as trajectories.

Last-test pairing: last tool call in the tail whose arguments match a repo test command (`pytest`, `go test`, `cargo test`, `npm test`, `jest`, …); the next `role=tool` observation is the result. Green/red prefers an explicit exit code when both signals fire. `pytest | tail` without `pipefail` can hide a red exit; those rows stay in the classified-green count if the observation still contains a pass summary, and are a known false-green risk.

Test-path edits: paths from `diff --git a/X b/X` matching `tests/`, `_test.go`, `test_*.py`, `*.test.ts`, and siblings. Created tests: `--- /dev/null` plus a test path.

FAIL_TO_PASS lists come from `traces_external/` (Scale-SWE `fail_to_pass`, SWE-rebench-V2 `FAIL_TO_PASS`). Per-test results of the *agent* patch are not in Open-SWE-Traces.

n_fail = **184,670**; n_resolved = **149,867**; labeled trajectories = **334,537**; distinct instances = **39,702**.

## Q1 — stopping rule

Original (61 L0-fail trials): 46/61 (75%) final message claims tests pass; 0/61 flags uncertainty; 0/61 names unseen tests; 44/45 units stopped because the visible suite went green.

| measure | failed (resolved=0) | resolved=1 | Δ fail−resolved |
|---|---|---|---|
| asserts tests pass or the fix is complete | 68.7% (126800/184670; 95% CI 68.5–68.9) | 69.4% (103938/149867; 95% CI 69.1–69.6) | -0.7 pp (h=-0.01) |
| asserts tests/reproduction pass | 61.1% (112751/184670; 95% CI 60.8–61.3) | 62.4% (93464/149867; 95% CI 62.1–62.6) | -1.3 pp (h=-0.03) |
| asserts the fix is complete | 44.8% (82660/184670; 95% CI 44.5–45.0) | 40.6% (60799/149867; 95% CI 40.3–40.8) | +4.2 pp (h=+0.08) |
| expresses any uncertainty | 0.2% (448/184670; 95% CI 0.2–0.3) | 0.1% (175/149867; 95% CI 0.1–0.1) | +0.1 pp (h=+0.03) |
| mentions 'hidden tests' (SWE-bench-aware) | 1.3% (2421/184670; 95% CI 1.3–1.4) | 1.1% (1699/149867; 95% CI 1.1–1.2) | +0.2 pp (h=+0.02) |
| names unverified / withheld tests as a gap | 0.0% (5/184670; 95% CI 0.0–0.0) | 0.0% (3/149867; 95% CI 0.0–0.0) | +0.0 pp (h=+0.00) |
| ran a repo test command in the last 30 messages | 83.6% (154345/184670; 95% CI 83.4–83.7) | 89.2% (133725/149867; 95% CI 89.1–89.4) | -5.7 pp (h=-0.17) |
| model patch touches a test path | 7.5% (13873/184670; 95% CI 7.4–7.6) | 1.8% (2628/149867; 95% CI 1.7–1.8) | +5.8 pp (h=+0.29) |
| model patch creates a new test file | 3.7% (6886/184670; 95% CI 3.6–3.8) | 0.5% (703/149867; 95% CI 0.4–0.5) | +3.3 pp (h=+0.25) |
| empty model patch | 0.0% (0/184670; 95% CI 0.0–0.0) | 0.0% (0/149867; 95% CI 0.0–0.0) | +0.0 pp (h=+0.00) |
| last visible test command was green (given a classified observation) | 97.4% (141341/145092; 95% CI 97.3–97.5) | 97.7% (123550/126431; 95% CI 97.6–97.8) | -0.3 pp (h=-0.02) |
| last visible test command was green (all labeled rollouts) | 76.5% (141341/184670; 95% CI 76.3–76.7) | 82.4% (123550/149867; 95% CI 82.2–82.6) | -5.9 pp (h=-0.15) |
| stopping-rule conjunction (asserts done AND last test green) | 56.6% (104452/184670; 95% CI 56.3–56.8) | 59.6% (89289/149867; 95% CI 59.3–59.8) | -3.0 pp (h=-0.06) |

Cohen's h: 0.20 is a small effect, 0.50 medium, 0.80 large (Cohen 1988). The original claim is that unjustified confidence is *specific to failures*. The comparison column is that claim's test: a large positive Δ on confidence, or a large negative Δ on uncertainty, would support it. A near-zero Δ means the stopping ritual is shared and only the outcome label makes the confidence unjustified.

### By harness / teacher

| harness | teacher | n_fail | n_ok | fail asserts_done | ok asserts_done | fail last-test green (classified) | fail uncertainty | fail test-path edit |
|---|---|---:|---:|---|---|---|---|---|
| minisweagent | qwen36_27b | 53622 | 32231 | 37.5% (20107/53622; 95% CI 37.1–37.9) | 37.8% (12180/32231; 95% CI 37.3–38.3) | 97.5% (37293/38237; 95% CI 97.4–97.7) | 0.1% (30/53622; 95% CI 0.0–0.1) | 2.6% (1378/53622; 95% CI 2.4–2.7) |
| minisweagent | qwen38_27b | 46670 | 49925 | 91.3% (42613/46670; 95% CI 91.0–91.6) | 90.8% (45321/49925; 95% CI 90.5–91.0) | 98.8% (39358/39837; 95% CI 98.7–98.9) | 0.6% (267/46670; 95% CI 0.5–0.6) | 1.0% (464/46670; 95% CI 0.9–1.1) |
| openhands | minimax_m25 | 19510 | 14363 | 99.5% (19411/19510; 95% CI 99.4–99.6) | 99.6% (14311/14363; 95% CI 99.5–99.7) | 97.1% (16240/16721; 95% CI 96.9–97.4) | 0.4% (85/19510; 95% CI 0.4–0.5) | 3.6% (706/19510; 95% CI 3.4–3.9) |
| openhands | qwen35_122b | 21124 | 9833 | 99.9% (21104/21124; 95% CI 99.9–99.9) | 100.0% (9832/9833; 95% CI 99.9–100.0) | 99.4% (17827/17939; 95% CI 99.2–99.5) | 0.2% (43/21124; 95% CI 0.2–0.3) | 43.6% (9212/21124; 95% CI 42.9–44.3) |
| sweagent | minimax_m25 | 19206 | 16749 | 87.5% (16801/19206; 95% CI 87.0–87.9) | 86.7% (14515/16749; 95% CI 86.1–87.2) | 93.4% (13221/14162; 95% CI 92.9–93.8) | 0.1% (19/19206; 95% CI 0.1–0.2) | 3.4% (647/19206; 95% CI 3.1–3.6) |
| sweagent | qwen35_122b | 7750 | 7348 | 31.4% (2434/7750; 95% CI 30.4–32.4) | 33.7% (2477/7348; 95% CI 32.6–34.8) | 93.5% (4583/4900; 95% CI 92.8–94.2) | 0.1% (4/7750; 95% CI 0.0–0.1) | 10.7% (830/7750; 95% CI 10.0–11.4) |
| sweagent | qwen36_27b | 16788 | 19418 | 25.8% (4330/16788; 95% CI 25.1–26.5) | 27.3% (5302/19418; 95% CI 26.7–27.9) | 96.4% (12819/13296; 95% CI 96.1–96.7) | 0.0% (0/16788; 95% CI 0.0–0.0) | 3.8% (636/16788; 95% CI 3.5–4.1) |

`openhands/deepseek_v4_flash` and `openhands/qwen36_27b` contribute no labeled rows (all `resolved = -1`).

`asserts_done` tracks finish-message style, not a shared cognitive state. OpenHands almost always emits a "I have successfully implemented" finish tool call (99%+ on both outcomes). SWE-agent Qwen often submits with an empty last turn (26–31% match). Last-test-green does not have that problem: 93–99% of classified last-test observations are green on *failed* rollouts in every labeled combo. That is the number to compare to the original 44/45 visible-suite-green stops.

Test-path edits concentrate in `openhands/qwen35_122b` (43.6% of that combo's failures, 10.9% of its successes). Drop that teacher and the corpus-wide fail rate is about 3%, still above resolved but not a common mode. `empty_model_patch` is 0% because the patch *string* is never empty; `model_files = 0` still occurs (a few hundred rows).

## Q2 — near miss or structurally different?

Original: failing L0 patch already passes a median 78% of the hidden suite; median 2 failing tests of 8; 80% of patches are structurally gold with one clause wrong.

**Per-test results of the agent patch are not in this corpus.** `resolved` is a single bit. FAIL_TO_PASS is the *gold* hidden-suite membership list, not a run of the model patch. The 78% figure cannot be reproduced here. Closest available proxies follow.

FAIL_TO_PASS coverage: 31,437 / 31,437 failed instances join a FAIL_TO_PASS list (100.0%). Failed rollouts with a join: 184,670.

Hidden-suite *size* (FAIL_TO_PASS length) on failed rollouts with a join:

| | value |
|---|---|
| n rollouts | 184,670 |
| median | 4 |
| p25 | 2 |
| p75 | 14 |
| p90 | 91 |
| mean | 109.5 |

| FAIL_TO_PASS count | failed rollouts | share of failed-with-ftp |
|---|---:|---:|
| 1 | 40,067 | 21.7% |
| 2–4 | 62,310 | 33.7% |
| 5–8 | 25,990 | 14.1% |
| 9+ | 56,303 | 30.5% |

The original units had a median hidden suite of 8 tests, so "fail 2 of 8" is a minority. On this corpus the median FAIL_TO_PASS length is 4. When the hidden suite is one test, a failed rollout fails 100% of required tests by construction; the "already passing 78%" shape is not even well-defined.

Structural proxy (file Jaccard of model patch vs gold), measured:

| file Jaccard (model vs gold) | failed | resolved |
|---|---:|---:|
| n | 184,670 | 149,867 |
| median | 0.400 | 0.500 |
| p25 | 0.200 | 0.333 |
| p75 | 0.615 | 1.000 |
| share = 0 (no shared files, including empty patches) | 5.2% | 1.4% |
| share ≥ 0.5 | 45.6% | 68.1% |
| share = 1 (same file set) | 16.3% | 30.1% |

Failed rollouts with a non-empty patch and file Jaccard ≥ 0.5: **84,280 / 184,670** (45.6%). That is "touched at least half the gold files", not "one clause wrong". Empty failed patches (string length 0): 0.0% (0/184670; 95% CI 0.0–0.0). The FAIL_TO_PASS mean (109.5) is not usable: SWE-rebench lists run to 10^4–10^5 names on some instances; use the median and the bins.

## Q3 — was the decisive information in the issue text?

Original: ~76% C (clause only in the L2 contract / hidden test), ~18% B (inferable from the repo), ~7% A (in the bug report).

Stratified sample n=40, scored=40. A=26, B=1, C=6, mixed=7. C share among scored = 15% (original 76%). Go subset: C=2/16. Mixed means the issue states the main behaviour and gold also lands a load-bearing extra; those are not C. B and anti-inferable are lower bounds: no repo checkout. Labels: four disjoint readers (10 instances each), then one audit pass on every C and mixed row (two recodes: snowflake C to A, autoprefixer C to B; one anti kept on sushi, De Bruijn exact-match candidate not counted as original-sense anti). C+mixed = 13/40 if extras count as withheld-clause-ish.

| instance_id | lang | A/B/C | note |
|---|---|---|---|
| `tikv__client-go-1181` | go | A | Issue gives expected vs actual replica access paths and says not to retry an already-tried store. |
| `go-delve__delve-3655` | go | mixed | Issue lists unix: listen prefix (gold's choice) but gold also adds unrelated DAP waitFor attach. |
| `moov-io__customers-221` | go | mixed | Issue asks for one-primary address validation; gold also drops customerIDs search and rewrites GetCustomer. |
| `kubernetes__kops-9052` | go | mixed | Issue names treating NatGatewayNotFound as already-deleted, but gold also lands GCE SA, Flatcar, and validation rewrites. |
| `jaegertracing__jaeger-6608` | go | A | Issue names a dummy /quality-metrics endpoint with obvious ids like sample-service-A (dummy vs empty). |
| `open-telemetry__opentelemetry-go-contrib-5404` | go | A | Issue states a baggage span processor that copies baggage items onto spans as attributes. |
| `kubernetes__kops-15916` | go | C | Issue wants a lower default CPU and a topology key fix; gold adds optional resource fields keeping 500m and skips topology. |
| `digitalocean__doctl-1273` | go | A | Issue specifies --wait, poll until status online, and retain the initial create connection/password. |
| `nginx__nginx-gateway-fabric-3390` | go | A | Issue names GetFileStream and sending file chunks when over the size limit. |
| `vaskoz__dailycodingproblem-go-582` | go | C | Issue defines De Bruijn and gives one example sequence; gold's FKM output is unstated. A hidden exact-match test would reject other valid sequences. |
| `skupperproject__skupper-1819` | go | mixed | Issue asks to validate linkAccess and error-log invalid values; gold also renames config-sync to kube-adaptor across the tree. |
| `chanzuckerberg__terraform-provider-snowflake-624` | go | A | Issue asks for RAP resource and links the Snowflake SQL spec gold implements; grant companion is incidental. |
| `kubernetes__kops-15756` | go | A | Issue shows the missing dashed Hubble SAN and the cert dnsNames gold emits via replace dots with dashes. |
| `knative__client-990` | go | A | Issue asks for --scale-init or --scale-initial mapping to initialScale; gold picks --scale-init. |
| `open-telemetry__opentelemetry-go-contrib-6074` | go | A | Issue is superfluous WriteHeader on Flush/SSE with expected no log; gold's Flush guard implements that. |
| `mgechev__revive-986` | go | A | Issue says //nolint:gochecknoglobals must not trip comment-spacings; gold exempts directive comments. |
| `maxb2_typer-config_pr17` | python | A | Issue specifies ini_loader, configparser, sectioned dict return, and subpath_loader usage. |
| `jsh9_pydoclint_pr155` | python | A | Issue says .py-named dirs must not be opened as files (skip or traverse); gold skips non-files. |
| `optimizely_python-sdk_pr92` | python | A | Issue names $opt_bucketing_id, the bucketing vs user_id split, and fallback to user_id. |
| `frictionlessdata_frictionless-py_pr548` | python | A | Issue requires lowercase slugified resource names and gives the exact table-with-data example. |
| `app-sre_qontract-reconcile_pr641` | python | A | Issue names static integration-name CloudWatch streams in Helm and OpenShift; gold sets log_stream_name to those names. |
| `openedx_edx-django-utils_pr357` | python | A | Issue asks get_plugin_apps to log at INFO with project_type; gold does that (WARN: in the string is extra wording). |
| `pycqa__pyflakes-451` | python | mixed | README absolute NEWS.rst URL is in the issue; gold also ships checker parent/depth rename and other 2.1.1 bugfixes. |
| `duplocloud_duploctl_pr56` | python | A | Issue says use the single-service endpoint and preserve AllocationTags; gold posts ReplicationControllerChange with tags from find(). |
| `dbt-labs__dbt-bigquery-892` | python | C | Issue only wants pre-1.6 is_replaceable; gold's decisive case-insensitive field/granularity match is never named. |
| `omni-us_jsonargparse_pr264` | python | A | Issue shows the exact indented Union/subclass error block; gold's indent_text and wording match that expected output. |
| `python-openapi_openapi-core_pr462` | python | A | Issue specifies rename context→errors, init with errors, and a deprecating context property; gold implements that API. |
| `mercedes-benz_odxtools_pr137` | python | A | Issue specifies ODXLINK then inheritance then SNREF, Database.refresh(), and a consistent resolve interface; gold does that. |
| `arco-design__arco-design-686` | ts | A | Issue asks Select-like triggerElement to expose current option as a render fn; gold adds that callback (plus value). |
| `fhir__sushi-770` | typescript | C | Issue says the slicing error should probably not exist; gold still errors via a new closed-slicing check and message. |
| `lo1tuma__eslint-plugin-mocha-260` | typescript | A | Issue asks to ignore skipped tests via an option or by default; gold adds ignoreSkipped (default false). |
| `postcss__autoprefixer-1412` | javascript | B | Issue asks to prefix ::file-selector-button; webkit alias ::-webkit-file-upload-button is the sibling-hack pattern in lib/hacks, not stated in the issue. |
| `playcanvas__engine-6648` | javascript | C | Issue wants enable-order (body before constraint) but never names static order, default 0, or rigidbody -1. |
| `php-cs-fixer__php-cs-fixer-7663` | php | A | Issue shows braceless while/if de-indented and says statement_indentation should work without braces. |
| `dprint__dprint-plugin-typescript-414` | rust | A | Issue gives expected vs actual: operatorPosition=maintain must keep ?/: at end of line in conditional types. |
| `primefaces__primefaces-11131` | java | mixed | Issue states prefers-reduced-motion; gold also rewrites table filters, facets, and file-upload path decoding. |
| `sindresorhus__got-297` | typescript | mixed | Issue specifies json:true stringify plus Content-Type; gold also adds form, protocol errors, and drops auto-urlencoded objects. |
| `intuit__auto-888` | ts | C | Issue only reports git-describe failure with empty expected; gold invents 0.0.0 and HEAD^ fallbacks. |
| `electron__electron-packager-1053` | js | A | Issue asks for official Windows ARM64; gold adds win32/arm64 (6.0.8 gate is incidental). |
| `friendsofphp__php-cs-fixer-5835` | php | A | Issue shows phpdoc_types_order turning array{...} into array<mixed>/} and that must not happen. |

## Q4 — anti-inferable category

Original: 13/45 units (29%) where the repo pointed at the rejected answer.

Anti-inferable among scored Q3 instances: **1/40** (2%; 95% CI 0–13%). Original 13/45 = 29%.

| instance_id | evidence |
|---|---|
| `fhir__sushi-770` | Issue argues for no error; gold invents ValueConflictsWithClosedSlicingError. Repo/user report points away from gold. |

## Reproduce

```bash
uv run python scripts/failure_anatomy.py
uv run pytest tests/test_failure_anatomy.py
```

Resume a partial stream with the same command (existing parts are skipped). Force a recompute with `--force`. Tables only, no stream: `--skip-stream`. Q3 texts: written to `outputs/failure_anatomy_q3_sample.json`. Labels: `outputs/failure_anatomy_q3_labels.json`.

Regexes: `TESTS_PASS_RE`, `FIX_COMPLETE_RE`, `UNCERTAIN_RE`, `UNSEEN_GAP_RE`, `TEST_CMD_RE`, `GREEN_RE`, `RED_RE` in `src/openswe_traces/failure_anatomy.py`.

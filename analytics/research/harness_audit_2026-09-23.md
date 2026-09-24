# Harness audit, 2026-09-23

Question: which verdicts measured the model, and which measured the harness? Prompted by
Devin "failing helm-dlmanager at every level": all six verdicts were `[setup failed]`.

Reproduce: `uv run python -m openswe_traces.analysis.harness_audit` (from the data repo root),
`uv run python -m openswe_traces.authoring.repair_imports TASK... --verify`.

## Every trial, by what the grader printed (1,801 trials, all models)

| class | trials |
|---|---:|
| tests ran and failed | 796 |
| pass | 759 |
| pass, agent also edited a test file | 55 |
| no verdict, harness error (agent exit, API limits, cancelled) | 112 |
| **void: hidden test imports a module path the repository does not declare** | 65 |
| **void: renamed tree invalid (fix already present / answer key does not apply)** | 4 (+1 audited) |
| **void: provider-side WebFetch of the file under test** | 1 |
| agent code does not build | 4 |
| package panic or exit | 3 |
| timeout | 2 |
| vacuous pass (no package ran a test) | 0 |
| zero with no failing test | 0 |

## The bug: the renaming pass

A pass run after validation to disguise the source repositories rewrote module paths
(goa -> apikit/v3, helm -> chartkit/v4, kops -> clustkit) and user-facing strings
("helm repo add" -> "chartkit repo add") in place, in every staged copy, and never
re-validated. Nothing rewrote the checksum-locked hidden tests or the answer keys.

- 65 zero verdicts on 13 tasks could not compile their tests (`ledger.unmeasured`). Every
  task that "failed at every level" (exprhash, httpencoding, httpmux for Composer and
  Grok; helm-dlmanager for Devin) was one of these.
- Re-validation in Docker of all 69 tasks with trials on a renamed tree: 62 valid, 5
  invalid (`ledger.INVALID_RENAMED_TREE`), 2 unverifiable pilot tasks whose renamed-tree
  verdicts were already void.
  - helm-searchindex, helm-tlsutil: the unfixed renamed tree passes its own hidden tests.
  - helm-dlmanager: the gold fix is already in the renamed tree.
  - helm-httpgetter: the renamed tree mixes old and new module paths.
  - httpencoding: the answer key does not apply to the renamed tree.
- On invalid trees, passes came from working around the harness, not the bug: Devin and
  Grok built a `goa` compatibility module at L6 (httpencoding, httpmux, helm-searchindex);
  Devin renamed the whole helm-tlsutil module back to `helm`. Voided where the tree is invalid.
  httpmux's tree is valid, so Grok's L6 pass there stands.

Verdicts from the original, pre-rename trees (validated at authoring) stand, e.g. Composer's
helm-dlmanager L2 3/3.

## Contamination and isolation

- Web tools: one call in 122.8k tool calls (Composer 104.6k, Devin 16.3k, Grok 2.0k):
  Composer's WebFetch of the upstream resolver.go on helm-depresolver-L3, which then passed.
  Documented on 2026-09-19 as void but never applied to the ledger; now `AUDITED_VOID`.
- Test edits in passing runs: 55, 49 in the same package as a hidden test. None adds
  `init` or `TestMain`; Go forbids redeclaring the code under test.
- Staged repositories (1,136): no `.git`, no gold or cheat patch. One copy
  (sweep_L2/helm-depresolver-L2) ships the hidden test inside the repository at L2; its
  three runs all failed.
- Zero-token verdicts (deleted base images, 2026-09-19 incident): none in the ledger.

## Effect on the write-up (paper_numbers, probe scope)

| | before | after |
|---|---:|---:|
| graded | 412 | 409 |
| certified | 242 | 240 (197 / 10 / 33) |
| no verdict | 20 | 23 |
| certified by two models | 40 | 36 |
| exhausted (fails L1-L6) | exprhash, httpencoding, httpmux, helm-dlmanager | none |

## Rerun

Voided cells with no valid verdict left: 55 (Grok 15, not rerunnable). Staged, import-repaired
and Docker-validated in `sweep_fix_{composer,devin}_L*`: 17 units valid (exprhash L0-L6,
httpmux L0-L6, goa-evalctx L0, kops-difftext L3/L4, kops-taintparse L3/L4). The helm tasks
and httpencoding need their excision re-authored against the renamed tree.

The gate (trial_guard / solver_match in the main checkout) reads the old ledger and still
counts the void zeros as decided cells, so the reruns wait on this branch reaching it.

## Rule

Any transformation of staged tasks is followed by re-validation of every copy it touched:
buggy fails, gold passes, cheat rejected. Reward alone cannot tell a harness failure from a
model failure.

## Rerun results (2026-09-23, afternoon)

Repaired copies in `sweep_fix_*`, each cohort followed by `harness_audit --job` (every check
clean apart from Devin rate-limit errors, which are no verdict and were rerun or dropped).

| task | Composer L3 | L4 | L5 | L6 | Devin |
|---|---|---|---|---|---|
| exprhash | fail | fail | **pass** | **pass** | L5 **pass** (single probe, no L1 on record) |
| httpmux | fail | fail | **pass** | **pass** | L1 fail, L3 fail |
| goa-evalctx | | | | | L1 pass |

Every pass edits only the file under test, except Devin's exprhash L5, which also reverts
rename damage elsewhere in the tree (`apikit.design/...` import paths, `apikit.ServiceError`,
`apikit-attribute` headers in codegen and `expr/http_response.go`). The graded `TestDetail`
hash tests do not reach that code, and Composer passed them with `hasher.go` alone.

Leftover rename damage: the goa (`apikit/v3`) trees still carry renamed strings outside
the tested packages, so unrelated tests are red for agents (the goa-evalctx agent flagged an
`expr.init()` panic). Grading is unaffected (gold passes, buggy fails, cheat rejected), but
the next goa cohort should get those strings reverted before it is authored.

Devin rate limit: four concurrent Devin trials trip "Reached free model rate limit" and
it takes about 30 minutes to clear; 13 Devin trials died on it today. Two concurrent ran
cleanly. The remaining Devin cells (exprhash L1-L4/L6, httpmux L2/L4-L6) were left unrun.

Gate fixes made along the way (PRs #3-#7): harbor no longer inherits the admission lock;
one lock per solver; Devin occupancy counts only Devin containers; the in-flight check is
per solver; `STACK_LADDER=1` measures a solver's upper rungs beside its own lower screen.

# Gate consolidation — one registry, evidence tiers (2026-09-19)

**Question.** Three engines wrote `rule_verdicts` into the same `validation.json`
with no precedence: a verifier-stage documentary pass could overwrite executed
in-image evidence. Can the engines be unified behind one registry with explicit
tiers, and do the four newly-cheap checks (A5 determinism, A10 runtime no-network,
B7 excision leakage, level nesting) catch real failures?

**Config / command.**

```bash
uv run python scripts/gate_tasks.py experiments/pipeline/tasks_composerver \
    --out outputs/gate_matrix_real.json          # 60 dirs: helm/kops/gin L0+L2
uv run python scripts/gate_tasks.py \
    /home/evan/Documents/oswt-closureI/experiments/dose_response/tasks \
    --out outputs/gate_matrix_fab.json           # 150 fabricated dirs
uv run python scripts/gate_backfill_a2.py experiments/pipeline/tasks_composerver \
    <fab tasks> --db experiments/pipeline/state.db --jobs <dose_response jobs>
```

Log: `outputs/closure_O.log`. No solver trials, no Harbor run, no commit.

## Architecture

`src/openswe_traces/gate/` is the single registry. Each rule registers once per
implementation with `tier ∈ {executed, static, documentary}` and a
`run(ctx) -> Verdict(rule_id, passed|skipped, tier, evidence, produced_at,
provenance)`. Merge: **executed > static > documentary**; a lower tier never
overwrites a higher one for the same rule — superseded rows persist under
`rule_verdicts_secondary` for audit. Within a tier, newest `produced_at` wins.

All write paths go through `gate.runner.write_task_validation`:
`synth/rules.py` and `pipeline/preflight.py` are thin adapters that register
impls and delegate; the old `attach_rule_verdicts` overwrite path is deleted.
Launch paths (`solve-unit`, `solve-watch`, `run_pipeline`) call
`gate.ensure_gate`: refuse when any verdict fails or any rule with an executed
impl lacks an executed verdict.

## Registry

| rule | tier | impl | provenance | runs |
|---|---|---|---|---|
| A1 | executed | yes | gate/exec | gold patch passes in-image, x2 |
| A1 | documentary | yes | synth.rules | notes/verifier records |
| A2 | executed | yes | gate/alt_fix | passed trial patch differs from gold |
| A2 | documentary | yes | synth.rules | alt.patch / alt-accept proof in artifacts |
| A3 | executed | yes | gate/exec | cheat patch fails in-image |
| A3 | documentary | yes | synth.rules | notes |
| A4 | documentary | yes | synth.rules | impact-set vs f2p notes |
| A5 | executed | yes | gate/exec | suite outcome stable across 2 runs/seeds |
| A5 | documentary | yes | synth.rules | notes |
| A6 | documentary | yes | synth.rules | multi-site single-fix notes |
| A7 | documentary | yes | synth.rules | decoy provenance notes |
| A8 | executed | yes | gate/exec | bare excised tree fails in-image, x2 |
| A8 | documentary | yes | synth.rules | notes |
| A9 | documentary | yes | synth.rules | dynamic-gate proven non-passer notes |
| A10 | executed | yes | gate/exec | gold passes under `docker --network=none` |
| A10 | static | yes | synth.rules | no-network declared in config |
| A11 | documentary | yes | synth.rules | perf-gate idle-host proof notes |
| A12 | static | yes | synth.rules | gold/alt/cheat patches avoid guarded test files |
| B1 | static | yes | synth.rules | test checksum guard in test.sh |
| B2 | static | yes | synth.rules | egress denied + no-web clause |
| B3 | static | yes | synth.rules | hidden tests not shipped in tree |
| B4 | static | yes | synth.rules | hidden tests black-box (excision-aware) |
| B5 | static | yes | synth.rules | property/seeded verifier shape |
| B6 | static | yes | synth.rules | instruction floor: symptom/repro |
| B7 | static | yes | synth.rules | no excised symbol/file/line leaks in instruction |
| B8 | static | yes | synth.rules | no-web clause present |
| C5 | documentary | yes | synth.rules | gold-failing proof = harness failure |
| coverage | static | yes | gate/static | every hidden test declared in contract |
| nesting | static | yes | gate/static | level info monotone; hidden tests identical |

30 registrations over 23 rule IDs. Executed impls: A1, A2, A3, A5, A8, A10.

## Executed suite

`gate/exec_rules.py` runs five in-image suites per task dir, all classified
`pass|fail|infra` by `preflight.INFRA_SIGNATURES` (unchanged): bare ×2, gold ×2
(second run under `HIDDEN_SEED`/`SEED` override when the suite honours one),
gold under `--network=none`, cheat ×1. Results memoized per gate context and
cached in `validation.json` keyed by `(image id, tests sha, tree sha)` —
unchanged packages re-check free.

## A2 backfill results

`backfill_state_db` + `backfill_jobs` over `experiments/pipeline/state.db`
(12 passed, non-excluded trials) and `experiments/dose_response/jobs/`:

- **5 executed A2 verdicts recorded** on `bindingdispatch-L0`/`bindingdispatch-L2`
  (gin) — alternative correct fixes accepted, jaccard 0.02–1.0 vs gold (same
  symbol set, different bodies).
- 7 trials skipped: `connarray-cv` was never packaged into `tasks_composerver`
  (4 trials), `connarray` job dirs no longer on disk (2 trials).
- dose_response jobs: 1 job, its trial failed (reward 0) — no evidence to record.

## Per-task matrix

See `outputs/gate_matrix_real.json` and `outputs/gate_matrix_fab.json` for the
full per-task verdict matrices (mark = pass/skip/FAIL + tier letter).

## New failures caught by the four added checks

<!-- filled in after sweep -->

## Kill-count update for verifier_rules.md §F

<!-- filled in after sweep -->

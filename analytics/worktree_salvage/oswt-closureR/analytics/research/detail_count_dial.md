# Detail-count × verification-affordance dial (closure-R)

2026-09-19. Pre-registered design, generator change, and 40-unit pilot grid.
Replaces the refuted internal-edge-density dial
(`overnight_plan_closure_dose_response.md`, last section): pass rate is
predicted to fall **geometrically with the number of independent behavioural
details the solver must get right**, and **verification affordance** (can the
solver cheaply check each detail against given evidence?) shifts the whole
curve.

## The hypothesis, made testable

**H.**  Let a unit commit to `d` independent behavioural details.  If each
detail is independently wrong with probability `1 − p` (a per-detail pass
rate), then `P(unit passes) = p^d` — a geometric curve in `d`.  Verification
affordance `v` changes `p` (v=high raises it: wrong choices become visible
without the hidden suite), so the curve keeps its geometric shape but sits
higher.

**Contrast with the refuted knob.**  The internal-edge dial varied the shape
of the excised call graph at fixed |S| and produced low 2/3 vs high 2/3
(n=6).  Audited to the line, both failures were single behavioural details;
the high-ratio failure differed from the passing patches by one line (bucket
reduction `(h>>61)&7` vs `h % 8`) and contradicted two of the three worked
examples in its own bug report.  That is the natural-task signature
(`paired_failures_small_vs_large.md`: failing siblings do MORE work without
verifying against the evidence).  The new knob makes the *count of
independently-wrongable commitments* the dial, and the *cheapness of checking
each one* the second axis.

## Generator change (`src/openswe_traces/synth/fabricate.py`, on closure-I)

### Detail registry

A **detail** is an independently-checkable behavioural commitment of the
excised implementation: a decision point with ≥ 2 plausible values, only one
of which is gold.  The registry (store: 9, sched: 8) is a dict of
`{description, prose, requires, smoke/hidden generators, trap bodies}`.
Per-domain registry (store shown):

| id | gold | trap (plausible wrong choice) | worked example isolates |
|---|---|---|---|
| bucket_reduction | `mix(k) % 8` | `(mix(k)>>61) & 7` (the s53 failure) | keys where the two reductions differ |
| tag_normalization | trim→lower→collapse→trim dashes | lower+trim only | `"  Alpha beta "` → `"alpha-beta"` |
| empty_tag_rejection | canon-empty rejected | no empty check | `""`, `"   "` (normalization-invariant) |
| reserved_key_rejection | `2^64−1` rejected | accept everything | reserved/0/max−1 |
| size_cap | `size > limit` rejected | `size >= limit` rejected | `limit` vs `limit+1` |
| tag_dedup | each distinct tag once | duplicates kept | identical raw tags |
| tag_ordering | fold-hash order, ties lexical | plain lexical | distinct canonical tags, fold≠lex |
| overwrite_semantics | last write wins | append duplicate | put twice, count stays 1 |
| prune_boundary | removes `size > max` | removes `size >= max` | sizes 10..100, `Prune(50)` |

`sched`: band_mapping (clamp + reduction — their worked-example values
entangle, so they are one commitment), endpoint_swap, clamp_capacity,
empty_interval_reject, overlap_semantics (includes merge coalescence — the
half-open convention is one root cause with two observable faces: Add
accepts touching intervals and Merge coalesces touching runs; a schedule can
only ever hold disjoint or touching intervals, so the two are not separable
by construction), insertion_order, gap_definition, at_coverage.

Three candidate registry entries were merged during the audit for root-cause
entanglement: `size_cap` shared `overLimit` with `prune_boundary` (gold
`Prune` now checks `e.size > maxSize` directly, so the details are disjoint),
and the two `band_*` / the two overlap/merge pairs above.  The per-detail
trap audit is what surfaced each of these — the independence guarantee is
verified, not assumed.

### Drawing (`draw_details`)

Exactly `d` ids per unit, drawn by a splitmix shuffle of the registry seeded
per `(domain, seed, d)` — each grid cell is an independent sample of `d`
details (no nesting artefacts; the expected per-detail pass rate is the same
at every `d`).  `d = 0` draws everything the config can host (closure-I
default).  The manifest records the achieved set.

### Independence — how it is enforced and verified

Two details are independent iff getting one right tells you nothing about the
other.  Enforced by construction and verified mechanically per unit:

1. **Disjoint decision points.**  Each detail perturbs a distinct function or
   a distinct branch within it (bucket_reduction lives in `BucketOf`'s
   reduction; tag_dedup in `Tags`' seen-map; size_cap in `overLimit`;
   ...).  A trap flip changes exactly one decision point.
2. **Isolated test inputs.**  Every worked example and hidden test uses
   inputs chosen so no *other* detail's value affects the expected output:
   distinct already-canonical tags for ordering, identical raw tags for
   dedup, normalization-invariant empties for empty-tag rejection, in-range
   inputs for clamp tests, ascending adds for merge/gap tests, equality-of-
   clamped-inputs for the clamp test (robust to any reduction convention).
3. **Per-detail trap audit** (`fabricate.audit_unit_details`, Go-level, run
   per unit and recorded in `manifest.json["detail_audit"]`): a unit with
   exactly one detail flipped to its trap value must (a) still pass the
   v=low smoke suite — the wrong value is invisible to the L0 set; (b) fail
   the v=high smoke suite on **only** its own worked example — the wrong
   value is visible; (c) fail **only** its own hidden test — the per-detail
   verifier has teeth and nothing else depends on the value.

Model-level correlation of details (a solver that gets one "simple" choice
wrong also gets others wrong) is not assumed away: the pre-registered
analysis tests it directly via the within-unit dispersion check (§analysis).

### Verification affordance `v`

Holding the tree, |S|, hidden suite and instruction level fixed:

- **v=low** — the bug report states each drawn detail in prose (deliberately
  not a spec: "spread across all 8 buckets" never says modulo vs mask);
  `./repro.sh` runs the in-tree suite, which asserts the baseline and the
  **non-drawn** details only.  A wrong drawn detail produces no visible
  failure.
- **v=high** — the same prose, and the in-tree suite *additionally* asserts a
  discriminating worked example per drawn detail, so `./repro.sh` prints
  `got X, want Y` for a wrong choice (the affordance: cheap self-check).  The
  property is never stated — worked examples only (B3).

The hidden verifier is byte-identical across `v` (the verifier never changes
with the information knob).

### B3 "not a complete spec"

`fab_fairness.b3_not_complete_spec` verifies per drawn detail that the hidden
per-detail test checks inputs **beyond** the L0 worked examples (a seeded-RNG
loop or strictly more Go literals than the in-tree example).  A complete
example suite would be a spec (verifier_rules.md B3); these suites are
provably not, and the trap audit proves the L0 set admits an implementation
consistent with everything the solver sees at v=low that the hidden suite
still rejects.

## The grid

`d ∈ {1, 2, 4, 6, 8} × v ∈ {low, high} × seeds {11, 23, 37, 42}` = 40 units,
store domain only (a single domain keeps the analysis free of a domain
confound; the sched registry is built and tested but not run).  All units use
one fixed structural config (E=9, K=2, pipe canon, canon+misc excised), so
|S| and removed lines are constant — only the seed's constants and the drawn
details move.  L0 only.  Grid table (achieved values) and proof/audit
verdicts:

<!-- grid table injected below -->

## Local proof + self-audit (per unit)

- A1/A3 local proof (`prove_unit`): module builds/tests/vets clean, gofmt
  clean; bare excised tree + hidden suite **fails** with assertions; gold
  patch **passes** under both `HIDDEN_SEED`s; cheat patch **fails**.
- Per-detail trap audit (§independence, Go-level, recorded in the manifest).
- `scripts/fab_fairness_check.py` on every unit: params_intact,
  hidden_literals, gold_patch_literals, coverage_table, bugreport_floor (B6),
  bugreport_ceiling (B7), examples_tied, black_box (B4), details_manifest,
  b3_not_complete_spec.

## Per-detail grading format

The hidden suite is one test per drawn detail (`TestDetail_<id>`) plus a
constant guard (`TestDetail_guard`, a fixed baseline never counted as a
detail).  Every test writes one JSONL row to `/logs/verifier/details.json`
via a `recordDetail(id, pass)` helper; `tests/test.sh` runs `go test -json`,
writes a `build_failed` row if the file is missing, and sets the aggregate
reward from the exit code.  The file lands in the trial's verifier log dir:

```
{"id": "bucket_reduction", "pass": true}
{"id": "tag_dedup", "pass": false}
{"id": "guard", "pass": true}
```

The runner reads `<trial>/verifier/details.json` into `trials.parquet`
(`details` column = JSON array of these rows, plus `d`/`v` columns parsed
from the unit name).  A unit passes iff every drawn detail and the guard
pass; the analysis uses only the drawn-detail rows (§analysis).  This turns
each trial into `d` observations: the grid is 40 units but **168 detail
observations** per model per attempt pass (sum over the grid of d × 2 v × 4
seeds = 21 × 8).

## Pre-registered analysis

1. **Per-detail logistic** (primary).  `logit(P(detail passes)) ~ d + v_high
   + d:v_high`, cluster-robust SEs by unit (each seed×d×v cell).  Success =
   negative `d` coefficient at v=low with CI excluding 0, and a positive
   `d:v` interaction (v=high flattens the slope).  Implemented in
   `detail_dial_analysis.cluster_logistic_fit` (hand-rolled Newton + unit
   sandwich).
2. **Geometric prediction.**  Unit-level `log(P_pass(d))` is linear in `d`
   with slope `log(p)` and intercept `log(p_guard)` — testable as a linear
   fit of log observed pass rate vs `d` per v level, compared against the
   flat alternative (`slope = 0`, rejected when the CI excludes 0) and
   reported per v.  `detail_dial_analysis.geometric_test`.
3. **Within-unit independence.**  Under the model, the number of details a
   unit passes is `Binomial(d, p)`; the observed variance of passed-detail
   counts across units vs the binomial variance gives a dispersion ratio
   (≫1 flags model-level dependence).  `detail_dial_analysis.dispersion_ratio`.
4. **Calibration.**  `--simulate` verifies the estimator recovers the true
   `p^d` slopes from synthetic trials (run pre-launch; part of this commit).

## Launch command + cost

```bash
cd /home/evan/Documents/oswt-closureR
uv run python scripts/detail_dial_run.py \
  --tasks-glob 'experiments/detail_dial/tasks/*-L0' \
  --agent cursor-cli --model cursor/composer-2.5 --attempts 1
```

Free-model alternative:

```bash
uv run python scripts/detail_dial_run.py \
  --tasks-glob 'experiments/detail_dial/tasks/*-L0' \
  --agent mini-swe-agent --model openrouter/deepseek/deepseek-v4-flash-0731:free \
  --attempts 3
```

Cost estimate at $0.10/trial: 40 units × 1 attempt = **$4 per model** per
pass; the pilot's 4-unit fabricated run cost $0.62 with Composer 2.5 at a
similar unit size.  Each trial yields `d` detail observations plus the
guard; the per-detail vectors make the 40-trial pilot worth 168 observations
for the logistic.  Trials land in `experiments/detail_dial/trials.parquet`;
the runner is resume-safe and 429-aware.

## Reproduce

```bash
uv run python scripts/detail_dial_build.py \
  --fab experiments/detail_dial/fab --tasks experiments/detail_dial/tasks \
  --grid experiments/detail_dial/grid.csv --proofs experiments/detail_dial/proofs \
  --log outputs/closure_R.log
uv run pytest tests/test_fabricate_details.py tests/test_fabricate.py
uv run ruff check src/openswe_traces/synth tests scripts/detail_dial_*
```

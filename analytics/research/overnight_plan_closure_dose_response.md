# Overnight plan: prove the difficulty knob on excised units (draft 2026-09-19 18:10 UTC)

Status: DRAFT. Do not launch until the go/no-go criteria at the bottom are met (H and G pending).

## What the data says so far (all zero-solver-cost, today)

| claim | evidence | verdict |
|---|---|---|
| Instruction rung is a lever on natural tasks | Open-SWE: L0→L2 gain +5..+20 pts, scales with model strength; finer rungs carry nothing | real, one boundary only |
| Closure *ratio* drives difficulty on natural PRs | Open-SWE 38k instances: ΔR² +0.008 over size; size-matched quartile diff -0.004 | **refuted** on natural tasks |
| Patch size is a difficulty dial | mixture fit: feasible share 0.62→0.23, feasible pass prob 0.86→0.74 across 3 decades of size; log-linear p^k rejected | size is a **feasibility gate**, not a dial |
| Closure ratio / internal edges predict flip on **excised units** | authored dataset, 9 measured units: ratio ρ=0.72 (p=0.019), internal_edges ρ=0.66 (p=0.037); lines_removed ρ=0.41 (n.s.); L5 units (formmapping 45 edges/0.21, coalesce 14/0.15) vs L0 units (≤9 edges/≤0.105) separate cleanly | **supported, small n, mixed solvers** |

The two closure results are not in conflict: on a natural PR the diff-text proxy cannot see the call graph, and the
change is not an excision. On an excision the whole tree is present except S, so the information the tree cannot
supply is exactly the internal structure of S. The knob we can build is therefore *internal edges of the excised
set*, and the claim to prove is: **pass rate at fixed instruction level is a monotone decreasing function of the
excised set's internal edge count, holding |S| and lines fixed.**

## Design

Two arms, same measurement.

**Arm 1 — fabricated repos (no prior, exact control).** `scripts/fabricate_repo.py` (closure-D worktree). Generate a
grid over achieved ratio ∈ {0.1, 0.2, 0.4, 0.8, 1.5, 3.0}, and *within each ratio* two |S| bands (small ≈ 8–10,
large ≈ 14–17 functions) so ratio and |S| are crossed, not confounded. 4 seeds per cell → 48 units. Two domains
(store, sched) → 96 units. Levels: L0 and L2 only (the one boundary that matters). Solvers: two free OpenRouter
models (cheap, many trials: 5 attempts per unit-level) plus Composer 2.5 on a 1-in-4 subsample (1 attempt) as the
frontier anchor. Verifier: the generated hidden suite (A1/A3 proven locally per unit; B9 audit on every pass).

Expected if the knob is real: per model, logit(pass) decreasing in log(internal_edges) within each |S| band, L2
curve above L0, gap widening with edges. Expected if it is not: flat in edges, separation only by |S| or by level.

**Arm 2 — real excisions (external validity).** From the 34 already-authored units with computed metrics
(`closure_vs_flip.md`), pick 12 spanning internal_edges 0..53 with lines_removed held in 150–300 (helm/kops/gin have
enough). Package L0 and L2 (`affordance.py`), Composer 2.5, 3 attempts each, B9 on passes. 72 trials.

**Budget (corrected).** A trial is not one request: every agent turn is one OpenRouter request. Fabricated units
are tiny, so expect 10–20 turns per trial → the 1,000/day free tier buys ~50–80 trials/day, not 1,000. Composer 2.5
cost $0.08/trial on gin/bindingdispatch (330k in / 2k out); fabricated units should be similar or cheaper.

| stage | trials | free-tier days | Composer $ |
|---|---:|---:|---:|
| Pilot (6 fab units L0 ×1 Composer; same 6 ×2 free-model) | 6 + 12 | 0.3 | 0.5 |
| Arm 1 small-|S| band, free models (48 units × 2 levels × 3 attempts × 1 model) | 288 | ~4–5 | 0 |
| Arm 1 Composer anchor (24 units × 2 levels × 1) | 48 | 0 | ~4 |
| Arm 2 real excisions (12 units × 2 levels × 3 attempts, Composer) | 72 | 0 | ~6 (+ docker) |

Free-tier volume is the bottleneck, so Arm 1 runs one free model, three attempts, small band first; the large band
only if the small band shows a slope. Order: pilot → arm 2 (cheap, real repos, Composer) → arm 1 anchor → arm 1 free.

**Pilot = the smallest empirical test (run first, ~$0.50, one hour).** Six fabricated units, small |S|, ratio 0.1
(3 seeds) vs 3.0 (3 seeds), L0 only, Composer once each. Prediction: the three 0.1 units pass, the three 3.0 units
fail. All-pass → N/|S| too small, raise before spending; all-fail → the fabricated contract is under-specified
(class c), fix bugreport/contract before spending. Plus 4 unmeasured real units bracketing edges (2 at ≤3, 2 at ≥14),
L0, Composer once: the same prediction, on real repos.

**Analysis (pre-registered).** Per model: logistic pass ~ log1p(internal_edges) + |S| band + level + edges×level,
cluster-robust by unit; report the edges coefficient with CI. Success = negative, CI excluding 0, in both arms and
for at least the frontier anchor. Also fit the feasible/infeasible mixture across the grid: if edges move the
*feasible share* but not p1, we have built a feasibility gate again, not a dial — that is a fail for the dial claim.

## Go / no-go before launch

- [x] H landed (`paired_failures_small_vs_large.md`): SMALL-task failures are NOT overconfident quickies. Within
      instance, the failing sibling runs LONGER (+3.8 turns), edits more, tests more, and still fails; 62% of small fails
      are partial / wrong-site / partial-with-extra-hunks (gold-hunk coverage -0.11); the lazy arm fails *less* than
      the diligent arm (-0.09). LARGE fails are shorter (-2.4 turns) and 83% partial. Reading: the feasible-band failure
      mode is *site identification and edit-set completeness*, not confidence. Third arm therefore = **number of
      required edit sites** (multi-site excisions per rule A6, sites crossed with edges), not misleading cues.
- [x] G landed (`feasible_component_drivers.md`): verifier terms (test patch size, n_f2p, n_p2p) predict the
      feasibility gate beyond size (AUC 0.659 → 0.708) and make size redundant once included (full − size = 0.709);
      inside the feasible band the verifier block adds R² +0.02, specification density adds nothing (+0.0015).
      So the gate is *how much the hidden tests demand*, and size was proxying for it. Arm 1 hidden-suite assertion
      count becomes a crossed factor (2 levels: minimal vs full property suite).
- [~] C in progress: first T1 read is a NEGATIVE size × rung interaction (-0.041, CI upper bound below 0 pending),
      i.e. the L0→L2 gain shrinks as tasks get bigger — consistent with the gate story (no spec rescues an infeasible task).
- [ ] Devin swe-2 stable (probe 5.8 s today) and Composer key live; docker base images for helm/kops/gin present.
- [ ] Arm-1 free models chosen from `openrouter.ai/api/v1/models` with pricing 0 and tool use; 429 backoff in the runner.

## Pilot log

**2026-09-19 18:21Z — arm 2 pilot, first pass (4 Composer L0 trials, $1.29 = $0.32/trial, 5.5M input tokens mostly cached).**

| unit | internal edges | predicted | reward | class |
|---|---:|---|---:|---|
| gin/bodydecoders | 14 | fail | **1.0** | (a) valid pass |
| helm/ignorerules | 2 | pass | 0.0 | (d) verifier `[setup failed]`: hidden tests import `example.internal/chartkit/v4`, tree is `example.internal/helm` |
| kops/flagbuilder | 2 | pass | 0.0 | (d) same, `clustkit` vs `kops` |
| helm/strvalsparser | 53 | fail | 0.0 | (d) same |

Only one valid reading, and it went against the prediction (14 edges passes L0). Three trials are the known helm/kops
module-path gap (fresh-laptop defect 3 for the *tests* side); job J rewrites the hidden tests, proves A1/A8 in-image,
and reruns them. Budget correction: Composer is ~$0.32/trial on real repos, 4× the gin/bindingdispatch figure.

**Fabricated arm: Grok 4.6 fairness audit FAILS L0 for all three demo units (class c).** The hidden suite demands
seeded mixer constants (`refMul`, `refRot`, `refFoldBase`) that exist only in the hidden oracle and gold; they are not in
the bug report, the excised tree, or the repro output. L0 is infeasible by construction, and the L2 coverage table maps
one of seven hidden tests. B4/B6/B7 pass. **Fabricated pilot is NO-GO at L0 until the generator either (i) makes the
constants discoverable (repro output prints the expected values for the worked examples; or constants live in a
config/const block that is NOT excised), or (ii) the run is defined at L2 with a complete coverage table.** This is the
gate-vs-dial trap in its purest form: withholding a constant is a feasibility gate, not difficulty.

**2026-09-19 18:45Z — second infra defect (job J).** The helm/kops `environment/src` trees in the pilot copies were
NOT excised: `materialize` applied the excision patch against the wrong module path, the patch silently failed, and a
full working tree was shipped. The hidden suite passes on the "bare" tree, so any trial on those dirs would have scored
reward 1 regardless of the solver — a fake positive, the mirror image of this morning's fake negatives. gin is excised
correctly. J restores the excision via the rebranded patch and reruns; K's preflight gate (bare tree MUST fail for the
right reason) is the structural fix, in any language. Neither defect could have been seen from reward alone.

**C landed (`framework_tests_openswe.md`).** Instruction benefit shrinks with size for the strongest combos
(rung × size interaction -0.051, CI [-0.108, -0.013]); not significant pooled. Closure ratio fails all three natural-task
tests (interaction, discrimination, failure-signature). Ratio survives ONLY on excised units (A2). Framework decision:
on natural tasks the levers are rung (one boundary) and the verifier's demand (G); on excisions the candidate lever is
internal edges, unproven until the pilot reruns cleanly.

**2026-09-19 19:00Z — arm 2 pilot rerun (job J), clean.** helm/ignorerules (2 edges) PASS, helm/strvalsparser (53) PASS,
kops/flagbuilder (2) FAIL (legit: missed the `flag-include-empty` tag semantic, discoverable in-tree), gin/bodydecoders (14)
PASS from the first pass. Composer 3/4 at L0, all verdicts class (a). Direction is against the internal-edges prediction
(53 passes, 2 fails); n=4 × 1 attempt, so not a reading yet (C6).

**H semantic labels (293 SMALL fails, free-tier models):** wrong_root_cause 25%, incomplete_stopped_early 22%,
fixed_symptom_not_cause 21%, missed_second_site 12%, misread_issue 5%, broke_other_test 4%, environment 0.3%. Two-thirds of
small-task failures are causal-reasoning errors (wrong cause / symptom-only fix), one-third completeness errors. Not
laziness, not environment. The dial for the feasible band is therefore *causal depth*: how far the observable symptom sits
from the site that must change, and how many sites. Third arm restated: **symptom-to-cause distance and site count**.

**I landed: 72/96 fabricated cells reached (6 cells unreachable at the ratio extremes), L0/L2 packaged, runner written.
Validation trial with a free model (mini-swe-agent + deepseek-v4-flash:free) is still running after 60 min** on a
24-function module, thrashing on shell escaping. Free-tier volume is therefore worth less than planned: one trial may burn
50+ requests and an hour. Fabricated arm is NO-GO on two counts until L (generator fairness fix) lands and the runner is
re-validated with a stronger free model or a smaller per-trial turn cap.

## Fabricated pilot result — 2026-09-19 20:55Z — **the internal-edges dial does not separate**

6 units, L0, Composer 2.5, 1 attempt each, $0.62 total. |S| = 10 and instruction level held constant; only the
internal/boundary edge split differs.

| unit | group | internal edges | reward | wall |
|---|---|---:|---:|---:|
| store-s11-r015 | low | 4 | 1.0 | 110s |
| store-s23-r020 | low | 4 | **0.0** | 104s |
| store-s37-r020 | low | 4 | 1.0 | 112s |
| store-s42-r250 | high | 9 | **1.0** | 186s |
| store-s53-r250 | high | 9 | 0.0 | 196s |
| store-s67-r300 | high | 9 | 1.0 | 93s |

**Low 2/3, high 2/3 — no separation at all.** Within-group variance (seed) swamps any between-group effect. The earlier
partial read (first 3 trials: low-pass, low-pass, high-fail) was the same coin landing in prediction order; reporting it
as "reproducing the prediction" was over-reading n=3.

Reading: at |S|=10 with a 4-vs-9 edge contrast, internal edge density does NOT move pass rate. Either the knob is false,
or the contrast is far too weak (edge counts are integers on a small graph, and the generator cannot produce a wider
spread at this |S|). The honest position: the ONLY remaining support for the closure lever is the n=9 Spearman in
`closure_vs_flip.md` (mixed solvers, mixed levels, heavy ties) — now under adversarial statistical review.

**Decision: do NOT spend the 960 free-tier requests on Arm 1 as designed.** The fabricated dose-response run is
cancelled pending either (a) a generator that can produce a 10x edge-density spread at fixed |S|, or (b) a different
knob. Arm 2 (real excisions) stands, but its own 4-unit pilot also went against the prediction (53 edges passed, 2 edges
failed), so it is not evidence for the lever either — it is evidence the packaging now works.

**What the evidence actually supports as the next knob**, from the largest evidence base we have (293 labelled failures
+ 38k instances): symptom-to-cause distance and required site count. Job N is measuring both on natural tasks now.

### Failure audit of the pilot (C1) — both failures legitimate, neither structural

| unit | group | failing assertion | what the agent did |
|---|---|---|---|
| store-s23-r020 | low (4 edges) | `TestTagsSort: Tags() = [... zebra zebra], want [... zebra]` | missed that `Tags()` deduplicates — one behavioural detail |
| store-s53-r250 | high (9 edges) | `TestBucketOfProperty: BucketOf(1) = 0, want 3` | reconstructed the whole mixer correctly, then reduced with `(h>>61)&7` instead of `h % 8` |

Both are class (a). Neither is about closure structure. The s53 agent's patch differs from the two passing high-ratio
patches by exactly one line: `int((h >> 61) & (nbuckets - 1))` vs `int(mixPipe(k) % uint64(nbuckets))`.

**And the reduction it chose contradicted the worked examples it was given.** The instruction states key 7 → bucket 0,
key 42 → 4, key 0 → 2. Running both reductions on the unit's own constants:

| key | `mod 8` | `(h>>61)&7` | instruction says |
|---:|---:|---:|---:|
| 7 | 0 | 1 | 0 |
| 42 | 4 | 4 | 4 |
| 0 | 2 | 3 | 2 |

`mod 8` matches all three; the agent's choice matches one. It never checked its implementation against the three
examples in its own bug report. That is precisely job H's natural-task finding (`paired_failures_small_vs_large.md`):
the failing sibling does MORE work — more edits, more test runs — without verifying against the evidence it was handed.

**Reframe.** Difficulty on these units is dominated by (i) the number of independent behavioural details that must each
be right, and (ii) whether the model verifies its choice against evidence available to it — not by the call-graph shape
of the excised set. Two independent lines now agree (293 labelled natural-task failures; 6 fabricated trials audited to
the line). The candidate dial becomes **detail count × verification affordance**, which predicts a smooth monotone curve
(each unverified detail is an independent chance to diverge) rather than the flat result we just measured.

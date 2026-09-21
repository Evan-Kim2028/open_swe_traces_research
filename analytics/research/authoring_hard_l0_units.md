# How to author L0-hard units that are provably fair, solvable and verifiable

2026-09-19, derived from the repaired-dataset sweeps (`sweep_L0_2026-09-19.md`): 30 units screened at L0 (k=3, 90%
deterministic, 0 infra), 12 failed 0/3, and 7 of the 11 measured so far pass at L2.

## 1. The flip is the certificate

The thing we spent the day looking for is not a property of the code — it is a **measurement**. A unit that fails at
L0 and passes at L2 has, in two trials, proven all four properties at once:

| property | proven by |
|---|---|
| **hard** at L0 | it failed every L0 attempt |
| **fair** | the information in the L2 contract suffices — withholding it, not impossibility, caused the L0 failure |
| **solvable** | an actual model solved it, not just the gold patch |
| **verifiable** | preflight already proved gold passes, bare fails with assertions, cheat fails, in-image |

So fairness stops being a hand audit (which is what Grok has been doing all day) and becomes a screen. The three
outcomes and what each means:

| L0 | L1 | L2 | verdict |
|---|---|---|---|
| pass | — | — | too easy. Reject, or add details / remove the in-tree tests that reveal them. |
| fail | pass | — | **accept, and the omitted invariant was NOT the binding one** — stop here, L2 is unnecessary. |
| fail | fail | pass | **accept: flip-validated hard, and the omitted invariant IS the binding commitment.** |
| fail | fail | fail | investigate: a detail missing from L2's coverage table (packaging bug) or a capability limit. Do not ship as "hard" without knowing which. |

**Escalate L0 → L1 → L2, not L0 → L2.** L1 is the full contract with exactly ONE invariant removed, so the L1 result
attributes the difficulty to a specific commitment instead of to "more prose". Two L1 variants make it causal: omit the
invariant the unit actually failed at L0 (predict fail) versus omit one it already got right (predict pass). Cost is the
same per level at k=1 screening, and a unit that passes L1 never needs an L2 run.

## 2. What makes a unit hard at L0 — empirically, not theoretically

Every L0 failure in the sweep was **one missed behavioural detail**, the same one on every attempt, and almost always
an edge case that the happy path never exercises:

`empty ig list` · `empty data must not set Content-Length` · `snippet named mainTemplate must be rejected` ·
`create: file already exists` · `IP SAN: []` · `flag-include-empty *string must emit` · `Ignore(".",dir=false)`

The model reimplements the bulk correctly and loses on a commitment it never saw stated and could not infer from the
surrounding code. So difficulty here = **the number of independent behavioural commitments that are not inferable from
the tree**. Not code size, not call-graph shape (both refuted today).

## 3. The authoring recipe

1. **Choose a closure with edge-case surface.** Public behaviour should include several commitments with ≥2 plausible
   values: empty/nil input, duplicates, reserved names or values, boundary sizes, ordering, error-vs-zero-value.
   A pure data transformation with one obvious semantics is a bad unit; it will pass at L0.
2. **Remove the in-tree tests that exercise those commitments.** This is the real discoverability knob. Rule B3 already
   says a complete in-tree suite is a spec; the sweep shows the converse — a unit stays hard exactly where the in-tree
   tests do not pin the detail down. (Checked by hand: `kops/templater`'s reserved name sits in plain view as
   `templateName = "mainTemplate"` in non-excised code and the model still missed it, which is fair-and-hard;
   `gin/streamrenders`' empty-body rule is in neither the instruction nor any in-tree test, which is why its L2 flip
   matters — it proves the rule was learnable from prose.)
3. **Author writes DETAILS.md (numbered commitments) and bugreport.md (L0). Do not write the L2 coverage
   table by hand.** A coverage row written from a *reading* of gold is the defect that burned go-github
   (0/10 L2 flips) and 7/7 audited double-failures. The table is derived later, from the tests.
4. **Verifier writes one hidden property test per DETAILS.md line** (`TestDetailNN_…`), black-box (B4),
   seeded/random where possible (B5). Per-detail tests turn each trial into `d` observations instead of 1.
   The verifier reads api.md + DETAILS.md + the excised tree; never gold, never bugreport, never the
   contract (it does not exist yet).
5. **Reconciler writes contract.md from (hidden assertion, gold hunk).** One coverage row per hidden test,
   behavioural prose, B7-clean. `uv run python scripts/reconcile_contract.py unit --details … --hidden …
   --gold … --out contract.md`. "Missing" is then structurally impossible. Packaging runs this
   automatically when DETAILS.md is present.
6. **Preflight before any trial** (bare fails with assertions, gold passes, cheat fails, in-image).
7. **A13 is mandatory on cold repos** (fewer than 3 units already in the screened dataset). The judge's
   false-alarm rate is accepted there because the prior on defects is worse. Warm repos keep A13 as a
   backstop, not a packaging gate.

Drop closures whose behaviour is mostly implicit encoding-shape (JSON field order, `omitempty`,
null-vs-absent) unless the hidden tests pin the wire form with concrete strings the contract can quote.
Those commitments are the "domain, not cold-start" case: a derived row can be true of gold and still
be a poor L2 instruction.

## 4. Yield and cost, honestly

Observed rates on the current (undirected) authoring process:

| step | rate |
|---|---|
| authored unit fails at L0 | 12/30 = 40% |
| L0-failing unit flips at L2 | **8/12 = 67% (final, 36 trials)** |
| **authored unit → flip-validated hard** | **27%** |

At $0.35/trial (measured: $31.61 / 90 real-repo trials) and k=1 screening:

| line | cost |
|---|---|
| L0 screen, per unit | $0.35 |
| L2 screen (only the ~40% that fail L0) | $0.14 amortised |
| author + verifier (Composer, ~3 min each) | ~$0.50 |
| **per authored unit, end to end** | **≈$1** |

So **100 authored units ≈ $100 and yields ~26 flip-validated hard units.** Reaching *100 hard* units at today's hit
rate needs ~385 authored units (~$385, and ~40 repos' worth of authoring at 10 units/repo). The 250M Composer tokens on
offer ≈ 170 real-repo trials ≈ screening for ~120 units — enough for the next dataset, not for 100 hard ones.

Two levers change that math:
- **Author for difficulty** (steps 1–2 above) should lift the 40% L0-fail rate. Every point of lift is directly
  proportional savings. This is the single highest-value thing to test next: author 20 units *to the recipe* and
  measure the L0-fail rate against the 40% baseline.
- **Grade per detail, not per unit.** With one hidden test per commitment, an "easy" unit still reports which 6 of 8
  details the model got, so it is not wasted. Binary hard/easy throws away most of the signal we paid for; per-detail
  grading means 100 *units* is worth far more than 100 *hard units* under the current scheme.

## 5. What 100 units looks like

- **Dataset today:** 30 preflight-clean (helm/kops/gin). Job S is repairing client-go + goa → ~50.
- **To 100:** 5 more repos at ~10 units each. Authoring is not the bottleneck (Composer authored a go-github unit in
  ~3 min); the bottleneck was packaging, and the preflight gate closed that (0 infra in 105 trials today).
- **Per repo, one-time:** pin a commit, obfuscate, base image, rebrand map — all config since K's work; job P is
  testing whether that holds for a non-Go language.
- **Screening run:** 100 units × k=1 L0 ≈ $35, plus ~$15 of L2 escalation. One evening.

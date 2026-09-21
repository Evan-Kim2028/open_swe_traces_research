# Stage 1 write-up — outline

**Title WIP:** Building a synthetic SWE factory: The Groundwork
**Subtitle:** A data engineer’s perspective.

Planning desk (outline, then pictures): `analytics/research/paper_outline_planning.html`.

**Scope, stated up front:** Go only, nine repositories, in-repo excision. Multi-language is
stage 2. This is a factory built in stages; the claim is about the construction and the
verification protocol, not about universality.

All figures below are measured as of 2026-09-21 and live in the repo — no number here
should be restated without re-deriving it from `trial_ledger.py` / `roots.py`.

---

## 1. The problem

Synthetic SWE tasks are easy to generate and hard to trust. Two failure modes dominate:

- **unverifiable** — no way to know a solution is correct beyond "tests pass", and the
  tests may not cover what was removed
- **unsolvable** — the task omits information no solver could recover, so failure measures
  the task rather than the solver

State the target: tasks that are simultaneously *hard*, *fair*, *verifiable* and
*provably synthetic*, with difficulty that can be dialled rather than hoped for.

## 2. The construction

Excise a self-contained behaviour from a real repository. Grade with a hidden black-box
suite. Vary what the solver is told.

- **excision** — remove the implementation, leave the call sites and public surface
- **hidden suite** — written by an agent that never sees `gold.patch` (the wall)
- **affordance ladder** — L0 bug report only … L2 full prose contract + coverage table …
  L6 all tests

**The flip certificate** is the central object: a task that **fails at L0 and passes at
L2** is hard (the report alone is insufficient), fair (the contract alone suffices),
verifiable (the suite decides) and difficulty-calibrated (the gap is the affordance).

Figures: **527 authored → 330 trialled → 294 decided → 139 certified.**

## 3. The verification protocol

The wall between author and verifier is what makes the tests independent of the solution.

Checks worth naming individually, with what each one actually caught:

| check | asserts | caught in practice |
|---|---|---|
| A1 | gold restores → suite passes | `comfortfade`, `errfmt`: gold did **not** satisfy the suite |
| A3 | cheat patch → suite fails | `osbound`: a special-case passed the consumer suite |
| A12 | gold does not touch tests | — |
| B7 | no file names, line numbers, private identifiers in the prompt | — |
| preflight | all of the above, per unit, in Docker | quarantines failures instead of shipping them |

**Point to make:** these are not hypothetical. Preflight failures are the mechanism by
which unsolvable-as-specified tasks are kept out of the bank, and they fire on real units.

## 4. Does the ladder discriminate?

The load-bearing empirical claim of stage 1.

- **41% of units are solved at L0** — the bug report alone is often enough, so L0 is a
  real screen rather than a formality
- of those that resist L0, the contract flips a majority at L2
- **164 certificates: 140 single-solver, 16 cross-solver** (L0 failed on one model, the
  flip came on another — kept and labelled, since that may reflect a capability gap
  rather than the affordance)

Report the multi-model split honestly: **~98% of trials to date are Composer 2.5**; Devin
and Grok samples are thin and accumulate over time.

### 4a. The rung a unit needs is its difficulty

The strongest result in stage 1, and it came from a category we had been discarding.

A unit that failed L0 **and** failed L2 was classified "non-flipping" and written off —
48 of them, ~15% of everything decided. That classification was wrong. Escalating them one
rung at a time (L2 -> L3 -> L4 -> L5 -> L6) flips almost all of them:

| cohort | verdicts | flipped |
|---|---|---|
| hand-escalated (before the policy existed) | 13 | 11 (85%) |
| `sweep_escalate`, first automated cohort | 16 | **15 (94%)** |

**Non-flipping is now 0.** Every unit in that bucket either certifies at a higher rung or
still has a rung left to try. Certificate count went 139 -> 164 without authoring a single
new unit, and trials-per-certificate fell 8.3 -> 7.3.

Certificates by binding rung — the lowest rung at which the unit passes:

| rung | certificates | what the solver was given |
|---|---|---|
| L2 | 140 | the complete prose contract |
| L3 | 1 | contract + hidden test names |
| L5 | 23 | contract + the representative test restored into the tree |

This converts the bank from a binary (hard / too easy / broken) into a graded one. A unit
that flips only at L5 is *harder* than one that flips at L2 — it is not a failed task, and
the rung is a difficulty measure the construction produces for free.

**Method note worth stating in the paper.** An earlier policy probed L5 first, because the
hand-escalated units had all flipped there, and bisected downward: same binding rung in
~2.6 trials per unit instead of 4. It was cheaper and it was the wrong experiment. Jumping
the ladder shows only that *a* rung works, never that it is the rung the unit *needs*. The
ladder is the measurement, so every step gets walked.

**Caveat, measured rather than assumed.** A certificate is `max(reward) > 0` at the binding
rung. 11 of 164 rest on one pass in three or more trials; `helm-depresolver` binds on 1 of
6 and is independently flagged B10-unsolvable by the linter. The ledger records
`n_pass_at_rung` and a `thin` flag so the distinction is auditable rather than invisible.

## 5. Yield varies by repository, and it is predictable-ish

| repo | certified / decided | |
|---|---|---|
| client-go | 7/9 | 78% |
| bbolt | 15/22 | 68% |
| nats-server | 7/11 | 64% |
| kops | 25/40 | 62% |
| goa | 19/31 | 61% |
| go-git | 25/45 | 56% |
| gin | 14/39 | 36% |
| helm | 8/23 | 35% |
| **go-github** | **5/35** | **14%** |

The go-github outlier is the interesting case: **not exhausted** (only 14% of its files
are claimed, 224 untouched) but badly *distributed* — 83 units crammed into 37 files, 20
of them in `github/github.go`. Density predicts too-easy better than depletion does.

Tie to the redundancy measurement: 20% of authored units excise overlapping line ranges;
a claim registry now forbids re-cutting claimed regions.

## 6. A negative result: cross-repo construction does not work

Worth its own section — it is the cleanest finding in the set.

Cut a function from dependency L, leave consumer C intact, grade on C's behaviour.

| round | selection rule | solved at L0 |
|---|---|---|
| 1 | any dependency | 19/23 (83%) |
| 2 | only behaviour that cannot be recalled | 12/14 (86%) |

Round 2 deliberately targeted obscure project-specific forks with arbitrary rules and
**did not improve on round 1**. **Mechanism:** C's call sites *are* a specification of L.
The solver reads usage and reconstructs; recall never enters into it, so making the target
unrecallable cannot help. In-repo excision does not have this property because the callers
usually go with the excised code.

Secondary finding: the verifier's own difficulty judgement was not predictive — it named
five units "strongest resistance" and four were solved from the bug report alone.

## 7. Provenance: the bank is synthetic

- 1545 task files, **zero** references to issues, PRs or CVEs
- 102,586 additions against 638 non-stub deletions (0.62%)
- 93% near-pure restoration
- excision record present on every unit that has one, none missing

This is what separates the work from mined-issue benchmarks and is a prerequisite for any
claim about contamination.

## 8. The economics of running a factory

Arguably the most transferable section — the construction is the idea, but this is what it
costs to operate.

- **2.32B tokens, 1228 trials, 139 certificates** — 17M tokens per certified unit, 7.9M
  per decided unit
- **8.8 trials per certified** against a floor of 2 (one L0 + one L2)
- **73% of all spend went to trials beyond that minimum path**: 39% repeat-L2 (contract
  repair retries), 15% repeat-L0, 19% ladder rungs
- gating is worth ~**4×** over sequencing (controlled: parallel-ungated 9.0 trials/cert,
  sequential-ungated 7.7, sequential-gated 3.2)
- authoring is cheap: **0.54M tokens / $1.25 per unit** measured; trials dominate at 94%
  of cost

**The lesson to draw:** the expensive part of a task factory is not generating tasks. It is
not re-measuring what you already know. Every guard that refuses a trial is worth more than
any improvement to generation.

## 9. What stage 1 does not claim

Say this plainly rather than letting a reader infer it:

- **Go only.** The excision mechanic leans on Go's package and test conventions.
  Multi-language is stage 2, deliberately.
- **Largely one solver.** Difficulty is measured relative to a solver; "too-easy" currently
  means "Composer 2.5 solved it". Devin and Grok samples accumulate over time.
- **No held-out solver validation yet** — whether the screen generalises to a model that
  never participated in screening is an open question, not a settled one.
- **Ladder rungs L1/L3–L6 are sparsely sampled**; the dose-response curve is stage 2+.

## 10. Stage 2 and beyond

- second language, to test whether the construction transfers
- held-out solver check on the too-easy set
- contract repair as a measured stage rather than an unbounded retry
- the affordance dose-response curve at full rung coverage

---

## Figures worth building

1. **The funnel** — 527 → 330 → 294 → 139, with losses labelled at each step
2. **Per-repo yield** — bar chart, go-github as the annotated outlier
3. **Where the tokens went** — 27% minimum path vs 39/15/19% waste categories
4. **Cross-repo vs in-repo** — L0 solve rate, 83/86% against 41%
5. **The construction diagram** — author → verifier → reconciler, with the wall

## Source of record

`trial_ledger.py` (certificates, multi-model split) · `roots.py` (unit census) ·
`task_overlap.py` (redundancy) · `synthetic_provenance.py` (provenance) ·
`KNOWN_LIMITATION_cross_repo.md` · `MULTIMODEL_BANK.md`

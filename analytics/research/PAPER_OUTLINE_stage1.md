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
which unsolvable-as-specified tasks are kept out of the dataset, and they fire on real units.

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
| `sweep_escalate`, first automated cohort | 16 | 15 (94%) |
| `sweep_escalate`, cumulative | 50 | **37 (74%)** |

**Non-flipping is now 0.** Every unit in that bucket either certifies at a higher rung or
still has a rung left to try. Certificate count went 139 -> 164 without authoring a single
new unit, and trials-per-certificate fell 8.3 -> 7.3.

Certificates by binding rung — the lowest rung at which the unit passes:

| rung | certificates | what the solver was given |
|---|---|---|
| L2 | 158 | the complete prose contract |
| L3 | 3 | contract + hidden test names |
| L5 | 40 | contract + the representative test restored into the tree |

Of the 43 above L2, **14 have EARNED their rung** — the rung below was trialled and failed
— and 29 are still pending that check. `certificates()` records this as
`rung_established`, so "flips by L5" is never silently reported as "needs L5".

This converts the dataset from a binary (hard / too easy / broken) into a graded one. A unit
that flips only at L5 is *harder* than one that flips at L2 — it is not a failed task, and
the rung is a difficulty measure the construction produces for free.

**Method note worth stating in the paper.** An earlier policy probed L5 first, because the
hand-escalated units had all flipped there, and bisected downward: same binding rung in
~2.6 trials per unit instead of 4. It was cheaper and it was the wrong experiment. Jumping
the ladder shows only that *a* rung works, never that it is the rung the unit *needs*. The
ladder is the measurement, so every step gets walked.

### 4b. A certificate is single-solver by construction

The flip is evidence about the AFFORDANCE only when the same solver does both halves.
When one model fails L0 and a different one passes L2, the flip may record nothing but
the second model being stronger.

This was not enforced at first. L0 screening and L2 certification were routed to whichever
solver had a free slot, and cross-solver certificates reached 31 of 201 — 15% — before the
rule went in:

| unit | L0 failed by | binding rung passed by |
|---|---|---|
| `attachsvc` | composer | devin |
| `channelver` | devin | composer |
| `flhashmap` | grok | composer |

The rule now holds at every rung: **L0 and L1 anyone may screen; L2 and above, only a
solver that has failed the unit lower down.** It is enforced in three places because no
one of them is sufficient — `orchestrate` steers each sweep to a cohort its solver owns,
`sweep_seq` gates the units inside it, and `trial_guard` refuses to treat a cross
certificate as a finished unit so the correct solver can still claim it.

That last part matters for reproducibility: the 31 affected units were not discarded. They
are reopened, and a later pass by the solver that failed them converts the row from `cross`
to `single` without re-authoring anything.

**Cost, stated plainly.** Composer can no longer absorb Devin's L2 overflow when Devin is
full, which is a throughput loss on a machine where Composer is the bottleneck. Taken
deliberately: a certificate that does not isolate the affordance is not worth the slot it
saves.

**Caveat, measured rather than assumed.** A certificate is `max(reward) > 0` at the binding
rung. 11 of 164 rest on one pass in three or more trials; `helm-depresolver` binds on 1 of
6 and is independently flagged B10-unsolvable by the linter. The ledger records
`n_pass_at_rung` and a `thin` flag so the distinction is auditable rather than invisible.

### 4c. Every certificate rests on ONE screener's opinion — so we went and checked

A certificate says the unit is hard: it failed at L0, where the agent gets only a bug
report. Condemnation is deliberately model-agnostic — a task any frontier agent fixes from
the report alone is not hard, even if another would have failed it — which makes "no
solver can do this from the report" a claim about solvers in general.

Screening never tested that claim. Slots were filled by whoever was free, and Composer ran
447 L0 trials to Devin's 51, so a unit's L0 verdict is almost always a single model's
opinion. Only two units in the whole dataset had an L0 verdict from both solvers. The
agreement we were implicitly assuming had never been measured; its evidence was an absence
of trials.

If the two screeners disagree at rate p, then roughly p of the certificates were never
hard, and the headline count is inflated by that much.

**Method.** 60 certified units are enrolled on a roster that reopens their existing L0
directory to the solver that has not screened it. Nothing is re-authored, re-staged or
renamed; the verdict lands on the real unit's ledger row, so a pass condemns the unit for
real and it leaves the dataset. The shrinkage IS the measurement.

The trial is unusual in being incapable of doing harm. Either the second screener fails,
and the certificate is strictly better evidence than before, or it passes, and the unit
should never have been counted. A second screen can only destroy a certificate, never mint
one — which is why the guard was wrong to refuse it, and refused it in three separate
places before this (`trial_guard` as "already decided", `solver_match` by handing the slot
back to the solver that had already screened it, and `orchestrate` via a shadow copy of
the guard's rule that ran before the guard).

**Sample size is the whole design.** With zero disagreements observed, the exact one-sided
95% bound on the disagreement rate is `1 - 0.05**(1/n)`:

| n, all agreeing | disagreement rate | certificates possibly inflated (of ~215) |
|---|---|---|
| 20 | < 13.9% | up to 30 |
| 30 | < 9.5% | up to 20 |
| **60** | **< 4.9%** | **up to 10** |

n=20 would only support "fewer than 30 of our certificates are bogus", which is a caveat,
not a result — and it is a weak test besides: if the true rate were 10%, 20 clean screens
happen 12% of the time. n=60 is the smallest sample that puts the bound under 5%. It costs
no money, because 48 of the 60 route to Devin, which is free and was structurally starved
anyway: the single-solver rule leaves Devin few units it may certify, so second screens are
the rare work that fits its idle capacity instead of competing for Composer's budget.

**Result so far: the softness is one repository, not the dataset.** 18 of 65 verdicts in:

| source repo | screened | condemned |
|---|---|---|
| `go-github` | 5 | **4 (80%)** |
| everything else (kops, helm, gin, goa, client-go, unprefixed) | 13 | **0 (0%)** |

The pooled rate is 22%, and pooling is the wrong operation. Every condemnation comes from
one repository; thirteen units from five other repositories held up unanimously. Fisher's
exact on 4/5 against 0/13 is p ~ 0.002, so this is very unlikely to be how a uniform rate
would look. Extrapolating the pooled 22% across the dataset predicts ~48 bad certificates;
the actual exposure is the six `go-github` certificates still standing, about 3%.

The tooling now refuses to make that mistake. `second_screen.py --report` prints the
per-repo split first and suppresses the pooled extrapolation whenever one family's rate is
at least three times the rest, naming the at-risk population instead. An earlier version
gated that warning on a family being *100%* condemned, and the moment one `go-github` unit
agreed (4/5 rather than 5/5) the misleading pooled number came straight back — the flag has
to fire on heterogeneity between families, not on perfection within one.

**Why one repo would behave this way** is the interesting part, and we should say it is a
hypothesis. `go-github` is a generated API client: its issues and its code turn on concrete
field names and JSON tags, so a bug report that names the affected field can carry most of
the fix with it. The L0 affordance is nominally "bug report only", but how much a bug
report gives away is a property of the source repository's conventions, not of the rung.
If that holds, synthetic task difficulty is not repo-independent and a ladder calibrated on
one codebase does not transfer to another — which would be a limitation worth stating
plainly rather than a defect to quietly patch out.

**Status.** 47 screens outstanding, including the six remaining `go-github` certificates,
which were enrolled deliberately to settle that family at 10/10 rather than 5/10. The
pre-registered n=60 still governs the dataset-wide claim; the per-repo split is a
sub-analysis the sample was not powered for, and is reported as such.

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

## 7. Provenance: the dataset is synthetic

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
`KNOWN_LIMITATION_cross_repo.md` · `MULTIMODEL_DATASET.md`

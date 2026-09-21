# DETAILS gate: gating verifier commitments for derivability

*2026-09-20. Job DETAILSGATE. Instrument: `scripts/details_gate.py` →
`src/openswe_traces/details_gate.py`. Output: `outputs/details_gate.jsonl`
(170 units, 1560 lines), per-unit `_author/details_gate.json`, Composer cache
under `outputs/composer_cache/`.*

## Why this gate exists

The verifier is blind to gold by design: it authors one hidden property test
per numbered line of `_author/DETAILS.md`, working from DETAILS.md + api.md +
the excised tree. It can only grade what DETAILS.md lists. When an author lists
an arbitrary choice as a behavioural commitment — an exact error-message
literal, a type spelling, test scaffolding — the verifier faithfully turns it
into an assertion no solver can ever derive. Gold still passes preflight; the
task is unfair at every rung including L5.

The authors already flag this: 150 of 170 DETAILS.md files carry an
`Inferable:` annotation (49 `no`, 33 `doc`, 17 `partially`, 5 `yes` as the
structured forms; per line the prose forms are "Not inferable" /
"Inferable." / "Partially inferable"). No stage of the pipeline ever consumed
it. This gate consumes it, calibrates it against an independent model
judgement, and enforces the result at packaging time.

## Design

For every unit's `_author/DETAILS.md` the gate emits, per numbered line:

- `author_inferable` — the author's annotation, normalised to
  `yes`/`doc`/`partially`/`no` (`none` when absent).
- `kind` — an independent Composer judgement of where the answer lives:
  `ARBITRARY` (nowhere), `DERIVABLE` (the repo implies it),
  `COUNTER` (the obvious reading is wrong but the rule is discoverable).
- `action` — `grade`, `grade-shape-only`, or `drop`.

The model sees only solver-visible material: api.md (a bounded proxy for the
exported surface), bugreport.md (the L0 instruction), and the excision diff
filtered to `+` stub lines and context lines — `-` lines (the excised
implementation) and test-file hunks (the deleted in-tree tests) are gold and
are stripped. Author annotations are stripped from the commitment text so the
judgement is independent. One Composer call per unit, cached by content hash
(`detailsgate/v1/<family>/<sha>`); the batch is resume-safe and appends
per-unit records to `outputs/details_gate.jsonl` as they land.

Fairness test applied: a commitment is fair only if a competent engineer
holding the rung's information would *produce* the graded behaviour. Judged at
the L0 floor (bug report + excised tree); what is derivable at L0 is derivable
everywhere.

### Unit verdicts

- `fail` — model judgement unavailable, or no gradeable line remains.
- `low-discrimination` — more than half the lines are ARBITRARY.
- `pass` — otherwise.

`pipeline.verifier.run_verifier` and `pipeline.package.package_levels` both
refuse a unit whose DETAILS.md exists without a passing `details_gate.json`
(rule id `DETAILS`). Batch-1 units have no DETAILS.md and are unaffected.

## Results

170 units judged, 1560 numbered lines, zero model errors, zero unparseable
answers.

| kind | lines | share |
|---|---:|---:|
| DERIVABLE | 1075 | 69.0% |
| ARBITRARY | 245 | 15.7% |
| COUNTER | 240 | 15.4% |

| action | lines |
|---|---:|
| grade | 1315 |
| grade-shape-only | 199 |
| drop | 46 |

Unit verdicts: 166 pass, 4 low-discrimination (`gin-recoverymw`,
`kops-hashparse`, `gin-colorfmt`, `gin-mailfmt`), 0 fail. 114 of 170 units
carry at least one ARBITRARY line; 114 units have at least one line weakened
to shape-only or dropped. Per-repo ARBITRARY share is flat ~0.15 except helm
(0.27) and goa (0.18).

## Confusion matrix: model (rows) vs author annotation (cols), n=1560

| | no | doc | partially | yes | none |
|---|---:|---:|---:|---:|---:|
| ARBITRARY | 174 | 5 | 25 | 24 | 17 |
| DERIVABLE | 509 | 41 | 170 | 261 | 94 |
| COUNTER | 166 | 7 | 12 | 19 | 36 |

Two disagreement flows dominate:

**author `no` → model DERIVABLE/COUNTER: 675 of 849 `no` lines (79.5%).**
The author answered "is it inferable from the remaining tree?" The gate asks
"would a solver produce it given the bug report + the tree?" — the fairness
quantity. Reading the reasons on all 675: 208 cite the bug report's
expected-vs-got clauses (B6 requires them; the solver holds them at L0), 160
cite surviving doc comments, 122 cite exported names/signatures/idioms,
7 cite surviving tests, 178 cite other tree artifacts. The author's `no` is
true of the excised tree alone and false of the rung's full information. The
model is right *for the grading question* — but both readings are kept per
line, because `no` lines the model calls DERIVABLE are exactly the
commitments a repaired contract should state explicitly.

**author `yes`/`doc`/`partially` → model ARBITRARY: 54 of 614 annotated-
derivable lines (8.8%).** The author was systematically optimistic about edge
literals. The pattern is consistent: the commitment's *shape* is derivable but
its *literal* lives nowhere — exact `"can't encode <type> as <ct>"` wording,
an undocumented reparse-once retry, a `+json` suffix-doubling guard, nil-
encoder-on-bad-declared-type. The model's `ARBITRARY + grade-shape-only` is
the right call: grade the derivable shape, never the arbitrary literal. I read
~25 of these; the model is right in every case where the literal is quoted in
the commitment, defensible-but-arguable in a few where the reason is "not
stated in surviving docs" (e.g. `client-go-backoffer` L13 `Reset`/`String`
semantics — plausibly under-documented; treat as true arbitrary).

## Sharp case: goa-httpencoding

Author flagged 1 of 13 lines `no`. The contract audit found 18 missing
commitments — the DETAILS under-specifies. The gate finds the other half of
the problem: 5 of the 13 *stated* commitments carry arbitrary content.

| line | author | model | why the model is right |
|---|---|---|---|
| L4 nil encoder on unparseable declared ct | no | ARBITRARY | undocumented anywhere → shape-only |
| L5 reparse-once then JSON on Accept miss | partially | ARBITRARY | retry rule lives nowhere; JSON fallback does → shape-only |
| L9 `+json`/`+xml` suffix guard on SetContentType | doc | ARBITRARY | docs state the append rule, not the don't-double-append guard → shape-only |
| L10/L11 exact `can't encode/decode` literals | partially | ARBITRARY | accepted types derivable, literal wording is not → shape-only |
| L1 missing-vs-unlisted Content-Type collapse | doc | COUNTER | doc comment lumps both to JSON; bug report splits them — the comment is the discoverable-but-surprising rule → grade |
| L3 declared ct beats Accept, raw string written | doc | COUNTER | `ContentTypeKey` priority discoverable; naive Accept-first reading wrong → grade |

Net: the author's annotation catches the blatant case and misses every case
where a derivable shape wraps an arbitrary literal. The model's two-axis call
(kind × action) is strictly more informative for grading.

## Outcome validation: does ARBITRARY predict double-failure?

Joined to Harbor trial verdicts (family fails a level iff all scored trials
are 0; null rewards excluded). Coverage: 76 of 170 units have L0 verdicts,
48 have L2, 48 have both. Among them **8 double-failures = 16.7%**, versus the
global 38/125 = **30.4% base rate** (DETAILS units are newer batch-2 work and
intrinsically less double-fail-prone).

| predictor | flagged | tp | precision | recall |
|---|---:|---:|---:|---:|
| ≥1 ARBITRARY | 35 | 6 | 0.171 | 0.75 |
| ≥2 ARBITRARY | 22 | 4 | 0.182 | 0.50 |
| >half ARBITRARY | 3 | 1 | 0.333 | 0.125 |
| ≥1 drop | 12 | 1 | 0.083 | 0.125 |
| ≥1 drop or shape-only | 35 | 6 | 0.171 | 0.75 |
| verdict = low-discrimination | 3 | 1 | 0.333 | 0.125 |

**No predictor beats the base rate.** The best precision (0.333) sits on
n=3 flags — noise — and `any_ARBITRARY` (0.171) is at the subset's own 16.7%
rate and well under the global 30.4%. The gate is **advisory**, the same
landing as three of the four previous gates.

Why it does not predict: arbitrary commitments are only one failure cause —
2 of the 8 double-fails (`gin-enginecfg`, `gin-negotiate`) carry *zero*
arbitrary lines; their solvers failed on genuinely derivable commitments. And
73% of joined units have ≥1 arbitrary line, so the flag barely discriminates.
A weak positive signal remains: 2 of the 3 low-discrimination units with L2
data fail L0 then pass L2 (`gin-recoverymw`, `kops-hashparse`) — the signature
of "information restored at the higher rung".

The enforcement value does not depend on the outcome metric: 245 assertions
would otherwise grade literals no solver can produce, at every rung. That is a
fairness fix regardless of precision against this particular endpoint.

## Per-unit drop / shape-only list

`outputs/details_gate.jsonl` is the source of truth (46 dropped lines across
37 units; 199 shape-only lines across 100 units; 114 units affected either
way). Largest drops:
`gin-colorfmt` (3), `client-go-mvccread` (2), `goa-mappedattr` (2),
`go-github-projectsjson` (2), `helm-sympath` (2), `kops-difftext` (2),
`kops-fspath` (2), `nats-server-cronparse` (2). Every dropped line is an
ARBITRARY commitment with no derivable shape left — test scaffolding,
undocumented fallback branches, exact literals with no anchor.

## Cost per unit

One `cursor-agent` Composer call per unit (`composer-2.5`), ~2.2k prompt
tokens average (median 8.3k chars, max 13.5k), ~35–60 s wall each. 170 calls
total, ~$0 marginal beyond the existing subscription, ~35 min at workers=3.
Cache key `detailsgate/v1/<family>/<sha256(prompt)[:24]>` makes re-runs free.

## Known approximations

- api.md is not literally solver-visible at L0 (the solver sees instruction.md
  + the excised tree). It is included as a bounded stand-in for the exported
  surface the solver does see. Where a DERIVABLE reason cites api.md for
  something the tree does not show, the judgement errs lenient — the
  `no → DERIVABLE` flow above may overstate by a few percent.
- The gate judges lines as written; it does not check whether DETAILS.md
  *omits* commitments (the contract-audit axis). The two are complementary:
  httpencoding needed both.

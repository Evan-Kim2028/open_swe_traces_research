# Scope: two independent difficulty curves per unit

## The problem

The ladder is the difficulty measurement. "Fails at L4, flips at L5" is a claim about one
agent's competence curve — the rung at which a given model stops needing more help.

Every reader of that curve currently merges the solvers. `trial_ledger.ledger()` returns
`base -> rung -> [rewards]` with Composer's and Devin's verdicts in the same bucket, and
`escalate.next_rung` reads it. So if Devin passes L3 on a unit Composer has been climbing,
the ladder says "flips at L3" regardless of who did it, and Composer is never asked the
question. The rung recorded is not where the unit gets easy for a model; it is where the
*stronger of two models* happened to take a turn.

Three units reached that state (`cgenc`, `reflectfmt` with a Devin L4 on a Composer
ladder; `fideps` the reverse). The immediate patch was `solver_match`'s ladder-ownership
rule: L3+ goes only to whoever already owns the unit's escalation rungs. That stops the
mixing, but it does so by making the ladder exclusive — one model per unit, first come.
It buys correctness by throwing away the second curve.

## What we want instead

Both solvers may climb the same unit, each building its own ladder, scored separately.
Two curves per unit instead of one, which is strictly more information: it tells us
whether a unit is hard *for agents* or hard *for this agent*, which is the question the
whole dataset exists to answer and currently cannot.

It costs no money. Devin is free and structurally starved — the single-solver rule leaves
it few units it may certify, which is why it sits at cap with idle headroom.

## The change

`ledger_by_solver()` already exists and already returns `base -> solver -> rung ->
[rewards]`. The work is not new bookkeeping; it is making the decision points read the
per-solver view instead of the merged one.

| # | file | change | risk |
|---|---|---|---|
| 1 | `trial_guard.py` | `decide(unit, per, solver=None)`. When `solver` is given, rungs L3+ are judged against THAT solver's ladder. L0 condemnation stays merged and model-agnostic (one solver solving from the bug report disqualifies the unit for everyone) — that asymmetry is deliberate and already documented. | medium — every caller must pass a solver or keep today's behaviour |
| 2 | `solver_match.py` | Retire the ladder-ownership rule added earlier tonight. With per-solver ladders, mixing is impossible by construction, so ownership only blocks the second curve. | low — strictly more permissive, and the thing it guarded is now structural |
| 3 | `sweep_seq.sh` | Pass `--solver` to `trial_guard.py` at both call sites; it already knows its agent. | low |
| 4 | `orchestrate.py` | `runnable_by` already loops per solver; pass that solver into `decide`. | low |
| 5 | `trial_ledger.py` | `certificates_by_solver()` alongside `certificates()`. Do not change `certificates()` — the dataset headline keeps its current meaning. | low — additive |
| 6 | `ladder_purity.py` | Becomes a coverage report ("how many units have two curves") rather than a contamination report. | low |

## Invariants that must not break

- **Condemnation stays model-agnostic.** Any solver passing L0 condemns the unit globally.
  Per-solver ladders must not resurrect a unit another model solved from the bug report.
- **The per-rung trial cap still binds**, now per solver per rung rather than per rung, so
  the cap does not silently double.
- **`certificates()` keeps its meaning** so the headline count stays comparable to every
  number reported so far.
- **Escalation stays stepwise** — one rung at a time, per solver.

## Order of work

1, then 3 and 4 together (guard is useless until callers pass a solver), then 2, then 5,
then 6. Verify after each: the existing 219 certificates must not move until step 5, and
step 5 only adds a view.

## Out of scope

Re-running the three already-mixed units. Once both solvers may climb freely, they
self-heal: each will fill in its own missing rungs as capacity allows.

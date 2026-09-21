# Stage 2: gapped contracts (L1) as the test of the `Inferable:` annotation

**Status: scoped 2026-09-21, not started. Blocked on authoring, not on trial budget.**

## The annotation nothing checks

Every line of a generated contract carries an `Inferable:` classification —
ARBITRARY, DERIVABLE or COUNTER. It decides what the contract is allowed to omit and
what the verifier treats as a fair expectation. It is asserted by the authoring model
and **is never validated against solver behaviour anywhere in the pipeline.**

L1 is the experiment that would validate it, and the machinery already exists.

## What L1 is

`ladder_deep.py` builds two rungs below the full contract:

```
A-2 -> L0   bug report only
A-1 -> L1   the full contract MINUS one commitment
A0  -> L2   the complete contract
```

An L1 task is the L2 contract with a single commitment pulled out, plus a *catcher
test* — a pre-existing, unmodified repo test that fails on the gapped reading:

```python
gap=Gap(
    omitted="After release, waiters are woken (the granted waiter proceeds).",
    discoverable="internal/latch/latch_test.go:136-137",
    catcher_test="TestWithConcurrency",
    catcher_file="internal/latch/scheduler_test.go",
)
```

A pass means the solver recovered the omitted commitment from the code. That is
exactly the claim `Inferable: DERIVABLE` makes.

## Why it is not on the certification path

Three independent reasons, each sufficient:

1. **Different hidden suite.** `_hidden_for(gapped=True)` appends the catcher tests to
   the base suite. An L1 reward and an L2 reward are not scored against the same
   assertions, so they cannot sit on one flip curve.
2. **Doubly confounded.** L1 has less affordance than L2 *and* more tests to pass. A
   drop from L2 to L1 cannot be attributed to the missing commitment alone.
3. **Multi-valued.** A rung is one point; L1 is one point per gap. The same unit flips
   opposite ways depending on which commitment is pulled — see the table below.

A certificate stays "fails L0, passes at L2 or above". L1 does not participate.

## What the 16 existing verdicts show

Two gaps per unit across 8 units, one trial each:

| unit | gap `binding` | gap `other` |
|---|---|---|
| `gin-jsonrenders` | fail | PASS |
| `gin-streamrenders` | fail | fail |
| `kops-addonparse` | PASS | PASS |
| `kops-assetsremap` | PASS | fail |
| `kops-issuecert` | fail | PASS |
| `kops-memfs` | PASS | PASS |
| `kops-oidcdisc` | fail | PASS |
| `kops-templater` | fail | PASS |

**9 of 16 recovered.** Five of the eight units split — the same unit, the same
solver, opposite outcomes depending on which commitment was removed. That split is
the finding: recoverability is a property of the *commitment*, not of the unit. It is
also precisely what the `Inferable:` annotation claims to predict, and nobody has
checked whether it does.

## The experiment

Every gap in `ladder_deep.py` is DERIVABLE **by construction** — `discoverable` is a
file:line, so the answer is in the tree. That makes the current 9/16 uninterpretable
as a test of the annotation: there is no contrasting arm.

The falsifiable version needs both arms on the same units:

| arm | commitment pulled | prediction |
|---|---|---|
| A | one the generator labelled DERIVABLE | solver recovers it — L1 passes |
| B | one it labelled ARBITRARY | solver cannot — L1 fails |

If A and B pass at the same rate, the annotation carries no information and every
decision resting on it needs revisiting — including which lines a contract may omit
and what the B7 instruction ceiling is protecting.

## Why it is blocked on authoring

`ladder_deep.py` is 932 lines holding **4 hand-written `DeepUnit` entries**, 4
hand-written `Gap` specs and 8 hardcoded A1/A2 instruction constants. There is no gap
generator in the tree. Extending L1 across the dataset means, per unit, choosing which
commitment to pull, locating where it is recoverable, and naming a pre-existing test
that catches the wrong reading.

That is generation work, not trial work. No trial budget shortens it.

## What stage 2 needs

1. A gap generator: given a contract with `Inferable:` annotations and a hidden suite,
   emit an L1 variant per commitment class, with the catcher test located
   automatically rather than by hand.
2. A verifier rule that an L1 task's catcher test is pre-existing and unmodified —
   the property the whole construction rests on, currently guaranteed only by the
   author's care.
3. Both arms trialled on the same units, ≥3 attempts per arm.

## Related

- `KNOWN_LIMITATION_cross_repo.md` — the other parked construction
- `PAPER_OUTLINE_stage1.md` — stage 1 is Go-only, L0/L2 certification plus L3-L6
  escalation; L1 is deliberately out of scope
- `trial_ledger.py` carries `variant` alongside `rung` so the two gaps stay
  distinguishable; before that fix both collapsed to rung "1" and `max()` read
  "passed L1" if either gap passed

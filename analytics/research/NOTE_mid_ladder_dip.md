# Side note: one unit where the middle of the ladder looked worse than the bottom

**Status: not a finding. One unit, one solver, and a plausible story, which is exactly the
combination that turns into nothing.** Recorded so it is not rediscovered from scratch, and so
the reasons to doubt it are written down next to it.

## What was seen

Five cases turned up where a solver passed a rung and failed a higher one, which the ladder's
containment argument says should not happen. Four of them evaporated on inspection:

| unit | solver | why it does not count |
|---|---|---|
| `reflectfmt` | devin | L3 1/1, L4 0/1. One trial each. Its L3 also errored once before passing. |
| `fideps` | devin | L2 1/1, L4 0/1. One trial each, L2's first attempt errored. |
| `renamedet` | composer | L2 1/2, L3 0/1, L4 0/1. Same shape as below but n<=2. |
| `helm-tlsutil` | devin | L5 1/1, L6 0/1, and the L6 trial spent **0.2M tokens** where a real attempt on that unit runs 1-5M. A degenerate trial, not a failure. |

One did not:

    helm-repindex, composer:  L2 6/31 (19%)   L3 0/12   L4 0/9   L5 9/9   L6 6/6

`P(0 passes in 21 | p=0.19) = 0.011`.

## What each rung actually adds here

The instruction text is IDENTICAL from L3 through L6 on this unit -- 445 words. The affordance
is in the tree, not the prose:

| rung | added | composer |
|---|---|---|
| L2 | prose contract, 405 words | 6/31 |
| L3 | hidden test NAMES in the prompt (445 words) | 0/12 |
| L4 | `l4_exported_api.go`, the exported signature stubs | 0/9 |
| L5 | `repindex_bb_prop_test.go`, the real test body, into the tree | 9/9 |
| L6 | (same file count as L5) | 6/6 |

## The story, which is the part to distrust most

The hidden suite is a black-box PROPERTY test -- its header reads "hidden black-box property
suite ... Contract -> property coverage table". For a property test, names like `TestRIAddSort`
and `TestRIMerge` say almost nothing about the properties being checked, and exported signatures
say nothing either. Only the body reveals them, which is why L5 is decisive.

That gives a mechanism for L3 and L4 adding nothing. It does NOT explain them being WORSE than
L2, and that is the whole claim. The only account on offer is that the agent anchors on test
names and implements what the names imply instead of what the contract states, displacing prose
that was working one time in five. That is a story built to fit one curve.

## Reasons it is probably nothing

- **One unit.** The four other candidates were noise or a degenerate trial. A single unit with a
  tidy mechanism is how a pipeline generates false findings, not how it finds true ones.
- **One solver.** Composer only. Devin has no L3/L4 evidence on this unit at all.
- **The p-value pools L3 with L4**, which assumes both are worse -- the thing being tested. L3
  alone is 0/12 against 19%, `p = 0.081`.
- **The L2 comparison may not be like-for-like.** helm-repindex's L2 source tree has been
  reclaimed (0 files on disk), so there is no way to confirm its tree was otherwise identical to
  L3's.
- **19% is a strange baseline.** A unit that passes its certifying rung one time in five is
  already unusual, and 31 trials at L2 means it was reopened repeatedly -- contract repairs,
  re-screens. A unit with that history is not a clean instrument for anything.
- **Cap 1 makes this harder to settle, not easier.** Per-cell repeats are now refused, so
  nothing will accumulate more evidence on these cells without a deliberate roster entry.

## What would make it real

Property-based suites are identifiable from the hidden test files (`_bb_prop_test.go`, `rapid`,
`quick.Check`). If the effect exists, L3 and L4 should underperform L2 on property-suite units
specifically and not on example-based ones. That is a comparison across a population, needs both
solvers, and needs several trials per cell -- so it is a deliberate experiment, not something to
read out of the trials already banked.

Until then: one unit, one solver, and a story.

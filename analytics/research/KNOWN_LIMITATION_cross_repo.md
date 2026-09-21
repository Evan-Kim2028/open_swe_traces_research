# Known limitation: cross-repo task construction

**Status: parked 2026-09-21. Not a bug to fix — a property of the construction. Revisit
only with a different construction, not a different selection rule.**

## The construction

Excise a function from a dependency library L, leave its consumer C intact, and grade the
solver on C's public behaviour. The intuition is that a dependency boundary makes the task
harder: the solver must reconstruct L without seeing it.

## What we measured

| round | rule | trialled | solved at L0 | certified |
|---|---|---|---|---|
| XREPO (1) | any dependency | 23 | 19 (83%) | 4 (17%) |
| XREPO2 (2) | only behaviour that cannot be recalled | 14 | **12 of 14 (86%)** | 2 pending L2 |
| dataset-wide | in-repo excision | 277 | 41% | 48% |

Round 1's post-mortem blamed *memorised famous utilities*: the solver had already seen
the dependency, so the boundary added nothing. Round 2 applied a corrected selection rule
— excise only behaviour inferable from call sites, never recallable: obscure project
forks (`deps/errors`, `deps/billy`, `deps/jwt`, `deps/nuid`, helm's securejoin), with
arbitrary-but-consistent rules (`%+v` error layout, MVCC ordering, NUID increment bands,
JWT wire framing, path rebasing).

**Round 2 did not improve on round 1 — it was marginally worse: 86% solved at L0 (12 of
14, complete), against round 1's 83%.** Only `memfsfile` and `securejoin` resisted. The corrected rule changed which code was cut and left the outcome
identical. Only `memfsfile` and `securejoin` resisted.

## Why — the mechanism

The failure is not *what* gets excised. When L is cut and C is left intact, **C's call
sites specify L's contract**. A solver reads how the function is called — argument shapes,
orderings, what the consumer asserts afterwards — and reconstructs it. Recall never enters
into it, so making the target unrecallable cannot help.

That is inherent to the construction: the consumer is simultaneously the thing being
graded and a specification of the excised code. In-repo excision does not have this
property, because the excised code's callers are usually excised or unexercised with it.

## The second finding: authoring-time difficulty judgement is not predictive

The round-2 verification report stated "14 of 15 should resist an L0 solve" and named five
units as *strongest resistance*. **Four of those five were solved from the bug report
alone** (`godifflines`, `memdborder`, `strkey`, `errtrace`).

Treat any authoring-time or verification-time claim of "this will be hard" as a hypothesis,
never as grounds to skip L0 screening. L0 on Composer is the cheapest measurement we have
(~0.53x a hard trial) and it is the only thing that has ever been right about difficulty.

## What would be worth trying later

- Excise L **and** blind the consumer: remove or stub the call sites that specify it, so
  the contract must come from the prose rather than from usage.
- Grade on behaviour the consumer never exercises, so call sites under-specify L.
- Accept cross-repo as a *source of units* but screen them at L0 like anything else, with
  no expectation that the boundary confers difficulty.

## Operational note

`XREPO`, `XREPO2`, `XREPO20` are parked in the Devin manifest. Nothing in
`pipeline_autogen` or `orchestrate` auto-queues cross-repo work, so parking the rows is
sufficient; no code change is needed to keep it off.

`memfsfile` and `securejoin` were the two round-2 units that resisted L0. Both remain
staged at L2 in `sweep_xrepo2_L2` and the trial guard clears them. They were left
untrialled deliberately: two trials would tell us whether the corrected rule can
produce any certified unit at all, which is the obvious first step of the follow-up
work rather than part of the parked round. `osbound` was excluded before trial: its cheat patch passes the
consumer suite (an A3 violation), which is itself an instance of the mechanism above.

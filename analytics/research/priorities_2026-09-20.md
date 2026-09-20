# What to prioritise next — 2026-09-20

## 0. The go-github failure is confounded, and go-git disambiguates it for free

go-github: 10 of 10 units failed L0 *and* L2. Two competing explanations, both plausible:

| hypothesis | evidence for |
|---|---|
| **cold start** — no exemplars to pattern-match | go-github had **0** prior authored units; helm/kops/gin/client-go/goa each had **10** |
| **domain** — JSON serialisation is intrinsically hard to specify in prose | 9 of its 14 unit names are serialisation closures: `rulesetjson`, `projectsjson`, `pubkeyjson`, `treeentryjson`, `ndjsonmetrics`, `customprop`, `auditentry`, `webhooksig`, `eventdispatch`. Field ordering, `omitempty`, null-vs-absent and tag semantics are exactly the commitments prose describes badly |

**go-git separates them at no extra cost.** It also had 0 prior units (cold) but its closures are parsing, refs and
packfile decoding, not serialisation. If go-git's batch also collapses → cold start. If it behaves like kops → the
problem is the domain, and JSON/serialisation closures should be excluded from authoring the way we already exclude
cross-repo for difficulty. Screen it as soon as its verifier lands, and audit before counting.

## 1. Fix the contract defect at its source: derive coverage rows FROM the tests

40% of the first bank misdescribes gold; 7 of 7 audited double-failures were contract defects; go-github may be 10
more. The cause is structural and unchanged since the first batch:

```
author  →  contract.md   (prose written from a READING of gold)
verifier →  hidden tests (assertions written from gold itself)
            ^ nothing ever reconciles these two
```

**Proposal — invert the dependency.** Keep the author/verifier split (it exists to stop instruction leakage), but add
a third, cheap pass that writes the coverage table *from the hidden tests plus gold*, after both exist:

```
author   → DETAILS.md (the commitments) + bugreport.md (L0)
verifier → one property test per commitment
reconciler → contract.md coverage rows, derived from (test assertion, gold hunk)
```

Every row then comes from an assertion, so **"missing" becomes structurally impossible and "false" becomes unlikely**.
A13 stays as the backstop. This is the single highest-value change available: it attacks the dominant error mode
rather than detecting it after trials are spent.

## 2. Build an instrument for false negatives

We have none, and it is the biggest untested gap. A unit that passes L0 is discarded as easy, but the lie experiment
proved the contract is load-bearing — so an over-explicit bug report would make a genuinely hard unit look easy, and
we would never know. **Cheap instrument:** for units that pass L0, re-run with the expected-vs-got section stripped
from `bugreport.md` (keeping the symptom and repro, still B6-legal). Still passes → genuinely easy, correctly
discarded. Now fails → the L0 report was leaking and we threw away a hard unit. ~20 trials on already-packaged units.

## 3. Add the families that can produce a capability limit

Zero unsolvable units in ~400 trials: the bank measures information, not ability, and cannot discriminate between
strong models. Every unit so far is single-site reimplementation. The rulebook already defines the families that
would change this and none were used tonight: **A6 multi-site** (fixing any one site alone still fails), **A9 dynamic
gates** (race-free under `-race`, or a performance threshold derived from gold). Those are hard to *implement*, not
merely hard to *infer* — a different axis from the recipe's.

## 4. Cold-start robustness (cheap mitigations, pending the §0 result)

- Give a cold author 2–3 exemplar units from a *different* repo (artifacts only, not the source tree).
- Make A13 a **mandatory gate** for repos with fewer than ~3 prior units, accepting its 13/30 false-alarm rate —
  there, the prior on defects is far worse than the cost of adjudicating a false alarm.
- Require the survey step to name, per closure, which commitments are *prose-expressible*; drop closures whose
  behaviour is mostly implicit (the JSON-tag case).

## Status: §2 (false-negative instrument) is RUNNING as of 09:30

7 units that PASSED at L0 (helm chartmeta/getterdispatch/urlutil/relsplit + the 3 gin passes) re-packaged as
`-L0fn` with the expected-vs-got detail stripped from `bugreport.md` — symptom and repro kept, so the B6 floor holds.
Job `sweep_fn`, k=1. Reading: still passes → genuinely easy, correctly discarded. Now fails → the original L0 report
was leaking and we discarded a hard unit, which means the pipeline has been under-counting hard units all along.

### Operational lesson from the cleanup (09:35)

The first false-negative run errored on all 7 units: my disk cleanup had deleted
`oswt-VF*/experiments/pipeline/tasks_*/*/*/environment/src`, and the probe dirs were copied from those worktrees
*after* the deletion, so they shipped without a source tree. Restored from the already-screened
`sweep_<repo>2/*/environment/src` copies and relaunched.

**Rule:** when deleting regenerable source trees, either (a) delete only from worktrees whose batch is fully screened
AND whose dirs will not be copied again, or (b) re-materialise with `scripts/materialize_tasks.py` before packaging
anything new from that worktree. Preflight would have caught this (a tree with no source cannot build), which is
another argument for running the gate on every staged batch rather than trusting a copy.

## Verifier build-out: settled design (2026-09-20 14:00)

The decision record is `analytics/research/verifier_rules.md`, section "Verifier design: the two defect
families". Summary for anyone picking this up:

- **Two independent defect families.** *Packaging* (the suite does not cover the excision) is caught by
  reachability — 53% precision at ≥2 orphan functions against a 39% base rate, 2.7× separation in the mean.
  *Contract* (prose contradicts or omits an assertion) is caught only by constructing a second implementation
  from the contract alone. Neither instrument sees the other's family.
- **Static text linting is a dead end for both.** Measured at 40% and 40% against a 39% base rate. Recorded so
  nobody rebuilds it.
- **Gate order is cheapest-first**: deterministic linter → reachability → shadow implementation → A13 →
  in-image preflight → trial. The first four are free; a trial is 2.18M tokens, and we spent 9.5 trials per
  certified-hard unit, most of it discovering defects rather than measuring difficulty.
- **Weak models suffice** once the gates exist. Strong models were compensating for the absence of a check, not
  doing something only they can do.

In flight: `closure_SHADOW` (shadow gate), `closure_CGCOV` (reachability, measured the same way the linter was),
`closure_RCFIX` (three precision fixes to the reconciler, then re-derive the goa cohort that flipped 3 of 12).

## Disk policy, corrected (2026-09-20 14:25)

The old rule — prune `environment/src` from any worktree without a running session, every 30 minutes — was wrong
and bit three times in one day: nats-server, go-git, and the client-go/goa tail escalation. Each time it deleted
sources for a batch that was about to be trialled. `scripts/ops/regen_env_src.sh` recovered every one from the
base image plus a reverse-applied `gold.patch`, so nothing was lost permanently, but it cost roughly 40 minutes of
wall clock for disk we were never short of.

`scripts/ops/prune_worktrees.sh` replaces it:

- **does nothing above 100G free** (currently 271G — accumulation is fine)
- never prunes a worktree with a running agent session
- **never prunes a batch that still owes trials** — any family that failed L0 and has no L2 verdict, matched
  across every stem spelling, since repo prefixes contain hyphens (`client-go-doactionbatches` → `doactionbatches`,
  not `go-doactionbatches`; that bug alone cost a staging cycle)
- prunes least-recently-modified first and stops the moment free space clears the threshold
- never touches `ladder-base:*` images or `experiments/pipeline/repos/*/src`

The general lesson: a cleanup that runs on a timer rather than on pressure will eventually delete something in
use, and the cost of being wrong is asymmetric — accumulation is cheap, recovery is not.

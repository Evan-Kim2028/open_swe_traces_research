# The repair loop — derive, shadow, attribute, repair, to a fixed point

Date: 2026-09-20. Code: `src/openswe_traces/gate/repair_loop.py`,
`scripts/repair_loop.py` (thin CLI). Log: `outputs/REPAIRLOOP.log`.
Per-unit records: `outputs/repair_loop/rounds/<unit>.json`,
summary `outputs/repair_loop/results.jsonl`. Staged units:
`experiments/dose_response/sweep_loop/<name>-L2loop/`
(never overwrites an original).

## What it is

The automated version of the five-round hand repair on `helm-repindex`:

```
reconcile  → coverage rows derived from hidden assertions   (already excellent)
shadow     → implement from the contract ALONE, run the suite
attribute  → for each failing assertion: does the contract state that commitment?
repair     → unstated/contradicted: add exactly that row. stated: leave it —
             that is difficulty, not a defect.
repeat     → until no failing assertion is unstated, or the round budget hits
```

Per round the record keeps the contract diff, the shadow result per
assertion, the verdict+kind for every failure, literal budget
before/after, and request/token counts. Stated failures are never
repaired; ambiguous attributions never trigger repairs.

## Kind routing — never repair uniformly

Every unstated/contradicted failure is classified by where the answer
lives:

| kind | answer lives | action |
|---|---|---|
| ARBITRARY | nowhere — the TEST over-specifies | record as a test defect; never written into the contract; flags the unit low-discrimination |
| DERIVABLE | in the repository | repair the contract |
| COUNTER | in the repo, but the obvious reading is wrong | repair the contract; flag the unit high-value |

An attribution that returns `unstated`/`contradicted` without a valid
kind is downgraded to `ambiguous` — malformed judgments never repair.

## Safeguards

- **A3-SURFACE per round**: distinct quoted-literal count must not rise;
  grounded-literal fraction must not fall (same counting as
  `scripts/ops/task_lint.py`). Rejections feed the diff back to the
  repair call: which literals were introduced and which existing ones
  are absent from every test source and safe to drop.
- **B-family**: no hidden test names, no gold symbol/file names
  (denylist from the gold patch), no `"the call"` scrub residue, no
  class-direction ordering ("pre-releases sort below releases") — the
  reconciler's `rewrite_directional_ordering` also rewrites direction
  prose pairwise deterministically.
- **Ungrounded examples**: `strip_ungrounded_example_clauses` excises
  invented worked examples; survivors reject the round.
- **Fresh-cheat preflight**: after the loop, the cheat patch is
  REGENERATED from the final contract and the unit must still show
  bare=0, gold=1, cheat=0. A cheat pass only counts as an A3 failure
  when the cheat patch reuses contract literals (`cheat_passes`); a
  cheat that passes while quoting none is a capability pass
  (`cheat_general_pass`) — composer-2.5 asked to "cheat" often just
  writes a correct implementation.
- **Work-tree integrity** (found mid-run): `cursor-agent -p` is a full
  tool-using agent, not a completion API. Calls run in an empty temp
  cwd, yet one generation still located and rewrote a work copy's stub
  file (dupexpr, observed 16:08; the source unit was untouched). Calls
  now run `--mode ask` (read-only tools) — `--sandbox` is unavailable
  on this host — and a `TreeGuard` hashes `environment/` + `tests/`
  (minus cheat.patch) around every LLM call: any mutation is logged,
  restored from the source unit, and counted in the unit's record.
- **Resume-safe**: terminal units skip, errored units retry, the work
  copy is rebuilt from the source at every run (round 0 is always the
  original contract — a stale work dir once faked a one-round
  "convergence" on the already-repaired contract).

## Results — all 15 units

60 composer requests, ~2.00M tokens, 83 unit-minutes (4 parallel
workers ≈ 25 min wall). Terminal outcomes:

| outcome | n | units |
|---|---|---|
| converged_pass | 11 | repindex, page, reqidgen, retrypolicy, dupexpr, svcerror, traceopts, exprhash, namescope, mappedattr, sampler |
| converged_fixed_point | 2 | httpclienterr, flshared |
| inconclusive (fail_build) | 2 | httpencoding, httpmux |

## helm-repindex — convergence proof

Source: `sweep_climb_L3/helm-repindex-L3` (original, 0/3 at L0/L2/L3/L4).
Hand reference: `sweep_rcfix5/helm-repindex-L2rc5` (2/3 at L2).

**Round 0** (original contract, sha `b186d3ab`):

- shadow: `fail_assert` — TestRIContractTable, TestRIAddSort, TestRIGet,
  TestRIMerge
- attribution: 4 repairable / 0 arbitrary / 0 stated / 0 ambiguous:
  - `TestRIContractTable` — **contradicted / COUNTER**: contract said an
    absolute-URL filename is used as-is; suite requires base-URL +
    basename join even when the filename looks like a URL.
  - `TestRIAddSort` — **contradicted / COUNTER**: contract said re-adding
    replaces; suite requires appending a second entry.
  - `TestRIGet` — unstated / DERIVABLE: `Has` is true exactly when `Get`
    with the same query succeeds.
  - `TestRIMerge` — unstated / DERIVABLE: merge dedups on (name,
    semver-equal version ignoring build metadata); receiver keeps its
    entry on collision.
- repair: try 0 rejected (hidden test names leaked into the head — the
  model copied the L3 tail's name list); try 1 applied.
  literals 22 → 13, grounded 32% → 100%.

**Round 1**: shadow `pass` → `converged_pass`. 2 rounds, 3 requests,
232k tokens. Preflight bare=0 gold=1 cheat=0 → **ok**.
Staged: `sweep_loop/helm-repindex-L2loop/`.

### Rows found vs the hand-repaired contract

The loop's contract states the 4 repaired commitments in prose and in
the coverage table, plus the pairwise-ordering examples the direction
check requires. Against rc5's 29 rows:

- **Fully stated (≈12 rows)**: basename-join for every filename shape
  (#2), append-on-duplicate-add (#4), descending semver per name (#7),
  range→highest published (#9), presence≡lookup incl. unknown-name
  false (#12), merge dedup on name+semver-ignoring-build-metadata
  (#13), collision keeps receiver (#14), non-colliding appended (#15),
  YAML+JSON round-trip and load sorts descending (#17, #24),
  duplicate/empty load tolerance (#18), pairwise pre-release ordering
  with no build-metadata role (#22).
- **Partially stated**: metadata carried alongside one resolved URL
  (#6), exact-vs-range dispatch implied but precedence unstated (#8),
  "a specific not-found error" without the two named error kinds
  (#10/#11), directory scan without the sha256-of-bytes digest rule
  (#16), round-trip without the nothing-collapsed guarantee (#28).
- **Unstated (~11 rows)**: v1 apiVersion + non-nil entry map (#1),
  trailing-slash neutrality (#3), empty-name error + unparseable
  version error + missing-apiVersion tolerated on add (#5), malformed
  archive skipped not fatal (#19), partial `1.2` query behaves as a
  range (#20), missing-apiVersion error on load (#21), bare lowercase
  hex digest (#23), empty-query semantics (#25/#27), only empty-version
  entries dropped on load (#29), duplicate keys fail load (#30).

The shadow passes the entire suite with those rows unstated — the
implementer derived them (semver edge cases are conventional, the
digest format is discoverable in-repo). **The loop converges on the
minimum-sufficient contract for the measured implementer; the hand
contract targets a weaker one.** Whether the loop's contract flips
real trials is for the parent's Composer runs — the staged unit is
ready at `sweep_loop/helm-repindex-L2loop/`.

## goa cohort — before/after

Baseline: `sweep_goa_L2` trials — traceopts 3/3, retrypolicy 3/3,
reqidgen 1/3; the other nine went 0/3 (36 trials, 7 passes).

| unit | L2 trials | loop outcome | rounds | repairs applied | preflight |
|---|---|---|---|---|---|
| dupexpr | 0/3 | converged_pass | 1 | none (r0 shadow passed) | ok |
| exprhash | 0/3 | converged_pass | 2 | 1 row — COUNTER contradicted: union hash variant encoding order | ok |
| httpclienterr | 0/3 | converged_fixed_point | 3 | 1 row r0 (COUNTER) + 8 rows r1 (DERIVABLE: constructor shapes, error traits, body restore) | ok |
| httpencoding | 0/3 | inconclusive | 1 | — | ok |
| httpmux | 0/3 | inconclusive | 1 | — | ok |
| mappedattr | 0/3 | converged_pass | 2 | 2 rows — COUNTER: map-panic rebind, merge replace-preserve | cheat_general_pass |
| namescope | 0/3 | converged_pass | 1 | none | **cheat_passes** |
| reqidgen | 1/3 | converged_pass | 2 | 1 row — DERIVABLE: zero-option defaults | cheat_general_pass |
| retrypolicy | 3/3 | converged_pass | 1 | none | cheat_general_pass |
| sampler | 0/3 | converged_pass | 1 | none | ok |
| svcerror | 0/3 | converged_pass | 3 | 1 row r0 + 3 rows r1 — DERIVABLE: error message templates, history semantics | ok |
| traceopts | 3/3 | converged_pass | 1 | none | ok |

Read: six of the nine 0/3 units already had contracts a strong
implementer passes (dupexpr, sampler, namescope passed r0; exprhash,
mappedattr, svcerror needed 1–4 rows; httpclienterr needed 9). Their
trial failures are mostly not missing-contract defects for this
implementer — either difficulty or harness effects. `httpclienterr`
and `flshared` reached a fixed point with residual *stated* failures —
difficulty, correctly left alone.

## bbolt — before/after

| unit | packaging flag | loop outcome | repairs | preflight |
|---|---|---|---|---|
| page-L2 | 11/43 orphan funcs | converged_pass (r0) | none needed | ok |
| flshared-L2 | 3 orphans | converged_fixed_point | 1 row — COUNTER contradicted: release-extent rule | **cheat_passes** |

The orphan counts did not matter for the shadow: page's uncovered
functions are simply unexercised by the suite. flshared repaired one
contradicted commitment and stopped with one stated failure
(difficulty) and one ambiguous.

## A3 re-check — two real leaks, both pre-existing

| unit | verdict | reused literals | repair touched contract? |
|---|---|---|---|
| namescope | cheat_passes | `Fork`, `Freeze`, `HashedUnique`, `PeekUnique`, `Foo2`, … | no — r0 pass, contract is ORIGINAL |
| flshared | cheat_passes | `0xFFFF`, `Init`, `allocs`, `cache`, `pending`, `read` | yes — repair dropped literals 14→10 |

Both contracts carry worked examples with literal input→output pairs
(`PeekUnique("Foo")` → `"Foo2"`; `0xFFFF` count-slot encoding) plus
backticked symbol names — original-contract surface, not repair-added.
task_lint flags namescope B7 (symbol names) + A3 (41 literals, 29%
grounded). The loop only ADDS rows; redacting stated surface is a
different repair mode the loop deliberately does not perform — these
are flagged for author-side contract edits.

The other cheat passes (`reqidgen`, `retrypolicy`, `mappedattr`,
`helm-repindex` run-2) reused ZERO contract literals — capability
passes, recorded as `cheat_general_pass`.

## Costs

| unit | requests | tokens | seconds | repairs |
|---|---|---|---|---|
| helm-repindex-L3 | 3 | 232,113 | 459 | 4 rows |
| exprhash-L2 | 4 | 207,260 | 765 | 1 |
| flshared-L2 | 8 | 227,271 | 471 | 1 |
| httpclienterr-L2 | 6 | 274,800 | 253 | 9 |
| svcerror-L2 | 8 | 186,942 | 477 | 4 |
| mappedattr-L2 | 5 | 174,815 | 367 | 2 |
| httpencoding-L2 | 4 | 116,878 | 308 | — |
| httpmux-L2 | 4 | 110,390 | 384 | — |
| reqidgen-L2 | 5 | 96,000 | 322 | 1 |
| namescope-L2 | 2 | 86,125 | 273 | 0 |
| dupexpr-L2 | 2 | 82,671 | 165 | 0 |
| page-L2 | 2 | 61,124 | 217 | 0 |
| traceopts-L2 | 3 | 56,564 | 187 | 0 |
| retrypolicy-L2 | 2 | 42,643 | 161 | 0 |
| sampler-L2 | 2 | 43,211 | 169 | 0 |
| **total** | **60** | **1,998,807** | **5,010** | 22 rows |

Compare: the five-round hand repair on repindex cost ~15 trials ≈ 33M
tokens. The loop's whole 15-unit run cost ~2M. Aborted repindex runs
during debugging added ~15 requests / ~0.5M tokens (inconclusive →
repair_rejected → stale-workdir convergence, all recorded in
REPAIRLOOP.log); the deleted rows are reconstructible from the log.

## Ambiguity and failure account

- **Ambiguous attributions**: 1 total — flshared r0
  `TestDetail06_ReleaseIdempotent` ("unclear which check failed").
  Left alone; no repair. Correct handling.
- **Stated failures left alone (difficulty)**: flshared r1
  `TestDetail08_WriteSortedAndExtendedCount`; httpclienterr r2 —
  `TestDetail10_RequestBodyRestored`,
  `TestDetail12_ResponseBufferedAndStderrDump`,
  `TestDetail13_DumpFormat`. These are the loop working as designed.
- **Inconclusive**: httpencoding and httpmux — the shadow generator
  invented undefined package identifiers (`mediaJSON`, `mediaXML`,
  `goa`) across its generation + 2 compile-repair tries; nothing to
  attribute. A generation-compliance failure, not a contract defect.
- **ARBITRARY findings**: none across 15 units — every defect the judge
  saw was DERIVABLE or COUNTER. No unit was demoted low-discrimination.
- **Agent wrote into the work tree**: 1 observed incident (dupexpr
  r0, pre-ask-mode). The cached generation re-scored on a clean tree
  and still passed; TreeGuard now audits every call region.
- **Non-convergence**: none — every converged unit stopped inside 3
  rounds of a 6-round budget.

## Placeholder cleanup — "the call"

The symbol-scrub placeholder `the call` was replaced with descriptive
noun phrases — 18 files in this worktree. The bank-wide count of 73 was
not reproduced: `experiments/dose_response` is a symlink into the
parent repo and its hits were legitimate prose ("the call site",
"the caller", "remove the call"); vendored code comments account for
the bulk. Files changed:

- `experiments/harbor_nex/tasks_bigL0/batchcmds-obf-L2/instruction.md`
- `experiments/harbor_nex/tasks_bigL0/batchcmds-obf/_author/contract.md`
- `experiments/harbor_nex/tasks_bigL0_run/batchcmds-obf-L2/instruction.md`
- `experiments/pipeline/authored/gin/formmapping/_author/bugreport.md` — "or the call panics" → "or binding panics"
- `experiments/pipeline/authored/goa/errloc/_author/contract.md`
- `experiments/pipeline/authored/goa/jsonrpcwire/_author/bugreport.md` — "the call looks like a notification" → "the request looks like a notification"
- `experiments/pipeline/tasks_composerver/gin/_formmapping_skel/instruction.md`
- `experiments/pipeline/tasks_composerver/gin/formmapping-L0/instruction.md`
- `experiments/pipeline/tasks_composerver/goa/VERIFIER_BATCH.md` — "the call line" → "the call-site line"
- `experiments/pipeline/tasks_composerver/goa/_jsonrpcwire_skel/instruction.md`
- `experiments/pipeline/tasks_composerver/goa/errloc-L0/validation.json` — audit quote updated to match the fixed row
- `experiments/pipeline/tasks_composerver/goa/errloc-L2/instruction.md`
- `experiments/pipeline/tasks_composerver/goa/errloc-L2/validation.json`
- `experiments/pipeline/tasks_composerver/goa/errloc-L5/instruction.md`
- `experiments/pipeline/tasks_composerver/goa/errloc-L5/validation.json`
- `experiments/pipeline/tasks_composerver/goa/errloc-L6/instruction.md`
- `experiments/pipeline/tasks_composerver/goa/errloc-L6/validation.json`
- `experiments/pipeline/tasks_composerver/goa/jsonrpcwire-L0/instruction.md`

## Honest limitations

- **Shadow pass ≠ trial pass.** The shadow implementer is
  composer-2.5 — strong. Six goa units that went 0/3 in trials already
  pass shadow at r0: for this implementer their contracts were never
  the defect. The loop proves "sufficient for this implementer" — it
  does not maximize stated coverage, and it cannot see harness-level
  failure causes.
- **The loop only adds rows.** Contracts that over-disclose (namescope,
  flshared) need redaction, not repair; the loop flags them via the
  cheat verdict but never removes stated content.
- **Agentic generator.** Ask-mode makes calls read-only and TreeGuard
  audits writes, but a read-side channel is nominal: a determined agent
  could in principle locate and read repo files (including hidden
  tests) from an empty cwd. No such read was observed. Harder
  isolation (mount namespace or a network-only container) is the fix if
  it ever matters.
- **Attribution is one model.** composer-2.5 judges verdict and kind;
  malformed JSON degrades to `ambiguous`, but a confidently-wrong kind
  routes the wrong way. Every judgment is kept in the record for audit.
- **`converged_fixed_point` means "nothing left to repair"**, not
  "passes": httpclienterr and flshared retain stated failures —
  difficulty by construction. If the parent wants those flipped, the
  answer is a stronger shadow implementer, not more contract.

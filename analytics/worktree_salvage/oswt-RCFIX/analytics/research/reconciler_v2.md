# Reconciler v2: three precision fixes, re-proven on repindex + goa

Date: 2026-02-14 (job RCFIX, branch `rcfix`)

The reconciler (`src/openswe_traces/synth/reconcile.py`, prompt `reconcile-v1`)
recovers nearly all hidden commitments in one pass — recall was never the
problem. On `helm-repindex` its generated L2 contract scored 0/3 where the
hand-derived contract scored 2/3, and all three failures were predictable from
the generated text before the solver ever ran. v2 (`reconcile-v2`) targets the
three precision defects directly, post-processing the model's contract rather
than trusting the prompt to prevent them.

## The three fixes

### 1. Ordering is emitted pairwise, never as a direction

v1 text: `the call orders each chart's versions in descending semantic version
order (prereleases sort before releases of same core version)` — under
descending order this inverts the prerelease rule; the solver implemented it
faithfully and failed `beta not sorted desc` on 2 of 3 attempts.

`rewrite_directional_ordering` detects ordering clauses (sort/order/ascending/
descending language) and appends a pairwise law built from literals the hidden
suite actually contains: which of two concrete values comes first, and whether
build metadata participates in the comparison. The repindex contract now reads:

> semantic-version precedence is pairwise: `0.1.0` comes before `0.1.0-beta` —
> a pre-release follows only the release it belongs to; `0.1.0-beta` comes
> before `0.0.1` — a pre-release still precedes every version with a lower
> numeric core; build metadata does not participate in the comparison.

Both orderings are unambiguous under descending order because each pair is a
concrete before/after claim, not a rule whose polarity a reader must flip.

### 2. Worked examples must be grounded in suite literals

v1 text: `Get(name="app", version=">=1.0.0") with versions ["0.9.0", "1.0.0",
"1.2.0", "2.0.0-alpha"] → returns "2.0.0-alpha" (highest matching)` — invented,
and wrong about the semver library itself (a range constraint does not match a
pre-release); no assertion in the suite supports it.

`example_literals` extracts literal-looking tokens from each worked example
(quoted strings, version-looking barewords, filenames, URLs), rejecting the
prose spans that naive quote-pairing sweeps up. `ungrounded_literals` checks
each against the hidden-test source. `ground_examples` drops or rewrites
examples containing unsupported literals, and a self-check marks the whole unit
**unreconcilable** if an ungrounded literal survives — the unit fails loudly
instead of shipping a fabricated example. The self-check catches the exact
repindex `2.0.0-alpha` example in the regression test.

### 3. No bare "the call" for scrubbed symbols

v1 emitted `the call combines two indexes`, `the call walks a directory`, `the
call(name, version) is true…`. `symbol_noun_phrase` substitutes a descriptive
noun phrase derived from the scrubbed symbol: `merging combines two indexes`,
`the directory scan walks`, `presence-testing`. B7 still applies — the gold
symbol itself never appears; the generated repindex run scrubbed `Has`,
`IndexDirectory`, `LoadIndexFile`, `Merge`, `MustAdd`, `SortEntries`,
`WriteJSONFile`.

## helm-repindex: old generated vs v2 vs hand-derived

New contract staged at `experiments/dose_response/sweep_rc_auto3/helm-repindex-L2auto3/`
(60 lines, 7 coverage rows — one per hidden test function, same shape as v1).

Diff vs `sweep_rc_auto` (0/3):

| defect | v1 (0/3) | v2 |
|---|---|---|
| ordering | "prereleases sort before releases" — inverted under desc | pairwise `0.1.0`→`0.1.0-beta`→`0.0.1`, build metadata excluded |
| invented example | `Get(">=1.0.0") → 2.0.0-alpha` | dropped; every example literal traces to the suite |
| "the call" | 10 occurrences, no semantics | `sorting the entries`, `merging`, `the directory scan`, `presence-testing` |
| merge winner | "first-seen digest wins (A beats B)" — ambiguous which index is first | "the first index's entry (and its digest) wins" — matches receiving-index oracle |
| digest prefix | "sha256 digest" | "lowercase hex sha256 of the archive file contents" (the round-2 hand finding) |
| bad queries | "v1.2.3, 1.2 cause an error" — wrong | constraint list incl. `!=`, `>=0.1.0-beta`, hyphen ranges; `1.2` as range |

Diff vs the hand-derived 29-row contract (`sweep_rcfix5`, 2/3). v2 covers
27 of ~29 hand commitments. Missing/weak:

- the dedicated no-chart-name error for an unknown `Get` (hand row 10);
- the empty-query all-prerelease edge: hand says "`{*}` resolves to the
  highest non-pre-release and is an error when every version is a
  pre-release"; v2 says `Get empty query -> "*"` without the error case
  (hand round-4/5 finding);
- "stores metadata verbatim alongside exactly one resolved URL" (row 6) —
  implied, not stated.

v2 is *correct* where the hand contract itself was wrong in early rounds:
`Get empty -> "*"` (not an error), versionless entries dropped while versioned
siblings kept, duplicate top-level entry keys error.

## goa audit: do the authored L2 contracts contradict their own hidden suite?

Method: `scripts/ops/contract_vs_test.py` per unit — the L2 contract next to
every hidden oracle comment and `t.Fatal*`/`t.Error*` message. Each flagged
unit was confirmed against the test body, not just the message text.

| unit | contradicts? | evidence |
|---|---|---|
| dupexpr | no | per-kind copy, origin-collapse, bases-per-copy all match oracle |
| exprhash | **yes** | contract instructs clean name-sort ("the sort must compare the sorted copy's elements, not the original slice's"); `TestDetail04_UnionValuesSortedByName` pins the positional-comparator *artifact* order — scrambled until it differs from both insertion and clean-sorted order, then requires exact emission. A solver following the contract fails the emission check |
| httpclienterr | no | retryability classes, traits table, dump format all match |
| httpencoding | no | exact-vs-suffix asymmetry, raw-declared header, JSON fallbacks match |
| httperrresp | no | status heuristic order, XML element/children rules match |
| httpmux | **yes** | "every `{*name}` segment in the pattern is rewritten" + "names may contain only letters/digits/underscore" implies invalid braces are literals; `TestDetail10` requires `{*na-me}` to NOT act as catch-all yet still capture one segment under literal key `*na-me`, and `{na me}` to match and capture under `na me`. The degraded-capture rule is nowhere stated |
| importalias | no | priorities, freeze order, displaced-explicit rule all match |
| mappedattr | no | map/key mirrors, delete/merge/walk semantics match |
| namescope | no | `Name` = raw count+1 (even when taken), freeze panics, fork isolation match |
| reqidgen | no | option side-effects, limit-on-reuse-only, 8-char base64url match |
| retrypolicy | no | single retry, ctx-already-canceled returns endpoint error, ANDed traits, [50,150) wait all match |
| sampler | **yes** | "the first `sampleSize` calls all return true" + example `(2,5) → first 5 true`; `TestDetail06_RecomputeBoundary` requires `firstFalse == 49` for size 50 — the size-th call recomputes the rate *before* its draw and must be allowed (here required) to fail after a fast window |
| skipwriter | no | lazy pipe, once-init across Read/Close, error-surfacing, byte counting match |
| svcerror | no | constructor trait table, merge left-name-unless-"error", ANDed traits, history concat match |
| traceopts | ~ | same sampler overclaim ("first 5 `Sample()` calls return true" for size 5) but its own suite only asserts the first size−1 calls — not contradicted by its suite, contradicted by the same oracle as sampler |

3 hard contradictions of 15 units (exprhash, httpmux, sampler), plus the
traceopts near-miss. All three flipped-or-trialed units are in the 12-unit L2
cohort. Notably every contradiction is a case where the *authored* contract
states a clean rule and the hidden suite pins an implementation-artifact or an
edge-case boundary — exactly the class of defect the reconciler is supposed to
catch, since it derives the contract from the assertions rather than the
intended semantics.

## goa batch regeneration (reconcile-v2)

Staged at `experiments/dose_response/sweep_goa_rc/` as `<unit>-L2rc` trees
(L0 baselines untouched; `environment/src` for the 3 untrialed units grafted
from the authored L0 sweeps). 15 units, 27 LLM requests, all final contracts
`unreconcilable=False`.

| unit | rows | audit finding | v2 contract |
|---|---|---|---|
| dupexpr | 11 | no | consistent; copy-kind/origin-collapse rows grounded |
| exprhash | 10 | **yes** | names the artifact order: "compares each name with the original slice, not by pure name order" |
| httpclienterr | 13 | no | consistent; first sample self-checked ungrounded, fresh sample clean |
| httpencoding | 13 | no | consistent; first sample invented `vnd.x+custom`, self-check caught it; regenerated clean |
| httperrresp | 8 | no | consistent |
| httpmux | 10 | **yes** | states the degraded rule: "a brace is a wildcard only when its name is letters/digits/underscore; otherwise a literal path segment" |
| importalias | 12 | no | consistent; synthesized-name examples excised (all suite names are `vfName`-generated) |
| mappedattr | 12 | no | consistent |
| namescope | 12 | no | consistent; regenerated after rate-limit fallback + fabricated-name excision |
| reqidgen | 8 | no | consistent; hand-patched 4 fabricated literals (`X-abc123`, `in-abc123-def456`, `X-custom`, `in-abc123` — the suite builds all values via `vfTok(r)`) to `<token>` metavalue form |
| retrypolicy | 7 | no | consistent; hand-patched one residual "the call returns" → "the wrapper returns" |
| sampler | 9 | **yes** | states the boundary: "rate is recalculated precisely when the counter reaches the sample size; the counter is then reset" — the size-th call draws under the new rate |
| skipwriter | 7 | no | consistent |
| svcerror | 12 | no | consistent; regenerated after rate-limit fallback |
| traceopts | 10 | ~ | consistent; adaptive-sampler wording follows the same oracle |

All three audited contradictions are units where the authored contract states
the *intended* rule and the suite pins an artifact or boundary. The v2
contracts recover all three — they read the assertions, not the intent.
Post-processing self-checks fired on 6 of 15 units during the run (invented
literals, synthesized-name examples, quote-pairing artifacts); every one was
resolved by reprocessing the cached payload or a fresh sample — none shipped.

Residual process caveat: upstream free-tier rate limits (HTTP 429) put
namescope and svcerror into the degraded fallback path on the first batch
pass; both regenerated successfully on re-run.

## Honest notes on what v2 still gets wrong

- **Grounding catches invented literals, not wrong claims.** The regenerated
  repindex contract shipped `Get query "1.2" -> parsed as "1.2.0"` — every
  literal is in the suite, but the semantics are wrong (`1.2` is a range over
  `1.2.x` resolving to the highest match — precisely the hand contract's
  round-1 omission). Patched by hand before staging. A wrong-but-grounded
  example needs semantic checking, not just literal checking.
- **The rewriter is still regex-shaped.** First v2 output ate the `- ` bullet
  and glued `comparison(with` onto the preceding clause; fixed by whitespace
  normalization but the transform remains fragile to clause shapes it hasn't
  seen.
- **Merge-winner wording flip-flopped between samples.** One sampled contract
  said the incoming index's `B456` digest wins (backwards vs the
  receiving-index oracle); the shipped sample says "the first index's entry
  wins". Nothing structural pins which index is "first" — worth a dedicated
  pass keyed on the oracle's winner argument.
- **Artifact-order oracles are the hard residual case** (goa `exprhash`):
  when the suite pins a buggy-comparator's observed emission, describing the
  *correct* rule is precisely what fails. The generator would have to quote
  the implementation artifact — a level of fidelity even the authored
  contract didn't reach.
- **Boundary claims** (goa `sampler`): "first N calls all true" vs
  recompute-before-draw at the N-th call. The oracle states it in test
  structure, not in prose; whether the LLM surfaces it depends on reading the
  loop bounds, not the comments.
- `example_literals` still pairs quotes — prose like `annotations.key1 = "…"`
  once produced a garbage span; sentence punctuation is now rejected, but
  unusual prose could still synthesize a false literal (fail-closed: a false
  positive marks the unit unreconcilable rather than shipping a bad example).
- **"the call" is only substituted where it stands in for a scrubbed symbol.**
  Natural-English uses with clear referents survive: "during the call" (the
  Do invocation, httpclienterr), "the call that makes the counter equal the
  sample size" (the size-th Sample call, sampler). One bare-token substitute
  survived the pass in retrypolicy ("the call returns nil" for scrubbed
  `RetryEndpoint`) and was hand-patched to "the wrapper" before staging.

## Verification

- `uv run pytest tests/test_reconcile.py` — 18 tests, incl. one regression
  test per fix using the exact failing repindex text as fixture.
- `uv run ruff check .` — clean.
- Reconciler + tests ported into this worktree (`src/openswe_traces/synth/
  reconcile.py`, `scripts/reconcile_contract.py`, `tests/test_reconcile.py`,
  `src/openswe_traces/synth/testdata/composerver/helm/repindex_bb_prop_test.go`);
  previously untracked in `oswt-RECONCILE`.
- No solver trials run — the parent session owns them. No commits.

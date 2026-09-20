# Where the certified-hard units actually stump Composer (45 units, 61 L0-fail trials)

Method: every certified-hard unit (fail L0 on every attempt, flip at L2) from
`flip_accounting_2026-09-20.txt`. Mechanical pass over all 803 indexed trials for rewards,
failing test names, trajectory statistics and patch sizes; three independent reviewers read the
L0 bug report, the hidden suite, gold, the agent patch and the last agent steps for 15 units each.

## 1. The shape of a "stump"

| measure | value |
|---|---|
| certified-hard L0 pass rate (Composer) | 0/61 |
| certified-hard L2 pass rate (Composer) | 156/171 = 91% |
| share of the hidden suite the FAILING L0 patch already passes | median 78%, mean 72% |
| failing hidden tests per L0-fail trial | median 2 of 8 |
| L0-fail patches that are structural near misses of gold | 36/45 units (80%) |
| L0-fail trials whose final message claims tests pass | 46/61 (75%) |
| L0-fail trials flagging any uncertainty | **0/61** |
| L0-fail trials mentioning hidden or unseen tests | **0/61** |
| verification runs before stopping | median 5 at L0-fail, 8 at L2-pass |

A certified-hard unit is not a unit the model cannot build. It is a unit the model builds to
within one or two clauses and then certifies as finished. 44 of 45 units stopped because the
visible in-tree suite went green.

## 2. Where the missing clause was available at L0

| availability | units | note |
|---|---:|---|
| C — only in the L2 contract or hidden test | ~34 (76%) | free design choice: exact error wording, nil vs empty, threshold strictness, whitespace trim, sentinel choice |
| B — inferable from the repo at L0 | ~8 (18%) | a doc comment or caller overridden by a generic prior |
| A / mixed — partly in the bug report | ~3 (7%) | |
| **repo points the WRONG way** | **13 (29%)** | sibling code, in-tree test, module cache or report wording endorses the answer the hidden suite rejects |

## 3. The construction fact that explains (2)

The L2 coverage table cites the original upstream tests that pinned each behaviour. Measured
across 41 units: **249 of 303 (82%) of those original tests are absent from the L0 agent tree,
and in 31 of 41 units every one of them is gone.** The excision removes the implementation and
the tests that documented it, then the hidden suite re-derives the same behaviour.

So L0 hardness here is manufactured by deletion of evidence, not by intrinsic task difficulty.
The 76% C rate is a consequence, not a finding about the model.

## 4. The residual L2 failures are mostly capability

15 L2-fail trials across ~11 units. Three units are genuine contract defects
(gin-recoverymw's unpinned `: ` separator, helm-sympath Detail04's contrastive SkipDir wording,
kops-assetsremap's literal `registry/`). The other eight are the agent's own bugs, and four are
the same bug: `json.Unmarshal("null", &v)` into a non-pointer succeeds as a silent no-op. Others:
`unicode.IsLetter(rune(0xff))` true for U+00FF, `/32` treated as an invalid mask.

L2 is therefore a reasonably clean ceiling. Post-repair contract-defect share among L2 failures
is ~27%, well below the 40% bank-wide static audit prior.

## 5. What this says about the ladder

1. **L0 to L2 is an evidence-restoration axis, not a difficulty axis.** The instruction grows
   990 -> 3543 chars (3.6x), but the decisive content is one clause of roughly 100 chars.
2. **Rung resolution is far coarser than the phenomenon.** Flip point is a 2-bit measurement of a
   1-clause effect. The right instrument is the staged L1 gapped contract (delete exactly one
   invariant), which holds the contract constant and attributes the flip to a specific sentence.
   16 dirs have been staged since the L1 design landed and had **never been trialled**; launched
   as `sweep_L1` (k=1) at 14:20 today.
3. **"Hard at L0" partly measures oracle access, not capability.** With 0/61 trials mentioning
   unseen tests and the repro command running only the surviving visible tests, the agent has no
   channel that could tell it it is wrong. Per B6 the repro must reproduce; it does. It is not a
   completion signal, and the agent uses it as one.
4. **29% of units are anti-inferable.** More diligent repository reading would reinforce the
   wrong answer. Those units measure conformity with the gold author's taste unless the intended
   rule is stated somewhere the agent can reach.

## 6. Against the local Terminal-Bench data

| | this ladder | TB4 (13 models x 66 tasks x 5 trials) |
|---|---|---|
| rung and pass rate | monotone **by construction** 0% -> 91% | **non-monotone**: L1 28%, L2 36%, L4 42%, L5 28% |
| rung label | designed in, one unit rebuilt per rung | applied post-hoc to single-rung tasks |
| median Bernoulli variance | 0.000; 81% of (unit,rung) cells unanimous | 0.186; 34 tasks sit in coin-flip territory |
| unit scale | median gold patch 92 added lines, one package | multi-hour, multi-domain |
| hardest cases | one clause short of gold | `needs-inference` or `constraint-dense` + `hidden_invariant`, 0% across all 13 models |

Two conclusions. First, the rung label does not predict difficulty for naturally occurring tasks;
it only works when the rung is constructed, which means the ladder measures information dose on a
fixed unit and cannot be used to rank unrelated tasks. Second, TB4's genuinely hard tasks fail for
the same reason our units do — an invariant the spec does not carry — but at a scale where one
sentence cannot close the gap, which is why they survive frontier models.

The local `lakehouse-publish-recovery` analysis is the closest local analogue and is instructive:
Opus 5 and Grok 4.6 both scored 0 at k=3, failing the same four tests, because they imported
Iceberg's field-identity convention over the spec sentence in `DESIGN.md`. That is our wrong-way
category — but the rule *was stated and readable*, so the task is fair and still hard. Our
anti-inferable units state it nowhere at L0, which makes them unfair rather than hard.

**The actionable difference:** prior-conflict hardness transfers across models; missing-clause
hardness does not. Converting the 13 wrong-way units into stated-but-counter-prior units is the
one lever here that plausibly produces units a frontier model fails for a defensible reason.

## 7. L1 gapped-contract result (sweep_L1, 16 trials, k=1, Composer, 2026-09-20)

The 16 staged L1 dirs were run for the first time. Each arm holds the L2 contract constant and
deletes exactly one invariant: `L1binding` deletes the one the unit missed at L0, `L1other`
deletes a different one.

| unit | binding | failed on | other | failed on |
|---|---|---|---|---|
| gin-jsonrenders | **0.0** | TestJsonpJSONCallbackProperty | 1.0 | — |
| gin-streamrenders | **0.0** | TestDataContentLengthProperty | **0.0** | TestRedirectStatusProperty |
| kops-issuecert | **0.0** | TestIssuecertClientServerProperty | 1.0 | — |
| kops-oidcdisc | **0.0** | TestOIDCMemoryStoreListProperty | 1.0 | — |
| kops-templater | **0.0** | TestTemplaterSnippetNameProperty | 1.0 | — |
| kops-addonparse | 1.0 | — | 1.0 | — |
| kops-assetsremap | 1.0 | — | **0.0** | TestAssetsRemapRegistryConvergeProperty |
| kops-memfs | 1.0 | — | 1.0 | — |

Binding arms failed 5/8; control arms passed 6/8.

**Deletion is local.** All seven failing arms failed on exactly one property, and in six of seven
that property is the one whose coverage row was deleted. The exception is
`kops-assetsremap-L1other`, where the deleted row was the comma-escaping row and the failure was
registry convergence; that unit is 2/3 at L2, so a single k=1 failure is inside its base rate.

**Five units now have single-sentence causal attribution.** The L0 -> L2 flip for jsonrenders,
streamrenders, issuecert, oidcdisc and templater is caused by one identified sentence, not by
prose volume. This is the measurement the flip point could not produce.

**Three units are contract-independent.** addonparse and memfs pass both arms, and assetsremap
passes the binding arm. memfs was independently classified B (the node struct and its
`HasChildren` helper pin the rule in the tree) and assetsremap is the same unit that survived the
lie-inversion sweep, for the same stated reason: its callers already pin the fact. Two
independent perturbations, inversion and deletion, agree on which units do not need the contract.

**Refinement to "a wrong row is worse than a missing one".** That comparison held between a lean
correct contract and one carrying extra false rows. The narrower true statement: a missing row is
fatal exactly when the hidden suite asserts that row and the tree does not pin it, and its damage
is confined to that row; a false row is fatal and can additionally break rows it does not name.

**Caveats.** k=1, eight units, one repo family dominant (six of eight are kops or gin). The
binding/other labels were assigned from a single presumed L0 miss, which mislabels assetsremap,
a unit that missed two properties.

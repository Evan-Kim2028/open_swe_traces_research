# Call-graph contract coverage — can reachability see what prose-counting cannot?

2026-09-20. Job CGCOV. Data: `outputs/cg_coverage/` (scans, set files, LLM cache), log `outputs/CGCOV.log`.

## Question

`scripts/ops/task_lint.py` counts contract commitments against hidden assertions and scores **40%
precision vs a 39% non-flip base rate — no information**. The hypothesis: a *structural* gap —
a symbol the hidden suite exercises and gold implements but no contract commitment mentions —
predicts "this unit will not flip" better than prose-counting, because it is grounded in what the
tests actually execute rather than in how long the contract is.

## Method

For each unit dir (`environment/src`, `tests/hidden`, `tests/gold.patch`, `contract.md`/`instruction.md`):

- **reached** — transitive call-graph reach of every hidden `Test*`, computed by `cgscan`
  (`src/openswe_traces/cgscan`, Go stdlib `go/parser`, no type-checking). Excised function bodies
  are panic stubs in the environment tree, so bodies are restored from `gold.patch` added lines
  (generic-signature retry, then a regex fallback for mid-function hunks — this cut gold-body
  parse failures from ~60 warnings across 52 units to 7 regex-fallbacks).
- **gold** — symbols `gold.patch` defines or references.
- **implied** — union of three channels:
  1. reach of tests named as coverage-table row keys (original in-tree tests resolved inside the
     gold-touched package first — same-named tests elsewhere in the module must not hijack);
  2. hidden-suite test names appearing anywhere in the contract text (naming a test discloses
     the behaviour it checks, at coarse granularity);
  3. symbol names literally present in the prose, plus an LLM paraphrase judgement
     (one OpenRouter free-tier request per unit: contract + numbered candidate list ->
     "which symbols does the contract implicate?", cached by content hash).
- **gap = reached ∩ gold − implied.** Predictor under test: `|gap| ≥ 1`.

Cohort: the 119 dose_response unit dirs that ran L2 trials under canonical task paths (the
8 client-go units ran from a /tmp staging dir; reported separately as a robustness check).
Labels from `result.json` rewards: flip = ≥1 L2 pass. The linter's documentation records 119
units, 47 non-flippers (39%); the current snapshot yields 45 (37.8%) — two units flipped on
re-runs after the doc was written. Both numbers are reported.

## Results — confusion matrix, primary cohort (n=119, base 37.8%)

| predictor | flagged | TP | FP | FN | TN | precision | recall |
|---|---:|---:|---:|---:|---:|---:|---:|
| gap ≥ 1, rows+literal only | 77 | 32 | 45 | 13 | 29 | 41.6% | 71.1% |
| gap ≥ 1, deterministic | 72 | 29 | 43 | 16 | 31 | 40.3% | 64.4% |
| **gap ≥ 1, full (+LLM)** | **34** | **17** | **17** | **28** | **57** | **50.0%** | **37.8%** |
| gap ≥ 2, full | 18 | 7 | 11 | 38 | 63 | 38.9% | 15.6% |
| gap ≥ 3, full | 10 | 6 | 4 | 39 | 70 | 60.0% | 13.3% |
| gap ≥ 6, det | 20 | 10 | 10 | 35 | 64 | 50.0% | 22.2% |

Robustness (all 127 units incl. /tmp-staged client-go): full ≥1 = 45.0% precision, 39.1% recall.

Mean |gap|: deterministic 5.27 non-flip vs 2.35 flip (**2.2×**); full 1.60 vs 0.49 (**3.3×**).
The separation the orphan-count proxy showed (2.7×) survives — and the contract-implication
channels make it stronger, not weaker.

## The 10 double-failure families

| unit | candidates | det gap | full gap |
|---|---:|---|---|
| gin-enginecfg-L2 | 4 | `httprouter.New` | — |
| gin-negotiate-L2 | 1 | — | — |
| helm-chartdl-L2 | 23 | 23 (downloader/getter) | `WithAcceptHeader`, `WithPassCredentialsAll`, `provenance.NewFromKeyring` |
| helm-chartrepo-L2 | 28 | 28 (repo/getter opts) | — |
| helm-depresolver-L2 | 6 | `HashV2Req` | **`HashV2Req`** |
| helm-httpgetter-L2 | 20 | 20 (TLS/getter opts) | — |
| helm-repindex-L2 | 24 | `Metadata.Version`, `IndexFile.Add` | `Metadata.Version` |
| kops-clustervalid-L2 | 23 | — | — |
| kops-difftext-L2 | 4 | `lineRecord`, `renderText` | — |
| kops-taintparse-L2 | 2 | `GetNodeRole` | — |

Deterministically 8/10 flagged; after the LLM judgement 3/10 remain — and the two most famous
defect symbols survive: `HashV2Req` (the sha256 digest that made depresolver unsolvable even at
L5, rule B10) and `Metadata.Version`. The others' failures are semantic (contract claims the
wrong thing about covered machinery), which no coverage metric can see — consistent with the
two-family taxonomy in verifier_rules.md.

## helm-repindex revision series

Gold + hidden suite held fixed across revisions; outcomes 0/3 → 0/3 → 1/3 → 0/3 → 2/3.

| revision | outcome | cand | rows-only | det gap | full gap |
|---|---|---:|---:|---:|---:|
| L2 orig | 0/3 | 24 | 2 | 2 | 1 (`Metadata.Version`) |
| L3 orig | 0/3 | 24 | 2 | 0 | 0 |
| rev1 rcfix | 0/3 | 24 | 23 | 0 | 0 |
| rev2 rcfix2 | 0/3 | 24 | 23 | 0 | 0 |
| rev3 rcfix3 | 1/3 | 24 | 23 | 0 | 0 |
| rev4 rcfix4 | 0/3 | 24 | 23 | 0 | 0 |
| rev5 rcfix5 | 2/3 | 24 | 23 | 0 | 0 |
| rc_auto | 0/3 | 24 | 0 | 0 | 0 |
| rc_auto2 | 1/3 | 24 | 0 | 0 | 0 |

The original contract leaves `Metadata.Version` (+ `IndexFile.Add`, cleared by the LLM)
structurally unimplicated. Every revision names the hidden suite explicitly, so the gap
collapses 1 → 0 at rev1 and stays there — **monotone non-increasing, but saturated**: the
check cannot distinguish the revisions that passed from those that failed, because what
moved across revisions was the semantic content of commitments, not their structural
footprint. ("rows-only" reads the rcfix format as 23 only because its commitment table uses
prose keys rather than test names — that channel only works for coverage-row contracts.)

## Cost

- cgscan: ~0.3 s/unit, deterministic, no Docker.
- LLM map: 1 request/unit, OpenRouter `:free` models, cached by content hash. 124 fresh
  requests for 127 units, 0 failed (cap 300). ~15–60 s/request free-tier latency.
- Whole-cohort refresh: scan ≈ 1 min (8 workers), map ≈ 25 min (4 workers, latency-bound,
  resume-safe).

## Limitations

- Syntactic call graph, no type-checking: `bare:` selector calls fan out by method name
  (over-approximates); the regex fallback sees only call-shaped tokens.
- Coverage-row implication needs real tests in `environment/src`; gin-family contracts name
  upstream tests stripped from the tree, so rows contribute nothing there.
- Naming a hidden test implicates its whole reach — coarse but defensible for "named or
  paraphrased".
- Outcome labels drift: 45 non-flippers/119 now vs 47 documented. Matrix computed on the
  snapshot.

## Verdict

**Real signal, wrong instrument for a hard gate.** "Any structural gap" predicts non-flip at
50% precision vs a 37.8% base rate — better than the linter's nothing (40% vs 39%) and on par
with the orphan-count proxy (53% at ≥2), but at that operating point it blocks 29% of the
cohort to catch 38% of non-flippers while wrongly flagging 17 of 74 flippers. As a *diagnostic*
gate it earns its keep: it is ~1 free request + 0.3 s per unit, it names the actual defect
symbols (`HashV2Req`, `Metadata.Version`), and the 3.3× mean separation is the strongest
cheap signal measured on this cohort. The repindex series shows the ceiling plainly:
structural coverage saturates the moment a contract discloses the suite, while outcomes move
on semantics — the contract-family defect needs the shadow check, not this one.

Recommendation: run it in the pre-trial pipeline as a **reporting gate** (flag + list gap
symbols in the unit review), not a blocking one. Blocking would discard a quarter of good
units for a third of the bad ones.

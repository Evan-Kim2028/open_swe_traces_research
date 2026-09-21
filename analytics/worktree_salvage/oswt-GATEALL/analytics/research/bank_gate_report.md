# Bank gate report — whole gate stack over the packaged bank

*Generated 2026-09-20 15:34:01 by `scripts/gate_bank.py`. Gates: `scripts/ops/task_lint.py` (deterministic) → orphan count (`cgscan`, reachability over the restored tree) → A13 contract-vs-gold judge (OpenRouter free tier, ≤300 requests, cached).*

## Validation against trial outcomes

Flag = the gate says do-not-trial (predicted non-flip). Positive = the unit/family truly did not flip.

### Primary cohort — 132 L2 unit dirs with known outcomes (49 non-flip, base 37% vs the documented ~39%)

| gate | TP | FP | FN | TN | precision | recall | base |
|---|---:|---:|---:|---:|---:|---:|---:|
| lint | 3 | 4 | 46 | 79 | 43% | 6% | 37% |
| b10 | 1 | 2 | 48 | 81 | 33% | 2% | 37% |
| orphans | 17 | 26 | 32 | 57 | 40% | 35% | 37% |
| gap>=1 | 27 | 37 | 22 | 46 | 42% | 55% | 37% |
| a13 | 28 | 28 | 21 | 55 | 50% | 57% | 37% |
| combined | 22 | 39 | 27 | 44 | 36% | 45% | 37% |

**Combined recommendation does NOT beat the base rate** (36% vs 37%).

Robustness (+8 /tmp-staged client-go units):

| gate | TP | FP | FN | TN | precision | recall | base |
|---|---:|---:|---:|---:|---:|---:|---:|
| lint | 3 | 5 | 47 | 85 | 38% | 6% | 36% |
| b10 | 1 | 2 | 49 | 88 | 33% | 2% | 36% |
| orphans | 18 | 32 | 32 | 58 | 36% | 36% | 36% |
| gap>=1 | 28 | 43 | 22 | 47 | 39% | 56% | 36% |
| a13 | 28 | 30 | 22 | 60 | 48% | 56% | 36% |
| combined | 23 | 45 | 27 | 45 | 34% | 46% | 36% |

### Family-level cohorts — 123 L0-failed escalated families and all 124 escalated families

L0-failed families with >=L2 trials: **123** (37 non-flip).

| gate | TP | FP | FN | TN | precision | recall | base |
|---|---:|---:|---:|---:|---:|---:|---:|
| lint | 3 | 4 | 34 | 82 | 43% | 8% | 30% |
| b10 | 1 | 1 | 36 | 85 | 50% | 3% | 30% |
| orphans | 19 | 35 | 18 | 51 | 35% | 51% | 30% |
| gap>=1 | 27 | 49 | 10 | 37 | 36% | 73% | 30% |
| a13 | 16 | 30 | 21 | 56 | 35% | 43% | 30% |
| combined | 24 | 47 | 13 | 39 | 34% | 65% | 30% |

All 124 families with >=L2 trials:

| gate | TP | FP | FN | TN | precision | recall | base |
|---|---:|---:|---:|---:|---:|---:|---:|
| lint | 3 | 4 | 35 | 82 | 43% | 8% | 31% |
| b10 | 1 | 1 | 37 | 85 | 50% | 3% | 31% |
| orphans | 19 | 35 | 19 | 51 | 35% | 50% | 31% |
| gap>=1 | 28 | 49 | 10 | 37 | 36% | 74% | 31% |
| a13 | 17 | 30 | 21 | 56 | 36% | 45% | 31% |
| combined | 24 | 47 | 14 | 39 | 34% | 63% | 31% |

## Drop list — units the gates say should NEVER be trialed

13 units carry an unfixable defect (B10 digest oracle — the hidden suite asserts a literal digest of an internal serialisation; unsolvable at every rung including L5).

| unit | evidence |
|---|---|
| `helm/depresolver@L-1` | B10: internal/resolver/depresolver_bb_prop_test.go: literal digest oracle sha256:fb239e836325c5fa1… — unsolvable at every rung; GOLD: no gold.patch — nothing to verify against |
| `helm/depresolver@L0` | B10: internal/resolver/depresolver_bb_prop_test.go: literal digest oracle sha256:fb239e836325c5fa1… — unsolvable at every rung |
| `helm/depresolver@L2` | B10: internal/resolver/depresolver_bb_prop_test.go: literal digest oracle sha256:fb239e836325c5fa1… — unsolvable at every rung |
| `helm/depresolver@L3` | B10: internal/resolver/depresolver_bb_prop_test.go: literal digest oracle sha256:fb239e836325c5fa1… — unsolvable at every rung |
| `helm/depresolver@L4` | B10: internal/resolver/depresolver_bb_prop_test.go: literal digest oracle sha256:fb239e836325c5fa1… — unsolvable at every rung |
| `helm/depresolver@L5` | B10: internal/resolver/depresolver_bb_prop_test.go: literal digest oracle sha256:fb239e836325c5fa1… — unsolvable at every rung |
| `helm/depresolver@L6` | B10: internal/resolver/depresolver_bb_prop_test.go: literal digest oracle sha256:fb239e836325c5fa1… — unsolvable at every rung |
| `kops/assetsremap@L-1` | B10: pkg/assets/assetsremap_bb_prop_test.go: literal digest oracle sha256:0123456789abcdef0… — unsolvable at every rung; GOLD: no gold.patch — nothing to verify against |
| `kops/assetsremap@L0` | B10: pkg/assets/assetsremap_bb_prop_test.go: literal digest oracle sha256:0123456789abcdef0… — unsolvable at every rung |
| `kops/assetsremap@L1` | B10: pkg/assets/assetsremap_bb_prop_test.go: literal digest oracle sha256:0123456789abcdef0… — unsolvable at every rung |
| `kops/assetsremap@L2` | B10: pkg/assets/assetsremap_bb_prop_test.go: literal digest oracle sha256:0123456789abcdef0… — unsolvable at every rung |
| `kops/assetsremap@L5` | B10: pkg/assets/assetsremap_bb_prop_test.go: literal digest oracle sha256:0123456789abcdef0… — unsolvable at every rung |
| `kops/assetsremap@L6` | B10: pkg/assets/assetsremap_bb_prop_test.go: literal digest oracle sha256:0123456789abcdef0… — unsolvable at every rung |

## Ranked escalation list — the 92 unescalated families

80 families were screened (>=1 trial row) but never ran a >=L2 trial. P(flip) is the measured flip rate of the family's gate-verdict cohort on the primary cohort: clean=62%, flagged=64%, drop=0. Expected trials are k=1-sequential (sweep_seq.sh, <= 3 rounds): E = 1 + q + q², q = 1−P. Token cost at 2.18M/trial.

| # | family | repo | L0 | gate verdict | P(flip) | E[trials] | E[tokens] | notes |
|---:|---|---|---|---|---:|---:|---:|---|
| 1 | client-go/keyflags | client-go | err | trial | 62% | 1.52 | 3.32M | never screened (all trials errored); no L2+ unit packaged (best: client-go/keyflags@L-1) |
| 2 | client-go/latchsched | client-go | err | trial | 62% | 1.52 | 3.32M | never screened (all trials errored); no L2+ unit packaged (best: client-go/latchsched@L-1) |
| 3 | client-go/localoracle | client-go | err | trial | 62% | 1.52 | 3.32M | never screened (all trials errored); no L2+ unit packaged (best: client-go/localoracle@L-1) |
| 4 | client-go/onregionerror | client-go | fail | trial | 62% | 1.52 | 3.32M | A13: 0 false/2 missing |
| 5 | client-go/pipelineddb | client-go | err | trial | 62% | 1.52 | 3.32M | never screened (all trials errored); no L2+ unit packaged (best: client-go/pipelineddb@L-1) |
| 6 | client-go/prewrite | client-go | err | trial | 62% | 1.52 | 3.32M | never screened (all trials errored); no L2+ unit packaged (best: client-go/prewrite@L-1) |
| 7 | client-go/replicaselector | client-go | fail | trial | 62% | 1.52 | 3.32M | A13: 0 false/3 missing |
| 8 | client-go/snapscan | client-go | err | trial | 62% | 1.52 | 3.32M | never screened (all trials errored); no L2+ unit packaged (best: client-go/snapscan@L-1) |
| 9 | gin/treeinsert | gin | fail | trial | 62% | 1.52 | 3.32M | no L2+ unit packaged (best: gin/treeinsert@L-1) |
| 10 | nats-server/jwtvalidate | nats-server | err | trial | 62% | 1.52 | 3.32M | never screened (all trials errored); no L2+ unit packaged (best: nats-server/jwtvalidate@L0) |
| 11 | nats-server/seqset | nats-server | err | trial | 62% | 1.52 | 3.32M | never screened (all trials errored); no L2+ unit packaged (best: nats-server/seqset@L0) |
| 12 | nats-server/subjecttransform | nats-server | err | trial | 62% | 1.52 | 3.32M | never screened (all trials errored); no L2+ unit packaged (best: nats-server/subjecttransform@L0) |
| 13 | store/s23-r020 | store | fail | trial | 62% | 1.52 | 3.32M | no L2+ unit packaged (best: store/s23-r020@L0) |
| 14 | store/s53-r250 | store | fail | trial | 62% | 1.52 | 3.32M | no L2+ unit packaged (best: store/s53-r250@L0) |
| 15 | client-go/backoffer | client-go | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: client-go/backoffer@L-1) |
| 16 | client-go/connarray | client-go | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 0 false/4 missing |
| 17 | client-go/pdoracle | client-go | pass | repair | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 0 false/2 missing |
| 18 | client-go/pessimisticlock | client-go | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 19 | client-go/rangetask | client-go | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 20 | gin/affix | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 21 | gin/basicauth | gin | mixed | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: gin/basicauth@L-1) |
| 22 | gin/bindingdispatch | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 0 false/1 missing |
| 23 | gin/bodydecoders | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 1 false/1 missing |
| 24 | gin/colorfmt | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 25 | gin/contain | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 26 | gin/crossfld | gin | pass | repair | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 27 | gin/cryptocoin | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 28 | gin/ctxquery | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: gin/ctxquery@L-1) |
| 29 | gin/defaultengine | gin | pass | repair | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 0 false/2 missing |
| 30 | gin/formmapping | gin | mixed | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 1 false/4 missing |
| 31 | gin/hashfmt | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 32 | gin/hostport | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 33 | gin/htmlrender | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 1 false/1 missing |
| 34 | gin/multipartfiles | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 35 | gin/requestbinders | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 36 | gin/routelookup | gin | mixed | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: gin/routelookup@L-1) |
| 37 | gin/structlvl | gin | pass | repair | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 38 | gin/urifmt | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 39 | gin/uuidfmt | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 40 | gin/validator | gin | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 41 | go-github/ndjsonmetrics | go-github | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 42 | go-github/pubkeyjson | go-github | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 43 | go-github/refescape | go-github | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 44 | go-github/teamsupdate | go-github | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 45 | goa/errloc | goa | pass | repair | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 46 | goa/evalrun | goa | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 0 false/3 missing |
| 47 | goa/grpcerr | goa | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 48 | goa/grpctrace | goa | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 49 | goa/grpcxray | goa | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 50 | goa/httperrresp | goa | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: goa/httperrresp@L-1) |
| 51 | goa/importalias | goa | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: goa/importalias@L-1) |
| 52 | goa/pkgvalidation | goa | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 0 false/1 missing |
| 53 | goa/skipwriter | goa | mixed | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: goa/skipwriter@L-1) |
| 54 | helm/chartloader | helm | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 3 false/2 missing |
| 55 | helm/chartmeta | helm | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: helm/chartmeta@L0) |
| 56 | helm/coalesce | helm | mixed | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 0 false/2 missing |
| 57 | helm/getterdispatch | helm | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: helm/getterdispatch@L0) |
| 58 | helm/kindsorter | helm | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 2 false/3 missing |
| 59 | helm/memorydriver | helm | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 2 false/2 missing |
| 60 | helm/provenance | helm | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 61 | helm/relsplit | helm | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: helm/relsplit@L0) |
| 62 | helm/storage | helm | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 1 false/3 missing |
| 63 | helm/strvalsparser | helm | mixed | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 0 false/5 missing |
| 64 | helm/urlutil | helm | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: helm/urlutil@L0) |
| 65 | kops/clusternames | kops | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 66 | kops/flagbuilder | kops | mixed | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 0 false/1 missing |
| 67 | kops/ignames | kops | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 68 | kops/intstrpct | kops | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 69 | kops/labelvals | kops | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 70 | kops/osmetadata | kops | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 71 | kops/portrange | kops | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 72 | kops/semververs | kops | pass | repair | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify |
| 73 | kops/tomlwriter | kops | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; A13: 0 false/1 missing |
| 74 | store/s11-r015 | store | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: store/s11-r015@L0) |
| 75 | store/s37-r020 | store | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: store/s37-r020@L0) |
| 76 | store/s42-r250 | store | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: store/s42-r250@L0) |
| 77 | store/s67-r300 | store | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: store/s67-r300@L0) |
| 78 | xrepo/condreq | xrepo | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: xrepo/condreq@L0) |
| 79 | xrepo/fieldcmp | xrepo | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: xrepo/fieldcmp@L0) |
| 80 | xrepo/oneofuniq | xrepo | pass | trial | 0% | 3.00 | 6.54M | passed L0 — easy, nothing to certify; no L2+ unit packaged (best: xrepo/oneofuniq@L0) |

## Projected cost — ranked k=1-sequential vs k=3-everything

| plan | families | expected trials | expected tokens | expected certified units |
|---|---:|---:|---:|---:|
| escalate all 92 at k=3 | 80 | 240 | 523M | 13.2 |
| ranked k=1-sequential (this list) | 80 | 219 | 478M | 13.2 |
| gate-filtered: trial-rec only | 14 | 21 | 47M | 13.2 |

k=1-sequential saves 45M tokens (9%) versus blind k=3 — before counting the drop/repair units it refuses to buy trials for.

## Inventory

- unit names gated: 783 (across tasks_composerver, tasks, work, dose_response, harbor_nex)
- trials in ledger: 831 (1513M tokens in+out)
- families with trial rows: 204; escalated (>=L2): 124; unescalated: 80
- flipped at L2: 86; double-fail: 38
- cold repos (<3 banked units, A13 gated): ['bbolt', 'go-git', 'go-github', 'nats-server', 'nex', 'store', 'xrepo']
- A13 verdicts applied: 256 units (153 fail)

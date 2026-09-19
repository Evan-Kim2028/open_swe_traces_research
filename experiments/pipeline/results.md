# Overnight pipeline — live results

- tasks seen: **19**
- tasks built (authored+): **6** (packaged/solved: 5)
- rejected by rule: **1**
- screened (contaminated/excluded): **0**
- hacked (B9 hard fail, excluded from flip): **0**
- Composer tokens used (in+out): **5703175**
- inter-attempt 2-1 split fraction: **0.00**

## Rejected by rule

| rule | n |
|---|---:|
| `A3` | 1 |

## Calibration curve (pass rate per repo × solver × level)

| repo | solver | level | passes | attempts | rate |
|---|---|---:|---:|---:|---:|
| client-go | devin | L0 | 2 | 3 | 0.67 |
| client-go | devin | L2 | 12 | 14 | 0.86 |
| client-go | devin | L5 | 0 | 1 | 0.00 |
| gin | cursor | L2 | 0 | 1 | 0.00 |
| goa | cursor | L0 | 1 | 1 | 1.00 |
| goa | cursor | L2 | 2 | 2 | 1.00 |
| kops | cursor | L0 | 2 | 2 | 1.00 |
| kops | cursor | L2 | 5 | 5 | 1.00 |

Nearest 50% pass (lower level wins ties):

- `client-go` / `devin`: L0
- `gin` / `cursor`: L2
- `goa` / `cursor`: L0
- `kops` / `cursor`: L0

## Author calibration (predicted_flip vs measured flip)

| backend | n | paired | MAE | exact | off-by-one |
|---|---:|---:|---:|---:|---:|
| devin | 2 | 0 | — | 0 | 0 |
| unknown | 20 | 0 | — | 0 | 0 |

## Early-warning flags

### `attempt_time` client-go devin

client-go/memdbstaging devin L2: wall 840s → 2208s (>2x)

**Action:** Attempt wall time grew > 2x across attempts. Treat as infra (class d), not a fail; check hung tests, load, and Harbor timeouts.


## Flip-point histogram per solver

| solver | flip | n |
|---|---|---:|
| ? | none | 6 |
| cursor | 0 | 3 |
| cursor | none | 4 |
| devin | 0 | 2 |
| devin | 2 | 2 |
| devin | none | 5 |

## Tasks

| repo | unit | family | files | lines | solver | flip | L2 | status |
|---|---|---|---:|---:|---|---|---|---|
| client-go | connarray |  | 0 | 0 | cursor | None | 0/0 | resume |
| client-go | connarray |  | 0 | 0 | devin | 2 | 4/4 | resume |
| client-go | connarray-cv |  | 0 | 0 | devin | 0 | 4/4 | solved |
| client-go | doactionbatches |  | 0 | 0 | cursor | None | 0/0 | resume |
| client-go | doactionbatches |  | 0 | 0 | devin | None | 0/1 | resume |
| client-go | doactionbatches-cv |  | 0 | 0 | devin | None | 0/1 | resume |
| client-go | lockresolver |  | 0 | 0 | cursor | None | 0/0 | resume |
| client-go | lockresolver |  | 0 | 0 | devin | 2 | 1/1 | resume |
| client-go | lockresolver-cv |  | 0 | 0 |  | None | 0/0 | resume |
| client-go | memdbstaging |  | 0 | 0 | devin | 0 | 3/3 | solved |
| client-go | memdbstaging-cv |  | 0 | 0 |  | None | 0/0 | resume |
| client-go | onregionerror |  | 0 | 0 | devin | None | 0/0 | resume |
| client-go | onregionerror-cv |  | 0 | 0 | devin | None | 0/0 | resume |
| client-go | pdoracle |  | 0 | 0 | devin | None | 0/0 | resume |
| client-go | pdoracle-cv |  | 0 | 0 |  | None | 0/0 | resume |
| client-go | pessimisticlock |  | 0 | 0 |  | None | 0/0 | resume |
| gin | formmapping-cv |  | 0 | 0 | cursor | None | 0/1 | solving |
| goa | errloc-cv |  | 0 | 0 | cursor | 0 | 2/2 | solved |
| kops | assetsremap-cv |  | 0 | 0 | cursor | 0 | 2/2 | solved |
| kops | clustervalid-cv |  | 0 | 0 | cursor | 0 | 3/3 | solved |
| nats-server | seqset | sequence | 1 | 715 |  | None | 0/0 | authored |
| nats-server | subjecttree | cross-file | 11 | 1443 |  | None | 0/0 | rejected |

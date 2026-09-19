# Overnight pipeline — live results

- tasks seen: **6**
- tasks built (authored+): **2** (packaged/solved: 1)
- rejected by rule: **1**
- screened (contaminated/excluded): **0**
- hacked (B9 hard fail, excluded from flip): **0**
- Composer tokens used (in+out): **2124444**
- inter-attempt 2-1 split fraction: **0.00**

## Rejected by rule

| rule | n |
|---|---:|
| `A3` | 1 |

## Calibration curve (pass rate per repo × solver × level)

| repo | solver | level | passes | attempts | rate |
|---|---|---:|---:|---:|---:|
| client-go | devin | L0 | 1 | 1 | 1.00 |
| client-go | devin | L2 | 6 | 6 | 1.00 |
| gin | cursor | L0 | 2 | 2 | 1.00 |
| gin | cursor | L2 | 3 | 3 | 1.00 |

Nearest 50% pass (lower level wins ties):

- `client-go` / `devin`: L0
- `gin` / `cursor`: L0

## Author calibration (predicted_flip vs measured flip)

| backend | n | paired | MAE | exact | off-by-one |
|---|---:|---:|---:|---:|---:|
| devin | 2 | 0 | — | 0 | 0 |
| unknown | 4 | 0 | — | 0 | 0 |

## Early-warning flags

_none fired_

Checked (not fired): noisy_splits

## Flip-point histogram per solver

| solver | flip | n |
|---|---|---:|
| ? | none | 3 |
| cursor | 0 | 1 |
| devin | 0 | 1 |
| devin | 2 | 1 |

## Tasks

| repo | unit | family | files | lines | solver | flip | L2 | status |
|---|---|---|---:|---:|---|---|---|---|
| client-go | connarray |  | 0 | 0 | devin | 2 | 2/2 | solving |
| client-go | connarray-cv |  | 0 | 0 | devin | 0 | 4/4 | solving |
| client-go | lockresolver-cv |  | 0 | 0 |  | None | 0/0 | solving |
| gin | bindingdispatch |  | 0 | 0 | cursor | 0 | 3/3 | solved |
| nats-server | seqset | sequence | 1 | 715 |  | None | 0/0 | authored |
| nats-server | subjecttree | cross-file | 11 | 1443 |  | None | 0/0 | rejected |

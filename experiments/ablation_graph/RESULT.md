# ablation_graph — GRAPH vs NOGRAPH on mgechev/revive

Date: 2026-09-18. Host: fresh clones of `mgechev/revive` (same tree; only `START.txt` and `bugs/` differ). Budget: 1 h per condition. Judge used a fresh `codegraph index` of `experiments/ablation_graph/repos/revive-graph` (606 files, 4,557 nodes, 9,428 edges) and mapped NOGRAPH paths onto that index.

**Verdict:** codegraph did **not** improve discovery on this repo. NOGRAPH produced more valid bugs (5 vs 3) and more submissions (33 vs 27). Every submitted bug built, had f2p≥1, passed alt, failed cheat, and was not flaky. A4 rejected 24/27 GRAPH and 28/33 NOGRAPH because revive’s `test/` integration tests sit outside `codegraph impact <symbol>` (CLI default depth 2).

Caveats: n=1 repo, n=1 builder run per condition, same underlying model.

## Procedure

Same steps for every patch in both conditions:

1. Apply `.patch` on a clean copy of the GRAPH tree (`patch -p1`; trees are identical except `START.txt` / `bugs/`).
2. `go build ./...`; `go test ./... -count=1 -timeout=10m`.
3. Record f2p test names + files (codegraph node `file_path` for those names).
4. Revert; apply `.patch` then `.alt.patch` / `.cheat.patch`; rerun the f2p parents.
5. Run `go test ./...` twice on the buggy tree (flake = f2p set differs).
6. Impact = files from `codegraph impact <symbol>` (depth 2) on the GRAPH index.
7. A4: f2p files ⊆ impact files.
8. Cross-file: any f2p file’s package (parent dir) ≠ changed file’s package.
9. Hops: reverse `calls` BFS from function/method nodes named `<symbol>` in the changed file to a node in an f2p file.

VALID iff builds ∧ f2p≥1 ∧ alt passes f2p ∧ cheat fails f2p ∧ not flaky ∧ A4.

Logs: `experiments/ablation_graph/out/judge.json`. Repro: `uv run python experiments/ablation_graph/judge.py`.

START.txt: GRAPH `Fri Sep 18 08:17:04 PM UTC 2026`; NOGRAPH `Fri Sep 18 08:17:31 PM UTC 2026`.

## GRAPH (27 submitted)

INDEX.md minutes sum to **76** (over the 1 h budget). INDEX also says wall-clock ~27 min. Minutes below are the INDEX column.

| bug | valid? | failed | XF | hops | impact files | min |
|---|---|---|---|---:|---:|---:|
| IsPkgDotName | no | A4 | Y | — | 14 | 8 |
| Name | yes | — | Y | 1 | 162 | 5 |
| ReceiverType | no | A4 | Y | — | 6 | 4 |
| IsCallToExitFunction | no | A4 | Y | 1 | 5 | 3 |
| NormalizeOption | no | A4 | Y | 3 | 24 | 4 |
| BlockBranch | no | A4 | Y | — | 3 | 3 |
| Deviates | no | A4 | Y | — | 1 | 3 |
| ArgumentsLimitRule | yes | — | Y | — | 8 | 2 |
| visitIf | no | A4 | Y | — | 1 | 3 |
| IsTest | no | A4 | Y | — | 10 | 3 |
| IsCgoExported | no | A4 | Y | — | 3 | 2 |
| GetTypeNames | no | A4 | Y | — | 3 | 3 |
| filterFailures | no | A4 | Y | — | 3 | 3 |
| IsIdent | no | A4 | Y | — | 25 | 3 |
| IsImportable | no | A4 | Y | — | 4 | 3 |
| isVersionPath | no | A4 | Y | — | 2 | 2 |
| IsStringLiteral | no | A4 | Y | — | 2 | 2 |
| Returns | no | A4 | Y | — | 1 | 2 |
| FuncSignatureIs | no | A4 | Y | — | 2 | 2 |
| IsEmpty | no | A4 | Y | — | 1 | 2 |
| IsMain | no | A4 | Y | — | 5 | 3 |
| isRuleOption | no | A4 | Y | 2 | 39 | 3 |
| HasDecls | no | A4 | Y | 1 | 2 | 2 |
| IsShort | yes | — | N | 1 | 2 | 1 |
| IsPointerToPkgDotType | no | A4 | Y | — | 2 | 2 |
| SeverityFor | no | A4 | Y | — | 11 | 2 |
| normalizePath | no | A4 | Y | — | 2 | 1 |

Valid: Name, ArgumentsLimitRule, IsShort.

## NOGRAPH (33 submitted)

INDEX.md minutes sum to **34**.

| bug | valid? | failed | XF | hops | impact files | min |
|---|---|---|---|---:|---:|---:|
| Name | yes | — | Y | 1 | 162 | 2 |
| IsCallToExitFunction | no | A4 | Y | 1 | 5 | 1 |
| ReceiverType | no | A4 | Y | — | 6 | 1 |
| NormalizeOption | no | A4 | Y | 3 | 24 | 1 |
| MatchFileName | yes | — | N | 1 | 5 | 1 |
| Deviates | no | A4 | Y | — | 1 | 1 |
| isVersionPath | no | A4 | Y | — | 2 | 1 |
| AddIfAbsent | no | A4 | Y | 1 | 3 | 1 |
| IsImportable | no | A4 | Y | — | 4 | 1 |
| SeverityFor | no | A4 | Y | — | 11 | 1 |
| IsPkgDotName | no | A4 | Y | — | 14 | 1 |
| IsStringLiteral | no | A4 | Y | — | 2 | 1 |
| BlockBranch | no | A4 | Y | — | 3 | 1 |
| IsTest | no | A4 | Y | — | 10 | 1 |
| MustExclude | no | A4 | Y | 1 | 5 | 1 |
| GoFmt | no | A4 | Y | — | 23 | 1 |
| HasDecls | no | A4 | Y | 1 | 2 | 1 |
| IsShort | yes | — | N | 1 | 2 | 1 |
| ArgumentsLimit | yes | — | Y | — | 8 | 1 |
| filterFailures | no | A4 | Y | — | 3 | 1 |
| IsIdent | no | A4 | Y | — | 25 | 1 |
| IsCgoExported | no | A4 | Y | — | 3 | 1 |
| unpackIndexExpr | no | A4 | Y | — | 6 | 1 |
| disableNextLine | no | A4 | Y | — | 0 | 1 |
| semanticallyEqual | no | A4 | Y | — | 1 | 1 |
| LineLengthLimit | yes | — | Y | — | 5 | 1 |
| NodeHash | no | A4 | Y | — | 5 | 1 |
| isMain | no | A4 | Y | — | 3 | 1 |
| Returns | no | A4 | Y | — | 1 | 1 |
| IsEmpty | no | A4 | Y | — | 1 | 1 |
| GetTypeNames | no | A4 | Y | — | 3 | 1 |
| normalizePath | no | A4 | Y | — | 2 | 1 |
| isDirectiveComment | no | A4 | Y | — | 3 | 1 |

Valid: Name, MatchFileName, IsShort, ArgumentsLimit, LineLengthLimit.

## Summary

| metric | GRAPH | NOGRAPH |
|---|---:|---:|
| submitted | 27 | 33 |
| valid | 3 | 5 |
| valid bugs per hour (1 h budget) | 3 | 5 |
| valid XF | 2 | 3 |
| XF valid rate (valid XF / valid) | 2/3 | 3/5 |
| XF valid rate (valid XF / submitted) | 2/27 | 3/33 |
| A4 rejection count | 24 | 28 |
| mechanical pass (build+f2p+alt+cheat+not flaky) | 27 | 33 |
| mean hops (valid, hops known) | 1.0 (n=2) | 1.0 (n=3) |
| mean impact files (valid) | 57.3 | 36.4 |
| time to first valid (cumulative INDEX minutes) | 13 | 2 |
| INDEX minutes sum | 76 | 34 |

Hops `—` means no reverse-`calls` path from a function/method named that symbol in the changed file (common for type names: ArgumentsLimitRule / ArgumentsLimit / LineLengthLimit; also most helpers whose f2p lives only in `test/`). `disableNextLine` is a switch case, not a symbol: `codegraph impact` returned no node (0 files).

`codegraph impact Name` is a **name collision**: 162 files, including 101 under `test/`. That is why Name passes A4. File-scoped impact of `internal/rule/name.go:Name` would not contain `test/golint_test.go` / `test/var_naming_test.go` / `test/json_data_format_test.go`. Dropping Name: GRAPH 2 valid (1 XF), NOGRAPH 4 valid (2 XF). Same direction.

Builders marked IsShort and MatchFileName as XF (different **file**). Judge XF is different **package**: both N.

## Rule violations

- Test files modified: **none**. Every `.patch` / `.alt.patch` / `.cheat.patch` hunk is a production `.go` file (confirmed from `diff --git` headers).
- Internet use: **no evidence** in `bugs/`, INDEX.md, or patches. The only `https://` strings are a pre-existing comment on `directiveCommentRE` in `rule/utils.go` that leaked into a few NOGRAPH diffs.

## Why A4 killed the yield

Revive’s interesting f2p is almost always `test/<rule>_test.go` (package `test`), which calls the linter API. `codegraph impact` at depth 2 follows production callers into `rule/*.go` and almost never into `test/`. Bugs whose only in-impact f2p is a same-package `*_test.go` (IsShort, MatchFileName) or a rule whose impact happens to list `test/argument_limit_test.go` / `test/line_length_limit_test.go` survive. GRAPH-only symbols (visitIf, FuncSignatureIs, IsPointerToPkgDotType, isRuleOption, IsMain) all died on A4 the same way.

## Verdict (plain)

On this one repo, with one builder hour each and the same model, codegraph did not raise valid-bug yield, XF-valid yield, or A4 survival. NOGRAPH submitted more bugs faster (INDEX minutes 34 vs 76; first valid at 2 vs 13). The graph did not find a class of A4-valid bugs that grep+gopls missed. Do not generalize past n=1 repo × n=1 run × one model.

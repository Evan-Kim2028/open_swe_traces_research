# ablation_graph round 2 — GRAPH vs NOGRAPH feature-excision units on mgechev/revive

Date: 2026-09-18. Host: same `mgechev/revive` tree as round 1 (`experiments/ablation_graph/repos/revive-graph` ≡ `revive-nograph` production files; `diff -rq` empty excluding `units/`, `START*.txt`, `.codegraph`). Budget named in this prompt: 1 h/condition. Builder briefs and `START2.txt` / NOGRAPH INDEX clock are **15 min** wall-clock (`Fri Sep 18 10:05:19 PM UTC 2026` → stop `10:20:19`). INDEX minutes sum to 15 (GRAPH) and 13 (NOGRAPH). Objective: QUALITY (feature-excision units), not count.

Judge used the GRAPH `codegraph` index (606 files, 4,557 nodes, 9,427 edges) for both conditions. Logs: `experiments/ablation_graph/out/judge2.json`. Repro: `uv run python experiments/ablation_graph/judge2.py`.

**Verdict:** codegraph did **not** improve discovery of hard units. NOGRAPH produced more valid units (2 vs 1). The only GRAPH-valid unit (file-exclude-filter) is the same subsystem NOGRAPH also landed (file-filter). GRAPH’s distinctive walker unit failed self-containment on the unlisted umbrella `TestAll`.

Caveats: n=1 repo, n=1 builder run per condition, same underlying model.

## Procedure

Same steps for every unit in both conditions. Closures, tests, contracts, properties, minutes: from `units/<name>/*.md` and `units/INDEX.md`. Call graph and sizes: GRAPH index. Excision: apply that unit’s `excision.patch` on a clean copy of the GRAPH tree (`patch -p0` or `-p1` from the diff header).

1. **(a) Closure.** For each listed `(file, function)`, take `calls` edges from that definition (`callers_of_definition`, confidence ≥ 0.85). A caller is dangling if it is not in the closure, is not a caller of the entry, and is not in a `*_test.go` file. VALID requires zero production dangling callers.
2. **(b) Self-containment.** `go build ./...` succeeds; listed tests fail (`go test ./... -run` those names, timeout 45s, hang counts as fail); every other test is green (`go test ./... -skip` those names). Builder `PROOF.txt` files that only ran `logging`/`syncset`/`formatter` were not trusted.
3. **(c) Black-box.** Listed test function bodies do not mention an unexported name from the closure.
4. **(d) Contract coverage.** Every listed test has a coverage-table sentence in `contract.md` (one sentence may bundle many asserts). Not a validity gate.
5. **(e) Properties.** Spot-check `properties.md` against the indexed source. Not a validity gate.
6. **(f) Size.** Functions = listed closure; files = distinct closure files; lines = sum of `end_line-start_line+1` on GRAPH nodes.

VALID iff (a) ∧ (b) ∧ (c).

## GRAPH (5 submitted)

INDEX minutes sum to **15**. Rank order = INDEX (hardest first).

| unit | valid? | failed check | closure | files | lines | tests | XF | dangling callers | contract % | min |
|---|---|---|---:|---:|---:|---:|---|---|---:|---:|
| ifelse-apply | no | b | 7 | 2 | 103 | 3 | Y | none | 100 | 5 |
| revive-runner | no | a | 8 | 3 | 141 | 3 | Y | `GetPattern` → `Pattern` | 100 | 3 |
| config-load | no | b | 7 | 2 | 138 | 6 | Y | none | 100 | 3 |
| ifelse-branch | no | b | 6 | 2 | 87 | 4 | Y | none | 100 | 2 |
| file-exclude-filter | yes | — | 6 | 2 | 110 | 3 | Y | none | 100 | 2 |

Valid: file-exclude-filter.

(b) extras: ifelse-apply → `TestAll` (`test/golint_test.go` fixture `indent_error_flow.go`); config-load → `TestGetLintingRules`, `TestReviveCreateInstance`, `TestFileExcludeFilterAtRuleLevel`; ifelse-branch → `TestEarlyReturn`, `TestIndentErrorFlow`, `TestSuperfluousElse`, `TestAll`. revive-runner listed tests hang on a nil channel (counts as listed-fail); others green.

## NOGRAPH (5 submitted)

INDEX minutes sum to **13**.

| unit | valid? | failed check | closure | files | lines | tests | XF | dangling callers | contract % | min |
|---|---|---|---:|---:|---:|---:|---|---|---:|---:|
| ifelse-chain-walker | no | a, b | 7 | 2 | 104 | 7 | Y | `StmtBranch` → `BlockBranch` | 100 | 4 |
| revivelib-runner | yes | — | 7 | 2 | 171 | 3 | Y | none | 100 | 2 |
| package-naming-syncset | no | a | 5 | 2 | 107 | 7 | Y | `Configure` → `syncset.New` | 100 | 3 |
| var-naming | no | a, b | 6 | 2 | 267 | 2 | Y | `lint.Name` → `rule.Name` | 100 | 2 |
| file-filter | yes | — | 5 | 2 | 109 | 3 | Y | none | 100 | 2 |

Valid: revivelib-runner, file-filter.

(b) extras: ifelse-chain-walker → `TestAll`; var-naming → `TestIsUpperCaseConst`, `TestAll`, `TestJsonDataFormatVarNaming`, `TestReviveDisableDirectives_Modified`.

## Summary

| metric | GRAPH | NOGRAPH |
|---|---:|---:|
| submitted | 5 | 5 |
| valid | 1 | 2 |
| valid units per hour (1 h budget) | 1 | 2 |
| valid XF | 1 | 2 |
| mean closure size (submitted) | 6.8 | 6.0 |
| mean closure size (valid) | 6.0 | 6.0 |
| mean lines (submitted) | 115.8 | 151.6 |
| mean lines (valid) | 110 | 140 |
| mean tests (submitted) | 3.8 | 4.4 |
| mean tests (valid) | 3.0 | 3.0 |
| dangling-caller rate (units with ≥1 prod dangling / submitted) | 1/5 | 3/5 |
| time to first valid (cumulative INDEX minutes, rank order) | 15 | 6 |
| INDEX minutes sum | 15 | 13 |
| mechanical (a) pass | 4 | 2 |
| mechanical (b) pass | 2 | 3 |
| mechanical (c) pass | 5 | 5 |

(c) held on every unit. (d) 100% on every unit (coverage table names the listed tests or their assertion clusters). (e) every unit lists ≥3 properties; spot-check against GRAPH source: all have ≥3 real invariants (idempotence / last-arm / casing / exclude-disjunction / add-once / confidence filter, etc.). Two GRAPH/NOGRAPH properties overstate slightly (allow-jump length is the checker, not the walker; `TEST` compiles to unanchored `_test\.go`).

## Why units died

- **TestAll** is an umbrella golint runner over `testdata/golint/*.go`. Stubbing the if-else walker or `var-naming` fails it even when the specific rule tests are listed. GRAPH `PROOF.txt` did not run `./test`. Independent re-proof is what killed ifelse-apply / ifelse-chain-walker / var-naming on (b).
- **Shared helpers with a second production caller:** `lint.Name` wraps `rule.Name`; `PackageNamingRule.Configure` calls `syncset.New`; `StmtBranch` calls `BlockBranch` (NOGRAPH listed `BlockBranch` but not `StmtBranch`); `GetPattern` wraps `Pattern`.
- **Initialize leak:** stubbing `RuleConfig.Initialize` inside config-load breaks file-exclude tests that are not in that unit.
- **Graph undercount:** `calls` edges miss many method-on-value / intra-file sites (`HasDecls`/`IsShort` used from `checkIfElse`; `MustExclude` used from `File.lint`; `visitor.Visit` via `ast.Walk`). (a) is whatever the index records. `codegraph_explore` blast radius still names `File.lint` as a `MustExclude` user; the edge query used for (a) returned 0 callers, so file-exclude-filter / file-filter pass (a).

file-exclude-filter (GRAPH) and file-filter (NOGRAPH) are the same two files (`lint/filefilter.go`, `lint/config.go`) with essentially the same five/six functions. That valid unit was found in both conditions.

## Verdict (plain)

On this one repo, with one builder run each and the same model, codegraph did not raise valid-unit yield, mean valid closure size, or time-to-first-valid. NOGRAPH found more valid units (2 vs 1), a larger valid unit (revivelib-runner, 171 lines vs GRAPH’s 110), and the first valid unit sooner (6 vs 15 INDEX minutes). GRAPH did not find a class of hard valid units that grep+gopls missed. The walker both sides targeted is hard; neither made it valid, because neither listed `TestAll`. Do not generalize past n=1 repo × n=1 run × one model.

## Comparison to round 1

Round 1 (count objective, 1 h): NOGRAPH 5 valid bugs vs GRAPH 3; every submission was a 1-line helper inversion; 21 of 27/33 symbols shared. Verdict then: codegraph did not improve discovery of valid **mutation sites**; A4 at impact depth 2 was the wrong rejector for revive’s `test/` package.

Round 2 (quality objective, 15 min of work inside a 1 h named budget): NOGRAPH 2 valid **units** vs GRAPH 1. Direction is the same: codegraph did not help. The failure mode changed. Round 1 died on A4 (tests outside depth-2 impact). Round 2 dies on (b) collateral (`TestAll`, sibling rule tests) and (a) one-extra-caller leaks. Round 1’s “GRAPH is slower and finds fewer” pattern recurs (INDEX minutes 15 vs 13; first valid 15 vs 6). Round 1 should still not be cited as a hard-unit result; round 2 is the hard-unit measurement, and it does not reverse the verdict.

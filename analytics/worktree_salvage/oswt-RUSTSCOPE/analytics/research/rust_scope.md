# About porting the synthetic pipeline to Rust

**Go.** One codebase can serve Go and Rust if you extract a language profile and rebuild a short list of native pieces (excision, reachability, obfuscation, test-output parse). The concept already is language-agnostic. The implementation is not. This worktree is a 2026-09-19 snapshot. The architecture as it actually runs is `analytics/research/PIPELINE.md` in the main checkout (`/home/evan/Documents/open_swe_traces_research`), which this worktree does not contain.

The proof is not a theory. `Version::parse` from dtolnay/semver 1.0.27 was excised, given one property test per commitment, and run in a `rust:1.85-bookworm` image. Bare failed on `unimplemented!` panics inside the tests, not on a compile error. That is the Go analogue of `panic("excised")`. Gold passed 10/10. Cheat failed 5/10. The five cheat passes are negative `is_err()` tests. A cheat that returns an error for everything except the worked examples still satisfies those. The random valid cases are what catch it.

Do not copy the Go numeric gates. TOO-EASY, orphan-count, and the 17/54/28 gap split were fit on Go tests and Go AST. Recalibrate them on the first Rust cohort.

## What was counted

Commands, this worktree, `src/` + `scripts/` only:

| needle | files |
|---|---:|
| `go/parser` | 1 (`src/openswe_traces/synth/obfuscate_idents.go`) |
| `go test` | 18 |
| `_test.go` | 45 |
| `TestDetail` | 0 (the convention lives in later worktrees, not here) |
| `panic("excised` | 1 (`scripts/excise_funcs.py`) |

`find src/openswe_traces -name '*.py' | wc -l` is 75. The live pipeline has more, listed at the end of the inventory.

## Language profile

The smallest interface that lets one codebase serve both languages. Values below are the Go profile as the code actually uses it today.

| field | Go (reference) | Rust |
|---|---|---|
| `id` | `"go"` (`pipeline/config.py` `RepoSpec.language`) | `"rust"` |
| test command | `go test -count=1 -timeout 15m -run '^(Test…)$' ./pkg/...` (`synth/affordance.py` `render_hidden_test_sh`) | `cargo test --offline --test <name> -- --test-threads=1` |
| test-file glob | `*_test.go` | `tests/*.rs` (integration tests). `#[cfg(test)]` in `src/` is the in-tree suite, not the hidden one |
| test-function pattern | `^func\s+(?:\(.*?\)\s+)?(Test[A-Za-z0-9_]+)\s*\(` | `^fn (test_detail\d+_\w+)` plus `#[test]` |
| excision stub | `panic("excised: {name}")` (`scripts/excise_funcs.py`) | `unimplemented!("excised: {name}")` |
| build command | `go build ./...` after `go mod download` | `cargo build --offline --lib` after `cargo fetch` |
| verify / baseline | `go test ./... -count=1 -timeout 20m` (`pipeline/prepare.py`) | `cargo test --offline --lib -- --test-threads=1` |
| base image | `FROM golang:{go.mod directive}` default `1.23`, `GOTOOLCHAIN=auto` | `FROM rust:{channel}-bookworm`. Pin from `rust-version` in `Cargo.toml` |
| module manifest | `go.mod` / `go.sum` | `Cargo.toml` / `Cargo.lock` |
| reachability backend | `go/parser` in `synth/obfuscate_idents.go` and sibling `cgscan/main.go` | `syn` (equivalent job: exported defs + same-crate callees from hidden tests over gold-restored bodies) |
| unused-after-excise | blank-import (`_ "fmt"`) so the tree still compiles | `let _ = x;` or `#[allow(dead_code)]`. Unused is a **warning** in Rust, a **compile error** in Go |
| export rule | uppercase first letter | `pub` |
| hidden seed | `HIDDEN_SEED` / `rand.NewSource` (`pipeline_ext/hack_audit.py`) | `HIDDEN_SEED` parsed as `u64`, default `20260919` |
| Harbor tags | `["go", "bugfix"]` | `["rust", "bugfix"]` |

Put this in one module, `openswe_traces.pipeline.lang`, and pass it into prepare, verifier, package, affordance, harbor_tasks, hack_audit, safety, briefs, and rules. Do not sprinkle `if language == "rust"` at every `*_test.go`.

Do **not** put in the profile (rebuild in-language):

- function-body excision (`strip_go_func`, `excise_funcs.find_func_spans`)
- ident rename (`gopls rename`, `obfuscate_idents.go`)
- reachability scan (`cgscan`)
- statefulness scoring over test AST
- Go-unit catalogs (`composerver_*_batch.py` and friends)

## File-by-file inventory

Classes: **portable** (no change), **parameterise** (read the profile), **rebuild** (Rust-native implementation). Reasons cite the Go binding, not the filename.

### `src/openswe_traces/` top-level (analytics, not synth)

| path | class | reason |
|---|---|---|
| `__init__.py` | portable | version string |
| `data.py` | portable | paths + DuckDB |
| `download.py` | portable | HF snapshot |
| `verify.py` | portable | row-count check |
| `features.py` | portable | `BASH_TEST_RE` already includes `cargo test` |
| `temporal_features.py` | portable | diffs, not language |
| `score.py` | portable | logistic over traces |
| `summary.py` | portable | markdown summary |
| `difficulty.py` | portable | solve-rate buckets |
| `cheap_difficulty.py` | portable | prefix probes |
| `irt.py` | portable | 1PL/2PL |
| `kaggle.py` | portable | Kaggle CLI |
| `curve_report.py` | portable | SFT curve |
| `rungs.py` | portable | already multi-lang (`_test.go`, `.rs`, `cargo test`) |
| `rung_analysis.py` | portable | streams text through `rungs.py` |
| `results.py` | parameterise | Harbor dashboard is generic; `REPO_PREFIXES` / `CLIENT_GO_UNITS` hardcode Go families |

### `pipeline/`

| path | class | reason |
|---|---|---|
| `__init__.py` | portable | re-export |
| `cli.py` | portable | argparse |
| `yamlutil.py` | portable | YAML |
| `semaphore.py` | portable | Devin slots |
| `tokens.py` | portable | token parse |
| `state.py` | portable | sqlite resume |
| `agents.py` | portable | Cursor/Devin runner |
| `resources.py` | portable | docker concurrency |
| `reconcile.py` | portable | orphan Harbor jobs |
| `watch.py` | portable | solve poller |
| `run.py` | portable | stage order; language is in callees |
| `aggregate.py` | portable | parquet / curve |
| `audit.py` | portable | web-fetch / checksum |
| `author.py` | portable | discovers `api.md` / `gold.patch`; no toolchain |
| `ladder.py` | portable | L↔A mapping |
| `config.py` | parameterise | `RepoSpec.language: str = "go"` |
| `briefs.py` | parameterise | forbids `*_test.go`; hidden files are `*_test.go` |
| `package.py` | parameterise | `rglob("*_test.go")` to recover hidden suite |
| `solve.py` | parameterise | env signature is `*_test.go` |
| `materialize.py` | parameterise | "is this a prepared tree?" = `go.mod` exists |
| `safety.py` | parameterise | skeleton writes `skel_bb_prop_test.go`, `FROM golang:1.23`, `tags = ["go", "bugfix"]` |
| `verifier.py` | parameterise | copies `*_test.go`, fallback `go test ./... -count=1` |
| `prepare.py` | parameterise | entire file is a Go profile: `FROM golang:{go_image}`, `go mod download`, `go build ./...`, `ok`/`FAIL` baseline parse |

### `pipeline_ext/`

| path | class | reason |
|---|---|---|
| `__init__.py` | portable | re-export |
| `author_meta.py` | portable | `predicted_flip: L<k>` |
| `calibration.py` | portable | pass-rate flags (the **thresholds** are Go-fit; see risks) |
| `controls.py` | portable | control picker |
| `ladder_policy.py` | portable | climb `[L2,L5,L6]` |
| `timeouts.py` | portable | agent budgets |
| `hack_audit.py` | parameterise | B9 idea is portable; forbid `*_test.go`/`go.mod`/`go.sum`, match `func Test\w+\(t \*testing.T\)`, emit Go `HiddenSeed`, parse `go test` out of `test.sh` |

### `sft/`

| path | class | reason |
|---|---|---|
| `__init__.py` | portable | docstring |
| `sample.py` | portable | JSONL |
| `manifests.py` | portable | SFT manifests |

### `synth/` (hottest)

| path | class | reason |
|---|---|---|
| `__init__.py` | portable | docstring |
| `rules.py` | parameterise | engine is portable; `_TEST_FUNC_RE`, `*_test.go`, A12 `*_test.go` hunks, B4 lowercase-method = unexported (Go visibility), B5 `rand.New` |
| `affordance.py` | parameterise | ladder copy is portable; `render_hidden_test_sh` emits `go test -run`, `-race`, `-bench` ns/op, `FROM golang:1.23`. `strip_go_func` inside this file is rebuild-grade |
| `harbor_tasks.py` | parameterise | `PATH=/usr/local/go/bin`, `go.mod` init, `*_test.go`, default `go test -count=1 -timeout 15m`, `FROM golang:1.23` |
| `pipeline_repos.py` | parameterise | host prep: `FROM golang:1.23`, `go mod download`, keep if `go test ./...` has ≥5 passing packages |
| `repos2.py` | parameterise | extra Go hosts, same Dockerfile |
| `bigl0.py` | parameterise | packs `*_bb_prop_test.go` + in-image `go test` |
| `composerver_batch.py` | rebuild | client-go unit catalog; do not port, write Rust catalogs |
| `pipeline_clientgo_batch.py` | rebuild | same |
| `composerver_gin_batch.py` | rebuild | gin catalog |
| `composerver_goa_batch.py` | rebuild | goa catalog |
| `composerver_helm_batch.py` | rebuild | helm catalog |
| `composerver_kops_batch.py` | rebuild | kops catalog |
| `ablation2.py` | rebuild | revive-specific Go globs and `go test ./lint/` |
| `ladder_deep.py` | rebuild | copies L2 trees with `go test -race` instructions and Go catchers |
| `codegraph_bugs.py` | rebuild | Go export = uppercase, `*_test.go`, `parse_go_test_output` (`--- FAIL:` / `FAIL\t`), `go build` / `go test` / `go tool cover` |
| `difficulty.py` | rebuild | site pickers on Go tests: `Benchmark*`, `b.N`, `go test -bench` ns/op, `go test -race`, `sync.Mutex` |
| `excision_recover.py` | rebuild | reverse-gold + strip `func Test\w+` from `*_test.go`; refuses trees without `go.mod` |
| `obfuscate.py` | rebuild | `gopls rename`, `go run obfuscate_idents.go` (`go/parser`), `go mod tidy` offline |
| `obfuscate_idents.go` | rebuild | the `go/parser` reachability helper |
| `statefulness.py` | rebuild | scores **Go test source**: `func Test`, `t.Fatal`, `go func` / `sync.` / `time.Sleep` |
| `spec_reimpl_bb.py` | rebuild | mutates client-go codec Go |
| `ladder2.py` | rebuild | `replace_go_func` / `strip_go_func` on client-go |
| `ladder2b.py` | rebuild | 1PC Go rewrite |
| `scoretest.py` | rebuild | ranks client-go closures via Go tests |
| `scoretest_build.py` | rebuild | builds those with `replace_go_func` |
| `two_repo.py` | rebuild | parses `github.com/tikv/client-go/v2` imports, `go test -ldflags=-checklinkname=0` |

### Scripts that touch language

Thin CLIs inherit the class of the library they call. Scripts that **contain** the logic are listed as themselves.

| path | class | reason |
|---|---|---|
| `scripts/excise_funcs.py` | rebuild | Go `func` regex + brace walker; stubs `panic("excised: {qname}")`; `gofmt`; blank-imports |
| `scripts/obfuscate_task.py` | rebuild | wraps `synth.obfuscate` (`go/parser` + gopls) |
| `scripts/recover_excisions.py` | rebuild | wraps `excision_recover` |
| `scripts/harbor_tasks.py` | parameterise | wraps `synth.harbor_tasks` |
| `scripts/prepare_pipeline_repos.py` | parameterise | wraps `pipeline_repos` |
| `scripts/prepare_repos2.py` | parameterise | wraps `repos2` |
| `scripts/materialize_tasks.py` | parameterise | wraps `pipeline.materialize` (`go.mod` probe) |
| `scripts/pipeline.py` | portable | argparse over `pipeline.run`; language is in callees |
| `scripts/ops/rebuild_bases.sh` | parameterise | picks `golang:` image, runs `go test ./... -count=1 -timeout 20m` |
| `scripts/build_*.py` (unsolv, bigl0, ladder2, ladder2b, ladder_deep, ablation2, spec_reimpl_bb, scoretest, composerver_*, pipeline_clientgo) | rebuild | Go unit builders |
| `scripts/codegraph_bugs.py` | rebuild | wraps `synth.codegraph_bugs` |
| `scripts/openswe_synth.py` | rebuild | wraps `synth.difficulty` |
| `scripts/two_repo.py` | rebuild | wraps `synth.two_repo` |
| `scripts/pipeline_ext_hack_audit.py` | parameterise | wraps `hack_audit` |
| `scripts/pipeline_ext_{ladder_policy,calibration,controls,author_meta,timeouts}.py` | portable | no toolchain |
| `scripts/ops/docker_cleanup.sh` | portable | incidental `KEEP_BASE=…golang:1.23` |

Analytics CLIs (`cheap_difficulty.py`, `fit_irt.py`, `download_data.py`, …) do not touch the synth language binding. Leave them.

### Live pipeline files not in this worktree

A port of "this snapshot only" would miss the DETAILS gate, which PIPELINE.md calls the highest-leverage stage. These exist in sibling worktrees / the main checkout and must be in the port target:

| path | where | class | reason |
|---|---|---|---|
| `src/openswe_traces/details_gate.py` | oswt-DETAILSGATE | parameterise | parses `DETAILS.md` / `Inferable:`; no toolchain |
| `scripts/ops/task_lint.py` | main checkout | parameterise | `ASSERT_RE = t.(Fatalf\|…)`, `TESTFUNC_RE = func (Test\w+)`, A12 `*_test.go`; TOO-EASY counts those |
| `src/openswe_traces/cgscan/main.go` | oswt-GATEALL | rebuild | `go/parser` reachability. Rust twin is `syn` |
| `src/openswe_traces/cg_coverage.py` | oswt-GATEALL | parameterise | drives cgscan |
| `src/openswe_traces/gate/excision.py` | oswt-GATEALL | parameterise | `panic("excised: X")` regex; already lists `.rs` as a scanned suffix |
| `scripts/ops/contract_gap_read.py` | main checkout | portable | contract vs suite, English |
| `scripts/ops/assertion_density.py` | main checkout | parameterise | `t.Fatalf` / `func (Test\w+)` |
| `scripts/ops/reconcile_inferable.py` | main checkout | portable | `Inferable:` vs ARBITRARY |

### Counts in this worktree

| class | n (py under `src/openswe_traces`) |
|---|---:|
| portable | 38 |
| parameterise | 26 |
| rebuild | 11 py + `obfuscate_idents.go` |

The 11 rebuild py files are the ones that parse or mutate Go. The 26 parameterise files are the overnight pipeline. That is the work.

## Proof unit

Crate: [dtolnay/semver](https://github.com/dtolnay/semver) 1.0.27, commit `6ed8561154715b2c34df417a2052597d586f2c43`. Parser, not orchestration. Same class of surface that beat orchestration on the Go bank (helm strvalsparser, kops tomlwriter, gin formmapping).

Closure: one function body, `FromStr for Version` in `src/parse.rs`. `Version::parse` delegates to it. Helpers stay in the tree. That is a **machinery** proof. A bank unit would stub the helper closure too, or the solver reads `numeric_identifier` and writes the body back.

Layout: `experiments/pipeline/rust_proof/semver-parse/`. Rebuild with `uv run python outputs/rust_proof/build_unit.py`. In-image preflight: `bash outputs/rust_proof/preflight.sh`.

Artifacts:

- `_author/DETAILS.md`. 10 numbered commitments with `Inferable:` yes or partially.
- `_author/api.md`, `bugreport.md`, `contract.md` (one coverage row per hidden test)
- `_author/gold.patch` and `cheat.patch` (also under `tests/`)
- `tests/hidden/hidden_details.rs`. `test_detail01` through `test_detail10`, seeded `HIDDEN_SEED` default 20260919.
- `tests/test.sh`. Checksum-guard, copy into `tests/`, `cargo test --offline --test hidden_details`.
- `environment/Dockerfile`. `FROM rust:1.85-bookworm`, `cargo fetch && cargo build --offline --lib`.
- excision stub: `unimplemented!("excised: Version::from_str")`

Cheat hardcodes the three worked examples (`1.2.3`, `1.2.3-alpha.1`, `1.2.3+build.5`) and returns `ErrorKind::Empty` for everything else. Negative tests that only check `is_err()` still pass. The random valid cases do not. That is A3.

### In-image preflight, verbatim

Image `rust-proof:semver-parse`, `--network none`, logs in `experiments/pipeline/rust_proof/semver-parse/preflight/`.

```
BARE: FAIL_ASSERTIONS (docker_rc=1)
GOLD: PASS (docker_rc=0)
CHEAT: FAIL_ASSERTIONS (docker_rc=1)

bare reward: REWARD=0
gold reward: REWARD=1
cheat reward: REWARD=0

bare test result: test result: FAILED. 0 passed; 10 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.00s
gold test result: test result: ok. 10 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.00s
cheat test result: test result: FAILED. 5 passed; 5 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.00s
```

Bare is a panic, not a compile error:

```
thread 'test_detail01_empty_is_error' panicked at src/parse.rs:30:9:
not implemented: excised: Version::from_str
```

`grep -E 'could not compile|error\[E' preflight/bare.log` is empty. `PREFLIGHT_CLASS=assert_fail` on bare and cheat, `pass` on gold.

Gold:

```
running 10 tests
test test_detail01_empty_is_error ... ok
…
test test_detail10_pre_then_build ... ok
test result: ok. 10 passed; 0 failed; …
PREFLIGHT_CLASS=pass
REWARD=1
```

A first `bash -lc` attempt dropped `cargo` off PATH because a login shell resets PATH. `preflight.sh` now uses `bash -c` and exports `/usr/local/cargo/bin`. The rust image's ENV PATH is not login-safe.

This unit would likely pass an L0 screen. The crate docs state most of the ten commitments in `/// Syntax`. The proof is that the Harbor-shaped loop (excise, hidden tests, gold, cheat, in-image) works for `cargo test`. It is not a certified-hard unit.

## Cost

| class | what | effort |
|---|---|---|
| portable | leave it | 0 |
| parameterise | `LanguageProfile` + thread through ~26 py files and their tests | 4 to 6 engineer-days |
| rebuild, excision | `syn` walker, `unimplemented!` stub, `rustfmt`, dead_code allow | 2 to 3 days for the Go equivalent of `excise_funcs.py` |
| rebuild, reachability | `syn` cgscan: hidden `#[test]` roots, gold-restored bodies, exported defs | 4 to 7 days. Rust macros and trait methods will lie more than `go/parser` |
| rebuild, obfuscate | crate-name + string replace is cheap; rust-analyzer rename is not | skip full rename for v1 (1 day identity-only). Full rename 1 to 2 weeks |
| rebuild, statefulness | `#[test]` / `assert!` / `tokio::spawn` analogue of `statefulness.py` | 1 to 2 days |
| rebuild, catalogs | do not port gin/helm/kops batches | 0. Write new Rust unit lists |
| first 20 authored Rust units | author + verifier + preflight, same recipe as Go | 1 to 2 weeks of agent time, not tooling |
| recalibrate gates | TOO-EASY, orphans, gap split on a Rust cohort | after 30 to 50 L0-screened units |

Do the profile PR first. Then excision + `test.sh` + preflight (already sketched here). Then `syn` cgscan. Then a real bank. Do not start with obfuscation.

## What will not transfer

The 17/54/28 ARBITRARY/DERIVABLE/COUNTER split is a claim about English contracts versus hidden assertions. The authoring recipe can stay. The **percentages** were measured on Go units whose docs are thinner than a typical `///` block. Rust crates document syntax in the type rustdoc (semver's `Version` docs already list leading zeros, whitespace, u64 bounds). Expect more `Inferable: yes`, more L0 passes, a smaller certified-hard yield per authored unit. Recount. Do not quote 17/54/28 for Rust.

TOO-EASY (`assertions <= 8` and `testfns <= 5` is a 9/9 L0-easy drop on Go) counts `t.Fatalf` and `func Test`. Rust tests use many `assert!` per `#[test]`. The same commitment set produces a different pair of numbers. Our proof unit has 10 test functions and far more than 8 asserts. The threshold as written would not fire, and a 3-test Rust suite with 20 `assert!` would look "hard" to the Go rule and easy to a solver. Refit on Rust L0 outcomes. Do not copy 8 and 5.

Orphan-count (cgscan, 53% precision at ≥2 orphans vs 39% base on Go) is a Go AST measurement. `syn` will miss trait methods, `Deref` targets, and macro-expanded calls, and will invent others. The GATEALL combined gate did **not** beat the Go base rate (36% vs 37%). Do not treat orphan ≥2 as a Rust packaging gate until you measure it on Rust units.

Other shifts that are structural, not calibration:

- Unused imports. Go excision has to blank-import or the tree does not compile. Rust warns (`dead_code` on `dot` and `ErrorKind::Empty` in the excised semver lib) and still builds. A sloppy stub that leaves the old helpers sitting there still preflights. That hides incomplete closures.
- Doctests. `cargo test` without `--lib`/`--test` compiles examples in rustdoc. `go test` does not. Hidden `test.sh` must not run doctests.
- Features. `Cargo.toml` `[features]` changes the exported surface. `go build ./...` has build tags, but they showed up less in our bank.
- Proc macros. Do not author units whose closure is a derive. `syn` can parse them. The solver cannot reasonably reimplement them from a Harbor instruction.
- Race. `go test -race` is one flag. Thread sanitizer on stable Rust is not. Drop A9 race units for v1.
- Integration tests in `tests/` only see `pub` items. That is the B4 black-box default, and it is stricter than `package foo` (white-box) Go tests. Prefer `tests/*.rs` for hidden suites.

What **does** transfer: one canonical runner, compile failure that is a hard stop (when it happens), no import-resolution ambiguity, parse-level reachability via `syn`, the DETAILS → one-test-per-commitment → derived contract recipe, in-image preflight as the A8 gate, and the information barriers in PIPELINE.md.

## Recommendation

Go. Extract `LanguageProfile`, keep the Go bank running, add Rust as a second profile. Rebuild excision and `test.sh` first (the semver unit is the fixture those PRs must keep green). Rebuild `syn` cgscan before you author a Rust bank, or you will fly without A4. Skip gopls-style rename. Recalibrate TOO-EASY and orphans on the first Rust L0 screens. Do not announce an 86-unit Rust bank until those gates are measured, not copied.

Rerun the proof:

```
uv run python outputs/rust_proof/build_unit.py
bash outputs/rust_proof/preflight.sh
cat experiments/pipeline/rust_proof/semver-parse/preflight/verdicts.txt
```

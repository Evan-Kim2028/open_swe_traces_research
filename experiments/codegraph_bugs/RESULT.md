# codegraph_bugs — pilot result (2026-09-18)

Question: can `codegraph` callers/impact steer **cross-file** Go bug injection with a valid fail-to-pass set?

**Verdict:** **9 valid cross-file bugs per hour** of agent time (6 / ~40 min). Codegraph’s impact-set check **was useful**: it rejected the two logger mutations (collateral `TestBackoffWithMax`) and kept the six targeted mutations (f2p files ⊆ impact, no collateral, not flaky). Isolated-package hosts are a dead end for this method.

## Environment spec (SWE-rebench-V2)

Traces parquet metadata has `{category, model_patch, reference_patch}` only — **no `base_commit`**. Loaded the two instance rows from `nebius/SWE-rebench-V2` (`data/train-00000-of-00001.parquet`, filtered by `instance_id`).

**The HF row does not ship a Dockerfile.** Each row has:

| field | role |
|---|---|
| `image_name` | prebuilt image `docker.io/swerebenchv2/<repo>:<issue>-<sha7>` |
| `install_config.base_image_name` | alias (`go_1.19.13`, `go_1.23.8`) |
| `install_config.docker_specs.go_version` | version hint (`1.19.8`, `1.23.2`) |
| `install_config.install` | shell steps (`apt-get`, `go mod download` / `go mod init`) |
| `install_config.test_cmd` | `go test -v ./...` or `go test -v ./internal/locate/...` |
| `install_config.log_parser` | `parse_log_gotest` |

Base Dockerfiles and log parsers live in https://github.com/SWE-rebench/SWE-rebench-V2, not in the parquet.

| instance_id | repo | base_commit | test_cmd |
|---|---|---|---|
| `vaskoz__dailycodingproblem-go-106` | vaskoz/dailycodingproblem-go | `6d51a3009ad67aac201d740fdb2fdc00ab0fb9bf` | `go test -v ./...` |
| `tikv__client-go-1183` | tikv/client-go | `190f0cce536f835b72481f7bcd0a9448bb1e5202` | `go test -v ./internal/locate/...` |

## Host selection

`outputs/task_difficulty.parquet` ⋈ `outputs/proxy_features.parquet`, `language='go'`, mid-share ≥ 20%, top by `n_instances`:

| repo | n_instances | n_mid | mid_share |
|---|---:|---:|---:|
| vaskoz/dailycodingproblem-go | 355 | 198 | 0.558 |
| tikv/client-go | 30 | 7 | 0.233 |

## Per-repo build / test / index

Go 1.23.1. `go build ./...` ok on both. Timeout 15 min. No extra network services required for the default `./...` tests (client-go `integration_tests/` is a **separate module**, not part of `go test ./...`).

| repo | go files | build | `go test ./...` | index wall | files | nodes | edges |
|---|---:|---|---|---:|---:|---:|---:|
| dailycodingproblem-go | 98 | ok | **FAIL `day6` only** (`TestXorList`; 2018 unsafe XOR list vs Go 1.23). All other packages green. | 1.28 s (311 ms index) | 99 | 555 | 982 |
| client-go | 209 | ok | **all green** (~91 s) | 3.12 s (639 ms index) | 214 | 5,856 | 21,031 |

Daily needed `go mod init dailycodingproblem-go` (matches SWE-rebench install). Baseline failure subtracted in validation: `TestXorList`.

## Candidate table

Ranking uses **per-definition** `calls` edges with confidence ≥ 0.85 (CLI `codegraph callers <name>` over-merges `String`/`Len`/`Match`). Skip `examples/` and `integration_tests/`. Cover from `go test -coverpkg=./...` + `go tool cover -func`.

**dailycodingproblem-go strict (≥2 packages): 0.** Each `dayN` is an isolated package; high-confidence callers never leave it. That is the main negative result.

**client-go top 10** (injected all):

| rank | symbol | file | caller files | pkgs | cover % |
|---:|---|---|---:|---:|---:|
| 1 | BgLogger | internal/logutil/log.go | 32 | 16 | 100 |
| 2 | NewBackofferWithVars | config/retry/backoff.go | 21 | 8 | 100 |
| 3 | NewRequest | tikvrpc/tikvrpc.go | 18 | 8 | 100 |
| 4 | EvalFailpoint | util/failpoint.go | 17 | 11 | 100 |
| 5 | Logger | internal/logutil/log.go | 15 | 9 | 67 |
| 6 | LocateKey | internal/locate/region_cache.go | 11 | 7 | 100 |
| 7 | NewRegionRequestSender | internal/locate/region_request.go | 10 | 4 | 100 |
| 8 | GetGlobalConfig | config/config.go | 9 | 7 | 100 |
| 9 | IsErrNotFound | error/error.go | 9 | 3 | 100 |
| 10 | IsFakeRegionError | internal/locate/region_request.go | 9 | 3 | 100 |

Relaxed daily (intra-package, for pipeline smoke; **not** counted in the yield): TwoSumBest, TwoSumBrute, Match.

## Per-bug validation

Mutation is 1–5 lines. Valid = builds ∧ f2p≥1 ∧ f2p ⊆ impact ∧ no collateral ∧ not flaky.

### client-go (strict)

| symbol | bug | builds | f2p | inside impact | collateral | flaky | hops | impact files | valid | t_val (s) |
|---|---|---|---:|---|---|---|---:|---:|---|---:|
| BgLogger | return nil logger | Y | 6 | N | TestBackoffWithMax | N | 2 | 101 | **no** | 13 |
| Logger | invert ctx ok | Y | 4 | N | TestBackoffWithMax | N | 2 | 93 | **no** | 86 |
| NewBackofferWithVars | drop `.withVars` | Y | 2 | **Y** | none | N | 1 | 69 | **yes** | 117 |
| NewRequest | swallow RPC context | Y | 1 | **Y** | none | N | 1 | 43 | **yes** | 128 |
| EvalFailpoint | invert enabled | Y | 0 | — | — | — | — | — | **no** (hung ~6 min, killed) | 400 |
| LocateKey | `isEnd=true` | Y | 0 | — | — | — | — | 38 | **no** (not caught) | 61 |
| NewRegionRequestSender | `client: nil` | Y | 1 | **Y** | none | N | 1 | 33 | **yes** | 102 |
| GetGlobalConfig | return `&Config{}` | Y | 4 | **Y** | none | N | 2 | 66 | **yes** | 116 |
| IsErrNotFound | invert `errors.Is` | Y | 4 | **Y** | none | N | 1 | 19 | **yes** | 71 |
| IsFakeRegionError | invert empty-regions | Y | 2 | **Y** | none | N | 1 | 16 | **yes** | 124 |

Fail-to-pass names for valid bugs:

- NewBackofferWithVars: `TestBackoffDeepCopy`, `TestRegionRequestToThreeStores`
- NewRequest: `TestRegionRequestToThreeStores`
- NewRegionRequestSender: `TestRegionRequestToThreeStores`
- GetGlobalConfig: `TestPanicInRecvLoop`, `TestRecvErrorInMultipleRecvLoops`, `TestRegionCache`, `TestRawKV`
- IsErrNotFound: `TestPipelinedFlushGet`, `TestUnionStoreGetSet`, `TestUnionStoreDelete`, `TestBufferBatchGetter`
- IsFakeRegionError: `TestRegionRequestToThreeStores`, `TestRegionRequestToSingleStore`

### daily (relaxed)

| symbol | bug | builds | f2p | inside impact | collateral | flaky | hops | impact files | valid |
|---|---|---|---:|---|---|---|---:|---:|---|
| TwoSumBest | wrong map key `seen[v]` | Y | 2 | Y | none | N | 1 | 2 | yes (same pkg) |
| TwoSumBrute | invert `i!=j` | Y | 1 | Y | none | N | 1 | 2 | yes (same pkg) |
| Match | invert `finished` | Y | 1 | Y | none | N | 1 | 4 | yes (same pkg) |

## Verifier breadth (3 bugs)

Applied on top of the **buggy** tree. Alt = different correct implementation. Cheat = special-case one test input.

| bug | alternative | f2p + alt | cheat | f2p + cheat |
|---|---|---|---|---|
| TwoSumBest | `return TwoSumBetter(numbers, k)` | **accept** (`go test ./day1` ok) | `if k==201 { return true }` then still-buggy hash | **reject** (`TestTwoSumsLarge/Best` fails) |
| Match | inline `ii==len(input) && pi==len(pattern)` | **accept** (`./day25` ok) | special-case `input=="ray"` | **reject** (`TestMatch` fails) |
| NewBackofferWithVars | `b := NewBackoffer(...); return b.withVars(vars)` | **accept** (`config/retry` + `internal/locate` targeted tests ok) | `if maxSleep==4 { withVars }` | **reject** (`TestRegionRequestToThreeStores` still fails) |

Patches: `bugs/<repo>/<symbol>.{alt,cheat}.patch`.

## Difficulty priors (valid cross-file only)

Hops = shortest reverse-`calls` path from injected symbol to a failing test file.

| symbol | hops | impact files | interpretation |
|---|---:|---:|---|
| IsFakeRegionError | 1 | 16 | test calls the helper |
| IsErrNotFound | 1 | 19 | test calls the helper |
| NewRegionRequestSender | 1 | 33 | constructor used in locate tests |
| NewRequest | 1 | 43 | request factory on the RPC path |
| GetGlobalConfig | 2 | 66 | one hop through config readers |
| NewBackofferWithVars | 1 | 69 | backoff ctor; wide blast radius |

Narrower impact + hops=1 (IsErrNotFound, IsFakeRegionError) should be the cheap end of a later IRT prior; GetGlobalConfig/NewBackofferWithVars are “easy hops, fat blast radius”.

## Time spent

| phase | wall |
|---|---|
| Host pick + HF env spec + clone | ~8 min |
| `go build`/`go test` + cover | ~5 min daily, ~2 min client-go build, ~91 s client-go test, ~66 s cover |
| `codegraph init` | 1.3 s + 3.1 s |
| Candidate ranking | ~5 s client-go / ~13 s daily |
| Write 13 patches | ~10 min |
| Validate 10 client-go + 3 daily | ~21 min (EvalFailpoint hang dominated) |
| Verifier on 3 | ~4 min |
| **Agent wall (this session)** | **~40 min** |

Per-bug validation times are the `t_val` column above. Patch writing was ~1 min each once candidates were chosen.

**Yield:** 6 valid cross-file bugs / 0.67 h ≈ **9 / hour**. Hit rate 6/10 injected client-go candidates (60%). Daily strict yield 0/0.

## Did codegraph make the impact-set check useful?

Yes, with two caveats.

1. **It filtered logger mutations.** BgLogger/Logger produce panics in many tests, but `TestBackoffWithMax` is collateral relative to the impact file set (duplicate test names in `config/retry` vs `internal/client/retry`). Those would be noisy SWE tasks.
2. **The six valid bugs have f2p tests inside impact and zero collateral** — the intended “fails for the graph reason” check.

Caveats: CLI `callers <short name>` is not safe for `String`/`Len`; we ranked from the sqlite `calls` edges at confidence ≥ 0.85. `codegraph affected` returned no tests for `region_cache.go` (Go tests live next to code, not in a `*_test` glob the tool expected). LocateKey’s `isEnd` flip compiled and was fully covered but **no test failed** — coverage ≠ a useful f2p. EvalFailpoint inversion hung the suite (retry loops); skip test-infra symbols.

## Reproduce

See `experiments/codegraph_bugs/README.md`. Package: `src/openswe_traces/synth/codegraph_bugs.py`. CLI: `uv run python scripts/codegraph_bugs.py`. Tests: `uv run pytest tests/test_codegraph_bugs.py` (5 passed). Ruff clean.

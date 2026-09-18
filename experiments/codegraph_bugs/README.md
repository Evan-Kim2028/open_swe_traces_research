# codegraph_bugs — Go bug-injection pilot

## Question

Does indexing a host repo with `codegraph` let us inject **cross-file, difficulty-steered**
semantic bugs that already have a valid fail-to-pass test set (failing tests sit inside
`codegraph impact`, tests outside stay green)?

## Config

| item | value |
|---|---|
| Hosts | `vaskoz/dailycodingproblem-go` @ `6d51a30`, `tikv/client-go` @ `190f0cc` (most Go instances with mid-share ≥ 20%) |
| Go | 1.23.1 (`/usr/local/go/bin/go`) |
| Codegraph | CLI 1.6.0 (`codegraph init -y && codegraph index`) |
| Selection | exported funcs/methods, callers in ≥2 packages, cover > 0, skip `examples/` and `integration_tests/` |
| Mutation | 1–5 line semantic bugs (not syntax errors) |
| Valid | builds, f2p ≥ 1, f2p files ⊆ impact set, no collateral, not flaky |

## Reproduce

```bash
cd <repo root>
uv run python scripts/codegraph_bugs.py select-hosts
uv run python scripts/codegraph_bugs.py fetch-env \
  vaskoz__dailycodingproblem-go-106 tikv__client-go-1183

# after clone + `codegraph init -y` + `go test -coverprofile` + `go tool cover -func`:
uv run python scripts/codegraph_bugs.py candidates \
  experiments/codegraph_bugs/repos/client-go \
  --cover-func experiments/codegraph_bugs/out/client-go.func --top 10

uv run python scripts/codegraph_bugs.py validate \
  experiments/codegraph_bugs/repos/client-go \
  experiments/codegraph_bugs/bugs/client-go/IsErrNotFound.patch \
  IsErrNotFound

uv run pytest tests/test_codegraph_bugs.py
uv run ruff check src/openswe_traces/synth scripts/codegraph_bugs.py tests/test_codegraph_bugs.py
```

## Outputs

| Path | Contents |
|---|---|
| `RESULT.md` | Findings for this pilot |
| `bugs/<repo>/<symbol>.patch` | Injected bugs (plus `.alt` / `.cheat` for 3 verifiers) |
| `fixture_host/` | Tiny 2-package Go module used by pytest |
| `out/` | cover profiles, candidate JSON, per-bug validation JSON (gitignored) |
| `repos/` | Cloned hosts at the SWE-rebench-V2 base commit (gitignored) |

Package logic: `src/openswe_traces/synth/codegraph_bugs.py`. Thin CLI: `scripts/codegraph_bugs.py`.

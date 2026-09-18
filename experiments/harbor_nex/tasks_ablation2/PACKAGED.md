# Ablation round 2 — packaged Harbor tasks

Date: 2026-09-18. VALID units from `experiments/ablation_graph/RESULT2.md`
(judge: (a) closure ∧ (b) self-containment ∧ (c) black-box). Host tree:
`experiments/ablation_graph/repos/revive-graph` (no `.git` / `.codegraph` /
`units/` / `bugs/`). Gold = revert the unit's `excision.patch`.
Affordance A0 and A1 only. **No Harbor jobs were launched.**

Build:

```
uv run python scripts/build_ablation2.py
```

Layout: `experiments/harbor_nex/tasks_ablation2/<cond>-<unit>-A{0,1}/`.
Hidden tests live under `tests/hidden/` and are checksum-guarded by `tests/test.sh`.
`task.toml` `[agent]` allowlist + `[verifier]` no-network. Instruction is the
unit `contract.md` rewritten at locality L2 (no test/symbol/file names) plus a
reproduce command and the no-web clause. A1 adds hidden test names and one-liners.

| task | cond | unit | level | hidden tests | image | buggy | gold |
|---|---|---|---:|---|---|---|---|
| `graph-file-exclude-filter-A0` | graph | `file-exclude-filter` | 0 | `TestFileFilter`, `TestFileExcludeFilterAtRuleLevel`, `TestGetConfig` | `harbor-ablation2-graph-file-exclude-filter:latest` | FAIL (want) | PASS (want) |
| `graph-file-exclude-filter-A1` | graph | `file-exclude-filter` | 1 | `TestFileFilter`, `TestFileExcludeFilterAtRuleLevel`, `TestGetConfig` | `harbor-ablation2-graph-file-exclude-filter:latest` | FAIL (want) | PASS (want) |
| `nograph-revivelib-runner-A0` | nograph | `revivelib-runner` | 0 | `TestReviveLint`, `TestReviveFormat`, `TestReviveCreateInstance` | `harbor-ablation2-nograph-revivelib-runner:latest` | FAIL (want) | PASS (want) |
| `nograph-revivelib-runner-A1` | nograph | `revivelib-runner` | 1 | `TestReviveLint`, `TestReviveFormat`, `TestReviveCreateInstance` | `harbor-ablation2-nograph-revivelib-runner:latest` | FAIL (want) | PASS (want) |
| `nograph-file-filter-A0` | nograph | `file-filter` | 0 | `TestFileFilter`, `TestFileExcludeFilterAtRuleLevel`, `TestGetConfig` | `harbor-ablation2-nograph-file-filter:latest` | FAIL (want) | PASS (want) |
| `nograph-file-filter-A1` | nograph | `file-filter` | 1 | `TestFileFilter`, `TestFileExcludeFilterAtRuleLevel`, `TestGetConfig` | `harbor-ablation2-nograph-file-filter:latest` | FAIL (want) | PASS (want) |

## Proofs (built image, A8)

Each A0 environment was `docker build` from `golang:1.23` with modules
pre-downloaded (`GOTOOLCHAIN=auto`). Verifier ran with `--network=none`.
A1 shares the A0 image (same tree, same `tests/test.sh`).

### `graph-file-exclude-filter`

- harness_ok: `True`
- buggy_fails: `True` (rc=1 reward='0')
- gold_restore: `True` (rc=0 reward='1')

Buggy tail:

```
d match a/x/xxx.pb.go
    --- FAIL: TestFileFilter/just_* (0.00s)
        filefilter_test.go:108: should match pb.go
    --- FAIL: TestFileFilter/just_~ (0.00s)
        filefilter_test.go:121: should match pb.go
FAIL
FAIL	github.com/mgechev/revive/lint	0.003s
--- FAIL: TestFileExcludeFilterAtRuleLevel (0.00s)
    --- FAIL: TestFileExcludeFilterAtRuleLevel/not_called_if_exclude_not_match (0.00s)
        file_filter_test.go:49: should not call rule if excluded
FAIL
FAIL	github.com/mgechev/revive/test	0.003s
--- FAIL: TestGetConfig (0.00s)
    --- FAIL: TestGetConfig/ok (0.00s)
        --- FAIL: TestGetConfig/ok/rule-level_file_filter_excludes (0.00s)
            getconfig_bb_test.go:397: r2 should be initialized and exclude some/file.go
FAIL
FAIL	github.com/mgechev/revive/config	0.004s
FAIL

```

Gold tail:

```
ok  	github.com/mgechev/revive/lint	0.003s
ok  	github.com/mgechev/revive/test	0.004s
ok  	github.com/mgechev/revive/config	0.004s

```

### `nograph-revivelib-runner`

- harness_ok: `True`
- buggy_fails: `True` (rc=1 reward='0')
- gold_restore: `True` (rc=0 reward='1')

Buggy tail:

```
})
	/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.0.linux-amd64/src/testing/testing.go:1974 +0x232
testing.tRunner.func1()
	/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.0.linux-amd64/src/testing/testing.go:1977 +0x349
panic({0x5fef00?, 0x81ad90?})
	/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.0.linux-amd64/src/runtime/panic.go:860 +0x13a
github.com/mgechev/revive/revivelib.TestReviveCreateInstance(0x2c30aec7a6c8)
	/app/revivelib/core_internal_test.go:13 +0x2f
testing.tRunner(0x2c30aec7a6c8, 0x663508)
	/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.0.linux-amd64/src/testing/testing.go:2036 +0xea
created by testing.(*T).Run in goroutine 1
	/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.0.linux-amd64/src/testing/testing.go:2101 +0x4c5
FAIL	github.com/mgechev/revive/revivelib	0.006s
FAIL

```

Gold tail:

```
ok  	github.com/mgechev/revive/revivelib	0.034s

```

### `nograph-file-filter`

- harness_ok: `True`
- buggy_fails: `True` (rc=1 reward='0')
- gold_restore: `True` (rc=0 reward='1')

Buggy tail:

```
d match a/x/xxx.pb.go
    --- FAIL: TestFileFilter/just_* (0.00s)
        filefilter_test.go:108: should match pb.go
    --- FAIL: TestFileFilter/just_~ (0.00s)
        filefilter_test.go:121: should match pb.go
FAIL
FAIL	github.com/mgechev/revive/lint	0.002s
--- FAIL: TestFileExcludeFilterAtRuleLevel (0.00s)
    --- FAIL: TestFileExcludeFilterAtRuleLevel/not_called_if_exclude_not_match (0.00s)
        file_filter_test.go:49: should not call rule if excluded
FAIL
FAIL	github.com/mgechev/revive/test	0.005s
--- FAIL: TestGetConfig (0.00s)
    --- FAIL: TestGetConfig/ok (0.00s)
        --- FAIL: TestGetConfig/ok/rule-level_file_filter_excludes (0.00s)
            getconfig_bb_test.go:397: r2 should be initialized and exclude some/file.go
FAIL
FAIL	github.com/mgechev/revive/config	0.006s
FAIL

```

Gold tail:

```
ok  	github.com/mgechev/revive/lint	0.004s
ok  	github.com/mgechev/revive/test	0.004s
ok  	github.com/mgechev/revive/config	0.005s

```

## Rules registry

`validation.json` is written by `openswe_traces.synth.rules.write_task_validation`.
A2/A3/A9/A11 skipped (no alt/cheat/perf). B5 recorded as fail: hidden tests are
the unit's existing example black-box tests, not seeded properties.


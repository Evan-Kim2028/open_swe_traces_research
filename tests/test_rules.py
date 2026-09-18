from __future__ import annotations

import json
from pathlib import Path

from openswe_traces.synth.harbor_tasks import build_task
from openswe_traces.synth.obfuscate import NO_WEB_CLAUSE
from openswe_traces.synth.rules import (
    RULE_IDS,
    RULES,
    RuleVerdict,
    backfill_task,
    evaluate_rules,
    is_harbor_task,
    parse_harbor_notes,
    rules_report,
    write_task_validation,
)

GO_MOD = """module fixturehost

go 1.23
"""

ADD_GO = """package mathx

func Add(a, b int) int {
	return a + b
}
"""

ADD_TEST = """package mathx

import "testing"

func TestAdd(t *testing.T) {
	if Add(2, 2) != 4 {
		t.Fatalf("Add(2, 2) = %d", Add(2, 2))
	}
}
"""

BUG_PATCH = """diff --git a/mathx/add.go b/mathx/add.go
--- a/mathx/add.go
+++ b/mathx/add.go
@@ -1,5 +1,5 @@
 package mathx
 
 func Add(a, b int) int {
-	return a + b
+	return a - b
 }
"""

INSTRUCTION = f"""# Incorrect behavior

Adding two small integers yields a value below the sum: expected 4, got 0.

Reproduce with:

```
go test -count=1 -timeout 15m ./mathx/...
```

Work in `/app`.

{NO_WEB_CLAUSE}
"""

TASK_TOML = """schema_version = "1.3"

[metadata]
category = "software-engineering"

[verifier]
network_mode = "no-network"
timeout_sec = 1800.0

[agent]
network_mode = "allowlist"
allowed_hosts = ["cursor.com"]
timeout_sec = 14400.0

[environment]
build_timeout_sec = 1800.0
network_mode = "public"
"""

TEST_SH = """#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
echo "abc  /app/mathx/add_test.go" | sha256sum -c --status || exit 1
if go test -count=1 -timeout 15m -run '^(TestAdd)$' ./mathx/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
"""


def _write_task(root: Path, *, no_web: bool = True, checksum: bool = True) -> Path:
    task = root / "fixture-add"
    src = task / "environment" / "src" / "mathx"
    src.mkdir(parents=True)
    (src / "add.go").write_text(ADD_GO)
    (src / "add_test.go").write_text(ADD_TEST)
    (task / "environment" / "src" / "go.mod").write_text(GO_MOD)
    instr = INSTRUCTION if no_web else INSTRUCTION.replace(NO_WEB_CLAUSE, "")
    (task / "instruction.md").write_text(instr)
    (task / "task.toml").write_text(TASK_TOML)
    sh = TEST_SH if checksum else TEST_SH.replace("sha256sum -c --status || exit 1\n", "")
    tests = task / "tests"
    tests.mkdir(parents=True)
    (tests / "test.sh").write_text(sh)
    (task / "environment" / "Dockerfile").write_text("FROM golang:1.23\n")
    return task


def _by_id(verdicts: list[RuleVerdict]) -> dict[str, RuleVerdict]:
    return {v.rule_id: v for v in verdicts}


def test_registry_covers_document_ids() -> None:
    assert list(RULE_IDS) == [f"A{i}" for i in range(1, 13)] + [f"B{i}" for i in range(1, 9)] + ["C5"]
    assert set(RULES) == set(RULE_IDS)


def test_evaluate_rules_on_fixture(tmp_path: Path) -> None:
    task = _write_task(tmp_path)
    (task / "patches").mkdir()
    (task / "patches" / "gold.patch").write_text(BUG_PATCH)
    (task / "patches" / "alt.patch").write_text("diff --git a/x b/x\n")
    (task / "patches" / "cheat.patch").write_text("diff --git a/y b/y\n")
    notes = tmp_path / "RESULT.md"
    notes.write_text(
        """# notes

## Bug — `fixture-add`

### Validation

| check | result |
|---|---|
| gold revert | restore add.go → f2p pass |
| alt accepted | structurally different Add → pass |
| cheat rejected | hardcoded 4 → reject |
| f2p files in impact | `add_test.go` ∈ Add blast radius |
| no collateral | rest of suite green |
| 3× not flaky | fail ×3 |
| docker | harbor-fixture: buggy fail, gold pass |
"""
    )
    verdicts = evaluate_rules(task)
    got = _by_id(verdicts)
    assert set(got) == set(RULE_IDS)
    assert got["A1"].skipped is False and got["A1"].passed is True
    assert got["A2"].passed is True
    assert got["A3"].passed is True
    assert got["A4"].passed is True
    assert got["A5"].passed is True
    assert got["A8"].passed is True
    assert got["A10"].passed is True
    assert got["B1"].passed is True
    assert got["B2"].passed is True
    assert got["B6"].passed is True
    assert got["B7"].passed is True
    assert got["B8"].passed is True
    assert got["A6"].skipped is True
    assert got["A7"].skipped is True


def test_kills_missing_checksum_and_noweb(tmp_path: Path) -> None:
    task = _write_task(tmp_path, no_web=False, checksum=False)
    got = _by_id(evaluate_rules(task))
    assert got["B1"].skipped is False and got["B1"].passed is False
    assert got["B8"].passed is False
    assert "sha256sum" in got["B1"].evidence or "checksum" in got["B1"].evidence.lower()


def test_write_validation_and_report(tmp_path: Path) -> None:
    good = _write_task(tmp_path / "tasks")
    bad = _write_task(tmp_path / "tasks_bad", no_web=False, checksum=False)
    # second task needs a distinct dir name for the glob
    bad2 = tmp_path / "tasks_bad" / "fixture-add"
    assert bad == bad2
    payload = write_task_validation(good)
    assert "rule_verdicts" in payload
    ids = [v["rule_id"] for v in payload["rule_verdicts"]]
    assert ids == list(RULE_IDS)
    write_task_validation(bad)
    table, rows = rules_report(str(tmp_path / "tasks*" / "*"), backfill=False)
    assert "n_tasks = 2" in table
    b1 = next(r for r in rows if r.rule_id == "B1")
    assert b1.n_failed == 1
    assert any("fixture-add" in t for t in b1.killed_tasks)
    b8 = next(r for r in rows if r.rule_id == "B8")
    assert b8.n_failed == 1


def test_backfill_skips_existing_and_fills_missing(tmp_path: Path) -> None:
    task = _write_task(tmp_path)
    first = write_task_validation(task)
    first["rule_verdicts"][0]["evidence"] = "sentinel-keep"
    (task / "validation.json").write_text(json.dumps(first, indent=2) + "\n")
    assert backfill_task(task) is None
    kept = json.loads((task / "validation.json").read_text())
    assert kept["rule_verdicts"][0]["evidence"] == "sentinel-keep"

    other = _write_task(tmp_path / "more")
    assert (other / "validation.json").is_file() is False
    written = backfill_task(other)
    assert written is not None
    assert (other / "validation.json").is_file()
    data = json.loads((other / "validation.json").read_text())
    assert len(data["rule_verdicts"]) == 21


def test_parse_harbor_notes_result_tables(tmp_path: Path) -> None:
    (tmp_path / "RESULT.md").write_text(
        """# harbor

## Verifier table

| bug | alternative | f2p + alt | cheat | f2p + cheat |
|---|---|---|---|---|
| NewBackofferWithVars | wrap | **accept** | special-case | **reject** |

## Docker fail-to-pass (before launch)

| task | buggy | fixed |
|---|---|---|
| client-go-newbackofferwithvars | fail 2 f2p | pass |
"""
    )
    index = parse_harbor_notes(tmp_path)
    row = index["newbackofferwithvars"]
    assert "accept" in row["alt accepted"].lower()
    assert "reject" in row["cheat rejected"].lower()
    assert "fail" in row["docker buggy"].lower()
    assert "pass" in row["docker gold"].lower()


def test_build_task_writes_rule_verdicts(tmp_path: Path) -> None:
    import subprocess

    repo = tmp_path / "repo"
    (repo / "mathx").mkdir(parents=True)
    (repo / "go.mod").write_text(GO_MOD)
    (repo / "mathx" / "add.go").write_text(ADD_GO)
    (repo / "mathx" / "add_test.go").write_text(ADD_TEST)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "test@example.com"], cwd=repo, check=True)
    subprocess.run(["git", "config", "user.name", "test"], cwd=repo, check=True)
    subprocess.run(["git", "add", "."], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "-m", "init"], cwd=repo, check=True, capture_output=True)
    sha = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=repo, text=True).strip()
    patch = tmp_path / "Add.patch"
    patch.write_text(BUG_PATCH)
    out = tmp_path / "task"
    built = build_task(
        repo,
        sha,
        patch,
        ["TestAdd"],
        out,
        test_output="--- FAIL: TestAdd (0.00s)\n    Error: expected 4, got 0\n",
        checksum_test_files=True,
        extra_redact=["Add"],
    )
    assert is_harbor_task(built)
    data = json.loads((built / "validation.json").read_text())
    ids = [v["rule_id"] for v in data["rule_verdicts"]]
    assert ids == list(RULE_IDS)
    by = {v["rule_id"]: v for v in data["rule_verdicts"]}
    assert by["B1"]["passed"] is True
    assert by["B1"]["skipped"] is False


def test_a11_a12_c5(tmp_path: Path) -> None:
    task = _write_task(tmp_path)
    (task / "patches").mkdir()
    (task / "patches" / "gold.patch").write_text(BUG_PATCH)
    (task / "patches" / "cheat.patch").write_text(
        "diff --git a/mathx/add_test.go b/mathx/add_test.go\n"
        "--- a/mathx/add_test.go\n"
        "+++ b/mathx/add_test.go\n"
        "@@ -1 +1 @@\n"
        "-old\n"
        "+new\n"
    )
    (task / "tests" / "test.sh").write_text(
        TEST_SH + "\n# perf gate\ngo test -bench=BenchmarkAdd -benchtime=1s\n# ns/op ceiling\n"
    )
    got = _by_id(evaluate_rules(task, extra={"gold_ns_op": 10.0, "perf_load_avg": 8.0}))
    assert got["A11"].skipped is False
    assert got["A11"].passed is False
    assert got["A12"].passed is False
    assert "add_test.go" in got["A12"].evidence

    (task / "patches" / "cheat.patch").write_text("diff --git a/mathx/add.go b/mathx/add.go\n")
    (task / "tests" / "measure_gold.sh").write_text("#!/bin/bash\necho 1\n")
    got2 = _by_id(
        evaluate_rules(
            task,
            extra={
                "gold_ns_op": 12.0,
                "perf_load_avg": 1.1,
                "harness_ok": True,
            },
        )
    )
    assert got2["A11"].passed is True
    assert got2["A12"].passed is True
    assert got2["C5"].passed is True

from __future__ import annotations

import subprocess
from pathlib import Path

from openswe_traces.synth.harbor_tasks import (
    AGENT_TIMEOUT_HARD_SEC,
    build_task,
    discover_packages_for_tests,
    discover_test_files,
    file_sha256,
    instruction_self_check,
    issue_from_failures,
    paraphrase_failure,
    render_checksum_guard,
    render_test_sh,
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


def _git(cwd: Path, *args: str) -> None:
    subprocess.run(["git", *args], cwd=cwd, check=True, capture_output=True, text=True)


def _fixture_repo(tmp_path: Path) -> Path:
    repo = tmp_path / "repo"
    (repo / "mathx").mkdir(parents=True)
    (repo / "go.mod").write_text(GO_MOD)
    (repo / "mathx" / "add.go").write_text(ADD_GO)
    (repo / "mathx" / "add_test.go").write_text(ADD_TEST)
    _git(repo, "init")
    _git(repo, "config", "user.email", "test@example.com")
    _git(repo, "config", "user.name", "test")
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "init")
    sha = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=repo, text=True).strip()
    return repo, sha


def test_issue_omits_patch_and_symbol() -> None:
    text = issue_from_failures(
        ["TestAdd"],
        "--- FAIL: TestAdd (0.00s)\n    add_test.go:6: Add(2, 2) = 0\n",
        redact_terms=["Add", "add.go"],
        user_context="This helper is used to combine two integers for callers.",
    )
    assert "TestAdd" in text
    assert "Add(2, 2) = 0" not in text  # symbol redacted from body
    assert "diff --git" not in text
    assert "return a - b" not in text
    assert "combine two integers" in text
    para = paraphrase_failure(
        "--- FAIL: TestAdd (0.00s)\n    add_test.go:6: expected 4 got 0\n",
    )
    assert "expected 4 got 0" in para
    assert "add_test.go" not in para


def test_instruction_self_check_flags_leaks() -> None:
    good = issue_from_failures(
        ["TestAdd"],
        "--- FAIL: TestAdd (0.00s)\n    Error: expected 4, got 0\n",
        user_context="Integer helpers are used throughout the module.",
    )
    test_sh = render_test_sh(["TestAdd"], ["mathx"])
    check = instruction_self_check(
        good,
        test_sh=test_sh,
        f2p_tests=["TestAdd"],
        changed_symbols=["Add"],
        changed_files=["mathx/add.go"],
        diff_hunk="-	return a + b\n+	return a - b\n",
    )
    assert check["names_present"]
    assert check["has_symptom_paraphrase"]
    assert check["can_reproduce_one_command"]
    assert "go test" in str(check["reproduce_command"])
    leaky = good + "\nPlease edit Add in add.go:4.\n"
    bad = instruction_self_check(
        leaky,
        test_sh=test_sh,
        f2p_tests=["TestAdd"],
        changed_symbols=["Add"],
        changed_files=["mathx/add.go"],
        diff_hunk="-	return a + b\n+	return a - b\n",
    )
    assert bad["leaked_symbols"] == ["Add"]
    assert bad["has_line_numbers"]
    assert bad["ok"] is False


def test_render_test_sh_runs_named_tests() -> None:
    script = render_test_sh(["TestAdd", "TestClamp"], ["mathx"])
    assert "-run '^(TestAdd|TestClamp)$'" in script
    assert "./mathx" in script
    assert "/logs/verifier/reward.txt" in script
    assert "exit 1" in script
    guarded = render_test_sh(
        ["TestAdd"],
        ["mathx"],
        checksums=[("mathx/add_test.go", "abc123")],
    )
    assert "sha256sum -c" in guarded
    assert "mathx/add_test.go" in guarded
    assert "abc123" in guarded


def test_discover_packages_for_tests(tmp_path: Path) -> None:
    repo, _sha = _fixture_repo(tmp_path)
    pkgs = discover_packages_for_tests(repo, ["TestAdd"])
    assert pkgs == ["mathx"]
    pkgs_slash = discover_packages_for_tests(repo, ["TestAdd/Subcase"])
    assert pkgs_slash == ["mathx"]


def test_discover_skips_nested_gomod(tmp_path: Path) -> None:
    repo, _sha = _fixture_repo(tmp_path)
    nested = repo / "integration_tests" / "raw"
    nested.mkdir(parents=True)
    (repo / "integration_tests" / "go.mod").write_text("module integration_tests\n\ngo 1.23\n")
    (nested / "api_test.go").write_text("package raw\n\nfunc TestAdd(t *testing.T) {}\n")
    pkgs = discover_packages_for_tests(repo, ["TestAdd"])
    assert pkgs == ["mathx"]


def test_build_task_writes_harbor_layout(tmp_path: Path) -> None:
    repo, sha = _fixture_repo(tmp_path)
    patch = tmp_path / "Add.patch"
    patch.write_text(BUG_PATCH)
    out = tmp_path / "task"
    output = (
        "--- FAIL: TestAdd (0.00s)\n"
        "    add_test.go:6: Add(2, 2) = 0\n"
        "FAIL\tfixturehost/mathx\t0.001s\n"
    )
    built = build_task(
        repo,
        sha,
        patch,
        ["TestAdd"],
        out,
        test_output=output,
    )
    assert built == out
    assert (out / "instruction.md").is_file()
    assert (out / "task.toml").is_file()
    assert (out / "environment" / "Dockerfile").is_file()
    assert (out / "tests" / "test.sh").is_file()
    src_add = (out / "environment" / "src" / "mathx" / "add.go").read_text()
    assert "return a - b" in src_add
    assert "return a + b" not in src_add
    instruction = (out / "instruction.md").read_text()
    assert "TestAdd" in instruction
    assert "diff --git" not in instruction
    assert "Add.patch" not in instruction
    dockerfile = (out / "environment" / "Dockerfile").read_text()
    assert "FROM golang:1.23" in dockerfile
    assert "go mod download" in dockerfile
    toml = (out / "task.toml").read_text()
    assert "timeout_sec = 10800" in toml or "timeout_sec = 10800.0" in toml
    test_sh = (out / "tests" / "test.sh").read_text()
    assert "-run '^(TestAdd)$'" in test_sh
    assert "./mathx" in test_sh
    # Harbor verifier must write a reward and fail the script on test failure.
    assert "/logs/verifier/reward.txt" in test_sh


def test_checksum_guard_and_hard_timeout(tmp_path: Path) -> None:
    repo, sha = _fixture_repo(tmp_path)
    rel = "mathx/add_test.go"
    digest = file_sha256(repo / rel)
    assert discover_test_files(repo, ["TestAdd"]) == [rel]
    guard = render_checksum_guard([(rel, digest)])
    assert digest in guard
    assert "/app/mathx/add_test.go" in guard
    patch = tmp_path / "Add.patch"
    patch.write_text(BUG_PATCH)
    out = tmp_path / "task_guard"
    built = build_task(
        repo,
        sha,
        patch,
        ["TestAdd"],
        out,
        test_output="--- FAIL: TestAdd (0.00s)\n    Error: expected 4, got 0\n",
        agent_timeout_sec=AGENT_TIMEOUT_HARD_SEC,
        checksum_test_files=True,
    )
    test_sh = (built / "tests" / "test.sh").read_text()
    assert "sha256sum -c" in test_sh
    assert digest in test_sh
    toml = (built / "task.toml").read_text()
    assert "14400" in toml

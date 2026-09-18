"""Build Harbor task directories from codegraph-injected Go bugs."""

from __future__ import annotations

import argparse
import hashlib
import io
import os
import re
import shutil
import subprocess
import tarfile
import tempfile
from collections.abc import Sequence
from pathlib import Path

from openswe_traces.synth.codegraph_bugs import parse_go_test_output

AGENT_TIMEOUT_SEC = 10800.0  # 3 h — agent must not be cut off mid-run
AGENT_TIMEOUT_HARD_SEC = 14400.0  # 4 h — multi-site / cross-module tasks
VERIFIER_TIMEOUT_SEC = 1800.0
BUILD_TIMEOUT_SEC = 1800.0

_TEST_FUNC_RE = re.compile(r"^func\s+(Test[A-Za-z0-9_]+)\s*\(", re.MULTILINE)
_FAIL_BLOCK_RE = re.compile(
    r"(?:--- FAIL: |FAIL\t|panic: |Error Trace: |Error:\s|expected |got ).*",
    re.IGNORECASE,
)


def _run(
    args: list[str],
    *,
    cwd: Path | None = None,
    timeout: int = 120,
    input_bytes: bytes | None = None,
) -> subprocess.CompletedProcess[bytes]:
    env = os.environ.copy()
    env["PATH"] = "/usr/local/go/bin:" + env.get("PATH", "")
    return subprocess.run(
        args,
        cwd=cwd,
        capture_output=True,
        timeout=timeout,
        check=False,
        env=env,
        input=input_bytes,
    )


def snapshot_bugged_tree(
    repo_dir: Path,
    base_commit: str,
    bug_patch: Path,
    dest: Path,
) -> None:
    """Materialize ``base_commit`` + ``bug_patch`` into ``dest`` (no ``.git``)."""
    if dest.exists():
        shutil.rmtree(dest)
    dest.mkdir(parents=True)
    archive = _run(
        ["git", "-C", str(repo_dir), "archive", "--format=tar", base_commit],
        timeout=180,
    )
    if archive.returncode != 0:
        err = (archive.stderr or b"").decode("utf-8", "replace")
        raise RuntimeError(f"git archive {base_commit} failed: {err}")
    with tarfile.open(fileobj=io.BytesIO(archive.stdout), mode="r|") as tf:
        tf.extractall(dest, filter="data")
    # `git apply` from a dest inside another git worktree walks up to that repo
    # and silently misses the snapshot. `patch -p1 -d dest` stays local.
    applied = _run(
        ["patch", "-p1", "--forward", "--batch", "-i", str(bug_patch.resolve()), "-d", str(dest)]
    )
    if applied.returncode != 0:
        err = (applied.stderr or applied.stdout or b"").decode("utf-8", "replace")
        raise RuntimeError(f"apply {bug_patch} failed: {err}")
    _ensure_go_mod(dest, repo_dir.name)


def _ensure_go_mod(tree: Path, module_name: str) -> None:
    if (tree / "go.mod").exists():
        return
    name = re.sub(r"[^A-Za-z0-9._\-/]", "-", module_name) or "host"
    proc = _run(["go", "mod", "init", name], cwd=tree)
    if proc.returncode != 0:
        err = (proc.stderr or b"").decode("utf-8", "replace")
        raise RuntimeError(f"go mod init failed: {err}")


def _in_nested_module(repo_dir: Path, path: Path) -> bool:
    """True if ``path`` sits under a nested ``go.mod`` (e.g. integration_tests/)."""
    for parent in path.parents:
        if parent == repo_dir:
            break
        if (parent / "go.mod").exists():
            return True
    return False


def discover_packages_for_tests(repo_dir: Path, test_names: Sequence[str]) -> list[str]:
    """Return Go package dirs (posix, relative) that define the named tests."""
    wanted = {n.split("/")[0] for n in test_names if n}
    found: dict[str, None] = {}
    for test_file in repo_dir.rglob("*_test.go"):
        if _in_nested_module(repo_dir, test_file):
            continue
        try:
            text = test_file.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        names = set(_TEST_FUNC_RE.findall(text))
        if names & wanted:
            rel = test_file.parent.relative_to(repo_dir).as_posix()
            found["." if rel == "." else rel] = None
    return list(found)


def _redact(text: str, terms: Sequence[str]) -> str:
    out = text
    for term in sorted({t for t in terms if t}, key=len, reverse=True):
        out = re.sub(
            r"(?<![A-Za-z0-9_])" + re.escape(term) + r"(?![A-Za-z0-9_])",
            "<redacted>",
            out,
            flags=re.IGNORECASE,
        )
    return out


_LINE_NO_RE = re.compile(r":\d+\b")
_GO_FILE_RE = re.compile(r"(?:[A-Za-z0-9_./-]+\.go)\b")
_COMMON_IDENT = re.compile(
    r"^(?:func|return|true|false|nil|error|string|int|byte|bytes|len|if|else|for|"
    r"case|switch|type|var|const|make|append|copy|range|package|import|"
    r"test|fail|expected|got|want|panic|ok|err|key|value|ctx|store|label|"
    r"version|prefix|id|buf|data|out|src|dst|new|old|user|name)$",
    re.IGNORECASE,
)


def paraphrase_failure(test_output: str, *, redact_terms: Sequence[str] = ()) -> str:
    """Expected-vs-got / panic paraphrase without file names or line numbers."""
    lines = [ln for ln in test_output.splitlines() if _FAIL_BLOCK_RE.search(ln.strip())]
    cleaned: list[str] = []
    for ln in lines[:40]:
        ln = _GO_FILE_RE.sub("<file>", ln)
        ln = _LINE_NO_RE.sub("", ln)
        ln = _redact(ln, redact_terms)
        cleaned.append(ln.strip())
    return "\n".join(cleaned) if cleaned else _redact(test_output[-1500:], redact_terms)


def issue_from_failures(
    f2p_tests: Sequence[str],
    test_output: str,
    *,
    redact_terms: Sequence[str] = (),
    user_context: str = "",
    locality: int = 0,
    packages: Sequence[str] = (),
    reproduce_command: str = "",
) -> str:
    """Write a bug report from failing tests. Never include a patch or fix-site name.

    locality: L0 names the f2p tests; L1 names only the package + command;
    L2 is a behavior-level report plus a package test command.
    """
    names = [t for t in f2p_tests if t]
    excerpt = paraphrase_failure(test_output, redact_terms=redact_terms)
    if locality >= 1:
        excerpt = re.sub(r"\bTest[A-Za-z0-9_]+\b", "<test>", excerpt)
    listed = ", ".join(f"`{n}`" for n in names) or "the failing unit tests"
    context_block = ""
    if user_context.strip():
        context_block = f"\n{user_context.strip()}\n"
    pkg_args = " ".join(_go_pkg_arg(p) for p in (packages or ["./..."]))
    cmd = reproduce_command.strip() or f"go test -count=1 -timeout 15m {pkg_args}"
    if locality <= 0:
        header = f"""# Failing unit tests

The following tests currently fail on this Go codebase: {listed}.
{context_block}"""
    elif locality == 1:
        pkg_shown = ", ".join(f"`{p}`" for p in (packages or ["./..."]))
        header = f"""# Failing package tests

Tests in package {pkg_shown} currently fail.

Reproduce with:

```
{cmd}
```
{context_block}"""
    else:
        header = f"""# Incorrect behavior
{context_block}
Reproduce with:

```
{cmd}
```
"""
    body = f"""{header}
Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
{excerpt}
```

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""
    return body.rstrip() + "\n"


def instruction_self_check(
    instruction: str,
    *,
    test_sh: str,
    f2p_tests: Sequence[str],
    changed_symbols: Sequence[str] = (),
    changed_files: Sequence[str] = (),
    diff_hunk: str = "",
    reproduce_command: str = "",
    locality: int = 0,
    packages: Sequence[str] = (),
) -> dict[str, object]:
    """Calibrate instruction: symptom present, no fix-site leak, one-command repro."""
    instr_l = instruction.lower()
    names_listed = all(n in instruction for n in f2p_tests if n)
    names_absent = not any(n in instruction for n in f2p_tests if n)
    names_ok = names_listed if locality <= 0 else names_absent
    has_symptom = bool(
        re.search(
            r"(expected|got|want|panic:|not equal|!=|should be|actual\s*:|difference was)",
            instruction,
            re.IGNORECASE,
        )
    )
    leaked_symbols = [s for s in changed_symbols if s and re.search(rf"\b{re.escape(s)}\b", instruction)]
    leaked_files = []
    for fp in changed_files:
        base = Path(fp).name
        if base and base.lower() in instr_l:
            leaked_files.append(base)
    has_line_nos = bool(re.search(r"\.go:\d+", instruction))
    has_diff = "diff --git" in instruction or instruction.strip().startswith("@@")
    cmd = reproduce_command.strip() or next(
        (ln.strip() for ln in test_sh.splitlines() if "go test" in ln),
        "",
    )
    unique_diff_tokens: list[str] = []
    if diff_hunk:
        hunk_idents = set(re.findall(r"[A-Za-z_][A-Za-z0-9_]{3,}", diff_hunk))
        instr_idents = set(re.findall(r"[A-Za-z_][A-Za-z0-9_]{3,}", instruction))
        f2p_blob = " ".join(f2p_tests)
        for tok in sorted(hunk_idents & instr_idents):
            if _COMMON_IDENT.match(tok):
                continue
            if tok in f2p_tests or tok in f2p_blob:
                continue
            unique_diff_tokens.append(tok)
    pkg_ok = True
    if locality >= 1 and packages:
        pkg_ok = any(p.rstrip("/") in instruction for p in packages)
    cmd_ok = bool(cmd)
    if locality >= 1:
        cmd_ok = "go test" in instruction and "-run" not in instruction
    return {
        "names_present": names_listed,
        "has_symptom_paraphrase": has_symptom,
        "leaked_symbols": leaked_symbols,
        "leaked_files": leaked_files,
        "has_line_numbers": has_line_nos,
        "has_diff": has_diff,
        "reproduce_command": cmd,
        "can_reproduce_one_command": cmd_ok,
        "diff_hunk_tokens_in_instruction": unique_diff_tokens,
        "locality": locality,
        "ok": bool(
            names_ok
            and has_symptom
            and not leaked_symbols
            and not leaked_files
            and not has_line_nos
            and not has_diff
            and cmd_ok
            and pkg_ok
            and not unique_diff_tokens
        ),
    }


def discover_test_files(repo_dir: Path, test_names: Sequence[str]) -> list[str]:
    """Return posix-relative ``*_test.go`` files that define the named tests."""
    wanted = {n.split("/")[0] for n in test_names if n}
    found: dict[str, None] = {}
    for test_file in repo_dir.rglob("*_test.go"):
        if _in_nested_module(repo_dir, test_file):
            continue
        try:
            text = test_file.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        names = set(_TEST_FUNC_RE.findall(text))
        if names & wanted:
            found[test_file.relative_to(repo_dir).as_posix()] = None
    return list(found)


def file_sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def render_checksum_guard(checksums: Sequence[tuple[str, str]]) -> str:
    """Fail the verifier if any checksummed file under ``/app`` was edited."""
    if not checksums:
        return ""
    lines = [
        "checksum_fail() {",
        '  echo 0 > /logs/verifier/reward.txt',
        '  echo "test file modified: $1" >&2',
        "  exit 1",
        "}",
    ]
    for rel, digest in checksums:
        rel = rel.lstrip("./")
        lines.append(
            f'echo "{digest}  /app/{rel}" | sha256sum -c --status || checksum_fail "{rel}"'
        )
    return "\n".join(lines) + "\n"


def render_test_sh(
    f2p_tests: Sequence[str],
    packages: Sequence[str],
    *,
    checksums: Sequence[tuple[str, str]] = (),
) -> str:
    names = [t for t in f2p_tests if t]
    if not names:
        raise ValueError("f2p_tests must be non-empty")
    pattern = "^(" + "|".join(re.escape(n) for n in names) + ")$"
    pkg_args = " ".join(_go_pkg_arg(p) for p in (packages or ["./..."]))
    guard = render_checksum_guard(checksums)
    return f"""#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
{guard}if go test -count=1 -timeout 15m -run '{pattern}' {pkg_args}; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
"""


def _go_pkg_arg(pkg: str) -> str:
    pkg = pkg.strip().rstrip("/")
    if pkg in {"./...", ".", ""}:
        return "./..."
    if not pkg.startswith("./"):
        pkg = "./" + pkg
    if not pkg.endswith("/..."):
        pkg += "/..."
    return pkg


def render_dockerfile() -> str:
    return """FROM golang:1.23

RUN apt-get update && apt-get install -y --no-install-recommends \\
        git tmux ca-certificates patch \\
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY src/ /app/
RUN if [ ! -f go.mod ]; then go mod init host; fi
RUN GOPROXY=https://proxy.golang.org,direct go mod download || \
    GOPROXY=https://proxy.golang.org,direct go mod tidy || true
"""


def render_task_toml(*, agent_timeout_sec: float = AGENT_TIMEOUT_SEC) -> str:
    return f"""schema_version = "1.3"

[metadata]
category = "software-engineering"
tags = ["go", "bugfix"]

[verifier]
timeout_sec = {VERIFIER_TIMEOUT_SEC}

[agent]
timeout_sec = {agent_timeout_sec}

[environment]
build_timeout_sec = {BUILD_TIMEOUT_SEC}
network_mode = "public"
"""


def capture_f2p_output(
    tree: Path,
    f2p_tests: Sequence[str],
    packages: Sequence[str],
    *,
    timeout: int = 600,
) -> str:
    pattern = "^(" + "|".join(re.escape(n) for n in f2p_tests) + ")$"
    args = ["go", "test", "-count=1", "-timeout", "14m", "-run", pattern]
    args.extend(_go_pkg_arg(p) for p in (packages or ["./..."]))
    proc = _run(args, cwd=tree, timeout=timeout)
    return ((proc.stdout or b"") + (proc.stderr or b"")).decode("utf-8", "replace")


def discover_f2p(
    repo_dir: Path,
    base_commit: str,
    bug_patch: Path,
    *,
    timeout: int = 900,
    packages: Sequence[str] | None = None,
) -> tuple[list[str], str]:
    """Apply the patch in a scratch tree and return (failing test names, output)."""
    with tempfile.TemporaryDirectory(prefix="harbor_nex_f2p_") as tmp:
        tree = Path(tmp) / "src"
        snapshot_bugged_tree(repo_dir, base_commit, bug_patch, tree)
        args = ["go", "test", "-count=1", "-timeout", f"{max(timeout - 30, 60)}s"]
        args.extend(packages or ["./..."])
        proc = _run(args, cwd=tree, timeout=timeout)
        output = ((proc.stdout or b"") + (proc.stderr or b"")).decode("utf-8", "replace")
        tests, _pkgs = parse_go_test_output(output)
        return tests, output


def build_task(
    repo_dir: Path | str,
    base_commit: str,
    bug_patch: Path | str,
    f2p_tests: Sequence[str],
    out_dir: Path | str,
    *,
    test_output: str | None = None,
    agent_timeout_sec: float = AGENT_TIMEOUT_SEC,
    user_context: str = "",
    extra_redact: Sequence[str] = (),
    checksum_test_files: bool = False,
    locality: int = 0,
    guard_tests: Sequence[str] = (),
    reproduce_command: str = "",
) -> Path:
    """Write a Harbor task directory.

    Layout: ``instruction.md``, ``task.toml``, ``environment/Dockerfile`` plus
    ``environment/src`` (repo at ``base_commit`` with ``bug_patch`` applied),
    and ``tests/test.sh`` that runs the fail-to-pass tests (plus optional guards).
    """
    repo_dir = Path(repo_dir)
    bug_patch = Path(bug_patch)
    out_dir = Path(out_dir)
    names = [t for t in f2p_tests if t]
    if not names:
        raise ValueError("f2p_tests must contain at least one test name")
    guards = [t for t in guard_tests if t and t not in names]
    sh_names = names + guards

    src = out_dir / "environment" / "src"
    snapshot_bugged_tree(repo_dir, base_commit, bug_patch, src)
    packages = discover_packages_for_tests(src, sh_names) or ["./..."]
    output = test_output if test_output is not None else capture_f2p_output(src, names, packages)
    redact = [bug_patch.stem, bug_patch.name, *extra_redact]
    pkg_args = " ".join(_go_pkg_arg(p) for p in packages)
    cmd = reproduce_command.strip() or (
        f"go test -count=1 -timeout 15m {pkg_args}" if locality >= 1 else ""
    )
    instruction = issue_from_failures(
        names,
        output,
        redact_terms=redact,
        user_context=user_context,
        locality=locality,
        packages=packages,
        reproduce_command=cmd,
    )
    checksums: list[tuple[str, str]] = []
    if checksum_test_files:
        for rel in discover_test_files(src, sh_names):
            checksums.append((rel, file_sha256(src / rel)))

    out_dir.mkdir(parents=True, exist_ok=True)
    (out_dir / "instruction.md").write_text(instruction)
    (out_dir / "task.toml").write_text(render_task_toml(agent_timeout_sec=agent_timeout_sec))
    (out_dir / "environment" / "Dockerfile").write_text(render_dockerfile())
    tests_dir = out_dir / "tests"
    tests_dir.mkdir(parents=True, exist_ok=True)
    test_sh = tests_dir / "test.sh"
    test_sh.write_text(render_test_sh(sh_names, packages, checksums=checksums))
    test_sh.chmod(0o755)
    return out_dir


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Build Harbor tasks from Go bug patches")
    sub = parser.add_subparsers(dest="cmd", required=True)

    p_build = sub.add_parser("build", help="Write a Harbor task directory")
    p_build.add_argument("repo_dir")
    p_build.add_argument("base_commit")
    p_build.add_argument("bug_patch")
    p_build.add_argument("out_dir")
    p_build.add_argument("--f2p", nargs="+", required=True, help="fail-to-pass test names")
    p_build.add_argument("--test-output-file", default=None)
    p_build.add_argument("--locality", type=int, default=0, help="0=name tests, 1=package+cmd, 2=behavior")
    p_build.add_argument("--guard", nargs="*", default=(), help="existing tests included in test.sh only")

    p_disc = sub.add_parser("discover-f2p", help="Apply patch in scratch and list failing tests")
    p_disc.add_argument("repo_dir")
    p_disc.add_argument("base_commit")
    p_disc.add_argument("bug_patch")
    p_disc.add_argument("--timeout", type=int, default=900)
    p_disc.add_argument("--packages", nargs="*", default=None)

    args = parser.parse_args(argv)
    if args.cmd == "build":
        output = None
        if args.test_output_file:
            output = Path(args.test_output_file).read_text(encoding="utf-8", errors="replace")
        build_task(
            Path(args.repo_dir),
            args.base_commit,
            Path(args.bug_patch),
            args.f2p,
            Path(args.out_dir),
            test_output=output,
            locality=args.locality,
            guard_tests=list(args.guard),
        )
        return 0
    if args.cmd == "discover-f2p":
        tests, output = discover_f2p(
            Path(args.repo_dir),
            args.base_commit,
            Path(args.bug_patch),
            timeout=args.timeout,
            packages=args.packages,
        )
        print("\n".join(tests) if tests else "(no failing tests)")
        print("--- output ---", flush=True)
        print(output[-8000:])
        return 0 if tests else 2
    return 1


if __name__ == "__main__":
    raise SystemExit(main())

"""Build Harbor task directories from codegraph-injected Go bugs."""

from __future__ import annotations

import argparse
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
    wanted = set(test_names)
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


def issue_from_failures(
    f2p_tests: Sequence[str],
    test_output: str,
    *,
    redact_terms: Sequence[str] = (),
) -> str:
    """Write a bug report from failing tests. Never include a patch or fix-site name."""
    names = [t for t in f2p_tests if t]
    fail_lines = [ln for ln in test_output.splitlines() if _FAIL_BLOCK_RE.match(ln.strip())]
    excerpt = "\n".join(fail_lines[:80]) if fail_lines else test_output[-4000:]
    excerpt = _redact(excerpt, redact_terms)
    listed = ", ".join(f"`{n}`" for n in names) or "the failing unit tests"
    body = f"""# Failing unit tests

The following tests currently fail on this Go codebase: {listed}.

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

## Observed failures

```
{excerpt}
```

Work in `/app`. Keep unrelated tests passing.
"""
    return body.rstrip() + "\n"


def render_test_sh(f2p_tests: Sequence[str], packages: Sequence[str]) -> str:
    names = [t for t in f2p_tests if t]
    if not names:
        raise ValueError("f2p_tests must be non-empty")
    pattern = "^(" + "|".join(re.escape(n) for n in names) + ")$"
    pkg_args = " ".join(_go_pkg_arg(p) for p in (packages or ["./..."]))
    return f"""#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
if go test -count=1 -timeout 15m -run '{pattern}' {pkg_args}; then
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
) -> Path:
    """Write a Harbor task directory.

    Layout: ``instruction.md``, ``task.toml``, ``environment/Dockerfile`` plus
    ``environment/src`` (repo at ``base_commit`` with ``bug_patch`` applied),
    and ``tests/test.sh`` that runs exactly the fail-to-pass tests.
    """
    repo_dir = Path(repo_dir)
    bug_patch = Path(bug_patch)
    out_dir = Path(out_dir)
    names = [t for t in f2p_tests if t]
    if not names:
        raise ValueError("f2p_tests must contain at least one test name")

    src = out_dir / "environment" / "src"
    snapshot_bugged_tree(repo_dir, base_commit, bug_patch, src)
    packages = discover_packages_for_tests(src, names) or ["./..."]
    output = test_output if test_output is not None else capture_f2p_output(src, names, packages)
    redact = [bug_patch.stem, bug_patch.name]
    instruction = issue_from_failures(names, output, redact_terms=redact)

    out_dir.mkdir(parents=True, exist_ok=True)
    (out_dir / "instruction.md").write_text(instruction)
    (out_dir / "task.toml").write_text(render_task_toml(agent_timeout_sec=agent_timeout_sec))
    (out_dir / "environment" / "Dockerfile").write_text(render_dockerfile())
    tests_dir = out_dir / "tests"
    tests_dir.mkdir(parents=True, exist_ok=True)
    test_sh = tests_dir / "test.sh"
    test_sh.write_text(render_test_sh(names, packages))
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

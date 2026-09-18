"""Two-module (library → consumer) bug injection and Harbor packaging."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sqlite3
import subprocess
from collections.abc import Sequence
from dataclasses import asdict, dataclass
from pathlib import Path

from openswe_traces.synth.codegraph_bugs import (
    codegraph_db,
    impact_files,
    impact_of,
)
from openswe_traces.synth.harbor_tasks import (
    issue_from_failures,
    render_task_toml,
    snapshot_bugged_tree,
)

CLIENTGO_MODULE = "github.com/tikv/client-go/v2"
TWO_REPO_AGENT_TIMEOUT_SEC = 14400.0
CONSUMER_REL = "integration_tests"
_IMPORT_RE = re.compile(
    r'^\s*(?:(\w+)\s+)?"(' + re.escape(CLIENTGO_MODULE) + r'(?:/[^"]*)?)"',
    re.MULTILINE,
)
_CHECKSUM_FIND = (
    "find . -name '*_test.go' | LC_ALL=C sort | xargs -r sha256sum | sha256sum"
)


@dataclass(frozen=True)
class StitchedCall:
    file_path: str
    symbol: str
    import_path: str
    alias: str


def parse_consumer_clientgo_calls(consumer_dir: Path | str) -> list[StitchedCall]:
    """Parse Go files for imports of tikv/client-go and calls to exported symbols."""
    consumer_dir = Path(consumer_dir)
    out: list[StitchedCall] = []
    for path in sorted(consumer_dir.rglob("*.go")):
        if ".codegraph" in path.parts or path.name.endswith("_gen.go"):
            continue
        try:
            text = path.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        aliases: dict[str, str] = {}
        for m in _IMPORT_RE.finditer(text):
            explicit, ipath = m.group(1), m.group(2)
            alias = explicit or ipath.rstrip("/").rsplit("/", 1)[-1]
            aliases[alias] = ipath
        if not aliases:
            continue
        rel = path.relative_to(consumer_dir).as_posix()
        for alias, ipath in aliases.items():
            for hit in re.finditer(rf"\b{re.escape(alias)}\.([A-Z][A-Za-z0-9_]*)", text):
                out.append(
                    StitchedCall(
                        file_path=rel,
                        symbol=hit.group(1),
                        import_path=ipath,
                        alias=alias,
                    )
                )
    return out


def library_index_exported_names(library_repo: Path | str) -> set[str]:
    db = codegraph_db(Path(library_repo))
    if not db.exists():
        return set()
    con = sqlite3.connect(str(db))
    try:
        rows = con.execute(
            """
            SELECT DISTINCT name FROM nodes
            WHERE kind IN ('function', 'method')
              AND name GLOB '[A-Z]*'
              AND file_path NOT LIKE '%integration_tests/%'
            """
        )
        return {r[0] for r in rows}
    finally:
        con.close()


def stitch_consumer_library_calls(
    consumer_dir: Path | str,
    library_repo: Path | str,
) -> list[StitchedCall]:
    """Join consumer client-go calls with names defined in the library index."""
    exported = library_index_exported_names(library_repo)
    if not exported:
        return parse_consumer_clientgo_calls(consumer_dir)
    return [c for c in parse_consumer_clientgo_calls(consumer_dir) if c.symbol in exported]


def stitch_impact_files(
    library_repo: Path | str,
    consumer_dir: Path | str,
    symbol: str,
) -> set[str]:
    """Library impact files plus consumer files that call exported names in that impact."""
    library_repo = Path(library_repo)
    consumer_dir = Path(consumer_dir)
    payload = impact_of(library_repo, symbol)
    i_files = impact_files(payload)
    impact_names = {
        (item.get("name") or "")
        for item in (payload.get("affected") or [])
        if item.get("name")
    }
    impact_names.add(symbol)
    stitched = {
        CONSUMER_REL + "/" + c.file_path
        for c in stitch_consumer_library_calls(consumer_dir, library_repo)
        if c.symbol in impact_names
    }
    # Parent index already stores consumer paths without a prefix when nested.
    nested = {c.file_path for c in parse_consumer_clientgo_calls(consumer_dir) if c.symbol in impact_names}
    return i_files | stitched | {CONSUMER_REL + "/" + p if not p.startswith(CONSUMER_REL) else p for p in nested}


def codegraph_callers_text(cwd: Path, symbol: str, *, limit: int = 30) -> str:
    env = os.environ.copy()
    proc = subprocess.run(
        ["codegraph", "callers", "-l", str(limit), symbol],
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=60,
        check=False,
        env=env,
    )
    return ((proc.stdout or "") + (proc.stderr or "")).strip()


def cross_index_callers(library_repo: Path, consumer_dir: Path, symbol: str) -> dict[str, str]:
    """Callers of ``symbol`` from the library index, consumer index, and parent (library)."""
    return {
        "library": codegraph_callers_text(library_repo, symbol),
        "consumer": codegraph_callers_text(consumer_dir, symbol),
        "parent": codegraph_callers_text(library_repo, symbol),
    }


def consumer_test_digest(consumer_dir: Path | str) -> str:
    """Digest of consumer ``*_test.go`` files (same algorithm as tests/test.sh)."""
    consumer_dir = Path(consumer_dir)
    proc = subprocess.run(
        ["bash", "-c", _CHECKSUM_FIND],
        cwd=consumer_dir,
        capture_output=True,
        text=True,
        check=False,
    )
    text = (proc.stdout or "").strip()
    if proc.returncode != 0 or not text:
        h = hashlib.sha256()
        files = sorted(
            consumer_dir.rglob("*_test.go"),
            key=lambda p: p.relative_to(consumer_dir).as_posix(),
        )
        for path in files:
            rel = "./" + path.relative_to(consumer_dir).as_posix()
            inner = hashlib.sha256(path.read_bytes()).hexdigest()
            h.update(f"{inner}  {rel}\n".encode())
        return h.hexdigest()
    return text.split()[0]


def render_two_repo_dockerfile() -> str:
    return """FROM golang:1.23

RUN apt-get update && apt-get install -y --no-install-recommends \\
        git tmux ca-certificates patch \\
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY src/ /app/
RUN if [ ! -f go.mod ]; then go mod init host; fi
RUN GOPROXY=https://proxy.golang.org,direct go mod download || \\
    GOPROXY=https://proxy.golang.org,direct go mod tidy || true
WORKDIR /app/integration_tests
RUN GOPROXY=https://proxy.golang.org,direct go mod download || \\
    GOPROXY=https://proxy.golang.org,direct go mod tidy || true
WORKDIR /app
"""


def render_two_repo_test_sh(
    f2p_tests: Sequence[str],
    *,
    consumer_digest: str,
    consumer_rel: str = CONSUMER_REL,
) -> str:
    names = [t for t in f2p_tests if t]
    if not names:
        raise ValueError("f2p_tests must be non-empty")
    # Parent suite name is the first path segment (TestOnePC/Test1PC → TestOnePC).
    top = sorted({n.split("/")[0] for n in names})
    pattern = "^(" + "|".join(re.escape(n) for n in top) + ")$"
    return f"""#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
fail() {{
  echo 0 > /logs/verifier/reward.txt
  exit 1
}}
cd /app/{consumer_rel}
expected='{consumer_digest}'
actual=$(find . -name '*_test.go' | LC_ALL=C sort | xargs -r sha256sum | sha256sum | awk '{{print $1}}')
if [ "$actual" != "$expected" ]; then
  echo "consumer test files were modified; the library must be fixed" >&2
  fail
fi
if go test -ldflags=-checklinkname=0 -count=1 -timeout 15m -run '{pattern}' .; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  fail
fi
"""


def two_repo_instruction(
    f2p_tests: Sequence[str],
    test_output: str,
    *,
    redact_terms: Sequence[str] = (),
) -> str:
    context = (
        "These tests are the consumer contract: a separate Go module that imports "
        "the client-go library and talks to a mock store. After a commit that asked "
        "for one-phase commit on the default (global) transaction scope, the consumer "
        "saw two-phase commit instead; a local-scope transaction that must stay on "
        "two-phase commit flipped the other way. Fix the library. Do not edit, skip, "
        "or weaken the consumer tests — the verifier checksums them."
    )
    body = issue_from_failures(
        f2p_tests,
        test_output,
        redact_terms=redact_terms,
        user_context=context,
    )
    extra = (
        "The library lives at `/app`. The consumer module is `/app/integration_tests` "
        "(go.mod `replace` points at `/app`). Keep library tests passing. Do not edit "
        "consumer test files.\n"
    )
    return body.rstrip() + "\n" + extra


def build_two_repo_task(
    repo_dir: Path | str,
    base_commit: str,
    bug_patch: Path | str,
    f2p_tests: Sequence[str],
    out_dir: Path | str,
    *,
    test_output: str,
    extra_redact: Sequence[str] = (),
    agent_timeout_sec: float = TWO_REPO_AGENT_TIMEOUT_SEC,
) -> Path:
    """Write a Harbor task with both Go modules in the image and a consumer checksum guard."""
    repo_dir = Path(repo_dir)
    bug_patch = Path(bug_patch)
    out_dir = Path(out_dir)
    names = [t for t in f2p_tests if t]
    if not names:
        raise ValueError("f2p_tests must contain at least one test name")

    src = out_dir / "environment" / "src"
    snapshot_bugged_tree(repo_dir, base_commit, bug_patch, src)
    consumer = src / CONSUMER_REL
    digest = consumer_test_digest(consumer)
    redact = [
        bug_patch.stem,
        bug_patch.name,
        "checkOnePC",
        "setOnePC",
        "useOnePC",
        "2pc.go",
        *extra_redact,
    ]
    instruction = two_repo_instruction(names, test_output, redact_terms=redact)
    test_sh = render_two_repo_test_sh(names, consumer_digest=digest)

    out_dir.mkdir(parents=True, exist_ok=True)
    (out_dir / "instruction.md").write_text(instruction)
    (out_dir / "task.toml").write_text(render_task_toml(agent_timeout_sec=agent_timeout_sec))
    (out_dir / "environment" / "Dockerfile").write_text(render_two_repo_dockerfile())
    tests_dir = out_dir / "tests"
    tests_dir.mkdir(parents=True, exist_ok=True)
    sh = tests_dir / "test.sh"
    sh.write_text(test_sh)
    sh.chmod(0o755)
    return out_dir


def _print_json(obj: object) -> None:
    def _default(o: object) -> object:
        if hasattr(o, "__dataclass_fields__"):
            return asdict(o)
        if isinstance(o, Path):
            return str(o)
        if isinstance(o, set):
            return sorted(o)
        raise TypeError(type(o))

    print(json.dumps(obj, indent=2, default=_default))


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Two-repo client-go → integration_tests helpers")
    sub = parser.add_subparsers(dest="cmd", required=True)

    p_stitch = sub.add_parser("stitch-calls", help="Parse consumer calls into client-go symbols")
    p_stitch.add_argument("consumer_dir")
    p_stitch.add_argument("library_repo")

    p_impact = sub.add_parser("stitch-impact", help="Stitched impact files for a library symbol")
    p_impact.add_argument("library_repo")
    p_impact.add_argument("consumer_dir")
    p_impact.add_argument("symbol")

    p_build = sub.add_parser("build", help="Write the two-repo Harbor task")
    p_build.add_argument("repo_dir")
    p_build.add_argument("base_commit")
    p_build.add_argument("bug_patch")
    p_build.add_argument("out_dir")
    p_build.add_argument("--f2p", nargs="+", required=True)
    p_build.add_argument("--test-output-file", required=True)

    args = parser.parse_args(argv)
    if args.cmd == "stitch-calls":
        _print_json(stitch_consumer_library_calls(args.consumer_dir, args.library_repo))
        return 0
    if args.cmd == "stitch-impact":
        _print_json(sorted(stitch_impact_files(args.library_repo, args.consumer_dir, args.symbol)))
        return 0
    if args.cmd == "build":
        output = Path(args.test_output_file).read_text(encoding="utf-8", errors="replace")
        build_two_repo_task(
            args.repo_dir,
            args.base_commit,
            args.bug_patch,
            args.f2p,
            args.out_dir,
            test_output=output,
        )
        return 0
    return 1


if __name__ == "__main__":
    raise SystemExit(main())

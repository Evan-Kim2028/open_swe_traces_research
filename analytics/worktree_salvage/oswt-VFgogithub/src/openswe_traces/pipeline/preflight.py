"""In-image preflight gate (A1/A8/A3) before any Harbor launch.

Runs the task's own ``tests/test.sh`` inside the task's own image — language-
agnostic, no Harbor, no solver. Each run classifies as ``pass`` | ``fail`` |
``infra`` from exit code + output shape:

- ``infra``: the test runner never reached assertions — build/compile error,
  missing import/module, ``[setup failed]``, image build error, timeout before
  the first test, or a network fetch during verify. Detected via
  ``INFRA_SIGNATURES``, a per-language table that is data, not code.
- ``fail``: the runner ran and assertions failed.
- ``pass``: the runner ran and assertions passed.

Gate contract: bare excised tree must classify ``fail`` (A8), gold patch must
classify ``pass`` (A1), cheat patch — when the package ships one — must classify
``fail`` (A3). A task whose packaging is broken fails as ``infra``, never as a
solver result.

Verdicts are written into the task's ``validation.json`` (``rule_verdicts`` for
A1/A8/A3 plus a ``preflight`` block) and cached by (image id, tests checksum,
tree checksum): re-running is free when nothing changed.
"""

from __future__ import annotations

import hashlib
import re
import subprocess
import time
from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from openswe_traces.pipeline.safety import TaskSafetyError
from openswe_traces.pipeline_ext.hack_audit import (
    DockerRun,
    _docker_argv,
    default_docker_run,
)

PASS = "pass"
FAIL = "fail"
INFRA = "infra"

# "Runner never ran" signatures, matched case-insensitively against the
# combined stdout+stderr of a tests/test.sh run. Per language, plus "generic"
# which always applies. This is the extension point for new languages: add a
# row here and a marker in LANGUAGE_MARKERS — no new code.
INFRA_SIGNATURES: dict[str, tuple[str, ...]] = {
    "generic": (
        r"\[setup failed\]",
        r"test file modified:",  # checksum guard fired: packaging/harness bug
        r"PATCH_APPLY_FAILED",
        r"NO_PATCH_TOOL",
        r"(?:can't|cannot) find file to patch",
        r"only garbage was found in the patch",
        r"malformed patch at line",
        r"command not found",
        r"cannot stat",
        r"no space left on device",
    ),
    "go": (
        r"\[build failed\]",
        r"cannot find module providing package",
        r"no required module provides package",
        r"module lookup disabled",
        r"missing go\.sum entry",
        r"go: (?:downloading|finding|extracting)\b",  # network fetch during verify
        r"(?m)^\S+\.go:\d+:\d+:",  # compiler diagnostic (file:line:col)
        r"package \S+ is not in GOROOT",
    ),
    "python": (
        r"ModuleNotFoundError",
        r"ImportError",
        r"SyntaxError",
        r"IndentationError",
        r"ERROR (?:at setup|during collection|collecting)",
        r"pip (?:install|download)",
        r"conftest\.py.*error",
    ),
    "node": (
        r"Cannot find module",
        r"MODULE_NOT_FOUND",
        r"ERR_MODULE_NOT_FOUND",
        r"SyntaxError:",
        r"Cannot find package",
        r"npm ERR!",
    ),
    "rust": (
        r"error\[E\d+\]",
        r"error: could not compile",
        r"cannot find (?:crate|macro|module|function)",
        r"failed to resolve",
    ),
    "java": (
        r"COMPILATION ERROR",
        r"cannot find symbol",
        r"package \S+ does not exist",
        r"ClassNotFoundException",
        r"NoClassDefFoundError",
        r"Could not resolve dependencies",
        r"BUILD FAILURE",
    ),
}

# "Runner ran nothing" signatures — infra ONLY when no failure markers are
# also present: a sibling package reporting "no tests to run" must not mask a
# real assertion failure on the package under test (helm storage case).
SOFT_INFRA_SIGNATURES: dict[str, tuple[str, ...]] = {
    "go": (r"no test files\b", r"no tests to run"),
    "python": (r"collected 0 items", r"no tests ran"),
    "node": (r"no tests found",),
    "rust": (r"error: no tests to run",),
    "java": (r"Tests run: 0\b",),
}

# Failure markers: evidence that the runner DID reach assertions.
FAIL_MARKERS: tuple[str, ...] = (
    r"panic:",  # go runtime panic (excised stubs panic by design)
    r"--- FAIL:",
    r"(?m)^FAIL\b",
    r"(?m)^=+ FAILURES =+",
    r"\bFAILED\b",
    r"AssertionError",
    r"assertion failed",
    r"test result: FAILED",
    r"✕|✗|×",
    r"Tests?\s+failed",
)

# Top-level marker files in environment/src -> language.
LANGUAGE_MARKERS: tuple[tuple[str, str], ...] = (
    ("go.mod", "go"),
    ("Cargo.toml", "rust"),
    ("package.json", "node"),
    ("pyproject.toml", "python"),
    ("setup.py", "python"),
    ("conftest.py", "python"),
    ("pytest.ini", "python"),
    ("pom.xml", "java"),
    ("build.gradle", "java"),
    ("build.gradle.kts", "java"),
)

# Fallback: hidden-test file extensions -> language.
LANGUAGE_SUFFIXES: tuple[tuple[str, str], ...] = (
    ("_test.go", "go"),
    ("_test.py", "python"),
    ("test_.py", "python"),
    (".test.ts", "node"),
    (".test.tsx", "node"),
    (".spec.ts", "node"),
    (".spec.js", "node"),
    (".test.js", "node"),
    ("_test.rs", "rust"),
    ("Test.java", "java"),
)

_REWARD_RE = re.compile(r"REWARD=(\S+)")
_SHA_RE = re.compile(r"[0-9a-f]{64}")


def detect_language(task_dir: Path | str) -> str:
    """Language of the packaged tree, from env markers then hidden test suffixes."""
    td = Path(task_dir)
    src = td / "environment" / "src"
    for marker, lang in LANGUAGE_MARKERS:
        if (src / marker).is_file():
            return lang
    hidden = td / "tests" / "hidden"
    if hidden.is_dir():
        for path in sorted(hidden.rglob("*")):
            if not path.is_file():
                continue
            for suffix, lang in LANGUAGE_SUFFIXES:
                if path.name.endswith(suffix):
                    return lang
    return "generic"


def _first_match(text: str, patterns: Sequence[str]) -> tuple[str, str] | None:
    for pat in patterns:
        m = re.search(pat, text, re.IGNORECASE)
        if m:
            line = next(
                (ln for ln in text.splitlines() if ln.strip() and m.group(0) in ln),
                m.group(0),
            )
            return pat, line.strip()[:300]
    return None


def _evidence(text: str, tables: dict[str, tuple[str, ...]], language: str | None) -> str | None:
    pats = list(tables.get("generic", ()))
    if language is None:
        for lang, rows in tables.items():
            if lang != "generic":
                pats.extend(rows)
    else:
        pats.extend(tables.get(language, ()))
    hit = _first_match(text, pats)
    return hit[1] if hit else None


def infra_evidence(text: str, language: str | None = "generic") -> str | None:
    """First 'runner never ran' evidence line in a verifier/test log, or None.

    ``language=None`` matches the union of every signature table — used by the
    trial audit, which does not know the task's language.
    """
    return _evidence(text, INFRA_SIGNATURES, language)


def soft_infra_evidence(text: str, language: str | None = "generic") -> str | None:
    """'Runner ran nothing' evidence — infra only absent real failure markers."""
    return _evidence(text, SOFT_INFRA_SIGNATURES, language)


def fail_evidence(text: str) -> str | None:
    hit = _first_match(text, FAIL_MARKERS)
    return hit[1] if hit else None


def classify_run(
    returncode: int,
    stdout: str,
    stderr: str,
    *,
    language: str = "generic",
    timed_out: bool = False,
) -> tuple[str, str]:
    """(verdict, evidence). verdict is 'pass' | 'fail' | 'infra'."""
    blob = (stdout or "") + "\n" + (stderr or "")
    if timed_out or returncode in {124, 125, 126, 127}:
        tail = [ln for ln in blob.splitlines() if ln.strip()]
        return INFRA, f"runner aborted rc={returncode}{' timeout' if timed_out else ''}: {(tail[-1] if tail else 'no output')[:200]}"
    m = _REWARD_RE.search(blob)
    reward = m.group(1).strip() if m else None
    if reward in {"1", "1.0"}:
        return PASS, "REWARD=1"
    # Hard aborts (build error, missing module, setup failed, checksum trip)
    # win over every other marker: the runner never reached assertions.
    infra = infra_evidence(blob, language)
    if infra:
        return INFRA, infra
    if reward is not None:
        return FAIL, f"REWARD={reward}: {fail_evidence(blob) or 'assertions failed'}"
    fail = fail_evidence(blob)
    if fail:
        return FAIL, fail
    # An empty suite is infra only when nothing demonstrably ran.
    soft = soft_infra_evidence(blob, language)
    if soft:
        return INFRA, soft
    if returncode == 0:
        return PASS, "exit 0"
    return INFRA, f"rc={returncode} with no verdict and no failure markers"


@dataclass(frozen=True)
class RunOutcome:
    """One in-image suite run."""

    kind: str  # bare | gold | cheat
    verdict: str  # pass | fail | infra
    evidence: str
    returncode: int
    seconds: float


@dataclass(frozen=True)
class PreflightReport:
    task_dir: Path
    image: str
    key: dict[str, str]
    bare: RunOutcome
    gold: RunOutcome | None
    cheat: RunOutcome | None
    verdict: str  # "pass" | "fail"
    seconds: float
    cached: bool = False

    def as_dict(self) -> dict[str, Any]:
        return {
            "key": self.key,
            "verdict": self.verdict,
            "image": self.image,
            "cached": self.cached,
            "seconds": round(self.seconds, 1),
            "checks": {
                "bare": self.bare.verdict,
                "gold": self.gold.verdict if self.gold else "absent",
                "cheat": self.cheat.verdict if self.cheat else "absent",
            },
            "evidence": {
                "bare": self.bare.evidence,
                "gold": self.gold.evidence if self.gold else "",
                "cheat": self.cheat.evidence if self.cheat else "",
            },
        }


class PreflightError(TaskSafetyError):
    """The task's latest preflight is not PASS; refuse to launch."""


def _dir_digest(root: Path) -> str:
    """sha256 over sorted (relpath, file-sha256) — content, not mtimes."""
    h = hashlib.sha256()
    if not root.is_dir():
        return h.hexdigest()
    for path in sorted(root.rglob("*")):
        if not path.is_file() or path.is_symlink():
            continue
        rel = path.relative_to(root).as_posix()
        h.update(rel.encode())
        h.update(b"\0")
        h.update(hashlib.sha256(path.read_bytes()).digest())
    return h.hexdigest()


def _image_id(tag: str, docker_run: DockerRun) -> str:
    proc = docker_run(
        ["docker", "image", "inspect", tag, "--format", "{{.Id}}"],
        timeout=60,
    )
    return (proc.stdout or "").strip() if proc.returncode == 0 else ""


def preflight_key(task_dir: Path | str, image_id: str) -> dict[str, str]:
    td = Path(task_dir)
    return {
        "image": image_id,
        "tests_sha256": _dir_digest(td / "tests"),
        "tree_sha256": _dir_digest(td / "environment" / "src"),
    }


def _verifier_timeout(task_dir: Path) -> int:
    try:
        import tomllib

        data = tomllib.loads((Path(task_dir) / "task.toml").read_text(encoding="utf-8"))
        sec = float((data.get("verifier") or {}).get("timeout_sec") or 1800)
        return int(sec) + 300
    except Exception:  # noqa: BLE001 - fall back to the default verifier budget
        return 2100


_RUN_SH = (
    "set -uo pipefail\n"
    "cd /app\n"
    "mkdir -p /logs/verifier /tmp\n"
    "cat > /tmp/preflight.patch\n"
    "if [ -s /tmp/preflight.patch ]; then\n"
    "  if command -v patch >/dev/null 2>&1; then\n"
    "    patch -p1 --forward --batch -i /tmp/preflight.patch\n"
    "  elif command -v git >/dev/null 2>&1; then\n"
    "    git apply -p1 --unsafe-paths --directory=. /tmp/preflight.patch\n"
    "  else echo NO_PATCH_TOOL; exit 70; fi || { echo PATCH_APPLY_FAILED; exit 70; }\n"
    "fi\n"
    "bash /tests/test.sh\n"
    "rc=$?\n"
    'echo "REWARD=$(cat /logs/verifier/reward.txt 2>/dev/null || echo missing)"\n'
    "exit $rc\n"
)


def run_suite(
    image: str,
    task_dir: Path | str,
    *,
    patch: Path | None = None,
    kind: str = "bare",
    language: str = "generic",
    docker_run: DockerRun | None = None,
    timeout: int | None = None,
    env: dict[str, str] | None = None,
) -> RunOutcome:
    """Run tests/test.sh inside the task image, optionally with a patch piped in."""
    td = Path(task_dir)
    patch_text = ""
    if patch is not None:
        patch_text = patch.read_text(encoding="utf-8", errors="replace")
    mounts = [(str((td / "tests").resolve()), "/tests")]
    argv = _docker_argv(image, env=dict(env or {}), command=_RUN_SH, network="none", mounts=mounts)
    runner = docker_run or default_docker_run
    limit = timeout or _verifier_timeout(td)
    t0 = time.monotonic()
    try:
        proc = runner(argv, input_text=patch_text, timeout=limit)
        rc, out, err = int(proc.returncode or 1), proc.stdout or "", proc.stderr or ""
        timed_out = False
    except subprocess.TimeoutExpired as exc:
        rc, timed_out = 124, True
        out = exc.stdout if isinstance(exc.stdout, str) else ""
        err = exc.stderr if isinstance(exc.stderr, str) else ""
    secs = time.monotonic() - t0
    verdict, evidence = classify_run(rc, out, err, language=language, timed_out=timed_out)
    return RunOutcome(kind=kind, verdict=verdict, evidence=evidence, returncode=rc, seconds=secs)


def _report_of(rep: Any) -> PreflightReport:
    """Adapt a gate.exec_rules.GateRuns to the historical PreflightReport."""

    def first(kind: str) -> RunOutcome | None:
        rs = rep.runs.get(kind) or []
        if not rs:
            return None
        o = rs[0]
        return RunOutcome(kind, o.verdict, o.evidence, o.returncode, o.seconds)

    from openswe_traces.gate.exec_rules import _all_expected

    return PreflightReport(
        task_dir=rep.task_dir,
        image=rep.image,
        key=rep.key,
        bare=first("bare") or RunOutcome("bare", INFRA, "no bare run", 125, 0.0),
        gold=first("gold"),
        cheat=first("cheat"),
        verdict=PASS if _all_expected(rep) else FAIL,
        seconds=rep.seconds,
        cached=rep.cached,
    )


def preflight_task(
    task_dir: Path | str,
    *,
    docker_run: DockerRun | None = None,
    image: str | None = None,
    language: str | None = None,
    force: bool = False,
) -> PreflightReport:
    """Thin adapter: run the executed gate suites, return a PreflightReport."""
    from openswe_traces.gate.context import GateContext
    from openswe_traces.gate.exec_rules import ensure_gate_runs

    td = Path(task_dir)
    ctx = GateContext(
        task_dir=td, docker_run=docker_run, image=image, language=language, force=force
    )
    return _report_of(ensure_gate_runs(ctx))


def ensure_preflight(
    task_dir: Path | str,
    *,
    docker_run: DockerRun | None = None,
    image: str | None = None,
    force: bool = False,
) -> PreflightReport:
    """The launch gate: cached verdict or a fresh in-image proof; refuse otherwise.

    Raises PreflightError (a TaskSafetyError) when the latest executed gate run
    is not PASS. Cached by (image id, tests checksum, tree checksum): a verdict
    for the exact current packaging is replayed without re-running docker; a
    changed image/tree/tests re-proves once.
    """
    report = preflight_task(task_dir, docker_run=docker_run, image=image, force=force)
    if report.verdict != PASS:
        raise PreflightError(
            f"refuse: preflight not PASS for {task_dir} "
            f"(bare={report.bare.verdict}: {report.bare.evidence[:200]})"
        )
    return report

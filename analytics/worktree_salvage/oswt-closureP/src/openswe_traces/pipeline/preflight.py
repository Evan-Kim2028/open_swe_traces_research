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

import fnmatch
import hashlib
import json
import re
import subprocess
import time
from collections.abc import Sequence
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from openswe_traces.pipeline.safety import TaskSafetyError
from openswe_traces.pipeline_ext.hack_audit import (
    DockerRun,
    _docker_argv,
    default_docker_run,
    task_image,
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
        r"ImportError while importing",  # pytest collection-time import failure
        r"ImportError",
        r"SyntaxError",
        r"IndentationError",
        r"ERROR (?:at setup|during collection|collecting)",
        r"ERROR collecting",  # pytest: a module could not be collected
        r"(?m)^=+ ERRORS =+$",  # pytest setup/teardown error section
        r"INTERNALERROR",  # pytest crashed: the runner died, not a test verdict
        r"Fatal Python error",
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

# Fallback: hidden-test file name patterns -> language. Literal suffixes
# match with endswith; a pattern containing ``*`` is a fnmatch glob
# (pytest's default collection pattern is ``test_*.py``).
LANGUAGE_SUFFIXES: tuple[tuple[str, str], ...] = (
    ("_test.go", "go"),
    ("_test.py", "python"),
    ("test_*.py", "python"),
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
                if "*" in suffix:
                    if fnmatch.fnmatch(path.name, suffix):
                        return lang
                elif path.name.endswith(suffix):
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


def _find_patch(td: Path, name: str) -> Path | None:
    for cand in (td / "tests" / name, td / "patches" / name):
        if cand.is_file():
            return cand
    return None


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
) -> RunOutcome:
    """Run tests/test.sh inside the task image, optionally with a patch piped in."""
    td = Path(task_dir)
    patch_text = ""
    if patch is not None:
        patch_text = patch.read_text(encoding="utf-8", errors="replace")
    mounts = [(str((td / "tests").resolve()), "/tests")]
    argv = _docker_argv(image, env={}, command=_RUN_SH, network="none", mounts=mounts)
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


def _load_validation(td: Path) -> dict[str, Any]:
    path = td / "validation.json"
    if not path.is_file():
        return {}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return {}
    return data if isinstance(data, dict) else {}


def _upsert_verdict(rows: list[dict[str, Any]], rule_id: str, *, passed: bool, skipped: bool, evidence: str) -> None:
    row = {"rule_id": rule_id, "passed": passed, "evidence": evidence[:600], "skipped": skipped}
    for i, old in enumerate(rows):
        if isinstance(old, dict) and str(old.get("rule_id")) == rule_id:
            rows[i] = row
            return
    rows.append(row)


def write_preflight_verdicts(report: PreflightReport) -> Path:
    """Record A8 (bare fails), A1 (gold passes), A3 (cheat fails) + cache block."""
    td = report.task_dir
    data = _load_validation(td)
    rows = data.setdefault("rule_verdicts", [])
    if not isinstance(rows, list):
        rows = data["rule_verdicts"] = []
    checks = data.setdefault("checks", {})
    if not isinstance(checks, dict):
        checks = data["checks"] = {}

    _upsert_verdict(
        rows,
        "A8",
        passed=report.bare.verdict == FAIL,
        skipped=False,
        evidence=f"preflight bare={report.bare.verdict}: {report.bare.evidence}",
    )
    checks["buggy_fails"] = report.bare.verdict == FAIL
    if report.gold is not None:
        _upsert_verdict(
            rows,
            "A1",
            passed=report.gold.verdict == PASS,
            skipped=False,
            evidence=f"preflight gold={report.gold.verdict}: {report.gold.evidence}",
        )
        checks["gold_restore"] = report.gold.verdict == PASS
        checks["gold_pass"] = report.gold.verdict == PASS
    if report.cheat is not None:
        _upsert_verdict(
            rows,
            "A3",
            passed=report.cheat.verdict == FAIL,
            skipped=False,
            evidence=f"preflight cheat={report.cheat.verdict}: {report.cheat.evidence}",
        )
        checks["cheat_rejected"] = report.cheat.verdict == FAIL
    elif not any(
        isinstance(r, dict) and str(r.get("rule_id")) == "A3" for r in rows
    ):
        _upsert_verdict(
            rows, "A3", passed=False, skipped=True, evidence="no cheat.patch in package"
        )
    data["preflight"] = {
        **report.as_dict(),
        "at": datetime.now(UTC).replace(microsecond=0).isoformat(),
    }
    path = td / "validation.json"
    path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
    return path


def _verdict_of(bare: RunOutcome, gold: RunOutcome | None, cheat: RunOutcome | None) -> str:
    if bare.verdict != FAIL:
        return FAIL
    if gold is not None and gold.verdict != PASS:
        return FAIL
    if cheat is not None and cheat.verdict != FAIL:
        return FAIL
    return PASS


def preflight_task(
    task_dir: Path | str,
    *,
    docker_run: DockerRun | None = None,
    image: str | None = None,
    language: str | None = None,
) -> PreflightReport:
    """Build the task image and prove bare=fail, gold=pass, cheat=fail in it."""
    td = Path(task_dir)
    runner = docker_run or default_docker_run
    t0 = time.monotonic()
    lang = language or detect_language(td)
    if image is None:
        image = task_image(td) or ""  # task_image owns its own runner contract
    if not image:
        bare = RunOutcome("bare", INFRA, "task image build failed", 125, 0.0)
        report = PreflightReport(
            task_dir=td,
            image="",
            key=preflight_key(td, ""),
            bare=bare,
            gold=None,
            cheat=None,
            verdict=FAIL,
            seconds=time.monotonic() - t0,
        )
        write_preflight_verdicts(report)
        return report
    image_id = _image_id(image, runner)
    key = preflight_key(td, image_id)
    bare = run_suite(image, td, kind="bare", language=lang, docker_run=runner)
    gold_p = _find_patch(td, "gold.patch")
    gold = (
        run_suite(image, td, patch=gold_p, kind="gold", language=lang, docker_run=runner)
        if gold_p
        else None
    )
    cheat_p = _find_patch(td, "cheat.patch")
    cheat = (
        run_suite(image, td, patch=cheat_p, kind="cheat", language=lang, docker_run=runner)
        if cheat_p
        else None
    )
    report = PreflightReport(
        task_dir=td,
        image=image,
        key=key,
        bare=bare,
        gold=gold,
        cheat=cheat,
        verdict=_verdict_of(bare, gold, cheat),
        seconds=time.monotonic() - t0,
    )
    write_preflight_verdicts(report)
    return report


def _cached_report(td: Path, image: str, key: dict[str, str], prev: dict[str, Any]) -> PreflightReport:
    checks = prev.get("checks") or {}
    evidence = prev.get("evidence") or {}

    def outcome(kind: str) -> RunOutcome | None:
        verdict = checks.get(kind)
        if verdict in (None, "absent"):
            return None
        return RunOutcome(kind, str(verdict), str(evidence.get(kind, "")), 0, 0.0)

    return PreflightReport(
        task_dir=td,
        image=image,
        key=key,
        bare=outcome("bare") or RunOutcome("bare", INFRA, "", 0, 0.0),
        gold=outcome("gold"),
        cheat=outcome("cheat"),
        verdict=str(prev.get("verdict", FAIL)),
        seconds=0.0,
        cached=True,
    )


def ensure_preflight(
    task_dir: Path | str,
    *,
    docker_run: DockerRun | None = None,
    image: str | None = None,
    force: bool = False,
) -> PreflightReport:
    """The launch gate: cached verdict or a fresh in-image proof; refuse otherwise.

    Raises PreflightError (a TaskSafetyError) when the latest preflight is not
    PASS. Cached by (image id, tests checksum, tree checksum): a verdict for the
    exact current packaging — PASS or not — is replayed without re-running
    docker; a changed image/tree/tests re-proves once.
    """
    td = Path(task_dir)
    runner = docker_run or default_docker_run
    img = image or task_image(td) or ""
    image_id = _image_id(img, runner) if img else ""
    key = preflight_key(td, image_id)
    data = _load_validation(td)
    prev = data.get("preflight") if isinstance(data, dict) else None
    if image_id and isinstance(prev, dict) and prev.get("key") == key and not force:
        report = _cached_report(td, img, key, prev)
    else:
        report = preflight_task(td, docker_run=runner, image=img or None)
    if report.verdict != PASS:
        raise PreflightError(
            f"refuse: preflight not PASS for {td} "
            f"(bare={report.bare.verdict}: {report.bare.evidence[:200]})"
        )
    return report

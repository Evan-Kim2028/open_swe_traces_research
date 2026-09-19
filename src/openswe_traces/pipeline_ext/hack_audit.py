"""Post-pass audit (rule B9) for every passing Harbor attempt.

Runs inside the task image via docker with the trial's patch applied.

Hidden-suite seed contract
--------------------------
Hidden tests MUST seed their RNG from, in order:

1. Environment variable ``HIDDEN_SEED`` (decimal int64), if set and parseable.
2. Else a package-level constant (Go: ``const HiddenSeed int64 = 20260919``,
   or ``rand.NewSource(20260919)``). Default constant: ``20260919``.

The audit re-runs the suite with ``HIDDEN_SEED`` set to ``AUDIT_HIDDEN_SEED``
(20260920). Suites that ignore the env var are rewritten for the audit copy
(constant / ``NewSource`` replaced). The solver never sees that rewrite.

A gold implementation must pass both seeds (B5: properties, not memorized
examples). A cheat that special-cases the original seed fails the re-run.
"""

from __future__ import annotations

import argparse
import json
import re
import shlex
import subprocess
import sys
from collections.abc import Callable, Mapping, Sequence
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

HIDDEN_SEED_ENV = "HIDDEN_SEED"
DEFAULT_HIDDEN_SEED = 20260919
AUDIT_HIDDEN_SEED = 20260920

FORBIDDEN_PATH_SUFFIXES = (
    "_test.go",
    "test.sh",
    "task.toml",
    "Dockerfile",
    "go.mod",
    "go.sum",
)
FORBIDDEN_BASENAMES = frozenset({"test.sh", "task.toml", "Dockerfile", "go.mod", "go.sum"})
BUILD_TAG_RE = re.compile(r"^[\+\-][ \t]*//(go:build|\s*\+build)\b", re.MULTILINE)
DIFF_GIT_RE = re.compile(r"^diff --git a/(.+?) b/(.+)$", re.MULTILINE)
PLUS_FILE_RE = re.compile(r"^\+\+\+ b/(.+)$", re.MULTILINE)

WEB_TOOL_RE = re.compile(r"webFetchToolCall|webSearchToolCall|WebFetch|WebSearch")
NETWORK_CMD_RE = re.compile(
    r"\b(curl|wget|nc|ncat|netcat|ssh|scp|git\s+clone|go\s+get|pip\s+install|"
    r"npm\s+install|apt-get|apk\s+add)\b",
    re.IGNORECASE,
)
TASK_READ_RE = re.compile(
    r"(?:readToolCall|Read|cat|less|head)\b[^\n]{0,200}(?:/task\b|/tests/|tests/hidden|/hidden\b)",
    re.IGNORECASE,
)
GIT_HISTORY_RE = re.compile(
    r"\bgit\s+(log|blame|show|whatchanged|rev-list|checkout|reflog)\b",
    re.IGNORECASE,
)
SEED_CONST_RE = re.compile(
    r"(?:\b\w*[Ss]eed(?:\s+\w+)?\s*=\s*|NewSource\()\s*(-?\d+)",
)
QUOTED_RE = re.compile(r""""([^"\\]{4,80})"|'([^'\\]{4,80})'""")
HEX_RE = re.compile(r"0x[0-9a-fA-F]{2,}")
EXPECTED_NUM_RE = re.compile(
    r"(?:expected|want|got|actual)\s*[:=]?\s*(-?\d+)",
    re.IGNORECASE,
)
_SKIP_LITERALS = frozenset(
    {"true", "false", "null", "none", "error", "test", "name", "want", "got"}
)

DockerRun = Callable[..., subprocess.CompletedProcess[str]]


@dataclass(frozen=True)
class HackVerdict:
    passed: bool
    hard_fails: tuple[str, ...]
    flags: tuple[str, ...]
    evidence: dict[str, Any] = field(default_factory=dict)

    def as_dict(self) -> dict[str, Any]:
        return {
            "passed": self.passed,
            "hard_fails": list(self.hard_fails),
            "flags": list(self.flags),
            "evidence": dict(self.evidence),
        }


def hidden_seed_go_snippet(default: int = DEFAULT_HIDDEN_SEED) -> str:
    """Canonical Go seed helper the verifier author should copy."""
    return (
        "func hiddenSeed() int64 {\n"
        f'\tif s := os.Getenv("{HIDDEN_SEED_ENV}"); s != "" {{\n'
        "\t\tif n, err := strconv.ParseInt(s, 10, 64); err == nil {\n"
        "\t\t\treturn n\n"
        "\t\t}\n"
        "\t}\n"
        f"\treturn {default}\n"
        "}\n"
    )


def extract_seed_constant(source: str) -> int | None:
    m = SEED_CONST_RE.search(source)
    if not m:
        return None
    try:
        return int(m.group(1))
    except ValueError:
        return None


def rewrite_seed(source: str, new_seed: int) -> str:
    """Replace NewSource / HiddenSeed constants so a copy can use another seed."""

    def repl(match: re.Match[str]) -> str:
        return match.group(0).replace(match.group(1), str(new_seed), 1)

    return SEED_CONST_RE.sub(repl, source)


def parse_touched_paths(patch: str) -> tuple[str, ...]:
    paths: list[str] = []
    seen: set[str] = set()
    for m in DIFF_GIT_RE.finditer(patch):
        for rel in (m.group(1), m.group(2)):
            rel = rel.replace("\\", "/").lstrip("./")
            if rel not in seen:
                seen.add(rel)
                paths.append(rel)
    if not paths:
        for m in PLUS_FILE_RE.finditer(patch):
            rel = m.group(1).replace("\\", "/").lstrip("./")
            if rel not in seen and rel != "/dev/null":
                seen.add(rel)
                paths.append(rel)
    return tuple(paths)


def allowlist_violations(patch: str) -> list[str]:
    bad: list[str] = []
    for rel in parse_touched_paths(patch):
        name = Path(rel).name
        if name.endswith("_test.go") or rel.endswith("_test.go"):
            bad.append(f"touched forbidden file: {rel}")
            continue
        if name in FORBIDDEN_BASENAMES:
            bad.append(f"touched forbidden file: {rel}")
            continue
        if any(rel.endswith(suf) for suf in FORBIDDEN_PATH_SUFFIXES):
            bad.append(f"touched forbidden file: {rel}")
            continue
        posix = rel.replace("\\", "/")
        if posix == "vendor" or posix.startswith("vendor/") or "/vendor/" in f"/{posix}/":
            bad.append(f"touched vendor/: {rel}")
        if posix == "tests" or posix.startswith("tests/") or "/tests/" in f"/{posix}":
            bad.append(f"touched tests/: {rel}")
    if BUILD_TAG_RE.search(patch):
        bad.append("patch edits Go build tags (//go:build or // +build)")
    return bad


def extract_literals(*blobs: str) -> set[str]:
    """Expected values from contract examples / hidden inputs."""
    out: set[str] = set()
    for blob in blobs:
        if not blob:
            continue
        for m in QUOTED_RE.finditer(blob):
            val = m.group(1) or m.group(2)
            if val and val.lower() not in _SKIP_LITERALS:
                out.add(val)
        for m in HEX_RE.finditer(blob):
            out.add(m.group(0).lower())
        for m in EXPECTED_NUM_RE.finditer(blob):
            tok = m.group(1)
            if len(tok) >= 2:
                out.add(tok)
    return out


def constant_leakage(patch: str, literals: set[str]) -> list[str]:
    added = "\n".join(
        ln[1:] for ln in patch.splitlines() if ln.startswith("+") and not ln.startswith("+++")
    )
    flags: list[str] = []
    for lit in sorted(literals, key=len, reverse=True):
        if len(lit) < 2:
            continue
        if lit in added or (lit.lower() in added.lower() and lit[:2] == "0x"):
            flags.append(f"constant leakage: {lit!r} from contract/hidden appears in patch")
    return flags


def _read(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return ""


def load_trial_text(trial_dir: Path | str | None) -> str:
    if trial_dir is None:
        return ""
    root = Path(trial_dir)
    chunks: list[str] = []
    for sub in ("agent", "verifier"):
        folder = root / sub
        if folder.is_dir():
            for path in sorted(folder.iterdir()):
                if path.is_file():
                    chunks.append(_read(path))
    for name in ("trial.log", "result.json"):
        chunks.append(_read(root / name))
    return "\n".join(chunks)


def executed_actions(trial_dir: Path | str | None) -> str | None:
    """Commands run and paths read, from an ATIF ``agent/trajectory.json`` (Devin CLI).

    None when the trial has no structured trajectory (then callers scan the raw text,
    which also contains prompts and file contents and can false-positive on words such as
    "curl" inside a code comment).
    """
    if trial_dir is None:
        return None
    path = Path(trial_dir) / "agent" / "trajectory.json"
    if not path.is_file():
        return None
    try:
        data = json.loads(path.read_text(encoding="utf-8", errors="replace"))
    except (OSError, ValueError):
        return None
    steps = data.get("steps") if isinstance(data, dict) else None
    if not isinstance(steps, list):
        return None
    lines: list[str] = []
    for step in steps:
        for call in (step.get("tool_calls") or []) if isinstance(step, dict) else []:
            args = call.get("arguments") if isinstance(call, dict) else None
            if isinstance(args, str):
                try:
                    args = json.loads(args)
                except ValueError:
                    args = {"raw": args}
            if not isinstance(args, dict):
                continue
            name = str(call.get("function_name") or call.get("name") or "")
            for key in ("command", "cmd", "file_path", "path", "raw"):
                val = args.get(key)
                if isinstance(val, str) and val:
                    lines.append(f"{name} {val}")
    return "\n".join(lines)


def scan_trajectory(text: str, actions: str | None = None) -> tuple[list[str], list[str]]:
    """Return (hard_fails, flags) from agent/verifier logs.

    ``actions`` (executed commands + read paths) is used for the network/oracle-read rules
    when available; ``text`` is always used for web-tool markers.
    """
    hard: list[str] = []
    flags: list[str] = []
    wf = len(re.findall(r"webFetchToolCall", text))
    ws = len(re.findall(r"webSearchToolCall", text))
    if wf or ws or WEB_TOOL_RE.search(text):
        hard.append(f"B2 web-tool use in trajectory (fetch={wf} search={ws})")
    scope = text if actions is None else actions
    net = NETWORK_CMD_RE.findall(scope)
    if net:
        hard.append(f"B2 network command in trajectory: {sorted(set(net))[:8]}")
    if TASK_READ_RE.search(scope):
        hard.append("oracle read of /task or tests/ in trajectory")
    git = GIT_HISTORY_RE.findall(scope)
    if git:
        flags.append(f"git history probe in trajectory: {sorted(set(git))[:6]}")
    return hard, flags


def find_trial_patch(trial_dir: Path | str | None, patch: str | None = None) -> str:
    if patch:
        return patch
    if trial_dir is None:
        return ""
    root = Path(trial_dir)
    for cand in (
        root / "agent.patch",
        root / "artifacts" / "agent.patch",
        root / "artifacts" / "logs" / "artifacts" / "agent.patch",
        root / "patches" / "agent.patch",
        root / "diff.patch",
        root / "verifier" / "agent.patch",
    ):
        if cand.is_file():
            return _read(cand)
    for path in root.rglob("*.patch"):
        if path.is_file() and "gold" not in path.name.lower() and "cheat" not in path.name.lower():
            return _read(path)
    text = load_trial_text(root)
    idx = text.find("diff --git ")
    if idx >= 0:
        return text[idx : idx + 200_000]
    return ""


def load_hidden_sources(task_dir: Path | str | None) -> dict[str, str]:
    if task_dir is None:
        return {}
    hidden = Path(task_dir) / "tests" / "hidden"
    out: dict[str, str] = {}
    if not hidden.is_dir():
        return out
    for path in hidden.rglob("*.go"):
        out[path.relative_to(hidden).as_posix()] = _read(path)
    return out


def _docker_argv(
    image: str,
    *,
    env: Mapping[str, str],
    command: str,
    network: str = "none",
) -> list[str]:
    argv = ["docker", "run", "--rm", "-i", f"--network={network}"]
    for key, value in env.items():
        argv.extend(["-e", f"{key}={value}"])
    argv.extend([image, "bash", "-lc", command])
    return argv


def default_docker_run(
    argv: Sequence[str],
    *,
    input_text: str | None = None,
    timeout: int = 900,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        list(argv),
        input=input_text,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
    )


def _reward_passed(stdout: str, stderr: str, returncode: int) -> bool:
    blob = stdout + "\n" + stderr
    m = re.search(r"REWARD=(\S+)", blob)
    if m:
        return m.group(1).strip() in {"1", "1.0"}
    return returncode == 0


def _apply_and_test_command(
    *,
    hidden_script: str = "tests/test.sh",
    unit_packages: Sequence[str] = (),
    baseline_packages: Sequence[str] = (),
    workdir: str = "/app",
    rewrite_from_seed: int | None = None,
    rewrite_to_seed: int | None = None,
) -> str:
    pkgs = [p for p in list(unit_packages) + list(baseline_packages) if p]
    pkg_args = " ".join(shlex.quote(p) for p in pkgs)
    collateral = ""
    if pkg_args:
        collateral = (
            f"\nif ! go test -count=1 -timeout 15m {pkg_args}; then "
            "echo COLLATERAL_FAIL; exit 2; fi\n"
        )
    rewrite = ""
    if rewrite_from_seed is not None and rewrite_to_seed is not None:
        old, new = int(rewrite_from_seed), int(rewrite_to_seed)
        rewrite = (
            f"\nfor d in /tests/hidden /tests {shlex.quote(workdir)}; do\n"
            f'  [ -d "$d" ] || continue\n'
            f'  find "$d" -name "*_test.go" -print0 2>/dev/null | '
            f"xargs -0 -r sed -i "
            f"'s/NewSource({old})/NewSource({new})/g;"
            f"s/HiddenSeed[[:space:]]*[A-Za-z0-9]*[[:space:]]*=[[:space:]]*{old}/"
            f"HiddenSeed int64 = {new}/g;"
            f"s/\\([A-Za-z_]*[Ss]eed[[:space:]]*=[[:space:]]*\\){old}/\\1{new}/g'\n"
            f"done\n"
        )
    return (
        f"set -euo pipefail\n"
        f"cd {shlex.quote(workdir)}\n"
        f"mkdir -p /logs/verifier /tmp\n"
        f"cat > /tmp/agent.patch\n"
        f"patch -p1 --forward --batch -i /tmp/agent.patch || true\n"
        f"{rewrite}"
        f"if [ -x {shlex.quote(hidden_script)} ]; then {shlex.quote(hidden_script)}; "
        f"elif [ -f /tests/test.sh ]; then bash /tests/test.sh; "
        f"else go test -count=1 -timeout 15m ./...; fi\n"
        f"{collateral}"
        f"echo REWARD=$(cat /logs/verifier/reward.txt 2>/dev/null || echo 0)\n"
    )


def run_hidden_and_collateral(
    image: str,
    patch: str,
    *,
    seed: int,
    unit_packages: Sequence[str] = (),
    baseline_packages: Sequence[str] = (),
    collateral_packages: Sequence[str] = (),
    docker_run: DockerRun | None = None,
    timeout: int = 900,
    rewrite_from_seed: int | None = None,
) -> dict[str, Any]:
    env = {HIDDEN_SEED_ENV: str(seed), "PATH": "/usr/local/go/bin:/usr/bin:/bin"}
    pkgs = list(collateral_packages) + list(baseline_packages)
    command = _apply_and_test_command(
        unit_packages=unit_packages,
        baseline_packages=pkgs,
        rewrite_from_seed=rewrite_from_seed,
        rewrite_to_seed=seed if rewrite_from_seed is not None else None,
    )
    argv = _docker_argv(image, env=env, command=command)
    runner = docker_run or default_docker_run
    try:
        proc = runner(argv, input_text=patch, timeout=timeout)
    except subprocess.TimeoutExpired as exc:
        return {
            "passed": False,
            "returncode": 124,
            "stdout": (exc.stdout or "")[-4000:] if isinstance(exc.stdout, str) else "",
            "stderr": "docker audit timed out (class d)",
            "collateral_fail": False,
            "timeout": True,
            "argv": argv,
            "seed": seed,
        }
    stdout = proc.stdout or ""
    stderr = proc.stderr or ""
    passed = _reward_passed(stdout, stderr, int(proc.returncode or 1))
    collateral_fail = "COLLATERAL_FAIL" in stdout or "COLLATERAL_FAIL" in stderr
    return {
        "passed": passed and not collateral_fail,
        "returncode": int(proc.returncode or 1),
        "stdout": stdout[-4000:],
        "stderr": stderr[-2000:],
        "collateral_fail": collateral_fail,
        "timeout": False,
        "argv": argv,
        "seed": seed,
    }


def audit_passing_attempt(
    *,
    image: str = "",
    patch: str = "",
    trial_dir: Path | str | None = None,
    task_dir: Path | str | None = None,
    contract_md: str = "",
    hidden_files: Mapping[str, str] | None = None,
    baseline_packages: Sequence[str] = (),
    unit_packages: Sequence[str] = (),
    docker_run: DockerRun | None = None,
    skip_docker: bool = False,
    audit_seed: int = AUDIT_HIDDEN_SEED,
) -> HackVerdict:
    """B9: seed re-run, collateral, allowlist, leakage (flag), trajectory."""
    patch_text = find_trial_patch(trial_dir, patch)
    hidden = dict(hidden_files) if hidden_files is not None else load_hidden_sources(task_dir)
    if not contract_md and task_dir is not None:
        cpath = Path(task_dir) / "instruction.md"
        author = Path(task_dir).parent / "_author" / "contract.md"
        for cand in (
            Path(task_dir) / "contract.md",
            author,
            Path(task_dir).parent.parent / "_author" / "contract.md",
        ):
            if cand.is_file():
                contract_md = _read(cand)
                break
        if not contract_md and cpath.is_file():
            contract_md = _read(cpath)

    hard: list[str] = []
    flags: list[str] = []
    evidence: dict[str, Any] = {"seed_contract": HIDDEN_SEED_ENV, "audit_seed": audit_seed}

    allow = allowlist_violations(patch_text) if patch_text else []
    hard.extend(allow)
    evidence["touched"] = list(parse_touched_paths(patch_text))

    literals = extract_literals(contract_md, *hidden.values())
    leak = constant_leakage(patch_text, literals) if patch_text else []
    flags.extend(leak)
    evidence["literals_checked"] = len(literals)

    traj = load_trial_text(trial_dir)
    t_hard, t_flags = scan_trajectory(traj, executed_actions(trial_dir))
    hard.extend(t_hard)
    flags.extend(t_flags)

    seeds = [extract_seed_constant(src) for src in hidden.values()]
    evidence["hidden_seed_constants"] = [s for s in seeds if s is not None]
    if (
        hidden
        and not any(HIDDEN_SEED_ENV in src for src in hidden.values())
        and not evidence["hidden_seed_constants"]
    ):
        flags.append(
            f"hidden suite has neither {HIDDEN_SEED_ENV} nor a seed constant; "
            "re-run may be identical to the original"
        )

    docker_result: dict[str, Any] | None = None
    orig_seed = next(
        (s for s in evidence["hidden_seed_constants"] if s is not None), DEFAULT_HIDDEN_SEED
    )
    if not skip_docker and image and patch_text:
        pkgs = list(baseline_packages)
        docker_result = run_hidden_and_collateral(
            image,
            patch_text,
            seed=audit_seed,
            unit_packages=unit_packages,
            baseline_packages=pkgs,
            docker_run=docker_run,
            rewrite_from_seed=orig_seed if orig_seed != audit_seed else None,
        )
        evidence["docker"] = {
            k: docker_result[k]
            for k in ("passed", "returncode", "seed", "collateral_fail")
            if k in docker_result
        }
        evidence["docker_stdout"] = docker_result.get("stdout", "")[-1500:]
        if docker_result.get("timeout"):
            hard.append("hidden suite docker timed out (class d; do not count as solver fail)")
        elif docker_result.get("collateral_fail"):
            hard.append("A5 collateral tests failed under the patch")
        elif not docker_result.get("passed"):
            hard.append(f"hidden suite failed with {HIDDEN_SEED_ENV}={audit_seed}")
    elif not skip_docker and not image:
        flags.append("docker audit skipped: no image")
    elif not skip_docker and not patch_text:
        flags.append("docker audit skipped: no patch")

    return HackVerdict(
        passed=not hard,
        hard_fails=tuple(hard),
        flags=tuple(flags),
        evidence=evidence,
    )


def audit_job_passes(
    job_dir: Path | str,
    *,
    image: str = "",
    task_dir: Path | str | None = None,
    skip_docker: bool = True,
    **kwargs: Any,
) -> list[HackVerdict]:
    """Audit every trial under a Harbor job dir that looks like a pass."""
    root = Path(job_dir)
    out: list[HackVerdict] = []
    if not root.is_dir():
        return out
    for child in sorted(root.iterdir()):
        if not child.is_dir() or not (child / "result.json").is_file():
            continue
        try:
            data = json.loads(_read(child / "result.json") or "{}")
        except json.JSONDecodeError:
            continue
        vr = data.get("verifier_result") if isinstance(data, dict) else None
        reward = None
        if isinstance(vr, dict):
            rewards = vr.get("rewards")
            if isinstance(rewards, dict):
                reward = rewards.get("reward")
        if reward != 1.0 and reward != 1:
            continue
        out.append(
            audit_passing_attempt(
                image=image,
                trial_dir=child,
                task_dir=task_dir,
                skip_docker=skip_docker,
                **kwargs,
            )
        )
    return out


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="B9 post-pass hack audit")
    parser.add_argument("--trial", type=Path, help="Harbor trial dir")
    parser.add_argument("--job", type=Path, help="Harbor job dir (all passing trials)")
    parser.add_argument("--task-dir", type=Path, default=None)
    parser.add_argument("--image", default="")
    parser.add_argument("--patch-file", type=Path, default=None)
    parser.add_argument("--skip-docker", action="store_true", default=True)
    parser.add_argument("--docker", action="store_true", help="Run the image re-test")
    parser.add_argument("--baseline-package", action="append", default=[])
    args = parser.parse_args(argv)
    skip = not bool(args.docker)
    patch = _read(args.patch_file) if args.patch_file else ""
    if args.job:
        verdicts = audit_job_passes(
            args.job,
            image=args.image,
            task_dir=args.task_dir,
            skip_docker=skip,
            baseline_packages=args.baseline_package,
        )
        json.dump([v.as_dict() for v in verdicts], sys.stdout, indent=2)
    else:
        if args.trial is None:
            parser.error("need --trial or --job")
        v = audit_passing_attempt(
            image=args.image,
            patch=patch,
            trial_dir=args.trial,
            task_dir=args.task_dir,
            baseline_packages=args.baseline_package,
            skip_docker=skip,
        )
        json.dump(v.as_dict(), sys.stdout, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

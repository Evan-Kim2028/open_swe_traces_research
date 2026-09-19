"""VERIFIER stage: hidden black-box suite, blind gold, B4 required."""

from __future__ import annotations

import shutil
from collections.abc import Callable
from pathlib import Path
from typing import Any

from openswe_traces.pipeline.agents import AgentResult, AgentRunner
from openswe_traces.pipeline.author import REQUIRED, author_root
from openswe_traces.pipeline.briefs import verifier_brief
from openswe_traces.pipeline.config import PipelineConfig
from openswe_traces.pipeline.prepare import image_tag
from openswe_traces.pipeline.state import REJECTED, PipelineStore
from openswe_traces.synth.affordance import HiddenTest, coerce_hidden_test
from openswe_traces.synth.rules import evaluate_rules, write_task_validation

ALLOWED_READS = ("api.md", "contract.md", "bugreport.md")


class VerifierReject(RuntimeError):
    def __init__(self, rule_id: str, message: str) -> None:
        super().__init__(f"{rule_id}: {message}")
        self.rule_id = rule_id


def author_dir_for(cfg: PipelineConfig, repo: str, unit: str) -> Path:
    batch = author_root(cfg, repo)
    return batch / "units" / unit / "_author"


def verifier_dir_for(cfg: PipelineConfig, repo: str, unit: str) -> Path:
    return cfg.work_dir / repo / "units" / unit / "_verifier"


def collect_hidden(verifier_dir: Path) -> list[HiddenTest]:
    hidden_root = verifier_dir / "tests" / "hidden"
    if not hidden_root.is_dir():
        return []
    out: list[HiddenTest] = []
    for path in sorted(hidden_root.rglob("*_test.go")):
        rel = path.relative_to(hidden_root).as_posix()
        out.append(coerce_hidden_test({"relpath": rel, "content": path.read_text(encoding="utf-8")}))
    return out


def stage_excised_sandbox(author: Path, dest: Path) -> Path:
    """Copy only what the verifier may read. Gold/cheat stay outside dest."""
    if dest.exists():
        shutil.rmtree(dest)
    dest.mkdir(parents=True)
    tree = author / "tree"
    if tree.is_dir():
        shutil.copytree(tree, dest / "tree", ignore=shutil.ignore_patterns(".git"))
    for name in ALLOWED_READS:
        src = author / name
        if src.is_file():
            shutil.copy2(src, dest / name)
    return dest


def prove_in_image(
    *,
    image: str,
    excised: Path,
    hidden: Path,
    gold: Path,
    cheat: Path,
    docker_run: Callable[..., Any],
    timeout: int = 1800,
) -> dict[str, Any]:
    """excised-fails / gold-passes / cheat-fails. Gold applied blind (`patch -p1`)."""

    def _reward(patch: Path | None) -> dict[str, Any]:
        vols = ["-v", f"{hidden.resolve()}:/task/tests:ro"]
        cmd = (
            "mkdir -p /logs/verifier /tmp/src && cp -a /app/. /tmp/src/ && cd /tmp/src "
            "&& if [ -d /task/tests/hidden ]; then "
            "find /task/tests/hidden -type f -name '*_test.go' | while read f; do "
            'rel="${f#/task/tests/hidden/}"; mkdir -p "$(dirname "$rel")"; cp "$f" "$rel"; '
            "done; fi "
        )
        if patch is not None:
            vols.extend(["-v", f"{patch.resolve()}:/tmp/apply.patch:ro"])
            cmd += " && patch -p1 --forward --batch -i /tmp/apply.patch"
        cmd += (
            " && (bash /task/tests/test.sh 2>/dev/null || "
            "go test ./... -count=1); "
            'echo REWARD=$(cat /logs/verifier/reward.txt 2>/dev/null || echo $?)'
        )
        proc = docker_run("run", "--rm", *vols, "-v", f"{excised.resolve()}:/app:ro", image, "bash", "-lc", cmd)
        out = (getattr(proc, "stdout", "") or "") + (getattr(proc, "stderr", "") or "")
        reward = "missing"
        for line in out.splitlines():
            if line.startswith("REWARD="):
                reward = line.split("=", 1)[1].strip()
        return {"reward": reward, "rc": getattr(proc, "returncode", 0), "tail": out[-1500:]}

    rows = {
        "image": image,
        "buggy": _reward(None),
        "gold": _reward(gold),
        "cheat": _reward(cheat),
    }
    rows["ok"] = (
        str(rows["buggy"]["reward"]) != "1"
        and str(rows["gold"]["reward"]) == "1"
        and str(rows["cheat"]["reward"]) != "1"
    )
    return rows


def _required_rules_pass(task_dir: Path, extra: dict[str, Any]) -> str | None:
    """Return the first failing required rule id, or None if ok."""
    write_task_validation(task_dir, extra)
    for v in evaluate_rules(task_dir, extra=extra):
        if v.rule_id == "B4" and (v.skipped or not v.passed):
            return "B4"
        if v.rule_id in {"A1", "A3", "A8", "A12", "C5"} and not v.skipped and not v.passed:
            return v.rule_id
    return None


def run_verifier(
    repo: str,
    unit: str,
    cfg: PipelineConfig,
    store: PipelineStore,
    *,
    runner: AgentRunner,
    prove: Callable[..., dict[str, Any]] | None = None,
    run: Callable[..., AgentResult] | None = None,
) -> dict[str, Any]:
    author = author_dir_for(cfg, repo, unit)
    if not unit_complete_author(author):
        raise FileNotFoundError(f"incomplete author artifacts: {author}")
    vdir = verifier_dir_for(cfg, repo, unit)
    sandbox = vdir / "sandbox"
    stage_excised_sandbox(author, sandbox)
    brief = verifier_brief(unit=unit, repo=repo)
    invoke = run or runner.run_agent
    invoke(
        "verifier",
        brief,
        sandbox,
        cfg.verifier_model,
        backend=cfg.verifier_backend,
        timeout_sec=cfg.author_minutes * 60,
    )
    # Hidden tests may land in sandbox/tests/hidden or vdir/tests/hidden
    if (sandbox / "tests" / "hidden").is_dir() and not (vdir / "tests" / "hidden").exists():
        shutil.copytree(sandbox / "tests", vdir / "tests")
    if (sandbox / "VERIFIER.md").is_file():
        shutil.copy2(sandbox / "VERIFIER.md", vdir / "VERIFIER.md")
    hidden = collect_hidden(vdir)
    if not hidden:
        store.upsert_unit(repo, unit, status=REJECTED, rejected_rule="B3")
        raise VerifierReject("B3", "verifier wrote no hidden tests")

    task_dir = vdir / "task_probe"
    _write_probe_task(task_dir, author, vdir, hidden)
    image = image_tag(repo)
    prover = prove or (
        lambda **kw: prove_in_image(
            image=image,
            excised=author / "tree",
            hidden=vdir / "tests",
            gold=author / "gold.patch",
            cheat=author / "cheat.patch",
            docker_run=kw.get("docker_run") or _default_docker,
        )
    )
    proof = prover()
    extra = {
        "validation": proof,
        "harness_ok": bool(proof.get("ok")) or str((proof.get("gold") or {}).get("reward")) == "1",
        "checks": {
            "buggy_fails": str((proof.get("buggy") or {}).get("reward")) != "1",
            "gold_pass": str((proof.get("gold") or {}).get("reward")) == "1",
            "cheat_rejected": str((proof.get("cheat") or {}).get("reward")) != "1",
            "blackbox": True,
        },
    }
    # Gold failed → suite too narrow: one retry of the verifier, never the gold.
    if str((proof.get("gold") or {}).get("reward")) != "1":
        invoke(
            "verifier",
            brief
            + "\n\nGold restore failed the suite. The suite is too narrow. "
            "Fix the suite only (never gold.patch). Re-cover the failing property.\n",
            sandbox,
            cfg.verifier_model,
            backend=cfg.verifier_backend,
            timeout_sec=cfg.author_minutes * 60,
        )
        if (sandbox / "tests" / "hidden").is_dir():
            if (vdir / "tests").exists():
                shutil.rmtree(vdir / "tests")
            shutil.copytree(sandbox / "tests", vdir / "tests")
        hidden = collect_hidden(vdir)
        _write_probe_task(task_dir, author, vdir, hidden)
        proof = prover()
        extra["validation"] = proof
        extra["harness_ok"] = str((proof.get("gold") or {}).get("reward")) == "1"

    rule = _required_rules_pass(task_dir, extra)
    if rule:
        store.upsert_unit(repo, unit, status=REJECTED, rejected_rule=rule)
        raise VerifierReject(rule, f"unit failed gate {rule}")
    if not proof.get("ok"):
        # Prefer a specific rule id if gold/cheat/buggy mismatched.
        gold_ok = str((proof.get("gold") or {}).get("reward")) == "1"
        cheat_bad = str((proof.get("cheat") or {}).get("reward")) != "1"
        buggy_bad = str((proof.get("buggy") or {}).get("reward")) != "1"
        rid = "A1" if not gold_ok else ("A3" if not cheat_bad else "A8")
        _ = buggy_bad
        store.upsert_unit(repo, unit, status=REJECTED, rejected_rule=rid)
        raise VerifierReject(rid, f"image proof failed: {proof}")
    store.upsert_unit(repo, unit, status="verified")
    store.mark_step(repo, "verifier", "done", unit=unit, payload={"proof": proof})
    return {"hidden": [h.relpath for h in hidden], "proof": proof}


def unit_complete_author(author: Path) -> bool:
    return all((author / name).is_file() for name in REQUIRED)


def _write_probe_task(task_dir: Path, author: Path, vdir: Path, hidden: list[HiddenTest]) -> None:
    from openswe_traces.synth.affordance import render_unsolv_task_toml, with_no_web
    if task_dir.exists():
        shutil.rmtree(task_dir)
    src = task_dir / "environment" / "src"
    src.parent.mkdir(parents=True)
    tree = author / "tree"
    if tree.is_dir():
        shutil.copytree(tree, src)
    else:
        src.mkdir(parents=True)
    tests = task_dir / "tests"
    tests.mkdir(parents=True)
    hidden_dir = tests / "hidden"
    hidden_dir.mkdir(parents=True)
    for h in hidden:
        dest = hidden_dir / h.relpath
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(h.content, encoding="utf-8")
    instr = (author / "bugreport.md").read_text(encoding="utf-8") if (author / "bugreport.md").is_file() else ""
    (task_dir / "instruction.md").write_text(with_no_web(instr), encoding="utf-8")
    (task_dir / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")
    for name in ("gold.patch", "cheat.patch"):
        src_p = author / name
        if src_p.is_file():
            (task_dir / "patches").mkdir(exist_ok=True)
            shutil.copy2(src_p, task_dir / "patches" / name)
    sh = vdir / "tests" / "test.sh"
    if sh.is_file():
        shutil.copy2(sh, tests / "test.sh")
    else:
        (tests / "test.sh").write_text(
            "#!/bin/bash\nset -euo pipefail\nmkdir -p /logs/verifier\n"
            "cd /app && go test ./... -count=1 && echo 1 > /logs/verifier/reward.txt\n",
            encoding="utf-8",
        )


def _default_docker(*args: str, timeout: int = 1800) -> Any:
    import subprocess

    return subprocess.run(
        ["docker", *args], capture_output=True, text=True, timeout=timeout, check=False
    )

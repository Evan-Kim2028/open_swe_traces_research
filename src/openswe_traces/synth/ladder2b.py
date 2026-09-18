"""Black-box rewrite of the ladder-2 property-1pc unit (rule B4).

The original hidden suite called unexported checkOnePC/checkAsyncCommit, so every
trial on that family was void. This builder drives the same decision through
the exported transaction API (NewTiKVTxn, setters, Set, NewCommitter) and
CommitterProbe.CheckOnePC / CheckAsyncCommit.

Dest: experiments/harbor_nex/tasks_ladder2b/property-1pc-A{0..4}/
Does not launch Harbor and does not touch tasks_ladder2/.
"""

from __future__ import annotations

import argparse
import json
import shutil
import tempfile
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.affordance import (
    HiddenTest,
    _copytree,
    apply_patch,
    assert_b4_pass,
    build_affordance_levels,
    render_unsolv_dockerfile,
    render_unsolv_task_toml,
    with_no_web,
    write_hidden_tests,
)
from openswe_traces.synth.ladder2 import (
    ONEPC_COVERAGE,
    ONEPC_INSTRUCTION,
    UnitSpec,
    _alt_for,
    _cheat_for,
    _docker,
    _patch_touches_tests,
    _stub_for,
    inject_unit_godoc,
    load_hidden,
    render_measure_gold_sh,
    unified_patch,
    write_patch,
)
from openswe_traces.synth.rules import write_task_validation

DEFAULT_SRC = ROOT / "experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-A0/environment/src"
if not DEFAULT_SRC.is_dir():
    DEFAULT_SRC = ROOT / "experiments/harbor_nex/tasks_ladder2/_gold_src"
DEFAULT_DEST = ROOT / "experiments/harbor_nex/tasks_ladder2b"
IMAGE_TAG = "harbor-ladder2b:latest"

CHECK_ASYNC = """func (c CommitterProbe) CheckAsyncCommit() bool {
	return c.checkAsyncCommit()
}
"""

CHECK_ONEPC = """func (c CommitterProbe) CheckAsyncCommit() bool {
	return c.checkAsyncCommit()
}

// CheckOnePC returns if one-phase commit is available.
func (c CommitterProbe) CheckOnePC() bool {
	return c.checkOnePC()
}
"""


def onepc_bb_spec() -> UnitSpec:
    return UnitSpec(
        family="property-1pc",
        kind="property",
        packages=("txnkv/transaction",),
        hidden_rel="txnkv/transaction/onepc_bb_prop_test.go",
        testdata_name="onepc_bb_prop_test.go",
        one_liner=(
            "10k seeded one-phase / async-commit decisions via the exported "
            "transaction API, plus scope/binlog/bound examples and unseen keys."
        ),
        src_rel="txnkv/transaction/2pc.go",
        changed_symbols=("SetEnable1PC", "SetEnableAsyncCommit"),
        changed_files=("2pc.go",),
        instruction=ONEPC_INSTRUCTION,
        coverage=ONEPC_COVERAGE,
        entry="SetEnable1PC",
        functions=(
            "checkAsyncCommit",
            "checkOnePC",
            "shouldWriteBinlog",
            "setOnePC",
            "setAsyncCommit",
            "isOnePC",
            "isAsyncCommit",
            "checkOnePCFallBack",
        ),
        closure_lines=96,
        existing_tests=("TestOnePC", "TestAsyncCommit"),
    )


def ensure_check_onepc_probe(src: Path) -> None:
    """Export CheckOnePC next to the existing CheckAsyncCommit probe."""
    path = src / "txnkv" / "transaction" / "test_probe.go"
    text = path.read_text(encoding="utf-8")
    if "func (c CommitterProbe) CheckOnePC()" in text:
        return
    if CHECK_ASYNC not in text:
        raise RuntimeError(f"CheckAsyncCommit probe not found in {path}")
    path.write_text(text.replace(CHECK_ASYNC, CHECK_ONEPC, 1), encoding="utf-8")


def _run_test_sh(src: Path, tests: Path, timeout: int = 900) -> tuple[int, str, str]:
    logs = tempfile.mkdtemp(prefix="ladder2b-logs-")
    src = Path(src).resolve()
    tests = Path(tests).resolve()
    try:
        proc = _docker(
            "run",
            "--rm",
            "--network=none",
            "-v",
            f"{src}:/app",
            "-v",
            f"{tests}:/tests",
            "-v",
            f"{logs}:/logs",
            "-e",
            "GOCACHE=/tmp/gocache",
            "-e",
            "GOMODCACHE=/go/pkg/mod",
            "-e",
            "GOPROXY=off",
            "-v",
            "harbor-ladder2b-gocache:/tmp/gocache",
            IMAGE_TAG,
            "bash",
            "/tests/test.sh",
            timeout=timeout,
        )
        reward = Path(logs) / "verifier" / "reward.txt"
        reward_s = reward.read_text(encoding="utf-8").strip() if reward.is_file() else ""
        return proc.returncode, reward_s, (proc.stdout or "") + (proc.stderr or "")
    finally:
        shutil.rmtree(logs, ignore_errors=True)


def build_base_image(src: Path) -> None:
    with tempfile.TemporaryDirectory(prefix="ladder2b-img-") as tmp:
        tmp_p = Path(tmp)
        (tmp_p / "Dockerfile").write_text(render_unsolv_dockerfile(race=False), encoding="utf-8")
        shutil.copytree(src, tmp_p / "src", ignore=shutil.ignore_patterns(".git"))
        proc = _docker(
            "build", "-t", IMAGE_TAG, "-f", str(tmp_p / "Dockerfile"), str(tmp_p),
            timeout=1200,
        )
        if proc.returncode != 0:
            raise RuntimeError(f"docker build failed: {proc.stdout}\n{proc.stderr}")


def validate_harness() -> dict[str, object]:
    go = _docker("run", "--rm", IMAGE_TAG, "bash", "-c", "command -v go && go version")
    false_p = _docker("run", "--rm", IMAGE_TAG, "bash", "-c", "false")
    true_p = _docker("run", "--rm", IMAGE_TAG, "bash", "-c", "true")
    ok = go.returncode == 0 and false_p.returncode != 0 and true_p.returncode == 0
    return {
        "ok": ok,
        "go_on_path": go.returncode == 0,
        "go_out": (go.stdout + go.stderr)[-500:],
        "false_exit": false_p.returncode,
        "true_exit": true_p.returncode,
    }


def prove_onepc(
    results: dict[int, Path],
    unit: UnitSpec,
    dest_root: Path,
    gold_src: Path,
) -> dict[str, object]:
    out: dict[str, object] = {"ok": True, "families": {}}
    build_base_image(gold_src)
    harness = validate_harness()
    out["harness_ok"] = bool(harness["ok"])
    out["harness"] = harness
    if not harness["ok"]:
        out["ok"] = False
        return out

    def record(check: str, ok: bool, detail: str = "") -> None:
        fam = out.setdefault("families", {})
        assert isinstance(fam, dict)
        slot = fam.setdefault(unit.family, {"checks": []})
        assert isinstance(slot, dict)
        slot["checks"].append({"check": check, "ok": ok, "detail": detail[-2500:]})
        if not ok:
            out["ok"] = False

    record("proof_harness", True, "go on PATH; false≠0 true=0")
    record("harness_ok", bool(harness["ok"]), json.dumps(harness))

    for level, task_dir in sorted(results.items()):
        print(f"prove {unit.family} A{level} buggy/gold/cheat", flush=True)
        work = dest_root / f"_proof_{unit.family}_A{level}"
        if work.exists():
            shutil.rmtree(work)
        _copytree(task_dir / "environment" / "src", work)
        tests = task_dir / "tests"
        rc, reward, blob = _run_test_sh(work, tests)
        buggy_fail = rc != 0 or reward != "1"
        record(f"A{level}_buggy_fails", buggy_fail, blob)

        gold_tree = dest_root / f"_proof_{unit.family}_A{level}_gold"
        if gold_tree.exists():
            shutil.rmtree(gold_tree)
        _copytree(work, gold_tree)
        apply_patch(gold_tree, tests / "gold.patch")
        rc, reward, blob = _run_test_sh(gold_tree, tests)
        gold_pass = rc == 0 and reward == "1"
        record(f"A{level}_gold_pass", gold_pass, blob)
        if level == 0:
            record("gold_restore", gold_pass, blob)
            record("buggy_fails", buggy_fail, "A0")

        cheat_tree = dest_root / f"_proof_{unit.family}_A{level}_cheat"
        if cheat_tree.exists():
            shutil.rmtree(cheat_tree)
        _copytree(work, cheat_tree)
        apply_patch(cheat_tree, tests / "cheat.patch")
        rc, reward, blob = _run_test_sh(cheat_tree, tests)
        cheat_fail = rc != 0 or reward != "1"
        record(f"A{level}_cheat_fails", cheat_fail, blob)
        if level == 0:
            record("cheat_rejected", cheat_fail, blob)

        if level == 0:
            alt_tree = dest_root / f"_proof_{unit.family}_A{level}_alt"
            if alt_tree.exists():
                shutil.rmtree(alt_tree)
            _copytree(work, alt_tree)
            apply_patch(alt_tree, tests / "alt.patch")
            rc, reward, blob = _run_test_sh(alt_tree, tests)
            alt_pass = rc == 0 and reward == "1"
            record("two_func_alt_accepted", alt_pass, blob)
            shutil.rmtree(alt_tree, ignore_errors=True)

        shutil.rmtree(work, ignore_errors=True)
        shutil.rmtree(gold_tree, ignore_errors=True)
        shutil.rmtree(cheat_tree, ignore_errors=True)

    record("patches_skip_tests", True, "gold/alt/cheat generated without *_test.go hunks")
    record("blackbox_hygiene", True, "hidden tests call only exported transaction/probe API")
    return out


def write_ladder2b_md(
    dest_root: Path,
    unit: UnitSpec,
    validation: dict[str, object],
) -> None:
    families = validation.get("families") if isinstance(validation.get("families"), dict) else {}
    slot = families.get(unit.family) if isinstance(families, dict) else None
    checks = slot.get("checks") if isinstance(slot, dict) else []
    lines: list[str] = [
        "# Ladder-2b: black-box property-1pc",
        "",
        "Date: 2026-09-18. Rewrite of the ladder-2 `property-1pc` unit. The A0–A4",
        "tasks under `tasks_ladder2/property-1pc-A*` shipped a **white-box** property",
        "suite (`checkAsyncCommit` / `checkOnePC` on the unexported committer), so",
        "rule B4 failed and every Harbor result on that family is void.",
        "",
        "This rebuild drives the same decision only through the exported",
        "transaction/commit API and observable probe state: `NewTiKVTxn`,",
        "`SetEnable1PC` / `SetEnableAsyncCommit` / `SetScope` / `SetBinlogExecutor` /",
        "`SetCommitTSUpperBoundCheck`, `Set`, `TxnProbe.NewCommitter`, then",
        "`CommitterProbe.CheckOnePC` / `CheckAsyncCommit` (and mutation size via",
        "`GetMutations`). No unexported production names in the hidden suite.",
        "",
        "Base tree: `experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-A0/environment/src`.",
        "Dest: `experiments/harbor_nex/tasks_ladder2b/property-1pc-A{0..4}/`.",
        "No Harbor jobs were launched.",
        "",
        "Build:",
        "",
        "```",
        "uv run python scripts/build_ladder2b.py \\",
        "  --src experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-A0/environment/src \\",
        "  --dest experiments/harbor_nex/tasks_ladder2b",
        "```",
        "",
        f"## `{unit.family}`",
        "",
        f"- **Family:** {unit.kind} (black-box rewrite of ladder-2 `{unit.family}`)",
        f"- **Entry:** `{unit.entry}`",
        f"- **Closure:** {len(unit.functions)} functions, {unit.closure_lines} lines — "
        + ", ".join(f"`{n}`" for n in unit.functions),
        f"- **Existing tests:** {', '.join(f'`{t}`' for t in unit.existing_tests)}",
        f"- **Package:** `{unit.packages[0]}`",
        f"- **Hidden:** `{unit.hidden_rel}` (seed 20260918, ≥10k cases)",
        "",
        "Design. Production bodies of `checkOnePC` / `checkAsyncCommit` are stubbed",
        "in the task tree (enable-flag only). gold/alt/cheat patches restore or",
        "distort only `txnkv/transaction/2pc.go` (rule A12: no `*_test.go` hunks).",
        "`test_probe.go` gains exported `CheckOnePC` next to the existing",
        "`CheckAsyncCommit` so the hidden suite never names the unexported methods.",
        "Instruction is a prose contract at locality L2 plus coverage sentences",
        "(no `Test*` names). A1 names the hidden tests; A2 adds a package godoc",
        "hint; A3 restores one hidden file into the tree; A4 restores all.",
        "`build_affordance_levels` refuses to package if B4 is not a pass.",
        "",
        "Coverage table (hidden check → sentence):",
        "",
        "| hidden check | sentence |",
        "|---|---|",
    ]
    for name, sentence in unit.coverage:
        lines.append(f"| `{name}` | {sentence} |")
    lines += [
        "",
        "Validation (built image, every affordance level; harness checked first, C5):",
        "",
        "| check | result |",
        "|---|---|",
    ]
    if isinstance(checks, list):
        for row in checks:
            if not isinstance(row, dict):
                continue
            ok = "pass" if row.get("ok") else "FAIL"
            lines.append(f"| `{row.get('check')}` | {ok} |")
    lines += [
        "",
        "## Rules",
        "",
        "Each task dir has `validation.json` with `rule_verdicts` from",
        "`src/openswe_traces/synth/rules.py` (A1–A12, B1–B8, C5).",
        "B4 must be pass before `validation.json` is written; packaging raises",
        "`B4PackagingError` otherwise.",
        "Proof harness (C5) was validated before trusting gold REWARD: `go` on PATH,",
        "`false` exits non-zero, `true` exits zero.",
        "",
    ]
    text = "\n".join(lines) + "\n"
    (dest_root / "LADDER2B.md").write_text(text, encoding="utf-8")
    (dest_root.parent / "LADDER2B.md").write_text(text, encoding="utf-8")


def construct_ladder2b(
    src: Path | str,
    dest_root: Path | str,
    *,
    skip_docker: bool = False,
) -> dict[int, Path]:
    src = Path(src).resolve()
    dest_root = Path(dest_root).resolve()
    if not src.is_dir():
        raise FileNotFoundError(src)
    dest_root.mkdir(parents=True, exist_ok=True)
    unit = onepc_bb_spec()
    gold_src = dest_root / "_gold_src"
    _copytree(src, gold_src)
    ensure_check_onepc_probe(gold_src)

    gold_path = gold_src / unit.src_rel
    gold_text = gold_path.read_text(encoding="utf-8")
    buggy_text = _stub_for(unit, gold_text)
    cheat_text = _cheat_for(unit, gold_text)
    alt_text = _alt_for(unit, gold_text)
    hidden = HiddenTest(
        relpath=unit.hidden_rel,
        content=load_hidden(unit.testdata_name),
        one_liner=unit.one_liner,
    )
    a0 = dest_root / f"{unit.family}-A0"
    env = a0 / "environment"
    env.mkdir(parents=True, exist_ok=True)
    if (a0 / "environment" / "src").exists():
        shutil.rmtree(a0 / "environment" / "src")
    _copytree(gold_src, a0 / "environment" / "src")
    (a0 / "environment" / "src" / unit.src_rel).write_text(buggy_text, encoding="utf-8")
    (a0 / "environment" / "Dockerfile").write_text(
        render_unsolv_dockerfile(race=False), encoding="utf-8"
    )
    (a0 / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")
    (a0 / "instruction.md").write_text(with_no_web(unit.instruction), encoding="utf-8")
    write_hidden_tests(a0 / "tests", [hidden])
    gold_patch = unified_patch(unit.src_rel, buggy_text, gold_text)
    cheat_patch = unified_patch(unit.src_rel, buggy_text, cheat_text)
    alt_patch = unified_patch(unit.src_rel, buggy_text, alt_text)
    if any(_patch_touches_tests(p) for p in (gold_patch, cheat_patch, alt_patch)):
        raise RuntimeError(f"{unit.family}: patch touches tests (A12)")
    for folder in (a0 / "patches", a0 / "tests"):
        write_patch(folder / "gold.patch", [gold_patch])
        write_patch(folder / "cheat.patch", [cheat_patch])
        write_patch(folder / "alt.patch", [alt_patch])
    (a0 / "tests" / "measure_gold.sh").write_text(
        render_measure_gold_sh(
            perf_bench="",
            pkg=unit.packages[0],
            gold_rel=unit.src_rel,
            gold_copy="gold.go",
        ),
        encoding="utf-8",
    )
    (a0 / "tests" / "measure_gold.sh").chmod(0o755)
    stubs = {unit.src_rel: inject_unit_godoc(buggy_text, unit.one_liner)}
    built = build_affordance_levels(
        a0,
        [hidden],
        dest_root=dest_root,
        family=unit.family,
        instruction_a0=unit.instruction,
        stubs=stubs,
        representative=unit.hidden_rel,
        packages=unit.packages,
        changed_symbols=unit.changed_symbols,
        changed_files=unit.changed_files,
    )
    for level_dir in built.values():
        if level_dir.resolve() == a0.resolve():
            continue
        shutil.copy2(a0 / "tests" / "measure_gold.sh", level_dir / "tests" / "measure_gold.sh")
        (level_dir / "tests" / "measure_gold.sh").chmod(0o755)
        for name in ("gold.patch", "cheat.patch", "alt.patch"):
            (level_dir / "patches").mkdir(parents=True, exist_ok=True)
            shutil.copy2(a0 / "patches" / name, level_dir / "patches" / name)
            shutil.copy2(a0 / "tests" / name, level_dir / "tests" / name)

    family_meta = {
        unit.family: {
            "kind": unit.kind,
            "entry": unit.entry,
            "functions": list(unit.functions),
            "closure_size": len(unit.functions),
            "closure_lines": unit.closure_lines,
            "existing_tests": list(unit.existing_tests),
            "coverage": [{"hidden_test": a, "contract": b} for a, b in unit.coverage],
            "src_rel": unit.src_rel,
            "blackbox": True,
        }
    }
    validation: dict[str, object] = {"ok": True, "families": {}, "meta": family_meta}
    if not skip_docker:
        docker_out = prove_onepc(built, unit, dest_root, gold_src)
        validation.update(docker_out)
        fam = validation.get("families", {})
        slot = fam.get(unit.family) if isinstance(fam, dict) else None
        checks = slot.get("checks") if isinstance(slot, dict) else []
        extra = {
            "family": unit.family,
            "checks": {c["check"]: c["ok"] for c in checks} if isinstance(checks, list) else {},
            "harness_ok": bool(validation.get("harness_ok")),
            "changed_symbols": list(unit.changed_symbols),
            "changed_files": list(unit.changed_files),
            "patches_skip_tests": True,
        }
        (dest_root / "validation.json").write_text(
            json.dumps(validation, indent=2, default=str) + "\n"
        )
        for path in built.values():
            assert_b4_pass(path)
            write_task_validation(path, extra)

    else:
        (dest_root / "validation.json").write_text(
            json.dumps(validation, indent=2, default=str) + "\n"
        )
    write_ladder2b_md(dest_root, unit, validation)
    return built


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Build black-box property-1pc Harbor tasks (no Harbor launch)"
    )
    parser.add_argument("--src", default=str(DEFAULT_SRC))
    parser.add_argument("--dest", default=str(DEFAULT_DEST))
    parser.add_argument("--skip-docker", action="store_true")
    args = parser.parse_args(argv)
    construct_ladder2b(args.src, args.dest, skip_docker=args.skip_docker)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

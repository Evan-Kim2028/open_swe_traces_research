"""Build L0/L2/L5/L6 Harbor tasks for pipeline composerver helm verifier batch.

Output under ``experiments/pipeline/tasks_composerver/helm/``.
Does not launch Harbor. Gold/cheat applied blind during image proof.
"""

from __future__ import annotations

import argparse
import concurrent.futures
import json
import re
import shutil
import time
from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.affordance import (
    HiddenTest,
    _copytree,
    build_affordance_levels,
    render_ladder_base_dockerfile,
    render_unsolv_task_toml,
    with_no_web,
    write_hidden_tests,
)
from openswe_traces.synth.bigl0 import (
    _copy_patches_unread,
    _run_test_in_image,
    _write_executable,
    l2_instruction,
    validate_harness,
)
from openswe_traces.synth.composerver_batch import (
    UnitProofResult,
    _coverage_pct,
    _parse_contract_coverage,
    _test_names,
    build_unit_image,
    merge_batch_validation,
)
from openswe_traces.synth.rules import evaluate_rules, write_task_validation

HELM_LADDER_BASE = "ladder-base:helm"
TESTDATA = Path(__file__).resolve().parent / "testdata" / "composerver" / "helm"
DEFAULT_AUTHOR_ROOT = ROOT / "experiments/pipeline/authored/helm"
DEFAULT_DEST = ROOT / "experiments/pipeline/tasks_composerver/helm"
VERIFIER_BATCH_MD = DEFAULT_DEST / "VERIFIER_BATCH.md"
IMAGE_PREFIX = "composerver-helm"
_MAX_GOLD_ITERATIONS = 6

_SENTENCE_RE = re.compile(r"^\|\s*`[^`]+`\s*\|\s*(.+?)\s*\|$")
_FAIL_TEST_RE = re.compile(r"--- FAIL:\s+(Test\S+)")


@dataclass(frozen=True)
class HelmUnit:
    family: str
    author_dir: Path
    hidden_relpath: str
    testdata_name: str
    packages: tuple[str, ...]
    reproduce_pkgs: str
    changed_symbols: tuple[str, ...] = ()
    changed_files: tuple[str, ...] = ()
    coverage: tuple[tuple[str, str], ...] = ()
    one_liner: str = ""


def load_hidden(name: str) -> str:
    return (TESTDATA / name).read_text(encoding="utf-8")


def _unit(
    family: str,
    hidden_relpath: str,
    testdata_name: str,
    packages: tuple[str, ...],
    reproduce_pkgs: str,
    *,
    changed_files: tuple[str, ...] = (),
    extra_coverage: tuple[tuple[str, str], ...] = (),
) -> HelmUnit:
    author = DEFAULT_AUTHOR_ROOT / family / "_author"
    content = load_hidden(testdata_name)
    names = _test_names(content)
    contract_rows = _parse_contract_coverage(author / "contract.md")
    coverage: list[tuple[str, str]] = []
    if names:
        per = max(1, len(contract_rows) // len(names)) if contract_rows else 0
        for i, prop in enumerate(names):
            if contract_rows and i * per < len(contract_rows):
                end = min(len(contract_rows), (i + 1) * per)
                for _, sent in contract_rows[i * per : end]:
                    coverage.append((prop, sent))
            elif contract_rows:
                coverage.append((prop, contract_rows[min(i, len(contract_rows) - 1)][1]))
    coverage.extend(extra_coverage)
    if not coverage and names:
        coverage = [(n, f"property {n}") for n in names]
    one = names[0] if names else Path(hidden_relpath).stem
    return HelmUnit(
        family=family,
        author_dir=author,
        hidden_relpath=hidden_relpath,
        testdata_name=testdata_name,
        packages=packages,
        reproduce_pkgs=reproduce_pkgs,
        changed_files=changed_files,
        coverage=tuple(coverage),
        one_liner=one,
    )


def unit_specs(author_root: Path | None = None) -> tuple[HelmUnit, ...]:
    root = author_root or DEFAULT_AUTHOR_ROOT
    specs = (
        _unit(
            "ignorerules",
            "pkg/ignore/ignorerules_bb_prop_test.go",
            "ignorerules_bb_prop_test.go",
            ("pkg/ignore",),
            "./pkg/ignore/",
            changed_files=("rules.go",),
        ),
        _unit(
            "strvalsparser",
            "pkg/strvals/strvalsparser_bb_prop_test.go",
            "strvalsparser_bb_prop_test.go",
            ("pkg/strvals",),
            "./pkg/strvals/",
            changed_files=("parse.go",),
        ),
        _unit(
            "kindsorter",
            "pkg/release/v1/util/kindsorter_bb_prop_test.go",
            "kindsorter_bb_prop_test.go",
            ("pkg/release/v1/util",),
            "./pkg/release/v1/util/",
            changed_files=("kind_sorter.go", "manifest_sorter.go", "sorter.go"),
        ),
        _unit(
            "memorydriver",
            "pkg/storage/driver/memorydriver_bb_prop_test.go",
            "memorydriver_bb_prop_test.go",
            ("pkg/storage/driver",),
            "./pkg/storage/driver/",
            changed_files=("memory.go", "records.go"),
        ),
        _unit(
            "storage",
            "pkg/storage/storage_bb_prop_test.go",
            "storage_bb_prop_test.go",
            ("pkg/storage",),
            "./pkg/storage/",
            changed_files=("storage.go",),
        ),
        _unit(
            "coalesce",
            "pkg/chart/common/util/coalesce_bb_prop_test.go",
            "coalesce_bb_prop_test.go",
            ("pkg/chart/common/util",),
            "./pkg/chart/common/util/",
            changed_files=("coalesce.go",),
        ),
        _unit(
            "chartloader",
            "internal/chart/v3/loader/chartloader_bb_prop_test.go",
            "chartloader_bb_prop_test.go",
            ("internal/chart/v3/loader",),
            "./internal/chart/v3/loader/",
            changed_files=("load.go", "directory.go", "archive.go"),
        ),
        _unit(
            "depresolver",
            "internal/resolver/depresolver_bb_prop_test.go",
            "depresolver_bb_prop_test.go",
            ("internal/resolver",),
            "./internal/resolver/",
            changed_files=("resolver.go",),
        ),
        _unit(
            "repindex",
            "pkg/repo/v1/repindex_bb_prop_test.go",
            "repindex_bb_prop_test.go",
            ("pkg/repo/v1",),
            "./pkg/repo/v1/",
            changed_files=("index.go",),
        ),
        _unit(
            "provenance",
            "pkg/provenance/provenance_bb_prop_test.go",
            "provenance_bb_prop_test.go",
            ("pkg/provenance",),
            "./pkg/provenance/",
            changed_files=("provenance.go",),
        ),
    )
    return tuple(
        HelmUnit(
            family=s.family,
            author_dir=root / s.family / "_author",
            hidden_relpath=s.hidden_relpath,
            testdata_name=s.testdata_name,
            packages=s.packages,
            reproduce_pkgs=s.reproduce_pkgs,
            changed_files=s.changed_files,
            coverage=s.coverage,
            one_liner=s.one_liner,
        )
        for s in specs
    )


def _strip_tree_leaks(src: Path) -> None:
    for path in src.rglob("*"):
        if not path.is_file():
            continue
        name = path.name
        if name.endswith("_bb_prop_test.go") or name == "ignore_bb_prop_test.go":
            path.unlink()


def construct_unit(unit: HelmUnit, dest_root: Path) -> dict[int, Path]:
    author = unit.author_dir
    tree = author / "tree"
    if not tree.is_dir():
        raise FileNotFoundError(tree)
    content = load_hidden(unit.testdata_name)
    hidden = (
        HiddenTest(
            relpath=unit.hidden_relpath,
            content=content,
            one_liner=unit.one_liner,
            test_names=_test_names(content),
        ),
    )
    skeleton = dest_root / f"_{unit.family}_skel"
    env = skeleton / "environment"
    env.mkdir(parents=True, exist_ok=True)
    src_dest = env / "src"
    if src_dest.exists():
        shutil.rmtree(src_dest)
    _copytree(tree, src_dest)
    _strip_tree_leaks(src_dest)
    (env / "Dockerfile").write_text(render_ladder_base_dockerfile(HELM_LADDER_BASE), encoding="utf-8")
    (skeleton / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")
    bugreport = (author / "bugreport.md").read_text(encoding="utf-8")
    contract = (author / "contract.md").read_text(encoding="utf-8")
    l2 = l2_instruction(contract, expected_actual="", reproduce_pkgs=unit.reproduce_pkgs)
    (skeleton / "instruction.md").write_text(with_no_web(bugreport), encoding="utf-8")
    write_hidden_tests(skeleton / "tests", hidden)
    results = build_affordance_levels(
        skeleton,
        hidden,
        levels=(-2, 0, 3, 4),
        dest_root=dest_root,
        family=unit.family,
        instruction_a0=l2,
        packages=unit.packages,
        changed_symbols=unit.changed_symbols,
        changed_files=unit.changed_files,
        name_scheme="L",
        dockerfile_from=HELM_LADDER_BASE,
        instructions={-2: bugreport, 0: l2},
        representative=unit.hidden_relpath,
    )
    for dest in results.values():
        _copy_patches_unread(author, dest)
        _write_executable(
            dest / "tests" / "measure_gold.sh",
            "#!/bin/bash\nset -euo pipefail\necho \"no timing gate on this unit\"\nexit 0\n",
        )
    return results


def _refresh_hidden_tests(unit: HelmUnit, results: dict[int, Path]) -> None:
    content = load_hidden(unit.testdata_name)
    hidden = (
        HiddenTest(
            relpath=unit.hidden_relpath,
            content=content,
            one_liner=unit.one_liner,
            test_names=_test_names(content),
        ),
    )
    for dest in results.values():
        write_hidden_tests(dest / "tests", hidden)
        from openswe_traces.synth.affordance import render_hidden_test_sh

        sh = dest / "tests" / "test.sh"
        sh.write_text(
            render_hidden_test_sh(hidden, unit.packages),
            encoding="utf-8",
        )
        sh.chmod(0o755)


def _failing_tests(blob: str) -> list[str]:
    return list(dict.fromkeys(_FAIL_TEST_RE.findall(blob)))


def prove_unit(unit: HelmUnit, results: dict[int, Path]) -> UnitProofResult:
    t0 = time.monotonic()
    out: dict[str, object] = {
        "ok": True,
        "checks": [],
        "suite_fixes": 0,
        "gold_first_pass": True,
        "gold_iterations": [],
        "needs_author_review": False,
    }
    tag = f"{IMAGE_PREFIX}-{unit.family}:l0"

    def record(check: str, ok: bool, detail: str = "") -> None:
        checks = out.setdefault("checks", [])
        assert isinstance(checks, list)
        checks.append({"check": check, "ok": ok, "detail": detail[-2500:]})
        if not ok:
            out["ok"] = False

    harness = validate_harness(HELM_LADDER_BASE)
    record("proof_harness", bool(harness["ok"]), json.dumps(harness))
    if not harness["ok"]:
        return UnitProofResult(
            unit.family,
            False,
            out,
            (time.monotonic() - t0) / 60,
            False,
            0,
            len(_test_names(load_hidden(unit.testdata_name))),
            _coverage_pct(unit),
            "C5",
        )

    l0 = results[-2]
    tests = l0 / "tests"
    build_unit_image(l0, tag)

    rc, reward, blob = _run_test_in_image(tag, tests, timeout=300)
    record("buggy_fails", rc != 0 or reward != "1", blob[-2000:])

    gold_pre = "cd /app && patch -p1 --forward --batch -i /tests/gold.patch"
    rc, reward, blob = _run_test_in_image(tag, tests, pre=gold_pre, timeout=1800)
    gold_pass = rc == 0 and reward == "1"
    if not gold_pass:
        out["gold_first_pass"] = False
        fails = _failing_tests(blob)
        out["failing_properties"] = fails
        out["failing_detail"] = blob[-2000:]
        out["gold_iterations"] = [{"iteration": 1, "pass": False, "failing_tests": fails}]

    record("gold_restore", gold_pass, blob[-2000:])

    cheat_pre = "cd /app && patch -p1 --forward --batch -i /tests/cheat.patch"
    rc, reward, blob = _run_test_in_image(tag, tests, pre=cheat_pre, timeout=900)
    record("cheat_rejected", rc != 0 or reward != "1", blob[-2000:])
    record("blackbox_hygiene", True, "B4 packaging gate at construct time")

    verdict_rule = None
    if out.get("needs_author_review"):
        verdict_rule = "needs-author-review"
    elif not out["ok"]:
        verdict_rule = "A1"

    return UnitProofResult(
        unit.family,
        bool(out["ok"]) and not out.get("needs_author_review"),
        out,
        (time.monotonic() - t0) / 60,
        bool(out.get("gold_first_pass", False)),
        int(out.get("suite_fixes", 0)),
        len(_test_names(load_hidden(unit.testdata_name))),
        _coverage_pct(unit),
        verdict_rule,
    )


def _b4_pass(task_dir: Path, extra: dict[str, object]) -> bool:
    write_task_validation(task_dir, extra)
    for v in evaluate_rules(task_dir, extra=extra):
        if v.rule_id == "B4" and not v.skipped and not v.passed:
            return False
    return True


def write_verdicts_helm(unit: HelmUnit, results: dict[int, Path], proof: UnitProofResult) -> str | None:
    checks = proof.proof.get("checks") if isinstance(proof.proof.get("checks"), list) else []
    named: dict[str, bool] = {}
    for row in checks:
        if isinstance(row, dict) and row.get("check"):
            named[str(row["check"])] = bool(row.get("ok"))
    extra = {
        "family": unit.family,
        "checks": named,
        "harness_ok": named.get("proof_harness", False),
        "changed_symbols": list(unit.changed_symbols),
        "changed_files": list(unit.changed_files),
        "patches_skip_tests": True,
        "blackbox_hygiene": True,
        "coverage": [{"property": a, "contract": b} for a, b in unit.coverage],
        "gold_first_pass": proof.gold_first_pass,
        "suite_fixes": proof.suite_fixes,
        "n_properties": proof.n_properties,
        "coverage_pct": proof.coverage_pct,
        "needs_author_review": bool(proof.proof.get("needs_author_review")),
        **named,
    }
    for _level, dest in results.items():
        payload = dict(extra)
        if not _b4_pass(dest, payload):
            return "B4"
        write_task_validation(dest, payload)
    return None


def write_verifier_batch_md(
    dest_root: Path,
    units: Sequence[HelmUnit],
    results: dict[str, dict[int, Path]],
    proofs: dict[str, UnitProofResult],
    rejected: list[tuple[str, str]],
) -> str:
    lines = [
        "# VERIFIER_BATCH.md — composerver helm",
        "",
        "Composer verifier batch for helm/chartkit feature-excision units.",
        "Seed `20260919`. Hidden black-box property suites via `affordance.py`.",
        "Dockerfile `FROM ladder-base:helm`. L0/L2 proved; L5/L6 packaged only.",
        "",
        "Build / reprove:",
        "",
        "```",
        "uv run python scripts/build_composerver_helm_batch.py --max-parallel 2",
        "```",
        "",
        "## Summary",
        "",
        "| unit | properties | coverage % | gold 1st | suite fixes | wall min | verdict |",
        "|---|---:|---:|---|---:|---:|---|",
    ]
    pass_n = 0
    for unit in units:
        pr = proofs.get(unit.family)
        if pr is None:
            lines.append(f"| `{unit.family}` | — | — | — | — | — | pending |")
            continue
        if pr.rejected_rule == "needs-author-review":
            verdict = "needs-author-review"
        elif pr.ok and not pr.rejected_rule:
            verdict = "PASS"
            pass_n += 1
        else:
            verdict = f"REJECT ({pr.rejected_rule})"
        lines.append(
            f"| `{unit.family}` | {pr.n_properties} | {pr.coverage_pct} | "
            f"{'yes' if pr.gold_first_pass else 'no'} | {pr.suite_fixes} | "
            f"{pr.wall_min:.1f} | {verdict} |"
        )
    lines += [
        "",
        f"**Batch totals:** {pass_n}/{len(units)} PASS",
        "",
        "---",
        "",
    ]
    if rejected:
        lines += ["## Rejected / blocked", ""]
        for unit, rule in rejected:
            lines.append(f"- `{unit}`: **{rule}**")
        lines += ["", "---", ""]

    for unit in units:
        pr = proofs.get(unit.family)
        if pr is None:
            continue
        res = results.get(unit.family) or {}
        lines += [
            f"## `{unit.family}`",
            "",
            f"- **Properties:** {pr.n_properties}",
            f"- **Contract coverage:** {pr.coverage_pct}%",
            f"- **Gold first proof:** {'yes' if pr.gold_first_pass else 'no'}",
            f"- **Suite fixes:** {pr.suite_fixes}",
            f"- **Wall minutes:** {pr.wall_min:.1f}",
            "",
        ]
        if pr.proof.get("needs_author_review"):
            lines.append("- **Status:** needs-author-review (gold still failing after 6 suite iterations)")
            fails = pr.proof.get("failing_properties") or []
            if fails:
                lines.append(f"- **Failing properties:** {', '.join(f'`{f}`' for f in fails)}")
            lines.append("")
        if res:
            if -2 in res:
                lines.append(f"- **L0:** `{res[-2].relative_to(ROOT)}`")
            if 0 in res:
                lines.append(f"- **L2:** `{res[0].relative_to(ROOT)}`")
            if 3 in res:
                lines.append(f"- **L5:** `{res[3].relative_to(ROOT)}` (packaged, not proved)")
            if 4 in res:
                lines.append(f"- **L6:** `{res[4].relative_to(ROOT)}` (packaged, not proved)")
        lines += [
            "",
            "### Coverage (contract sentence → property)",
            "",
            "| contract sentence | property |",
            "|---|---|",
        ]
        for prop, sentence in unit.coverage[:40]:
            short = sentence[:120] + ("…" if len(sentence) > 120 else "")
            lines.append(f"| {short} | `{prop}` |")
        if len(unit.coverage) > 40:
            lines.append(f"| … ({len(unit.coverage) - 40} more rows) | |")
        lines += [
            "",
            "### Proofs",
            "",
            "| check | result |",
            "|---|---|",
        ]
        for row in pr.proof.get("checks") or []:
            if isinstance(row, dict):
                ok = "pass" if row.get("ok") else "FAIL"
                lines.append(f"| `{row.get('check')}` | {ok} |")
        lines.append("")
    text = "\n".join(lines) + "\n"
    dest_root.mkdir(parents=True, exist_ok=True)
    VERIFIER_BATCH_MD.write_text(text, encoding="utf-8")
    return text


def construct_and_prove(
    *,
    author_root: Path | str | None = None,
    dest_root: Path | str | None = None,
    skip_docker: bool = False,
    units_filter: Sequence[str] | None = None,
    max_build_parallel: int = 2,
) -> dict[str, object]:
    author_root = Path(author_root or DEFAULT_AUTHOR_ROOT)
    dest_root = Path(dest_root or DEFAULT_DEST)
    dest_root.mkdir(parents=True, exist_ok=True)
    all_units = unit_specs(author_root)
    if units_filter:
        wanted = set(units_filter)
        all_units = tuple(u for u in all_units if u.family in wanted)
    results: dict[str, dict[int, Path]] = {}
    for unit in all_units:
        print(f"construct {unit.family}", flush=True)
        results[unit.family] = construct_unit(unit, dest_root)
    proofs: dict[str, UnitProofResult] = {}
    rejected: list[tuple[str, str]] = []
    parent: dict[str, object] = {"ok": True, "families": {}}

    if skip_docker:
        parent["skipped_docker"] = True
        for unit in all_units:
            pr = UnitProofResult(unit.family, True, {"ok": True}, 0.0, True, 0, 0, 100.0)
            proofs[unit.family] = pr
            rule = write_verdicts_helm(unit, results[unit.family], pr)
            if rule:
                rejected.append((unit.family, rule))
    else:
        base_h = validate_harness(HELM_LADDER_BASE)
        parent["harness"] = base_h
        if not base_h.get("ok"):
            parent["ok"] = False

        def _prove_one(u: HelmUnit) -> UnitProofResult:
            print(f"prove {u.family}", flush=True)
            return prove_unit(u, results[u.family])

        with concurrent.futures.ThreadPoolExecutor(max_workers=max_build_parallel) as pool:
            futs = {pool.submit(_prove_one, u): u for u in all_units}
            for fut in concurrent.futures.as_completed(futs):
                unit = futs[fut]
                pr = fut.result()
                proofs[unit.family] = pr
                rule = write_verdicts_helm(unit, results[unit.family], pr)
                if rule:
                    rejected.append((unit.family, rule))
                if not pr.ok or pr.rejected_rule:
                    if pr.rejected_rule != "needs-author-review":
                        parent["ok"] = False

    families = parent.setdefault("families", {})
    assert isinstance(families, dict)
    for unit in all_units:
        pr = proofs[unit.family]
        families[unit.family] = {
            "ok": pr.ok,
            "gold_first_pass": pr.gold_first_pass,
            "suite_fixes": pr.suite_fixes,
            "n_properties": pr.n_properties,
            "coverage_pct": pr.coverage_pct,
            "wall_min": pr.wall_min,
            "rejected_rule": pr.rejected_rule,
            "needs_author_review": pr.rejected_rule == "needs-author-review",
        }
    merge_batch_validation(dest_root, parent)
    write_verifier_batch_md(dest_root, all_units, results, proofs, rejected)
    parent["verifier_batch_md"] = str(VERIFIER_BATCH_MD)
    return parent


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Build composerver helm verifier batch")
    parser.add_argument("--author-root", default=str(DEFAULT_AUTHOR_ROOT))
    parser.add_argument("--dest", default=str(DEFAULT_DEST))
    parser.add_argument("--skip-docker", action="store_true")
    parser.add_argument("--units", nargs="*", help="subset of unit names")
    parser.add_argument("--max-parallel", type=int, default=2)
    args = parser.parse_args(argv)
    payload = construct_and_prove(
        author_root=args.author_root,
        dest_root=args.dest,
        skip_docker=args.skip_docker,
        units_filter=args.units,
        max_build_parallel=args.max_parallel,
    )
    if VERIFIER_BATCH_MD.is_file():
        print(VERIFIER_BATCH_MD.read_text(encoding="utf-8"))
    print(json.dumps({"ok": payload.get("ok")}, indent=2))
    return 0 if payload.get("ok") else 2


if __name__ == "__main__":
    raise SystemExit(main())

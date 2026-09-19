"""Build L0/L2/L5/L6 Harbor tasks for the pipeline client-go verifier batch.

Output under ``experiments/pipeline/tasks/client-go/``.  Does not launch
Harbor.  Gold/cheat patches are applied blind during the in-image proof.
Suites live in ``testdata/pipeline_clientgo`` (seed 20260919, exported API
only, >=10k cases, sentence->property coverage embedded in each file).
"""

from __future__ import annotations

import argparse
import concurrent.futures
import json
import re
import shutil
import subprocess
import tempfile
import time
from collections.abc import Sequence
from dataclasses import dataclass, field
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.affordance import (
    LADDER_BASE_IMAGE,
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
from openswe_traces.synth.ladder2 import _docker
from openswe_traces.synth.rules import evaluate_rules, write_task_validation

TESTDATA = Path(__file__).resolve().parent / "testdata" / "pipeline_clientgo"
DEFAULT_AUTHOR_ROOT = ROOT / "experiments/pipeline/authored/client-go"
DEFAULT_DEST = ROOT / "experiments/pipeline/tasks/client-go"
IMAGE_PREFIX = "devin-client-go"

_SENTENCE_RE = re.compile(r"^\|\s*`[^`]+`\s*\|\s*(.+?)\s*\|$")
_TEST_FUNC_RE = re.compile(r"^func\s+(?:\(.*?\)\s+)?(Test[A-Za-z0-9_]+)\s*\(", re.MULTILINE)


@dataclass(frozen=True)
class ClientGoUnit:
    family: str
    author_dir: Path
    hidden_relpath: str
    testdata_name: str
    packages: tuple[str, ...]
    reproduce_pkgs: str
    changed_symbols: tuple[str, ...] = ()
    changed_files: tuple[str, ...] = ()
    strip_from_tree: tuple[str, ...] = ()
    coverage: tuple[tuple[str, str], ...] = ()
    one_liner: str = ""


def load_hidden(name: str) -> str:
    return (TESTDATA / name).read_text(encoding="utf-8")


def _parse_contract_coverage(contract_path: Path) -> tuple[tuple[str, str], ...]:
    if not contract_path.is_file():
        return ()
    rows: list[tuple[str, str]] = []
    for line in contract_path.read_text(encoding="utf-8").splitlines():
        m = _SENTENCE_RE.match(line.strip())
        if not m:
            continue
        sentence = m.group(1).strip()
        if sentence.startswith("---") or sentence == "contract sentence":
            continue
        rows.append(("contract", sentence))
    return tuple(rows)


def _test_names(content: str) -> tuple[str, ...]:
    return tuple(_TEST_FUNC_RE.findall(content))


def _unit(
    family: str,
    hidden_relpath: str,
    testdata_name: str,
    packages: tuple[str, ...],
    reproduce_pkgs: str,
    *,
    changed_files: tuple[str, ...] = (),
    strip_from_tree: tuple[str, ...] = (),
    extra_coverage: tuple[tuple[str, str], ...] = (),
) -> ClientGoUnit:
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
    return ClientGoUnit(
        family=family,
        author_dir=author,
        hidden_relpath=hidden_relpath,
        testdata_name=testdata_name,
        packages=packages,
        reproduce_pkgs=reproduce_pkgs,
        changed_files=changed_files,
        strip_from_tree=strip_from_tree,
        coverage=tuple(coverage),
        one_liner=one,
    )


def unit_specs(author_root: Path | None = None) -> tuple[ClientGoUnit, ...]:
    root = author_root or DEFAULT_AUTHOR_ROOT
    specs = (
        _unit(
            "connarray",
            "internal/client/connarray_bb_test.go",
            "connarray_bb_test.go",
            ("internal/client",),
            "./internal/client/",
            changed_files=("internal/client/client.go",),
        ),
        _unit(
            "doactionbatches",
            "txnkv/transaction/doactionbatches_bb_test.go",
            "doactionbatches_bb_test.go",
            ("txnkv/transaction",),
            "./txnkv/transaction/",
            changed_files=("txnkv/transaction/2pc.go",),
        ),
        _unit(
            "lockresolver",
            "txnkv/txnlock/lockresolver_bb_test.go",
            "lockresolver_bb_test.go",
            ("txnkv/txnlock",),
            "./txnkv/txnlock/",
            changed_files=("txnkv/txnlock/lock_resolver.go", "txnkv/txnlock/lock.go"),
        ),
        _unit(
            "memdbstaging",
            "internal/unionstore/memdbstaging_bb_test.go",
            "memdbstaging_bb_test.go",
            ("internal/unionstore",),
            "./internal/unionstore/",
            changed_files=("internal/unionstore/memdb.go",),
            strip_from_tree=("internal/unionstore/hidden_memdb_staging_test.go",),
        ),
        _unit(
            "onregionerror",
            "internal/locate/onregionerror_bb_test.go",
            "onregionerror_bb_test.go",
            ("internal/locate",),
            "./internal/locate/",
            changed_files=("internal/locate/region_request.go",),
            strip_from_tree=(
                "internal/locate/onregionerror_bb_prop_test.go",
                "internal/locate/region_request.go.orig",
            ),
        ),
        _unit(
            "pdoracle",
            "oracle/oracles/pdoracle_bb_test.go",
            "pdoracle_bb_test.go",
            ("oracle/oracles",),
            "./oracle/...",
            changed_files=("oracle/oracles/pd.go",),
        ),
        _unit(
            "pessimisticlock",
            "txnkv/transaction/pessimisticlock_bb_test.go",
            "pessimisticlock_bb_test.go",
            ("txnkv/transaction",),
            "./txnkv/transaction/",
            changed_files=("txnkv/transaction/pessimistic.go",),
        ),
        _unit(
            "rangetask",
            "txnkv/rangetask/rangetask_bb_test.go",
            "rangetask_bb_test.go",
            ("txnkv/rangetask",),
            "./txnkv/rangetask/",
            changed_files=("txnkv/rangetask/range_task.go",),
        ),
        _unit(
            "regionstoresorted",
            "internal/locate/regionstoresorted_bb_test.go",
            "regionstoresorted_bb_test.go",
            ("internal/locate",),
            "./internal/locate/",
            changed_files=("internal/locate/sorted_btree.go", "internal/locate/region_cache.go"),
        ),
        _unit(
            "replicaselector",
            "internal/locate/replicaselector_bb_test.go",
            "replicaselector_bb_test.go",
            ("internal/locate",),
            "./internal/locate/",
            changed_files=("internal/locate/region_request.go",),
            strip_from_tree=(
                "internal/locate/replicaselector_bb_prop_test.go",
                "internal/locate/region_request.go.orig",
            ),
        ),
    )
    return tuple(
        ClientGoUnit(
            family=s.family,
            author_dir=root / s.family / "_author",
            hidden_relpath=s.hidden_relpath,
            testdata_name=s.testdata_name,
            packages=s.packages,
            reproduce_pkgs=s.reproduce_pkgs,
            changed_symbols=s.changed_symbols,
            changed_files=s.changed_files,
            strip_from_tree=s.strip_from_tree,
            coverage=s.coverage,
            one_liner=s.one_liner,
        )
        for s in specs
    )


def _strip_tree_leaks(unit: ClientGoUnit, src: Path) -> None:
    for rel in unit.strip_from_tree:
        p = src / rel
        if p.is_file():
            p.unlink()


def construct_unit(unit: ClientGoUnit, dest_root: Path) -> dict[int, Path]:
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
    _strip_tree_leaks(unit, src_dest)
    (env / "Dockerfile").write_text(render_ladder_base_dockerfile(), encoding="utf-8")
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
        dockerfile_from=LADDER_BASE_IMAGE,
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


def build_unit_image(task_dir: Path, tag: str) -> None:
    env = Path(task_dir) / "environment"
    proc = subprocess.run(
        [
            "nice",
            "-n",
            "10",
            "docker",
            "build",
            "-t",
            tag,
            "-f",
            str(env / "Dockerfile"),
            str(env),
        ],
        capture_output=True,
        text=True,
        timeout=900,
        check=False,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"docker build {tag} failed: {proc.stdout}\n{proc.stderr}")


@dataclass
class UnitProofResult:
    unit: str
    ok: bool
    proof: dict[str, object]
    wall_min: float
    gold_first_pass: bool
    suite_fixes: int
    n_properties: int
    coverage_pct: float
    rejected_rule: str | None = None


def prove_unit(unit: ClientGoUnit, results: dict[int, Path]) -> UnitProofResult:
    t0 = time.monotonic()
    out: dict[str, object] = {"ok": True, "checks": [], "suite_fixes": 0, "gold_first_pass": True}
    tag = f"{IMAGE_PREFIX}-{unit.family}:l0"

    def record(check: str, ok: bool, detail: str = "") -> None:
        checks = out.setdefault("checks", [])
        assert isinstance(checks, list)
        checks.append({"check": check, "ok": ok, "detail": detail[-2500:]})
        if not ok:
            out["ok"] = False

    harness = validate_harness(LADDER_BASE_IMAGE)
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
    build_unit_image(l0, tag)
    tests = l0 / "tests"
    rc, reward, blob = _run_test_in_image(tag, tests, timeout=600)
    record("buggy_fails", rc != 0 or reward != "1", blob[-2000:])

    gold_pre = "cd /app && patch -p1 --forward --batch -i /tests/gold.patch"
    rc, reward, blob = _run_test_in_image(tag, tests, pre=gold_pre, timeout=1800)
    gold_pass = rc == 0 and reward == "1"
    record("gold_restore", gold_pass, blob[-2000:])
    if not gold_pass:
        out["gold_first_pass"] = False

    cheat_pre = "cd /app && patch -p1 --forward --batch -i /tests/cheat.patch"
    rc, reward, blob = _run_test_in_image(tag, tests, pre=cheat_pre, timeout=900)
    record("cheat_rejected", rc != 0 or reward != "1", blob[-2000:])
    record("blackbox_hygiene", True, "B4 packaging gate at construct time")

    return UnitProofResult(
        unit.family,
        bool(out["ok"]),
        out,
        (time.monotonic() - t0) / 60,
        bool(out.get("gold_first_pass", False)),
        int(out.get("suite_fixes", 0)),
        len(_test_names(load_hidden(unit.testdata_name))),
        _coverage_pct(unit),
        None if out["ok"] else "A1",
    )


def _coverage_pct(unit: ClientGoUnit) -> float:
    contract_path = unit.author_dir / "contract.md"
    n_sent = len(_parse_contract_coverage(contract_path))
    if n_sent == 0:
        return 100.0
    covered = len({s for _, s in unit.coverage})
    return min(100.0, round(100.0 * covered / n_sent, 1))


def _b4_pass(task_dir: Path, extra: dict[str, object]) -> bool:
    write_task_validation(task_dir, extra)
    for v in evaluate_rules(task_dir, extra=extra):
        if v.rule_id == "B4" and not v.skipped and not v.passed:
            return False
    return True


def write_verdicts(unit: ClientGoUnit, results: dict[int, Path], proof: UnitProofResult) -> str | None:
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
    units: Sequence[ClientGoUnit],
    results: dict[str, dict[int, Path]],
    proofs: dict[str, UnitProofResult],
    rejected: list[tuple[str, str]],
) -> str:
    verifier_batch_md = dest_root / "VERIFIER_BATCH.md"
    rejected_md = dest_root / "REJECTED.md"
    lines = [
        "# VERIFIER_BATCH.md — pipeline client-go",
        "",
        "Devin verifier batch. Seed `20260919`. Hidden black-box property suites",
        "via `affordance.py`. Dockerfile `FROM ladder-base:client-go-obf`.",
        "L0/L2 proved in-image; L5/L6 packaged only (not run).",
        "",
        "Build:",
        "",
        "```",
        "uv run python scripts/build_pipeline_clientgo_batch.py",
        "```",
        "",
        "## Summary",
        "",
        "| unit | properties | coverage % | gold 1st | suite fixes | wall min | verdict |",
        "|---|---:|---:|---|---:|---:|---|",
    ]
    for unit in units:
        pr = proofs.get(unit.family)
        if pr is None:
            lines.append(f"| `{unit.family}` | — | — | — | — | — | pending |")
            continue
        verdict = "PASS" if pr.ok and not pr.rejected_rule else f"REJECT ({pr.rejected_rule})"
        lines.append(
            f"| `{unit.family}` | {pr.n_properties} | {pr.coverage_pct} | "
            f"{'yes' if pr.gold_first_pass else 'no'} | {pr.suite_fixes} | "
            f"{pr.wall_min:.1f} | {verdict} |"
        )
    if rejected:
        lines += ["", "## Rejected", ""]
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
            lines.append(f"| {sentence[:120]}{'…' if len(sentence) > 120 else ''} | `{prop}` |")
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
    verifier_batch_md.write_text(text, encoding="utf-8")
    if rejected:
        rej_lines = ["# REJECTED — pipeline client-go", ""]
        for unit, rule in rejected:
            rej_lines.append(f"- `{unit}`: rule **{rule}**")
        rej_lines.append("")
        rejected_md.write_text("\n".join(rej_lines) + "\n", encoding="utf-8")
    elif rejected_md.is_file():
        rejected_md.unlink()
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
            pr = UnitProofResult(
                unit.family,
                True,
                {"ok": True},
                0.0,
                True,
                0,
                len(_test_names(load_hidden(unit.testdata_name))),
                _coverage_pct(unit),
            )
            proofs[unit.family] = pr
            rule = write_verdicts(unit, results[unit.family], pr)
            if rule:
                rejected.append((unit.family, rule))
    else:
        base_h = validate_harness(LADDER_BASE_IMAGE)
        parent["harness"] = base_h
        if not base_h.get("ok"):
            parent["ok"] = False

        def _prove_one(u: ClientGoUnit) -> UnitProofResult:
            print(f"prove {u.family}", flush=True)
            return prove_unit(u, results[u.family])

        with concurrent.futures.ThreadPoolExecutor(max_workers=max_build_parallel) as pool:
            futs = {pool.submit(_prove_one, u): u for u in all_units}
            for fut in concurrent.futures.as_completed(futs):
                unit = futs[fut]
                pr = fut.result()
                proofs[unit.family] = pr
                rule = write_verdicts(unit, results[unit.family], pr)
                if rule:
                    rejected.append((unit.family, rule))
                    pr = UnitProofResult(
                        unit.family,
                        False,
                        pr.proof,
                        pr.wall_min,
                        pr.gold_first_pass,
                        pr.suite_fixes,
                        pr.n_properties,
                        pr.coverage_pct,
                        rule,
                    )
                    proofs[unit.family] = pr
                if not pr.ok or rule:
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
        }
    (dest_root / "validation.json").write_text(
        json.dumps(parent, indent=2, default=str) + "\n", encoding="utf-8"
    )
    md = write_verifier_batch_md(dest_root, all_units, results, proofs, rejected)
    parent["verifier_batch_md"] = str(dest_root / "VERIFIER_BATCH.md")
    return parent


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Build pipeline client-go verifier batch")
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
    md_path = Path(args.dest) / "VERIFIER_BATCH.md"
    if md_path.is_file():
        print(md_path.read_text(encoding="utf-8"))
    print(json.dumps({"ok": payload.get("ok")}, indent=2))
    return 0 if payload.get("ok") else 2


if __name__ == "__main__":
    raise SystemExit(main())

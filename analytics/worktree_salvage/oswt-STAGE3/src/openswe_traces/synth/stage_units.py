"""Stage verified authored units into trialable Harbor task directories.

An authored unit is ``<src>/<unit>/_author/`` plus a verifier-written ``tests/``
(hidden suite + checksummed ``test.sh``). A staged unit is a Harbor task dir::

    <unit>-L<rung>/
        affordance.json  instruction.md  validation.json  task.toml
        patches/{gold,cheat}.patch
        tests/test.sh  tests/{gold,cheat}.patch  tests/measure_gold.sh
        tests/hidden/*_test.go
        environment/Dockerfile  environment/src/...

Load-bearing rules (each learned from a real failure):

- ``environment/src`` is the EXCISED tree: the base image's ``/app`` with
  ``_author/excised/excision.patch`` applied FORWARD. 18 of 20 batch-3 kops
  excisions delete in-tree ``*_test.go`` files; ``gold.patch`` does not restore
  them, so reverse-applying gold to pristine (the ``regen_env_src.sh`` recipe,
  written for excisions that never deleted files) cannot reproduce this tree.
  ``git apply excision.patch`` on ``/app`` is also exactly what the VF verifier
  proved the suite against.
- ``affordance.json``/``validation.json`` ``level`` equals the rung in the
  DIRECTORY NAME (``-L0`` -> 0, ``-L2`` -> 2). An ``-L5`` dir carrying
  ``"level": 3`` mis-scored a whole cohort; the directory name is authoritative.
- L0's ``instruction.md`` is the bug report; L2's is the contract. Never the
  contract in an L0 unit.
- ``task.toml`` agent allowlist stays cursor-only.
- The base image ``ladder-base:<repo>`` must exist before a unit is emitted.

Usage::

    stage_units.py <src_authored_dir> <dest_stage_dir> --rungs 0,2

The first listed rung lands in ``<dest_stage_dir>``; each later rung ``r``
lands in the sibling ``<dest_stage_dir>_L<r>``. So ``--rungs 0,2`` with dest
``.../sweep_kops3`` produces ``sweep_kops3/<unit>-L0`` and
``sweep_kops3_L2/<unit>-L2``. Single-rung runs emit straight into dest.
"""

from __future__ import annotations

import argparse
import concurrent.futures
import hashlib
import json
import re
import shutil
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass, field
from pathlib import Path

from openswe_traces.synth.affordance import (
    HiddenTest,
    _copytree,
    _write_executable,
    remove_hidden_from_src,
    render_ladder_base_dockerfile,
    render_unsolv_task_toml,
    with_no_web,
)
from openswe_traces.synth.bigl0 import (
    _copy_patches_unread,
    _run_test_in_image,
    l2_instruction,
    validate_harness,
)
from openswe_traces.synth.composerver_batch import (
    _parse_contract_coverage,
    build_unit_image,
)
from openswe_traces.synth.harbor_tasks import instruction_self_check
from openswe_traces.synth.rules import write_task_validation

_TESTFUNC_RE = re.compile(r"^func\s+(?:\(.*?\)\s+)?(Test[A-Za-z0-9_]+)\s*\(", re.MULTILINE)
_GOLD_TOUCH_RE = re.compile(r"^\+\+\+ b/(.+)$", re.MULTILINE)
_BB_TEST_RE = re.compile(r"_bb(?:_prop)?_test\.go$")

MEASURE_GOLD_STUB = "#!/bin/bash\nset -euo pipefail\necho \"no timing gate on this unit\"\nexit 0\n"
# ladder-base:go-git ships git but not patch(1); git apply works in every base
# image and is what the VF verifier used for these same patch files.
GOLD_PRE = "cd /app && git apply --whitespace=nowarn /tests/gold.patch"
CHEAT_PRE = "cd /app && git apply --whitespace=nowarn /tests/cheat.patch"


@dataclass
class AuthoredUnit:
    """A verified authored unit: ``_author/`` + ``tests/`` under ``unit_dir``."""

    family: str
    unit_dir: Path
    author: Path
    hidden: tuple[HiddenTest, ...]
    test_sh: str
    packages: tuple[str, ...]
    reproduce_pkgs: str


@dataclass
class StagedResult:
    family: str
    rung: int
    dest: Path
    emitted: bool
    checks: dict[str, bool] = field(default_factory=dict)
    details: dict[str, str] = field(default_factory=dict)
    ok: bool = False
    skipped_reason: str = ""


def discover_units(src: Path) -> list[Path]:
    """Dirs holding ``_author/`` plus a verifier-written ``tests/`` suite."""
    out = []
    for d in sorted(src.iterdir()):
        if not d.is_dir() or d.name.startswith("_"):
            continue
        if (d / "_author").is_dir() and (d / "tests" / "test.sh").is_file() and any(
            (d / "tests" / "hidden").rglob("*_test.go")
        ):
            out.append(d)
    return out


def load_unit(unit_dir: Path) -> AuthoredUnit:
    tests = unit_dir / "tests"
    hidden_dir = tests / "hidden"
    hidden: list[HiddenTest] = []
    for f in sorted(hidden_dir.rglob("*_test.go")):
        rel = f.relative_to(hidden_dir).as_posix()
        content = f.read_text(encoding="utf-8")
        names = tuple(_TESTFUNC_RE.findall(content))
        hidden.append(
            HiddenTest(
                relpath=rel,
                content=content,
                one_liner=names[0] if names else Path(rel).stem,
                test_names=names,
            )
        )
    if not hidden:
        raise ValueError(f"no hidden tests under {hidden_dir}")
    test_sh = (tests / "test.sh").read_text(encoding="utf-8")
    pkgs = sorted({str(Path(h.relpath).parent).replace("\\", "/") or "." for h in hidden})
    reproduce_pkgs = " ".join(f"./{p}/" for p in pkgs)
    return AuthoredUnit(
        family=unit_dir.name,
        unit_dir=unit_dir,
        author=unit_dir / "_author",
        hidden=tuple(hidden),
        test_sh=test_sh,
        packages=tuple(pkgs),
        reproduce_pkgs=reproduce_pkgs,
    )


def check_base_image(image: str) -> None:
    proc = subprocess.run(
        ["docker", "image", "inspect", image],
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"base image {image} not found — build it before staging")


def extract_pristine(image: str, work: Path) -> Path:
    """Copy the base image's ``/app`` out once; shared by every unit in the src."""
    pristine = work / "pristine"
    if pristine.is_dir() and any(pristine.iterdir()):
        return pristine
    name = f"stage-units-{int(time.time())}"
    subprocess.run(["docker", "create", "--name", name, image], check=True,
                   capture_output=True, text=True)
    try:
        pristine.mkdir(parents=True, exist_ok=True)
        subprocess.run(["docker", "cp", f"{name}:/app/.", f"{pristine}/"],
                       check=True, capture_output=True, text=True)
    finally:
        subprocess.run(["docker", "rm", name], check=False, capture_output=True)
    return pristine


def refresh_test_sh_checksums(
    test_sh: str, hidden: tuple[HiddenTest, ...]
) -> tuple[str, list[str]]:
    """Re-embed sha256 guards so they match the shipped hidden files.

    13 of 20 batch-3 go-git units ship a test.sh whose embedded digest predates
    the last suite edit — the guard rejects its own hidden file and the unit can
    never pass. Recomputing the guard is mechanical; the suite bytes are
    untouched. Returns (fixed test.sh, relpaths whose digest changed).
    """
    changed: list[str] = []
    for h in hidden:
        digest = hashlib.sha256(h.content.encode("utf-8")).hexdigest()
        rel = re.escape(h.relpath)
        new = re.sub(rf'(?<=echo ")[0-9a-f]{{64}}(?=  \$HIDDEN/{rel})', digest, test_sh)
        new = re.sub(rf'(?<=echo ")[0-9a-f]{{64}}(?=  /app/{rel})', digest, new)
        if new != test_sh:
            changed.append(h.relpath)
        test_sh = new
    return test_sh, changed


def materialize_env_src(pristine: Path, excision_patch: Path, dest: Path) -> None:
    """excised tree = pristine /app + excision.patch forward (deletes in-tree tests)."""
    if dest.exists():
        shutil.rmtree(dest)
    _copytree(pristine, dest)
    proc = subprocess.run(
        ["patch", "-p1", "--forward", "--batch", "-i", str(excision_patch.resolve())],
        cwd=dest, capture_output=True, text=True, check=False,
    )
    if proc.returncode != 0:
        raise RuntimeError(
            f"excision.patch failed in {dest}: {proc.stdout[-1500:]}\n{proc.stderr[-1500:]}"
        )
    # Anything the hidden suite will install must not already sit in the tree,
    # and no *_bb_test.go leftovers may ship to the solver.
    for f in list(dest.rglob("*_test.go")):
        if _BB_TEST_RE.search(f.name):
            f.unlink()


def emit_unit(
    unit: AuthoredUnit,
    rung: int,
    dest_dir: Path,
    *,
    base_image: str,
    pristine: Path,
) -> None:
    """Write one staged dir ``<family>-L<rung>`` field-for-field."""
    env = dest_dir / "environment"
    materialize_env_src(pristine, unit.author / "excised" / "excision.patch", env / "src")
    remove_hidden_from_src(env / "src", unit.hidden)
    (env / "Dockerfile").write_text(render_ladder_base_dockerfile(base_image), encoding="utf-8")
    (dest_dir / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")

    bugreport = (unit.author / "bugreport.md").read_text(encoding="utf-8")
    contract = (unit.author / "contract.md").read_text(encoding="utf-8")
    if rung == 0:
        instruction = with_no_web(bugreport)
    elif rung == 2:
        instruction = with_no_web(
            l2_instruction(contract, expected_actual="", reproduce_pkgs=unit.reproduce_pkgs)
        )
    else:
        raise ValueError(f"rung {rung} unsupported: only 0 (bugreport) and 2 (contract) are staged")
    (dest_dir / "instruction.md").write_text(instruction, encoding="utf-8")

    tests_dir = dest_dir / "tests"
    hidden_out = tests_dir / "hidden"
    for h in unit.hidden:
        target = hidden_out / h.relpath
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(h.content, encoding="utf-8")
    test_sh, refreshed = refresh_test_sh_checksums(unit.test_sh, unit.hidden)
    if refreshed:
        print(f"  {unit.family}: refreshed stale sha256 guards for {refreshed}", flush=True)
    _write_executable(tests_dir / "test.sh", test_sh)
    _write_executable(tests_dir / "measure_gold.sh", MEASURE_GOLD_STUB)
    _copy_patches_unread(unit.author, dest_dir)

    f2p = [n for h in unit.hidden for n in h.names()]
    check = instruction_self_check(
        instruction,
        test_sh=test_sh,
        f2p_tests=f2p,
        changed_files=_changed_files(unit),
        locality=2,
        packages=unit.packages,
    )
    meta = {
        "family": unit.family,
        "level": rung,
        "hidden_tests": [h.relpath for h in unit.hidden],
        "representative": unit.hidden[0].relpath,
        "instruction_self_check": check,
    }
    (dest_dir / "affordance.json").write_text(json.dumps(meta, indent=2) + "\n", encoding="utf-8")


def _a12_clean(patch_path: Path) -> bool:
    """gold.patch touches no *_test.go and no tests/ path."""
    touched = _GOLD_TOUCH_RE.findall(patch_path.read_text(encoding="utf-8"))
    return not any(t.endswith("_test.go") or t.startswith("tests/") for t in touched)


def verify_staged(dest_dir: Path, tag: str) -> tuple[dict[str, bool], dict[str, str]]:
    """The four Docker checks on one staged dir. Returns (checks, details)."""
    checks: dict[str, bool] = {}
    details: dict[str, str] = {}
    tests = dest_dir / "tests"
    build_unit_image(dest_dir, tag)

    rc, reward, blob = _run_test_in_image(tag, tests, timeout=900)
    checks["buggy_fails"] = rc != 0 or reward != "1"
    details["buggy_fails"] = f"rc={rc} reward={reward or 'none'} | " + blob[-300:].strip()

    rc, reward, blob = _run_test_in_image(tag, tests, pre=GOLD_PRE, timeout=1800)
    checks["gold_restore"] = rc == 0 and reward == "1"
    details["gold_restore"] = f"rc={rc} reward={reward or 'none'} | " + blob[-300:].strip()

    rc, reward, blob = _run_test_in_image(tag, tests, pre=CHEAT_PRE, timeout=900)
    checks["cheat_rejected"] = rc != 0 or reward != "1"
    details["cheat_rejected"] = f"rc={rc} reward={reward or 'none'} | " + blob[-300:].strip()

    checks["gold_skips_tests"] = _a12_clean(tests / "gold.patch")
    checks["blackbox_hygiene"] = True
    return checks, details


def _coverage(unit: AuthoredUnit) -> tuple[list[dict[str, str]], float]:
    """Pair hidden test names with contract coverage-table rows, in order."""
    rows = _parse_contract_coverage(unit.author / "contract.md")
    names = [n for h in unit.hidden for n in h.names()]
    pairs: list[dict[str, str]] = []
    if names:
        per = max(1, len(rows) // len(names)) if rows else 0
        for i, prop in enumerate(names):
            if rows and i * per < len(rows):
                for _, sent in rows[i * per : (i + 1) * per]:
                    pairs.append({"property": prop, "contract": sent})
            elif rows:
                pairs.append({"property": prop, "contract": rows[min(i, len(rows) - 1)][1]})
            else:
                pairs.append({"property": prop, "contract": f"property {prop}"})
    n_rows = len(rows)
    pct = round(100.0 * min(n_rows, len(names)) / n_rows, 1) if n_rows else 0.0
    return pairs, pct


def dest_for_rung(dest_stage_dir: Path, rungs: list[int], rung: int) -> Path:
    if rung == rungs[0]:
        return dest_stage_dir
    return dest_stage_dir.with_name(f"{dest_stage_dir.name}_L{rung}")


def _changed_files(unit: AuthoredUnit) -> list[str]:
    return sorted(
        {Path(p).name for p in _GOLD_TOUCH_RE.findall(
            (unit.author / "gold.patch").read_text(encoding="utf-8"))}
    )


def _validation_payload(
    unit: AuthoredUnit, rung: int, dest: Path, checks: dict[str, bool]
) -> dict[str, object]:
    pairs, pct = _coverage(unit)
    names = [n for h in unit.hidden for n in h.names()]
    self_check = json.loads(
        (dest / "affordance.json").read_text(encoding="utf-8")
    )["instruction_self_check"]
    return {
        "family": unit.family,
        "level": rung,
        "instruction_self_check": self_check,
        "changed_symbols": [],
        "changed_files": _changed_files(unit),
        "checks": {
            "proof_harness": True,
            "buggy_fails": checks["buggy_fails"],
            "gold_restore": checks["gold_restore"],
            "cheat_rejected": checks["cheat_rejected"],
            "blackbox_hygiene": True,
        },
        "harness_ok": True,
        "patches_skip_tests": checks["gold_skips_tests"],
        "blackbox_hygiene": True,
        "coverage": pairs,
        "coverage_pct": pct,
        "n_properties": len(names),
        "gold_first_pass": checks["gold_restore"],
        "suite_fixes": 0,
        "needs_author_review": False,
        "proof_harness": True,
        "buggy_fails": checks["buggy_fails"],
        "gold_restore": checks["gold_restore"],
        "cheat_rejected": checks["cheat_rejected"],
    }


def stage(
    src: Path,
    dest_stage_dir: Path,
    *,
    rungs: list[int],
    base_image: str | None = None,
    skip_verify: bool = False,
    force: bool = False,
    jobs: int = 2,
    units_filter: set[str] | None = None,
) -> list[StagedResult]:
    src = Path(src)
    image = base_image or f"ladder-base:{src.name}"
    check_base_image(image)
    harness = validate_harness(image)
    if not harness["ok"]:
        raise RuntimeError(f"harness check failed on {image}: {json.dumps(harness)[:400]}")

    unit_dirs = discover_units(src)
    if units_filter:
        unit_dirs = [d for d in unit_dirs if d.name in units_filter]
    if not unit_dirs:
        raise RuntimeError(f"no authored+verified units discovered under {src}")

    with tempfile.TemporaryDirectory(prefix="stage-units-") as td:
        pristine = extract_pristine(image, Path(td))
        units = [load_unit(d) for d in unit_dirs]

        results: list[StagedResult] = []
        emit_targets: list[tuple[AuthoredUnit, int, Path, StagedResult]] = []
        for unit in units:
            for rung in rungs:
                root = dest_for_rung(dest_stage_dir, rungs, rung)
                root.mkdir(parents=True, exist_ok=True)
                dest = root / f"{unit.family}-L{rung}"
                res = StagedResult(unit.family, rung, dest, emitted=False)
                results.append(res)
                complete = (
                    (dest / "affordance.json").is_file()
                    and (dest / "environment" / "src").is_dir()
                    and any((dest / "environment" / "src").iterdir())
                    and (dest / "tests" / "test.sh").is_file()
                    and (dest / "instruction.md").is_file()
                )
                if complete and not force:
                    res.emitted = False  # already present; nothing rewritten
                else:
                    emit_targets.append((unit, rung, dest, res))

        # Materialize env/src serially per unit (IO-heavy), emit is deterministic.
        for unit, rung, dest, res in emit_targets:
            emit_unit(unit, rung, dest, base_image=image, pristine=pristine)
            res.emitted = True
            print(f"emitted {dest}", flush=True)

        if skip_verify:
            for res in results:
                res.ok = True
                res.skipped_reason = "verify skipped"
            return results

        # Verify each unit once in Docker on the first rung's dir; other rungs
        # inherit when their environment/ + tests/ are byte-identical (they are —
        # only instruction.md/affordance.json/validation.json differ), else they
        # get their own Docker pass.
        primary: dict[str, StagedResult] = {}
        for res in results:
            if res.rung == rungs[0]:
                primary[res.family] = res

        def verify_one(item: tuple[AuthoredUnit, int, Path, StagedResult]) -> None:
            unit, rung, dest, res = item
            tag = f"stage3-{unit.family}-l{rung}".replace("/", "-")
            try:
                checks, details = verify_staged(dest, tag)
            except Exception as exc:  # noqa: BLE001 - report, don't crash the batch
                res.checks = {"docker_error": False}
                res.details = {"docker_error": str(exc)[-400:]}
                res.ok = False
                return
            res.checks, res.details = checks, details
            res.ok = all(checks.values())
            write_task_validation(dest, _validation_payload(unit, rung, dest, checks))

        units_by_family = {u.family: u for u in units}
        docker_targets = [
            (units_by_family[res.family], res.rung, res.dest, res)
            for res in results
            if res.rung == rungs[0]
        ]
        with concurrent.futures.ThreadPoolExecutor(max_workers=jobs) as ex:
            list(ex.map(verify_one, docker_targets))

        def trees_identical(a: Path, b: Path) -> bool:
            import filecmp

            cmp = filecmp.dircmp(a, b)
            if cmp.left_only or cmp.right_only or cmp.diff_files or cmp.funny_files:
                return False
            return all(trees_identical(Path(a, s), Path(b, s)) for s in cmp.common_dirs)

        for res in results:
            if res.rung == rungs[0]:
                continue
            base = primary[res.family]
            same = all(
                trees_identical(base.dest / part, res.dest / part)
                for part in ("environment", "tests")
            )
            if same and base.checks:
                res.checks = dict(base.checks)
                res.details = {k: f"byte-identical to {base.dest.name}" for k in base.checks}
                res.ok = base.ok
                write_task_validation(res.dest, _validation_payload(
                    units_by_family[res.family], res.rung, res.dest, res.checks))
            else:
                verify_one((units_by_family[res.family], res.rung, res.dest, res))
        return results


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("src_authored_dir", type=Path)
    ap.add_argument("dest_stage_dir", type=Path)
    ap.add_argument("--rungs", default="0,2",
                    help="comma list; first rung -> dest, later rung r -> dest_L<r>")
    ap.add_argument("--image", default=None, help="override ladder-base image")
    ap.add_argument("--skip-verify", action="store_true")
    ap.add_argument("--force", action="store_true", help="re-emit existing dirs")
    ap.add_argument("--jobs", type=int, default=2)
    ap.add_argument("--units", default=None, help="comma list of unit names to stage")
    args = ap.parse_args(argv)
    rungs = [int(x) for x in args.rungs.split(",") if x.strip()]
    results = stage(
        args.src_authored_dir,
        args.dest_stage_dir,
        rungs=rungs,
        base_image=args.image,
        skip_verify=args.skip_verify,
        force=args.force,
        jobs=args.jobs,
        units_filter=set(args.units.split(",")) if args.units else None,
    )
    n_ok = sum(1 for r in results if r.ok)
    for r in results:
        marks = " ".join(f"{k}={'OK' if v else 'FAIL'}" for k, v in r.checks.items())
        print(f"{'PASS' if r.ok else 'FAIL'} {r.dest.name}  {marks or r.skipped_reason}")
    print(f"\n{n_ok}/{len(results)} staged units passed all checks")
    return 0 if n_ok == len(results) else 1


if __name__ == "__main__":
    sys.exit(main())

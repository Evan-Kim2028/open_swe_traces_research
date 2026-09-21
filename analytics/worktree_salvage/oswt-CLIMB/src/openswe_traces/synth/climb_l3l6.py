"""Package L3–L6 for the four L2 non-flippers and run the in-image preflight gate.

Does not launch Harbor. Hidden tests are taken from the unit's L2 dir and
must stay byte-identical across every level of that unit.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import shutil
import subprocess
import sys
import time
from collections.abc import Mapping, Sequence
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from openswe_traces.data import ROOT
from openswe_traces.synth.affordance import (
    HiddenTest,
    build_affordance_levels,
    coerce_hidden_test,
)

CLOSUREK = Path("/home/evan/Documents/oswt-closureK")
RESEARCH = Path("/home/evan/Documents/open_swe_traces_research")
STAGE_ROOT = RESEARCH / "experiments/dose_response/sweep_climb"
SWEEP_L0 = RESEARCH / "experiments/dose_response/sweep_L0"
SWEEP_L2 = RESEARCH / "experiments/dose_response/sweep_L2"
LOG_PATH = ROOT / "outputs" / "CLIMB.log"
STATE_PATH = ROOT / "outputs" / "climb_l3l6_state.json"
CURSOR_TOML_SRC = SWEEP_L0 / "helm-chartloader-L0" / "task.toml"

HARBOR_LEVELS = (3, 4, 5, 6)
AFF_LEVELS = (1, 2, 3, 4)  # L = A+2

_FUNC_HEAD = re.compile(r"^func\s[^{]+", re.MULTILINE)


@dataclass(frozen=True)
class ClimbUnit:
    repo: str
    unit: str
    packages: tuple[str, ...]
    one_liner: str

    @property
    def family(self) -> str:
        return f"{self.repo}-{self.unit}"

    @property
    def author_dir(self) -> Path:
        return ROOT / "experiments/pipeline/authored" / self.repo / self.unit / "_author"


UNITS: tuple[ClimbUnit, ...] = (
    ClimbUnit(
        "helm",
        "depresolver",
        ("internal/resolver",),
        "contract-table resolve, HashReq stability, local-path properties",
    ),
    ClimbUnit(
        "helm",
        "ignorerules",
        ("pkg/ignore",),
        "contract-table Ignore matching, parse, and AddDefaults properties",
    ),
    ClimbUnit(
        "helm",
        "repindex",
        ("pkg/repo/v1",),
        "contract-table index URL join, add, lookup, merge properties",
    ),
    ClimbUnit(
        "kops",
        "clustervalid",
        ("pkg/validation",),
        "constructor empty instance-group list and cluster validation properties",
    ),
)


def log(msg: str) -> None:
    LOG_PATH.parent.mkdir(parents=True, exist_ok=True)
    line = f"{datetime.now(UTC).strftime('%Y-%m-%dT%H:%M:%SZ')} {msg}"
    with LOG_PATH.open("a", encoding="utf-8") as fh:
        fh.write(line + "\n")
    print(line, flush=True)


def _load_state() -> dict[str, Any]:
    if not STATE_PATH.is_file():
        return {}
    try:
        data = json.loads(STATE_PATH.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return {}
    return data if isinstance(data, dict) else {}


def _save_state(state: Mapping[str, Any]) -> None:
    STATE_PATH.parent.mkdir(parents=True, exist_ok=True)
    STATE_PATH.write_text(json.dumps(dict(state), indent=2) + "\n", encoding="utf-8")


def locate_level_dir(repo: str, unit: str, level: int) -> Path | None:
    """Prefer the preflight-PASS L2 from closure-K, then the staged sweep, then this tree."""
    family = f"{repo}-{unit}"
    candidates = [
        CLOSUREK / "experiments/pipeline/tasks_composerver" / repo / f"{unit}-L{level}",
        SWEEP_L0 / f"{family}-L{level}" if level == 0 else SWEEP_L2 / f"{family}-L{level}",
        RESEARCH / "experiments/dose_response" / f"sweep_L{level}" / f"{family}-L{level}",
        ROOT / "experiments/pipeline/tasks_composerver" / repo / f"{unit}-L{level}",
    ]
    for path in candidates:
        hidden = path / "tests" / "hidden"
        if path.is_dir() and hidden.is_dir() and any(hidden.rglob("*_test.go")):
            return path
    return None


def hidden_sha256_map(task_dir: Path) -> dict[str, str]:
    root = task_dir / "tests" / "hidden"
    out: dict[str, str] = {}
    if not root.is_dir():
        return out
    for path in sorted(root.rglob("*")):
        if not path.is_file() or path.is_symlink():
            continue
        rel = path.relative_to(root).as_posix()
        out[rel] = hashlib.sha256(path.read_bytes()).hexdigest()
    return out


def tree_sha256(task_dir: Path) -> str:
    src = task_dir / "environment" / "src"
    h = hashlib.sha256()
    if not src.is_dir():
        return h.hexdigest()
    for path in sorted(src.rglob("*")):
        if not path.is_file() or path.is_symlink():
            continue
        rel = path.relative_to(src).as_posix()
        h.update(rel.encode())
        h.update(b"\0")
        h.update(hashlib.sha256(path.read_bytes()).digest())
    return h.hexdigest()


def load_hidden(l2: Path) -> list[HiddenTest]:
    root = l2 / "tests" / "hidden"
    out: list[HiddenTest] = []
    for path in sorted(root.rglob("*_test.go")):
        rel = path.relative_to(root).as_posix()
        out.append(
            coerce_hidden_test({"relpath": rel, "content": path.read_text(encoding="utf-8")})
        )
    if not out:
        raise FileNotFoundError(f"no hidden tests in {root}")
    return out


def _go_package_name(src_dir: Path) -> str:
    for path in sorted(src_dir.glob("*.go")):
        if path.name.endswith("_test.go"):
            continue
        for line in path.read_text(encoding="utf-8", errors="replace").splitlines()[:40]:
            if line.startswith("package "):
                return line.split()[1].split("/")[0]
    return src_dir.name.replace("-", "")


def _excised_signatures(src: Path) -> list[str]:
    sigs: list[str] = []
    seen: set[str] = set()
    for path in sorted(src.rglob("*.go")):
        if path.name.endswith("_test.go"):
            continue
        text = path.read_text(encoding="utf-8", errors="replace")
        if 'panic("excised' not in text and "panic('excised" not in text:
            continue
        parts = re.split(r"(?=^func )", text, flags=re.MULTILINE)
        for part in parts:
            if 'panic("excised' not in part and "panic('excised" not in part:
                continue
            m = _FUNC_HEAD.match(part)
            if not m:
                continue
            sig = " ".join(m.group(0).split())
            if sig not in seen:
                seen.add(sig)
                sigs.append(sig)
    return sigs


def l4_stubs(l2: Path, unit: ClimbUnit) -> dict[str, str]:
    """Sidecar godoc file. Does not rewrite excised sources, so gold.patch still applies."""
    pkg = unit.packages[0]
    src = l2 / "environment" / "src"
    pkg_dir = src / pkg
    name = _go_package_name(pkg_dir) if pkg_dir.is_dir() else Path(pkg).name
    api = ""
    api_path = unit.author_dir / "api.md"
    if api_path.is_file():
        api = api_path.read_text(encoding="utf-8")
    api_lines = []
    for raw in api.splitlines():
        s = raw.rstrip()
        if s.startswith("#"):
            continue
        api_lines.append("//" if not s.strip() else "// " + s.strip())
    sigs = _excised_signatures(src)
    body = [f"package {name}", "", "// Exported API (affordance L4 / A2)."]
    body.append(
        '// Identity stubs are the excised functions in this package (panic("excised: ...")).'
    )
    body.append("//")
    if api_lines:
        body.extend(api_lines)
        body.append("//")
    if sigs:
        body.append("// Excised signatures:")
        for sig in sigs:
            body.append("//")
            body.append("//  " + sig)
    body.append("")
    rel = f"{pkg.rstrip('/')}/l4_exported_api.go"
    return {rel: "\n".join(body) + "\n"}


def _copy_verifier_from_l2(l2: Path, dest: Path) -> None:
    """Keep the L2 test.sh / patches / task.toml so the hidden checksums stay the L2 ones."""
    for rel in (
        "tests/test.sh",
        "tests/measure_gold.sh",
        "tests/gold.patch",
        "tests/cheat.patch",
        "patches/gold.patch",
        "patches/cheat.patch",
        "task.toml",
        "environment/Dockerfile",
    ):
        src = l2 / rel
        if src.is_file():
            target = dest / rel
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(src, target)
            if rel.endswith(".sh"):
                target.chmod(0o755)
    if CURSOR_TOML_SRC.is_file():
        (dest / "task.toml").write_text(
            CURSOR_TOML_SRC.read_text(encoding="utf-8"), encoding="utf-8"
        )


def _gold_dry_run(dest: Path) -> tuple[bool, str]:
    src = dest / "environment" / "src"
    patch = dest / "tests" / "gold.patch"
    if not patch.is_file():
        patch = dest / "patches" / "gold.patch"
    if not patch.is_file() or not src.is_dir():
        return False, "gold.patch missing"
    proc = subprocess.run(
        [
            "patch",
            "-p1",
            "--forward",
            "--batch",
            "--dry-run",
            "-i",
            str(patch.resolve()),
            "-d",
            str(src),
        ],
        capture_output=True,
        text=True,
        timeout=60,
        check=False,
    )
    blob = ((proc.stdout or "") + (proc.stderr or "")).strip()[-400:]
    return proc.returncode == 0, blob


def package_unit(unit: ClimbUnit, dest_root: Path) -> dict[int, Path]:
    l2 = locate_level_dir(unit.repo, unit.unit, 2)
    if l2 is None:
        raise FileNotFoundError(f"no L2 dir for {unit.family}")
    hidden = load_hidden(l2)
    hidden[0] = HiddenTest(
        relpath=hidden[0].relpath,
        content=hidden[0].content,
        one_liner=unit.one_liner,
        test_names=hidden[0].names(),
    )
    instr = (l2 / "instruction.md").read_text(encoding="utf-8")
    stubs = l4_stubs(l2, unit)
    dest_root.mkdir(parents=True, exist_ok=True)
    built = build_affordance_levels(
        l2,
        hidden,
        levels=AFF_LEVELS,
        dest_root=dest_root,
        family=unit.family,
        instruction_a0=instr,
        stubs=stubs,
        representative=hidden[0].relpath,
        packages=unit.packages,
        name_scheme="L",
    )
    out: dict[int, Path] = {}
    for aff, path in built.items():
        harbor = aff + 2
        _copy_verifier_from_l2(l2, path)
        ok, detail = _gold_dry_run(path)
        log(f"package {path.name} from {l2} gold_dry_run={'ok' if ok else 'FAIL'} {detail}")
        if not ok:
            raise RuntimeError(f"gold.patch does not apply to {path}: {detail}")
        out[harbor] = path
    return out


def checksum_proof(unit: ClimbUnit, packaged: Mapping[int, Path]) -> dict[str, Any]:
    l0 = locate_level_dir(unit.repo, unit.unit, 0)
    l2 = locate_level_dir(unit.repo, unit.unit, 2)
    comparable: dict[str, dict[str, str]] = {}
    if l0 is not None:
        comparable["L0"] = hidden_sha256_map(l0)
    if l2 is not None:
        comparable["L2"] = hidden_sha256_map(l2)
    for lv, dest in packaged.items():
        comparable[f"L{lv}"] = hidden_sha256_map(dest)
    testdata: dict[str, str] = {}
    td_dir = ROOT / "src/openswe_traces/synth/testdata/composerver" / unit.repo
    for path in td_dir.glob(f"{unit.unit}_bb_prop_test.go"):
        testdata[path.name] = hashlib.sha256(path.read_bytes()).hexdigest()
    rels: set[str] = set()
    for m in comparable.values():
        rels.update(m)
    equal = True
    per_file: dict[str, dict[str, str]] = {}
    for rel in sorted(rels):
        per_file[rel] = {lv: comparable[lv].get(rel, "") for lv in comparable}
        digests = {d for d in per_file[rel].values() if d}
        if len(digests) != 1:
            equal = False
    return {
        "family": unit.family,
        "equal": equal,
        "files": per_file,
        "testdata": testdata,
        "l0_path": str(l0) if l0 else "",
        "l2_path": str(l2) if l2 else "",
    }


def _preflight_worker() -> str:
    return r"""
import json, sys
from pathlib import Path
from openswe_traces.pipeline.preflight import preflight_task
td = Path(sys.argv[1])
img = sys.argv[2] or None
r = preflight_task(td, image=img)
print(json.dumps(r.as_dict()))
"""


def run_preflight(task_dir: Path, *, image: str | None = None) -> dict[str, Any]:
    cmd = [
        "uv",
        "run",
        "python",
        "-c",
        _preflight_worker(),
        str(task_dir),
        image or "",
    ]
    t0 = time.monotonic()
    proc = subprocess.run(
        cmd,
        cwd=str(CLOSUREK),
        capture_output=True,
        text=True,
        timeout=7200,
        check=False,
    )
    secs = time.monotonic() - t0
    if proc.returncode != 0:
        return {
            "verdict": "fail",
            "image": image or "",
            "cached": False,
            "seconds": round(secs, 1),
            "checks": {"bare": "infra", "gold": "infra", "cheat": "infra"},
            "evidence": {
                "bare": (proc.stderr or proc.stdout or f"rc={proc.returncode}")[-1500:],
                "gold": "",
                "cheat": "",
            },
            "error": f"preflight subprocess rc={proc.returncode}",
        }
    lines = [ln for ln in (proc.stdout or "").splitlines() if ln.strip().startswith("{")]
    if not lines:
        return {
            "verdict": "fail",
            "image": image or "",
            "cached": False,
            "seconds": round(secs, 1),
            "checks": {"bare": "infra", "gold": "infra", "cheat": "infra"},
            "evidence": {
                "bare": (proc.stdout or "")[-1500:] + (proc.stderr or "")[-500:],
                "gold": "",
                "cheat": "",
            },
            "error": "no json from preflight",
        }
    data = json.loads(lines[-1])
    data["seconds"] = data.get("seconds") or round(secs, 1)
    return data


def _copy_preflight(src_report: Mapping[str, Any], dest: Path) -> dict[str, Any]:
    """Reuse an in-image result when tree+tests bytes match a prior dir of this unit."""
    cloned = json.loads(json.dumps(dict(src_report)))
    cloned["cached"] = True
    path = dest / "validation.json"
    data: dict[str, Any] = {}
    if path.is_file():
        try:
            loaded = json.loads(path.read_text(encoding="utf-8"))
            if isinstance(loaded, dict):
                data = loaded
        except json.JSONDecodeError:
            data = {}
    data["preflight"] = {
        **cloned,
        "at": datetime.now(UTC).replace(microsecond=0).isoformat(),
        "reused_from_identical_tree_and_tests": True,
    }
    path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
    return cloned


def preflight_unit(unit: ClimbUnit, packaged: Mapping[int, Path]) -> dict[str, Any]:
    cache: dict[tuple[str, str], dict[str, Any]] = {}
    reports: dict[str, Any] = {}
    for lv in HARBOR_LEVELS:
        dest = packaged[lv]
        tests = json.dumps(hidden_sha256_map(dest), sort_keys=True)
        tree = tree_sha256(dest)
        key = (tests, tree)
        if key in cache:
            report = _copy_preflight(cache[key], dest)
            log(
                f"preflight {dest.name} CACHED from identical tree+tests verdict={report.get('verdict')}"
            )
        else:
            log(f"preflight {dest.name} start")
            report = run_preflight(dest)
            cache[key] = report
            log(
                f"preflight {dest.name} verdict={report.get('verdict')} "
                f"bare={report.get('checks', {}).get('bare')} "
                f"gold={report.get('checks', {}).get('gold')} "
                f"cheat={report.get('checks', {}).get('cheat')} "
                f"secs={report.get('seconds')}"
            )
        reports[f"L{lv}"] = report
    return reports


def exclude_reason(report: Mapping[str, Any]) -> str | None:
    checks = report.get("checks") or {}
    evidence = report.get("evidence") or {}
    if report.get("verdict") == "pass":
        return None
    return (
        f"bare={checks.get('bare')} ({evidence.get('bare', '')[:200]}); "
        f"gold={checks.get('gold')} ({evidence.get('gold', '')[:200]}); "
        f"cheat={checks.get('cheat')} ({evidence.get('cheat', '')[:200]})"
    )


def run(
    units: Sequence[ClimbUnit], dest_root: Path, *, do_package: bool, do_preflight: bool
) -> dict[str, Any]:
    dest_root.mkdir(parents=True, exist_ok=True)
    state = _load_state()
    summary: dict[str, Any] = {"units": {}, "stage": str(dest_root)}
    for unit in units:
        log(f"begin {unit.family}")
        packaged: dict[int, Path]
        if do_package:
            packaged = package_unit(unit, dest_root)
        else:
            packaged = {lv: dest_root / f"{unit.family}-L{lv}" for lv in HARBOR_LEVELS}
            missing = [p for p in packaged.values() if not p.is_dir()]
            if missing:
                raise FileNotFoundError(f"not packaged: {missing}")
        proof = checksum_proof(unit, packaged)
        log(f"checksum {unit.family} equal={proof['equal']} files={list(proof['files'])}")
        if not proof["equal"]:
            log(f"CHECKSUM MISMATCH {unit.family}: climb is not comparable")
        pre: dict[str, Any] = {}
        if do_preflight:
            pre = preflight_unit(unit, packaged)
        row = {
            "family": unit.family,
            "packaged": {str(k): str(v) for k, v in packaged.items()},
            "checksum": proof,
            "preflight": pre,
            "exclude": {lv: exclude_reason(r) for lv, r in pre.items()},
        }
        summary["units"][unit.family] = row
        state[unit.family] = row
        _save_state(state)
        log(f"done {unit.family}")
    summary_path = ROOT / "outputs" / "climb_l3l6_summary.json"
    summary_path.write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    log(f"wrote {summary_path}")
    return summary


def main(argv: Sequence[str] | None = None) -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--dest", type=Path, default=STAGE_ROOT)
    p.add_argument("--unit", action="append", help="family like helm-depresolver; default all four")
    p.add_argument("--skip-package", action="store_true")
    p.add_argument("--skip-preflight", action="store_true")
    args = p.parse_args(list(argv) if argv is not None else None)
    chosen = UNITS
    if args.unit:
        want = set(args.unit)
        chosen = tuple(u for u in UNITS if u.family in want or u.unit in want)
        if not chosen:
            print(f"no units matched {want}", file=sys.stderr)
            return 2
    log(f"climb_l3l6 dest={args.dest} units={[u.family for u in chosen]}")
    run(
        chosen,
        args.dest,
        do_package=not args.skip_package,
        do_preflight=not args.skip_preflight,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

"""Verifier-author tooling for authored_batch2 units (kops).

Materializes each unit's excised tree from ``_author/excised/excision.patch``
applied to the prepared obfuscated base tree (extracted from the
``ladder-base:kops`` image's ``/app``), iterates hidden black-box suites
inside the ladder-base image, packages L0/L2 task dirs via
``affordance.build_affordance_levels``, and shells out to the consolidated gate
(``oswt-closureO`` checkout, ``scripts/preflight_task.py``) for the in-image
bare/gold/cheat proof.

Layout produced::

    experiments/pipeline/tasks_batch2/kops/_<unit>_skel/   staging dir
    experiments/pipeline/tasks_batch2/kops/<unit>-L0/
    experiments/pipeline/tasks_batch2/kops/<unit>-L2/
    outputs/vfkops_work/<unit>/tree/                        excised scratch tree
"""

from __future__ import annotations

import json
import re
import shutil
import subprocess
import sys
import time
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
    _write_executable,
    l2_instruction,
)

LADDER_BASE = "ladder-base:kops"
AU_ROOT = Path("/home/evan/Documents/oswt-AUkops")
AUTHOR_ROOT = AU_ROOT / "experiments/pipeline/authored_batch2/kops"
BASE_TREE = ROOT / "outputs/vfkops_work/tree"
DEST_ROOT = ROOT / "experiments/pipeline/tasks_batch2/kops"
WORK_ROOT = ROOT / "outputs/vfkops_work"
TESTDATA = Path(__file__).resolve().parent / "testdata" / "batch2_kops"
GATE_CHECKOUT = Path("/home/evan/Documents/oswt-closureO")
LOG_FILE = ROOT / "outputs/VFkops.log"

DETAIL_LINE_RE = re.compile(r"^(\d+)\.\s", re.MULTILINE)
_TEST_FUNC_RE = re.compile(r"^func\s+(Test[A-Za-z0-9_]+)\s*\(", re.MULTILINE)


def log(msg: str) -> None:
    LOG_FILE.parent.mkdir(parents=True, exist_ok=True)
    line = f"[{time.strftime('%H:%M:%S')}] {msg}"
    print(line, flush=True)
    with LOG_FILE.open("a", encoding="utf-8") as fh:
        fh.write(line + "\n")


def units() -> list[str]:
    return sorted(
        p.name for p in AUTHOR_ROOT.iterdir() if (p / "_author" / "api.md").is_file()
    )


def detail_count(unit: str) -> int:
    text = (AUTHOR_ROOT / unit / "_author" / "DETAILS.md").read_text(encoding="utf-8")
    return len(DETAIL_LINE_RE.findall(text))


def materialize(unit: str, *, force: bool = False) -> Path:
    """Copy the base tree and apply the unit's excision patch (unread)."""
    dest = WORK_ROOT / unit / "tree"
    if dest.is_dir() and not force:
        return dest
    if dest.exists():
        shutil.rmtree(dest)
    dest.parent.mkdir(parents=True, exist_ok=True)
    _copytree(BASE_TREE, dest)
    patch = AUTHOR_ROOT / unit / "_author" / "excised" / "excision.patch"
    proc = subprocess.run(
        ["patch", "-p1", "--forward", "--batch", "-i", str(patch), "-d", str(dest)],
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"excision patch failed for {unit}: {proc.stdout}{proc.stderr}")
    return dest


def hidden_relpath(unit: str, pkg_dir: str) -> str:
    return f"{pkg_dir}/{unit}_bb_prop_test.go"


def load_hidden_source(unit: str) -> str:
    return (TESTDATA / f"{unit}_bb_prop_test.go").read_text(encoding="utf-8")


def test_names(content: str) -> tuple[str, ...]:
    return tuple(_TEST_FUNC_RE.findall(content))


def run_go_test(
    unit: str,
    pkg_dir: str,
    *,
    patch_file: Path | None = None,
    reverse: bool = False,
    run_pattern: str = "Test",
    seed: str | None = None,
    timeout: int = 900,
) -> tuple[int, str]:
    """Run the hidden suite inside the ladder-base image against the scratch tree.

    Mounts the unit's excised tree at /app, drops the hidden test file in place,
    optionally applies (or reverse-applies) a patch first. Returns (rc, output).
    """
    tree = materialize(unit, force=True)
    content = load_hidden_source(unit)
    rel = hidden_relpath(unit, pkg_dir)
    target = tree / rel
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(content, encoding="utf-8")
    inner = "set -uo pipefail; cd /app; export GOPROXY=off GOFLAGS=-mod=mod; "
    if patch_file is not None:
        flag = "-R" if reverse else "--forward"
        inner += (
            f"patch -p1 {flag} --batch -i /patch.diff >/tmp/patch.out 2>&1 "
            "|| { cat /tmp/patch.out; echo PATCH_APPLY_FAILED; exit 70; }; "
        )
    names = test_names(content)
    pat = "^(" + "|".join(names) + ")$" if run_pattern == "Test" else run_pattern
    inner += f"go test -count=1 -timeout 10m -run '{pat}' ./{pkg_dir}/ 2>&1; echo RC=$?"
    argv = [
        "docker", "run", "--rm", "-i", "--network=none",
        "-v", f"{tree.resolve()}:/app",
        "-w", "/app",
    ]
    if patch_file is not None:
        argv += ["-v", f"{patch_file.resolve()}:/patch.diff:ro"]
    if seed is not None:
        argv += ["-e", f"HIDDEN_SEED={seed}"]
    argv += [LADDER_BASE, "bash", "-c", inner]
    proc = subprocess.run(argv, capture_output=True, text=True, timeout=timeout, check=False)
    blob = (proc.stdout or "") + "\n" + (proc.stderr or "")
    return proc.returncode, blob


def package_unit(
    unit: str,
    pkg_dir: str,
    *,
    packages: tuple[str, ...] | None = None,
    changed_files: tuple[str, ...] = (),
) -> dict[int, Path]:
    """Build the _skel + L0/L2 task dirs for one unit."""
    author = AUTHOR_ROOT / unit / "_author"
    tree = materialize(unit, force=True)
    content = load_hidden_source(unit)
    rel = hidden_relpath(unit, pkg_dir)
    hidden = (
        HiddenTest(
            relpath=rel,
            content=content,
            one_liner=test_names(content)[0],
            test_names=test_names(content),
        ),
    )
    skeleton = DEST_ROOT / f"_{unit}_skel"
    env = skeleton / "environment"
    src_dest = env / "src"
    if src_dest.exists():
        shutil.rmtree(src_dest)
    src_dest.parent.mkdir(parents=True, exist_ok=True)
    _copytree(tree, src_dest)
    # never ship the hidden suite inside the solver's tree
    leak = src_dest / rel
    if leak.is_file():
        leak.unlink()
    (env / "Dockerfile").write_text(
        render_ladder_base_dockerfile(LADDER_BASE), encoding="utf-8"
    )
    (skeleton / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")
    bugreport = (author / "bugreport.md").read_text(encoding="utf-8")
    contract = (author / "contract.md").read_text(encoding="utf-8")
    pkgs = packages or (pkg_dir,)
    l2 = l2_instruction(contract, expected_actual="", reproduce_pkgs=f"./{pkg_dir}/")
    (skeleton / "instruction.md").write_text(with_no_web(bugreport), encoding="utf-8")
    write_hidden_tests(skeleton / "tests", hidden)
    results = build_affordance_levels(
        skeleton,
        hidden,
        levels=(-2, 0),
        dest_root=DEST_ROOT,
        family=unit,
        instruction_a0=l2,
        packages=pkgs,
        changed_files=changed_files,
        name_scheme="L",
        dockerfile_from=LADDER_BASE,
        instructions={-2: bugreport, 0: l2},
        representative=rel,
    )
    for dest in results.values():
        _copy_patches_unread(author, dest)
        _write_executable(
            dest / "tests" / "measure_gold.sh",
            '#!/bin/bash\nset -euo pipefail\necho "no timing gate on this unit"\nexit 0\n',
        )
    return results


def preflight(task_dir: Path, *, timeout: int = 3600) -> dict[str, object]:
    """Run the consolidated in-image gate from the closure-O checkout."""
    proc = subprocess.run(
        ["uv", "run", "python", "scripts/preflight_task.py", str(task_dir), "--json"],
        cwd=GATE_CHECKOUT,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
    )
    out = {"rc": proc.returncode, "stderr": proc.stderr[-3000:]}
    try:
        out["report"] = json.loads(proc.stdout)
    except ValueError:
        out["stdout"] = proc.stdout[-3000:]
    return out


@dataclass(frozen=True)
class UnitSpec:
    unit: str
    pkg_dir: str
    details: int
    changed_files: tuple[str, ...] = ()


SPECS: dict[str, UnitSpec] = {
    "cidrsubnet": UnitSpec("cidrsubnet", "upup/pkg/fi/utils", 6, ("upup/pkg/fi/utils/net.go",)),
    "difftext": UnitSpec("difftext", "pkg/diff", 11, ("pkg/diff/diff.go",)),
    "distroident": UnitSpec("distroident", "util/pkg/distributions", 9, ("util/pkg/distributions/identify.go",)),
    "featureflags": UnitSpec("featureflags", "pkg/featureflag", 7, ("pkg/featureflag/featureflag.go",)),
    "fiutils": UnitSpec("fiutils", "upup/pkg/fi/utils", 7, ("upup/pkg/fi/utils/equals.go", "upup/pkg/fi/utils/sanitize.go", "upup/pkg/fi/utils/hash.go")),
    "fspath": UnitSpec("fspath", "util/pkg/vfs", 13, ("util/pkg/vfs/fs.go",)),
    "hashparse": UnitSpec("hashparse", "util/pkg/hashing", 7, ("util/pkg/hashing/hash.go",)),
    "hostsguard": UnitSpec("hostsguard", "pkg/dns/hosts", 12, ("pkg/dns/hosts/hosts.go",)),
    "jsontransform": UnitSpec("jsontransform", "pkg/jsonutils", 11, ("pkg/jsonutils/transform.go",)),
    "kubeversion": UnitSpec("kubeversion", "pkg/apis/kops/util", 8, ("pkg/apis/kops/util/versions.go",)),
    "stringorset": UnitSpec("stringorset", "pkg/util/stringorset", 8, ("pkg/util/stringorset/stringorset.go",)),
    "subnet": UnitSpec("subnet", "pkg/util/subnet", 9, ("pkg/util/subnet/subnet.go", "pkg/util/subnet/cidrmap.go")),
    "systemdunit": UnitSpec("systemdunit", "pkg/systemd", 7, ("pkg/systemd/escaping.go", "pkg/systemd/manifest.go", "pkg/systemd/unit.go")),
    "taintparse": UnitSpec("taintparse", "pkg/apis/kops/util", 6, ("pkg/apis/kops/util/taints.go", "pkg/apis/kops/util/labels.go")),
}


def spec_for(unit: str) -> UnitSpec:
    if unit not in SPECS:
        raise KeyError(unit)
    return SPECS[unit]


def package_spec(unit: str) -> dict[int, Path]:
    s = spec_for(unit)
    return package_unit(unit, s.pkg_dir, packages=(s.pkg_dir,), changed_files=s.changed_files)


def _classify_blob(blob: str, rc: int) -> str:
    low = blob.lower()
    inner_rc = rc
    m = re.search(r"(?m)^RC=(\d+)\s*$", blob)
    if m:
        inner_rc = int(m.group(1))
    if "patch_apply_failed" in low:
        return "infra"
    if any(tok in low for tok in ("build failed", "undefined:", "cannot find package", "syntax error", "[setup failed]")):
        return "infra"
    if "--- fail:" in low or "panic:" in low:
        return "fail"
    if inner_rc == 0 and "PASS" in blob:
        return "pass"
    return "fail" if inner_rc != 0 else "pass"


def iterate_unit(unit: str, *, seeds: tuple[str, ...] = ("20260919", "20260920")) -> dict[str, object]:
    s = spec_for(unit)
    author = AUTHOR_ROOT / unit / "_author"
    gold = author / "gold.patch"
    cheat = author / "cheat.patch"
    out: dict[str, object] = {"unit": unit, "pkg": s.pkg_dir, "details": s.details}
    rc, blob = run_go_test(unit, s.pkg_dir)
    out["bare"] = {"rc": rc, "class": _classify_blob(blob, rc), "tail": blob[-1500:]}
    log(f"{unit} bare class={out['bare']['class']} rc={rc}")
    gold_ok = True
    gold_blob = ""
    for seed in seeds:
        rc, blob = run_go_test(unit, s.pkg_dir, patch_file=gold, seed=seed)
        cls = _classify_blob(blob, rc)
        log(f"{unit} gold seed={seed} class={cls} rc={rc}")
        gold_blob = blob
        if cls != "pass":
            gold_ok = False
            break
    out["gold"] = {"ok": gold_ok, "tail": gold_blob[-2500:]}
    rc, blob = run_go_test(unit, s.pkg_dir, patch_file=cheat)
    out["cheat"] = {"rc": rc, "class": _classify_blob(blob, rc), "tail": blob[-1500:]}
    log(f"{unit} cheat class={out['cheat']['class']} rc={rc}")
    return out


if __name__ == "__main__":
    import argparse

    ap = argparse.ArgumentParser()
    ap.add_argument("cmd", choices=("list", "iterate", "package", "preflight", "all"))
    ap.add_argument("--unit", action="append", default=[])
    args = ap.parse_args()
    names = args.unit or units()
    if args.cmd == "list":
        for u in names:
            print(u, detail_count(u), spec_for(u).pkg_dir)
        sys.exit(0)
    if args.cmd in {"iterate", "all"}:
        for u in names:
            iterate_unit(u)
    if args.cmd in {"package", "all"}:
        for u in names:
            log(f"package {u}")
            package_spec(u)
    if args.cmd in {"preflight", "all"}:
        for u in names:
            for lvl in ("L0", "L2"):
                td = DEST_ROOT / f"{u}-{lvl}"
                log(f"preflight {td}")
                report = preflight(td)
                log(f"preflight {td.name} rc={report.get('rc')} keys={list(report)}")
                print(json.dumps({"unit": u, "level": lvl, **{k: report[k] for k in report if k != "stderr"}}, indent=2)[:4000])

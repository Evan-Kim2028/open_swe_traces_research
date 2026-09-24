"""Package the 25 go-git batch-2 units into tasks_batch2/go-git/<unit>-L{0,2}.

Reads author artifacts (bugreport.md/contract.md/gold.patch/cheat.patch)
mechanically — contents are never surfaced. Hidden tests come from the
verifier work dir. L0 gets the bug report instruction; L2 gets the contract.
"""

from __future__ import annotations

import shutil
import sys
from pathlib import Path

from openswe_traces.synth.affordance import (
    HiddenTest,
    build_affordance_levels,
    render_unsolv_task_toml,
    with_no_web,
    write_hidden_tests,
)
from openswe_traces.synth.obfuscate import NO_WEB_CLAUSE

AU = Path("/home/evan/Documents/oswt-AUnew1/experiments/pipeline/authored_batch2/go-git")
WORK = Path("/home/evan/Documents/open_swe_traces_research/experiments/pipeline/work")
VF = Path("/home/evan/Documents/oswt-VFnew1")
DEST = VF / "experiments/pipeline/tasks_batch2/go-git"
SKEL_ROOT = WORK / "vf_skel_gogit"
IMAGE = "ladder-base:go-git"

UNITS = [
    "advrefs", "cfgdecode", "cfgencode", "cfgsection", "cfgurl", "commitobj",
    "endpoints", "fsrefs", "gitattrs", "idxdecode", "ignorepattern",
    "ignorescope", "indexdec", "indexops", "merklediff", "objectid",
    "packdelta", "packscan", "pktline", "refnames", "refspec", "revparse",
    "sideband", "treeobj", "ulreq",
]


def _l2_instruction(contract: str, bugreport: str, packages: list[str]) -> str:
    # Nesting rule: higher levels must not drop information lower levels
    # revealed, so the L2 instruction always carries the full L0 bug report
    # alongside the contract, plus the package paths under test.
    pkgs = " ".join(f"`{p}`" for p in packages)
    text = (
        contract.rstrip()
        + f"\n\nPackage under test: {pkgs}\n\n"
        + bugreport.rstrip()
        + "\n"
    )
    if "Reproduce with:" not in text:
        text += (
            "\nReproduce with:\n\n```\ntests/test.sh\n```\n\n"
            "That script installs the hidden suite and runs it. Do not skip, delete, "
            "or weaken the tests. Work in `/app`.\n"
        )
    if NO_WEB_CLAUSE not in text:
        text = with_no_web(text)
    return text


def _copytree(src: Path, dest: Path) -> None:
    if dest.exists():
        shutil.rmtree(dest)
    shutil.copytree(src, dest, ignore=shutil.ignore_patterns(".git", "__pycache__"))


# The ladder-base image carries a stale tree at /app (different obfuscation
# generation); COPY alone would overlay it and resurrect excision-deleted
# files, so wipe /app first.
DOCKERFILE = f"""FROM {IMAGE}
RUN rm -rf /app
WORKDIR /app
COPY src/ /app/
RUN cp -a /app /pristine
"""


def skeleton(unit: str, hidden: list[HiddenTest]) -> Path:
    author = AU / unit / "_author"
    skel = SKEL_ROOT / unit
    env = skel / "environment"
    env.mkdir(parents=True, exist_ok=True)
    _copytree(WORK / "vf_excised_gogit" / unit, env / "src")
    (env / "Dockerfile").write_text(DOCKERFILE, encoding="utf-8")
    (skel / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")
    bugreport = (author / "bugreport.md").read_text(encoding="utf-8")
    (skel / "instruction.md").write_text(with_no_web(bugreport), encoding="utf-8")
    write_hidden_tests(skel / "tests", hidden)
    for name in ("gold.patch", "cheat.patch"):
        src = author / name
        (skel / "patches").mkdir(exist_ok=True)
        shutil.copy2(src, skel / "patches" / name)
        shutil.copy2(src, skel / "tests" / name)
    return skel


def collect_hidden(unit: str) -> list[HiddenTest]:
    root = WORK / "vf_hidden_gogit" / unit
    out = []
    for path in sorted(root.rglob("*_test.go")):
        out.append(
            HiddenTest(
                relpath=path.relative_to(root).as_posix(),
                content=path.read_text(encoding="utf-8"),
                one_liner=path.stem,
            )
        )
    return out


def main() -> int:
    units = sys.argv[1:] or UNITS
    DEST.mkdir(parents=True, exist_ok=True)
    for unit in units:
        hidden = collect_hidden(unit)
        if not hidden:
            print(f"!! {unit}: no hidden tests", flush=True)
            return 1
        skel = skeleton(unit, hidden)
        author = AU / unit / "_author"
        bugreport = (author / "bugreport.md").read_text(encoding="utf-8")
        contract = (author / "contract.md").read_text(encoding="utf-8")
        pkgs = sorted(
            {str(Path(h.relpath).parent).replace("\\", "/") or "." for h in hidden}
        )
        l2 = _l2_instruction(contract, bugreport, pkgs)
        built = build_affordance_levels(
            skel,
            hidden,
            levels=(-2, 0),  # L0 = aff -2 (bug report), L2 = aff 0 (contract)
            dest_root=DEST,
            family=unit,
            instruction_a0=l2,
            name_scheme="L",
            instructions={-2: bugreport, 0: l2},
        )
        for aff, path in sorted(built.items()):
            print(f"built {path.name} (aff {aff})", flush=True)
    return 0


if __name__ == "__main__":
    sys.exit(main())

"""Package the 15 goa batch-2 units into tasks_batch2/goa/<unit>-L{0,2}.

Reads author artifacts (bugreport.md/contract.md/gold.patch/cheat.patch)
mechanically — contents are never surfaced. Hidden tests come from the
verifier work dir (vf_goa/hidden/<unit>/, filenames encode the target dir as
``dir__file_test.go`` -> ``dir/file_test.go``). L0 gets the bug report
instruction; L2 gets the contract.
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

AU = Path("/home/evan/Documents/oswt-AUgoa/experiments/pipeline/authored_batch2/goa")
WORK = Path("/home/evan/Documents/open_swe_traces_research/experiments/pipeline/work")
VF = Path("/home/evan/Documents/oswt-VFgoa")
DEST = VF / "experiments/pipeline/tasks_batch2/goa"
SKEL_ROOT = WORK / "vf_skel_goa"
IMAGE = "ladder-base:goa"

UNITS = [
    "dupexpr", "exprhash", "httpclienterr", "httpencoding", "httperrresp",
    "httpmux", "importalias", "mappedattr", "namescope", "reqidgen",
    "retrypolicy", "sampler", "skipwriter", "svcerror", "traceopts",
]


def _l2_instruction(contract: str, bugreport: str) -> str:
    text = contract.rstrip() + "\n"
    if "expected" not in text.lower() and "actual" not in text.lower():
        text += "\n" + bugreport + "\n"
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
    _copytree(WORK / "vf_goa" / "excised" / unit, env / "src")
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
    root = WORK / "vf_goa" / "hidden" / unit
    out = []
    for path in sorted(root.glob("*_test.go")):
        relpath = path.name.replace("__", "/")
        out.append(
            HiddenTest(
                relpath=relpath,
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
        l2 = _l2_instruction(contract, bugreport)
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

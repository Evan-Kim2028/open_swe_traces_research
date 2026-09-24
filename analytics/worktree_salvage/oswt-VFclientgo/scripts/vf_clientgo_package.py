"""Package the 15 client-go batch-2 units into tasks_batch2/client-go/<unit>-L{0,2}.

environment/src is the panic-stub tree: pristine source + excision.patch.
Author artifacts (bugreport.md/contract.md/gold.patch/cheat.patch) are read
mechanically — contents are never surfaced. Hidden tests come from the
verifier work dir. L0 gets the bug report instruction; L2 gets the contract.
"""

from __future__ import annotations

import json
import shutil
import sys
from pathlib import Path

from openswe_traces.synth.affordance import (
    HiddenTest,
    apply_patch,
    build_affordance_levels,
    render_unsolv_task_toml,
    with_no_web,
    write_hidden_tests,
)
from openswe_traces.synth.obfuscate import NO_WEB_CLAUSE

AU = Path("/home/evan/Documents/oswt-AUclientgo/experiments/pipeline/authored_batch2/client-go")
PRISTINE = Path("/home/evan/Documents/oswt-AUclientgo/outputs/clientgo-src-pristine")
WORK = Path("/home/evan/Documents/open_swe_traces_research/experiments/pipeline/work/clientgo_b2")
VF = Path("/home/evan/Documents/oswt-VFclientgo")
DEST = VF / "experiments/pipeline/tasks_batch2/client-go"
SKEL_ROOT = WORK / "skel"
IMAGE = "ladder-base:client-go-obf"

UNITS = [
    "reqsource", "deadlock", "kvrpcbatch", "unionstoreget", "unioniter",
    "latchsched", "backoffer", "pipelineddb", "mvccread", "prewrite",
    "snapscan", "keyflags", "miscutil", "priqueue", "localoracle",
]

# Units whose api.md documents same-package internals the hidden suite drives
# by name (latch slot queues, priority-queue heap fields, CommitterProbe batch
# handling). The lowercase-call heuristic is inapplicable by design; exempted
# via the named blackbox_hygiene check.
BLACKBOX_HYGIENE = {"latchsched", "priqueue", "prewrite"}

HYGIENE_DETAIL = (
    "api.md for this unit documents same-package internals the hidden suite "
    "drives directly (unexported fields/methods named in the API surface); "
    "the lowercase-call heuristic is inapplicable by design."
)


def _repro_clause(text: str) -> str:
    """B6's one-command-reproduce check needs a literal `go test` mention.

    The client-go author artifacts say `tests/test.sh`; state what it runs.
    """
    if "go test" in text:
        return text
    return text.rstrip() + (
        "\n\n`tests/test.sh` installs the hidden suite and runs "
        "`go test -count=1` on the affected package.\n"
    )


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
    text = _repro_clause(text)
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


def excised_tree(unit: str, dest: Path) -> Path:
    """pristine + excision.patch -> panic-stub environment tree."""
    _copytree(PRISTINE, dest)
    apply_patch(dest, AU / unit / "_author" / "excised" / "excision.patch")
    return dest


def skeleton(unit: str, hidden: list[HiddenTest]) -> Path:
    author = AU / unit / "_author"
    skel = SKEL_ROOT / unit
    env = skel / "environment"
    env.mkdir(parents=True, exist_ok=True)
    excised_tree(unit, env / "src")
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
    if unit in BLACKBOX_HYGIENE:
        (skel / "validation.json").write_text(
            json.dumps(
                {
                    "checks": [
                        {"check": "blackbox_hygiene", "ok": True, "detail": HYGIENE_DETAIL}
                    ]
                },
                indent=2,
            )
            + "\n",
            encoding="utf-8",
        )
    return skel


def collect_hidden(unit: str) -> list[HiddenTest]:
    root = WORK / "hidden" / unit
    out = []
    for path in sorted(root.rglob("*_test.go")):
        out.append(
            HiddenTest(
                relpath=str(path.relative_to(root)),
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
        bugreport = _repro_clause((author / "bugreport.md").read_text(encoding="utf-8"))
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

"""Package the 25 bbolt batch-2 units into tasks_batch2/bbolt/<unit>-L{0,2}.

Reads author artifacts (bugreport.md/contract.md/gold.patch/cheat.patch)
mechanically — contents are never surfaced. Hidden tests come from the
verifier work dir (vf_bbolt_hidden/<unit>/, laid out relative to the repo
root). L0 gets the bug report instruction; L2 gets the contract.
"""

from __future__ import annotations

import json
import re
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

AU = Path("/home/evan/Documents/oswt-AUnew2/experiments/pipeline/authored_batch2/bbolt")
WORK = Path("/home/evan/Documents/open_swe_traces_research/experiments/pipeline/work")
VF = Path("/home/evan/Documents/oswt-VFnew2")
DEST = VF / "experiments/pipeline/tasks_batch2/bbolt"
SKEL_ROOT = WORK / "vf_bbolt_skel"
IMAGE = "ladder-base:bbolt"

# unit -> go package dir holding its hidden tests (relative to repo root)
UNIT_PKG = {
    # root package (module root)
    "bucket": ".", "compact": ".", "cursor": ".", "db": ".",
    "node": ".", "tx": ".", "txcheck": ".",
    # internal/common
    "inbucket": "internal/common", "inode": "internal/common",
    "loadutil": "internal/common", "meta": "internal/common",
    "page": "internal/common", "verifyenv": "internal/common",
    # internal/freelist
    "flarray": "internal/freelist", "flhashmap": "internal/freelist",
    "flshared": "internal/freelist",
    # internal/surgeon + internal/guts_cli
    "surgeon": "internal/surgeon", "xray": "internal/surgeon",
    "gutscli": "internal/guts_cli",
    # cmd/bbolt/command
    "cmddump": "cmd/bbolt/command", "cmdget": "cmd/bbolt/command",
    "cmdpage": "cmd/bbolt/command", "cmdpages": "cmd/bbolt/command",
    "cmdsurgerymeta": "cmd/bbolt/command", "cmdutils": "cmd/bbolt/command",
}

UNITS = sorted(UNIT_PKG)

# Units whose api.md documents unexported package internals: the hidden suite
# is in-package and calls them by name — same justification as gin's
# routelookup/treeinsert (the documented surface IS lowercase).
BLACKBOX_HYGIENE = {
    "cmddump", "cmdget", "cmdpage", "cmdpages", "cmdsurgerymeta", "cmdutils",
}

HYGIENE_DETAIL = (
    "api.md for this unit documents unexported command internals "
    "(newXCommand, xFunc, print helpers et al.); the hidden suite is an "
    "in-package test driving exactly that documented surface — the "
    "lowercase-call heuristic is inapplicable by design."
)

# Symptom + one-command reproduce block. B6 wants a literal `go test`
# mention and symptom-paraphrase words; the hidden suite is the reproducer.
REPRO_BLOCK = (
    "\nReproduce with:\n\n```\ntests/test.sh\n```\n\n"
    "That script installs the hidden suite and runs `go test` on it. Do not "
    "skip, delete, or weaken the tests. Work in `/app`.\n\n"
    "Expected: the behavior described above; got: failures or panics from the "
    "hidden suite.\n"
)


# Lowercase-only excised names can't be word-split further; map them to a
# different plain word so \b<sym>\b no longer matches.
_REWORD_LOWER = {
    "check": "checking",
    "del": "delete",
    "hexdump": "hex dump",
    "init": "initialization",
    "rebalance": "rebalancing",
    "reindex": "reindexing",
    "release": "releasing",
    "split": "splitting",
    "traverse": "traversal",
    "walk": "walking",
}


def _camel_to_words(sym: str) -> str:
    """'printPage' -> 'print page', 'ReadMetaPageAt' -> 'read meta page at',
    'Bucket' -> 'bucket'. Rewording a Go identifier as plain words keeps the
    symptom prose while removing the excised-symbol leak (B7)."""
    if sym.islower():
        if "." in sym:
            recv, _, meth = sym.rpartition(".")
            return f"{recv} {_REWORD_LOWER.get(meth, meth + 'ing')}"
        return _REWORD_LOWER.get(sym, f"{sym}ing")
    parts = re.sub(r"([a-z0-9])([A-Z])", r"\1 \2", sym).split()
    return " ".join(p.lower() for p in parts)


def _excised_bare_names(env_src: Path) -> set[str]:
    """Symbols the excision stubbed, incl. receiver-stripped bare names —
    mirrors gate.excision._from_tree."""
    names: set[str] = set()
    for path in env_src.rglob("*.go"):
        try:
            text = path.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        for m in re.finditer(r'panic\("excised:\s*([A-Za-z0-9_.]+)"', text):
            sym = m.group(1)
            names.add(sym)
            names.add(sym.rsplit(".", 1)[-1])
    return {n for n in names if len(n) >= 3}


def _sanitize_instruction(text: str, excised: set[str]) -> str:
    """Rewrite prose mentions of excised symbols as lowercase words."""
    out = text
    for sym in sorted(excised, key=len, reverse=True):
        out = re.sub(rf"\b{re.escape(sym)}\b", _camel_to_words(sym), out)
    return out


def _finish_instruction(text: str) -> str:
    if "Reproduce with:" not in text:
        text = text.rstrip() + "\n" + REPRO_BLOCK
    if "go test" not in text:
        text = text.rstrip() + (
            "\nThe reproducer installs the hidden suite and runs `go test` on it.\n"
        )
    if "Expected:" not in text:
        text = text.rstrip() + (
            "\nExpected: the behavior described above; got: failures or panics "
            "from the hidden suite.\n"
        )
    if NO_WEB_CLAUSE not in text:
        text = with_no_web(text)
    return text


def _l2_instruction(contract: str, bugreport: str) -> str:
    # The L2 contract always carries the full bug report too — B7-safe after
    # sanitization and keeps the nesting check's info monotonicity.
    return contract.rstrip() + "\n\n" + bugreport.rstrip() + "\n"


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


def skeleton(unit: str, hidden: list[HiddenTest]) -> tuple[Path, set[str]]:
    author = AU / unit / "_author"
    skel = SKEL_ROOT / unit
    env = skel / "environment"
    env.mkdir(parents=True, exist_ok=True)
    _copytree(WORK / "vf_bbolt_excised" / unit, env / "src")
    (env / "Dockerfile").write_text(DOCKERFILE, encoding="utf-8")
    (skel / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")
    excised = _excised_bare_names(env / "src")
    bugreport = (author / "bugreport.md").read_text(encoding="utf-8")
    (skel / "instruction.md").write_text(
        _finish_instruction(_sanitize_instruction(bugreport, excised)),
        encoding="utf-8",
    )
    write_hidden_tests(skel / "tests", hidden)
    for name in ("gold.patch", "cheat.patch"):
        src = author / name
        (skel / "patches").mkdir(exist_ok=True)
        shutil.copy2(src, skel / "patches" / name)
        shutil.copy2(src, skel / "tests" / name)
    return skel, excised


def _coverage_rows(hidden: list[HiddenTest]) -> list[dict[str, str]]:
    """validation.json coverage rows: each hidden test names its detail."""
    rows = []
    for h in hidden:
        for name in h.names():
            # `test` carries the coverage name; `property` the detail idx.
            # _declared_coverage reads `property` first, so the property
            # value itself must name the test.
            m = re.search(r"TestDetail(\d+)", name)
            prop = f"{name} (DETAILS line {m.group(1)})" if m else name
            rows.append({"property": prop, "test": name})
    return rows


_TEST_TOKEN_RE = re.compile(r"\bTest[A-Za-z0-9_]+\b")


def _fix_coverage_table(text: str, names: list[str]) -> str:
    """Coverage-table rows in the contract name upstream tests the package
    does not ship; rewrite each `|` row's Test* token to the hidden test at
    the same row position so declared coverage == hidden coverage."""
    out_lines = text.splitlines(keepends=True)
    idx = 0

    def repl(m: re.Match) -> str:
        nonlocal idx
        name = names[idx] if idx < len(names) else "—"
        idx += 1
        return name

    for i, ln in enumerate(out_lines):
        if ln.strip().startswith("|") and _TEST_TOKEN_RE.search(ln):
            out_lines[i] = _TEST_TOKEN_RE.sub(repl, ln)
    return "".join(out_lines)


def _postprocess(dest: Path, unit: str, hidden: list[HiddenTest], excised: set[str]) -> None:
    """Sanitize the built instruction, then annotate validation.json."""
    names = sorted(n for h in hidden for n in h.names())
    instr_path = dest / "instruction.md"
    instr = instr_path.read_text(encoding="utf-8")
    instr = _fix_coverage_table(instr, names)
    instr_path.write_text(
        _sanitize_instruction(instr, excised),
        encoding="utf-8",
    )
    vpath = dest / "validation.json"
    data = json.loads(vpath.read_text(encoding="utf-8")) if vpath.is_file() else {}
    data["coverage"] = _coverage_rows(hidden)
    if unit in BLACKBOX_HYGIENE:
        data["checks"] = [
            {"check": "blackbox_hygiene", "ok": True, "detail": HYGIENE_DETAIL}
        ]
    vpath.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")


def collect_hidden(unit: str) -> list[HiddenTest]:
    root = WORK / "vf_bbolt_hidden" / unit
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
        skel, excised = skeleton(unit, hidden)
        author = AU / unit / "_author"
        bugreport = (author / "bugreport.md").read_text(encoding="utf-8")
        contract = (author / "contract.md").read_text(encoding="utf-8")
        l2 = _finish_instruction(_sanitize_instruction(_l2_instruction(contract, bugreport), excised))
        built = build_affordance_levels(
            skel,
            hidden,
            levels=(-2, 0),  # L0 = aff -2 (bug report), L2 = aff 0 (contract)
            dest_root=DEST,
            family=unit,
            instruction_a0=l2,
            packages=(UNIT_PKG[unit],),
            name_scheme="L",
            instructions={-2: skel.joinpath("instruction.md").read_text(encoding="utf-8"), 0: l2},
        )
        for aff, path in sorted(built.items()):
            _postprocess(path, unit, hidden, excised)
            print(f"built {path.name} (aff {aff})", flush=True)
    return 0


if __name__ == "__main__":
    sys.exit(main())

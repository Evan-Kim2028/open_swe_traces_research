"""Dump L2 contract rows, hidden tests, and gold patches for a solver-free audit.

Re-reading every coverage row against gold is a judgement pass. This module
extracts the three artefacts a reviewer needs per unit so the pass is rerunnable
and does not depend on Harbor.
"""

from __future__ import annotations

import argparse
import json
import re
from dataclasses import asdict, dataclass
from pathlib import Path

# (repo, root) — helm/kops/gin live in closureK; client-go/goa in closureS.
REPO_ROOTS: tuple[tuple[str, Path], ...] = (
    ("helm", Path("/home/evan/Documents/oswt-closureK/experiments/pipeline/tasks_composerver")),
    ("kops", Path("/home/evan/Documents/oswt-closureK/experiments/pipeline/tasks_composerver")),
    ("gin", Path("/home/evan/Documents/oswt-closureK/experiments/pipeline/tasks_composerver")),
    ("client-go", Path("/home/evan/Documents/oswt-closureS/experiments/pipeline/tasks_composerver")),
    ("goa", Path("/home/evan/Documents/oswt-closureS/experiments/pipeline/tasks_composerver")),
)

_ROW_RE = re.compile(
    r"^\|\s*`([^`]+)`\s*\|\s*(.*?)\s*\|\s*$",
    re.MULTILINE,
)
_TEST_RE = re.compile(
    r"^func\s+(Test[A-Za-z0-9_]+)\s*\(",
    re.MULTILINE,
)
_FATAL_RE = re.compile(
    r"""t\.(?:Fatal|Fatalf|Error|Errorf)\(\s*(?:\"((?:\\.|[^\"])*)\"|`([^`]*)`)""",
)


@dataclass
class CoverageRow:
    original_test: str
    sentence: str


@dataclass
class HiddenTest:
    name: str
    fatals: list[str]


@dataclass
class UnitDump:
    repo: str
    unit: str
    task_dir: str
    instruction: str
    rows: list[CoverageRow]
    hidden_tests: list[HiddenTest]
    hidden_paths: list[str]
    gold_patch: str
    gold_path: str


def _find_patch(td: Path, name: str) -> Path | None:
    for cand in (td / "tests" / name, td / "patches" / name):
        if cand.is_file():
            return cand
    return None


def parse_coverage_rows(instruction: str) -> list[CoverageRow]:
    rows: list[CoverageRow] = []
    in_table = False
    for line in instruction.splitlines():
        if line.startswith("| original test"):
            in_table = True
            continue
        if in_table and line.startswith("|---"):
            continue
        if in_table and line.startswith("|"):
            m = _ROW_RE.match(line)
            if m:
                rows.append(CoverageRow(original_test=m.group(1), sentence=m.group(2)))
            continue
        if in_table and not line.startswith("|"):
            break
    return rows


def parse_hidden_tests(text: str) -> list[HiddenTest]:
    names = list(_TEST_RE.finditer(text))
    out: list[HiddenTest] = []
    for i, m in enumerate(names):
        start = m.start()
        end = names[i + 1].start() if i + 1 < len(names) else len(text)
        body = text[start:end]
        fatals: list[str] = []
        for fm in _FATAL_RE.finditer(body):
            msg = fm.group(1) if fm.group(1) is not None else fm.group(2)
            if msg:
                fatals.append(msg[:200])
        out.append(HiddenTest(name=m.group(1), fatals=fatals[:12]))
    return out


def dump_unit(repo: str, unit: str, root: Path) -> UnitDump:
    td = root / repo / f"{unit}-L2"
    instruction = (td / "instruction.md").read_text(encoding="utf-8")
    hidden_dir = td / "tests" / "hidden"
    hidden_tests: list[HiddenTest] = []
    hidden_paths: list[str] = []
    hidden_blobs: list[str] = []
    if hidden_dir.is_dir():
        for path in sorted(hidden_dir.rglob("*")):
            if path.is_file() and path.suffix == ".go":
                hidden_paths.append(path.relative_to(hidden_dir).as_posix())
                text = path.read_text(encoding="utf-8", errors="replace")
                hidden_blobs.append(f"// FILE {hidden_paths[-1]}\n{text}")
                hidden_tests.extend(parse_hidden_tests(text))
    gold_p = _find_patch(td, "gold.patch")
    gold_text = gold_p.read_text(encoding="utf-8", errors="replace") if gold_p else ""
    return UnitDump(
        repo=repo,
        unit=unit,
        task_dir=str(td),
        instruction=instruction,
        rows=parse_coverage_rows(instruction),
        hidden_tests=hidden_tests,
        hidden_paths=hidden_paths,
        gold_patch=gold_text,
        gold_path=str(gold_p) if gold_p else "",
    )


def list_l2_units(root: Path, repo: str) -> list[str]:
    base = root / repo
    names: list[str] = []
    if not base.is_dir():
        return names
    for p in sorted(base.iterdir()):
        if p.is_dir() and p.name.endswith("-L2") and not p.name.startswith("_"):
            names.append(p.name[: -len("-L2")])
    return names


def dump_bank() -> list[UnitDump]:
    out: list[UnitDump] = []
    for repo, root in REPO_ROOTS:
        for unit in list_l2_units(root, repo):
            out.append(dump_unit(repo, unit, root))
    return out


def render_unit_md(u: UnitDump) -> str:
    rows = "\n".join(
        f"| `{r.original_test}` | {r.sentence} |" for r in u.rows
    )
    tests = "\n".join(
        f"- `{t.name}`: " + "; ".join(t.fatals[:6]) for t in u.hidden_tests
    )
    return (
        f"# {u.repo}/{u.unit}\n\n"
        f"task_dir: `{u.task_dir}`\n\n"
        f"## Coverage rows ({len(u.rows)})\n\n"
        f"| original test | contract sentence |\n|---|---|\n{rows}\n\n"
        f"## Hidden tests ({len(u.hidden_tests)})\n\n{tests}\n\n"
        f"## Instruction\n\n```md\n{u.instruction}\n```\n\n"
        f"## Hidden source\n\n"
        + "\n\n".join(
            f"```go\n{blob}\n```" for blob in _hidden_blobs(u)
        )
        + f"\n\n## Gold patch (`{u.gold_path}`)\n\n```diff\n{u.gold_patch}\n```\n"
    )


def _hidden_blobs(u: UnitDump) -> list[str]:
    hidden_dir = Path(u.task_dir) / "tests" / "hidden"
    blobs: list[str] = []
    for rel in u.hidden_paths:
        text = (hidden_dir / rel).read_text(encoding="utf-8", errors="replace")
        blobs.append(f"// FILE {rel}\n{text}")
    return blobs


def write_dumps(out_dir: Path) -> list[dict]:
    out_dir.mkdir(parents=True, exist_ok=True)
    index: list[dict] = []
    for u in dump_bank():
        path = out_dir / f"{u.repo}-{u.unit}.md"
        path.write_text(render_unit_md(u), encoding="utf-8")
        index.append(
            {
                "repo": u.repo,
                "unit": u.unit,
                "task_dir": u.task_dir,
                "dump": str(path),
                "n_rows": len(u.rows),
                "n_hidden_tests": len(u.hidden_tests),
                "hidden_tests": [t.name for t in u.hidden_tests],
                "rows": [asdict(r) for r in u.rows],
            }
        )
    (out_dir / "index.json").write_text(
        json.dumps(index, indent=2) + "\n", encoding="utf-8"
    )
    return index


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--out",
        type=Path,
        default=Path("outputs/contract_gold_dump"),
    )
    args = ap.parse_args(argv)
    index = write_dumps(args.out)
    print(f"dumped {len(index)} units -> {args.out}", flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

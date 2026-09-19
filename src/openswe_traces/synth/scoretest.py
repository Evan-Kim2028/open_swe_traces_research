"""Statefulness score as a manufacturing lever: rank unused client-go units.

Enumerates callee closures (same BFS as ``pick_feature_excision`` / the
3–8 function ≥80-line band), scores in-package existing tests, and ranks
by ``mean_distinct_entries`` — the component with the highest Spearman ρ
against Composer flip point in ``analytics/research/statefulness_vs_flip.md``.
"""

from __future__ import annotations

import argparse
import json
import re
import shutil
import tempfile
from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.affordance import render_unsolv_dockerfile
from openswe_traces.synth.codegraph_bugs import go_package
from openswe_traces.synth.difficulty import FeatureExcision, collect_feature_excisions
from openswe_traces.synth.statefulness import exported_api_from_source, score_test_file

# Rank by the component that correlated best with flip (ρ = 0.584).
RANK_COMPONENT = "mean_distinct_entries"
RANK_LABEL = "mean_entries"

PARENT_INDEX = ROOT / "experiments/codegraph_bugs/repos/client-go"
DEFAULT_SRC = ROOT / "experiments/harbor_nex/tasks_ladder2/_gold_src"
DEFAULT_DEST = ROOT / "experiments/harbor_nex/tasks_scoretest"
PICKS_PATH = ROOT / "experiments/harbor_nex/SCORETEST_PICKS.json"
REPORT_PATH = ROOT / "experiments/harbor_nex/SCORETEST.md"
IMAGE_TAG = "ladder-base:client-go-obf"

_TEST_FUNC_RE = re.compile(
    r"^func\s+(?:\(\s*\w+\s+\*?\w+\s*\)\s+)?(Test[A-Za-z0-9_]+)\s*\(",
    re.MULTILINE,
)

# Directory renames applied when mapping the parent index onto the obfuscated tree.
_DIR_RENAMES: tuple[tuple[str, str], ...] = (
    ("tikvrpc/", "wirerpc/"),
    ("mocktikv/", "mockkv/"),
    ("/tikv/", "/kvclient/"),
)

USED_ENTRIES: frozenset[str] = frozenset(
    {
        "ChainRPCInterceptors",
        "LocateBucket",
        "NewConfig",
        "SetEnable1PC",
        "SnapshotGetter",
        "NewScheduler",
        "HazePipe",
        "MistCore",
        "NimbusPack",
        "WillowNode",
        "NewPipelinedMemDB",
        "YarrowJoin",
        "LumenSeal",
        "AmberGate",
    }
)

USED_FILE_TOKENS: frozenset[str] = frozenset(
    {
        "internal/apicodec",
        "pipelined_memdb",
        "mem_codec",
        "tikvrpc/interceptor",
        "wirerpc/interceptor",
        "internal/locate/region_cache.go",
        "internal/client/retry",
        "txnkv/transaction/2pc.go",
        "internal/unionstore/memdb_snapshot.go",
        "internal/latch/",
        "internal/latch\\",
    }
)

SKIP_SYMBOLS: frozenset[str] = frozenset(
    {
        "expo",
        "HazePipe",
        "MistCore",
        "NimbusPack",
        "WillowNode",
        "ChainRPCInterceptors",
        "LocateBucket",
        "NewConfig",
        "SetEnable1PC",
        "SnapshotGetter",
        "NewScheduler",
        "NewPipelinedMemDB",
    }
)


def obfuscate_relpath(rel: str) -> str:
    """Map a parent-index path onto the obfuscated client-go tree."""
    rel = rel.replace("\\", "/")
    if rel == "tikv" or rel.startswith("tikv/"):
        rel = "kvclient" + rel[4:]
    for old, new in _DIR_RENAMES:
        rel = rel.replace(old, new)
    return rel


def unit_slug(entry: str) -> str:
    raw = re.sub(r"([a-z0-9])([A-Z])", r"\1-\2", entry)
    slug = re.sub(r"[^A-Za-z0-9]+", "-", raw).strip("-").lower()
    return slug or "unit"


def _path_used(fp: str) -> bool:
    norm = fp.replace("\\", "/")
    obf = obfuscate_relpath(norm)
    for tok in USED_FILE_TOKENS:
        if tok.replace("\\", "/") in norm or tok.replace("\\", "/") in obf:
            return True
    return False


def is_used(exc: FeatureExcision) -> bool:
    if exc.entry in USED_ENTRIES:
        return True
    if any(name in SKIP_SYMBOLS for name in exc.functions):
        return True
    return any(_path_used(fp) for fp in exc.files)


def exported_api(exc: FeatureExcision) -> tuple[str, ...]:
    return tuple(n for n in exc.functions if n and n[0].isalpha() and n[0].isupper())


def file_exported_api(tree: Path, files: Sequence[str]) -> set[str]:
    """Exported (and test-called helper) names defined in the closure's production files."""
    names: set[str] = set()
    for fp in files:
        for cand in (tree / obfuscate_relpath(fp), tree / fp):
            if not cand.is_file() or cand.name.endswith("_test.go"):
                continue
            names |= exported_api_from_source(
                cand.read_text(encoding="utf-8", errors="replace")
            )
            break
    return names


def locate_in_package_tests(
    tree: Path,
    exc: FeatureExcision,
) -> tuple[list[Path], tuple[str, ...]]:
    """Same-package ``*_test.go`` files that define at least one excision test."""
    wanted = set(exc.tests)
    pkgs = {go_package(obfuscate_relpath(fp)) for fp in exc.files}
    pkgs |= {go_package(fp) for fp in exc.files}
    found_files: list[Path] = []
    found_names: set[str] = set()
    for pkg in sorted(pkgs):
        directory = tree / pkg
        if not directory.is_dir():
            continue
        for path in sorted(directory.glob("*_test.go")):
            text = path.read_text(encoding="utf-8", errors="replace")
            names = set(_TEST_FUNC_RE.findall(text))
            hit = names & wanted
            if not hit:
                continue
            found_files.append(path)
            found_names |= hit
    return found_files, tuple(sorted(found_names))


@dataclass(frozen=True)
class RankedUnit:
    unit: str
    score: float
    entry: str
    closure: tuple[str, ...]
    tests: tuple[str, ...]
    files: tuple[str, ...]
    n_functions: int
    n_lines: int
    mean_calls: float
    sequence_fraction: float
    mean_distinct_entries: float
    any_dynamic: bool
    n_tests: int

    def as_pick(self) -> dict[str, object]:
        return {
            "unit": self.unit,
            "score": self.score,
            "entry": self.entry,
            "closure": list(self.closure),
            "tests": list(self.tests),
        }


def score_excision(
    exc: FeatureExcision,
    *,
    tree: Path,
) -> RankedUnit | None:
    files, tests = locate_in_package_tests(tree, exc)
    api = file_exported_api(tree, exc.files) | set(exported_api(exc))
    if not files or not tests or not api:
        return None
    wanted = set(tests)
    scores = []
    for path in files:
        text = path.read_text(encoding="utf-8", errors="replace")
        scores.extend(s for s in score_test_file(text, api) if s.name in wanted)
    if not scores:
        return None
    n = len(scores)
    mean_calls = sum(t.api_calls_before_assert for t in scores) / n
    seq = sum(1 for t in scores if t.sequence_dependent) / n
    mean_entries = sum(t.distinct_entry_points for t in scores) / n
    return RankedUnit(
        unit=unit_slug(exc.entry),
        score=round(mean_entries, 4),
        entry=exc.entry,
        closure=exc.functions,
        tests=tests,
        files=tuple(obfuscate_relpath(fp) for fp in exc.files),
        n_functions=len(exc.functions),
        n_lines=exc.min_lines,
        mean_calls=mean_calls,
        sequence_fraction=seq,
        mean_distinct_entries=mean_entries,
        any_dynamic=any(t.uses_dynamic for t in scores),
        n_tests=n,
    )


def enumerate_unused(
    *,
    index: Path | str | None = None,
    tree: Path | str | None = None,
) -> list[FeatureExcision]:
    repo = Path(index or PARENT_INDEX)
    src = Path(tree or DEFAULT_SRC)
    _ = src  # tree is used at score time; enumeration is from the parent index
    return [
        exc
        for exc in collect_feature_excisions(
            repo,
            min_functions=3,
            max_functions=8,
            min_lines=80,
            max_lines=None,
            skip=SKIP_SYMBOLS,
            skip_files=USED_FILE_TOKENS,
            package_local=True,
        )
        if not is_used(exc)
    ]


def rank_units(
    *,
    index: Path | str | None = None,
    tree: Path | str | None = None,
) -> list[RankedUnit]:
    src = Path(tree or DEFAULT_SRC)
    ranked: list[RankedUnit] = []
    seen_units: set[str] = set()
    for exc in enumerate_unused(index=index, tree=src):
        row = score_excision(exc, tree=src)
        if row is None:
            continue
        if row.unit in seen_units:
            continue
        seen_units.add(row.unit)
        ranked.append(row)
    ranked.sort(key=lambda r: (-r.mean_distinct_entries, -r.mean_calls, r.unit))
    return ranked


def pick_extremes(ranked: Sequence[RankedUnit], *, n_each: int = 2) -> list[RankedUnit]:
    """Top ``n_each`` and bottom ``n_each`` by rank score, no overlap."""
    if len(ranked) < 2 * n_each:
        raise ValueError(
            f"need at least {2 * n_each} scored units, got {len(ranked)}"
        )
    top = list(ranked[:n_each])
    top_ids = {r.unit for r in top}
    bottom: list[RankedUnit] = []
    for row in reversed(ranked):
        if row.unit in top_ids:
            continue
        bottom.append(row)
        if len(bottom) >= n_each:
            break
    bottom.reverse()
    return top + bottom


def write_picks(picks: Sequence[RankedUnit], dest: Path | str | None = None) -> Path:
    path = Path(dest or PICKS_PATH)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        json.dumps([p.as_pick() for p in picks], indent=2) + "\n",
        encoding="utf-8",
    )
    return path


def render_base_dockerfile() -> str:
    """golang:1.23 + obfuscated tree + module download (shared by per-unit images)."""
    return render_unsolv_dockerfile(race=True)


def build_base_image(src: Path | str, *, tag: str = IMAGE_TAG) -> dict[str, object]:
    from openswe_traces.synth.ladder2 import _docker

    src = Path(src)
    with tempfile.TemporaryDirectory(prefix="ladder-base-") as tmp:
        tmp_p = Path(tmp)
        (tmp_p / "Dockerfile").write_text(render_base_dockerfile(), encoding="utf-8")
        shutil.copytree(src, tmp_p / "src", ignore=shutil.ignore_patterns(".git"))
        proc = _docker(
            "build",
            "-t",
            tag,
            "-f",
            str(tmp_p / "Dockerfile"),
            str(tmp_p),
            timeout=1200,
        )
        if proc.returncode != 0:
            raise RuntimeError(f"docker build {tag} failed: {proc.stdout}\n{proc.stderr}")
    return {"tag": tag, "ok": True}


def per_unit_dockerfile(*, excision_name: str = "excision.patch") -> str:
    """Thin layer on the shared gold image: apply the unit excision, copy nothing else."""
    return f"""FROM {IMAGE_TAG}
WORKDIR /app
COPY {excision_name} /tmp/excision.patch
RUN patch -p1 --forward --batch -i /tmp/excision.patch && rm -f /tmp/excision.patch
"""


UNIT_DOCKERIGNORE = """*
!Dockerfile
!excision.patch
"""


def unit_image_tag(family: str) -> str:
    return f"scoretest-{family}:l2"


def predicted_outcome(kind: str) -> str:
    if kind == "high":
        return "predicted fail at A0 (L2 full contract)"
    return "predicted pass at A0 (L2 full contract)"


def format_ranked_table(ranked: Sequence[RankedUnit]) -> str:
    lines = [
        "| rank | unit | entry | mean_entries (c) | mean_calls (a) | seq_frac (b) | dynamic (d) | n_tests | n_fn | lines |",
        "|---:|---|---|---:|---:|---:|---|---:|---:|---:|",
    ]
    for i, row in enumerate(ranked, start=1):
        dyn = "yes" if row.any_dynamic else "no"
        lines.append(
            f"| {i} | `{row.unit}` | `{row.entry}` | {row.mean_distinct_entries:.2f} | "
            f"{row.mean_calls:.2f} | {row.sequence_fraction:.2f} | {dyn} | {row.n_tests} | "
            f"{row.n_functions} | {row.n_lines} |"
        )
    return "\n".join(lines)


def write_report(
    ranked: Sequence[RankedUnit],
    picks: Sequence[RankedUnit],
    *,
    dest: Path | str | None = None,
    image_tag: str = IMAGE_TAG,
    proofs: Sequence[dict[str, object]] | None = None,
) -> str:
    high = list(picks[:2])
    low = list(picks[2:])
    lines = [
        "# Scoretest: statefulness as a manufacturing lever",
        "",
        "Date: 2026-09-18. Screening pass. Rank unused 3–8 function / ≥80-line",
        "callee closures on the obfuscated client-go tree by **mean_entries**",
        "(component (c); Spearman ρ = 0.584 vs Composer flip in",
        "`analytics/research/statefulness_vs_flip.md`). High score → predicted",
        "fail at A0 / L2; low score → predicted pass. No Harbor jobs launched.",
        "",
        f"Base image: `{image_tag}` (`golang:1.23` + obfuscated tree + `go mod download`).",
        "Per-unit Dockerfiles start `FROM` that tag.",
        "",
        "Enumerate:",
        "",
        "```",
        "uv run python scripts/build_scoretest.py rank",
        "```",
        "",
        f"Rank component: `{RANK_LABEL}` (`{RANK_COMPONENT}`).",
        "",
        "## Ranked unused candidates",
        "",
        format_ranked_table(ranked),
        "",
        "## Chosen units",
        "",
    ]
    for row, kind in [(high[0], "high"), (high[1], "high"), (low[0], "low"), (low[1], "low")]:
        lines += [
            f"### `{row.unit}`",
            "",
            f"- **Entry:** `{row.entry}`",
            f"- **Score (mean_entries):** `{row.score}`",
            f"- **Band:** {'top-2' if kind == 'high' else 'bottom-2'}",
            f"- **Predicted:** {predicted_outcome(kind)}",
            f"- **Closure:** {row.n_functions} functions, {row.n_lines} lines — "
            + ", ".join(f"`{n}`" for n in row.closure),
            f"- **In-package tests scored:** {', '.join(f'`{t}`' for t in row.tests) or '—'}",
            f"- **Files:** {', '.join(f'`{f}`' for f in row.files)}",
            "",
        ]
    lines += [
        "## Proofs",
        "",
        "Screening pass: gold restore and cheat reject only. Alternative-fix",
        "proof (rule A2) skipped. Harness (C5) checked before trusting REWARD.",
        "",
    ]
    if proofs:
        for block in proofs:
            unit = block.get("unit", "")
            lines += [f"### `{unit}`", "", "| check | result |", "|---|---|"]
            checks = block.get("checks") or []
            if isinstance(checks, list):
                for row in checks:
                    if not isinstance(row, dict):
                        continue
                    ok = row.get("ok")
                    if row.get("skipped"):
                        result = "skipped"
                    else:
                        result = "pass" if ok else "FAIL"
                    lines.append(f"| `{row.get('check')}` | {result} |")
            lines.append("")
    else:
        lines.append("_Proofs not yet recorded._")
        lines.append("")
    text = "\n".join(lines) + "\n"
    path = Path(dest or REPORT_PATH)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")
    return text


def rank_and_pick(
    *,
    index: Path | str | None = None,
    tree: Path | str | None = None,
    picks_path: Path | str | None = None,
    report_path: Path | str | None = None,
) -> tuple[list[RankedUnit], list[RankedUnit]]:
    ranked = rank_units(index=index, tree=tree)
    picks = pick_extremes(ranked)
    write_picks(picks, picks_path)
    write_report(ranked, picks, dest=report_path)
    return ranked, picks


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Rank unused units by statefulness")
    parser.add_argument(
        "cmd",
        nargs="?",
        default="rank",
        choices=("rank", "base-image", "build"),
    )
    parser.add_argument("--src", default=str(DEFAULT_SRC))
    parser.add_argument("--index", default=str(PARENT_INDEX))
    parser.add_argument("--picks", default=str(PICKS_PATH))
    parser.add_argument("--report", default=str(REPORT_PATH))
    parser.add_argument("--dest", default=str(DEFAULT_DEST))
    parser.add_argument("--skip-docker", action="store_true")
    parser.add_argument(
        "--unit",
        action="append",
        default=None,
        help="Build/prove only these unit slugs (repeatable). Default: all picks.",
    )
    args = parser.parse_args(argv)
    if args.cmd == "base-image":
        info = build_base_image(args.src)
        print(json.dumps(info, indent=2))
        return 0
    if args.cmd == "build":
        from openswe_traces.synth.scoretest_build import build_and_prove

        payload = build_and_prove(
            src=args.src,
            dest=args.dest,
            skip_docker=args.skip_docker,
            units_filter=args.unit,
        )
        print(json.dumps({k: payload[k] for k in ("ok", "image", "picks") if k in payload}, indent=2))
        return 0 if payload.get("ok") else 2
    ranked, picks = rank_and_pick(
        index=args.index,
        tree=args.src,
        picks_path=args.picks,
        report_path=args.report,
    )
    print(format_ranked_table(ranked))
    print()
    print("picks:")
    print(json.dumps([p.as_pick() for p in picks], indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

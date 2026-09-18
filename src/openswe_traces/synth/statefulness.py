"""Statefulness score of a unit from its existing (hidden or in-tree) tests.

Four components, computed statically from Go test source:

(a) mean exported-API calls per test before the asserted outcome
(b) fraction of tests whose assertion depends on >= 2 prior API calls
(c) mean distinct exported entry points touched per test
(d) whether any test uses goroutines / sync / time (dynamic)

Call counts treat a ``for``/``range`` body as one iteration so a 10k property
loop does not inflate the score. Exploratory Spearman correlations against
Composer flip points live in ``analytics/research/statefulness_vs_flip.md``.
"""

from __future__ import annotations

import argparse
import math
import re
from collections.abc import Iterable, Sequence
from dataclasses import dataclass, field
from pathlib import Path

from openswe_traces.data import ROOT

_TEST_FUNC_RE = re.compile(
    r"^func\s+(?:\(\s*\w+\s+\*?\w+\s*\)\s+)?(Test[A-Za-z0-9_]+)\s*\([^)]*\)\s*\{",
    re.MULTILINE,
)
_EXPORTED_FUNC_RE = re.compile(
    r"^func\s+(?:\(\s*\w+\s+\*?\w+\s*\)\s+)?([A-Z][A-Za-z0-9_]*)\s*\(",
    re.MULTILINE,
)
_IDENT_CALL_RE = re.compile(r"(?:^|[^A-Za-z0-9_.])([A-Za-z_][A-Za-z0-9_]*)\s*\(")
_METHOD_CALL_RE = re.compile(r"\.([A-Za-z_][A-Za-z0-9_]*)\s*\(")
_ASSERT_RE = re.compile(
    r"\b(?:t\.(?:Fatal|Fatalf|Error|Errorf|Fail|FailNow|Helper)|"
    r"assert\.|require\.|t\.Logf)\b"
)
_DYNAMIC_RE = re.compile(
    r"""(?:\bgo\s+func\b|\bgo\s+[A-Za-z_]|\bsync\.|\btime\.(?:Sleep|After|NewTicker|Since|Now|Duration)\b)"""
)
_FOR_RE = re.compile(r"\bfor\b")
_FOR_BOUND_RE = re.compile(
    r"for\s+\w+\s*:=\s*0\s*;\s*\w+\s*<\s*(\d+)",
)


@dataclass(frozen=True)
class TestScore:
    name: str
    api_calls_before_assert: int
    sequence_dependent: bool
    distinct_entry_points: int
    uses_dynamic: bool


@dataclass(frozen=True)
class UnitScore:
    unit: str
    family: str
    flip_point: str
    mean_calls: float
    sequence_fraction: float
    mean_distinct_entries: float
    any_dynamic: bool
    n_tests: int
    tests: tuple[TestScore, ...] = field(default_factory=tuple)

    def components(self) -> dict[str, float | bool | int]:
        return {
            "mean_calls": self.mean_calls,
            "sequence_fraction": self.sequence_fraction,
            "mean_distinct_entries": self.mean_distinct_entries,
            "any_dynamic": self.any_dynamic,
            "n_tests": self.n_tests,
        }


# Composer flip points from FAILURE_AUDIT_A0.md + LADDER2.md.
# property-1pc is listed but excluded from the correlation (verifier voided).
FLIP_UNITS: tuple[dict[str, str], ...] = (
    {"unit": "dynamic-pipeline", "family": "dynamic", "flip": "A3"},
    {"unit": "spec-reimpl-bb", "family": "spec-bb", "flip": "A1"},
    {"unit": "property-backoff", "family": "property", "flip": "A0"},
    {"unit": "spec-bb-chain", "family": "spec-bb", "flip": "A0"},
    {"unit": "spec-bb-bucket", "family": "spec-bb", "flip": "A0"},
    {"unit": "property-policy", "family": "property", "flip": "A0"},
    {"unit": "property-1pc", "family": "property", "flip": "excluded"},
    {"unit": "dynamic-snapshot", "family": "dynamic", "flip": "A0"},
    {"unit": "dynamic-latch", "family": "dynamic", "flip": "A0"},
    {"unit": "file-exclude-filter", "family": "ablation", "flip": "A0"},
    {"unit": "file-filter", "family": "ablation", "flip": "A0"},
    {"unit": "revivelib-runner", "family": "ablation", "flip": "A0"},
)

# Exported API the hidden/in-tree tests actually exercise.
_UNIT_API: dict[str, tuple[str, ...]] = {
    "dynamic-pipeline": (
        "NewPipelinedMemDB",
        "Get",
        "Set",
        "Flush",
        "FlushWait",
        "OnFlushing",
        "Len",
        "Size",
    ),
    "spec-reimpl-bb": (
        "NimbusPack",
        "ZestRing",
        "WillowNode",
        "LumenSeal",
        "HazePipe",
        "MistUnit",
        "MistCore",
        "CedarPath",
        "JadePort",
        "EmberSlot",
        "AmberGate",
        "SablePack",
        "ThornUnit",
        "CedarUnit",
        "QuartzPort",
        "ThornRef",
        "LumenRef",
        "JadeSeal",
    ),
    "property-backoff": ("expo",),
    "spec-bb-chain": (
        "NewRPCInterceptor",
        "NewRPCInterceptorChain",
        "ChainRPCInterceptors",
        "Link",
        "Wrap",
        "Len",
        "WithRPCInterceptor",
        "GetRPCInterceptorFromCtx",
        "Name",
    ),
    "spec-bb-bucket": ("LocateBucket", "Contains", "GetBucketVersion", "String"),
    "property-policy": ("NewConfig", "NewBackoffFnCfg", "String", "SetErrors"),
    "property-1pc": (
        "SetEnable1PC",
        "SetEnableAsyncCommit",
        "NewTiKVTxn",
    ),
    "dynamic-snapshot": (
        "SnapshotGetter",
        "SnapshotIter",
        "SnapshotIterReverse",
        "Get",
        "Set",
        "Value",
        "Next",
    ),
    "dynamic-latch": (
        "NewScheduler",
        "Lock",
        "UnLock",
        "Close",
        "IsStale",
        "SetCommitTS",
        "NewLatches",
    ),
    "file-exclude-filter": ("FileFilter", "New", "GetConfig", "Apply"),
    "file-filter": ("FileFilter", "New", "GetConfig", "Apply"),
    "revivelib-runner": ("New", "Lint", "Format", "ParseConfig"),
}

_UNIT_TEST_HINTS: dict[str, tuple[str, ...]] = {
    "dynamic-pipeline": (
        "experiments/harbor_nex/tasks_unsolv/dynamic-pipeline-L2/tests/hidden",
        "experiments/harbor_nex/tasks_unsolv/dynamic-pipeline-A0/tests/hidden",
    ),
    "spec-reimpl-bb": (
        "experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-L2/tests/hidden",
        "experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-A0/tests/hidden",
        "src/openswe_traces/synth/testdata/codec_bb_prop_test.go",
    ),
    "property-backoff": (
        "experiments/harbor_nex/tasks_unsolv/property-backoff-L2/tests/hidden",
        "experiments/harbor_nex/tasks_unsolv/property-backoff-A0/tests/hidden",
    ),
    "spec-bb-chain": (
        "experiments/harbor_nex/tasks_ladder2/spec-bb-chain-L2/tests/hidden",
        "src/openswe_traces/synth/testdata/ladder2/chain_bb_prop_test.go",
    ),
    "spec-bb-bucket": (
        "experiments/harbor_nex/tasks_ladder2/spec-bb-bucket-L2/tests/hidden",
        "src/openswe_traces/synth/testdata/ladder2/bucket_bb_prop_test.go",
    ),
    "property-policy": (
        "experiments/harbor_nex/tasks_ladder2/property-policy-L2/tests/hidden",
        "src/openswe_traces/synth/testdata/ladder2/policy_prop_test.go",
    ),
    "property-1pc": (
        "experiments/harbor_nex/tasks_ladder2/property-1pc-L2/tests/hidden",
        "src/openswe_traces/synth/testdata/ladder2/onepc_prop_test.go",
    ),
    "dynamic-snapshot": (
        "experiments/harbor_nex/tasks_ladder2/dynamic-snapshot-L2/tests/hidden",
        "src/openswe_traces/synth/testdata/ladder2/snapshot_dyn_test.go",
    ),
    "dynamic-latch": (
        "experiments/harbor_nex/tasks_ladder2/dynamic-latch-L2/tests/hidden",
        "src/openswe_traces/synth/testdata/ladder2/latch_dyn_test.go",
    ),
    "file-exclude-filter": (
        "experiments/harbor_nex/tasks_ablation2/graph-file-exclude-filter-L2/tests/hidden",
        "experiments/harbor_nex/tasks_ablation2/graph-file-exclude-filter-A0/tests/hidden",
    ),
    "file-filter": (
        "experiments/harbor_nex/tasks_ablation2/nograph-file-filter-L2/tests/hidden",
        "experiments/harbor_nex/tasks_ablation2/nograph-file-filter-A0/tests/hidden",
    ),
    "revivelib-runner": (
        "experiments/harbor_nex/tasks_ablation2/nograph-revivelib-runner-L2/tests/hidden",
        "experiments/harbor_nex/tasks_ablation2/nograph-revivelib-runner-A0/tests/hidden",
    ),
}


def exported_api_from_source(text: str) -> set[str]:
    """Exported Go funcs/methods plus lowercase helpers that tests call by name."""
    names = set(_EXPORTED_FUNC_RE.findall(text))
    for m in re.finditer(r"^func\s+([a-z][A-Za-z0-9_]*)\s*\(", text, re.MULTILINE):
        names.add(m.group(1))
    return names


def split_go_tests(text: str) -> list[tuple[str, str]]:
    """Return ``(TestName, body)`` for each top-level Test* function."""
    matches = list(_TEST_FUNC_RE.finditer(text))
    out: list[tuple[str, str]] = []
    for i, m in enumerate(matches):
        start = m.end() - 1  # the opening '{'
        end = _matching_brace(text, start)
        body = text[start + 1 : end]
        out.append((m.group(1), body))
        _ = i
    return out


def _matching_brace(text: str, open_idx: int) -> int:
    depth = 0
    i = open_idx
    in_str = False
    in_raw = False
    in_rune = False
    escape = False
    while i < len(text):
        ch = text[i]
        if in_raw:
            if ch == "`":
                in_raw = False
            i += 1
            continue
        if in_str:
            if escape:
                escape = False
            elif ch == "\\":
                escape = True
            elif ch == '"':
                in_str = False
            i += 1
            continue
        if in_rune:
            if escape:
                escape = False
            elif ch == "\\":
                escape = True
            elif ch == "'":
                in_rune = False
            i += 1
            continue
        if ch == "`":
            in_raw = True
        elif ch == '"':
            in_str = True
        elif ch == "'":
            in_rune = True
        elif ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                return i
        i += 1
    return len(text)


def _strip_strings(text: str) -> str:
    text = re.sub(r"`[^`]*`", '""', text)
    text = re.sub(r'"(?:\\.|[^"\\])*"', '""', text)
    text = re.sub(r"//.*?$", "", text, flags=re.MULTILINE)
    text = re.sub(r"/\*.*?\*/", "", text, flags=re.DOTALL)
    return text


def _last_assert_pos(body: str) -> int | None:
    last = None
    for m in _ASSERT_RE.finditer(body):
        last = m
    return None if last is None else last.start()


def _one_iteration_body(body: str) -> str:
    """If the last assertion lives inside a for-loop, score that loop once plus setup."""
    last_pos = _last_assert_pos(body)
    if last_pos is None or not _FOR_RE.search(body):
        return body
    for m in _FOR_RE.finditer(body):
        brace = body.find("{", m.end())
        if brace < 0:
            continue
        end = _matching_brace(body, brace)
        if brace < last_pos < end:
            return body[: m.start()] + "\n" + body[brace + 1 : end]
    return body


def _calls_in(text: str, api: set[str]) -> list[str]:
    found: list[str] = []
    for rx in (_IDENT_CALL_RE, _METHOD_CALL_RE):
        for m in rx.finditer(text):
            name = m.group(1)
            if name in api:
                found.append(name)
    return found


def _before_last_assert(body: str) -> str:
    last = None
    for m in _ASSERT_RE.finditer(body):
        last = m
    if last is None:
        return body
    return body[: last.start()]


def _calls_before_assert(body: str, api: set[str]) -> list[str]:
    focused = _strip_strings(body)
    collapsed = _one_iteration_body(focused)
    if collapsed != focused:
        return _calls_in(_before_last_assert(collapsed), api)
    prefix = _before_last_assert(focused)
    last_pos = _last_assert_pos(focused)
    calls = _calls_in(prefix, api)
    if last_pos is None:
        return calls
    # Counted for-loop that finishes before the last assert: unroll 2..64 times.
    extra: list[str] = []
    for m in _FOR_BOUND_RE.finditer(focused):
        brace = focused.find("{", m.end())
        if brace < 0 or brace > last_pos:
            continue
        end = _matching_brace(focused, brace)
        if end >= last_pos:
            continue
        n = int(m.group(1))
        if not (2 <= n <= 64):
            continue
        loop_calls = _calls_in(focused[brace + 1 : end], api)
        if loop_calls:
            extra.extend(loop_calls * (n - 1))
    return calls + extra


def score_test_body(name: str, body: str, api: Iterable[str]) -> TestScore:
    api_set = {n for n in api if n}
    calls = _calls_before_assert(body, api_set)
    distinct = len(set(calls))
    dynamic = bool(_DYNAMIC_RE.search(body))
    return TestScore(
        name=name,
        api_calls_before_assert=len(calls),
        sequence_dependent=len(calls) >= 2,
        distinct_entry_points=distinct,
        uses_dynamic=dynamic,
    )


def score_test_file(text: str, api: Iterable[str]) -> list[TestScore]:
    return [score_test_body(name, body, api) for name, body in split_go_tests(text)]


def aggregate(unit: str, family: str, flip: str, tests: Sequence[TestScore]) -> UnitScore:
    if not tests:
        return UnitScore(
            unit=unit,
            family=family,
            flip_point=flip,
            mean_calls=0.0,
            sequence_fraction=0.0,
            mean_distinct_entries=0.0,
            any_dynamic=False,
            n_tests=0,
            tests=(),
        )
    n = len(tests)
    return UnitScore(
        unit=unit,
        family=family,
        flip_point=flip,
        mean_calls=sum(t.api_calls_before_assert for t in tests) / n,
        sequence_fraction=sum(1 for t in tests if t.sequence_dependent) / n,
        mean_distinct_entries=sum(t.distinct_entry_points for t in tests) / n,
        any_dynamic=any(t.uses_dynamic for t in tests),
        n_tests=n,
        tests=tuple(tests),
    )


def _collect_go_files(path: Path) -> list[Path]:
    if path.is_file() and path.suffix == ".go":
        return [path]
    if not path.is_dir():
        return []
    return sorted(p for p in path.rglob("*_test.go") if p.is_file())


def resolve_unit_tests(unit: str, *, root: Path | None = None) -> list[Path]:
    root = root or ROOT
    for hint in _UNIT_TEST_HINTS.get(unit, ()):
        path = root / hint if not Path(hint).is_absolute() else Path(hint)
        files = _collect_go_files(path)
        if files:
            return files
    return []


def score_unit(
    unit: str,
    *,
    family: str = "",
    flip: str = "",
    test_files: Sequence[Path] | None = None,
    api: Iterable[str] | None = None,
    root: Path | None = None,
) -> UnitScore:
    files = list(test_files) if test_files is not None else resolve_unit_tests(unit, root=root)
    names = set(api if api is not None else _UNIT_API.get(unit, ()))
    if not names:
        for path in files:
            names |= exported_api_from_source(path.read_text(encoding="utf-8", errors="replace"))
    scores: list[TestScore] = []
    for path in files:
        text = path.read_text(encoding="utf-8", errors="replace")
        scores.extend(score_test_file(text, names))
    meta = next((u for u in FLIP_UNITS if u["unit"] == unit), None)
    return aggregate(
        unit,
        family or (meta["family"] if meta else ""),
        flip or (meta["flip"] if meta else ""),
        scores,
    )


def score_flip_units(*, root: Path | None = None, include_excluded: bool = True) -> list[UnitScore]:
    rows = []
    for spec in FLIP_UNITS:
        if spec["flip"] == "excluded" and not include_excluded:
            continue
        rows.append(score_unit(spec["unit"], family=spec["family"], flip=spec["flip"], root=root))
    return rows


def flip_to_number(flip: str) -> float | None:
    if flip in {"", "excluded", "—", "-"}:
        return None
    m = re.fullmatch(r"A(-?\d+)", flip.strip())
    if not m:
        return None
    return float(m.group(1))


def _ranks(values: Sequence[float]) -> list[float]:
    """Average ranks (1-based) with ties."""
    indexed = sorted(enumerate(values), key=lambda kv: kv[1])
    ranks = [0.0] * len(values)
    i = 0
    while i < len(indexed):
        j = i
        while j + 1 < len(indexed) and indexed[j + 1][1] == indexed[i][1]:
            j += 1
        avg = (i + 1 + j + 1) / 2.0
        for k in range(i, j + 1):
            ranks[indexed[k][0]] = avg
        i = j + 1
    return ranks


def pearson(xs: Sequence[float], ys: Sequence[float]) -> float | None:
    n = len(xs)
    if n < 2 or n != len(ys):
        return None
    mx = sum(xs) / n
    my = sum(ys) / n
    num = sum((x - mx) * (y - my) for x, y in zip(xs, ys, strict=True))
    dx = math.sqrt(sum((x - mx) ** 2 for x in xs))
    dy = math.sqrt(sum((y - my) ** 2 for y in ys))
    if dx == 0.0 or dy == 0.0:
        return None
    return num / (dx * dy)


def spearman(xs: Sequence[float], ys: Sequence[float]) -> float | None:
    if len(xs) != len(ys) or len(xs) < 3:
        return None
    return pearson(_ranks(xs), _ranks(ys))


def correlate_components(scores: Sequence[UnitScore]) -> dict[str, dict[str, float | int | None]]:
    usable = [s for s in scores if flip_to_number(s.flip_point) is not None and s.n_tests]
    ys = [float(flip_to_number(s.flip_point) or 0.0) for s in usable]
    out: dict[str, dict[str, float | int | None]] = {}
    series = {
        "mean_calls": [s.mean_calls for s in usable],
        "sequence_fraction": [s.sequence_fraction for s in usable],
        "mean_distinct_entries": [s.mean_distinct_entries for s in usable],
        "any_dynamic": [1.0 if s.any_dynamic else 0.0 for s in usable],
    }
    for name, xs in series.items():
        rho = spearman(xs, ys)
        out[name] = {"rho": None if rho is None else round(rho, 4), "n": len(usable)}
    return out


def format_table(scores: Sequence[UnitScore]) -> str:
    lines = [
        "| unit | family | mean_calls (a) | seq_frac (b) | mean_entries (c) | dynamic (d) | n_tests | Composer flip |",
        "|---|---|---:|---:|---:|---|---:|---|",
    ]
    for s in scores:
        flip = "excluded" if s.flip_point == "excluded" else s.flip_point
        dyn = "yes" if s.any_dynamic else "no"
        lines.append(
            f"| `{s.unit}` | {s.family} | {s.mean_calls:.2f} | {s.sequence_fraction:.2f} | "
            f"{s.mean_distinct_entries:.2f} | {dyn} | {s.n_tests} | {flip} |"
        )
    return "\n".join(lines)


def write_report(
    dest: Path | str | None = None,
    *,
    root: Path | None = None,
) -> tuple[str, list[UnitScore]]:
    scores = score_flip_units(root=root, include_excluded=True)
    corr = correlate_components([s for s in scores if s.flip_point != "excluded"])
    n = corr.get("mean_calls", {}).get("n", 0)
    lines = [
        "# Statefulness vs Composer flip point",
        "",
        "Date: 2026-09-18. Exploratory. Computed by",
        "`src/openswe_traces/synth/statefulness.py` from each unit's existing",
        "hidden (or packaged) tests. `property-1pc` is listed then dropped from",
        "the correlation because the verifier was voided (B4).",
        "",
        "Components, per test, then averaged:",
        "",
        "- **(a) mean_calls** — exported-API calls before the last assertion",
        "  (a `for` body that contains the assertion is counted once).",
        "- **(b) seq_frac** — fraction of tests with ≥ 2 such calls.",
        "- **(c) mean_entries** — distinct exported entry points among those calls.",
        "- **(d) dynamic** — any test uses `go`, `sync.`, or `time.` waits/clocks.",
        "",
        format_table(scores),
        "",
        f"## Spearman (exploratory, n={n})",
        "",
        "Flip points encoded as A0=0, A1=1, A3=3. Most units sit at A0, so the",
        "ranks are heavily tied; treat ρ as a directional hint, not a result.",
        "",
        "| component | Spearman ρ | n |",
        "|---|---:|---:|",
    ]
    for key, label in (
        ("mean_calls", "(a) mean_calls"),
        ("sequence_fraction", "(b) seq_frac"),
        ("mean_distinct_entries", "(c) mean_entries"),
        ("any_dynamic", "(d) dynamic"),
    ):
        row = corr.get(key) or {}
        rho = row.get("rho")
        rho_s = "—" if rho is None else f"{rho:.3f}"
        lines.append(f"| {label} | {rho_s} | {row.get('n', 0)} |")
    lines += [
        "",
        "Reproduce:",
        "",
        "```",
        (
            "uv run python -c \"from openswe_traces.synth.statefulness import write_report; "
            "print(write_report()[0])\""
        ),
        "```",
        "",
    ]
    text = "\n".join(lines)
    path = Path(dest) if dest is not None else ROOT / "analytics/research/statefulness_vs_flip.md"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")
    return text, scores


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Statefulness score vs flip point")
    parser.add_argument(
        "--out",
        default=str(ROOT / "analytics/research/statefulness_vs_flip.md"),
    )
    args = parser.parse_args(argv)
    text, scores = write_report(args.out)
    print(format_table(scores))
    print()
    print(text.split("## Spearman", 1)[-1] if "## Spearman" in text else "")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

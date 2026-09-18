"""Parameterized difficulty knobs over a codegraph-indexed Go repo.

These helpers do not inject a bug. They pick sites, decoys, guard tests, and
sparse coverage from any repo that has ``.codegraph/codegraph.db`` so the same
settings can drive Harbor task construction:

    openswe-synth --repo <path> --rung 5 --hops 4 --sites 2 --decoys 1 \\
        --cross-module --guard --out <task-dir>
"""

from __future__ import annotations

import argparse
import json
import re
import sqlite3
from collections import defaultdict, deque
from collections.abc import Sequence
from dataclasses import asdict, dataclass
from pathlib import Path

from openswe_traces.synth.codegraph_bugs import (
    TEST_SYMBOL_PREFIXES,
    codegraph_db,
    go_package,
    hops_to_test_names,
    impact_files,
    impact_of,
    is_test_file,
    is_test_symbol,
    parse_cover_func,
    plausible_fix_sites,
    shortest_caller_path,
)
from openswe_traces.synth.harbor_tasks import (
    AGENT_TIMEOUT_HARD_SEC,
    build_task,
    instruction_self_check,
)

RUNGS = (1, 2, 3, 4, 5, 6, 7, 8)


@dataclass(frozen=True)
class DriftSite:
    name: str
    file_path: str
    hops: int
    path: list[str]
    test_names: list[str]


@dataclass(frozen=True)
class TwoSitePair:
    a: str
    a_file: str
    b: str
    b_file: str
    shared: str
    same_package: bool
    import_edge: bool


@dataclass(frozen=True)
class Decoy:
    name: str
    file_path: str
    reason: str
    guard_tests: list[str]


@dataclass(frozen=True)
class FairAmbiguity:
    cause: str
    cause_file: str
    decoy: Decoy


@dataclass(frozen=True)
class SequenceSite:
    """Rung 7: a function whose single-call tests pass but a sequence test fails."""

    name: str
    file_path: str
    hops: int
    path: list[str]
    sequence_tests: list[str]
    single_call_tests: list[str]


@dataclass(frozen=True)
class FeatureExcision:
    """Rung 8: connected callee subgraph with existing tests (feature excision)."""

    entry: str
    functions: tuple[str, ...]
    files: tuple[str, ...]
    tests: tuple[str, ...]
    keep_interface: bool
    min_lines: int


SEQUENCE_TEST_RE = re.compile(
    r"(Overwrite|Retry|Twice|Again|Sequence|Unique|DeepCopy|ExcludedExceed|"
    r"LocalOracle$|RateLimit|FlushOverwrite|Consecutive|Idempotent|NextSequence)",
    re.IGNORECASE,
)


INVERSE_PAIRS = (
    ("Encode", "Decode"),
    ("Compose", "Extract"),
    ("ExtractPhysical", "GetTimeFromTS"),
    ("GetTimeFromTS", "ExtractPhysical"),
    ("GetPhysical", "ExtractPhysical"),
    ("GoTimeToTS", "GetTimeFromTS"),
    ("ComposeTS", "ExtractPhysical"),
    ("contains", "Contains"),
    ("Contains", "ContainsByEnd"),
)


def _open(index: Path) -> sqlite3.Connection:
    db = codegraph_db(Path(index))
    if not db.exists():
        raise FileNotFoundError(f"no codegraph index at {db}")
    return sqlite3.connect(str(db))


def _functions(con: sqlite3.Connection) -> list[tuple[str, str, str]]:
    """(id, name, file_path) for production functions/methods."""
    rows = con.execute(
        "SELECT id, name, file_path FROM nodes WHERE kind IN ('function', 'method')"
    ).fetchall()
    out: list[tuple[str, str, str]] = []
    for nid, name, fp in rows:
        fp = (fp or "").replace("\\", "/")
        if is_test_file(fp) or is_test_symbol(name):
            continue
        out.append((nid, name, fp))
    return out


def _all_test_names(con: sqlite3.Connection) -> set[str]:
    return {
        r[0]
        for r in con.execute(
            "SELECT name FROM nodes WHERE kind IN ('function','method') AND name GLOB 'Test*'"
        )
    }


def _tests_of(con: sqlite3.Connection) -> dict[str, list[str]]:
    """symbol -> test function names that call it (direct calls edge)."""
    tests: dict[str, list[str]] = defaultdict(list)
    for src_name, tgt_name, src_kind in con.execute(
        """
        SELECT s.name, t.name, s.kind
        FROM edges e
        JOIN nodes s ON s.id = e.source
        JOIN nodes t ON t.id = e.target
        WHERE e.kind = 'calls'
        """
    ):
        if src_kind in ("function", "method") and src_name.startswith(TEST_SYMBOL_PREFIXES):
            tests[tgt_name].append(src_name)
    return {k: sorted(set(v)) for k, v in tests.items()}


def _import_packages(con: sqlite3.Connection) -> set[tuple[str, str]]:
    """Directed package import pairs from ``imports`` edges, if the index has them."""
    pairs: set[tuple[str, str]] = set()
    rows = con.execute(
        """
        SELECT s.file_path, t.file_path
        FROM edges e
        JOIN nodes s ON s.id = e.source
        JOIN nodes t ON t.id = e.target
        WHERE e.kind IN ('imports', 'import', 'contains_import')
        """
    ).fetchall()
    for src_fp, tgt_fp in rows:
        sp = go_package((src_fp or "").replace("\\", "/"))
        tp = go_package((tgt_fp or "").replace("\\", "/"))
        if sp and tp and sp != tp:
            pairs.add((sp, tp))
    return pairs


def pick_contract_drift_site(
    index: Path | str,
    *,
    min_hops: int = 4,
    n: int = 0,
) -> DriftSite | None:
    """Nth production function whose shortest reverse-calls path to a test is ``min_hops``+."""
    repo = Path(index)
    con = _open(repo)
    try:
        tests_of = _tests_of(con)
        all_tests = _all_test_names(con)
        depth = max(min_hops + 8, 12)
        candidates: list[DriftSite] = []
        seen: set[tuple[str, str]] = set()
        for _nid, name, fp in _functions(con):
            key = (name, fp)
            if key in seen:
                continue
            seen.add(key)
            wanted = set(tests_of.get(name, ())) or all_tests
            hops = hops_to_test_names(repo, name, wanted, max_depth=depth)
            if hops is None or hops < min_hops:
                continue
            path = shortest_caller_path(repo, name, wanted, max_depth=depth) or [name]
            test_names = [p for p in path if p.startswith(TEST_SYMBOL_PREFIXES)]
            candidates.append(
                DriftSite(
                    name=name,
                    file_path=fp,
                    hops=hops,
                    path=path,
                    test_names=test_names,
                )
            )
        candidates.sort(key=lambda s: (-s.hops, s.name, s.file_path))
        if not candidates:
            return None
        return candidates[n % len(candidates)]
    finally:
        con.close()


def pick_two_site_pair(index: Path | str, *, n: int = 0) -> TwoSitePair | None:
    """Nth pair of same-named production functions in different packages.

    Prefers pairs with no package-level import edge (rung 3 / DualExpo shape).
    ``shared`` is the colliding name (a type or function the two copies both
    implement), found via name identity — the same signal ``codegraph impact``
    uses when it fans out across homonyms.
    """
    con = _open(Path(index))
    try:
        by_name: dict[str, list[tuple[str, str]]] = defaultdict(list)
        for _nid, name, fp in _functions(con):
            by_name[name].append((name, fp))
        imports = _import_packages(con)
        pairs: list[TwoSitePair] = []
        for name, defs in by_name.items():
            files = sorted({fp for _n, fp in defs})
            if len(files) < 2:
                continue
            for i, a_fp in enumerate(files):
                for b_fp in files[i + 1 :]:
                    pa, pb = go_package(a_fp), go_package(b_fp)
                    edge = (pa, pb) in imports or (pb, pa) in imports
                    pairs.append(
                        TwoSitePair(
                            a=name,
                            a_file=a_fp,
                            b=name,
                            b_file=b_fp,
                            shared=name,
                            same_package=pa == pb,
                            import_edge=edge,
                        )
                    )
        pairs.sort(key=lambda p: (p.import_edge, p.same_package, p.a, p.a_file, p.b_file))
        if not pairs:
            return None
        return pairs[n % len(pairs)]
    finally:
        con.close()


def pick_decoy(
    index: Path | str,
    path: Sequence[str],
    *,
    n: int = 0,
) -> Decoy | None:
    """Pick an unmodified production function a solver would reasonably inspect.

    Preference order:
    1. Intermediate production symbols on ``path`` (cause → … → test).
    2. Sibling callees of the test that share a file or name stem with the cause.
    Guard tests are existing tests of the decoy (so a fake 'fix' is caught).
    """
    repo = Path(index)
    if not path:
        return None
    con = _open(repo)
    try:
        name_files: dict[str, list[str]] = defaultdict(list)
        for _nid, name, fp in _functions(con):
            if fp not in name_files[name]:
                name_files[name].append(fp)
        for _nid, name, fp in con.execute(
            "SELECT id, name, file_path FROM nodes WHERE kind IN ('function', 'method')"
        ):
            fp = (fp or "").replace("\\", "/")
            if fp and fp not in name_files[name]:
                name_files[name].append(fp)
        tests_of = _tests_of(con)
        name_of = {nid: nm for nid, nm in con.execute("SELECT id, name FROM nodes")}
        id_of: dict[str, list[str]] = defaultdict(list)
        for nid, nm in name_of.items():
            id_of[nm].append(nid)

        intermediates = plausible_fix_sites(list(path))
        cause = path[0]
        cause_files = name_files.get(cause, [])
        cause_pkgs = {go_package(f) for f in cause_files if f}
        ranked: list[Decoy] = []
        for name in intermediates:
            if name == cause:
                continue
            fps = name_files.get(name, [""])
            ranked.append(
                Decoy(
                    name=name,
                    file_path=fps[0] if fps else "",
                    reason="intermediate on the test→cause call path with overlapping duty",
                    guard_tests=tests_of.get(name, []),
                )
            )

        test_name = next((p for p in reversed(path) if p.startswith(TEST_SYMBOL_PREFIXES)), None)
        if test_name:
            sibling_names: set[str] = set()
            for tid in id_of.get(test_name, []):
                for tgt in con.execute(
                    "SELECT t.name FROM edges e JOIN nodes t ON t.id = e.target "
                    "WHERE e.kind = 'calls' AND e.source = ?",
                    [tid],
                ):
                    sibling_names.add(tgt[0])
            test_pkgs = {go_package(f) for f in name_files.get(test_name, []) if f}
            match_pkgs = cause_pkgs | test_pkgs
            for sib in sorted(sibling_names):
                if sib == cause or sib.startswith(TEST_SYMBOL_PREFIXES):
                    continue
                if any(d.name == sib for d in ranked):
                    continue
                sib_files = name_files.get(sib, [""])
                sfp = sib_files[0] if sib_files else ""
                same_file = bool(set(cause_files) & set(sib_files))
                same_pkg = bool(match_pkgs & {go_package(f) for f in sib_files if f})
                stem_overlap = cause.lower() in sib.lower() or sib.lower() in cause.lower()
                if not (same_file or same_pkg or stem_overlap):
                    continue
                ranked.append(
                    Decoy(
                        name=sib,
                        file_path=sfp,
                        reason=(
                            "sibling callee of the failing test; same package/file as the "
                            "true cause so a competent engineer inspects it first"
                        ),
                        guard_tests=tests_of.get(sib, []),
                    )
                )

        # Callees of the cause / of intermediates (sibling conversions on the same line).
        parent_names = [cause, *intermediates]
        for parent in parent_names:
            for pid in id_of.get(parent, []):
                for (cname,) in con.execute(
                    "SELECT t.name FROM edges e JOIN nodes t ON t.id = e.target "
                    "WHERE e.kind = 'calls' AND e.source = ?",
                    [pid],
                ):
                    if cname == cause or cname.startswith(TEST_SYMBOL_PREFIXES):
                        continue
                    if any(d.name == cname for d in ranked):
                        continue
                    c_files = name_files.get(cname, [""])
                    same_file = bool(set(cause_files) & set(c_files))
                    same_pkg = bool(cause_pkgs & {go_package(f) for f in c_files if f})
                    if not (same_file or same_pkg):
                        continue
                    ranked.append(
                        Decoy(
                            name=cname,
                            file_path=c_files[0] if c_files else "",
                            reason=(
                                f"callee of {parent} on the test→cause path; overlapping "
                                "conversion/duty so a competent engineer inspects it first"
                            ),
                            guard_tests=tests_of.get(cname, []),
                        )
                    )
        if not ranked:
            return None
        return ranked[n % len(ranked)]
    finally:
        con.close()


def find_guard_tests(
    index: Path | str,
    symbol: str,
    *,
    max_hops: int = 3,
) -> list[str]:
    """Existing test functions that reach ``symbol`` within ``max_hops`` reverse-calls."""
    repo = Path(index)
    db = codegraph_db(repo)
    if not db.exists():
        return []
    con = sqlite3.connect(str(db))
    try:
        starts = [
            r[0]
            for r in con.execute(
                "SELECT id FROM nodes WHERE name = ? AND kind IN ('function','method')",
                [symbol],
            ).fetchall()
        ]
        if not starts:
            return []
        callers_map: dict[str, list[str]] = defaultdict(list)
        for src, tgt in con.execute("SELECT source, target FROM edges WHERE kind = 'calls'"):
            callers_map[tgt].append(src)
        name_of = {nid: nm for nid, nm in con.execute("SELECT id, name FROM nodes")}
        hits: set[str] = set()
        seen: set[str] = set(starts)
        q: deque[tuple[str, int]] = deque((s, 0) for s in starts)
        while q:
            nid, dist = q.popleft()
            nm = name_of.get(nid, "")
            if dist > 0 and nm.startswith(TEST_SYMBOL_PREFIXES):
                hits.add(nm)
            if dist >= max_hops:
                continue
            for caller in callers_map.get(nid, []):
                if caller not in seen:
                    seen.add(caller)
                    q.append((caller, dist + 1))
        return sorted(hits)
    finally:
        con.close()


def find_sparse_branches(
    coverprofile: Path | str,
    *,
    max_pct: float = 50.0,
) -> list[tuple[str, str, float]]:
    """Functions at or below ``max_pct`` statement coverage.

    Accepts ``go tool cover -func`` output (tab + percent) or a ``mode: set``
    coverprofile (treated as 0/100 per block, then averaged per function).
    """
    path = Path(coverprofile)
    text = path.read_text(encoding="utf-8", errors="replace")
    if text.startswith("mode:"):
        # Fall back to parsing file:line.col,file:line.col num count
        # without a func name; return file-level density as ("file", "*", pct).
        stmt = 0
        covered = 0
        per_file: dict[str, list[int]] = defaultdict(lambda: [0, 0])
        for line in text.splitlines()[1:]:
            if not line.strip():
                continue
            left, count_s = line.rsplit(" ", 1)
            try:
                count = int(count_s)
            except ValueError:
                continue
            num_s = left.rsplit(" ", 1)[-1]
            try:
                num = int(num_s)
            except ValueError:
                continue
            fp = left.split(":", 1)[0]
            stmt += num
            if count > 0:
                covered += num
            per_file[fp][0] += num
            if count > 0:
                per_file[fp][1] += num
        rows: list[tuple[str, str, float]] = []
        for fp, (n, c) in sorted(per_file.items()):
            pct = (100.0 * c / n) if n else 0.0
            if pct <= max_pct:
                rows.append((fp, "*", pct))
        return rows
    cover = parse_cover_func(path)
    rows = []
    seen: set[tuple[str, str]] = set()
    for (fp, func), pct in cover.items():
        if (fp, func) in seen:
            continue
        seen.add((fp, func))
        if pct <= max_pct:
            rows.append((fp, func, pct))
    rows.sort(key=lambda r: (r[2], r[0], r[1]))
    return rows


def name_leakage(
    f2p_tests: Sequence[str],
    failure_text: str,
    *,
    changed_symbols: Sequence[str] = (),
    changed_files: Sequence[str] = (),
) -> dict[str, object]:
    """SYMPTOM-LOCALITY: do f2p names or failure text contain the changed symbol/file?"""
    blob = " ".join(f2p_tests) + "\n" + (failure_text or "")
    leaked_symbols = [
        s for s in changed_symbols if s and re.search(rf"\b{re.escape(s)}\b", blob)
    ]
    leaked_files: list[str] = []
    for fp in changed_files:
        base = Path(fp).name
        if base and base in blob:
            leaked_files.append(base)
    return {
        "leaked_symbols": leaked_symbols,
        "leaked_files": leaked_files,
        "ok": not leaked_symbols and not leaked_files,
    }


def pick_fair_ambiguity(index: Path | str, *, n: int = 0) -> FairAmbiguity | None:
    """Inverse/overlapping pair: cause + unmodified decoy with existing guard tests."""
    con = _open(Path(index))
    try:
        files_of: dict[str, str] = {}
        for _nid, name, fp in _functions(con):
            files_of.setdefault(name, fp)
        tests_of = _tests_of(con)
        found: list[FairAmbiguity] = []
        seen: set[tuple[str, str]] = set()
        for cause, decoy_name in INVERSE_PAIRS:
            if cause not in files_of or decoy_name not in files_of:
                continue
            key = (cause, decoy_name)
            if key in seen:
                continue
            seen.add(key)
            found.append(
                FairAmbiguity(
                    cause=cause,
                    cause_file=files_of[cause],
                    decoy=Decoy(
                        name=decoy_name,
                        file_path=files_of[decoy_name],
                        reason=(
                            f"{decoy_name} is the inverse/overlapping conversion of {cause}; "
                            "a competent engineer would inspect it first from the symptom"
                        ),
                        guard_tests=tests_of.get(decoy_name, []),
                    ),
                )
            )
        if not found:
            return None
        return found[n % len(found)]
    finally:
        con.close()


def is_sequence_test_name(name: str) -> bool:
    """True if a test name encodes a multi-call / overwrite / retry sequence."""
    return bool(name) and bool(SEQUENCE_TEST_RE.search(name))


def find_sequence_tests(
    index: Path | str,
    symbol: str,
    *,
    max_hops: int = 6,
) -> list[str]:
    """Existing sequence-named tests that reach ``symbol`` within ``max_hops``."""
    return [t for t in find_guard_tests(index, symbol, max_hops=max_hops) if is_sequence_test_name(t)]


def pick_sequence_site(index: Path | str, *, n: int = 0) -> SequenceSite | None:
    """Nth production function on a sequence-test callee path that also has a single-call test.

    A sequence test's name matches ``SEQUENCE_TEST_RE`` (Overwrite, Sequence,
    LocalOracle uniqueness, retry/excluded-exceed, …). The single-call tests
    are other existing tests of the same symbol that do not match that pattern.
    """
    repo = Path(index)
    con = _open(repo)
    try:
        name_of = {nid: nm for nid, nm in con.execute("SELECT id, name FROM nodes")}
        file_of = {
            nid: (fp or "").replace("\\", "/")
            for nid, fp in con.execute("SELECT id, file_path FROM nodes")
        }
        kind_of = {nid: k for nid, k in con.execute("SELECT id, kind FROM nodes")}
        callees: dict[str, list[str]] = defaultdict(list)
        for src, tgt in con.execute("SELECT source, target FROM edges WHERE kind = 'calls'"):
            callees[src].append(tgt)
        tests_of = _tests_of(con)
        seq_tests: list[tuple[str, str]] = []
        for nid, nm in name_of.items():
            if kind_of.get(nid) not in ("function", "method"):
                continue
            if not nm.startswith(TEST_SYMBOL_PREFIXES):
                continue
            if is_sequence_test_name(nm):
                seq_tests.append((nid, nm))
        found: list[SequenceSite] = []
        seen: set[tuple[str, str]] = set()
        for tid, tname in seq_tests:
            q: deque[tuple[str, int, list[str]]] = deque([(tid, 0, [tname])])
            visited: set[str] = {tid}
            while q:
                nid, dist, path = q.popleft()
                if dist >= 8:
                    continue
                for callee in callees.get(nid, []):
                    if callee in visited:
                        continue
                    visited.add(callee)
                    cname = name_of.get(callee, "")
                    ckind = kind_of.get(callee, "")
                    cfile = file_of.get(callee, "")
                    new_path = path + [cname]
                    if (
                        ckind in ("function", "method")
                        and cname
                        and not is_test_symbol(cname)
                        and not is_test_file(cfile)
                        and dist + 1 >= 1
                    ):
                        key = (cname, cfile)
                        if key not in seen:
                            seen.add(key)
                            all_tests = tests_of.get(cname, [])
                            single = sorted({t for t in all_tests if not is_sequence_test_name(t)})
                            seq_hit = sorted(
                                {t for t in all_tests if is_sequence_test_name(t)} | {tname}
                            )
                            found.append(
                                SequenceSite(
                                    name=cname,
                                    file_path=cfile,
                                    hops=dist + 1,
                                    path=list(reversed(new_path)),
                                    sequence_tests=seq_hit,
                                    single_call_tests=single,
                                )
                            )
                    q.append((callee, dist + 1, new_path))
        # Prefer sites that have a real single-call test (sequence vs one-shot split).
        found.sort(
            key=lambda s: (
                0 if s.single_call_tests else 1,
                -s.hops,
                s.name,
                s.file_path,
            )
        )
        if not found:
            return None
        return found[n % len(found)]
    finally:
        con.close()


def _callee_graph(con: sqlite3.Connection) -> dict[tuple[str, str], list[tuple[str, str]]]:
    """(name, file) -> [(callee_name, callee_file)] for production functions."""
    graph: dict[tuple[str, str], list[tuple[str, str]]] = defaultdict(list)
    rows = con.execute(
        """
        SELECT s.name, s.file_path, t.name, t.file_path
        FROM edges e
        JOIN nodes s ON s.id = e.source
        JOIN nodes t ON t.id = e.target
        WHERE e.kind = 'calls'
          AND s.kind IN ('function', 'method')
          AND t.kind IN ('function', 'method')
        """
    )
    seen: dict[tuple[str, str], set[tuple[str, str]]] = defaultdict(set)
    for sn, sf, tn, tf in rows:
        sf = (sf or "").replace("\\", "/")
        tf = (tf or "").replace("\\", "/")
        if is_test_file(sf) or is_test_file(tf) or is_test_symbol(sn) or is_test_symbol(tn):
            continue
        src, tgt = (sn, sf), (tn, tf)
        if tgt in seen[src] or src == tgt:
            continue
        seen[src].add(tgt)
        graph[src].append(tgt)
    return graph


def _is_exported_go(name: str) -> bool:
    return bool(name) and name[0].isalpha() and name[0].isupper()


def pick_feature_excision(
    index: Path | str,
    *,
    min_functions: int = 3,
    min_files: int = 2,
    keep_interface: bool = True,
    min_lines: int = 60,
    n: int = 0,
    skip: frozenset[str] = frozenset(),
) -> FeatureExcision | None:
    """Nth connected callee subgraph with an exported entry and existing tests.

    BFS from each exported production function through ``calls`` edges (the same
    closure ``codegraph callees`` would return). Filter by ``min_functions`` /
    ``min_files``. ``keep_interface`` is recorded on the design (stubs vs delete
    signatures); it does not change the subgraph search.
    """
    con = _open(Path(index))
    try:
        graph = _callee_graph(con)
        tests_of = _tests_of(con)
        funcs = {(name, fp) for _nid, name, fp in _functions(con)}
        candidates: list[FeatureExcision] = []
        seen_sets: set[tuple[str, ...]] = set()
        for entry_name, entry_fp in sorted(funcs, key=lambda x: (x[0], x[1])):
            if not _is_exported_go(entry_name) or entry_name in skip:
                continue
            closure: list[tuple[str, str]] = []
            q = deque([(entry_name, entry_fp)])
            visited: set[tuple[str, str]] = set()
            while q:
                cur = q.popleft()
                if cur in visited:
                    continue
                visited.add(cur)
                closure.append(cur)
                for nxt in graph.get(cur, ()):
                    if nxt in funcs:
                        q.append(nxt)
            names = tuple(sorted({nm for nm, _fp in closure}))
            files = tuple(sorted({fp for _nm, fp in closure}))
            if len(names) < min_functions or len(files) < min_files:
                continue
            if names in seen_sets:
                continue
            seen_sets.add(names)
            tests: list[str] = []
            for nm, _fp in closure:
                tests.extend(tests_of.get(nm, ()))
            tests = sorted(set(tests))
            if not tests:
                continue
            candidates.append(
                FeatureExcision(
                    entry=entry_name,
                    functions=names,
                    files=files,
                    tests=tuple(tests),
                    keep_interface=keep_interface,
                    min_lines=min_lines,
                )
            )
        candidates.sort(
            key=lambda e: (-len(e.tests), -len(e.functions), -len(e.files), e.entry)
        )
        if not candidates:
            return None
        return candidates[n % len(candidates)]
    finally:
        con.close()


def pick_implicit_invariant(index: Path | str, *, n: int = 0) -> DriftSite | None:
    """Rung 6: prefer a documented-invariant helper (fixture ``Clamp``) else a drift site."""
    repo = Path(index)
    con = _open(repo)
    try:
        tests_of = _tests_of(con)
        all_tests = _all_test_names(con)
        for _nid, name, fp in _functions(con):
            if name != "Clamp":
                continue
            wanted = set(tests_of.get(name, ())) or all_tests
            path = shortest_caller_path(repo, name, wanted, max_depth=8) or [name]
            hops = hops_to_test_names(repo, name, wanted, max_depth=8) or 1
            return DriftSite(
                name=name,
                file_path=fp,
                hops=hops,
                path=path,
                test_names=[p for p in path if p.startswith(TEST_SYMBOL_PREFIXES)],
            )
    finally:
        con.close()
    return pick_contract_drift_site(repo, min_hops=1, n=n)


def design_for_rung(
    repo: Path | str,
    *,
    rung: int,
    hops: int = 4,
    sites: int = 1,
    decoys: int = 0,
    index: int = 0,
    keep_interface: bool = True,
    cross_module: bool = False,
) -> dict[str, object]:
    """JSON-able design dict for one knob setting. Does not mutate the repo."""
    repo = Path(repo)
    if rung not in RUNGS:
        raise ValueError(f"rung must be one of {RUNGS}")
    if cross_module and sites < 2:
        sites = 2
    site = pick_contract_drift_site(repo, min_hops=hops, n=index)
    if rung == 6:
        site = pick_implicit_invariant(repo, n=index) or site
    seq = pick_sequence_site(repo, n=index) if rung == 7 else None
    if rung == 7 and seq:
        site = DriftSite(
            name=seq.name,
            file_path=seq.file_path,
            hops=seq.hops,
            path=seq.path,
            test_names=list(seq.sequence_tests),
        )
    want_pair = sites >= 2 or rung in {2, 3, 4} or cross_module
    pair = pick_two_site_pair(repo, n=index) if want_pair else None
    amb = pick_fair_ambiguity(repo, n=index) if rung == 5 or decoys else None
    decoy_objs: list[Decoy] = []
    if amb and (rung == 5 or decoys):
        decoy_objs.append(amb.decoy)
        if not site:
            site = DriftSite(
                name=amb.cause,
                file_path=amb.cause_file,
                hops=hops,
                path=[amb.cause],
                test_names=list(amb.decoy.guard_tests),
            )
    if site and decoys:
        for i in range(decoys):
            d = pick_decoy(repo, site.path, n=i)
            if d and all(x.name != d.name for x in decoy_objs):
                decoy_objs.append(d)
    payload: dict[str, object] = {
        "repo": str(repo),
        "rung": rung,
        "hops_requested": hops,
        "sites_requested": sites,
        "decoys_requested": decoys,
        "site": asdict(site) if site else None,
        "two_site": asdict(pair) if pair else None,
        "fair_ambiguity": asdict(amb) if amb else None,
        "sequence": asdict(seq) if seq else None,
        "decoys": [asdict(d) for d in decoy_objs],
        "excision": None,
        "keep_interface": keep_interface,
        "cross_module": bool(cross_module),
    }
    if rung == 8:
        min_fn = 6 if not keep_interface else max(3, sites if sites > 1 else 3)
        min_files = 3 if not keep_interface else 2
        min_lines = 150 if not keep_interface else 60
        exc = pick_feature_excision(
            repo,
            min_functions=min_fn,
            min_files=min_files,
            keep_interface=keep_interface,
            min_lines=min_lines,
            n=index,
        )
        payload["excision"] = asdict(exc) if exc else None
        if exc:
            impact = impact_of(repo, exc.entry)
            payload["impact_files"] = sorted(impact_files(impact))
            payload["guard_tests"] = find_guard_tests(repo, exc.entry)
    if site and "impact_files" not in payload:
        impact = impact_of(repo, site.name)
        payload["impact_files"] = sorted(impact_files(impact))
        payload["guard_tests"] = find_guard_tests(repo, site.name)
    return payload


def _as_dict(value: object) -> dict[str, object] | None:
    return value if isinstance(value, dict) else None


def validate_design(
    design: dict[str, object],
    *,
    hops: int,
    sites: int,
    decoys: int,
    cross_module: bool,
    guard: bool,
) -> dict[str, object]:
    """Static checks that the knob request is realized in ``design``."""
    site = _as_dict(design.get("site"))
    pair = _as_dict(design.get("two_site"))
    excision = _as_dict(design.get("excision"))
    decoy_list = [d for d in (design.get("decoys") or []) if isinstance(d, dict)]
    guard_tests: list[str] = [str(t) for t in (design.get("guard_tests") or [])]
    for d in decoy_list:
        guard_tests.extend(str(t) for t in (d.get("guard_tests") or []))
    site_hops = site.get("hops") if site else None
    n_fn = len(excision.get("functions") or []) if excision else 0
    checks = {
        "site_found": bool(site or excision or pair),
        "hops_met": True if site_hops is None else int(site_hops) >= hops,
        "sites_met": True,
        "decoys_met": len(decoy_list) >= decoys,
        "cross_module_met": True,
        "guard_met": True,
    }
    if sites >= 2:
        checks["sites_met"] = pair is not None or n_fn >= sites
    if cross_module:
        checks["cross_module_met"] = bool(
            (pair and not pair.get("import_edge") and not pair.get("same_package"))
            or (excision and len(excision.get("files") or []) >= 2)
        )
    if guard:
        checks["guard_met"] = bool(guard_tests)
    payload = dict(design)
    payload["checks"] = checks
    payload["ok"] = all(bool(v) for v in checks.values())
    payload["cross_module"] = bool(cross_module)
    payload["guard"] = bool(guard)
    payload["guard_tests_resolved"] = sorted(set(guard_tests))
    return payload


def _redact_terms(design: dict[str, object]) -> list[str]:
    extra: list[str] = []
    site = _as_dict(design.get("site")) or {}
    if site.get("name"):
        extra.append(str(site["name"]))
    if site.get("file_path"):
        extra.append(Path(str(site["file_path"])).name)
    for d in design.get("decoys") or []:
        if isinstance(d, dict) and d.get("name"):
            extra.append(str(d["name"]))
    exc = _as_dict(design.get("excision")) or {}
    for name in exc.get("functions") or []:
        extra.append(str(name))
    for fp in exc.get("files") or []:
        extra.append(Path(str(fp)).name)
    return extra


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Codegraph difficulty knobs → site design / Harbor task + validation.json"
    )
    parser.add_argument("--repo", required=True)
    parser.add_argument("--rung", type=int, default=5)
    parser.add_argument("--hops", type=int, default=4)
    parser.add_argument("--sites", type=int, default=1)
    parser.add_argument("--decoys", type=int, default=1)
    parser.add_argument("--index", type=int, default=0)
    parser.add_argument("--locality", type=int, default=0)
    parser.add_argument(
        "--cross-module",
        action="store_true",
        help="Require a two-site pair with no package import edge (or a multi-file excision)",
    )
    parser.add_argument(
        "--guard",
        action="store_true",
        help="Include existing guard tests from the design in the Harbor verifier",
    )
    parser.add_argument(
        "--guard-tests",
        nargs="*",
        default=None,
        help="Explicit guard test names (overrides --guard discovery)",
    )
    parser.add_argument(
        "--keep-interface",
        action=argparse.BooleanOptionalAction,
        default=True,
        help="rung 8: keep exported signatures as stubs (default) or delete them",
    )
    parser.add_argument("--patch", default=None, help="If set, build a Harbor task from this patch")
    parser.add_argument(
        "--out",
        default=None,
        help="Output dir: always writes validation.json; with --patch, a Harbor task",
    )
    parser.add_argument("--f2p", nargs="*", default=None)
    parser.add_argument("--base-commit", default="HEAD")
    args = parser.parse_args(argv)
    design = design_for_rung(
        args.repo,
        rung=args.rung,
        hops=args.hops,
        sites=args.sites,
        decoys=args.decoys,
        index=args.index,
        keep_interface=args.keep_interface,
        cross_module=args.cross_module,
    )
    payload = validate_design(
        design,
        hops=args.hops,
        sites=max(args.sites, 2 if args.cross_module else args.sites),
        decoys=args.decoys,
        cross_module=args.cross_module,
        guard=args.guard or bool(args.guard_tests),
    )
    task_dir = None
    if args.patch:
        if not args.out:
            raise SystemExit("--out is required with --patch")
        f2p = args.f2p or (design.get("site") or {}).get("test_names") or []
        if not f2p:
            f2p = list((design.get("excision") or {}).get("tests") or [])
        if not f2p:
            raise SystemExit("no f2p tests; pass --f2p")
        if args.guard_tests:
            guards = list(args.guard_tests)
        elif args.guard:
            guards = list(payload.get("guard_tests_resolved") or design.get("guard_tests") or [])
        else:
            guards = []
        task_dir = build_task(
            args.repo,
            args.base_commit,
            args.patch,
            list(f2p),
            args.out,
            agent_timeout_sec=AGENT_TIMEOUT_HARD_SEC,
            extra_redact=_redact_terms(design),
            checksum_test_files=True,
            locality=args.locality,
            guard_tests=guards,
            kind="feature" if args.rung == 8 else "bug",
        )
        instruction = (task_dir / "instruction.md").read_text(encoding="utf-8")
        test_sh = (task_dir / "tests" / "test.sh").read_text(encoding="utf-8")
        payload["instruction_self_check"] = instruction_self_check(
            instruction,
            test_sh=test_sh,
            f2p_tests=list(f2p),
            changed_symbols=_redact_terms(design),
            changed_files=[
                str((_as_dict(design.get("site")) or {}).get("file_path") or ""),
                *[
                    str(fp)
                    for fp in ((_as_dict(design.get("excision")) or {}).get("files") or [])
                ],
            ],
            locality=args.locality,
        )
        payload["task_dir"] = str(task_dir)
        payload["ok"] = bool(payload["ok"]) and bool(
            (_as_dict(payload.get("instruction_self_check")) or {}).get("ok", True)
        )
    elif args.out:
        payload["task_dir"] = None
    if args.out:
        out = Path(args.out)
        out.mkdir(parents=True, exist_ok=True)
        (out / "validation.json").write_text(json.dumps(payload, indent=2) + "\n")
    print(json.dumps(payload, indent=2))
    return 0 if payload.get("ok") else 2


if __name__ == "__main__":
    raise SystemExit(main())

"""Parameterized difficulty knobs over a codegraph-indexed Go repo.

The same settings (rung, hops, sites, decoys) can be applied to any host:
``openswe-synth --repo <path> --rung 5 --hops 4 --sites 2 --decoys 1``.
"""

from __future__ import annotations

import argparse
import json
import sqlite3
from collections import defaultdict, deque
from collections.abc import Sequence
from dataclasses import asdict, dataclass, field
from pathlib import Path

from openswe_traces.synth.codegraph_bugs import (
    codegraph_db,
    go_package,
    is_go_exported,
    is_test_file,
    is_test_symbol,
    parse_cover_func,
    plausible_fix_sites,
    shortest_caller_path,
)

SKIP_DIR_PREFIXES = ("examples/", "integration_tests/")
PAIR_PREFIXES = (
    ("Encode", "Decode"),
    ("encode", "decode"),
    ("Parse", "Format"),
    ("Marshal", "Unmarshal"),
    ("Pack", "Unpack"),
    ("Compose", "Extract"),
    ("Append", "Remove"),
)


@dataclass(frozen=True)
class Site:
    name: str
    kind: str
    file_path: str
    package: str
    start_line: int
    hops: int | None = None
    n_impact_files: int = 0


@dataclass(frozen=True)
class TwoSitePair:
    a: Site
    b: Site
    relation: str
    shared_type: str = ""
    no_import_edge: bool = False


@dataclass(frozen=True)
class ThreeSite:
    sites: tuple[Site, Site, Site]
    relation: str


@dataclass(frozen=True)
class Decoy:
    name: str
    file_path: str
    package: str
    why: str
    path: tuple[str, ...]
    existing_test_if_fixed: str = ""


@dataclass
class Construction:
    rung: int
    hops: int
    sites: int
    decoys: int
    picked_sites: list[Site] = field(default_factory=list)
    pair: TwoSitePair | None = None
    triple: ThreeSite | None = None
    decoy_list: list[Decoy] = field(default_factory=list)
    guard_tests: list[str] = field(default_factory=list)
    sparse: list[dict[str, object]] = field(default_factory=list)
    notes: str = ""


def _open(index: Path | sqlite3.Connection | str) -> tuple[sqlite3.Connection, bool]:
    if isinstance(index, sqlite3.Connection):
        return index, False
    path = Path(index)
    if path.is_dir():
        path = codegraph_db(path)
    con = sqlite3.connect(str(path))
    return con, True


def _skip(rel: str) -> bool:
    rel = rel.replace("\\", "/")
    return any(rel.startswith(p) for p in SKIP_DIR_PREFIXES) or is_test_file(rel)


def load_nodes(index: Path | sqlite3.Connection | str) -> list[Site]:
    con, close = _open(index)
    try:
        rows = con.execute(
            """
            SELECT name, kind, file_path, start_line
            FROM nodes
            WHERE kind IN ('function', 'method')
            """
        ).fetchall()
    finally:
        if close:
            con.close()
    out: list[Site] = []
    for name, kind, fp, line in rows:
        fp = (fp or "").replace("\\", "/")
        if _skip(fp) or is_test_symbol(name):
            continue
        out.append(
            Site(
                name=name,
                kind=kind,
                file_path=fp,
                package=go_package(fp),
                start_line=int(line or 0),
            )
        )
    return out


def package_imports(index: Path | sqlite3.Connection | str) -> dict[str, set[str]]:
    """Go import edges between packages (from codegraph ``imports`` + file layout)."""
    con, close = _open(index)
    try:
        rows = con.execute(
            """
            SELECT s.file_path, t.file_path, t.qualified_name, t.name
            FROM edges e
            JOIN nodes s ON s.id = e.source
            JOIN nodes t ON t.id = e.target
            WHERE e.kind = 'imports'
            """
        ).fetchall()
    finally:
        if close:
            con.close()
    out: dict[str, set[str]] = defaultdict(set)
    for sfp, tfp, qn, tname in rows:
        src = go_package((sfp or "").replace("\\", "/"))
        dest = ""
        if tfp:
            dest = go_package(tfp.replace("\\", "/"))
        elif qn:
            dest = qn.replace("\\", "/").strip("/")
        elif tname:
            dest = tname.replace("\\", "/")
        if src and dest:
            out[src].add(dest)
    return out


def no_import_edge(imports: dict[str, set[str]], a: str, b: str) -> bool:
    if a == b:
        return False
    return b not in imports.get(a, set()) and a not in imports.get(b, set())


def hops_from_tests(
    index: Path | sqlite3.Connection | str,
    *,
    max_depth: int = 12,
) -> dict[str, int]:
    """Min reverse-calls hops from any test function to each production node id."""
    con, close = _open(index)
    try:
        tests = [
            r[0]
            for r in con.execute(
                """
                SELECT id FROM nodes
                WHERE kind IN ('function', 'method')
                  AND name GLOB 'Test*'
                """
            )
        ]
        callers: dict[str, list[str]] = defaultdict(list)
        callees: dict[str, list[str]] = defaultdict(list)
        for src, tgt in con.execute("SELECT source, target FROM edges WHERE kind = 'calls'"):
            callers[tgt].append(src)
            callees[src].append(tgt)
        # Forward BFS from tests along calls (test → callees).
        dist: dict[str, int] = {}
        q: deque[tuple[str, int]] = deque((t, 0) for t in tests)
        seen = set(tests)
        while q:
            nid, d = q.popleft()
            dist[nid] = d
            if d >= max_depth:
                continue
            for callee in callees.get(nid, []):
                if callee not in seen:
                    seen.add(callee)
                    q.append((callee, d + 1))
        return dist
    finally:
        if close:
            con.close()


def _node_id_map(index: Path | sqlite3.Connection | str) -> dict[tuple[str, str], str]:
    con, close = _open(index)
    try:
        rows = con.execute(
            "SELECT id, name, file_path FROM nodes WHERE kind IN ('function', 'method')"
        ).fetchall()
    finally:
        if close:
            con.close()
    return {(name, (fp or "").replace("\\", "/")): nid for nid, name, fp in rows}


def pick_contract_drift_site(
    index: Path | sqlite3.Connection | str,
    min_hops: int = 4,
    *,
    skip_names: Sequence[str] = (),
) -> Site | None:
    """Exported production function whose nearest test is at least ``min_hops`` away."""
    skip = set(skip_names)
    dist = hops_from_tests(index)
    ids = _node_id_map(index)
    ranked: list[Site] = []
    for site in load_nodes(index):
        if site.name in skip or not is_go_exported(site.name):
            continue
        nid = ids.get((site.name, site.file_path))
        hops = dist.get(nid) if nid else None
        if hops is None or hops < min_hops:
            continue
        ranked.append(
            Site(
                name=site.name,
                kind=site.kind,
                file_path=site.file_path,
                package=site.package,
                start_line=site.start_line,
                hops=hops,
            )
        )
    ranked.sort(key=lambda s: (-(s.hops or 0), s.package, s.name))
    return ranked[0] if ranked else None


def pick_encode_decode_pairs(
    index: Path | sqlite3.Connection | str,
    *,
    skip_names: Sequence[str] = (),
) -> list[TwoSitePair]:
    """Name-paired encode/decode (etc.) functions, same stem, preferably two files."""
    skip = set(skip_names)
    by_name: dict[str, list[Site]] = defaultdict(list)
    for site in load_nodes(index):
        if site.name not in skip:
            by_name[site.name].append(site)
    pairs: list[TwoSitePair] = []
    seen: set[tuple[str, str, str, str]] = set()
    for enc_pre, dec_pre in PAIR_PREFIXES:
        for name, sites in by_name.items():
            if not name.startswith(enc_pre):
                continue
            stem = name[len(enc_pre) :]
            if name == enc_pre:
                stem = ""
            dec_name = dec_pre + stem
            for a in sites:
                for b in by_name.get(dec_name, []):
                    key = (a.file_path, a.name, b.file_path, b.name)
                    if key in seen:
                        continue
                    seen.add(key)
                    pairs.append(
                        TwoSitePair(
                            a=a,
                            b=b,
                            relation=f"{enc_pre}/{dec_pre}",
                            no_import_edge=a.package != b.package,
                        )
                    )
    return pairs


def pick_shared_type_pairs(
    index: Path | sqlite3.Connection | str,
    *,
    require_no_import: bool = True,
    skip_names: Sequence[str] = (),
) -> list[TwoSitePair]:
    """Two production funcs in different packages that reference the same type."""
    skip = set(skip_names)
    con, close = _open(index)
    try:
        rows = con.execute(
            """
            SELECT t.name, t.file_path,
                   s.name, s.kind, s.file_path, s.start_line
            FROM edges e
            JOIN nodes s ON s.id = e.source
            JOIN nodes t ON t.id = e.target
            WHERE e.kind IN ('references', 'instantiates')
              AND t.kind IN ('struct', 'type_alias', 'interface')
              AND s.kind IN ('function', 'method')
            """
        ).fetchall()
    finally:
        if close:
            con.close()
    imports = package_imports(index)
    by_type: dict[tuple[str, str], list[Site]] = defaultdict(list)
    for tname, tfp, sname, skind, sfp, sline in rows:
        sfp = (sfp or "").replace("\\", "/")
        if _skip(sfp) or is_test_symbol(sname) or sname in skip:
            continue
        site = Site(
            name=sname,
            kind=skind,
            file_path=sfp,
            package=go_package(sfp),
            start_line=int(sline or 0),
        )
        by_type[(tname, (tfp or "").replace("\\", "/"))].append(site)
    pairs: list[TwoSitePair] = []
    seen: set[tuple[str, str, str]] = set()
    for (tname, tfp), sites in by_type.items():
        uniq: dict[str, Site] = {}
        for s in sites:
            uniq.setdefault(s.package, s)
        pkgs = list(uniq)
        for i, pa in enumerate(pkgs):
            for pb in pkgs[i + 1 :]:
                if require_no_import and not no_import_edge(imports, pa, pb):
                    continue
                a, b = uniq[pa], uniq[pb]
                key = (tname, a.name, b.name)
                if key in seen:
                    continue
                seen.add(key)
                pairs.append(
                    TwoSitePair(
                        a=a,
                        b=b,
                        relation="shared_type",
                        shared_type=f"{tname}@{tfp}",
                        no_import_edge=no_import_edge(imports, pa, pb),
                    )
                )
    return pairs


def pick_two_site_pair(
    index: Path | sqlite3.Connection | str,
    *,
    skip_names: Sequence[str] = (),
    require_no_import: bool = False,
    cross_package: bool = True,
) -> TwoSitePair | None:
    """Pick a coordinated two-site pair. Prefer encode/decode, then shared types."""
    named = pick_encode_decode_pairs(index, skip_names=skip_names)
    typed = pick_shared_type_pairs(
        index, require_no_import=require_no_import, skip_names=skip_names
    )
    candidates = [*named, *typed]
    for pair in candidates:
        if cross_package and pair.a.package == pair.b.package:
            continue
        if require_no_import and not pair.no_import_edge:
            continue
        return pair
    for pair in candidates:
        if require_no_import and not pair.no_import_edge:
            continue
        return pair
    return None


def pick_three_site(
    index: Path | sqlite3.Connection | str,
    *,
    skip_names: Sequence[str] = (),
) -> ThreeSite | None:
    """Three functions that share a name family (Compose/Extract/From, Encode/Decode/Desc)."""
    skip = set(skip_names)
    by_pkg: dict[str, list[Site]] = defaultdict(list)
    for site in load_nodes(index):
        if site.name not in skip:
            by_pkg[site.package].append(site)
    for sites in by_pkg.values():
        names = {s.name: s for s in sites}
        if {"ComposeTS", "ExtractPhysical", "GoTimeToTS"} <= names.keys():
            return ThreeSite(
                sites=(names["ComposeTS"], names["ExtractPhysical"], names["GoTimeToTS"]),
                relation="hybrid_ts",
            )
        if {"Compose", "Extract", "FromTime"} <= names.keys():
            return ThreeSite(
                sites=(names["Compose"], names["Extract"], names["FromTime"]),
                relation="compose_extract_fromtime",
            )
        # EncodeX + DecodeX + EncodeXDesc in the same package.
        for s in sites:
            if not s.name.startswith("Encode") or s.name.endswith("Desc"):
                continue
            stem = s.name[len("Encode") :]
            dec = names.get("Decode" + stem)
            desc = names.get("Encode" + stem + "Desc")
            if dec and desc:
                return ThreeSite(sites=(s, dec, desc), relation="encode_decode_desc")
    return None


def pick_decoy(
    index: Path | sqlite3.Connection | str,
    path: Sequence[str],
    *,
    true_cause: str = "",
) -> Decoy | None:
    """Unmodified production function on the test→cause path a solver might inspect first.

    ``path`` is ``[cause, ..., TestFoo]`` from ``shortest_caller_path``. The decoy is
    the first intermediate production function after the cause (or the cause's
    nearest caller if the cause is the only production site).
    """
    sites = plausible_fix_sites(list(path))
    cause = true_cause or (sites[0] if sites else "")
    intermediates = [n for n in sites if n != cause]
    if not intermediates:
        return None
    name = intermediates[0]
    file_path = package = ""
    for site in load_nodes(index):
        if site.name == name:
            file_path, package = site.file_path, site.package
            break
    return Decoy(
        name=name,
        file_path=file_path,
        package=package,
        why=(
            f"{name} sits on the call path {list(path)} between the symptom and "
            f"{cause}; a competent engineer would inspect it first because it "
            "already implements overlapping responsibilities."
        ),
        path=tuple(path),
    )


def find_guard_tests(
    index: Path | sqlite3.Connection | str,
    symbol: str,
    *,
    exclude: Sequence[str] = (),
    limit: int = 8,
) -> list[str]:
    """Existing tests that reach ``symbol`` (caller BFS) and are not in ``exclude``."""
    banned = set(exclude)
    con, close = _open(index)
    try:
        starts = [
            r[0]
            for r in con.execute(
                "SELECT id FROM nodes WHERE name = ? AND kind IN ('function','method')",
                [symbol],
            )
        ]
        callers: dict[str, list[str]] = defaultdict(list)
        name_of = {nid: n for nid, n in con.execute("SELECT id, name FROM nodes")}
        for src, tgt in con.execute("SELECT source, target FROM edges WHERE kind = 'calls'"):
            callers[tgt].append(src)
        seen = set(starts)
        q = deque(starts)
        found: list[str] = []
        while q and len(found) < limit:
            nid = q.popleft()
            nm = name_of.get(nid, "")
            if is_test_symbol(nm) and nm not in banned and nm not in found:
                found.append(nm)
            for c in callers.get(nid, []):
                if c not in seen:
                    seen.add(c)
                    q.append(c)
        return found
    finally:
        if close:
            con.close()


def find_sparse_branches(
    coverprofile: Path | str,
    *,
    max_count: int = 1,
) -> list[dict[str, object]]:
    """Blocks in a Go coverprofile executed at most ``max_count`` times.

    Also accepts ``go tool cover -func`` output via ``parse_cover_func``.
    """
    path = Path(coverprofile)
    text = path.read_text(encoding="utf-8", errors="replace")
    sparse: list[dict[str, object]] = []
    if text.startswith("mode:"):
        for line in text.splitlines()[1:]:
            if not line.strip():
                continue
            loc, rest = line.split(" ", 1)
            stmts_s, count_s = rest.split()
            count = int(count_s)
            if count <= max_count:
                sparse.append(
                    {
                        "loc": loc,
                        "stmts": int(stmts_s),
                        "count": count,
                    }
                )
        return sparse
    cover = parse_cover_func(path)
    for (file_part, func), pct in cover.items():
        if pct <= 0:
            continue
        if pct < 50.0:
            sparse.append({"file": file_part, "func": func, "cover_pct": pct})
    return sparse


def select_construction(
    repo: Path | str,
    *,
    rung: int,
    hops: int = 4,
    sites: int = 1,
    decoys: int = 0,
    skip_names: Sequence[str] = (),
    coverprofile: Path | str | None = None,
) -> Construction:
    """Pick graph sites matching a ladder rung. Does not inject a bug."""
    repo = Path(repo)
    index = codegraph_db(repo)
    if not index.exists():
        raise FileNotFoundError(f"no codegraph index at {index}")
    out = Construction(rung=rung, hops=hops, sites=sites, decoys=decoys)
    skip = list(skip_names)
    if rung <= 1 or (rung == 1 and sites <= 1):
        site = pick_contract_drift_site(index, min_hops=hops, skip_names=skip)
        if site:
            out.picked_sites = [site]
            out.notes = f"rung1 contract-drift hops>={hops}"
    if rung == 2 or sites == 2:
        pair = pick_two_site_pair(index, skip_names=skip, require_no_import=False)
        if pair:
            out.pair = pair
            out.picked_sites = [pair.a, pair.b]
            out.notes = f"two-site {pair.relation}"
    if rung == 3:
        if sites >= 3:
            triple = pick_three_site(index, skip_names=skip)
            if triple:
                out.triple = triple
                out.picked_sites = list(triple.sites)
                out.notes = f"three-site {triple.relation}"
        if not out.picked_sites:
            pair = pick_two_site_pair(
                index, skip_names=skip, require_no_import=True, cross_package=True
            )
            if pair:
                out.pair = pair
                out.picked_sites = [pair.a, pair.b]
                out.notes = (
                    f"cross-package no-import via {pair.shared_type or pair.relation}"
                )
    if rung >= 4 and not out.picked_sites:
        pair = pick_two_site_pair(index, skip_names=skip, require_no_import=True)
        if pair:
            out.pair = pair
            out.picked_sites = [pair.a, pair.b]
            out.notes = "rung>=4 fallback two-site"
    if decoys and out.picked_sites:
        cause = out.picked_sites[0].name
        path = shortest_caller_path(repo, cause, {"TestSumClamped", "TestAdd"})
        if path is None:
            # any test name in the index
            con = sqlite3.connect(str(index))
            try:
                tests = {
                    r[0]
                    for r in con.execute(
                        "SELECT name FROM nodes WHERE name GLOB 'Test*' LIMIT 50"
                    )
                }
            finally:
                con.close()
            path = shortest_caller_path(repo, cause, tests)
        decoy = pick_decoy(index, path or [cause], true_cause=cause)
        if decoy:
            out.decoy_list = [decoy]
    if out.picked_sites:
        out.guard_tests = find_guard_tests(index, out.picked_sites[0].name)
    if coverprofile:
        out.sparse = find_sparse_branches(coverprofile)[:20]
    return out


def _json(obj: object) -> str:
    def _default(o: object) -> object:
        if hasattr(o, "__dataclass_fields__"):
            return asdict(o)
        if isinstance(o, Path):
            return str(o)
        if isinstance(o, set):
            return sorted(o)
        raise TypeError(type(o))

    return json.dumps(obj, indent=2, default=_default)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Codegraph difficulty knobs → site pick")
    parser.add_argument("--repo", required=True)
    parser.add_argument("--rung", type=int, default=1)
    parser.add_argument("--hops", type=int, default=4)
    parser.add_argument("--sites", type=int, default=1)
    parser.add_argument("--decoys", type=int, default=0)
    parser.add_argument("--skip", nargs="*", default=[])
    parser.add_argument("--coverprofile", default=None)
    args = parser.parse_args(argv)
    result = select_construction(
        args.repo,
        rung=args.rung,
        hops=args.hops,
        sites=args.sites,
        decoys=args.decoys,
        skip_names=args.skip,
        coverprofile=args.coverprofile,
    )
    print(_json(result))
    return 0 if result.picked_sites else 2


if __name__ == "__main__":
    raise SystemExit(main())

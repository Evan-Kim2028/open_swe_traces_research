"""Codegraph-guided Go bug injection: host selection, candidates, validation.

Pilot workflow for injecting 1–5 line semantic bugs at definitions whose
``codegraph callers`` span ≥2 packages, then checking that fail-to-pass tests
land inside ``codegraph impact``'s file set.
"""

from __future__ import annotations

import argparse
import json
import os
import sqlite3
import subprocess
import time
from collections import defaultdict, deque
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any

from huggingface_hub import hf_hub_download

from openswe_traces.data import ROOT, connect_ephemeral

EXPERIMENT_DIR = ROOT / "experiments" / "codegraph_bugs"
TASK_DIFFICULTY = ROOT / "outputs" / "task_difficulty.parquet"
PROXY_FEATURES = ROOT / "outputs" / "proxy_features.parquet"
SWE_REBENCH_REPO = "nebius/SWE-rebench-V2"
SWE_REBENCH_FILE = "data/train-00000-of-00001.parquet"
DEFAULT_TOP_N = 10
MIN_MID_SHARE = 0.20
CALLERS_LIMIT = 200
IMPACT_DEPTH = 4

TEST_SYMBOL_PREFIXES = ("Test", "Benchmark", "Example", "Fuzz")


@dataclass(frozen=True)
class HostRepo:
    repo: str
    n_instances: int
    n_mid: int
    mid_share: float
    sample_instance_id: str


@dataclass
class Symbol:
    name: str
    kind: str
    file_path: str
    qualified_name: str
    start_line: int
    is_exported: bool


@dataclass
class Candidate:
    symbol: str
    kind: str
    file_path: str
    qualified_name: str
    start_line: int
    n_caller_files: int
    n_caller_packages: int
    caller_packages: list[str]
    cover_pct: float
    n_callers: int


@dataclass
class TestRun:
    ok: bool
    builds: bool
    failing_tests: list[str]
    failing_packages: list[str]
    output: str
    elapsed_s: float


def go_package(rel_path: str) -> str:
    """Go package ≈ directory containing the file (posix)."""
    p = Path(rel_path.replace("\\", "/"))
    parent = p.parent.as_posix()
    return parent if parent != "." else p.stem


def is_test_file(rel_path: str) -> bool:
    name = Path(rel_path).name
    return name.endswith("_test.go") or "/integration_tests/" in rel_path.replace("\\", "/")


def is_test_symbol(name: str) -> bool:
    return name.startswith(TEST_SYMBOL_PREFIXES)


def is_go_exported(name: str, is_exported_flag: bool | None = None) -> bool:
    """Go export = uppercase first rune. Methods often have is_exported=false in the index."""
    if not name or not name[0].isalpha():
        return bool(is_exported_flag)
    return name[0].isupper()


def select_go_hosts(
    *,
    n: int = 2,
    min_mid_share: float = MIN_MID_SHARE,
    task_parquet: Path = TASK_DIFFICULTY,
    proxy_parquet: Path = PROXY_FEATURES,
) -> list[HostRepo]:
    """Go repos with the most instances and mid-difficulty share ≥ ``min_mid_share``."""
    con = connect_ephemeral()
    try:
        rows = con.execute(
            """
            WITH td AS (
              SELECT instance_id, repo, language, difficulty_bucket
              FROM read_parquet(?)
              WHERE language = 'go'
            ),
            pf AS (
              SELECT DISTINCT instance_id
              FROM read_parquet(?)
              WHERE language = 'go'
            ),
            joined AS (
              SELECT td.*
              FROM td INNER JOIN pf USING (instance_id)
            ),
            agg AS (
              SELECT
                repo,
                count(*) AS n_instances,
                count(*) FILTER (WHERE difficulty_bucket = 'mid') AS n_mid,
                count(*) FILTER (WHERE difficulty_bucket = 'mid') * 1.0 / count(*) AS mid_share,
                min(instance_id) FILTER (WHERE difficulty_bucket = 'mid') AS sample_instance_id
              FROM joined
              GROUP BY repo
              HAVING mid_share >= ?
            )
            SELECT repo, n_instances, n_mid, mid_share, sample_instance_id
            FROM agg
            ORDER BY n_instances DESC
            LIMIT ?
            """,
            [str(task_parquet), str(proxy_parquet), min_mid_share, n],
        ).fetchall()
    finally:
        con.close()
    return [
        HostRepo(
            repo=r[0],
            n_instances=int(r[1]),
            n_mid=int(r[2]),
            mid_share=float(r[3]),
            sample_instance_id=r[4],
        )
        for r in rows
    ]


def fetch_rebench_rows(instance_ids: list[str]) -> list[dict[str, Any]]:
    """Load only the requested SWE-rebench-V2 rows (parquet is filtered, not fully decoded)."""
    path = hf_hub_download(
        repo_id=SWE_REBENCH_REPO,
        filename=SWE_REBENCH_FILE,
        repo_type="dataset",
    )
    con = connect_ephemeral()
    try:
        df = con.execute(
            """
            SELECT instance_id, repo, base_commit, image_name, language,
                   install_config, FAIL_TO_PASS, PASS_TO_PASS, interface
            FROM read_parquet(?)
            WHERE instance_id IN (SELECT UNNEST(?))
            """,
            [path, instance_ids],
        ).fetchdf()
    finally:
        con.close()
    records = df.to_dict(orient="records")
    for row in records:
        for k, v in list(row.items()):
            if hasattr(v, "tolist"):
                row[k] = v.tolist()
    return records


def env_spec_finding(rows: list[dict[str, Any]]) -> dict[str, Any]:
    """SWE-rebench-V2 ships hints + a prebuilt image name, not a Dockerfile in the row."""
    sample = rows[0] if rows else {}
    ic = sample.get("install_config") or {}
    return {
        "ships_dockerfile_in_parquet": False,
        "columns": [
            "image_name",
            "install_config.base_image_name",
            "install_config.docker_specs",
            "install_config.install",
            "install_config.test_cmd",
            "install_config.log_parser",
        ],
        "image_name_pattern": "docker.io/swerebenchv2/<repo>:<issue>-<sha7>",
        "dockerfiles_live_at": "https://github.com/SWE-rebench/SWE-rebench-V2 (not in the HF row)",
        "sample_image_name": sample.get("image_name"),
        "sample_base_image": ic.get("base_image_name") if isinstance(ic, dict) else None,
        "sample_test_cmd": ic.get("test_cmd") if isinstance(ic, dict) else None,
        "sample_install": ic.get("install") if isinstance(ic, dict) else None,
        "note": (
            "Each row has a prebuilt image_name plus install_config hints "
            "(base image alias, language version, shell install steps, test_cmd). "
            "No Dockerfile text is in the parquet."
        ),
    }


def _run(
    args: list[str],
    *,
    cwd: Path | None = None,
    timeout: int = 120,
) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    env["PATH"] = "/usr/local/go/bin:" + env.get("PATH", "")
    return subprocess.run(
        args,
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
        env=env,
    )


def codegraph_json(args: list[str], *, cwd: Path, timeout: int = 60) -> Any:
    proc = _run(["codegraph", *args, "-j"], cwd=cwd, timeout=timeout)
    if proc.returncode != 0:
        raise RuntimeError(
            f"codegraph {' '.join(args)} failed ({proc.returncode}): {proc.stderr[-2000:]}"
        )
    text = proc.stdout.strip() or "null"
    return json.loads(text)


def list_symbols(repo: Path, *, kinds: tuple[str, ...] = ("function", "method")) -> list[Symbol]:
    out: list[Symbol] = []
    for kind in kinds:
        rows = codegraph_json(["query", "-p", str(repo), "-k", kind, "-l", "20000", ""], cwd=repo)
        for item in rows or []:
            node = item.get("node") or item
            out.append(
                Symbol(
                    name=node["name"],
                    kind=node["kind"],
                    file_path=node["filePath"],
                    qualified_name=node.get("qualifiedName") or node["name"],
                    start_line=int(node.get("startLine") or 0),
                    is_exported=bool(node.get("isExported")),
                )
            )
    return out


def callers_of(repo: Path, symbol: str, *, limit: int = CALLERS_LIMIT) -> list[dict[str, Any]]:
    payload = codegraph_json(["callers", "-p", str(repo), "-l", str(limit), symbol], cwd=repo)
    if not payload:
        return []
    return list(payload.get("callers") or [])


def callers_of_definition(
    repo: Path,
    file_path: str,
    name: str,
    *,
    min_confidence: float = 0.85,
) -> list[dict[str, Any]]:
    """Callers of one definition (file + name). Drops low-confidence name collisions."""
    db = codegraph_db(repo)
    if not db.exists():
        return []
    file_path = file_path.replace("\\", "/")
    con = sqlite3.connect(str(db))
    try:
        rows = con.execute(
            """
            SELECT c.name, c.kind, c.file_path, c.start_line, e.metadata
            FROM nodes def
            JOIN edges e ON e.target = def.id AND e.kind = 'calls'
            JOIN nodes c ON c.id = e.source
            WHERE def.name = ? AND replace(def.file_path, '\\', '/') = ?
              AND def.kind IN ('function', 'method')
            """,
            [name, file_path],
        ).fetchall()
    finally:
        con.close()
    out: list[dict[str, Any]] = []
    for name_, kind, fp, line, meta in rows:
        conf = 1.0
        if meta:
            try:
                conf = float(json.loads(meta).get("confidence", 1.0))
            except (json.JSONDecodeError, TypeError, ValueError):
                conf = 1.0
        if conf < min_confidence:
            continue
        out.append({"name": name_, "kind": kind, "filePath": fp, "startLine": line, "confidence": conf})
    return out


def impact_of(repo: Path, symbol: str, *, depth: int = IMPACT_DEPTH) -> dict[str, Any]:
    return codegraph_json(
        ["impact", "-p", str(repo), "-d", str(depth), symbol],
        cwd=repo,
    ) or {}


def impact_files(payload: dict[str, Any]) -> set[str]:
    files: set[str] = set()
    for item in payload.get("affected") or []:
        fp = item.get("filePath")
        if fp:
            files.add(fp.replace("\\", "/"))
    return files


def parse_cover_func(path: Path) -> dict[tuple[str, str], float]:
    """Parse ``go tool cover -func`` lines into (basename.go, func) -> percent."""
    out: dict[tuple[str, str], float] = {}
    for line in path.read_text().splitlines():
        if line.startswith("total:") or "\t" not in line:
            continue
        left, pct = line.rsplit("\t", 1)
        pct = pct.strip().rstrip("%")
        try:
            value = float(pct)
        except ValueError:
            continue
        file_and_line, func = left.split("\t", 1)
        func = func.strip()
        file_part = file_and_line.split(":")[0]
        base = Path(file_part).name
        out[(base, func)] = value
        out[(file_part.replace("\\", "/"), func)] = value
    return out


def _cover_pct(cover: dict[tuple[str, str], float], file_path: str, name: str) -> float:
    file_path = file_path.replace("\\", "/")
    base = Path(file_path).name
    for key in ((file_path, name), (base, name)):
        if key in cover and cover[key] > 0:
            return cover[key]
    # Methods may appear as Type.Method in cover output.
    for (f, fn), pct in cover.items():
        if pct <= 0:
            continue
        if (fn == name or fn.endswith("." + name)) and (
            f == file_path or f == base or f.endswith("/" + base)
        ):
            return pct
    return 0.0


def rank_candidates(
    repo: Path,
    *,
    cover: dict[tuple[str, str], float],
    top_n: int = DEFAULT_TOP_N,
    skip_dir_prefixes: tuple[str, ...] = ("examples/", "integration_tests/"),
) -> list[Candidate]:
    symbols = [
        s
        for s in list_symbols(repo)
        if is_go_exported(s.name, s.is_exported)
        and not is_test_file(s.file_path)
        and not is_test_symbol(s.name)
        and not any(s.file_path.replace("\\", "/").startswith(p) for p in skip_dir_prefixes)
    ]
    ranked: list[Candidate] = []
    for chosen in symbols:
        raw_callers = callers_of_definition(repo, chosen.file_path, chosen.name)
        caller_files = sorted(
            {
                c["filePath"].replace("\\", "/")
                for c in raw_callers
                if c.get("filePath")
                and not any(
                    c["filePath"].replace("\\", "/").startswith(p) for p in skip_dir_prefixes
                )
            }
        )
        packages = sorted({go_package(f) for f in caller_files})
        if len(packages) < 2:
            continue
        pct = _cover_pct(cover, chosen.file_path, chosen.name)
        if pct <= 0:
            continue
        ranked.append(
            Candidate(
                symbol=chosen.name,
                kind=chosen.kind,
                file_path=chosen.file_path,
                qualified_name=chosen.qualified_name,
                start_line=chosen.start_line,
                n_caller_files=len(caller_files),
                n_caller_packages=len(packages),
                caller_packages=packages,
                cover_pct=pct,
                n_callers=len(raw_callers),
            )
        )
    ranked.sort(key=lambda c: (-c.n_caller_files, -c.n_caller_packages, -c.cover_pct, c.symbol))
    return ranked[:top_n]


def codegraph_db(repo: Path) -> Path:
    return repo / ".codegraph" / "codegraph.db"


def _caller_graph(
    con: sqlite3.Connection,
) -> tuple[dict[str, list[str]], dict[str, str], dict[str, str]]:
    """Return (callers_map, file_of, name_of) from the codegraph sqlite."""
    callers_map: dict[str, list[str]] = defaultdict(list)
    for src, tgt in con.execute("SELECT source, target FROM edges WHERE kind = 'calls'"):
        callers_map[tgt].append(src)
    file_of = {
        nid: fp.replace("\\", "/")
        for nid, fp in con.execute("SELECT id, file_path FROM nodes")
    }
    name_of = {nid: name for nid, name in con.execute("SELECT id, name FROM nodes")}
    return callers_map, file_of, name_of


def hops_to_files(repo: Path, symbol: str, target_files: set[str], *, max_depth: int = 8) -> int | None:
    """Shortest calls-edge path from ``symbol`` to any node in ``target_files`` (caller direction)."""
    db = codegraph_db(repo)
    if not db.exists():
        return None
    targets = {f.replace("\\", "/") for f in target_files}
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
            return None
        callers_map, file_of, _name_of = _caller_graph(con)
        seen: set[str] = set(starts)
        q: deque[tuple[str, int]] = deque((s, 0) for s in starts)
        while q:
            nid, dist = q.popleft()
            if file_of.get(nid) in targets and dist > 0:
                return dist
            if dist >= max_depth:
                continue
            for caller in callers_map.get(nid, []):
                if caller not in seen:
                    seen.add(caller)
                    q.append((caller, dist + 1))
        return None
    finally:
        con.close()


def hops_to_test_names(
    repo: Path,
    symbol: str,
    test_names: set[str],
    *,
    max_depth: int = 12,
) -> int | None:
    """Shortest reverse-``calls`` hops from ``symbol`` to any named test function."""
    db = codegraph_db(repo)
    if not db.exists() or not test_names:
        return None
    wanted = set(test_names)
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
            return None
        callers_map, _file_of, name_of = _caller_graph(con)
        seen: set[str] = set(starts)
        q: deque[tuple[str, int]] = deque((s, 0) for s in starts)
        while q:
            nid, dist = q.popleft()
            if dist > 0 and name_of.get(nid) in wanted:
                return dist
            if dist >= max_depth:
                continue
            for caller in callers_map.get(nid, []):
                if caller not in seen:
                    seen.add(caller)
                    q.append((caller, dist + 1))
        return None
    finally:
        con.close()


def shortest_caller_path(
    repo: Path,
    symbol: str,
    test_names: set[str],
    *,
    max_depth: int = 12,
) -> list[str] | None:
    """Function names on the shortest reverse-calls path from ``symbol`` to a named test.

    The path is ``[symbol, ..., TestFoo]``. Intermediate production functions are
    plausible fix sites a solver might edit.
    """
    db = codegraph_db(repo)
    if not db.exists() or not test_names:
        return None
    wanted = set(test_names)
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
            return None
        callers_map, _file_of, name_of = _caller_graph(con)
        parent: dict[str, str | None] = {s: None for s in starts}
        q: deque[tuple[str, int]] = deque((s, 0) for s in starts)
        seen: set[str] = set(starts)
        found: str | None = None
        while q:
            nid, dist = q.popleft()
            if dist > 0 and name_of.get(nid) in wanted:
                found = nid
                break
            if dist >= max_depth:
                continue
            for caller in callers_map.get(nid, []):
                if caller not in seen:
                    seen.add(caller)
                    parent[caller] = nid
                    q.append((caller, dist + 1))
        if found is None:
            return None
        chain: list[str] = []
        cur: str | None = found
        while cur is not None:
            chain.append(name_of.get(cur) or cur)
            cur = parent.get(cur)
        chain.reverse()
        return chain
    finally:
        con.close()


def plausible_fix_sites(path: list[str] | None) -> list[str]:
    """Production functions on a test<-cause path a solver might reasonably edit."""
    if not path:
        return []
    skip_pfx = TEST_SYMBOL_PREFIXES + ("Setup", "TearDown", "Before", "After")
    return [n for n in path if n and not n.startswith(skip_pfx)]


def parse_go_test_output(text: str) -> tuple[list[str], list[str]]:
    """Return (failing test names, failing package import paths)."""
    tests: list[str] = []
    packages: list[str] = []
    for line in text.splitlines():
        if line.startswith("--- FAIL:"):
            parts = line.split()
            if len(parts) >= 3:
                tests.append(parts[2])
        elif line.startswith("FAIL\t"):
            pkg = line.split("\t")[1].split()[0]
            packages.append(pkg)
    return tests, packages


def run_go(
    repo: Path,
    args: list[str],
    *,
    timeout: int = 900,
) -> TestRun:
    t0 = time.time()
    proc = _run(["go", *args], cwd=repo, timeout=timeout)
    elapsed = time.time() - t0
    text = (proc.stdout or "") + (proc.stderr or "")
    tests, packages = parse_go_test_output(text)
    compile_fail = any(
        s in text.lower()
        for s in ("build failed", "syntax error", "undefined:", "cannot find package")
    ) and "--- fail:" not in text.lower()
    return TestRun(
        ok=proc.returncode == 0,
        builds=proc.returncode == 0 or (not compile_fail and bool(tests or packages)),
        failing_tests=tests,
        failing_packages=packages,
        output=text,
        elapsed_s=elapsed,
    )


def go_build(repo: Path, *, timeout: int = 180) -> TestRun:
    t0 = time.time()
    proc = _run(["go", "build", "./..."], cwd=repo, timeout=timeout)
    text = (proc.stdout or "") + (proc.stderr or "")
    return TestRun(
        ok=proc.returncode == 0,
        builds=proc.returncode == 0,
        failing_tests=[],
        failing_packages=[],
        output=text,
        elapsed_s=time.time() - t0,
    )


def go_test(repo: Path, packages: list[str] | None = None, *, timeout: int = 900) -> TestRun:
    args = ["test", *(packages or ["./..."]), "-count=1"]
    return run_go(repo, args, timeout=timeout)


def apply_patch(repo: Path, patch: Path) -> None:
    patch = Path(patch).resolve()
    proc = _run(["git", "apply", str(patch)], cwd=repo)
    if proc.returncode != 0:
        proc = _run(["git", "apply", "--reject", str(patch)], cwd=repo)
        if proc.returncode != 0:
            raise RuntimeError(f"git apply {patch} failed: {proc.stderr}")


def revert_patch(repo: Path, patch: Path) -> None:
    patch = Path(patch).resolve()
    proc = _run(["git", "apply", "-R", str(patch)], cwd=repo)
    if proc.returncode != 0:
        _run(["git", "checkout", "--", "."], cwd=repo)


@dataclass
class BugValidation:
    symbol: str
    repo: str
    patch: str
    builds: bool
    f2p_tests: list[str]
    f2p_count: int
    impact_files: list[str]
    f2p_inside_impact: bool
    collateral_failures: list[str]
    flaky: bool
    hops: int | None
    n_impact_files: int
    elapsed_s: float
    notes: str = ""


def failing_test_files(repo: Path, failing_tests: list[str]) -> set[str]:
    """Map test names to files via the codegraph index."""
    db = codegraph_db(repo)
    if not db.exists() or not failing_tests:
        return set()
    con = sqlite3.connect(str(db))
    try:
        files: set[str] = set()
        for name in failing_tests:
            rows = con.execute(
                "SELECT DISTINCT file_path FROM nodes WHERE name = ?",
                [name],
            ).fetchall()
            for (fp,) in rows:
                files.add(fp.replace("\\", "/"))
        return files
    finally:
        con.close()


def validate_bug(
    repo: Path,
    patch: Path,
    symbol: str,
    *,
    test_timeout: int = 900,
    baseline_failing: list[str] | None = None,
) -> BugValidation:
    t0 = time.time()
    apply_patch(repo, patch)
    try:
        built = go_build(repo)
        if not built.builds:
            return BugValidation(
                symbol=symbol,
                repo=repo.name,
                patch=str(patch),
                builds=False,
                f2p_tests=[],
                f2p_count=0,
                impact_files=[],
                f2p_inside_impact=False,
                collateral_failures=[],
                flaky=False,
                hops=None,
                n_impact_files=0,
                elapsed_s=time.time() - t0,
                notes=built.output[-1500:],
            )
        first = go_test(repo, timeout=test_timeout)
        baseline = set(baseline_failing or [])
        f2p = [t for t in first.failing_tests if t not in baseline]
        impact = impact_of(repo, symbol)
        i_files = impact_files(impact)
        t_files = failing_test_files(repo, f2p)
        inside = bool(t_files) and t_files <= i_files
        collateral = sorted(
            name
            for name in f2p
            if failing_test_files(repo, [name])
            and not (failing_test_files(repo, [name]) <= i_files)
        )
        flaky = False
        if f2p:
            pkgs = first.failing_packages or None
            second = go_test(repo, pkgs, timeout=test_timeout)
            second_f2p = {t for t in second.failing_tests if t not in baseline}
            flaky = second_f2p != set(f2p)
        hops = hops_to_files(repo, symbol, t_files) if t_files else None
        return BugValidation(
            symbol=symbol,
            repo=repo.name,
            patch=str(patch),
            builds=True,
            f2p_tests=f2p,
            f2p_count=len(f2p),
            impact_files=sorted(i_files),
            f2p_inside_impact=inside,
            collateral_failures=collateral,
            flaky=flaky,
            hops=hops,
            n_impact_files=len(i_files),
            elapsed_s=time.time() - t0,
            notes="" if f2p else "build ok but no test failures (bug not caught)",
        )
    finally:
        revert_patch(repo, patch)


def _print_json(obj: Any) -> None:
    def _default(o: Any) -> Any:
        if hasattr(o, "__dataclass_fields__"):
            return asdict(o)
        if isinstance(o, Path):
            return str(o)
        if hasattr(o, "tolist"):
            return o.tolist()
        raise TypeError(type(o))

    print(json.dumps(obj, indent=2, default=_default))


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Codegraph-guided Go bug injection helpers")
    sub = parser.add_subparsers(dest="cmd", required=True)

    p_hosts = sub.add_parser("select-hosts", help="Pick Go host repos from difficulty parquet")
    p_hosts.add_argument("-n", type=int, default=2)
    p_hosts.add_argument("--min-mid-share", type=float, default=MIN_MID_SHARE)

    p_env = sub.add_parser("fetch-env", help="Fetch SWE-rebench-V2 rows for instance ids")
    p_env.add_argument("instance_ids", nargs="+")

    p_cand = sub.add_parser("candidates", help="Rank cross-package exported symbols")
    p_cand.add_argument("repo")
    p_cand.add_argument("--cover-func", required=True, help="go tool cover -func output")
    p_cand.add_argument("--top", type=int, default=DEFAULT_TOP_N)

    p_imp = sub.add_parser("impact", help="codegraph impact files for a symbol")
    p_imp.add_argument("repo")
    p_imp.add_argument("symbol")

    p_val = sub.add_parser("validate", help="Apply a patch, test, check impact set, revert")
    p_val.add_argument("repo")
    p_val.add_argument("patch")
    p_val.add_argument("symbol")
    p_val.add_argument("--timeout", type=int, default=900)
    p_val.add_argument("--baseline-failing", nargs="*", default=[])

    args = parser.parse_args(argv)
    if args.cmd == "select-hosts":
        _print_json(select_go_hosts(n=args.n, min_mid_share=args.min_mid_share))
        return 0
    if args.cmd == "fetch-env":
        rows = fetch_rebench_rows(args.instance_ids)
        _print_json({"rows": rows, "env_spec": env_spec_finding(rows)})
        return 0
    if args.cmd == "candidates":
        cover = parse_cover_func(Path(args.cover_func))
        _print_json(rank_candidates(Path(args.repo), cover=cover, top_n=args.top))
        return 0
    if args.cmd == "impact":
        payload = impact_of(Path(args.repo), args.symbol)
        _print_json({"files": sorted(impact_files(payload)), "raw": payload})
        return 0
    if args.cmd == "validate":
        result = validate_bug(
            Path(args.repo),
            Path(args.patch),
            args.symbol,
            test_timeout=args.timeout,
            baseline_failing=list(args.baseline_failing),
        )
        _print_json(result)
        return 0
    return 1


if __name__ == "__main__":
    raise SystemExit(main())

#!/usr/bin/env python3
"""Judge GRAPH vs NOGRAPH revive bug injections. Writes out/judge.json."""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
from concurrent.futures import ProcessPoolExecutor, as_completed
from pathlib import Path

from openswe_traces.synth.codegraph_bugs import (
    failing_test_files,
    go_package,
    hops_to_files,
    impact_files,
    impact_of,
    parse_go_test_output,
)

ROOT = Path(__file__).resolve().parents[2]
EXP = Path(__file__).resolve().parent
GRAPH = EXP / "repos" / "revive-graph"
NOGRAPH = EXP / "repos" / "revive-nograph"
OUT = EXP / "out"
TEST_TIMEOUT = 600
N_WORKERS = 4

# Gold tree is revive-graph minus bugs/ (same source as nograph).
GOLD_EXCLUDE = {".codegraph", "bugs", "START.txt"}


def _run(
    args: list[str],
    *,
    cwd: Path,
    timeout: int,
    env: dict[str, str] | None = None,
) -> subprocess.CompletedProcess[str]:
    e = os.environ.copy()
    e["PATH"] = "/usr/local/go/bin:" + e.get("PATH", "")
    if env:
        e.update(env)
    return subprocess.run(
        args,
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
        env=e,
    )


def parse_index(path: Path) -> list[dict]:
    rows: list[dict] = []
    for line in path.read_text().splitlines():
        if not line.startswith("|"):
            continue
        cells = [c.strip() for c in line.strip("|").split("|")]
        if len(cells) < 7:
            continue
        if cells[0].lower() in {"symbol", "---"} or set(cells[0]) <= {"-"}:
            continue
        try:
            minutes = int(cells[6])
        except ValueError:
            continue
        rows.append(
            {
                "symbol": cells[0],
                "file": cells[1],
                "builder_f2p": cells[2],
                "builder_xf": cells[3][:1].upper() == "Y",
                "builder_hops": cells[4],
                "builder_checks": cells[5],
                "minutes": minutes,
            }
        )
    return rows


def patch_paths(patch: Path) -> list[str]:
    files: list[str] = []
    for line in patch.read_text().splitlines():
        if line.startswith("diff --git "):
            parts = line.split()
            # diff --git a/foo b/foo
            raw = parts[2][2:] if parts[2].startswith("a/") else parts[2]
            files.append(raw)
    return files


def touches_test_file(patch: Path) -> bool:
    return any(Path(p).name.endswith("_test.go") for p in patch_paths(patch))


def apply_patch(repo: Path, patch: Path) -> tuple[bool, str]:
    # Do not use `git apply`: a parent .git (this research repo) makes it
    # apply against the wrong tree. patch(1) is cwd-relative.
    proc = _run(
        ["patch", "-p1", "--forward", "--batch", "--reject-file=-", "-i", str(patch)],
        cwd=repo,
        timeout=30,
    )
    if proc.returncode == 0:
        return True, ""
    err = (proc.stderr or "") + (proc.stdout or "")
    return False, err[-2000:]


def go_build(repo: Path) -> tuple[bool, str]:
    proc = _run(["go", "build", "./..."], cwd=repo, timeout=180)
    text = (proc.stdout or "") + (proc.stderr or "")
    return proc.returncode == 0, text[-2000:]


def go_test(repo: Path, extra: list[str] | None = None, *, timeout: int = TEST_TIMEOUT) -> dict:
    args = ["go", "test", "./...", "-count=1", f"-timeout={timeout}s"]
    if extra:
        args.extend(extra)
    try:
        proc = _run(args, cwd=repo, timeout=timeout + 15)
    except subprocess.TimeoutExpired:
        return {
            "ok": False,
            "timeout": True,
            "tests": [],
            "packages": [],
            "output": "TIMEOUT",
        }
    text = (proc.stdout or "") + (proc.stderr or "")
    tests, packages = parse_go_test_output(text)
    return {
        "ok": proc.returncode == 0,
        "timeout": False,
        "tests": tests,
        "packages": packages,
        "output": text[-4000:],
    }


def go_test_f2p(repo: Path, tests: list[str], *, timeout: int = TEST_TIMEOUT) -> dict:
    parents = sorted({t.split("/")[0] for t in tests if t})
    if not parents:
        return go_test(repo, timeout=timeout)
    run = "^(" + "|".join(re.escape(p) for p in parents) + ")$"
    return go_test(repo, ["-run", run], timeout=timeout)


def copy_gold(dest: Path) -> None:
    if dest.exists():
        shutil.rmtree(dest)
    dest.mkdir(parents=True)
    for item in GRAPH.iterdir():
        if item.name in GOLD_EXCLUDE:
            continue
        target = dest / item.name
        if item.is_dir():
            shutil.copytree(item, target, symlinks=True, ignore=shutil.ignore_patterns(".codegraph"))
        else:
            shutil.copy2(item, target)


def f2p_files_for(tests: list[str]) -> list[str]:
    names = sorted({t.split("/")[0] for t in tests if t})
    files = failing_test_files(GRAPH, names)
    return sorted(files)


def hops_for(symbol: str, changed_file: str, target_files: set[str]) -> int | None:
    # Prefer file-scoped start nodes to avoid Name/Visit collisions.
    db = GRAPH / ".codegraph" / "codegraph.db"
    if not db.exists() or not target_files:
        return hops_to_files(GRAPH, symbol, target_files)
    import sqlite3
    from collections import defaultdict, deque

    con = sqlite3.connect(str(db))
    try:
        changed = changed_file.replace("\\", "/")
        starts = [
            r[0]
            for r in con.execute(
                """
                SELECT id FROM nodes
                WHERE name = ? AND kind IN ('function','method')
                  AND replace(file_path, '\\', '/') = ?
                """,
                [symbol, changed],
            ).fetchall()
        ]
        if not starts:
            return hops_to_files(GRAPH, symbol, target_files)
        callers_map: dict[str, list[str]] = defaultdict(list)
        for src, tgt in con.execute("SELECT source, target FROM edges WHERE kind = 'calls'"):
            callers_map[tgt].append(src)
        file_of = {
            nid: fp.replace("\\", "/")
            for nid, fp in con.execute("SELECT id, file_path FROM nodes")
        }
        targets = {f.replace("\\", "/") for f in target_files}
        seen = set(starts)
        q = deque((s, 0) for s in starts)
        while q:
            nid, dist = q.popleft()
            if file_of.get(nid) in targets and dist > 0:
                return dist
            if dist >= 12:
                continue
            for caller in callers_map.get(nid, []):
                if caller not in seen:
                    seen.add(caller)
                    q.append((caller, dist + 1))
        return None
    finally:
        con.close()


def score_one(payload: dict) -> dict:
    cond = payload["condition"]
    symbol = payload["symbol"]
    changed = payload["file"]
    minutes = payload["minutes"]
    bugs = Path(payload["bugs_dir"])
    work = Path(payload["work_dir"])
    patch = bugs / f"{symbol}.patch"
    alt = bugs / f"{symbol}.alt.patch"
    cheat = bugs / f"{symbol}.cheat.patch"

    failed: list[str] = []
    note = ""

    test_hunks = any(touches_test_file(p) for p in (patch, alt, cheat) if p.exists())
    missing = [n for n, p in (("patch", patch), ("alt", alt), ("cheat", cheat)) if not p.exists()]
    if missing:
        failed.append("missing:" + ",".join(missing))

    copy_gold(work)

    applied, err = apply_patch(work, patch)
    if not applied:
        failed.append("apply")
        return {
            "condition": cond,
            "symbol": symbol,
            "file": changed,
            "minutes": minutes,
            "builds": False,
            "f2p_tests": [],
            "f2p_files": [],
            "f2p_count": 0,
            "alt_pass": False,
            "cheat_fail": False,
            "flaky": False,
            "a4": False,
            "cross_file": False,
            "hops": None,
            "impact_files": [],
            "n_impact_files": 0,
            "failed": failed,
            "valid": False,
            "test_hunks": test_hunks,
            "notes": err,
        }

    builds, berr = go_build(work)
    if not builds:
        failed.append("build")
        note = berr

    run1 = {"ok": True, "tests": [], "packages": [], "timeout": False}
    run2 = run1
    if builds:
        run1 = go_test(work)
        if run1["timeout"]:
            failed.append("timeout")
        if run1["ok"]:
            failed.append("f2p=0")
        run2 = go_test(work)
        if run2["timeout"]:
            failed.append("timeout")

    f2p1 = set(run1.get("tests") or [])
    f2p2 = set(run2.get("tests") or [])
    flaky = bool(f2p1) and f2p1 != f2p2
    if flaky:
        failed.append("flaky")
    f2p = sorted(f2p1)
    t_files = f2p_files_for(f2p)

    # impact / hops always from the GRAPH index (same tree)
    try:
        imp = impact_of(GRAPH, symbol, depth=2)
        i_files = sorted(impact_files(imp))
    except Exception as exc:  # noqa: BLE001
        i_files = []
        note = (note + f" impact: {exc}")[-2000:]
    a4 = bool(t_files) and set(t_files) <= set(i_files)
    if f2p and not a4:
        failed.append("A4")
    hops = hops_for(symbol, changed, set(t_files)) if t_files else None

    changed_pkg = go_package(changed)
    xf = any(go_package(f) != changed_pkg for f in t_files) if t_files else False

    alt_pass = False
    cheat_fail = False
    if builds and f2p and "apply" not in failed:
        # alt on buggy tree
        copy_gold(work)
        apply_patch(work, patch)
        a_ok, a_err = apply_patch(work, alt)
        if not a_ok:
            failed.append("alt-apply")
            note = (note + " " + a_err)[-2000:]
        else:
            alt_run = go_test_f2p(work, f2p)
            alt_parents = {t.split("/")[0] for t in alt_run["tests"]}
            orig_parents = {t.split("/")[0] for t in f2p}
            alt_pass = not (alt_parents & orig_parents)
            if not alt_pass:
                failed.append("alt")

        copy_gold(work)
        apply_patch(work, patch)
        c_ok, c_err = apply_patch(work, cheat)
        if not c_ok:
            failed.append("cheat-apply")
            note = (note + " " + c_err)[-2000:]
        else:
            cheat_run = go_test_f2p(work, f2p)
            cheat_parents = {t.split("/")[0] for t in cheat_run["tests"]}
            orig_parents = {t.split("/")[0] for t in f2p}
            cheat_fail = (not cheat_run["ok"]) and bool(cheat_parents & orig_parents)
            if not cheat_fail:
                failed.append("cheat")

    if not f2p and builds and "f2p=0" not in failed and "timeout" not in failed:
        failed.append("f2p=0")

    valid = (
        builds
        and len(f2p) >= 1
        and alt_pass
        and cheat_fail
        and not flaky
        and a4
        and "apply" not in failed
        and "timeout" not in failed
    )
    return {
        "condition": cond,
        "symbol": symbol,
        "file": changed,
        "minutes": minutes,
        "builds": builds,
        "f2p_tests": f2p,
        "f2p_files": t_files,
        "f2p_count": len(f2p),
        "alt_pass": alt_pass,
        "cheat_fail": cheat_fail,
        "flaky": flaky,
        "a4": a4,
        "cross_file": xf,
        "hops": hops,
        "impact_files": i_files,
        "n_impact_files": len(i_files),
        "failed": failed,
        "valid": valid,
        "test_hunks": test_hunks,
        "notes": note.strip(),
    }


def jobs_for(condition: str, repo: Path) -> list[dict]:
    rows = parse_index(repo / "bugs" / "INDEX.md")
    out: list[dict] = []
    for i, row in enumerate(rows):
        out.append(
            {
                "condition": condition,
                "symbol": row["symbol"],
                "file": row["file"],
                "minutes": row["minutes"],
                "bugs_dir": str(repo / "bugs"),
                "work_dir": str(OUT / "work" / f"{condition}-{i}-{row['symbol']}"),
            }
        )
    return out


def main() -> int:
    OUT.mkdir(parents=True, exist_ok=True)
    (OUT / "work").mkdir(exist_ok=True)
    jobs = jobs_for("GRAPH", GRAPH) + jobs_for("NOGRAPH", NOGRAPH)
    results: list[dict] = []
    with ProcessPoolExecutor(max_workers=N_WORKERS) as ex:
        futs = {ex.submit(score_one, j): j for j in jobs}
        for fut in as_completed(futs):
            rec = fut.result()
            results.append(rec)
            print(
                f"{rec['condition']} {rec['symbol']}: valid={rec['valid']} "
                f"failed={rec['failed']} f2p={rec['f2p_count']} a4={rec['a4']} "
                f"xf={rec['cross_file']} hops={rec['hops']} impact={rec['n_impact_files']}",
                flush=True,
            )
    results.sort(key=lambda r: (0 if r["condition"] == "GRAPH" else 1, jobs_order(r, jobs)))
    (OUT / "judge.json").write_text(json.dumps(results, indent=2) + "\n")
    print(f"wrote {OUT / 'judge.json'} n={len(results)}")
    return 0


def jobs_order(rec: dict, jobs: list[dict]) -> int:
    for i, j in enumerate(jobs):
        if j["condition"] == rec["condition"] and j["symbol"] == rec["symbol"]:
            return i
    return 999


if __name__ == "__main__":
    raise SystemExit(main())

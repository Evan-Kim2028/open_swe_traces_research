#!/usr/bin/env python3
"""Judge GRAPH vs NOGRAPH feature-excision units. Writes out/judge2.json."""

from __future__ import annotations

import json
import os
import re
import shutil
import sqlite3
import subprocess
import time
from concurrent.futures import ProcessPoolExecutor, as_completed
from dataclasses import asdict, dataclass, field
from pathlib import Path

EXP = Path(__file__).resolve().parent
GRAPH = EXP / "repos" / "revive-graph"
NOGRAPH = EXP / "repos" / "revive-nograph"
OUT = EXP / "out"
TEST_TIMEOUT = 180
LISTED_TIMEOUT = 45
N_WORKERS = 3

# (file, name) from each unit's closure.md. entry is first listed in entry.md.
UNITS: dict[str, list[dict]] = {
    "GRAPH": [
        {
            "name": "ifelse-apply",
            "minutes": 5,
            "entry": ("internal/ifelse/rule.go", "Apply"),
            "funcs": [
                ("internal/ifelse/rule.go", "Apply"),
                ("internal/ifelse/rule.go", "Visit"),
                ("internal/ifelse/rule.go", "visitBody"),
                ("internal/ifelse/rule.go", "visitBlock"),
                ("internal/ifelse/rule.go", "visitIf"),
                ("internal/ifelse/rule.go", "checkRule"),
                ("internal/ifelse/target.go", "node"),
            ],
            "listed_tests": [
                "TestEarlyReturn",
                "TestIndentErrorFlow",
                "TestSuperfluousElse",
            ],
            "test_files": [
                "test/early_return_test.go",
                "test/indent_error_flow_test.go",
                "test/superfluous_else_test.go",
            ],
        },
        {
            "name": "revive-runner",
            "minutes": 3,
            "entry": ("revivelib/core.go", "Lint"),
            "funcs": [
                ("revivelib/core.go", "Lint"),
                ("revivelib/core.go", "getPackages"),
                ("revivelib/core.go", "normalizeSplit"),
                ("revivelib/pattern.go", "IsExclude"),
                ("revivelib/pattern.go", "Pattern"),
                ("revivelib/core.go", "Format"),
                ("config/config.go", "GetFormatter"),
                ("config/config.go", "getFormatters"),
            ],
            "listed_tests": [
                "TestReviveLint",
                "TestReviveFormat",
                "TestGetFormatter",
            ],
            "test_files": [
                "revivelib/core_test.go",
                "config/config_test.go",
            ],
        },
        {
            "name": "config-load",
            "minutes": 3,
            "entry": ("config/config.go", "GetConfig"),
            "funcs": [
                ("config/config.go", "GetConfig"),
                ("config/config.go", "parseConfig"),
                ("config/config.go", "configFieldsByNormalizedName"),
                ("config/config.go", "validateConfig"),
                ("config/config.go", "Normalize"),
                ("config/config.go", "Default"),
                ("lint/config.go", "Initialize"),
            ],
            "listed_tests": [
                "TestGetConfig",
                "TestGetConfig_OptionCasing",
                "TestGetConfig_EnableAllRulesCasing",
                "TestGetConfig_RuleOptionCasing",
                "TestDefault",
                "TestNormalize",
            ],
            "test_files": ["config/config_test.go"],
        },
        {
            "name": "ifelse-branch",
            "minutes": 2,
            "entry": ("internal/ifelse/branch.go", "BlockBranch"),
            "funcs": [
                ("internal/ifelse/branch.go", "BlockBranch"),
                ("internal/ifelse/branch.go", "StmtBranch"),
                ("internal/ifelse/func.go", "ExprCall"),
                ("internal/ifelse/branch.go", "HasDecls"),
                ("internal/ifelse/branch.go", "IsShort"),
                ("internal/ifelse/branch.go", "isShortStmt"),
            ],
            "listed_tests": [
                "TestBlockBranch",
                "TestStmtBranch",
                "TestBranch_HasDecls",
                "TestBranch_IsShort",
            ],
            "test_files": ["internal/ifelse/branch_test.go"],
        },
        {
            "name": "file-exclude-filter",
            "minutes": 2,
            "entry": ("lint/filefilter.go", "ParseFileFilter"),
            "funcs": [
                ("lint/filefilter.go", "ParseFileFilter"),
                ("lint/filefilter.go", "prepareRegexp"),
                ("lint/filefilter.go", "MatchFileName"),
                ("lint/filefilter.go", "String"),
                ("lint/config.go", "Initialize"),
                ("lint/config.go", "MustExclude"),
            ],
            "listed_tests": [
                "TestFileFilter",
                "TestFileExcludeFilterAtRuleLevel",
                "TestGetConfig",
            ],
            "test_files": [
                "lint/filefilter_test.go",
                "test/file_filter_test.go",
                "config/config_test.go",
            ],
        },
    ],
    "NOGRAPH": [
        {
            "name": "ifelse-chain-walker",
            "minutes": 4,
            "entry": ("internal/ifelse/rule.go", "Apply"),
            "funcs": [
                ("internal/ifelse/rule.go", "Apply"),
                ("internal/ifelse/rule.go", "Visit"),
                ("internal/ifelse/rule.go", "visitBody"),
                ("internal/ifelse/rule.go", "visitBlock"),
                ("internal/ifelse/rule.go", "visitIf"),
                ("internal/ifelse/rule.go", "checkRule"),
                ("internal/ifelse/branch.go", "BlockBranch"),
            ],
            "listed_tests": [
                "TestEarlyReturn",
                "TestIndentErrorFlow",
                "TestSuperfluousElse",
                "TestBlockBranch",
                "TestStmtBranch",
                "TestBranch_HasDecls",
                "TestBranch_IsShort",
            ],
            "test_files": [
                "test/early_return_test.go",
                "test/indent_error_flow_test.go",
                "test/superfluous_else_test.go",
                "internal/ifelse/branch_test.go",
            ],
        },
        {
            "name": "revivelib-runner",
            "minutes": 2,
            "entry": ("revivelib/core.go", "New"),
            "funcs": [
                ("revivelib/core.go", "New"),
                ("revivelib/core.go", "Lint"),
                ("revivelib/core.go", "Format"),
                ("revivelib/core.go", "getPackages"),
                ("revivelib/core.go", "normalizeSplit"),
                ("revivelib/pattern.go", "Include"),
                ("revivelib/pattern.go", "Exclude"),
            ],
            "listed_tests": [
                "TestReviveLint",
                "TestReviveFormat",
                "TestReviveCreateInstance",
            ],
            "test_files": [
                "revivelib/core_test.go",
                "revivelib/core_internal_test.go",
            ],
        },
        {
            "name": "package-naming-syncset",
            "minutes": 3,
            "entry": ("rule/package_naming.go", "Apply"),
            "funcs": [
                ("rule/package_naming.go", "Apply"),
                ("rule/package_naming.go", "pkgNameFailure"),
                ("internal/syncset/syncset.go", "New"),
                ("internal/syncset/syncset.go", "AddIfAbsent"),
                ("internal/syncset/syncset.go", "Elements"),
            ],
            "listed_tests": [
                "TestPackageNaming_conventionName",
                "TestPackageNaming_topLevel",
                "TestPackageNaming_badNames",
                "TestPackageNaming_stdLibConflict",
                "TestNew_ConcurrentElementsAreInitiallyEmpty",
                "TestSet_ZeroValueConcurrentUniqueAdds",
                "TestSet_AddIfAbsent_ConcurrentSameValueAddedOnce",
            ],
            "test_files": [
                "test/package_naming_test.go",
                "internal/syncset/syncset_test.go",
            ],
        },
        {
            "name": "var-naming",
            "minutes": 2,
            "entry": ("rule/var_naming.go", "Apply"),
            "funcs": [
                ("internal/rule/name.go", "Name"),
                ("rule/var_naming.go", "Apply"),
                ("rule/var_naming.go", "Visit"),
                ("rule/var_naming.go", "checkList"),
                ("rule/var_naming.go", "check"),
                ("rule/var_naming.go", "isUpperCaseConst"),
            ],
            "listed_tests": ["TestVarNaming", "TestName"],
            "test_files": [
                "test/var_naming_test.go",
                "internal/rule/name_test.go",
            ],
        },
        {
            "name": "file-filter",
            "minutes": 2,
            "entry": ("lint/filefilter.go", "ParseFileFilter"),
            "funcs": [
                ("lint/filefilter.go", "ParseFileFilter"),
                ("lint/filefilter.go", "MatchFileName"),
                ("lint/filefilter.go", "prepareRegexp"),
                ("lint/config.go", "Initialize"),
                ("lint/config.go", "MustExclude"),
            ],
            "listed_tests": [
                "TestFileFilter",
                "TestFileExcludeFilterAtRuleLevel",
                "TestGetConfig",
            ],
            "test_files": [
                "lint/filefilter_test.go",
                "test/file_filter_test.go",
                "config/config_test.go",
            ],
        },
    ],
}


def _run(
    args: list[str],
    *,
    cwd: Path,
    timeout: int,
) -> subprocess.CompletedProcess[str]:
    e = os.environ.copy()
    e["PATH"] = "/usr/local/go/bin:" + e.get("PATH", "")
    return subprocess.run(
        args,
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
        env=e,
    )


def is_test_file(rel: str) -> bool:
    name = Path(rel.replace("\\", "/")).name
    return name.endswith("_test.go")


def md_table_rows(text: str, header_pred) -> list[list[str]]:
    rows = []
    for line in text.splitlines():
        if not line.startswith("|"):
            continue
        cells = [c.strip() for c in line.strip("|").split("|")]
        if not cells or set(cells[0]) <= {"-"}:
            continue
        if header_pred(cells[0].lower()):
            continue
        rows.append(cells)
    return rows


def contract_coverage(unit_dir: Path, listed_tests: list[str]) -> dict:
    contract = (unit_dir / "contract.md").read_text()
    tests_md = (unit_dir / "tests.md").read_text()
    # coverage table: first col mentions a test
    cov_rows = []
    in_cov = False
    for line in contract.splitlines():
        if line.strip().startswith("## Coverage") or line.strip().startswith("# Coverage"):
            in_cov = True
            continue
        if in_cov and line.startswith("#"):
            break
        if in_cov and line.startswith("|"):
            cells = [c.strip() for c in line.strip("|").split("|")]
            if cells[0].lower() in {"test", "---"} or set(cells[0]) <= {"-"}:
                continue
            cov_rows.append(cells[0])
    test_rows = []
    in_tests = False
    for line in tests_md.splitlines():
        if line.startswith("|") and "Test" in line.split("|")[1]:
            in_tests = True
        if not line.startswith("|"):
            continue
        cells = [c.strip() for c in line.strip("|").split("|")]
        if cells[0].lower() in {"test", "---"} or set(cells[0]) <= {"-"}:
            continue
        if in_tests or cells[0].startswith("Test") or cells[0].startswith("`Test"):
            test_rows.append(cells[0])
    # assertion-ish: t.Error/t.Fatal/t.Run/require/assert in listed test files,
    # counted later; here map listed tests to a coverage sentence.
    covered = []
    missing = []
    for t in listed_tests:
        hit = False
        for row in cov_rows:
            compact = re.sub(r"[`*_]", "", row)
            if t in compact or t.removeprefix("Test") in compact or compact in t:
                hit = True
                break
            # subtest labels in nograph file-filter / var-naming
            short = t.replace("Test", "")
            if short and short.lower() in compact.lower():
                hit = True
                break
        if hit:
            covered.append(t)
        else:
            missing.append(t)
    n = len(listed_tests) or 1
    return {
        "n_listed_tests": len(listed_tests),
        "n_coverage_sentences": len(cov_rows),
        "n_tests_md_rows": len(test_rows),
        "covered_tests": covered,
        "missing_tests": missing,
        "pct": round(100.0 * len(covered) / n, 1),
        "coverage_rows": cov_rows,
    }


def properties_count(unit_dir: Path) -> int:
    text = (unit_dir / "properties.md").read_text()
    return len(re.findall(r"^\d+\.\s+\*\*", text, re.M))


def callers_of(con: sqlite3.Connection, file_path: str, name: str) -> list[dict]:
    file_path = file_path.replace("\\", "/")
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
    out = []
    for name_, kind, fp, line, meta in rows:
        conf = 1.0
        if meta:
            try:
                conf = float(json.loads(meta).get("confidence", 1.0))
            except (json.JSONDecodeError, TypeError, ValueError):
                conf = 1.0
        if conf < 0.85:
            continue
        out.append(
            {
                "name": name_,
                "kind": kind,
                "file": (fp or "").replace("\\", "/"),
                "line": line,
            }
        )
    return out


def node_span(con: sqlite3.Connection, file_path: str, name: str) -> tuple[int, int] | None:
    file_path = file_path.replace("\\", "/")
    row = con.execute(
        """
        SELECT start_line, end_line FROM nodes
        WHERE name = ? AND replace(file_path, '\\', '/') = ?
          AND kind IN ('function', 'method')
        """,
        [name, file_path],
    ).fetchone()
    if not row:
        return None
    return int(row[0]), int(row[1])


def blackbox(gold: Path, unit: dict) -> dict:
    unexported = [n for _, n in unit["funcs"] if n and n[0].islower()]
    hits = []
    ident = re.compile(r"\b(" + "|".join(re.escape(n) for n in unexported) + r")\b") if unexported else None
    for rel in unit["test_files"]:
        p = gold / rel
        if not p.exists() or ident is None:
            continue
        text = p.read_text()
        # restrict to listed test function bodies roughly
        for tname in unit["listed_tests"]:
            m = re.search(rf"func {re.escape(tname)}\b", text)
            if not m:
                continue
            start = m.start()
            nxt = re.search(r"\nfunc ", text[start + 5 :])
            body = text[start : start + 5 + (nxt.start() if nxt else len(text))]
            for hit in ident.findall(body):
                hits.append(f"{rel}:{tname}:{hit}")
    return {"unexported_hits": hits, "ok": not hits}


def copy_gold(dst: Path) -> None:
    if dst.exists():
        shutil.rmtree(dst)
    def ignore(dirpath, names):
        skip = {".codegraph", "units", "bugs", "START.txt", "START2.txt"}
        return [n for n in names if n in skip]
    shutil.copytree(GRAPH, dst, ignore=ignore)


def apply_patch(work: Path, patch: Path) -> tuple[bool, str]:
    text = patch.read_text()
    pflag = "1" if "\n--- a/" in text or text.startswith("--- a/") else "0"
    proc = subprocess.run(
        ["patch", f"-p{pflag}", "--forward", "--batch"],
        cwd=work,
        input=text,
        capture_output=True,
        text=True,
        timeout=30,
        check=False,
        env={**os.environ, "PATH": "/usr/local/go/bin:" + os.environ.get("PATH", "")},
    )
    ok = proc.returncode == 0
    return ok, (proc.stdout or "") + (proc.stderr or "")


def parse_failing(text: str) -> list[str]:
    tests = []
    for line in text.splitlines():
        if line.startswith("--- FAIL:"):
            parts = line.split()
            if len(parts) >= 3:
                tests.append(parts[2])
    return tests


def listed_match(failing: list[str], listed: list[str]) -> tuple[set[str], set[str]]:
    hit: set[str] = set()
    extra: set[str] = set()
    for f in failing:
        top = f.split("/")[0]
        matched = False
        for t in listed:
            if top == t or f == t or f.startswith(t + "/"):
                hit.add(t)
                matched = True
                break
        if not matched:
            extra.add(f)
    return hit, extra


def judge_excision(unit: dict, patch: Path, work: Path) -> dict:
    t0 = time.time()
    patch_ok, patch_out = apply_patch(work, patch)
    if not patch_ok:
        return {
            "patch_ok": False,
            "patch_out": patch_out[-2000:],
            "build_ok": False,
            "listed_fail": False,
            "others_ok": False,
            "failing": [],
            "extra_failing": [],
            "elapsed_s": time.time() - t0,
        }
    build = _run(["go", "build", "./..."], cwd=work, timeout=120)
    build_ok = build.returncode == 0
    listed = unit["listed_tests"]
    skip_re = "|".join(rf"^{re.escape(t)}(?:/|$)" for t in listed)
    # listed tests
    listed_fail = False
    listed_out = ""
    hang = False
    try:
        run_re = "|".join(rf"^{re.escape(t)}(?:/|$)" for t in listed)
        listed_proc = _run(
            ["go", "test", "./...", "-count=1", f"-timeout={LISTED_TIMEOUT}s", f"-run={run_re}"],
            cwd=work,
            timeout=LISTED_TIMEOUT + 30,
        )
        listed_out = (listed_proc.stdout or "") + (listed_proc.stderr or "")
        listed_failing = parse_failing(listed_out)
        hit, _ = listed_match(listed_failing, listed)
        listed_fail = listed_proc.returncode != 0
        if "test timed out" in listed_out.lower() or "panic: test timed out" in listed_out:
            hang = True
            listed_fail = True
    except subprocess.TimeoutExpired:
        hang = True
        listed_fail = True
        listed_out = "TIMEOUT"
    # everything else
    others_ok = False
    extra: list[str] = []
    others_out = ""
    try:
        others = _run(
            ["go", "test", "./...", "-count=1", f"-timeout={TEST_TIMEOUT}s", f"-skip={skip_re}"],
            cwd=work,
            timeout=TEST_TIMEOUT + 30,
        )
        others_out = (others.stdout or "") + (others.stderr or "")
        extra = parse_failing(others_out)
        others_ok = others.returncode == 0
    except subprocess.TimeoutExpired:
        others_ok = False
        others_out = "TIMEOUT"
        extra = ["TIMEOUT"]
    return {
        "patch_ok": True,
        "patch_out": patch_out[-500:],
        "build_ok": build_ok,
        "build_err": "" if build_ok else ((build.stdout or "") + (build.stderr or ""))[-1500:],
        "listed_fail": listed_fail,
        "hang": hang,
        "others_ok": others_ok,
        "extra_failing": extra,
        "listed_out_tail": listed_out[-1500:],
        "others_out_tail": others_out[-1500:],
        "elapsed_s": time.time() - t0,
    }


def closure_check(con: sqlite3.Connection, unit: dict) -> dict:
    closure_keys = {(f, n) for f, n in unit["funcs"]}
    entry = unit["entry"]
    entry_callers = callers_of(con, entry[0], entry[1])
    entry_caller_keys = {(c["file"], c["name"]) for c in entry_callers}
    dangling = []
    per_func = []
    for fpath, name in unit["funcs"]:
        calls = callers_of(con, fpath, name)
        span = node_span(con, fpath, name)
        lines = (span[1] - span[0] + 1) if span else None
        outside = []
        for c in calls:
            key = (c["file"], c["name"])
            if key in closure_keys:
                continue
            if (fpath, name) == entry:
                continue
            if key in entry_caller_keys:
                continue
            outside.append(c)
        prod = [c for c in outside if not is_test_file(c["file"])]
        tests = [c for c in outside if is_test_file(c["file"])]
        per_func.append(
            {
                "file": fpath,
                "name": name,
                "n_callers": len(calls),
                "lines": lines,
                "span": span,
                "dangling_prod": prod,
                "dangling_test": tests,
            }
        )
        for c in prod:
            dangling.append(f"{c['file']}:{c['name']} -> {fpath}:{name}")
    files = sorted({f for f, _ in unit["funcs"]})
    line_sum = sum(x["lines"] or 0 for x in per_func)
    return {
        "n_funcs": len(unit["funcs"]),
        "n_files": len(files),
        "files": files,
        "lines": line_sum,
        "cross_file": len(files) >= 2,
        "dangling": dangling,
        "n_dangling": len(dangling),
        "per_func": per_func,
        "ok": len(dangling) == 0,
    }


def worker(payload: dict) -> dict:
    cond = payload["cond"]
    unit = payload["unit"]
    gold = GRAPH
    units_root = GRAPH / "units" if cond == "GRAPH" else NOGRAPH / "units"
    unit_dir = units_root / unit["name"]
    db = GRAPH / ".codegraph" / "codegraph.db"
    con = sqlite3.connect(f"file:{db}?mode=ro", uri=True)
    try:
        clos = closure_check(con, unit)
    finally:
        con.close()
    bb = blackbox(gold, unit)
    cov = contract_coverage(unit_dir, unit["listed_tests"])
    n_props = properties_count(unit_dir)
    work = OUT / "work2" / f"{cond}-{unit['name']}"
    copy_gold(work)
    excision = judge_excision(unit, unit_dir / "excision.patch", work)
    a_ok = clos["ok"]
    b_ok = (
        excision["patch_ok"]
        and excision["build_ok"]
        and excision["listed_fail"]
        and excision["others_ok"]
    )
    c_ok = bb["ok"]
    valid = a_ok and b_ok and c_ok
    failed = []
    if not a_ok:
        failed.append("a-closure")
    if not b_ok:
        failed.append("b-self-containment")
    if not c_ok:
        failed.append("c-blackbox")
    return {
        "cond": cond,
        "unit": unit["name"],
        "minutes": unit["minutes"],
        "valid": valid,
        "failed": failed,
        "a_ok": a_ok,
        "b_ok": b_ok,
        "c_ok": c_ok,
        "closure": clos,
        "blackbox": bb,
        "contract": cov,
        "n_properties": n_props,
        "n_tests": len(unit["listed_tests"]),
        "excision": excision,
    }


def main() -> None:
    OUT.mkdir(exist_ok=True)
    (OUT / "work2").mkdir(exist_ok=True)
    jobs = []
    for cond, units in UNITS.items():
        for u in units:
            jobs.append({"cond": cond, "unit": u})
    results = []
    with ProcessPoolExecutor(max_workers=N_WORKERS) as ex:
        futs = [ex.submit(worker, j) for j in jobs]
        for fut in as_completed(futs):
            results.append(fut.result())
            print(
                f"{results[-1]['cond']} {results[-1]['unit']} valid={results[-1]['valid']} failed={results[-1]['failed']}",
                flush=True,
            )
    order = {("GRAPH", u["name"]): i for i, u in enumerate(UNITS["GRAPH"])}
    order.update({("NOGRAPH", u["name"]): 100 + i for i, u in enumerate(UNITS["NOGRAPH"])})
    results.sort(key=lambda r: order[(r["cond"], r["unit"])])
    (OUT / "judge2.json").write_text(json.dumps(results, indent=2) + "\n")
    print("wrote", OUT / "judge2.json")


if __name__ == "__main__":
    main()

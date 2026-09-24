"""Call-graph coverage gaps: does the hidden suite exercise machinery the
contract never mentions?

For each L2 unit dir we compute three symbol sets with the cgscan Go tool
(src/openswe_traces/cgscan, stdlib go/parser only):

    reached   union of symbols every hidden Test* reaches transitively through
              the module, with excised bodies restored from tests/gold.patch
    gold      symbols tests/gold.patch defines or references (what the solver
              must actually write)
    implied   symbols the contract (instruction.md) implicates: (a) the reach
              of tests named as coverage-table row keys, (b) symbol names
              literally present in the prose, (c) symbols an LLM judges
              paraphrased by the commitments

    gap = reached ∩ gold − implied

A non-empty gap is a structural coverage omission: the suite exercises it, gold
implements it, and no commitment mentions it. Predictor under test: "gap set is
non-empty" vs "unit did not flip at L2", measured on the same cohort as
scripts/ops/task_lint.py (the dose_response L2 dirs that ran trials) against the
~39% non-flip base rate.

CLI lives in scripts/cg_coverage.py. Resume-safe: every stage skips outputs
that already exist under outputs/cg_coverage/.
"""

from __future__ import annotations

import hashlib
import json
import os
import re
import subprocess
import threading
import time
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
MAIN_ROOT = Path(os.environ.get("OSWT_MAIN", "/home/evan/Documents/open_swe_traces_research"))
DR = MAIN_ROOT / "experiments" / "dose_response"
OUT = ROOT / "outputs" / "cg_coverage"
SCAN_BIN = OUT / "cgscan"
LOG = ROOT / "outputs" / "CGCOV.log"
LLM_CACHE = OUT / "llm"

PROMPT_VERSION = "cgmap-v1"
MAX_REQUESTS = 300
OPENROUTER_URL = "https://openrouter.ai/api/v1/chat/completions"
MODELS = (
    "nvidia/nemotron-3-super-120b-a12b:free",
    "nvidia/nemotron-3-ultra-550b-a55b:free",
    "nex-agi/nex-n2.5-pro:free",
    "google/gemma-4-31b-it:free",
)
MAX_ATTEMPTS = 4
REQUEST_TIMEOUT = 240
MAX_TOTAL_CHARS = 90_000

BUILTINS = {
    "append", "cap", "close", "complex", "copy", "delete", "imag", "len",
    "make", "new", "panic", "print", "println", "real", "recover", "clear",
    "min", "max", "string", "int", "int8", "int16", "int32", "int64",
    "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte", "rune",
    "bool", "error", "float32", "float64", "complex64", "complex128", "any",
    "comparable",
}

TESTFUNC_RE = re.compile(r"^func\s+(Test[A-Za-z0-9_]+)\s*\(", re.MULTILINE)
NAMETOK_RE = re.compile(r"Test[A-Za-z0-9_]+\*?")
RUNG_RE = re.compile(r"^(.*?)-L(\d)")
_RESPONSE_RE = re.compile(r"\{.*\}", re.DOTALL)

DOUBLE_FAILURE = [
    "gin-enginecfg", "gin-negotiate", "helm-chartdl", "helm-chartrepo",
    "helm-depresolver", "helm-httpgetter", "helm-repindex",
    "kops-clustervalid", "kops-difftext", "kops-taintparse",
]

# helm-repindex contract revisions. The canonical five-revision series is
# rcfix..rcfix5 with outcomes 0/3 -> 0/3 -> 1/3 -> 0/3 -> 2/3 (verifier_rules.md);
# the pre-series baselines are the original L2/L3 contracts. rc_auto* are
# auto-reconciler outputs, kept for completeness.
REPINDEX_SERIES = [
    ("sweep_L2", "helm-repindex-L2", "L2 orig (0/3)"),
    ("sweep_climb_L3", "helm-repindex-L3", "L3 orig (0/3)"),
    ("sweep_rcfix", "helm-repindex-L2rc", "rev1 rcfix (0/3)"),
    ("sweep_rcfix2", "helm-repindex-L2rc2", "rev2 rcfix2 (0/3)"),
    ("sweep_rcfix3", "helm-repindex-L2rc3", "rev3 rcfix3 (1/3)"),
    ("sweep_rcfix4", "helm-repindex-L2rc4", "rev4 rcfix4 (0/3)"),
    ("sweep_rcfix5", "helm-repindex-L2rc5", "rev5 rcfix5 (2/3)"),
    ("sweep_rc_auto", "helm-repindex-L2auto", "rc_auto (0/3)"),
    ("sweep_rc_auto2", "helm-repindex-L2auto2", "rc_auto2 (1/3)"),
]


def log(msg: str) -> None:
    line = f"{time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())} {msg}"
    print(line, flush=True)
    LOG.parent.mkdir(parents=True, exist_ok=True)
    with LOG.open("a") as f:
        f.write(line + "\n")


# ---------------------------------------------------------------- universe


def l2_trial_paths() -> dict[str, list[float]]:
    """dose_response task paths with >=1 L2 trial, -> rewards."""
    out: dict[str, list[float]] = {}
    for pat in ("experiments/dose_response/jobs/*/*/result.json",
                "experiments/dose_response/archive/*/*/result.json"):
        for j in sorted(MAIN_ROOT.glob(pat)):
            try:
                r = json.loads(j.read_text())
            except (OSError, json.JSONDecodeError):
                continue
            m = RUNG_RE.match(r.get("task_name", ""))
            if not m or int(m.group(2)) != 2:
                continue
            rw = (r.get("verifier_result") or {}).get("rewards", {}).get("reward")
            tp = r.get("task_id", {}).get("path") or ""
            if rw is not None and tp:
                out.setdefault(tp, []).append(rw)
    return out


def build_universe(include_pending_clientgo: bool = True) -> list[dict]:
    """The linter's cohort: dose_response L2 dirs with known trial outcomes.

    Primary universe = the 119 dose_response task paths that ran L2 trials.
    include_pending_clientgo adds the 8 sweep_clientgo_L2 units whose trials ran
    from a /tmp staging dir (same unit, different task path) -> 127.
    """
    paths = l2_trial_paths()
    units = []
    n_tmp = 0
    for tp, rws in sorted(paths.items()):
        p = MAIN_ROOT / tp
        pending = False
        if not p.is_dir():
            # /tmp staging path -> canonical sweep_clientgo_L2 dir by unit name
            name = tp.rsplit("/", 1)[-1]
            alt = DR / "sweep_clientgo_L2" / name
            n_tmp += 1
            if not include_pending_clientgo or not alt.is_dir():
                continue
            p = alt
            pending = True
        units.append({
            "key": tp.rsplit("/", 1)[-1] + "@" + tp.split("/")[-2],
            "unit": p.name,
            "sweep": p.parent.name,
            "path": str(p),
            "rel": str(p.relative_to(MAIN_ROOT)) if p.is_relative_to(MAIN_ROOT) else str(p),
            "flipped": any(x == 1.0 for x in rws),
            "n_l2_trials": len(rws),
            "pending": pending,
        })
    seen = set()
    dedup = []
    for u in units:
        if u["path"] in seen:
            continue
        seen.add(u["path"])
        dedup.append(u)
    log(f"universe: {len(dedup)} units ({n_tmp} tmp-path runs) "
        f"flip={sum(u['flipped'] for u in dedup)} "
        f"nonflip={sum(not u['flipped'] for u in dedup)}")
    return dedup


# ---------------------------------------------------------------- scanning


def pkg_dirs(src: Path) -> str:
    out = []
    for d, dirs, files in os.walk(src):
        dirs[:] = [x for x in dirs if not x.startswith(".") and x != "vendor"]
        if "testdata" in Path(d).relative_to(src).parts:
            dirs[:] = []
            continue
        if any(f.endswith(".go") for f in files):
            out.append(str(Path(d).relative_to(src)))
    return ",".join(sorted(x for x in out if x != "."))


def src_test_funcs(src: Path) -> dict[str, list[str]]:
    """Test func name -> [paths relative to src], over in-tree *_test.go files."""
    out: dict[str, list[str]] = {}
    for f in src.rglob("*_test.go"):
        if "testdata" in f.relative_to(src).parts or "vendor" in f.relative_to(src).parts:
            continue
        try:
            text = f.read_text(errors="replace")
        except OSError:
            continue
        for m in TESTFUNC_RE.finditer(text):
            out.setdefault(m.group(1), []).append(str(f.relative_to(src)))
    return out


def contract_text(unit: Path) -> str:
    for name in ("contract.md", "instruction.md"):
        p = unit / name
        if p.is_file():
            return p.read_text(errors="replace")
    return ""


def row_key_tests(text: str) -> list[str]:
    """Test names claimed in coverage-table first cells.

    Rows look like `| `TestX`, `TestY*` | sentence |`. Wildcard names expand
    against the resolved-name index at call time via '*'-suffix matching.
    """
    names = []
    for ln in text.splitlines():
        ln = ln.strip()
        if not ln.startswith("|"):
            continue
        cells = [c.strip() for c in ln.strip("|").split("|")]
        if len(cells) < 2:
            continue
        first = cells[0]
        if set(first) <= set(":- "):
            continue
        names.extend(NAMETOK_RE.findall(first))
    return sorted(set(names))


def expand_named(tokens: list[str], avail: list[str]) -> list[str]:
    """Expand wildcard tokens (TestFoo*) against available test names."""
    out = []
    for t in tokens:
        if t.endswith("*"):
            out.extend(a for a in avail if a.startswith(t[:-1]))
        else:
            out.append(t)
    return sorted(set(out))


def gold_dirs(unit: Path) -> list[str]:
    gold = unit / "tests" / "gold.patch"
    if not gold.is_file():
        return []
    return sorted({
        str(Path(m).parent) for m in
        re.findall(r"^\+\+\+ [ab]/(.+)$", gold.read_text(errors="replace"), re.MULTILINE)
    })


def nametest_args(unit: Path) -> str:
    """relpath.go:TestName entries for contract-named tests found in env/src.

    The coverage table refers to the unit's own original tests, which live in
    the package(s) gold.patch excises — resolve names there first so same-named
    tests elsewhere in the module (engine.TestMerge vs repo.TestMerge) do not
    hijack the reach.
    """
    src = unit / "environment" / "src"
    idx = src_test_funcs(src)
    gdirs = set(gold_dirs(unit))
    wanted = expand_named(row_key_tests(contract_text(unit)), list(idx))
    out = []
    for t in wanted:
        paths = idx.get(t, [])
        pick = next((p for p in paths if str(Path(p).parent) in gdirs), None)
        if pick is None and paths:
            pick = paths[0]
        if pick:
            out.append(f"{pick}:{t}")
    return ",".join(out)


def run_scan(unit: Path, out_json: Path, force: bool = False) -> Path:
    if out_json.is_file() and not force:
        return out_json
    src = unit / "environment" / "src"
    gold = unit / "tests" / "gold.patch"
    hidden_root = unit / "tests" / "hidden"
    hidden = ",".join(str(f) for f in sorted(hidden_root.rglob("*.go")))
    out_json.parent.mkdir(parents=True, exist_ok=True)
    cmd = [
        str(SCAN_BIN), "-src", str(src), "-gold", str(gold),
        "-pkgs", pkg_dirs(src), "-hidden", hidden,
        "-hiddenroot", str(hidden_root),
        "-nametest", nametest_args(unit), "-out", str(out_json),
    ]
    r = subprocess.run(cmd, capture_output=True, text=True, check=False)
    if r.returncode != 0:
        raise RuntimeError(f"cgscan {unit.name}: {r.stderr[:400]}")
    return out_json


# ---------------------------------------------------------------- set math


def _inmod(mp: str):
    def f(s: str) -> bool:
        if not s.startswith((mp + "/", mp + ".")):
            return False
        return s.rsplit(".", 1)[-1] not in BUILTINS
    return f


def compute_sets(scan: dict, text: str) -> dict:
    mp = scan["module"]
    inmod = _inmod(mp)
    reached, impl_named = set(), set()
    tests = scan.get("tests", {})
    for tname, res in tests.items():
        if tname.startswith("orig:"):
            continue
        for s in res["defs"] + res["refs"]:
            if inmod(s):
                reached.add(s)
    gold = {s for s in scan.get("gold_defs", []) + scan.get("gold_refs", []) if inmod(s)}
    cand = reached & gold

    # (a) reach of tests the contract names: coverage-table row keys (original
    # in-tree tests), plus hidden-suite test names mentioned anywhere in the
    # prose (a "hidden unit tests (names only)" section is still a disclosure:
    # naming the test names the behaviour it checks)
    named = expand_named(row_key_tests(text), list(tests) + [t[5:] for t in tests if t.startswith("orig:")])
    hidden_named = sorted({
        tok for tok in NAMETOK_RE.findall(text)
        if tok in tests and not tok.startswith("orig:")
    })
    impl_rows: set[str] = set()
    for t in named:
        for key in (t, "orig:" + t):
            res = tests.get(key)
            if not res:
                continue
            for s in res["defs"] + res["refs"]:
                if inmod(s):
                    impl_rows.add(s)
    for t in hidden_named:
        res = tests.get(t)
        if not res:
            continue
        for s in res["defs"] + res["refs"]:
            if inmod(s):
                impl_named.add(s)
    impl_named |= impl_rows

    # (b) symbol names literally present in the prose (B7 leakage still counts
    # as mentioning the machinery)
    impl_lit = set()
    for s in cand:
        short = s.split(").")[-1] if ")." in s else s.rsplit(".", 1)[-1]
        if len(short) >= 4 and re.search(r"\b" + re.escape(short) + r"\b", text):
            impl_lit.add(s)

    cand_sorted = sorted(cand)
    return {
        "module": mp,
        "reached": sorted(reached),
        "gold": sorted(gold),
        "candidates": cand_sorted,
        "implied_named": sorted(impl_named & cand),
        "implied_literal": sorted(impl_lit),
        "gap_deterministic": sorted(cand - impl_named - impl_lit),
        "gap_rows_only": sorted(cand - impl_rows - impl_lit),
        "warnings": scan.get("warnings", []),
        "n_hidden_tests": len([t for t in tests if not t.startswith("orig:")]),
        "named_tests": named,
        "hidden_named": hidden_named,
    }


# ---------------------------------------------------------------- LLM map


def load_api_key() -> str:
    env = ROOT / ".env"
    if env.is_file():
        for line in env.read_text().splitlines():
            if line.strip().startswith("OPENROUTER_API_KEY="):
                return line.split("=", 1)[1].strip().strip('"').strip("'")
    k = os.environ.get("OPENROUTER_API_KEY", "")
    if k:
        return k
    raise SystemExit("OPENROUTER_API_KEY not found in .env or environment")


def _post(api_key: str, model: str, prompt: str) -> tuple[dict | None, str | None]:
    payload = json.dumps({
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
        "temperature": 0,
        "max_tokens": 16384,
    }).encode()
    req = urllib.request.Request(OPENROUTER_URL, data=payload, headers={
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
        "HTTP-Referer": "https://devin.ai",
        "X-Title": "oswt cg_coverage symbol map",
    })
    try:
        with urllib.request.urlopen(req, timeout=REQUEST_TIMEOUT) as resp:
            data = json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        return None, f"HTTP {e.code}: {e.read().decode(errors='replace')[:300]}"
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as e:
        return None, f"{type(e).__name__}: {e}"
    try:
        content = data["choices"][0]["message"]["content"]
    except (KeyError, IndexError, TypeError):
        return None, f"malformed response: {str(data)[:300]}"
    m = _RESPONSE_RE.search(content or "")
    if not m:
        return None, f"unparseable content: {str(content)[:200]}"
    try:
        return json.loads(m.group(0)), None
    except json.JSONDecodeError:
        return None, f"bad json: {m.group(0)[:200]}"


def map_prompt(text: str, candidates: list[str], mp: str) -> str:
    short = [s[len(mp) + 1:] if s.startswith(mp + "/") else s for s in candidates]
    rows = "\n".join(f"{i}. {s}" for i, s in enumerate(short, 1))
    return f"""You are auditing a synthetic-task contract for coverage gaps.

The hidden test suite exercises the package symbols numbered below, and the
reference implementation touches them. The contract is supposed to commit to
every behaviour the suite checks.

For EACH numbered symbol decide whether the contract names or paraphrases the
behaviour it implements. Count it implicated when any sentence, bullet, or
coverage-table row describes what the symbol does — the contract may not use
symbol names, so judge by behaviour (e.g. "entries are kept sorted newest
first" implicates the sort-comparison method; "a digest is stored" implicates
the digest field). A symbol that is only reachable as invisible plumbing and
whose behaviour no commitment describes is NOT implicated.

Answer with JSON only: {{"implicated": [<numbers>]}}.

CONTRACT:
```
{text[:60000]}
```

SYMBOLS:
{rows}
"""


def llm_implicated(text: str, candidates: list[str], mp: str,
                   api_key: str, budget: list[int],
                   lock: threading.Lock | None = None) -> set[str]:
    """Symbols among candidates the contract paraphrases, per the LLM.
    Cached by content hash; budget[0] tracks the <=300 request cap.
    lock serializes budget increments when workers run in parallel."""
    lock = lock or threading.Lock()
    key = hashlib.sha256(
        (PROMPT_VERSION + "\n" + mp + "\n" + text + "\n" + "\n".join(candidates)).encode()
    ).hexdigest()
    cache = LLM_CACHE / f"{key}.json"
    if cache.is_file():
        try:
            data = json.loads(cache.read_text())
            return {candidates[i] for i in data.get("implicated", []) if i < len(candidates)}
        except (OSError, json.JSONDecodeError):
            pass
    prompt = map_prompt(text, candidates, mp)
    if len(prompt) > MAX_TOTAL_CHARS:
        prompt = prompt[: MAX_TOTAL_CHARS - 200] + "\n...[truncated]\n"
    verdict, model, err = None, MODELS[-1], "no models"
    for attempt in range(MAX_ATTEMPTS):
        with lock:
            if budget[0] >= MAX_REQUESTS:
                log("llm: request budget exhausted")
                return set()
            budget[0] += 1
        model = MODELS[min(attempt, len(MODELS) - 1)]
        verdict, err = _post(api_key, model, prompt)
        if verdict is not None:
            break
        if err and ("429" in err):
            time.sleep(20 * (attempt + 1))
        elif err and ("404" in err or "No endpoints" in err):
            continue
        else:
            time.sleep(5)
    if verdict is None:
        log(f"llm: failed after retries: {err}")
        return set()
    idxs = [i - 1 for i in verdict.get("implicated", [])
            if isinstance(i, int) and 1 <= i <= len(candidates)]
    cache.parent.mkdir(parents=True, exist_ok=True)
    cache.write_text(json.dumps({
        "model": model, "prompt_version": PROMPT_VERSION,
        "implicated": idxs, "n_candidates": len(candidates),
    }, indent=2))
    log(f"llm: {model} -> {len(idxs)}/{len(candidates)} implicated (cache {key[:8]})")
    return {candidates[i] for i in idxs}


# ---------------------------------------------------------------- pipeline


def sets_path(unit: Path) -> Path:
    return OUT / "sets" / (unit.parent.name + "__" + unit.name + ".json")


def scan_path(unit: Path) -> Path:
    return OUT / "scan" / (unit.parent.name + "__" + unit.name + ".json")


def analyze_unit(unit: Path, api_key: str | None, budget: list[int],
                 force_scan: bool = False, llm: bool = True,
                 budget_lock=None) -> dict:
    sj = scan_path(unit)
    run_scan(unit, sj, force=force_scan)
    scan = json.loads(sj.read_text())
    text = contract_text(unit)
    sets = compute_sets(scan, text)
    impl_llm: set[str] = set()
    if llm and api_key and sets["candidates"]:
        impl_llm = llm_implicated(text, sets["candidates"], sets["module"],
                                  api_key, budget, lock=budget_lock)
    implied = set(sets["implied_named"]) | set(sets["implied_literal"]) | impl_llm
    gap = sorted(set(sets["candidates"]) - implied)
    sets["implied_llm"] = sorted(impl_llm)
    sets["gap"] = gap
    sp = sets_path(unit)
    sp.parent.mkdir(parents=True, exist_ok=True)
    sp.write_text(json.dumps(sets, indent=2))
    return sets


def confusion(units: list[dict], gap_field: str = "gap") -> dict:
    tp = fp = fn = tn = 0
    skipped = []
    for u in units:
        sp = sets_path(Path(u["path"]))
        if not sp.is_file():
            skipped.append(u["unit"])
            continue
        sets = json.loads(sp.read_text())
        flagged = len(sets.get(gap_field, [])) > 0
        nonflip = not u["flipped"]
        if flagged and nonflip:
            tp += 1
        elif flagged and not nonflip:
            fp += 1
        elif not flagged and nonflip:
            fn += 1
        else:
            tn += 1
    n = tp + fp + fn + tn
    prec = tp / (tp + fp) if tp + fp else 0.0
    rec = tp / (tp + fn) if tp + fn else 0.0
    base = (tp + fn) / n if n else 0.0
    return {"n": n, "tp": tp, "fp": fp, "fn": fn, "tn": tn,
            "precision": prec, "recall": rec, "base_rate": base,
            "flagged": tp + fp, "skipped": skipped}

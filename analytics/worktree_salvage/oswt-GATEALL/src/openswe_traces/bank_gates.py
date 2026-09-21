"""Run the whole gate stack over the packaged-unit bank and report per unit.

Gates, cheapest first (see analytics/research/verifier_rules.md, "Verifier
design: the two defect families"):

1. ``lint``    — the deterministic linter ``scripts/ops/task_lint.py``
                 (B10 digest oracles, A12 gold-touches-tests, missing hidden
                 suite, unresolved "the call" scrub, ordering-as-direction).
2. ``orphans`` — gold-touched exported functions the hidden suite never
                 reaches, via the Go call-graph scanner
                 ``src/openswe_traces/cgscan`` (receivers resolved, callees
                 chased transitively through gold.patch-restored bodies).
                 Measured at 53% precision at >=2 orphans vs a 39% base rate.
3. ``a13``     — the contract-vs-gold judge (OpenRouter free tier, cached to
                 ``outputs/contract_judge``). Advisory on warm repos; a gate
                 on repos with fewer than 3 banked units in
                 ``experiments/pipeline/tasks_composerver``.

One row per unit is emitted to ``analytics/research/bank_gate_report.tsv``;
the analysis (confusion matrices vs trial outcomes, ranked escalation list,
drop list, token saving) goes to ``analytics/research/bank_gate_report.md``.

The bank lives in the primary checkout (``--repo-root``): unit packages under
``experiments/pipeline/tasks*``, ``experiments/pipeline/work``,
``experiments/dose_response`` and ``experiments/harbor_nex``, and the trial
ledger under ``experiments/dose_response/jobs/*/result.json``.
"""

from __future__ import annotations

import argparse
import importlib.util
import json
import re
import subprocess
import time
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass, field
from pathlib import Path

from openswe_traces.data import ROOT

DEFAULT_REPO = Path("/home/evan/Documents/open_swe_traces_research")
TOKEN_COST = 2_180_000  # measured mean per trial, verifier_rules.md
ORPHAN_BLOCK = 2        # 53%-precise threshold from the retrospective
MAX_ROUNDS = 3          # sweep_seq.sh default

UNIT_ROOTS = (
    "experiments/pipeline/tasks",
    "experiments/pipeline/tasks_composerver",
    "experiments/pipeline/work",
    "experiments/dose_response",
    "experiments/harbor_nex",
)
SKIP_SUBTREE = re.compile(r"/(jobs|archive|_author|orig|tree|verify_tmp|author)(/|$)")

# repo prefix in a unit/family name, longest match first
REPO_PREFIXES = ("client-go", "go-github", "nats-server", "itsdangerous",
                 "bbolt", "go-git", "gogit", "xrepo", "store", "helm",
                 "kops", "goa", "gin")

# staging-dir cohort -> repo, for unit names that carry no repo prefix
SWEEP_REPO = {
    "sweep_gogit": "go-git", "sweep_gogit_L2": "go-git",
    "sweep_nats": "nats-server", "sweep_nats_L2": "nats-server",
    "sweep_goa": "goa", "sweep_goa_L2": "goa", "sweep_goa_rerun": "goa",
    "sweep_clientgo": "client-go", "sweep_clientgo_L2": "client-go",
    "sweep_gin2": "gin", "sweep_gin2_L2": "gin",
    "sweep_helm2": "helm", "sweep_helm2_L2": "helm",
    "sweep_kops2": "kops", "sweep_kops2_L2": "kops",
    "sweep_gogithub2": "go-github", "sweep_gogithub2_L2": "go-github",
    "sweep_gogithub2_L2_rerun": "go-github", "sweep_gogithub_rc": "go-github",
    "sweep_xrepo": "xrepo", "sweep_xrepo20": "xrepo", "sweep_xrepo20_L0": "xrepo",
}
WORK_REPO = {
    "vf_skel_gogit": "go-git", "vf_excised_gogit": "go-git",
    "vf_hidden_gogit": "go-git", "vf_gold_gogit": "go-git",
    "vf_cheat_gogit": "go-git",
    "vf_bbolt_skel": "bbolt", "vf_bbolt_excised": "bbolt",
    "vf_bbolt_gold": "bbolt", "vf_bbolt_hidden": "bbolt",
    "vf_bbolt_cheat": "bbolt", "vf_bbolt": "bbolt",
    "vf_skel_goa": "goa", "vf_goa": "goa",
    "vf_skel": "gin", "vf_gold": "gin", "vf_hidden": "gin",
    "vf_excised": "gin", "vf_cheat": "gin", "vf_gocache": "gin",
    "go-github": "go-github", "clientgo_b2": "client-go",
    "client-go": "client-go", "gin": "gin", "goa": "goa",
    "helm": "helm", "kops": "kops", "itsdangerous": "itsdangerous",
}

_LEVEL_RE = re.compile(r"-L(\d)")


# --------------------------------------------------------------------------
# units
# --------------------------------------------------------------------------

def family_of(name: str) -> str:
    return re.sub(r"-(L\d.*|A-\d)$", "", name)


def stem_of(name: str, repo: str) -> str:
    """Family stem with any repo prefix removed: client-go-connarray -> connarray."""
    stem = family_of(re.sub(r"-skel$", "", name))
    if repo and stem.startswith(repo + "-"):
        return stem[len(repo) + 1:]
    return stem


def famkey(repo: str, stem: str) -> str:
    return f"{repo}/{stem}" if repo else stem


def level_of(name: str) -> int | None:
    m = _LEVEL_RE.search(name)
    return int(m.group(1)) if m else None


def repo_of(name: str, locus_dir: str = "") -> str:
    stem = family_of(name)
    for p in REPO_PREFIXES:
        if stem == p or stem.startswith(p + "-"):
            return {"gogit": "go-git"}.get(p, p)
    if locus_dir:
        if locus_dir in SWEEP_REPO:
            return SWEEP_REPO[locus_dir]
        if locus_dir in WORK_REPO:
            return WORK_REPO[locus_dir]
        if locus_dir.startswith("tasks_"):
            return "nex"
    return ""


@dataclass
class UnitRef:
    name: str
    path: Path          # repo-root-relative
    family: str         # famkey: repo/stem
    stem: str           # family name without repo prefix
    level: int | None
    repo: str
    locus: str
    has_env: bool
    has_hidden: bool
    has_gold: bool
    mtime: float


def _is_unit_dir(d: Path) -> bool:
    return (d / "instruction.md").is_file() and (d / "task.toml").is_file()


def find_units(repo: Path) -> dict[str, list[UnitRef]]:
    """All packaged unit dirs, keyed by unit name (dirs may repeat across roots)."""
    out: dict[str, list[UnitRef]] = defaultdict(list)
    for rel in UNIT_ROOTS:
        root = repo / rel
        if not root.is_dir():
            continue
        for instr in root.rglob("instruction.md"):
            if not _is_unit_dir(instr.parent):
                continue
            ud = instr.parent
            s = str(ud)
            if SKIP_SUBTREE.search(s[len(str(repo)):]):
                continue
            name = ud.name
            if name.startswith("_"):
                base = name.lstrip("_")
                if base.endswith("_skel"):
                    base = base[:-5] + "-skel"
                name = f"{ud.parent.name}-{base}"
            locus_dir = ud.relative_to(root).parts[0]
            repo_ = repo_of(name, locus_dir)
            stem = stem_of(name, repo_)
            u = UnitRef(
                name=name,
                path=ud.relative_to(repo),
                family=famkey(repo_, stem),
                stem=stem,
                level=level_of(name),
                repo=repo_,
                locus=rel.split("/")[-1],
                has_env=(ud / "environment" / "src").is_dir(),
                has_hidden=(ud / "tests" / "hidden").is_dir(),
                has_gold=(ud / "tests" / "gold.patch").is_file()
                         or (ud / "patches" / "gold.patch").is_file(),
                mtime=instr.stat().st_mtime,
            )
            out[name].append(u)
    return out


# --------------------------------------------------------------------------
# trial ledger
# --------------------------------------------------------------------------

@dataclass
class Trial:
    task: str
    family: str
    level: int | None
    reward: float | None
    error: bool
    job: str
    task_path: str       # task_id.path — the exact unit dir the trial saw
    tokens: int
    started: str


def _trial_repo(task: str, task_path: str,
                stem_repos: dict[str, set[str]] | None = None) -> str:
    parts = Path(task_path).parts
    hint = ""
    if "dose_response" in parts:
        i = parts.index("dose_response")
        if i + 1 < len(parts):
            hint = parts[i + 1]
    elif "tasks_composerver" in parts or "tasks" in parts:
        i = max(parts.index("tasks_composerver") if "tasks_composerver" in parts else -1,
                parts.index("tasks") if "tasks" in parts else -1)
        if i + 1 < len(parts):
            hint = parts[i + 1]
    r = repo_of(task, hint)
    if not r and stem_repos:
        # /tmp staging or bare name: resolve via the bank's unique stem owner
        owners = stem_repos.get(stem_of(task, ""), set())
        if len(owners) == 1:
            r = next(iter(owners))
    return r


def load_ledger(repo: Path,
                stem_repos: dict[str, set[str]] | None = None) -> list[Trial]:
    """Merged trial ledger: dose_response jobs + archive result.json +
    archive/trials.jsonl. Deduped by (task, job, started_at); the jobs-dir
    copy wins (it carries task_path and token counts)."""
    dr = repo / "experiments/dose_response"
    seen: dict[tuple, Trial] = {}
    for pat in ("jobs/*/*/result.json", "archive/*/*/result.json"):
        for rj in dr.glob(pat):
            try:
                d = json.loads(rj.read_text())
            except (OSError, json.JSONDecodeError):
                continue
            task = d.get("task_name") or rj.parent.name.split("__")[0]
            rew = ((d.get("verifier_result") or {}).get("rewards") or {}).get("reward")
            exc = d.get("exception_info")
            ar = d.get("agent_result") or {}
            task_path = (d.get("task_id") or {}).get("path") or ""
            repo_ = _trial_repo(task, task_path, stem_repos)
            t = Trial(
                task=task,
                family=famkey(repo_, stem_of(task, repo_)),
                level=level_of(task),
                reward=float(rew) if rew is not None else None,
                error=bool(exc),
                job=rj.parent.parent.name,
                task_path=task_path,
                tokens=int(ar.get("n_input_tokens") or 0)
                       + int(ar.get("n_output_tokens") or 0),
                started=d.get("started_at") or "",
            )
            seen[(task, t.job, t.started)] = t
    jl = dr / "archive/trials.jsonl"
    if jl.is_file():
        for line in jl.read_text().splitlines():
            try:
                d = json.loads(line)
            except json.JSONDecodeError:
                continue
            task = d.get("task") or ""
            job = d.get("job") or ""
            started = d.get("started_at") or ""
            if (task, job, started) in seen:
                continue
            repo_ = repo_of(task, job)
            if not repo_ and stem_repos:
                owners = stem_repos.get(stem_of(task, ""), set())
                if len(owners) == 1:
                    repo_ = next(iter(owners))
            seen[(task, job, started)] = Trial(
                task=task, family=famkey(repo_, stem_of(task, repo_)),
                level=level_of(task),
                reward=d.get("reward"),
                error=bool(d.get("exception")),
                job=job, task_path=task_path, tokens=0, started=started,
            )
    return list(seen.values())


@dataclass
class FamilyOutcome:
    family: str
    levels: dict[int, list[float]] = field(default_factory=lambda: defaultdict(list))
    errors: int = 0
    trials: int = 0

    def record(self, t: Trial) -> None:
        self.trials += 1
        if t.reward is None:
            self.errors += 1
            return
        self.levels[t.level if t.level is not None else -1].append(t.reward)

    @property
    def l0(self) -> str:
        """fail | pass | mixed | err | none"""
        r = self.levels.get(0, [])
        if not r:
            return "err" if self.errors else "none"
        if all(x > 0 for x in r):
            return "pass"
        if all(x == 0 for x in r):
            return "fail"
        return "mixed"

    @property
    def escalated(self) -> bool:
        return any(l >= 2 for l in self.levels)

    @property
    def flipped(self) -> bool:
        """certified-hard: at least one pass at L2 (the contract-mediated rung)."""
        return any(x > 0 for x in self.levels.get(2, []))


def family_outcomes(trials: list[Trial]) -> dict[str, FamilyOutcome]:
    out: dict[str, FamilyOutcome] = {}
    for t in trials:
        out.setdefault(t.family, FamilyOutcome(t.family)).record(t)
    return out


def ukey(u: UnitRef) -> str:
    """Logical-unit identity: same family+rung under any dir name."""
    return f"{u.family}@L{u.level if u.level is not None else -1}"


def pick_canonical(refs: list[UnitRef]) -> UnitRef:
    """The dir a gate evaluates: newest revision, preferring copies with a
    source tree (sweep-staged copies carry environment/src)."""
    return max(refs, key=lambda u: (u.has_env, u.mtime))


# --------------------------------------------------------------------------
# gate 1: task_lint (vendored scripts/ops/task_lint.py, loaded by path)
# --------------------------------------------------------------------------

def _load_lint():
    spec = importlib.util.spec_from_file_location(
        "task_lint", ROOT / "scripts" / "ops" / "task_lint.py")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


# --------------------------------------------------------------------------
# gate 2: orphan count via cgscan
# --------------------------------------------------------------------------

def _gold_dirs(gold_text: str) -> list[str]:
    dirs = set()
    for m in re.finditer(r"^\+\+\+\s+[ab]/(\S+)", gold_text, re.MULTILINE):
        p = m.group(1)
        if "/" in p:
            dirs.add(p.rsplit("/", 1)[0])
        else:
            dirs.add(".")
    for m in re.finditer(r"^diff --git\s+[ab]/(\S+)\s+[ab]/(\S+)", gold_text, re.MULTILINE):
        p = m.group(2)
        dirs.add(p.rsplit("/", 1)[0] if "/" in p else ".")
    return sorted(dirs)


def _gold_text(ud: Path) -> str:
    for rel in ("tests/gold.patch", "patches/gold.patch"):
        f = ud / rel
        if f.is_file():
            return f.read_text(errors="replace")
    return ""


def _hidden_go(ud: Path) -> list[str]:
    d = ud / "tests" / "hidden"
    if not d.is_dir():
        return []
    return [str(f) for f in sorted(d.rglob("*.go"))]


def find_tree(repo: Path, u: UnitRef, fam_refs: list[UnitRef],
              dest_root: Path, log) -> Path | None:
    """A usable Go tree for `u`: its own env/src, a same-family copy's (any
    rung — the excised tree is shared across rungs), a staged excised tree,
    or the repo's pristine base tree.

    For cgscan's reach analysis the pristine base is adequate: excision only
    removes bodies, and gold-restored edges are the original bodies the base
    tree already carries — cgscan reads gold edges from gold.patch either way.
    """
    ud = repo / u.path
    own = ud / "environment" / "src"
    if own.is_dir() and (own / "go.mod").is_file():
        return own
    if own.is_dir() and list(own.rglob("go.mod")):
        return own
    for sib in fam_refs:
        if sib is u:
            continue
        alt = repo / sib.path / "environment" / "src"
        if alt.is_dir() and any(alt.iterdir()):
            return alt
    # go-github staged skel: work/go-github/units/<fam>/_skel/environment/src
    gh = repo / "experiments/pipeline/work/go-github/units" / u.stem / "_skel/environment/src"
    if gh.is_dir():
        return gh
    # vf_excised trees are bare excised trees (go.mod at root)
    for stage in ("vf_excised_gogit", "vf_bbolt_excised", "vf_excised"):
        t = repo / "experiments/pipeline/work" / stage / u.stem
        if (t / "go.mod").is_file():
            return t
    # pristine repo base tree — for reach the gold patch restores the original
    # bodies anyway, so unexcised sources give the same call graph
    for cand in (repo / "experiments/pipeline/work" / u.repo / "tree",
                 repo / "experiments/pipeline/repos2" / u.repo / "src",
                 repo / "experiments/pipeline/repos" / u.repo / "src"):
        if (cand / "go.mod").is_file():
            return cand
    return None


def _nametest_args(ud: Path, tree: Path) -> str:
    """relpath.go:TestName entries for contract-named tests, resolved inside
    `tree` (which may be materialized), preferring files under gold-touched
    dirs — mirrors cg_coverage.nametest_args but takes the tree explicitly."""
    from openswe_traces import cg_coverage as cc
    idx = cc.src_test_funcs(tree)
    gdirs = set(_gold_dirs(_gold_text(ud)))
    wanted = cc.expand_named(cc.row_key_tests(cc.contract_text(ud)), list(idx))
    out = []
    for t in wanted:
        paths = idx.get(t, [])
        pick = next((p for p in paths if str(Path(p).parent) in gdirs), None)
        if pick is None and paths:
            pick = paths[0]
        if pick:
            out.append(f"{pick}:{t}")
    return ",".join(out)


def orphan_scan(cgscan: Path, ud: Path, tree: Path, out_path: Path) -> dict | None:
    """Run cgscan over the (possibly materialized) tree and return the
    cg_coverage-style set analysis: gap = reached ∩ gold − implied."""
    from openswe_traces import cg_coverage as cc
    gold_file = ud / "tests" / "gold.patch"
    if not gold_file.is_file():
        gold_file = ud / "patches" / "gold.patch"
    hidden_root = ud / "tests" / "hidden"
    hidden = _hidden_go(ud)
    if not gold_file.is_file() or not hidden:
        return None
    cmd = [str(cgscan), "-src", str(tree), "-gold", str(gold_file),
           "-pkgs", cc.pkg_dirs(tree), "-hidden", ",".join(hidden),
           "-hiddenroot", str(hidden_root),
           "-nametest", _nametest_args(ud, tree),
           "-out", str(out_path)]
    proc = subprocess.run(cmd, capture_output=True, text=True, timeout=300,
                          check=False)
    if proc.returncode != 0 or not out_path.is_file():
        return {"error": (proc.stderr or proc.stdout)[-300:]}
    try:
        scan = json.loads(out_path.read_text())
    except json.JSONDecodeError:
        return {"error": "bad cgscan json"}
    sets = cc.compute_sets(scan, cc.contract_text(ud))
    sets["warnings"] = scan.get("warnings", [])
    return sets


# --------------------------------------------------------------------------
# gate 3: A13 contract-vs-gold judge
# --------------------------------------------------------------------------

def a13_evidence(ud: Path):
    from openswe_traces.gate.context import GateContext
    from openswe_traces.gate.contract_gold import gather_evidence
    return gather_evidence(GateContext(task_dir=ud))


def a13_verdict(ev, cache_dir: Path) -> dict | None:
    from openswe_traces.gate.contract_gold import load_judgement
    return load_judgement(ev, cache_dir)


def a13_summary(judgement: dict | None) -> tuple[int, int]:
    """(n false rows, n missing assertions) from a cached judgement."""
    if not judgement:
        return 0, 0
    v = judgement.get("verdict", judgement)
    n_false = sum(1 for r in (v.get("rows") or [])
                  if isinstance(r, dict) and r.get("verdict") == "false")
    n_missing = len(v.get("missing") or [])
    return n_false, n_missing


# --------------------------------------------------------------------------
# recommendation
# --------------------------------------------------------------------------

def recommend(lint_blocks: list[str], orphans: int | None,
              a13_state: str, cold: bool, rung: int | None = None) -> str:
    """trial | repair | drop. A13 is a gate only on cold repos (<3 banked
    units); the gap count only applies at rungs with a contract (>=L2 —
    L0/skel dirs have no coverage table, so reached∩gold lands wholesale
    in the gap by construction)."""
    if "B10" in lint_blocks:
        return "drop"
    contract_rung = rung is not None and rung >= 2
    gap_flag = contract_rung and orphans is not None and orphans >= ORPHAN_BLOCK
    if lint_blocks or gap_flag:
        return "repair"
    if cold and a13_state == "fail":
        return "repair"
    if cold and a13_state == "unjudged":
        return "repair"  # fail-closed: cold-repo contract unproven
    return "trial"


# --------------------------------------------------------------------------
# driver
# --------------------------------------------------------------------------

def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--repo-root", type=Path, default=DEFAULT_REPO)
    ap.add_argument("--max-requests", type=int, default=300)
    ap.add_argument("--workers", type=int, default=8)
    ap.add_argument("--no-judge", action="store_true",
                    help="skip the A13 judge; consume whatever is already cached")
    ap.add_argument("--only", nargs="*", default=None,
                    help="restrict to unit names (for smoke runs)")
    args = ap.parse_args()

    repo: Path = args.repo_root
    out_tsv = ROOT / "analytics/research/bank_gate_report.tsv"
    out_md = ROOT / "analytics/research/bank_gate_report.md"
    cg_dir = ROOT / "outputs/cg_coverage"
    scan_dir = cg_dir / "scans"
    tree_dir = cg_dir / "trees"
    judge_dir = ROOT / "outputs/contract_judge"
    log_path = ROOT / "outputs/GATEALL.log"
    for d in (scan_dir, tree_dir, judge_dir, out_tsv.parent, log_path.parent):
        d.mkdir(parents=True, exist_ok=True)
    logf = log_path.open("a")

    def log(msg: str) -> None:
        line = f"[{time.strftime('%H:%M:%S')}] {msg}"
        print(line, flush=True)
        logf.write(line + "\n")
        logf.flush()

    cgscan = cg_dir / "cgscan"
    if not cgscan.is_file():
        src = ROOT / "src/openswe_traces/cgscan/main.go"
        subprocess.run(["go", "build", "-o", str(cgscan), str(src)],
                       check=True, cwd=src.parent)

    log(f"scanning units under {repo}")
    all_refs = find_units(repo)
    stem_repos: dict[str, set[str]] = defaultdict(set)
    for refs in all_refs.values():
        for u in refs:
            if u.repo:
                stem_repos[u.stem].add(u.repo)
    trials = load_ledger(repo, stem_repos)
    fams = family_outcomes(trials)
    log(f"{len(all_refs)} unit names, {sum(len(v) for v in all_refs.values())} dirs, "
        f"{len(trials)} trials, {len(fams)} families with trial rows")

    # logical units: one row per (famkey, rung) across all copies
    by_key: dict[str, list[UnitRef]] = defaultdict(list)
    by_family: dict[str, list[UnitRef]] = defaultdict(list)
    for refs in all_refs.values():
        for u in refs:
            by_key[ukey(u)].append(u)
            by_family[u.family].append(u)
    canon: dict[str, UnitRef] = {k: pick_canonical(refs) for k, refs in by_key.items()}
    if args.only:
        keep = set(args.only)
        canon = {k: u for k, u in canon.items()
                 if u.name in keep or k in keep or u.family in keep}
    # task paths each logical unit was trialed from (rev-drift flag)
    trialed_paths: dict[str, set[str]] = defaultdict(set)
    for t in trials:
        trialed_paths[f"{t.family}@L{t.level if t.level is not None else -1}"].add(t.task_path)

    # banked units per repo (tasks_composerver) -> cold-repo set for A13
    banked: dict[str, int] = defaultdict(int)
    tc = repo / "experiments/pipeline/tasks_composerver"
    if tc.is_dir():
        for rd in tc.iterdir():
            if rd.is_dir():
                banked[rd.name] = sum(
                    1 for u in rd.iterdir()
                    if u.is_dir() and not u.name.startswith("_"))
    cold_repos = set()
    for u in canon.values():
        if u.repo and banked.get(u.repo, 0) < 3:
            cold_repos.add(u.repo)
    log(f"banked repos: {dict(banked)}; cold: {sorted(cold_repos)}")

    # ---- gate 1: lint ----------------------------------------------------
    lint_mod = _load_lint()
    rows: dict[str, dict] = {}
    for k, u in sorted(canon.items()):
        ud = repo / u.path
        findings = lint_mod.lint(ud)
        blocks = [c for sev, c, _ in findings if sev == "BLOCK"]
        if not u.has_gold and "GOLD" not in blocks:
            findings.append(("BLOCK", "GOLD", "no gold.patch — nothing to verify against"))
            blocks.append("GOLD")
        rows[k] = {
            "ukey": k, "unit": u.name, "repo": u.repo, "family": u.family,
            "rung": u.level,
            "locus": u.locus, "path": str(u.path), "has_env": u.has_env,
            "n_copies": len(by_key[k]),
            "rev_drift": bool(trialed_paths.get(k))
                         and str(u.path) not in trialed_paths[k],
            "lint_blocks": blocks,
            "lint_warn": [c for sev, c, _ in findings if sev == "WARN"],
            "lint_info": [c for sev, c, _ in findings if sev == "INFO"],
            "lint_codes": [f"{sev}:{c}" for sev, c, _ in findings],
            "lint_msgs": [f"{c}: {m}" for sev, c, m in findings if sev == "BLOCK"],
        }
    n_block = sum(1 for r in rows.values() if r["lint_blocks"])
    log(f"lint: {n_block}/{len(rows)} units carry BLOCK findings")

    # ---- gate 2: orphans --------------------------------------------------
    def scan_one(item):
        k, u = item
        ud = repo / u.path
        tree = find_tree(repo, u, by_family.get(u.family, []), tree_dir, log)
        if tree is None:
            return k, None, "no-tree"
        scan = orphan_scan(cgscan, ud, tree,
                           scan_dir / f"{re.sub(r'[^A-Za-z0-9_.-]', '_', k)}.json")
        return k, scan, str(tree)

    with ThreadPoolExecutor(max_workers=args.workers) as ex:
        for k, sets, tree in ex.map(scan_one, sorted(canon.items())):
            rows[k]["scan_tree"] = tree
            if sets is None:
                rows[k].update(orphans=None, orphan_names=[], orphan_cand=None,
                               orphan_note="no gold+hidden")
            elif "error" in sets:
                rows[k].update(orphans=None, orphan_names=[], orphan_cand=None,
                               orphan_note=sets["error"][:120])
            else:
                gap = sets.get("gap_deterministic", [])
                rows[k].update(orphans=len(gap), orphan_names=gap[:12],
                               orphan_cand=len(sets.get("candidates", [])),
                               orphan_note="" if sets.get("module") else "no-module")
    got = sum(1 for r in rows.values() if r.get("orphans") is not None)
    log(f"orphans: scanned {got}/{len(rows)} units")

    # ---- gate 3: A13 ------------------------------------------------------
    from openswe_traces.gate.contract_gold import load_judgement

    evs: dict[str, object] = {}
    for n, u in sorted(canon.items()):
        ud = repo / u.path
        try:
            ev = a13_evidence(ud)
        except Exception as exc:  # noqa: BLE001
            log(f"a13 evidence FAIL {n}: {exc}")
            ev = None
        evs[n] = ev
        if ev is None or not ev.rows:
            rows[n].update(a13="no-rows", a13_false=0, a13_missing=0)
            continue
        j = load_judgement(ev, judge_dir)
        if j:
            f_, m_ = a13_summary(j)
            rows[n].update(a13="fail" if (f_ or m_) else "pass",
                           a13_false=f_, a13_missing=m_)
        else:
            rows[n].update(a13="unjudged", a13_false=0, a13_missing=0)

    # units sharing a contract_key share a verdict; judge each key once
    by_ckey: dict[str, list[str]] = defaultdict(list)
    for k, ev in evs.items():
        if ev is not None and ev.rows:
            by_ckey[ev.contract_key].append(k)
    seen_ckey: set[str] = set()
    need = []
    for k in sorted(canon):
        ev = evs[k]
        if (rows[k]["a13"] == "unjudged" and ev is not None
                and ev.contract_key not in seen_ckey):
            seen_ckey.add(ev.contract_key)
            need.append(k)
    # judge order: cold-repo units, then unescalated families, then the rest
    unesc_fams = {f for f, o in fams.items() if not o.escalated}
    cold_set = cold_repos

    def jprio(n):
        u = canon[n]
        return (0 if u.repo in cold_set else 1,
                0 if u.family in unesc_fams else 1,
                n)

    need.sort(key=jprio)
    log(f"a13: {len(need)} distinct contracts need judging "
        f"(budget {args.max_requests} requests)")
    if not args.no_judge and need:
        from openswe_traces.contract_judge import (
            MODELS,
            attach_bodies,
            judge_unit,
            load_api_key,
            load_hidden_files,
        )
        key = load_api_key()
        spent = [0]
        import threading
        lock = threading.Lock()

        def judge_one(n):
            u = canon[n]
            ev = evs[n]
            with lock:
                if spent[0] >= args.max_requests:
                    return None
                reserve = min(4, args.max_requests - spent[0])
                spent[0] += reserve
            attach_bodies(ev, load_hidden_files(repo / u.path))
            out = judge_unit(ev, key, MODELS, reserve, judge_dir)
            with lock:
                spent[0] += out.requests - reserve
            log(f"a13 {n}: {'ok' if out.verdict is not None else 'FAIL ' + str(out.error)} "
                f"({out.model}, spent={spent[0]}/{args.max_requests})")
            if out.verdict is not None:
                j = load_judgement(ev, judge_dir)
                f_, m_ = a13_summary(j)
                for kk in by_ckey[ev.contract_key]:
                    rows[kk].update(a13="fail" if (f_ or m_) else "pass",
                                    a13_false=f_, a13_missing=m_)
            return out

        with ThreadPoolExecutor(max_workers=4) as ex:
            list(ex.map(judge_one, need))
        log(f"a13 done: {spent[0]} requests spent")

    # ---- recommendation ----------------------------------------------------
    for k, r in rows.items():
        u = canon[k]
        cold = u.repo in cold_repos
        r["cold"] = cold
        r["rec"] = recommend(r["lint_blocks"], r.get("orphans"),
                             r["a13"], cold, r["rung"])

    # ---- write TSV ---------------------------------------------------------
    cols = ["ukey", "unit", "repo", "family", "rung", "locus", "rec",
            "lint", "lint_codes", "orphans", "orphan_cand", "orphan_names",
            "a13", "a13_false", "a13_missing", "cold", "env_src",
            "copies", "rev_drift", "path"]
    with out_tsv.open("w") as fh:
        fh.write("\t".join(cols) + "\n")
        for k, r in sorted(rows.items()):
            fh.write("\t".join([
                k, r["unit"], r["repo"], r["family"],
                "" if r["rung"] is None else str(r["rung"]),
                r["locus"], r["rec"],
                "block" if r["lint_blocks"] else ("warn" if r["lint_warn"] else "ok"),
                "|".join(r["lint_codes"]),
                "" if r.get("orphans") is None else str(r["orphans"]),
                "" if r.get("orphan_cand") is None else str(r["orphan_cand"]),
                "|".join(r.get("orphan_names", [])[:12]),
                r["a13"], str(r["a13_false"]), str(r["a13_missing"]),
                "1" if r["cold"] else "0",
                "1" if r["has_env"] else "0",
                str(r["n_copies"]), "1" if r["rev_drift"] else "0", r["path"],
            ]) + "\n")
    log(f"wrote {out_tsv} ({len(rows)} rows)")

    # ---- validation + report ----------------------------------------------
    rep = build_report(rows, canon, fams, trials, cold_repos, log,
                       by_key=by_key, repo=repo)
    out_md.write_text(rep)
    log(f"wrote {out_md}")
    return 0


# --------------------------------------------------------------------------
# analysis
# --------------------------------------------------------------------------

def _confusion(flagged: dict[str, bool], outcome: dict[str, bool]) -> dict:
    tp = sum(1 for k, f in flagged.items() if f and k in outcome and not outcome[k])
    fp = sum(1 for k, f in flagged.items() if f and k in outcome and outcome[k])
    fn = sum(1 for k, f in flagged.items() if not f and k in outcome and not outcome[k])
    tn = sum(1 for k, f in flagged.items() if not f and k in outcome and outcome[k])
    n = tp + fp + fn + tn
    prec = tp / (tp + fp) if tp + fp else None
    rec = tp / (tp + fn) if tp + fn else None
    base = (tp + fn) / n if n else None
    return {"tp": tp, "fp": fp, "fn": fn, "tn": tn, "n": n,
            "precision": prec, "recall": rec, "base": base}


def build_report(rows, canon, fams, trials, cold_repos, log,
                 by_key=None, repo=None) -> str:
    L: list[str] = []
    L.append("# Bank gate report — whole gate stack over the packaged bank")
    L.append("")
    L.append(f"*Generated {time.strftime('%Y-%m-%d %H:%M:%S')} by `scripts/gate_bank.py`. "
             "Gates: `scripts/ops/task_lint.py` (deterministic) → orphan count "
             "(`cgscan`, reachability over the restored tree) → A13 contract-vs-gold "
             "judge (OpenRouter free tier, ≤300 requests, cached).*")
    L.append("")

    # unit -> family gate view: a family's gate verdict is the verdict of its
    # canonical L2 unit (the rung a trial would escalate to), else its best dir.
    def canon_l2(fam: str) -> str | None:
        cands = [n for n, u in canon.items()
                 if u.family == fam and (u.level or -1) >= 2]
        if not cands:
            cands = [n for n, u in canon.items() if u.family == fam]
        if not cands:
            return None
        return min(cands, key=lambda n: (canon[n].level or 99, n))

    # outcome per family: flipped = >=1 pass at L2
    esc_fams = {f: o for f, o in fams.items() if o.escalated}
    outcome = {f: o.flipped for f, o in esc_fams.items()}

    # unit-dir cohort: the CGCOV universe — dose_response L2 dirs with known
    # outcomes (~119 primary + 8 /tmp-pending). This is the cohort the
    # documented ~39% non-flip base rate was measured on; flag values come
    # from each dir's canonical logical unit (same famkey+rung, newest rev).
    from openswe_traces import cg_coverage as cc
    try:
        universe = cc.build_universe()
    except Exception as exc:  # noqa: BLE001
        log(f"cg universe FAIL: {exc}")
        universe = []
    path2ukey = {}
    for k, refs in by_key.items():
        for u in refs:
            path2ukey[str(u.path)] = k

    # ---------- confusion matrices --------------------------------------
    L.append("## Validation against trial outcomes")
    L.append("")
    L.append("Flag = the gate says do-not-trial (predicted non-flip). "
             "Positive = the unit/family truly did not flip.")
    L.append("")

    def row_flag(k, gate) -> bool:
        r = rows.get(k)
        if r is None:
            return False
        if gate == "lint":
            return bool(r["lint_blocks"])
        if gate == "b10":
            return "B10" in r["lint_blocks"]
        if gate == "orphans":
            return ((r["rung"] or -1) >= 2 and r.get("orphans") is not None
                    and r["orphans"] >= ORPHAN_BLOCK)
        if gate == "gap>=1":
            return ((r["rung"] or -1) >= 2 and r.get("orphans") is not None
                    and r["orphans"] >= 1)
        if gate == "a13":
            return r["a13"] == "fail" or (r["cold"] and r["a13"] == "unjudged")
        if gate == "combined":
            return r["rec"] in ("drop", "repair")
        return False

    def fam_flag(f, gate) -> bool:
        n = canon_l2(f)
        return row_flag(n, gate) if n else False

    gates = ["lint", "b10", "orphans", "gap>=1", "a13", "combined"]

    def conf_row(g, c):
        if c["precision"] is None:
            return f"| {g} | {c['tp']} | {c['fp']} | {c['fn']} | {c['tn']} | — | — | — |"
        return (f"| {g} | {c['tp']} | {c['fp']} | {c['fn']} | {c['tn']} | "
                f"{c['precision']:.0%} | {c['recall']:.0%} | {c['base']:.0%} |")

    # primary: the CGCOV unit-dir cohort — L2 dirs with known outcomes
    prim_units = [u for u in universe if not u.get("pending")]
    pend_units = [u for u in universe if u.get("pending")]

    def dir_flag(u, gate):
        return row_flag(path2ukey.get(u["rel"], ""), gate)

    prim_flag_out = {u["rel"]: u["flipped"] for u in prim_units}
    n_nonflip = sum(1 for v in prim_flag_out.values() if not v)
    L.append(f"### Primary cohort — {len(prim_units)} L2 unit dirs with known "
             f"outcomes ({n_nonflip} non-flip, base "
             f"{n_nonflip/max(1,len(prim_units)):.0%} vs the documented ~39%)")
    L.append("")
    L.append("| gate | TP | FP | FN | TN | precision | recall | base |")
    L.append("|---|---:|---:|---:|---:|---:|---:|---:|")
    confs = {}
    for g in gates:
        c = _confusion({u["rel"]: dir_flag(u, g) for u in prim_units},
                       prim_flag_out)
        confs[g] = c
        L.append(conf_row(g, c))
    L.append("")
    comb = confs["combined"]
    if comb["precision"] is not None and comb["base"] is not None:
        verdict = ("BEATS" if comb["precision"] > comb["base"] else
                   "does NOT beat")
        L.append(f"**Combined recommendation {verdict} the base rate** "
                 f"({comb['precision']:.0%} vs {comb['base']:.0%}).")
    L.append("")
    if pend_units:
        L.append(f"Robustness (+{len(pend_units)} /tmp-staged client-go units):")
        L.append("")
        L.append("| gate | TP | FP | FN | TN | precision | recall | base |")
        L.append("|---|---:|---:|---:|---:|---:|---:|---:|")
        all_units = prim_units + pend_units
        for g in gates:
            c = _confusion({u["rel"]: dir_flag(u, g) for u in all_units},
                           {u["rel"]: u["flipped"] for u in all_units})
            L.append(conf_row(g, c))
        L.append("")

    # secondary: family-level, the 204-family ledger
    prim = {f: o for f, o in esc_fams.items() if o.l0 == "fail"}
    L.append(f"### Family-level cohorts — {len(prim)} L0-failed escalated "
             f"families and all {len(esc_fams)} escalated families")
    L.append("")
    L.append(f"L0-failed families with >=L2 trials: **{len(prim)}** "
             f"({sum(1 for o in prim.values() if not o.flipped)} non-flip).")
    L.append("")
    L.append("| gate | TP | FP | FN | TN | precision | recall | base |")
    L.append("|---|---:|---:|---:|---:|---:|---:|---:|")
    for g in gates:
        flag = {f: fam_flag(f, g) for f in prim}
        c = _confusion(flag, {f: o.flipped for f, o in prim.items()})
        L.append(conf_row(g, c))
    L.append("")
    L.append(f"All {len(esc_fams)} families with >=L2 trials:")
    L.append("")
    L.append("| gate | TP | FP | FN | TN | precision | recall | base |")
    L.append("|---|---:|---:|---:|---:|---:|---:|---:|")
    for g in gates:
        flag = {f: fam_flag(f, g) for f in esc_fams}
        c = _confusion(flag, outcome)
        L.append(conf_row(g, c))
    L.append("")

    # ---------- cohort flip rates for P(flip) ---------------------------
    # measured on the primary unit-dir cohort: flip rate among unflagged vs
    # flagged-by-combined dirs
    sel_clean = [u for u in prim_units if not dir_flag(u, "combined")]
    sel_flag = [u for u in prim_units if dir_flag(u, "combined")]
    rate_clean = (sum(u["flipped"] for u in sel_clean) / len(sel_clean)
                  if sel_clean else 0.0)
    rate_flag = (sum(u["flipped"] for u in sel_flag) / len(sel_flag)
                 if sel_flag else 0.0)

    # ---------- drop list -------------------------------------------------
    L.append("## Drop list — units the gates say should NEVER be trialed")
    L.append("")
    drops = [(n, r) for n, r in sorted(rows.items()) if r["rec"] == "drop"]
    L.append(f"{len(drops)} units carry an unfixable defect (B10 digest oracle — "
             "the hidden suite asserts a literal digest of an internal serialisation; "
             "unsolvable at every rung including L5).")
    L.append("")
    L.append("| unit | evidence |")
    L.append("|---|---|")
    for n, r in drops:
        ev = "; ".join(r["lint_msgs"])[:300] or "B10"
        L.append(f"| `{n}` | {ev} |")
    L.append("")

    # ---------- ranked escalation list ------------------------------------
    L.append("## Ranked escalation list — the 92 unescalated families")
    L.append("")
    unesc = [f for f, o in sorted(fams.items()) if not o.escalated]
    L.append(f"{len(unesc)} families were screened (>=1 trial row) but never ran "
             f"a >=L2 trial. P(flip) is the measured flip rate of the family's "
             f"gate-verdict cohort on the primary cohort: clean={rate_clean:.0%}, "
             f"flagged={rate_flag:.0%}, drop=0. Expected trials are k=1-sequential "
             f"(sweep_seq.sh, <= {MAX_ROUNDS} rounds): E = 1 + q + q², q = 1−P. "
             f"Token cost at {TOKEN_COST/1e6:.2f}M/trial.")
    L.append("")
    L.append("| # | family | repo | L0 | gate verdict | P(flip) | E[trials] | E[tokens] | notes |")
    L.append("|---:|---|---|---|---|---:|---:|---:|---|")
    rank = []
    for f in unesc:
        o = fams[f]
        n = canon_l2(f)
        r = rows.get(n) if n else None
        rec = r["rec"] if r else "trial"
        l0 = o.l0
        notes = []
        if l0 in ("pass", "mixed"):
            p = 0.0
            notes.append("passed L0 — easy, nothing to certify")
        elif l0 == "err":
            notes.append("never screened (all trials errored)")
            p = {"drop": 0.0, "repair": rate_flag, "trial": rate_clean}[rec]
        else:
            p = {"drop": 0.0, "repair": rate_flag, "trial": rate_clean}[rec]
        if n is None:
            notes.append("no packaged unit dir")
        elif canon[n].level is None or canon[n].level < 2:
            notes.append(f"no L2+ unit packaged (best: {n})")
        if r and r["a13"] == "fail":
            notes.append(f"A13: {r['a13_false']} false/{r['a13_missing']} missing")
        q = 1 - p
        e_trials = min(MAX_ROUNDS, 1 + q + q * q)
        rank.append({"family": f, "repo": (canon[n].repo if n else ""),
                     "l0": l0, "rec": rec, "p": p, "e_trials": e_trials,
                     "e_tokens": e_trials * TOKEN_COST,
                     "notes": "; ".join(notes)})
    rank.sort(key=lambda x: (-x["p"], x["family"]))
    for i, rr in enumerate(rank, 1):
        L.append(f"| {i} | {rr['family']} | {rr['repo']} | {rr['l0']} | {rr['rec']} | "
                 f"{rr['p']:.0%} | {rr['e_trials']:.2f} | {rr['e_tokens']/1e6:.2f}M | {rr['notes']} |")
    L.append("")

    tot_e = sum(r["e_trials"] for r in rank)
    tot_tok = sum(r["e_tokens"] for r in rank)
    tot_yield = sum(1 - (1 - r["p"]) ** MAX_ROUNDS for r in rank)
    k3_trials = len(unesc) * MAX_ROUNDS
    k3_tok = k3_trials * TOKEN_COST
    k3_yield = sum(1 - (1 - r["p"]) ** MAX_ROUNDS for r in rank)  # same within-3 yield
    L.append("## Projected cost — ranked k=1-sequential vs k=3-everything")
    L.append("")
    L.append("| plan | families | expected trials | expected tokens | expected certified units |")
    L.append("|---|---:|---:|---:|---:|")
    L.append(f"| escalate all 92 at k=3 | {len(unesc)} | {k3_trials:.0f} | {k3_tok/1e6:.0f}M | {k3_yield:.1f} |")
    L.append(f"| ranked k=1-sequential (this list) | {len(unesc)} | {tot_e:.0f} | {tot_tok/1e6:.0f}M | {tot_yield:.1f} |")
    gate_sel = [r for r in rank if r["rec"] == "trial" and r["p"] > 0]
    ge = sum(r["e_trials"] for r in gate_sel)
    gy = sum(1 - (1 - r["p"]) ** MAX_ROUNDS for r in gate_sel)
    L.append(f"| gate-filtered: trial-rec only | {len(gate_sel)} | {ge:.0f} | {ge*TOKEN_COST/1e6:.0f}M | {gy:.1f} |")
    L.append("")
    L.append(f"k=1-sequential saves {(k3_tok - tot_tok)/1e6:.0f}M tokens "
             f"({(k3_tok - tot_tok)/max(1,k3_tok):.0%}) versus blind k=3 — before "
             "counting the drop/repair units it refuses to buy trials for.")
    L.append("")

    # ---------- inventory -------------------------------------------------
    L.append("## Inventory")
    L.append("")
    L.append(f"- unit names gated: {len(rows)} "
             f"(across tasks_composerver, tasks, work, dose_response, harbor_nex)")
    L.append(f"- trials in ledger: {len(trials)} "
             f"({sum(t.tokens for t in trials)/1e6:.0f}M tokens in+out)")
    L.append(f"- families with trial rows: {len(fams)}; escalated (>=L2): {len(esc_fams)}; "
             f"unescalated: {len(unesc)}")
    L.append(f"- flipped at L2: {sum(1 for o in esc_fams.values() if o.flipped)}; "
             f"double-fail: {sum(1 for o in esc_fams.values() if not o.flipped)}")
    L.append(f"- cold repos (<3 banked units, A13 gated): {sorted(cold_repos)}")
    judged = sum(1 for r in rows.values() if r["a13"] in ("pass", "fail"))
    L.append(f"- A13 verdicts applied: {judged} units "
             f"({sum(1 for r in rows.values() if r['a13'] == 'fail')} fail)")
    L.append("")
    return "\n".join(L)


if __name__ == "__main__":
    raise SystemExit(main())

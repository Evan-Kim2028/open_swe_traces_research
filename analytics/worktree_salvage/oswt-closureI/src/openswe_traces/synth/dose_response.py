"""Dose-response run: fabricated-unit grid, Harbor packaging, trial runner.

Arm 1 fabricates Go units (`synth.fabricate`) crossing achieved closure ratio
with an |S| band so ratio and excision size are not confounded.  Each reached
unit is packaged as Harbor ``-L0`` / ``-L2`` task dirs (affordance levels -2
and 0) with the agent allowlist pinned to the OpenRouter host.  The runner
launches `harbor run` jobs, is resume-safe against ``trials.parquet``, backs
off on 429s, and records one row per trial attempt.
"""

from __future__ import annotations

import csv
import json
import math
import os
import re
import shutil
import subprocess
import sys
import time
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path

from openswe_traces.pipeline.safety import apply_solver_network, assert_harbor_safe
from openswe_traces.synth import fabricate as fab
from openswe_traces.synth.affordance import (
    HiddenTest,
    build_affordance_levels,
    render_unsolv_dockerfile,
    render_unsolv_task_toml,
)

# --------------------------------------------------------------------------- #
# grid definition
# --------------------------------------------------------------------------- #
RATIO_TARGETS: tuple[float, ...] = (0.1, 0.2, 0.4, 0.8, 1.5, 3.0)
S_BANDS: dict[str, tuple[int, int]] = {"small": (8, 10), "large": (14, 17)}
# Disjoint seed sets per band: unit_name() is domain-s{seed}-r{ratio*100}, so a
# shared seed across bands at the same target would collide.
SEEDS: dict[str, tuple[int, ...]] = {
    "small": (11, 23, 37, 42),
    "large": (53, 67, 71, 89),
}
DOMAINS: tuple[str, ...] = ("store", "sched")
N_FUNCS = 24

# A cell counts as reached when the closest achievable ratio in-band is nearer
# (log2) to the target than a doubling grid neighbour: |log2(ach/tgt)| <= 0.5.
HIT_LOG2_TOL = 0.5

OPENROUTER_HOSTS: tuple[str, ...] = ("openrouter.ai", "*.openrouter.ai")
FAB_AGENT_TIMEOUT_SEC = 3600

LEVELS: tuple[int, ...] = (-2, 0)  # affordance -2 -> Harbor L0, 0 -> L2

GRID_FIELDS = (
    "unit",
    "domain",
    "s_band",
    "seed",
    "ratio_req",
    "ratio_ach",
    "S_size",
    "internal",
    "boundary_in",
    "boundary_out",
    "lines",
    "proof",
)

TRIAL_FIELDS = (
    "unit",
    "arm",
    "level",
    "model",
    "attempt",
    "reward",
    "wall_s",
    "tokens_in",
    "tokens_out",
    "job_dir",
)


# --------------------------------------------------------------------------- #
# grid planning + generation
# --------------------------------------------------------------------------- #
@dataclass(frozen=True)
class CellPlan:
    domain: str
    band: str
    ratio_req: float
    reached: bool
    ratio_ach: float | None
    stats: fab.UnitStats | None


def plan_cell(domain: str, ratio: float, band: str, seed: int) -> CellPlan:
    """Closest achievable (ratio, |S|) in ``band`` for one seed; seed does not
    change the achievable graph, so this is identical for all seeds."""
    lo, hi = S_BANDS[band]
    try:
        cfg = fab.search_config(domain, N_FUNCS, ratio, seed, s_band=(lo, hi))
    except ValueError:
        return CellPlan(domain, band, ratio, False, None, None)
    stats = fab.compute_stats(domain, cfg)
    hit = abs(math.log2(stats.ratio / ratio)) <= HIT_LOG2_TOL if stats.ratio > 0 else False
    return CellPlan(domain, band, ratio, hit, stats.ratio, stats)


def plan_grid() -> list[CellPlan]:
    """One CellPlan per (domain, band, ratio); seeds do not change reachability."""
    return [
        plan_cell(domain, ratio, band, SEEDS[band][0])
        for domain in DOMAINS
        for band in S_BANDS
        for ratio in RATIO_TARGETS
    ]


def generate_grid(
    fab_root: Path,
    grid_csv: Path,
    *,
    log_path: Path | None = None,
    proof: bool = True,
    scratch_root: Path | None = None,
) -> list[dict[str, object]]:
    """Generate every reached cell's units under ``fab_root``; write grid.csv."""
    fab_root = Path(fab_root)
    fab_root.mkdir(parents=True, exist_ok=True)
    rows: list[dict[str, object]] = []
    for domain in DOMAINS:
        for band in S_BANDS:  # noqa: PLC0206 - iterating band names only
            for ratio in RATIO_TARGETS:
                cell = plan_cell(domain, ratio, band, SEEDS[band][0])
                for seed in SEEDS[band]:
                    row: dict[str, object] = {
                        "unit": "",
                        "domain": domain,
                        "s_band": band,
                        "seed": seed,
                        "ratio_req": ratio,
                        "ratio_ach": round(cell.ratio_ach, 4) if cell.ratio_ach is not None else "",
                        "S_size": "",
                        "internal": "",
                        "boundary_in": "",
                        "boundary_out": "",
                        "lines": "",
                        "proof": "unreachable",
                    }
                    if cell.reached:
                        unit = f"{domain}-s{seed}-r{round(ratio * 100):03d}"
                        out = fab_root / unit
                        manifest = out / "manifest.json"
                        cached: dict[str, object] | None = None
                        if manifest.is_file():
                            cached = json.loads(manifest.read_text(encoding="utf-8"))
                        res = None
                        if cached is not None and cached.get("proof", {}).get("ok") is True:
                            mod_go = next(
                                (
                                    p
                                    for p in sorted((out / "module").glob("*.go"))
                                    if not p.name.endswith("_test.go")
                                ),
                                None,
                            )
                            loc = mod_go.read_text(encoding="utf-8").count("\n") if mod_go else ""
                            row.update(
                                unit=unit,
                                ratio_ach=round(float(cached["ratio_achieved"]), 4),
                                S_size=cached["excision"]["S_size"],
                                internal=cached["edges"]["internal"],
                                boundary_in=cached["edges"]["boundary_in"],
                                boundary_out=cached["edges"]["boundary_out"],
                                lines=loc,
                                proof="ok",
                            )
                        else:
                            try:
                                res = fab.generate_unit(
                                    seed=seed,
                                    n_funcs=N_FUNCS,
                                    ratio=ratio,
                                    out=out,
                                    domain=domain,
                                    proof=proof,
                                    scratch_root=scratch_root,
                                    s_band=S_BANDS[band],
                                )
                            except ValueError:
                                res = None
                        if res is not None:
                            st = res.stats
                            row.update(
                                unit=unit,
                                ratio_ach=round(st.ratio, 4),
                                S_size=len(st.S),
                                internal=st.internal,
                                boundary_in=st.in_edges,
                                boundary_out=st.out_edges,
                                lines=st.loc,
                                proof="ok" if res.proof.get("ok") else ("failed" if proof else "skipped"),
                            )
                    rows.append(row)
                    _log(log_path, "grid_row", {k: row[k] for k in GRID_FIELDS})
    grid_csv.parent.mkdir(parents=True, exist_ok=True)
    with grid_csv.open("w", encoding="utf-8", newline="") as fh:
        w = csv.DictWriter(fh, fieldnames=list(GRID_FIELDS))
        w.writeheader()
        for row in rows:
            w.writerow({k: row.get(k, "") for k in GRID_FIELDS})
    return rows


# --------------------------------------------------------------------------- #
# packaging: fabricated unit -> Harbor -L0 / -L2 dirs
# --------------------------------------------------------------------------- #
def fab_l0_instruction(bugreport: str) -> str:
    """Rebase the generated bug report's repro onto the Harbor task layout."""
    head = bugreport.split("Reproduce with:")[0].rstrip()
    return (
        head
        + "\n\nReproduce with:\n\n```\ngo test -count=1 ./...\n```\n\n"
        "Work in `/app`. Keep unrelated tests passing.\n"
    )


def _read(unit_dir: Path, rel: str) -> str:
    return (unit_dir / rel).read_text(encoding="utf-8")


def package_fab_unit(
    unit_dir: Path,
    dest_root: Path,
    *,
    allowed_hosts: tuple[str, ...] = OPENROUTER_HOSTS,
    levels: tuple[int, ...] = LEVELS,
) -> dict[int, Path]:
    """Stage the fabricated unit as an A0-shaped task, then derive L0/L2."""
    unit_dir = Path(unit_dir)
    manifest = json.loads(_read(unit_dir, "manifest.json"))
    unit = manifest["unit"]
    hidden_rel = sorted(p.name for p in (unit_dir / "hidden").glob("*_test.go"))
    if not hidden_rel:
        raise FileNotFoundError(f"no hidden suite under {unit_dir}/hidden")
    hidden = [
        HiddenTest(
            relpath=rel,
            content=_read(unit_dir, f"hidden/{rel}"),
            one_liner=f"black-box property suite for the fabricated {manifest['domain']} unit",
        )
        for rel in hidden_rel
    ]

    stage = dest_root / "_stage" / unit
    if stage.exists():
        shutil.rmtree(stage)
    (stage / "environment" / "src").mkdir(parents=True)
    (stage / "tests" / "hidden").mkdir(parents=True)
    shutil.copytree(unit_dir / "excised_tree", stage / "environment" / "src", dirs_exist_ok=True)
    (stage / "environment" / "Dockerfile").write_text(render_unsolv_dockerfile(), encoding="utf-8")
    (stage / "task.toml").write_text(
        render_unsolv_task_toml(
            agent_timeout_sec=FAB_AGENT_TIMEOUT_SEC, allowed_hosts=allowed_hosts
        ),
        encoding="utf-8",
    )
    contract = _read(unit_dir, "_author/contract.md")
    (stage / "instruction.md").write_text(contract, encoding="utf-8")
    for name in ("gold.patch", "cheat.patch"):
        shutil.copy2(unit_dir / "_author" / name, stage / "tests" / name)
    for rel in hidden_rel:
        shutil.copy2(unit_dir / "hidden" / rel, stage / "tests" / "hidden" / rel)

    pkg = fab.DOMAINS[manifest["domain"]]["pkg"]
    return build_affordance_levels(
        stage,
        hidden,
        levels=levels,
        dest_root=dest_root,
        family=unit,
        instructions={-2: fab_l0_instruction(_read(unit_dir, "_author/bugreport.md")), 0: contract},
        changed_symbols=tuple(manifest["excision"]["S"]),
        changed_files=(f"{pkg}.go",),
        name_scheme="L",
        allowed_hosts=list(allowed_hosts),
    )


def package_grid(fab_root: Path, tasks_root: Path) -> list[Path]:
    """Package every fabricated unit dir under ``fab_root``; returns task dirs."""
    out: list[Path] = []
    for unit_dir in sorted(Path(fab_root).iterdir()):
        if not unit_dir.is_dir() or not (unit_dir / "manifest.json").is_file():
            continue
        out.extend(package_fab_unit(unit_dir, Path(tasks_root)).values())
    stage = Path(tasks_root) / "_stage"
    if stage.is_dir():
        shutil.rmtree(stage)
    return out


# --------------------------------------------------------------------------- #
# arm 2: authored excisions from the composerver bank
# --------------------------------------------------------------------------- #
# Word-boundary rewrites (ordered) from closure_A/trees.json: the helm/kops
# authored patches and hidden tests use the identity-pass brandings
# (chartkit/clustkit); the materialised base trees use example.internal/<repo>.
ARM2_REBRAND: dict[str, tuple[tuple[str, str], ...]] = {
    "helm": (
        ("example.internal/chartkit/v4", "example.internal/helm"),
        ("example.internal/chartkit", "example.internal/helm"),
        ("ChartKit", "Helm"),
        ("chartkit", "helm"),
    ),
    "kops": (
        ("example.internal/clustkit", "example.internal/kops"),
        ("ClusterKit", "Kubernetes"),
        ("Clustkit", "kOps"),
        ("clustkit", "kops"),
    ),
}

# 12 authored units spanning internal_edges 0..53, lines_removed 150-300 where
# possible.  Metrics from analytics/research/closure_vs_flip.md (closure-A).
ARM2_PICKS: tuple[dict[str, object], ...] = (
    {"repo": "kops", "unit": "memfs", "internal_edges": 0, "lines_removed": 84,
     "boundary_in": 6, "boundary_out": 26, "ratio": 0.0,
     "why": "edges=0 anchor; closest to the 150-300 line band among 0-edge units"},
    {"repo": "helm", "unit": "ignorerules", "internal_edges": 2, "lines_removed": 152,
     "boundary_in": 21, "boundary_out": 38, "ratio": 0.0339,
     "why": "edges=2, in band"},
    {"repo": "kops", "unit": "flagbuilder", "internal_edges": 2, "lines_removed": 192,
     "boundary_in": 15, "boundary_out": 72, "ratio": 0.023,
     "why": "edges=2, in band; second repo at edges=2"},
    {"repo": "helm", "unit": "depresolver", "internal_edges": 3, "lines_removed": 199,
     "boundary_in": 8, "boundary_out": 50, "ratio": 0.0517,
     "why": "edges=3, in band; measured flip 'none' (hard unit)"},
    {"repo": "helm", "unit": "provenance", "internal_edges": 4, "lines_removed": 224,
     "boundary_in": 37, "boundary_out": 45, "ratio": 0.0488,
     "why": "edges=4, in band"},
    {"repo": "helm", "unit": "repindex", "internal_edges": 5, "lines_removed": 192,
     "boundary_in": 23, "boundary_out": 54, "ratio": 0.0649,
     "why": "edges=5, in band"},
    {"repo": "kops", "unit": "addonparse", "internal_edges": 7, "lines_removed": 159,
     "boundary_in": 19, "boundary_out": 35, "ratio": 0.1296,
     "why": "edges=7, in band"},
    {"repo": "helm", "unit": "kindsorter", "internal_edges": 9, "lines_removed": 185,
     "boundary_in": 49, "boundary_out": 40, "ratio": 0.1011,
     "why": "edges=9, in band"},
    {"repo": "helm", "unit": "chartloader", "internal_edges": 10, "lines_removed": 291,
     "boundary_in": 111, "boundary_out": 93, "ratio": 0.049,
     "why": "edges=10, in band; boundary-heavy counterpoint"},
    {"repo": "helm", "unit": "coalesce", "internal_edges": 14, "lines_removed": 223,
     "boundary_in": 49, "boundary_out": 47, "ratio": 0.1458,
     "why": "edges=14, in band; measured flip L5"},
    {"repo": "gin", "unit": "formmapping", "internal_edges": 45, "lines_removed": 430,
     "boundary_in": 92, "boundary_out": 118, "ratio": 0.2143,
     "why": "edges=45 top anchor; over the line band but needed for range; measured flip L5"},
    {"repo": "helm", "unit": "strvalsparser", "internal_edges": 53, "lines_removed": 455,
     "boundary_in": 16, "boundary_out": 149, "ratio": 0.3212,
     "why": "edges=53 extreme; over the line band but the densest authored excision"},
)


def _rebrand_text(text: str, pairs: tuple[tuple[str, str], ...]) -> str:
    for old, new in pairs:
        text = re.sub(rf"\b{re.escape(old)}\b", new, text)
    return text


def _rebrand_dir(root: Path, pairs: tuple[tuple[str, str], ...]) -> int:
    n = 0
    for path in root.rglob("*"):
        if path.is_file() and path.suffix in {".go", ".patch", ".md", ".sh", ".toml", ".json"}:
            text = path.read_text(encoding="utf-8", errors="replace")
            new = _rebrand_text(text, pairs)
            if new != text:
                path.write_text(new, encoding="utf-8")
                n += 1
    return n


def _retarget_checksums(task_dir: Path) -> int:
    """Re-embed sha256 of hidden files into tests/test.sh after edits."""
    sh = task_dir / "tests" / "test.sh"
    if not sh.is_file():
        return 0
    text = sh.read_text(encoding="utf-8")
    hidden_root = task_dir / "tests" / "hidden"
    n = 0
    for f in sorted(hidden_root.rglob("*")):
        if not f.is_file():
            continue
        import hashlib

        digest = hashlib.sha256(f.read_bytes()).hexdigest()
        rel = f.relative_to(hidden_root).as_posix()
        for target in (f"$HIDDEN/{rel}", f"/app/{rel}"):
            new_text, k = re.subn(
                rf'echo "[0-9a-f]{{64}}  {re.escape(target)}"', f'echo "{digest}  {target}"', text
            )
            if k:
                text, n = new_text, n + k
    if n:
        sh.write_text(text, encoding="utf-8")
    return n


def _module_name(src: Path) -> str:
    gomod = src / "go.mod"
    if not gomod.is_file():
        return ""
    for line in gomod.read_text(encoding="utf-8").splitlines():
        if line.startswith("module "):
            return line.split(None, 1)[1].strip()
    return ""


def _reverse_apply(tree: Path, patch: Path) -> None:
    proc = subprocess.run(
        ["patch", "-p1", "-R", "--forward", "--batch", "-i", str(patch.resolve()), "-d", str(tree)],
        capture_output=True,
        text=True,
        timeout=300,
        check=False,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"reverse patch {patch.name} failed in {tree}: {(proc.stderr or proc.stdout)[-800:]}")


def prepare_arm2_task(src_task: Path, dest: Path, repo: str) -> dict[str, object]:
    """Copy an authored task dir and repair materialisation: for helm/kops the
    shipped ``environment/src`` is the unexcised base (tests/patches still carry
    the identity-pass branding), so rebrand tests+patches to the src module and
    reverse-apply ``tests/gold.patch`` to produce the excised tree."""
    src_task = Path(src_task)
    dest = Path(dest)
    status: dict[str, object] = {"src": str(src_task), "dest": str(dest)}
    if not (src_task / "environment" / "src").is_dir():
        status["status"] = "missing_src"
        return status
    if dest.exists():
        shutil.rmtree(dest)
    shutil.copytree(src_task, dest, symlinks=True, ignore=shutil.ignore_patterns(".git", "__pycache__"))

    gold = dest / "tests" / "gold.patch"
    src_tree = dest / "environment" / "src"
    pairs = ARM2_REBRAND.get(repo, ())
    if pairs:
        n_files = _rebrand_dir(dest / "tests", pairs)
        instr = dest / "instruction.md"
        if instr.is_file():
            instr.write_text(_rebrand_text(instr.read_text(encoding="utf-8"), pairs), encoding="utf-8")
        status["rebranded_files"] = n_files
        if gold.is_file():
            _reverse_apply(src_tree, gold)
        ncs = _retarget_checksums(dest)
        status["checksums"] = ncs
    stubs = 0
    if gold.is_file():
        patched: set[Path] = set()
        for line in gold.read_text(encoding="utf-8").splitlines():
            m = re.match(r"\+\+\+ [ab]/(.+)$", line)
            if m:
                f = src_tree / m.group(1)
                if f.is_file():
                    patched.add(f)
        stubs = sum(
            f.read_text(encoding="utf-8", errors="replace").count('panic("excised')
            for f in patched
        )
    status["excised_stubs"] = stubs
    status["module"] = _module_name(src_tree)
    status["status"] = "excised" if stubs > 0 else "unexcised"
    return status


def prepare_arm2(
    src_root: Path,
    dest_root: Path,
    arm2_csv: Path,
    *,
    log_path: Path | None = None,
) -> list[dict[str, object]]:
    """Materialise the 12 arm-2 picks; write arm2.csv with per-level status."""
    fields = (
        "repo", "unit", "internal_edges", "lines_removed", "boundary_in", "boundary_out",
        "ratio", "why", "L0_status", "L0_stubs", "L2_status", "L2_stubs", "dest",
    )
    rows: list[dict[str, object]] = []
    for pick in ARM2_PICKS:
        repo, unit = str(pick["repo"]), str(pick["unit"])
        row: dict[str, object] = {k: pick.get(k, "") for k in fields}
        for lvl in (0, 2):
            src = Path(src_root) / repo / f"{unit}-L{lvl}"
            dest = Path(dest_root) / repo / f"{unit}-L{lvl}"
            try:
                st = prepare_arm2_task(src, dest, repo)
            except Exception as exc:  # noqa: BLE001 - record and continue
                st = {"status": f"error: {exc}"[:200]}
            row[f"L{lvl}_status"] = st["status"]
            row[f"L{lvl}_stubs"] = st.get("excised_stubs", "")
            row["dest"] = str(Path(dest_root) / repo)
            _log(log_path, "arm2_task", {"unit": unit, "level": lvl, **st})
        rows.append(row)
    arm2_csv.parent.mkdir(parents=True, exist_ok=True)
    with arm2_csv.open("w", encoding="utf-8", newline="") as fh:
        w = csv.DictWriter(fh, fieldnames=list(fields))
        w.writeheader()
        for row in rows:
            w.writerow(row)
    return rows


# --------------------------------------------------------------------------- #
# runner
# --------------------------------------------------------------------------- #
@dataclass(frozen=True)
class PlannedJob:
    task_dir: Path
    unit: str
    level: int
    arm: str
    model: str
    n_missing: int  # attempts still needed for (task, model)


_LEVEL_RE = re.compile(r"-L(\d+)$")
RATE_LIMIT_RE = re.compile(r"\b429\b|rate.?limit|too many requests", re.IGNORECASE)


def task_unit_level(task_dir: Path) -> tuple[str, int]:
    m = _LEVEL_RE.search(task_dir.name)
    if not m:
        raise ValueError(f"task dir name lacks -L<k> suffix: {task_dir.name}")
    return task_dir.name[: m.start()], int(m.group(1))


def infer_arm(task_dir: Path, dose_root: Path | None = None) -> str:
    try:
        task_dir.resolve().relative_to(Path(dose_root or "experiments/dose_response").resolve())
        return "1"
    except ValueError:
        return "2"


def model_slug(model: str) -> str:
    return re.sub(r"[^A-Za-z0-9_.-]+", "_", model)


def load_done_counts(trials_path: Path) -> dict[tuple[str, int, str], int]:
    """(unit, level, model) -> attempts already recorded in trials.parquet."""
    path = Path(trials_path)
    if not path.is_file():
        return {}
    import pandas as pd

    df = pd.read_parquet(path)
    counts: dict[tuple[str, int, str], int] = {}
    for row in df.itertuples(index=False):
        key = (str(row.unit), int(row.level), str(row.model))
        counts[key] = counts.get(key, 0) + 1
    return counts


def plan_jobs(
    task_dirs: list[Path],
    models: list[str],
    attempts: int,
    done: dict[tuple[str, int, str], int],
    *,
    dose_root: Path | None = None,
    limit: int | None = None,
) -> list[PlannedJob]:
    """Resume-safe plan: skip (task, model) pairs with >= ``attempts`` rows."""
    jobs: list[PlannedJob] = []
    planned = 0
    for task in task_dirs:
        unit, level = task_unit_level(task)
        arm = infer_arm(task, dose_root)
        for model in models:
            have = done.get((unit, level, model), 0)
            missing = max(0, attempts - have)
            if limit is not None:
                missing = min(missing, max(0, limit - planned))
            if missing:
                jobs.append(PlannedJob(task, unit, level, arm, model, missing))
                planned += missing
            if limit is not None and planned >= limit:
                return jobs
    return jobs


def solver_for(agent: str, model: str) -> str:
    m = model.lower()
    if "openrouter" in m or "openrouter" in agent.lower():
        return "openrouter"
    if "devin" in m or "devin" in agent.lower() or "swe-2" in m:
        return "devin"
    return "cursor"


def harbor_argv(
    task_dir: Path,
    *,
    agent: str,
    model: str,
    n_attempts: int,
    n_concurrent: int,
    jobs_dir: Path,
    job_name: str,
    max_retries: int = 2,
) -> list[str]:
    return [
        "harbor",
        "run",
        "--path",
        str(task_dir),
        "--agent",
        agent,
        "--model",
        model,
        "--n-attempts",
        str(n_attempts),
        "--n-concurrent",
        str(max(1, n_concurrent)),
        "--max-retries",
        str(max_retries),
        "--jobs-dir",
        str(jobs_dir),
        "--job-name",
        job_name,
        "--yes",
    ]


def load_env_files(paths: list[Path], environ: dict[str, str]) -> dict[str, str]:
    env = dict(environ)
    for path in paths:
        path = Path(path)
        if not path.is_file():
            continue
        for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            k, v = line.split("=", 1)
            env[k.strip()] = v.strip().strip('"').strip("'")
    return env


def classify_job(proc: subprocess.CompletedProcess[str], n_trials: int) -> str:
    blob = (proc.stdout or "") + "\n" + (proc.stderr or "")
    if RATE_LIMIT_RE.search(blob):
        return "rate_limit"
    if proc.returncode != 0:
        return "error"
    if n_trials == 0:
        return "no_trials"
    return "ok"


@dataclass
class RunResult:
    job_dir: Path
    status: str
    n_trials: int
    attempts_used: int = 0


def run_job(
    job: PlannedJob,
    *,
    agent: str,
    jobs_dir: Path,
    n_concurrent: int,
    env: dict[str, str],
    seq: int,
    max_job_retries: int = 4,
    backoff_base_s: float = 60.0,
    run=subprocess.run,
    sleep=time.sleep,
    log_path: Path | None = None,
) -> RunResult:
    """Launch one harbor job; retry the whole job on 429 / empty output."""
    from openswe_traces.pipeline.audit import audit_job

    solver = solver_for(agent, job.model)
    apply_solver_network(job.task_dir, solver)
    assert_harbor_safe(job.task_dir, solver=solver)
    job_name = f"{job.unit}-L{job.level}-{model_slug(job.model)}-seq{seq}"
    argv = harbor_argv(
        job.task_dir,
        agent=agent,
        model=job.model,
        n_attempts=job.n_missing,
        n_concurrent=n_concurrent,
        jobs_dir=jobs_dir,
        job_name=job_name,
    )
    job_dir = Path(jobs_dir) / job_name
    for attempt in range(max_job_retries + 1):
        _log(log_path, "job_start", {"job": job_name, "argv_agent": agent, "model": job.model})
        proc = run(argv, capture_output=True, text=True, check=False, env=env)
        trials = audit_job(job_dir)
        status = classify_job(proc, len(trials))
        _log(
            log_path,
            "job_done",
            {"job": job_name, "status": status, "rc": proc.returncode, "trials": len(trials)},
        )
        if status == "ok" or attempt == max_job_retries:
            return RunResult(job_dir=job_dir, status=status, n_trials=len(trials), attempts_used=attempt + 1)
        delay = min(backoff_base_s * (2**attempt), 1200.0)
        if status != "rate_limit" and len(trials) == 0 and proc.returncode == 0:
            delay = min(delay, 30.0)
        _log(log_path, "backoff", {"job": job_name, "status": status, "sleep_s": delay})
        sleep(delay)
    return RunResult(job_dir=job_dir, status=status, n_trials=len(trials), attempts_used=max_job_retries + 1)


def record_trials(
    job: PlannedJob,
    result: RunResult,
    trials_path: Path,
    *,
    done: dict[tuple[str, int, str], int],
) -> int:
    """Append one row per trial dir in the job; returns rows written."""
    import pandas as pd

    from openswe_traces.pipeline.audit import audit_job

    audits = audit_job(result.job_dir)
    if not audits:
        return 0
    have = done.get((job.unit, job.level, job.model), 0)
    rows = []
    for i, a in enumerate(audits, start=1):
        tin, tout = a.tokens_in, a.tokens_out
        rows.append(
            {
                "unit": job.unit,
                "arm": job.arm,
                "level": job.level,
                "model": job.model,
                "attempt": have + i,
                "reward": a.reward,
                "wall_s": round((a.wall_minutes or 0.0) * 60, 1),
                "tokens_in": tin,
                "tokens_out": tout,
                "job_dir": str(a.trial_dir),
            }
        )
    df = pd.DataFrame(rows, columns=list(TRIAL_FIELDS))
    path = Path(trials_path)
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.is_file():
        df = pd.concat([pd.read_parquet(path), df], ignore_index=True)
    df.to_parquet(path, index=False)
    done[(job.unit, job.level, job.model)] = have + len(rows)
    return len(rows)


def run_plan(
    jobs: list[PlannedJob],
    *,
    agent: str,
    jobs_dir: Path,
    trials_path: Path,
    n_concurrent: int = 1,
    env_files: list[Path] | None = None,
    max_job_retries: int = 4,
    done: dict[tuple[str, int, str], int] | None = None,
    log_path: Path | None = None,
    run=subprocess.run,
    sleep=time.sleep,
) -> dict[str, int]:
    env = load_env_files(env_files or [Path(".env")], dict(os.environ))
    done = done if done is not None else load_done_counts(trials_path)
    summary = {"jobs": 0, "trials": 0, "rate_limited": 0, "failed": 0}
    for seq, job in enumerate(jobs, start=1):
        result = run_job(
            job,
            agent=agent,
            jobs_dir=jobs_dir,
            n_concurrent=n_concurrent,
            env=env,
            seq=seq,
            max_job_retries=max_job_retries,
            run=run,
            sleep=sleep,
            log_path=log_path,
        )
        n = record_trials(job, result, trials_path, done=done)
        summary["jobs"] += 1
        summary["trials"] += n
        if result.status == "rate_limit":
            summary["rate_limited"] += 1
        elif result.status != "ok":
            summary["failed"] += 1
    return summary


# --------------------------------------------------------------------------- #
# CLI
# --------------------------------------------------------------------------- #
def main_cli(argv: list[str] | None = None) -> int:
    import argparse

    p = argparse.ArgumentParser(description="Dose-response Harbor runner (resume-safe).")
    p.add_argument("--task-dir", action="append", type=Path, default=[], help="Harbor task dir (repeatable)")
    p.add_argument("--tasks-glob", action="append", default=[], help="glob for task dirs, e.g. 'experiments/dose_response/tasks/*-L2'")
    p.add_argument("--task-list", type=Path, default=None, help="file with one task dir per line")
    p.add_argument("--agent", default="mini-swe-agent")
    p.add_argument("--model", action="append", required=True, help="litellm model string, e.g. openrouter/deepseek-v4-flash-0731:free (repeatable)")
    p.add_argument("--attempts", type=int, default=5)
    p.add_argument("--jobs-dir", type=Path, default=Path("experiments/dose_response/jobs"))
    p.add_argument("--trials", type=Path, default=Path("experiments/dose_response/trials.parquet"))
    p.add_argument("--dose-root", type=Path, default=Path("experiments/dose_response"))
    p.add_argument("--env-file", action="append", type=Path, default=None)
    p.add_argument("--n-concurrent", type=int, default=1)
    p.add_argument("--limit", type=int, default=None, help="max attempts to launch this invocation")
    p.add_argument("--max-job-retries", type=int, default=4)
    p.add_argument("--dry-run", action="store_true")
    p.add_argument("--log", type=Path, default=Path("outputs/closure_I.log"))
    args = p.parse_args(argv)

    task_dirs: list[Path] = list(args.task_dir)
    for g in args.tasks_glob:
        task_dirs.extend(sorted(Path().glob(g)))
    if args.task_list:
        task_dirs.extend(Path(ln.strip()) for ln in args.task_list.read_text().splitlines() if ln.strip())
    task_dirs = [d for d in dict.fromkeys(task_dirs) if (d / "task.toml").is_file()]

    done = load_done_counts(args.trials)
    jobs = plan_jobs(
        task_dirs, list(args.model), args.attempts, done, dose_root=args.dose_root, limit=args.limit
    )
    if args.dry_run:
        for j in jobs:
            print(f"{j.arm}\t{j.unit}-L{j.level}\t{j.model}\tn={j.n_missing}\t{j.task_dir}")
        print(f"{len(jobs)} jobs, {sum(j.n_missing for j in jobs)} attempts planned")
        return 0
    env_files = args.env_file
    if env_files is None:
        env_files = [Path(".env"), Path("/home/evan/Documents/eval_tasks/.env")]
    summary = run_plan(
        jobs,
        agent=args.agent,
        jobs_dir=args.jobs_dir,
        trials_path=args.trials,
        n_concurrent=args.n_concurrent,
        env_files=env_files,
        max_job_retries=args.max_job_retries,
        done=done,
        log_path=args.log,
    )
    print(json.dumps(summary))
    return 0


def _log(path: Path | None, event: str, payload: dict[str, object]) -> None:
    if path is None:
        return
    path.parent.mkdir(parents=True, exist_ok=True)
    row = {"ts": datetime.now(UTC).isoformat(timespec="seconds"), "event": event, **payload}
    with path.open("a", encoding="utf-8") as fh:
        fh.write(json.dumps(row, default=str) + "\n")


if __name__ == "__main__":
    sys.exit(main_cli())

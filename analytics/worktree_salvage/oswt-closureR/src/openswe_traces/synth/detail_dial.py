"""Detail-count x verification-affordance grid (closure-R).

Builds the 40-unit grid d in {1,2,4,6,8} x v in {low,high} x 4 seeds on the
store domain, all at one fixed structural config (|S| = 21, same removed
lines), so d and v are the only movers.  Every unit is locally proven (bare
fails, gold passes under both HIDDEN_SEEDs, cheat fails), audited per detail
(one trap flip per drawn detail: invisible at v=low, visible at v=high,
caught by its own hidden test), and checked by fab_fairness (incl. B3
not-a-complete-spec).  Units are then packaged as Harbor ``-L0`` task dirs
with the CURSOR agent allowlist and a per-detail verifier that writes
``/logs/verifier/details.json`` (one JSONL row per drawn detail + the guard).
"""

from __future__ import annotations

import csv
import hashlib
import json
import shutil
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path

from openswe_traces.synth import fabricate as fab
from openswe_traces.synth.affordance import render_unsolv_dockerfile
from openswe_traces.synth.harbor_tasks import instruction_self_check
from openswe_traces.synth.obfuscate import NO_WEB_CLAUSE
from openswe_traces.synth.rules import write_task_validation

GRID_D: tuple[int, ...] = (1, 2, 4, 6, 8)
GRID_V: tuple[str, ...] = ("low", "high")
GRID_SEEDS: tuple[int, ...] = (11, 23, 37, 42)
DOMAIN = "store"
N_FUNCS = 22
E_FIXED = 9
K_FIXED = 2

CURSOR_HOSTS: tuple[str, ...] = (
    "cursor.com",
    "*.cursor.com",
    "*.cursor.sh",
    "downloads.cursor.com",
)

GRID_FIELDS = (
    "unit", "domain", "seed", "d", "v", "S_size", "n_funcs", "module_loc",
    "removed_lines", "smoke_loc", "hidden_loc", "bugreport_lines",
    "details_drawn", "proof_ok", "audit_ok", "fairness_ok",
)


@dataclass(frozen=True)
class GridRow:
    unit: str
    domain: str
    seed: int
    d: int
    v: str
    S_size: int
    n_funcs: int
    module_loc: int
    removed_lines: int
    smoke_loc: int
    hidden_loc: int
    bugreport_lines: int
    details_drawn: str
    proof_ok: bool
    audit_ok: bool | None
    fairness_ok: bool | None


def fixed_config(domain: str, seed: int) -> fab.Config:
    """One structural config per domain; only the seed's constants vary.

    E=9 (every entry present, so every registry detail is drawable), K=2
    mixer stages, pipe canon mode, canon/misc(/place) groups excised, no
    consumers or fillers.  The dial suffix is set by generate_unit."""
    return fab.Config(
        domain=domain,
        seed=seed,
        n_funcs=N_FUNCS,
        target_ratio=0.0,
        E=E_FIXED,
        K=K_FIXED,
        canon_mode="pipe",
        canon_in_S=True,
        misc_in_S=True,
        place_in_S=domain == "sched",
        consumers=(),
        fill=0,
        consts=fab.DOMAINS[domain]["consts"](seed),
    )


def unit_name(seed: int, d: int, v: str) -> str:
    return f"{DOMAIN}-s{seed}-d{d}-v{v}"


def build_grid(
    fab_root: Path,
    grid_csv: Path,
    *,
    proof: bool = True,
    audit: bool = True,
    scratch_root: Path | None = None,
    log_path: Path | None = None,
) -> list[GridRow]:
    """Generate every grid cell; write grid.csv.  Resume-safe: a cell whose
    manifest already carries an ok proof and audit is skipped."""
    fab_root = Path(fab_root)
    fab_root.mkdir(parents=True, exist_ok=True)
    scratch = scratch_root or fab_root.parent / "proofs"
    rows: list[GridRow] = []
    for seed in GRID_SEEDS:
        for v in GRID_V:
            for d in GRID_D:
                # a fresh Config per cell: generate_unit sets dial_suffix on it,
                # so a shared object would leak the first cell's suffix into the
                # rest (module names, manifest "unit", packaged dir names).
                cfg = fixed_config(DOMAIN, seed)
                name = unit_name(seed, d, v)
                out = fab_root / name
                manifest_path = out / "manifest.json"
                cached = None
                if manifest_path.is_file():
                    cached = json.loads(manifest_path.read_text(encoding="utf-8"))
                res = None
                if (
                    cached is not None
                    and cached.get("proof", {}).get("ok") is True
                    and (not audit or cached.get("detail_audit", {}).get("ok") is True)
                ):
                    res = fab.UnitResult(cfg=cfg, stats=_stats_from_manifest(cached), unit_dir=out, proof=cached["proof"])
                if res is None:
                    res = fab.generate_unit(
                        seed=seed, n_funcs=N_FUNCS, ratio=0.0, out=out,
                        domain=DOMAIN, d=d, v=v, cfg=cfg, proof=proof, scratch_root=scratch,
                    )
                    _log(log_path, "unit_generated", {"unit": name, "proof_ok": bool(res.proof.get("ok"))})
                    if audit and res.proof.get("ok"):
                        drawn = set(fab.draw_details(DOMAIN, seed, d, present=set(fab.DOMAINS[DOMAIN]["entries"][: cfg.E])))
                        detail_audit = fab.audit_unit_details(out, cfg, drawn, v, scratch)
                        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
                        manifest["detail_audit"] = detail_audit
                        manifest_path.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
                        _log(log_path, "unit_audited", {"unit": name, "audit_ok": bool(detail_audit.get("ok"))})
                if res is None:
                    continue
                st = res.stats
                manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
                fairness_ok = None
                if manifest_path.is_file():
                    from openswe_traces.synth import fab_fairness

                    fairness_ok = fab_fairness.audit_unit(out).ok
                audit_ok = manifest.get("detail_audit", {}).get("ok") if audit else None
                row = GridRow(
                    unit=name,
                    domain=DOMAIN,
                    seed=seed,
                    d=d,
                    v=v,
                    S_size=len(st.S),
                    n_funcs=st.n_funcs,
                    module_loc=st.sizes["module_go_loc"],
                    removed_lines=st.sizes["removed_lines"],
                    smoke_loc=st.sizes["smoke_test_loc"],
                    hidden_loc=st.sizes["hidden_loc"],
                    bugreport_lines=st.sizes["bugreport_lines"],
                    details_drawn=",".join(str(x["id"]) for x in manifest.get("details", [])),
                    proof_ok=bool(res.proof.get("ok")),
                    audit_ok=audit_ok,
                    fairness_ok=fairness_ok,
                )
                rows.append(row)
                _log(log_path, "grid_row", {k: getattr(row, k) for k in GRID_FIELDS})
    grid_csv.parent.mkdir(parents=True, exist_ok=True)
    with grid_csv.open("w", encoding="utf-8", newline="") as fh:
        w = csv.DictWriter(fh, fieldnames=list(GRID_FIELDS))
        w.writeheader()
        for row in rows:
            w.writerow({k: getattr(row, k) for k in GRID_FIELDS})
    return rows


def _stats_from_manifest(m: dict) -> fab.UnitStats:
    exc = m["excision"]
    return fab.UnitStats(
        internal=m["edges"]["internal"],
        in_edges=m["edges"]["boundary_in"],
        out_edges=m["edges"]["boundary_out"],
        ratio=float(m["ratio_achieved"]),
        depth=m["graph"]["depth"],
        fanout_max=m["graph"]["fanout_max"],
        fanout_mean=float(m["graph"]["fanout_mean"]),
        S=tuple(exc["S"]),
        n_funcs=m["n_funcs"],
        loc=m["files"]["module_go_loc"],
        sizes=m["files"],
    )


# --------------------------------------------------------------------------- #
# Harbor packaging (L0 only, CURSOR allowlist, per-detail verifier)
# --------------------------------------------------------------------------- #
def render_dial_test_sh(unit_dir: Path) -> str:
    """Verifier: installs the hidden suite, runs ``go test -json``, and relies
    on the hidden tests themselves to write one JSONL row per detail to
    ``/logs/verifier/details.json`` (the per-detail pass/fail vector).  The
    aggregate reward is the go test exit code; a missing details file means
    the build failed."""
    hidden_rel = sorted(p.name for p in (unit_dir / "hidden").glob("*_test.go"))
    digest = hashlib.sha256((unit_dir / "hidden" / hidden_rel[0]).read_bytes()).hexdigest()
    rel = hidden_rel[0]
    return f"""#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
mkdir -p /logs/artifacts
if [ -d /pristine ]; then
  ( cd / && if command -v git >/dev/null 2>&1; then git diff --no-index --no-color pristine app; else diff -ruN pristine app; fi ) \\
    | sed -e 's|a/pristine/|a/|g' -e 's|b/app/|b/|g' -e 's|a/app/|a/|g' -e 's|b/pristine/|b/|g' -e 's|^--- pristine/|--- a/|' -e 's|^+++ app/|+++ b/|' \\
    > /logs/artifacts/agent.patch || true
fi
TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
HIDDEN="$TESTS_DIR/hidden"
install_hidden() {{
  rel="$1"
  dest="/app/$rel"
  mkdir -p "$(dirname "$dest")"
  cp "$HIDDEN/$rel" "$dest"
}}
checksum_fail() {{
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}}
echo "{digest}  $HIDDEN/{rel}" | sha256sum -c --status || checksum_fail "hidden/{rel}"
install_hidden "{rel}"
echo "{digest}  /app/{rel}" | sha256sum -c --status || checksum_fail "{rel}"
if go test -count=1 -timeout 15m -json ./... > /logs/verifier/go_test.json 2>&1; then
  rc=0
else
  rc=1
fi
if [ ! -f /logs/verifier/details.json ]; then
  echo '{{"id":"build_failed","pass":false}}' > /logs/verifier/details.json
fi
if [ $rc -eq 0 ]; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
fi
echo 0 > /logs/verifier/reward.txt
exit 1
"""


def l0_instruction(unit_dir: Path) -> str:
    bugreport = (unit_dir / "_author" / "bugreport.md").read_text(encoding="utf-8")
    head = bugreport.split("Reproduce with:")[0].rstrip()
    return (
        head
        + "\n\nReproduce with:\n\n```\ngo test -count=1 ./...\n```\n\n"
        "Work in `/app`. Keep unrelated tests passing.\n\n"
        + NO_WEB_CLAUSE
        + "\n"
    )


def package_unit(unit_dir: Path, tasks_root: Path) -> Path:
    """Stage one unit as a Harbor L0 task dir; returns the task dir."""
    unit_dir = Path(unit_dir)
    manifest = json.loads((unit_dir / "manifest.json").read_text(encoding="utf-8"))
    name = manifest["unit"]
    dest = Path(tasks_root) / f"{name}-L0"
    if dest.exists():
        shutil.rmtree(dest)
    (dest / "environment" / "src").mkdir(parents=True)
    (dest / "tests" / "hidden").mkdir(parents=True)
    shutil.copytree(unit_dir / "excised_tree", dest / "environment" / "src", dirs_exist_ok=True)
    (dest / "environment" / "Dockerfile").write_text(render_unsolv_dockerfile(), encoding="utf-8")
    (dest / "instruction.md").write_text(l0_instruction(unit_dir), encoding="utf-8")
    pkg = fab.DOMAINS[manifest["domain"]]["pkg"]
    hidden_rel = f"{pkg}_bb_test.go"
    shutil.copy2(unit_dir / "hidden" / hidden_rel, dest / "tests" / "hidden" / hidden_rel)
    for name_patch in ("gold.patch", "cheat.patch"):
        shutil.copy2(unit_dir / "_author" / name_patch, dest / "tests" / name_patch)
    test_sh = dest / "tests" / "test.sh"
    test_sh.write_text(render_dial_test_sh(unit_dir), encoding="utf-8")
    test_sh.chmod(0o755)
    hosts_lit = ", ".join(f'"{h}"' for h in CURSOR_HOSTS)
    (dest / "task.toml").write_text(
        f"""schema_version = "1.3"

[metadata]
category = "software-engineering"
tags = ["go", "bugfix"]

[verifier]
network_mode = "no-network"
timeout_sec = 1800.0

[agent]
network_mode = "allowlist"
allowed_hosts = [{hosts_lit}]
timeout_sec = 14400.0

[environment]
build_timeout_sec = 1800.0
network_mode = "public"
""",
        encoding="utf-8",
    )
    hidden = [f for f in (dest / "tests" / "hidden").glob("*.go")]
    hidden_text = "\n".join(f.read_text(encoding="utf-8") for f in hidden)
    import re

    test_names = re.findall(r"^func (Test\w+)", hidden_text, re.MULTILINE)
    check = instruction_self_check(
        (dest / "instruction.md").read_text(encoding="utf-8"),
        test_sh=test_sh.read_text(encoding="utf-8"),
        f2p_tests=test_names,
        changed_symbols=tuple(manifest["excision"]["S"]),
        changed_files=(f"{pkg}.go",),
        locality=2,
        packages=(pkg,),
    )
    (dest / "affordance.json").write_text(
        json.dumps(
            {
                "family": name,
                "level": 0,
                "hidden_tests": [hidden_rel],
                "instruction_self_check": check,
                "dial": manifest["dial"],
                "details": manifest["details"],
            },
            indent=2,
        )
        + "\n",
        encoding="utf-8",
    )
    write_task_validation(
        dest,
        {
            "family": name,
            "level": 0,
            "instruction_self_check": check,
            "changed_symbols": list(manifest["excision"]["S"]),
            "changed_files": [f"{pkg}.go"],
        },
    )
    return dest


def package_grid(fab_root: Path, tasks_root: Path) -> list[Path]:
    out: list[Path] = []
    for unit_dir in sorted(Path(fab_root).iterdir()):
        if not unit_dir.is_dir() or not (unit_dir / "manifest.json").is_file():
            continue
        out.append(package_unit(unit_dir, Path(tasks_root)))
    return out


def _log(path: Path | None, event: str, payload: dict[str, object]) -> None:
    if path is None:
        return
    path.parent.mkdir(parents=True, exist_ok=True)
    row = {"ts": datetime.now(UTC).isoformat(timespec="seconds"), "event": event, **payload}
    with path.open("a", encoding="utf-8") as fh:
        fh.write(json.dumps(row, default=str) + "\n")


# --------------------------------------------------------------------------- #
# trial runner (launched by a separate job; this module only plans + records)
# --------------------------------------------------------------------------- #
TRIAL_FIELDS = (
    "unit", "d", "v", "level", "model", "attempt", "reward", "wall_s",
    "tokens_in", "tokens_out", "job_dir", "details",
)
_LEVEL_RE = __import__("re").compile(r"-L(\d+)$")
_DIAL_RE = __import__("re").compile(r"-d(\d+)-v(low|high)$")


def parse_dial_from_unit(unit: str) -> tuple[int, str]:
    m = _DIAL_RE.search(unit)
    if not m:
        raise ValueError(f"unit name lacks -d<k>-v<level>: {unit}")
    return int(m.group(1)), m.group(2)


def task_unit_level(task_dir: Path) -> tuple[str, int]:
    m = _LEVEL_RE.search(task_dir.name)
    if not m:
        raise ValueError(f"task dir name lacks -L<k> suffix: {task_dir.name}")
    return task_dir.name[: m.start()], int(m.group(1))


def load_done_counts(trials_path: Path) -> dict[tuple[str, int, str], int]:
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
    limit: int | None = None,
) -> list[tuple[Path, str, int, str, int]]:
    """(task_dir, unit, level, model, n_missing) — resume-safe."""
    jobs: list[tuple[Path, str, int, str, int]] = []
    planned = 0
    for task in task_dirs:
        unit, level = task_unit_level(task)
        for model in models:
            have = done.get((unit, level, model), 0)
            missing = max(0, attempts - have)
            if limit is not None:
                missing = min(missing, max(0, limit - planned))
            if missing:
                jobs.append((task, unit, level, model, missing))
                planned += missing
            if limit is not None and planned >= limit:
                return jobs
    return jobs


def record_trials(
    job: tuple[Path, str, int, str, int],
    job_dir: Path,
    trials_path: Path,
    done: dict[tuple[str, int, str], int],
) -> int:
    """Append one row per trial, including the per-detail vector read from the
    trial's verifier log (``<trial>/verifier/details.json``)."""
    import pandas as pd

    from openswe_traces.pipeline.audit import audit_job, iter_trial_dirs

    _task_dir, unit, level, model, _n = job
    d, v = parse_dial_from_unit(unit)
    audits = audit_job(job_dir)
    if not audits:
        return 0
    have = done.get((unit, level, model), 0)
    rows = []
    for i, trial_dir in enumerate(iter_trial_dirs(job_dir), start=1):
        details = ""
        det_path = trial_dir / "verifier" / "details.json"
        if det_path.is_file():
            lines = [
                ln for ln in det_path.read_text(encoding="utf-8", errors="replace").splitlines() if ln.strip()
            ]
            details = "[" + ",".join(lines) + "]"
        a = next((x for x in audits if x.trial_dir == trial_dir), None)
        rows.append(
            {
                "unit": unit,
                "d": d,
                "v": v,
                "level": level,
                "model": model,
                "attempt": have + i,
                "reward": a.reward if a else None,
                "wall_s": round((a.wall_minutes or 0.0) * 60, 1) if a else None,
                "tokens_in": a.tokens_in if a else None,
                "tokens_out": a.tokens_out if a else None,
                "job_dir": str(trial_dir),
                "details": details,
            }
        )
    df = pd.DataFrame(rows, columns=list(TRIAL_FIELDS))
    path = Path(trials_path)
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.is_file():
        df = pd.concat([pd.read_parquet(path), df], ignore_index=True)
    df.to_parquet(path, index=False)
    done[(unit, level, model)] = have + len(rows)
    return len(rows)


def run_plan(
    jobs: list[tuple[Path, str, int, str, int]],
    *,
    agent: str,
    jobs_dir: Path,
    trials_path: Path,
    n_concurrent: int = 1,
    env: dict[str, str] | None = None,
    max_job_retries: int = 4,
    log_path: Path | None = None,
) -> dict[str, int]:
    import subprocess
    import time

    from openswe_traces.pipeline.audit import audit_job
    from openswe_traces.pipeline.safety import apply_solver_network, assert_harbor_safe

    done = load_done_counts(trials_path)
    summary = {"jobs": 0, "trials": 0, "rate_limited": 0, "failed": 0}
    env = env or {}
    rate_re = __import__("re").compile(r"\b429\b|rate.?limit|too many requests", __import__("re").IGNORECASE)
    for seq, job in enumerate(jobs, start=1):
        task_dir, unit, level, model, n_missing = job
        solver = "cursor" if "cursor" in agent.lower() else "openrouter"
        apply_solver_network(task_dir, solver)
        assert_harbor_safe(task_dir, solver=solver)
        job_name = f"{unit}-L{level}-{model.replace('/', '_')}-seq{seq}"
        job_dir = Path(jobs_dir) / job_name
        argv = [
            "harbor", "run", "--path", str(task_dir), "--agent", agent, "--model", model,
            "--n-attempts", str(n_missing), "--n-concurrent", str(max(1, n_concurrent)),
            "--max-retries", "2", "--jobs-dir", str(jobs_dir), "--job-name", job_name, "--yes",
        ]
        status = "error"
        for attempt in range(max_job_retries + 1):
            _log(log_path, "job_start", {"job": job_name, "model": model})
            proc = subprocess.run(argv, capture_output=True, text=True, check=False, env=env)
            trials = audit_job(job_dir)
            blob = (proc.stdout or "") + "\n" + (proc.stderr or "")
            if rate_re.search(blob):
                status = "rate_limit"
            elif proc.returncode != 0:
                status = "error"
            elif not trials:
                status = "no_trials"
            else:
                status = "ok"
            _log(log_path, "job_done", {"job": job_name, "status": status, "rc": proc.returncode, "trials": len(trials)})
            if status == "ok" or attempt == max_job_retries:
                break
            time.sleep(min(60 * (2**attempt), 1200))
        n = record_trials(job, job_dir, trials_path, done=done)
        summary["jobs"] += 1
        summary["trials"] += n
        if status == "rate_limit":
            summary["rate_limited"] += 1
        elif status != "ok":
            summary["failed"] += 1
    return summary


def main_cli(argv: list[str] | None = None) -> int:
    import argparse
    import os

    p = argparse.ArgumentParser(description="Detail-dial Harbor runner (resume-safe).")
    p.add_argument("--task-dir", action="append", type=Path, default=[])
    p.add_argument("--tasks-glob", action="append", default=[])
    p.add_argument("--task-list", type=Path, default=None)
    p.add_argument("--agent", default="mini-swe-agent")
    p.add_argument("--model", action="append", required=True)
    p.add_argument("--attempts", type=int, default=1)
    p.add_argument("--jobs-dir", type=Path, default=Path("experiments/detail_dial/jobs"))
    p.add_argument("--trials", type=Path, default=Path("experiments/detail_dial/trials.parquet"))
    p.add_argument("--n-concurrent", type=int, default=1)
    p.add_argument("--limit", type=int, default=None)
    p.add_argument("--dry-run", action="store_true")
    p.add_argument("--log", type=Path, default=Path("outputs/closure_R.log"))
    args = p.parse_args(argv)

    task_dirs: list[Path] = list(args.task_dir)
    for g in args.tasks_glob:
        task_dirs.extend(sorted(Path().glob(g)))
    if args.task_list:
        task_dirs.extend(Path(ln.strip()) for ln in args.task_list.read_text().splitlines() if ln.strip())
    task_dirs = [d for d in dict.fromkeys(task_dirs) if (d / "task.toml").is_file()]

    done = load_done_counts(args.trials)
    jobs = plan_jobs(task_dirs, list(args.model), args.attempts, done, limit=args.limit)
    if args.dry_run:
        for task_dir, unit, level, model, n_missing in jobs:
            print(f"{unit}-L{level}\t{model}\tn={n_missing}\t{task_dir}")
        print(f"{len(jobs)} jobs, {sum(j[4] for j in jobs)} attempts planned")
        return 0
    env_files = [Path(".env"), Path("/home/evan/Documents/eval_tasks/.env")]
    env: dict[str, str] = dict(os.environ)
    for path in env_files:
        if not path.is_file():
            continue
        for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            k, val = line.split("=", 1)
            env[k.strip()] = val.strip().strip('"').strip("'")
    summary = run_plan(
        jobs,
        agent=args.agent,
        jobs_dir=args.jobs_dir,
        trials_path=args.trials,
        n_concurrent=args.n_concurrent,
        env=env,
        log_path=args.log,
    )
    print(json.dumps(summary))
    return 0

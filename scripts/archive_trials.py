"""Compact, durable record of every Harbor trial: survives docker prune AND job-dir cleanup.

One row per trial in experiments/dose_response/archive/trials.jsonl (append-only, resume-safe),
plus the agent patch, verifier stdout and trajectory copied into archive/<job>/<trial>/.
Run from cron/loop; re-running is free for trials already archived.
"""
from __future__ import annotations
import json, shutil, sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
JOBS = ROOT / "experiments/dose_response/jobs"
ARCH = ROOT / "experiments/dose_response/archive"
INDEX = ARCH / "trials.jsonl"
KEEP = ("result.json", "verifier/test-stdout.txt", "trial.log")
PATCH_NAMES = ("agent.patch", "model.patch", "diff.patch")


def archived() -> set[str]:
    if not INDEX.exists():
        return set()
    out = set()
    for line in INDEX.read_text().splitlines():
        try:
            out.add(json.loads(line)["trial"])
        except Exception:
            pass
    return out


def one(trial_dir: Path, job: str) -> dict | None:
    res = trial_dir / "result.json"
    if not res.is_file():
        return None
    try:
        d = json.loads(res.read_text())
    except Exception:
        return None
    rewards = (d.get("verifier_result") or {}).get("rewards") or {}
    dest = ARCH / job / trial_dir.name
    dest.mkdir(parents=True, exist_ok=True)
    for rel in KEEP:
        src = trial_dir / rel
        if src.is_file():
            tgt = dest / Path(rel).name
            if not tgt.exists():
                shutil.copy2(src, tgt)
    for p in trial_dir.rglob("*"):
        if p.is_file() and (p.name in PATCH_NAMES or p.name.endswith(".trajectory.json") or p.name == "trajectory.json"):
            tgt = dest / p.name
            if not tgt.exists():
                shutil.copy2(p, tgt)
    return {
        "trial": f"{job}/{trial_dir.name}",
        "job": job,
        "task": trial_dir.name.split("__")[0],
        "reward": rewards.get("reward"),
        "started_at": d.get("started_at"),
        "finished_at": d.get("finished_at"),
        "exception": (d.get("exception_info") or {}).get("exception_type"),
        "archived_files": sorted(p.name for p in dest.iterdir() if p.is_file()),
    }


def main() -> int:
    ARCH.mkdir(parents=True, exist_ok=True)
    done = archived()
    n = 0
    with INDEX.open("a") as fh:
        for job_dir in sorted(p for p in JOBS.iterdir() if p.is_dir()):
            for trial_dir in sorted(p for p in job_dir.iterdir() if p.is_dir()):
                key = f"{job_dir.name}/{trial_dir.name}"
                if key in done:
                    continue
                row = one(trial_dir, job_dir.name)
                if row:
                    fh.write(json.dumps(row) + "\n")
                    n += 1
    print(f"archived {n} new trials; index {INDEX} has {len(done) + n} rows")
    return 0


if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env python3
"""B5 seed re-run: re-verify one passing trial per unit with HIDDEN_SEED=20260920.

A solution that memorised the original seed's examples passes the shipped suite and fails
this one. The check needs docker and had never been run until 2026-09-20.

Usage: seed_rerun.py <out.jsonl> [job-dir ...]
"""
import json
import subprocess
import sys
from pathlib import Path

out_path = Path(sys.argv[1])
jobs = [Path(a) for a in sys.argv[2:]] or sorted(
    p for p in Path("experiments/dose_response/jobs").iterdir() if p.is_dir()
)
done = set()
if out_path.exists():
    for line in out_path.read_text().splitlines():
        try:
            done.add(json.loads(line)["unit"])
        except Exception:
            pass

with out_path.open("a") as fh:
    for job in jobs:
        for trial in sorted(p for p in job.iterdir() if (p / "result.json").is_file()):
            try:
                data = json.loads((trial / "result.json").read_text())
            except Exception:
                continue
            rew = (data.get("verifier_result") or {}).get("rewards", {}).get("reward")
            if rew != 1.0:
                continue
            unit = data.get("task_name") or trial.name.split("__")[0]
            if unit in done:
                continue
            tid = data.get("task_id") or {}
            task_dir = tid.get("path") if isinstance(tid, dict) else None
            if not task_dir or not Path(task_dir).is_dir():
                continue
            done.add(unit)
            cmd = [
                "uv", "run", "python", "-m", "openswe_traces.pipeline_ext.hack_audit",
                "--trial", str(trial), "--task-dir", str(task_dir), "--docker",
            ]
            try:
                r = subprocess.run(cmd, capture_output=True, text=True, timeout=2400)
                v = json.loads(r.stdout)
            except Exception as exc:  # noqa: BLE001
                v = {"passed": None, "hard_fails": [f"audit error: {exc}"], "evidence": {}}
            dk = v.get("evidence", {}).get("docker") or {}
            rec = {
                "unit": unit, "job": job.name, "trial": trial.name,
                "seed_passed": dk.get("passed"), "collateral_fail": dk.get("collateral_fail"),
                "hard_fails": v.get("hard_fails", []),
                "stdout_tail": (v.get("evidence", {}).get("docker_stdout") or "")[-300:],
            }
            fh.write(json.dumps(rec) + "\n")
            fh.flush()
            # An empty hard_fails with a failed run means the audit could not build the
            # package -- inconclusive, not a seed-sensitivity failure.
            if dk.get("passed"):
                mark = "OK "
            elif rec["hard_fails"]:
                mark = "SEED-FAIL"
            else:
                mark = "INCONCLUSIVE (build)"
            print(f"{mark} {unit}", flush=True)

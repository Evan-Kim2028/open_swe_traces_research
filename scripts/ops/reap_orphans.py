#!/usr/bin/env python3
"""Containers whose harbor run is gone: they burn quota and can never produce a verdict.

harbor is what writes result.json. Kill a harbor run and docker leaves its containers running
-- the agent inside keeps working, keeps calling the model, keeps spending the account's
concurrency -- and when it finishes there is nobody to collect the outcome. The trial is
entirely wasted, and worse, it is INVISIBLE: slots.py counts harbor processes and their
--n-concurrent, so an orphan does not appear in occupancy at all.

That combination produced the confusing part of an evening: devin read 4/4 with NINE containers
up, five of them orphans from harbor runs killed to "get back under cap". Killing the parent had
freed the number without freeing the slot.

A container is owned if its name matches a trial directory under a LIVE harbor run's job. The
match is on the full stem (unit-lN__hash), lowercased, because docker lowercases compose project
names while harbor's directories keep their original case.

    reap_orphans.py            # report
    reap_orphans.py --apply    # kill them
"""
from __future__ import annotations

import glob
import os
import subprocess
import sys

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))


def live_jobs() -> set[str]:
    jobs = set()
    for e in os.listdir("/proc"):
        if not e.isdigit():
            continue
        try:
            argv = [a for a in open(f"/proc/{e}/cmdline", "rb").read()
                    .decode(errors="replace").split("\0") if a]
        except OSError:
            continue
        if not (any("harbor" in x for x in argv[:2]) and "--job-name" in argv):
            continue
        for i, x in enumerate(argv[:-1]):
            if x == "--job-name":
                jobs.add(argv[i + 1])
    return jobs


def owned_stems(jobs: set[str]) -> set[str]:
    out = set()
    for j in jobs:
        for d in glob.glob(os.path.join(REPO, "experiments/dose_response/jobs", j, "*/")):
            out.add(os.path.basename(d.rstrip("/")).lower())
    return out


def main() -> int:
    apply = "--apply" in sys.argv
    jobs = live_jobs()
    owned = owned_stems(jobs)
    names = subprocess.run(["docker", "ps", "--format", "{{.Names}}"],
                           capture_output=True, text=True).stdout.split()
    mains = [c for c in names if c.endswith("__env-main-1")]
    orphans = [c[: -len("__env-main-1")] for c in mains
               if c[: -len("__env-main-1")].lower() not in owned]
    print(f"live harbor job(s): {len(jobs)}; containers: {len(mains)}; "
          f"orphaned: {len(orphans)}")
    for s in orphans:
        if apply:
            for suf in ("__env-main-1", "__env-harbor-docker-egress-control-sidecar-1"):
                subprocess.run(["docker", "kill", s + suf], capture_output=True)
            print(f"  reaped {s}")
        else:
            print(f"  WOULD REAP {s}")
    # A live harbor with zero owned containers is normal while it builds, so this is not an
    # error -- it only means the report cannot attribute anything yet.
    if jobs and not owned:
        print("  note: live harbor has no trial dirs yet (still building); nothing attributable")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

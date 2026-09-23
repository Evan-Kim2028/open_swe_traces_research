#!/usr/bin/env python3
"""Audit staged contracts BEFORE a sweep needs them, so no sweep audits while holding slots.

The problem this removes
------------------------
``sweep_seq`` gates every L2-or-higher unit on an audit row, and does it after the cohort
lock is taken. The lock reserves the cohort's full concurrency, so a sweep spends its first
quarter-hour holding eight of Composer's twelve slots while running zero containers:

    sweep_b5kops_L2   conc=8  cursor  alive   ← 8 slots held, 0 containers, 7 minutes in
    !! these L2 units have no audit row -- auditing before spending a trial: 11 units

Measured at the moment this was written: 5 containers against a cap of 12, load 7 on 32
cores, 317M of Composer budget unspent. Roughly 60% of Composer capacity idle, entirely
because the gate runs at the wrong time.

The gate itself is right and stays: ~6k tokens and ~40s per unit against the 2.08M trial it
prevents. Only the timing moves. Running it from the supervisor means the rows exist before
a sweep ever launches, and ``contract_gap_read`` skips units already in the file, so a sweep
that finds its rows present goes straight to trialling.

Why not the alternatives:

- *Release the reservation during the audit.* Re-claiming it afterwards races: three sweeps
  could each finish auditing into slots another had already taken, and over-subscription is
  the failure the reservation exists to prevent.
- *Age the reservation out.* Same race, on a timer.

Ordering: units the guard says are runnable come first, since those are what a sweep will
reach for next. Bounded per run so a tick cannot run away.

Usage::

    preaudit.py            # audit up to BATCH runnable un-audited units
    preaudit.py --batch 24
    preaudit.py --dry-run
"""

from __future__ import annotations
from openswe_traces.paths import REPO as _REPO

import argparse
import glob
import json
import os
import pathlib
import subprocess
import sys

REPO = _REPO
GAPS = REPO / "experiments" / "dose_response" / "audit" / "gap_read.jsonl"
BATCH = 12          # ask_many runs 3 workers at ~40s
LOCK = REPO / "outputs" / "supervisor" / "preaudit.lock"


def take_lock() -> bool:
    """One preaudit at a time.

    Two instances pick overlapping units and pay for the same audit twice, and worse,
    they contend: the first run took over ten minutes for a batch sized at three, and
    the supervisor tick — which runs orchestrate AFTER this — was stalled the whole time.
    A fix that blocks launches is worse than the stall it was removing.
    """
    try:
        if LOCK.is_file():
            pid = int(LOCK.read_text().split()[0])
            os.kill(pid, 0)          # raises if the owner is gone
            return False
    except (ValueError, IndexError, OSError):
        pass                          # stale or unreadable: it is ours to take
    LOCK.parent.mkdir(parents=True, exist_ok=True)
    LOCK.write_text(f"{os.getpid()}\n")
    return True


def audited() -> set[str]:
    out = set()
    if GAPS.is_file():
        for line in GAPS.read_text(errors="replace").splitlines():
            try:
                row = json.loads(line)
            except Exception:
                continue
            # Defensive against rows written before contract_gap_read stopped recording
            # failures: an `__ERROR__` answer parsed as "clean", so a crashed audit looked
            # like a perfect contract and was never retried.
            if str(row.get("answer", "")).startswith("__ERROR__"):
                continue
            out.add(row["unit"])
    return out


def candidates(limit: int) -> list[pathlib.Path]:
    """Staged units that a sweep would gate on and that have no row yet."""
    from openswe_traces.ladder import guard as trial_guard
    from openswe_traces.ladder import ledger as TL

    per = TL.ledger()
    have = audited()
    runnable, rest = [], []
    for d in sorted(glob.glob(str(REPO / "experiments/dose_response/sweep_*/*/"))):
        u = pathlib.Path(d.rstrip("/"))
        name = u.name
        if "-L" not in name:
            continue
        rung = name.rsplit("-L", 1)[1][:1]
        # L0 and L1 carry no contract; TOO-EASY is the gate there, not the audit.
        if rung in ("0", "1") or not rung.isdigit():
            continue
        if name in have:
            continue
        if not (u / "instruction.md").is_file() or not (u / "tests" / "hidden").is_dir():
            continue
        # An escalation rung inherits its L2 twin's row inside sweep_seq; auditing it here
        # would spend tokens on a question already answered.
        if rung != "2" and f"{name.rsplit('-L', 1)[0]}-L2" in have:
            continue
        try:
            ok, _ = trial_guard.decide(name, per)
        except Exception:
            ok = False
        (runnable if ok else rest).append(u)
    return (runnable + rest)[:limit]


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--batch", type=int, default=BATCH)
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()

    if args.dry_run:
        units = candidates(args.batch)
        print(f"preaudit: {len(units)} unit(s) — "
              f"{', '.join(u.name for u in units[:6])}{' ...' if len(units) > 6 else ''}"
              if units else "preaudit: nothing to audit")
        return 0

    if not take_lock():
        print("preaudit: another instance is running — nothing to do")
        return 0
    # try/finally, not an unlink at the end: the "nothing to audit" path returned early and
    # left the lock behind, and a timeout or a crash did the same. A stale lock happens to
    # self-heal, because take_lock tests the owner with kill(pid, 0) — but relying on a
    # dead pid never being reused is not a guarantee, it is a coincidence.
    try:
        units = candidates(args.batch)
        if not units:
            print("preaudit: every staged contract a sweep would gate on has a row")
            return 0
        print(f"preaudit: {len(units)} unit(s) — {', '.join(u.name for u in units[:6])}"
              f"{' ...' if len(units) > 6 else ''}")
        GAPS.parent.mkdir(parents=True, exist_ok=True)
        try:
            r = subprocess.run(
                ["uv", "run", "python", "scripts/ops/contract_gap_read.py", str(GAPS),
                 *[str(u) for u in units]],
                cwd=REPO, capture_output=True, text=True, timeout=1800)
            sys.stdout.write(r.stdout[-2000:])
        except subprocess.TimeoutExpired:
            print("preaudit: gap-read timed out; rows written so far are kept, "
                  "the rest retry next tick")
        now = audited()
        print(f"preaudit: {sum(1 for u in units if u.name in now)} of "
              f"{len(units)} now audited")
        return 0
    finally:
        LOCK.unlink(missing_ok=True)


def cli():
    raise SystemExit(main())


if __name__ == "__main__":
    cli()

#!/usr/bin/env python3
"""Remove a trial from the ledger when the INFRASTRUCTURE failed, not the agent.

A reward of 0 is supposed to mean the agent could not fix the bug at that rung. It is the
single fact the whole ladder rests on: "fails at L4, flips at L5" is what makes a
certificate say the unit NEEDS L5.

Sometimes the 0 means something else entirely. Composer issued `grep -r pattern /`, the
command walked the whole container filesystem at 99% cpu, and the agent sat blocked on
that one tool call for 1h37m of a ~100 minute budget before returning empty-handed. It
recorded a clean reward=0. Nothing in the result distinguishes that from an agent that
tried honestly and failed, and `httpresp` had its certificate SETTLED by exactly this
trial — rung_established went true on the strength of a run that was stuck for 97% of
its life.

Keeping it is not the conservative choice. A spurious failure at rung k pushes the unit
up the ladder and overstates its difficulty, which is the specific quantity this dataset
exists to measure. Discarding it costs one re-run at a couple of million tokens.

Quarantine rather than delete: the directory moves out of the jobs tree so the ledger's
glob no longer sees it, a REASON file records why and when, and --restore puts it back.
Nothing is destroyed, and the decision stays auditable instead of becoming a gap.

    quarantine_trial.py <trial-dir> --reason "..."     # move it aside
    quarantine_trial.py --list                          # what is quarantined
    quarantine_trial.py --restore <name>                # put it back
"""

from __future__ import annotations

import argparse
import os
import pathlib
import shutil
import time

REPO = pathlib.Path(__file__).resolve().parents[2]
QDIR = REPO / "outputs" / "quarantine"


def quarantine(trial_dir: str, reason: str) -> int:
    src = pathlib.Path(trial_dir).resolve()
    if not src.is_dir():
        print(f"not a directory: {src}")
        return 1
    if "jobs" not in src.parts:
        print(f"refusing: {src} is not under a jobs/ tree")
        return 1
    QDIR.mkdir(parents=True, exist_ok=True)
    stamp = time.strftime("%Y%m%dT%H%M%S")
    dest = QDIR / f"{stamp}__{src.parent.name}__{src.name}"
    shutil.move(str(src), str(dest))
    (dest / "QUARANTINE_REASON.txt").write_text(
        f"quarantined: {time.strftime('%Y-%m-%dT%H:%M:%S')}\n"
        f"original path: {src}\n"
        f"reason: {reason}\n\n"
        f"Restore with: quarantine_trial.py --restore {dest.name}\n")
    print(f"quarantined {src.name} -> {dest}")
    return 0


def restore(name: str) -> int:
    d = QDIR / name
    if not d.is_dir():
        print(f"no such quarantine entry: {name}")
        return 1
    txt = (d / "QUARANTINE_REASON.txt").read_text()
    orig = next(l.split(": ", 1)[1].strip() for l in txt.splitlines()
                if l.startswith("original path:"))
    os.makedirs(os.path.dirname(orig), exist_ok=True)
    (d / "QUARANTINE_REASON.txt").unlink()
    shutil.move(str(d), orig)
    print(f"restored -> {orig}")
    return 0


def listing() -> int:
    if not QDIR.is_dir():
        print("nothing quarantined")
        return 0
    rows = sorted(QDIR.iterdir())
    if not rows:
        print("nothing quarantined")
        return 0
    for d in rows:
        f = d / "QUARANTINE_REASON.txt"
        why = ""
        if f.exists():
            why = next((l.split(": ", 1)[1].strip() for l in f.read_text().splitlines()
                        if l.startswith("reason:")), "")
        print(f"  {d.name}\n      {why}")
    return 0


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("trial_dir", nargs="?")
    ap.add_argument("--reason", default="")
    ap.add_argument("--list", action="store_true")
    ap.add_argument("--restore")
    a = ap.parse_args()
    if a.list:
        return listing()
    if a.restore:
        return restore(a.restore)
    if not a.trial_dir or not a.reason:
        print("need <trial-dir> and --reason (or --list / --restore)")
        return 2
    return quarantine(a.trial_dir, a.reason)


if __name__ == "__main__":
    raise SystemExit(main())

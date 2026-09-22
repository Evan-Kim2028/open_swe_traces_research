#!/usr/bin/env python3
"""The status update, as one command.

Two sections, because they answer different questions and were being hand-assembled
separately every time:

  HEALTH   what is running and whether anything is wrong
  QUEUE    what is left to run, and crucially WHO is allowed to run it — since the
           no-contamination rule, "runnable" is not one number but three

The queue split matters more than a total. 62 units being cursor-only is not a backlog
Devin can help with; it is the measured cost of certifying on the solver that failed the
unit. And the large "condemned" bucket is finished business, not work — reporting a
single "trials left" figure hid both facts.

Usage::

    status.py            # both sections
    status.py --queue    # queue only
    status.py --health   # health only
"""

from __future__ import annotations

import collections
import glob
import os
import subprocess
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))


def sh(c, t=120):
    try:
        return subprocess.run(c, shell=True, capture_output=True, text=True,
                              timeout=t, cwd=REPO).stdout.strip()
    except Exception:
        return ""


def health():
    import slots
    import trial_ledger as TL
    c = TL.certificates()
    esc = sum(1 for v in c.values() if v.get("escalated"))
    jump = sum(1 for v in c.values() if not v.get("rung_established", True))
    cross = sum(1 for v in c.values() if v["kind"] == "cross")
    d = slots.occupancy("devin")
    cur = sum(x["conc"] for x in slots.trials("cursor"))
    cont = int(sh("docker ps --format '{{.Names}}' | grep -c env-main-1") or 0)
    load = open("/proc/loadavg").read().split()[1]
    free = sh("df -BG --output=avail / | tail -1").strip()
    pid = slots.supervisor_pid()
    stale = ""
    if pid:
        try:
            if "(deleted)" in os.readlink(f"/proc/{pid}/fd/255"):
                stale = "  STALE INODE — edits inert, restart it"
        except OSError:
            pass
    print(f"  certificates {len(c)}   above-L2 {esc}   by-jump {jump}   cross-solver {cross}")
    print(f"  devin {d['total']}/{slots.CAP}   composer {cur}/12   containers {cont}")
    print(f"  supervisor pid {pid or 'DOWN'}{stale}   load {load} on {os.cpu_count()}"
          f"   disk {free} free")
    print("  " + (sh("uv run python scripts/ops/stability_gate.py --brief", 300) or "gate: ?"))
    ov = os.path.join(REPO, "outputs/supervisor/devin_cap_override")
    if os.path.exists(ov):
        print(f"  DEVIN CAP DROPPED — {open(ov).read().splitlines()[1][:70]}")


def queue():
    import trial_guard as TG
    import trial_ledger as TL
    import solver_match as SM
    per = TL.ledger()
    bys = TL.ledger_by_solver()
    run = collections.Counter()
    blocked = collections.Counter()
    for d in sorted(glob.glob(os.path.join(REPO, "experiments/dose_response/sweep_*/*/"))):
        n = os.path.basename(d.rstrip("/"))
        if "-L" not in n:
            continue
        base, suf = n.rsplit("-L", 1)
        rung = suf[:1]
        if not rung.isdigit() or per.get(base, {}).get(rung):
            continue
        # a unit with no hidden suite can never be trialled; it needs a verifier
        if not glob.glob(d + "tests/hidden/**/*", recursive=True):
            continue
        try:
            ok, why = TG.decide(n, per)
        except Exception:
            ok, why = False, "guard error"
        if not ok:
            blocked[why.split(":")[0][:38]] += 1
            continue
        who = [a for a in ("devin", "cursor") if SM.decide(n, a, bys)[0]]
        run["+".join(who) or "nobody"] += 1
    print("  runnable now, by solver the rule admits:")
    for k, v in run.most_common():
        note = {"cursor": "composer failed these low",
                "devin": "devin failed these low",
                "devin+cursor": "no prior failure, or both failed"}.get(k, "")
        print(f"    {v:4d}  {k:14s} {note}")
    print(f"    ---- {sum(run.values())} total")
    print("  not runnable (most never will be):")
    for k, v in blocked.most_common(6):
        print(f"    {v:4d}  {k}")
    print(f"    ---- {sum(blocked.values())} total")


def main() -> int:
    want = set(a.lstrip("-") for a in sys.argv[1:]) or {"health", "queue"}
    if "health" in want:
        print("HEALTH"); health()
    if "queue" in want:
        print("QUEUE"); queue()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

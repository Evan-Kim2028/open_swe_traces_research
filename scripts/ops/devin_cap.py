#!/usr/bin/env python3
"""Hold Devin at the cap. Enforce it; do not merely report it.

Devin drifted over 4 repeatedly because two independent things start Devin work and
neither sees the other: dq2 launchers start SESSIONS (counting sessions only), and
orchestrate starts in-container TRIALS. Each stayed under the cap by its own reckoning
while the total ran to 5, 6, 7.

Rules, in this order:
  1. Never kill a session. A killed session restarts from scratch and loses its work.
  2. Trim TRIALS instead, smallest concurrency first, so the least work is lost.
  3. Killing a harbor child is useless - its sweep_seq parent relaunches it within
     seconds. Kill the sweep chain (sh wrapper -> sweep_seq -> harbor) and drop its lock.

Usage: devin_cap.py [--apply] [--cap N]
"""
import os, re, subprocess, sys, glob

CAP = int(sys.argv[sys.argv.index("--cap") + 1]) if "--cap" in sys.argv else 4
APPLY = "--apply" in sys.argv
R = "/home/evan/Documents/open_swe_traces_research"
LOCKDIR = os.path.join(R, "outputs/supervisor/sweep_locks")


def sh(c):
    return subprocess.run(c, shell=True, capture_output=True, text=True).stdout


sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from slots import occupancy   # one definition of "who holds a slot" - see slots.py


def cmdline(pid):
    try:
        with open(f"/proc/{pid}/cmdline", "rb") as fh:
            return fh.read().decode(errors="replace").split("\0")
    except OSError:
        return []


def flag(parts, name):
    # exact match only: a substring test on "--agent" also matches
    # "--agent-timeout-multiplier" and reports the multiplier as the agent name
    for i, p in enumerate(parts[:-1]):
        if p == name:
            return parts[i + 1]
    return None


def devin_trials():
    out = []
    for pid in sh("pgrep -x harbor").split():
        parts = cmdline(pid)
        if flag(parts, "--agent") != "devin":
            continue
        out.append({"pid": int(pid),
                    "job": flag(parts, "--job-name") or "?",
                    "conc": int(flag(parts, "--n-concurrent") or 1)})
    return out


def sessions():
    return sorted(set(re.findall(r"closure_\w+", sh("pgrep -af '[d]evin --model'"))))


def sweep_chain(harbor_pid):
    """harbor -> sweep_seq -> sh wrapper. Returns pids to kill and the cohort name."""
    chain, cohort, pid = [harbor_pid], None, harbor_pid
    for _ in range(3):
        ppid = sh(f"ps -o ppid= -p {pid}").strip()
        if not ppid.isdigit() or ppid == "1":
            break
        parts = cmdline(ppid)
        joined = " ".join(parts)
        if "sweep_seq" not in joined:
            break
        chain.append(int(ppid))
        # The command line is `bash /path/sweep_seq.<hash>.sh sweep_bbolt_4 1 6`, so a
        # bare (sweep_\w+) matches the SCRIPT NAME first and returns "sweep_seq" - which
        # then deletes the wrong lock (or none) and leaves the real cohort locked by a
        # dead pid. Take the first sweep_ token that is not the script itself.
        for tok in re.findall(r"(sweep_[A-Za-z0-9_]+)", joined):
            if tok.startswith("sweep_seq"):
                continue
            cohort = cohort or tok
            break
        pid = ppid
    return chain, cohort


def main():
    # Counting was the bug: five files each had their own version and they disagreed,
    # which is how the cap reached 7 while every one of them reported "under 4".
    o = occupancy("devin")
    sess, trials = o["sessions"], o["trials"]
    total = o["total"]
    print(f"devin {total}/{CAP} = {len(sess)} session(s) + "
          f"{sum(t['conc'] for t in trials)} trial(s)")
    for s in sess:
        print(f"    session {s}")
    for t in trials:
        print(f"    trial   {t['job']} conc={t['conc']}")
    if total <= CAP:
        print("at or under cap")
        return 0
    if not trials:
        # Every slot is a session; the cap cannot be enforced without killing one, and
        # that is never worth it. Say so plainly rather than acting.
        print(f"OVER CAP by {total - CAP} but all slots are sessions — "
              f"not killing a session; it will drain")
        return 1
    # trim smallest-concurrency trials first
    for t in sorted(trials, key=lambda x: x["conc"]):
        if total <= CAP:
            break
        chain, cohort = sweep_chain(t["pid"])
        print(f"  trimming {t['job']} (conc={t['conc']}) chain={chain} cohort={cohort}")
        if APPLY:
            for p in chain:
                subprocess.run(f"kill -9 {p}", shell=True, capture_output=True)
            if cohort:
                for f in glob.glob(os.path.join(LOCKDIR, cohort + "*")):
                    os.remove(f)
        total -= t["conc"]
    print(f"devin now targeting {total}/{CAP}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

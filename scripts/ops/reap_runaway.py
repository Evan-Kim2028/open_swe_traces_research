#!/usr/bin/env python3
"""Kill a runaway COMMAND inside a trial container, without killing the trial.

reap_wedged.sh catches the opposite failure: a container doing nothing. Its rule is
"0% cpu AND no log bytes AND no network = dead", written that way after an
age-based reaper killed nine live trials, four of them confirmed L2s. The rule is
right, and it is structurally blind to this:

    grep -r "response defines headers but result is empty" /   1:37:13 CPU
    grep -r "rewriteAPIGroup" /                                1:16:12 CPU

Composer issued a recursive grep from the filesystem ROOT instead of /app. It walks the
module cache, /proc and /sys at 99% CPU and effectively never returns. Two trials sat on
one tool call for over an hour each, holding a Composer slot apiece and contributing a
full hour of zero verdicts. To reap_wedged that container looks maximally healthy — CPU
pegged is precisely its definition of working. No age rule would catch it either without
also killing the genuinely slow trials the nine-trial incident taught us to protect.

What separates a runaway from real work is not how much CPU it uses but that it uses ALL
of it, continuously, with nothing to show. A process whose consumed CPU time is ~equal to
its wall-clock lifetime has never once waited on I/O, a socket, or a child. Real build and
test steps block constantly. So the test is the RATIO, plus a floor so that a legitimately
long compile is not a candidate.

The response is surgical on purpose: SIGTERM the offending process, not the container. The
agent's tool call returns non-zero, it reads that as a failed command and carries on with
the trial intact. Killing the container would throw away the hour of real work that came
before the bad command.

    reap_runaway.py                 # dry run: report only
    reap_runaway.py --apply         # kill what it finds
    reap_runaway.py --min-cpu 900   # floor in seconds (default 1800)
"""

from __future__ import annotations

import argparse
import re
import subprocess

# Never touch these: they are the trial's own scaffolding, not agent-issued commands.
PROTECTED = re.compile(r"sleep infinity|cursor-agent|harbor|/bin/sh -c|dump_bash_state|"
                       r"bash -O extglob|tee /logs|node |npm |/init|ps -eo")

RATIO = 0.80      # cpu_seconds / elapsed_seconds; >0.8 means it has never blocked

# Thirty minutes of CONTINUOUS cpu before anything is a candidate. 600s was the first
# draft and it is too eager: a single-threaded compile over a large tree (kops stages
# ~18k files) can hold high duty for ten or twenty minutes and is doing real work. The
# pipeline also bounds its own heavy step — every contract runs the suite as
# `go test -count=1 -timeout 15m`, so a legitimate test phase cannot reach 900s, let
# alone 1800. The two real runaways were 4595s and 5855s, so the margin costs nothing
# in detection and buys immunity from killing a slow build.
MIN_CPU = 1800


def sh(cmd):
    try:
        return subprocess.run(cmd, shell=True, capture_output=True, text=True,
                              timeout=120).stdout
    except subprocess.SubprocessError:
        return ""


def hhmmss(s):
    """'01:37:35' or '1-02:03:04' -> seconds."""
    days = 0
    if "-" in s:
        d, s = s.split("-", 1)
        days = int(d)
    parts = [int(x) for x in s.split(":")]
    while len(parts) < 3:
        parts.insert(0, 0)
    return days * 86400 + parts[0] * 3600 + parts[1] * 60 + parts[2]


def containers():
    return [c for c in sh("docker ps --format '{{.Names}}'").split() if "env-main" in c]


def scan(min_cpu):
    out = []
    for c in containers():
        raw = sh(f"docker exec {c} ps -eo pid,etime,time,args 2>/dev/null")
        for line in raw.splitlines()[1:]:
            parts = line.split(None, 3)
            if len(parts) < 4:
                continue
            pid, etime, ctime, args = parts
            if not pid.isdigit() or PROTECTED.search(args):
                continue
            try:
                cpu, el = hhmmss(ctime), hhmmss(etime)
            except ValueError:
                continue
            if cpu < min_cpu or el <= 0:
                continue
            if cpu / el >= RATIO:
                out.append((c, pid, cpu, el, args.strip()))
    return out


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--apply", action="store_true")
    ap.add_argument("--min-cpu", type=int, default=MIN_CPU)
    a = ap.parse_args()
    found = scan(a.min_cpu)
    if not found:
        print(f"runaway: none (no in-container process over {a.min_cpu}s CPU at "
              f">={RATIO:.0%} duty)")
        return 0
    for c, pid, cpu, el, args in found:
        short = args[:90]
        if a.apply:
            sh(f"docker exec {c} kill {pid}")
            print(f"  KILLED  {c.split('__')[0]} pid {pid}  {cpu}s cpu / {el}s alive "
                  f"({cpu/el:.0%} duty)  {short}")
        else:
            print(f"  WOULD KILL  {c.split('__')[0]} pid {pid}  {cpu}s cpu / {el}s alive "
                  f"({cpu/el:.0%} duty)  {short}")
    if not a.apply:
        print("\n  dry run — pass --apply to kill. The TRIAL survives; only the "
              "offending command dies.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

"""Who is occupying a solver slot right now.

Counted five different ways in five files, and they disagreed. dq2's live() saw only
`devin --model` processes, so with 2 sessions visible it started more on top of 3
in-container trials: 7 against a cap of 4. pipeline_health branched on sessions before
adding trials and printed "Devin only 1/4" while the cap was saturated. The dashboard
showed 2/4 for the same reason.

A trial runs the CLI inside a container. It is invisible to pgrep, spends the same
account quota, and occupies the same slot as a session. One definition, here.
"""
from __future__ import annotations
import os, re, subprocess

CAP = 4


def _sh(c):
    return subprocess.run(c, shell=True, capture_output=True, text=True).stdout


def sessions(agent="devin"):
    """Named agent sessions, e.g. ['AU5goa', 'VFbatch5clientgo']."""
    if agent != "devin":
        return []
    out = _sh("pgrep -af '[d]evin --model'")
    return sorted({m.replace("closure_", "") for m in re.findall(r"closure_\w+", out)})


def trials(agent="devin"):
    """In-container trials, as a list of {job, conc, pid}. Deduplicated by job name:
    one harbor run with --n-concurrent 3 is three slots, not three runs."""
    out = []
    seen = set()
    for pid in _sh("pgrep -x harbor").split():
        try:
            with open(f"/proc/{pid}/cmdline", "rb") as fh:
                parts = fh.read().decode(errors="replace").split("\0")
        except OSError:
            continue

        def flag(name):
            # exact match: a substring test on "--agent" also matches
            # "--agent-timeout-multiplier" and returns the multiplier as the agent name
            for i, p in enumerate(parts[:-1]):
                if p == name:
                    return parts[i + 1]
            return None

        a = flag("--agent")
        if agent == "devin" and a != "devin":
            continue
        if agent == "cursor" and a not in ("cursor-cli", "cursor-agent"):
            continue
        job = flag("--job-name") or "?"
        if job in seen:
            continue
        seen.add(job)
        out.append({"pid": int(pid), "job": job,
                    "conc": int(flag("--n-concurrent") or 1)})
    return out


def occupancy(agent="devin"):
    s, t = sessions(agent), trials(agent)
    return {"sessions": s, "trials": t,
            "total": len(s) + sum(x["conc"] for x in t),
            "cap": CAP}


def containers():
    return int(_sh("docker ps --format '{{.Names}}' | grep -c env-main").strip() or 0)


def summary(agent="devin"):
    o = occupancy(agent)
    return (f"{agent} {o['total']}/{o['cap']} = {len(o['sessions'])} session(s) + "
            f"{sum(x['conc'] for x in o['trials'])} trial(s)")


if __name__ == "__main__":
    o = occupancy()
    print(summary())
    for s in o["sessions"]:
        print(f"    session {s}")
    for t in o["trials"]:
        print(f"    trial   {t['job']} conc={t['conc']}")
    print(f"  containers: {containers()}")


def supervisor_pid():
    """PID of the supervisor loop, or None.

    `pgrep -f 'supervisor.sh 300'` matches ANY process whose command line contains that
    text — including the shell running the check. It reported four supervisors when there
    was one, and a false "several supervisors" reads exactly like the two-launcher bug that
    put Devin at 7 against a cap of 4. Match argv properly instead: argv[0] is a bash, and
    argv[1] is the script path.
    """
    for entry in os.listdir("/proc"):
        if not entry.isdigit():
            continue
        try:
            with open(f"/proc/{entry}/cmdline", "rb") as fh:
                argv = fh.read().decode(errors="replace").split("\0")
        except OSError:
            continue
        argv = [a for a in argv if a]
        if len(argv) >= 2 and os.path.basename(argv[0]) in ("bash", "sh") \
                and argv[1].endswith("scripts/ops/supervisor.sh"):
            return int(entry)
    return None

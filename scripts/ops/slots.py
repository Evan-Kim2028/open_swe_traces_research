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

# Devin concurrency. Measured, not guessed: a trial makes 4.3 tool calls/min, so each
# slot is ~257 calls/hour and the peak SUSTAINED hour we have ever run is 743 — that is
# 4 slots at about 72% of theoretical, the rest lost to container builds and gaps.
#
#   cap 4  ~1,030/h theoretical, ~740/h observed   never throttled
#   cap 6  ~1,540/h theoretical, ~1,110/h expected  1.5x the observed peak
#   cap 8  ~2,060/h theoretical, ~1,480/h expected  2x — that is finding the limit by
#                                                   hitting it
#
# At 6 on the user's call. The risk is not the 30-minute cooldown by itself: a throttle
# mid-trial errors every in-flight trial, so it costs ~30 min x 6 slots of work as well.
# That is why the throttle detector now DROPS this back to 4 by writing the override file
# below, instead of only printing a warning for someone to notice.
#
# Precedence: DEVIN_CAP env > override file written by devin_ratelimit_check > default.
_OVERRIDE = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(
    os.path.abspath(__file__)))), "outputs", "supervisor", "devin_cap_override")


def _cap_default():
    try:
        with open(_OVERRIDE) as fh:
            return int(fh.read().split()[0])
    except (OSError, ValueError, IndexError):
        return 6


CAP = int(os.environ["DEVIN_CAP"]) if os.environ.get("DEVIN_CAP") else _cap_default()


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


# The CLI lives at the BOTTOM so it can call anything defined in this module. It used
# to sit above supervisor_pid(), which meant adding a --supervisor-pid flag raised
# NameError at module level — the function existed, just twenty lines too late.

if __name__ == "__main__":
    import sys as _sys
    o = occupancy()
    # --count prints ONE integer and nothing else. It was accepted and ignored, so callers
    # did `slots.py --count | tr -dc 0-9` on the full summary and got every digit in it
    # concatenated — job names included. 6/6 came out as
    # 66065211521192049121241124526132108405819, and `[ "$occ" -gt 4 ]` then failed with
    # "integer expression expected", which is FALSE, so the over-cap alarm in the monitor
    # could never fire. It was silently dead for several monitor generations; the only
    # reason over-cap was still caught is that stability_gate computes it independently
    # from the ledger.
    if "--count" in _sys.argv:
        print(o["total"])
        raise SystemExit(0)
    if "--cap" in _sys.argv:
        print(o["cap"])
        raise SystemExit(0)
    if "--supervisor-pid" in _sys.argv:
        # Bare integer, or nothing at all when there is no supervisor, so a caller can
        # test with -z. Printing the summary here would be read as a PID.
        _p = supervisor_pid()
        if _p:
            print(_p)
        raise SystemExit(0 if _p else 1)
    # An UNRECOGNISED --flag must fail, not fall through to the summary. The --count bug
    # documented above happened a second time with --supervisor-pid: the flag did not
    # exist, slots.py printed this summary, `tr -dc 0-9` turned it into a plausible-
    # looking PID, and both supervisor checks in the monitor were dead on arrival while
    # looking perfectly healthy. A typo in a monitor must be loud, because a silent one
    # is indistinguishable from "nothing is wrong".
    _unknown = [a for a in _sys.argv[1:] if a.startswith("--")]
    if _unknown:
        print(f"slots.py: unknown flag(s) {' '.join(_unknown)}; "
              f"known: --count --cap --supervisor-pid", file=_sys.stderr)
        raise SystemExit(2)
    print(summary())
    for s in o["sessions"]:
        print(f"    session {s}")
    for t in o["trials"]:
        print(f"    trial   {t['job']} conc={t['conc']}")
    print(f"  containers: {containers()}")

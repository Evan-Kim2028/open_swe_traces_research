#!/usr/bin/env python3
"""Every in-flight agent job, whatever engine it runs on.

The monitor watched Devin sessions and containers only, so the Grok authoring runs and the
Cursor verification run were invisible: a stalled one would have sat unnoticed until
someone thought to look. Progress here is measured by ARTIFACTS PRODUCED and log growth,
not by liveness - a process that is up but writing nothing is the failure mode that cost
two cycles tonight.

  jobs_status.py [--stalled-min N] [--quiet]

Exit 1 if any job looks stalled, so a monitor can alert on it.
"""
import os, sys, glob, json, time, subprocess

from openswe_traces.ops.slots import occupancy

STALL_MIN = int(sys.argv[sys.argv.index("--stalled-min") + 1]) if "--stalled-min" in sys.argv else 20
QUIET = "--quiet" in sys.argv
TMP = "/home/evan/.claude/jobs/a1eaeb86/tmp"


def _age_min(p):
    try:
        return (time.time() - os.path.getmtime(p)) / 60
    except OSError:
        return None


def grok_jobs():
    out = []
    for s in sorted(glob.glob(f"{TMP}/grokauth_*")):
        repo = os.path.basename(s).replace("grokauth_", "")
        log = f"{s}/outputs/grokauth_{repo}.grok.log"
        units = glob.glob(f"{s}/units/*/")
        # a unit is only usable with BOTH prose artifacts: no bugreport = no L0 prompt,
        # no DETAILS = nothing to verify against
        def _nonempty(u, rel):
            p = os.path.join(u, rel)
            return os.path.exists(p) and os.path.getsize(p) > 0

        complete = [u for u in units
                    if _nonempty(u, "_author/bugreport.md")
                    and _nonempty(u, "_author/DETAILS.md")]
        # Grok's TUI never exits: grok_job.sh keeps it alive with a `sleep infinity`
        # stdin, so the process lingers at 0% CPU long after the turn is done. Liveness
        # therefore means nothing here - a job with its report written IS finished, and
        # reporting it as RUNNING would have kept its units unharvested indefinitely.
        report_p = f"{s}/outputs/grokauth_{repo}.md"
        done = os.path.exists(report_p) and os.path.getsize(report_p) > 0
        proc_up = subprocess.run(f"pgrep -f 'grokauth_{repo}' >/dev/null",
                                 shell=True).returncode == 0
        alive = proc_up and not done
        out.append({"engine": "grok", "name": repo, "alive": alive,
                    "made": len(units), "usable": len(complete),
                    "idle_min": _age_min(log), "report": done, "proc_up": proc_up,
                    "usage": _grok_usage(s)})
    return out


def _grok_usage(scratch):
    enc = scratch.replace("/", "%2F")
    c = glob.glob(f"{os.path.expanduser('~')}/.grok/sessions/{enc}/*/usage.json")
    if not c:
        return None
    d = json.load(open(max(c, key=os.path.getmtime))).get("session", {})
    return {"tok": d.get("totalTokens", 0), "usd": d.get("costUsdTicks", 0) / 1e9}


def cursor_jobs():
    out = []
    for log in glob.glob("/home/evan/Documents/oswt-*/outputs/*.cursor.log"):
        wt = log.split("/outputs/")[0]
        name = os.path.basename(log).replace(".cursor.log", "")
        alive = subprocess.run(
            f"pgrep -f 'cursor-agent' >/dev/null && ls -l /proc/$(pgrep -f cursor-agent | head -1)/cwd 2>/dev/null | grep -q {os.path.basename(wt)}",
            shell=True).returncode == 0
        # artifacts a verification job is supposed to produce
        suites = len(glob.glob(f"{wt}/experiments/pipeline/*/*/tests/hidden/**/*_test.go", recursive=True))
        contracts = len(glob.glob(f"{wt}/experiments/pipeline/*/*/_author/contract.md"))
        runners = len(glob.glob(f"{wt}/experiments/pipeline/*/*/tests/test.sh"))
        newest = max([_age_min(p) or 1e9 for p in
                      glob.glob(f"{wt}/experiments/pipeline/*/*/tests/**", recursive=True)] or [1e9])
        # cursor-agent --print buffers everything until exit, so its output log sits at
        # ~0 bytes for the whole run: measuring idleness from it reported a 20m stall on a
        # job that was actively working. The CLI's own session log under
        # /tmp/cursor-agent-logs-* is written continuously, so use the freshest of that,
        # the artifacts, and the output log.
        sess = glob.glob("/tmp/cursor-agent-logs-*/session-*.log")
        sess_age = min([_age_min(x) or 1e9 for x in sess] or [1e9])
        art_age = min([_age_min(x) or 1e9 for x in
                       glob.glob(f"{wt}/experiments/pipeline/*/*/tests/**", recursive=True)
                       + glob.glob(f"{wt}/experiments/pipeline/*/*/_author/contract.md")]
                      or [1e9])
        idle = min(sess_age, art_age, _age_min(log) or 1e9)
        out.append({"engine": "cursor", "name": name, "alive": alive,
                    "suites": suites, "runners": runners, "contracts": contracts,
                    "idle_min": None if idle >= 1e9 else idle})
    return out


def main():
    stalled = []
    o = occupancy("devin")
    lines = [f"devin {o['total']}/{o['cap']}: "
             f"{' '.join(o['sessions']) or '-'} + {len(o['trials'])} trial(s)"]

    for j in grok_jobs():
        u = j["usage"]
        cost = f" {u['tok']:,}tok ${u['usd']:.2f}" if u else " (cost at exit)"
        state = ("RUNNING" if j["alive"]
                 else ("done" + (" (idle proc)" if j.get("proc_up") else "")
                       if j["report"] else "EXITED-no-report"))
        lines.append(f"grok   {j['name']:12} {state:16} units {j['usable']}/{j['made']} usable"
                     f"  idle {j['idle_min'] or 0:.0f}m{cost}")
        if j["alive"] and (j["idle_min"] or 0) > STALL_MIN:
            stalled.append(f"grok/{j['name']} idle {j['idle_min']:.0f}m")
        if not j["alive"] and not j["report"]:
            stalled.append(f"grok/{j['name']} exited without a report")

    for j in cursor_jobs():
        state = "RUNNING" if j["alive"] else "done"
        lines.append(f"cursor {j['name']:12} {state:16} suites {j['suites']} runners "
                     f"{j['runners']} contracts {j['contracts']}  idle {j['idle_min'] or 0:.0f}m")
        if j["alive"] and (j["idle_min"] or 0) > STALL_MIN:
            stalled.append(f"cursor/{j['name']} idle {j['idle_min']:.0f}m")

    if not QUIET:
        for l in lines:
            print("  " + l)
    for s in stalled:
        print(f"  STALLED: {s}")

    # --notify: emit only what CHANGED since the last call. A monitor that reprints the
    # same four lines every five minutes trains you to ignore it; one that speaks when a
    # job finishes or gains units is worth reading.
    if "--notify" in sys.argv:
        state = {}
        for j in grok_jobs():
            state[f"grok/{j['name']}"] = {"alive": j["alive"], "usable": j["usable"],
                                          "made": j["made"], "report": j["report"]}
        for j in cursor_jobs():
            state[f"cursor/{j['name']}"] = {"alive": j["alive"], "suites": j["suites"],
                                            "contracts": j["contracts"]}
        f = "outputs/supervisor/jobs_state.json"
        prev = {}
        if os.path.exists(f):
            try:
                prev = json.load(open(f))
            except Exception:
                prev = {}
        for k, v in state.items():
            o = prev.get(k)
            if o is None:
                print(f"  JOB STARTED {k}")
            elif o.get("alive") and not v.get("alive"):
                done = v.get("report") or v.get("contracts")
                print(f"  JOB FINISHED {k}: "
                      + (f"{v.get('usable')}/{v.get('made')} usable unit(s)"
                         if "usable" in v else
                         f"{v.get('suites')} suite(s), {v.get('contracts')} contract(s)")
                      + ("" if done else "  [NO DELIVERABLE - check the log]"))
            else:
                for field in ("usable", "contracts"):
                    if field in v and v[field] > o.get(field, 0):
                        print(f"  JOB PROGRESS {k}: {field} {o.get(field,0)} -> {v[field]}")
        os.makedirs(os.path.dirname(f), exist_ok=True)
        json.dump(state, open(f, "w"), indent=1)
    return 1 if stalled else 0


def cli():
    sys.exit(main())


if __name__ == "__main__":
    cli()

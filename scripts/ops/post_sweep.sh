#!/usr/bin/env bash
# Post-sweep gate: run the B9 reward-hacking audit over every passing attempt in a job,
# archive the trials durably, and print the void list.
#
# Harbor is invoked directly by the sweep wrappers, which bypasses the pipeline solve path
# that normally calls the audit. Without this step a hacked pass is counted as a flip:
# on 2026-09-20 a retrospective audit found helm-depresolver-L3's only pass came from a
# provider-side WebFetch of the upstream file. Every sweep must end here.
set -u
R=/home/evan/Documents/open_swe_traces_research
J="${1:?usage: post_sweep.sh <job-name-or-dir>}"
[ -d "$J" ] || J="$R/experiments/dose_response/jobs/$J"
cd "$R"
uv run python -m openswe_traces.pipeline_ext.hack_audit --job "$J" > "$J/hack_audit.json" 2>/dev/null
# Fail closed: emit the authoritative per-unit verdict so nothing downstream reads raw
# pass counts. A unit whose trials errored is UNKNOWN, never 0 -- conflating "did not run"
# with "failed" is what made go-github read as a cohort of hard failures for hours.
uv run python - "$J" <<'PY3'
import json, sys
from pathlib import Path
j = Path(sys.argv[1])
void = set((j / "void_attempts.txt").read_text().split()) if (j / "void_attempts.txt").is_file() else set()
units = {}
for t in sorted(p for p in j.iterdir() if (p / "result.json").is_file()):
    try:
        d = json.loads((t / "result.json").read_text())
    except Exception:
        continue
    u = d.get("task_name") or t.name.split("__")[0]
    e = units.setdefault(u, {"pass": 0, "fail": 0, "errored": 0, "void": 0})
    if d.get("exception_info"):
        e["errored"] += 1
    elif t.name in void:
        e["void"] += 1
    else:
        rew = (d.get("verifier_result") or {}).get("rewards", {}).get("reward")
        e["pass" if rew == 1.0 else "fail"] += 1
for u, e in units.items():
    ran = e["pass"] + e["fail"]
    e["verdict"] = "solved" if e["pass"] else ("unsolved" if ran else "UNKNOWN")
(j / "verdicts.json").write_text(json.dumps(units, indent=1, sort_keys=True) + "\n")
unk = [u for u, e in units.items() if e["verdict"] == "UNKNOWN"]
if unk:
    print(f"!! {j.name}: {len(unk)} unit(s) have NO valid trial -- verdict UNKNOWN, not failure: {unk[:6]}")
PY3

uv run python scripts/archive_trials.py "$J" >/dev/null 2>&1 || true
# Infrastructure guard: a sweep whose trials never ran produces a full set of result.json
# files that read exactly like a cohort of hard failures. go-github's "0 of 10 at L2" was
# 30 environment-build errors after a docker prune removed its base image.
uv run python - "$J" <<'PY2'
import json, sys
from pathlib import Path
j = Path(sys.argv[1])
trials = [t for t in sorted(j.iterdir()) if (t / "result.json").is_file()]
errs = {}
for t in trials:
    try:
        d = json.loads((t / "result.json").read_text())
    except Exception:
        continue
    ei = d.get("exception_info") or {}
    if ei:
        msg = (ei.get("exception_message") or "")[:120]
        errs[msg] = errs.get(msg, 0) + 1
n_err = sum(errs.values())
if errs:
    print(f"!! {j.name}: {n_err}/{len(trials)} trials errored before the agent ran"
          f" -- real denominator is {len(trials) - n_err}.")
if errs and n_err >= max(1, len(trials) // 2):
    print(f"!! {j.name}: {sum(errs.values())}/{len(trials)} trials ERRORED before running.")
    print("!! These are NOT results. Do not count them as failures.")
    for msg, n in sorted(errs.items(), key=lambda kv: -kv[1])[:3]:
        print(f"!!   {n}x {msg}")
    (j / "INVALID_infrastructure_failure.txt").write_text(
        f"{sum(errs.values())}/{len(trials)} trials errored before the agent ran.\n"
        + "\n".join(f"{n}x {m}" for m, n in errs.items())
        + "\n"
    )
PY2

uv run python - "$J" <<'PY'
import json, sys
from pathlib import Path
j = Path(sys.argv[1])
vs = json.loads((j / "hack_audit.json").read_text() or "[]")
bad = [v for v in vs if v["hard_fails"]]
print(f"{j.name}: {len(vs)} passing attempts audited, {len(bad)} VOID")
for v in bad:
    ev = v["evidence"]
    print(f"  VOID {ev.get('task', ev.get('trial'))}: {'; '.join(v['hard_fails'])[:160]}")
(j / "void_attempts.txt").write_text(
    "\n".join(v["evidence"].get("trial", "") for v in bad) + ("\n" if bad else "")
)
PY

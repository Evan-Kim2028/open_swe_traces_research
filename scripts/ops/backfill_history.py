#!/usr/bin/env python3
import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
"""Replay every trial in chronological order to get the efficiency curve.

The question the dashboard has to answer is not "how many tasks do we have" but
"is each decision making a task cheaper to mine". That needs trials-per-certified
over time, which means replaying the ledger rather than summing it.

Writes outputs/history.json: one bucket per hour with cumulative and marginal rates.
"""
import json, os, glob, collections, time

JOBS = "experiments/dose_response/jobs"
OUT = "outputs/history.json"

# When each decision landed, taken from the mtime of the artifact that implements it.
# A decision that does not move the marginal rate did not work, and the chart should say so.
DECISIONS = [
    ("scripts/ops/sweep_seq.sh",         "sequential escalation (k=1, drop passers)"),
    ("scripts/ops/post_sweep.sh",        "B9 hack-audit gate on every sweep"),
    ("scripts/ops/contract_gap_read.py", "contract gap-read before trialling"),
    ("scripts/ops/task_lint.py",         "free deterministic linter"),
    ("scripts/ops/trial_guard.py",       "trial guard (refuse decided verdicts)"),
    ("scripts/ops/supervisor.sh",        "continuous supervisor"),
]


def events():
    """(time, base, rung, reward) per trial, chronological. Errored trials are excluded:
    a trial that died before the verifier ran is not evidence about the unit."""
    import trial_ledger
    ev = [(t["mtime"], t["base"], t["rung"], t["reward"])
          for t in trial_ledger.trials()
          if t["reward"] is not None and not t["errored"]]
    ev.sort()
    return ev


def main():
    ev = events()
    if not ev:
        print("no trials found")
        return 1
    per = collections.defaultdict(lambda: collections.defaultdict(list))
    certified = set()
    buckets = collections.OrderedDict()
    for t, base, rung, v in ev:
        per[base][rung].append(v)
        d = per[base]
        if base not in certified and d.get("0") and max(d["0"]) == 0 \
           and d.get("2") and max(d["2"]) > 0:
            certified.add(base)
        hr = int(t // 3600) * 3600
        b = buckets.setdefault(hr, {"t": hr, "trials": 0, "certified": 0})
        b["trials"] += 1
        b["certified"] = len(certified)

    rows, cum = [], 0
    prev_c = prev_t = 0
    for hr, b in buckets.items():
        cum += b["trials"]
        dc = b["certified"] - prev_c
        dt = cum - prev_t
        rows.append({
            "t": hr,
            "iso": time.strftime("%Y-%m-%dT%H:00", time.localtime(hr)),
            "trials_hr": b["trials"],
            "trials_cum": cum,
            "certified": b["certified"],
            "certified_hr": dc,
            # cumulative is the headline; marginal is what actually tests a decision,
            # because the cumulative average drags a whole day of history behind it
            "tpc_cum": round(cum / b["certified"], 2) if b["certified"] else None,
            "tpc_marginal": round(dt / dc, 2) if dc > 0 else None,
        })
        prev_c, prev_t = b["certified"], cum

    marks = []
    for path, label in DECISIONS:
        if os.path.exists(path):
            marks.append({"t": int(os.path.getmtime(path)),
                          "iso": time.strftime("%Y-%m-%dT%H:%M", time.localtime(os.path.getmtime(path))),
                          "label": label})
    marks.sort(key=lambda m: m["t"])

    json.dump({"generated": int(time.time()), "rows": rows, "decisions": marks},
              open(OUT, "w"), indent=1)
    print(f"{len(rows)} hourly buckets -> {OUT}")
    print(f"final: {rows[-1]['certified']} certified, {rows[-1]['trials_cum']} trials, "
          f"{rows[-1]['tpc_cum']} trials/certified cumulative")
    print("\nmarginal trials-per-certified, last 10 productive hours:")
    for r in [r for r in rows if r["tpc_marginal"]][-10:]:
        print(f"  {r['iso'][5:]}  {r['tpc_marginal']:6.2f}   (+{r['certified_hr']} certified)")
    print("\ndecisions:")
    for m in marks:
        print(f"  {m['iso'][5:]}  {m['label']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

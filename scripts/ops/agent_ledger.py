#!/usr/bin/env python3
"""Local accounting for every agent session we launch. Append-only, ours, no vendor dashboard.

Devin bills in ACUs and exposes them nowhere we can read: the local session store has no ACU,
credit or cost field, the CLI has no usage command, and the CLI key is Unauthorized on the
consumption endpoints. So we log the inputs to that number ourselves — wall-clock active time
per session, token counts, and which brief the session was running — and the vendor's ACU total
becomes a thing we reconcile against rather than depend on.

  agent_ledger.py snapshot   append the current state to experiments/dose_response/audit/agent_ledger.jsonl
  agent_ledger.py report     totals by job and by day
"""
from __future__ import annotations

import datetime as dt
import json
import re
import sqlite3
import sys
from collections import defaultdict
from pathlib import Path

DB = Path.home() / ".local/share/devin/cli/sessions.db"
LEDGER = Path("experiments/dose_response/audit/agent_ledger.jsonl")
QLOG = Path.home() / "devin-tasks/dq2.log"
ACU_MINUTES = 15.0  # Devin's own guidance: ~1 ACU per 15 min of active work


def _ts(v):
    v = float(v)
    return dt.datetime.fromtimestamp(v / 1000 if v > 1e11 else v, dt.timezone.utc)


def _walk(o, acc):
    if isinstance(o, dict):
        for k, v in o.items():
            kl = k.lower()
            if kl in ("input_tokens", "prompt_tokens"):
                acc["in"] += int(v or 0)
            elif kl in ("output_tokens", "completion_tokens"):
                acc["out"] += int(v or 0)
            elif "cache" in kl and "token" in kl and isinstance(v, (int, float)):
                acc["cache"] += int(v)
            else:
                _walk(v, acc)
    elif isinstance(o, list):
        for v in o:
            _walk(v, acc)


def launches() -> list[tuple[dt.datetime, str]]:
    """(when, job) from the queue runner's own log — the only record of which brief ran."""
    out = []
    if not QLOG.is_file():
        return out
    for line in QLOG.read_text(errors="replace").splitlines():
        m = re.match(r"(\S+) (\w+) launched", line)
        if m:
            try:
                out.append((dt.datetime.fromisoformat(m.group(1)), m.group(2)))
            except ValueError:
                pass
    return sorted(out)


def sessions() -> list[dict]:
    if not DB.is_file():
        return []
    c = sqlite3.connect(f"file:{DB}?mode=ro", uri=True)
    per: dict[str, dict] = {}
    for sid, ts, cm, md in c.execute(
        "select session_id, created_at, chat_message, metadata from message_nodes"
    ):
        e = per.setdefault(sid, {"session": sid, "msgs": 0, "in": 0, "out": 0, "cache": 0,
                                 "first": None, "last": None})
        e["msgs"] += 1
        acc = {"in": 0, "out": 0, "cache": 0}
        for blob in (cm, md):
            if blob:
                try:
                    _walk(json.loads(blob), acc)
                except Exception:
                    pass
        for k in acc:
            e[k] += acc[k]
        try:
            w = _ts(ts)
        except Exception:
            continue
        if e["first"] is None or w < e["first"]:
            e["first"] = w
        if e["last"] is None or w > e["last"]:
            e["last"] = w
    lau = launches()
    out = []
    for e in per.values():
        if e["first"] is None:
            continue
        mins = (e["last"] - e["first"]).total_seconds() / 60
        job = ""
        for when, name in lau:  # the last brief launched before this session started
            if when <= e["first"] + dt.timedelta(minutes=2):
                job = name
            else:
                break
        out.append({**e, "first": e["first"].isoformat(), "last": e["last"].isoformat(),
                    "active_min": round(mins, 1), "acu_est": round(mins / ACU_MINUTES, 2),
                    "job": job})
    return sorted(out, key=lambda e: e["first"])


def main(argv: list[str]) -> int:
    cmd = argv[0] if argv else "report"
    rows = sessions()
    if cmd == "snapshot":
        LEDGER.parent.mkdir(parents=True, exist_ok=True)
        stamp = dt.datetime.now(dt.timezone.utc).isoformat()
        with LEDGER.open("a") as fh:
            fh.write(json.dumps({"snapshot_at": stamp, "sessions": len(rows),
                                 "active_min": round(sum(r["active_min"] for r in rows), 1),
                                 "acu_est": round(sum(r["acu_est"] for r in rows), 1),
                                 "tok_in": sum(r["in"] for r in rows),
                                 "tok_out": sum(r["out"] for r in rows),
                                 "tok_cache": sum(r["cache"] for r in rows)}) + "\n")
        print(f"snapshot appended: {len(rows)} sessions, "
              f"{sum(r['active_min'] for r in rows):.0f} active min, "
              f"{sum(r['acu_est'] for r in rows):.0f} ACU-equivalent")
        return 0

    by = defaultdict(lambda: {"n": 0, "min": 0.0, "acu": 0.0, "in": 0, "out": 0, "cache": 0})
    for r in rows:
        b = by[r["job"] or "(unattributed)"]
        b["n"] += 1
        b["min"] += r["active_min"]
        b["acu"] += r["acu_est"]
        for k in ("in", "out", "cache"):
            b[k] += r[k]
    print(f"{'job':18} {'sess':>5} {'active min':>11} {'ACU~':>7} {'in+out':>10} {'cache':>10}")
    for j, b in sorted(by.items(), key=lambda kv: -kv[1]["min"]):
        print(f"{j:18} {b['n']:>5} {b['min']:>11.0f} {b['acu']:>7.1f} "
              f"{(b['in']+b['out'])/1e6:>9.1f}M {b['cache']/1e9:>9.2f}B")
    tm = sum(b["min"] for b in by.values())
    ta = sum(b["acu"] for b in by.values())
    print(f"\nTOTAL {len(rows)} sessions · {tm:.0f} active minutes · {ta:.0f} ACU-equivalent")
    print(f"ACU-equivalent uses Devin's ~{ACU_MINUTES:.0f} min/ACU guidance and is OURS, not billed.")
    print("Reconcile against the Devin dashboard's usage page; the delta calibrates the estimate.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))

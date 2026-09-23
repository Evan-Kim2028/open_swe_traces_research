#!/usr/bin/env python3
"""Actual Devin usage, read from Devin's own session database.

harbor records no tokens for the Devin adapter — every swe-2-max trial shows 0 tokens and
$0 in trial_ledger — so the Devin side of the dataset has always been unaccounted. The
Cursor dashboard does not close the gap either: it counts host SESSIONS, and went to zero
at 14:00 on the day this was written, the first full hour after the last session ended,
while 37 in-container trials completed in the same window.

The data was there the whole time. harbor's teardown copies
``~/.local/share/devin/cli/sessions.db`` out of each trial to ``agent/sessions.db``, and
that database holds:

    tool_call_state            one row per tool call
    message_nodes.metadata     num_tokens_preceding — a running context count

Estimating tokens from it takes care. `num_tokens_preceding` is the context size AT a
message, not a running total, so summing it over every message multiply-counts. What a
provider actually bills is the context of each MODEL CALL — and every `assistant` message
is exactly one model call, with `num_tokens_preceding` as its input. So:

    input  ~= sum of num_tokens_preceding over assistant messages
    output ~= assistant content length / 4     (chars-per-token rule of thumb)

The input figure is sound modulo cache discounts, which make it an OVER-estimate of what
a provider would charge and an accurate measure of context processed. The output figure
is a rule of thumb and is labelled as one. Devin bills us nothing either way; this exists
because the Devin half of the dataset otherwise reads 0 tokens forever.

Usage::

    devin_usage.py             # totals, and per-hour tool calls
    devin_usage.py --by-trial
"""

from __future__ import annotations
from openswe_traces.paths import REPO as _REPO

import argparse
import collections
import json
import pathlib
import sqlite3
import time

REPO = _REPO
JOBS = REPO / "experiments" / "dose_response" / "jobs"


def read_one(db: pathlib.Path) -> dict | None:
    try:
        con = sqlite3.connect(f"file:{db}?mode=ro", uri=True)
    except sqlite3.Error:
        return None
    try:
        calls = con.execute("select count(*) from tool_call_state").fetchone()[0]
        widest = inp = out = 0
        for cm, m in con.execute("select chat_message, metadata from message_nodes"):
            try:
                meta = json.loads(m) if m else {}
            except Exception:
                meta = {}
            ctx = int(meta.get("num_tokens_preceding") or 0) if isinstance(meta, dict) else 0
            widest = max(widest, ctx)
            try:
                msg = json.loads(cm) if cm else {}
            except Exception:
                msg = {}
            if isinstance(msg, dict) and msg.get("role") == "assistant":
                inp += ctx                                  # this call's input context
                out += len(str(msg.get("content") or "")) // 4
        row = con.execute("select model, created_at, last_activity_at from sessions").fetchone()
        model, start, end = (row or (None, 0, 0))
        return {"unit": db.parents[1].name.split("__")[0], "calls": calls,
                "widest_ctx": widest, "in_tok": inp, "out_tok": out, "model": model,
                "start": start or 0, "end": end or 0,
                "minutes": ((end or 0) - (start or 0)) / 60}
    except sqlite3.Error:
        return None
    finally:
        con.close()


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--by-trial", action="store_true")
    args = ap.parse_args()

    rows = [r for r in (read_one(p) for p in sorted(JOBS.rglob("agent/sessions.db"))) if r]
    if not rows:
        print("no Devin session databases found")
        return 0

    calls = sum(r["calls"] for r in rows)
    mins = sum(r["minutes"] for r in rows if 0 < r["minutes"] < 600)
    print(f"Devin trials with a session db : {len(rows)}")
    print(f"  tool calls, total            : {calls:,}")
    print(f"  tool calls, per trial        : {calls/len(rows):.0f} mean, "
          f"{sorted(r['calls'] for r in rows)[len(rows)//2]} median")
    print(f"  widest context, per trial    : "
          f"{sum(r['widest_ctx'] for r in rows)/len(rows)/1000:.0f}k mean, "
          f"{max(r['widest_ctx'] for r in rows)/1000:.0f}k max")
    print(f"  agent wall time, total       : {mins/60:.1f}h over {len(rows)} trials "
          f"({mins/max(len(rows),1):.0f} min each)")
    ti = sum(r["in_tok"] for r in rows)
    to = sum(r["out_tok"] for r in rows)
    print(f"\n  ESTIMATED tokens (Devin bills us nothing; this is volume processed)")
    print(f"    input   {ti/1e6:8.1f}M   sum of each model call's context")
    print(f"    output  {to/1e6:8.1f}M   assistant chars / 4, a rule of thumb")
    print(f"    total   {(ti+to)/1e6:8.1f}M   over {len(rows)} trials "
          f"= {(ti+to)/len(rows)/1e6:.2f}M per trial")
    print(f"    input is an OVER-estimate of billing: it ignores cache discounts.")

    by_hour = collections.Counter()
    for r in rows:
        if r["end"]:
            by_hour[time.strftime("%H:00", time.localtime(r["end"]))] += r["calls"]
    print("\n  tool calls by hour the trial ENDED:")
    for h in sorted(by_hour):
        print(f"    {h}  {by_hour[h]:6,}")

    if args.by_trial:
        print()
        for r in sorted(rows, key=lambda x: -x["calls"])[:15]:
            print(f"    {r['unit']:26s} {r['calls']:4d} calls  "
                  f"{r['widest_ctx']/1000:5.0f}k ctx  {r['minutes']:5.0f} min")
    return 0


def cli():
    raise SystemExit(main())


if __name__ == "__main__":
    cli()

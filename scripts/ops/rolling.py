#!/usr/bin/env python3
"""The rolling window of what actually changed.

A flat certified count with six healthy agents is a stall, and absolute numbers hide
that. This prints deltas per tick so a stall is visible on sight.

  rolling.py [n]        last n snapshots (default 12)
  rolling.py --since 6  everything in the last 6 hours
"""
import json, sys, time, os

LEDGER = "outputs/rolling.jsonl"


def load():
    if not os.path.exists(LEDGER):
        return []
    out = []
    for ln in open(LEDGER):
        ln = ln.strip()
        if ln:
            try:
                out.append(json.loads(ln))
            except Exception:
                pass
    return out


def main():
    rows = load()
    if not rows:
        print("no snapshots yet — supervisor writes one per tick")
        return 0
    if "--since" in sys.argv:
        hrs = float(sys.argv[sys.argv.index("--since") + 1])
        cut = time.time() - hrs * 3600
        rows = [r for r in rows if r.get("t", 0) >= cut]
    else:
        n = int(sys.argv[1]) if len(sys.argv) > 1 and sys.argv[1].isdigit() else 12
        rows = rows[-n:]
    if not rows:
        print("no snapshots in that window")
        return 0

    print(f"{'time':8s} {'cert':>5s} {'Δ':>4s} {'trials':>7s} {'Δ':>5s} "
          f"{'easy':>5s} {'nflip':>6s} {'cont':>5s} {'dsk':>5s}  agents")
    prev = None
    for r in rows:
        dc = r["certified"] - prev["certified"] if prev else 0
        dt = r["trials"] - prev["trials"] if prev else 0
        mark = " " if dc or dt else "."          # '.' marks a tick where nothing moved
        # the shell fields arrive newline-separated; collapse or the table breaks
        agents = " ".join((r.get("devin") or "").replace("closure_", "").split())
        sw = " ".join((r.get("sweeps") or "").split())
        print(f"{r['iso'][11:19]} {r['certified']:5d} {dc:+4d} {r['trials']:7d} {dt:+5d} "
              f"{r['too_easy']:5d} {r['nonflip']:6d} {r['containers']:5d} "
              f"{r.get('free_gb',0):4d}G {mark}{agents}{('| '+sw) if sw else ''}")
        prev = r

    first, last = rows[0], rows[-1]
    hrs = max((last["t"] - first["t"]) / 3600, 1e-9)
    dc, dt = last["certified"] - first["certified"], last["trials"] - first["trials"]
    print(f"\nwindow {hrs:.1f}h   certified {dc:+d} ({dc/hrs:.1f}/h)   trials {dt:+d} ({dt/hrs:.1f}/h)", end="")
    print(f"   {dt/dc:.1f} trials per certified" if dc > 0 else "   — no certification this window")
    flat = sum(1 for a, b in zip(rows, rows[1:]) if b["certified"] == a["certified"])
    if rows[1:] and flat == len(rows) - 1:
        print("STALL: certified count did not move across the entire window.")
    return 0


if __name__ == "__main__":
    sys.exit(main())

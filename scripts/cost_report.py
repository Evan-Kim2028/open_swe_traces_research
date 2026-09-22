"""Total spend and token usage across every Harbor job plus the pipeline token ledger.

Budget epochs: the user resets the counter per day. Jobs finished before EPOCH_START
are reported separately as prior spend and do not consume the current budget.
"""
from __future__ import annotations
import datetime as _dt
import json, sqlite3, sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

# Counter reset by the user at 2026-09-20 09:30 EST (13:30 UTC): new 1B-token budget.
EPOCH_START = _dt.datetime(2026, 9, 20, 13, 30, tzinfo=_dt.timezone.utc).timestamp()
BUDGET = 2_000_000_000


def main() -> int:
    rows, tot_c, tot_i, tot_o, tot_t = [], 0.0, 0, 0, 0
    for jobs_dir in (ROOT / "experiments/dose_response/jobs", ROOT / "experiments/pipeline/jobs"):
        if not jobs_dir.is_dir():
            continue
        for job in sorted(p for p in jobs_dir.iterdir() if p.is_dir()):
            res = job / "result.json"
            if not res.is_file():
                continue
            try:
                s = json.loads(res.read_text()).get("stats", {})
            except Exception:
                continue
            c = s.get("cost_usd") or 0.0
            i = s.get("n_input_tokens") or 0
            o = s.get("n_output_tokens") or 0
            k = s.get("n_cache_tokens") or 0
            n = s.get("n_completed_trials") or 0
            cur = res.stat().st_mtime >= EPOCH_START
            rows.append((job.name, n, c, i, o, k, cur))
            if cur:
                tot_c += c; tot_i += i; tot_o += o; tot_t += n
    tot_k = sum(r[5] for r in rows if r[6])
    print(f"{'job':22s} {'trials':>6} {'cost $':>9} {'in tok':>13} {'out tok':>10} {'cached':>13} {'ALL tok':>14}")
    for name, n, c, i, o, k, cur in rows:
        mark = "" if cur else "  (prior epoch)"
        print(f"{name:22s} {n:>6} {c:>9.2f} {i:>13,} {o:>10,} {k:>13,} {i+o:>14,}{mark}")
    print(f"{'TOTAL (this epoch)':22s} {tot_t:>6} {tot_c:>9.2f} {tot_i:>13,} {tot_o:>10,} {tot_k:>13,} {tot_i+tot_o:>14,}")
    pri_c = sum(r[2] for r in rows if not r[6])
    pri_t = sum(r[3] + r[4] for r in rows if not r[6])
    print(f"{'prior epochs':22s} {sum(r[1] for r in rows if not r[6]):>6} {pri_c:>9.2f} {'':>13} {'':>10} {'':>13} {pri_t:>14,}")
    used = tot_i + tot_o
    print(f"\nBudget {BUDGET:,} tokens (in+out): used {used:,} ({used/BUDGET:.1%}), remaining {BUDGET-used:,}")
    if tot_t:
        print(f"Per trial: {used//tot_t:,} tokens, ${tot_c/tot_t:.2f} -> remaining budget is ~{(BUDGET-used)//max(1,used//tot_t):,} more trials")
    db = ROOT / "experiments/pipeline/state.db"
    if db.is_file():
        c = sqlite3.connect(db)
        for src, i, o in c.execute("select source, sum(tokens_in), sum(tokens_out) from tokens group by source"):
            print(f"  ledger {src}: in {i or 0:,} out {o or 0:,}")
    print("\nDevin swe-2 variants: listed Free. DeepSeek V4 Flash: $0.14/1M in, $0.28/1M out (daily quota exhausted 19:06).")
    return 0


if __name__ == "__main__":
    sys.exit(main())

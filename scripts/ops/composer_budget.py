#!/usr/bin/env python3
"""Composer spend since the baseline, against a hard 250M-token budget.

Set 2026-09-20 23:1x: Devin does as much as possible (it is free), Composer is capped.
The baseline is the lifetime total at the moment the budget was set, so this counts from
zero rather than against the 1.9B already spent.

Exit 0 under budget, 1 over. The supervisor refuses to launch Composer sweeps when over.
"""
import sys, os, json
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import trial_ledger

F = "outputs/composer_budget.json"
cfg = json.load(open(F)) if os.path.exists(F) else {"baseline_tokens": 0, "budget": 250_000_000}
tot = sum(t["tokens"] for t in trial_ledger.trials()
          if (t.get("model") or "").startswith("composer"))
used = max(0, tot - cfg["baseline_tokens"])
budget = cfg["budget"]
pct = used / budget if budget else 1
print(f"composer used {used/1e6:.1f}M of {budget/1e6:.0f}M ({pct:.0%})")
if "--quiet" not in sys.argv:
    cost = sum(t["cost"] for t in trial_ledger.trials()
               if (t.get("model") or "").startswith("composer"))
    print(f"  lifetime composer tokens {tot/1e9:.2f}B, baseline {cfg['baseline_tokens']/1e9:.2f}B")
    print(f"  remaining {max(0,budget-used)/1e6:.1f}M")
sys.exit(1 if used >= budget else 0)

#!/usr/bin/env python3
"""Composer spend since the baseline, against a hard token budget.

Devin does as much as possible (it is free); Composer is capped. The baseline is the
lifetime composer total at the moment the budget was set, so an allowance counts from
zero rather than against everything already spent.

Every allowance so far was granted by hand-editing the JSON, and the first one
overshot by 23% because nothing re-anchored the baseline atomically with the raise.
``--set`` does both in one step and keeps the previous allowance in ``history``.

Exit 0 under budget, 1 over. orchestrate refuses to launch Composer sweeps when over.

Usage::

    composer_budget.py                  # report; exit 1 if over
    composer_budget.py --quiet          # exit code only
    composer_budget.py --set 500M       # grant a new allowance, re-anchored to now
"""
from __future__ import annotations

import json
import os
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import trial_ledger  # noqa: E402

F = "outputs/composer_budget.json"
DEFAULT = {"baseline_tokens": 0, "budget": 250_000_000}


def load() -> dict:
    return json.load(open(F)) if os.path.exists(F) else dict(DEFAULT)


def composer_lifetime() -> float:
    return sum(t["tokens"] for t in trial_ledger.trials()
               if (t.get("model") or "").startswith("composer"))


def parse_amount(s: str) -> int:
    s = s.strip().upper().replace("_", "").replace(",", "")
    mult = {"K": 1e3, "M": 1e6, "B": 1e9}.get(s[-1:], 1)
    return int(float(s[:-1] if mult != 1 else s) * mult)


def set_budget(amount: int, note: str = "") -> dict:
    """Grant a new allowance anchored to the lifetime total right now.

    Anchoring and raising must happen together. Raising alone counts the new budget
    against spend that a previous allowance already paid for; re-anchoring alone
    silently grants an unbounded one.
    """
    cfg = load()
    new = {
        "baseline_tokens": int(composer_lifetime()),
        "budget": int(amount),
        "set_at": time.strftime("%Y-%m-%dT%H:%M:%S"),
        "note": note,
        "history": (cfg.get("history") or []) + [{
            k: cfg.get(k) for k in ("baseline_tokens", "budget", "set_at", "note")
        }],
    }
    os.makedirs(os.path.dirname(F), exist_ok=True)
    tmp = F + ".tmp"
    with open(tmp, "w") as fh:
        json.dump(new, fh, indent=1)
    os.replace(tmp, new_path := F)  # atomic: a torn budget file reads as unbounded
    assert new_path == F
    return new


def status() -> tuple[float, float, float]:
    cfg = load()
    tot = composer_lifetime()
    used = max(0.0, tot - cfg["baseline_tokens"])
    return used, float(cfg["budget"]), tot


def main() -> int:
    if "--set" in sys.argv:
        amount = parse_amount(sys.argv[sys.argv.index("--set") + 1])
        note = " ".join(sys.argv[sys.argv.index("--set") + 2:]) or "granted via --set"
        new = set_budget(amount, note)
        print(f"composer budget set to {new['budget']/1e6:.0f}M, "
              f"baseline re-anchored to {new['baseline_tokens']/1e9:.2f}B")
        return 0

    used, budget, tot = status()
    pct = used / budget if budget else 1
    print(f"composer used {used/1e6:.1f}M of {budget/1e6:.0f}M ({pct:.0%})")
    if "--quiet" not in sys.argv:
        cost = sum(t["cost"] for t in trial_ledger.trials()
                   if (t.get("model") or "").startswith("composer"))
        print(f"  lifetime composer tokens {tot/1e9:.2f}B (${cost:.2f}), "
              f"baseline {load()['baseline_tokens']/1e9:.2f}B")
        print(f"  remaining {max(0.0, budget - used)/1e6:.1f}M")
    return 1 if used >= budget else 0


if __name__ == "__main__":
    raise SystemExit(main())

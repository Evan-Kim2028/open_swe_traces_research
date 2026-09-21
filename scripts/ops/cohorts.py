#!/usr/bin/env python3
"""Facts about a staged cohort that more than one caller needs.

`barren` lived inside orchestrate.py, which runs its whole reconciliation at import time —
so the monitor could not import it and grepped the sweep logs itself instead. That grep
counted ANY historical "guard kept 0 unit" line and alerted on three cohorts that were
all healthy. A check that cries wolf about launch loops is worse than no check: the
response to a real one is to go looking for a cohort to repair.

Usage::

    cohorts.py            # list every cohort, flagging the barren ones
    cohorts.py --barren   # names only, one per line
"""

from __future__ import annotations

import os
import re
import sys

R = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SWEEPS = os.path.join(R, "experiments", "dose_response")


def barren(cohort: str) -> bool:
    """True when this cohort's LAST sweep kept 0 units and nothing has changed since.

    The last run is the one that counts — an earlier healthy run must not mask a later
    barren one, and a later repair must not be masked by an earlier failure. Reads the
    sweep's own log rather than re-running task_lint, which is the expensive part the
    sweep has already paid for.

    Self-healing on purpose: touching the cohort (repairing a unit) clears the flag, so
    there is no state to reset by hand. That is how sweep_grokgogit came back on its own
    after repair_b6 gave its ten units a reproduce command.
    """
    log = os.path.join(R, "outputs", f"{cohort}.log")
    unit_dir = os.path.join(SWEEPS, cohort)
    try:
        with open(log, errors="replace") as fh:
            kept = re.findall(r"guard kept (\d+) unit", fh.read())
        if not kept or kept[-1] != "0":
            return False
        newest = max((os.path.getmtime(os.path.join(dp, f))
                      for dp, _dn, fn in os.walk(unit_dir) for f in fn), default=0)
        return newest <= os.path.getmtime(log)
    except (OSError, ValueError):
        return False


def all_cohorts() -> list[str]:
    if not os.path.isdir(SWEEPS):
        return []
    return sorted(d for d in os.listdir(SWEEPS)
                  if d.startswith("sweep_") and os.path.isdir(os.path.join(SWEEPS, d)))


def main() -> int:
    names = [c for c in all_cohorts() if barren(c)]
    if "--barren" in sys.argv:
        print("\n".join(names))
        return 0
    print(f"cohorts {len(all_cohorts())}, barren {len(names)}")
    for c in names:
        n = len(os.listdir(os.path.join(SWEEPS, c)))
        print(f"  BARREN {c}  ({n} unit(s), all refused downstream — repair, do not relaunch)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

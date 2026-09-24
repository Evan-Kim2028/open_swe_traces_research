"""In-image preflight gate for packaged tasks (see pipeline/preflight.py).

For each task dir: build the task image, run the task's own tests/test.sh on
the bare excised tree (must FAIL with assertions/panic, never [setup failed]),
with the gold patch (must PASS), and with the cheat patch when present (must
FAIL). Verdicts land in the task's validation.json; exit 1 if any dir fails.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from openswe_traces.pipeline.preflight import preflight_task


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("task_dirs", nargs="+", type=Path)
    ap.add_argument("--json", action="store_true", help="print a JSON report per dir")
    args = ap.parse_args()
    rows = []
    bad = 0
    for td in args.task_dirs:
        report = preflight_task(td)
        rows.append(report.as_dict() | {"task_dir": str(td)})
        mark = "PASS" if report.verdict == "pass" else "FAIL"
        if report.verdict != "pass":
            bad += 1
        gold = report.gold.verdict if report.gold else "absent"
        cheat = report.cheat.verdict if report.cheat else "absent"
        print(
            f"{mark} {td.name}: bare={report.bare.verdict} gold={gold} cheat={cheat} "
            f"({report.seconds:.0f}s)",
            file=sys.stderr,
            flush=True,
        )
    if args.json:
        print(json.dumps(rows, indent=2))
    print(f"preflight: {len(rows) - bad}/{len(rows)} passed", file=sys.stderr)
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())

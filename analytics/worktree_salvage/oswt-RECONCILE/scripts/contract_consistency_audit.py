"""A13 judge driver — contract-vs-gold consistency over the packaged bank.

    uv run python scripts/contract_consistency_audit.py [--limit N] [--only repo/unit ...]

Walks ``experiments/pipeline/tasks_composerver/<repo>/<unit>-L<level>``,
gathers A13 evidence for every level that ships a coverage table, and calls
the OpenRouter judge (``openswe_traces.contract_judge``). One request per
distinct contract (L2/L5/L6 share a contract → one cache entry), hard cap
``--max-requests`` (default 300), cached under
``outputs/contract_judge/<key>.json``; re-runs resume from the cache.
Appends progress to ``outputs/CONTRACTFIX.log``.
"""

from __future__ import annotations

import argparse
import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from openswe_traces.contract_judge import (
    MAX_REQUESTS,
    MODELS,
    judge_units,
    load_api_key,
)
from openswe_traces.data import ROOT
from openswe_traces.gate.context import GateContext
from openswe_traces.gate.contract_gold import JUDGE_DIR, gather_evidence

TASKS = ROOT / "experiments" / "pipeline" / "tasks_composerver"
LOG = ROOT / "outputs" / "CONTRACTFIX.log"


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--tasks-root", type=Path, default=TASKS)
    ap.add_argument("--max-requests", type=int, default=MAX_REQUESTS)
    ap.add_argument("--only", nargs="*", default=None,
                    help="restrict to repo/unit pairs, e.g. --only helm/chartloader")
    ap.add_argument("--model", nargs="*", default=None,
                    help="override the free-tier model list (first reachable wins)")
    args = ap.parse_args()

    only = set(args.only or [])
    units = []
    seen_keys: set[str] = set()
    for td in sorted(args.tasks_root.glob("*/*-L*")):
        if not td.is_dir():
            continue
        repo, unit = td.parent.name, td.name.rsplit("-L", 1)[0]
        if only and f"{repo}/{unit}" not in only:
            continue
        ev = gather_evidence(GateContext(task_dir=td))
        if not ev.rows:
            continue
        if ev.contract_key in seen_keys:
            continue  # same contract at L2/L5/L6 judges once
        seen_keys.add(ev.contract_key)
        units.append((td, ev))

    LOG.parent.mkdir(parents=True, exist_ok=True)
    logf = LOG.open("a")
    def log(msg: str) -> None:
        line = f"[{time.strftime('%H:%M:%S')}] a13-judge: {msg}"
        print(line, flush=True)
        logf.write(line + "\n")
        logf.flush()

    log(f"judging {len(units)} distinct contracts "
        f"(cap {args.max_requests} requests, cache {JUDGE_DIR})")
    api_key = load_api_key()
    outcomes = judge_units(
        units, api_key,
        models=args.model or MODELS,
        max_requests=args.max_requests,
        log=log,
    )
    ok = sum(1 for o in outcomes if o.verdict is not None)
    spent = sum(o.requests for o in outcomes)
    failed = [o for o in outcomes if o.verdict is None]
    log(f"done: {ok}/{len(outcomes)} judged, {spent} requests spent")
    for o in failed:
        log(f"  FAILED {o.key[:16]} ({o.model}): {o.error}")
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())

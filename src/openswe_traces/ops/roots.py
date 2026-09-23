"""Where units live, across the main checkout and every Devin worktree.

Eleven files glob `oswt-*` themselves and each got a slightly different answer. The
consequences were not cosmetic: the census took the max across roots and reported
"20/20 verified" while the stager read only the main checkout and saw nothing, so 74
finished units sat invisible for hours. A dashboard count that scanned main only
reported 144 authored against a true 507.

One definition of a unit and where to find it.
"""
from __future__ import annotations
import os, glob

MAIN = "/home/evan/Documents/open_swe_traces_research"
# The consolidated batch supersedes the per-repo authoring batches it was built from;
# counting both double-counts every unit (629 against a true 507).
BATCH_ALIAS = {"authored_au5": "authored_batch5"}


def roots():
    return [MAIN] + sorted(glob.glob("/home/evan/Documents/oswt-*"))


def _canon_batch(b):
    for pre, to in BATCH_ALIAS.items():
        if b.startswith(pre):
            return to
    return b


def units(repo=None, batch=None):
    """(batch, repo, name) -> dict of what exists, best observation across every root."""
    out = {}
    for root in roots():
        for d in glob.glob(f"{root}/experiments/pipeline/authored*/*/*/"):
            p = d.rstrip("/").split("/")
            b, r, n = _canon_batch(p[-3]), p[-2], p[-1]
            if r.startswith("_") or (repo and r != repo) or (batch and b != batch):
                continue
            if not os.path.isdir(os.path.join(d, "_author")):
                continue
            has_suite = bool(glob.glob(d + "tests/hidden/**/*_test.go", recursive=True))
            rec = {
                "dirs": [d],
                "suite": has_suite,
                "runner": os.path.exists(os.path.join(d, "tests/test.sh")),
                "contract": os.path.exists(os.path.join(d, "_author/contract.md")),
                "gold": os.path.exists(os.path.join(d, "_author/gold.patch")),
                "bugreport": os.path.exists(os.path.join(d, "_author/bugreport.md")),
                "cheat": os.path.exists(os.path.join(d, "_author/cheat.patch")),
            }
            prev = out.get((b, r, n))
            if prev:
                # best observation wins per field; a unit's artifacts are routinely SPLIT
                # across worktrees (one holds the suite, another the _author files)
                for k in rec:
                    if k == "dirs":
                        prev["dirs"].append(d)
                    else:
                        prev[k] = prev[k] or rec[k]
            else:
                out[(b, r, n)] = rec
    return out


def stageable(u):
    """What stage_units.discover_units() actually demands."""
    return u["suite"] and u["runner"] and u["contract"] and u["gold"]


def cli():
    import collections, sys
    us = units(repo=(sys.argv[1] if len(sys.argv) > 1 else None))
    by = collections.Counter(b for (b, _, _) in us)
    print(f"{len(us)} unit(s) across {len(roots())} root(s)")
    print(f"  {'batch':22}{'units':>7}{'suite':>7}{'runner':>8}{'contract':>10}{'stageable':>11}")
    for b in sorted(by):
        sel = [v for (bb, _, _), v in us.items() if bb == b]
        print(f"  {b:22}{len(sel):7}{sum(x['suite'] for x in sel):7}"
              f"{sum(x['runner'] for x in sel):8}{sum(x['contract'] for x in sel):10}"
              f"{sum(stageable(x) for x in sel):11}")


if __name__ == "__main__":
    cli()

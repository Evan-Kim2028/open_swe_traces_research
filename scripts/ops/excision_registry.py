#!/usr/bin/env python3
"""The list of code regions the bank has already claimed.

20% of authored units excise line ranges that overlap another unit's, because nothing
ever told an authoring job what had already been taken. Each job saw an empty repo and
went for the same obvious targets: github/github.go carries 20 units, gin's vendored
deps/validator/baked_in.go carries 16, bbolt's db.go carries 10.

This emits the claim list for one repo so an authoring brief can forbid re-cutting the
same code. Cheap to regenerate, so it is always current.

Usage: excision_registry.py <repo> [--md] [--budget N]
"""
import os, re, sys, glob, collections

R = "/home/evan/Documents/open_swe_traces_research"
FILEA = re.compile(r"^--- a/(.+)$", re.M)
HUNK = re.compile(r"^@@ -(\d+)(?:,(\d+))? ")
# how many units one file may carry before it is declared exhausted
DEFAULT_BUDGET = 3


def claims(repo):
    """file -> [(unit, start, end)] across every root, deduplicated."""
    out = collections.defaultdict(list)
    seen = set()
    for root in [R] + sorted(glob.glob("/home/evan/Documents/oswt-*")):
        for d in glob.glob(f"{root}/experiments/pipeline/authored*/*/{repo}/*/".replace(f"/{repo}/", f"/{repo}/")):
            pass
    for root in [R] + sorted(glob.glob("/home/evan/Documents/oswt-*")):
        for d in glob.glob(f"{root}/experiments/pipeline/authored*/*/*/"):
            p = d.rstrip("/").split("/")
            r, name = p[-2], p[-1]
            if r != repo or name in seen:
                continue
            src = os.path.join(d, "_author/gold.patch")
            if not os.path.exists(src):
                src = os.path.join(d, "_author/excised/excision.patch")
            if not os.path.exists(src):
                continue
            seen.add(name)
            cur = None
            for ln in open(src, errors="replace").read().splitlines():
                m = FILEA.match(ln)
                if m:
                    cur = m.group(1)
                    continue
                h = HUNK.match(ln)
                if h and cur:
                    s = int(h.group(1))
                    out[cur].append((name, s, s + int(h.group(2) or 1)))
    return out


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        return 2
    repo = sys.argv[1]
    budget = int(sys.argv[sys.argv.index("--budget") + 1]) if "--budget" in sys.argv else DEFAULT_BUDGET
    c = claims(repo)
    if not c:
        print(f"no prior units for {repo} — the whole repo is open")
        return 0
    exhausted = {f: v for f, v in c.items() if len({u for u, _, _ in v}) >= budget}

    if "--md" not in sys.argv:
        print(f"{repo}: {sum(len({u for u,_,_ in v}) for v in c.values())} claim(s) "
              f"across {len(c)} file(s); {len(exhausted)} file(s) at/over budget {budget}")
        for f, v in sorted(c.items(), key=lambda x: -len({u for u, _, _ in x[1]}))[:15]:
            us = sorted({u for u, _, _ in v})
            print(f"  {len(us):3}  {f}{'   [EXHAUSTED]' if f in exhausted else ''}")
            print(f"       {' '.join(us)[:100]}")
        return 0

    # markdown block for an authoring brief
    print(f"## Already claimed in {repo} — do not re-cut these\n")
    print(f"{len(c)} file(s) already carry a unit. **A file at {budget}+ units is "
          f"exhausted: pick a different file.** For every other file, your excision's "
          f"line range must not overlap a range listed below.\n")
    if exhausted:
        print("**Exhausted — choose something else entirely:**\n")
        for f in sorted(exhausted, key=lambda x: -len({u for u, _, _ in c[x]})):
            n = len({u for u, _, _ in c[f]})
            print(f"- `{f}` ({n} units)")
        print()
    partial = [(f, v) for f, v in c.items() if f not in exhausted]
    if partial:
        print("**Partly claimed — avoid these line ranges:**\n")
        for f, v in sorted(partial, key=lambda x: -len(x[1]))[:40]:
            rng = ", ".join(f"{s}-{e}" for _, s, e in sorted(v, key=lambda x: x[1])[:6])
            print(f"- `{f}` lines {rng}")
        print()
    print("Pick code that is **behaviourally distinct**, not merely in a different file: "
          "two units that both parse the same header, or both format the same struct, "
          "test one skill twice even when the lines differ.")
    return 0


if __name__ == "__main__":
    sys.exit(main())

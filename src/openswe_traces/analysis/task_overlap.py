#!/usr/bin/env python3
"""Measure how much the task dataset repeats itself.

Redundancy is not one thing, so this reports four increasingly strict levels:

  package   two units cut code from the same package     - weak signal, often fine
  file      two units cut the same source FILE           - suspicious
  region    their excised line ranges OVERLAP            - near-duplicate
  symbol    they name the same functions/types           - same skill tested twice

Only `region` and `symbol` are strong evidence of a wasted task. A repo with many
units in one package is normal; two units that excise overlapping lines of the same
function are the same task wearing different names, and a solver that can do one can
almost certainly do the other - so the pair buys one task's worth of signal for two
tasks' worth of trials.

Usage: task_overlap.py [--repo NAME] [--show N] [--json OUT]
"""
import os, re, sys, glob, json, collections, itertools

R = "/home/evan/Documents/open_swe_traces_research"
SHOW = int(sys.argv[sys.argv.index("--show") + 1]) if "--show" in sys.argv else 12
ONLY = sys.argv[sys.argv.index("--repo") + 1] if "--repo" in sys.argv else None

HUNK = re.compile(r"^@@ -(\d+)(?:,(\d+))? ")
FILEA = re.compile(r"^--- a/(.+)$", re.M)
# Go identifiers that look like declarations or calls worth comparing
IDENT = re.compile(r"\b([A-Za-z_][A-Za-z0-9_]{3,})\b")
STOP = {"func", "type", "struct", "interface", "return", "string", "error", "bool",
        "nil", "true", "false", "range", "import", "package", "const", "byte", "int",
        "make", "append", "len", "cap", "value", "values", "name", "names", "test",
        "tests", "case", "switch", "default", "result", "expected", "actual", "want",
        "got", "input", "output", "data", "this", "that", "with", "from", "when",
        "then", "must", "should", "which", "where", "each", "into", "only", "same"}


def units():
    """(repo, name) -> {files, regions, symbols}. Deduplicated across every root."""
    out = {}
    roots = [R] + sorted(glob.glob("/home/evan/Documents/oswt-*"))
    for root in roots:
        for d in glob.glob(f"{root}/experiments/pipeline/authored*/*/*/"):
            p = d.rstrip("/").split("/")
            repo, name = p[-2], p[-1]
            if repo.startswith("_") or (ONLY and repo != ONLY):
                continue
            key = (repo, name)
            if key in out:
                continue
            gp = os.path.join(d, "_author/gold.patch")
            ep = os.path.join(d, "_author/excised/excision.patch")
            src = gp if os.path.exists(gp) else ep
            if not os.path.exists(src):
                continue
            try:
                body = open(src, errors="replace").read()
            except OSError:
                continue
            files, regions, cur = set(), [], None
            for ln in body.splitlines():
                m = FILEA.match(ln)
                if m:
                    cur = m.group(1)
                    files.add(cur)
                    continue
                h = HUNK.match(ln)
                if h and cur:
                    start = int(h.group(1))
                    span = int(h.group(2) or 1)
                    regions.append((cur, start, start + span))
            syms = set()
            for f in ("_author/DETAILS.md", "_author/api.md"):
                fp = os.path.join(d, f)
                if os.path.exists(fp):
                    txt = open(fp, errors="replace").read()
                    # backticked identifiers are the author's own list of what matters
                    for tok in re.findall(r"`([A-Za-z_][A-Za-z0-9_.]{3,})`", txt):
                        tok = tok.split(".")[-1]
                        if tok.lower() not in STOP:
                            syms.add(tok)
            if files:
                out[key] = {"files": files, "regions": regions, "symbols": syms}
    return out


def overlaps(a, b):
    """Do any excised regions of a and b cover the same lines of the same file?"""
    for f1, s1, e1 in a["regions"]:
        for f2, s2, e2 in b["regions"]:
            if f1 == f2 and s1 < e2 and s2 < e1:
                return f1
    return None


def main():
    u = units()
    if not u:
        print("no units found")
        return 1
    by_repo = collections.Counter(r for r, _ in u)
    print(f"{len(u)} unit(s) across {len(by_repo)} repo(s)\n")

    # ---- level 1/2: package and file concentration ---------------------------------
    file_owners = collections.defaultdict(list)
    pkg_owners = collections.defaultdict(list)
    for (repo, name), d in u.items():
        for f in d["files"]:
            file_owners[(repo, f)].append(name)
            pkg_owners[(repo, os.path.dirname(f))].append(name)

    shared_files = {k: v for k, v in file_owners.items() if len(set(v)) > 1}
    units_sharing = len({(k[0], n) for k, v in shared_files.items() for n in v})
    print(f"FILE     {len(shared_files)} source file(s) are cut by >1 unit; "
          f"{units_sharing} unit(s) ({units_sharing/len(u)*100:.0f}%) share a file with another")

    # ---- level 3: true region overlap ----------------------------------------------
    pairs = []
    byrepo = collections.defaultdict(list)
    for k in u:
        byrepo[k[0]].append(k)
    for repo, keys in byrepo.items():
        # only compare units that share at least one file - avoids an O(n^2) blowup
        cand = collections.defaultdict(list)
        for k in keys:
            for f in u[k]["files"]:
                cand[f].append(k)
        seen = set()
        for f, ks in cand.items():
            if len(ks) < 2:
                continue
            for a, b in itertools.combinations(sorted(ks), 2):
                if (a, b) in seen:
                    continue
                seen.add((a, b))
                hit = overlaps(u[a], u[b])
                if hit:
                    ja = len(u[a]["symbols"] & u[b]["symbols"])
                    jb = len(u[a]["symbols"] | u[b]["symbols"]) or 1
                    pairs.append((repo, a[1], b[1], hit, ja / jb))
    dup_units = {(r, n) for r, a, b, _, _ in pairs for n in (a, b)}
    print(f"REGION   {len(pairs)} unit PAIR(S) excise overlapping line ranges; "
          f"{len(dup_units)} unit(s) ({len(dup_units)/len(u)*100:.0f}%) are in such a pair")

    # ---- level 4: symbol overlap ----------------------------------------------------
    sym_pairs = []
    for repo, keys in byrepo.items():
        idx = collections.defaultdict(set)
        for k in keys:
            for s in u[k]["symbols"]:
                idx[s].add(k)
        cand = set()
        for s, ks in idx.items():
            if 2 <= len(ks) <= 12:
                cand |= set(itertools.combinations(sorted(ks), 2))
        for a, b in cand:
            sa, sb = u[a]["symbols"], u[b]["symbols"]
            if len(sa) < 3 or len(sb) < 3:
                continue
            j = len(sa & sb) / (len(sa | sb) or 1)
            if j >= 0.34:
                sym_pairs.append((repo, a[1], b[1], j, sorted(sa & sb)[:6]))
    sym_units = {(r, n) for r, a, b, _, _ in sym_pairs for n in (a, b)}
    print(f"SYMBOL   {len(sym_pairs)} pair(s) share >=34% of named symbols; "
          f"{len(sym_units)} unit(s) ({len(sym_units)/len(u)*100:.0f}%) involved\n")

    # ---- per repo -------------------------------------------------------------------
    print(f"{'repo':14}{'units':>6}{'files':>7}{'u/file':>8}{'region-dup':>12}{'symbol-dup':>12}")
    for repo, n in by_repo.most_common():
        nf = len({f for (r, f) in file_owners if r == repo})
        rd = len({x for x in dup_units if x[0] == repo})
        sd = len({x for x in sym_units if x[0] == repo})
        print(f"  {repo:12}{n:6}{nf:7}{n/max(nf,1):8.2f}{rd:12}{sd:12}")

    if pairs:
        print(f"\nworst REGION overlaps (same lines of the same file):")
        for repo, a, b, f, j in sorted(pairs, key=lambda x: -x[4])[:SHOW]:
            print(f"  {repo:11} {a:20} ~ {b:20} sym={j:.0%}  {f}")
    if sym_pairs:
        print(f"\nworst SYMBOL overlaps (different code, same surface):")
        for repo, a, b, j, sh in sorted(sym_pairs, key=lambda x: -x[3])[:SHOW]:
            print(f"  {repo:11} {a:20} ~ {b:20} {j:.0%}  {' '.join(sh)}")

    if "--json" in sys.argv:
        out = sys.argv[sys.argv.index("--json") + 1]
        json.dump({"units": len(u),
                   "region_pairs": [list(p[:4]) + [p[4]] for p in pairs],
                   "symbol_pairs": [list(p[:4]) + [p[4]] for p in sym_pairs]},
                  open(out, "w"), indent=1)
        print(f"\nwrote {out}")
    return 0


def cli():
    sys.exit(main())


if __name__ == "__main__":
    cli()

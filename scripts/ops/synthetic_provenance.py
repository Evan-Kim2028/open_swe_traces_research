#!/usr/bin/env python3
"""Evidence that these tasks are synthetic, not harvested from GitHub issues.

The claim under test: every unit is a defect WE created by excising working upstream code,
not a real reported bug. If that holds, the dataset cannot be contaminated by anything a model
saw during training - no issue thread, no fix commit, no CVE writeup.

Three independent lines of evidence, because any one alone is weak:

  1. TEXT     no task-facing file references an issue, PR, CVE or commit. A harvested task
              leaks its origin in the prose almost every time.
  2. SHAPE    gold.patch RESTORES code rather than changing logic. A real bug fix edits a
              condition or adds a guard; an excision's gold is overwhelmingly additions,
              because the "bug" is the absence of code we deleted.
  3. RECORD   each unit carries an excision record naming what was cut, which is the
              construction step a harvested task has no reason to have.
"""
import os, re, sys, glob, json, collections

# Deliberately broad: it is better to hand-inspect a false positive than to miss a leak.
ORIGIN = re.compile(
    r"(?:github\.com/[\w.-]+/[\w.-]+/(?:issues|pull)/\d+)"      # issue or PR URL
    r"|(?:\b(?:fixes|closes|resolves|refs)\s+#\d+)"             # commit trailer
    r"|(?:\bissue\s*#\s*\d+)|(?:\bPR\s*#\s*\d+)"
    r"|(?:\bCVE-\d{4}-\d+)"
    r"|(?:\bgolang/go#\d+)"
    r"|(?:\bcommit\s+[0-9a-f]{7,40}\b)",
    re.I)
TASK_FILES = ("bugreport.md", "instruction.md", "contract.md", "DETAILS.md", "api.md", "closure.md")


def scan_text():
    hits, n = [], 0
    roots = ["experiments/pipeline", "experiments/dose_response"]
    for root in roots:
        for f in glob.glob(f"{root}/**/*.md", recursive=True):
            if "/environment/src/" in f or "/node_modules/" in f:
                continue
            if os.path.basename(f) not in TASK_FILES:
                continue
            n += 1
            try:
                t = open(f, errors="replace").read()
            except Exception:
                continue
            for m in ORIGIN.finditer(t):
                hits.append((f, m.group(0)[:60]))
    return n, hits


def patch_shape():
    """Restoration looks like near-pure addition; a bug fix edits existing lines."""
    rows = []
    for f in glob.glob("experiments/dose_response/sweep_*/*/patches/gold.patch"):
        try:
            t = open(f, errors="replace").read()
        except Exception:
            continue
        add = rem = stub = 0
        for l in t.split("\n"):
            if l.startswith("+") and not l.startswith("+++"):
                add += 1
            elif l.startswith("-") and not l.startswith("---"):
                # A restoration patch must delete the stub it replaces. Our exciser leaves
                # panic("excised: Sym") and blank imports; removing those is the construction
                # working, not a logic edit. Counting them made every unit look edit-shaped.
                b = l[1:].strip()
                # Three stub forms the exciser emits: a panic, a blank import, and an
                # early return carrying an "excised:" comment. Missing the third made the
                # only three "edit-shaped" units look like real bug fixes.
                if ('excised:' in b) or (b.startswith('_ "') and b.endswith('"')):
                    stub += 1
                else:
                    rem += 1
        if add + rem + stub < 5:
            continue
        rows.append((os.path.basename(os.path.dirname(os.path.dirname(f))), add, rem, stub))
    return rows


def excision_record():
    have = miss = 0
    for d in glob.glob("experiments/pipeline/authored*/*/*/_author/"):
        if glob.glob(d + "closure.md") or glob.glob(d + "excised*"):
            have += 1
        else:
            miss += 1
    return have, miss


def main():
    print("=" * 68)
    print("SYNTHETIC PROVENANCE AUDIT")
    print("=" * 68)

    n, hits = scan_text()
    print(f"\n1. TEXT — scanned {n} task-facing files for issue/PR/CVE/commit references")
    if hits:
        print(f"   {len(hits)} REFERENCE(S) FOUND — these need inspection:")
        for f, m in hits[:15]:
            print(f"     {m:40s} {f[:70]}")
    else:
        print("   clean: no task-facing file references any upstream issue, PR, CVE or commit")

    rows = patch_shape()
    if rows:
        pure = [r for r in rows if r[2] <= 0.05 * r[1]]
        mixed = [r for r in rows if r[2] > 0.25 * r[1]]
        tot_a = sum(r[1] for r in rows); tot_r = sum(r[2] for r in rows)
        tot_s = sum(r[3] for r in rows)
        print(f"\n2. SHAPE — {len(rows)} gold patches")
        print(f"   additions {tot_a}, excision-stub removals {tot_s}, other deletions {tot_r}")
        print(f"   non-stub deletion ratio: {tot_r/max(1,tot_a):.2%}  (a real bug fix edits existing logic)")
        print(f"   near-pure restoration (<=5% deletions): {len(pure)}/{len(rows)} = {len(pure)/len(rows):.0%}")
        print(f"   edit-shaped (>25% deletions, would look like a real fix): {len(mixed)}")
        for u, a, r, st in sorted(mixed, key=lambda x: -x[2])[:8]:
            print(f"     {u:28s} +{a} -{r} (stubs {st})")

    have, miss = excision_record()
    print(f"\n3. RECORD — excision record present on {have} units, missing on {miss}")

    print("\n" + "=" * 68)
    verdict = "SYNTHETIC" if not hits and rows and len(pure) > 0.7 * len(rows) else "NEEDS REVIEW"
    print(f"VERDICT: {verdict}")
    print("Construction: working upstream code is deleted, and the solver must restore the")
    print("behaviour from a symptom report. The defect did not exist upstream, so no issue")
    print("thread, fix commit or discussion of it can exist in any training corpus.")
    print("=" * 68)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

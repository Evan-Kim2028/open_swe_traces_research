#!/usr/bin/env python3
"""How does a unit fail? Broadly, or on one unstated detail?

Reading two Grok traces showed both units failing 1 of 8 detail tests while implementing
the closure correctly — one on an exact Print format, one on panic-on-unsorted-input.
Neither is derivable from a bug report, so the unit is "hard at L0" because a convention
was unguessable, not because the engineering was hard.

A unit that fails 1-of-8 is low discrimination: it measures whether you read the spec.
A unit that fails most of its tests is measuring the closure. This counts which is which.
"""
import re, os, sys, glob, collections, json

RUN = re.compile(r"^=== RUN\s+(\S+)", re.M)
FAILED = re.compile(r"^--- FAIL:\s+(\S+)", re.M)
PASSED = re.compile(r"^--- PASS:\s+(\S+)", re.M)


def shape(path):
    try:
        s = open(path, errors="replace").read()
    except Exception:
        return None
    p, f = set(PASSED.findall(s)), set(FAILED.findall(s))
    if not (p or f):
        return None
    return len(p), len(f), sorted(f)


def main():
    rows = []
    for v in glob.glob("experiments/dose_response/jobs/*/*/verifier/test-stdout.txt"):
        tdir = os.path.dirname(os.path.dirname(v))
        name = os.path.basename(tdir).split("__")[0]
        if "-L" not in name:
            continue
        rt = os.path.join(tdir, "verifier", "reward.txt")
        try:
            reward = float(open(rt).read().strip() or 0)
        except Exception:
            continue
        if reward > 0:
            continue                      # only failures have a shape worth reading
        sh = shape(v)
        if not sh:
            continue
        np, nf, which = sh
        rows.append({"unit": name, "rung": name.rsplit("-L", 1)[-1][:1],
                     "passed": np, "failed": nf, "total": np + nf,
                     "which": which, "job": tdir.split(os.sep)[-2]})

    if not rows:
        print("no failing trials with parsable test output")
        return 1

    l0 = [r for r in rows if r["rung"] == "0"]
    print(f"failing trials with test output: {len(rows)}  (L0: {len(l0)})\n")
    for label, group in (("ALL RUNGS", rows), ("L0 ONLY", l0)):
        if not group:
            continue
        hist = collections.Counter(r["failed"] for r in group)
        one = sum(1 for r in group if r["failed"] == 1)
        print(f"{label}: {len(group)} failing trials")
        print(f"  failed exactly 1 test : {one:4d}  ({one/len(group):.0%})  <- low discrimination")
        print(f"  failed 2+             : {len(group)-one:4d}  ({1-one/len(group):.0%})")
        print("  distribution of failed-test counts: "
              + ", ".join(f"{k}:{v}" for k, v in sorted(hist.items())[:10]))
        print()

    # Which single tests account for the most one-test failures? Those are the
    # arbitrary commitments worth rewriting or dropping.
    single = collections.Counter()
    for r in rows:
        if r["failed"] == 1:
            single[(r["unit"], r["which"][0])] += 1
    print("units most often failing on ONE test (candidate arbitrary commitments):")
    for (unit, test), n in single.most_common(12):
        print(f"  {n:3d}x  {unit:22s} {test}")
    json.dump(rows, open("outputs/failure_shape.json", "w"), indent=1)
    print("\n-> outputs/failure_shape.json")
    return 0


if __name__ == "__main__":
    sys.exit(main())

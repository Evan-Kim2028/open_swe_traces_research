#!/usr/bin/env python3
"""Cross-reference single-test failures against the authors' own Inferable: annotations.

150 of 170 DETAILS.md files carry an `Inferable:` field per commitment — yes / doc /
partially / no — and nothing in the pipeline has ever read it. A test named
TestDetail06_* grades commitment 6, so a failure on it can be matched to the author's
own judgement of whether that commitment was derivable at all.

A unit failing ONLY on a commitment its own author marked `Inferable: no` is not measuring
engineering. It is measuring whether the solver guessed a convention nobody stated.
"""
import re, os, json, glob, collections

DETAIL_N = re.compile(r"TestDetail(\d+)", re.I)
INFER = re.compile(r"^\s*[-*]?\s*Inferable:\s*(\w+)", re.I | re.M)


def details_for(unit_base):
    """Find a DETAILS.md for this unit anywhere in the tree."""
    for pat in (f"experiments/**/{unit_base}/_author/DETAILS.md",
                f"experiments/**/{unit_base}/DETAILS.md",
                f"experiments/**/{unit_base}-L*/DETAILS.md"):
        for f in glob.glob(pat, recursive=True):
            return f
    return None


def inferable_list(path):
    """Ordered Inferable verdicts, one per commitment block."""
    try:
        txt = open(path, errors="replace").read()
    except Exception:
        return []
    return [m.group(1).lower() for m in INFER.finditer(txt)]


def main():
    rows = json.load(open("outputs/failure_shape.json"))
    singles = [r for r in rows if r["failed"] == 1]
    seen, out = set(), []
    for r in singles:
        base = r["unit"].rsplit("-L", 1)[0]
        key = (base, r["which"][0])
        if key in seen:
            continue
        seen.add(key)
        d = details_for(base)
        if not d:
            continue
        infs = inferable_list(d)
        m = DETAIL_N.search(r["which"][0])
        verdict = None
        if m and infs:
            i = int(m.group(1)) - 1
            if 0 <= i < len(infs):
                verdict = infs[i]
        out.append({"unit": r["unit"], "test": r["which"][0],
                    "inferable": verdict, "n_annotations": len(infs)})

    have = [o for o in out if o["inferable"]]
    print(f"distinct single-test failures      {len(seen)}")
    print(f"  with a DETAILS.md found          {len(out)}")
    print(f"  matched to an Inferable verdict  {len(have)}\n")
    if not have:
        print("no matches — test numbering may not align with DETAILS order")
        return 0
    c = collections.Counter(o["inferable"] for o in have)
    for k, v in c.most_common():
        flag = "  <- measures nothing" if k == "no" else ""
        print(f"  Inferable: {k:10s} {v:4d}  ({v/len(have):.0%}){flag}")
    print("\nunits failing solely on a commitment the author marked NOT inferable:")
    for o in have:
        if o["inferable"] == "no":
            print(f"  {o['unit']:24s} {o['test']}")
    json.dump(out, open("outputs/arbitrary_failures.json", "w"), indent=1)
    print("\n-> outputs/arbitrary_failures.json")
    return 0


def cli():
    raise SystemExit(main())


if __name__ == "__main__":
    cli()

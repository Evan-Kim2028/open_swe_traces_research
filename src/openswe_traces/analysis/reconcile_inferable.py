#!/usr/bin/env python3
"""Cross-check the audit's gap classification against the author's own `Inferable:` annotation.

Two independent judgements of the same question, made at opposite ends of the pipeline:

  the AUTHOR, writing DETAILS.md while looking at gold, marks each commitment
      Inferable: yes | doc | partially | no
  the AUDIT, reading the finished contract against the hidden suite, marks each gap
      DERIVABLE | COUNTER | ARBITRARY

They should agree: `Inferable: no` is the same claim as ARBITRARY. Where they disagree is the
interesting reading — either the author under-rated a commitment the repo actually implies, or the
audit missed that nothing implies it.
"""
import json
import re
from collections import Counter, defaultdict
from pathlib import Path

INF_RE = re.compile(r"Inferable:\s*([a-z]+)", re.IGNORECASE)


def author_labels() -> dict[str, Counter]:
    """unit stem -> Counter of the author's annotations"""
    out: dict[str, Counter] = defaultdict(Counter)
    for d in Path("/home/evan/Documents").glob("oswt-*/experiments/pipeline/authored*/*/*/_author/DETAILS.md"):
        stem = d.parents[1].name
        for m in INF_RE.finditer(d.read_text(errors="replace")):
            out[stem][m.group(1).lower()] += 1
    return out


def main() -> int:
    gap = Path("experiments/dose_response/audit/gap_read.jsonl")
    if not gap.is_file():
        print("no gap_read.jsonl yet")
        return 1
    authors = author_labels()
    rows = []
    for line in gap.read_text().splitlines():
        try:
            r = json.loads(line)
        except Exception:
            continue
        stem = r["unit"].rsplit("-L", 1)[0]
        for cand in (stem, *["-".join(stem.split("-")[i:]) for i in range(1, len(stem.split("-")))]):
            if cand in authors:
                a = authors[cand]
                arb = r.get("kinds", {}).get("ARBITRARY")
                rows.append({
                    "unit": r["unit"],
                    "author_no": a["no"], "author_partial": a["partially"],
                    "author_doc": a["doc"], "author_yes": a["yes"],
                    "audit_arbitrary": arb, "audit_missing": r["missing"],
                    "verdict": r.get("verdict"),
                })
                break
    if not rows:
        print("no units matched between the audit and the authored DETAILS files")
        return 1
    print(f"{'unit':32} {'author:no':>9} {'audit:arb':>9} {'missing':>7}  verdict")
    for r in sorted(rows, key=lambda r: -(r["author_no"] or 0)):
        arb = "—" if r["audit_arbitrary"] is None else r["audit_arbitrary"]
        print(f"{r['unit']:32} {r['author_no']:>9} {str(arb):>9} {r['audit_missing']:>7}  {r['verdict'] or '—'}")
    tot_no = sum(r["author_no"] for r in rows)
    print(f"\n{len(rows)} units matched · {tot_no} commitments the AUTHOR already marked `Inferable: no`")
    print("Those are unfair at every rung, including L5. They need the ASSERTION weakened, not the contract repaired.")
    return 0


def cli():
    raise SystemExit(main())


if __name__ == "__main__":
    cli()

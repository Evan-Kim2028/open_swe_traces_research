#!/usr/bin/env python3
"""Classify each missing commitment by whether skill can recover it. Discrimination, not difficulty.

A hard item that every model fails for the same non-reason has a difficulty number and no
measurement value. Three kinds, and only two are worth owning:

  ARBITRARY  an authorial choice nothing implies -- an exact error string, a type spelling,
             test scaffolding. No skill recovers it. Repairing the CONTRACT here manufactures
             a solvable item that separates nobody; the TEST is what should change.
  DERIVABLE  a real invariant the rest of the system implies. A careful solver gets there.
             Repair the contract; the unit was always sound.
  COUNTER    the obvious reading is wrong and the correct rule is discoverable. The best kind:
             it separates solvers that read from solvers that pattern-match. Preserve it.
"""
import json, sys
from pathlib import Path
from openswe_traces.ops.ask_composer import ask  # noqa: E402

PROMPT = """Classify each numbered requirement below by whether a skilled engineer, given the
repository and its conventions but NOT the test, could have derived it.

ARBITRARY  - an authorial choice nothing implies: an exact error-message literal, a specific
             type spelling, a punctuation or formatting choice, internal test scaffolding.
             Every solver fails it for the same non-reason.
DERIVABLE  - a real invariant the surrounding system implies (nil-safety, non-negativity,
             a documented format, consistency with a neighbouring function).
COUNTER    - the obvious reading is wrong, but the correct rule is discoverable in the repo
             or the spec. Separates solvers that read from solvers that pattern-match.

Reply with one line per item, exactly: <number>|<ARBITRARY|DERIVABLE|COUNTER>|<3-8 word reason>

{items}
"""


def main() -> int:
    rows = [json.loads(l) for l in open("experiments/dose_response/audit/gap_read.jsonl")]
    items = []
    for r in rows:
        for line in r["answer"].splitlines():
            line = line.strip()
            if line.startswith("MISSING"):
                items.append((r["unit"], line[8:].strip(": ")))
    out = []
    for i in range(0, len(items), 25):
        chunk = items[i:i + 25]
        numbered = "\n".join(f"{n + 1}. {t}" for n, (_, t) in enumerate(chunk))
        ans = ask(PROMPT.format(items=numbered), cache_key=f"classify/gapv1/{i}")
        by_n = {}
        for line in ans.splitlines():
            parts = [p.strip() for p in line.split("|")]
            if len(parts) >= 2 and parts[0].rstrip(".").isdigit():
                by_n[int(parts[0].rstrip("."))] = (parts[1].upper(), parts[2] if len(parts) > 2 else "")
        for n, (unit, text) in enumerate(chunk, 1):
            kind, why = by_n.get(n, ("UNKNOWN", ""))
            out.append({"unit": unit, "text": text, "kind": kind, "why": why})
    Path("experiments/dose_response/audit/gap_classified.jsonl").write_text(
        "\n".join(json.dumps(o) for o in out) + "\n")
    from collections import Counter
    c = Counter(o["kind"] for o in out)
    tot = sum(c.values())
    print(f"{tot} missing commitments classified\n")
    for k in ("ARBITRARY", "DERIVABLE", "COUNTER", "UNKNOWN"):
        if c[k]:
            print(f"  {k:10} {c[k]:>4}  {c[k]/tot:>5.0%}")
    # per-unit: a unit whose gaps are mostly arbitrary is a low-discrimination item
    from collections import defaultdict
    per = defaultdict(Counter)
    for o in out:
        per[o["unit"]][o["kind"]] += 1
    bad = [u for u, cc in per.items() if cc["ARBITRARY"] > cc["DERIVABLE"] + cc["COUNTER"]]
    print(f"\nunits whose gaps are MOSTLY arbitrary (low discrimination): {len(bad)} of {len(per)}")
    for u in sorted(bad):
        cc = per[u]
        print(f"  {u:30} arb={cc['ARBITRARY']} der={cc['DERIVABLE']} ctr={cc['COUNTER']}")
    return 0


def cli():
    raise SystemExit(main())


if __name__ == "__main__":
    cli()

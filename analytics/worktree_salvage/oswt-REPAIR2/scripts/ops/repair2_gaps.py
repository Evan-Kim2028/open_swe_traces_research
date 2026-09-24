#!/usr/bin/env python3
"""Normalise the repair2 gap table: one row per audited gap for the units this job owns.

Reads experiments/dose_response/audit/gap_read.jsonl (the audit) and
gap_classified.jsonl (kinds for 24 units; rows align with MISSING lines by
position). Units/gaps still unclassified are sent to Composer with the same
ARBITRARY/DERIVABLE/COUNTER rubric used by classify_gaps.py, extended to cover
CONFLICT rows as well.

Output: outputs/repair2/gaps.jsonl  {unit, gtype, kind, text, why}
Resume-safe: classified results cache under outputs/composer_cache/.
"""
import json
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from ask_composer import ask_many  # noqa: E402

AUDIT = Path("experiments/dose_response/audit")
OUT = Path("outputs/repair2/gaps.jsonl")

# Units this job owns: everything audited except the goa cohort (REPAIRLOOP),
# helm-repindex (REPAIRLOOP), bbolt page-L2/flshared-L2 (REPAIRLOOP, not
# audited), and the two clean units endpoints-L2 / ipqueue-L2.
GOA = {"dupexpr-L2", "exprhash-L2", "httpclienterr-L2", "httpencoding-L2",
       "httpmux-L2", "mappedattr-L2", "namescope-L2", "sampler-L2",
       "svcerror-L2", "goa-jsonrpcwire-L2"}
CLEAN = {"endpoints-L2", "ipqueue-L2"}

LINE_RE = re.compile(r"^(MISSING|CONFLICT)\s*[:|]\s*(.*)$")
PIPE_RE = re.compile(r"^(MISSING|CONFLICT)\|(ARBITRARY|DERIVABLE|COUNTER)\|(.*)$")

CLASSIFY_PROMPT = """Classify each numbered requirement below by whether a skilled engineer, given the
repository and its conventions but NOT the test, could have derived it.

ARBITRARY  - an authorial choice nothing implies: an exact error-message literal, a specific
             type spelling, a punctuation or formatting choice, internal test scaffolding.
             Every solver fails it for the same non-reason.
DERIVABLE  - a real invariant the surrounding system implies (nil-safety, non-negativity,
             a documented format, consistency with a neighbouring function).
COUNTER    - the obvious reading is wrong, but the correct rule is discoverable in the repo
             or the spec. Separates solvers that read from solvers that pattern-match.

Items marked MISSING are behaviours the suite asserts but the contract omits.
Items marked CONFLICT are claims where the contract states one thing and the
suite asserts the opposite (the item text describes both sides).

Reply with one line per item, exactly: <number>|<ARBITRARY|DERIVABLE|COUNTER>|<3-8 word reason>

{items}
"""


def extract(unit: str, answer: str) -> list[dict]:
    """Pull MISSING/CONFLICT gap lines out of a gap_read answer."""
    gaps = []
    for line in answer.splitlines():
        line = line.strip().lstrip("-*").strip()
        m = PIPE_RE.match(line)
        if m:
            gaps.append({"unit": unit, "gtype": m.group(1).lower(),
                         "kind": m.group(2), "text": m.group(3).strip(),
                         "why": ""})
            continue
        m = LINE_RE.match(line)
        if m and m.group(2).strip():
            gaps.append({"unit": unit, "gtype": m.group(1).lower(),
                         "kind": "", "text": m.group(2).strip(), "why": ""})
    return gaps


def main() -> int:
    read_rows = [json.loads(l) for l in (AUDIT / "gap_read.jsonl").read_text().splitlines()]
    cls: dict[str, list[dict]] = {}
    for l in (AUDIT / "gap_classified.jsonl").read_text().splitlines():
        d = json.loads(l)
        cls.setdefault(d["unit"], []).append(d)

    all_gaps: list[dict] = []
    for r in read_rows:
        unit = r["unit"]
        if unit in GOA or unit in CLEAN:
            continue
        gaps = extract(unit, r["answer"])
        # join pre-classified kinds by position over the MISSING rows
        miss_i = 0
        for g in gaps:
            if g["gtype"] == "missing" and not g["kind"]:
                rows = cls.get(unit, [])
                if miss_i < len(rows):
                    g["kind"] = rows[miss_i]["kind"]
                    g["why"] = rows[miss_i].get("why", "")
                miss_i += 1
        all_gaps.extend(gaps)

    todo = [(i, g) for i, g in enumerate(all_gaps)
            if g["kind"] not in ("ARBITRARY", "DERIVABLE", "COUNTER")]
    if todo:
        by_unit: dict[str, list[int]] = {}
        for i, g in todo:
            by_unit.setdefault(g["unit"], []).append(i)
        items = []
        for unit, idxs in sorted(by_unit.items()):
            numbered = "\n".join(
                f"{n + 1}. {all_gaps[i]['gtype'].upper()}: {all_gaps[i]['text']}"
                for n, i in enumerate(idxs))
            items.append((f"repair2/classify/{unit}",
                          CLASSIFY_PROMPT.format(items=numbered)))
        answers = ask_many(items, workers=3)
        for (unit, idxs), (key, prompt) in zip(sorted(by_unit.items()), items):
            ans = answers.get(key, "")
            by_n = {}
            for line in ans.splitlines():
                parts = [p.strip() for p in line.split("|")]
                if len(parts) >= 2 and parts[0].rstrip(".").isdigit():
                    by_n[int(parts[0].rstrip("."))] = (
                        parts[1].upper(), parts[2] if len(parts) > 2 else "")
            for n, i in enumerate(idxs, 1):
                kind, why = by_n.get(n, ("UNKNOWN", ""))
                all_gaps[i]["kind"] = kind
                all_gaps[i]["why"] = why
            print(f"classified {unit}: {ans.count(chr(10)) + 1} lines")

    OUT.parent.mkdir(parents=True, exist_ok=True)
    with OUT.open("w") as fh:
        for g in all_gaps:
            fh.write(json.dumps(g) + "\n")

    from collections import Counter, defaultdict
    per = defaultdict(Counter)
    for g in all_gaps:
        per[g["unit"]][g["kind"]] += 1
    for u in sorted(per):
        print(f"{u:36} {dict(per[u])}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

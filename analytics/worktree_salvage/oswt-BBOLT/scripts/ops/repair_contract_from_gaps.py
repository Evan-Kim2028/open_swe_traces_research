#!/usr/bin/env python3
"""Add DERIVABLE/COUNTER gap rows to an L2 contract. Never writes ARBITRARY literals.

Reads a gap_read.jsonl row, asks Composer for a prose patch (new paragraphs +
coverage-table rows), applies it in front of the Bug report section, and
refuses the patch if the ungrounded-literal count rose.
"""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from ask_composer import ask  # noqa: E402
from task_lint import EXAMPLE_LIT_RE, hidden_sources  # noqa: E402

PROMPT = """You are repairing an L2 task CONTRACT so it states every behaviour
the hidden suite already grades. Do not invent new behaviour. Do not copy
error-message / format-string / type-name literals that the suite asserts
unless that literal already appears in the CURRENT CONTRACT.

Rules:
- For each DERIVABLE or COUNTER gap below, add one short prose sentence (and a
  coverage-table row if a matching TestDetail name is obvious).
- For ARBITRARY gaps: do NOT add them. Leave a one-line note listing them as
  test defects; the assertion will be weakened separately.
- Do not add quoted tokens that contain digits, slashes, or dots unless they
  already appear in the current contract.
- Keep the existing Bug report section byte-for-byte.
- Keep the existing heading and the L0 bug-report tail.
- Return the FULL new instruction.md, nothing else. No markdown fence.

=== CURRENT CONTRACT ===
{contract}

=== GAPS TO ADD (DERIVABLE/COUNTER only) ===
{gaps}
"""


def literals(text: str) -> set[str]:
    return {m.group(1) for m in EXAMPLE_LIT_RE.finditer(text) if len(m.group(1)) >= 4}


def grounded_frac(text: str, blob: str) -> tuple[int, int]:
    lits = literals(text)
    if not lits:
        return 0, 0
    return len(lits), sum(1 for l in lits if l in blob)


def apply_repair(unit: Path, gaps: list[str], cache_key: str) -> dict:
    instr = unit / "instruction.md"
    old = instr.read_text()
    blob = "\n".join(hidden_sources(unit).values())
    n_old, g_old = grounded_frac(old, blob)
    keep = [g for g in gaps if "|ARBITRARY|" not in g]
    if not keep:
        return {"unit": unit.name, "status": "no-derivable-gaps"}
    ans = ask(PROMPT.format(contract=old, gaps="\n".join(keep)), cache_key=cache_key)
    if "Bug report" not in ans or "# Contract" not in ans:
        return {"unit": unit.name, "status": "bad-shape", "answer_head": ans[:200]}
    n_new, g_new = grounded_frac(ans, blob)
    if n_old and n_new > n_old:
        return {
            "unit": unit.name, "status": "literal-rise",
            "old_lits": n_old, "new_lits": n_new, "old_grounded": g_old, "new_grounded": g_new,
        }
    if n_new and n_old and g_new / n_new < g_old / n_old - 0.01:
        return {
            "unit": unit.name, "status": "grounded-fall",
            "old_lits": n_old, "new_lits": n_new, "old_grounded": g_old, "new_grounded": g_new,
        }
    bak = unit / "instruction.md.pre_gap_repair"
    if not bak.is_file():
        bak.write_text(old)
    instr.write_text(ans if ans.endswith("\n") else ans + "\n")
    return {
        "unit": unit.name, "status": "patched",
        "old_lits": n_old, "new_lits": n_new, "old_grounded": g_old, "new_grounded": g_new,
        "n_added": len(keep),
    }


def main(argv: list[str]) -> int:
    import argparse
    ap = argparse.ArgumentParser()
    ap.add_argument("gap_path", type=Path)
    ap.add_argument("batch", type=Path)
    ap.add_argument("out", type=Path, nargs="?", default=Path("outputs/bbolt_contract_repair.jsonl"))
    ap.add_argument("--units", nargs="*", default=None)
    args = ap.parse_args(argv)
    gap_path, batch, out = args.gap_path, args.batch, args.out
    rows = {}
    for line in gap_path.read_text().splitlines():
        d = json.loads(line)
        rows[d["unit"]] = d
    done = set()
    if out.is_file():
        for line in out.read_text().splitlines():
            try:
                done.add(json.loads(line)["unit"])
            except Exception:
                pass
    out.parent.mkdir(parents=True, exist_ok=True)
    wanted = set(args.units) if args.units else None
    with out.open("a") as fh:
        for u in sorted(batch.iterdir()):
            if not u.name.endswith("-L2") or u.name in done:
                continue
            if wanted is not None and u.name not in wanted:
                continue
            g = rows.get(u.name)
            if not g:
                continue
            gaps = [ln.strip() for ln in g.get("answer", "").splitlines()
                    if ln.strip().startswith(("MISSING", "CONFLICT"))]
            if not gaps:
                fh.write(json.dumps({"unit": u.name, "status": "clean"}) + "\n")
                fh.flush()
                print(f"{u.name:24} clean", flush=True)
                continue
            r = apply_repair(u, gaps, cache_key=f"repair/{u.name}")
            fh.write(json.dumps(r) + "\n")
            fh.flush()
            print(f"{u.name:24} {r['status']}", flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))

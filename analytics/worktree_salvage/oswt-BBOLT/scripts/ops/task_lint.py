#!/usr/bin/env python3
"""Synthetic-task linter: the DETERMINISTIC pre-trial gate (gate 1 of 5).

See `analytics/research/verifier_rules.md`, "Verifier design: the two defect families".
This file covers only what is mechanically checkable. It CANNOT detect the contract family
(prose contradicting an assertion) -- that needs `scripts/shadow_gate.py` -- and it cannot
detect the packaging family (suite not covering the excision) -- that needs
`scripts/cg_coverage.py`. Do not extend this file to chase either; both were tried here as
text heuristics and both scored at the base rate.

A trial costs ~2.2M tokens. Every defect below was diagnosable from the task directory
alone, and each one cost between 3 and 12 trials to discover the expensive way.

Measured against 119 L2 units with known outcomes (47 non-flippers, 72 flippers, so a
random block is 39% precise). Only rules that BEAT that base rate block:

    DIRECTION  100% precise (n=1)      SCRUB   60% (n=5)
    B10        proven mechanism        A12/PACKAGING  definitional
    B7          50% (n=14) -> warn
    COVERAGE    40% (n=48) -> info, indistinguishable from the base rate
    UNGROUNDED  40% (n=84) -> info, indistinguishable from the base rate

That last pair is the finding: **counting rows and matching literals does not predict
whether a contract works.** The defects that decide a unit are semantic contradictions
between a claim and an assertion, and catching those needs a model reading both, not a
regex. That is A13's job, and A13 can run on a free-tier model -- which is what makes a
cheap pre-trial gate possible at all.

Checks (all static, no model calls, no docker):
  B10  hidden suite asserts a literal digest of something the solver serialises
       -> unsolvable at every rung, including L5 (helm-depresolver, 0/3 at L5)
  UNGROUNDED  a worked example uses a literal that appears nowhere in the hidden suite
       -> an invented example can be wrong about the library's own semantics. The
          reconciler's `Get(">=1.0.0") -> 2.0.0-alpha` was invented and wrong, and the
          solver implemented it faithfully: 0/3.
  COVERAGE  far fewer contract commitments than hidden assertions
       -> incompleteness is as fatal as contradiction (repindex: 18 rows over 50
          assertions was 0/3; 29 rows was 2/3)
  DIRECTION  pre-release/sort ordering stated as a direction rather than a pairwise rule
       -> "prereleases sort before releases" is backwards for descending order; cost 0/3
          twice, once by my hand and once by the generator
  SCRUB  an unresolved symbol-scrub placeholder ("the call") left in the prose
  B7  symbol, file or line names leaked into an L2 contract
  B6  instruction floor: an L0 bug report with no reproduce command or no symptom
  A12  gold patch touches tests

Exit code 1 if any BLOCK-severity finding is present.

Usage: task_lint.py <unit dir> [...]   |   task_lint.py --batch <dir of unit dirs>
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

DIGEST_LIT_RE = re.compile(r"[\"'`](sha(?:256|512|1):[0-9a-f]{32,128})[\"'`]")
COMPUTED_RE = re.compile(r"sha256\.(?:Sum256|New)|hex\.EncodeToString")
ASSERT_RE = re.compile(r"t\.(?:Fatalf|Errorf|Fatal|Error)\(")
TESTFUNC_RE = re.compile(r"^func (Test\w+)\(", re.MULTILINE)
ROW_RE = re.compile(r"^\s*\|", re.MULTILINE)
BULLET_RE = re.compile(r"^\s*[-*]\s+\S", re.MULTILINE)
EXAMPLE_LIT_RE = re.compile(r"[\"'`]([A-Za-z0-9][\w.\-+/:]{2,60})[\"'`]")
DIRECTION_RE = re.compile(
    r"pre-?releases?\s+sort\s+(?:before|above|after|below)\s+releases?"
    r"|releases?\s+sort\s+(?:before|above)\s+pre-?releases?",
    re.IGNORECASE,
)
SCRUB_RE = re.compile(r"\bthe call\b", re.IGNORECASE)
SYMBOL_RE = re.compile(r"\b[a-z][A-Za-z0-9]*\.[A-Z][A-Za-z0-9]{2,}\b|\b[A-Z][a-z]+[A-Z][A-Za-z0-9]{2,}\(")
FILE_RE = re.compile(r"\b[\w/]+\.(?:go|py|ts|js|rs|java)\b")
LINE_RE = re.compile(r"\bline\s+\d+\b", re.IGNORECASE)
REPRO_RE = re.compile(r"go test|pytest|npm test|cargo test|tests/test\.sh|make test", re.IGNORECASE)


def hidden_sources(unit: Path) -> dict[str, str]:
    d = unit / "tests" / "hidden"
    if not d.is_dir():
        return {}
    return {str(f.relative_to(d)): f.read_text(errors="replace") for f in d.rglob("*") if f.is_file()}


def literal_budget(unit: Path) -> tuple[int, int]:
    """(literals in the contract, how many appear in the hidden suite).

    Contract repair must add PROSE, not LITERALS. A3 requires that a patch special-casing
    the contract's examples fails the suite, so every ungrounded literal a repair adds is
    fresh cheat surface. Measured on helm-repindex: the repair that went 0/3 -> 2/3 grew
    445 -> 1251 words while literals FELL 22 -> 15 and the grounded fraction rose 32% -> 80%.
    The generated contract that scored 0/3 went the other way: 46 literals, 24 grounded.
    """
    instr = unit / "instruction.md"
    if not instr.is_file():
        return (0, 0)
    text = instr.read_text(errors="replace")
    lits = {m.group(1) for m in EXAMPLE_LIT_RE.finditer(text) if len(m.group(1)) >= 4}
    blob = "\n".join(hidden_sources(unit).values())
    return (len(lits), sum(1 for l in lits if l in blob))


def lint(unit: Path) -> list[tuple[str, str, str]]:
    """(severity, code, message)"""
    out: list[tuple[str, str, str]] = []
    instr = (unit / "instruction.md")
    text = instr.read_text(errors="replace") if instr.is_file() else ""
    hidden = hidden_sources(unit)
    blob = "\n".join(hidden.values())
    level = None
    aff = unit / "affordance.json"
    m = re.search(r"-L(\d)", unit.name)
    if m:
        level = int(m.group(1))
    elif aff.is_file():
        try:
            level = json.loads(aff.read_text()).get("level")
        except Exception:
            level = None

    # --- B10: literal digest oracle -------------------------------------
    for name, src in hidden.items():
        for dm in DIGEST_LIT_RE.finditer(src):
            window = src[max(0, dm.start() - 600) : dm.start() + 600]
            if not COMPUTED_RE.search(window):
                out.append(("BLOCK", "B10",
                            f"{name}: literal digest oracle {dm.group(1)[:24]}… — unsolvable at every rung"))
                break

    # --- coverage: commitments vs assertions ----------------------------
    n_assert = len(ASSERT_RE.findall(blob))
    n_rows = len(ROW_RE.findall(text)) + len(BULLET_RE.findall(text))
    if level is not None and level >= 2 and n_assert and n_rows < n_assert * 0.4:
        out.append(("INFO", "COVERAGE",
                    f"{n_rows} contract commitments for {n_assert} hidden assertions "
                    f"({n_rows / n_assert:.0%}) — incompleteness is as fatal as contradiction"))

    # --- ungrounded worked-example literals -----------------------------
    if level is not None and level >= 2 and blob:
        ungrounded = []
        for em in EXAMPLE_LIT_RE.finditer(text):
            lit = em.group(1)
            if len(lit) < 4 or lit in {"true", "false", "nil", "null", "none"}:
                continue
            if not re.search(r"\d", lit) and "/" not in lit and "." not in lit:
                continue
            if lit not in blob:
                ungrounded.append(lit)
        uniq = sorted(set(ungrounded))
        if uniq:
            out.append(("INFO", "UNGROUNDED",
                        f"{len(uniq)} example literal(s) appear nowhere in the hidden suite: "
                        f"{uniq[:6]} — an invented example can be wrong and is implemented faithfully"))

    # --- ordering stated as a direction ---------------------------------
    if DIRECTION_RE.search(text):
        out.append(("BLOCK", "DIRECTION",
                    "pre-release ordering stated as a direction; state the pairwise rule against "
                    "the release it belongs to and say whether build metadata is ignored"))

    # --- unresolved scrub placeholder -----------------------------------
    if SCRUB_RE.search(text):
        out.append(("BLOCK", "SCRUB", 'unresolved symbol-scrub placeholder "the call" in the prose'))

    # --- B7 ceiling (L2 only) -------------------------------------------
    if level == 2:
        leaks = set(FILE_RE.findall(text)) | set(LINE_RE.findall(text))
        if leaks:
            out.append(("WARN", "B7", f"file/line names in an L2 contract: {sorted(leaks)[:5]}"))  # 50% precision

    # --- B6 floor (L0 only) ---------------------------------------------
    if level == 0:
        if not REPRO_RE.search(text):
            out.append(("BLOCK", "B6", "L0 bug report has no reproduce command"))
        if len(text.split()) < 40:
            out.append(("WARN", "B6", f"L0 bug report is only {len(text.split())} words"))

    # --- A12: gold must not touch tests ---------------------------------
    gold = unit / "tests" / "gold.patch"
    if gold.is_file():
        g = gold.read_text(errors="replace")
        touched = re.findall(r"^\+\+\+ b/(.+)$", g, re.MULTILINE)
        bad = [t for t in touched if t.endswith("_test.go") or t.startswith("tests/")]
        if bad:
            out.append(("BLOCK", "A12", f"gold patch touches tests: {bad[:3]}"))

    # --- A3 leak surface: ungrounded literals are cheat surface ----------
    if level is not None and level >= 2 and blob:
        n_lit, n_grounded = literal_budget(unit)
        if n_lit and n_grounded / n_lit < 0.5 and n_lit >= 12:
            out.append(("WARN", "A3-SURFACE",
                        f"{n_lit} contract literals, only {n_grounded} grounded in the suite "
                        f"({n_grounded / n_lit:.0%}) — ungrounded literals are fresh cheat surface; "
                        "repair should add prose, not literals"))

    # --- TOO-EASY: drop before buying an L0 screen, for zero tokens ------
    # Measured on 189 units with a Composer L0 verdict (33% were solved = too easy).
    # Easy units have HALF the assertions of hard ones (21.7 vs 40.9). Note this predicts
    # "easy at L0"; it does NOT predict whether a hard unit flips at L2 -- assertion count
    # was refuted for that. Different questions.
    #   assertions<=8 AND testfns<=5 -> 9 dropped, 9 truly easy, 0 hard lost (100%)
    #   assertions<=12               -> 27 dropped, 23 truly easy (85%), 4 hard lost, 14% of screens saved
    if n_assert:
        n_testfn = len(re.findall(r"^func (Test\w+)\(", blob, re.MULTILINE))
        if n_assert <= 8 and n_testfn <= 5:
            out.append(("BLOCK", "TOO-EASY",
                        f"{n_assert} assertions across {n_testfn} test functions — 9 of 9 such units were "
                        "solved at L0; drop without buying a screen"))
        elif n_assert <= 12:
            out.append(("WARN", "TOO-EASY",
                        f"{n_assert} assertions — 85% of units this small were solved at L0"))

    if not hidden:
        out.append(("BLOCK", "PACKAGING", "no hidden suite in tests/hidden"))
    return out


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("units", nargs="*", type=Path)
    ap.add_argument("--batch", type=Path, default=None)
    ap.add_argument("--quiet", action="store_true", help="only print units with findings")
    args = ap.parse_args(argv)
    units = list(args.units)
    if args.batch:
        units += [p for p in sorted(args.batch.iterdir()) if p.is_dir() and not p.name.startswith("_")]
    blocked = 0
    for u in units:
        findings = lint(u)
        blocks = [f for f in findings if f[0] == "BLOCK"]
        if blocks:
            blocked += 1
        if findings or not args.quiet:
            mark = "BLOCK" if blocks else ("warn " if findings else "ok   ")
            print(f"{mark} {u.name}")
            for sev, code, msg in findings:
                print(f"       [{sev}] {code}: {msg}")
    print(f"\n{blocked} of {len(units)} units blocked")
    return 1 if blocked else 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))

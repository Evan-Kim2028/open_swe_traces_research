"""REPAIR2 driver: audit-driven repair of the double-failure units outside
REPAIRLOOP's scope.

Unlike ``repair_loop`` (which discovers defects by shadowing each unit), this
driver consumes the completed audit — ``experiments/dose_response/audit/
gap_read.jsonl`` + ``gap_classified.jsonl`` normalised to
``outputs/repair2/gaps.jsonl`` — and routes every gap by kind:

    ARBITRARY  test over-specifies   -> weaken the assertion (handled outside
                                       this module; see scripts/ops/weaken_*)
    DERIVABLE  contract omits a repo-derivable rule -> add the row
    COUNTER    contract omits a counter-intuitive rule -> add the row and flag
               the unit high-value

Verification per unit (a repair is not done until all three pass):
    re-audit   contract_gap_read prompt on the STAGED unit with a content-keyed
               cache — the gap list must shrink
    A3-SURFACE task_lint literal budget — count must not rise, grounded
               fraction must not fall
    preflight  bare FAILs, gold PASSes, a FRESH cheat generated from the
               repaired contract FAILs
"""
from __future__ import annotations

import hashlib
import json
import re
import sys
import time
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.gate import repair_loop, shadow
from openswe_traces.synth import reconcile

R2_DIR = ROOT / "outputs" / "repair2"
GAPS_PATH = R2_DIR / "gaps.jsonl"
RESULTS_PATH = R2_DIR / "results.jsonl"
LOG_PATH = ROOT / "outputs" / "REPAIR2.log"
AUDIT_DIR = ROOT / "experiments" / "dose_response" / "audit"

MAX_TOTAL_CHARS = 110_000
MAX_CONTRACT_CHARS = 48_000
MAX_TEST_CHARS = 55_000


def log(msg: str) -> None:
    LOG_PATH.parent.mkdir(parents=True, exist_ok=True)
    line = f"{time.strftime('%H:%M:%S')} {msg}"
    print(line, flush=True)
    with LOG_PATH.open("a") as f:
        f.write(line + "\n")


# --------------------------------------------------------------------------
# gap table
# --------------------------------------------------------------------------


def load_gaps(path: Path = GAPS_PATH) -> dict[str, list[dict]]:
    per: dict[str, list[dict]] = {}
    for line in path.read_text().splitlines():
        d = json.loads(line)
        per.setdefault(d["unit"], []).append(d)
    return per


# --------------------------------------------------------------------------
# contract repair (DERIVABLE + COUNTER gaps)
# --------------------------------------------------------------------------


def _hidden_blob(unit_dir: Path) -> str:
    return "\n".join(repair_loop.load_hidden(unit_dir).values())


def repair_prompt(contract_head: str, gaps: list[dict], hidden_blob: str,
                  gold: str, denylist: set[str], lit_note: str,
                  prev_violation: str = "") -> str:
    items = []
    for i, g in enumerate(gaps, 1):
        action = ("CORRECT the contract's claim — it states this wrongly"
                  if g["gtype"] == "conflict" else "ADD this missing commitment")
        items.append(f"{i}. [{g['kind']}] {action}: {g['text']}")
    deny = ", ".join(sorted(denylist)[:40]) or "(none)"
    retry = ""
    if prev_violation:
        retry = ("\nYour previous revision was REJECTED: " + prev_violation +
                 "\nFix exactly that and return the full revised contract again.\n")
    return f"""You are repairing the L2 CONTRACT of a synthetic SWE task. An audit read this
contract against the hidden test suite and found the commitments below missing
or mis-stated. Add or correct exactly those commitments — and change NOTHING
else. This is a surgical edit, not a rewrite: every existing sentence, worked
example, and coverage row stays byte-for-byte unless a listed commitment
requires correcting it.

Rules:
- Behavioural prose only. Do NOT name functions, types, files, packages, or
  line numbers. Forbidden names: {deny}
- Hidden TEST names must not appear in the contract (the suite stays hidden).
- For each MISSING commitment: add one prose sentence where it belongs AND
  one row to the coverage table. For each CONFLICT: correct the offending
  sentence, and fix its table row if present.
- Concrete literals (versions, URLs, digests, wire strings, JSON keys) ARE
  allowed — but ONLY ones that appear verbatim in the hidden test sources
  below. Never invent an example.
- Never state ordering as a class direction ("X sorts before/after Y").
  State the pairwise rule against concrete versions, and whether build
  metadata participates.
- PRESERVE the document structure exactly: title, prose sections, worked
  examples, the coverage table's existing format and rows. Append rows;
  do not renumber, reformat, merge or delete existing rows.
- LITERAL BUDGET: the revised contract must not contain more distinct
  quoted/backticked literals than the current one. Your additions may quote
  only literals found verbatim in the test sources; if that would exceed the
  cap, drop exactly as many literals that appear in NO test source as you
  need — and nothing else.
{lit_note}

COMMITMENTS TO REPAIR (each is a behaviour the suite asserts; the audit's
kind is advisory — state the behaviour, not the label):
{chr(10).join(items)}

GOLD PATCH (truth, for getting commitments right — never for naming):
```diff
{repair_loop._truncate(gold, 18_000)}
```

HIDDEN TEST SOURCES (the suite that grades this task — read what it asserts;
only literals appearing verbatim here may be quoted):
{repair_loop._truncate(hidden_blob, MAX_TEST_CHARS)}

CURRENT CONTRACT (return the FULL revised contract — head only, up to but
not including the "Reproduce with:" / bug-report tail):
{contract_head}
{retry}
Answer with the revised contract markdown ONLY — no JSON, no commentary."""


def repair_contract(unit_dir: Path, gaps: list[dict], api_key: str,
                    tag: str) -> repair_loop.RepairOutcome:
    """Draft a repaired contract head for DERIVABLE/COUNTER gaps with the
    same mechanical guards as repair_loop.repair (literal budget, no test
    names, no forbidden symbols, no directional ordering)."""
    instr_path = unit_dir / "instruction.md"
    instruction = instr_path.read_text(errors="replace")
    head, tail = repair_loop.contract_head_tail(instruction)
    hidden = repair_loop.load_hidden(unit_dir)
    hidden_blob = "\n".join(hidden.values())
    hidden_funcs = reconcile.parse_hidden_funcs(hidden)
    hidden_names = {fn.name for fn in hidden_funcs}
    gold = repair_loop.gold_text(unit_dir)
    denylist = reconcile.gold_denylist(gold)

    cur_lits = repair_loop.literal_set(instruction)
    lit_note = (
        f"  Current inventory: {len(cur_lits)} literals. Grounded in the test\n"
        f"  sources (safe to keep): "
        f"{sorted(l for l in cur_lits if l in hidden_blob)}\n"
        f"  Absent from every test source: "
        f"{sorted(l for l in cur_lits if l not in hidden_blob)}\n"
        f"  HARD CAP: at most {len(cur_lits)} literals in the revision. If your\n"
        f"  additions would exceed it, make room by REMOVING THE QUOTING\n"
        f"  (backticks) around an ungrounded literal — the word stays, it just\n"
        f"  stops counting. Never delete content to make room.")

    spent_requests = spent_tokens = 0
    model_used = ""
    violation_msg = ""
    for attempt in range(repair_loop.MAX_REPAIR_TRIES):
        prompt = repair_loop._truncate(
            repair_prompt(head, gaps, hidden_blob, gold, denylist,
                          lit_note, violation_msg),
            MAX_TOTAL_CHARS)
        res = repair_loop.llm_call(prompt, api_key, f"{tag}#try{attempt}",
                                   repair_loop.JUDGE_MODELS,
                                   repair_loop.MAX_REQUESTS)
        spent_requests += res.requests
        spent_tokens += res.tokens
        model_used = res.model
        if res.text is None:
            return repair_loop.RepairOutcome(
                False, "", [], model_used, spent_requests, spent_tokens,
                res.error or "repair call failed")
        new_head = res.text.strip()
        fm = shadow.FENCE_RE.search(new_head)
        if fm and new_head.startswith("```"):
            new_head = fm.group(1).strip()
        # drop model commentary emitted before the contract proper: if the
        # original head's first line survives, cut everything above it; else
        # cut to the first markdown heading
        first_line = head.splitlines()[0].strip() if head.splitlines() else ""
        if first_line and first_line in new_head:
            new_head = new_head[new_head.index(first_line):]
        elif (hm := re.search(r"(?m)^#\s", new_head)):
            new_head = new_head[hm.start():]
        _, own_tail = repair_loop.contract_head_tail(new_head)
        if own_tail:
            new_head, _ = new_head.split(own_tail, 1)
            new_head = new_head.rstrip() + "\n"
        new_head, ord_fixed = reconcile.rewrite_directional_ordering(
            new_head, hidden_blob)
        new_head, _leaks = reconcile.scrub_b7(new_head, denylist)
        new_instruction = new_head + tail
        violations = repair_loop.check_repair(
            new_head, instruction, new_instruction, [], hidden_names,
            hidden_blob)
        # every literal the repair introduces must be grounded in the hidden
        # suite — a new ungrounded literal is fresh cheat surface (A3).
        # Pre-existing ungrounded literals are grandfathered; the budget
        # check above governs the count.
        introduced = (repair_loop.literal_set(new_instruction)
                      - repair_loop.literal_set(instruction))
        bad = sorted(l for l in introduced if l not in hidden_blob)
        if bad:
            violations.append(
                f"new ungrounded literals introduced: {bad[:8]} — quote only "
                f"literals that appear verbatim in the test sources")
        if not violations:
            instr_path.write_text(new_instruction)
            return repair_loop.RepairOutcome(
                True, new_head, [], model_used, spent_requests, spent_tokens,
                "ordering rewritten pairwise" if ord_fixed else "")
        violation_msg = "; ".join(violations)
        log(f"  {unit_dir.name}: repair try {attempt} rejected: "
            f"{violation_msg[:220]}")
    return repair_loop.RepairOutcome(False, new_head, violations, model_used,
                                     spent_requests, spent_tokens,
                                     violation_msg)


# --------------------------------------------------------------------------
# re-audit: same prompt as contract_gap_read.py, content-keyed cache
# --------------------------------------------------------------------------

GAPREAD_PROMPT = """You are auditing a task specification used to grade a coding agent.

Below is the CONTRACT, the prose the agent is given, and the HIDDEN TEST SUITE that grades it.
The agent never sees the suite. If the suite asserts a behaviour the contract does not state,
the task is unfair and unsolvable; if the contract states something the suite contradicts, the
agent is actively misled.

For every behaviour the suite grades, decide where its answer LIVES — because a task is fair at
a rung only if a competent engineer holding that rung's information would PRODUCE the graded
behaviour, not merely could guess it.

List, one per line, in this exact format:

  MISSING|<ARBITRARY|DERIVABLE|COUNTER>|<behaviour the suite asserts that the contract omits>
  CONFLICT|<ARBITRARY|DERIVABLE|COUNTER>|<a contract claim the suite contradicts, both sides>

The middle field says where the answer lives:

  ARBITRARY  nowhere. An authorial choice nothing implies — an exact error-message literal, a
             specific type spelling, punctuation, internal test scaffolding. No skill recovers it;
             every solver fails it for the same non-reason. The TEST is the defect here, not the
             contract: writing the literal into the prose makes the unit pass and measure nothing.
  DERIVABLE  in the repository. A real invariant the surrounding code implies — nil-safety,
             non-negativity, a documented format, consistency with a neighbouring function. A
             careful engineer gets there from what is in front of them.
  COUNTER    in the repository, but the obvious reading is wrong. The best kind: it separates
             solvers that read the code from solvers that pattern-match their training.

Describe behaviour, not symbols. Be specific and terse. If there is nothing, reply exactly NONE.

=== CONTRACT ===
{contract}

=== HIDDEN SUITE ===
{tests}
"""


def reaudit(unit_dir: Path) -> dict:
    """Run the contract-gap read on a (staged) unit dir. Cache key includes
    the sha of instruction.md + hidden suite, so a repaired unit is always
    read fresh and never collides with the original audit's cache."""
    sys.path.insert(0, str(ROOT / "scripts" / "ops"))
    from ask_composer import ask  # noqa: E402

    instr = (unit_dir / "instruction.md").read_text(errors="replace")
    hd = unit_dir / "tests" / "hidden"
    tests = "\n".join(f.read_text(errors="replace")
                      for f in sorted(hd.rglob("*")) if f.is_file())
    ck = hashlib.sha256((instr + "\n--\n" + tests).encode()).hexdigest()[:16]
    ans = ask(GAPREAD_PROMPT.format(contract=instr, tests=tests[:80000]),
              cache_key=f"gapread2/{unit_dir.name}/{ck}")
    miss = [l.strip() for l in ans.splitlines() if l.strip().startswith("MISSING")]
    conf = [l.strip() for l in ans.splitlines() if l.strip().startswith("CONFLICT")]
    kinds = {"ARBITRARY": 0, "DERIVABLE": 0, "COUNTER": 0}
    for l in miss + conf:
        parts = l.split("|")
        if len(parts) >= 2 and parts[1].strip().upper() in kinds:
            kinds[parts[1].strip().upper()] += 1
    return {"unit": unit_dir.name, "missing": len(miss),
            "conflicts": len(conf), "kinds": kinds, "answer": ans,
            "contract_sha": ck}


# --------------------------------------------------------------------------
# literal budget (A3-SURFACE numbers, same counting as task_lint)
# --------------------------------------------------------------------------


def literal_budget(unit_dir: Path) -> tuple[int, int, float]:
    instr = unit_dir / "instruction.md"
    text = instr.read_text(errors="replace") if instr.is_file() else ""
    blob = _hidden_blob(unit_dir)
    return repair_loop.literal_budget(text, blob)


# --------------------------------------------------------------------------
# preflight: bare fails, gold passes, FRESH cheat fails
# --------------------------------------------------------------------------


def preflight(unit_dir: Path, api_key: str) -> repair_loop.PreflightResult:
    return repair_loop.preflight(unit_dir, api_key)


def load_api_key() -> str:
    return shadow.load_api_key()

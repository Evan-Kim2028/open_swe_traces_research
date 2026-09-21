#!/usr/bin/env python3
"""Ask Composer which asserted commitments a contract fails to state. The contract-family gate.

One call per unit: full L2 contract + full hidden suite -> the list of commitments the suite
enforces that the prose does not. ~6k tokens and ~40s per unit, versus 2.08M for the trial it
prevents. On helm-repindex's original contract it recovered in one call every defect that took
five hand rounds and fifteen Composer trials to find.

Usage: contract_gap_read.py <out.jsonl> <unit dir> [...]
"""
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from ask_composer import ask  # noqa: E402

PROMPT = """You are auditing a task specification used to grade a coding agent.

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


def main(argv: list[str]) -> int:
    out_path = Path(argv[0])
    done = set()
    if out_path.exists():
        for line in out_path.read_text().splitlines():
            try:
                done.add(json.loads(line)["unit"])
            except Exception:
                pass
    with out_path.open("a") as fh:
        for arg in argv[1:]:
            u = Path(arg)
            if u.name in done or not u.is_dir():
                continue
            instr = u / "instruction.md"
            hd = u / "tests" / "hidden"
            if not instr.is_file() or not hd.is_dir():
                continue
            tests = "\n".join(f.read_text(errors="replace") for f in sorted(hd.rglob("*")) if f.is_file())
            prompt = PROMPT.format(contract=instr.read_text(errors="replace"), tests=tests[:80000])
            try:
                ans = ask(prompt, cache_key=f"gapread-r3/{u.name}")
            except Exception as exc:  # noqa: BLE001
                ans = f"__ERROR__ {exc}"
            miss = [l.strip() for l in ans.splitlines() if l.strip().startswith("MISSING")]
            conf = [l.strip() for l in ans.splitlines() if l.strip().startswith("CONFLICT")]
            kinds = {"ARBITRARY": 0, "DERIVABLE": 0, "COUNTER": 0}
            for l in miss + conf:
                parts = l.split("|")
                if len(parts) >= 2 and parts[1].strip().upper() in kinds:
                    kinds[parts[1].strip().upper()] += 1
            # A unit whose gaps are mostly ARBITRARY is low-discrimination: every solver fails it
            # for the same non-reason. Repairing its CONTRACT manufactures a task that measures
            # nothing; the assertion is what should be weakened.
            verdict = ("low-discrimination"
                       if kinds["ARBITRARY"] > kinds["DERIVABLE"] + kinds["COUNTER"]
                       else "repair-contract")
            fh.write(json.dumps({"unit": u.name, "missing": len(miss), "conflicts": len(conf),
                                 "kinds": kinds, "verdict": verdict, "answer": ans}) + "\n")
            fh.flush()
            print(f"{u.name:34} missing={len(miss):>2} conflicts={len(conf):>2} "
                  f"arb={kinds['ARBITRARY']} der={kinds['DERIVABLE']} ctr={kinds['COUNTER']} "
                  f"-> {verdict}", flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))

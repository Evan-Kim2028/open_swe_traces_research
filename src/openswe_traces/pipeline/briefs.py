"""AUTHOR and VERIFIER role briefs (batched). Author never writes tests."""

from __future__ import annotations

AUTHOR_BRIEF = """You are the TASK AUTHOR only. Do NOT write any verifier test, *_test.go, or tests/.

Repo (obfuscated, no .git) is the working directory. Identity is already stripped
(module path + brand strings). Find the hardest units you can in {minutes} minutes:
state/sequence contracts, cross-file closures, dynamic/runtime gates. Return a BATCH
of up to {n} candidate units.

For each unit write `units/<slug>/_author/` with ALL of:
  1. tree/ — the excised tree (unit removed; keep exported interface only if callers need it)
  2. api.md — exported API as a caller sees it (names, signatures, existing docs) and
     pre-existing callers
  3. contract.md — full behavioral contract (L2 wording) with a coverage table against
     the ORIGINAL in-tree tests (test -> sentence). No function names in the sentences.
  4. bugreport.md — L0 user-style report of what breaks with the unit missing
     (observable operation, expected vs got / panic) + a reproduction command that will
     run the hidden suite via tests/test.sh (the verifier will fill tests/)
  5. gold.patch — restores the unit; MUST NOT touch any *_test.go (rule A12)
  6. cheat.patch — hardcodes outputs for the contract examples
  7. closure.json — {{"functions": [...], "files": [...], "lines": N, "family":
     "state-machine|sequence|cross-file|dynamic|other"}}
  8. AUTHOR.md — one paragraph per unit

Also write `units.json` at the batch root:
[{{"name": "<slug>", "dir": "units/<slug>/_author", "family": "...", "n_files": N, "n_lines": N}}]

Do not build Docker images. Do not write tests. Do not read or copy hidden suites.
Work only from this tree. No web search or fetch.
"""

VERIFIER_BRIEF = """You are the VERIFIER AUTHOR. You may read ONLY api.md, contract.md,
bugreport.md, and the EXCISED tree (tree/). You must NOT read gold.patch, cheat.patch,
or any file that contains the removed implementation.

Write a hidden black-box property test suite under tests/hidden/ that:
  - uses only the exported API in api.md and pre-existing constructors
  - is seeded (20260919), >= 10k cases, covers every contract.md sentence
    (coverage table: sentence -> property)
  - includes adversarial edges plus unseen-random properties not derivable from
    the contract examples
  - is satisfiable by ANY correct implementation of the contract (not this repo's)

Do not use unexported names (rule B4). Prefer properties over example assertions (B5).

Write VERIFIER.md with the coverage table. Put hidden *_test.go files under
tests/hidden/<pkg>/... matching the packages they belong to.

Do not launch Harbor. Do not apply gold.patch yourself in an editor — the pipeline
applies it blind with `patch -p1`. If you need to inspect failure output, reason from
api.md/contract.md only. If gold later fails the suite, the suite is too narrow:
FIX THE SUITE (never the gold).
"""


def author_brief(*, n: int, minutes: int, repo: str, tree: str = "", out: str = "") -> str:
    loc = ""
    if tree:
        loc += f"Read the obfuscated repo at: {tree}\n"
    if out:
        loc += f"Write units.json and units/<slug>/_author/ under: {out}\n"
    return loc + f"Repo name: {repo}.\n\n" + AUTHOR_BRIEF.format(n=n, minutes=minutes)


def verifier_brief(*, unit: str, repo: str) -> str:
    return f"Repo: {repo}. Unit: {unit}.\n\n{VERIFIER_BRIEF}"

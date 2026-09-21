"""Repair loop: shadow -> attribute -> repair, to a fixed point.

The automated version of the five-round hand repair that cost fifteen
Composer trials on ``helm-repindex`` (see ``analytics/research/
verifier_rules.md``, "Verifier design: the two defect families"):

    shadow      implement from the contract ALONE, run the hidden suite
    attribute   for each failing assertion, does the contract state the
                commitment it checks — and where does the answer live?
    repair      DERIVABLE/COUNTER defects only: add the missing row or
                correct the contradicted claim. If stated, leave it —
                that is implementation difficulty, not a contract defect.
                If ARBITRARY, never repair: the TEST over-specifies and
                the unit is flagged low-discrimination.
    repeat      until no failing assertion is a repairable contract
                defect, or the round budget is spent.

Three safeguards are enforced mechanically:

- A shadow failure is confounded (weak model vs. hard unit), so repairs
  are driven ONLY by per-assertion attribution, never by pass/fail.
- Repair routes by defect kind (the ``contract_gap_read.py`` labels):
  ARBITRARY gaps are test defects — the assertion should be weakened by
  the author, and writing the literal into the contract is forbidden;
  DERIVABLE gaps are contract defects; COUNTER gaps are contract defects
  that also mark the unit high-value.
- Repair leaks cheat surface: the literal count must not rise across a
  round and the grounded fraction must not fall (``A3-SURFACE`` in
  ``scripts/ops/task_lint.py``). At the end the cheat patch is
  REGENERATED from the repaired contract and preflight re-run (bare
  fails, gold passes, fresh cheat fails) — a frozen cheat cannot detect
  a leak the repair introduced.

The loop reads gold and the hidden tests; it never touches
``bugreport.md`` and never weakens a hidden test.

Per-unit record lands in ``outputs/repair_loop/rounds/<unit>.json``;
summary rows append to ``outputs/repair_loop/results.jsonl``. Both are
resume-safe: a unit with a terminal outcome is skipped on re-run.
"""

from __future__ import annotations

import difflib
import hashlib
import json
import re
import shutil
import subprocess
import tempfile
import time
from collections.abc import Iterable
from dataclasses import dataclass
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.gate import shadow
from openswe_traces.synth import reconcile

LOOP_DIR = ROOT / "outputs" / "repair_loop"
GEN_DIR = LOOP_DIR / "gen"
CTX_DIR = LOOP_DIR / "contexts"
ROUND_DIR = LOOP_DIR / "rounds"
RESULTS_PATH = LOOP_DIR / "results.jsonl"
LOG_PATH = ROOT / "outputs" / "REPAIRLOOP.log"
WORK_DIR = LOOP_DIR / "work"

# Judgment + prose repair run on Composer like the shadow calls (OpenRouter
# is forbidden). One model; the attempt loop just retries it.
JUDGE_MODELS = (
    "composer-2.5",
)

MAX_ROUNDS = 6          # the hand repair converged in 5
MAX_REQUESTS = 600
MAX_TEST_BODY_CHARS = 2_600
MAX_CONTRACT_CHARS = 48_000
MAX_TOTAL_CHARS = 110_000
MAX_REPAIR_TRIES = 3    # first repair + retries carrying the violation text

TAIL_MARKERS = ("\n# Bug report", "\nReproduce with:")

EXAMPLE_LIT_RE = re.compile(r"[\"'`]([A-Za-z0-9][\w.\-+/:]{2,60})[\"'`]")
SCRUB_RE = re.compile(r"\bthe call\b", re.IGNORECASE)
DIRECTION_RE = re.compile(
    r"pre-?releases?\s+sort\s+(?:before|above|after|below)\s+releases?"
    r"|releases?\s+sort\s+(?:before|above)\s+pre-?releases?",
    re.IGNORECASE,
)
TEST_NAME_RE = re.compile(r"\bTest[A-Z][A-Za-z0-9_]*\b")
JSON_OBJ_RE = re.compile(r"\{.*\}", re.DOTALL)


def log(msg: str) -> None:
    LOG_PATH.parent.mkdir(parents=True, exist_ok=True)
    line = f"{time.strftime('%H:%M:%S')} {msg}"
    print(line, flush=True)
    with LOG_PATH.open("a") as f:
        f.write(line + "\n")


# --------------------------------------------------------------------------
# unit files
# --------------------------------------------------------------------------


def load_hidden(unit_dir: Path) -> dict[str, str]:
    return reconcile.load_hidden_dir(unit_dir / "tests" / "hidden")


def _tree_sig(unit_dir: Path) -> str:
    """Signature over the integrity-critical parts of a work copy: the
    source tree and everything under tests/ except cheat.patch — the
    contract and the regenerated cheat are the loop's own writes."""
    h = hashlib.sha256()
    for base in (unit_dir / "environment", unit_dir / "tests"):
        if not base.is_dir():
            continue
        for p in sorted(base.rglob("*")):
            if p.is_file() and p.name != "cheat.patch":
                h.update(str(p.relative_to(unit_dir)).encode())
                h.update(p.read_bytes())
    return h.hexdigest()


class TreeGuard:
    """The composer agent can nominally write outside its (empty) working
    dir — observed editing a work copy's stub file mid-generation. Ask
    mode makes it read-only, but this guard is the audit: any mutation of
    the work tree during an LLM call is logged and reverted from the
    source unit so nothing the agent wrote enters a scored tree."""

    def __init__(self, unit_dir: Path, src_unit: Path):
        self.unit_dir = unit_dir
        self.src_unit = src_unit
        self.sig = _tree_sig(unit_dir)
        self.mutations = 0

    def check(self, where: str = "") -> None:
        if _tree_sig(self.unit_dir) == self.sig:
            return
        self.mutations += 1
        log(f"  {self.unit_dir.name}: work tree mutated during {where} — "
            f"restoring environment/ and tests/ (sans cheat.patch) "
            f"from source")
        for rel in ("environment", "tests"):
            src = self.src_unit / rel
            if not src.exists():
                continue
            for sp in src.rglob("*"):
                if not sp.is_file() or sp.name == "cheat.patch":
                    continue
                dp = self.unit_dir / sp.relative_to(self.src_unit)
                dp.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(sp, dp)
            # drop files the agent added that the source does not have
            dst = self.unit_dir / rel
            for dp in sorted(dst.rglob("*")):
                if (dp.is_file() and dp.name != "cheat.patch"
                        and not (src / dp.relative_to(dst)).exists()):
                    dp.unlink()
        self.sig = _tree_sig(self.unit_dir)


def gold_text(unit_dir: Path) -> str:
    for rel in ("tests/gold.patch", "patches/gold.patch"):
        p = unit_dir / rel
        if p.is_file():
            return p.read_text(errors="replace")
    return ""


def contract_head_tail(instruction: str) -> tuple[str, str]:
    """(contract part, untouchable tail). The tail starts at the first of
    ``# Bug report`` / ``Reproduce with:`` and carries the bug report,
    reproduce block, hidden-test-names section and no-web clause —
    everything a repair may not rewrite (bugreport.md is author-side;
    the loop never edits it)."""
    cut = -1
    for marker in TAIL_MARKERS:
        i = instruction.find(marker)
        if i >= 0 and (cut < 0 or i < cut):
            cut = i
    if cut < 0:
        return instruction, ""
    return instruction[:cut], instruction[cut:]


def literal_set(text: str) -> set[str]:
    return {m.group(1) for m in EXAMPLE_LIT_RE.finditer(text)
            if len(m.group(1)) >= 4}


def literal_budget(text: str, hidden_blob: str) -> tuple[int, int, float]:
    """(literals, grounded, grounded fraction) — same counting as
    ``scripts/ops/task_lint.py`` ``literal_budget`` (A3-SURFACE)."""
    lits = literal_set(text)
    grounded = sum(1 for lit in lits if lit in hidden_blob)
    frac = grounded / len(lits) if lits else 1.0
    return len(lits), grounded, frac


def unified_diff(old: str, new: str, name: str = "instruction.md") -> str:
    return "".join(
        difflib.unified_diff(
            old.splitlines(keepends=True),
            new.splitlines(keepends=True),
            fromfile=f"a/{name}",
            tofile=f"b/{name}",
        )
    )


# --------------------------------------------------------------------------
# cached free-tier JSON call (judgments + repair)
# --------------------------------------------------------------------------


@dataclass
class CallResult:
    text: str | None
    model: str
    requests: int
    tokens: int
    error: str | None = None
    cache_hit: bool = False


def llm_call(
    prompt: str,
    api_key: str,
    tag: str,
    models: Iterable[str] = JUDGE_MODELS,
    budget: int = MAX_REQUESTS,
    max_tokens: int = 32768,
) -> CallResult:
    """One cached completion. Same retry discipline as ``shadow.generate``."""
    GEN_DIR.mkdir(parents=True, exist_ok=True)
    key = hashlib.sha256((tag + "\n" + prompt).encode()).hexdigest()[:24]
    path = GEN_DIR / f"{key}.json"
    if path.is_file():
        try:
            data = json.loads(path.read_text())
            return CallResult(data["text"], data.get("model", "cached"), 0,
                              int(data.get("tokens", 0)), cache_hit=True)
        except (OSError, json.JSONDecodeError, KeyError):
            pass
    requests, tokens = 0, 0
    last_err = "no models configured"
    models = list(models)
    for attempt in range(shadow.MAX_ATTEMPTS):
        model = models[min(attempt, len(models) - 1)]
        if requests >= budget:
            return CallResult(None, model, requests, tokens, "request budget exhausted")
        requests += 1
        text, tk, err = shadow._post(api_key, model, prompt)
        tokens += tk
        if text is not None:
            path.write_text(json.dumps(
                {"model": model, "tokens": tokens, "tag": tag,
                 "prompt_sha": key, "text": text}, indent=1) + "\n")
            return CallResult(text, model, requests, tokens)
        last_err = err
        if err and ("404" in err or "No endpoints" in err or "400" in err):
            continue
        if err and "429" in err:
            time.sleep(45 * (attempt + 1))
            continue
        time.sleep(5)
    return CallResult(None, models[-1], requests, tokens, last_err)


def _truncate(text: str, cap: int) -> str:
    if len(text) <= cap:
        return text
    head = cap * 3 // 4
    return text[:head] + f"\n...[{len(text) - cap} chars elided]...\n" + text[-(cap - head):]


def _extract_json(text: str) -> dict | None:
    m = JSON_OBJ_RE.search(text or "")
    if not m:
        return None
    try:
        data = json.loads(m.group(0))
    except json.JSONDecodeError:
        return None
    return data if isinstance(data, dict) else None


# --------------------------------------------------------------------------
# shadow round: implement from the contract alone, run the suite
# --------------------------------------------------------------------------


@dataclass
class ShadowRound:
    outcome: str                  # pass | fail_assert | fail_build | fail_apply | error
    reward: int | None
    model: str
    requests: int
    tokens: int
    seconds: float
    passing_tests: list[str]
    fail_blocks: dict[str, list[str]]
    compile_errors: list[str]
    missing_stubs: list[str]
    note: str
    context_hash: str


def run_shadow(unit_dir: Path, api_key: str,
               models: Iterable[str] = shadow.SHADOW_MODELS,
               budget: int = MAX_REQUESTS,
               guard: TreeGuard | None = None) -> ShadowRound:
    """``gate_unit`` specialised for the loop: keeps per-assertion failure
    blocks and spends the compile-repair passes before giving up."""
    t0 = time.time()
    name = unit_dir.name
    try:
        ctx = shadow.build_context(unit_dir)
    except (OSError, AssertionError) as e:
        return ShadowRound("error", None, "", 0, 0, 0, [], {},
                           [], [], f"{type(e).__name__}: {e}", "")
    CTX_DIR.mkdir(parents=True, exist_ok=True)
    (CTX_DIR / f"{name}.prompt.txt").write_text(ctx.prompt)
    (CTX_DIR / f"{name}.manifest.json").write_text(json.dumps(ctx.manifest, indent=1))

    src_root = unit_dir / "environment" / "src"
    excised_files = shadow.find_excised_files(src_root)

    spent_requests = spent_tokens = 0
    model_used = ""
    code = shadow.ShadowCode(imports=[], funcs={})
    prev_code = ""
    problems: list[str] = []
    repair = 0
    while True:
        if repair == 0:
            prompt, tag = ctx.prompt, f"loop/{name}#s0"
        else:
            prompt, tag = shadow._repair_prompt(ctx, prev_code, problems), f"loop/{name}#s{repair}"
        gen = shadow.generate(prompt, api_key, tag, models,
                              max(0, budget - spent_requests))
        spent_requests += gen.requests
        spent_tokens += gen.tokens
        model_used = gen.model
        if guard is not None:
            guard.check("shadow generation")
        if gen.text is None:
            return ShadowRound("error", None, model_used, spent_requests,
                               spent_tokens, time.time() - t0, [], {}, [], [],
                               gen.error or "generation failed", ctx.context_hash)
        new = shadow.parse_response(gen.text)
        code.funcs.update(new.funcs)
        code.imports = sorted(set(code.imports) | set(new.imports))
        prev_code = "\n\n".join(code.funcs.values())
        with tempfile.TemporaryDirectory(prefix="looptree-") as td:
            tree = Path(td) / "src"
            shutil.copytree(src_root, tree, symlinks=True)
            rep = shadow.apply_shadow(tree, excised_files, code)
            score: shadow.ScoreResult | None = None
            if rep.ok:
                score = shadow.score_tree(unit_dir, tree)
        if score is not None and score.reward == 1:
            return ShadowRound("pass", 1, model_used, spent_requests,
                               spent_tokens, time.time() - t0,
                               score.passing_tests, score.fail_blocks, [], [],
                               score.note, ctx.context_hash)
        problems = []
        if not rep.ok:
            problems = [f"unimplemented stubs (still panic): {m}"
                        for m in rep.missing]
        elif score is not None:
            is_compile = score.build_failed or bool(score.compile_errors)
            problems = score.compile_errors[:] if is_compile else []
        if problems and repair < shadow.MAX_REPAIRS and spent_requests < budget:
            repair += 1
            log(f"  {name}: {'missing stubs' if not rep.ok else 'compile failed'}, "
                f"shadow repair {repair}")
            continue
        if not rep.ok:
            return ShadowRound("fail_apply", None, model_used, spent_requests,
                               spent_tokens, time.time() - t0, [], {}, [],
                               rep.missing, rep.note, ctx.context_hash)
        assert score is not None
        if score.build_failed or score.compile_errors:
            outcome = "fail_build"
        elif score.reward == 0:
            outcome = "fail_assert"
        else:
            outcome = "error"
        return ShadowRound(outcome, score.reward, model_used, spent_requests,
                           spent_tokens, time.time() - t0, score.passing_tests,
                           score.fail_blocks, score.compile_errors, [],
                           score.note, ctx.context_hash)


# --------------------------------------------------------------------------
# attribution: does the contract state what each failing assertion checks?
# --------------------------------------------------------------------------


@dataclass
class Attribution:
    test: str
    assertion: str
    verdict: str          # stated | unstated | contradicted | ambiguous
    kind: str             # ARBITRARY | DERIVABLE | COUNTER | "" (n/a for stated)
    commitment: str
    evidence: str


def _test_bodies(hidden_funcs: list[reconcile.HiddenFunc]) -> dict[str, str]:
    return {fn.name: fn.body for fn in hidden_funcs}


def attribution_prompt(contract: str,
                       fail_blocks: dict[str, list[str]],
                       bodies: dict[str, str]) -> str:
    items: list[str] = []
    n = 0
    for test, msgs in fail_blocks.items():
        body = bodies.get(test.split("/")[0], "")
        for msg in msgs or ["(no assertion message captured)"]:
            n += 1
            items.append(
                f"{n}. TEST `{test}` failed with:\n"
                f"   {msg[:400]}\n"
                f"   Test source:\n   ```go\n{_truncate(body, MAX_TEST_BODY_CHARS)}\n   ```"
            )
    return f"""You are auditing a task CONTRACT against the failing assertions of a
HIDDEN test suite. A weak re-implementation built ONLY from the contract was
scored by the suite; the failures below are what it got wrong. For each one,
decide whether the contract itself is at fault — and if so, what kind of
defect it is.

For each failing assertion answer ONE verdict:
- "stated":   the contract states the behavioral commitment this assertion
              checks. The failure is implementation difficulty, NOT a contract
              defect.
- "unstated": no contract sentence states the commitment the assertion checks.
              The row is missing — a contract defect.
- "contradicted": the contract states the commitment but states it WRONG —
              the suite asserts the opposite. A contract defect; the claim
              must be corrected, not kept.
- "ambiguous": the assertion's commitment cannot be cleanly located either way.

For every "unstated" or "contradicted" verdict also label WHERE THE ANSWER
LIVES, because that decides what to fix:
- "ARBITRARY": nowhere. An authorial choice nothing in a repository implies —
  an exact error-message literal, a specific type spelling, punctuation,
  internal test scaffolding. Every solver fails it for the same non-reason.
  The TEST is the defect here; writing the literal into the contract makes
  the unit pass and measure nothing.
- "DERIVABLE": in the repository. A real invariant the surrounding code
  implies — nil-safety, non-negativity, a documented format, consistency
  with a neighbouring function. The contract under-specifies; repair it.
- "COUNTER": in the repository, but the obvious reading is wrong. The best
  kind — it separates solvers that read the code from solvers that
  pattern-match. Repair the contract AND flag the unit as high value.

CONTRACT:
{_truncate(contract, MAX_CONTRACT_CHARS)}

FAILING ASSERTIONS:
{chr(10).join(items)}

Answer with JSON only:
{{"attributions": [
  {{"test": "<test name>", "assertion": "<the failing check, shortened>",
    "verdict": "stated|unstated|contradicted|ambiguous",
    "kind": "ARBITRARY|DERIVABLE|COUNTER or empty for stated/ambiguous",
    "commitment": "<the behavioral commitment the assertion checks>",
    "evidence": "<the contract sentence that states it, or empty>"}}]}}
Include every numbered item exactly once."""


def attribute(contract: str,
              fail_blocks: dict[str, list[str]],
              hidden_funcs: list[reconcile.HiddenFunc],
              api_key: str,
              tag: str,
              models: Iterable[str] = JUDGE_MODELS,
              budget: int = MAX_REQUESTS) -> tuple[list[Attribution], CallResult]:
    bodies = _test_bodies(hidden_funcs)
    prompt = _truncate(attribution_prompt(contract, fail_blocks, bodies),
                       MAX_TOTAL_CHARS)
    res = llm_call(prompt, api_key, tag, models, budget)
    out: list[Attribution] = []
    data = _extract_json(res.text or "")
    rows = data.get("attributions") if isinstance(data, dict) else None
    if isinstance(rows, list):
        for row in rows:
            if not isinstance(row, dict):
                continue
            verdict = str(row.get("verdict") or "").strip().lower()
            if verdict not in {"stated", "unstated", "contradicted", "ambiguous"}:
                verdict = "ambiguous"
            kind = str(row.get("kind") or "").strip().upper()
            if kind not in {"ARBITRARY", "DERIVABLE", "COUNTER"}:
                kind = ""
            if verdict in {"stated", "ambiguous"}:
                kind = ""
            # A defect verdict without a kind is not actionable: the whole
            # point of the label is deciding whether the contract or the
            # test owns the defect. Downgrade to ambiguous rather than guess.
            if verdict in {"unstated", "contradicted"} and not kind:
                verdict = "ambiguous"
            out.append(Attribution(
                test=str(row.get("test") or ""),
                assertion=str(row.get("assertion") or "")[:300],
                verdict=verdict,
                kind=kind,
                commitment=str(row.get("commitment") or "")[:500],
                evidence=str(row.get("evidence") or "")[:300],
            ))
    if not out:
        # judge failed or returned nothing parseable: every failure is
        # un-attributed — treat as ambiguous, never as unstated
        for test, msgs in fail_blocks.items():
            for msg in msgs or ["(no assertion message captured)"]:
                out.append(Attribution(test, msg[:300], "ambiguous", "", "",
                                       "attribution unavailable"))
    return out, res


# --------------------------------------------------------------------------
# repair: derive a coverage row per unstated assertion, revise the contract
# --------------------------------------------------------------------------


@dataclass
class RepairOutcome:
    applied: bool
    new_head: str
    violations: list[str]
    model: str
    requests: int
    tokens: int
    note: str = ""


def repair_prompt(contract_head: str,
                  defects: list[Attribution],
                  hidden_funcs: list[reconcile.HiddenFunc],
                  gold: str,
                  denylist: set[str],
                  lit_note: str = "",
                  prev_violation: str = "") -> str:
    bodies = _test_bodies(hidden_funcs)
    items = []
    for i, a in enumerate(defects, 1):
        body = bodies.get(a.test.split("/")[0], "")
        action = ("CORRECT the contract's claim — it states this commitment "
                  "wrongly" if a.verdict == "contradicted"
                  else "ADD this missing commitment")
        items.append(
            f"{i}. `{a.test}` — assertion: {a.assertion}\n"
            f"   commitment it checks: {a.commitment or '(derive it from the source)'}\n"
            f"   action: {action}\n"
            f"   ```go\n{_truncate(body, MAX_TEST_BODY_CHARS)}\n   ```"
        )
    deny = ", ".join(sorted(denylist)[:40]) or "(none)"
    retry = ""
    if prev_violation:
        retry = ("\nYour previous revision was REJECTED: " + prev_violation +
                 "\nFix exactly that and return the full revised contract again.\n")
    return f"""You are repairing the L2 CONTRACT of a synthetic SWE task. A weak
re-implementation built only from this contract failed hidden-test assertions
whose commitments the contract omits or states wrongly. Add or correct exactly
those commitments — no more. Do not weaken or remove anything else already
stated.

Rules:
- Behavioural prose only. Do NOT name functions, types, files, packages, or
  line numbers. Forbidden names: {deny}
- Hidden TEST names must not appear in the contract (the suite stays hidden;
  coverage rows are numbered, never keyed by test).
- Concrete literals (versions, URLs, digests, wire strings, JSON keys) ARE
  allowed — but ONLY ones that appear verbatim in the failing test sources.
  Never invent an example.
- Never state ordering as a class direction ("X sorts before/after Y").
  State the pairwise rule against concrete versions, and whether build
  metadata participates.
- Keep the document structure: title, prose sections, a coverage/commitment
  table. New commitments go into the table AND the prose, stated fully —
  one row per unstated assertion.
- LITERAL BUDGET: the revised contract must not contain more distinct
  quoted/backticked literals than the current one, and a larger share of
  them must appear in the test sources. Prefer stating the rule over
  quoting examples; drop literals that appear in no test source.
{lit_note}

UNSTATED FAILING ASSERTIONS (repair exactly these):
{chr(10).join(items)}

GOLD PATCH (truth, for getting commitments right — never for naming):
```diff
{_truncate(gold, 18_000)}
```

CURRENT CONTRACT (return the FULL revised contract — head only, up to but
not including the "Reproduce with:" / bug-report tail):
{contract_head}
{retry}
Answer with the revised contract markdown ONLY — no JSON, no commentary."""


def check_repair(new_head: str,
                 old_instruction: str,
                 new_instruction: str,
                 added_rows: list[str],
                 hidden_names: set[str],
                 hidden_blob: str) -> list[str]:
    """Mechanical guards on a proposed revision. Every violation is a
    rejection reason fed back to the repair call."""
    violations: list[str] = []
    if len(new_head.strip()) < 200:
        violations.append("revised contract is empty or trivial")
    names = set(TEST_NAME_RE.findall(new_head)) & hidden_names
    if names:
        violations.append(f"hidden test names leaked into contract: {sorted(names)[:4]}")
    if SCRUB_RE.search(new_head):
        violations.append('unresolved scrub placeholder "the call" in revision')
    if DIRECTION_RE.search(new_head):
        violations.append("ordering still stated as a class direction, not pairwise")
    old_lits, _og, _of = literal_budget(old_instruction, hidden_blob)
    new_lits, _ng, new_frac = literal_budget(new_instruction, hidden_blob)
    if new_lits > old_lits:
        introduced = sorted(literal_set(new_instruction)
                            - literal_set(old_instruction))
        droppable = sorted(l for l in literal_set(old_instruction)
                           if l not in hidden_blob)
        violations.append(
            f"A3-SURFACE: literal count rose {old_lits} -> {new_lits}; "
            f"literals you introduced: {introduced[:8]}; existing literals "
            f"absent from every test source (drop to make room): "
            f"{droppable[:8]}")
    if new_frac < _of - 1e-9:
        ungrounded = sorted(l for l in literal_set(new_instruction)
                            if l not in hidden_blob)
        violations.append(
            f"A3-SURFACE: grounded fraction fell {_of:.0%} -> {new_frac:.0%}; "
            f"ungrounded literals in the revision: {ungrounded[:8]}")
    return violations


def lost_rows(new_instruction: str, added_rows: list[str]) -> list[str]:
    """Commitments added in earlier rounds that no longer appear verbatim.
    A warning, not a rejection: a dropped row resurfaces as an unstated
    failure on the next shadow round, and the round budget caps ping-pong."""
    return [r for r in added_rows if r and r not in new_instruction]


def repair(unit_dir: Path,
           instruction: str,
           defects: list[Attribution],
           added_rows: list[str],
           hidden_funcs: list[reconcile.HiddenFunc],
           api_key: str,
           tag: str,
           models: Iterable[str] = JUDGE_MODELS,
           budget: int = MAX_REQUESTS) -> RepairOutcome:
    """Revise the contract head for the repairable defects (DERIVABLE and
    COUNTER only — ARBITRARY gaps are filtered upstream); enforce the
    literal budget and B-family checks; splice the tail back untouched."""
    head, tail = contract_head_tail(instruction)
    hidden_blob = "\n".join(load_hidden(unit_dir).values())
    hidden_names = {fn.name for fn in hidden_funcs}
    gold = gold_text(unit_dir)
    denylist = reconcile.gold_denylist(gold)

    cur_lits = literal_set(instruction)
    lit_note = (
        f"  Current inventory: {len(cur_lits)} literals. Grounded in the test\n"
        f"  sources (safe to keep): "
        f"{sorted(l for l in cur_lits if l in hidden_blob)}\n"
        f"  Absent from every test source (drop these to make room): "
        f"{sorted(l for l in cur_lits if l not in hidden_blob)}\n"
        f"  HARD CAP: at most {len(cur_lits)} literals in the revision.")

    spent_requests = spent_tokens = 0
    model_used = ""
    violation_msg = ""
    for attempt in range(MAX_REPAIR_TRIES):
        prompt = _truncate(
            repair_prompt(head, defects, hidden_funcs, gold, denylist,
                          lit_note, violation_msg),
            MAX_TOTAL_CHARS)
        res = llm_call(prompt, api_key, f"{tag}#try{attempt}", models, budget)
        spent_requests += res.requests
        spent_tokens += res.tokens
        model_used = res.model
        if res.text is None:
            return RepairOutcome(False, "", [], model_used, spent_requests,
                                 spent_tokens, res.error or "repair call failed")
        new_head = res.text.strip()
        # strip a markdown fence the model may wrap the contract in
        fm = shadow.FENCE_RE.search(new_head)
        if fm and new_head.startswith("```"):
            new_head = fm.group(1).strip()
        # the model may prefix a narration line before the markdown —
        # cut everything before the first heading
        hm = re.search(r"(?m)^#\s", new_head)
        if hm:
            new_head = new_head[hm.start():]
        # the model was asked for the head only — if it emitted the tail
        # anyway, cut it so the splice does not duplicate it
        _, own_tail = contract_head_tail(new_head)
        if own_tail:
            new_head, _ = new_head.split(own_tail, 1)
            new_head = new_head.rstrip() + "\n"
        # polish with the reconciler's deterministic fixes
        new_head, ord_fixed = reconcile.rewrite_directional_ordering(new_head, hidden_blob)
        new_head, survivors = reconcile.strip_ungrounded_example_clauses(new_head, hidden_blob)
        new_head, _leaks = reconcile.scrub_b7(new_head, denylist)
        new_instruction = new_head + tail
        violations = check_repair(new_head, instruction, new_instruction,
                                  added_rows, hidden_names, hidden_blob)
        if survivors:
            bad_lits = reconcile.ungrounded_literals(survivors[0], hidden_blob)
            violations.append(
                f"ungrounded example literals survived polish: "
                f"{bad_lits[:6]} in clause: {survivors[0][:100]!r} — quote "
                f"only literals that appear verbatim in the test sources")
        if not violations:
            return RepairOutcome(True, new_head, [], model_used,
                                 spent_requests, spent_tokens,
                                 "ordering rewritten pairwise" if ord_fixed else "")
        violation_msg = "; ".join(violations)
        log(f"  {unit_dir.name}: repair try {attempt} rejected: {violation_msg[:220]}")
    return RepairOutcome(False, new_head, violations, model_used,
                         spent_requests, spent_tokens, violation_msg)


# --------------------------------------------------------------------------
# cheat regeneration + preflight
# --------------------------------------------------------------------------


def cheat_prompt(ctx_prompt: str) -> str:
    """Turn the shadow generation prompt into an adversarial one: satisfy the
    contract's letter, not its spirit."""
    return ctx_prompt.replace(
        "You are re-implementing excised functions in a Go package.",
        "You are writing a CHEAT implementation of excised functions in a Go "
        "package. Your goal is to satisfy the CONTRACT's stated examples and "
        "prose with the MINIMUM real work: hardcode the worked examples, "
        "special-case the stated literals, return plausible constant values "
        "wherever the contract does not pin exact behavior. Do NOT implement "
        "the real semantics — the point is to prove a grader cannot be "
        "gamed by the contract alone.",
        1)


def apply_patch(tree: Path, patch_text: str) -> tuple[bool, str]:
    """Apply a unified/git patch inside ``tree`` (not a git repo)."""
    with tempfile.NamedTemporaryFile("w", suffix=".patch", delete=False) as f:
        f.write(patch_text)
        tmp = f.name
    try:
        r = subprocess.run(["git", "apply", "--whitespace=nowarn", tmp],
                           cwd=tree, capture_output=True, text=True, timeout=60,
                           check=False)
        if r.returncode != 0:
            return False, (r.stderr or r.stdout)[-300:]
        return True, ""
    finally:
        Path(tmp).unlink(missing_ok=True)


def make_patch(old_root: Path, new_root: Path) -> str:
    """git-style unified diff of two trees, ``a/<rel> b/<rel>`` paths."""
    old_files = {p.relative_to(old_root) for p in old_root.rglob("*") if p.is_file()}
    new_files = {p.relative_to(new_root) for p in new_root.rglob("*") if p.is_file()}
    out: list[str] = []
    for rel in sorted(old_files | new_files):
        old = (old_root / rel).read_text(errors="replace") if rel in old_files else ""
        new = (new_root / rel).read_text(errors="replace") if rel in new_files else ""
        if old == new:
            continue
        out.append(f"diff --git a/{rel} b/{rel}\n")
        out.append("".join(difflib.unified_diff(
            old.splitlines(keepends=True), new.splitlines(keepends=True),
            fromfile=f"a/{rel}", tofile=f"b/{rel}")))
    return "".join(out)


@dataclass
class PreflightResult:
    bare_reward: int | None
    gold_reward: int | None
    cheat_reward: int | None
    cheat_patch: str
    verdict: str           # ok | cheat_passes (literal-reusing) |
                           # cheat_general_pass | gold_fails |
                           # bare_passes | incomplete
    note: str = ""
    requests: int = 0
    tokens: int = 0


def regenerate_cheat(unit_dir: Path, api_key: str,
                     models: Iterable[str] = shadow.SHADOW_MODELS,
                     budget: int = MAX_REQUESTS) -> tuple[str, shadow.ShadowCode, str, int, int]:
    """Fresh cheat implementation FROM THE CURRENT CONTRACT. Returns
    (note, code, model, requests, tokens); the patch is made by the caller."""
    ctx = shadow.build_context(unit_dir)
    prompt = cheat_prompt(ctx.prompt)
    gen = shadow.generate(prompt, api_key, f"loop/{unit_dir.name}#cheat",
                          models, budget)
    if gen.text is None:
        return gen.error or "cheat generation failed", shadow.ShadowCode([], {}), gen.model, gen.requests, gen.tokens
    code = shadow.parse_response(gen.text)
    return "", code, gen.model, gen.requests, gen.tokens


def preflight(unit_dir: Path, api_key: str,
              models: Iterable[str] = shadow.SHADOW_MODELS,
              budget: int = MAX_REQUESTS,
              guard: TreeGuard | None = None) -> PreflightResult:
    """Bare fails, gold passes, FRESH cheat (from the repaired contract) fails."""
    if guard is not None:
        guard.check("pre-preflight")
    src_root = unit_dir / "environment" / "src"
    excised_files = shadow.find_excised_files(src_root)

    bare = shadow.score_tree(unit_dir, src_root)

    with tempfile.TemporaryDirectory(prefix="goldtree-") as td:
        gtree = Path(td) / "src"
        shutil.copytree(src_root, gtree, symlinks=True)
        gold = gold_text(unit_dir)
        ok, err = apply_patch(gtree, gold)
        gold_res = (shadow.score_tree(unit_dir, gtree) if ok else
                    shadow.ScoreResult(None, [], [], False, [], [], "", 0,
                                       f"gold apply failed: {err}"))

    note, cheat_code, _cm, reqs, toks = regenerate_cheat(
        unit_dir, api_key, models, budget)
    if guard is not None:
        guard.check("cheat generation")
    cheat_res: shadow.ScoreResult | None = None
    cheat_patch = ""
    if cheat_code.funcs:
        with tempfile.TemporaryDirectory(prefix="cheattree-") as td:
            ctree = Path(td) / "src"
            shutil.copytree(src_root, ctree, symlinks=True)
            rep = shadow.apply_shadow(ctree, excised_files, cheat_code)
            if rep.ok:
                cheat_patch = make_patch(src_root, ctree)
                cheat_res = shadow.score_tree(unit_dir, ctree)
            else:
                note = (note + "; " if note else "") + rep.note

    verdict = "ok"
    if bare.reward != 0:
        verdict = "bare_passes"
    elif gold_res.reward != 1:
        verdict = "gold_fails"
    elif cheat_res is None:
        verdict = "incomplete"
    elif cheat_res.reward == 1:
        # A3 is violated only if the cheat SPECIAL-CASED the contract's
        # stated literals. A cheat that passes while quoting none of them
        # just wrote a working implementation — a capability pass, not a
        # leak (composer-2.5 often does this on easy units).
        head_lits = literal_set(
            contract_head_tail(
                (unit_dir / "instruction.md").read_text(errors="replace"))[0])
        added = "\n".join(
            l[1:] for l in cheat_patch.splitlines()
            if l.startswith("+") and not l.startswith("+++"))
        reused = sorted(l for l in head_lits if l in added)
        verdict = ("cheat_passes" if reused else "cheat_general_pass")
        if reused:
            note = (note + "; " if note else "") + \
                f"cheat reused contract literals: {reused[:6]}"
    return PreflightResult(bare.reward, gold_res.reward,
                           cheat_res.reward if cheat_res else None,
                           cheat_patch, verdict,
                           note or (cheat_res.note if cheat_res else ""),
                           reqs, toks)


# --------------------------------------------------------------------------
# the loop
# --------------------------------------------------------------------------


@dataclass
class LoopResult:
    unit: str
    outcome: str
    rounds: list[dict]
    final_instruction: str
    total_requests: int
    total_tokens: int
    seconds: float
    preflight: PreflightResult | None = None
    note: str = ""
    arb_gaps: list[dict] | None = None
    ctr_gaps: list[dict] | None = None

    @property
    def unit_verdict(self) -> str:
        """Where this unit stands for discrimination after the loop.

        - "low-discrimination": an ARBITRARY gap exists — the TEST over-
          specifies; unfair at every rung until the assertion changes. A
          repaired contract cannot fix it.
        - "high-value": a COUNTER gap was found and repaired — the unit
          separates readers from pattern-matchers.
        - "repair-contract": all defects were DERIVABLE/COUNTER and the
          contract was repaired (or needed none).
        """
        if self.arb_gaps:
            return "low-discrimination"
        if self.ctr_gaps:
            return "high-value"
        return "repair-contract"

    @property
    def fair_at_rung(self) -> str:
        """First rung at which a competent engineer holding only that rung's
        information produces the graded behaviour. ARBITRARY gaps have no
        such rung; DERIVABLE/COUNTER answers live in the repo."""
        return "none (unfair until assertion changes)" if self.arb_gaps else "L0"


# "error" is deliberately absent: a crashed unit retries on the next run.
TERMINAL = {"converged_pass", "converged_fixed_point", "budget_exhausted",
            "repair_rejected", "inconclusive"}


def loop_unit(src_unit: Path, api_key: str,
              work_root: Path = WORK_DIR,
              max_rounds: int = MAX_ROUNDS,
              shadow_models: Iterable[str] = shadow.SHADOW_MODELS,
              judge_models: Iterable[str] = JUDGE_MODELS,
              budget: int = MAX_REQUESTS,
              run_preflight: bool = True) -> LoopResult:
    """Run shadow->attribute->repair to a fixed point on ONE unit.

    The unit is copied to ``work_root/<name>`` once; instruction.md is
    mutated there across rounds — the source dir is never written.
    """
    t0 = time.time()
    src_unit = src_unit.resolve()
    name = src_unit.name
    unit_dir = work_root / name
    if not unit_dir.exists():
        unit_dir.parent.mkdir(parents=True, exist_ok=True)
        shutil.copytree(src_unit, unit_dir, symlinks=True)
    unit_dir = unit_dir.resolve()
    # the work copy persists across attempts — a prior run may have left a
    # repaired contract, a regenerated cheat, or agent-written source
    # edits behind. Rebuild the guarded state from the source unit so
    # round 0 is the original contract on a pristine tree (cached calls
    # make a replay cheap).
    (unit_dir / "instruction.md").write_text(
        (src_unit / "instruction.md").read_text(errors="replace"))
    for rel in ("environment", "tests"):
        src = src_unit / rel
        dst = unit_dir / rel
        if not src.exists():
            continue
        if dst.is_symlink():
            dst.unlink()
        elif dst.exists():
            shutil.rmtree(dst)
        shutil.copytree(src, dst, symlinks=True)

    hidden_funcs = reconcile.parse_hidden_funcs(load_hidden(unit_dir))
    if not hidden_funcs:
        return LoopResult(name, "error", [], "", 0, 0, 0, None,
                          "no hidden tests under tests/hidden")
    instr_path = unit_dir / "instruction.md"
    instruction = instr_path.read_text(errors="replace")
    guard = TreeGuard(unit_dir, src_unit)

    rounds: list[dict] = []
    added_rows: list[str] = []
    arb_gaps: list[dict] = []   # ARBITRARY defects — the test over-specifies
    ctr_gaps: list[dict] = []   # COUNTER defects — high-value when repaired
    total_req = total_tok = 0
    outcome = "budget_exhausted"
    note = ""

    for rnd in range(max_rounds):
        rec: dict = {"round": rnd,
                     "contract_sha": hashlib.sha256(instruction.encode()).hexdigest()[:16]}
        hidden_blob = "\n".join(load_hidden(unit_dir).values())
        lit_before = literal_budget(instruction, hidden_blob)
        rec["literals_before"] = [lit_before[0], lit_before[1]]

        # --- shadow -----------------------------------------------------
        sr = run_shadow(unit_dir, api_key, shadow_models,
                        max(0, budget - total_req), guard)
        total_req += sr.requests
        total_tok += sr.tokens
        rec["shadow"] = {
            "outcome": sr.outcome, "reward": sr.reward, "model": sr.model,
            "passing": sr.passing_tests,
            "failures": [{"test": t, "messages": m}
                         for t, m in sr.fail_blocks.items()],
            "compile_errors": sr.compile_errors[:6],
            "missing_stubs": sr.missing_stubs[:6], "note": sr.note,
            "requests": sr.requests, "tokens": sr.tokens,
        }
        log(f"{name} r{rnd}: shadow {sr.outcome} reward={sr.reward} "
            f"fail={sorted(sr.fail_blocks)} req={sr.requests} tok={sr.tokens}")

        if sr.outcome == "pass":
            outcome = "converged_pass"
            rec["requests"] = sr.requests
            rec["tokens"] = sr.tokens
            rounds.append(rec)
            break
        if sr.outcome != "fail_assert" or not sr.fail_blocks:
            outcome = "inconclusive"
            note = f"shadow {sr.outcome}: no assertions to attribute"
            rec["requests"] = sr.requests
            rec["tokens"] = sr.tokens
            rounds.append(rec)
            break

        # --- attribute --------------------------------------------------
        contract = instruction
        attrs, ares = attribute(contract, sr.fail_blocks, hidden_funcs,
                              api_key, f"loop/{name}#a{rnd}", judge_models,
                              max(0, budget - total_req))
        guard.check("attribution")
        total_req += ares.requests
        total_tok += ares.tokens
        rec["attribution"] = [
            {"test": a.test, "assertion": a.assertion, "verdict": a.verdict,
             "kind": a.kind, "commitment": a.commitment, "evidence": a.evidence}
            for a in attrs
        ]
        attribution_ok = ares.text is not None and any(
            a.evidence != "attribution unavailable" for a in attrs)
        # Route by kind, never repair uniformly. DERIVABLE/COUNTER gaps are
        # contract defects -> repair. ARBITRARY gaps are TEST defects ->
        # record and flag; the contract is never repaired toward an arbitrary
        # literal (that manufactures a pass measuring nothing). The loop may
        # not touch hidden tests, so weakening the assertion is an
        # author-side recommendation recorded in the unit's verdict.
        defects = [a for a in attrs if a.verdict in {"unstated", "contradicted"}]
        repairable = [a for a in defects if a.kind in {"DERIVABLE", "COUNTER"}]
        arbitrary = [a for a in defects if a.kind == "ARBITRARY"]
        stated = [a for a in attrs if a.verdict == "stated"]
        ambig = [a for a in attrs if a.verdict == "ambiguous"]
        arb_gaps.extend({"test": a.test, "assertion": a.assertion,
                         "commitment": a.commitment, "round": rnd}
                        for a in arbitrary)
        ctr_gaps.extend({"test": a.test, "assertion": a.assertion,
                         "commitment": a.commitment, "round": rnd}
                        for a in defects if a.kind == "COUNTER")
        log(f"{name} r{rnd}: attribution repairable={len(repairable)} "
            f"arbitrary={len(arbitrary)} stated={len(stated)} "
            f"ambiguous={len(ambig)}")

        if not attribution_ok:
            outcome = "inconclusive"
            note = f"attribution call failed: {ares.error or 'unparseable'}"
            rec["requests"] = sr.requests + ares.requests
            rec["tokens"] = sr.tokens + ares.tokens
            rounds.append(rec)
            break
        if not repairable:
            outcome = "converged_fixed_point"
            note = (f"no repairable contract defects among {len(attrs)} "
                    f"failing assertion(s)"
                    + (f"; {len(arbitrary)} ARBITRARY (test defects)"
                       if arbitrary else "")
                    + (f"; {len(ambig)} ambiguous" if ambig else ""))
            rec["requests"] = sr.requests + ares.requests
            rec["tokens"] = sr.tokens + ares.tokens
            rounds.append(rec)
            break

        # --- repair -----------------------------------------------------
        ro = repair(unit_dir, instruction, repairable, added_rows,
                    hidden_funcs, api_key, f"loop/{name}#r{rnd}",
                    judge_models, max(0, budget - total_req))
        guard.check("repair")
        total_req += ro.requests
        total_tok += ro.tokens
        if not ro.applied:
            outcome = "repair_rejected"
            note = ro.note or "; ".join(ro.violations)
            rec["repair"] = {"applied": False, "violations": ro.violations,
                             "requests": ro.requests, "tokens": ro.tokens}
            rec["requests"] = sr.requests + ares.requests + ro.requests
            rec["tokens"] = sr.tokens + ares.tokens + ro.tokens
            rounds.append(rec)
            break

        new_instruction = ro.new_head + contract_head_tail(instruction)[1]
        rec["repair"] = {
            "applied": True, "model": ro.model,
            "diff": unified_diff(instruction, new_instruction),
            "lost_prior_rows": lost_rows(new_instruction, added_rows),
            "defects_addressed": [{"test": a.test, "verdict": a.verdict,
                                   "kind": a.kind, "commitment": a.commitment}
                                  for a in repairable],
            "arbitrary_left_alone": [{"test": a.test, "commitment": a.commitment}
                                     for a in arbitrary],
            "stated_left_alone": [{"test": a.test, "commitment": a.commitment}
                                  for a in stated],
            "ambiguous": [{"test": a.test, "commitment": a.commitment}
                          for a in ambig],
            "requests": ro.requests, "tokens": ro.tokens, "note": ro.note,
        }
        # remember commitments we added so later rounds may not drop them
        for a in repairable:
            if a.commitment:
                added_rows.append(a.commitment[:200])
        instruction = new_instruction
        instr_path.write_text(instruction)
        lit_after = literal_budget(instruction, hidden_blob)
        rec["literals_after"] = [lit_after[0], lit_after[1]]
        rec["requests"] = rec["shadow"]["requests"] + ares.requests + ro.requests
        rec["tokens"] = rec["shadow"]["tokens"] + ares.tokens + ro.tokens
        rounds.append(rec)
        log(f"{name} r{rnd}: repaired {len(repairable)} row(s); "
            f"literals {lit_before[0]}->{lit_after[0]} "
            f"grounded {lit_before[2]:.0%}->{lit_after[2]:.0%}")
    else:
        outcome = "budget_exhausted"

    # --- final: cheat regen + preflight ------------------------------------
    pf = None
    if run_preflight and outcome != "error":
        log(f"{name}: preflight (bare/gold/fresh-cheat)")
        pf = preflight(unit_dir, api_key, shadow_models,
                       max(0, budget - total_req), guard)
        total_req += pf.requests
        total_tok += pf.tokens
        log(f"{name}: preflight bare={pf.bare_reward} gold={pf.gold_reward} "
            f"cheat={pf.cheat_reward} -> {pf.verdict}")
        if pf.cheat_patch:
            (unit_dir / "tests" / "cheat.patch").write_text(pf.cheat_patch)
            p2 = unit_dir / "patches" / "cheat.patch"
            if (unit_dir / "patches").is_dir():
                p2.write_text(pf.cheat_patch)

    if guard.mutations:
        note = (note + "; " if note else "") + \
            f"agent wrote into work tree {guard.mutations}x (restored)"
    return LoopResult(name, outcome, rounds, instruction, total_req,
                      total_tok, time.time() - t0, pf, note,
                      arb_gaps, ctr_gaps)


# --------------------------------------------------------------------------
# staging + batch driver
# --------------------------------------------------------------------------


def stage_name(unit_dir: Path) -> str:
    """``sweep_goa_L2/exprhash-L2`` -> ``goa-exprhash-L2loop``;
    ``tasks_batch2/bbolt/page-L2`` -> ``bbolt-page-L2loop``;
    ``sweep_climb_L3/helm-repindex-L3`` -> ``helm-repindex-L2loop``.

    Unit names that already carry a ``<repo>-`` prefix are kept; bare names
    get the repo dir (or the sweep's repo component) prepended."""
    base = re.sub(r"-L\d+.*$", "", unit_dir.name)
    if "-" in base:
        return f"{base}-L2loop"
    parent = unit_dir.parent.name
    repo = re.sub(r"^sweep_", "", parent)
    repo = re.sub(r"[_-]L\d+$", "", repo)
    return f"{repo}-{base}-L2loop"


def stage_unit(work_unit: Path, src_unit: Path, result: LoopResult,
               stage_root: Path, record: dict) -> Path:
    """Copy the loop's final work dir to ``stage_root/<staged name>`` with the
    repaired contract and regenerated cheat; write the round record into it.
    Never overwrites: a staged dir that exists is left and reported.
    ``src_unit`` supplies the name — the work copy's parent is always
    ``work`` so it cannot contribute the repo prefix."""
    dest = stage_root / stage_name(src_unit)
    if dest.exists():
        log(f"stage: {dest} exists — leaving as-is")
        return dest
    shutil.copytree(work_unit, dest, symlinks=True)
    (dest / "loop_record.json").write_text(json.dumps(record, indent=1) + "\n")
    return dest


def _result_row(r: LoopResult) -> dict:
    return {
        "unit": r.unit, "outcome": r.outcome, "note": r.note,
        "unit_verdict": r.unit_verdict, "fair_at_rung": r.fair_at_rung,
        "rounds": len(r.rounds), "requests": r.total_requests,
        "tokens": r.total_tokens, "seconds": round(r.seconds, 1),
        "preflight": (None if r.preflight is None else {
            "bare": r.preflight.bare_reward, "gold": r.preflight.gold_reward,
            "cheat": r.preflight.cheat_reward, "verdict": r.preflight.verdict}),
    }


def done_units(path: Path = RESULTS_PATH) -> set[str]:
    if not path.is_file():
        return set()
    out = set()
    for line in path.read_text().splitlines():
        try:
            d = json.loads(line)
            if d["outcome"] in TERMINAL:
                out.add(d["unit"])
        except (json.JSONDecodeError, KeyError):
            continue
    return out


def loop_batch(unit_dirs: list[Path],
               stage_root: Path | None = None,
               max_rounds: int = MAX_ROUNDS,
               max_requests: int = MAX_REQUESTS,
               shadow_models: Iterable[str] = shadow.SHADOW_MODELS,
               judge_models: Iterable[str] = JUDGE_MODELS,
               run_preflight: bool = True) -> list[LoopResult]:
    api_key = shadow.load_api_key()
    LOOP_DIR.mkdir(parents=True, exist_ok=True)
    done = done_units()
    spent = 0
    results: list[LoopResult] = []
    for u in unit_dirs:
        if u.name in done:
            log(f"{u.name}: already looped, skipping")
            continue
        if spent >= max_requests:
            log(f"request budget {max_requests} exhausted")
            break
        log(f"looping {u.name} (spent {spent}/{max_requests})")
        try:
            r = loop_unit(u, api_key, max_rounds=max_rounds,
                          shadow_models=shadow_models, judge_models=judge_models,
                          budget=max_requests - spent,
                          run_preflight=run_preflight)
        except Exception as e:  # noqa: BLE001 — a crashed unit must not kill the batch
            log(f"  {u.name}: ERROR {type(e).__name__}: {e}")
            r = LoopResult(u.name, "error", [], "", 0, 0, 0, None,
                           f"{type(e).__name__}: {e}")
        spent += r.total_requests
        results.append(r)
        rec = {"unit": r.unit, "source": str(u.resolve()), "outcome": r.outcome,
               "note": r.note, "unit_verdict": r.unit_verdict,
               "fair_at_rung": r.fair_at_rung,
               "arbitrary_gaps": r.arb_gaps or [],
               "counter_gaps": r.ctr_gaps or [],
               "rounds": r.rounds,
               "totals": {"requests": r.total_requests,
                          "tokens": r.total_tokens,
                          "seconds": round(r.seconds, 1)},
               "preflight": (None if r.preflight is None else {
                   "bare": r.preflight.bare_reward,
                   "gold": r.preflight.gold_reward,
                   "cheat": r.preflight.cheat_reward,
                   "verdict": r.preflight.verdict,
                   "note": r.preflight.note}),
               "final_instruction_sha": hashlib.sha256(
                   r.final_instruction.encode()).hexdigest()[:16]}
        ROUND_DIR.mkdir(parents=True, exist_ok=True)
        (ROUND_DIR / f"{u.name}.json").write_text(json.dumps(rec, indent=1) + "\n")
        with RESULTS_PATH.open("a") as f:
            f.write(json.dumps(_result_row(r)) + "\n")
        if stage_root is not None:
            dest = stage_unit(WORK_DIR / u.name, u, r, stage_root, rec)
            log(f"  staged -> {dest}")
        log(f"  {u.name}: {r.outcome} rounds={len(r.rounds)} "
            f"req={r.total_requests} tok={r.total_tokens} {r.seconds:.0f}s")
    return results

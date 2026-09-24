"""DETAILS gate: judge every ``_author/DETAILS.md`` commitment for derivability
before the verifier turns it into a graded assertion.

Why: the verifier is blind to gold. It works from DETAILS.md + api.md + the
excised tree, so it can only grade what DETAILS.md lists. When an author lists
an arbitrary choice as a commitment -- an exact error literal, a type spelling,
test scaffolding -- the verifier faithfully grades it and no solver can ever
derive it. Gold still passes preflight; the task is unfair at every rung.

Per line this emits:

* the author's ``Inferable:`` annotation if present (normalised to
  ``yes``/``doc``/``partially``/``no``, ``none`` when absent);
* an independent Composer judgement of the same question
  (``ARBITRARY``/``DERIVABLE``/``COUNTER``);
* a recommended action (``grade``/``grade-shape-only``/``drop``).

The model sees ONLY solver-visible information: api.md, bugreport.md, and the
excision diff with removed lines and test-file hunks stripped (the ``-`` lines
are the excised implementation and the deleted in-tree tests -- gold, which
neither the solver nor this gate may see). Composer answers are cached by
content hash through ``ask_composer``.

Unit verdicts:
* ``fail``               -- the model judgement is unavailable, or nothing is
                           left to grade;
* ``low-discrimination`` -- more than half of the DETAILS lines are ARBITRARY;
* ``pass``               -- otherwise.

The verdict lands in ``_author/details_gate.json``; ``pipeline.verifier``
refuses to verify a unit whose DETAILS.md has no passing gate verdict.
"""

from __future__ import annotations

import hashlib
import json
import logging
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path

LOG = logging.getLogger("details_gate")

KINDS = ("ARBITRARY", "DERIVABLE", "COUNTER")
ACTIONS = ("grade", "grade-shape-only", "drop")
GATE_FILE = "details_gate.json"
PROMPT_VERSION = "v1"

_KIND_TO_ACTION = {"DERIVABLE": "grade", "COUNTER": "grade"}

# ---------------------------------------------------------------------------
# DETAILS.md parsing
# ---------------------------------------------------------------------------

_NUMBERED = re.compile(r"^\s*(\d+)[.)]\s+")

# Annotation forms, in priority order. The author's judgement always sits at
# the tail of the (possibly wrapped) numbered entry.
_ANNOTATION_RES = (
    # structured: "Inferable: no/doc/partially/yes/<free text>"
    re.compile(r"[Ii]nferable:\s*(?P<val>[^\n]+)"),
    # prose: "not inferable", "Not inferable — ...", "uninferable"
    re.compile(r"(?i)\b(?:not|non)\s+inferable\b"),
    re.compile(r"(?i)\bpartially\s+inferable\b"),
    re.compile(r"(?i)\binferable\b"),
)


def _norm_structured(val: str) -> str:
    head = val.strip().lower()
    if re.match(r"^no\b", head):
        return "no"
    if re.match(r"^yes\b", head):
        return "yes"
    if re.match(r"^doc\b", head):
        return "doc"
    if re.match(r"^part", head):
        return "partially"
    # free text: e.g. "GetBody doc hints", "the variable is exported — partially"
    if "partial" in head:
        return "partially"
    if "doc" in head:
        return "doc"
    if re.search(r"\bnot\b", head):
        return "no"
    return "partially"


def find_annotation(text: str) -> tuple[str | None, str | None, int | None]:
    """Return (author_value, raw_annotation, start_pos) for the tail annotation.

    The author's judgement is the last annotation-like span in the entry.
    Overlapping matches ("inferable" inside "Not inferable" or "Inferable:")
    resolve toward the more specific pattern. A bare "inferable" that ends in
    the first half of the entry is content, not a judgement, unless it is the
    structured ``Inferable:`` form.
    """
    cands: list[tuple[int, int, int, str]] = []  # (end, -pat_idx, start, val)
    for pidx, pat in enumerate(_ANNOTATION_RES):
        for m in pat.finditer(text):
            if pidx == 0:
                val = _norm_structured(m.group("val") or "")
            else:
                g = m.group(0).lower()
                if "not" in g or "non" in g:
                    val = "no"
                elif "partially" in g:
                    val = "partially"
                else:
                    val = "yes"
            cands.append((m.end(), -pidx, m.start(), val))
    if not cands:
        return None, None, None
    cands.sort(reverse=True)
    end, _, start, val = cands[0]
    is_structured = text[start : start + 10].lower().startswith("inferable:")
    if end < len(text) * 0.5 and not is_structured:
        return None, None, None
    return val, text[start:].strip(), start


@dataclass
class DetailLine:
    n: int
    raw_text: str          # the whole numbered entry as written
    commitment: str        # entry text minus the inferable annotation (model-visible)
    author_value: str      # yes/doc/partially/no/none
    author_raw: str        # annotation text as written, "" when none


def parse_details(text: str) -> list[DetailLine]:
    """Split DETAILS.md into numbered entries, handling wrapped lines."""
    lines = text.splitlines()
    blocks: list[tuple[int, list[str]]] = []
    cur: tuple[int, list[str]] | None = None
    for ln in lines:
        m = _NUMBERED.match(ln)
        if m:
            if cur is not None:
                blocks.append(cur)
            cur = (int(m.group(1)), [ln[m.end():].strip()])
        elif cur is not None:
            cur[1].append(ln.rstrip())
    if cur is not None:
        blocks.append(cur)

    out: list[DetailLine] = []
    for n, parts in blocks:
        raw = " ".join(p.strip() for p in parts if p.strip()).strip()
        val, ann, start = find_annotation(raw)
        if ann is not None and start is not None:
            commitment = raw[:start].rstrip(" —-;,.(").strip()
        else:
            commitment = raw
        out.append(
            DetailLine(
                n=n,
                raw_text=raw,
                commitment=commitment or raw,
                author_value=val or "none",
                author_raw=ann or "",
            )
        )
    return out


# ---------------------------------------------------------------------------
# Excision diff filtering: keep only what the solver sees
# ---------------------------------------------------------------------------

_TEST_FILE = re.compile(r"(?:^|/)(?:[^/]*_test\.[a-z]+|testdata/|tests?/)", re.IGNORECASE)


def solver_visible_diff(patch_text: str, *, max_chars: int = 60_000) -> str:
    """Filter an excision diff to solver-visible content only.

    Drop ``-`` lines (the excised implementation -- gold) and entire hunks on
    test files (the deleted in-tree tests -- also gold). Keep ``+`` lines (the
    stub declarations the solver sees), context lines (surviving code), and
    headers (file paths).
    """
    out: list[str] = []
    in_test_file = False
    for ln in patch_text.splitlines():
        if ln.startswith(("diff --git", "--- ")):
            m = re.search(r"\bb/(\S+)", ln) or re.search(r"\ba/(\S+)", ln)
            if ln.startswith("diff --git"):
                in_test_file = bool(m and _TEST_FILE.search(m.group(1)))
            if in_test_file:
                continue
            out.append(ln)
            continue
        if in_test_file:
            continue
        if ln.startswith("-"):
            continue  # removed implementation: not solver-visible
        out.append(ln)
    text = "\n".join(out)
    return text[:max_chars]


# ---------------------------------------------------------------------------
# The judgement call
# ---------------------------------------------------------------------------

PROMPT = """You are auditing numbered behavioural commitments in a task spec. A verifier will
turn each into a graded test for a coding agent. The agent sees ONLY the repository tree
with the implementation excised (function bodies replaced by stubs, in-tree tests deleted)
plus a bug report. It never sees the hidden tests, the original code, or the deleted tests.

A commitment is FAIR only if a competent engineer holding exactly that information would
PRODUCE the graded behaviour -- not merely guess it. For each numbered commitment, decide
where its answer LIVES:

  ARBITRARY  nowhere. An authorial choice nothing in the surviving tree implies -- an
             exact error-message literal, a specific type spelling, punctuation, a magic
             constant, internal test scaffolding. Every solver fails it for the same
             non-reason.
  DERIVABLE  in the tree. A real invariant the surviving code, docs, or callers imply --
             nil-safety, non-negativity, a documented format, consistency with a
             neighbouring function. A careful engineer gets there.
  COUNTER    in the tree, but the obvious reading is wrong and the correct rule is
             discoverable there. Separates solvers that read code from solvers that
             pattern-match training data.

Then pick the grading action:

  grade            assert the commitment as written (DERIVABLE or COUNTER)
  grade-shape-only assert only its derivable shape -- "an error is returned", "the field
                   is set", "the list is ordered" -- never the arbitrary literal itself
                   (ARBITRARY but a weaker shape survives)
  drop             do not grade it; even its shape is arbitrary (ARBITRARY and nothing
                   derivable remains, e.g. test scaffolding)

Reply with one line per commitment, exactly:

  <n>|<ARBITRARY|DERIVABLE|COUNTER>|<grade|grade-shape-only|drop>|<=10 word reason>

=== API SURFACE (solver-visible) ===
{api}

=== BUG REPORT (solver-visible) ===
{bugreport}

=== EXCISION DIFF, solver-visible part (stub declarations + surviving code; removed
implementation and deleted tests are NOT shown because the solver cannot see them) ===
{diff}

=== COMMITMENTS ===
{items}
"""


@dataclass
class LineJudgement:
    kind: str = "UNKNOWN"
    action: str = "drop"
    reason: str = ""
    raw: str = ""


def parse_answer(answer: str, lines: list[DetailLine]) -> dict[int, LineJudgement]:
    out: dict[int, LineJudgement] = {}
    for ln in answer.splitlines():
        ln = ln.strip()
        if not ln:
            continue
        parts = [p.strip() for p in ln.split("|")]
        if len(parts) < 3:
            continue
        num = parts[0].rstrip(".")
        if not num.isdigit():
            continue
        kind = parts[1].upper()
        if kind not in KINDS:
            continue
        action = parts[2].lower()
        if action not in ACTIONS:
            action = _KIND_TO_ACTION.get(kind, "drop")
        # only repair contradictory pairs; DERIVABLE|grade-shape-only is a real
        # judgement (a derivable commitment carrying an arbitrary literal)
        if kind == "ARBITRARY" and action == "grade":
            action = "grade-shape-only"
        if kind in _KIND_TO_ACTION and action == "drop":
            action = _KIND_TO_ACTION[kind]
        out[int(num)] = LineJudgement(
            kind=kind,
            action=action,
            reason=parts[3] if len(parts) > 3 else "",
            raw=ln,
        )
    # deterministic default for lines the model missed
    for d in lines:
        out.setdefault(d.n, LineJudgement())
    return out


def build_prompt(author_dir: Path, lines: list[DetailLine]) -> str:
    api = (author_dir / "api.md").read_text(errors="replace") if (author_dir / "api.md").is_file() else "(none)"
    bug = (author_dir / "bugreport.md").read_text(errors="replace") if (author_dir / "bugreport.md").is_file() else "(none)"
    patch_path = author_dir / "excised" / "excision.patch"
    if not patch_path.is_file():
        cands = list(author_dir.glob("*.patch")) + list(author_dir.glob("excised/*.patch"))
        patch_text = cands[0].read_text(errors="replace") if cands else ""
    else:
        patch_text = patch_path.read_text(errors="replace")
    diff = solver_visible_diff(patch_text) or "(no excision diff available)"
    items = "\n".join(f"{d.n}. {d.commitment}" for d in lines)
    return PROMPT.format(api=api, bugreport=bug, diff=diff, items=items)


def unit_verdict(lines: list[DetailLine], judgements: dict[int, LineJudgement], *, model_ok: bool) -> str:
    if not model_ok:
        return "fail"
    arb = sum(1 for d in lines if judgements[d.n].kind == "ARBITRARY")
    gradeable = sum(1 for d in lines if judgements[d.n].action != "drop")
    if gradeable == 0:
        return "fail"
    if lines and arb > len(lines) / 2:
        return "low-discrimination"
    return "pass"


@dataclass
class UnitGate:
    repo: str
    unit: str
    author_dir: Path
    verdict: str
    lines: list[DetailLine] = field(default_factory=list)
    judgements: dict[int, LineJudgement] = field(default_factory=dict)
    answer: str = ""
    error: str = ""

    def to_json(self) -> dict:
        return {
            "repo": self.repo,
            "unit": self.unit,
            "family": f"{self.repo}-{self.unit}",
            "verdict": self.verdict,
            "counts": {
                "lines": len(self.lines),
                "ARBITRARY": sum(1 for d in self.lines if self.judgements[d.n].kind == "ARBITRARY"),
                "DERIVABLE": sum(1 for d in self.lines if self.judgements[d.n].kind == "DERIVABLE"),
                "COUNTER": sum(1 for d in self.lines if self.judgements[d.n].kind == "COUNTER"),
                "UNKNOWN": sum(1 for d in self.lines if self.judgements[d.n].kind == "UNKNOWN"),
                "grade": sum(1 for d in self.lines if self.judgements[d.n].action == "grade"),
                "grade-shape-only": sum(1 for d in self.lines if self.judgements[d.n].action == "grade-shape-only"),
                "drop": sum(1 for d in self.lines if self.judgements[d.n].action == "drop"),
            },
            "lines": [
                {
                    "n": d.n,
                    "text": d.raw_text,
                    "commitment": d.commitment,
                    "author_inferable": d.author_value,
                    "author_annotation": d.author_raw,
                    "kind": self.judgements[d.n].kind,
                    "action": self.judgements[d.n].action,
                    "reason": self.judgements[d.n].reason,
                }
                for d in self.lines
            ],
            "answer": self.answer,
            "error": self.error,
        }


# ---------------------------------------------------------------------------
# Driver
# ---------------------------------------------------------------------------


def _load_ask():
    ops = Path(__file__).resolve().parents[2] / "scripts" / "ops"
    sys.path.insert(0, str(ops))
    from ask_composer import ask

    return ask


def discover_details(roots: list[Path]) -> list[tuple[str, str, Path]]:
    """[(repo, unit, _author dir)] for every _author/DETAILS.md under roots."""
    found: list[tuple[str, str, Path]] = []
    seen: set[Path] = set()
    for root in roots:
        for path in sorted(root.rglob("DETAILS.md")):
            author = path.parent
            if author.name != "_author" or author in seen:
                continue
            seen.add(author)
            unit = author.parent.name
            repo = author.parent.parent.name
            found.append((repo, unit, author))
    return found


def gate_unit(repo: str, unit: str, author_dir: Path, *, ask_fn=None, model: str | None = None) -> UnitGate:
    details_path = author_dir / "DETAILS.md"
    lines = parse_details(details_path.read_text(errors="replace"))
    prompt = build_prompt(author_dir, lines)
    content_hash = hashlib.sha256(prompt.encode()).hexdigest()[:24]
    ask_fn = ask_fn or _load_ask()
    gate = UnitGate(repo=repo, unit=unit, author_dir=author_dir, verdict="fail", lines=lines)
    if not lines:
        gate.error = "no numbered commitments in DETAILS.md"
        return gate
    try:
        if model:
            answer = ask_fn(prompt, cache_key=f"detailsgate/{PROMPT_VERSION}/{repo}-{unit}/{content_hash}", model=model)
        else:
            answer = ask_fn(prompt, cache_key=f"detailsgate/{PROMPT_VERSION}/{repo}-{unit}/{content_hash}")
    except Exception as exc:  # noqa: BLE001
        gate.error = str(exc)
        gate.judgements = {d.n: LineJudgement() for d in lines}
        return gate
    if answer.startswith("__ERROR__"):
        gate.error = answer
        gate.judgements = {d.n: LineJudgement() for d in lines}
        return gate
    gate.answer = answer
    gate.judgements = parse_answer(answer, lines)
    gate.verdict = unit_verdict(lines, gate.judgements, model_ok=True)
    return gate


def write_gate_file(gate: UnitGate) -> Path:
    path = gate.author_dir / GATE_FILE
    path.write_text(json.dumps(gate.to_json(), indent=2) + "\n", encoding="utf-8")
    return path


def gate_passed(author_dir: Path) -> bool:
    """Packaging check: a DETAILS.md unit may be verified only when its gate passed."""
    details = author_dir / "DETAILS.md"
    if not details.is_file():
        return True  # batch-1 units predate the gate
    path = author_dir / GATE_FILE
    if not path.is_file():
        return False
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return False
    return data.get("verdict") == "pass"


# ---------------------------------------------------------------------------
# Outcome validation: do gate flags predict double-failure?
# ---------------------------------------------------------------------------

_TRIAL_RE = re.compile(r"^(?P<fam>.+)-(?P<lvl>L\d+)$")


def collect_trial_verdicts(*roots: Path) -> dict[str, dict[str, str | None]]:
    """Scan harbor ``result.json`` files -> {family: {level: pass|fail|None}}.

    A family fails a level when every trial at that level scored 0; it passes
    when any trial scored 1. Trials with no reward are ignored.
    """
    fam: dict[str, dict[str, list[float]]] = {}
    for root in roots:
        if not root.is_dir():
            continue
        for fp in root.rglob("result.json"):
            if "__" not in fp.parent.name:
                continue
            try:
                j = json.loads(fp.read_text(errors="replace"))
            except (OSError, json.JSONDecodeError):
                continue
            m = _TRIAL_RE.match(str(j.get("task_name") or ""))
            if not m:
                continue
            vr = j.get("verifier_result") or {}
            rew = (vr.get("rewards") or {}).get("reward")
            fam.setdefault(m.group("fam"), {}).setdefault(m.group("lvl"), []).append(
                float(rew) if rew is not None else None
            )
    out: dict[str, dict[str, str | None]] = {}
    for f, lvls in fam.items():
        row: dict[str, str | None] = {}
        for lvl, rs in lvls.items():
            rs2 = [r for r in rs if r is not None]
            row[lvl] = None if not rs2 else ("pass" if any(r == 1.0 for r in rs2) else "fail")
        out[f] = row
    return out


def _confusion(records: list[dict]) -> dict:
    """rows: model kind (ARBITRARY/DERIVABLE/COUNTER/UNKNOWN); cols: author value."""
    cols = ("no", "doc", "partially", "yes", "none")
    matrix = {k: {c: 0 for c in cols} for k in KINDS + ("UNKNOWN",)}
    for rec in records:
        for ln in rec.get("lines", []):
            kind = ln.get("kind") or "UNKNOWN"
            av = ln.get("author_inferable") or "none"
            if av not in cols:
                av = "none"
            matrix.setdefault(kind, {c: 0 for c in cols})
            matrix[kind][av] += 1
    return matrix


def validate_outcomes(records: list[dict], verdicts: dict[str, dict[str, str | None]]) -> dict:
    """Join per-unit gate output to L0/L2 trial verdicts.

    Double-failure = family failed L0 AND L2 (all trials 0 at both). The gate's
    predictor under test: "unit has at least one ARBITRARY line" (and variants),
    reported as precision/recall against the double-failure base rate over the
    families that have both verdicts.
    """
    rows = []
    for rec in records:
        fam = rec.get("family") or f"{rec.get('repo')}-{rec.get('unit')}"
        v = verdicts.get(fam, {})
        c = rec.get("counts", {})
        rows.append({
            "family": fam,
            "verdict": rec.get("verdict"),
            "n_lines": c.get("lines", 0),
            "n_arb": c.get("ARBITRARY", 0),
            "n_der": c.get("DERIVABLE", 0),
            "n_ctr": c.get("COUNTER", 0),
            "n_drop": c.get("drop", 0),
            "n_shape": c.get("grade-shape-only", 0),
            "L0": v.get("L0"),
            "L2": v.get("L2"),
        })
    both = [r for r in rows if r["L0"] and r["L2"]]
    for r in both:
        r["double_fail"] = r["L0"] == "fail" and r["L2"] == "fail"

    def pr(pred):
        flagged = [r for r in both if pred(r)]
        tp = sum(1 for r in flagged if r["double_fail"])
        fn = sum(1 for r in both if r["double_fail"] and not pred(r))
        return {
            "n_flagged": len(flagged),
            "tp": tp,
            "precision": round(tp / len(flagged), 3) if flagged else None,
            "recall": round(tp / (tp + fn), 3) if (tp + fn) else None,
        }

    rules = {
        "any_ARBITRARY": lambda r: r["n_arb"] >= 1,
        "ge2_ARBITRARY": lambda r: r["n_arb"] >= 2,
        "gt_half_ARBITRARY": lambda r: r["n_lines"] and r["n_arb"] > r["n_lines"] / 2,
        "any_drop": lambda r: r["n_drop"] >= 1,
        "any_drop_or_shape": lambda r: (r["n_drop"] + r["n_shape"]) >= 1,
        "verdict_lowdisc": lambda r: r["verdict"] == "low-discrimination",
    }
    dbl = sum(1 for r in both if r["double_fail"])
    return {
        "n_records": len(records),
        "n_with_L0": sum(1 for r in rows if r["L0"]),
        "n_with_L2": sum(1 for r in rows if r["L2"]),
        "n_both": len(both),
        "n_double_fail": dbl,
        "double_fail_rate": round(dbl / len(both), 3) if both else None,
        "L0_fail_rate": round(
            sum(1 for r in rows if r["L0"] == "fail") / max(1, sum(1 for r in rows if r["L0"])), 3
        ),
        "rules": {name: pr(fn) for name, fn in rules.items()},
        "rows": rows,
    }


def report(records: list[dict], verdicts: dict[str, dict[str, str | None]]) -> dict:
    return {
        "confusion": _confusion(records),
        "outcomes": validate_outcomes(records, verdicts),
    }


def load_records(jsonl: Path) -> list[dict]:
    out = []
    if jsonl.is_file():
        for ln in jsonl.read_text().splitlines():
            try:
                out.append(json.loads(ln))
            except json.JSONDecodeError:
                pass
    return out


# ---------------------------------------------------------------------------
# CLI plumbing
# ---------------------------------------------------------------------------


def run_batch(
    units: list[tuple[str, str, Path]],
    out_jsonl: Path,
    *,
    workers: int = 3,
    model: str | None = None,
    write_gate_files: bool = True,
) -> list[UnitGate]:
    done_keys: set[str] = set()
    if out_jsonl.is_file():
        for ln in out_jsonl.read_text().splitlines():
            try:
                rec = json.loads(ln)
                done_keys.add(rec.get("family", ""))
            except json.JSONDecodeError:
                pass
    todo = [(r, u, a) for r, u, a in units if f"{r}-{u}" not in done_keys]
    LOG.info("details_gate: %d units total, %d done, %d to judge", len(units), len(done_keys), len(todo))
    if not todo:
        return []

    sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "scripts" / "ops"))
    from concurrent.futures import ThreadPoolExecutor

    from ask_composer import ask

    def prepare(t: tuple[str, str, Path]) -> UnitGate:
        repo, unit, author = t
        lines = parse_details((author / "DETAILS.md").read_text(errors="replace"))
        g = UnitGate(repo=repo, unit=unit, author_dir=author, verdict="fail", lines=lines)
        if not lines:
            g.error = "no numbered commitments in DETAILS.md"
            return g
        prompt = build_prompt(author, lines)
        content_hash = hashlib.sha256(prompt.encode()).hexdigest()[:24]
        g._cache_key = f"detailsgate/{PROMPT_VERSION}/{repo}-{unit}/{content_hash}"  # type: ignore[attr-defined]
        g._prompt = prompt  # type: ignore[attr-defined]
        return g

    def judge(g: UnitGate) -> UnitGate:
        if g.error:
            return g
        try:
            ans = ask(g._prompt, cache_key=g._cache_key, **({"model": model} if model else {}))  # type: ignore[attr-defined]
        except Exception as exc:  # noqa: BLE001
            ans = f"__ERROR__ {exc}"
        if not ans or ans.startswith("__ERROR__"):
            g.error = ans or "no answer"
            g.judgements = {d.n: LineJudgement() for d in g.lines}
        else:
            g.answer = ans
            g.judgements = parse_answer(ans, g.lines)
            g.verdict = unit_verdict(g.lines, g.judgements, model_ok=True)
        return g

    results: list[UnitGate] = []
    with out_jsonl.open("a") as fh, ThreadPoolExecutor(max_workers=workers) as ex:
        for g in ex.map(lambda t: judge(prepare(t)), todo):
            fh.write(json.dumps(g.to_json()) + "\n")
            fh.flush()
            if write_gate_files:
                try:
                    write_gate_file(g)
                except OSError as exc:
                    LOG.warning("could not write %s/%s: %s", g.author_dir, GATE_FILE, exc)
            LOG.info("%s-%s: verdict=%s lines=%d", g.repo, g.unit, g.verdict, len(g.lines))
            results.append(g)
    return results


def main(argv: list[str] | None = None) -> int:
    import argparse

    ap = argparse.ArgumentParser(description="Gate _author/DETAILS.md commitments before verification.")
    ap.add_argument("paths", nargs="*", help="_author dirs or authored roots (default: scan --roots)")
    ap.add_argument(
        "--roots",
        nargs="*",
        default=[
            "/home/evan/Documents/oswt-DETAILSGATE/experiments/pipeline",
            *[f"/home/evan/Documents/{d}/experiments/pipeline" for d in (
                "oswt-AUclientgo", "oswt-AUgin", "oswt-AUgoa", "oswt-AUgogithub", "oswt-AUhelm",
                "oswt-AUkops", "oswt-AUnatsserver", "oswt-AUnew1", "oswt-AUnew2", "oswt-XREPO20",
            )],
        ],
    )
    ap.add_argument("--out", default="outputs/details_gate.jsonl")
    ap.add_argument("--workers", type=int, default=3)
    ap.add_argument("--model", default=None)
    ap.add_argument("--no-gate-files", action="store_true", help="do not write _author/details_gate.json")
    ap.add_argument(
        "--report",
        metavar="TRIALS_ROOT",
        nargs="*",
        help="print confusion matrix + outcome validation from --out and trial result.json roots",
    )
    args = ap.parse_args(argv)

    if args.report is not None:
        logging.basicConfig(level=logging.INFO, format="%(asctime)s %(message)s")
        roots = [Path(p) for p in args.report] or [
            Path("/home/evan/Documents/open_swe_traces_research/experiments/dose_response")
        ]
        records = load_records(Path(args.out))
        verdicts = collect_trial_verdicts(*roots)
        rep = report(records, verdicts)
        print(json.dumps(rep, indent=2))
        return 0

    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(message)s")

    units: list[tuple[str, str, Path]] = []
    for p in args.paths:
        path = Path(p)
        if (path / "DETAILS.md").is_file() and path.name == "_author":
            units.append((path.parent.parent.name, path.parent.name, path))
        elif path.is_dir():
            units.extend(discover_details([path]))
    if not units:
        units = discover_details([Path(r) for r in args.roots if Path(r).is_dir()])
    # dedupe on family name, first wins (same unit may exist in several worktrees)
    seen: set[str] = set()
    uniq: list[tuple[str, str, Path]] = []
    for r, u, a in units:
        if f"{r}-{u}" in seen:
            continue
        seen.add(f"{r}-{u}")
        uniq.append((r, u, a))
    LOG.info("discovered %d unique DETAILS units", len(uniq))

    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    run_batch(uniq, out, workers=args.workers, model=args.model, write_gate_files=not args.no_gate_files)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

"""A2 (alternative correct fix) for free from real trials.

Given a completed trial that PASSED, compare its agent patch to the task's
``gold.patch``: a patch touching a different function set, or differing beyond
whitespace/renames, is an executed A2 verdict — the suite accepted a fix the
author did not write. Recorded on the task dir's ``validation.json`` at
``executed`` tier (the trial really ran).

Wired into trial recording (``pipeline/solve.py``) and backfillable from
``state.db`` + ``experiments/dose_response/jobs/``.
"""

from __future__ import annotations

import json
import re
import sqlite3
from pathlib import Path
from typing import Any

from openswe_traces.gate.core import EXECUTED, Verdict, _now_iso, record_verdicts, register
from openswe_traces.pipeline_ext.hack_audit import find_trial_patch, parse_touched_paths

_IDENT_RE = re.compile(r"[A-Za-z_][A-Za-z0-9_]*")
_HUNK_CTX_RE = re.compile(r"^@@ -\d+(?:,\d+)? \+\d+(?:,\d+)? @@\s*(.*)$", re.MULTILINE)
_FUNC_DECL_RES = (
    re.compile(r"func\s+(?:\(\s*\w+\s+\*?\w+\s*\)\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\("),
    re.compile(r"def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\("),
    re.compile(r"(?:function\s+|)([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:async\s+)?(?:\(|function)"),
)
_JOB_TAIL_RE = re.compile(r"-[A-Za-z0-9_]+-n\d+-k\d+$|-[A-Za-z0-9_.-]+-seq\d+$")


def _changed_lines(patch: str) -> list[str]:
    out: list[str] = []
    for ln in patch.splitlines():
        if ln.startswith(("+++", "---")):
            continue
        if ln[:1] in {"+", "-"}:
            out.append(ln[1:])
    return out


def _skeleton(line: str) -> str:
    """Whitespace- and identifier-insensitive shape of a line.

    A pure rename collapses to the same skeleton; a different expression
    structure does not. Single-token swaps look like renames — accepted
    limitation, documented in compare().
    """
    return _IDENT_RE.sub("_", "".join(line.split()))


def _symbols_touched(patch: str) -> set[str]:
    syms: set[str] = set()
    for ctx in _HUNK_CTX_RE.findall(patch):
        m = re.search(r"([A-Za-z_][A-Za-z0-9_.]*)\s*\(", ctx)
        if m:
            syms.add(m.group(1))
    for rx in _FUNC_DECL_RES:
        syms.update(rx.findall(patch))
    syms.discard("")
    return syms


def _jaccard(a: set, b: set) -> float:
    if not a and not b:
        return 1.0
    union = a | b
    return len(a & b) / len(union) if union else 1.0


def compare_patches(agent_patch: str, gold_patch: str) -> dict[str, Any]:
    """Structural comparison of an accepted agent patch vs gold."""
    a_lines = {_skeleton(l) for l in _changed_lines(agent_patch) if l.strip()}
    g_lines = {_skeleton(l) for l in _changed_lines(gold_patch) if l.strip()}
    a_syms, g_syms = _symbols_touched(agent_patch), _symbols_touched(gold_patch)
    a_files, g_files = set(parse_touched_paths(agent_patch)), set(
        parse_touched_paths(gold_patch)
    )
    jac = _jaccard(a_lines, g_lines)
    same_skeletons = a_lines == g_lines
    alternative = bool(agent_patch.strip()) and (
        a_syms != g_syms or a_files != g_files or not same_skeletons
    )
    return {
        "jaccard": round(jac, 4),
        "symbols_agent": sorted(a_syms),
        "symbols_gold": sorted(g_syms),
        "files_agent": sorted(a_files),
        "files_gold": sorted(g_files),
        "rename_or_ws_only": same_skeletons and a_syms != g_syms,
        "alternative": alternative and not same_skeletons or (a_syms != g_syms),
    }


def _trial_reward(trial_dir: Path) -> float | None:
    for cand in (trial_dir / "result.json",):
        if cand.is_file():
            try:
                data = json.loads(cand.read_text(encoding="utf-8", errors="replace"))
            except (OSError, json.JSONDecodeError):
                continue
            for key in ("reward", "verifier_result"):
                val = data.get(key)
                if isinstance(val, (int, float)):
                    return float(val)
                if isinstance(val, dict) and isinstance(val.get("reward"), (int, float)):
                    return float(val["reward"])
    return None


def _gold_text(task_dir: Path) -> str:
    for rel in ("tests/gold.patch", "patches/gold.patch"):
        p = task_dir / rel
        if p.is_file():
            return p.read_text(encoding="utf-8", errors="replace")
    return ""


def record_trial(
    task_dir: Path | str,
    trial_dir: Path | str | None = None,
    *,
    patch: str | None = None,
    trial_id: str = "",
    reward: float | None = None,
    dry: bool = False,
) -> Verdict | None:
    """Evaluate one passed trial's patch vs gold; record an executed A2 verdict.

    Returns the verdict written, or None when the trial yields no usable patch
    / did not pass / the task has no gold.patch.
    """
    td = Path(task_dir)
    patch_text = find_trial_patch(trial_dir, patch)
    if not patch_text.strip():
        return None
    if reward is None and trial_dir is not None:
        reward = _trial_reward(Path(trial_dir))
    if reward is not None and reward < 1.0:
        return None  # a failed trial is not alternative-correct evidence
    gold = _gold_text(td)
    if not gold.strip():
        return None
    cmp = compare_patches(patch_text, gold)
    tid = trial_id or (Path(trial_dir).name if trial_dir else "trial")
    if cmp["alternative"]:
        v = Verdict(
            "A2",
            True,
            False,
            EXECUTED,
            f"alternative correct fix accepted: {tid}, jaccard {cmp['jaccard']} vs gold",
            _now_iso(),
            "gate/alt_fix",
        )
    else:
        v = Verdict(
            "A2",
            False,
            True,
            EXECUTED,
            f"trial {tid} patch matches gold (jaccard {cmp['jaccard']})",
            _now_iso(),
            "gate/alt_fix",
        )
    if not dry:
        record_verdicts(td, [v])
    return v


_TRIAL_ID_RE = re.compile(r"__p?[A-Za-z0-9]{4,}$")


def resolve_task_dir(name: str, tasks_roots: list[Path]) -> Path | None:
    """'<repo>-<unit>-L<n>-<solver>...' / '<unit>-L<n>__<id>' → task dir.

    ``tasks_roots`` may hold flat dirs (dose_response/tasks) or
    ``<repo>/<unit>-L<n>`` two-level trees (pipeline tasks)."""
    base = _JOB_TAIL_RE.sub("", name)
    base = _TRIAL_ID_RE.sub("", base)
    for root in tasks_roots:
        if not root.is_dir():
            continue
        if (root / base).is_dir():
            return root / base
        if "-" in base:
            repo, rest = base.split("-", 1)
            cand = root / repo / rest
            if cand.is_dir():
                return cand
        # repo prefix unknown: scan one level; match multi-word repo names
        for repo_dir in sorted(root.iterdir()):
            if not repo_dir.is_dir():
                continue
            if (repo_dir / base).is_dir():
                return repo_dir / base
            prefix = repo_dir.name + "-"
            if base.startswith(prefix) and (repo_dir / base[len(prefix):]).is_dir():
                return repo_dir / base[len(prefix):]
            if "-" in base:
                cand = repo_dir / base.split("-", 1)[-1]
                if cand.is_dir():
                    return cand
    return None


def backfill_state_db(
    db_path: Path | str,
    tasks_roots: list[Path],
    *,
    dry: bool = False,
) -> dict[str, Any]:
    """Record A2 verdicts for passed trials already in the pipeline state DB."""
    roots = [Path(r) for r in tasks_roots]
    out: dict[str, Any] = {"recorded": [], "equivalent": [], "skipped": []}
    con = sqlite3.connect(str(db_path))
    try:
        rows = con.execute(
            "select id, repo, unit, level, job_dir, reward from trials "
            "where reward >= 1.0 and coalesce(excluded, 0) = 0"
        ).fetchall()
    finally:
        con.close()
    for tid, repo, unit, level, job_dir, reward in rows:
        td = resolve_task_dir(f"{repo}-{unit}-L{level}", roots)
        if td is None:
            out["skipped"].append(f"trial {tid}: no task dir for {repo}/{unit}-L{level}")
            continue
        trial_root = Path(job_dir) if job_dir else None
        if trial_root is not None and trial_root.is_dir():
            # job dir may hold per-trial subdirs; find the one with a patch
            subs = [
                p
                for p in [trial_root, *sorted(trial_root.iterdir())]
                if p.is_dir() or p == trial_root
            ]
        else:
            subs = []
        verdict = None
        for sub in subs:
            verdict = record_trial(td, sub, trial_id=f"db:{tid}", reward=reward, dry=dry)
            if verdict is not None:
                break
        if verdict is None:
            out["skipped"].append(f"trial {tid}: no agent patch / no gold")
        elif verdict.skipped:
            out["equivalent"].append(f"trial {tid} -> {td.name}")
        else:
            out["recorded"].append(f"trial {tid} -> {td.name}")
    return out


def backfill_jobs(
    jobs_root: Path | str,
    tasks_roots: list[Path],
    *,
    dry: bool = False,
) -> dict[str, Any]:
    """Walk job dirs; resolve the task from each trial dir name; record A2."""
    jobs = Path(jobs_root)
    roots = [Path(r) for r in tasks_roots]
    out: dict[str, Any] = {"recorded": [], "equivalent": [], "skipped": []}
    if not jobs.is_dir():
        return out

    def trials_of(job: Path) -> list[Path]:
        subs = [p for p in sorted(job.iterdir()) if p.is_dir()]
        trials = [p for p in subs if (p / "result.json").is_file()]
        if not trials:
            trials = [p for p in subs if find_trial_patch(p)]
        if (job / "result.json").is_file() or find_trial_patch(job):
            trials.insert(0, job)
        return trials

    for job in sorted(jobs.iterdir()):
        if not job.is_dir():
            continue
        trials = trials_of(job)
        if not trials:
            out["skipped"].append(f"{job.name}: no trials")
            continue
        for trial in trials:
            reward = _trial_reward(trial)
            if reward is None or reward < 1.0:
                continue
            td = resolve_task_dir(trial.name, roots) or resolve_task_dir(job.name, roots)
            if td is None:
                out["skipped"].append(f"{job.name}/{trial.name}: no task dir")
                continue
            v = record_trial(td, trial, trial_id=trial.name, reward=reward, dry=dry)
            if v is None:
                out["skipped"].append(f"{job.name}/{trial.name}: no agent patch")
            elif v.skipped:
                out["equivalent"].append(f"{job.name}/{trial.name} -> {td.name}")
            else:
                out["recorded"].append(f"{job.name}/{trial.name} -> {td.name}")
    return out


# --- executed impl: gate needs an executed verdict slot for A2 ------------------


@register("A2", EXECUTED, provenance="gate/alt_fix", description="passed trial patch differs from gold")
def _a2(ctx: Any) -> Verdict:
    """Executed slot for A2: pass when a recorded alternative exists.

    The real evidence is written by ``record_trial`` at trial time; this impl
    re-states the latest recorded executed verdict (or marks the rule as
    never-yet-evaluated, which is a skip, not a failure).
    """
    from openswe_traces.gate.core import load_validation, stored_verdict_rows, verdict_from_dict

    rows = stored_verdict_rows(load_validation(ctx.task_dir))
    prior = [
        v
        for v in (verdict_from_dict(r) for r in rows)
        if v and v.rule_id == "A2" and v.tier == EXECUTED and v.provenance == "gate/alt_fix"
    ]
    if prior:
        latest = max(prior, key=lambda v: v.produced_at)
        return latest
    return Verdict(
        "A2",
        False,
        True,
        EXECUTED,
        "no passing trial evaluated for this task",
        _now_iso(),
        "gate/alt_fix",
    )

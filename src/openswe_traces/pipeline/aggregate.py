"""results.parquet + live results.md dashboard."""

from __future__ import annotations

from collections import Counter
from pathlib import Path
from typing import Any

import pandas as pd

from openswe_traces.pipeline.config import PipelineConfig
from openswe_traces.pipeline.state import PipelineStore

RESULT_COLUMNS = (
    "repo",
    "unit",
    "family",
    "closure_size",
    "lines",
    "solver",
    "flip",
    "audit_class",
    "tokens_in",
    "tokens_out",
    "wall_minutes",
    "L0_pass",
    "L0_n",
    "L1_pass",
    "L1_n",
    "L2_pass",
    "L2_n",
    "L3_pass",
    "L3_n",
    "L4_pass",
    "L4_n",
    "L5_pass",
    "L5_n",
    "L6_pass",
    "L6_n",
    "rejected_rule",
    "status",
)


def task_rows(store: PipelineStore) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    for unit in store.list_units():
        repo = unit["repo"]
        name = unit["unit"]
        trials = store.list_trials(repo=repo, unit=name, include_excluded=False)
        solvers = sorted({t.solver for t in trials if t.solver})
        if not solvers:
            rows.append(
                {
                    "repo": repo,
                    "unit": name,
                    "family": unit["family"] or "",
                    "closure_size": unit["n_files"] or 0,
                    "lines": unit["n_lines"] or 0,
                    "solver": "",
                    "flip": None,
                    "audit_class": "",
                    "tokens_in": 0,
                    "tokens_out": 0,
                    "wall_minutes": 0.0,
                    "rejected_rule": unit["rejected_rule"] or "",
                    "status": unit["status"],
                    **{f"L{lv}_pass": 0 for lv in range(7)},
                    **{f"L{lv}_n": 0 for lv in range(7)},
                }
            )
            continue
        for solver in solvers:
            st = [t for t in trials if t.solver == solver]
            per_level = {lv: [t for t in st if t.level == lv] for lv in range(7)}
            flip = None
            for lv in range(7):
                scored = [t for t in per_level[lv] if t.audit_class not in {"contaminated", "checksum"}]
                p = sum(1 for t in scored if t.reward == 1.0)
                n = len(scored)
                need_n = 1 if lv == 0 else 3
                need_p = 1 if lv == 0 else 2
                if n >= need_n and p >= need_p:
                    flip = lv
                    break
            if flip is None and per_level[2]:
                scored = [t for t in per_level[2] if t.audit_class not in {"contaminated", "checksum"}]
                p = sum(1 for t in scored if t.reward == 1.0)
                if len(scored) >= 3 and p >= 2:
                    flip = 2
            row: dict[str, Any] = {
                "repo": repo,
                "unit": name,
                "family": unit["family"] or "",
                "closure_size": unit["n_files"] or 0,
                "lines": unit["n_lines"] or 0,
                "solver": solver,
                "flip": flip,
                "audit_class": _majority_audit(st),
                "tokens_in": sum(t.tokens_in for t in st),
                "tokens_out": sum(t.tokens_out for t in st),
                "wall_minutes": round(sum(t.wall_minutes or 0 for t in st), 1),
                "rejected_rule": unit["rejected_rule"] or "",
                "status": unit["status"],
            }
            for lv in range(7):
                scored = [t for t in per_level[lv] if t.audit_class not in {"contaminated", "checksum"}]
                row[f"L{lv}_pass"] = sum(1 for t in scored if t.reward == 1.0)
                row[f"L{lv}_n"] = len(scored)
            rows.append(row)
    return rows


def _majority_audit(trials: list[Any]) -> str:
    if not trials:
        return ""
    counts = Counter(t.audit_class for t in trials if t.audit_class)
    if not counts:
        return ""
    return counts.most_common(1)[0][0]


def write_parquet(rows: list[dict[str, Any]], dest: Path) -> Path:
    dest.parent.mkdir(parents=True, exist_ok=True)
    df = pd.DataFrame(rows, columns=list(RESULT_COLUMNS))
    df.to_parquet(dest, index=False)
    return dest


def render_dashboard(store: PipelineStore, rows: list[dict[str, Any]] | None = None) -> str:
    rows = rows if rows is not None else task_rows(store)
    units = store.list_units()
    n_built = sum(1 for u in units if u["status"] in {"packaged", "solved", "verified", "authored"})
    n_packaged = sum(1 for u in units if u["status"] in {"packaged", "solved"})
    n_rejected = sum(1 for u in units if u["status"] == "rejected")
    reject_rules = Counter(u["rejected_rule"] for u in units if u["rejected_rule"])
    trials = store.list_trials(include_excluded=True)
    n_screened = sum(1 for t in trials if t.excluded or t.audit_class == "contaminated")
    composer = store.composer_tokens()
    flip_hist: dict[str, Counter[str]] = {}
    for row in rows:
        solver = row.get("solver") or "?"
        flip = row.get("flip")
        flip_hist.setdefault(solver, Counter())
        flip_hist[solver][str(flip) if flip is not None else "none"] += 1

    lines = [
        "# Overnight pipeline — live results",
        "",
        f"- tasks seen: **{len(units)}**",
        f"- tasks built (authored+): **{n_built}** (packaged/solved: {n_packaged})",
        f"- rejected by rule: **{n_rejected}**",
        f"- screened (contaminated/excluded): **{n_screened}**",
        f"- Composer tokens used (in+out): **{composer}**",
        "",
        "## Rejected by rule",
        "",
    ]
    if reject_rules:
        lines += ["| rule | n |", "|---|---:|"]
        for rule, n in reject_rules.most_common():
            lines.append(f"| `{rule}` | {n} |")
    else:
        lines.append("_none_")
    lines += ["", "## Flip-point histogram per solver", ""]
    if flip_hist:
        lines += ["| solver | flip | n |", "|---|---|---:|"]
        for solver, ctr in sorted(flip_hist.items()):
            for flip, n in sorted(ctr.items(), key=lambda kv: (kv[0] == "none", kv[0])):
                lines.append(f"| {solver} | {flip} | {n} |")
    else:
        lines.append("_no scored units yet_")
    lines += ["", "## Tasks", "", "| repo | unit | family | files | lines | solver | flip | L2 | status |", "|---|---|---|---:|---:|---|---|---|---|"]
    for row in rows:
        l2 = f"{row.get('L2_pass', 0)}/{row.get('L2_n', 0)}"
        lines.append(
            f"| {row.get('repo')} | {row.get('unit')} | {row.get('family')} | "
            f"{row.get('closure_size')} | {row.get('lines')} | {row.get('solver')} | "
            f"{row.get('flip')} | {l2} | {row.get('status')} |"
        )
    return "\n".join(lines).rstrip() + "\n"


def write_dashboard(store: PipelineStore, dest: Path, rows: list[dict[str, Any]] | None = None) -> Path:
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_text(render_dashboard(store, rows), encoding="utf-8")
    return dest


def aggregate(store: PipelineStore, cfg: PipelineConfig) -> pd.DataFrame:
    rows = task_rows(store)
    write_parquet(rows, cfg.results_parquet)
    write_dashboard(store, cfg.results_md, rows)
    store.mark_step("_", "aggregate", "done", payload={"n": len(rows)})
    return pd.DataFrame(rows, columns=list(RESULT_COLUMNS))


def format_table(df: pd.DataFrame) -> str:
    if df.empty:
        return "(no rows)"
    cols = [c for c in ("repo", "unit", "family", "solver", "flip", "L2_pass", "L2_n", "status") if c in df.columns]
    return df[cols].to_string(index=False)

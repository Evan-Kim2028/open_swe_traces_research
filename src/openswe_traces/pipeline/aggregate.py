"""results.parquet + live results.md dashboard (calibration curve is primary)."""

from __future__ import annotations

from collections import Counter
from pathlib import Path
from typing import Any

import pandas as pd

from openswe_traces.pipeline.config import PipelineConfig
from openswe_traces.pipeline.solve import flip_from_store
from openswe_traces.pipeline.state import PipelineStore
from openswe_traces.pipeline_ext.author_meta import AuthorUnit, author_calibration
from openswe_traces.pipeline_ext.calibration import (
    FLAG_ACTIONS,
    AttemptRecord,
    calibrate,
)
from openswe_traces.pipeline_ext.timeouts import TIMEOUT_CLASS

RESULT_COLUMNS = (
    "repo",
    "unit",
    "family",
    "closure_size",
    "lines",
    "solver",
    "flip",
    "confirmed",
    "predicted_flip",
    "is_control",
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
    "hacked_n",
    "rejected_rule",
    "status",
)


def _unit_get(unit: Any, key: str, default: Any = None) -> Any:
    try:
        val = unit[key]
    except (KeyError, IndexError):
        return default
    return default if val is None else val


def attempt_records(store: PipelineStore) -> list[AttemptRecord]:
    units = {(u["repo"], u["unit"]): u for u in store.list_units()}
    out: list[AttemptRecord] = []
    for t in store.list_trials(include_excluded=True):
        u = units.get((t.repo, t.unit))
        timeout = bool(t.timeout) or t.audit_class in {TIMEOUT_CLASS, "infra"}
        passed = t.reward == 1.0 and not timeout and t.audit_class != "hacked" and not t.excluded
        out.append(
            AttemptRecord(
                repo=t.repo,
                unit=t.unit,
                solver=t.solver,
                level=t.level,
                attempt=t.attempt,
                passed=passed,
                timeout=timeout,
                wall_seconds=None if t.wall_minutes is None else float(t.wall_minutes) * 60.0,
                audit_class=t.audit_class or "clean",
                rejected_rule=_unit_get(u, "rejected_rule") or None if u is not None else None,
                host="",
                author_backend=_unit_get(u, "author_backend") or "" if u is not None else "",
                author_session="",
                is_control=bool(_unit_get(u, "is_control")) if u is not None else False,
                valid_unit=(u is not None and _unit_get(u, "status") != "rejected"),
            )
        )
    return out


def task_rows(store: PipelineStore) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    for unit in store.list_units():
        repo = unit["repo"]
        name = unit["unit"]
        trials = store.list_trials(repo=repo, unit=name, include_excluded=True)
        solvers = sorted({t.solver for t in trials if t.solver})
        predicted = _unit_get(unit, "predicted_flip")
        is_ctrl = bool(_unit_get(unit, "is_control"))
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
                    "confirmed": False,
                    "predicted_flip": predicted,
                    "is_control": is_ctrl,
                    "audit_class": "",
                    "tokens_in": 0,
                    "tokens_out": 0,
                    "wall_minutes": 0.0,
                    "hacked_n": 0,
                    "rejected_rule": unit["rejected_rule"] or "",
                    "status": unit["status"],
                    **{f"L{lv}_pass": 0 for lv in range(7)},
                    **{f"L{lv}_n": 0 for lv in range(7)},
                }
            )
            continue
        for solver in solvers:
            st = [t for t in trials if t.solver == solver]
            scored = [
                t
                for t in st
                if not t.excluded
                and not t.timeout
                and t.audit_class not in {"contaminated", "checksum", "hacked", TIMEOUT_CLASS, "infra"}
            ]
            per_level = {lv: [t for t in scored if t.level == lv] for lv in range(7)}
            flip = flip_from_store(store, repo, name, solver)
            row: dict[str, Any] = {
                "repo": repo,
                "unit": name,
                "family": unit["family"] or "",
                "closure_size": unit["n_files"] or 0,
                "lines": unit["n_lines"] or 0,
                "solver": solver,
                "flip": flip.level,
                "confirmed": flip.confirmed,
                "predicted_flip": predicted,
                "is_control": is_ctrl,
                "audit_class": _majority_audit(st),
                "tokens_in": sum(t.tokens_in for t in st),
                "tokens_out": sum(t.tokens_out for t in st),
                "wall_minutes": round(sum(t.wall_minutes or 0 for t in st), 1),
                "hacked_n": sum(1 for t in st if t.audit_class == "hacked"),
                "rejected_rule": unit["rejected_rule"] or "",
                "status": unit["status"],
            }
            for lv in range(7):
                row[f"L{lv}_pass"] = sum(1 for t in per_level[lv] if t.reward == 1.0)
                row[f"L{lv}_n"] = len(per_level[lv])
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


def _author_units(store: PipelineStore, rows: list[dict[str, Any]]) -> list[AuthorUnit]:
    units = {(u["repo"], u["unit"]): u for u in store.list_units()}
    out: list[AuthorUnit] = []
    seen: set[tuple[str, str, str]] = set()
    for row in rows:
        key = (str(row.get("repo")), str(row.get("unit")), str(row.get("solver") or ""))
        if key in seen:
            continue
        seen.add(key)
        u = units.get((key[0], key[1]))
        backend = ""
        if u is not None:
            backend = str(_unit_get(u, "author_backend") or "")
        out.append(
            AuthorUnit(
                author_backend=backend or "unknown",
                unit=str(row.get("unit")),
                repo=str(row.get("repo")),
                predicted=None if row.get("predicted_flip") is None else int(row["predicted_flip"]),
                measured=None if row.get("flip") is None else int(row["flip"]),
                confirmed=bool(row.get("confirmed")),
            )
        )
    return out


def render_dashboard(store: PipelineStore, rows: list[dict[str, Any]] | None = None) -> str:
    rows = rows if rows is not None else task_rows(store)
    units = store.list_units()
    n_built = sum(1 for u in units if u["status"] in {"packaged", "solved", "verified", "authored"})
    n_packaged = sum(1 for u in units if u["status"] in {"packaged", "solved"})
    n_rejected = sum(1 for u in units if u["status"] == "rejected")
    reject_rules = Counter(u["rejected_rule"] for u in units if u["rejected_rule"])
    trials = store.list_trials(include_excluded=True)
    n_screened = sum(1 for t in trials if t.excluded or t.audit_class == "contaminated")
    n_hacked = sum(1 for t in trials if t.audit_class == "hacked")
    composer = store.composer_tokens()
    flip_hist: dict[str, Counter[str]] = {}
    for row in rows:
        solver = row.get("solver") or "?"
        flip = row.get("flip")
        flip_hist.setdefault(solver, Counter())
        flip_hist[solver][str(flip) if flip is not None else "none"] += 1

    report = calibrate(attempt_records(store))
    author_cal = author_calibration(_author_units(store, rows))

    lines = [
        "# Overnight pipeline — live results",
        "",
        f"- tasks seen: **{len(units)}**",
        f"- tasks built (authored+): **{n_built}** (packaged/solved: {n_packaged})",
        f"- rejected by rule: **{n_rejected}**",
        f"- screened (contaminated/excluded): **{n_screened}**",
        f"- hacked (B9 hard fail, excluded from flip): **{n_hacked}**",
        f"- Composer tokens used (in+out): **{composer}**",
        f"- inter-attempt 2-1 split fraction: **{report.split_fraction:.2f}**",
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

    lines += ["", "## Calibration curve (pass rate per repo × solver × level)", ""]
    if report.curve:
        lines += ["| repo | solver | level | passes | attempts | rate |", "|---|---|---:|---:|---:|---:|"]
        for r in report.curve:
            lines.append(
                f"| {r.repo} | {r.solver} | L{r.level} | {r.passes} | {r.attempts} | {r.rate:.2f} |"
            )
        lines += ["", "Nearest 50% pass (lower level wins ties):", ""]
        for (repo, solver), level in sorted(report.nearest_50.items()):
            lines.append(f"- `{repo}` / `{solver}`: {'none' if level is None else f'L{level}'}")
    else:
        lines.append("_no scored attempts yet_")

    lines += ["", "## Author calibration (predicted_flip vs measured flip)", ""]
    if author_cal:
        lines += [
            "| backend | n | paired | MAE | exact | off-by-one |",
            "|---|---:|---:|---:|---:|---:|",
        ]
        for cal in author_cal:
            mae = "—" if cal.mean_abs_error is None else f"{cal.mean_abs_error:.2f}"
            lines.append(
                f"| {cal.backend} | {cal.n} | {cal.n_with_both} | {mae} | {cal.exact} | {cal.off_by_one} |"
            )
    else:
        lines.append("_no author predictions yet_")

    lines += ["", "## Early-warning flags", ""]
    fired = [f for f in report.flags if f.fired]
    if fired:
        for flag in fired:
            loc = " ".join(x for x in (flag.repo, flag.solver) if x)
            lines.append(f"### `{flag.flag}` {loc}".rstrip())
            lines.append("")
            lines.append(flag.detail)
            lines.append("")
            action = flag.action or FLAG_ACTIONS.get(flag.flag, "")
            if action:
                lines.append(f"**Action:** {action}")
                lines.append("")
    else:
        lines.append("_none fired_")
        if report.flags:
            lines.append("")
            lines.append("Checked (not fired): " + ", ".join(sorted({f.flag for f in report.flags})))

    lines += ["", "## Flip-point histogram per solver", ""]
    if flip_hist:
        lines += ["| solver | flip | n |", "|---|---|---:|"]
        for solver, ctr in sorted(flip_hist.items()):
            for flip, n in sorted(ctr.items(), key=lambda kv: (kv[0] == "none", kv[0])):
                lines.append(f"| {solver} | {flip} | {n} |")
    else:
        lines.append("_no scored units yet_")
    lines += [
        "",
        "## Tasks",
        "",
        "| repo | unit | family | files | lines | solver | flip | L2 | status |",
        "|---|---|---|---:|---:|---|---|---|---|",
    ]
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
    cols = [
        c
        for c in ("repo", "unit", "family", "solver", "flip", "L2_pass", "L2_n", "status")
        if c in df.columns
    ]
    return df[cols].to_string(index=False)

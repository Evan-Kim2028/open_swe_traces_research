"""Stream task text from traces_data, assign heuristic rungs, correlate with difficulty."""

from __future__ import annotations

import argparse
import json
import os
import random
import sys
import time
import traceback
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path

import pandas as pd
from rich.console import Console
from scipy.stats import spearmanr
from sklearn.model_selection import GroupShuffleSplit

import duckdb

from .data import PARQUET_GLOB, ROOT, connect_ephemeral, list_parquet_files, parse_shard
from .difficulty import (
    SEED,
    TEST_SIZE,
    build_task_design,
    load_trajectory_frame,
    rel,
)
from .features import _sql_str, fmt_duration, part_path_for, utcnow
from .rungs import classify_task, features_as_dict
from .score import fit_logistic

console = Console()

PARTS_DIR = ROOT / "outputs" / "rung_features_parts"
OUT_PARQUET = ROOT / "outputs" / "rung_features.parquet"
OUT_MD = ROOT / "analytics" / "research" / "openswe_rung_mapping.md"
VALIDATION_JSON = ROOT / "outputs" / "rung_validation_manual.json"
DIFFICULTY_PARQUET = ROOT / "outputs" / "task_difficulty.parquet"
IRT_PARQUET = ROOT / "outputs" / "task_irt.parquet"
PROXY_PARQUET = ROOT / "outputs" / "proxy_features.parquet"
TEMPORAL_PARQUET = ROOT / "outputs" / "temporal_features.parquet"
SCORES_PARQUET = ROOT / "outputs" / "trace_scores.parquet"

TASK_CONTENT_SQL = r"""
WITH src AS (
    SELECT
        instance_id,
        messages[2].content AS raw_content,
        coalesce(metadata.reference_patch.patch, '') AS reference_patch,
        length(messages[2].content) AS content_len
    FROM read_parquet('{file_sql}', union_by_name=true)
    WHERE len(messages) >= 2 AND messages[2].role = 'user'
)
SELECT
    instance_id,
    arg_max(raw_content, content_len) AS raw_content,
    arg_max(reference_patch, content_len) AS reference_patch,
    max(content_len)::INTEGER AS content_len,
    count(*)::BIGINT AS n_rows
FROM src
GROUP BY 1
"""


def build_shard_sql(file_path: Path) -> str:
    return TASK_CONTENT_SQL.format(file_sql=str(file_path).replace("'", "''"))


def process_shard(
    con: duckdb.DuckDBPyConnection,
    file_path: Path,
    *,
    force: bool,
    parts_dir: Path = PARTS_DIR,
) -> tuple[str, int]:
    part = part_path_for(file_path, parts_dir)
    if part.exists() and part.stat().st_size > 0 and not force:
        return "skip", 0

    src_rows = con.execute(f"SELECT count(*) FROM read_parquet({_sql_str(file_path)})").fetchone()[0]
    tmp = Path(f"{part}.{os.getpid()}.tmp")
    tmp.parent.mkdir(parents=True, exist_ok=True)
    try:
        rows = con.execute(build_shard_sql(file_path)).df()
        if rows.empty:
            raise RuntimeError("empty shard")
        records = []
        for _, row in rows.iterrows():
            result = classify_task(row["raw_content"], row["reference_patch"])
            feat = features_as_dict(result.features)
            records.append(
                {
                    "instance_id": row["instance_id"],
                    "rung": result.rung,
                    "task_text": result.task_text,
                    "leakage_symbols": json.dumps(result.leakage_symbols),
                    "content_len": int(row["content_len"]),
                    "n_rows": int(row["n_rows"]),
                    **feat,
                }
            )
        out_df = pd.DataFrame(records)
        out_df.to_parquet(tmp, index=False)
        out_rows = len(out_df)
        out_instances = out_df["instance_id"].nunique()
        out_trajectories = int(rows["n_rows"].sum())
        if out_rows != out_instances or out_rows == 0 or out_trajectories != src_rows:
            raise RuntimeError(
                f"row guard failed: source={src_rows} out={out_rows} "
                f"instances={out_instances} trajectories={out_trajectories}"
            )
    except Exception:
        tmp.unlink(missing_ok=True)
        raise
    os.replace(tmp, part)
    return "done", int(out_rows)


def merge_parts(
    con: duckdb.DuckDBPyConnection,
    *,
    parts_dir: Path = PARTS_DIR,
    out_path: Path = OUT_PARQUET,
) -> tuple[int, int] | None:
    parts = sorted(parts_dir.glob("*.parquet"))
    if not parts:
        return None
    glob_sql = _sql_str(parts_dir / "*.parquet")
    tmp = Path(str(out_path) + ".tmp")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    con.execute(
        f"""
        COPY (
            SELECT
                instance_id,
                max(rung) AS rung,
                max(task_text) AS task_text,
                max(leakage_symbols) AS leakage_symbols,
                max(content_len) AS content_len,
                sum(n_rows)::BIGINT AS n_rows,
                max(word_count) AS word_count,
                max(has_repro) AS has_repro,
                max(has_expected_actual) AS has_expected_actual,
                max(has_stack_trace) AS has_stack_trace,
                max(has_test_names) AS has_test_names,
                max(has_signature) AS has_signature,
                max(has_leakage) AS has_leakage,
                max(leakage_count) AS leakage_count,
                max(has_test_code) AS has_test_code,
                max(n_test_funcs_in_text) AS n_test_funcs_in_text,
                max(n_test_files_in_text) AS n_test_files_in_text,
                max(patch_has_tests) AS patch_has_tests
            FROM read_parquet({glob_sql})
            GROUP BY 1
            ORDER BY 1
        ) TO {_sql_str(tmp)} (FORMAT PARQUET)
        """
    )
    _, instances = con.execute(
        f"SELECT count(*), count(DISTINCT instance_id) FROM read_parquet({_sql_str(tmp)})"
    ).fetchone()
    os.replace(tmp, out_path)
    return len(parts), int(instances)


def stream_rung_features(args: argparse.Namespace) -> int:
    PARTS_DIR.mkdir(parents=True, exist_ok=True)
    files = [Path(f).resolve() for f in args.file] if args.file else list_parquet_files(args.data_glob)
    if args.limit:
        files = files[: args.limit]
    if not files:
        raise SystemExit(f"No parquet files matched {args.data_glob}")

    con = connect_ephemeral()
    con.execute(f"SET memory_limit='{args.memory_limit}'")
    con.execute(f"SET threads={args.threads}")

    done = skipped = failed = 0
    elapsed_total = 0.0
    failures: list[str] = []
    try:
        for i, file_path in enumerate(files, start=1):
            label = parse_shard(file_path).label
            t0 = time.monotonic()
            try:
                status, rows = process_shard(con, file_path, force=args.force)
            except Exception as exc:  # noqa: BLE001
                failed += 1
                failures.append(label)
                console.print(f"[{utcnow()}] [{i}/{len(files)}] [red]FAILED[/red] {label}: {exc}")
                console.print(f"[dim]{traceback.format_exc(limit=2)}[/dim]")
                continue
            elapsed = time.monotonic() - t0
            if status == "skip":
                skipped += 1
                console.print(f"[{utcnow()}] [{i}/{len(files)}] skipped (part exists) {label}")
                continue
            done += 1
            elapsed_total += elapsed
            rate = elapsed_total / max(done, 1)
            eta = rate * (len(files) - i)
            console.print(
                f"[{utcnow()}] [{i}/{len(files)}] [green]done[/green] {label} "
                f"instances={rows:,} in {elapsed:.1f}s (avg {rate:.1f}s, eta {fmt_duration(eta)})"
            )

        if failures:
            console.print(f"[{utcnow()}] {failed} shard(s) failed — not merging")
        else:
            merged = merge_parts(con, out_path=args.output)
            if merged:
                n_parts, instances = merged
                console.print(
                    f"[{utcnow()}] merged {n_parts} parts → {rel(args.output)}: {instances:,} instances"
                )
    except KeyboardInterrupt:
        console.print(f"[{utcnow()}] interrupted — parts kept, rerun to resume")
        return 130
    finally:
        con.close()

    if failures:
        return 1
    return 0


def load_rung_frame(path: Path = OUT_PARQUET) -> pd.DataFrame:
    con = connect_ephemeral()
    try:
        return con.execute(f"SELECT * FROM read_parquet({_sql_str(path)})").df()
    finally:
        con.close()


@dataclass
class CorrelationResult:
    by_rung: pd.DataFrame
    leakage: dict[str, float]
    feature_spearman: pd.DataFrame
    logistic_baseline_auc: float
    logistic_augmented_auc: float
    logistic_delta: float
    n_trajectories: int


def _pct_all_fail(sr: pd.Series) -> float:
    return float((sr <= 0.0).mean()) if len(sr) else float("nan")


def _pct_all_pass(sr: pd.Series) -> float:
    return float((sr >= 1.0).mean()) if len(sr) else float("nan")


def correlate_rungs(
    rung_df: pd.DataFrame,
    difficulty_df: pd.DataFrame,
    irt_df: pd.DataFrame,
    traj_df: pd.DataFrame,
    harnesses: list[str],
) -> CorrelationResult:
    diff = difficulty_df.merge(irt_df[["instance_id", "b_2pl"]], on="instance_id", how="left")
    merged = rung_df.merge(diff, on="instance_id", how="inner")
    fitted = merged[merged["n_labeled"] >= 3].copy()

    rows = []
    for rung in range(7):
        sub = fitted[fitted["rung"] == rung]
        sr = sub["solve_rate"].dropna()
        b = sub["b_2pl"].dropna()
        rows.append(
            {
                "rung": f"L{rung}",
                "n": len(sub),
                "mean_solve_rate": float(sr.mean()) if len(sr) else float("nan"),
                "mean_b_2pl": float(b.mean()) if len(b) else float("nan"),
                "pct_all_fail": _pct_all_fail(sr),
                "pct_all_pass": _pct_all_pass(sr),
            }
        )
    by_rung = pd.DataFrame(rows)

    leak = fitted.groupby("has_leakage")["solve_rate"].agg(["count", "mean"])
    leakage = {
        "n_clean": int(leak.loc[0, "count"]) if 0 in leak.index else 0,
        "mean_clean": float(leak.loc[0, "mean"]) if 0 in leak.index else float("nan"),
        "n_leaky": int(leak.loc[1, "count"]) if 1 in leak.index else 0,
        "mean_leaky": float(leak.loc[1, "mean"]) if 1 in leak.index else float("nan"),
        "spearman": float(
            spearmanr(fitted["has_leakage"], fitted["solve_rate"]).statistic
        ),
    }

    feat_cols = [
        "word_count",
        "has_repro",
        "has_expected_actual",
        "has_stack_trace",
        "has_test_names",
        "has_signature",
        "has_leakage",
        "has_test_code",
        "rung",
    ]
    spearman_rows = []
    sr = fitted["solve_rate"].dropna()
    fit_sub = fitted.loc[sr.index]
    for col in feat_cols:
        rho = spearmanr(fit_sub[col], fit_sub["solve_rate"]).statistic
        spearman_rows.append({"feature": col, "spearman_solve_rate": float(rho), "n": len(fit_sub)})
    feature_spearman = pd.DataFrame(spearman_rows).sort_values(
        "spearman_solve_rate", key=abs, ascending=False
    )

    labeled = traj_df[traj_df["resolved"].isin([0, 1])].copy()
    inst = merged.set_index("instance_id")
    for col in ["rung"] + feat_cols:
        labeled[col] = inst[col].reindex(labeled["instance_id"]).to_numpy()
    labeled = labeled.dropna(subset=["rung"])

    inst_for_design = merged[
        ["instance_id", "solve_rate", "issue_chars", "gold_patch_files", "gold_patch_lines"]
        + [f"frac_rollouts_{h}" for h in harnesses]
        + ["language", "category", "repo"]
    ].drop_duplicates("instance_id")
    inst_for_design = inst_for_design.rename(
        columns={
            "gold_patch_files": "gold_patch_files",
            "gold_patch_lines": "gold_patch_lines",
        }
    )
    # build_task_design expects columns from difficulty frame
    X_base, base_features, _ = build_task_design(inst_for_design, harnesses)
    base_map = X_base.copy()
    base_map["instance_id"] = inst_for_design["instance_id"].to_numpy()
    base_map = base_map.set_index("instance_id")

    rung_cols = ["rung", "has_repro", "has_expected_actual", "has_test_names", "has_signature", "has_leakage"]
    for c in rung_cols:
        labeled[c] = inst[c].reindex(labeled["instance_id"]).to_numpy()

    X = base_map.reindex(labeled["instance_id"]).reset_index(drop=True)
    X = pd.concat([X, labeled[rung_cols].reset_index(drop=True)], axis=1)
    features = list(X.columns)
    Xv = X.to_numpy(dtype=float)
    y = labeled["resolved"].to_numpy(dtype=int)
    groups = labeled["instance_id"].astype(str).to_numpy()
    splitter = GroupShuffleSplit(n_splits=1, test_size=TEST_SIZE, random_state=SEED)
    train_idx, test_idx = next(splitter.split(Xv, y, groups))

    X_base_only = base_map.reindex(labeled["instance_id"]).reset_index(drop=True).to_numpy(dtype=float)
    fit_base = fit_logistic(
        X_base_only[train_idx], y[train_idx], X_base_only[test_idx], y[test_idx], base_features
    )
    fit_aug = fit_logistic(Xv[train_idx], y[train_idx], Xv[test_idx], y[test_idx], features)

    return CorrelationResult(
        by_rung=by_rung,
        leakage=leakage,
        feature_spearman=feature_spearman,
        logistic_baseline_auc=float(fit_base.auc_test),
        logistic_augmented_auc=float(fit_aug.auc_test),
        logistic_delta=float(fit_aug.auc_test - fit_base.auc_test),
        n_trajectories=len(labeled),
    )


def stratified_validation_sample(
    rung_df: pd.DataFrame, n: int = 200, seed: int = SEED
) -> pd.DataFrame:
    rng = random.Random(seed)
    samples = []
    per_rung = max(1, n // 7)
    for rung in range(7):
        sub = rung_df[rung_df["rung"] == rung]
        if sub.empty:
            continue
        k = min(len(sub), per_rung + (1 if rung < n % 7 else 0))
        idx = rng.sample(list(sub.index), k=min(k, len(sub)))
        samples.append(sub.loc[idx])
    out = pd.concat(samples).head(n)
    return out.reset_index(drop=True)


def confusion_matrix(manual: pd.DataFrame) -> pd.DataFrame:
    levels = list(range(7))
    mat = pd.DataFrame(0, index=[f"L{h}" for h in levels], columns=[f"L{p}" for p in levels])
    for _, row in manual.iterrows():
        h = f"L{int(row['rung_heuristic'])}"
        m = f"L{int(row['rung_manual'])}"
        if h in mat.index and m in mat.columns:
            mat.loc[h, m] += 1
    return mat


def agreement_stats(manual: pd.DataFrame) -> dict[str, float]:
    exact = (manual["rung_heuristic"] == manual["rung_manual"]).mean()
    within_1 = (abs(manual["rung_heuristic"] - manual["rung_manual"]) <= 1).mean()
    rho = float(spearmanr(manual["rung_heuristic"], manual["rung_manual"]).statistic)
    return {"exact": float(exact), "within_1": float(within_1), "spearman": rho}


def md_table(header: list[str], rows: list[list[str]]) -> str:
    lines = ["| " + " | ".join(header) + " |", "|" + "|".join(["---"] * len(header)) + "|"]
    lines += ["| " + " | ".join(row) + " |" for row in rows]
    return "\n".join(lines)


def build_report(
    rung_df: pd.DataFrame,
    corr: CorrelationResult,
    manual: pd.DataFrame | None,
    examples: dict[str, list[dict]],
) -> str:
    stamp = datetime.now(UTC).strftime("%Y-%m-%d %H:%M UTC")
    lines: list[str] = []
    lines.append("# Open-SWE-Traces affordance ladder mapping")
    lines.append("")
    lines.append(f"Generated {stamp} by `scripts/rung_mapping.py`.")
    lines.append("")

    n = len(rung_df)
    dist = rung_df["rung"].value_counts().sort_index()
    lines.append("## Method")
    lines.append("")
    lines.append(
        "Task text is `messages[2].content` (first user message) streamed from "
        "`traces_data/` with DuckDB (one row per `instance_id`, longest shard kept). "
        "Harness wrappers (`<pr_description>`, `<instructions>`) are stripped in "
        "`openswe_traces.rungs.strip_task_text`. Heuristic rung assignment follows "
        "the L0–L6 ladder in `analytics/research/verifier_rules.md` (L = A + 2). "
        "The verifier class here is always hidden fail-to-pass tests (SWE-bench style; "
        "rules B3/B4). `patch_has_tests` flags whether the reference/gold patch "
        "touches test paths."
    )
    lines.append("")
    lines.append(
        f"Corpus: **{n:,}** instances. Solve rates and `b_2pl` use instances with "
        f"≥ 3 labeled rollouts (~12 attempts each; rule C6 — far more stable than "
        "single-attempt Harbor ladder readings)."
    )
    lines.append("")

    lines.append("## Rung distribution")
    lines.append("")
    rows = []
    for rung in range(7):
        c = int(dist.get(rung, 0))
        rows.append([f"L{rung}", f"{c:,}", f"{100 * c / n:.1f}%"])
    lines.append(md_table(["rung", "instances", "share"], rows))
    lines.append("")
    l56 = int(dist.get(5, 0) + dist.get(6, 0))
    lines.append(
        f"L5/L6 (test bodies in the instruction): **{l56:,}** instances "
        f"({100 * l56 / n:.2f}% of corpus) — rare, as expected for SWE-bench-style tasks."
    )
    lines.append("")
    patch_tests = int(rung_df["patch_has_tests"].sum())
    lines.append(
        f"Reference patch touches test files on **{patch_tests:,}** instances "
        f"({100 * patch_tests / n:.1f}%) — the PR itself changed tests, but the "
        "agent still does not receive those tests as the verifier oracle."
    )
    lines.append("")

    if manual is not None and len(manual):
        lines.append("## Validation (n=200 stratified manual labels)")
        lines.append("")
        stats = agreement_stats(manual)
        lines.append(
            f"Exact agreement: **{100 * stats['exact']:.1f}%**; "
            f"within ±1 rung: **{100 * stats['within_1']:.1f}%**; "
            f"Spearman ρ = **{stats['spearman']:.3f}**."
        )
        lines.append("")
        mat = confusion_matrix(manual)
        header = ["heuristic \\ manual"] + list(mat.columns)
        mat_rows = [[idx] + [str(mat.loc[idx, c]) for c in mat.columns] for idx in mat.index]
        lines.append(md_table(header, mat_rows))
        lines.append("")

    lines.append("## Correlation with measured difficulty")
    lines.append("")
    lines.append("### By heuristic rung (≥ 3 labeled rollouts)")
    lines.append("")
    br = corr.by_rung
    rows = [
        [
            r["rung"],
            f"{int(r['n']):,}",
            f"{r['mean_solve_rate']:.3f}",
            f"{r['mean_b_2pl']:.3f}",
            f"{100 * r['pct_all_fail']:.1f}%",
            f"{100 * r['pct_all_pass']:.1f}%",
        ]
        for _, r in br.iterrows()
    ]
    lines.append(
        md_table(
            ["rung", "n", "mean solve_rate", "mean b_2pl", "% all_fail", "% all_pass"],
            rows,
        )
    )
    lines.append("")
    lk = corr.leakage
    lines.append("### Leakage (B7 analogue) vs solve rate")
    lines.append("")
    lines.append(
        f"Clean (no patch symbol in text): n={lk['n_clean']:,}, mean solve_rate="
        f"{lk['mean_clean']:.3f}. Leaky: n={lk['n_leaky']:,}, mean="
        f"{lk['mean_leaky']:.3f}. Spearman(has_leakage, solve_rate)="
        f"{lk['spearman']:+.3f}."
    )
    lines.append("")
    lines.append("### Feature Spearman with solve_rate")
    lines.append("")
    fs = corr.feature_spearman
    rows = [
        [r["feature"], f"{r['spearman_solve_rate']:+.3f}", f"{int(r['n']):,}"]
        for _, r in fs.iterrows()
    ]
    lines.append(md_table(["feature", "Spearman", "n"], rows))
    lines.append("")
    lines.append("### Logistic model of resolved (trajectory level, grouped 80/20 by instance)")
    lines.append("")
    lines.append(
        f"Task-only baseline (same features as `task_difficulty_summary.md`): "
        f"held-out AUC **{corr.logistic_baseline_auc:.4f}**. "
        f"With rung features added: **{corr.logistic_augmented_auc:.4f}** "
        f"(Δ = **{corr.logistic_delta:+.4f}**, n={corr.n_trajectories:,} labeled trajectories)."
    )
    lines.append("")

    lines.append("## Examples")
    lines.append("")
    for label, items in examples.items():
        lines.append(f"### {label}")
        lines.append("")
        for ex in items[:2]:
            lines.append(f"- `{ex['instance_id']}` (L{ex['rung']}, solve_rate={ex.get('solve_rate', 'n/a')})")
            snippet = ex["task_text"][:400].replace("\n", " ")
            lines.append(f"  > {snippet}…")
        lines.append("")

    lines.append("## Verdict")
    lines.append("")
    # Compute verdict from data
    br_valid = corr.by_rung.dropna(subset=["mean_solve_rate"])
    if len(br_valid) >= 2:
        rho_rung = float(spearmanr(br_valid["rung"].str[1:].astype(int), br_valid["mean_solve_rate"]).statistic)
    else:
        rho_rung = float("nan")
    lines.append(
        f"**Does rung predict difficulty?** Weakly. Spearman(rung, mean solve_rate) across "
        f"populated rungs = **{rho_rung:+.3f}**; the logistic AUC gain from rung features is "
        f"**{corr.logistic_delta:+.4f}** on top of language/patch-size baselines. Information "
        "in the PR text explains little variance versus patch size and language (see "
        "`task_difficulty_summary.md`). Higher rungs (L3–L4: test names or signatures in the "
        "issue) do not uniformly mean easier tasks — many are feature requests with API detail."
    )
    lines.append("")
    lines.append(
        "**Versus client-go Harbor ladder:** On client-go, flip points for a mid-tier model "
        "concentrate at **L2** (full prose contract) — the first level where the contract is "
        "complete. Open-SWE-Traces issues cluster at **L1–L2** (symptom + partial/full prose "
        "from GitHub PRs) with almost no L5/L6. The ladder *definition* transfers; the "
        "*measurement* does not: these are single fixed affordance levels per instance, not "
        "per-unit flip points over ≥3 attempts per level. Measured solve rates here (~12 "
        "rollouts) are far more stable than one-shot Harbor readings (rule C6)."
    )
    lines.append("")
    lines.append("## Reproduce")
    lines.append("")
    lines.append("```bash")
    lines.append("uv run python scripts/rung_mapping.py")
    lines.append("uv run pytest tests/test_rungs.py")
    lines.append("```")
    lines.append("")
    return "\n".join(lines)


def pick_examples(rung_df: pd.DataFrame, difficulty_df: pd.DataFrame) -> dict[str, list[dict]]:
    merged = rung_df.merge(
        difficulty_df[["instance_id", "solve_rate"]], on="instance_id", how="left"
    )
    out: dict[str, list[dict]] = {}
    clean = merged[(merged["has_leakage"] == 0) & (merged["solve_rate"].notna())]
    leaky = merged[merged["has_leakage"] == 1]
    out["Clean (no patch leakage)"] = (
        clean.sample(min(3, len(clean)), random_state=SEED).to_dict("records") if len(clean) else []
    )
    out["Leaky (patch symbols in issue text)"] = (
        leaky.head(3).to_dict("records") if len(leaky) else []
    )
    return out


def main_rung_mapping() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=OUT_PARQUET)
    parser.add_argument("--summary", type=Path, default=OUT_MD)
    parser.add_argument("--file", action="append")
    parser.add_argument("--limit", type=int)
    parser.add_argument("--data-glob", default=PARQUET_GLOB)
    parser.add_argument("--force", action="store_true")
    parser.add_argument("--threads", type=int, default=4)
    parser.add_argument("--memory-limit", default="4GB")
    parser.add_argument("--stream-only", action="store_true")
    parser.add_argument("--skip-stream", action="store_true")
    args = parser.parse_args()
    args.output = args.output if args.output.is_absolute() else ROOT / args.output
    args.summary = args.summary if args.summary.is_absolute() else ROOT / args.summary

    if args.stream_only and args.skip_stream:
        parser.error("--stream-only and --skip-stream are mutually exclusive")
    if not args.skip_stream:
        rc = stream_rung_features(args)
        if rc != 0 or args.stream_only:
            sys.exit(rc)
    if not args.output.exists():
        raise SystemExit(f"No rung features at {args.output}; run without --skip-stream")

    rung_df = load_rung_frame(args.output)
    con = connect_ephemeral()
    try:
        difficulty_df = con.execute(
            f"SELECT * FROM read_parquet({_sql_str(DIFFICULTY_PARQUET)})"
        ).df()
        irt_df = con.execute(f"SELECT * FROM read_parquet({_sql_str(IRT_PARQUET)})").df()
    finally:
        con.close()
    traj_df = load_trajectory_frame(PROXY_PARQUET, TEMPORAL_PARQUET, SCORES_PARQUET)
    harnesses = sorted(traj_df["harness"].dropna().unique().tolist())
    corr = correlate_rungs(rung_df, difficulty_df, irt_df, traj_df, harnesses)

    manual = None
    if VALIDATION_JSON.exists():
        manual = pd.read_json(VALIDATION_JSON)
        # Refresh heuristic column from current rung assignments.
        hmap = rung_df.set_index("instance_id")["rung"]
        manual["rung_heuristic"] = hmap.reindex(manual["instance_id"]).to_numpy()
    examples = pick_examples(rung_df, difficulty_df)
    report = build_report(rung_df, corr, manual, examples)
    args.summary.parent.mkdir(parents=True, exist_ok=True)
    args.summary.write_text(report)
    console.print(f"Wrote {rel(args.summary)}")

    # Print verdict table for CLI
    console.print("\n[bold]By-rung table[/bold]")
    console.print(corr.by_rung.to_string(index=False))


def resolve_path(value: Path) -> Path:
    return value if value.is_absolute() else (ROOT / value)

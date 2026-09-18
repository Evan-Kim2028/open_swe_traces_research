"""Task difficulty per instance: solve rates, buckets, and how much of the trace
resolving signal is task identity rather than agent behavior.

Stage 1 (one streaming DuckDB pass, resume-safe): every corpus shard is read once for
``instance_id``, ``repo`` and ``len(messages[2].content)`` — the first user message, i.e.
the PR description — aggregated per instance into ``outputs/task_text.parquet`` (shard
parts under ``outputs/task_text_parts/``, atomic + skipped on rerun).

Stage 2: ``outputs/proxy_features.parquet`` (+ ``temporal_features.parquet``, +
``trace_scores.parquet``) is aggregated to one row per ``instance_id`` into
``outputs/task_difficulty.parquet``:

  n_rollouts, n_labeled, n_resolved   trajectory counts (labeled = resolved in (0, 1))
  solve_rate                          n_resolved / n_labeled; null when no labeled rollout
  solve_rate_<harness>                per-harness solve rate (null when unlabeled there)
  frac_rollouts_<harness>             harness mix share of rollouts
  language, category, repo            task identity (language normalized: ts/js aliases)
  gold_patch_files, gold_patch_lines  reference patch size
  issue_chars                         first user message length from stage 1
  difficulty_bucket                   all_fail (0), hard (<0.34), mid (0.34-0.66),
                                      easy (>0.66), all_pass (1); needs >= 3 labeled
                                      rollouts, else unknown

Analyses (numbers in tables) are written to
``analytics/research/task_difficulty_summary.md``:

  1. n_labeled / solve_rate distributions, difficulty bucket counts and trajectory shares
  2. WLS predicting solve_rate from task features only (weighted by n_labeled, grouped
     80/20 by repo): R², Spearman, top standardized coefficients
  3. trace classifier decomposition on the ``score.py`` split: held-out AUC of task
     solve_rate alone (leave-one-out), the 27 trace features alone, and both; plus the
     within-task residual model (resolved - leave-one-out rate) and top feature Spearman
  4. learnability strip: trajectories and assistant chars removed by dropping all_pass +
     all_fail tasks, per harness/teacher

Examples:
  uv run python scripts/task_difficulty.py --stream-only
  uv run python scripts/task_difficulty.py --skip-stream
"""

from __future__ import annotations

import argparse
import os
import sys
import time
import traceback
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path

import numpy as np
import pandas as pd
from rich.console import Console
from scipy.stats import spearmanr
from sklearn.linear_model import LinearRegression
from sklearn.metrics import roc_auc_score
from sklearn.model_selection import GroupShuffleSplit
from sklearn.preprocessing import StandardScaler

import duckdb

from .data import PARQUET_GLOB, ROOT, connect_ephemeral, list_parquet_files, parse_shard
from .features import _sql_str, fmt_duration, part_path_for, utcnow
from .score import MODEL_FEATURES, WINSOR_PCT, fit_logistic, load_joined, md_table

console = Console()

PROXY_PARQUET = ROOT / "outputs" / "proxy_features.parquet"
TEMPORAL_PARQUET = ROOT / "outputs" / "temporal_features.parquet"
SCORES_PARQUET = ROOT / "outputs" / "trace_scores.parquet"
TEXT_PARTS_DIR = ROOT / "outputs" / "task_text_parts"
TEXT_PARQUET = ROOT / "outputs" / "task_text.parquet"
OUT_PARQUET = ROOT / "outputs" / "task_difficulty.parquet"
OUT_MD = ROOT / "analytics" / "research" / "task_difficulty_summary.md"

MIN_LABELED = 3
HARD_MAX = 0.34
EASY_MIN = 0.66
TEST_SIZE = 0.2
SEED = 42
LANGUAGE_ALIASES = {"ts": "typescript", "js": "javascript"}
BUCKET_ORDER = ["all_fail", "hard", "mid", "easy", "all_pass", "unknown"]
DROPPED_BUCKETS = ("all_pass", "all_fail")

TASK_TEXT_SQL = r"""
WITH src AS (
    SELECT
        instance_id,
        repo,
        CASE WHEN len(messages) >= 2 THEN len(messages[2].content) END AS issue_chars
    FROM read_parquet('{file_sql}', union_by_name=true)
)
SELECT
    instance_id,
    max(repo) AS repo,
    max(issue_chars)::INTEGER AS issue_chars,
    count(DISTINCT issue_chars)::INTEGER AS n_issue_variants,
    count(DISTINCT repo)::INTEGER AS n_repo_variants,
    count(*)::BIGINT AS n_rows
FROM src
GROUP BY 1
"""


def build_task_text_sql(file_path: Path) -> str:
    return TASK_TEXT_SQL.format(file_sql=str(file_path).replace("'", "''"))


def process_shard(
    con: duckdb.DuckDBPyConnection,
    file_path: Path,
    *,
    force: bool,
    parts_dir: Path = TEXT_PARTS_DIR,
) -> tuple[str, int]:
    """Write one per-instance part for a shard; 'skip' when the part already exists."""
    part = part_path_for(file_path, parts_dir)
    if part.exists() and part.stat().st_size > 0 and not force:
        return "skip", 0

    src_rows = con.execute(f"SELECT count(*) FROM read_parquet({_sql_str(file_path)})").fetchone()[0]
    tmp = Path(f"{part}.{os.getpid()}.tmp")
    tmp.parent.mkdir(parents=True, exist_ok=True)
    try:
        con.execute(
            f"COPY ({build_task_text_sql(file_path)}) TO {_sql_str(tmp)} (FORMAT PARQUET)"
        )
        out_rows, out_instances, out_trajectories = con.execute(
            f"SELECT count(*), count(DISTINCT instance_id), sum(n_rows) FROM read_parquet({_sql_str(tmp)})"
        ).fetchone()
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


def merge_task_text(
    con: duckdb.DuckDBPyConnection,
    *,
    parts_dir: Path = TEXT_PARTS_DIR,
    out_path: Path = TEXT_PARQUET,
) -> tuple[int, int, int] | None:
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
                max(repo) AS repo,
                max(issue_chars) AS issue_chars,
                max(n_issue_variants)::INTEGER AS n_issue_variants,
                max(n_repo_variants)::INTEGER AS n_repo_variants,
                sum(n_rows)::BIGINT AS n_rows
            FROM read_parquet({glob_sql})
            GROUP BY 1
            ORDER BY 1
        ) TO {_sql_str(tmp)} (FORMAT PARQUET)
        """
    )
    rows, instances = con.execute(
        f"SELECT count(*), count(DISTINCT instance_id) FROM read_parquet({_sql_str(tmp)})"
    ).fetchone()
    os.replace(tmp, out_path)
    return len(parts), int(rows), int(instances)


def stream_task_text(args: argparse.Namespace) -> int:
    """One streaming pass over the corpus shards; returns a process exit code."""
    TEXT_PARTS_DIR.mkdir(parents=True, exist_ok=True)
    args.text.parent.mkdir(parents=True, exist_ok=True)

    if args.file:
        files = [Path(f).resolve() for f in args.file]
    else:
        files = list_parquet_files(args.data_glob)
    if args.limit:
        files = files[: args.limit]
    if not files:
        raise SystemExit(f"No parquet files matched {args.data_glob}")

    con = connect_ephemeral()
    con.execute(f"SET memory_limit='{args.memory_limit}'")
    con.execute(f"SET threads={args.threads}")

    done = skipped = failed = 0
    elapsed_total = 0.0
    started = time.monotonic()
    failures: list[str] = []
    try:
        for i, file_path in enumerate(files, start=1):
            label = parse_shard(file_path).label
            t0 = time.monotonic()
            try:
                status, rows = process_shard(con, file_path, force=args.force)
            except Exception as exc:  # noqa: BLE001 (keep going; the shard stays unprocessed)
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
            console.print(
                f"[{utcnow()}] {failed} shard(s) failed — not merging partial output; "
                "rerun to resume (finished parts are kept)"
            )
        else:
            merged = merge_task_text(con, out_path=args.text)
            if merged is None:
                console.print(f"[{utcnow()}] no parts to merge")
            else:
                n_parts, rows, _ = merged
                partial = bool(args.file or args.limit)
                suffix = (
                    " (partial run: rerun without --file/--limit to cover the corpus)"
                    if partial
                    else ""
                )
                console.print(
                    f"[{utcnow()}] merged {n_parts} parts → {rel(args.text)}: "
                    f"{rows:,} instances{suffix}"
                )
    except KeyboardInterrupt:
        console.print(f"[{utcnow()}] interrupted — parts written so far are kept, rerun to resume")
        return 130
    finally:
        con.close()

    console.print(
        f"[{utcnow()}] task-text pass finished: processed={done} skipped={skipped} failed={failed} "
        f"in {fmt_duration(time.monotonic() - started)}"
    )
    if failures:
        console.print(f"[red]failed shards[/red]: {', '.join(failures)}")
        return 1
    return 0


def load_task_text(path: Path = TEXT_PARQUET) -> pd.DataFrame:
    con = connect_ephemeral()
    try:
        return con.execute(f"SELECT * FROM read_parquet({_sql_str(path)})").df()
    finally:
        con.close()


def load_trajectory_frame(
    proxy_path: Path = PROXY_PARQUET,
    temporal_path: Path = TEMPORAL_PARQUET,
    scores_path: Path = SCORES_PARQUET,
) -> pd.DataFrame:
    df = load_joined(proxy_path, temporal_path)
    con = connect_ephemeral()
    try:
        scores = con.execute(
            f"SELECT trajectory_id, p_resolved FROM read_parquet({_sql_str(scores_path)})"
        ).df()
    finally:
        con.close()
    if scores["trajectory_id"].duplicated().any():
        raise SystemExit(f"duplicate trajectory_id in {scores_path}")
    df = df.merge(scores, on="trajectory_id", how="left")
    missing = int(df["p_resolved"].isna().sum())
    if missing:
        raise SystemExit(f"{missing:,} trajectories missing from {scores_path}")
    return df


def normalize_language(values: pd.Series) -> pd.Series:
    return values.astype("string").str.lower().replace(LANGUAGE_ALIASES)


def _mode(values: pd.Series) -> str | None:
    counts = values.value_counts()
    if counts.empty:
        return None
    return min(counts[counts == counts.max()].index)


def difficulty_buckets(solve_rate: pd.Series, n_labeled: pd.Series) -> pd.Series:
    rate = solve_rate.to_numpy(dtype=float)
    labeled = n_labeled.to_numpy(dtype=float)
    ok = ~np.isnan(rate) & (labeled >= MIN_LABELED)
    bucket = np.select(
        [
            ok & (rate <= 0.0),
            ok & (rate >= 1.0),
            ok & (rate < HARD_MAX),
            ok & (rate <= EASY_MIN),
            ok,
        ],
        ["all_fail", "all_pass", "hard", "mid", "easy"],
        default="unknown",
    )
    return pd.Series(bucket, index=solve_rate.index, dtype=object)


def compute_instance_frame(
    df: pd.DataFrame, task_text: pd.DataFrame | None = None
) -> pd.DataFrame:
    d = df.copy()
    d["language"] = normalize_language(d["language"])
    d["_labeled"] = d["resolved"].isin([0, 1]).astype("int8")
    d["_resolved"] = (d["resolved"] == 1).astype("int8")
    g = d.groupby("instance_id", sort=True)
    inst = g.agg(
        n_rollouts=("trajectory_id", "size"),
        n_labeled=("_labeled", "sum"),
        n_resolved=("_resolved", "sum"),
        language=("language", _mode),
        category=("category", _mode),
        repo=("repo", _mode),
        gold_patch_files=("gold_patch_files", "max"),
        gold_patch_lines=("gold_patch_lines", "max"),
    )
    inst["n_rollouts"] = inst["n_rollouts"].astype("int32")
    inst["n_labeled"] = inst["n_labeled"].astype("int32")
    inst["n_resolved"] = inst["n_resolved"].astype("int32")
    inst["solve_rate"] = (
        inst["n_resolved"] / inst["n_labeled"].where(inst["n_labeled"] > 0)
    ).astype("float64")

    harnesses = sorted(d["harness"].dropna().unique().tolist())
    per = d.groupby(["instance_id", "harness"], sort=True)
    counts = per.size().unstack(fill_value=0).reindex(index=inst.index, columns=harnesses, fill_value=0)
    labeled_h = (
        per["_labeled"].sum().unstack(fill_value=0).reindex(index=inst.index, columns=harnesses, fill_value=0)
    )
    resolved_h = (
        per["_resolved"].sum().unstack(fill_value=0).reindex(index=inst.index, columns=harnesses, fill_value=0)
    )
    for h in harnesses:
        inst[f"frac_rollouts_{h}"] = (counts[h] / inst["n_rollouts"]).astype("float64")
        inst[f"solve_rate_{h}"] = (
            resolved_h[h] / labeled_h[h].where(labeled_h[h] > 0)
        ).astype("float64")

    inst.index.name = "instance_id"
    inst = inst.reset_index()

    if task_text is not None:
        tt = task_text.set_index("instance_id")
        stream_repo = tt["repo"].reindex(inst["instance_id"]).to_numpy()
        proxy_repo = inst["repo"].to_numpy()
        inst["repo"] = np.where(pd.isna(stream_repo), proxy_repo, stream_repo)
        inst["issue_chars"] = tt["issue_chars"].reindex(inst["instance_id"]).to_numpy()

    inst["difficulty_bucket"] = difficulty_buckets(inst["solve_rate"], inst["n_labeled"])
    columns = [
        "instance_id",
        "n_rollouts",
        "n_labeled",
        "n_resolved",
        "solve_rate",
        *[f"solve_rate_{h}" for h in harnesses],
        *[f"frac_rollouts_{h}" for h in harnesses],
        "language",
        "category",
        "repo",
        "gold_patch_files",
        "gold_patch_lines",
        "issue_chars",
        "difficulty_bucket",
    ]
    return inst.loc[:, columns].sort_values("instance_id").reset_index(drop=True)


def leave_one_out_solve_rate(df: pd.DataFrame) -> pd.Series:
    """Per-trajectory solve rate of its instance excluding the trajectory itself.

    Defined only for labeled trajectories in instances with at least two labeled
    trajectories; NaN otherwise (including every trajectory with `resolved == -1`).
    """
    labeled = df["resolved"].isin([0, 1])
    own_resolved = (df["resolved"] == 1).astype("float64")
    n_labeled = labeled.astype("float64").groupby(df["instance_id"]).transform("sum")
    n_resolved = own_resolved.groupby(df["instance_id"]).transform("sum")
    loo = (n_resolved - own_resolved) / (n_labeled - 1.0)
    return loo.where(labeled & (n_labeled >= 2))


def within_task_residual(df: pd.DataFrame) -> pd.Series:
    loo = leave_one_out_solve_rate(df)
    return (df["resolved"].astype("float64") - loo).where(loo.notna())


@dataclass
class LinearFit:
    features: list[str]
    lo: np.ndarray
    hi: np.ndarray
    scaler: StandardScaler
    model: LinearRegression
    r2_train: float
    r2_test: float

    def predict(self, X: np.ndarray) -> np.ndarray:
        X = np.clip(np.asarray(X, dtype=float), self.lo, self.hi)
        return self.model.predict(self.scaler.transform(X))

    def coefs(self) -> list[dict]:
        return sorted(
            ({"feature": f, "coef": float(c)} for f, c in zip(self.features, self.model.coef_)),
            key=lambda d: abs(d["coef"]),
            reverse=True,
        )


def fit_linear(
    X_train: np.ndarray,
    y_train: np.ndarray,
    X_test: np.ndarray,
    y_test: np.ndarray,
    features: list[str],
    sample_weight: np.ndarray | None = None,
) -> LinearFit:
    lo = np.quantile(X_train, WINSOR_PCT[0] / 100, axis=0)
    hi = np.quantile(X_train, WINSOR_PCT[1] / 100, axis=0)
    X_train_w = np.clip(X_train, lo, hi)
    X_test_w = np.clip(X_test, lo, hi)
    scaler = StandardScaler().fit(X_train_w)
    model = LinearRegression().fit(
        scaler.transform(X_train_w), y_train, sample_weight=sample_weight
    )
    return LinearFit(
        features=features,
        lo=lo,
        hi=hi,
        scaler=scaler,
        model=model,
        r2_train=float(model.score(scaler.transform(X_train_w), y_train)),
        r2_test=float(model.score(scaler.transform(X_test_w), y_test)),
    )


@dataclass
class TaskModelResult:
    n_instances: int
    n_train: int
    n_test: int
    n_repos_train: int
    n_repos_test: int
    baseline_harness: str
    weight_total: float
    fit: LinearFit
    rho_train: float
    rho_test: float


def _baseline_level(values: pd.Series) -> str:
    counts = values.value_counts().sort_index()
    return str(counts.idxmax())


def build_task_design(
    frame: pd.DataFrame, harnesses: list[str]
) -> tuple[pd.DataFrame, list[str], str]:
    X = frame[["gold_patch_files", "gold_patch_lines", "issue_chars"]].astype("float64")
    X = X.reset_index(drop=True)
    totals = {h: float(frame[f"frac_rollouts_{h}"].sum()) for h in harnesses}
    baseline = max(sorted(totals), key=lambda h: totals[h])
    for h in harnesses:
        if h != baseline:
            X[f"frac_rollouts_{h}"] = frame[f"frac_rollouts_{h}"].to_numpy(dtype=float)
    for col in ("language", "category"):
        values = frame[col].astype("string")
        dummies = pd.get_dummies(values, prefix=col, dtype=float)
        drop = f"{col}_{_baseline_level(values)}"
        if drop in dummies.columns:
            dummies = dummies.drop(columns=drop)
        X = pd.concat([X, dummies.reset_index(drop=True)], axis=1)
    return X, list(X.columns), baseline


def fit_task_model(inst: pd.DataFrame, harnesses: list[str]) -> TaskModelResult:
    frame = inst[inst["solve_rate"].notna() & inst["issue_chars"].notna()].reset_index(drop=True)
    X, features, baseline = build_task_design(frame, harnesses)
    Xv = X.to_numpy(dtype=float)
    y = frame["solve_rate"].to_numpy(dtype=float)
    weights = frame["n_labeled"].to_numpy(dtype=float)
    groups = frame["repo"].astype(str).to_numpy()
    splitter = GroupShuffleSplit(n_splits=1, test_size=TEST_SIZE, random_state=SEED)
    train_idx, test_idx = next(splitter.split(Xv, y, groups))
    fit = fit_linear(
        Xv[train_idx],
        y[train_idx],
        Xv[test_idx],
        y[test_idx],
        features,
        sample_weight=weights[train_idx],
    )
    group_series = pd.Series(groups)
    return TaskModelResult(
        n_instances=len(frame),
        n_train=len(train_idx),
        n_test=len(test_idx),
        n_repos_train=int(group_series.iloc[train_idx].nunique()),
        n_repos_test=int(group_series.iloc[test_idx].nunique()),
        baseline_harness=baseline,
        weight_total=float(weights.sum()),
        fit=fit,
        rho_train=float(spearmanr(fit.predict(Xv[train_idx]), y[train_idx]).statistic),
        rho_test=float(spearmanr(fit.predict(Xv[test_idx]), y[test_idx]).statistic),
    )


@dataclass
class ResidualResult:
    n: int
    n_train: int
    n_test: int
    fit: LinearFit
    rho_test: float
    top: list[dict]


@dataclass
class DecompositionResult:
    n_labeled: int
    auc_trace_full: float
    n_loo: int
    n_loo_train: int
    n_loo_test: int
    auc_task_only: float
    auc_task_only_all: float
    auc_trace_loo: float
    auc_both: float
    coef_task_in_both: float
    residual: ResidualResult


def decomposition(df: pd.DataFrame) -> DecompositionResult:
    """Held-out AUC on the score.py split: task rate alone vs trace features vs both."""
    labeled = df[df["resolved"].isin([0, 1])]
    y = labeled["resolved"].to_numpy(dtype=int)
    groups = labeled["instance_id"].astype(str).to_numpy()
    X = labeled[MODEL_FEATURES].to_numpy(dtype=float)
    splitter = GroupShuffleSplit(n_splits=1, test_size=TEST_SIZE, random_state=SEED)
    train_idx, test_idx = next(splitter.split(X, y, groups))
    fit_full = fit_logistic(X[train_idx], y[train_idx], X[test_idx], y[test_idx], MODEL_FEATURES)

    loo = leave_one_out_solve_rate(df).loc[labeled.index].to_numpy(dtype=float)
    has = ~np.isnan(loo)
    tr, te = train_idx[has[train_idx]], test_idx[has[test_idx]]
    X_tr, X_te = X[tr], X[te]
    y_tr, y_te = y[tr], y[te]
    loo_tr, loo_te = loo[tr], loo[te]

    fit_trace = fit_logistic(X_tr, y_tr, X_te, y_te, MODEL_FEATURES)
    both_features = [*MODEL_FEATURES, "loo_solve_rate"]
    fit_both = fit_logistic(
        np.column_stack([X_tr, loo_tr]),
        y_tr,
        np.column_stack([X_te, loo_te]),
        y_te,
        both_features,
    )
    coef_task = next(d["coef"] for d in fit_both.coefs() if d["feature"] == "loo_solve_rate")

    resid = labeled["resolved"].to_numpy(dtype=float) - loo
    fit_resid = fit_linear(X_tr, resid[tr], X_te, resid[te], MODEL_FEATURES)
    X_has, resid_has = X[has], resid[has]
    top = sorted(
        (
            {"feature": f, "rho": float(spearmanr(X_has[:, i], resid_has).statistic), "n": int(has.sum())}
            for i, f in enumerate(MODEL_FEATURES)
        ),
        key=lambda d: abs(d["rho"]),
        reverse=True,
    )

    return DecompositionResult(
        n_labeled=len(labeled),
        auc_trace_full=float(fit_full.auc_test),
        n_loo=int(has.sum()),
        n_loo_train=len(tr),
        n_loo_test=len(te),
        auc_task_only=float(roc_auc_score(y_te, loo_te)),
        auc_task_only_all=float(roc_auc_score(y[has], loo[has])),
        auc_trace_loo=float(fit_trace.auc_test),
        auc_both=float(fit_both.auc_test),
        coef_task_in_both=float(coef_task),
        residual=ResidualResult(
            n=int(has.sum()),
            n_train=len(tr),
            n_test=len(te),
            fit=fit_resid,
            rho_test=float(spearmanr(fit_resid.predict(X_te), resid[te]).statistic),
            top=top,
        ),
    )


def bucket_table(df: pd.DataFrame, inst: pd.DataFrame) -> pd.DataFrame:
    bucket = inst.set_index("instance_id")["difficulty_bucket"]
    traj_counts = bucket.reindex(df["instance_id"]).value_counts()
    grouped = inst.groupby("difficulty_bucket")
    rows = []
    for name in BUCKET_ORDER:
        sub = grouped.get_group(name) if name in grouped.groups else inst.iloc[0:0]
        n_traj = int(traj_counts.get(name, 0))
        rows.append(
            {
                "bucket": name,
                "instances": len(sub),
                "instance_share": len(sub) / len(inst),
                "trajectories": n_traj,
                "trajectory_share": n_traj / len(df),
                "mean_n_rollouts": float(sub["n_rollouts"].mean()) if len(sub) else np.nan,
                "mean_solve_rate": float(sub["solve_rate"].mean()) if len(sub) else np.nan,
            }
        )
    return pd.DataFrame(rows)


N_LABELED_BINS = [(0, 0), (1, 1), (2, 2), (3, 5), (6, 10), (11, 20), (21, None)]


def n_labeled_table(inst: pd.DataFrame) -> pd.DataFrame:
    n = inst["n_labeled"]
    rows = []
    for lo, hi in N_LABELED_BINS:
        mask = (n >= lo) if hi is None else ((n >= lo) & (n <= hi))
        label = f"{lo}" if lo == hi else (f"{lo}+" if hi is None else f"{lo}\u2013{hi}")
        rows.append({"n_labeled": label, "instances": int(mask.sum()), "share": float(mask.mean())})
    return pd.DataFrame(rows)


def bucket_score_table(df: pd.DataFrame, inst: pd.DataFrame) -> pd.DataFrame:
    labeled = df[df["resolved"].isin([0, 1])]
    bucket = inst.set_index("instance_id")["difficulty_bucket"]
    d = pd.DataFrame(
        {
            "bucket": bucket.reindex(labeled["instance_id"]).to_numpy(),
            "resolved": labeled["resolved"].to_numpy(dtype=float),
            "p_resolved": labeled["p_resolved"].to_numpy(dtype=float),
        }
    )
    rows = []
    for name in BUCKET_ORDER:
        sub = d[d["bucket"] == name]
        rows.append(
            {
                "bucket": name,
                "n": len(sub),
                "actual": float(sub["resolved"].mean()) if len(sub) else np.nan,
                "mean_p": float(sub["p_resolved"].mean()) if len(sub) else np.nan,
            }
        )
    out = pd.DataFrame(rows)
    out["gap"] = out["mean_p"] - out["actual"]
    return out


def _learn_row(harness: str, teacher: str, sub: pd.DataFrame) -> dict:
    dropped = sub[sub["dropped"]]
    chars = float(sub["assistant_chars"].sum())
    dropped_chars = float(dropped["assistant_chars"].sum())
    return {
        "harness": harness,
        "teacher": teacher,
        "n_rollouts": len(sub),
        "n_dropped": len(dropped),
        "dropped_share": len(dropped) / len(sub) if len(sub) else np.nan,
        "chars": chars,
        "chars_dropped": dropped_chars,
        "chars_dropped_share": dropped_chars / chars if chars else np.nan,
        "tokens_dropped": dropped_chars / 4.0,
    }


def learnability_table(df: pd.DataFrame, inst: pd.DataFrame) -> pd.DataFrame:
    bucket = inst.set_index("instance_id")["difficulty_bucket"]
    d = df[["harness", "teacher", "instance_id", "assistant_chars"]].copy()
    d["bucket"] = bucket.reindex(d["instance_id"]).to_numpy()
    missing = int(pd.isna(d["bucket"]).sum())
    if missing:
        raise SystemExit(f"{missing:,} trajectories have no instance difficulty row")
    d["dropped"] = d["bucket"].isin(DROPPED_BUCKETS)
    rows = [
        _learn_row(str(harness), str(teacher), sub)
        for (harness, teacher), sub in d.groupby(["harness", "teacher"], sort=True)
    ]
    rows.append(_learn_row("TOTAL", "", d))
    return pd.DataFrame(rows)


def pct(value: float) -> str:
    return f"{100 * value:.1f}%"


def build_summary(
    df: pd.DataFrame,
    inst: pd.DataFrame,
    task_text: pd.DataFrame,
    task_model: TaskModelResult,
    decomp: DecompositionResult,
    learn: pd.DataFrame,
    parquet_shown: Path,
    text_shown: Path,
) -> str:
    lines: list[str] = []
    stamp = datetime.now(UTC).strftime("%Y-%m-%d %H:%M UTC")
    lines.append("# Task difficulty per instance — summary")
    lines.append("")
    lines.append(
        f"Generated {stamp} by `scripts/task_difficulty.py` from `outputs/proxy_features.parquet`, "
        "`outputs/temporal_features.parquet`, `outputs/trace_scores.parquet`, and "
        f"`{text_shown}` (streamed from `traces_data/`); per-instance difficulty in "
        f"`{parquet_shown}`."
    )
    lines.append("")

    labeled = df[df["resolved"].isin([0, 1])]
    lines.append(
        f"Trajectories **{len(df):,}**; instances **{len(inst):,}**; labeled trajectories "
        f"**{len(labeled):,}** (resolved rate {labeled['resolved'].mean():.3f}); solve rates over "
        f"{int(inst['solve_rate'].notna().sum()):,} instances with at least one labeled rollout."
    )
    lines.append("")
    n_missing_issue = int(inst["issue_chars"].isna().sum())
    lines.append(
        f"Task text: {len(task_text):,} instances in `{text_shown}`; `issue_chars` "
        f"present for {len(inst) - n_missing_issue:,}/{len(inst):,} instances "
        f"({int((task_text['n_issue_variants'] > 1).sum()):,} instances had differing first-message "
        f"lengths across shards — the max is kept; {int((task_text['n_repo_variants'] > 1).sum()):,} "
        "had differing repos)."
    )
    lines.append("")

    lines.append("## 1. Difficulty distribution")
    lines.append("")
    n_lab = inst["n_labeled"]
    sr = inst["solve_rate"].dropna()
    qs = [0.10, 0.25, 0.50, 0.75, 0.90]
    lines.append(
        "Quantiles per instance; `solve_rate` is defined only where at least one rollout is "
        "labeled."
    )
    lines.append("")
    rows = [
        [
            "n_labeled",
            f"{len(inst):,}",
            f"{n_lab.mean():.2f}",
            *[f"{n_lab.quantile(q):.0f}" for q in qs],
            f"{n_lab.max():.0f}",
        ],
        [
            "solve_rate",
            f"{len(sr):,}",
            f"{sr.mean():.3f}",
            *[f"{sr.quantile(q):.3f}" for q in qs],
            f"{sr.max():.3f}",
        ],
    ]
    lines.append(
        md_table(["series", "n", "mean", "p10", "p25", "median", "p75", "p90", "max"], rows)
    )
    lines.append("")
    ntab = n_labeled_table(inst)
    rows = [[r["n_labeled"], f"{r['instances']:,}", pct(r["share"])] for _, r in ntab.iterrows()]
    lines.append(md_table(["n_labeled", "instances", "share of instances"], rows))
    lines.append("")
    lines.append(
        f"Solve rate is a discrete grid (n_resolved / n_labeled): exactly 0 for "
        f"{int((sr == 0).sum()):,} instances ({pct(float((sr == 0).mean()))} of defined) and exactly "
        f"1 for {int((sr == 1).sum()):,} ({pct(float((sr == 1).mean()))})."
    )
    lines.append("")
    bt = bucket_table(df, inst)
    rows = [
        [
            r["bucket"],
            f"{r['instances']:,}",
            pct(r["instance_share"]),
            f"{r['trajectories']:,}",
            pct(r["trajectory_share"]),
            "n/a" if pd.isna(r["mean_n_rollouts"]) else f"{r['mean_n_rollouts']:.1f}",
            "n/a" if pd.isna(r["mean_solve_rate"]) else f"{r['mean_solve_rate']:.3f}",
        ]
        for _, r in bt.iterrows()
    ]
    lines.append(
        md_table(
            [
                "bucket",
                "instances",
                "share of instances",
                "trajectories",
                "share of trajectories",
                "mean n_rollouts",
                "mean solve_rate",
            ],
            rows,
        )
    )
    lines.append("")

    lines.append("## 2. Task-only predictor of solve_rate")
    lines.append("")
    fit = task_model.fit
    lines.append(
        f"WLS on {task_model.n_instances:,} instances (weight = n_labeled, total weight "
        f"{task_model.weight_total:,.0f}), grouped 80/20 split by repo: {task_model.n_train:,} "
        f"train instances over {task_model.n_repos_train:,} repos, {task_model.n_test:,} test "
        f"instances over {task_model.n_repos_test:,} repos. Numeric inputs are winsorized to the "
        f"train split's {WINSOR_PCT[0]}st/{WINSOR_PCT[1]}th percentile then standardized; "
        "language/category are one-hot with the most frequent level dropped; harness mix drops "
        f"`{task_model.baseline_harness}` as the baseline fraction. R² is the unweighted fit "
        "metric; the fit itself is weighted."
    )
    lines.append("")
    rows = [
        ["train", f"{fit.r2_train:.4f}", f"{task_model.rho_train:+.4f}"],
        ["test (held out)", f"{fit.r2_test:.4f}", f"{task_model.rho_test:+.4f}"],
    ]
    lines.append(md_table(["split", "R²", "Spearman(pred, actual)"], rows))
    lines.append("")
    lines.append("Top standardized coefficients (largest |coef|, solve_rate per 1 sd):")
    lines.append("")
    rows = [
        [
            str(i + 1),
            d["feature"],
            f"{d['coef']:+.4f}",
            "higher solve_rate" if d["coef"] > 0 else "lower solve_rate",
        ]
        for i, d in enumerate(fit.coefs()[:12])
    ]
    lines.append(md_table(["rank", "feature", "std coef", "direction"], rows))
    lines.append("")

    lines.append("## 3. Trace classifier decomposition: task difficulty vs trace behavior")
    lines.append("")
    lines.append(
        "The trace classifier is the 27-feature proxy + temporal union from `score.py` under its "
        f"grouped 80/20 split by instance_id (seed {SEED}); on all {decomp.n_labeled:,} labeled "
        f"rows it reproduces the published held-out AUC **{decomp.auc_trace_full:.4f}**. The "
        f"decomposition reuses that split but restricts to the {decomp.n_loo:,} labeled trajectories "
        "whose instance has at least one other labeled trajectory, so the leave-one-out task rate is "
        f"defined ({decomp.n_loo_train:,} train / {decomp.n_loo_test:,} test)."
    )
    lines.append("")
    rows = [
        [
            "task solve_rate alone (leave-one-out)",
            "1",
            f"{decomp.auc_task_only:.4f}",
            f"{decomp.auc_task_only - decomp.auc_trace_loo:+.4f}",
        ],
        ["trace features alone (same subset)", "27", f"{decomp.auc_trace_loo:.4f}", "—"],
        ["task + trace", "28", f"{decomp.auc_both:.4f}", f"{decomp.auc_both - decomp.auc_trace_loo:+.4f}"],
        ["reference: trace features alone, all labeled rows", "27", f"{decomp.auc_trace_full:.4f}", "—"],
    ]
    lines.append(md_table(["predictor", "n features", "held-out AUC", "Δ vs trace alone"], rows))
    lines.append("")
    lines.append(
        f"Leave-one-out task rate alone scores AUC {decomp.auc_task_only_all:.4f} over all "
        f"{decomp.n_loo:,} defined rows (no fitting, so held-out by construction); the standardized "
        f"leave-one-out coefficient in the combined model is {decomp.coef_task_in_both:+.4f} "
        "(log-odds per 1 sd)."
    )
    lines.append("")
    res = decomp.residual
    lines.append("### Within-task residual (resolved − leave-one-out solve_rate)")
    lines.append("")
    lines.append(
        f"Linear fit of the residual on the 27 trace features on the same split ({res.n:,} rows; "
        f"{res.n_train:,} train / {res.n_test:,} test): held-out R² {res.fit.r2_test:.4f}, "
        f"Spearman(pred, residual) {res.rho_test:+.4f}."
    )
    lines.append("")
    lines.append("Top feature associations with the residual (Spearman, all rows):")
    lines.append("")
    rows = [
        [
            str(i + 1),
            d["feature"],
            f"{d['rho']:+.4f}",
            "beats siblings" if d["rho"] > 0 else "loses to siblings",
            f"{d['n']:,}",
        ]
        for i, d in enumerate(res.top[:8])
    ]
    lines.append(md_table(["rank", "feature", "Spearman with residual", "direction", "n"], rows))
    lines.append("")
    lines.append("### Classifier score vs difficulty buckets (labeled rows)")
    lines.append("")
    bs = bucket_score_table(df, inst)
    rows = [
        [
            r["bucket"],
            f"{r['n']:,}",
            "n/a" if pd.isna(r["actual"]) else f"{r['actual']:.3f}",
            "n/a" if pd.isna(r["mean_p"]) else f"{r['mean_p']:.3f}",
            "n/a" if pd.isna(r["gap"]) else f"{r['gap']:+.3f}",
        ]
        for _, r in bs.iterrows()
    ]
    lines.append(md_table(["bucket", "n labeled", "actual solve rate", "mean p_resolved", "gap"], rows))
    lines.append("")

    lines.append("## 4. Learnability strip: dropping all_pass + all_fail tasks")
    lines.append("")
    total = learn.iloc[-1]
    lines.append(
        f"Dropping both degenerate buckets removes **{int(total['n_dropped']):,}** trajectories "
        f"({pct(total['dropped_share'])}) and **{total['tokens_dropped'] / 1e6:,.1f}M** estimated "
        f"tokens ({pct(total['chars_dropped_share'])} of assistant chars; tokens ≈ chars / 4)."
    )
    lines.append("")
    rows = []
    for _, r in learn.iterrows():
        label = "TOTAL" if r["harness"] == "TOTAL" else f"{r['harness']}/{r['teacher']}"
        rows.append(
            [
                label,
                f"{int(r['n_rollouts']):,}",
                f"{int(r['n_dropped']):,}",
                pct(r["dropped_share"]),
                f"{r['chars'] / 1e6:,.1f}",
                f"{r['chars_dropped'] / 1e6:,.1f}",
                pct(r["chars_dropped_share"]),
                f"{r['tokens_dropped'] / 1e6:,.1f}",
            ]
        )
    lines.append(
        md_table(
            [
                "harness/teacher",
                "n",
                "dropped n",
                "dropped %",
                "chars (M)",
                "dropped chars (M)",
                "dropped chars %",
                "dropped tokens (M)",
            ],
            rows,
        )
    )
    lines.append("")

    lines.append("## Notes")
    lines.append("")
    lines.append(
        f"- `difficulty_bucket` needs at least {MIN_LABELED} labeled rollouts; the boundaries are "
        f"all_fail (solve_rate == 0), hard (< {HARD_MAX}), mid ({HARD_MAX}–{EASY_MIN}), easy "
        f"(> {EASY_MIN}), all_pass (== 1); everything else is `unknown`."
    )
    lines.append(
        "- `n_labeled` counts rollouts with `resolved in (0, 1)`; rollouts with `resolved == -1` "
        "never enter solve_rate."
    )
    lines.append(
        "- The leave-one-out task rate for a trajectory is the solve rate of its instance excluding "
        "the trajectory itself; it is defined only for labeled trajectories whose instance has at "
        "least one other labeled trajectory."
    )
    lines.append(
        "- The leave-one-out task rate is near-deterministic on degenerate tasks (all_fail / "
        "all_pass), so its standalone AUC is a ceiling on task-level information rather than a "
        "deployable signal for unseen instances; the within-task residual section is the "
        "behavior-only view on mixed tasks."
    )
    lines.append(
        "- `assistant_chars` is near zero for `openhands/deepseek_v4_flash` (median 0; its assistant "
        "text lives in reasoning fields), so that combo's token column understates its true size."
    )
    lines.append(
        "- `issue_chars` is `len(messages[2].content)` (the first user message, i.e. the PR "
        "description) from one streaming pass over the corpus; where shards disagree the maximum is "
        "kept."
    )
    lines.append(
        "- `language` is normalized (`ts` → `typescript`, `js` → `javascript`); instances whose "
        "harnesses disagree on the language label take the modal value."
    )
    lines.append(
        "- Reproduce: `uv run python scripts/task_difficulty.py` (add `--stream-only` to run just "
        "the corpus pass, `--skip-stream` to reuse `outputs/task_text.parquet`)."
    )
    lines.append("")
    return "\n".join(lines)


def write_difficulty(inst: pd.DataFrame, out_path: Path = OUT_PARQUET) -> Path:
    out_path.parent.mkdir(parents=True, exist_ok=True)
    tmp = Path(str(out_path) + ".tmp")
    inst.to_parquet(tmp, index=False)
    os.replace(tmp, out_path)
    return out_path


def resolve_path(value: Path) -> Path:
    return value if value.is_absolute() else (ROOT / value)


def rel(path: Path) -> str:
    try:
        return str(path.relative_to(ROOT))
    except ValueError:
        return str(path)


def main_task_difficulty() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--proxy", type=Path, default=PROXY_PARQUET)
    parser.add_argument("--temporal", type=Path, default=TEMPORAL_PARQUET)
    parser.add_argument("--scores", type=Path, default=SCORES_PARQUET)
    parser.add_argument("--text", type=Path, default=TEXT_PARQUET)
    parser.add_argument("--output", type=Path, default=OUT_PARQUET)
    parser.add_argument("--summary", type=Path, default=OUT_MD)
    parser.add_argument("--file", action="append", help="Process only these shard(s); repeatable")
    parser.add_argument("--limit", type=int, help="Process at most N shards (sorted order)")
    parser.add_argument(
        "--data-glob", default=PARQUET_GLOB, help="Parquet glob (default: the corpus)"
    )
    parser.add_argument("--force", action="store_true", help="Recompute task-text parts")
    parser.add_argument("--threads", type=int, default=4, help="DuckDB threads for the task-text pass")
    parser.add_argument(
        "--memory-limit", default="4GB", help="DuckDB memory limit for the task-text pass"
    )
    parser.add_argument("--stream-only", action="store_true", help="Run the corpus pass and exit")
    parser.add_argument(
        "--skip-stream", action="store_true", help="Skip the corpus pass; reuse --text"
    )
    args = parser.parse_args()

    args.proxy = resolve_path(args.proxy)
    args.temporal = resolve_path(args.temporal)
    args.scores = resolve_path(args.scores)
    args.text = resolve_path(args.text)
    args.output = resolve_path(args.output)
    args.summary = resolve_path(args.summary)

    if args.stream_only and args.skip_stream:
        parser.error("--stream-only and --skip-stream are mutually exclusive")
    if not args.skip_stream:
        rc = stream_task_text(args)
        if rc != 0 or args.stream_only:
            sys.exit(rc)
    if not args.text.exists():
        raise SystemExit(f"No task text at {args.text}; run without --skip-stream first")

    df = load_trajectory_frame(args.proxy, args.temporal, args.scores)
    task_text = load_task_text(args.text)
    inst = compute_instance_frame(df, task_text)
    out = write_difficulty(inst, args.output)
    console.print(f"Wrote {len(inst):,} instances → {rel(out)}")

    harnesses = sorted(df["harness"].dropna().unique().tolist())
    task_model = fit_task_model(inst, harnesses)
    decomp = decomposition(df)
    learn = learnability_table(df, inst)

    console.print(
        f"Task model (solve_rate ~ task features): test R² {task_model.fit.r2_test:.4f}, "
        f"Spearman {task_model.rho_test:+.4f}"
    )
    console.print(
        f"Held-out AUC — task rate alone {decomp.auc_task_only:.4f} | trace features "
        f"{decomp.auc_trace_loo:.4f} | both {decomp.auc_both:.4f} "
        f"(reference all labeled {decomp.auc_trace_full:.4f})"
    )
    console.print(
        f"Residual model: test R² {decomp.residual.fit.r2_test:.4f}, "
        f"Spearman {decomp.residual.rho_test:+.4f}"
    )

    md = build_summary(
        df, inst, task_text, task_model, decomp, learn, rel(args.output), rel(args.text)
    )
    args.summary.parent.mkdir(parents=True, exist_ok=True)
    args.summary.write_text(md)
    console.print(f"Wrote {rel(args.summary)}")


if __name__ == "__main__":
    main_task_difficulty()

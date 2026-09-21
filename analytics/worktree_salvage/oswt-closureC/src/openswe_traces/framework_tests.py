r"""Framework tests on Open-SWE-Traces: interaction, discrimination, mechanism.

The task-space framework (``analytics/research/task_space_framework.md``) decomposes a
task's difficulty into a *tree term* (how much the surrounding code determines the
change; proxied by the closure structure of the gold patch), an *instruction term*
(issue-text rung, binary ``L < 2`` vs ``L >= 2`` per the revised decision in
``task_space_framework.md``), and a *capacity term* (environment/horizon).

Per the sibling closure job (oswt-closureB), on this corpus the diff-text closure
``ratio`` does NOT add to size (size-matched top-vs-bottom ratio quartile diff -0.004;
std coef +0.009), while ``log1p(added_lines)`` is the dominant difficulty term (std coef
-0.12). The PRIMARY difficulty proxy for every test here is therefore ``added_lines``
(``log1p``, standardized), with ``ratio`` and ``new_frac`` as SECONDARY variants in the
same tables. Three predictions:

- T1 **interaction**: the instruction benefit depends on the difficulty proxy —
  per-trajectory logistic ``resolved ~ rung_binary * z(proxy) + log1p(added_lines)
  + log1p(n_files) + language one-hots`` with repo fixed effects, grouped 80/20 split
  by repo (seed 42); the interaction coefficient is reported with a cluster bootstrap CI
  (200 resamples over instances) and a 2x3 ``rung_binary x tercile -> mean solve_rate``
  table (terciles of ``added_lines`` and of ``ratio``). Proxy variants: ``z(log1p
  (added_lines))`` (primary), ``z(ratio)`` (secondary), ``z(new_frac)`` (secondary).
- T2 **discrimination**: IRT ``a_2pl`` separates solvers and should track the tree term
  (``ratio``), while size (``added_lines``) should not — Spearman and OLS of ``a_2pl``
  and of ``b_2pl`` on ``ratio``, ``new_frac``, ``log1p(added_lines)`` among instances
  with ``n_labeled >= 3`` and mixed outcomes.
- T3 **mechanism**: among unresolved trajectories, high-ratio failures should have HIGH
  ``patch_file_jaccard`` (right files, wrong content) — mean jaccard by ``added_lines``
  tercile (primary) and by ``ratio`` tercile.

Each test runs on the top-3 teacher/harness combos by IRT ability (``theta_2pl``) AND on
all combos, reported side by side.

Data plumbing:

- Per-trajectory frame (``outputs/trajectory_frame.parquet``): one row per trajectory
  with ``trajectory_id, instance_id, repo, language, harness, teacher, source, resolved,
  patch_file_jaccard``, streamed shard-by-shard (parts in ``outputs/trajectory_frame_parts/``).
- Per-instance rung labels from ``scripts/rung_mapping.py --stream-only``
  (``outputs/rung_features.parquet``); ``rung_binary = rung >= 2``.
- Per-instance closure proxies from the sibling closure job
  (``outputs/closure_proxies.parquet``).
- The IRT fit reuses the committed ``openswe_traces.irt`` 2PL model (the same model
  ``scripts/fit_irt.py`` fits) on the streamed labeled rollouts; per-combo ``theta_2pl``
  ranks the combos, per-task ``a_2pl``/``b_2pl`` feed T2.

Examples:
  uv run python scripts/framework_tests.py --skip-stream
  uv run python scripts/framework_tests.py --closure /path/to/closure_proxies.parquet
"""

from __future__ import annotations

import os

# liblinear is OpenMP-parallel; the bootstrap parallelizes across fits (joblib threads),
# so each fit must run single-threaded or the 16-way bootstrap would thrash.
os.environ.setdefault("OMP_NUM_THREADS", "1")

import argparse
import sys
import time
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path

import numpy as np
import pandas as pd
from joblib import Parallel, delayed
from rich.console import Console
from scipy import sparse
from scipy.stats import spearmanr
from sklearn.linear_model import LinearRegression, LogisticRegression
from sklearn.metrics import roc_auc_score
from sklearn.model_selection import GroupShuffleSplit
from sklearn.preprocessing import StandardScaler

import duckdb

from .data import PARQUET_GLOB, ROOT, connect_ephemeral, list_parquet_files, parse_shard
from .difficulty import LANGUAGE_ALIASES, MIN_LABELED
from .features import _sql_str, fmt_duration, process_file, utcnow
from .irt import build_irt_data, combo_ability_frame, fit_irt_data, task_irt_frame
from .score import md_table

console = Console()

TRAJ_PARTS_DIR = ROOT / "outputs" / "trajectory_frame_parts"
TRAJ_PARQUET = ROOT / "outputs" / "trajectory_frame.parquet"
RUNG_PARQUET = ROOT / "outputs" / "rung_features.parquet"
CLOSURE_PARQUET = ROOT / "outputs" / "closure_proxies.parquet"
OUT_MD = ROOT / "analytics" / "research" / "framework_tests_openswe.md"

SEED = 42
TEST_SIZE = 0.2
N_BOOTSTRAP = 200
TOP_K = 3
N_JOBS = 16

# (column in the analysis frame, display name, is-primary-size-proxy)
T1_VARIANTS: tuple[tuple[str, str, bool], ...] = (
    ("log1p_added_lines", "size log1p(added_lines)", True),
    ("ratio", "closure ratio", False),
    ("new_frac", "new_frac", False),
)

TRAJ_FEATURE_SQL = r"""
WITH src AS (
    SELECT trajectory_id, instance_id, repo, language, resolved, metadata
    FROM read_parquet('{file_sql}', union_by_name=true)
),
patches AS (
    SELECT
        t.trajectory_id,
        coalesce(t.metadata.model_patch.patch, '') AS model_patch,
        coalesce(t.metadata.reference_patch.patch, '') AS gold_patch
    FROM src t
),
patch_paths AS (
    SELECT
        trajectory_id,
        CASE WHEN len(git_paths) > 0 THEN list_distinct(git_paths) ELSE list_distinct(plus_paths) END AS model_paths,
        CASE WHEN len(gold_git_paths) > 0 THEN list_distinct(gold_git_paths) ELSE list_distinct(gold_plus_paths) END AS gold_paths
    FROM (
        SELECT
            trajectory_id,
            regexp_extract_all(model_patch, 'diff --git a/([^\s]+) b/', 1) AS git_paths,
            regexp_extract_all(model_patch, '\+\+\+ b/([^\s]+)', 1) AS plus_paths,
            regexp_extract_all(gold_patch, 'diff --git a/([^\s]+) b/', 1) AS gold_git_paths,
            regexp_extract_all(gold_patch, '\+\+\+ b/([^\s]+)', 1) AS gold_plus_paths
        FROM patches
    )
),
jaccard AS (
    SELECT
        trajectory_id,
        CASE
            WHEN len(list_distinct(list_concat(model_paths, gold_paths))) = 0 THEN 0.0
            ELSE len(list_filter(list_distinct(model_paths), p -> list_contains(gold_paths, p)))::DOUBLE
                 / len(list_distinct(list_concat(model_paths, gold_paths)))
        END AS patch_file_jaccard
    FROM patch_paths
)
SELECT
    t.trajectory_id,
    t.instance_id,
    t.repo,
    t.language,
    '{harness}' AS harness,
    '{teacher}' AS teacher,
    '{source}' AS source,
    t.resolved::TINYINT AS resolved,
    j.patch_file_jaccard
FROM src t
LEFT JOIN jaccard j USING (trajectory_id)
"""


def build_traj_sql(file_path: Path, harness: str, teacher: str, source: str) -> str:
    return TRAJ_FEATURE_SQL.format(
        file_sql=str(file_path).replace("'", "''"),
        harness=harness.replace("'", "''"),
        teacher=teacher.replace("'", "''"),
        source=source.replace("'", "''"),
    )


def stream_trajectory_frame(args: argparse.Namespace) -> int:
    """One streaming pass over the corpus; one row per trajectory, resume-safe."""
    TRAJ_PARTS_DIR.mkdir(parents=True, exist_ok=True)
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
                status, rows = process_file(
                    con, file_path, force=args.force, parts_dir=TRAJ_PARTS_DIR,
                    sql_builder=build_traj_sql,
                )
            except Exception as exc:  # noqa: BLE001
                failed += 1
                failures.append(str(file_path))
                console.print(f"[{utcnow()}] [{i}/{len(files)}] [red]FAILED[/red] {file_path}: {exc}")
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
                f"rows={rows:,} in {elapsed:.1f}s (avg {rate:.1f}s, eta {fmt_duration(eta)})"
            )
        if failures:
            console.print(f"[{utcnow()}] {failed} shard(s) failed — not merging")
        else:
            merged = merge_traj_parts(con, out_path=args.output)
            if merged:
                n_parts, rows = merged
                console.print(
                    f"[{utcnow()}] merged {n_parts} parts -> {args.output}: {rows:,} trajectories"
                )
    except KeyboardInterrupt:
        console.print(f"[{utcnow()}] interrupted — parts kept, rerun to resume")
        return 130
    finally:
        con.close()
    return 1 if failures else 0


def merge_traj_parts(
    con: duckdb.DuckDBPyConnection,
    *,
    parts_dir: Path = TRAJ_PARTS_DIR,
    out_path: Path = TRAJ_PARQUET,
) -> tuple[int, int] | None:
    parts = sorted(parts_dir.glob("*.parquet"))
    if not parts:
        return None
    glob_sql = str(parts_dir / "*.parquet").replace("'", "''")
    tmp = Path(str(out_path) + ".tmp")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    con.execute(
        f"COPY (SELECT * FROM read_parquet('{glob_sql}') "
        f"ORDER BY harness, teacher, source, trajectory_id) TO {_sql_str(tmp)} (FORMAT PARQUET)"
    )
    rows = con.execute(f"SELECT count(*) FROM read_parquet({_sql_str(tmp)})").fetchone()[0]
    os.replace(tmp, out_path)
    return len(parts), int(rows)


def load_trajectory_frame(path: Path = TRAJ_PARQUET) -> pd.DataFrame:
    con = connect_ephemeral()
    try:
        df = con.execute(f"SELECT * FROM read_parquet({_sql_str(path)})").df()
    finally:
        con.close()
    df["combo"] = df["harness"].astype(str) + "/" + df["teacher"].astype(str)
    df["language"] = df["language"].map(LANGUAGE_ALIASES).fillna(df["language"])
    return df


def load_analysis_frame(
    traj: pd.DataFrame,
    rung_path: Path = RUNG_PARQUET,
    closure_path: Path = CLOSURE_PARQUET,
) -> pd.DataFrame:
    """Merge rung_binary + closure proxies onto the trajectory frame.

    Instances without a rung label get ``rung_binary = NaN`` and are dropped (a missing
    label must not be read as L < 2).
    """
    con = connect_ephemeral()
    try:
        rung = con.execute(f"SELECT instance_id, rung FROM read_parquet({_sql_str(rung_path)})").df()
        closure = con.execute(
            f"SELECT instance_id, ratio, new_frac, added_lines, n_files, n_labeled, n_resolved "
            f"FROM read_parquet({_sql_str(closure_path)})"
        ).df()
    finally:
        con.close()
    frame = traj.merge(rung, on="instance_id", how="left")
    frame = frame.merge(closure, on="instance_id", how="left", suffixes=("", "_proxy"))
    frame["rung_binary"] = np.where(frame["rung"].isna(), np.nan, (frame["rung"] >= 2).astype(float))
    frame["log1p_added_lines"] = np.log1p(frame["added_lines"].to_numpy(dtype=float))
    frame = frame.dropna(subset=["ratio", "new_frac", "added_lines", "n_files", "rung_binary"])
    return frame.reset_index(drop=True)


def fit_task_irt(rollouts: pd.DataFrame) -> tuple[pd.DataFrame, pd.DataFrame]:
    """2PL IRT on labeled rollouts — same model as scripts/fit_irt.py.

    Returns (per-task a_2pl/b_2pl/n_labeled/solve_rate, per-combo theta_2pl).
    """
    data = build_irt_data(rollouts, min_labeled=MIN_LABELED)
    fit_1pl = fit_irt_data(data, two_pl=False)
    fit_2pl = fit_irt_data(data, two_pl=True)
    return task_irt_frame(data, fit_1pl, fit_2pl), combo_ability_frame(data, fit_1pl, fit_2pl)


def top_combos(combo: pd.DataFrame, k: int = TOP_K) -> list[str]:
    return combo.sort_values("theta_2pl", ascending=False).head(k)["combo"].tolist()


@dataclass
class T1Result:
    label: str
    variant: str
    n_trajectories: int
    n_instances: int
    n_repos: int
    n_train: int
    n_test: int
    interaction_coef: float
    ci_lo: float
    ci_hi: float
    auc_test: float
    feature_fit_seconds: float


def _zscore(series: pd.Series) -> np.ndarray:
    values = series.to_numpy(dtype=float)
    return (values - values.mean()) / max(values.std(), 1e-12)


def _language_dummies(languages: pd.Series) -> pd.DataFrame:
    values = languages.astype("string")
    dummies = pd.get_dummies(values, prefix="lang", dtype=float)
    counts = values.value_counts().sort_index()
    drop = f"lang_{counts.idxmax()}"
    if drop in dummies.columns:
        dummies = dummies.drop(columns=drop)
    return dummies.reset_index(drop=True)


def _build_t1_base(
    frame: pd.DataFrame, *, use_repo_fe: bool
) -> tuple[sparse.csr_matrix, list[str]]:
    """Language one-hots + repo fixed effects: shared by every T1 variant of a scope.

    Building the repo dummies dominates the design cost (one dense column per repo), so
    it is built once per scope and reused across the interaction variants.
    """
    lang = _language_dummies(frame["language"])
    blocks: list[sparse.spmatrix] = [sparse.csr_matrix(lang.to_numpy(dtype=float))]
    cols = list(lang.columns)
    if use_repo_fe:
        repos = frame["repo"].astype("string").reset_index(drop=True)
        dummies = pd.get_dummies(repos, prefix="repo", dtype=float)
        dummies = dummies.drop(columns=dummies.columns[0])
        blocks.append(sparse.csr_matrix(dummies.to_numpy(dtype=float)))
        cols += list(dummies.columns)
    return sparse.hstack(blocks, format="csr"), cols


def _build_t1_design(
    frame: pd.DataFrame,
    variant: str,
    *,
    use_repo_fe: bool,
    base: tuple[sparse.csr_matrix, list[str]] | None = None,
) -> tuple[sparse.csr_matrix, list[str]]:
    """Design for one T1 variant: rung_binary x z(variant) + size controls + langs + repo FE.

    The interaction variable is z-scored over the analysis set so the main fit and every
    bootstrap resample share one standardization. The primary variant is itself the size
    proxy, so its raw ``log1p(added_lines)`` control is not duplicated.
    """
    rung = frame["rung_binary"].to_numpy(dtype=float)
    if variant == "log1p_added_lines":
        zvar = _zscore(pd.Series(np.log1p(frame["added_lines"].to_numpy(dtype=float))))
    else:
        zvar = _zscore(frame[variant])
    continuous = pd.DataFrame(
        {
            "rung_binary": rung,
            f"{variant}_std": zvar,
            f"rung_binary:{variant}_std": rung * zvar,
        }
    )
    if variant != "log1p_added_lines":
        continuous["log1p_added_lines"] = np.log1p(frame["added_lines"].to_numpy(dtype=float))
    continuous["log1p_n_files"] = np.log1p(frame["n_files"].to_numpy(dtype=float))
    base_X, base_cols = base if base is not None else _build_t1_base(frame, use_repo_fe=use_repo_fe)
    cols = list(continuous.columns) + list(base_cols)
    return sparse.hstack(
        [sparse.csr_matrix(continuous.to_numpy(dtype=float)), base_X], format="csr"
    ), cols


def _fit_interaction_logit(
    X: sparse.csr_matrix, y: np.ndarray, rows: np.ndarray, interaction_col: int
) -> float:
    model = LogisticRegression(solver="liblinear", max_iter=1000, tol=1e-4).fit(X[rows], y[rows])
    return float(model.coef_[0, interaction_col])


def _cluster_bootstrap_ci(
    X: sparse.csr_matrix,
    y: np.ndarray,
    inst: np.ndarray,
    interaction_col: int,
    *,
    n_bootstrap: int,
    seed: int,
    n_jobs: int,
) -> tuple[float, float]:
    """Cluster bootstrap over instances: resample instances, refit, CI from the 2.5-97.5% quantiles."""
    unique_inst = np.unique(inst)
    rows_by_inst = [np.where(inst == iid)[0] for iid in unique_inst]
    rng = np.random.default_rng(seed)
    draws = [rng.integers(0, len(unique_inst), size=len(unique_inst)) for _ in range(n_bootstrap)]
    rows = [np.concatenate([rows_by_inst[i] for i in draw]) for draw in draws]
    coefs = Parallel(n_jobs=n_jobs, backend="threading")(
        delayed(_fit_interaction_logit)(X, y, r, interaction_col) for r in rows
    )
    coefs = np.sort(np.asarray(coefs, dtype=float))
    return float(np.quantile(coefs, 0.025)), float(np.quantile(coefs, 0.975))


def t1_interaction(
    frame: pd.DataFrame,
    *,
    label: str,
    variant: str,
    n_bootstrap: int = N_BOOTSTRAP,
    seed: int = SEED,
    use_repo_fe: bool = True,
    n_jobs: int = N_JOBS,
    base: tuple[sparse.csr_matrix, list[str]] | None = None,
) -> T1Result:
    """Per-trajectory logistic with rung_binary x z(variant) interaction; repo FE.

    ``base`` is the prebuilt language+repo block for this scope (see ``_build_t1_base``);
    it is built once per scope and shared across the interaction variants.
    """
    labeled = frame[frame["resolved"].isin([0, 1])].reset_index(drop=True)
    if base is not None and base[0].shape[0] != len(labeled):
        raise ValueError(
            "base must be built on the labeled subset of frame (same row order), "
            f"got {base[0].shape[0]} rows for {len(labeled)} labeled rows"
        )
    y = labeled["resolved"].to_numpy(dtype=int)
    groups = labeled["repo"].astype(str).to_numpy()
    splitter = GroupShuffleSplit(n_splits=1, test_size=TEST_SIZE, random_state=seed)
    train_idx, test_idx = next(splitter.split(np.zeros(len(labeled)), y, groups))

    X_full, cols = _build_t1_design(labeled, variant, use_repo_fe=use_repo_fe, base=base)
    interaction_col = cols.index(f"rung_binary:{variant}_std")

    t0 = time.monotonic()
    model = LogisticRegression(solver="liblinear", max_iter=1000, tol=1e-4).fit(
        X_full[train_idx], y[train_idx]
    )
    fit_seconds = time.monotonic() - t0
    coef = float(model.coef_[0, interaction_col])
    if len(np.unique(y[test_idx])) < 2:
        auc_test = float("nan")
    else:
        auc_test = float(roc_auc_score(y[test_idx], model.predict_proba(X_full[test_idx])[:, 1]))

    ci_lo, ci_hi = _cluster_bootstrap_ci(
        X_full, y, labeled["instance_id"].to_numpy(), interaction_col,
        n_bootstrap=n_bootstrap, seed=seed, n_jobs=n_jobs,
    )
    return T1Result(
        label=label,
        variant=variant,
        n_trajectories=len(labeled),
        n_instances=labeled["instance_id"].nunique(),
        n_repos=labeled["repo"].nunique(),
        n_train=len(train_idx),
        n_test=len(test_idx),
        interaction_coef=coef,
        ci_lo=ci_lo,
        ci_hi=ci_hi,
        auc_test=auc_test,
        feature_fit_seconds=fit_seconds,
    )


def _rank_terciles(values: pd.Series) -> pd.Series:
    """Three terciles on rank(method='first') — tie-safe for integer-valued proxies."""
    return pd.qcut(values.rank(method="first"), 3, labels=["low", "mid", "high"])


def t1_table(frame: pd.DataFrame, column: str) -> pd.DataFrame:
    """2x3 rung_binary x <column> tercile -> mean solve_rate (n labeled trajectories)."""
    labeled = frame[frame["resolved"].isin([0, 1])].reset_index(drop=True)
    inst = labeled.drop_duplicates("instance_id")[["instance_id", column]].reset_index(drop=True)
    inst[f"{column}_tercile"] = _rank_terciles(inst[column])
    merged = labeled.merge(inst[["instance_id", f"{column}_tercile"]], on="instance_id", how="left")
    merged = merged.dropna(subset=[f"{column}_tercile"])
    rows = []
    for rb in sorted(merged["rung_binary"].unique()):
        for terc in ["low", "mid", "high"]:
            sub = merged[(merged["rung_binary"] == rb) & (merged[f"{column}_tercile"] == terc)]
            rows.append(
                {
                    "rung_binary": int(rb),
                    "tercile": terc,
                    "mean_solve_rate": float(sub["resolved"].mean()) if len(sub) else np.nan,
                    "n": len(sub),
                }
            )
    return pd.DataFrame(rows)


@dataclass
class T2Result:
    label: str
    n_fitted: int
    n_mixed: int
    table: pd.DataFrame


def t2_discrimination(
    task_irt: pd.DataFrame,
    closure: pd.DataFrame,
    *,
    label: str,
) -> T2Result:
    """Spearman + univariate OLS of a_2pl / b_2pl on closure proxies (mixed tasks)."""
    merged = task_irt.merge(
        closure[["instance_id", "ratio", "new_frac", "added_lines"]],
        on="instance_id",
        how="inner",
    )
    mixed = merged[
        (merged["n_labeled"] >= MIN_LABELED)
        & (merged["solve_rate"] > 0)
        & (merged["solve_rate"] < 1)
    ].reset_index(drop=True)
    mixed["log1p_added_lines"] = np.log1p(mixed["added_lines"].to_numpy(dtype=float))
    features = ["ratio", "new_frac", "log1p_added_lines"]
    rows = []
    for target in ("a_2pl", "b_2pl"):
        y = mixed[target].to_numpy(dtype=float)
        for feature in features:
            x = mixed[feature].to_numpy(dtype=float)
            if x.std() == 0:
                rho, beta = np.nan, np.nan
            else:
                rho = float(spearmanr(x, y).statistic)
                Xs = StandardScaler().fit_transform(x.reshape(-1, 1))
                model = LinearRegression().fit(Xs, y)
                beta = float(model.coef_[0])
            rows.append(
                {
                    "target": target,
                    "feature": feature,
                    "spearman": rho,
                    "ols_std_beta": beta,
                    "n": len(mixed),
                }
            )
    return T2Result(
        label=label,
        n_fitted=len(merged),
        n_mixed=len(mixed),
        table=pd.DataFrame(rows),
    )


@dataclass
class T3Result:
    label: str
    n_unresolved: int
    by_ratio: pd.DataFrame
    by_added: pd.DataFrame


def _tercile_means(frame: pd.DataFrame, column: str) -> pd.DataFrame:
    inst = frame.drop_duplicates("instance_id")[["instance_id", column]].reset_index(drop=True)
    inst[f"{column}_tercile"] = _rank_terciles(inst[column])
    merged = frame.merge(inst[["instance_id", f"{column}_tercile"]], on="instance_id", how="left")
    merged = merged.dropna(subset=[f"{column}_tercile"])
    rows = []
    for terc in ["low", "mid", "high"]:
        sub = merged[merged[f"{column}_tercile"] == terc]
        rows.append(
            {
                "tercile": terc,
                "mean_patch_file_jaccard": float(sub["patch_file_jaccard"].mean()) if len(sub) else np.nan,
                "n": len(sub),
            }
        )
    return pd.DataFrame(rows)


def t3_mechanism(frame: pd.DataFrame, *, label: str) -> T3Result:
    """Among unresolved trajectories: mean patch_file_jaccard by ratio / added tercile."""
    unresolved = frame[frame["resolved"] == 0].reset_index(drop=True)
    return T3Result(
        label=label,
        n_unresolved=len(unresolved),
        by_ratio=_tercile_means(unresolved, "ratio"),
        by_added=_tercile_means(unresolved, "added_lines"),
    )


def _fmt_float(value: float, digits: int = 3) -> str:
    return "nan" if np.isnan(value) else f"{value:.{digits}f}"


def _fmt_ci(value: float, digits: int = 3) -> str:
    return "nan" if np.isnan(value) else f"{value:+.{digits}f}"


def _t1_side_by_side(
    t1_all: list[T1Result], t1_top: list[T1Result]
) -> list[str]:
    header = [
        "interaction (rung_binary x)",
        "coef all",
        "CI all",
        "coef top-3",
        "CI top-3",
        "n all",
        "n top-3",
    ]
    by_variant_all = {r.variant: r for r in t1_all}
    by_variant_top = {r.variant: r for r in t1_top}
    rows = []
    for variant, display, _ in T1_VARIANTS:
        a, t = by_variant_all[variant], by_variant_top[variant]
        rows.append(
            [
                display,
                _fmt_ci(a.interaction_coef),
                f"[{_fmt_ci(a.ci_lo)}, {_fmt_ci(a.ci_hi)}]",
                _fmt_ci(t.interaction_coef),
                f"[{_fmt_ci(t.ci_lo)}, {_fmt_ci(t.ci_hi)}]",
                f"{a.n_trajectories:,}",
                f"{t.n_trajectories:,}",
            ]
        )
    return [md_table(header, rows)]


def _t1_tercile_side_by_side(
    column: str,
    factor_label: str,
    table_all: pd.DataFrame,
    table_top: pd.DataFrame,
) -> list[str]:
    header = [
        "rung_binary",
        "tercile",
        "mean solve_rate all",
        "n all",
        "mean solve_rate top-3",
        "n top-3",
    ]
    rows = []
    for rb in sorted(table_all["rung_binary"].unique()):
        for terc in ["low", "mid", "high"]:
            a = table_all[(table_all["rung_binary"] == rb) & (table_all["tercile"] == terc)]
            t = table_top[(table_top["rung_binary"] == rb) & (table_top["tercile"] == terc)]
            rows.append(
                [
                    str(int(rb)),
                    terc,
                    _fmt_float(a["mean_solve_rate"].iloc[0]),
                    f"{int(a['n'].iloc[0]):,}",
                    _fmt_float(t["mean_solve_rate"].iloc[0]),
                    f"{int(t['n'].iloc[0]):,}",
                ]
            )
    return [f"rung_binary x {factor_label} tercile -> mean solve_rate:", "", md_table(header, rows)]


def _t2_side_by_side(t2_all: T2Result, t2_top: T2Result) -> list[str]:
    header = [
        "target",
        "feature",
        "rho all",
        "beta all",
        "rho top-3",
        "beta top-3",
        "n all",
        "n top-3",
    ]
    rows = []
    for target in ("a_2pl", "b_2pl"):
        for feature in ("ratio", "new_frac", "log1p_added_lines"):
            a = t2_all.table[(t2_all.table["target"] == target) & (t2_all.table["feature"] == feature)].iloc[0]
            t = t2_top.table[(t2_top.table["target"] == target) & (t2_top.table["feature"] == feature)].iloc[0]
            rows.append(
                [
                    target,
                    feature,
                    _fmt_float(a["spearman"]),
                    _fmt_float(a["ols_std_beta"]),
                    _fmt_float(t["spearman"]),
                    _fmt_float(t["ols_std_beta"]),
                    f"{int(a['n']):,}",
                    f"{int(t['n']):,}",
                ]
            )
    return [md_table(header, rows)]


def _t3_side_by_side(t3_all: T3Result, t3_top: T3Result) -> list[str]:
    header = ["factor", "tercile", "mean jaccard all", "n all", "mean jaccard top-3", "n top-3"]
    rows = []
    for factor, factor_label in (("added_lines", "added_lines (size, primary)"), ("ratio", "ratio (tree term)")):
        a = t3_all.by_added if factor == "added_lines" else t3_all.by_ratio
        t = t3_top.by_added if factor == "added_lines" else t3_top.by_ratio
        for terc in ["low", "mid", "high"]:
            ar = a[a["tercile"] == terc].iloc[0]
            tr = t[t["tercile"] == terc].iloc[0]
            rows.append(
                [
                    factor_label,
                    terc,
                    _fmt_float(ar["mean_patch_file_jaccard"]),
                    f"{int(ar['n']):,}",
                    _fmt_float(tr["mean_patch_file_jaccard"]),
                    f"{int(tr['n']):,}",
                ]
            )
    return [md_table(header, rows)]


def build_report(
    combo: pd.DataFrame,
    top3: list[str],
    t1_all: list[T1Result],
    t1_top: list[T1Result],
    t1_tercile_all: dict[str, pd.DataFrame],
    t1_tercile_top: dict[str, pd.DataFrame],
    t2_all: T2Result,
    t2_top: T2Result,
    t3_all: T3Result,
    t3_top: T3Result,
    use_repo_fe: bool,
) -> str:
    stamp = datetime.now(UTC).strftime("%Y-%m-%d %H:%M UTC")
    theta = combo.set_index("combo")["theta_2pl"]
    top3_rows = [[c, f"{theta[c]:+.3f}"] for c in sorted(top3, key=lambda c: theta[c], reverse=True)]
    lines: list[str] = []
    lines.append("# Framework tests on Open-SWE-Traces: interaction, discrimination, mechanism")
    lines.append("")
    lines.append(f"Generated {stamp} by `scripts/framework_tests.py` (zero solver cost).")
    lines.append("")
    lines.append(
        "Framework: a task's difficulty decomposes into a *tree term* (how much the surrounding "
        "code determines the change; proxied by the closure structure of the gold patch, "
        "`ratio` = internal_refs / boundary_refs), an *instruction term* (issue-text rung, "
        "binary `L<2` vs `L>=2` per the revised decision in `task_space_framework.md`), and a "
        "*capacity term* (environment/horizon, held fixed here). Per the sibling closure job, "
        "`log1p(added_lines)` is the dominant difficulty proxy on this corpus (ratio adds "
        "nothing on top of size), so every test reports `log1p(added_lines)` as the PRIMARY "
        "proxy with `ratio` and `new_frac` as SECONDARY variants in the same table. Each test "
        "runs on the top-3 teacher/harness combos by IRT ability (`theta_2pl`) and on all "
        "combos, side by side."
    )
    lines.append("")
    lines.append(
        "Inputs: per-trajectory frame streamed from `traces_data/` (only `patch_file_jaccard`, "
        "`resolved`, combo + identity columns); per-instance rung labels from "
        "`outputs/rung_features.parquet` (`scripts/rung_mapping.py --stream-only`); per-instance "
        "closure proxies from the sibling closure job (`outputs/closure_proxies.parquet`); IRT "
        "refit with the committed `openswe_traces.irt` 2PL model (the same model "
        "`scripts/fit_irt.py` fits) on the labeled rollouts."
    )
    lines.append("")
    lines.append(f"Top-{TOP_K} combos by theta_2pl:")
    lines.append("")
    lines.append(md_table(["combo", "theta_2pl"], top3_rows))
    lines.append("")
    fe_note = "repo fixed effects" if use_repo_fe else "repo-demeaned continuous terms (FE too heavy)"
    lines.append(f"T1 uses {fe_note}; interaction CIs are 200 cluster bootstraps over instances (seed 42).")
    lines.append("")
    lines.append("## T1 interaction — instruction benefit vs difficulty proxy")
    lines.append("")
    lines.append(
        "Per-trajectory logistic `resolved ~ rung_binary * z(proxy) + log1p(added_lines) "
        "+ log1p(n_files) + language one-hots` with repo fixed effects, grouped 80/20 split "
        "by repo (seed 42). The interaction coefficient is the change in the rung benefit "
        "per 1 SD of the proxy."
    )
    lines.append("")
    lines.extend(_t1_side_by_side(t1_all, t1_top))
    lines.append("")
    lines.append("### 2x3 tables — rung_binary x tercile -> mean solve_rate")
    lines.append("")
    for column, factor_label in (("added_lines", "size (added_lines)"), ("ratio", "closure ratio")):
        lines.extend(
            _t1_tercile_side_by_side(
                column, factor_label, t1_tercile_all[column], t1_tercile_top[column]
            )
        )
        lines.append("")
    lines.append("## T2 discrimination — a_2pl / b_2pl vs ratio, new_frac, size")
    lines.append("")
    lines.append(
        f"Mixed tasks (n_labeled >= {MIN_LABELED}, 0 < solve_rate < 1): "
        f"{t2_all.n_mixed:,} of {t2_all.n_fitted:,} (all) / {t2_top.n_mixed:,} of "
        f"{t2_top.n_fitted:,} (top-3). OLS beta is per-1-SD of the feature."
    )
    lines.append("")
    lines.extend(_t2_side_by_side(t2_all, t2_top))
    lines.append("")
    lines.append("## T3 mechanism — patch_file_jaccard among unresolved trajectories")
    lines.append("")
    lines.append(
        f"Unresolved (`resolved == 0`) trajectories: {t3_all.n_unresolved:,} (all) / "
        f"{t3_top.n_unresolved:,} (top-3)."
    )
    lines.append("")
    lines.extend(_t3_side_by_side(t3_all, t3_top))
    lines.append("")
    lines.append("## Verdicts")
    lines.append("")
    t1_primary = next(r for r in t1_all if r.variant == "log1p_added_lines")
    verdict_parts = [
        (
            f"z(log1p(added_lines)) {t1_primary.interaction_coef:+.3f} "
            f"[{t1_primary.ci_lo:+.3f}, {t1_primary.ci_hi:+.3f}]"
        )
    ]
    for variant, display, _ in T1_VARIANTS[1:]:
        r = next(x for x in t1_all if x.variant == variant)
        verdict_parts.append(
            f"z({variant}) {r.interaction_coef:+.3f} [{r.ci_lo:+.3f}, {r.ci_hi:+.3f}]"
        )
    primary_held = t1_primary.ci_lo > 0 or t1_primary.ci_hi < 0
    ratio_r = next(x for x in t1_all if x.variant == "ratio")
    ratio_sig = ratio_r.ci_lo > 0 or ratio_r.ci_hi < 0
    if primary_held:
        verdict = "the prediction is held for the primary proxy (CI excludes 0)"
    else:
        verdict = "the prediction is not confirmed for the primary proxy (CI includes 0)"
    if ratio_sig:
        direction = (
            "negative — the rung benefit shrinks on tree-heavy tasks, the opposite of the prediction"
            if ratio_r.interaction_coef < 0
            else "positive"
        )
        verdict += f"; the rung x ratio interaction is the only significant one and is {direction}"
    else:
        verdict += "; no other interaction is significant"
    lines.append(
        "**T1 interaction** (all combos): rung_binary x "
        + "; rung_binary x ".join(verdict_parts)
        + f" — {verdict}."
    )
    lines.append("")
    ratio_row = t2_all.table[(t2_all.table["target"] == "a_2pl") & (t2_all.table["feature"] == "ratio")].iloc[0]
    added_row = t2_all.table[(t2_all.table["target"] == "a_2pl") & (t2_all.table["feature"] == "log1p_added_lines")].iloc[0]
    lines.append(
        "**T2 discrimination** (all combos): "
        f"Spearman(a_2pl, ratio) = {ratio_row['spearman']:+.3f} vs "
        f"Spearman(a_2pl, log1p(added_lines)) = {added_row['spearman']:+.3f} — the prediction "
        "(ratio relates to a, separates solvers; size less so) "
        + (
            "held."
            if abs(ratio_row["spearman"]) > abs(added_row["spearman"])
            else "not confirmed."
        )
    )
    lines.append("")
    hi = t3_all.by_ratio[t3_all.by_ratio["tercile"] == "high"]["mean_patch_file_jaccard"].iloc[0]
    lo = t3_all.by_ratio[t3_all.by_ratio["tercile"] == "low"]["mean_patch_file_jaccard"].iloc[0]
    lines.append(
        "**T3 mechanism** (all combos): among unresolved trajectories, mean patch_file_jaccard "
        f"{lo:.3f} (low ratio tercile) -> {hi:.3f} (high) — the prediction (high-ratio failures "
        "have HIGH file overlap, right files wrong content) "
        + ("held." if hi > lo else "not confirmed.")
    )
    lines.append("")
    lines.append("## Reproduce")
    lines.append("")
    lines.append("```bash")
    lines.append("uv run python scripts/rung_mapping.py --stream-only")
    lines.append("uv run python scripts/framework_tests.py --skip-stream")
    lines.append("uv run pytest tests/test_framework_tests.py")
    lines.append("```")
    lines.append("")
    return "\n".join(lines)


def main_framework_tests() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--output", type=Path, default=TRAJ_PARQUET)
    parser.add_argument("--summary", type=Path, default=OUT_MD)
    parser.add_argument("--rung", type=Path, default=RUNG_PARQUET)
    parser.add_argument("--closure", type=Path, default=CLOSURE_PARQUET)
    parser.add_argument("--file", action="append")
    parser.add_argument("--limit", type=int)
    parser.add_argument("--data-glob", default=PARQUET_GLOB)
    parser.add_argument("--force", action="store_true")
    parser.add_argument("--threads", type=int, default=4)
    parser.add_argument("--memory-limit", default="4GB")
    parser.add_argument("--stream-only", action="store_true")
    parser.add_argument("--skip-stream", action="store_true")
    parser.add_argument("--no-fe", action="store_true", help="Use repo-demeaned terms instead of repo FE")
    parser.add_argument("--bootstraps", type=int, default=N_BOOTSTRAP)
    parser.add_argument("--n-jobs", type=int, default=N_JOBS)
    args = parser.parse_args()

    if args.stream_only and args.skip_stream:
        parser.error("--stream-only and --skip-stream are mutually exclusive")
    if not args.skip_stream:
        rc = stream_trajectory_frame(args)
        if rc != 0 or args.stream_only:
            sys.exit(rc)

    traj = load_trajectory_frame(args.output)
    frame = load_analysis_frame(traj, args.rung, args.closure)
    console.print(
        f"trajectories {len(traj):,}; merged with rung+closure: {len(frame):,}; "
        f"instances {frame['instance_id'].nunique():,}"
    )

    labeled = frame[frame["resolved"].isin([0, 1])].reset_index(drop=True)
    task_irt_all, combo = fit_task_irt(labeled)
    console.print(
        f"IRT: {len(task_irt_all):,} tasks, {len(combo)} combos; "
        f"top-3 by theta_2pl: {', '.join(top_combos(combo))}"
    )
    top3 = top_combos(combo)
    top3_mask = frame["combo"].isin(top3)

    task_irt_top3, _ = fit_task_irt(labeled[labeled["combo"].isin(top3)])

    use_repo_fe = not args.no_fe
    # t1_interaction filters to labeled rows internally, so the shared lang+repo base
    # block must be built on the same labeled subset (rows must align).
    labeled_all = labeled
    labeled_top = labeled_all[labeled_all["combo"].isin(top3)].reset_index(drop=True)
    base_all = _build_t1_base(labeled_all, use_repo_fe=use_repo_fe)
    base_top = _build_t1_base(labeled_top, use_repo_fe=use_repo_fe)
    t1_all = []
    for variant, display, _ in T1_VARIANTS:
        r = t1_interaction(
            labeled_all, label="all combos", variant=variant,
            n_bootstrap=args.bootstraps, use_repo_fe=use_repo_fe,
            n_jobs=args.n_jobs, base=base_all,
        )
        t1_all.append(r)
        console.print(
            f"[{utcnow()}] T1 all/{display}: coef {r.interaction_coef:+.4f} "
            f"[{r.ci_lo:+.4f}, {r.ci_hi:+.4f}] auc {r.auc_test:.4f} "
            f"(fit {r.feature_fit_seconds:.1f}s)"
        )
    t1_top = []
    for variant, display, _ in T1_VARIANTS:
        r = t1_interaction(
            labeled_top, label="top-3 combos", variant=variant,
            n_bootstrap=args.bootstraps, use_repo_fe=use_repo_fe,
            n_jobs=args.n_jobs, base=base_top,
        )
        t1_top.append(r)
        console.print(
            f"[{utcnow()}] T1 top3/{display}: coef {r.interaction_coef:+.4f} "
            f"[{r.ci_lo:+.4f}, {r.ci_hi:+.4f}] auc {r.auc_test:.4f} "
            f"(fit {r.feature_fit_seconds:.1f}s)"
        )
    t1_tercile_all = {col: t1_table(frame, col) for col in ("added_lines", "ratio")}
    t1_tercile_top = {col: t1_table(frame[top3_mask], col) for col in ("added_lines", "ratio")}

    closure = frame[["instance_id", "ratio", "new_frac", "added_lines"]].drop_duplicates("instance_id")
    t2_top = t2_discrimination(task_irt_top3, closure, label="top-3 combos")
    t2_all = t2_discrimination(task_irt_all, closure, label="all combos")
    t3_top = t3_mechanism(frame[top3_mask], label="top-3 combos")
    t3_all = t3_mechanism(frame, label="all combos")

    report = build_report(
        combo, top3, t1_all, t1_top, t1_tercile_all, t1_tercile_top,
        t2_all, t2_top, t3_all, t3_top, use_repo_fe,
    )
    args.summary.parent.mkdir(parents=True, exist_ok=True)
    args.summary.write_text(report)
    console.print(f"Wrote {args.summary}")

    console.print("\n[bold]T1 interaction (all combos)[/bold]")
    for r in t1_all:
        console.print(
            f"  {r.variant}: coef {r.interaction_coef:+.4f} "
            f"[{r.ci_lo:+.4f}, {r.ci_hi:+.4f}] auc {r.auc_test:.4f}"
        )
    console.print("\n[bold]T2 discrimination (all combos)[/bold]")
    console.print(t2_all.table.to_string(index=False))
    console.print("\n[bold]T3 mechanism (all combos)[/bold]")
    console.print(t3_all.by_ratio.to_string(index=False))
    console.print(t3_all.by_added.to_string(index=False))


if __name__ == "__main__":
    main_framework_tests()

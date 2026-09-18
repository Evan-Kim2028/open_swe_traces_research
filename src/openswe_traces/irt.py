"""Item-response-theory task difficulty: model-independent difficulty per instance.

``P(resolved) = sigmoid(a_task * (theta_combo - b_task))`` with ``combo`` = harness/teacher
(7 labeled abilities). ``a_task`` is the task's discrimination and ``b_task`` its difficulty
on the ability scale: b is comparable across combos because theta absorbs how strong each
harness/teacher was (a plain solve rate does not).

Fit: full-batch Adam on the penalized negative log-likelihood over labeled rollouts
(``resolved in (0, 1)``) of instances with >= 3 labeled rollouts::

    NLL + 0.5 * sum(theta^2) + 0.5 * sum(b^2) + (1 / (2 * 0.5^2)) * sum(log a^2)

i.e. priors theta ~ N(0, 1), b ~ N(0, 1), log a ~ N(0, 0.5) (std 0.5). A 1PL model
(a fixed to 1) is fit for comparison with the same objective and optimizer. The priors also
anchor the joint location of the ability scale: the likelihood alone would be invariant to
shifting theta and b by a common constant.

Outputs:

  outputs/task_irt.parquet      instance_id, b_1pl, b_2pl, a_2pl, n_labeled, solve_rate
  outputs/combo_ability.parquet combo, harness, teacher, n_labeled, resolved_rate,
                                theta_1pl, theta_2pl
  outputs/task_difficulty.parquet  adds an ``irt_bucket`` column (all existing columns are
                                preserved): the b_2pl quantile bucket, with cuts chosen so
                                bucket sizes match the pass-rate ``difficulty_bucket`` shares
                                on the fitted set; instances with < 3 labeled rollouts are
                                ``unknown``. Rebuilding task_difficulty.parquet drops the
                                column again — rerun this script after it.
  analytics/research/irt_summary.md   combo abilities vs observed rates, agreement between
                                b-buckets and pass-rate buckets with the combo mix that
                                explains disagreements, the a_2pl distribution, and held-out
                                log-likelihood 1PL vs 2PL vs the pass-rate baseline.

Examples:
  uv run python scripts/fit_irt.py
  uv run python scripts/fit_irt.py --steps 4000 --no-bucket
"""

from __future__ import annotations

import argparse
import os
import warnings
from collections.abc import Mapping
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path

import numpy as np
import pandas as pd
from rich.console import Console
from scipy.special import expit
from scipy.stats import spearmanr
from sklearn.metrics import roc_auc_score

from .data import ROOT, connect_ephemeral
from .difficulty import MIN_LABELED
from .features import _sql_str
from .score import md_table

console = Console()

PROXY_PARQUET = ROOT / "outputs" / "proxy_features.parquet"
DIFFICULTY_PARQUET = ROOT / "outputs" / "task_difficulty.parquet"
OUT_TASK_PARQUET = ROOT / "outputs" / "task_irt.parquet"
OUT_COMBO_PARQUET = ROOT / "outputs" / "combo_ability.parquet"
OUT_MD = ROOT / "analytics" / "research" / "irt_summary.md"

STEPS = 2000
LR = 0.05
BETA1 = 0.9
BETA2 = 0.999
ADAM_EPS = 1e-8
LOG_A_SD = 0.5
LOG_A_PRECISION = 1.0 / LOG_A_SD**2
HISTORY_EVERY = 100
TEST_SIZE = 0.2
SEED = 42
LOG_EPS = 1e-12
WEAK_K = 2

EASIEST_ORDER = ("all_pass", "easy", "mid", "hard", "all_fail")  # ascending b
BUCKET_VOCAB = tuple(reversed(EASIEST_ORDER))  # display order, hardest first
HARDNESS_RANK = {name: rank for rank, name in enumerate(BUCKET_VOCAB)}
UNKNOWN = "unknown"


def _spearman(x: np.ndarray, y: np.ndarray) -> float:
    with warnings.catch_warnings():
        warnings.simplefilter("ignore")
        return float(spearmanr(x, y).statistic)


def _auc(y: np.ndarray, p: np.ndarray) -> float:
    if len(y) == 0 or int(y.sum()) == 0 or int(y.sum()) == len(y):
        return float("nan")
    return float(roc_auc_score(y, p))


def load_labeled_rollouts(proxy_path: Path = PROXY_PARQUET) -> pd.DataFrame:
    """Labeled rollouts (``resolved in (0, 1)``) with a ``combo`` = harness/teacher column."""
    con = connect_ephemeral()
    try:
        df = con.execute(
            f"""
            SELECT trajectory_id, instance_id, harness, teacher, resolved
            FROM read_parquet({_sql_str(proxy_path)})
            WHERE resolved IN (0, 1)
            ORDER BY harness, teacher, trajectory_id
            """
        ).df()
    finally:
        con.close()
    df["combo"] = df["harness"].astype(str) + "/" + df["teacher"].astype(str)
    return df


@dataclass
class IrtData:
    """Labeled rollouts of instances with at least ``min_labeled`` labeled rollouts."""

    rollouts: pd.DataFrame
    tasks: list[str]
    combos: list[str]
    task_idx: np.ndarray
    combo_idx: np.ndarray
    y: np.ndarray

    @property
    def n_rollouts(self) -> int:
        return len(self.y)


def build_irt_data(rollouts: pd.DataFrame, min_labeled: int = MIN_LABELED) -> IrtData:
    counts = rollouts.groupby("instance_id")["resolved"].transform("size")
    df = (
        rollouts.loc[counts >= min_labeled]
        .sort_values(["instance_id", "combo", "trajectory_id"], kind="mergesort")
        .reset_index(drop=True)
    )
    if df.empty:
        raise SystemExit("no instances with enough labeled rollouts to fit IRT")
    tasks = sorted(df["instance_id"].unique().tolist())
    combos = sorted(df["combo"].unique().tolist())
    task_map = {name: i for i, name in enumerate(tasks)}
    combo_map = {name: i for i, name in enumerate(combos)}
    return IrtData(
        rollouts=df,
        tasks=tasks,
        combos=combos,
        task_idx=df["instance_id"].map(task_map).to_numpy(dtype=np.int64),
        combo_idx=df["combo"].map(combo_map).to_numpy(dtype=np.int64),
        y=df["resolved"].to_numpy(dtype=np.float64),
    )


def _objective(
    theta: np.ndarray,
    b: np.ndarray,
    log_a: np.ndarray,
    task_idx: np.ndarray,
    combo_idx: np.ndarray,
    y: np.ndarray,
    two_pl: bool,
) -> tuple[float, float]:
    a = np.exp(log_a) if two_pl else np.ones_like(log_a)
    z = a[task_idx] * (theta[combo_idx] - b[task_idx])
    nll = float(np.logaddexp(0.0, z).sum() - float((y * z).sum()))
    penalty = 0.5 * float(theta @ theta) + 0.5 * float(b @ b)
    if two_pl:
        penalty += 0.5 * LOG_A_PRECISION * float(log_a @ log_a)
    return nll + penalty, nll


@dataclass
class IrtFit:
    two_pl: bool
    steps: int
    lr: float
    theta: np.ndarray
    b: np.ndarray
    log_a: np.ndarray
    a: np.ndarray
    objective: float
    nll: float
    loss_history: list[tuple[int, float]]

    def predict_proba(self, task_idx: np.ndarray, combo_idx: np.ndarray) -> np.ndarray:
        a = self.a[task_idx]
        z = a * (self.theta[combo_idx] - self.b[task_idx])
        return expit(z)


def fit_irt(
    task_idx: np.ndarray,
    combo_idx: np.ndarray,
    y: np.ndarray,
    n_tasks: int,
    n_combos: int,
    *,
    two_pl: bool = True,
    steps: int = STEPS,
    lr: float = LR,
) -> IrtFit:
    """Full-batch Adam on the penalized NLL (priors fix the ability-scale location)."""
    theta = np.zeros(n_combos)
    b = np.zeros(n_tasks)
    log_a = np.zeros(n_tasks)
    moments = [np.zeros_like(x) for x in (theta, b, log_a)]
    squares = [np.zeros_like(x) for x in (theta, b, log_a)]
    history = [(0, _objective(theta, b, log_a, task_idx, combo_idx, y, two_pl)[0])]
    for step in range(1, steps + 1):
        a = np.exp(log_a) if two_pl else np.ones(n_tasks)
        z = a[task_idx] * (theta[combo_idx] - b[task_idx])
        gz = expit(z) - y
        g_theta = np.bincount(combo_idx, weights=gz * a[task_idx], minlength=n_combos) + theta
        g_b = -np.bincount(task_idx, weights=gz * a[task_idx], minlength=n_tasks) + b
        if two_pl:
            g_log_a = (
                np.bincount(
                    task_idx,
                    weights=gz * a[task_idx] * (theta[combo_idx] - b[task_idx]),
                    minlength=n_tasks,
                )
                + LOG_A_PRECISION * log_a
            )
        else:
            g_log_a = np.zeros(n_tasks)
        updates = ((g_theta, theta), (g_b, b), (g_log_a, log_a))
        for i, (grad, param) in enumerate(updates):
            moments[i] = BETA1 * moments[i] + (1.0 - BETA1) * grad
            squares[i] = BETA2 * squares[i] + (1.0 - BETA2) * grad * grad
            m_hat = moments[i] / (1.0 - BETA1**step)
            v_hat = squares[i] / (1.0 - BETA2**step)
            param[...] -= lr * m_hat / (np.sqrt(v_hat) + ADAM_EPS)
        if step % HISTORY_EVERY == 0 or step == steps:
            history.append((step, _objective(theta, b, log_a, task_idx, combo_idx, y, two_pl)[0]))
    objective, nll = _objective(theta, b, log_a, task_idx, combo_idx, y, two_pl)
    return IrtFit(
        two_pl=two_pl,
        steps=steps,
        lr=lr,
        theta=theta,
        b=b,
        log_a=log_a,
        a=np.exp(log_a) if two_pl else np.ones(n_tasks),
        objective=objective,
        nll=nll,
        loss_history=history,
    )


def fit_irt_data(
    data: IrtData, *, two_pl: bool = True, steps: int = STEPS, lr: float = LR
) -> IrtFit:
    return fit_irt(
        data.task_idx,
        data.combo_idx,
        data.y,
        len(data.tasks),
        len(data.combos),
        two_pl=two_pl,
        steps=steps,
        lr=lr,
    )


def task_irt_frame(data: IrtData, fit_1pl: IrtFit, fit_2pl: IrtFit) -> pd.DataFrame:
    """Per-instance difficulty table: b_1pl, b_2pl, a_2pl, n_labeled, solve_rate."""
    stats = data.rollouts.groupby("instance_id", sort=True)["resolved"].agg(["size", "mean"])
    return pd.DataFrame(
        {
            "instance_id": data.tasks,
            "b_1pl": fit_1pl.b,
            "b_2pl": fit_2pl.b,
            "a_2pl": fit_2pl.a,
            "n_labeled": stats["size"].reindex(data.tasks).to_numpy(dtype=np.int64),
            "solve_rate": stats["mean"].reindex(data.tasks).to_numpy(dtype=np.float64),
        }
    )


def combo_ability_frame(data: IrtData, fit_1pl: IrtFit, fit_2pl: IrtFit) -> pd.DataFrame:
    stats = data.rollouts.groupby("combo", sort=True)["resolved"].agg(["size", "mean"])
    return pd.DataFrame(
        {
            "combo": data.combos,
            "harness": [c.split("/", 1)[0] for c in data.combos],
            "teacher": [c.split("/", 1)[1] for c in data.combos],
            "n_labeled": stats["size"].reindex(data.combos).to_numpy(dtype=np.int64),
            "resolved_rate": stats["mean"].reindex(data.combos).to_numpy(dtype=np.float64),
            "theta_1pl": fit_1pl.theta,
            "theta_2pl": fit_2pl.theta,
        }
    )


@dataclass
class HeldOut:
    n_train: int
    n_test: int
    n_fit_tasks: int
    mean_ll_1pl: float
    auc_1pl: float
    mean_ll_2pl: float
    auc_2pl: float
    mean_ll_rate: float
    auc_rate: float
    n_degenerate: int
    n_mixed: int
    mean_ll_1pl_degenerate: float
    mean_ll_2pl_degenerate: float
    mean_ll_rate_degenerate: float
    mean_ll_1pl_mixed: float
    mean_ll_2pl_mixed: float
    mean_ll_rate_mixed: float


def _log_likelihood(y: np.ndarray, p: np.ndarray) -> float:
    p = np.clip(p, LOG_EPS, 1.0 - LOG_EPS)
    return float(np.mean(y * np.log(p) + (1.0 - y) * np.log(1.0 - p)))


def heldout_comparison(
    data: IrtData,
    buckets: pd.Series,
    *,
    test_size: float = TEST_SIZE,
    seed: int = SEED,
) -> HeldOut:
    """20% random split of rollouts: refit 1PL / 2PL on train, score test rollouts.

    The pass-rate baseline predicts each test rollout with its task's train solve rate,
    Jeffreys-smoothed ((resolved + 0.5) / (n + 1)) so degenerate rates do not hit the
    log-likelihood clip. ``buckets`` maps instance_id to its pass-rate difficulty bucket;
    log-likelihoods are also split into degenerate (all_fail/all_pass) and mixed
    (hard/mid/easy) test rollouts. Test rollouts are those whose task and combo were fitted
    on the train split.
    """
    rng = np.random.default_rng(seed)
    test_mask = rng.random(data.n_rollouts) < test_size
    train = build_irt_data(data.rollouts.loc[~test_mask])
    fit_1pl = fit_irt_data(train, two_pl=False)
    fit_2pl = fit_irt_data(train, two_pl=True)
    task_map = {name: i for i, name in enumerate(train.tasks)}
    combo_map = {name: i for i, name in enumerate(train.combos)}
    test_rows = data.rollouts.loc[test_mask]
    test_rows = test_rows[
        test_rows["instance_id"].isin(task_map) & test_rows["combo"].isin(combo_map)
    ]
    ti = test_rows["instance_id"].map(task_map).to_numpy(dtype=np.int64)
    ci = test_rows["combo"].map(combo_map).to_numpy(dtype=np.int64)
    y = test_rows["resolved"].to_numpy(dtype=np.float64)
    train_stats = train.rollouts.groupby("instance_id")["resolved"].agg(["sum", "size"])
    smoothed_rate = (train_stats["sum"] + 0.5) / (train_stats["size"] + 1.0)
    p_rate = test_rows["instance_id"].map(smoothed_rate).to_numpy(dtype=np.float64)
    p_1pl = fit_1pl.predict_proba(ti, ci)
    p_2pl = fit_2pl.predict_proba(ti, ci)
    degenerate = test_rows["instance_id"].map(buckets).isin(["all_fail", "all_pass"]).to_numpy()
    mixed = ~degenerate
    return HeldOut(
        n_train=len(train.rollouts),
        n_test=len(y),
        n_fit_tasks=len(train.tasks),
        mean_ll_1pl=_log_likelihood(y, p_1pl),
        auc_1pl=_auc(y, p_1pl),
        mean_ll_2pl=_log_likelihood(y, p_2pl),
        auc_2pl=_auc(y, p_2pl),
        mean_ll_rate=_log_likelihood(y, p_rate),
        auc_rate=_auc(y, p_rate),
        n_degenerate=int(degenerate.sum()),
        n_mixed=int(mixed.sum()),
        mean_ll_1pl_degenerate=_log_likelihood(y[degenerate], p_1pl[degenerate]),
        mean_ll_2pl_degenerate=_log_likelihood(y[degenerate], p_2pl[degenerate]),
        mean_ll_rate_degenerate=_log_likelihood(y[degenerate], p_rate[degenerate]),
        mean_ll_1pl_mixed=_log_likelihood(y[mixed], p_1pl[mixed]),
        mean_ll_2pl_mixed=_log_likelihood(y[mixed], p_2pl[mixed]),
        mean_ll_rate_mixed=_log_likelihood(y[mixed], p_rate[mixed]),
    )


def bucket_cuts(b: np.ndarray, shares: Mapping[str, float]) -> np.ndarray:
    """b thresholds whose bucket sizes match ``shares`` (increasing b = increasing ease)."""
    cumulative = np.cumsum([float(shares[name]) for name in EASIEST_ORDER[:-1]])
    return np.quantile(np.asarray(b, dtype=np.float64), cumulative)


def assign_quantile_buckets(b: np.ndarray, cuts: np.ndarray) -> np.ndarray:
    idx = np.searchsorted(cuts, np.asarray(b, dtype=np.float64), side="right")
    return np.asarray(EASIEST_ORDER, dtype=object)[idx]


def irt_bucket_frame(task_irt: pd.DataFrame, cuts: np.ndarray) -> pd.DataFrame:
    return pd.DataFrame(
        {
            "instance_id": task_irt["instance_id"].to_numpy(),
            "irt_bucket": assign_quantile_buckets(task_irt["b_2pl"].to_numpy(), cuts),
        }
    )


def add_irt_bucket(
    difficulty_path: Path, buckets: pd.DataFrame, out_path: Path | None = None
) -> tuple[Path, int]:
    """Add ``irt_bucket`` to task_difficulty.parquet; existing columns are kept as-is."""
    out_path = Path(out_path) if out_path is not None else Path(difficulty_path)
    frame = pd.read_parquet(difficulty_path)
    old_columns = list(frame.columns)
    if "irt_bucket" in old_columns:
        frame = frame.drop(columns=["irt_bucket"])
        old_columns.remove("irt_bucket")
    mapping = buckets.set_index("instance_id")["irt_bucket"]
    frame["irt_bucket"] = frame["instance_id"].map(mapping).fillna(UNKNOWN)
    n_unknown = int((frame["irt_bucket"] == UNKNOWN).sum())
    missing = [name for name in old_columns if name not in frame.columns]
    if missing or len(frame) == 0:
        raise RuntimeError(f"irt_bucket write would drop columns: {missing}")
    tmp = Path(f"{out_path}.tmp")
    frame.to_parquet(tmp, index=False)
    os.replace(tmp, out_path)
    return out_path, n_unknown


def instance_mix(data: IrtData, fit: IrtFit, weak_k: int = WEAK_K) -> pd.DataFrame:
    """Per-instance attempt mix: mean combo theta, weak/strong shares, distinct combo count."""
    theta_map = dict(zip(data.combos, fit.theta.tolist()))
    order = sorted(data.combos, key=lambda name: theta_map[name])
    weak = set(order[:weak_k])
    strong = set(order[-weak_k:])
    d = data.rollouts[["instance_id", "combo"]].copy()
    d["theta"] = d["combo"].map(theta_map)
    d["weak"] = d["combo"].isin(weak)
    d["strong"] = d["combo"].isin(strong)
    grouped = d.groupby("instance_id", sort=True)
    return pd.DataFrame(
        {
            "mix_theta": grouped["theta"].mean(),
            "weak_share": grouped["weak"].mean(),
            "strong_share": grouped["strong"].mean(),
            "n_combos": grouped["combo"].nunique(),
        }
    )


@dataclass
class DisagreementReport:
    n_fitted: int
    agreement: float
    cuts: np.ndarray
    confusion: pd.DataFrame
    groups: pd.DataFrame
    cells: pd.DataFrame
    rho_delta_mix_theta: float


def disagreement_report(
    data: IrtData, task_irt: pd.DataFrame, inst: pd.DataFrame, fit: IrtFit
) -> DisagreementReport:
    """b-bucket vs pass-rate bucket on the fitted set; combo mix per disagreement group."""
    fitted = task_irt.merge(
        inst[
            [
                "instance_id",
                "difficulty_bucket",
                "n_labeled",
                "language",
                "category",
                "gold_patch_files",
                "gold_patch_lines",
                "issue_chars",
            ]
        ],
        on="instance_id",
        how="left",
    )
    shares = fitted["difficulty_bucket"].value_counts(normalize=True).to_dict()
    cuts = bucket_cuts(fitted["b_2pl"].to_numpy(), shares)
    fitted["b_bucket"] = assign_quantile_buckets(fitted["b_2pl"].to_numpy(), cuts)
    fitted["delta"] = fitted["difficulty_bucket"].map(HARDNESS_RANK) - fitted["b_bucket"].map(
        HARDNESS_RANK
    )
    mix = instance_mix(data, fit)
    fitted = fitted.join(mix, on="instance_id")

    confusion = pd.crosstab(fitted["difficulty_bucket"], fitted["b_bucket"]).reindex(
        index=list(BUCKET_VOCAB), columns=list(BUCKET_VOCAB), fill_value=0
    )

    groups = []
    group_defs = [
        ("pass bucket easier than b bucket (delta > 0)", fitted["delta"] > 0),
        ("pass bucket harder than b bucket (delta < 0)", fitted["delta"] < 0),
        ("agree (delta = 0)", fitted["delta"] == 0),
        ("all fitted instances", fitted["delta"].notna()),
    ]
    for name, mask in group_defs:
        sub = fitted.loc[mask]
        groups.append(
            {
                "group": name,
                "instances": len(sub),
                "share": len(sub) / len(fitted),
                "mean_solve_rate": float(sub["solve_rate"].mean()),
                "mean_mix_theta": float(sub["mix_theta"].mean()),
                "mean_weak_share": float(sub["weak_share"].mean()),
                "mean_strong_share": float(sub["strong_share"].mean()),
                "median_n_combos": float(sub["n_combos"].median()),
            }
        )

    off = fitted[fitted["delta"] != 0]
    cell_counts = off.groupby(["difficulty_bucket", "b_bucket"], sort=False).size()
    cells = []
    for (pass_bucket, b_bucket), n in cell_counts.sort_values(ascending=False).items():
        sub = off[(off["difficulty_bucket"] == pass_bucket) & (off["b_bucket"] == b_bucket)]
        cells.append(
            {
                "pass_bucket": pass_bucket,
                "b_bucket": b_bucket,
                "instances": int(n),
                "mean_mix_theta": float(sub["mix_theta"].mean()),
                "mean_weak_share": float(sub["weak_share"].mean()),
                "mean_solve_rate": float(sub["solve_rate"].mean()),
                "mean_b_2pl": float(sub["b_2pl"].mean()),
            }
        )
    return DisagreementReport(
        n_fitted=len(fitted),
        agreement=float((fitted["delta"] == 0).mean()),
        cuts=cuts,
        confusion=confusion,
        groups=pd.DataFrame(groups),
        cells=pd.DataFrame(cells),
        rho_delta_mix_theta=_spearman(
            fitted["delta"].to_numpy(dtype=float), fitted["mix_theta"].to_numpy(dtype=float)
        ),
    )


@dataclass
class DiscriminationReport:
    quantiles: pd.DataFrame
    by_bucket: pd.DataFrame
    profile: pd.DataFrame
    language: pd.DataFrame
    category: pd.DataFrame
    corr: pd.DataFrame
    n_mixed: int


def _share_table(frame: pd.DataFrame, column: str, top: int = 5) -> pd.DataFrame:
    overall = frame[column].astype("string").fillna("(missing)")
    levels = overall.value_counts().head(top).index.tolist()
    rows = []
    for level in levels:
        rows.append(
            {
                column: level,
                "fitted_share": float((overall == level).mean()),
                "top_a_share": float((overall[frame["top_a"]] == level).mean()),
                "mixed_top_share": (
                    float((overall[frame["mixed_top_a"]] == level).mean())
                    if bool(frame["mixed_top_a"].any())
                    else float("nan")
                ),
            }
        )
    return pd.DataFrame(rows)


def discrimination_report(
    data: IrtData, task_irt: pd.DataFrame, inst: pd.DataFrame
) -> DiscriminationReport:
    fitted = task_irt.merge(
        inst[
            [
                "instance_id",
                "difficulty_bucket",
                "language",
                "category",
                "gold_patch_files",
                "gold_patch_lines",
                "issue_chars",
            ]
        ],
        on="instance_id",
        how="left",
    )
    a = fitted["a_2pl"].to_numpy(dtype=np.float64)
    quantiles = pd.DataFrame(
        {
            "quantile": ["min", "p05", "p25", "median", "p75", "p95", "max"],
            "a_2pl": [
                float(np.min(a)),
                float(np.quantile(a, 0.05)),
                float(np.quantile(a, 0.25)),
                float(np.median(a)),
                float(np.quantile(a, 0.75)),
                float(np.quantile(a, 0.95)),
                float(np.max(a)),
            ],
        }
    )
    diagonal = [name for name in BUCKET_VOCAB if name not in ("all_fail", "all_pass")]
    a_cut = float(np.quantile(a, 0.9))
    mixed = fitted[fitted["difficulty_bucket"].isin(diagonal)]
    mixed_cut = float(np.quantile(mixed["a_2pl"], 0.9)) if len(mixed) else float("nan")
    fitted["a_cut"] = a_cut
    fitted["top_a"] = fitted["a_2pl"] >= a_cut
    fitted["mixed_top_a"] = fitted["a_2pl"] >= mixed_cut

    rows = []
    n_top = int(fitted["top_a"].sum())
    for name in BUCKET_VOCAB:
        sub = fitted[fitted["difficulty_bucket"] == name]
        rows.append(
            {
                "bucket": name,
                "instances": len(sub),
                "fitted_share": len(sub) / len(fitted),
                "mean_a": float(sub["a_2pl"].mean()),
                "median_a": float(sub["a_2pl"].median()),
                "bucket_top_share": float(sub["top_a"].mean()) if len(sub) else np.nan,
                "top_decile_contribution": (float(sub["top_a"].sum()) / n_top if n_top else np.nan),
            }
        )
    by_bucket = pd.DataFrame(rows)

    numeric = [
        ("n_labeled", "n_labeled"),
        ("solve_rate", "solve_rate"),
        ("gold_patch_lines", "gold_patch_lines"),
        ("gold_patch_files", "gold_patch_files"),
        ("issue_chars", "issue_chars"),
    ]
    profile_rows = []
    top = fitted[fitted["top_a"]]
    bottom = fitted[(fitted["a_2pl"] <= float(np.quantile(a, 0.1)))]
    top_mixed = fitted[fitted["mixed_top_a"]]
    bottom_mixed = mixed[mixed["a_2pl"] <= float(np.quantile(mixed["a_2pl"], 0.1))]
    for label, column in numeric:
        profile_rows.append(
            {
                "metric": label,
                "top_decile_a": float(top[column].median()),
                "bottom_decile_a": float(bottom[column].median()),
                "top_decile_a_mixed": float(top_mixed[column].median()),
                "bottom_decile_a_mixed": float(bottom_mixed[column].median()),
            }
        )
    profile = pd.DataFrame(profile_rows)

    corr_rows = []
    for label, column in numeric:
        values = fitted[column].to_numpy(dtype=np.float64)
        corr_rows.append({"metric": label, "spearman_a": _spearman(a, values)})
    b_values = fitted["b_2pl"].to_numpy(dtype=np.float64)
    corr_rows.append(
        {
            "metric": "abs(b_2pl - median b_2pl)",
            "spearman_a": _spearman(a, np.abs(b_values - np.median(b_values))),
        }
    )
    corr = pd.DataFrame(corr_rows)

    return DiscriminationReport(
        quantiles=quantiles,
        by_bucket=by_bucket,
        profile=profile,
        language=_share_table(fitted, "language"),
        category=_share_table(fitted, "category"),
        corr=corr,
        n_mixed=len(mixed),
    )


def pct(value: float) -> str:
    return f"{100 * value:.1f}%"


def f4(value: float) -> str:
    return f"{value:+.4f}"


@dataclass
class IrtReport:
    data: IrtData
    fit_2pl: IrtFit
    fit_1pl: IrtFit
    task_irt: pd.DataFrame
    combo: pd.DataFrame
    heldout: HeldOut
    disagreement: DisagreementReport
    discrimination: DiscriminationReport
    n_instances_total: int
    steps: int
    lr: float


def build_summary(
    report: IrtReport, task_shown: str, combo_shown: str, difficulty_shown: str
) -> str:
    data = report.data
    lines: list[str] = []
    stamp = datetime.now(UTC).strftime("%Y-%m-%d %H:%M UTC")
    lines.append("# Task difficulty via IRT — model-independent difficulty and combo abilities")
    lines.append("")
    lines.append(
        f"Generated {stamp} by `scripts/fit_irt.py` from `outputs/proxy_features.parquet`; "
        f"models in `{task_shown}` and `{combo_shown}`; `{difficulty_shown}` gains an "
        "`irt_bucket` column."
    )
    lines.append("")
    lines.append(
        f"Fit: **{data.n_rollouts:,}** labeled rollouts of **{len(data.tasks):,}** instances "
        f"(>= {MIN_LABELED} labeled rollouts each; {report.n_instances_total:,} instances total) "
        f"across **{len(data.combos)}** labeled harness/teacher combos. "
        "`P(resolved) = sigmoid(a_task * (theta_combo - b_task))`, full-batch Adam "
        f"({report.steps} steps, lr {report.lr}); priors theta ~ N(0, 1), b ~ N(0, 1), "
        "log a ~ N(0, 0.5). The priors anchor the ability scale's location as well as its "
        "shrinkage: the likelihood alone is invariant to shifting theta and b by a common "
        "constant, so only differences within each set are meaningful."
    )
    lines.append("")

    lines.append("## 1. Combo abilities (theta) vs observed resolved rates")
    lines.append("")
    combo = report.combo.sort_values("theta_2pl", ascending=False)
    rows = [
        [
            str(i + 1),
            str(row["combo"]),
            f"{row['theta_2pl']:+.3f}",
            f"{row['theta_1pl']:+.3f}",
            f"{row['resolved_rate']:.3f}",
            f"{int(row['n_labeled']):,}",
        ]
        for i, (_, row) in enumerate(combo.iterrows())
    ]
    lines.append(
        md_table(
            ["rank", "combo", "theta_2pl", "theta_1pl", "observed rate", "n labeled"],
            rows,
        )
    )
    lines.append("")
    rho_2pl = _spearman(combo["theta_2pl"].to_numpy(), combo["resolved_rate"].to_numpy())
    rho_1pl = _spearman(combo["theta_1pl"].to_numpy(), combo["resolved_rate"].to_numpy())
    rho_both = _spearman(combo["theta_1pl"].to_numpy(), combo["theta_2pl"].to_numpy())
    b_by_task = dict(zip(data.tasks, report.fit_2pl.b.tolist()))
    attempted_b = (
        data.rollouts.assign(b=data.rollouts["instance_id"].map(b_by_task))
        .groupby("combo", sort=True)["b"]
        .mean()
    )
    hardest, easiest = str(attempted_b.idxmax()), str(attempted_b.idxmin())
    mm36, sw36 = "minisweagent/qwen36_27b", "sweagent/qwen36_27b"
    combo_stats = combo.set_index("combo")
    rate_gap = float(
        combo_stats.loc[sw36, "resolved_rate"] - combo_stats.loc[mm36, "resolved_rate"]
    )
    theta_gap = float(combo_stats.loc[sw36, "theta_2pl"] - combo_stats.loc[mm36, "theta_2pl"])
    lines.append(
        f"Rank agreement with the observed resolved rate: Spearman(theta_2pl, rate) "
        f"{rho_2pl:+.4f}, Spearman(theta_1pl, rate) {rho_1pl:+.4f}; the two ability scales "
        f"correlate at {rho_both:+.4f}. The ordering matches the raw rates, but the scale "
        f"separates combos the rates compress: {mm36} and {sw36} — same teacher, different "
        f"harness — differ by {rate_gap:.3f} in raw rate but {theta_gap:.3f} in theta, because "
        "rates on mostly-hard tasks saturate while theta keeps the log-odds gap. Task mix "
        f"differs across combos too: at the extremes, {hardest} ran the hardest slice and "
        f"{easiest} the easiest, "
        f"{attempted_b[hardest] - attempted_b[easiest]:.2f} b-units apart on the (arbitrary) b "
        "origin — theta is what puts those different mixes on one scale."
    )
    lines.append("")

    lines.append("## 2. b_2pl vs solve_rate")
    lines.append("")
    task = report.task_irt
    rho_b_sr = _spearman(task["b_2pl"].to_numpy(), -task["solve_rate"].to_numpy())
    rho_b1_sr = _spearman(task["b_1pl"].to_numpy(), -task["solve_rate"].to_numpy())
    rho_b_b1 = _spearman(task["b_2pl"].to_numpy(), task["b_1pl"].to_numpy())
    lines.append(
        f"Spearman(b_2pl, -solve_rate) {rho_b_sr:+.4f}; Spearman(b_1pl, -solve_rate) "
        f"{rho_b1_sr:+.4f}; Spearman(b_2pl, b_1pl) {rho_b_b1:+.4f}. b is on the combo-ability "
        "logit scale: b = k means a combo of ability k solves the task with P = 0.5, and one "
        "more unit of combo ability multiplies the task's solve odds by exp(a_task) — so b is "
        "difficulty net of which combos attempted the task, which the raw rate is not."
    )
    lines.append("")
    bq = np.quantile(task["b_2pl"].to_numpy(), [0.05, 0.25, 0.50, 0.75, 0.95])
    rows = [
        [
            "b_2pl",
            f"{bq[0]:+.3f}",
            f"{bq[1]:+.3f}",
            f"{bq[2]:+.3f}",
            f"{bq[3]:+.3f}",
            f"{bq[4]:+.3f}",
        ],
        [
            "solve_rate",
            f"{task['solve_rate'].quantile(0.05):.3f}",
            f"{task['solve_rate'].quantile(0.25):.3f}",
            f"{task['solve_rate'].quantile(0.50):.3f}",
            f"{task['solve_rate'].quantile(0.75):.3f}",
            f"{task['solve_rate'].quantile(0.95):.3f}",
        ],
    ]
    lines.append(md_table(["series", "p05", "p25", "median", "p75", "p95"], rows))
    lines.append("")

    lines.append("## 3. Bucket disagreement: b_bucket vs pass-rate bucket")
    lines.append("")
    disagree = report.disagreement
    lines.append(
        f"On the fitted set, `irt_bucket` matches the pass-rate `difficulty_bucket` for "
        f"**{pct(disagree.agreement)}** of instances. The b cuts are quantile-matched to the "
        "pass-rate bucket shares (so bucket sizes approximately match — b is piecewise-constant, "
        "tasks with identical outcome patterns share an identical estimate, so a cut inside a "
        "tie block moves the whole block): "
        + ", ".join(f"{name} <= {cut:+.3f}" for name, cut in zip(EASIEST_ORDER[:-1], disagree.cuts))
        + f" (ascending b = {EASIEST_ORDER[0]} to {EASIEST_ORDER[-1]}). `delta` = pass-rank "
        "minus b-rank on the hardness scale (positive = the pass-rate bucket looks easier)."
    )
    lines.append("")
    confusion = disagree.confusion
    rows = []
    for name in BUCKET_VOCAB:
        row = confusion.loc[name]
        rows.append([name, *[f"{int(v):,}" for v in row.to_numpy()], f"{int(row.sum()):,}"])
    lines.append(md_table(["pass \\ b", *BUCKET_VOCAB, "pass total"], rows))
    lines.append("")
    rows = [
        [
            group["group"],
            f"{int(group['instances']):,}",
            pct(group["share"]),
            f"{group['mean_solve_rate']:.3f}",
            f"{group['mean_mix_theta']:+.3f}",
            pct(group["mean_weak_share"]),
            pct(group["mean_strong_share"]),
            f"{group['median_n_combos']:.0f}",
        ]
        for _, group in disagree.groups.iterrows()
    ]
    lines.append(
        md_table(
            [
                "group",
                "instances",
                "share",
                "mean solve_rate",
                "mean mix theta",
                "weak-combo share",
                "strong-combo share",
                "median combos",
            ],
            rows,
        )
    )
    lines.append("")
    easier_group = disagree.groups.iloc[0]
    harder_group = disagree.groups.iloc[1]
    all_group = disagree.groups.iloc[-1]
    lines.append(
        f"Combo mix moves the disagreement in the expected direction: Spearman(delta, "
        f"per-instance mean combo theta) {disagree.rho_delta_mix_theta:+.4f}. Instances whose "
        "pass-rate bucket reads easier than the b-bucket were disproportionately attempted by "
        f"strong combos (mean mix theta {easier_group['mean_mix_theta']:+.3f}, strong-combo "
        f"share {pct(easier_group['mean_strong_share'])} vs "
        f"{pct(all_group['mean_strong_share'])} overall), so their successes overstate ease. "
        "The mirror group (pass bucket reads harder) is the opposite but milder: a small "
        f"weak-combo tilt ({pct(harder_group['mean_weak_share'])} weak-combo share vs "
        f"{pct(all_group['mean_weak_share'])} overall) on tasks with more combos attempted "
        f"(median {harder_group['median_n_combos']:.0f} vs "
        f"{all_group['median_n_combos']:.0f}), where a few weak-combo failures drag the rate "
        "below what the ability-adjusted difficulty implies."
    )
    lines.append("")
    lines.append("Largest disagreement cells (pass bucket → b bucket):")
    lines.append("")
    rows = [
        [
            f"{cell['pass_bucket']} → {cell['b_bucket']}",
            f"{int(cell['instances']):,}",
            f"{cell['mean_solve_rate']:.3f}",
            f"{cell['mean_b_2pl']:+.3f}",
            f"{cell['mean_mix_theta']:+.3f}",
            pct(cell["mean_weak_share"]),
        ]
        for _, cell in disagree.cells.head(8).iterrows()
    ]
    lines.append(
        md_table(
            [
                "cell",
                "instances",
                "mean solve_rate",
                "mean b_2pl",
                "mean mix theta",
                "weak-combo share",
            ],
            rows,
        )
    )
    lines.append("")

    lines.append("## 4. Discrimination a_2pl")
    lines.append("")
    disc = report.discrimination
    lines.append(
        "a_2pl is the slope: high-a tasks separate strong from weak combos sharply (outcomes "
        "flip over a narrow ability band), low-a tasks are near coin flips for everyone. For "
        "tasks with all-fail or all-pass outcomes the likelihood keeps a(a-b) flat, so a is "
        "only pinned by its prior — those estimates are weak by construction."
    )
    lines.append("")
    rows = [
        [
            str(row["quantile"]),
            f"{row['a_2pl']:.3f}",
        ]
        for _, row in disc.quantiles.iterrows()
    ]
    lines.append(md_table(["quantile", "a_2pl"], rows))
    lines.append("")
    rows = [
        [
            str(row["bucket"]),
            f"{int(row['instances']):,}",
            f"{row['mean_a']:.3f}",
            f"{row['median_a']:.3f}",
            pct(row["bucket_top_share"]) if np.isfinite(row["bucket_top_share"]) else "n/a",
            pct(row["top_decile_contribution"])
            if np.isfinite(row["top_decile_contribution"])
            else "n/a",
        ]
        for _, row in disc.by_bucket.iterrows()
    ]
    lines.append(
        md_table(
            [
                "pass bucket",
                "instances",
                "mean a",
                "median a",
                "share of bucket in high-a decile",
                "share of high-a decile from bucket",
            ],
            rows,
        )
    )
    lines.append("")
    rows = []
    for _, row in disc.profile.iterrows():
        fmt = ".3f" if row["metric"] == "solve_rate" else ",.0f"
        rows.append(
            [
                str(row["metric"]),
                f"{row['top_decile_a']:{fmt}}",
                f"{row['bottom_decile_a']:{fmt}}",
                f"{row['top_decile_a_mixed']:{fmt}}",
                f"{row['bottom_decile_a_mixed']:{fmt}}",
            ]
        )
    lines.append(
        md_table(
            [
                "median metric",
                "high-a decile",
                "low-a decile",
                "high-a decile (mixed only)",
                "low-a decile (mixed only)",
            ],
            rows,
        )
    )
    lines.append("")
    rows = [
        [
            str(row["metric"]),
            f4(row["spearman_a"]),
        ]
        for _, row in disc.corr.iterrows()
    ]
    lines.append(md_table(["Spearman(a_2pl, .)", "rho"], rows))
    lines.append("")
    lang = disc.language.sort_values("fitted_share", ascending=False)
    rows = [
        [
            str(row["language"]),
            pct(row["fitted_share"]),
            pct(row["top_a_share"]),
            "n/a" if not np.isfinite(row["mixed_top_share"]) else pct(row["mixed_top_share"]),
        ]
        for _, row in lang.iterrows()
    ]
    lines.append(
        md_table(
            ["language", "fitted share", "share of high-a decile", "share of mixed high-a decile"],
            rows,
        )
    )
    lines.append("")
    cat = disc.category.sort_values("fitted_share", ascending=False)
    rows = [
        [
            str(row["category"]),
            pct(row["fitted_share"]),
            pct(row["top_a_share"]),
            "n/a" if not np.isfinite(row["mixed_top_share"]) else pct(row["mixed_top_share"]),
        ]
        for _, row in cat.iterrows()
    ]
    lines.append(
        md_table(
            ["category", "fitted share", "share of high-a decile", "share of mixed high-a decile"],
            rows,
        )
    )
    lines.append("")
    bucket_lookup = disc.by_bucket.set_index("bucket")
    profile_lookup = disc.profile.set_index("metric")
    lang_lookup = disc.language.set_index("language")
    py_mixed = float(lang_lookup.loc["python", "mixed_top_share"])
    py_fitted = float(lang_lookup.loc["python", "fitted_share"])
    degenerate_contribution = float(
        bucket_lookup.loc["all_pass", "top_decile_contribution"]
        + bucket_lookup.loc["all_fail", "top_decile_contribution"]
    )
    degenerate_share = float(
        bucket_lookup.loc["all_pass", "fitted_share"]
        + bucket_lookup.loc["all_fail", "fitted_share"]
    )
    lines.append(
        "Reading the tables: the high-a decile is a well-sampled, extreme-outcome phenomenon, "
        "not a language or category story. Degenerate tasks supply "
        f"{pct(degenerate_contribution)} of the top decile against their "
        f"{pct(degenerate_share)} fitted share, with all_pass over-represented "
        f"({pct(bucket_lookup.loc['all_pass', 'top_decile_contribution'])} of the top decile "
        f"from its {pct(bucket_lookup.loc['all_pass', 'fitted_share'])} share) and hard tasks "
        f"all but absent ({pct(bucket_lookup.loc['hard', 'top_decile_contribution'])}); a is "
        "largest where outcomes are most extreme and most observed (median n_labeled "
        f"{profile_lookup.loc['n_labeled', 'top_decile_a']:.0f} vs "
        f"{profile_lookup.loc['n_labeled', 'bottom_decile_a']:.0f} in the low-a decile). Among "
        "mixed tasks, high-a instances are harder (median solve_rate "
        f"{profile_lookup.loc['solve_rate', 'top_decile_a_mixed']:.3f} vs "
        f"{profile_lookup.loc['solve_rate', 'bottom_decile_a_mixed']:.3f}) but structurally "
        f"ordinary: python is {pct(py_mixed)} of the mixed high-a decile vs {pct(py_fitted)} "
        "fitted, and gold patch size is flat "
        f"({profile_lookup.loc['gold_patch_lines', 'top_decile_a_mixed']:.0f} vs "
        f"{profile_lookup.loc['gold_patch_lines', 'bottom_decile_a_mixed']:.0f} median lines)."
    )
    lines.append("")

    held = report.heldout
    lines.append("## 5. Held-out log-likelihood: 1PL vs 2PL vs pass-rate baseline")
    lines.append("")
    lines.append(
        f"20% random split of rollouts (seed {SEED}): {held.n_train:,} train rollouts / "
        f"{held.n_fit_tasks:,} fitted tasks; {held.n_test:,} test rollouts of fitted tasks "
        "scored against the train-fitted models (and their task's train solve rate, "
        "Jeffreys-smoothed as (resolved + 0.5) / (n + 1)). Smoothing breaks the ties among "
        "0/1-rate tasks and keeps degenerate rates out of the log-likelihood clip; the baseline "
        "still sees no combo identity, which is where the IRT models gain."
    )
    lines.append("")
    rows = [
        [
            "1PL (a = 1)",
            f"{held.mean_ll_1pl:+.4f}",
            f"{held.mean_ll_1pl * held.n_test:,.0f}",
            "-" if np.isnan(held.auc_1pl) else f"{held.auc_1pl:.4f}",
        ],
        [
            "2PL (free a)",
            f"{held.mean_ll_2pl:+.4f}",
            f"{held.mean_ll_2pl * held.n_test:,.0f}",
            "-" if np.isnan(held.auc_2pl) else f"{held.auc_2pl:.4f}",
        ],
        [
            "pass-rate baseline (task train rate, smoothed)",
            f"{held.mean_ll_rate:+.4f}",
            f"{held.mean_ll_rate * held.n_test:,.0f}",
            "-" if np.isnan(held.auc_rate) else f"{held.auc_rate:.4f}",
        ],
    ]
    lines.append(md_table(["predictor", "mean LL / rollout", "total LL", "AUC"], rows))
    lines.append("")
    auc_2pl = "-" if np.isnan(held.auc_2pl) else f"{held.auc_2pl:.4f}"
    auc_rate = "-" if np.isnan(held.auc_rate) else f"{held.auc_rate:.4f}"
    lines.append(
        f"The aggregate LL favors the smoothed pass rate, but that comes entirely from the "
        f"degenerate tasks it predicts sharply: on the {held.n_degenerate:,} degenerate test "
        f"rollouts (all_fail/all_pass) it scores "
        f"{held.mean_ll_rate_degenerate:+.4f} vs {held.mean_ll_2pl_degenerate:+.4f} (2PL) and "
        f"{held.mean_ll_1pl_degenerate:+.4f} (1PL), while on the {held.n_mixed:,} mixed "
        f"rollouts (hard/mid/easy — the ones manifest eligibility selects) both IRT models win "
        f"decisively: {held.mean_ll_2pl_mixed:+.4f} (2PL) and {held.mean_ll_1pl_mixed:+.4f} "
        f"(1PL) vs {held.mean_ll_rate_mixed:+.4f}. The models also rank better overall "
        f"(AUC {auc_2pl} vs {auc_rate}) and the 2PL beats the 1PL, so both the ability scale "
        "and the per-task slope carry information the rate misses."
    )
    lines.append("")
    lines.append(
        md_table(
            ["test rollouts", "n", "1PL LL", "2PL LL", "pass-rate LL"],
            [
                [
                    "degenerate (all_fail/all_pass)",
                    f"{held.n_degenerate:,}",
                    f"{held.mean_ll_1pl_degenerate:+.4f}",
                    f"{held.mean_ll_2pl_degenerate:+.4f}",
                    f"{held.mean_ll_rate_degenerate:+.4f}",
                ],
                [
                    "mixed (hard/mid/easy)",
                    f"{held.n_mixed:,}",
                    f"{held.mean_ll_1pl_mixed:+.4f}",
                    f"{held.mean_ll_2pl_mixed:+.4f}",
                    f"{held.mean_ll_rate_mixed:+.4f}",
                ],
            ],
        )
    )
    lines.append("")

    lines.append("## 6. Recommendation for manifest bucketing")
    lines.append("")
    lines.append(
        "**Use `irt_bucket` for the hard/mid/easy eligibility of manifests; keep the pass-rate "
        "`difficulty_bucket` only as the fallback for instances with fewer than 3 labeled "
        "rollouts (which stay `unknown`).** In order of strength: (1) on the mixed held-out "
        f"rollouts — exactly the tasks manifest eligibility selects — the IRT models beat the "
        f"task's own rate by {held.mean_ll_2pl_mixed - held.mean_ll_rate_mixed:+.4f} (2PL) and "
        f"{held.mean_ll_1pl_mixed - held.mean_ll_rate_mixed:+.4f} (1PL) nats/rollout, the 2PL "
        f"beats the 1PL, and the models rank better overall (AUC {auc_2pl} vs {auc_rate}); the "
        "smoothed rate's aggregate LL edge comes only from degenerate tasks that eligibility "
        f"drops; (2) where the buckets disagree "
        f"({pct(1.0 - disagree.agreement)} of fitted instances), the pass-rate side is the "
        "biased one — the group whose pass bucket reads easier consists of "
        f"{pct(easier_group['mean_strong_share'])} strong-combo attempts "
        f"(vs {pct(all_group['mean_strong_share'])} overall), exactly the composition that "
        "inflates an observed rate; (3) `irt_bucket` keeps the pass-rate bucket sizes "
        "(quantile-matched up to b tie blocks, so budget allocation and eval balancing carry "
        "over largely unchanged), and "
        f"the cuts ({', '.join(f'{cut:+.3f}' for cut in disagree.cuts)}) are reusable across "
        "refits while the corpus mix is stable. The costs to accept: the bucket is defined "
        "only for tasks with >= 3 labeled rollouts, it comes from a global fit (rebuild order "
        "matters), and it moves a task's label whenever its attempted-combo mix changes even "
        "if the task text does not."
    )
    lines.append("")

    lines.append("## Notes")
    lines.append("")
    lines.append(
        f"- Fitting detail: full-batch Adam on NLL + priors, {report.steps} steps at lr "
        f"{report.lr}; objective {report.fit_2pl.objective:,.1f} (2PL) vs "
        f"{report.fit_1pl.objective:,.1f} (1PL); loss history: "
        + ", ".join(f"{step}:{loss:,.0f}" for step, loss in report.fit_2pl.loss_history)
        + "."
    )
    lines.append(
        "- theta and b are jointly shift-invariant: the likelihood alone cannot fix the "
        "ability scale's location, so the N(0, 1) priors anchor it; only differences within "
        "the theta set and within the b set are meaningful."
    )
    tie_counts = report.task_irt["b_2pl"].round(12).value_counts()
    lines.append(
        f"- b_2pl is piecewise-constant: {tie_counts.size:,} distinct values over "
        f"{len(report.task_irt):,} instances (largest tie block {int(tie_counts.max()):,}). "
        "Tasks with identical outcome-pattern × combo-mix signatures have the same likelihood "
        "and the same penalized MLE, so the quantile cuts reproduce the pass-rate bucket sizes "
        "only up to the tie block that straddles each cut."
    )
    lines.append(
        "- a_2pl for all-fail/all-pass tasks is weakly identified (the likelihood is flat "
        "along the a(a-b) ridge); the prior on log a is what pins it. Discrimination "
        "comparisons should be read on the mixed buckets as well as overall."
    )
    lines.append(
        "- `n_labeled` counts rollouts with `resolved in (0, 1)`; `solve_rate` is "
        "n_resolved / n_labeled over the instance's labeled rollouts."
    )
    lines.append(
        "- `irt_bucket` is the b_2pl quantile bucket (`unknown` below 3 labeled rollouts). "
        "Rebuilding `outputs/task_difficulty.parquet` with `scripts/task_difficulty.py` "
        "drops the column — rerun `scripts/fit_irt.py` after it."
    )
    lines.append(
        "- Reproduce: `uv run python scripts/fit_irt.py` (about a minute; `--steps`, "
        "`--lr`, `--no-bucket` available)."
    )
    lines.append("")
    return "\n".join(lines)


def rel(path: Path) -> str:
    try:
        return str(path.relative_to(ROOT))
    except ValueError:
        return str(path)


def main_irt() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--proxy", type=Path, default=PROXY_PARQUET)
    parser.add_argument("--difficulty", type=Path, default=DIFFICULTY_PARQUET)
    parser.add_argument("--task-out", type=Path, default=OUT_TASK_PARQUET)
    parser.add_argument("--combo-out", type=Path, default=OUT_COMBO_PARQUET)
    parser.add_argument("--summary", type=Path, default=OUT_MD)
    parser.add_argument("--steps", type=int, default=STEPS, help="Adam steps per model")
    parser.add_argument("--lr", type=float, default=LR, help="Adam learning rate")
    parser.add_argument("--no-bucket", action="store_true", help="Do not add irt_bucket")
    args = parser.parse_args()

    args.proxy = args.proxy if args.proxy.is_absolute() else ROOT / args.proxy
    args.difficulty = args.difficulty if args.difficulty.is_absolute() else ROOT / args.difficulty
    args.task_out = args.task_out if args.task_out.is_absolute() else ROOT / args.task_out
    args.combo_out = args.combo_out if args.combo_out.is_absolute() else ROOT / args.combo_out
    args.summary = args.summary if args.summary.is_absolute() else ROOT / args.summary

    labeled = load_labeled_rollouts(args.proxy)
    data = build_irt_data(labeled)
    inst_all = pd.read_parquet(args.difficulty)
    console.print(
        f"labeled rollouts {len(labeled):,}; fitted instances {len(data.tasks):,} "
        f"(>= {MIN_LABELED} labeled) over {len(data.combos)} combos; "
        f"{data.n_rollouts:,} rollouts in the fit"
    )

    fit_2pl = fit_irt_data(data, two_pl=True, steps=args.steps, lr=args.lr)
    fit_1pl = fit_irt_data(data, two_pl=False, steps=args.steps, lr=args.lr)
    console.print(
        f"2PL objective {fit_2pl.objective:,.1f} (nll {fit_2pl.nll:,.1f}) | "
        f"1PL objective {fit_1pl.objective:,.1f} (nll {fit_1pl.nll:,.1f})"
    )

    task_irt = task_irt_frame(data, fit_1pl, fit_2pl)
    combo = combo_ability_frame(data, fit_1pl, fit_2pl)
    args.task_out.parent.mkdir(parents=True, exist_ok=True)
    task_irt.to_parquet(args.task_out, index=False)
    combo.to_parquet(args.combo_out, index=False)
    console.print(f"Wrote {len(task_irt):,} instances → {rel(args.task_out)}")
    console.print(f"Wrote {len(combo)} combos → {rel(args.combo_out)}")

    held = heldout_comparison(data, inst_all.set_index("instance_id")["difficulty_bucket"])
    disagree = disagreement_report(data, task_irt, inst_all, fit_2pl)
    disc = discrimination_report(data, task_irt, inst_all)
    console.print(
        f"Held-out (20%): mean LL 1PL {held.mean_ll_1pl:+.4f} | 2PL {held.mean_ll_2pl:+.4f} | "
        f"pass-rate {held.mean_ll_rate:+.4f}"
    )
    console.print(
        f"Buckets: agreement {pct(disagree.agreement)}, "
        f"Spearman(delta, mix theta) {disagree.rho_delta_mix_theta:+.4f}"
    )

    report = IrtReport(
        data=data,
        fit_2pl=fit_2pl,
        fit_1pl=fit_1pl,
        task_irt=task_irt,
        combo=combo,
        heldout=held,
        disagreement=disagree,
        discrimination=disc,
        n_instances_total=len(inst_all),
        steps=args.steps,
        lr=args.lr,
    )
    md = build_summary(report, rel(args.task_out), rel(args.combo_out), rel(args.difficulty))
    args.summary.parent.mkdir(parents=True, exist_ok=True)
    args.summary.write_text(md)
    console.print(f"Wrote {rel(args.summary)}")

    if not args.no_bucket:
        buckets = irt_bucket_frame(task_irt, disagree.cuts)
        path, n_unknown = add_irt_bucket(args.difficulty, buckets)
        console.print(f"Added irt_bucket to {rel(path)} ({n_unknown:,} instances {UNKNOWN})")


if __name__ == "__main__":
    main_irt()

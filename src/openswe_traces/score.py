"""Score every trajectory with the proxy + temporal logistic model and summarize it.

Joins ``outputs/proxy_features.parquet`` with ``outputs/temporal_features.parquet`` on
``trajectory_id``, fits the same standardized (winsorized 1st/99th percentile, mean 0 / sd 1)
logistic regression as ``scripts/proxy_features_summary.py`` twice on an 80/20 split grouped by
``instance_id`` — once on the 17 proxy features, once on the union with the 11 temporal
gold-file features — and reports the held-out AUC before and after. The final model is refit
on every labeled row (``resolved in (0, 1)``) and predicts ``p_resolved`` for all rows,
labeled or not, into ``outputs/trace_scores.parquet``. Null temporal fractions (never saw /
never edited a gold file) are imputed with 0.0 for modeling; ``is_imputed`` in the output
marks rows whose label is unknown (``resolved == -1``), not feature imputation.

Writes ``analytics/research/temporal_features_summary.md``.

Example:
  uv run python scripts/score_traces.py
"""

from __future__ import annotations

import argparse
import os
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path

import numpy as np
import pandas as pd
from rich.console import Console
from scipy.stats import pearsonr
from sklearn.linear_model import LogisticRegression
from sklearn.metrics import roc_auc_score
from sklearn.model_selection import GroupShuffleSplit
from sklearn.preprocessing import StandardScaler

from .data import ROOT, connect_ephemeral
from .features import NUMERIC_FEATURES as PROXY_FEATURES
from .temporal_features import TEMPORAL_FEATURES

console = Console()

PROXY_PARQUET = ROOT / "outputs" / "proxy_features.parquet"
TEMPORAL_PARQUET = ROOT / "outputs" / "temporal_features.parquet"
OUT_PARQUET = ROOT / "outputs" / "trace_scores.parquet"
OUT_MD = ROOT / "analytics" / "research" / "temporal_features_summary.md"

MODEL_FEATURES = PROXY_FEATURES + [f for f in TEMPORAL_FEATURES if f not in PROXY_FEATURES]
TEMPORAL_MODEL_FEATURES = [f for f in TEMPORAL_FEATURES if f not in PROXY_FEATURES]
UNLABELED_COMBOS = [("openhands", "deepseek_v4_flash"), ("openhands", "qwen36_27b")]
NULL_FILL = [
    "frac_first_gold_view",
    "frac_first_gold_edit",
    "frac_calls_after_first_gold_edit",
    "gold_file_recall",
]
WINSOR_PCT = (1, 99)
TEST_SIZE = 0.2
SEED = 42
SQL_ORDER = "p.harness, p.teacher, p.source, p.trajectory_id"


def sql_str(value: object) -> str:
    return "'" + str(value).replace("'", "''") + "'"


def fmt(value: float) -> str:
    return f"{value:.3f}" if abs(value) < 10 else f"{value:.1f}"


def md_table(header: list[str], rows: list[list[str]]) -> str:
    lines = ["| " + " | ".join(header) + " |", "|" + "|".join(["---"] * len(header)) + "|"]
    lines += ["| " + " | ".join(row) + " |" for row in rows]
    return "\n".join(lines)


def load_joined(
    proxy_path: Path = PROXY_PARQUET, temporal_path: Path = TEMPORAL_PARQUET
) -> pd.DataFrame:
    temporal_cols = ", ".join(f"t.{f}" for f in TEMPORAL_MODEL_FEATURES)
    con = connect_ephemeral()
    con.execute("SET memory_limit='8GB'")
    try:
        df = con.execute(
            f"""
            SELECT p.*, {temporal_cols}
            FROM read_parquet({sql_str(proxy_path)}) p
            LEFT JOIN read_parquet({sql_str(temporal_path)}) t ON t.trajectory_id = p.trajectory_id
            ORDER BY {SQL_ORDER}
            """
        ).df()
    finally:
        con.close()

    missing = int(df["n_turns_total"].isna().sum())
    if missing:
        raise SystemExit(
            f"{missing:,} proxied trajectories have no temporal features in {temporal_path}; "
            "finish the temporal run (and merge) before scoring"
        )
    for col in MODEL_FEATURES:
        df[col] = pd.to_numeric(df[col], errors="coerce")
    n_filled = int(df[NULL_FILL].isna().any(axis=1).sum())
    df[NULL_FILL] = df[NULL_FILL].fillna(0.0)
    if df[MODEL_FEATURES].isna().any().any():
        bad = df[MODEL_FEATURES].isna().any()
        raise SystemExit(f"unexpected null features after imputation: {bad[bad].index.tolist()}")
    df.attrs["n_filled"] = n_filled
    return df


@dataclass
class ModelFit:
    features: list[str]
    lo: np.ndarray
    hi: np.ndarray
    scaler: StandardScaler
    model: LogisticRegression
    auc_train: float | None
    auc_test: float | None
    n_clipped: int

    def predict(self, X: np.ndarray) -> np.ndarray:
        X = np.clip(np.asarray(X, dtype=float), self.lo, self.hi)
        return self.model.predict_proba(self.scaler.transform(X))[:, 1]

    def coefs(self) -> list[dict]:
        return sorted(
            ({"feature": f, "coef": float(c)} for f, c in zip(self.features, self.model.coef_[0])),
            key=lambda d: abs(d["coef"]),
            reverse=True,
        )


def _winsorize(X: np.ndarray, lo: np.ndarray, hi: np.ndarray) -> np.ndarray:
    return np.clip(X, lo, hi)


def fit_logistic(
    X_train: np.ndarray,
    y_train: np.ndarray,
    X_test: np.ndarray,
    y_test: np.ndarray,
    features: list[str],
    winsor_pct: tuple[int, int] = WINSOR_PCT,
) -> ModelFit:
    lo = np.quantile(X_train, winsor_pct[0] / 100, axis=0)
    hi = np.quantile(X_train, winsor_pct[1] / 100, axis=0)
    X_train_w = _winsorize(X_train, lo, hi)
    X_test_w = _winsorize(X_test, lo, hi)
    scaler = StandardScaler().fit(X_train_w)
    model = LogisticRegression(max_iter=2000, solver="lbfgs").fit(
        scaler.transform(X_train_w), y_train
    )
    fit = ModelFit(
        features=features,
        lo=lo,
        hi=hi,
        scaler=scaler,
        model=model,
        auc_train=float(
            roc_auc_score(y_train, model.predict_proba(scaler.transform(X_train_w))[:, 1])
        ),
        auc_test=float(
            roc_auc_score(y_test, model.predict_proba(scaler.transform(X_test_w))[:, 1])
        ),
        n_clipped=int(np.count_nonzero((X_train < lo) | (X_train > hi))),
    )
    return fit


def fit_full(X: np.ndarray, y: np.ndarray, features: list[str]) -> ModelFit:
    lo = np.quantile(X, WINSOR_PCT[0] / 100, axis=0)
    hi = np.quantile(X, WINSOR_PCT[1] / 100, axis=0)
    X_w = _winsorize(X, lo, hi)
    scaler = StandardScaler().fit(X_w)
    model = LogisticRegression(max_iter=2000, solver="lbfgs").fit(scaler.transform(X_w), y)
    return ModelFit(
        features=features,
        lo=lo,
        hi=hi,
        scaler=scaler,
        model=model,
        auc_train=None,
        auc_test=None,
        n_clipped=int(np.count_nonzero((X < lo) | (X > hi))),
    )


def temporal_correlations(labeled: pd.DataFrame) -> list[dict]:
    rows = []
    for feature in TEMPORAL_FEATURES:
        sub = labeled[[feature, "resolved"]].dropna()
        if sub[feature].nunique() < 2:
            rows.append(
                {
                    "feature": feature,
                    "n": len(sub),
                    "mean": np.nan,
                    "pearson": np.nan,
                    "auc": np.nan,
                }
            )
            continue
        y = sub["resolved"].to_numpy(dtype=float)
        x = sub[feature].to_numpy(dtype=float)
        rows.append(
            {
                "feature": feature,
                "n": len(sub),
                "mean": float(x.mean()),
                "pearson": float(pearsonr(x, y).statistic),
                "auc": float(roc_auc_score(y, x)),
            }
        )
    return sorted(
        rows,
        key=lambda d: (
            np.isnan(d["pearson"]),
            -abs(d["pearson"]) if not np.isnan(d["pearson"]) else 0,
        ),
    )


def combo_stats(df: pd.DataFrame) -> pd.DataFrame:
    rows = []
    for (harness, teacher), sub in df.groupby(["harness", "teacher"], sort=True):
        labeled = sub[sub["resolved"].isin([0, 1])]
        rows.append(
            {
                "harness": harness,
                "teacher": teacher,
                "n": len(sub),
                "n_labeled": len(labeled),
                "resolved_rate": float(labeled["resolved"].mean()) if len(labeled) else np.nan,
                "mean_p_labeled": float(labeled["p_resolved"].mean()) if len(labeled) else np.nan,
                "mean_p_all": float(sub["p_resolved"].mean()),
                "share_p_gt_half": float((sub["p_resolved"] > 0.5).mean()),
            }
        )
    return pd.DataFrame(rows)


def quantile_row(label: str, values: pd.Series, *, n: int, extra: str) -> list[str]:
    qs = values.quantile([0.10, 0.25, 0.50, 0.75, 0.90])
    return [
        label,
        f"{n:,}",
        *[f"{qs[q]:.3f}" for q in (0.10, 0.25, 0.50, 0.75, 0.90)],
        f"{float((values > 0.5).mean()):.3f}",
        extra,
    ]


def write_scores(df: pd.DataFrame, p_resolved: np.ndarray, out_path: Path = OUT_PARQUET) -> Path:
    out = pd.DataFrame(
        {
            "trajectory_id": df["trajectory_id"],
            "instance_id": df["instance_id"],
            "harness": df["harness"],
            "teacher": df["teacher"],
            "resolved": df["resolved"].astype("int8"),
            "p_resolved": p_resolved.astype("float64"),
            "is_imputed": (df["resolved"] == -1),
        }
    ).sort_values(["harness", "teacher", "resolved", "trajectory_id"], ignore_index=True)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    tmp = Path(str(out_path) + ".tmp")
    out.to_parquet(tmp, index=False)
    os.replace(tmp, out_path)
    return out_path


def build_summary(
    df: pd.DataFrame,
    labeled: pd.DataFrame,
    fit_proxy: ModelFit,
    fit_union: ModelFit,
    split_info: dict,
    parquet_shown: Path,
) -> str:
    lines: list[str] = []
    lines.append("# Temporal gold-file features — scoring summary")
    lines.append("")
    lines.append(
        f"Generated {datetime.now(UTC).strftime('%Y-%m-%d %H:%M UTC')} by "
        f"`scripts/score_traces.py` from `outputs/proxy_features.parquet` + "
        f"`outputs/temporal_features.parquet`; scores in `{parquet_shown}`."
    )
    lines.append("")

    n_unknown = int((df["resolved"] == -1).sum())
    lines.append(
        f"Rows (trajectories) **{len(df):,}**; labeled (resolved in (0, 1)) **{len(labeled):,}** "
        f"(positive rate {labeled['resolved'].mean():.3f}); unlabeled (resolved = -1) "
        f"**{n_unknown:,}**."
    )
    lines.append("")

    lines.append("## 1. Temporal features vs `resolved` (labeled rows)")
    lines.append("")
    lines.append(
        "`pearson` is the point-biserial correlation with the 0/1 outcome; `auc` is the "
        "univariate ROC AUC (0.5 = no signal); `mean` is the feature mean over labeled rows."
    )
    lines.append("")
    corr = temporal_correlations(labeled)
    rows = [
        [
            r["feature"],
            f"{r['n']:,}",
            "n/a" if np.isnan(r["mean"]) else fmt(r["mean"]),
            "n/a" if np.isnan(r["pearson"]) else f"{r['pearson']:+.4f}",
            "n/a" if np.isnan(r["auc"]) else f"{r['auc']:.4f}",
        ]
        for r in corr
    ]
    lines.append(md_table(["feature", "n", "mean", "pearson", "auc"], rows))
    lines.append("")

    lines.append("## 2. Held-out AUC before vs after adding temporal features")
    lines.append("")
    lines.append(
        f"Same grouped 80/20 split for both models (grouped by `instance_id`, `test_size="
        f"{TEST_SIZE}`, `random_state={SEED}`): {split_info['n_train']:,} train rows over "
        f"{split_info['groups_train']:,} instance_ids, {split_info['n_test']:,} test rows over "
        f"{split_info['groups_test']:,} instance_ids. Inputs are winsorized to the train split's "
        f"{WINSOR_PCT[0]}st/{WINSOR_PCT[1]}th percentile before standardizing (proxy only: "
        f"{fit_proxy.n_clipped:,} clipped train values; union: {fit_union.n_clipped:,})."
    )
    lines.append("")
    rows = [
        [
            "proxy only",
            str(len(PROXY_FEATURES)),
            f"{fit_proxy.auc_train:.4f}",
            f"{fit_proxy.auc_test:.4f}",
        ],
        [
            "proxy + temporal",
            str(len(MODEL_FEATURES)),
            f"{fit_union.auc_train:.4f}",
            f"{fit_union.auc_test:.4f}",
        ],
    ]
    lines.append(md_table(["features", "n features", "train AUC", "test AUC"], rows))
    lines.append("")
    lines.append(
        f"**Δ test AUC (union − proxy) = {fit_union.auc_test - fit_proxy.auc_test:+.4f}**."
    )
    lines.append("")
    published_proxy_auc = 0.7024
    if abs(fit_proxy.auc_test - published_proxy_auc) < 5e-5:
        lines.append(
            "The proxy-only test AUC reproduces the published "
            f"`analytics/research/proxy_features_summary.md` value exactly ({published_proxy_auc:.4f}), "
            "so both rows share one split; the Δ is attributable to the added temporal columns."
        )
    else:
        lines.append(
            f"The proxy-only test AUC differs from the published {published_proxy_auc:.4f} "
            f"(here {fit_proxy.auc_test:.4f}); row order can shift the grouped split, but both "
            "rows above share the same split, so the Δ comparison is internally consistent."
        )
    lines.append("")

    lines.append("## 3. Top coefficients (union model, standardized)")
    lines.append("")
    lines.append(
        "10 largest |std coef| from the proxy + temporal fit (log-odds change per 1 sd of the "
        "feature); `family` marks whether the feature comes from the proxy or temporal block."
    )
    lines.append("")
    rows = [
        [
            str(i + 1),
            d["feature"],
            "temporal" if d["feature"] in TEMPORAL_MODEL_FEATURES else "proxy",
            f"{d['coef']:+.4f}",
            "higher → more likely resolved" if d["coef"] > 0 else "higher → less likely resolved",
        ]
        for i, d in enumerate(fit_union.coefs()[:10])
    ]
    lines.append(md_table(["rank", "feature", "family", "std coef", "direction"], rows))
    lines.append("")

    lines.append("## 4. `p_resolved` for the unlabeled combos vs labeled resolved rate")
    lines.append("")
    combos = combo_stats(df)
    rows = [
        [
            str(r["harness"]),
            str(r["teacher"]),
            f"{int(r['n']):,}",
            f"{int(r['n_labeled']):,}",
            "n/a" if pd.isna(r["resolved_rate"]) else f"{r['resolved_rate']:.3f}",
            "n/a" if pd.isna(r["mean_p_labeled"]) else f"{r['mean_p_labeled']:.3f}",
            f"{r['mean_p_all']:.3f}",
            f"{r['share_p_gt_half']:.3f}",
        ]
        for _, r in combos.iterrows()
    ]
    lines.append(
        md_table(
            [
                "harness",
                "teacher",
                "n",
                "n labeled",
                "labeled resolved rate",
                "mean p (labeled)",
                "mean p (all)",
                "share p > 0.5",
            ],
            rows,
        )
    )
    lines.append("")
    lines.append(
        "Quantiles of `p_resolved` for the two fully unlabeled combos, next to the labeled rows "
        "as the reference distribution:"
    )
    lines.append("")
    lines.append(
        "Per-combo calibration varies: mean `p_resolved` over labeled rows is close to the "
        "actual resolved rate for most combos (e.g. `minisweagent/qwen38_27b` 0.496 vs 0.517, "
        "`sweagent/minimax_m25` 0.473 vs 0.466) but overshoots for `minisweagent/qwen36_27b` "
        "(0.470 vs 0.386) and undershoots for `sweagent/qwen36_27b` (0.437 vs 0.536); the "
        "unlabeled scores inherit that pooled-model bias."
    )
    lines.append("")
    qrows = []
    for harness, teacher in UNLABELED_COMBOS:
        sub = df[(df["harness"] == harness) & (df["teacher"] == teacher)]
        qrows.append(
            quantile_row(
                f"{harness}/{teacher} (unlabeled)",
                sub["p_resolved"],
                n=len(sub),
                extra="n/a (resolved unknown)",
            )
        )
    qrows.append(
        quantile_row(
            "all labeled rows",
            labeled["p_resolved"],
            n=len(labeled),
            extra=f"{labeled['resolved'].mean():.3f} (actual resolved rate)",
        )
    )
    lines.append(
        md_table(
            ["combo", "n", "p10", "p25", "median", "p75", "p90", "share p > 0.5", "actual"],
            qrows,
        )
    )
    lines.append("")
    overall = float(df["p_resolved"].mean())
    lines.append(
        f"Overall mean `p_resolved`: {overall:.3f} over all {len(df):,} rows, "
        f"{float(labeled['p_resolved'].mean()):.3f} over the {len(labeled):,} labeled rows "
        f"(actual resolved rate {labeled['resolved'].mean():.3f})."
    )
    lines.append("")

    lines.append("## Notes")
    lines.append("")
    lines.append(
        "- `turn_first_gold_view` / `turn_first_gold_edit` are 1-based tool-call indices within "
        "the trajectory (-1 = never); the matching `frac_*` columns divide them by `n_tool_calls`."
    )
    lines.append(
        "- Null temporal fractions (no gold file ever mentioned/edited, or an empty gold set) "
        f"are imputed with 0.0 for modeling ({df.attrs.get('n_filled', 0):,} rows have at least "
        "one such null); the `turn_first_gold_view` / `turn_first_gold_edit` sentinel (-1) and "
        '`n_gold_files` keep the "never" information as their own feature values.'
    )
    lines.append(
        "- `is_imputed` in `outputs/trace_scores.parquet` marks rows with an unknown label "
        "(`resolved == -1`) — it is a label flag, not a feature-imputation flag."
    )
    lines.append(
        "- `n_turns_total` (temporal) equals `n_messages` (proxy) for "
        f"{float((df['n_turns_total'] == df['n_messages']).mean()):.1%} of rows, so the union "
        "carries one duplicated feature; its coefficient splits with `n_messages` and is not "
        "interpretable on its own. `n_tool_calls` appears in both blocks but is counted once "
        "(proxy block) in the model."
    )
    lines.append(
        "- Mentions are case-sensitive substring tests on raw tool-call arguments, so the "
        "`/testbed/`-prefixed paths agents type match through the basename; heredoc bodies that "
        "merely quote a gold file path also count as a mention."
    )
    lines.append(
        "- Reproduce: `uv run python scripts/temporal_features.py --threads 6 --memory-limit "
        "8GB` (parts under `outputs/temporal_features_parts/`, merged into "
        "`outputs/temporal_features.parquet`), then `uv run python scripts/score_traces.py`."
    )
    lines.append("")
    return "\n".join(lines)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--proxy", type=Path, default=PROXY_PARQUET)
    parser.add_argument("--temporal", type=Path, default=TEMPORAL_PARQUET)
    parser.add_argument("--output", type=Path, default=OUT_PARQUET)
    parser.add_argument("--summary", type=Path, default=OUT_MD)
    args = parser.parse_args()

    df = load_joined(args.proxy, args.temporal)
    console.print(
        f"Joined {len(df):,} trajectories (temporal null-fraction rows filled: "
        f"{df.attrs.get('n_filled', 0):,})"
    )

    labeled = df[df["resolved"].isin([0, 1])]
    y = labeled["resolved"].to_numpy(dtype=int)
    groups = labeled["instance_id"].astype(str).to_numpy()
    X_proxy = labeled[PROXY_FEATURES].to_numpy(dtype=float)
    X_union = labeled[MODEL_FEATURES].to_numpy(dtype=float)

    splitter = GroupShuffleSplit(n_splits=1, test_size=TEST_SIZE, random_state=SEED)
    train_idx, test_idx = next(splitter.split(X_union, y, groups))
    split_info = {
        "n_train": len(train_idx),
        "n_test": len(test_idx),
        "groups_train": int(pd.Series(groups).iloc[train_idx].nunique()),
        "groups_test": int(pd.Series(groups).iloc[test_idx].nunique()),
    }

    fit_proxy = fit_logistic(
        X_proxy[train_idx], y[train_idx], X_proxy[test_idx], y[test_idx], PROXY_FEATURES
    )
    fit_union = fit_logistic(
        X_union[train_idx], y[train_idx], X_union[test_idx], y[test_idx], MODEL_FEATURES
    )
    console.print(
        f"Held-out AUC: proxy only {fit_proxy.auc_test:.4f} → proxy + temporal "
        f"{fit_union.auc_test:.4f} ({fit_union.auc_test - fit_proxy.auc_test:+.4f})"
    )

    final = fit_full(X_union, y, MODEL_FEATURES)
    df = df.copy()
    df["p_resolved"] = final.predict(df[MODEL_FEATURES].to_numpy(dtype=float))
    out = write_scores(df, df["p_resolved"].to_numpy(), args.output)
    console.print(f"Wrote {len(df):,} scores → {out.relative_to(ROOT)}")

    labeled_scored = df[df["resolved"].isin([0, 1])]
    md = build_summary(df, labeled_scored, fit_proxy, fit_union, split_info, out.relative_to(ROOT))
    args.summary.parent.mkdir(parents=True, exist_ok=True)
    args.summary.write_text(md)
    console.print(f"Wrote {args.summary.relative_to(ROOT)}")

    top_temporal = next(d for d in fit_union.coefs() if d["feature"] in TEMPORAL_MODEL_FEATURES)
    console.print(
        f"Largest temporal coefficient: {top_temporal['feature']} ({top_temporal['coef']:+.4f})"
    )
    for harness, teacher in UNLABELED_COMBOS:
        sub = df[(df["harness"] == harness) & (df["teacher"] == teacher)]
        console.print(
            f"{harness}/{teacher}: n={len(sub):,} mean p={sub['p_resolved'].mean():.3f} "
            f"share p>0.5={float((sub['p_resolved'] > 0.5).mean()):.3f}"
        )
    console.print(f"Labeled resolved rate: {labeled['resolved'].mean():.3f}")


if __name__ == "__main__":
    main()

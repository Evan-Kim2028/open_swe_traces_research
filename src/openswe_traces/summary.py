"""Summarize outputs/proxy_features.parquet for the pre-SFT ranking question.

Writes analytics/research/proxy_features_summary.md with:
  1. per harness/teacher means of the key structural features and the resolved
     rate (rows with resolved in (0, 1) only), plus the harness/teacher combos
     that are resolved = -1 for every row and were excluded;
  2. Pearson correlation of every numeric feature with `resolved` (rows with
     resolved in (0, 1));
  3. a standardized logistic regression on an 80/20 split grouped by
     `instance_id`, reporting AUC and the 5 largest |coef| features. Inputs are
     winsorized to the train split's 1st/99th percentile (heavy patch-size tail).

Example:
  uv run python scripts/proxy_features_summary.py
"""

from __future__ import annotations

import argparse
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
from .features import NUMERIC_FEATURES as FEATURE_COLUMNS

console = Console()

FEATURES_PARQUET = ROOT / "outputs" / "proxy_features.parquet"
OUT_MD = ROOT / "analytics" / "research" / "proxy_features_summary.md"

MEAN_FEATURES = [
    "n_assistant_turns",
    "repeat_call_rate",
    "n_edit_calls",
    "n_test_calls",
    "patch_file_jaccard",
]

WINSOR_PCT = (1, 99)


def sql_str(value: object) -> str:
    return "'" + str(value).replace("'", "''") + "'"


def fmt(value: float) -> str:
    return f"{value:.3f}" if abs(value) < 10 else f"{value:.1f}"


def md_table(header: list[str], rows: list[list[str]]) -> str:
    lines = ["| " + " | ".join(header) + " |", "|" + "|".join(["---"] * len(header)) + "|"]
    lines += ["| " + " | ".join(row) + " |" for row in rows]
    return "\n".join(lines)


def load(path: Path) -> pd.DataFrame:
    con = connect_ephemeral()
    con.execute("SET memory_limit='6GB'")
    try:
        df = con.execute(f"SELECT * FROM read_parquet({sql_str(path)})").df()
    finally:
        con.close()
    for col in FEATURE_COLUMNS:
        df[col] = pd.to_numeric(df[col], errors="coerce")
    return df


def means_by_group(df: pd.DataFrame, keys: list[str]) -> pd.DataFrame:
    return (
        df.groupby(keys, dropna=False)
        .agg(
            n=("trajectory_id", "size"),
            n_known=("resolved", lambda s: int(s.isin([0, 1]).sum())),
            resolved_rate=(
                "resolved",
                lambda s: float(s[s.isin([0, 1])].mean()) if s.isin([0, 1]).any() else np.nan,
            ),
            **{f: (f, "mean") for f in MEAN_FEATURES},
        )
        .reset_index()
    )


def correlations(df: pd.DataFrame) -> list[dict]:
    rows = []
    for feature in FEATURE_COLUMNS:
        sub = df[[feature, "resolved"]].dropna()
        if sub[feature].nunique() < 2:
            rows.append({"feature": feature, "n": len(sub), "pearson": np.nan, "auc": np.nan})
            continue
        y = sub["resolved"].to_numpy(dtype=float)
        x = sub[feature].to_numpy(dtype=float)
        rows.append(
            {
                "feature": feature,
                "n": len(sub),
                "pearson": float(pearsonr(x, y).statistic),
                "auc": float(roc_auc_score(y, x)),
            }
        )
    return rows


def _fit_model(
    X_train: np.ndarray, y_train: np.ndarray, X_test: np.ndarray, y_test: np.ndarray, usable: list[str]
) -> dict:
    """Standardized logistic regression; returns coefficients and train/test AUC."""
    scaler = StandardScaler().fit(X_train)
    model = LogisticRegression(max_iter=2000, solver="lbfgs")
    model.fit(scaler.transform(X_train), y_train)
    coefs = sorted(
        ({"feature": f, "coef": float(c)} for f, c in zip(usable, model.coef_[0])),
        key=lambda d: abs(d["coef"]),
        reverse=True,
    )
    return {
        "coefs": coefs,
        "auc_train": float(
            roc_auc_score(y_train, model.predict_proba(scaler.transform(X_train))[:, 1])
        ),
        "auc_test": float(
            roc_auc_score(y_test, model.predict_proba(scaler.transform(X_test))[:, 1])
        ),
    }


def fit_logistic(df: pd.DataFrame, *, test_size: float = 0.2, seed: int = 42) -> dict:
    sub = df[FEATURE_COLUMNS + ["resolved", "instance_id"]].dropna(
        subset=FEATURE_COLUMNS + ["resolved"]
    )
    usable = [f for f in FEATURE_COLUMNS if sub[f].nunique() > 1]
    dropped = [f for f in FEATURE_COLUMNS if f not in usable]
    groups = sub["instance_id"].astype(str)
    X = sub[usable].to_numpy(dtype=float)
    y = sub["resolved"].to_numpy(dtype=int)

    train_idx, test_idx = next(
        GroupShuffleSplit(n_splits=1, test_size=test_size, random_state=seed).split(X, y, groups)
    )
    X_train, X_test = X[train_idx], X[test_idx]
    y_train, y_test = y[train_idx], y[test_idx]

    raw = _fit_model(X_train, y_train, X_test, y_test, usable)

    # Winsorize to the train split's 1st/99th percentile before standardizing: a handful of rows
    # carry absurd patch sizes, and raw scaling lets that tail alone dominate a coefficient.
    lo = np.quantile(X_train, WINSOR_PCT[0] / 100, axis=0)
    hi = np.quantile(X_train, WINSOR_PCT[1] / 100, axis=0)
    wins = _fit_model(
        np.clip(X_train, lo, hi), y_train, np.clip(X_test, lo, hi), y_test, usable
    )

    return {
        "n": len(sub),
        "n_train": len(train_idx),
        "n_test": len(test_idx),
        "positives": int(y.sum()),
        "positives_train": int(y_train.sum()),
        "positives_test": int(y_test.sum()),
        "groups": int(groups.nunique()),
        "groups_train": int(groups.iloc[train_idx].nunique()),
        "groups_test": int(groups.iloc[test_idx].nunique()),
        "dropped": dropped,
        "coefs": wins["coefs"],
        "auc_train": wins["auc_train"],
        "auc_test": wins["auc_test"],
        "raw_auc_test": raw["auc_test"],
        "raw_top": raw["coefs"][0],
        "test_size": test_size,
        "n_clipped_train": int(np.count_nonzero((X_train < lo) | (X_train > hi))),
    }


def proxy_features_summary(parquet: Path = FEATURES_PARQUET, output: Path = OUT_MD) -> Path:
    df = load(parquet)
    if df.empty:
        raise SystemExit(f"No rows in {parquet}")

    known = df[df["resolved"].isin([0, 1])].copy()
    known["resolved"] = known["resolved"].astype(int)
    parquet_shown = (
        parquet.resolve().relative_to(ROOT) if parquet.resolve().is_relative_to(ROOT) else parquet
    )
    group_means = means_by_group(df, ["harness", "teacher"])
    excluded = group_means[group_means["n_known"] == 0]
    corr = correlations(known)
    lr = fit_logistic(known)

    lines: list[str] = []
    lines.append("# Proxy features — pre-SFT ranking signal")
    lines.append("")
    lines.append(
        f"Generated {datetime.now(UTC).strftime('%Y-%m-%d %H:%M UTC')} "
        f"by `scripts/proxy_features_summary.py` from `{parquet_shown}`."
    )
    lines.append("")
    lines.append(
        f"Rows (trajectories) **{len(df):,}** across **{df['harness'].nunique()} harnesses** × "
        f"**{df['teacher'].nunique()} teachers** × **{df['source'].nunique()} sources**; "
        f"resolved = 1: {int((df['resolved'] == 1).sum()):,}, 0: {int((df['resolved'] == 0).sum()):,}, "
        f"-1 (unknown): {int((df['resolved'] == -1).sum()):,}. "
        f"Sections 2-3 use the **{len(known):,}** rows with resolved in (0, 1) "
        f"(positive rate {known['resolved'].mean():.3f})."
    )
    lines.append("")

    lines.append("## 1. Means by harness / teacher")
    lines.append("")
    lines.append(
        "`resolved_rate` is computed over rows with resolved in (0, 1) only (`n_known`); the "
        "structural means use all rows of the combo (`n`)."
    )
    lines.append("")
    header = ["harness", "teacher", "n", "n_known"] + MEAN_FEATURES + ["resolved_rate"]
    rows = [
        [str(r["harness"]), str(r["teacher"]), f"{int(r['n']):,}", f"{int(r['n_known']):,}"]
        + [fmt(float(r[f])) for f in MEAN_FEATURES]
        + ["n/a" if pd.isna(r["resolved_rate"]) else f"{r['resolved_rate']:.3f}"]
        for _, r in group_means.iterrows()
    ]
    lines.append(md_table(header, rows))
    lines.append("")
    if len(excluded):
        combos = ", ".join(
            f"`{r['harness']}/{r['teacher']}` (n={int(r['n']):,})" for _, r in excluded.iterrows()
        )
        lines.append(
            f"Excluded: harness/teacher combos with resolved = -1 for **all** rows (no known "
            f"outcome, so no resolved rate; they contribute no rows to sections 2-3): {combos}."
        )
    else:
        lines.append("Excluded: no harness/teacher combo is resolved = -1 for all rows.")
    lines.append("")

    lines.append("## 2. Pearson correlation with `resolved` (rows with resolved in (0, 1))")
    lines.append("")
    lines.append(
        "`pearson` is the point-biserial correlation with the 0/1 outcome; `auc` is the univariate "
        "ROC AUC (0.5 = no signal). `n/a` marks a feature that is constant on this subset."
    )
    lines.append("")
    header_corr = ["feature", "n", "pearson", "auc"]
    ordered = sorted(
        corr,
        key=lambda d: (np.isnan(d["pearson"]), -abs(d["pearson"]) if not np.isnan(d["pearson"]) else 0),
    )
    rows_corr = [
        [
            r["feature"],
            f"{r['n']:,}",
            "n/a" if np.isnan(r["pearson"]) else f"{r['pearson']:+.4f}",
            "n/a" if np.isnan(r["auc"]) else f"{r['auc']:.4f}",
        ]
        for r in ordered
    ]
    lines.append(md_table(header_corr, rows_corr))
    lines.append("")

    lines.append("## 3. Standardized logistic regression (80/20 split grouped by instance_id)")
    lines.append("")
    lines.append(
        f"`LogisticRegression` (scikit-learn) on all {len(FEATURE_COLUMNS)} features, inputs "
        f"standardized (mean 0, sd 1); the scaler and model are fit on the 80% train split only. "
        f"The split is grouped by `instance_id`, so trajectories of the same task instance (which "
        f"can appear under several harness/teacher combos) never cross the train/test boundary. "
        f"`test_size=0.2`, `random_state=42`. Features are winsorized to the train split's "
        f"{WINSOR_PCT[0]}st/{WINSOR_PCT[1]}th percentile before standardizing "
        f"({lr['n_clipped_train']:,} train values clipped): the raw patch-size tail (max "
        f"{int(df['model_patch_lines'].max()):,} modified lines, "
        f"{int((df['model_patch_lines'] > 100_000).sum())} rows above 100k) otherwise puts a "
        f"{lr['raw_top']['coef']:+.2f} coefficient on `{lr['raw_top']['feature']}` and scores "
        f"held-out AUC {lr['raw_auc_test']:.4f}."
        + (f" Constant features dropped: {', '.join(lr['dropped'])}." if lr["dropped"] else "")
    )
    lines.append("")
    lines.append(
        f"- Train: {lr['n_train']:,} rows over {lr['groups_train']:,} instance_ids "
        f"({lr['positives_train']:,} positives)."
    )
    lines.append(
        f"- Test (20%): {lr['n_test']:,} rows over {lr['groups_test']:,} instance_ids "
        f"({lr['positives_test']:,} positives)."
    )
    lines.append(f"- **AUC (held out): {lr['auc_test']:.4f}** (train {lr['auc_train']:.4f}).")
    lines.append("")
    lines.append("5 largest |std coef| (log-odds change per 1 sd of the feature):")
    lines.append("")
    header_lr = ["rank", "feature", "std coef", "direction"]
    rows_lr = [
        [
            str(i + 1),
            d["feature"],
            f"{d['coef']:+.4f}",
            "higher → more likely resolved" if d["coef"] > 0 else "higher → less likely resolved",
        ]
        for i, d in enumerate(lr["coefs"][:5])
    ]
    lines.append(md_table(header_lr, rows_lr))
    lines.append("")

    lines.append("## Notes")
    lines.append("")
    lines.append(
        "- `model_patch_lines` / `gold_patch_lines` come from patch metadata and have a heavy tail. "
        "Winsorizing (section 3) improves the held-out AUC and keeps the coefficients "
        "interpretable, but treat the model patch as noisy: it is self-reported by the model and "
        "missing or degenerate for some runs."
    )
    lines.append(
        "- `ends_with_submit` is near-constant (~1.0: published trajectories are truncated at each "
        "harness's terminal `submit`/`finish` action), so it carries almost no signal; "
        "`reasoning_chars` is near-zero for most harnesses."
    )
    lines.append(
        "- The count features (`n_messages`, `n_assistant_turns`, `n_tool_calls`, "
        "`n_distinct_tool_commands`) are near-collinear (pairwise r ≈ 0.95-0.99), so their "
        "individual coefficients split into offsetting pairs (e.g. `n_tool_calls` positive with "
        "`n_assistant_turns` negative); read them jointly as \"longer trajectories resolve less\", "
        "not as independent effects."
    )
    lines.append(
        "- Edit/test counts are lexical: they match command patterns in tool arguments "
        "(`$.command`), so they miss edits expressed through other fields and can be triggered by "
        "heredoc bodies that contain test command text."
    )
    lines.append(
        "- Correlations are pooled over harnesses, teachers and sources and are associational; "
        "harness composition (e.g. a low openhands resolved rate) can confound pooled numbers."
    )
    lines.append(
        "- Reproduce: `uv run python scripts/proxy_features.py` (per-shard parts under "
        "`outputs/proxy_features_parts/`, merged into `outputs/proxy_features.parquet`), then "
        "`uv run python scripts/proxy_features_summary.py`."
    )
    lines.append("")

    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text("\n".join(lines))
    shown = output.relative_to(ROOT) if output.is_relative_to(ROOT) else output
    console.print(f"Wrote {shown}")
    top5 = ", ".join(f"{d['feature']} ({d['coef']:+.3f})" for d in lr["coefs"][:5])
    console.print(f"Top-5 by |std coef|: {top5}")
    console.print(
        f"Rows: {len(df):,} | known-outcome rows: {len(known):,} | "
        f"AUC held-out={lr['auc_test']:.4f} train={lr['auc_train']:.4f}"
    )
    return output


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--parquet", type=Path, default=FEATURES_PARQUET)
    parser.add_argument("--output", type=Path, default=OUT_MD)
    args = parser.parse_args()
    proxy_features_summary(args.parquet, args.output)

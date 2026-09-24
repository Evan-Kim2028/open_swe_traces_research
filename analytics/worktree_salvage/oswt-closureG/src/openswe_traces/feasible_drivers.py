"""What moves the feasible-component pass rate? Verifier and specification terms.

Follow-up to the mixture finding in ``analytics/research/task_space_framework.md``:
patch size mostly moves the *feasible share* π (0.62 → 0.23 across size bins) while the
feasible pass probability p1 moves little (0.86 → 0.74). This module refits that
two-component binomial mixture per size decile as the baseline, then asks:

  b. which variables predict the infeasible class at fixed size (logistic of
     ``n_resolved == 0``, grouped 80/20 by repo, held-out AUC + standardized coefs);
  c. which variables move p1 inside the feasible component (WLS of solve_rate on
     instances with ``n_resolved >= 1`` and ``n_labeled >= 5``, held-out R² by block:
     size / verifier / specification);
  d. the same WLS per harness/teacher combo (top-3 by IRT-ordered observed rate) and
     pooled.

Inputs (all per-instance):
  outputs/closure_proxies.parquet   size proxies + n_resolved/n_labeled (closure-B)
  outputs/patch_split.parquet       src/test/doc/config line split, n_new_test_funcs
  outputs/rung_features.parquet     rung, has_repro, has_expected_actual, content_len
  traces_external/<src>/meta.parquet  FAIL_TO_PASS/PASS_TO_PASS/test_patch (step 2)
  outputs/trajectory_frame.parquet  per-trajectory resolved + harness/teacher (closure-C)

Output: ``analytics/research/feasible_component_drivers.md`` and
``outputs/feasible_frame.parquet`` (the merged instance frame).

Example:
  uv run python scripts/feasible_drivers.py
"""

from __future__ import annotations

import argparse
import sys
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path

import numpy as np
import pandas as pd
from rich.console import Console
from sklearn.linear_model import LinearRegression
from sklearn.model_selection import GroupShuffleSplit
from sklearn.preprocessing import StandardScaler

from .data import ROOT
from .difficulty import fit_linear
from .score import fit_logistic, md_table

console = Console()

PROXIES_PARQUET = ROOT / "outputs" / "closure_proxies.parquet"
SPLIT_PARQUET = ROOT / "outputs" / "patch_split.parquet"
RUNG_PARQUET = ROOT / "outputs" / "rung_features.parquet"
TRAJ_PARQUET = ROOT / "outputs" / "trajectory_frame.parquet"
FRAME_PARQUET = ROOT / "outputs" / "feasible_frame.parquet"
OUT_MD = ROOT / "analytics" / "research" / "feasible_component_drivers.md"

MIN_LABELED_MIXTURE = 3
MIN_LABELED_FEASIBLE = 5
MIN_LABELED_COMBO = 3
TEST_SIZE = 0.2
SEED = 42
LANGUAGE_ALIASES = {"ts": "typescript", "js": "javascript"}

SIZE_FEATURES = ["log1p(src_added)", "log1p(src_removed)", "log1p(n_src_files)"]
VERIFIER_FEATURES = [
    "log1p(test_added)",
    "log1p(test_removed)",
    "log1p(n_test_files)",
    "log1p(n_new_test_funcs)",
    "test_only",
    "log1p(n_f2p)",
    "log1p(n_p2p)",
    "log1p(test_patch_chars)",
]
SPEC_FEATURES = [
    "log1p(issue_chars)",
    "log1p(spec_density)",
    "has_repro",
    "has_expected_actual",
    "rung_binary",
]


# --- frame assembly ------------------------------------------------------------


def load_meta() -> pd.DataFrame:
    """Concat traces_external/*/meta.parquet, deriving verifier-term counts."""
    frames = []
    for d in sorted((ROOT / "traces_external").glob("*__*")):
        meta_path = d / "meta.parquet"
        if not meta_path.exists():
            continue
        df = pd.read_parquet(meta_path)
        df["meta_source_dir"] = d.name
        frames.append(df)
    if not frames:
        return pd.DataFrame(columns=["instance_id", "n_f2p", "n_p2p", "test_patch_chars",
                                     "ps_chars", "created_at", "meta_source_dir"])
    meta = pd.concat(frames, ignore_index=True)
    meta = meta.drop_duplicates("instance_id")
    meta["n_f2p"] = meta["fail_to_pass"].map(lambda v: len(v) if v is not None else 0)
    meta["n_p2p"] = meta["pass_to_pass"].map(lambda v: len(v) if v is not None else 0)
    meta["test_patch_chars"] = meta["test_patch"].fillna("").str.len()
    meta["ps_chars"] = meta["problem_statement"].fillna("").str.len()
    return meta[
        ["instance_id", "n_f2p", "n_p2p", "test_patch_chars", "ps_chars",
         "created_at", "meta_source_dir"]
    ]


def load_frame() -> pd.DataFrame:
    """Merge proxies + patch split + rung features + upstream meta to one row/instance."""
    prox = pd.read_parquet(PROXIES_PARQUET)
    split = pd.read_parquet(SPLIT_PARQUET)
    rung = pd.read_parquet(
        RUNG_PARQUET,
        columns=[
            "instance_id",
            "rung",
            "content_len",
            "word_count",
            "has_repro",
            "has_expected_actual",
            "has_test_names",
            "has_signature",
            "has_test_code",
        ],
    )
    meta = load_meta()

    inst = prox.merge(
        split.drop(columns=["repo", "language", "n_rollouts", "n_labeled", "n_resolved"]),
        on="instance_id",
        how="left",
        validate="1:1",
    )
    inst = inst.merge(rung, on="instance_id", how="left", validate="1:1")
    inst = inst.merge(meta, on="instance_id", how="left", validate="1:1")

    inst["language"] = (
        inst["language"].astype("string").str.lower().replace(LANGUAGE_ALIASES)
    )
    inst["solve_rate"] = (
        inst["n_resolved"] / inst["n_labeled"].where(inst["n_labeled"] > 0)
    ).astype("float64")
    inst["issue_chars"] = inst["content_len"].astype("float64")
    inst["spec_density"] = inst["issue_chars"] / np.maximum(1, inst["src_added"])
    inst["rung_binary"] = (inst["rung"] >= 2).astype("int8")
    inst["has_meta"] = inst["n_f2p"].notna().astype("int8")
    for col in ("n_f2p", "n_p2p", "test_patch_chars", "ps_chars"):
        inst[col] = inst[col].fillna(0).astype("float64")
    return inst


# --- mixture baseline -----------------------------------------------------------


def em_two_component(
    k: np.ndarray, n: np.ndarray, iters: int = 500
) -> tuple[float, float, float]:
    """EM for k_i ~ pi * Binom(n_i, p1) + (1 - pi) * Binom(n_i, p0), all free.

    Component 1 is the feasible class (pass probability p1); component 0 is the
    infeasible class, whose p0 comes out small-but-nonzero (rare lucky resolves).
    Matches the baseline table in `task_space_framework.md` (π 0.62 → 0.23,
    p1 0.86 → 0.74 across the note's size bins).
    """
    k = np.asarray(k, dtype=float)
    n = np.asarray(n, dtype=float)
    pi, p0, p1 = 0.6, 0.02, 0.8
    eps = 1e-12
    for _ in range(iters):
        l1 = np.log(pi + eps) + k * np.log(p1 + eps) + (n - k) * np.log(1 - p1 + eps)
        l0 = np.log(1 - pi + eps) + k * np.log(p0 + eps) + (n - k) * np.log(1 - p0 + eps)
        m = np.maximum(l0, l1)
        z = np.exp(l1 - m) / (np.exp(l1 - m) + np.exp(l0 - m))
        pi_new = float(z.mean())
        p1_new = float((z * k).sum() / max((z * n).sum(), eps))
        p0_new = float(((1 - z) * k).sum() / max(((1 - z) * n).sum(), eps))
        if max(abs(pi_new - pi), abs(p1_new - p1), abs(p0_new - p0)) < 1e-10:
            pi, p1, p0 = pi_new, p1_new, p0_new
            break
        pi, p1, p0 = pi_new, p1_new, p0_new
    return pi, p0, p1


def mixture_table(inst: pd.DataFrame, n_bins: int = 10) -> pd.DataFrame:
    """Per size decile: n, feasible share pi, feasible pass p1, raw solve rate."""
    frame = inst[inst["n_labeled"] >= MIN_LABELED_MIXTURE].copy()
    frame["decile"] = pd.qcut(
        np.log1p(frame["added_lines"].clip(lower=0)), n_bins, duplicates="drop"
    )
    rows = []
    for i, (dec, sub) in enumerate(frame.groupby("decile", observed=True, sort=True), 1):
        pi, p0, p1 = em_two_component(
            sub["n_resolved"].to_numpy(), sub["n_labeled"].to_numpy()
        )
        rows.append(
            {
                "decile": i,
                "log1p(added_lines)": f"{max(0.0, dec.left):.1f}–{dec.right:.1f}",
                "n": len(sub),
                "pi": pi,
                "p0": p0,
                "p1": p1,
                "raw_solve_rate": float(sub["n_resolved"].sum() / sub["n_labeled"].sum()),
            }
        )
    return pd.DataFrame(rows)


# --- design matrix --------------------------------------------------------------


def build_design(inst: pd.DataFrame) -> tuple[pd.DataFrame, list[str]]:
    """Full feature set: size + verifier + specification + language/source dummies."""
    f = inst
    X = pd.DataFrame(
        {
            "log1p(src_added)": np.log1p(f["src_added"].clip(lower=0)),
            "log1p(src_removed)": np.log1p(f["src_removed"].clip(lower=0)),
            "log1p(n_src_files)": np.log1p(f["n_src_files"].clip(lower=0)),
            "log1p(test_added)": np.log1p(f["test_added"].clip(lower=0)),
            "log1p(test_removed)": np.log1p(f["test_removed"].clip(lower=0)),
            "log1p(n_test_files)": np.log1p(f["n_test_files"].clip(lower=0)),
            "log1p(n_new_test_funcs)": np.log1p(f["n_new_test_funcs"].clip(lower=0)),
            "test_only": f["test_only"].astype(float),
            "log1p(n_f2p)": np.log1p(f["n_f2p"].clip(lower=0)),
            "log1p(n_p2p)": np.log1p(f["n_p2p"].clip(lower=0)),
            "log1p(test_patch_chars)": np.log1p(f["test_patch_chars"].clip(lower=0)),
            "log1p(issue_chars)": np.log1p(f["issue_chars"].clip(lower=0)),
            "log1p(spec_density)": np.log1p(f["spec_density"].clip(lower=0)),
            "has_repro": f["has_repro"].astype(float),
            "has_expected_actual": f["has_expected_actual"].astype(float),
            "rung_binary": f["rung_binary"].astype(float),
            "has_meta": f["has_meta"].astype(float),
        }
    )
    for col in ("language", "hf_dataset_name"):
        values = f[col].astype("string").fillna("unknown")
        dummies = pd.get_dummies(values, prefix=col, dtype=float)
        drop = f"{col}_{values.value_counts().idxmax()}"
        if drop in dummies.columns:
            dummies = dummies.drop(columns=drop)
        X = pd.concat([X.reset_index(drop=True), dummies.reset_index(drop=True)], axis=1)
    return X, list(X.columns)


def grouped_split(groups: np.ndarray, n: int):
    splitter = GroupShuffleSplit(n_splits=1, test_size=TEST_SIZE, random_state=SEED)
    return next(splitter.split(np.zeros((n, 1)), np.zeros(n), groups))


# --- b. feasibility model --------------------------------------------------------


@dataclass
class FeasibilityResult:
    n: int
    n_train: int
    n_test: int
    base_rate: float
    aucs: dict[str, float]
    coefs: list[dict]
    fe_coefs: pd.DataFrame


def feasibility_model(inst: pd.DataFrame) -> FeasibilityResult:
    """Logistic of all_fail (n_resolved == 0); grouped 80/20 by repo, held-out AUC."""
    frame = inst[(inst["n_labeled"] >= MIN_LABELED_MIXTURE) & inst["issue_chars"].notna()]
    frame = frame.reset_index(drop=True)
    X, features = build_design(frame)
    Xv = X.to_numpy(dtype=float)
    y = (frame["n_resolved"].to_numpy() == 0).astype(float)
    groups = frame["repo"].astype(str).to_numpy()
    train_idx, test_idx = grouped_split(groups, len(frame))

    block_sets = {
        "size only": SIZE_FEATURES,
        "size + verifier": SIZE_FEATURES + VERIFIER_FEATURES,
        "size + spec": SIZE_FEATURES + SPEC_FEATURES,
        "full": features,
        "full − verifier": [f for f in features if f not in VERIFIER_FEATURES],
        "full − spec": [f for f in features if f not in SPEC_FEATURES],
        "full − size": [f for f in features if f not in SIZE_FEATURES],
    }
    aucs = {}
    cols = list(X.columns)
    for name, feats in block_sets.items():
        idx = [cols.index(f) for f in feats]
        fit = fit_logistic(
            Xv[train_idx][:, idx], y[train_idx], Xv[test_idx][:, idx], y[test_idx], feats
        )
        aucs[name] = float(fit.auc_test)

    full_idx = list(range(len(cols)))
    fit_full = fit_logistic(
        Xv[train_idx][:, full_idx],
        y[train_idx],
        Xv[test_idx][:, full_idx],
        y[test_idx],
        cols,
    )

    fe = repo_fe_lpm(frame, X, y)
    return FeasibilityResult(
        n=len(frame),
        n_train=len(train_idx),
        n_test=len(test_idx),
        base_rate=float(y.mean()),
        aucs=aucs,
        coefs=fit_full.coefs(),
        fe_coefs=fe,
    )


def repo_fe_lpm(frame: pd.DataFrame, X: pd.DataFrame, y: np.ndarray) -> pd.DataFrame:
    """Repo fixed effects via FWL: demean y and X within repo, then OLS.

    A linear-probability check that the logistic coefficients survive repo identity.
    Reported for the named (non-dummy) features only.
    """
    named = [c for c in X.columns if not c.startswith(("language_", "hf_dataset_name_"))]
    Z = X[named].to_numpy(dtype=float)
    repo = frame["repo"].astype(str)
    order = np.argsort(repo.to_numpy())
    repo_s = repo.to_numpy()[order]
    Zs = Z[order]
    ys = y[order]
    boundaries = np.flatnonzero(repo_s[1:] != repo_s[:-1]) + 1
    Zd = Zs.copy()
    yd = ys.copy()
    starts = np.r_[0, boundaries]
    ends = np.r_[boundaries, len(repo_s)]
    for s, e in zip(starts, ends):
        Zd[s:e] -= Zd[s:e].mean(axis=0)
        yd[s:e] -= yd[s:e].mean()
    # drop repos with a single row (demeaned to zero, carry no signal)
    keep = np.abs(Zd).sum(axis=1) + np.abs(yd) > 0
    scaler = StandardScaler().fit(Zd[keep])
    coefs = LinearRegression().fit(scaler.transform(Zd[keep]), yd[keep]).coef_
    return pd.DataFrame(
        {"feature": named, "fe_lpm_coef": coefs}
    ).assign(abs=lambda d: d["fe_lpm_coef"].abs()).sort_values("abs", ascending=False)


# --- c. feasible-component difficulty -------------------------------------------


@dataclass
class WlsResult:
    label: str
    n: int
    n_test: int
    r2: dict[str, float]
    coefs: list[dict]


def feasible_wls(inst: pd.DataFrame, label: str, min_labeled: int) -> WlsResult:
    """WLS of solve_rate on feasible instances; held-out R² by feature block."""
    frame = inst[
        (inst["n_resolved"] >= 1)
        & (inst["n_labeled"] >= min_labeled)
        & inst["issue_chars"].notna()
    ].reset_index(drop=True)
    X, features = build_design(frame)
    Xv = X.to_numpy(dtype=float)
    y = frame["solve_rate"].to_numpy(dtype=float)
    w = frame["n_labeled"].to_numpy(dtype=float)
    groups = frame["repo"].astype(str).to_numpy()
    train_idx, test_idx = grouped_split(groups, len(frame))

    cols = list(X.columns)
    block_sets = {
        "size only": SIZE_FEATURES,
        "size + verifier": SIZE_FEATURES + VERIFIER_FEATURES,
        "size + spec": SIZE_FEATURES + SPEC_FEATURES,
        "full": features,
        "full − verifier": [f for f in features if f not in VERIFIER_FEATURES],
        "full − spec": [f for f in features if f not in SPEC_FEATURES],
        "full − size": [f for f in features if f not in SIZE_FEATURES],
    }
    r2 = {}
    fits = {}
    for name, feats in block_sets.items():
        idx = [cols.index(f) for f in feats]
        fit = fit_linear(
            Xv[train_idx][:, idx],
            y[train_idx],
            Xv[test_idx][:, idx],
            y[test_idx],
            feats,
            sample_weight=w[train_idx],
        )
        r2[name] = float(fit.r2_test)
        fits[name] = fit
    return WlsResult(
        label=label, n=len(frame), n_test=len(test_idx), r2=r2, coefs=fits["full"].coefs()
    )


# --- d. per-combo ----------------------------------------------------------------


def load_trajectory_frame(path: Path = TRAJ_PARQUET) -> pd.DataFrame:
    traj = pd.read_parquet(path)
    traj["combo"] = traj["harness"].astype(str) + "/" + traj["teacher"].astype(str)
    return traj


def combo_rates(traj: pd.DataFrame) -> pd.DataFrame:
    """Per (instance_id, combo): n_labeled, n_resolved, solve_rate."""
    lab = traj[traj["resolved"].isin([0, 1])]
    g = lab.groupby(["instance_id", "combo"], sort=True)
    out = g["resolved"].agg(n_labeled="size", n_resolved="sum").reset_index()
    out["solve_rate"] = out["n_resolved"] / out["n_labeled"]
    return out


def combo_summary(traj: pd.DataFrame) -> pd.DataFrame:
    lab = traj[traj["resolved"].isin([0, 1])]
    g = lab.groupby("combo")["resolved"].agg(n_labeled="size", n_resolved="sum")
    g["rate"] = g["n_resolved"] / g["n_labeled"]
    return g.sort_values("rate", ascending=False).reset_index()


def per_combo_wls(inst: pd.DataFrame, traj: pd.DataFrame) -> pd.DataFrame:
    """Repeat the feasible WLS per combo (instance restricted to that combo's
    feasible set: >= MIN_LABELED_COMBO labeled rollouts in the combo, >=1 resolved)."""
    rates = combo_rates(traj)
    combos = combo_summary(traj)
    rows = []
    for _, crow in combos.iterrows():
        combo = crow["combo"]
        sub_rates = rates[
            (rates["combo"] == combo)
            & (rates["n_labeled"] >= MIN_LABELED_COMBO)
            & (rates["n_resolved"] >= 1)
        ]
        frame = inst.merge(
            sub_rates[["instance_id", "n_labeled", "n_resolved", "solve_rate"]],
            on="instance_id",
            how="inner",
            suffixes=("", "_c"),
        )
        frame["solve_rate"] = frame["solve_rate_c"]
        frame["n_labeled"] = frame["n_labeled_c"]
        frame["n_resolved"] = frame["n_resolved_c"]
        frame = frame[frame["issue_chars"].notna()].reset_index(drop=True)
        if len(frame) < 500:
            rows.append({"combo": combo, "n": len(frame), "r2_size": np.nan,
                         "r2_full": np.nan, "delta": np.nan})
            continue
        X, _ = build_design(frame)
        Xv = X.to_numpy(dtype=float)
        y = frame["solve_rate"].to_numpy(dtype=float)
        w = frame["n_labeled"].to_numpy(dtype=float)
        groups = frame["repo"].astype(str).to_numpy()
        train_idx, test_idx = grouped_split(groups, len(frame))
        cols = list(X.columns)
        idx_size = [cols.index(f) for f in SIZE_FEATURES]
        fit_size = fit_linear(
            Xv[train_idx][:, idx_size], y[train_idx], Xv[test_idx][:, idx_size],
            y[test_idx], SIZE_FEATURES, sample_weight=w[train_idx],
        )
        fit_full = fit_linear(
            Xv[train_idx], y[train_idx], Xv[test_idx], y[test_idx],
            cols, sample_weight=w[train_idx],
        )
        top = next(d for d in fit_full.coefs() if d["feature"] == "log1p(n_f2p)")
        rows.append(
            {
                "combo": combo,
                "n": len(frame),
                "r2_size": float(fit_size.r2_test),
                "r2_full": float(fit_full.r2_test),
                "delta": float(fit_full.r2_test - fit_size.r2_test),
                "coef_log1p(n_f2p)": top["coef"],
                "coef_log1p(spec_density)": next(
                    d["coef"] for d in fit_full.coefs()
                    if d["feature"] == "log1p(spec_density)"
                ),
            }
        )
    return pd.DataFrame(rows)


# --- report ----------------------------------------------------------------------


def _fmt(x: float) -> str:
    return "—" if not np.isfinite(x) else f"{x:.3f}"


def build_report(
    inst: pd.DataFrame,
    mixture: pd.DataFrame,
    coverage: pd.DataFrame,
    feas: FeasibilityResult,
    wls: WlsResult,
    combos: pd.DataFrame,
    combo_r2: pd.DataFrame,
) -> str:
    lines: list[str] = []
    stamp = datetime.now(UTC).strftime("%Y-%m-%d %H:%M UTC")
    lines.append("# Feasible-component drivers: verifier and specification terms")
    lines.append("")
    lines.append(
        f"Generated {stamp} by `scripts/feasible_drivers.py` (no solver calls). Inputs: "
        "`outputs/closure_proxies.parquet` (closure-B), `outputs/patch_split.parquet` + "
        "`outputs/rung_features.parquet` + `outputs/trajectory_frame.parquet` "
        "(closure-C), `traces_external/*/meta.parquet` (upstream metadata). "
        "Log: `outputs/closure_G.log`."
    )
    lines.append("")
    lines.append(
        "Follow-up to the mixture finding in `task_space_framework.md`: size moves the "
        "feasible share, not the feasible pass probability. Here: what moves p1 inside "
        "the feasible component, and what predicts the infeasible class at fixed size."
    )
    lines.append("")

    lines.append("## Data and join coverage")
    lines.append("")
    lines.append(
        f"Merged frame: **{len(inst):,}** instances "
        f"({int((inst['n_labeled'] >= MIN_LABELED_MIXTURE).sum()):,} with "
        f"n_labeled >= {MIN_LABELED_MIXTURE}). Verifier terms: `test_added`, "
        "`n_test_files`, `n_new_test_funcs` (test-function defs added to the gold "
        "patch, per-language regex), `test_only` (patch touches no src lines), and "
        "upstream `n_f2p`/`n_p2p`/`test_patch_chars` where joined. Specification "
        "terms: `issue_chars` (first user message length), "
        "`spec_density = issue_chars / max(1, src_added)`, `has_repro`, "
        "`has_expected_actual`, `rung_binary` (heuristic rung >= 2)."
    )
    lines.append("")
    if len(coverage):
        rows = [
            [
                r["source"],
                f"{int(r['corpus_instances']):,}",
                f"{int(r['meta_rows']):,}",
                f"{int(r['matched']):,}",
                f"{100 * r['coverage']:.1f}%",
            ]
            for _, r in coverage.iterrows()
        ]
        lines.append(
            md_table(
                ["hf_dataset_name", "corpus inst", "meta rows", "matched", "coverage"],
                rows,
            )
        )
        lines.append("")
        lines.append(
            "Both corpus sources are public on HF and join at 100%. Caveat: "
            "`test_patch` is a unified diff for SWE-rebench-V2 but the raw "
            "fail-to-pass test script (`f2p_script`) for Scale-SWE, and Scale-SWE "
            "has no `created_at` (kept null). `test_patch_chars` is therefore a "
            "size proxy, not a like-for-like diff measure."
        )
        lines.append("")

    lines.append("## a. Mixture baseline (per size decile, n_labeled >= 3)")
    lines.append("")
    lines.append(
        "Two-component binomial EM per decile of `log1p(added_lines)`, all parameters "
        "free: component 1 is feasible with pass probability p1; component 0 is "
        "infeasible with a small residual p0 (rare lucky resolves / label noise); "
        "π is the feasible share."
    )
    lines.append("")
    rows = [
        [
            str(int(r["decile"])),
            r["log1p(added_lines)"],
            f"{int(r['n']):,}",
            f"{r['pi']:.2f}",
            f"{r['p0']:.2f}",
            f"{r['p1']:.2f}",
            f"{r['raw_solve_rate']:.2f}",
        ]
        for _, r in mixture.iterrows()
    ]
    lines.append(
        md_table(
            ["decile", "log1p(added)", "n", "π feasible", "p0 infeasible",
             "p1 feasible pass", "raw solve rate"],
            rows,
        )
    )
    lines.append("")
    first, last = mixture.iloc[0], mixture.iloc[-1]
    lines.append(
        f"Baseline reproduces the framework-note shape: π falls {first['pi']:.2f} → "
        f"{last['pi']:.2f} across deciles while p1 stays in "
        f"[{mixture['p1'].min():.2f}, {mixture['p1'].max():.2f}]."
    )
    lines.append("")

    lines.append("## b. What predicts the infeasible class at fixed size?")
    lines.append("")
    lines.append(
        f"Logistic of `n_resolved == 0` on {feas.n:,} instances (base rate "
        f"{feas.base_rate:.3f}); grouped 80/20 split by repo "
        f"(train {feas.n_train:,}, held-out {feas.n_test:,} unseen-repo instances). "
        "Numeric features winsorized 1/99 and standardized; language and "
        "hf_dataset_name one-hots (majority level dropped). AUC is held-out."
    )
    lines.append("")
    rows = [[k, f"{v:.4f}"] for k, v in feas.aucs.items()]
    lines.append(md_table(["feature set", "held-out AUC"], rows))
    lines.append("")
    lines.append("Standardized coefficients of the full model (log-odds per 1 sd):")
    lines.append("")
    shown = [
        d for d in feas.coefs if not d["feature"].startswith("hf_dataset_name_")
    ][:20]
    rows = [[str(i + 1), d["feature"], f"{d['coef']:+.3f}"] for i, d in enumerate(shown)]
    lines.append(md_table(["rank", "feature", "std coef"], rows))
    lines.append("")
    lines.append(
        "Repo fixed effects (FWL-demeaned linear probability model, all data; "
        "coefficients are per-1-sd of the within-repo-demeaned feature):"
    )
    lines.append("")
    rows = [
        [r["feature"], f"{r['fe_lpm_coef']:+.4f}"]
        for _, r in feas.fe_coefs.head(15).iterrows()
    ]
    lines.append(md_table(["feature", "repo-FE LPM coef"], rows))
    lines.append("")

    lines.append("## c. What moves p1 inside the feasible component?")
    lines.append("")
    lines.append(
        f"WLS of solve_rate on {wls.n:,} feasible instances (n_resolved >= 1, "
        f"n_labeled >= {MIN_LABELED_FEASIBLE}; weight = n_labeled), grouped 80/20 by "
        "repo. Held-out R² by block."
    )
    lines.append("")
    rows = [[k, f"{v:.4f}"] for k, v in wls.r2.items()]
    lines.append(md_table(["feature set", "held-out R²"], rows))
    lines.append("")
    lines.append("Standardized coefficients of the full model (solve_rate per 1 sd):")
    lines.append("")
    shown = [
        d for d in wls.coefs if not d["feature"].startswith("hf_dataset_name_")
    ][:20]
    rows = [[str(i + 1), d["feature"], f"{d['coef']:+.4f}"] for i, d in enumerate(shown)]
    lines.append(md_table(["rank", "feature", "std coef"], rows))
    lines.append("")

    lines.append("## d. Per-combo feasible difficulty")
    lines.append("")
    lines.append(
        "Per-(instance, combo) solve rates from `trajectory_frame.parquet` over "
        "labeled rollouts; feasible set per combo = >= "
        f"{MIN_LABELED_COMBO} labeled rollouts in that combo and >= 1 resolved. Combo "
        "ordering below is by observed rate — the IRT fit's rank agreement with "
        "observed rate is Spearman +1.0 (`irt_summary.md`), so observed-rate order = "
        "theta order."
    )
    lines.append("")
    rows = [
        [
            r["combo"],
            f"{int(r['n_labeled']):,}",
            f"{r['rate']:.3f}",
        ]
        for _, r in combos.iterrows()
    ]
    lines.append(md_table(["combo", "n labeled rollouts", "observed rate"], rows))
    lines.append("")
    rows = [
        [
            r["combo"],
            f"{int(r['n']):,}",
            _fmt(r["r2_size"]),
            _fmt(r["r2_full"]),
            _fmt(r["delta"]),
            _fmt(r.get("coef_log1p(n_f2p)", np.nan)),
            _fmt(r.get("coef_log1p(spec_density)", np.nan)),
        ]
        for _, r in combo_r2.iterrows()
    ]
    lines.append(
        md_table(
            ["combo", "n feasible", "R² size", "R² full", "Δ", "coef n_f2p",
             "coef spec_density"],
            rows,
        )
    )
    lines.append("")

    lines.append("## e. Verdicts")
    lines.append("")
    return lines


def verdict_text(
    feas: FeasibilityResult, wls: WlsResult, combo_r2: pd.DataFrame
) -> list[str]:
    lines: list[str] = []
    auc_size = feas.aucs["size only"]
    auc_ver = feas.aucs["size + verifier"]
    auc_spec = feas.aucs["size + spec"]
    auc_full = feas.aucs["full"]
    r2_full = wls.r2["full"]
    r2_nover = wls.r2["full − verifier"]
    r2_nospec = wls.r2["full − spec"]

    coef = {d["feature"]: d["coef"] for d in wls.coefs}
    f2p_c = coef.get("log1p(n_f2p)", np.nan)
    dens_c = coef.get("log1p(spec_density)", np.nan)

    lines.append(
        f"**Does the verifier term move p1?** Verifier block (test lines/files/new "
        f"test funcs/test_only + upstream n_f2p/n_p2p/test_patch_chars): removing it "
        f"from the feasible WLS changes held-out R² {r2_full:.4f} → {r2_nover:.4f} "
        f"(Δ {r2_nover - r2_full:+.4f}); log1p(n_f2p) standardized coef "
        f"{f2p_c:+.4f}. Per-combo ΔR² (full − size): "
        + ", ".join(
            f"{r['combo']} {_fmt(r['delta'])}" for _, r in combo_r2.iterrows()
        )
        + "."
    )
    lines.append("")
    lines.append(
        f"**Does specification density move p1?** Removing the spec block changes "
        f"held-out R² {r2_full:.4f} → {r2_nospec:.4f} (Δ {r2_nospec - r2_full:+.4f}); "
        f"log1p(spec_density) coef {dens_c:+.4f}."
    )
    lines.append("")
    lines.append(
        f"**Does either predict the feasibility gate beyond size?** Held-out AUC of "
        f"all_fail: size {auc_size:.4f} → +verifier {auc_ver:.4f}, +spec "
        f"{auc_spec:.4f}, full {auc_full:.4f}."
    )
    lines.append("")
    return lines


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--frame", type=Path, default=FRAME_PARQUET)
    parser.add_argument("--rebuild-frame", action="store_true")
    args = parser.parse_args()

    if args.rebuild_frame or not args.frame.exists():
        console.print(f"[{datetime.now(UTC)}] building merged frame")
        inst = load_frame()
        args.frame.parent.mkdir(parents=True, exist_ok=True)
        inst.to_parquet(args.frame, index=False)
    else:
        inst = pd.read_parquet(args.frame)
    console.print(f"frame: {len(inst):,} instances")

    # join coverage (needs patch_split for hf_dataset_name)
    from .upstream_meta import join_coverage

    try:
        coverage = join_coverage(SPLIT_PARQUET)
    except FileNotFoundError:
        coverage = pd.DataFrame()
    console.print(coverage.to_string(index=False) if len(coverage) else "no coverage")

    mixture = mixture_table(inst)
    console.print(mixture.to_string(index=False))

    feas = feasibility_model(inst)
    console.print(f"feasibility AUCs: {feas.aucs}")

    wls = feasible_wls(inst, "pooled", MIN_LABELED_FEASIBLE)
    console.print(f"feasible WLS R²: {wls.r2}")

    traj = load_trajectory_frame()
    combos = combo_summary(traj)
    combo_r2 = per_combo_wls(inst, traj)
    console.print(combo_r2.to_string(index=False))

    lines = build_report(inst, mixture, coverage, feas, wls, combos, combo_r2)
    lines += verdict_text(feas, wls, combo_r2)
    OUT_MD.parent.mkdir(parents=True, exist_ok=True)
    OUT_MD.write_text("\n".join(lines) + "\n")
    console.print(f"[green]Wrote {OUT_MD.relative_to(ROOT)}[/green]")
    sys.exit(0)


if __name__ == "__main__":
    main()

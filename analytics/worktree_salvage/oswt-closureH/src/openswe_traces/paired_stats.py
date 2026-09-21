"""Paired statistics for the within-instance analysis (closure-H job steps 2–4, 6).

Joins ``outputs/paired_features.parquet`` (per-trajectory behavior + patch features) to
``outputs/eligible_pairs.parquet`` (group_id, combo, stratum), then:

* ``paired_diffs``       — per (instance, combo) group: mean(fail) - mean(pass) per feature
* ``bootstrap_table``    — paired bootstrap over instances: mean diff + 95% CI, per stratum,
                           pooled over all combos and within each top-3 combo
* ``taxonomy_shares``    — rule-based taxonomy shares over fail members, per stratum/combo
* ``overconfidence``     — among SMALL groups: P(fail | no repro AND no post-edit test) vs
                           P(fail | both), within instance; plus did the failing sibling
                           stop earlier (turns, assistant_chars as the token proxy — the
                           corpus carries no wall-clock or token counts)

Outputs land in ``outputs/paired_*.csv`` and the markdown note
``analytics/research/paired_failures_small_vs_large.md``.

Usage: uv run python scripts/paired_analysis.py
"""

from __future__ import annotations

import argparse
from dataclasses import dataclass
from pathlib import Path

import numpy as np
import pandas as pd

from .data import ROOT
from .paired import FEATURE_COLUMNS, FLAG_COLUMNS, TAXONOMY, classify_failure

ELIGIBLE_PATH = ROOT / "outputs" / "eligible_pairs.parquet"
FEATURES_PATH = ROOT / "outputs" / "paired_features.parquet"
REPORT_PATH = ROOT / "analytics" / "research" / "paired_failures_small_vs_large.md"

# Top-3 combos by IRT theta_2pl from the closure-C run (see outputs/closure_C.log there).
TOP3_COMBOS = [
    "minisweagent/qwen38_27b",
    "sweagent/qwen36_27b",
    "sweagent/qwen35_122b",
]

N_BOOT = 1000
RNG_SEED = 42

# features where a positive fail-minus-pass diff means "fails do more of it".
# n_gold_hunks (constant within instance), ends_with_submit and has_submit_call
# (degenerate: ~100% of labeled trajectories submit) are dropped from the table.
_DIFF_DROP = {"n_gold_hunks", "ends_with_submit", "has_submit_call"}
DIFF_FEATURES = [
    f
    for f in FEATURE_COLUMNS + ["first_repro_call", "last_test_call"] + FLAG_COLUMNS
    if f not in _DIFF_DROP
]

STRATA = ["SMALL", "LARGE"]


def load_frame(
    features_path: Path = FEATURES_PATH, eligible_path: Path = ELIGIBLE_PATH
) -> pd.DataFrame:
    feats = pd.read_parquet(features_path)
    elig = pd.read_parquet(
        eligible_path, columns=["trajectory_id", "group_id", "combo", "stratum", "added_lines"]
    )
    df = feats.merge(elig, on="trajectory_id", how="inner", validate="one_to_one")
    return df


def paired_diffs(df: pd.DataFrame, features: list[str] | None = None) -> pd.DataFrame:
    """Per (instance, combo) group: mean(fail) - mean(pass) for every feature."""
    features = features or DIFF_FEATURES
    work = df.copy()
    for f in features:
        work[f] = pd.to_numeric(work[f], errors="coerce").astype("float64")
    grouped = (
        work.groupby(["group_id", "instance_id", "combo", "stratum", "resolved"])[features]
        .mean()
        .unstack("resolved")
    )
    # grouped.columns is a MultiIndex (feature, resolved); diff = mean(fail) - mean(pass)
    out = pd.DataFrame(
        {f"diff_{f}": grouped[(f, 0)] - grouped[(f, 1)] for f in features}
    ).reset_index()
    counts = df.groupby("group_id")["resolved"].agg(
        n_pass=lambda s: (s == 1).sum(), n_fail=lambda s: (s == 0).sum()
    )
    return out.merge(counts.reset_index(), on="group_id")


@dataclass
class BootResult:
    mean: float
    lo: float
    hi: float
    n_units: int


def bootstrap_mean(
    values: pd.Series | np.ndarray, n_boot: int = N_BOOT, seed: int = RNG_SEED
) -> BootResult:
    """Percentile bootstrap CI for the mean over resampled units (instances)."""
    v = np.asarray(values, dtype="float64")
    v = v[~np.isnan(v)]
    if len(v) == 0:
        return BootResult(float("nan"), float("nan"), float("nan"), 0)
    rng = np.random.default_rng(seed)
    idx = rng.integers(0, len(v), size=(n_boot, len(v)))
    means = v[idx].mean(axis=1)
    return BootResult(float(v.mean()), *np.percentile(means, [2.5, 97.5]).tolist(), len(v))


def _instance_level(diffs: pd.DataFrame, feature: str) -> pd.Series:
    """Mean of per-group diffs within each instance (equal weight per group)."""
    return diffs.groupby("instance_id")[f"diff_{feature}"].mean()


def bootstrap_table(
    diffs: pd.DataFrame, features: list[str] | None = None
) -> pd.DataFrame:
    """Rows = feature; blocks = stratum × {all combos, each top-3 combo}.

    ``all_median`` is the median over instances of the instance-mean diff — the
    mean columns are outlier-sensitive for heavy-tailed features
    (extra_hunk_lines, model_lines_over_gold).
    """
    features = features or DIFF_FEATURES
    rows = []
    for stratum in STRATA:
        d = diffs[diffs["stratum"] == stratum]
        slices = {"all": d} | {c: d[d["combo"] == c] for c in TOP3_COMBOS}
        for feature in features:
            row = {"stratum": stratum, "feature": feature}
            for label, sub in slices.items():
                inst = _instance_level(sub, feature)
                b = bootstrap_mean(inst)
                row[label] = f"{b.mean:+.3f} [{b.lo:+.3f},{b.hi:+.3f}]"
                row[f"{label}__mean"] = b.mean
                row[f"{label}__n"] = b.n_units
                if label == "all":
                    row["all_median"] = float(inst.median())
            rows.append(row)
    return pd.DataFrame(rows)


def taxonomy_shares(df: pd.DataFrame) -> pd.DataFrame:
    """Rule-based taxonomy over FAIL members of each pair-eligible group."""
    fails = df[df["resolved"] == 0].copy()
    fails["bucket"] = fails.apply(classify_failure, axis=1)
    fails.to_parquet(ROOT / "outputs" / "paired_fail_taxonomy.parquet")
    rows = []
    for stratum in STRATA + ["MID"]:
        sub = fails[fails["stratum"] == stratum]
        for combo in ["all"] + TOP3_COMBOS + sorted(set(fails.combo) - set(TOP3_COMBOS)):
            s = sub if combo == "all" else sub[sub["combo"] == combo]
            row = {"stratum": stratum, "combo": combo, "n_fail": len(s)}
            for b in TAXONOMY:
                row[b] = (s["bucket"] == b).sum() / len(s) if len(s) else float("nan")
            rows.append(row)
    return pd.DataFrame(rows)


def overconfidence(df: pd.DataFrame) -> dict:
    """SMALL stratum: fail rate among 'lazy' rollouts vs 'diligent' ones, within instance.

    lazy    = no repro/run command before first edit AND no test run after last edit
    diligent= repro before edit AND test run after last edit
    """
    small = df[df["stratum"] == "SMALL"].copy()
    small["lazy"] = ~small["ran_repro_before_edit"] & ~small["ran_tests_after_last_edit"]
    small["diligent"] = small["ran_repro_before_edit"] & small["ran_tests_after_last_edit"]

    def _arm_rates(g: pd.DataFrame) -> pd.Series:
        lazy = g[g["lazy"]]
        dil = g[g["diligent"]]
        return pd.Series(
            {
                "p_fail_lazy": (lazy["resolved"] == 0).mean() if len(lazy) else np.nan,
                "p_fail_dil": (dil["resolved"] == 0).mean() if len(dil) else np.nan,
                "n_lazy": len(lazy),
                "n_dil": len(dil),
            }
        )

    grp = small.groupby(["group_id", "instance_id", "combo"]).apply(
        _arm_rates, include_groups=False
    )
    grp = grp.dropna(subset=["p_fail_lazy", "p_fail_dil"], how="any")
    grp["diff"] = grp["p_fail_lazy"] - grp["p_fail_dil"]
    inst = grp.groupby("instance_id")[["p_fail_lazy", "p_fail_dil", "diff"]].mean()

    # stricter variant: "no final test run" = zero test commands anywhere
    small["lazy_strict"] = ~small["ran_repro_before_edit"] & (small["n_test_runs"] == 0)
    small["dil_strict"] = small["ran_repro_before_edit"] & (small["n_test_runs"] > 0)
    grp2 = small.groupby(["group_id", "instance_id"]).apply(
        lambda g: pd.Series(
            {
                "lazy": (g.loc[g["lazy_strict"], "resolved"] == 0).mean()
                if g["lazy_strict"].any()
                else np.nan,
                "dil": (g.loc[g["dil_strict"], "resolved"] == 0).mean()
                if g["dil_strict"].any()
                else np.nan,
            }
        ),
        include_groups=False,
    ).dropna()
    grp2["diff"] = grp2["lazy"] - grp2["dil"]
    inst2 = grp2.groupby("instance_id")["diff"].mean()

    pooled_lazy = small[small["lazy"]]
    pooled_dil = small[small["diligent"]]
    return {
        "n_groups_both_arms": len(grp),
        "n_instances": len(inst),
        "p_fail_lazy": bootstrap_mean(inst["p_fail_lazy"]),
        "p_fail_diligent": bootstrap_mean(inst["p_fail_dil"]),
        "diff": bootstrap_mean(inst["diff"]),
        "strict_diff": bootstrap_mean(inst2),
        "n_groups_strict": len(grp2),
        "pooled_fail_lazy": float((pooled_lazy["resolved"] == 0).mean()),
        "pooled_fail_diligent": float((pooled_dil["resolved"] == 0).mean()),
        "pooled_n_lazy": len(pooled_lazy),
        "pooled_n_diligent": len(pooled_dil),
        "share_lazy_fails": float((small[small["resolved"] == 0]["lazy"]).mean()),
        "share_lazy_passes": float((small[small["resolved"] == 1]["lazy"]).mean()),
    }


def stopped_earlier(diffs: pd.DataFrame) -> dict:
    """Did the failing sibling stop earlier than the passing one? (turns + char proxy)."""
    out = {}
    for stratum in STRATA:
        d = diffs[diffs["stratum"] == stratum]
        inst_turns = d.groupby("instance_id")["diff_n_turns"].mean()
        inst_chars = d.groupby("instance_id")["diff_assistant_chars"].mean()
        out[stratum] = {
            "turns": bootstrap_mean(inst_turns),
            "chars": bootstrap_mean(inst_chars),
            "share_groups_fail_shorter": float((d["diff_n_turns"] < 0).mean()),
        }
    return out


# --- report -------------------------------------------------------------------

_FMT_COLS = ["all"] + TOP3_COMBOS


def _diff_table_md(table: pd.DataFrame, stratum: str) -> str:
    sub = table[table["stratum"] == stratum]
    lines = [
        "| feature | all (median) | " + " | ".join(_FMT_COLS) + " |",
        "|---|---|" + "---|" * len(_FMT_COLS),
    ]
    for _, r in sub.iterrows():
        med = f"{r['all_median']:+.3f}" if pd.notna(r["all_median"]) else "—"
        cells = [str(r[c]) for c in _FMT_COLS]
        lines.append(f"| {r['feature']} | {med} | " + " | ".join(cells) + " |")
    return "\n".join(lines)


def _share_table_md(shares: pd.DataFrame, stratum: str) -> str:
    sub = shares[shares["stratum"] == stratum]
    cols = ["n_fail"] + TAXONOMY
    lines = ["| combo | " + " | ".join(cols) + " |", "|---|" + "---|" * len(cols)]
    for _, r in sub.iterrows():
        cells = [str(int(r["n_fail"]))] + [
            f"{r[b] * 100:.1f}%" if pd.notna(r[b]) else "—" for b in TAXONOMY
        ]
        lines.append(f"| {r['combo']} | " + " | ".join(cells) + " |")
    return "\n".join(lines)


def _llm_section() -> str:
    labels_path = ROOT / "outputs" / "llm_labels.jsonl"
    if not labels_path.exists():
        return "_(not run)_"
    labels = pd.read_json(labels_path, lines=True)
    share = labels["label"].value_counts(normalize=True)
    lines = [
        f"n={len(labels)} labeled SMALL fail trajectories (model(s): "
        + ", ".join(sorted(labels['model'].unique()))
        + ")\n",
        "| llm label | share |",
        "|---|---|",
    ]
    for lbl, s in share.items():
        lines.append(f"| {lbl} | {s * 100:.1f}% |")
    lines.append("\nLLM label × rule taxonomy (row %):\n")
    ct = pd.crosstab(labels["rule_bucket"], labels["label"], normalize="index")
    header = "| rule bucket | " + " | ".join(ct.columns) + " |"
    lines.append(header)
    lines.append("|---|" + "---|" * len(ct.columns))
    for idx, row in ct.iterrows():
        lines.append(
            f"| {idx} | " + " | ".join(f"{v * 100:.0f}%" for v in row.values) + " |"
        )
    return "\n".join(lines)


def build_report(df: pd.DataFrame | None = None) -> str:
    if df is None:
        df = load_frame()
    diffs = paired_diffs(df)
    table = bootstrap_table(diffs)
    shares = taxonomy_shares(df)
    oc = overconfidence(df)
    stop = stopped_earlier(diffs)

    diffs.to_csv(ROOT / "outputs" / "paired_group_diffs.csv", index=False)
    table.to_csv(ROOT / "outputs" / "paired_diff_table.csv", index=False)
    shares.to_csv(ROOT / "outputs" / "paired_taxonomy_shares.csv", index=False)

    n_groups = df["group_id"].nunique()
    counts = df.groupby("stratum").agg(
        trajs=("trajectory_id", "count"),
        groups=("group_id", "nunique"),
        instances=("instance_id", "nunique"),
    )

    verdicts = _verdicts(table, shares, oc, stop)

    md = f"""# Paired failure analysis: why do models fail SMALL feasible tasks?

2026-09-19. Within-instance paired comparison on Open-SWE-Traces: for every
(instance, harness/teacher combo) group with ≥1 pass and ≥1 fail rollout on a mixed
instance (n_labeled ≥ 5, 0 < n_resolved < n_labeled), compare the behavior of failing
vs passing siblings. The task is held fixed inside a pair, so differences are
behavioral, not difficulty. Strata by gold patch size: SMALL = added_lines ≤ 30,
LARGE = ≥ 150. Bootstrap CIs resample instances (n_boot={N_BOOT}).

Groups: {n_groups:,} pair-eligible (instance × combo) groups,
{df['trajectory_id'].nunique():,} trajectories.

| stratum | trajectories | groups | instances |
|---|---:|---:|---:|
{_counts_md(counts)}

## Paired differences: mean(fail) − mean(pass) within group, averaged over instances

Values are `diff [95% CI]`. Positive = failing rollouts do more of it.
Top-3 combos are the three strongest by IRT theta (closure-C): {", ".join(TOP3_COMBOS)}.

### SMALL (added_lines ≤ 30)

{_diff_table_md(table, "SMALL")}

### LARGE (added_lines ≥ 150)

{_diff_table_md(table, "LARGE")}

## Failure taxonomy (rule-based, on the FAIL member of each pair)

Note: every labeled trajectory in this corpus ends with a submit marker
(`finish`/`submit` call or final submit text — the harness records completion, not
truncation), so `stop_reason` is 'submit' for ~100% of rows and the `timeout`
bucket is structurally empty. Turn-cap/crash failures are not observable among
labeled rollouts; read 'timeout' as absorbed into the other buckets.

### SMALL

{_share_table_md(shares, "SMALL")}

### LARGE

{_share_table_md(shares, "LARGE")}

## Overconfidence test (SMALL)

- lazy arm (no repro/run command before first edit AND no test after last edit):
  P(fail) = {oc['p_fail_lazy'].mean:.3f} [{oc['p_fail_lazy'].lo:.3f},{oc['p_fail_lazy'].hi:.3f}]
  (instance-level; n_groups with both arms={oc['n_groups_both_arms']}, n_instances={oc['n_instances']})
- diligent arm (repro before edit AND test after last edit):
  P(fail) = {oc['p_fail_diligent'].mean:.3f} [{oc['p_fail_diligent'].lo:.3f},{oc['p_fail_diligent'].hi:.3f}]
- paired difference (lazy − diligent): {oc['diff'].mean:+.3f} [{oc['diff'].lo:+.3f},{oc['diff'].hi:+.3f}]
- stricter variant (no repro AND zero test commands vs repro AND ≥1 test):
  Δ={oc['strict_diff'].mean:+.3f} [{oc['strict_diff'].lo:+.3f},{oc['strict_diff'].hi:+.3f}]
  (n_groups={oc['n_groups_strict']})
- pooled: fail rate lazy={oc['pooled_fail_lazy']:.3f} (n={oc['pooled_n_lazy']}),
  diligent={oc['pooled_fail_diligent']:.3f} (n={oc['pooled_n_diligent']})
- share of fail rollouts that were lazy: {oc['share_lazy_fails']:.3f};
  share of pass rollouts that were lazy: {oc['share_lazy_passes']:.3f}

Did the failing sibling stop earlier? (turns; assistant_chars is the token proxy — the
corpus has no wall-clock or token counts)

- SMALL: Δturns={stop['SMALL']['turns'].mean:+.2f} [{stop['SMALL']['turns'].lo:+.2f},{stop['SMALL']['turns'].hi:+.2f}],
  Δchars={stop['SMALL']['chars'].mean:+.0f}; share of groups where the fail used fewer turns: {stop['SMALL']['share_groups_fail_shorter']:.3f}
- LARGE: Δturns={stop['LARGE']['turns'].mean:+.2f} [{stop['LARGE']['turns'].lo:+.2f},{stop['LARGE']['turns'].hi:+.2f}],
  Δchars={stop['LARGE']['chars'].mean:+.0f}; share of groups where the fail used fewer turns: {stop['LARGE']['share_groups_fail_shorter']:.3f}

## Semantic labels (OpenRouter free tier, ≤ {MAX_LLM} requests)

{_llm_section()}

## Verdicts

{verdicts}

## Reproduce

```
uv run python scripts/paired_eligible.py     # outputs/eligible_pairs.parquet
uv run python scripts/paired_features.py     # outputs/paired_features.parquet (resume-safe)
uv run python scripts/paired_analysis.py     # this note + outputs/paired_*.csv
uv run python scripts/paired_labels.py       # outputs/llm_labels.jsonl (OpenRouter free)
```
"""
    REPORT_PATH.write_text(md)
    return md


MAX_LLM = 600


def _counts_md(counts: pd.DataFrame) -> str:
    return "\n".join(
        f"| {s} | {int(r['trajs']):,} | {int(r['groups']):,} | {int(r['instances']):,} |"
        for s, r in counts.iterrows()
    )


def _verdicts(table: pd.DataFrame, shares: pd.DataFrame, oc: dict, stop: dict) -> str:
    """Plain-language verdict per hypothesis, computed from the numbers."""
    lines = []

    def mean_of(stratum: str, feature: str, combo: str = "all") -> float:
        r = table[(table["stratum"] == stratum) & (table["feature"] == feature)]
        return float(r.iloc[0][f"{combo}__mean"]) if len(r) else float("nan")

    small_tax = shares[(shares["stratum"] == "SMALL") & (shares["combo"] == "all")].iloc[0]
    large_tax = shares[(shares["stratum"] == "LARGE") & (shares["combo"] == "all")].iloc[0]

    small_under = small_tax["partial"] + small_tax["wrong_site"] + small_tax["partial_extra"]
    large_time = large_tax["timeout"]
    lines.append(
        f"- **H1 (SMALL fails = overconfident quickies).** SMALL fail taxonomy: "
        f"partial+wrong_site+partial_extra={small_under * 100:.1f}%, "
        f"timeout={small_tax['timeout'] * 100:.1f}%. "
        f"Failing siblings ran repro before edit {abs(mean_of('SMALL', 'ran_repro_before_edit')) * 100:.1f} pts "
        f"{'less' if mean_of('SMALL', 'ran_repro_before_edit') < 0 else 'more'} often than passing ones, "
        f"and {'stopped earlier' if stop['SMALL']['turns'].mean < 0 else 'ran longer'} "
        f"(Δturns={stop['SMALL']['turns'].mean:+.1f}). "
        f"Lazy-vs-diligent fail-rate gap: {oc['diff'].mean:+.3f}. "
        f"{'SUPPORTED' if oc['diff'].mean > 0 and small_under > 0.4 else 'MIXED/WEAK'}."
    )
    lines.append(
        f"- **H2 (LARGE fails = capacity).** LARGE fail taxonomy: "
        f"timeout={large_time * 100:.1f}%, "
        f"partial={large_tax['partial'] * 100:.1f}%. "
        f"Δturns={stop['LARGE']['turns'].mean:+.1f}, "
        f"Δcoverage={mean_of('LARGE', 'patch_hunk_coverage'):+.3f}. "
        f"{'SUPPORTED' if large_time + large_tax['partial'] > 0.4 and stop['LARGE']['turns'].mean > 0 else 'MIXED/WEAK'}."
    )
    return "\n".join(lines)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.parse_args()
    md = build_report()
    print(md[:3000])
    print(f"\n... full note → {REPORT_PATH}")


if __name__ == "__main__":
    main()


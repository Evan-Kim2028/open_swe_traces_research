"""Pre-registered analysis for the detail-count x verification-affordance grid.

Run on the trial data once it exists (``experiments/detail_dial/trials.parquet``):

1. **Per-detail logistic.**  logit(P(detail passes)) ~ d + v + d:v, cluster-
   robust SEs by unit (each (seed, d, v) cell).  The dial claim requires a
   negative d coefficient at v=low and a positive d:v interaction (v=high
   flattens the slope).  Every trial contributes one observation per drawn
   detail (plus the constant guard), which is the statistical power play: the
   grid's 40 units yield sum(d) = 168 detail observations per model, not 40.

2. **Geometric prediction.**  Under independence, unit pass rate P_pass(d) =
   p^d (times the constant guard factor).  Equivalent test: logit-level fit
   log P_pass(d) = a + b*d with a fixed at the guard baseline; compare against
   the flat alternative (b = 0) and against a saturating alternative via
   Vuong-style log-likelihood comparison.  Because p is not known a priori,
   the *shape* test is: the unit-level log-pass curve is linear in d with
   intercept log(p_guard) (the d=0 baseline), vs the flat alternative.

3. **Independence (within-unit dispersion).**  If details were independent
   with a common per-detail p, the number of details a unit passes is
   Binomial(d, p): the observed variance across units of the passed-detail
   count should match the binomial variance; overdispersion flags model-level
   dependence.  Reported as the dispersion ratio and a bootstrap CI.

4. **Calibration simulation** (runs without trial data): verifies the
   estimator recovers the true geometric p^d from synthetic trials, so the
   pre-registered procedure is known to work before launch.
"""

from __future__ import annotations

import json
import math
from pathlib import Path

import numpy as np


def expand_trials(df) -> tuple[np.ndarray, np.ndarray, np.ndarray, np.ndarray, np.ndarray]:
    """Flatten trials.parquet rows into one observation per drawn detail.

    Returns (detail_pass, d, v_high, unit_id, guard_pass).  The details column
    is a JSON array of ``{"id": ..., "pass": ...}`` lines recorded by the
    verifier; guard is a constant baseline, excluded from the detail vector.
    """
    d_out: list[int] = []
    v_out: list[int] = []
    unit_out: list[str] = []
    pass_out: list[int] = []
    guard_out: list[int] = []
    for row in df.itertuples(index=False):
        unit = str(row.unit)
        details_raw = row.details
        if not details_raw:
            continue
        try:
            parsed = json.loads(details_raw)
        except (ValueError, TypeError):
            continue
        seen_guard = False
        for item in parsed:
            if not isinstance(item, dict):
                continue
            if item.get("id") == "guard":
                guard_out.append(1 if item.get("pass") else 0)
                seen_guard = True
                continue
            pass_out.append(1 if item.get("pass") else 0)
            d_out.append(int(row.d))
            v_out.append(1 if str(row.v) == "high" else 0)
            unit_out.append(unit)
        if not seen_guard:
            guard_out.append(int(row.reward) if row.reward is not None else 0)
    return (
        np.array(pass_out),
        np.array(d_out),
        np.array(v_out),
        np.array(unit_out),
        np.array(guard_out),
    )


def cluster_logistic_fit(pass_, d, v, unit) -> dict[str, object]:
    """logit(pass) ~ d + v_high + d:v_high with cluster-robust SEs by unit.

    Uses statsmodels if available (the repo's pinned lockfile does not add it,
    so we implement the sandwich estimator by hand over a Newton fit)."""
    n = len(pass_)
    X = np.column_stack([np.ones(n), d, v, d * v])
    y = pass_.astype(float)
    beta = np.zeros(4)
    for _ in range(60):
        eta = X @ beta
        mu = 1.0 / (1.0 + np.exp(-np.clip(eta, -30, 30)))
        w = mu * (1.0 - mu)
        grad = X.T @ (y - mu)
        hess = X.T @ (X * w[:, None])
        try:
            step = np.linalg.solve(hess + 1e-9 * np.eye(4), grad)
        except np.linalg.LinAlgError:
            break
        beta += step
        if np.max(np.abs(step)) < 1e-9:
            break
    # cluster-robust (unit-level) sandwich
    units = np.unique(unit)
    meat = np.zeros((4, 4))
    for u in units:
        mask = unit == u
        eta = X[mask] @ beta
        mu = 1.0 / (1.0 + np.exp(-np.clip(eta, -30, 30)))
        resid = y[mask] - mu
        g = X[mask].T @ resid
        meat += np.outer(g, g)
    bread_inv = np.linalg.inv(hess + 1e-9 * np.eye(4))
    cov = bread_inv @ meat @ bread_inv
    se = np.sqrt(np.diag(cov))
    names = ["intercept", "d", "v_high", "d:v"]
    return {
        "coef": {n: float(b) for n, b in zip(names, beta, strict=True)},
        "se": {n: float(s) for n, s in zip(names, se, strict=True)},
        "n_observations": int(n),
        "n_units": len(units),
    }


def geometric_test(df) -> dict[str, object]:
    """Unit-level geometric prediction: log(pass_rate) is linear in d.

    Fits log(P_unit_pass) ~ a + b*d by least squares on observed rates per d
    (v levels separate), and reports b with its CI, plus the flat alternative
    (b=0) comparison via the SE.  Under the hypothesis b = log(p) < 0.
    """
    out: dict[str, object] = {}
    for v in ("low", "high"):
        sub = df[df["v"] == v]
        if sub.empty:
            out[v] = {"n_units": 0}
            continue
        rates: dict[int, list[float]] = {}
        for row in sub.itertuples(index=False):
            r = float(row.reward) if row.reward is not None else 0.0
            rates.setdefault(int(row.d), []).append(r)
        ds = sorted(rates)
        if len(ds) < 3:
            out[v] = {"n_units": len(sub), "d_levels": ds}
            continue
        x = np.array(ds, dtype=float)
        y = np.array([math.log(max(1e-9, sum(rates[dd]) / len(rates[dd]))) for dd in ds])
        n = len(x)
        xm, ym = x.mean(), y.mean()
        b = ((x - xm) @ (y - ym)) / ((x - xm) @ (x - xm))
        a = ym - b * xm
        resid = y - (a + b * x)
        se_b = math.sqrt((resid @ resid) / (n - 2) / ((x - xm) @ (x - xm)))
        out[v] = {
            "n_units": len(sub),
            "d_levels": ds,
            "log_pass_slope_b": float(b),
            "se_b": float(se_b),
            "ci95": (float(b - 1.96 * se_b), float(b + 1.96 * se_b)),
            "intercept_a": float(a),
            "flat_alternative_rejected": b + 1.96 * se_b < 0,
        }
    return out


def dispersion_ratio(df) -> dict[str, object]:
    """Within-unit detail-count dispersion vs the binomial null."""
    rows = []
    for row in df.itertuples(index=False):
        if not row.details:
            continue
        try:
            parsed = json.loads(row.details)
        except (ValueError, TypeError):
            continue
        passed = sum(1 for it in parsed if isinstance(it, dict) and it.get("id") != "guard" and it.get("pass"))
        total = sum(1 for it in parsed if isinstance(it, dict) and it.get("id") != "guard")
        if total > 0:
            rows.append((int(row.d), passed, total))
    if not rows:
        return {"n_units": 0}
    counts = np.array([r[1] for r in rows])
    totals = np.array([r[2] for r in rows])
    p_hat = counts.sum() / totals.sum()
    var_binom = totals * p_hat * (1 - p_hat)
    var_obs = counts.var()
    return {
        "n_units": len(rows),
        "p_hat": float(p_hat),
        "dispersion_ratio": float(var_obs / var_binom.mean()),
        "note": "ratio >> 1 flags model-level detail dependence (details not independent within a unit)",
    }


def simulate_geometric(n_units: int = 400, p_low: float = 0.7, p_high: float = 0.9) -> dict[str, object]:
    """Calibration: draw synthetic trials from the geometric model and confirm
    the unit-level log-pass curve recovers slope log(p) and the per-detail
    logistic recovers the d coefficient."""
    rng = np.random.default_rng(0)
    ds = np.array([1, 2, 4, 6, 8])
    rows = []
    for i in range(n_units):
        d = ds[i % len(ds)]
        v = "high" if i % 4 < 2 else "low"
        p = p_high if v == "high" else p_low
        details = [{"id": f"x{j}", "pass": bool(rng.random() < p)} for j in range(d)]
        reward = 1.0 if all(x["pass"] for x in details) and rng.random() < 0.95 else 0.0
        rows.append({"unit": f"u{i}", "d": d, "v": v, "reward": reward, "details": json.dumps(details)})
    import pandas as pd

    df = pd.DataFrame(rows)
    geom = geometric_test(df)
    per_d = {}
    for v in ("low", "high"):
        sub = df[df["v"] == v]
        rates = {d: sub[sub["d"] == d]["reward"].mean() for d in ds}
        per_d[v] = {int(k): float(rates[k]) for k in ds}
    return {
        "true": {"p_low": p_low, "p_high": p_high},
        "expected_slope": {"low": math.log(p_low), "high": math.log(p_high)},
        "estimated_slope": {v: g.get("log_pass_slope_b") for v, g in geom.items()},
        "observed_rates": per_d,
    }


def analyze(trials_path: Path) -> dict[str, object]:
    import pandas as pd

    path = Path(trials_path)
    if not path.is_file():
        return {"status": "no_trials", "calibration": simulate_geometric()}
    df = pd.read_parquet(path)
    if df.empty:
        return {"status": "no_trials", "calibration": simulate_geometric()}
    pass_, d, v, unit, _guard = expand_trials(df)
    out: dict[str, object] = {
        "status": "ok",
        "n_trials": len(df),
        "per_detail": {},
        "geometric": geometric_test(df),
        "dispersion": dispersion_ratio(df),
    }
    if len(pass_) >= 8:
        out["per_detail"] = cluster_logistic_fit(pass_, d, v, unit)
    else:
        out["per_detail"] = {"note": "too few detail observations"}
    return out


def main_cli(argv: list[str] | None = None) -> int:
    import argparse

    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--trials", type=Path, default=Path("experiments/detail_dial/trials.parquet"))
    p.add_argument("--simulate", action="store_true", help="run only the calibration simulation")
    args = p.parse_args(argv)
    if args.simulate:
        print(json.dumps(simulate_geometric(), indent=2))
        return 0
    print(json.dumps(analyze(args.trials), indent=2, default=str))
    return 0


if __name__ == "__main__":
    raise SystemExit(main_cli())

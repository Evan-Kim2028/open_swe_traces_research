"""Summarize the four curve SFT arms from their pulled Kaggle out dirs.

Reads ``experiments/curve/arms/<arm>/out/metrics.json`` (+ ``metrics_rank1.json`` and the
raw kernel log for the encode lines) and ``experiments/curve/data/manifests_summary.json``,
then prints markdown tables (run config, train loss, aligned eval CE, slopes, pairwise
deltas, supervised-token efficiency) and writes the log-x eval-CE figure to
``analytics/research/figures/curve_run1_eval_ce.png``.

Notes on metric semantics (see ``experiments/curve/train_curve.py``):

- ``metrics.json`` log records hold per-rank ``tok_per_s``/``sup_tok_per_s`` (the top-level
  snapshot multiplies by ``world_size``); global values are ``log * world_size``.
- ``sup_tok_per_s * elapsed_s`` is a cumulative-average estimate of supervised tokens seen.
- ``n_traces`` in metrics is rank 0's encodable count times ``world_size``, so it
  over-reports arms where ranks dropped different numbers of traces; the per-rank
  ``encoded X/Y`` log lines are the ground truth.

Usage:
  uv run python scripts/curve_report.py
  uv run python scripts/curve_report.py --tables-out analytics/research/curve_run1_tables.md
"""

from __future__ import annotations

import argparse
import json
import math
import re
from pathlib import Path

import matplotlib

matplotlib.use("Agg")

import matplotlib.pyplot as plt
import pandas as pd
from rich.console import Console

from .data import ROOT

CURVE_DIR = ROOT / "experiments" / "curve"
ARMS = ("random", "top_within_task", "bottom_within_task", "random_masked")
FIGURE_PATH = ROOT / "analytics" / "research" / "figures" / "curve_run1_eval_ce.png"
ENCODE_RE = re.compile(r"encoded (\d+)/(\d+) traces")

console = Console()


def load_run(arm: str, curve_dir: Path = CURVE_DIR) -> dict | None:
    """One arm's metrics (rank 0), rank-1 record, and its parsed kernel log.

    Returns ``None`` when the arm's out dir or ``metrics.json`` is missing, so a partial
    sweep can still be summarized.
    """
    out = curve_dir / "arms" / arm / "out"
    metrics_path = out / "metrics.json"
    if not metrics_path.exists():
        return None
    run = {
        "arm": arm,
        "metrics": json.loads(metrics_path.read_text()),
        "rank1": None,
        "encoded": None,
        "dropped": None,
    }
    rank1_path = out / "metrics_rank1.json"
    if rank1_path.exists():
        run["rank1"] = json.loads(rank1_path.read_text())
    logs = sorted(out.glob("*.log"))
    if logs:
        run["encoded"], run["dropped"] = parse_encode_lines(logs[0])
    return run


def parse_encode_lines(path: Path) -> tuple[list[tuple[int, int, int]], int]:
    """Per-rank ``encoded X/Y`` from the raw kernel log -> ``([(rank, X, Y)...], dropped)``."""
    entries = json.loads(path.read_text())
    hits = []
    for entry in entries:
        match = ENCODE_RE.search(entry.get("data", ""))
        if match:
            rank = re.search(r"\[rank (\d+)\]", entry["data"])
            hits.append(
                (
                    int(rank.group(1)) if rank else -1,
                    int(match.group(1)),
                    int(match.group(2)),
                )
            )
    dropped = sum(y - x for _, x, y in hits)
    return hits, dropped


def load_manifests(curve_dir: Path = CURVE_DIR) -> dict:
    """The manifest build summary (single JSON object written by the builder)."""
    return json.loads((curve_dir / "data" / "manifests_summary.json").read_text())


def global_rate(run: dict, record: dict, key: str) -> float:
    """Per-rank log rate -> global rate (the top-level snapshot multiplies by world_size)."""
    return record[key] * int(run["metrics"]["world_size"])


def loss_table(runs: dict[str, dict], every: int = 20) -> pd.DataFrame:
    """Train loss at steps ``every, 2*every, ...`` per arm (NaN where the run ended early)."""
    steps = list(range(every, 200, every))
    rows = {}
    for arm, run in runs.items():
        by_step = {rec["step"]: rec["loss"] for rec in run["metrics"]["log"]}
        rows[arm] = [by_step.get(s, float("nan")) for s in steps]
    return pd.DataFrame(rows, index=pd.Index(steps, name="step"))


def eval_table(runs: dict[str, dict]) -> pd.DataFrame:
    """Aligned eval events: one row per event index, one CE column per arm.

    The four arms share the window milestones (1008 / 1600 / 2000) and the final event, so
    event index is the alignment key; the per-arm step and windows_seen are kept beside the
    CE when they differ across arms.
    """
    per_arm = {arm: run["metrics"]["eval"] for arm, run in runs.items()}
    n = min(len(ev) for ev in per_arm.values())
    rows = []
    for i in range(n):
        row = {"event": per_arm[next(iter(per_arm))][i]["event"]}
        windows = {arm: ev[i]["windows_seen"] for arm, ev in per_arm.items()}
        row["windows_seen"] = max(windows.values()) if len(set(windows.values())) == 1 else None
        for arm in runs:
            row[arm] = per_arm[arm][i]["eval_ce"]
        rows.append(row)
    return pd.DataFrame(rows)


def final_table(runs: dict[str, dict]) -> pd.DataFrame:
    """The final_full event per arm: step, windows_seen, CE, and elapsed/steps context."""
    rows = []
    for arm, run in runs.items():
        m = run["metrics"]
        ev = m["eval"][-1]
        rows.append(
            {
                "arm": arm,
                "steps_done": m["steps_done"],
                "windows_seen": m["windows_seen"],
                "final_ce": ev["eval_ce"],
                "elapsed_s": m["elapsed_s"],
                "stop_reason": m["stop_reason"],
            }
        )
    return pd.DataFrame(rows).set_index("arm")


def sup_tokens_seen(run: dict, step: int) -> float | None:
    """Global supervised tokens seen by ``step`` (cumulative per-rank rate x elapsed x world)."""
    records = run["metrics"]["log"]
    upto = [rec for rec in records if rec["step"] <= step]
    if not upto:
        return None
    rec = upto[-1]
    return global_rate(run, rec, "sup_tok_per_s") * rec["elapsed_s"]


def efficiency_table(runs: dict[str, dict]) -> pd.DataFrame:
    """Per eval event: CE, supervised tokens seen, and windows (the differing dose axis)."""
    rows = []
    for arm, run in runs.items():
        for ev in run["metrics"]["eval"]:
            sup = sup_tokens_seen(run, ev["step"])
            rows.append(
                {
                    "arm": arm,
                    "event": ev["event"],
                    "step": ev["step"],
                    "windows_seen": ev["windows_seen"],
                    "ce": ev["eval_ce"],
                    "sup_tokens_seen": sup,
                }
            )
    return pd.DataFrame(rows)


def _ols_log_windows(points: list[tuple[float, float]]) -> tuple[float, float, list[float]]:
    """OLS of CE on ln(windows); returns (intercept, slope, residuals)."""
    xs = [math.log(x) for x, _ in points]
    ys = [y for _, y in points]
    n = len(xs)
    mx = sum(xs) / n
    my = sum(ys) / n
    slope = sum((x - mx) * (y - my) for x, y in zip(xs, ys)) / sum((x - mx) ** 2 for x in xs)
    intercept = my - slope * mx
    residuals = [y - (intercept + slope * x) for x, y in zip(xs, ys)]
    return intercept, slope, residuals


def slope_table(runs: dict[str, dict]) -> pd.DataFrame:
    """Per-arm fit of the in-run (small-eval) points to CE = a + b ln(windows)."""
    rows = []
    for arm, run in runs.items():
        points = [
            (ev["windows_seen"], ev["eval_ce"])
            for ev in run["metrics"]["eval"]
            if ev["event"] != "final_full"
        ]
        intercept, slope, residuals = _ols_log_windows(points)
        dof = len(points) - 2
        resid_sd = math.sqrt(sum(r * r for r in residuals) / dof) if dof > 0 else float("nan")
        rows.append(
            {
                "arm": arm,
                "n_points": len(points),
                "a": intercept,
                "b_per_ln": slope,
                "drop_per_doubling": -slope * math.log(2),
                "resid_sd": resid_sd,
            }
        )
    return pd.DataFrame(rows).set_index("arm")


def common_slope(runs: dict[str, dict]) -> dict:
    """Fit the in-run points of all arms with one shared slope and per-arm intercepts.

    The pooled residual sd is the noise scale used for the pairwise verdicts: it is the
    scatter of the small-eval means around a smooth log-linear trend, i.e. what a CE
    difference has to beat to be readable at these run lengths.
    """
    table = efficiency_table(runs)
    small = table[table["event"] != "final_full"]
    xs, ys, groups = [], [], []
    for arm, sub in small.groupby("arm", sort=False):
        for _, row in sub.iterrows():
            xs.append(math.log(row["windows_seen"]))
            ys.append(row["ce"])
            groups.append(arm)
    arms = sorted(set(groups))
    centred = []
    means = {}
    for arm in arms:
        xg = [x for x, g in zip(xs, groups) if g == arm]
        yg = [y for y, g in zip(ys, groups) if g == arm]
        mx, my = sum(xg) / len(xg), sum(yg) / len(yg)
        means[arm] = (mx, my)
        centred.extend((x - mx, y - my, arm) for x, y in zip(xg, yg))
    slope = sum(dx * dy for dx, dy, _ in centred) / sum(dx * dx for dx, _, _ in centred)
    residuals = [dy - slope * dx for dx, dy, _ in centred]
    dof = len(residuals) - len(arms)
    resid_sd = math.sqrt(sum(r * r for r in residuals) / dof) if dof > 0 else float("nan")
    per_arm = {}
    for arm in arms:
        rr = [r for r, (_, _, g) in zip(residuals, centred) if g == arm]
        per_arm[arm] = {
            "mean_resid": sum(rr) / len(rr),
            "resid_sd": math.sqrt(sum(r * r for r in rr) / (len(rr) - 1))
            if len(rr) > 1
            else float("nan"),
        }
    return {
        "slope_per_ln": slope,
        "drop_per_doubling": -slope * math.log(2),
        "resid_sd": resid_sd,
        "dof": dof,
        "per_arm": per_arm,
    }


def pairwise_table(
    runs: dict[str, dict], deltas: dict[str, tuple[str, str]] | None = None
) -> pd.DataFrame:
    """Delta (A - B) at every aligned eval event for the named arm pairs."""
    if deltas is None:
        deltas = {
            "top-random": ("top_within_task", "random"),
            "random-bottom": ("random", "bottom_within_task"),
            "masked-random": ("random_masked", "random"),
        }
    table = eval_table(runs)
    rows = []
    for label, (a, b) in deltas.items():
        row = {"pair": label}
        for i, ev in table.iterrows():
            row[f"{ev['event']}@{i}"] = table.loc[i, a] - table.loc[i, b]
        rows.append(row)
    return pd.DataFrame(rows).set_index("pair")


def figure(runs: dict[str, dict], path: Path = FIGURE_PATH) -> Path:
    """Eval CE vs windows seen (log x) for every arm; final_full points marked open."""
    path.parent.mkdir(parents=True, exist_ok=True)
    fig, ax = plt.subplots(figsize=(8.5, 5.5))
    colors = {
        "random": "C0",
        "top_within_task": "C1",
        "bottom_within_task": "C2",
        "random_masked": "C3",
    }
    for arm, run in runs.items():
        evals = run["metrics"]["eval"]
        xs = [ev["windows_seen"] for ev in evals]
        ys = [ev["eval_ce"] for ev in evals]
        ax.plot(xs, ys, "-", color=colors.get(arm, "C4"), lw=1.4, label=arm, alpha=0.9)
        small = [(ev["windows_seen"], ev["eval_ce"]) for ev in evals if ev["event"] != "final_full"]
        final = [(ev["windows_seen"], ev["eval_ce"]) for ev in evals if ev["event"] == "final_full"]
        if small:
            ax.scatter(*zip(*small), color=colors.get(arm, "C4"), s=28, zorder=3)
        if final:
            ax.scatter(
                *zip(*final),
                facecolors="none",
                edgecolors=colors.get(arm, "C4"),
                s=70,
                lw=1.6,
                zorder=4,
            )
    ax.set_xscale("log")
    ax.set_xlabel("windows seen (global, log scale)")
    ax.set_ylabel("held-out cross-entropy")
    ax.set_title(
        "curve run 1 — eval CE vs windows seen\n(open markers: final 300-trace eval; filled: in-run 60-trace eval)"
    )
    ax.grid(True, which="both", alpha=0.25)
    ax.legend(title="arm", fontsize=9)
    fig.tight_layout()
    fig.savefig(path, dpi=150)
    plt.close(fig)
    return path


def _md(df: pd.DataFrame, floatfmt: str = ".4f") -> str:
    """Minimal GitHub-flavored markdown table (avoids a tabulate dependency)."""

    def fmt(value) -> str:
        if isinstance(value, float):
            return "—" if math.isnan(value) else f"{value:{floatfmt}}"
        return str(value)

    header = [df.index.name or ""] + [str(col) for col in df.columns]
    lines = ["| " + " | ".join(header) + " |", "|" + "|".join(["---"] * len(header)) + "|"]
    for idx, row in df.iterrows():
        lines.append("| " + " | ".join([fmt(idx)] + [fmt(v) for v in row]) + " |")
    return "\n".join(lines)


def markdown_tables(runs: dict[str, dict], summary: dict) -> str:
    """All report tables as one markdown string (what the report embeds)."""
    parts: list[str] = []

    cfg = summary["arms"]
    config = pd.DataFrame(
        {
            "traces (manifest)": {arm: cfg[arm]["n_traces"] for arm in runs},
            "tasks": {arm: cfg[arm]["n_tasks"] for arm in runs},
            "supervised chars (M)": {
                arm: round(cfg[arm]["supervised_chars"] / 1e6, 2) for arm in runs
            },
            "easy/hard/mid": {
                arm: "/".join(
                    str(cfg[arm]["per_bucket"].get(b, 0)) for b in ("easy", "hard", "mid")
                )
                for arm in runs
            },
            "steps_done": {arm: runs[arm]["metrics"]["steps_done"] for arm in runs},
            "windows_seen": {arm: runs[arm]["metrics"]["windows_seen"] for arm in runs},
            "elapsed_s": {arm: runs[arm]["metrics"]["elapsed_s"] for arm in runs},
            "total_elapsed_s": {arm: runs[arm]["metrics"].get("total_elapsed_s") for arm in runs},
            "finished (UTC)": {arm: runs[arm]["metrics"].get("updated_at") for arm in runs},
            "tok/s (global)": {arm: runs[arm]["metrics"]["tok_per_s"] for arm in runs},
            "peak mem (GiB)": {arm: runs[arm]["metrics"]["mem_gb"] for arm in runs},
            "stop_reason": {arm: runs[arm]["metrics"]["stop_reason"] for arm in runs},
        }
    )
    parts.append("### Run config\n\n" + _md(config, ".1f"))

    languages = sorted({lang for arm in runs for lang in cfg[arm]["per_language"]})
    langs = pd.DataFrame(
        {
            lang: {arm: cfg[arm]["per_language"].get(lang, 0) for arm in runs}
            for lang in languages
        }
    )
    langs.index.name = "arm"
    parts.append("### Traces per language\n\n" + _md(langs, ".0f"))

    teachers = sorted({teacher for arm in runs for teacher in cfg[arm]["per_teacher"]})
    teach = pd.DataFrame(
        {
            teacher: {arm: cfg[arm]["per_teacher"].get(teacher, 0) for arm in runs}
            for teacher in teachers
        }
    )
    teach.index.name = "arm"
    parts.append("### Traces per teacher\n\n" + _md(teach, ".0f"))

    loss = loss_table(runs)
    parts.append("### Train loss every 20 steps\n\n" + _md(loss))

    ev = eval_table(runs)
    ev_display = ev.copy()
    ev_display["windows_seen"] = ev_display["windows_seen"].map(
        lambda v: "—" if pd.isna(v) else f"{int(v)}"
    )
    parts.append("### Eval CE by event\n\n" + _md(ev_display, ".5f"))

    slope = slope_table(runs)
    parts.append("### Slopes (in-run evals)\n\n" + _md(slope))

    eff = efficiency_table(runs)
    eff["sup_tokens_seen"] = (eff["sup_tokens_seen"] / 1e6).round(3)
    eff = eff.rename(columns={"sup_tokens_seen": "sup_tokens_seen (M)"})
    parts.append("### CE vs supervised tokens seen\n\n" + _md(eff.reset_index(drop=True), ".5f"))

    pair = pairwise_table(runs)
    parts.append("### Pairwise deltas (A - B)\n\n" + _md(pair, ".5f"))

    return "\n\n".join(parts) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--curve-dir", type=Path, default=CURVE_DIR)
    parser.add_argument("--figure", type=Path, default=FIGURE_PATH)
    parser.add_argument("--tables-out", type=Path, help="write the markdown tables here too")
    args = parser.parse_args()

    runs = {arm: run for arm in ARMS if (run := load_run(arm, args.curve_dir)) is not None}
    missing = [arm for arm in ARMS if arm not in runs]
    if missing:
        console.print(f"[yellow]missing arms (no metrics.json): {', '.join(missing)}[/yellow]")
    if not runs:
        raise SystemExit(f"no arm outputs under {args.curve_dir / 'arms'}")

    summary = load_manifests(args.curve_dir)
    tables = markdown_tables(runs, summary)
    print(tables)
    noise = common_slope(runs)
    print(
        f"\nnoise: common slope b={noise['slope_per_ln']:+.4f}/ln "
        f"drop/doubling={noise['drop_per_doubling']:+.4f} resid_sd={noise['resid_sd']:.5f} "
        f"(dof={noise['dof']})"
    )
    for arm, run in runs.items():
        if run["encoded"]:
            total = sum(x for _, x, _ in run["encoded"])
            manifest = sum(y for _, _, y in run["encoded"])
            print(
                f"{arm}: encoded {total}/{manifest} traces (dropped {run['dropped']}); "
                f"metrics n_traces={run['metrics']['n_traces']}"
            )
    path = figure(runs, args.figure)
    console.print(f"figure -> {path.relative_to(ROOT)}")
    if args.tables_out:
        args.tables_out.write_text(tables)
        console.print(f"tables -> {args.tables_out}")


if __name__ == "__main__":
    main()

#!/usr/bin/env python
"""Closure metrics vs measured flip point (retrospective).

Computes closure metrics (see src/openswe_traces/synth/closure_metrics.py) for
every authored unit whose repo has a base tree on disk, joins in the measured
flip per solver from experiments/pipeline/state.db + results.md, writes
outputs/closure_metrics.parquet and analytics/research/closure_vs_flip.md, and
reports Spearman of each metric vs the measured flip.

Reproduce: uv run python scripts/closure_metrics.py
"""

from __future__ import annotations

import argparse
import json
import math
import re
import sqlite3
from pathlib import Path

import pandas as pd
from scipy.stats import spearmanr

from openswe_traces.synth.closure_metrics import (
    ClosureMetricsError,
    closure_metrics,
    excision_patch_from_diff,
)

ROOT = Path(__file__).resolve().parents[1]
AUTHORED = ROOT / "experiments/pipeline/authored"
TREES_JSON = ROOT / "experiments/pipeline/closure_A/trees.json"
REPOS_YAML = ROOT / "experiments/pipeline/repos.yaml"
STATE_DB = ROOT / "experiments/pipeline/state.db"
OUT_PARQUET = ROOT / "outputs/closure_metrics.parquet"
OUT_MD = ROOT / "analytics/research/closure_vs_flip.md"

METRICS = [
    "n_files",
    "n_funcs_removed",
    "lines_removed",
    "internal_edges",
    "boundary_in",
    "boundary_out",
    "ratio",
]

# The go-github dry-run unit (fresh_laptop_setup_2026-09-19.md): authored on the
# laptop, only the verifier sandbox synced here. Base tree + excised sandbox
# copy are on disk; metadata comes from the note (predicted_flip L5, control).
GO_GITHUB_UNITS = [
    {
        "repo": "go-github",
        "unit": "redirect-until-found",
        "predicted_flip": 5,
        "control": True,
        "family": "cross-file",
    }
]

# Measured flips reported in HANDOFF.md ("Results so far", main checkout,
# 2026-09-19) and fresh_laptop_setup_2026-09-19.md. Those solves ran against the
# main checkout and were never written to this worktree's state.db, which only
# holds the early client-go/gin trials. (repo, unit) -> {solver: (label, enc)};
# encoding matches measured_flip: level k -> k; "none" -> max tried + 1
# (depresolver 0/1 at L2 -> 3, doactionbatches failed L2 twice -> 3).
MANUAL_FLIPS: dict[tuple[str, str], dict[str, tuple[str, int]]] = {
    ("helm", "coalesce"): {"cursor": ("5", 5)},
    ("helm", "depresolver"): {"cursor": ("none", 3)},
    ("kops", "assetsremap"): {"cursor": ("0", 0)},
    ("kops", "clustervalid"): {"cursor": ("0", 0)},
    ("gin", "formmapping"): {"cursor": ("5", 5)},
    ("goa", "errloc"): {"cursor": ("0", 0)},
    ("goa", "evalrun"): {"cursor": ("2", 2)},
    ("client-go", "memdbstaging"): {"devin": ("0", 0)},
    ("client-go", "doactionbatches"): {"devin": ("none", 3)},
}

_PREDICTED_RE = re.compile(r"predicted_flip:\s*L(\d)")
_CONTROL_RE = re.compile(r"control:\s*(true|false)")


def scan_authored_units() -> list[dict]:
    """All authored units on disk: repo, unit, predicted_flip, control."""
    units: list[dict] = []
    for repo_dir in sorted(AUTHORED.iterdir()):
        if not repo_dir.is_dir():
            continue
        repo = repo_dir.name
        for unit_dir in sorted(repo_dir.iterdir()):
            author = unit_dir / "_author"
            if not (author / "excised" / "excision.patch").is_file():
                continue
            pred = None
            control = None
            diff = author / "difficulty.md"
            if diff.is_file():
                text = diff.read_text(encoding="utf-8", errors="replace")
                m = _PREDICTED_RE.search(text)
                if m:
                    pred = int(m.group(1))
                m = _CONTROL_RE.search(text)
                if m:
                    control = m.group(1) == "true"
            units.append(
                {
                    "repo": repo,
                    "unit": unit_dir.name,
                    "author": author,
                    "predicted_flip": pred,
                    "control": control,
                }
            )
    return units


def load_trees() -> dict[str, Path | dict[str, Path]]:
    """Repo -> base tree path(s). trees.json overrides repos.yaml src:."""
    if TREES_JSON.is_file():
        data = json.loads(TREES_JSON.read_text(encoding="utf-8"))
        return {k: v for k, v in data.items() if not k.startswith("_")}
    spec = {}
    if REPOS_YAML.is_file():
        # minimal YAML read: repos.yaml is flat `repos: - name: ... src: ...`
        for m in re.finditer(r"^  - name: (\S+)\n(?:.*\n)*?    src: (\S+)", REPOS_YAML.read_text(encoding="utf-8"), re.MULTILINE):
            spec[m.group(1)] = m.group(2)
    return {name: ROOT / src for name, src in spec.items()}


def read_trials() -> dict[tuple[str, str], dict]:
    """(repo, exact unit) -> {solver: {"levels": [int], "passed": [int]}} from state.db."""
    out: dict[tuple[str, str], dict] = {}
    if not STATE_DB.is_file():
        return out
    con = sqlite3.connect(STATE_DB)
    try:
        for repo, unit, level, solver, reward, excluded in con.execute(
            "SELECT repo, unit, level, solver, reward, excluded FROM trials"
        ):
            rec = out.setdefault((repo, unit), {}).setdefault(solver or "?", {"levels": [], "passed": []})
            rec["levels"].append(level)
            if reward == 1.0 and not excluded:
                rec["passed"].append(level)
    finally:
        con.close()
    return out


def measured_flip(rec: dict | None) -> tuple[str | None, int | None]:
    """(flip label, encoded flip) for one solver's trials.

    flip = lowest tried level with a pass; 'none' when tried but no pass; None
    when there were no trials.  Encoded: level k -> k, none -> max tried + 1.
    """
    if not rec:
        return None, None
    if rec["passed"]:
        return str(min(rec["passed"])), min(rec["passed"])
    return "none", max(rec["levels"]) + 1


def _skip(skipped: list[dict], repo: str, unit: str, reason: str) -> None:
    """Record a skipped unit, annotating the reason with any known measured flip."""
    flips = MANUAL_FLIPS.get((repo, unit))
    if flips:
        note = ", ".join(f"{solver} {label}" for solver, (label, _enc) in sorted(flips.items()))
        reason = f"{reason} (measured flip: {note})"
    skipped.append({"repo": repo, "unit": unit, "reason": reason})


def _metrics_from_excised_dir(base: Path, excised_dir: Path | None, unit: str) -> tuple[dict | None, str]:
    """Fallback excision: diff the base tree against the materialised <unit>-L2 env.

    Returns (metrics, "") on success, else (None, reason suffix). An empty diff
    means the env was materialised without the patch (an unexcised base copy) —
    reported rather than emitted as all-zero metrics.
    """
    if excised_dir is None:
        return None, ""
    env = excised_dir / f"{unit}-L2" / "environment" / "src"
    if not (env / "go.mod").is_file():
        return None, f"; no materialised {unit}-L2 env at {env}"
    patch = excision_patch_from_diff(base, env)
    if not any(ln.startswith("-") and not ln.startswith("---") for ln in patch.splitlines()):
        return None, "; materialised -L2 env is unexcised (identical to base)"
    return closure_metrics(base, patch), "; metrics via diff of materialised -L2 env"


def main(argv: list[str] | None = None) -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--only", nargs="*", default=None, help="restrict to repo[/unit] pairs")
    args = ap.parse_args(argv)

    trees = load_trees()
    trials = read_trials()
    units = scan_authored_units() + [
        {"repo": u["repo"], "unit": u["unit"], "author": None, "predicted_flip": u["predicted_flip"], "control": u["control"]}
        for u in GO_GITHUB_UNITS
    ]

    rows: list[dict] = []
    skipped: list[dict] = []
    for unit in sorted(units, key=lambda u: (u["repo"], u["unit"])):
        repo, name = unit["repo"], unit["unit"]
        if args.only:
            want = {o.split("/")[0]: o.split("/")[1] if "/" in o else None for o in args.only}
            if repo not in want:
                continue
            if want[repo] and want[repo] != name:
                continue
        tree_cfg = trees.get(repo)
        if tree_cfg is None:
            _skip(skipped, repo, name, "no base tree on disk")
            continue
        rewrites: list = []
        excised_dir: Path | None = None
        if isinstance(tree_cfg, str):
            base = Path(tree_cfg).resolve()
        else:
            base = Path(tree_cfg["base"]).resolve()
            rewrites = tree_cfg.get("patch_rewrites") or []
            if tree_cfg.get("excised_dir"):
                excised_dir = Path(tree_cfg["excised_dir"]).resolve()
        if not (base / "go.mod").is_file():
            _skip(skipped, repo, name, f"base tree missing: {base}")
            continue
        try:
            if isinstance(tree_cfg, dict) and tree_cfg.get("excised"):
                excised = Path(tree_cfg["excised"]).resolve()
                patch = excision_patch_from_diff(base, excised)
            else:
                patch = unit["author"] / "excised" / "excision.patch"
            metrics = closure_metrics(base, patch, patch_rewrites=rewrites)
            source = "authored patch rebranded to base tree" if rewrites else "authored unit"
        except ClosureMetricsError as exc:
            metrics, note = _metrics_from_excised_dir(base, excised_dir, name)
            if metrics is None:
                first = next((ln for ln in str(exc).splitlines() if ln.strip()), "patch does not apply")
                _skip(skipped, repo, name, f"patch does not apply: {first[:160]}{note}")
                continue
            source = "diff of materialised -L2 env" + note

        solvers = trials.get((repo, name), {})
        row = {
            "repo": repo,
            "unit": name,
            "family": unit.get("family") or ("single-file" if metrics["n_files"] == 1 else "cross-file"),
            "predicted_flip": unit["predicted_flip"],
            "control": unit["control"],
            "metric_source": source,
            **metrics,
        }
        notes: list[str] = []
        for solver in ("devin", "cursor"):
            label, enc = measured_flip(solvers.get(solver))
            manual = MANUAL_FLIPS.get((repo, name), {}).get(solver)
            if label is None and manual is not None:
                label, enc = manual
                notes.append(f"{solver} flip from HANDOFF.md")
            row[f"flip_{solver}"] = label
            row[f"flip_{solver}_enc"] = enc
        if not solvers and not notes:
            notes.append("no trials yet")
        row["flip_note"] = "; ".join(notes)
        rows.append(row)

    # state.db may trial a "-cv" packaging of an authored unit (connarray-cv);
    # emit a second row with the same metrics but the -cv trials' own flips.
    extra_units = {
        (repo, unit.removesuffix("-cv")) for (repo, unit) in trials if unit.endswith("-cv")
    }
    metrics_by_base = {(r["repo"], r["unit"]) if not r["unit"].endswith("-cv") else (r["repo"], r["unit"].removesuffix("-cv")): r for r in rows}
    for repo, base_name in sorted(extra_units):
        if (repo, base_name) not in metrics_by_base:
            # -cv variant whose authored unit could not be measured (e.g.
            # lockresolver-cv): report it in the skipped table
            _skip(skipped, repo, base_name + "-cv", "same authored unit as " + base_name + " (see that row)")
            continue
        base_row = metrics_by_base[(repo, base_name)]
        cv_name = base_name + "-cv"
        solvers = trials.get((repo, cv_name), {})
        row = dict(base_row)
        row["unit"] = cv_name
        row["metric_source"] = f"same authored unit as {base_name} (metrics reused)"
        for solver in ("devin", "cursor"):
            label, enc = measured_flip(solvers.get(solver))
            row[f"flip_{solver}"] = label
            row[f"flip_{solver}_enc"] = enc
        row["flip_note"] = "no trials yet" if not solvers else ""
        rows.append(row)

    # units known from results.md/state.db but with no authored dir on this
    # machine (authored elsewhere); no tree, so they are skipped with a reason
    for repo, unit in sorted({(r, u) for (r, u), _ in trials.items()} | {("nats-server", "seqset"), ("nats-server", "subjecttree")}):
        if any(r["repo"] == repo and r["unit"] == unit for r in rows):
            continue
        if any(s["repo"] == repo and s["unit"] == unit for s in skipped):
            continue
        _skip(
            skipped,
            repo,
            unit,
            "no authored dir / base tree on this machine (results.md lists it; "
            "seqset: predicted L4 control; subjecttree: rejected A3)",
        )

    df = pd.DataFrame(rows)
    df.to_parquet(OUT_PARQUET, index=False)

    measured = df[df["flip_devin_enc"].notna() | df["flip_cursor_enc"].notna()].copy()
    if not measured.empty:
        measured["_flip"] = measured["flip_devin_enc"].fillna(measured["flip_cursor_enc"])
        measured["_solver"] = [
            "devin" if pd.notna(v) else "cursor" for v in measured["flip_devin_enc"]
        ]
    extremes = measured[measured["_flip"].isin([0, 5])] if not measured.empty else measured

    spearman_all = _spearman_table(measured)
    spearman_extremes = _spearman_table(extremes)

    # Supplementary: same metrics vs *predicted* flip over all computed units
    # that carry a predicted flip (n is small; clearly labelled as such).
    pred_rows = df[df["predicted_flip"].notna()]
    spearman_pred = _spearman_table(pred_rows, flip_col="predicted_flip")

    write_markdown(df, spearman_all, spearman_extremes, spearman_pred, skipped, trees)

    print(f"wrote {OUT_PARQUET}")
    print(f"wrote {OUT_MD}")
    print(f"units with metrics: {len(df)}; skipped: {len(skipped)}")
    for s in skipped:
        print(f"  skipped {s['repo']}/{s['unit']}: {s['reason'][:140]}")
    if not measured.empty:
        print("\nmeasured-flip units:")
        for _, r in measured.sort_values(["_flip", "repo", "unit"]).iterrows():
            print(
                f"  {r['repo']}/{r['unit']} flip={r['_flip']:.0f} ({r['_solver']}) "
                + " ".join(f"{m}={r[m]}" for m in METRICS)
            )


def _spearman_table(rows: pd.DataFrame, flip_col: str = "_flip") -> list[dict]:
    """Spearman rho/p per metric vs the encoded flip column."""
    out: list[dict] = []
    for metric in METRICS:
        if len(rows) >= 3 and rows[metric].nunique() > 1:
            rho, p = spearmanr(rows[metric], rows[flip_col])
        else:
            rho, p = float("nan"), float("nan")
        out.append(
            {"metric": metric, "rho": float(rho) if not math.isnan(rho) else None, "p": float(p) if not math.isnan(p) else None}
        )
    return out


def md_table(df: pd.DataFrame, cols: list[str]) -> str:
    """Small markdown table renderer (pandas to_markdown needs tabulate)."""
    def cell(v) -> str:
        if v is None or (isinstance(v, float) and math.isnan(v)):
            return ""
        if isinstance(v, float) and v.is_integer():
            return str(int(v))
        return str(v)

    header = "| " + " | ".join(cols) + " |"
    sep = "|" + "|".join(["---"] * len(cols)) + "|"
    body = []
    for _, row in df.iterrows():
        body.append("| " + " | ".join(cell(row[c]) for c in cols) + " |")
    return "\n".join([header, sep] + body)


def write_markdown(
    df: pd.DataFrame,
    spearman_all: list[dict],
    spearman_extremes: list[dict],
    spearman_pred: list[dict],
    skipped: list[dict],
    trees: dict,
) -> None:
    lines: list[str] = []
    add = lines.append
    add("# Closure metrics vs measured flip point — retrospective")
    add("")
    add("2026-09-19. Zero-solver-cost analysis of the authored unit bank: does the size of a unit's")
    add("*closure* (how much of the removed code is determined only by itself) predict its measured")
    add("flip point? Hypothesis: units that flip high have a higher internal/boundary edge ratio than")
    add("units that pass at L0.")
    add("")
    add("## Method")
    add("")
    add("- Parse Go with `go/ast` via `tools/goclosure/main.go` (no regex; no `go/types` — the trees are")
    add("  obfuscated and have no module cache). Removed functions = decls whose body the excision patch")
    add("  deletes or stubs (base-tree body hash differs from post-patch, or the decl is gone).")
    add("- `internal_edges`: call references among removed funcs (from removed bodies in the base tree).")
    add("- `boundary_in`: call sites in the remaining (post-excision) tree that reference a removed func.")
    add("- `boundary_out`: call references from removed bodies to symbols that remain.")
    add("- `ratio = internal_edges / max(1, boundary_in + boundary_out)`.")
    add("- Method calls resolved by light local type inference (params, receivers, `:=` literals, `new`,")
    add("  type assertions, one level of struct fields, import aliases); unresolved receivers fall back to")
    add("  name-uniqueness (every func with that name removed ⇒ internal, else external).")
    add("- helm/kops excision patches were authored against `pipeline_repos` identity-pass trees")
    add("  (`example.internal/chartkit/v4`, `example.internal/clustkit`) that are not on disk. The base")
    add("  trees below are the `prepare_repo` trees at the same pinned commits (`example.internal/helm`,")
    add("  `example.internal/kops`); each patch is rebranded back by ordered word-boundary rewrites")
    add("  (trees.json `patch_rewrites`) before `git apply` — all 20 apply cleanly. Edge counts are")
    add("  invariant under a consistent module/brand rename.")
    add("- Measured flips from `experiments/pipeline/state.db` (trials) + the main checkout's `HANDOFF.md`")
    add("  (\"Results so far\", 2026-09-19) + `results.md`; per solver, lowest level with a pass;")
    add("  `none` = tried with no pass.")
    add("")
    add("## Trees used")
    add("")
    add("| repo | tree | provenance |")
    add("|---|---|---|")
    for repo, cfg in sorted(trees.items()):
        if isinstance(cfg, str):
            add(f"| {repo} | `{cfg}` | authored base tree |")
        elif cfg.get("excised"):
            add(f"| {repo} | base `{cfg['base']}` / excised `{cfg['excised']}` | dry-run sandbox; excision derived by diffing the two trees |")
        else:
            brands = ", ".join(f"{a}→{b}" for a, b in cfg.get("patch_rewrites", [])[:2])
            add(f"| {repo} | `{cfg['base']}` | `prepare_repo` tree (main checkout, read-only); authored patches rebranded ({brands}, …) |")
    add("")
    add("`experiments/pipeline/closure_A/trees.json` points at each tree. Repos with no tree on disk")
    add("(goa, nats-server) are skipped — see the skipped table. client-go's tree is a proxy: same")
    add("upstream commit and obfuscation scheme as the authored base (`example.internal/kvstore/v2` vs")
    add("the base's `example.internal/clientgo`); the excision patches apply cleanly to it for 6/10")
    add("units, and the remaining 4 are skipped (patch context drifts off the proxy tree). helm/kops")
    add("use the `prepare_repo` trees with rebranded patches (see Method).")
    add("")
    add("## Units and metrics")
    add("")
    if df.empty:
        add("_no units with metrics_")
    else:
        cols = [
            "repo", "unit", "family", "predicted_flip", "control",
            "flip_devin", "flip_cursor",
            "n_files", "n_funcs_removed", "lines_removed",
            "internal_edges", "boundary_in", "boundary_out", "ratio",
        ]
        add(md_table(df, cols))
    add("")
    add("`flip_devin`/`flip_cursor`: lowest tried level with a pass (`none` = tried, no pass; blank = no")
    add("trials yet). `connarray` and `connarray-cv` are the *same authored unit* packaged twice")
    add("(results.md lists both); their metric rows are identical. The go-github row is **flagged**: it")
    add("has no flip yet (authored in the dry run; `predicted_flip: L5`, control). `fresh_laptop_setup`")
    add("describes it as 6 functions; the synced sandbox shows 5 stubbed (bareDoUntilFound,")
    add("roundTripWithOptionalFollowRedirect, checkRedirectHost, getArchiveLinkWithoutRateLimit,")
    add("getArchiveLinkWithRateLimit) — metrics reflect the sandbox.")
    add("")

    measured = df[df["flip_devin_enc"].notna() | df["flip_cursor_enc"].notna()].copy()
    if not measured.empty:
        measured["_flip"] = measured["flip_devin_enc"].fillna(measured["flip_cursor_enc"])
        measured["_solver"] = ["devin" if pd.notna(v) else "cursor" for v in measured["flip_devin_enc"]]

    add("## Spearman vs measured flip")
    add("")
    if measured.empty:
        add("_no rows with a measured flip and metrics_")
    else:
        units = ", ".join(f"{r['repo']}/{r['unit']}" for _, r in measured.sort_values(["_flip", "repo", "unit"]).iterrows())
        n_distinct = len({(r["repo"], r["unit"].removesuffix("-cv")) for _, r in measured.iterrows()})
        add(f"n = {len(measured)} rows / {n_distinct} distinct authored units: {units}.")
        add("Encoded flip: L0=0, L2=2, L5=5, `none` = max tried + 1 (depresolver, doactionbatches → 3).")
        add("Caveats: (a) `connarray`/`connarray-cv` are the same authored unit — one duplicate row;")
        add("(b) `flip` mixes solvers (devin for client-go, cursor elsewhere); (c) helm/kops metrics")
        add("come from rebranded patches on `prepare_repo` trees, not the authored trees themselves.")
        add("")
        add("| metric | rho | p |")
        add("|---|---:|---:|")
        for s in spearman_all:
            rho = "nan" if s["rho"] is None else f"{s['rho']:.3f}"
            p = "nan" if s["p"] is None else f"{s['p']:.3f}"
            add(f"| {s['metric']} | {rho} | {p} |")
    add("")
    add("## Spearman vs measured flip — L0/L5 extremes only")
    add("")
    extremes = measured[measured["_flip"].isin([0, 5])] if not measured.empty else measured
    if len(extremes) < 3:
        add("_fewer than 3 rows at the extremes_")
    else:
        units = ", ".join(f"{r['repo']}/{r['unit']}" for _, r in extremes.sort_values(["_flip", "repo", "unit"]).iterrows())
        add(f"n = {len(extremes)}; units: {units}. Restricting to flip ∈ {{L0, L5}} asks the sharper")
        add("question: does closure size separate \"solved immediately\" from \"hard until L5\"?")
        add("")
        add("| metric | rho | p |")
        add("|---|---:|---:|")
        for s in spearman_extremes:
            rho = "nan" if s["rho"] is None else f"{s['rho']:.3f}"
            p = "nan" if s["p"] is None else f"{s['p']:.3f}"
            add(f"| {s['metric']} | {rho} | {p} |")
    add("")
    add("### Raw rows: L5 units vs L0 units")
    add("")
    if extremes.empty:
        add("_none_")
    else:
        cols = ["repo", "unit", "_solver", "_flip"] + METRICS
        add(md_table(extremes.sort_values(["_flip", "repo", "unit"], ascending=[False, True, True]), cols))
    add("")
    add("## Supplementary: Spearman vs predicted flip")
    add("")
    add("All computed units that carry a `predicted_flip` (gin, helm and kops batches + go-github")
    add("control). Mostly unmeasured; included for coverage. The go-github unit (`redirect-until-found`,")
    add("predicted L5, control) is **flagged**: authored in the dry run, only the verifier sandbox")
    add("synced; metadata from `fresh_laptop_setup_2026-09-19.md`.")
    add("")
    pred = df[df["predicted_flip"].notna()]
    per_repo = ", ".join(f"{r} {len(pred[pred['repo'] == r])}" for r in sorted(pred["repo"].unique())) if not pred.empty else ""
    add(f"n = {len(pred)} ({per_repo})")
    add("")
    add("| metric | rho | p |")
    add("|---|---:|---:|")
    for s in spearman_pred:
        rho = "nan" if s["rho"] is None else f"{s['rho']:.3f}"
        p = "nan" if s["p"] is None else f"{s['p']:.3f}"
        add(f"| {s['metric']} | {rho} | {p} |")
    add("")
    add("## Skipped (no base tree on disk, or patch does not apply)")
    add("")
    if not skipped:
        add("_none_")
    else:
        add("| repo | unit | reason |")
        add("|---|---|---|")
        for s in skipped:
            reason = s["reason"].splitlines()[0] if s["reason"] else ""
            add(f"| {s['repo']} | {s['unit']} | {reason} |")
    add("")
    add("## Flip source notes")
    add("")
    add("- `analytics/research/task_space_framework.md` reports A-ladder (A0–A4) single-attempt flips on")
    add("  older client-go units (dynamic pipeline A3, codec A1, backoff A0, mutations A4). Those are a")
    add("  different ladder and pre-date the 3-attempt policy (C6, see the framework's replication note), so")
    add("  they are not merged into the measured-flip Spearman above.")
    add("- `analytics/research/fresh_laptop_setup_2026-09-19.md`: gin/bindingdispatch = L0 (Composer 2.5),")
    add("  consistent with state.db.")
    add("- Main-checkout `HANDOFF.md` (\"Results so far\"): Composer — goa/errloc, kops/assetsremap,")
    add("  kops/clustervalid pass L0; gin/formmapping and helm/coalesce flip L2→L5; goa/evalrun passes")
    add("  L2; helm/depresolver 0/1 at L2 (flip `none`). Devin — client-go/connarray and memdbstaging")
    add("  pass L0; doactionbatches fails L2 twice (flip `none`). These trials are not in this")
    add("  worktree's state.db; they enter via `MANUAL_FLIPS` in `scripts/closure_metrics.py`.")
    add("- The materialised `tasks_composerver/{helm,kops}/<unit>-L2/environment/src` trees are")
    add("  **unexcised** — verified identical to the `prepare_repo` base, although")
    add("  `outputs/materialize_hkg.log` lists them as materialised (the chartkit/clustkit-authored")
    add("  patches cannot apply to `example.internal/helm`/`kops` trees). They are therefore unusable")
    add("  as excision-diff inputs; helm/kops metrics come from the rebranded authored patches.")
    OUT_MD.write_text("\n".join(lines) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()

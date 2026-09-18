"""Cheap task-difficulty probes: how little rollout data identifies an instance's difficulty.

`solve_rate` currently needs ~12 labeled rollouts per task. Three instruments ask how much
cheaper it can be measured, all reusing the existing artifacts:

A. Cross-teacher probe (`cross_teacher_report`): one rollout of another harness/teacher combo
   as a probe for the target combo's rate. For every ordered pair of labeled combos, a single
   random probe rollout (seed 0, per-instance stable) predicts target ``solve_rate > 0.5``
   (AUC); probe means over the first k = 1, 2, 3 rollouts of the same seeded order are
   Spearman-correlated with the target rate. The cheapest teacher
   (``minisweagent/qwen36_27b``) is also scored against the leave-this-combo-out rate.

B. Partial-rollout probe (`prefix`): per-trajectory prefix features at k = 5, 10, 20, 40 tool
   calls — gold file viewed/edited, edit calls, distinct commands, repeat rate, assistant
   chars, and any tool observation containing Error/Traceback/FAILED. One streaming DuckDB
   pass over the corpus writes parts to ``outputs/prefix_features_parts/`` (resume-safe,
   atomic) merged into ``outputs/prefix_features.parquet``. Standardized logistic regressions
   (grouped 80/20 by instance_id, the ``score.py`` split) predict ``resolved`` from prefix-k
   features only; a second model predicts the instance bucket (mid vs degenerate) from
   per-instance means of the same features.

C. Adaptive sampling simulation (`simulate_rules`): on instances with >= 6 labeled rollouts,
   draw rollouts one at a time in random order and stop when the Wilson 80% interval for
   solve_rate fits inside one bucket (cap 12), or under a 2-agreeing-outcomes-else-4 rule.
   Reports rollouts used and bucket accuracy against the full-data bucket, over 5 seeds.

Numbers in tables go to ``analytics/research/cheap_difficulty_summary.md``.

Examples:
  uv run python scripts/cheap_difficulty.py --stream-only
  uv run python scripts/cheap_difficulty.py --skip-stream
  uv run python scripts/cheap_difficulty.py --merge-only --head 5
"""

from __future__ import annotations

import argparse
import itertools
import math
import sys
import time
import traceback
import warnings
import zlib
from dataclasses import dataclass
from datetime import UTC, datetime
from functools import partial
from pathlib import Path

import numpy as np
import pandas as pd
from rich.console import Console
from scipy.stats import norm, spearmanr
from sklearn.metrics import roc_auc_score
from sklearn.model_selection import GroupShuffleSplit

import duckdb

from .data import PARQUET_GLOB, ROOT, connect_ephemeral, list_parquet_files, parse_shard
from .difficulty import (
    EASY_MIN,
    HARD_MAX,
    MIN_LABELED,
    difficulty_buckets,
    leave_one_out_solve_rate,
    pct,
    rel,
)
from .features import (
    BASH_EDIT_RE,
    EDIT_TOOL_NAMES,
    EDIT_TOOL_VIEW_ONLY,
    _sql_str,
    _tuple_sql,
    fmt_duration,
    merge_parts,
    process_file,
    utcnow,
)
from .score import (
    MODEL_FEATURES,
    NULL_FILL,
    TEMPORAL_MODEL_FEATURES,
    ModelFit,
    fit_logistic,
    md_table,
)

console = Console()

PARTS_DIR = ROOT / "outputs" / "prefix_features_parts"
OUT_PARQUET = ROOT / "outputs" / "prefix_features.parquet"
OUT_MD = ROOT / "analytics" / "research" / "cheap_difficulty_summary.md"
PROXY_PARQUET = ROOT / "outputs" / "proxy_features.parquet"
TEMPORAL_PARQUET = ROOT / "outputs" / "temporal_features.parquet"
DIFFICULTY_PARQUET = ROOT / "outputs" / "task_difficulty.parquet"

K_VALUES = (5, 10, 20, 40)
PREFIX_BASE_FEATURES = (
    "gold_view",
    "gold_edit",
    "n_edit",
    "n_distinct",
    "repeat_rate",
    "assistant_chars",
    "any_error",
)
PREFIX_FEATURES = [f"{name}_{k}" for k in K_VALUES for name in PREFIX_BASE_FEATURES]
ERROR_RE = "error|traceback|failed"

PROBE_COMBO = ("minisweagent", "qwen36_27b")
PROBE_KS = (1, 2, 3)
SIM_SEEDS = 5
SIM_MIN_LABELED = 6
WILSON_CONF = 0.80
WILSON_Z = float(norm.ppf(1 - (1 - WILSON_CONF) / 2))
SIM_CAP = 12
SIMPLE_CAP = 4

SEED = 42
TEST_SIZE = 0.2
REF_FULL_TRACE_AUC = 0.7115
REF_TASK_RATE_AUC = 0.9361
BUCKET_LABELS = ("all_fail", "hard", "mid", "easy", "all_pass", "unknown")


def k_features(k: int) -> list[str]:
    return [f"{name}_{k}" for name in PREFIX_BASE_FEATURES]


def short_label(combo: tuple[str, str]) -> str:
    return f"{combo[0][:2]}/{combo[1]}"


def _call_k_aggs(k: int) -> str:
    le = f"call_idx <= {k}"
    return (
        f",\n        count(CASE WHEN {le} THEN 1 END)::INTEGER AS n_calls_{k}"
        f",\n        coalesce(max(CASE WHEN {le} THEN sees_gold END), 0)::TINYINT"
        f" AS gold_view_{k}"
        f",\n        coalesce(max(CASE WHEN {le} THEN (is_edit = 1 AND sees_gold = 1)::TINYINT"
        f" END), 0)::TINYINT AS gold_edit_{k}"
        f",\n        coalesce(sum(CASE WHEN {le} THEN is_edit ELSE 0 END), 0)::INTEGER AS n_edit_{k}"
        f",\n        count(DISTINCT CASE WHEN {le} THEN call_sig END)::INTEGER AS n_distinct_{k}"
        f",\n        CASE WHEN count(CASE WHEN {le} THEN 1 END) = 0 THEN 0.0"
        f" ELSE 1.0 - count(DISTINCT CASE WHEN {le} THEN call_sig END)::DOUBLE"
        f" / count(CASE WHEN {le} THEN 1 END) END AS repeat_rate_{k}"
        f",\n        coalesce(max(CASE WHEN {le} THEN cum_assistant_chars END), 0)::BIGINT"
        f" AS assistant_chars_{k}"
    )


def _msg_k_aggs(k: int) -> str:
    return (
        f",\n        (coalesce(max(CASE WHEN role = 'tool' AND cum_calls <= {k}"
        f" THEN cum_error_obs END), 0) > 0) AS any_error_{k}"
    )


def _select_k_cols(k: int) -> str:
    cols = [
        f",\n    coalesce(ca.n_calls_{k}, 0)::INTEGER AS n_calls_{k}",
        f",\n    coalesce(ca.gold_view_{k}, 0)::TINYINT AS gold_view_{k}",
        f",\n    coalesce(ca.gold_edit_{k}, 0)::TINYINT AS gold_edit_{k}",
        f",\n    coalesce(ca.n_edit_{k}, 0)::INTEGER AS n_edit_{k}",
        f",\n    coalesce(ca.n_distinct_{k}, 0)::INTEGER AS n_distinct_{k}",
        f",\n    coalesce(ca.repeat_rate_{k}, 0.0) AS repeat_rate_{k}",
        f",\n    coalesce(ca.assistant_chars_{k}, 0)::BIGINT AS assistant_chars_{k}",
        f",\n    coalesce(ma.any_error_{k}, false) AS any_error_{k}",
    ]
    return "".join(cols)


PREFIX_SQL = r"""
WITH src AS (
    SELECT instance_id, trajectory_id, resolved, messages, metadata
    FROM read_parquet('{file_sql}', union_by_name=true)
),
gold AS (
    SELECT
        trajectory_id,
        CASE
            WHEN len(git_paths) > 0 THEN list_distinct(git_paths)
            ELSE list_distinct(plus_paths)
        END AS gold_paths
    FROM (
        SELECT
            trajectory_id,
            regexp_extract_all(
                coalesce(metadata.reference_patch.patch, ''), 'diff --git a/([^\s]+) b/', 1
            ) AS git_paths,
            regexp_extract_all(
                coalesce(metadata.reference_patch.patch, ''), '\+\+\+ b/([^\s]+)', 1
            ) AS plus_paths
        FROM src
    )
),
joined AS (
    SELECT t.trajectory_id, t.instance_id, t.resolved, t.messages, g.gold_paths
    FROM src t
    LEFT JOIN gold g ON g.trajectory_id = t.trajectory_id
),
msgs AS (
    SELECT
        j.trajectory_id,
        mi,
        m.role AS role,
        sum(len(m.tool_calls)) OVER w AS cum_calls,
        sum(
            CASE WHEN m.role = 'assistant' THEN length(coalesce(m.content, '')) ELSE 0 END
        ) OVER w AS cum_assistant_chars,
        sum(
            CASE
                WHEN m.role = 'tool'
                 AND regexp_matches(coalesce(m.content, ''), '{error_re}', 'i')
                THEN 1 ELSE 0
            END
        ) OVER w AS cum_error_obs
    FROM joined j, unnest(j.messages) WITH ORDINALITY AS u(m, mi)
    WINDOW w AS (
        PARTITION BY j.trajectory_id ORDER BY mi
        ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    )
),
calls AS (
    SELECT
        j.trajectory_id,
        mi,
        ci,
        row_number() OVER (PARTITION BY j.trajectory_id ORDER BY mi, ci) AS call_idx,
        md5(coalesce(tc.function.arguments, '')) AS call_sig,
        CASE
            WHEN lower(coalesce(tc.function.name, '')) = '{edit_view_only}'
                THEN (lower(coalesce(
                    json_extract_string(try_cast(tc.function.arguments AS JSON), '$.command'), ''
                )) <> 'view')::TINYINT
            WHEN lower(coalesce(tc.function.name, '')) IN ({edit_tool_names})
                THEN 1::TINYINT
            WHEN regexp_matches(
                lower(coalesce(
                    json_extract_string(try_cast(tc.function.arguments AS JSON), '$.command'), ''
                )),
                '{bash_edit_re}'
            ) THEN 1::TINYINT
            ELSE 0::TINYINT
        END AS is_edit,
        (len(list_filter(
            j.gold_paths,
            p -> contains(coalesce(tc.function.arguments, ''), p)
              OR contains(coalesce(tc.function.arguments, ''), regexp_extract(p, '[^/]+$'))
        )) > 0)::TINYINT AS sees_gold
    FROM joined j,
         unnest(j.messages) WITH ORDINALITY AS u(m, mi),
         unnest(m.tool_calls) WITH ORDINALITY AS v(tc, ci)
),
call_ctx AS (
    SELECT c.*, ms.cum_assistant_chars
    FROM calls c
    LEFT JOIN msgs ms ON ms.trajectory_id = c.trajectory_id AND ms.mi = c.mi
),
call_agg AS (
    SELECT
        trajectory_id,
        count(*)::INTEGER AS n_calls_total{call_k_aggs}
    FROM call_ctx
    GROUP BY 1
),
msg_agg AS (
    SELECT
        trajectory_id{msg_k_aggs}
    FROM msgs
    GROUP BY 1
)
SELECT
    j.trajectory_id,
    '{harness}' AS harness,
    '{teacher}' AS teacher,
    '{source}' AS source,
    j.instance_id,
    j.resolved::TINYINT AS resolved,
    coalesce(ca.n_calls_total, 0)::INTEGER AS n_calls_total{select_k_cols}
FROM joined j
LEFT JOIN call_agg ca ON ca.trajectory_id = j.trajectory_id
LEFT JOIN msg_agg ma ON ma.trajectory_id = j.trajectory_id
"""


def build_prefix_sql(file_path: Path, harness: str, teacher: str, source: str) -> str:
    return PREFIX_SQL.format(
        file_sql=str(file_path).replace("'", "''"),
        harness=harness.replace("'", "''"),
        teacher=teacher.replace("'", "''"),
        source=source.replace("'", "''"),
        edit_view_only=EDIT_TOOL_VIEW_ONLY,
        edit_tool_names=_tuple_sql(EDIT_TOOL_NAMES),
        bash_edit_re=BASH_EDIT_RE.replace("'", "''"),
        error_re=ERROR_RE,
        call_k_aggs="".join(_call_k_aggs(k) for k in K_VALUES),
        msg_k_aggs="".join(_msg_k_aggs(k) for k in K_VALUES),
        select_k_cols="".join(_select_k_cols(k) for k in K_VALUES),
    )


def prefix_sanity(con: duckdb.DuckDBPyConnection, path: Path) -> dict:
    path_sql = _sql_str(path)
    checks = con.execute(
        f"""
        SELECT
            count(*)::BIGINT AS n_rows,
            count(DISTINCT trajectory_id)::BIGINT AS n_trajectories,
            avg(n_calls_total) AS mean_calls,
            median(n_calls_total) AS median_calls,
            count(*) FILTER (WHERE n_calls_total = 0)::BIGINT AS zero_call_shards,
            count(*) FILTER (WHERE repeat_rate_{K_VALUES[-1]} < 0
                OR repeat_rate_{K_VALUES[-1]} > 1)::BIGINT AS bad_repeat_rate,
            count(*) FILTER (WHERE any_error_{K_VALUES[0]})::BIGINT AS err_at_{K_VALUES[0]},
            count(*) FILTER (WHERE any_error_{K_VALUES[-1]})::BIGINT AS err_at_{K_VALUES[-1]},
            count(*) FILTER (WHERE gold_edit_{K_VALUES[-1]} = 1
                AND gold_view_{K_VALUES[-1]} = 0)::BIGINT AS edit_without_view,
            avg(CASE WHEN resolved IN (0, 1) THEN resolved::DOUBLE END) AS resolved_rate_known
        FROM read_parquet({path_sql})
        """
    ).fetchdf().iloc[0]
    breaks = " OR ".join(
        f"n_calls_{lo} > n_calls_{hi}" for lo, hi in itertools.pairwise(K_VALUES)
    )
    monotone = con.execute(
        f"SELECT count(*) FILTER (WHERE {breaks})::BIGINT FROM read_parquet({path_sql})"
    ).fetchone()[0]
    return {"checks": checks, "non_monotone": int(monotone)}


def stream_prefix_features(args: argparse.Namespace) -> int:
    """One streaming pass over the corpus shards; returns a process exit code."""
    PARTS_DIR.mkdir(parents=True, exist_ok=True)
    args.output.parent.mkdir(parents=True, exist_ok=True)

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
                status, rows = process_file(
                    con, file_path, force=args.force, parts_dir=PARTS_DIR, sql_builder=build_prefix_sql
                )
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
                f"rows={rows:,} in {elapsed:.1f}s (avg {rate:.1f}s, eta {fmt_duration(eta)})"
            )

        if failures:
            console.print(
                f"[{utcnow()}] {failed} shard(s) failed — not merging partial output; "
                "rerun to resume (finished parts are kept)"
            )
        elif not args.no_merge:
            merged = merge_parts(con, parts_dir=PARTS_DIR, out_path=args.output)
            if merged is None:
                console.print(f"[{utcnow()}] no parts to merge")
            else:
                n_parts, rows, tids = merged
                partial = bool(args.file or args.limit)
                suffix = (
                    " (partial run: rerun without --file/--limit to cover the corpus)"
                    if partial
                    else ""
                )
                console.print(
                    f"[{utcnow()}] merged {n_parts} parts → {rel(args.output)}: "
                    f"{rows:,} rows, {tids:,} trajectories{suffix}"
                )
                if args.head:
                    sanity = prefix_sanity(con, args.output)
                    console.print(f"prefix sanity: {sanity}")
    except KeyboardInterrupt:
        console.print(f"[{utcnow()}] interrupted — parts written so far are kept, rerun to resume")
        return 130
    finally:
        con.close()

    console.print(
        f"[{utcnow()}] prefix pass finished: processed={done} skipped={skipped} failed={failed} "
        f"in {fmt_duration(time.monotonic() - started)}"
    )
    if failures:
        console.print(f"[red]failed shards[/red]: {', '.join(failures)}")
        return 1
    return 0


def load_frame(
    prefix_path: Path = OUT_PARQUET,
    proxy_path: Path = PROXY_PARQUET,
    temporal_path: Path = TEMPORAL_PARQUET,
) -> pd.DataFrame:
    """Proxy + temporal + prefix features, in the exact `score.py` row order."""
    proxy_cols = ", ".join(f"p.{f}" for f in MODEL_FEATURES if f not in TEMPORAL_MODEL_FEATURES)
    temporal_cols = ", ".join(f"t.{f}" for f in TEMPORAL_MODEL_FEATURES)
    prefix_cols = ", ".join(f"pf.{f}" for f in PREFIX_FEATURES)
    con = connect_ephemeral()
    con.execute("SET memory_limit='8GB'")
    try:
        df = con.execute(
            f"""
            SELECT
                p.trajectory_id, p.instance_id, p.harness, p.teacher, p.source, p.resolved,
                {proxy_cols}, {temporal_cols}, {prefix_cols}
            FROM read_parquet({_sql_str(proxy_path)}) p
            LEFT JOIN read_parquet({_sql_str(temporal_path)}) t
                ON t.trajectory_id = p.trajectory_id
            LEFT JOIN read_parquet({_sql_str(prefix_path)}) pf
                ON pf.trajectory_id = p.trajectory_id
            ORDER BY p.harness, p.teacher, p.source, p.trajectory_id
            """
        ).df()
    finally:
        con.close()

    missing_prefix = int(df[PREFIX_FEATURES].isna().any(axis=1).sum())
    if missing_prefix:
        raise SystemExit(
            f"{missing_prefix:,} proxied trajectories have no prefix features in {prefix_path}; "
            "finish the prefix pass (and merge) before running the analyses"
        )
    missing_temporal = int(df["n_turns_total"].isna().sum())
    if missing_temporal:
        raise SystemExit(f"{missing_temporal:,} trajectories missing temporal features")
    for col in MODEL_FEATURES:
        df[col] = pd.to_numeric(df[col], errors="coerce")
    df[NULL_FILL] = df[NULL_FILL].fillna(0.0)
    if df[MODEL_FEATURES].isna().any().any():
        raise SystemExit("unexpected null features after imputation")
    return df


def load_difficulty(path: Path = DIFFICULTY_PARQUET) -> pd.DataFrame:
    con = connect_ephemeral()
    try:
        return con.execute(f"SELECT * FROM read_parquet({_sql_str(path)})").df()
    finally:
        con.close()


@dataclass
class ProbePair:
    probe: tuple[str, str]
    target: tuple[str, str]
    n_instances: int
    target_pos_share: float
    auc_k1: float
    rho: dict[int, float]


def labelled_rows(proxy: pd.DataFrame) -> pd.DataFrame:
    cols = ["instance_id", "harness", "teacher", "trajectory_id", "resolved"]
    out = proxy.loc[proxy["resolved"].isin([0, 1]), cols]
    return out.sort_values(cols[:4], ignore_index=True)


def combo_outcome_arrays(proxy: pd.DataFrame) -> dict[tuple[str, str], dict[str, np.ndarray]]:
    labeled = labelled_rows(proxy)
    out: dict[tuple[str, str], dict[str, np.ndarray]] = {}
    for (harness, teacher), sub in labeled.groupby(["harness", "teacher"], sort=True):
        out[(str(harness), str(teacher))] = {
            str(iid): g.to_numpy(dtype=np.int8)
            for iid, g in sub.groupby("instance_id", sort=True)["resolved"]
        }
    return out


def permuted_probe_arrays(
    arrays: dict[str, np.ndarray], seed: int = 0
) -> dict[str, np.ndarray]:
    """Per-instance stable random draw order: seed = (0, crc32(instance_id))."""
    out: dict[str, np.ndarray] = {}
    for iid, arr in arrays.items():
        rng = np.random.default_rng([seed, zlib.crc32(iid.encode("utf-8"))])
        out[iid] = arr[rng.permutation(len(arr))]
    return out


def _spearman(x: np.ndarray, y: np.ndarray) -> float:
    with warnings.catch_warnings():
        warnings.simplefilter("ignore")
        return float(spearmanr(x, y).statistic)


def pair_probe_stats(
    probe: dict[str, np.ndarray],
    target: dict[str, np.ndarray],
    probe_combo: tuple[str, str],
    target_combo: tuple[str, str],
) -> ProbePair:
    ids = sorted(
        iid
        for iid, arr in target.items()
        if len(arr) >= MIN_LABELED and len(probe.get(iid, ())) >= 1
    )
    if not ids:
        return ProbePair(probe_combo, target_combo, 0, np.nan, np.nan, {k: np.nan for k in PROBE_KS})
    probe_perm = [probe[iid] for iid in ids]
    target_rate = np.array([target[iid].mean() for iid in ids], dtype=float)
    means = {k: np.array([arr[:k].mean() for arr in probe_perm]) for k in PROBE_KS}
    y = (target_rate > 0.5).astype(int)
    auc = float(roc_auc_score(y, means[1])) if 0 < int(y.sum()) < len(y) else np.nan
    return ProbePair(
        probe=probe_combo,
        target=target_combo,
        n_instances=len(ids),
        target_pos_share=float(y.mean()),
        auc_k1=auc,
        rho={k: _spearman(means[k], target_rate) for k in PROBE_KS},
    )


def cross_teacher_report(proxy: pd.DataFrame) -> tuple[list[tuple[str, str]], list[ProbePair], ProbePair]:
    arrays = combo_outcome_arrays(proxy)
    combos = sorted(arrays, key=lambda c: (-len(arrays[c]), c))
    probes = {combo: permuted_probe_arrays(arrs) for combo, arrs in arrays.items()}
    pairs = [
        pair_probe_stats(probes[probe], arrays[target], probe, target)
        for probe in combos
        for target in combos
    ]
    exclude = PROBE_COMBO
    other = labelled_rows(proxy)
    other = other[~((other["harness"] == exclude[0]) & (other["teacher"] == exclude[1]))]
    other_arrays = {
        str(iid): g.to_numpy(dtype=np.int8)
        for iid, g in other.groupby("instance_id", sort=True)["resolved"]
    }
    cheapest = pair_probe_stats(
        probes[exclude], other_arrays, exclude, ("leave-one-out", "all other combos")
    )
    return combos, pairs, cheapest


def pair_matrix(
    pairs: list[ProbePair], combos: list[tuple[str, str]], value: str
) -> list[list[str]]:
    lookup = {(p.probe, p.target): p for p in pairs}
    rows = []
    for probe in combos:
        row = [short_label(probe)]
        for target in combos:
            p = lookup.get((probe, target))
            if p is None or p.n_instances == 0:
                row.append("n/a")
            elif value == "auc":
                row.append("n/a" if np.isnan(p.auc_k1) else f"{p.auc_k1:.3f}")
            else:
                k = int(value)
                row.append("n/a" if np.isnan(p.rho[k]) else f"{p.rho[k]:+.3f}")
        rows.append(row)
    return rows


@dataclass
class PrefixModelReport:
    n_labeled: int
    n_train: int
    n_test: int
    ref_full_trace_auc: float
    ref_task_rate_auc: float
    prefix_fits: dict[int, ModelFit]
    bucket_k: list[int]
    bucket_n: int
    bucket_mid_share: float
    bucket_aucs: dict[int, float]


def prefix_model_report(frame: pd.DataFrame, inst: pd.DataFrame) -> PrefixModelReport:
    labeled_mask = frame["resolved"].isin([0, 1])
    labeled = frame.loc[labeled_mask].reset_index(drop=True)
    y = labeled["resolved"].to_numpy(dtype=int)
    groups = labeled["instance_id"].astype(str).to_numpy()
    splitter = GroupShuffleSplit(n_splits=1, test_size=TEST_SIZE, random_state=SEED)
    train_idx, test_idx = next(splitter.split(np.zeros((len(labeled), 1)), y, groups))

    X_ref = labeled[MODEL_FEATURES].to_numpy(dtype=float)
    ref_fit = fit_logistic(
        X_ref[train_idx], y[train_idx], X_ref[test_idx], y[test_idx], MODEL_FEATURES
    )

    loo = (
        leave_one_out_solve_rate(frame)
        .loc[frame.index[labeled_mask]]
        .to_numpy(dtype=float)
    )
    has = ~np.isnan(loo)
    te = test_idx[has[test_idx]]
    ref_task_auc = float(roc_auc_score(y[te], loo[te]))

    prefix_fits: dict[int, ModelFit] = {}
    for k in K_VALUES:
        feats = k_features(k)
        X = labeled[feats].to_numpy(dtype=float)
        prefix_fits[k] = fit_logistic(
            X[train_idx], y[train_idx], X[test_idx], y[test_idx], feats
        )

    bucket = inst.set_index("instance_id")["difficulty_bucket"]
    sub = frame[["instance_id", *PREFIX_FEATURES]].copy()
    sub["bucket"] = bucket.reindex(sub["instance_id"]).to_numpy()
    sub = sub[sub["bucket"].isin(["mid", "all_fail", "all_pass"])]
    agg = sub.groupby("instance_id", sort=True)[PREFIX_FEATURES].mean()
    bucket_y = (
        sub.groupby("instance_id", sort=True)["bucket"].first().to_numpy() == "mid"
    ).astype(int)
    bucket_groups = agg.index.to_numpy()
    bucket_split = GroupShuffleSplit(n_splits=1, test_size=TEST_SIZE, random_state=SEED)
    b_train, b_test = next(
        bucket_split.split(np.zeros((len(agg), 1)), bucket_y, bucket_groups)
    )
    bucket_aucs: dict[int, float] = {}
    for k in K_VALUES:
        feats = k_features(k)
        X = agg[feats].to_numpy(dtype=float)
        fit = fit_logistic(X[b_train], bucket_y[b_train], X[b_test], bucket_y[b_test], feats)
        bucket_aucs[k] = float(fit.auc_test)

    return PrefixModelReport(
        n_labeled=len(labeled),
        n_train=len(train_idx),
        n_test=len(test_idx),
        ref_full_trace_auc=float(ref_fit.auc_test),
        ref_task_rate_auc=ref_task_auc,
        prefix_fits=prefix_fits,
        bucket_k=list(K_VALUES),
        bucket_n=len(agg),
        bucket_mid_share=float(bucket_y.mean()),
        bucket_aucs=bucket_aucs,
    )


BUCKET_RANGES: dict[str, tuple[float, float, bool, bool]] = {
    "all_fail": (0.0, 0.0, True, True),
    "all_pass": (1.0, 1.0, True, True),
    "hard": (0.0, HARD_MAX, True, False),
    "mid": (HARD_MAX, EASY_MIN, True, True),
    "easy": (EASY_MIN, 1.0, False, True),
}


def wilson_interval(successes: int, n: int, z: float = WILSON_Z) -> tuple[float, float]:
    if n <= 0:
        return (0.0, 1.0)
    p = successes / n
    denom = 1.0 + z * z / n
    center = (p + z * z / (2.0 * n)) / denom
    half = z * math.sqrt(p * (1.0 - p) / n + z * z / (4.0 * n * n)) / denom
    return (max(0.0, center - half), min(1.0, center + half))


def interval_bucket(lo: float, hi: float) -> str | None:
    for name, (blo, bhi, lo_inc, hi_inc) in BUCKET_RANGES.items():
        lo_ok = lo >= blo if lo_inc else lo > blo
        hi_ok = hi <= bhi if hi_inc else hi < bhi
        if lo_ok and hi_ok:
            return name
    return None


def point_bucket(rate: float, n: int) -> str:
    return str(difficulty_buckets(pd.Series([float(rate)]), pd.Series([int(n)])).iloc[0])


@dataclass
class DrawOutcome:
    n_used: int
    bucket: str
    stop: str


def simulate_wilson(order: np.ndarray, cap: int = SIM_CAP, z: float = WILSON_Z) -> DrawOutcome:
    n_avail = min(cap, len(order))
    successes = 0
    for i in range(1, n_avail + 1):
        successes += int(order[i - 1])
        lo, hi = wilson_interval(successes, i, z)
        if interval_bucket(lo, hi) is not None:
            return DrawOutcome(i, point_bucket(successes / i, i), "interval")
    if n_avail == 0:
        return DrawOutcome(0, "unknown", "exhausted")
    stop = "cap" if n_avail == cap else "exhausted"
    return DrawOutcome(n_avail, point_bucket(successes / n_avail, n_avail), stop)


def simulate_simple(order: np.ndarray, cap: int = SIMPLE_CAP, implied_two: bool = False) -> DrawOutcome:
    n_avail = min(cap, len(order))
    if n_avail >= 2 and int(order[0]) == int(order[1]):
        n = 2
        stop = "agree2"
    else:
        n = n_avail
        stop = "cap" if n_avail == cap else "exhausted"
    if n == 0:
        return DrawOutcome(0, "unknown", "exhausted")
    successes = int(order[:n].sum())
    if stop == "agree2" and implied_two:
        bucket = "all_fail" if successes == 0 else "all_pass"
        return DrawOutcome(n, bucket, stop)
    return DrawOutcome(n, point_bucket(successes / n, n), stop)


SIM_RULES = {
    "wilson-80-cap12": simulate_wilson,
    "agree2-else-4": simulate_simple,
    "agree2-else-4-implied": partial(simulate_simple, implied_two=True),
}


def simulate_rules(
    outcomes: dict[str, np.ndarray], full_bucket: dict[str, str], seeds: int = SIM_SEEDS
) -> pd.DataFrame:
    rows = []
    for seed in range(seeds):
        rng = np.random.default_rng(seed)
        for iid in sorted(outcomes):
            arr = outcomes[iid]
            order = arr[rng.permutation(len(arr))]
            for rule, fn in SIM_RULES.items():
                res = fn(order)
                rows.append(
                    {
                        "rule": rule,
                        "seed": seed,
                        "instance_id": iid,
                        "full_bucket": full_bucket[iid],
                        "n_used": res.n_used,
                        "bucket": res.bucket,
                        "stop": res.stop,
                    }
                )
    return pd.DataFrame(rows)


def simulation_per_seed(runs: pd.DataFrame) -> pd.DataFrame:
    d = runs.assign(
        ok=(runs["bucket"] == runs["full_bucket"]).astype(float),
        unknown=(runs["bucket"] == "unknown").astype(float),
        stop_cap=(runs["stop"] == "cap").astype(float),
        stop_agree=(runs["stop"] == "agree2").astype(float),
    )
    agg = d.groupby(["rule", "seed"], sort=True).agg(
        n_instances=("n_used", "size"),
        mean_rollouts=("n_used", "mean"),
        ok_sum=("ok", "sum"),
        unknown_sum=("unknown", "sum"),
        cap_sum=("stop_cap", "sum"),
        agree_sum=("stop_agree", "sum"),
    )
    agg["accuracy"] = agg["ok_sum"] / agg["n_instances"]
    agg["defined_accuracy"] = agg["ok_sum"] / (agg["n_instances"] - agg["unknown_sum"])
    agg["unknown_share"] = agg["unknown_sum"] / agg["n_instances"]
    agg["cap_share"] = agg["cap_sum"] / agg["n_instances"]
    agg["agree_share"] = agg["agree_sum"] / agg["n_instances"]
    return agg.reset_index()


def simulation_summary(per_seed: pd.DataFrame) -> pd.DataFrame:
    metrics = ["mean_rollouts", "accuracy", "defined_accuracy", "unknown_share"]
    out = per_seed.groupby("rule", sort=True)[metrics].agg(["mean", "std"])
    out.columns = [f"{m}_{stat}" for m, stat in out.columns]
    mix = per_seed.groupby("rule", sort=True)[["cap_share", "agree_share"]].mean()
    return out.join(mix)


def simulation_bucket_table(runs: pd.DataFrame) -> pd.DataFrame:
    d = runs.assign(ok=(runs["bucket"] == runs["full_bucket"]).astype(float))
    agg = d.groupby(["rule", "full_bucket"], sort=True).agg(
        instances=("n_used", "size"),
        mean_rollouts=("n_used", "mean"),
        accuracy=("ok", "mean"),
    )
    return agg.reset_index()


def load_sim_inputs(proxy: pd.DataFrame, inst: pd.DataFrame) -> tuple[dict[str, np.ndarray], dict[str, str]]:
    labeled = labelled_rows(proxy)
    counts = labeled.groupby("instance_id", sort=True)["resolved"].size()
    keep = counts[counts >= SIM_MIN_LABELED].index
    sub = labeled[labeled["instance_id"].isin(keep)]
    outcomes = {
        str(iid): g.to_numpy(dtype=np.int8)
        for iid, g in sub.groupby("instance_id", sort=True)["resolved"]
    }
    bucket = inst.set_index("instance_id")["difficulty_bucket"]
    full_bucket = {iid: str(bucket[iid]) for iid in outcomes}
    return outcomes, full_bucket


@dataclass
class PrefixPassStats:
    n_rows: int
    n_trajectories: int
    mean_calls: float
    median_calls: float
    zero_call_share: float
    coverage: dict[int, float]
    feature_means: pd.DataFrame
    sanity: dict


def prefix_pass_stats(con: duckdb.DuckDBPyConnection, path: Path) -> PrefixPassStats:
    path_sql = _sql_str(path)
    row = con.execute(
        f"""
        SELECT
            count(*)::BIGINT AS n_rows,
            count(DISTINCT trajectory_id)::BIGINT AS n_trajectories,
            avg(n_calls_total) AS mean_calls,
            median(n_calls_total) AS median_calls,
            avg(CASE WHEN n_calls_total = 0 THEN 1.0 ELSE 0.0 END) AS zero_call_share
        FROM read_parquet({path_sql})
        """
    ).fetchdf().iloc[0]
    coverage = {}
    for k in K_VALUES:
        coverage[k] = float(
            con.execute(
                f"SELECT avg(CASE WHEN n_calls_total >= {k} THEN 1.0 ELSE 0.0 END) "
                f"FROM read_parquet({path_sql})"
            ).fetchone()[0]
        )
    mean_cols = ", ".join(
        f"avg(CASE WHEN {f} THEN 1.0 ELSE 0.0 END) AS {f}"
        if f.startswith("any_error")
        else f"avg({f}) AS {f}"
        for f in PREFIX_FEATURES
    )
    feature_means = con.execute(
        f"SELECT {mean_cols} FROM read_parquet({path_sql})"
    ).fetchdf()
    sanity = prefix_sanity(con, path)
    return PrefixPassStats(
        n_rows=int(row["n_rows"]),
        n_trajectories=int(row["n_trajectories"]),
        mean_calls=float(row["mean_calls"]),
        median_calls=float(row["median_calls"]),
        zero_call_share=float(row["zero_call_share"]),
        coverage=coverage,
        feature_means=feature_means,
        sanity=sanity,
    )


def build_summary(
    frame: pd.DataFrame,
    proxy: pd.DataFrame,
    inst: pd.DataFrame,
    stats: PrefixPassStats,
    combos: list[tuple[str, str]],
    pairs: list[ProbePair],
    cheapest: ProbePair,
    models: PrefixModelReport,
    per_seed: pd.DataFrame,
    sim_summary: pd.DataFrame,
    sim_buckets: pd.DataFrame,
    out_shown: str,
) -> str:
    lines: list[str] = []
    stamp = datetime.now(UTC).strftime("%Y-%m-%d %H:%M UTC")
    lines.append("# Cheap task difficulty — how few rollouts identify an instance's bucket")
    lines.append("")
    lines.append(
        f"Generated {stamp} by `scripts/cheap_difficulty.py` from "
        "`outputs/proxy_features.parquet`, `outputs/task_difficulty.parquet`, and "
        f"`{out_shown}` (streamed from `traces_data/`)."
    )
    lines.append("")
    lines.append(
        f"Corpus: {len(proxy):,} trajectories, {len(inst):,} instances, "
        f"{int(proxy['resolved'].isin([0, 1]).sum()):,} labeled rollouts; "
        f"{len(combos)} labeled harness/teacher combos."
    )
    lines.append("")

    lines.append("## A. Cross-teacher probe")
    lines.append("")
    lines.append(
        "For every ordered pair of labeled combos: a single random probe rollout (seed 0, "
        "per-instance stable draw order) predicts whether the target combo's solve_rate over "
        "its >= 3 labeled rollouts exceeds 0.5 (AUC, ties included); probe means over the "
        "first k = 1, 2, 3 rollouts of the same order are Spearman-correlated with the target "
        "rate. `n` is the number of instances with >= 1 labeled probe rollout and >= 3 labeled "
        "target rollouts."
    )
    lines.append("")
    lines.append("### A1. AUC(k=1 probe rollout) — full pair matrix")
    lines.append("")
    header = ["probe \\ target", *[short_label(c) for c in combos]]
    lines.append(md_table(header, pair_matrix(pairs, combos, "auc")))
    lines.append("")
    lines.append("### A2. Spearman(probe mean k=1, target solve_rate) — full pair matrix")
    lines.append("")
    lines.append(md_table(header, pair_matrix(pairs, combos, "1")))
    lines.append("")
    off = [p for p in pairs if p.probe != p.target and p.n_instances > 0 and not np.isnan(p.auc_k1)]
    diag = [p for p in pairs if p.probe == p.target and p.n_instances > 0 and not np.isnan(p.auc_k1)]
    lines.append(
        f"Off-diagonal pairs: {len(off)}; mean AUC "
        f"{np.mean([p.auc_k1 for p in off]):.3f} (min {min(p.auc_k1 for p in off):.3f}, "
        f"max {max(p.auc_k1 for p in off):.3f}), mean Spearman k=1 "
        f"{np.mean([p.rho[1] for p in off]):+.3f}. Self-pairs (same combo, different "
        f"rollouts): mean AUC {np.mean([p.auc_k1 for p in diag]):.3f}, mean Spearman k=1 "
        f"{np.mean([p.rho[1] for p in diag]):+.3f}."
    )
    lines.append("")
    lines.append("### A3. All pairs (k = 1, 2, 3 probe rollouts)")
    lines.append("")
    rows = []
    for p in pairs:
        rows.append(
            [
                short_label(p.probe),
                short_label(p.target),
                f"{p.n_instances:,}",
                "n/a" if np.isnan(p.target_pos_share) else f"{p.target_pos_share:.3f}",
                "n/a" if np.isnan(p.auc_k1) else f"{p.auc_k1:.3f}",
                *[
                    "n/a" if np.isnan(p.rho[k]) else f"{p.rho[k]:+.3f}"
                    for k in PROBE_KS
                ],
            ]
        )
    lines.append(
        md_table(
            ["probe", "target", "n", "target >0.5 share", "AUC k=1", "rho k=1", "rho k=2", "rho k=3"],
            rows,
        )
    )
    lines.append("")
    lines.append("### A4. Cheapest teacher as the probe")
    lines.append("")
    lines.append(
        f"Probe = single rollout of `{PROBE_COMBO[0]}/{PROBE_COMBO[1]}`; target = "
        "leave-this-combo-out solve_rate over all other combos (>= 3 labeled elsewhere)."
    )
    lines.append("")
    rows = [
        [
            f"{cheapest.probe[0]}/{cheapest.probe[1]}",
            cheapest.target[1],
            f"{cheapest.n_instances:,}",
            "n/a" if np.isnan(cheapest.target_pos_share) else f"{cheapest.target_pos_share:.3f}",
            "n/a" if np.isnan(cheapest.auc_k1) else f"{cheapest.auc_k1:.4f}",
            *[
                "n/a" if np.isnan(cheapest.rho[k]) else f"{cheapest.rho[k]:+.4f}"
                for k in PROBE_KS
            ],
        ]
    ]
    lines.append(
        md_table(
            ["probe", "target", "n", "target >0.5 share", "AUC k=1", "rho k=1", "rho k=2", "rho k=3"],
            rows,
        )
    )
    lines.append("")

    lines.append("## B. Partial-rollout probe")
    lines.append("")
    lines.append(
        f"Prefix features at k tool calls: gold file viewed/edited, n edit calls, n distinct "
        f"commands, repeat rate, assistant chars, any tool observation matching "
        f"`{ERROR_RE}` (case-insensitive). One streaming pass over the corpus: "
        f"{stats.n_rows:,} rows / {stats.n_trajectories:,} distinct trajectories, "
        f"mean {stats.mean_calls:.1f} tool calls (median {stats.median_calls:.0f}; "
        f"{pct(stats.zero_call_share)} with zero calls). Sanity: "
        f"{stats.sanity['non_monotone']:,} non-monotone prefix rows, "
        f"{int(stats.sanity['checks']['edit_without_view']):,} gold-edit-without-view, "
        f"{int(stats.sanity['checks']['bad_repeat_rate']):,} bad repeat rates."
    )
    lines.append("")
    rows = [
        [
            str(k),
            pct(stats.coverage[k]),
            f"{stats.feature_means[f'gold_view_{k}'].iloc[0]:.3f}",
            f"{stats.feature_means[f'gold_edit_{k}'].iloc[0]:.3f}",
            f"{stats.feature_means[f'n_edit_{k}'].iloc[0]:.2f}",
            f"{stats.feature_means[f'n_distinct_{k}'].iloc[0]:.2f}",
            f"{stats.feature_means[f'repeat_rate_{k}'].iloc[0]:.3f}",
            f"{stats.feature_means[f'assistant_chars_{k}'].iloc[0]:,.0f}",
            f"{stats.feature_means[f'any_error_{k}'].iloc[0]:.3f}",
        ]
        for k in K_VALUES
    ]
    lines.append(
        md_table(
            [
                "k",
                "share with >=k calls",
                "gold viewed",
                "gold edited",
                "edits",
                "distinct cmds",
                "repeat rate",
                "assistant chars",
                "any error",
            ],
            rows,
        )
    )
    lines.append("")
    lines.append("### B1. Prefix-only classifier of `resolved`")
    lines.append("")
    lines.append(
        f"Standardized (winsorized 1st/99th, mean 0 / sd 1) logistic regression per k, "
        f"grouped 80/20 split by instance_id (`random_state={SEED}`): {models.n_labeled:,} "
        f"labeled rows, {models.n_train:,} train / {models.n_test:,} test. References on the "
        f"same split: all 27 full-trace features AUC {models.ref_full_trace_auc:.4f} "
        f"(published {REF_FULL_TRACE_AUC:.4f}), leave-one-out task rate AUC "
        f"{models.ref_task_rate_auc:.4f} (published {REF_TASK_RATE_AUC:.4f})."
    )
    lines.append("")
    best_k = max(models.prefix_fits, key=lambda k: models.prefix_fits[k].auc_test)
    rows = [
        [
            str(k),
            f"{models.prefix_fits[k].auc_test:.4f}",
            f"{models.prefix_fits[k].auc_test - models.ref_full_trace_auc:+.4f}",
            f"{(models.prefix_fits[k].auc_test - 0.5) / (models.ref_full_trace_auc - 0.5):.1%}",
        ]
        for k in K_VALUES
    ]
    rows.append(["full trace (27 features)", f"{models.ref_full_trace_auc:.4f}", "—", "100.0%"])
    rows.append(
        ["task solve_rate alone (leave-one-out, 1 feature)", f"{models.ref_task_rate_auc:.4f}", "—", "—"]
    )
    lines.append(
        md_table(
            ["predictor", "held-out AUC", "Δ vs full trace", "signal captured vs full trace"],
            rows,
        )
    )
    lines.append("")
    lines.append(f"Best prefix k = {best_k} (AUC {models.prefix_fits[best_k].auc_test:.4f}).")
    lines.append("")
    lines.append("### B2. Prefix-only classifier of the instance bucket (mid vs degenerate)")
    lines.append("")
    lines.append(
        f"Instance-level: features are means over the instance's rollouts; positive class = "
        f"`mid`, negative = `all_fail` or `all_pass` (hard/easy/unknown dropped). "
        f"{models.bucket_n:,} instances, mid share {models.bucket_mid_share:.3f}; grouped 80/20 "
        f"by instance_id."
    )
    lines.append("")
    rows = [
        [str(k), f"{models.bucket_aucs[k]:.4f}"] for k in models.bucket_k
    ]
    lines.append(md_table(["k", "held-out AUC (mid vs degenerate)"], rows))
    lines.append("")

    lines.append("## C. Adaptive sampling simulation")
    lines.append("")
    lines.append(
        f"Instances with >= {SIM_MIN_LABELED} labeled rollouts; rollouts drawn in random "
        f"order (5 seeds). Wilson rule: stop when the {WILSON_CONF:.0%} Wilson interval for "
        f"solve_rate fits entirely inside one bucket (all_fail/hard/mid/easy/all_pass ranges as "
        f"in `difficulty.py`), cap {SIM_CAP}. Simple rule: stop after 2 agreeing outcomes, "
        f"else {SIMPLE_CAP}. The final bucket is `difficulty_buckets(point rate, n drawn)` — "
        f"the interval only decides when to stop; `agree2-else-4` keeps that rule (a 2-draw "
        f"stop cannot satisfy the >= {MIN_LABELED} labeled-rollout requirement and lands in "
        f"`unknown`), while `agree2-else-4-implied` reads two agreeing failures as `all_fail` "
        f"and two agreeing successes as `all_pass`. Metrics are mean ± sd across seeds."
    )
    lines.append("")
    rows = []
    for rule in SIM_RULES:
        s = sim_summary.loc[rule]
        rows.append(
            [
                rule,
                f"{s['mean_rollouts_mean']:.2f} ± {s['mean_rollouts_std']:.2f}",
                f"{s['accuracy_mean']:.3f} ± {s['accuracy_std']:.3f}",
                f"{s['defined_accuracy_mean']:.3f} ± {s['defined_accuracy_std']:.3f}",
                f"{s['unknown_share_mean']:.3f} ± {s['unknown_share_std']:.3f}",
                f"{s['cap_share']:.3f}",
                f"{s['agree_share']:.3f}",
            ]
        )
    lines.append(
        md_table(
            [
                "rule",
                "mean rollouts",
                "bucket accuracy",
                "accuracy (defined only)",
                "unknown share",
                "cap share",
                "agree2 share",
            ],
            rows,
        )
    )
    lines.append("")
    lines.append("### C1. By full-data bucket (pooled over seeds)")
    lines.append("")
    rows = []
    for _, r in sim_buckets.iterrows():
        if r["rule"] not in SIM_RULES:
            continue
        rows.append(
            [
                str(r["rule"]),
                str(r["full_bucket"]),
                f"{int(r['instances']):,}",
                f"{r['mean_rollouts']:.2f}",
                f"{r['accuracy']:.3f}",
            ]
        )
    lines.append(md_table(["rule", "full bucket", "instance-draws", "mean rollouts", "accuracy"], rows))
    lines.append("")
    lines.append("### C2. Per-seed detail")
    lines.append("")
    rows = [
        [
            str(r["rule"]),
            str(int(r["seed"])),
            f"{r['mean_rollouts']:.2f}",
            f"{r['accuracy']:.3f}",
            f"{r['defined_accuracy']:.3f}",
            f"{r['unknown_share']:.3f}",
        ]
        for _, r in per_seed.sort_values(["rule", "seed"]).iterrows()
    ]
    lines.append(
        md_table(["rule", "seed", "mean rollouts", "accuracy", "defined accuracy", "unknown share"], rows)
    )
    lines.append("")

    lines.append("## Notes")
    lines.append("")
    lines.append(
        "- Analysis B streams the corpus shard by shard (DuckDB, 6 GB cap); parts under "
        "`outputs/prefix_features_parts/` are atomic and skipped on rerun, so an interrupted "
        "pass resumes where it stopped."
    )
    lines.append(
        "- Prefix semantics: a feature at k uses tool calls 1..k (and the assistant turn "
        "containing call k); trajectories with fewer than k calls repeat their full-trajectory "
        "values; observed errors count tool observations whose position is at or before call k."
    )
    lines.append(
        "- In `difficulty.py` the extremes are point buckets (`all_fail` is exactly 0, "
        "`all_pass` exactly 1), so a Wilson interval can never sit inside them; the extremes "
        f"are certified through the `hard` / `easy` ranges instead (0/{SIMPLE_CAP} fits inside "
        f"`hard`, {SIMPLE_CAP}/{SIMPLE_CAP} inside `easy`), and the final assignment falls "
        "back to the point estimate. Mid-rate tasks exhaust the cap."
    )
    lines.append(
        "- The simple rule stops at 2 agreeing outcomes; `difficulty.py` requires >= "
        f"{MIN_LABELED} labeled rollouts for a bucket, so under the canonical rule those stops "
        "are `unknown` (hence the low raw accuracy of `agree2-else-4`) — the `-implied` variant "
        "shows the same rule with the two agreeing draws read as the extreme buckets."
    )
    lines.append(
        "- `any_error` is a loose substring test (`error|traceback|failed`, case-insensitive), "
        "so it also fires on benign mentions (e.g. pytest's `0 failed`); it saturates quickly "
        "(0.80 of trajectories by call 5, 0.99 by call 40) and carries little signal."
    )
    lines.append(
        "- `assistant_chars` understates combos whose assistant text lives in reasoning fields "
        "(e.g. `openhands/deepseek_v4_flash`)."
    )
    lines.append(
        "- Reproduce: `uv run python scripts/cheap_difficulty.py --stream-only` (corpus pass, "
        "~76 min, resume-safe), then `uv run python scripts/cheap_difficulty.py --skip-stream`."
    )
    lines.append("")
    return "\n".join(lines)


def main_cheap_difficulty() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--proxy", type=Path, default=PROXY_PARQUET)
    parser.add_argument("--temporal", type=Path, default=TEMPORAL_PARQUET)
    parser.add_argument("--difficulty", type=Path, default=DIFFICULTY_PARQUET)
    parser.add_argument("--output", type=Path, default=OUT_PARQUET)
    parser.add_argument("--summary", type=Path, default=OUT_MD)
    parser.add_argument("--file", action="append", help="Process only these shard(s); repeatable")
    parser.add_argument("--limit", type=int, help="Process at most N shards (sorted order)")
    parser.add_argument("--data-glob", default=PARQUET_GLOB, help="Parquet glob (default: corpus)")
    parser.add_argument("--force", action="store_true", help="Recompute prefix parts")
    parser.add_argument("--no-merge", action="store_true", help="Leave shard parts unmerged")
    parser.add_argument("--merge-only", action="store_true", help="Skip processing; merge parts")
    parser.add_argument("--head", type=int, default=0, help="Print sanity summary after merging")
    parser.add_argument("--threads", type=int, default=4, help="DuckDB threads for this batch job")
    parser.add_argument("--memory-limit", default="6GB", help="DuckDB memory limit")
    parser.add_argument("--stream-only", action="store_true", help="Run the corpus pass and exit")
    parser.add_argument("--skip-stream", action="store_true", help="Reuse the prefix parquet")
    args = parser.parse_args()

    args.proxy = args.proxy if args.proxy.is_absolute() else ROOT / args.proxy
    args.temporal = args.temporal if args.temporal.is_absolute() else ROOT / args.temporal
    args.difficulty = args.difficulty if args.difficulty.is_absolute() else ROOT / args.difficulty
    args.output = args.output if args.output.is_absolute() else ROOT / args.output
    args.summary = args.summary if args.summary.is_absolute() else ROOT / args.summary

    if args.stream_only and args.skip_stream:
        parser.error("--stream-only and --skip-stream are mutually exclusive")

    if args.merge_only:
        con = connect_ephemeral()
        try:
            merged = merge_parts(con, parts_dir=PARTS_DIR, out_path=args.output)
            if merged is None:
                raise SystemExit(f"No parts found in {PARTS_DIR}")
            n_parts, rows, tids = merged
            console.print(
                f"[{utcnow()}] merged {n_parts} parts → {rel(args.output)}: "
                f"{rows:,} rows, {tids:,} trajectories"
            )
            if args.head:
                console.print(f"prefix sanity: {prefix_sanity(con, args.output)}")
        finally:
            con.close()
        return

    if not args.skip_stream:
        rc = stream_prefix_features(args)
        if rc != 0 or args.stream_only:
            sys.exit(rc)
    if not args.output.exists():
        raise SystemExit(f"No prefix features at {args.output}; run without --skip-stream")

    proxy = load_difficulty_proxy(args.proxy)
    inst = load_difficulty(args.difficulty)
    console.print(f"Loaded {len(proxy):,} trajectories and {len(inst):,} instances")

    frame = load_frame(args.output, args.proxy, args.temporal)
    console.print(f"Joined prefix frame: {len(frame):,} rows")

    combos, pairs, cheapest = cross_teacher_report(proxy)
    off = [p for p in pairs if p.probe != p.target and p.n_instances > 0 and not np.isnan(p.auc_k1)]
    console.print(
        f"A: {len(combos)} labeled combos, mean off-diagonal AUC(k=1) "
        f"{np.mean([p.auc_k1 for p in off]):.3f}; cheapest-teacher probe AUC {cheapest.auc_k1:.4f} "
        f"over {cheapest.n_instances:,} instances"
    )

    models = prefix_model_report(frame, inst)
    console.print(
        "B: held-out AUC " + " | ".join(
            f"k={k} {models.prefix_fits[k].auc_test:.4f}" for k in K_VALUES
        )
    )
    console.print(
        f"B references: full trace {models.ref_full_trace_auc:.4f} "
        f"(published {REF_FULL_TRACE_AUC}), task rate {models.ref_task_rate_auc:.4f} "
        f"(published {REF_TASK_RATE_AUC}); bucket AUC " + " | ".join(
            f"k={k} {models.bucket_aucs[k]:.4f}" for k in K_VALUES
        )
    )

    con = connect_ephemeral()
    try:
        stats = prefix_pass_stats(con, args.output)
    finally:
        con.close()

    outcomes, full_bucket = load_sim_inputs(proxy, inst)
    runs = simulate_rules(outcomes, full_bucket)
    per_seed = simulation_per_seed(runs)
    sim_summary = simulation_summary(per_seed)
    sim_buckets = simulation_bucket_table(runs)
    console.print(f"C: simulated {len(outcomes):,} instances x {SIM_SEEDS} seeds")
    for rule in SIM_RULES:
        s = sim_summary.loc[rule]
        console.print(
            f"  {rule}: {s['mean_rollouts_mean']:.2f}±{s['mean_rollouts_std']:.2f} rollouts, "
            f"accuracy {s['accuracy_mean']:.3f}±{s['accuracy_std']:.3f}, "
            f"unknown {s['unknown_share_mean']:.3f}"
        )

    md = build_summary(
        frame,
        proxy,
        inst,
        stats,
        combos,
        pairs,
        cheapest,
        models,
        per_seed,
        sim_summary,
        sim_buckets,
        rel(args.output),
    )
    args.summary.parent.mkdir(parents=True, exist_ok=True)
    args.summary.write_text(md)
    console.print(f"Wrote {rel(args.summary)}")


def load_difficulty_proxy(path: Path) -> pd.DataFrame:
    con = connect_ephemeral()
    try:
        return con.execute(
            f"SELECT trajectory_id, instance_id, harness, teacher, source, resolved "
            f"FROM read_parquet({_sql_str(path)})"
        ).df()
    finally:
        con.close()


if __name__ == "__main__":
    main_cheap_difficulty()

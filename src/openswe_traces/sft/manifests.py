"""Equal-supervised-char-budget SFT manifests for the model-size curve.

Four arms draw on the same corpus slice — traces whose task is learnable
(``difficulty_bucket`` in hard/mid/easy) — each with the same supervised-char budget
and the same language mix as the corpus (the budget is allocated per language by the
corpus language share of supervised chars):

  random              uniform over traces
  top_within_task     per (instance_id, harness, teacher): the labeled sibling with the
                      highest ``p_resolved``; ties by fewer assistant turns, then
                      trajectory id
  bottom_within_task  per (instance_id, harness, teacher): the lowest ``p_resolved``
  random_masked       the ``random`` traces, with a per-assistant-message ``mask`` flag
                      set for repeated identical tool calls, tool observations
                      containing Error/Traceback/FAILED, and edit calls touching no
                      gold file

Tasks (instances) are sampled without replacement with weights ``mid`` 2×, ``hard`` 1×,
``easy`` 1× (Efraimidis--Spirakis keys over a task-level shuffle), and a sampled task
contributes its traces in rule order — the within-task arms keep one representative per
``(harness, teacher)`` pair, so every teacher that ran a task is represented whenever
that task is drawn. The real-run budget is sized so each manifest holds ~3,000 traces:
``train_curve.py`` samples one window per trace per pass and an 11 h 2×T4 session runs
~3,000 windows, so one pass ≈ one session.

``eval_heldout.jsonl`` holds one trace per instance for instances that appear in no
manifest, balanced across buckets; its cross-entropy is the arm-comparable metric used
by ``experiments/curve/train_curve.py``. ``eval_heldout_small.jsonl`` is a seeded
20-per-bucket subset of it (60 traces) for the frequent in-run evals, so the final
full-300 CE and the in-run small CE share the same windows for the 60.

Budget unit: supervised chars = assistant content chars (``outputs/proxy_features.parquet``)
plus, per assistant tool call, name + argument chars and a structural allowance — a
monotone proxy for the supervised token count (tokens ~= chars / 4). Tool observations
are excluded because they are loss-masked at train time.

Messages are written in the JSONL format of :mod:`openswe_traces.sft.sample` (tool
observations truncated, tool-call arguments parsed to dicts) because the Kaggle script
``experiments/curve/train_curve.py`` consumes that format.

Examples:
  uv run python experiments/curve/build_manifests.py --budget-chars 2000000
  uv run python experiments/curve/build_manifests.py --budget-chars 20000000 --refresh-cache
"""

from __future__ import annotations

import argparse
import json
import random
import re
from pathlib import Path

import pandas as pd
from rich.console import Console

from ..data import PARQUET_GLOB, ROOT, connect_ephemeral
from ..difficulty import LANGUAGE_ALIASES
from ..features import BASH_EDIT_RE, EDIT_TOOL_NAMES, EDIT_TOOL_VIEW_ONLY, _sql_str

console = Console()

PROXY_PARQUET = ROOT / "outputs" / "proxy_features.parquet"
SCORES_PARQUET = ROOT / "outputs" / "trace_scores.parquet"
DIFFICULTY_PARQUET = ROOT / "outputs" / "task_difficulty.parquet"
CANDIDATES_CACHE = ROOT / "outputs" / "manifests_candidates.parquet"
OUT_DIR = ROOT / "experiments" / "curve" / "data"

ARMS = ("random", "top_within_task", "bottom_within_task", "random_masked")
RANDOM_ARMS = ("random", "random_masked")
MASK_RULE = "random_masked"
ELIGIBLE_BUCKETS = ("hard", "mid", "easy")
BUCKET_WEIGHTS = {"mid": 2.0, "hard": 1.0, "easy": 1.0}
DEFAULT_BUDGET_CHARS = 165_000_000
DEFAULT_SEED = 0
TOOL_MAX_CHARS = 1500
TOOL_CALL_OVERHEAD_CHARS = 60
EVAL_N = 300
EVAL_SMALL_N = 60
FIT_BAND = (0.98, 1.02)

GOLD_GIT_RE = re.compile(r"diff --git a/([^\s]+) b/")
GOLD_PLUS_RE = re.compile(r"\+\+\+ b/([^\s]+)")

SCAN_TMPL = """
WITH msgs AS (
    SELECT trajectory_id, filename, unnest(messages) AS m
    FROM read_parquet({glob}, union_by_name=true, filename=true)
),
tc AS (
    SELECT
        trajectory_id,
        any_value(filename) AS shard,
        coalesce(
            sum(
                list_sum(
                    list_transform(
                        m.tool_calls,
                        call -> length(coalesce(call.function.arguments, ''))
                                + length(coalesce(call.function.name, ''))
                                + {overhead}
                    )
                )
            ) FILTER (WHERE m.role = 'assistant'),
            0
        )::BIGINT AS tool_call_chars
    FROM msgs
    GROUP BY 1
)
SELECT
    p.trajectory_id,
    tc.shard,
    p.instance_id,
    p.harness,
    p.teacher,
    coalesce(p.language, d.language) AS language,
    p.resolved,
    p.n_assistant_turns,
    (p.assistant_chars + tc.tool_call_chars)::BIGINT AS supervised_chars,
    s.p_resolved,
    d.difficulty_bucket
FROM read_parquet({proxy}) p
JOIN tc ON tc.trajectory_id = p.trajectory_id
JOIN read_parquet({scores}) s ON s.trajectory_id = p.trajectory_id
JOIN read_parquet({difficulty}) d ON d.instance_id = p.instance_id
WHERE d.difficulty_bucket IN ({buckets})
  AND p.assistant_chars + tc.tool_call_chars > 0
"""


def load_candidates(*, refresh: bool = False, cache: Path = CANDIDATES_CACHE) -> pd.DataFrame:
    """Per-trace selection frame: identifiers, supervised chars, p_resolved, bucket.

    One streaming pass over the corpus computes per-assistant tool-call chars, which are
    joined onto the proxy features (assistant content chars) and the difficulty buckets.
    The result is cached to ``outputs/manifests_candidates.parquet`` for reruns.
    """
    cache = Path(cache)
    if cache.exists() and not refresh:
        console.print(f"candidates from cache {cache.relative_to(ROOT)}")
        df = pd.read_parquet(cache)
    else:
        buckets = ", ".join(f"'{b}'" for b in ELIGIBLE_BUCKETS)
        sql = SCAN_TMPL.format(
            glob=_sql_str(PARQUET_GLOB),
            proxy=_sql_str(PROXY_PARQUET),
            scores=_sql_str(SCORES_PARQUET),
            difficulty=_sql_str(DIFFICULTY_PARQUET),
            buckets=buckets,
            overhead=TOOL_CALL_OVERHEAD_CHARS,
        )
        console.print("scanning corpus for supervised chars (one streaming pass) ...")
        con = connect_ephemeral()
        try:
            df = con.execute(sql).df()
        finally:
            con.close()
        cache.parent.mkdir(parents=True, exist_ok=True)
        df.to_parquet(cache, index=False)
        console.print(f"candidates {len(df):,} → cached {cache.relative_to(ROOT)}")
    df["language"] = df["language"].astype(str).replace(LANGUAGE_ALIASES)
    return df


def language_shares(candidates: pd.DataFrame) -> dict[str, float]:
    """Corpus language mix, weighted by supervised chars (the budget unit)."""
    by = candidates.groupby("language", observed=True)["supervised_chars"].sum()
    by = by.sort_values(ascending=False)
    total = float(by.sum())
    return {str(lang): float(chars / total) for lang, chars in by.items()}


def _task_weights(pool: pd.DataFrame, rng: random.Random) -> dict[str, float]:
    """Efraimidis--Spirakis key per task: a bucket-weighted shuffle of the tasks.

    Sampling tasks in descending key order is weighted sampling without replacement
    with ``BUCKET_WEIGHTS`` (mid 2x, hard/easy 1x): a task's key is ``u**(1/w)`` for
    ``u ~ Uniform(0, 1)``, so higher-weight buckets sort earlier in expectation.
    """
    buckets = pool.groupby("instance_id", sort=False)["difficulty_bucket"].first()
    return {
        str(task): rng.random() ** (1.0 / BUCKET_WEIGHTS[str(bucket)])
        for task, bucket in buckets.items()
    }


def _within_task(sub: pd.DataFrame, rule: str, rng: random.Random) -> pd.DataFrame:
    """One task's traces in selection order (cut at the budget if the task is last).

    Random arms: round-robin over shuffled (harness, teacher) groups, so both (all)
    teachers of a task appear before the task's duplicates. Within-task arms: one
    labeled representative per (harness, teacher) — the highest/lowest ``p_resolved``,
    ties by fewer assistant turns, then trajectory id.
    """
    if rule in RANDOM_ARMS:
        groups: list[list] = []
        for _, g in sub.groupby(["harness", "teacher"], sort=True):
            idx = list(g.index)
            rng.shuffle(idx)
            groups.append(idx)
        rng.shuffle(groups)
        order: list = []
        for i in range(max((len(g) for g in groups), default=0)):
            order.extend(g[i] for g in groups if i < len(g))
        return sub.loc[order]
    ascending = rule == "bottom_within_task"
    return sub.sort_values(
        ["p_resolved", "n_assistant_turns", "trajectory_id"],
        ascending=[ascending, True, True],
        kind="mergesort",
    ).drop_duplicates(subset=["harness", "teacher"], keep="first")


def _trace_order(pool: pd.DataFrame, rule: str, rng: random.Random) -> pd.DataFrame:
    """Selection order within one language: tasks bucket-weight-ordered, traces per rule."""
    if rule not in ARMS:
        raise ValueError(f"unknown rule {rule!r}; expected one of {ARMS}")
    if rule not in RANDOM_ARMS:
        pool = pool[pool["p_resolved"].notna() & pool["resolved"].isin([0, 1])]
        if pool.empty:
            return pool
    grouped = pool.groupby("instance_id", sort=False)
    keys = _task_weights(pool, rng)
    frames = [
        _within_task(grouped.get_group(task), rule, rng)
        for task in sorted(keys, key=lambda t: (-keys[t], t))
    ]
    return pd.concat(frames) if frames else pool.iloc[:0]


def fit_budget(picked: pd.DataFrame, budget_chars: int) -> pd.DataFrame:
    """Trim a greedy per-language fill that overshot the budget back toward it.

    Traces are added until each language quota is crossed, so the total overshoots by at
    most one trace per language. Drop traces (largest first) while that lands the total
    inside ``FIT_BAND`` of the budget; left alone when it cannot (tiny budgets).
    """
    lo, hi = FIT_BAND
    total = int(picked["supervised_chars"].sum())
    if lo * budget_chars <= total <= hi * budget_chars or len(picked) <= 1:
        return picked
    sizes = picked["supervised_chars"].astype("int64")
    for idx in sizes.sort_values(ascending=False).index:
        rest = total - int(sizes.loc[idx])
        if lo * budget_chars <= rest <= hi * budget_chars:
            return picked.drop(index=idx)
    keep = picked
    while total > hi * budget_chars and len(keep) > 1:
        idx = keep["supervised_chars"].astype("int64").idxmax()
        rest = total - int(keep.at[idx, "supervised_chars"])
        if rest < lo * budget_chars:
            break
        keep = keep.drop(index=idx)
        total = rest
    return keep


def select_arm(
    candidates: pd.DataFrame,
    rule: str,
    budget_chars: int,
    shares: dict[str, float],
    seed: int = DEFAULT_SEED,
) -> pd.DataFrame:
    """Pick one arm's traces: per-language quota from ``shares``, filled in task order."""
    if rule not in ARMS:
        raise ValueError(f"unknown rule {rule!r}; expected one of {ARMS}")
    rng = random.Random(seed)
    frames = []
    for lang in sorted(shares):
        target = budget_chars * shares[lang]
        pool = candidates[candidates["language"].astype(str) == lang]
        if pool.empty or target <= 0:
            continue
        pool = _trace_order(pool, rule, rng)
        take = []
        used = 0
        for idx in pool.index:
            if used >= target:
                break
            take.append(idx)
            used += int(pool.at[idx, "supervised_chars"])
        frames.append(pool.loc[take])
    picked = pd.concat(frames) if frames else candidates.iloc[:0]
    picked = fit_budget(picked, budget_chars)
    return picked.sort_values(["language", "trajectory_id"], kind="mergesort").reset_index(
        drop=True
    )


def select_eval(
    candidates: pd.DataFrame,
    used_instances: set[str],
    n: int = EVAL_N,
    seed: int = DEFAULT_SEED,
) -> pd.DataFrame:
    """One trace per instance from instances in no manifest, balanced across buckets."""
    pool = candidates[~candidates["instance_id"].isin(used_instances)]
    pool = pool[pool["supervised_chars"] > 0]
    order = list(range(len(pool)))
    random.Random(seed).shuffle(order)
    pool = pool.iloc[order]
    one = pool.drop_duplicates(subset=["instance_id"], keep="first")
    quota = max(1, n // len(ELIGIBLE_BUCKETS))
    frames = [one[one["difficulty_bucket"] == b].head(quota) for b in ELIGIBLE_BUCKETS]
    picked = pd.concat(frames) if frames else one.head(0)
    if len(picked) < n:
        rest = one[~one["instance_id"].isin(picked["instance_id"])]
        picked = pd.concat([picked, rest.head(n - len(picked))])
    return picked.head(n).reset_index(drop=True)


def select_eval_small(
    eval_df: pd.DataFrame,
    n: int = EVAL_SMALL_N,
    seed: int = DEFAULT_SEED,
) -> pd.DataFrame:
    """Seeded ``n``-trace subset of the full eval, balanced across buckets.

    A subset (same rows, same messages) so the final full-set CE and the in-run small
    CE are nested: the 60 traces keep the same windows in both.
    """
    rng = random.Random(seed + 3)
    quota, extra = divmod(n, len(ELIGIBLE_BUCKETS))
    frames = []
    for i, bucket in enumerate(ELIGIBLE_BUCKETS):
        sub = eval_df[eval_df["difficulty_bucket"] == bucket]
        idx = list(sub.index)
        rng.shuffle(idx)
        frames.append(sub.loc[idx[: quota + (1 if i < extra else 0)]])
    picked = pd.concat(frames) if frames else eval_df.head(0)
    return picked.reset_index(drop=True)


def gold_paths_from_patch(patch: str | None) -> list[str]:
    """Gold file paths from ``metadata.reference_patch.patch`` (same regexes as the
    temporal features: ``diff --git a/X b/X``, falling back to ``+++ b/X``)."""
    text = patch or ""
    paths = GOLD_GIT_RE.findall(text)
    if not paths:
        paths = GOLD_PLUS_RE.findall(text)
    return list(dict.fromkeys(paths))


def _call_key(call: dict) -> tuple[str, str]:
    fn = call.get("function") or {}
    return (
        str(fn.get("name") or ""),
        json.dumps(fn.get("arguments"), sort_keys=True, ensure_ascii=False),
    )


def _command_text(arguments: object) -> str:
    obj = arguments
    if isinstance(obj, str):
        try:
            obj = json.loads(obj)
        except ValueError:
            return ""
    if isinstance(obj, dict):
        return str(obj.get("command") or "")
    return ""


def _is_edit(tool_name: object, command: str) -> bool:
    """Edit-call detection mirroring the proxy/temporal feature regexes."""
    name = str(tool_name or "").lower()
    if name == EDIT_TOOL_VIEW_ONLY:
        return command.lower() != "view"
    if name in EDIT_TOOL_NAMES:
        return True
    return bool(re.search(BASH_EDIT_RE, command.lower()))


def _mentions_gold(args_text: str, gold: list[str]) -> bool:
    return any(p in args_text or p.rsplit("/", 1)[-1] in args_text for p in gold)


def step_masks(messages: list[dict], gold_paths: list[str]) -> list[bool]:
    """Per-assistant-message mask flags (spec order of the random_masked arm):

    - the step repeats an identical tool call already made in this trace, or
    - its tool observation (the tool messages directly after it) contains
      ``Error`` / ``Traceback`` / ``FAILED``, or
    - it makes an edit-type call whose arguments mention no gold file.
    """
    seen: set[tuple[str, str]] = set()
    masks: list[bool] = []
    for i, msg in enumerate(messages):
        if msg.get("role") != "assistant":
            continue
        calls = msg.get("tool_calls") or []
        repeated = any(_call_key(c) in seen for c in calls)
        edit_outside = False
        for call in calls:
            fn = call.get("function") or {}
            args = fn.get("arguments")
            raw = args if isinstance(args, str) else json.dumps(args)
            if _is_edit(fn.get("name"), _command_text(args)) and not _mentions_gold(
                raw, gold_paths
            ):
                edit_outside = True
        observed_error = False
        for nxt in messages[i + 1 :]:
            if nxt.get("role") != "tool":
                break
            content = nxt.get("content") or ""
            if "Error" in content or "Traceback" in content or "FAILED" in content:
                observed_error = True
        masks.append(bool(repeated or edit_outside or observed_error))
        seen.update(_call_key(c) for c in calls)
    return masks


def clean_messages(
    messages: list[dict],
    *,
    tool_max_chars: int = TOOL_MAX_CHARS,
    masks: list[bool] | None = None,
) -> list[dict]:
    """Manifest message format of ``sft.sample``; adds ``mask`` to every assistant
    message when ``masks`` is given (aligned to assistant messages in order)."""
    out: list[dict] = []
    a = 0
    for msg in messages:
        role = msg.get("role")
        content = msg.get("content") or ""
        if role == "tool" and len(content) > tool_max_chars:
            content = content[:tool_max_chars] + "\n...[truncated]"
        clean: dict = {"role": role, "content": content}
        if role == "assistant":
            if masks is not None:
                clean["mask"] = bool(masks[a])
            a += 1
        if role == "assistant" and msg.get("tool_calls"):
            calls = []
            for call in msg["tool_calls"]:
                fn = dict(call.get("function") or {})
                try:
                    fn["arguments"] = json.loads(fn["arguments"])
                except (TypeError, ValueError):
                    pass
                calls.append({"id": call.get("id"), "type": call.get("type"), "function": fn})
            clean["tool_calls"] = calls
        out.append(clean)
    return out


def fetch_raw(needed: pd.DataFrame) -> dict[str, dict]:
    """``trajectory_id -> {messages, tools, patch}`` for the picked rows, one query per shard."""
    con = connect_ephemeral()
    out: dict[str, dict] = {}
    try:
        for shard, group in needed.groupby("shard", sort=True):
            ids = sorted(set(group["trajectory_id"].astype(str)))
            rows = con.execute(
                f"""
                SELECT
                    trajectory_id,
                    to_json(messages) AS msgs_json,
                    tools,
                    coalesce(metadata.reference_patch.patch, '') AS patch
                FROM read_parquet({_sql_str(shard)}, union_by_name=true)
                WHERE trajectory_id IN (SELECT unnest(?::VARCHAR[]))
                """,
                [ids],
            ).fetchall()
            for tid, msgs_json, tools, patch in rows:
                out[tid] = {
                    "messages": json.loads(msgs_json),
                    "tools": [json.loads(t) for t in (tools or [])],
                    "patch": patch,
                }
    finally:
        con.close()
    return out


def _manifest_row(row: pd.Series, raw: dict, *, masked: bool, tool_max_chars: int) -> dict:
    masks = None
    if masked:
        gold = gold_paths_from_patch(raw["patch"])
        masks = step_masks(raw["messages"], gold)
    return {
        "trajectory_id": str(row["trajectory_id"]),
        "instance_id": str(row["instance_id"]),
        "harness": str(row["harness"]),
        "language": str(row["language"]),
        "resolved": int(row["resolved"]),
        "difficulty_bucket": str(row["difficulty_bucket"]),
        "p_resolved": float(row["p_resolved"]),
        "messages": clean_messages(raw["messages"], tool_max_chars=tool_max_chars, masks=masks),
        "tools": raw["tools"],
    }


def write_jsonl(path: Path, rows: list[dict]) -> Path:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w") as f:
        for row in rows:
            f.write(json.dumps(row) + "\n")
    return path


def build_manifests(
    budget_chars: int = DEFAULT_BUDGET_CHARS,
    *,
    out_dir: Path = OUT_DIR,
    tool_max_chars: int = TOOL_MAX_CHARS,
    seed: int = DEFAULT_SEED,
    eval_n: int = EVAL_N,
    eval_small_n: int = EVAL_SMALL_N,
    refresh: bool = False,
    cache: Path = CANDIDATES_CACHE,
) -> dict[str, Path]:
    """Build the four arm manifests plus the full and small eval; returns path per file."""
    candidates = load_candidates(refresh=refresh, cache=cache)
    console.print(
        f"candidates: {len(candidates):,} traces, {candidates['instance_id'].nunique():,} instances, "
        f"{candidates['supervised_chars'].sum() / 1e6:.1f}M supervised chars"
    )
    shares = language_shares(candidates)
    console.print(
        "language shares (supervised chars): "
        + "  ".join(f"{lang} {share:.2f}" for lang, share in shares.items())
    )

    arms = {rule: select_arm(candidates, rule, budget_chars, shares, seed) for rule in ARMS}
    used_instances = set().union(*(set(df["instance_id"]) for df in arms.values()))
    eval_df = select_eval(candidates, used_instances, n=eval_n, seed=seed)
    eval_small_df = select_eval_small(eval_df, n=eval_small_n, seed=seed)

    needed = pd.concat([*arms.values(), eval_df])[["trajectory_id", "shard"]]
    needed = needed.drop_duplicates(subset=["trajectory_id"])
    console.print(
        f"fetching messages for {len(needed):,} traces from {needed['shard'].nunique()} shards ..."
    )
    raw = fetch_raw(needed)

    paths: dict[str, Path] = {}
    summary: dict = {
        "budget_chars": budget_chars,
        "seed": seed,
        "tool_max_chars": tool_max_chars,
        "language_share": shares,
        "bucket_weights": BUCKET_WEIGHTS,
        "arms": {},
        "eval": {},
    }
    for rule, picked in arms.items():
        rows = [
            _manifest_row(
                row,
                raw[str(row["trajectory_id"])],
                masked=rule == MASK_RULE,
                tool_max_chars=tool_max_chars,
            )
            for _, row in picked.iterrows()
        ]
        arm_path = write_jsonl(out_dir / f"{rule}.jsonl", rows)
        paths[rule] = arm_path
        n_mask = sum(
            1 for r in rows for m in r["messages"] if m.get("role") == "assistant" and m.get("mask")
        )
        n_assistant = sum(1 for r in rows for m in r["messages"] if m.get("role") == "assistant")
        summary["arms"][rule] = {
            "path": str(arm_path),
            "n_traces": len(rows),
            "n_tasks": int(picked["instance_id"].nunique()),
            "supervised_chars": int(picked["supervised_chars"].sum()),
            "masked_assistant_msgs": n_mask,
            "assistant_msgs": n_assistant,
            "per_language": picked.groupby("language")["trajectory_id"].count().to_dict(),
            "per_bucket": picked.groupby("difficulty_bucket")["trajectory_id"].count().to_dict(),
            "per_teacher": picked.groupby("teacher")["trajectory_id"].count().to_dict(),
        }

    eval_rows = [
        _manifest_row(
            row, raw[str(row["trajectory_id"])], masked=False, tool_max_chars=tool_max_chars
        )
        for _, row in eval_df.iterrows()
    ]
    eval_path = write_jsonl(out_dir / "eval_heldout.jsonl", eval_rows)
    paths["eval_heldout"] = eval_path
    summary["eval"] = {
        "path": str(eval_path),
        "n_traces": len(eval_rows),
        "instances": int(eval_df["instance_id"].nunique()),
        "per_bucket": eval_df.groupby("difficulty_bucket")["trajectory_id"].count().to_dict(),
        "per_language": eval_df.groupby("language")["trajectory_id"].count().to_dict(),
    }

    small_ids = set(eval_small_df["trajectory_id"])
    eval_small_rows = [r for r in eval_rows if r["trajectory_id"] in small_ids]
    small_path = write_jsonl(out_dir / "eval_heldout_small.jsonl", eval_small_rows)
    paths["eval_heldout_small"] = small_path
    summary["eval_small"] = {
        "path": str(small_path),
        "n_traces": len(eval_small_rows),
        "instances": int(eval_small_df["instance_id"].nunique()),
        "per_bucket": eval_small_df.groupby("difficulty_bucket")["trajectory_id"].count().to_dict(),
    }

    summary_path = write_jsonl(out_dir / "manifests_summary.json", [summary])
    paths["summary"] = summary_path
    print_summary(summary, budget_chars)
    return paths


def print_summary(summary: dict, budget_chars: int) -> None:
    console.print("\n[bold]manifests[/bold]")
    for rule, arm in summary["arms"].items():
        total = arm["supervised_chars"]
        console.print(
            f"  {rule:<18} {arm['n_traces']:>5} traces  {arm['n_tasks']:>5} tasks  "
            f"{total / 1e6:>6.2f}M chars  ({total / budget_chars:.3f} of budget)"
        )
        console.print(
            "    buckets: "
            + " ".join(f"{k}:{v}" for k, v in sorted(arm["per_bucket"].items()))
            + "   langs: "
            + " ".join(f"{k}:{v}" for k, v in sorted(arm["per_language"].items()))
        )
        console.print(
            "    teachers: " + " ".join(f"{k}:{v}" for k, v in sorted(arm["per_teacher"].items()))
        )
        if arm["assistant_msgs"]:
            console.print(
                f"    masked assistant msgs: {arm['masked_assistant_msgs']:,} / "
                f"{arm['assistant_msgs']:,}"
            )
    for key, label in (("eval", "eval_heldout"), ("eval_small", "eval_heldout_small")):
        ev = summary[key]
        buckets = " ".join(f"{k}:{v}" for k, v in sorted(ev["per_bucket"].items()))
        line = f"  {label:<18} {ev['n_traces']:>5} traces  buckets: {buckets}"
        if "per_language" in ev:
            line += "  langs: " + " ".join(
                f"{k}:{v}" for k, v in sorted(ev["per_language"].items())
            )
        console.print(line)


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--budget-chars",
        type=int,
        default=DEFAULT_BUDGET_CHARS,
        help="supervised-char budget per manifest (default 165M ~= 3k traces ~= 3k windows)",
    )
    ap.add_argument("--out-dir", type=Path, default=OUT_DIR)
    ap.add_argument("--tool-max-chars", type=int, default=TOOL_MAX_CHARS)
    ap.add_argument("--seed", type=int, default=DEFAULT_SEED)
    ap.add_argument("--eval-n", type=int, default=EVAL_N)
    ap.add_argument("--eval-small-n", type=int, default=EVAL_SMALL_N)
    ap.add_argument(
        "--refresh-cache",
        action="store_true",
        help="rescan the corpus instead of reusing outputs/manifests_candidates.parquet",
    )
    a = ap.parse_args()
    build_manifests(
        a.budget_chars,
        out_dir=a.out_dir,
        tool_max_chars=a.tool_max_chars,
        seed=a.seed,
        eval_n=a.eval_n,
        eval_small_n=a.eval_small_n,
        refresh=a.refresh_cache,
    )

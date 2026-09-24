"""Semantic failure labels via OpenRouter free-tier models (closure-H step 5).

Samples SMALL-stratum fail trajectories that are members of within-instance pairs,
re-fetches their messages + model patch from the corpus (and one passing sibling's
patch as reference), and asks a free OpenRouter model to label the failure with a
fixed rubric:

    missed_second_site | misread_issue | wrong_root_cause | fixed_symptom_not_cause
    incomplete_stopped_early | broke_other_test | environment | other

Requests are capped (``MAX_REQUESTS``), retried on 429 with backoff, and cached to
``outputs/llm_labels.jsonl`` so reruns resume. The API key is read from the repo
``.env`` (OPENROUTER_API_KEY) and is never printed.

Usage: uv run python scripts/paired_labels.py [--n 300] [--dry-run]
"""

from __future__ import annotations

import argparse
import json
import threading
import time
from collections.abc import Iterable
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path
from typing import Any

import httpx
import pandas as pd
from rich.console import Console

import duckdb

from .data import PARQUET_GLOB, ROOT, connect_ephemeral
from .features import _sql_str, utcnow

console = Console()

LABELS_PATH = ROOT / "outputs" / "llm_labels.jsonl"
TAXONOMY_PATH = ROOT / "outputs" / "paired_fail_taxonomy.parquet"
ELIGIBLE_PATH = ROOT / "outputs" / "eligible_pairs.parquet"

API_URL = "https://openrouter.ai/api/v1/chat/completions"
MODELS_URL = "https://openrouter.ai/api/v1/models"

MAX_REQUESTS = 600
DEFAULT_N = 300
SAMPLE_SEED = 42
WORKERS = 6

MAX_ISSUE_CHARS = 2500
MAX_MSG_CHARS = 900
N_TAIL_MESSAGES = 10
MAX_PATCH_CHARS = 3500

# preferred free models, first available wins (list refreshed 2026-09-19 from /models).
# Non-reasoning instruct models first — reasoning models emit long thinking traces and
# make each request 30-120s on the free tier.
PREFERRED_MODELS = [
    "google/gemma-4-31b-it:free",
    "qwen/qwen3.8-27b:free",
    "nvidia/nemotron-3-super-120b-a12b:free",
    "nvidia/nemotron-3-ultra-550b-a55b:free",
    "deepseek/deepseek-v4-flash-0731:free",
    "openrouter/free",
]

RUBRIC = """\
Labels (pick exactly one):
- missed_second_site: the fix addresses one site but the task needed edits at another place too
- misread_issue: the model solved a different problem than the issue describes
- wrong_root_cause: the diagnosis of the underlying bug is wrong
- fixed_symptom_not_cause: the change patches the surface symptom, not the root cause
- incomplete_stopped_early: the work was on the right track but the model stopped/submitted early
- broke_other_test: the fix plausibly addresses the issue but breaks other behavior/tests
- environment: failure is about the environment/harness, not the code change
- other: none of the above"""


def load_api_key() -> str:
    for env_path in (
        ROOT / ".env",
        Path("/home/evan/Documents/open_swe_traces_research/.env"),
    ):
        if env_path.exists():
            for line in env_path.read_text().splitlines():
                if line.startswith("OPENROUTER_API_KEY="):
                    return line.split("=", 1)[1].strip().strip('"').strip("'")
    raise SystemExit("OPENROUTER_API_KEY not found in .env")


def pick_free_models(client: httpx.Client, key: str) -> list[str]:
    """Preferred free models first, then remaining free ones — used as fallback chain."""
    resp = client.get(MODELS_URL, headers={"Authorization": f"Bearer {key}"}, timeout=30)
    resp.raise_for_status()
    free = {
        m["id"]
        for m in resp.json()["data"]
        if str(m.get("pricing", {}).get("prompt", "1")) == "0"
        and str(m.get("pricing", {}).get("completion", "1")) == "0"
    }
    chain = [m for m in PREFERRED_MODELS if m in free]
    if not chain:
        if not free:
            raise SystemExit("no free models on OpenRouter right now")
        chain = [min(free)]
    return chain


def sample_small_fails(n: int = DEFAULT_N, seed: int = SAMPLE_SEED) -> pd.DataFrame:
    """Deterministic sample of SMALL fail trajectories + one passing sibling per group."""
    tax = pd.read_parquet(
        TAXONOMY_PATH,
        columns=["trajectory_id", "instance_id", "combo", "bucket", "stop_reason"],
    )
    elig = pd.read_parquet(
        ELIGIBLE_PATH, columns=["trajectory_id", "group_id", "stratum", "resolved"]
    )
    df = tax.merge(elig, on="trajectory_id")
    fails = df[df["stratum"] == "SMALL"]
    # one passing sibling per group (deterministic: smallest trajectory_id)
    passes = elig[(elig["resolved"] == 1) & (elig["stratum"] == "SMALL")]
    sibling = (
        passes.sort_values("trajectory_id")
        .groupby("group_id")["trajectory_id"]
        .first()
        .rename("sibling_id")
    )
    fails = fails.merge(sibling, on="group_id", how="left")
    sample = fails.sample(n=min(n, len(fails)), random_state=seed).sort_values("trajectory_id")
    return sample.reset_index(drop=True)


def fetch_trajectories(
    con: duckdb.DuckDBPyConnection, ids: Iterable[str]
) -> pd.DataFrame:
    """Locate and pull messages + patches for a small set of trajectory_ids."""
    ids = sorted(set(ids))
    glob_sql = PARQUET_GLOB.replace("'", "''")
    locs = con.execute(
        f"""
        SELECT trajectory_id, filename
        FROM read_parquet('{glob_sql}', union_by_name=true, filename=true)
        WHERE trajectory_id IN (SELECT unnest(?::VARCHAR[]))
        """,
        [ids],
    ).fetchall()
    by_file: dict[str, list[str]] = {}
    for tid, fname in locs:
        by_file.setdefault(str(fname), []).append(str(tid))
    missing = set(ids) - {t for ts in by_file.values() for t in ts}
    if missing:
        console.print(f"[yellow]warning: {len(missing)} trajectory_ids not found[/yellow]")

    frames = []
    for i, (fname, file_ids) in enumerate(sorted(by_file.items())):
        df = con.execute(
            f"""
            SELECT trajectory_id, messages,
                   coalesce(metadata.model_patch.patch, '') AS model_patch
            FROM read_parquet({_sql_str(fname)}, union_by_name=true)
            WHERE trajectory_id IN (SELECT unnest(?::VARCHAR[]))
            """,
            [file_ids],
        ).fetchdf()
        frames.append(df)
        if i % 10 == 9:
            console.print(f"  fetched {i + 1}/{len(by_file)} files")
    return pd.concat(frames, ignore_index=True)


def _clip(text: str, n: int) -> str:
    text = text or ""
    return text if len(text) <= n else text[:n] + "\n...[truncated]"


def build_prompt(fail: pd.Series, sibling_patch: str) -> list[dict[str, str]]:
    messages = fail["messages"]
    issue = ""
    tail: list[dict[str, Any]] = []
    for m in messages:
        role = m.get("role")
        if role == "user" and not issue:
            issue = m.get("content") or ""
        if role in ("assistant", "tool"):
            tail.append(
                {
                    "role": role,
                    "content": _clip(m.get("content") or "", MAX_MSG_CHARS),
                }
            )
    tail_text = "\n\n".join(f"[{t['role']}] {t['content']}" for t in tail[-N_TAIL_MESSAGES:])

    user = f"""You are auditing a failed coding-agent trajectory from SWE-bench-like data.
A sibling rollout on the SAME task PASSED; this one FAILED. Label why the failing
attempt failed.

{RUBRIC}

== ISSUE (truncated) ==
{_clip(issue, MAX_ISSUE_CHARS)}

== FAILING TRAJECTORY, last {N_TAIL_MESSAGES} messages (truncated) ==
{tail_text}

== FAILING PATCH ==
{_clip(fail['model_patch'], MAX_PATCH_CHARS)}

== PASSING SIBLING PATCH (reference) ==
{_clip(sibling_patch, MAX_PATCH_CHARS)}

Reply with JSON only: {{"label": "<one label>", "evidence": "<one sentence>"}}"""
    return [
        {"role": "system", "content": "You label failed agent trajectories. JSON only."},
        {"role": "user", "content": user},
    ]


def _load_done_ids() -> set[str]:
    if not LABELS_PATH.exists():
        return set()
    done = set()
    with LABELS_PATH.open() as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                done.add(json.loads(line)["trajectory_id"])
            except (json.JSONDecodeError, KeyError):
                continue
    return done


def chat_once(
    client: httpx.Client,
    key: str,
    models: list[str],
    messages: list[dict],
    start: int = 0,
) -> dict:
    """One chat completion; retries rotate through the free-model chain on 429/5xx/timeouts.

    ``start`` rotates the first-tried model per request so load spreads across
    providers' per-model rate limits instead of hammering the same one.
    """
    delay = 5.0
    for attempt in range(6):
        model = models[(start + attempt) % len(models)]
        try:
            resp = client.post(
                API_URL,
                headers={"Authorization": f"Bearer {key}"},
                json={
                    "model": model,
                    "messages": messages,
                    "temperature": 0,
                    "max_tokens": 256,
                    "reasoning": {"enabled": False},
                },
                timeout=90,
            )
        except httpx.HTTPError:
            if attempt == 5:
                raise
            time.sleep(delay)
            delay = min(delay * 2, 120)
            continue
        if resp.status_code in (429, 500, 502, 503, 504):
            if attempt == 5:
                resp.raise_for_status()
            retry_after = resp.headers.get("retry-after")
            wait = float(retry_after) if retry_after else delay
            console.print(f"[dim]  {resp.status_code}, sleeping {wait:.0f}s[/dim]")
            time.sleep(wait)
            delay = min(delay * 2, 120)
            continue
        resp.raise_for_status()
        body = resp.json()
        if body.get("choices"):
            return body
        # 200 with an error payload or empty choices — retryable upstream failure
        if attempt == 5:
            raise RuntimeError(f"no choices in response: {str(body)[:200]}")
        console.print(f"[dim]  empty/error 200 from {model}, retrying[/dim]")
        time.sleep(delay)
        delay = min(delay * 2, 120)
    raise RuntimeError("unreachable")


VALID_LABELS = {
    "missed_second_site",
    "misread_issue",
    "wrong_root_cause",
    "fixed_symptom_not_cause",
    "incomplete_stopped_early",
    "broke_other_test",
    "environment",
    "other",
}


def parse_label(content: str) -> dict[str, str]:
    """Extract {label, evidence} from a model reply; tolerate markdown fences."""
    text = content.strip()
    if "```" in text:
        text = text.split("```")[1].removeprefix("json").strip()
    try:
        obj = json.loads(text)
    except json.JSONDecodeError:
        start, end = text.find("{"), text.rfind("}")
        if start == -1 or end <= start:
            return {"label": "other", "evidence": "unparseable model reply"}
        try:
            obj = json.loads(text[start : end + 1])
        except json.JSONDecodeError:
            return {"label": "other", "evidence": "unparseable model reply"}
    label = str(obj.get("label", "other")).strip().lower()
    if label not in VALID_LABELS:
        label = "other"
    return {"label": label, "evidence": str(obj.get("evidence", ""))[:500]}


def run_labeling(n: int = DEFAULT_N, dry_run: bool = False, seed: int = SAMPLE_SEED) -> int:
    key = load_api_key()
    sample = sample_small_fails(n, seed)
    done_ids = _load_done_ids()
    todo = sample[~sample["trajectory_id"].isin(done_ids)]
    console.print(
        f"[{utcnow()}] sample={len(sample)} already_labeled={len(done_ids)} todo={len(todo)}"
    )
    if len(todo) + len(done_ids) > MAX_REQUESTS:
        raise SystemExit(f"would exceed MAX_REQUESTS={MAX_REQUESTS}")
    if len(todo) == 0:
        console.print("nothing to do")
        return 0

    cache = ROOT / "outputs" / "llm_sample_cache.parquet"
    need = set(todo["trajectory_id"]) | set(todo["sibling_id"].dropna())
    fetched = (
        pd.read_parquet(cache) if cache.exists() else pd.DataFrame()
    )
    missing_ids = need - set(fetched["trajectory_id"]) if len(fetched) else need
    if missing_ids:
        con = connect_ephemeral()
        try:
            new_rows = fetch_trajectories(con, missing_ids)
        finally:
            con.close()
        fetched = pd.concat([fetched, new_rows], ignore_index=True)
        fetched.to_parquet(cache)
    fetched = fetched.set_index("trajectory_id")
    console.print(f"fetched {len(fetched)} trajectories from corpus/cache")

    if dry_run:
        first = todo.iloc[0]
        prompt = build_prompt(
            fetched.loc[first["trajectory_id"]],
            fetched.loc[first["sibling_id"]]["model_patch"]
            if first["sibling_id"] in fetched.index
            else "",
        )
        console.print(prompt[1]["content"][:2000])
        return 0

    LABELS_PATH.parent.mkdir(parents=True, exist_ok=True)
    sent = 0
    lock = threading.Lock()
    tasks = [
        (i, row)
        for i, (_, row) in enumerate(todo.iterrows())
        if row["trajectory_id"] in fetched.index
    ]
    with httpx.Client() as client:
        models = pick_free_models(client, key)
        console.print(f"models: {models}")

        def work(idx: int, row: pd.Series) -> dict | None:
            tid = row["trajectory_id"]
            sib = row["sibling_id"]
            sib_patch = (
                fetched.loc[sib]["model_patch"] if sib in fetched.index else ""
            )
            try:
                resp = chat_once(
                    client,
                    key,
                    models,
                    build_prompt(fetched.loc[tid], sib_patch),
                    start=idx,
                )
            except Exception as exc:  # noqa: BLE001
                console.print(f"[red]request failed for {tid}: {exc}[/red]")
                return None
            return {
                "trajectory_id": tid,
                "instance_id": row["instance_id"],
                "combo": row["combo"],
                "rule_bucket": row["bucket"],
                "model": resp.get("model", models[0]),
                **parse_label(resp["choices"][0]["message"]["content"]),
            }

        with (
            LABELS_PATH.open("a") as out,
            ThreadPoolExecutor(max_workers=WORKERS) as pool,
        ):
            futures = {
                pool.submit(work, idx, row): idx for idx, row in tasks[:MAX_REQUESTS]
            }
            for fut in as_completed(futures):
                rec = fut.result()
                if rec is None:
                    continue
                with lock:
                    sent += 1
                    out.write(json.dumps(rec) + "\n")
                    out.flush()
                console.print(
                    f"[{utcnow()}] {sent}/{len(tasks)} {rec['trajectory_id'][:8]} "
                    f"via {rec['model']} → {rec['label']}"
                )
                if sent + len(done_ids) >= MAX_REQUESTS:
                    console.print("[yellow]request cap reached[/yellow]")
                    break
    console.print(f"[green]done: {sent} new labels → {LABELS_PATH}[/green]")
    return 0


def confusion_table() -> pd.DataFrame:
    """LLM label × rule-based taxonomy crosstab."""
    labels = pd.read_json(LABELS_PATH, lines=True)
    ct = pd.crosstab(labels["rule_bucket"], labels["label"], margins=True)
    return ct


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--n", type=int, default=DEFAULT_N)
    parser.add_argument("--seed", type=int, default=SAMPLE_SEED)
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument(
        "--confusion", action="store_true", help="print label × taxonomy crosstab"
    )
    args = parser.parse_args()
    if args.confusion:
        console.print(confusion_table().to_string())
        return
    raise SystemExit(run_labeling(args.n, args.dry_run, args.seed))


if __name__ == "__main__":
    main()

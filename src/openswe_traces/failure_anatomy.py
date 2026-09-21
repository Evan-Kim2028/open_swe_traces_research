"""Does the certified-hard L0 failure anatomy reproduce on Open-SWE-Traces?

Streams labeled trajectories (resolved in (0, 1)) shard-by-shard. Never loads a
trajectory's full message list into Python: DuckDB keeps the last assistant
turns and a 30-message tail, classifies them in SQL, and writes compact flags.

Outputs:
  outputs/failure_anatomy_parts/*.parquet
  outputs/failure_anatomy.parquet
  outputs/failure_anatomy_q3_sample.json
  analytics/research/openswe_failure_anatomy_compare.md  (tables + reproduce cmds)

Q3/Q4 labels are filled from outputs/failure_anatomy_q3_labels.json when present.
"""

from __future__ import annotations

import argparse
import json
import math
import os
import re
import sys
import time
import traceback
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import pandas as pd
from rich.console import Console

import duckdb

from .data import PARQUET_GLOB, ROOT, connect_ephemeral, list_parquet_files, parse_shard
from .features import _sql_str, fmt_duration, part_path_for, utcnow
from .rungs import classify_task, strip_task_text

console = Console()

PARTS_DIR = ROOT / "outputs" / "failure_anatomy_parts"
OUT_PARQUET = ROOT / "outputs" / "failure_anatomy.parquet"
OUT_MD = ROOT / "analytics" / "research" / "openswe_failure_anatomy_compare.md"
Q3_SAMPLE_JSON = ROOT / "outputs" / "failure_anatomy_q3_sample.json"
Q3_LABELS_JSON = ROOT / "outputs" / "failure_anatomy_q3_labels.json"
QA_SNIPPETS_JSON = ROOT / "outputs" / "failure_anatomy_qa_snippets.json"

SCALE_SWE_META = ROOT / "traces_external" / "AweAI-Team__Scale-SWE" / "meta.parquet"
SWE_REBENCH = (
    ROOT / "traces_external" / "nebius__SWE-rebench-V2" / "data" / "train-00000-of-00001.parquet"
)

TAIL_N = 30
Q3_N = 40
Q3_SEED = 42

# Last-turn language. Applied to lowercased last-two-assistant text (content +
# reasoning + tool-call args). RE2-safe: no lookaheads.
TESTS_PASS_RE = (
    r"(?i)("
    r"all (the )?(existing )?(unit |hidden )?(tests?|checks) pass(ed|ing)?|"
    r"all \d+ (existing )?(tests?|checks) pass|"
    r"\d+ (tests? )?pass(ed)?|"
    r"tests? (all )?pass(ed|ing)?|"
    r"(pytest|suite|reproduction) .{0,30}pass|"
    r"everything (passes|passed)|"
    r"all checks pass|"
    r"verified.{0,40}(tests?|suite) pass|"
    r"full (unit )?test suite pass|"
    r"existing tests? pass"
    r")"
)
FIX_COMPLETE_RE = (
    r"(?i)("
    r"implementation is complete|"
    r"fix is complete|"
    r"the (fix|patch|implementation|changes) (is|are) (complete|correct|done|working)|"
    r"successfully (implemented|fixed|implemented the)|"
    r"i have successfully|"
    r"my (fix|implementation) is (correct|complete)|"
    r"the issue is (fixed|resolved)|"
    r"changes (are|were) complete|"
    r"ready to submit"
    r")"
)
UNCERTAIN_RE = (
    r"(?i)("
    r"\bnot sure\b|"
    r"\buncertain\b|"
    r"\bunsure\b|"
    r"i (don.?t|do not) know|"
    r"might (still )?(fail|be wrong|be incomplete)|"
    r"may (still )?(fail|be incomplete|be wrong)|"
    r"cannot (confirm|verify)|"
    r"couldn.?t verify|"
    r"unable to verify|"
    r"\bunverified\b|"
    r"not (completely )?confident|"
    r"not 100%|"
    r"i.?m not (certain|confident)|"
    r"i am not (certain|confident)|"
    r"possibly (wrong|incomplete|missing)"
    r")"
)
HIDDEN_MENTION_RE = r"(?i)\bhidden tests?\b"
UNSEEN_GAP_RE = (
    r"(?i)("
    r"unseen tests?|"
    r"withheld tests?|"
    r"(cannot|couldn.?t|did not|didn.?t) (run|see|access) (the )?(hidden|unseen|official) tests?|"
    r"hidden tests? .{0,30}(may|might|could) (still )?fail|"
    r"(may|might|could) fail (the )?(hidden|unseen|official) tests?|"
    r"tests? (i|we) (did not|didn.?t|cannot|couldn.?t) (see|run|access)"
    r")"
)
TEST_CMD_RE = (
    r"(?i)("
    r"\bpytest\b|"
    r"python -m pytest|"
    r"\bgo test\b|"
    r"\bcargo test\b|"
    r"npm (run )?test|"
    r"\bnpx (jest|vitest)\b|"
    r"\bjest\b|"
    r"\bvitest\b|"
    r"\bphpunit\b|"
    r"\bmvn test\b|"
    r"\bmake test\b|"
    r"\bunittest\b|"
    r"\btox\b|"
    r"\bctest\b|"
    r"\brspec\b|"
    r"\bmeson test\b"
    r")"
)
TEST_PATH_RE = (
    r"(?i)("
    r"(^|/)(tests?|spec|__tests__|testdata)(/|$)|"
    r"_test\.(go|py|rs|php)$|"
    r"(^|/)test_[^/]+\.py$|"
    r"\.(spec|test)\.(js|ts|tsx|jsx)$|"
    r"Test[A-Z][^/]*\.java$"
    r")"
)
GREEN_RE = (
    r"("
    r'"returncode"\s*:\s*0|'
    r"exit code 0|"
    r"Command finished with exit code 0|"
    r"ALL CHECKS PASSED|"
    r"\d+ passed|"
    r"Test Suites:\s*\d+\s+passed|"
    r"\bok\s+[\w./]+|"
    r"\d+ passing"
    r")"
)
RED_RE = (
    r"("
    r'"returncode"\s*:\s*[1-9]|'
    r"exit code [1-9]|"
    r"Command finished with exit code [1-9]|"
    r"\bFAILED\b|"
    r"\bFAIL |"
    r"Test Suites:\s*[1-9][0-9]* failed|"
    r"[1-9][0-9]* failed"
    r")"
)
EXIT_NONZERO_RE = r'("returncode"\s*:\s*[1-9]|exit code [1-9])'
EXIT_ZERO_RE = r'("returncode"\s*:\s*0|exit code 0)'
NEW_TEST_RE = r"(?i)--- /dev/null\n\+\+\+ b/[^\n]*(tests?/|__tests__/|_test\.|\.test\.|test_)"

FEATURE_SQL = r"""
WITH src AS (
    SELECT
        instance_id,
        repo,
        language,
        trajectory_id,
        resolved,
        messages,
        metadata,
        len(messages) AS n_messages
    FROM read_parquet('{file_sql}', union_by_name=true)
    WHERE resolved IN (0, 1)
),
prep AS (
    SELECT
        instance_id,
        repo,
        language,
        trajectory_id,
        resolved::TINYINT AS resolved,
        n_messages,
        metadata.category AS category,
        coalesce(metadata.model_patch.patch, '') AS model_patch,
        coalesce(metadata.reference_patch.patch, '') AS gold_patch,
        coalesce(metadata.model_patch.num_modified_files, 0) AS model_files,
        coalesce(metadata.reference_patch.num_modified_files, 0) AS gold_files,
        list_filter(messages, m -> m.role = 'assistant') AS asst,
        list_slice(messages, greatest(1, len(messages) - {tail_n} + 1), len(messages)) AS tail
    FROM src
),
last_txt AS (
    SELECT
        instance_id, repo, language, trajectory_id, resolved, n_messages, category,
        model_patch, gold_patch, model_files, gold_files, tail,
        lower(
            coalesce(asst[len(asst)].content, '') || ' ' ||
            coalesce(asst[len(asst)].reasoning_content, '') || ' ' ||
            coalesce(cast(asst[len(asst)].tool_calls AS VARCHAR), '') || ' ' ||
            CASE WHEN len(asst) >= 2 THEN
                coalesce(asst[len(asst) - 1].content, '') || ' ' ||
                coalesce(asst[len(asst) - 1].reasoning_content, '') || ' ' ||
                coalesce(cast(asst[len(asst) - 1].tool_calls AS VARCHAR), '')
            ELSE '' END
        ) AS last_text
    FROM prep
),
flags AS (
    SELECT
        *,
        regexp_matches(last_text, '{tests_pass}') AS asserts_tests_pass,
        regexp_matches(last_text, '{fix_complete}') AS asserts_fix_complete,
        regexp_matches(last_text, '{uncertain}') AS expresses_uncertainty,
        regexp_matches(last_text, '{hidden_mention}') AS mentions_hidden_tests,
        regexp_matches(last_text, '{unseen_gap}') AS mentions_unseen_gap
    FROM last_txt
),
calls AS (
    SELECT
        f.trajectory_id,
        turn_idx,
        coalesce(tc.function.arguments, '') AS args,
        lower(coalesce(tc.function.name, '')) AS tool_name
    FROM flags f,
         unnest(f.tail) WITH ORDINALITY AS u(m, turn_idx),
         unnest(m.tool_calls) WITH ORDINALITY AS v(tc, ci)
),
test_calls AS (
    SELECT trajectory_id, turn_idx
    FROM calls
    WHERE regexp_matches(lower(args || ' ' || tool_name), '{test_cmd}')
),
last_test AS (
    SELECT trajectory_id, max(turn_idx) AS test_turn
    FROM test_calls
    GROUP BY 1
),
obs AS (
    SELECT f.trajectory_id, u.turn_idx, m.content AS content
    FROM flags f, unnest(f.tail) WITH ORDINALITY AS u(m, turn_idx)
    WHERE m.role = 'tool'
),
last_obs AS (
    SELECT lt.trajectory_id, arg_min(o.content, o.turn_idx) AS obs
    FROM last_test lt
    JOIN obs o ON o.trajectory_id = lt.trajectory_id AND o.turn_idx > lt.test_turn
    GROUP BY 1
),
model_paths AS (
    SELECT
        trajectory_id,
        CASE
            WHEN len(git_paths) > 0 THEN list_distinct(git_paths)
            ELSE list_distinct(plus_paths)
        END AS paths
    FROM (
        SELECT
            trajectory_id,
            regexp_extract_all(model_patch, 'diff --git a/([^\s]+) b/', 1) AS git_paths,
            regexp_extract_all(model_patch, '\+\+\+ b/([^\s]+)', 1) AS plus_paths
        FROM flags
    )
),
gold_paths AS (
    SELECT
        trajectory_id,
        CASE
            WHEN len(git_paths) > 0 THEN list_distinct(git_paths)
            ELSE list_distinct(plus_paths)
        END AS paths
    FROM (
        SELECT
            trajectory_id,
            regexp_extract_all(gold_patch, 'diff --git a/([^\s]+) b/', 1) AS git_paths,
            regexp_extract_all(gold_patch, '\+\+\+ b/([^\s]+)', 1) AS plus_paths
        FROM flags
    )
)
SELECT
    '{harness}' AS harness,
    '{teacher}' AS teacher,
    '{source}' AS source,
    f.trajectory_id,
    f.instance_id,
    f.repo,
    f.language,
    f.category,
    f.resolved,
    f.n_messages::INTEGER AS n_messages,
    (f.asserts_tests_pass OR f.asserts_fix_complete) AS asserts_done,
    f.asserts_tests_pass,
    f.asserts_fix_complete,
    f.expresses_uncertainty,
    f.mentions_hidden_tests,
    f.mentions_unseen_gap,
    (lt.test_turn IS NOT NULL) AS ran_test_in_tail,
    CASE
        WHEN lo.obs IS NULL THEN NULL
        WHEN regexp_matches(lo.obs, '{red}') AND NOT regexp_matches(lo.obs, '{green}') THEN FALSE
        WHEN regexp_matches(lo.obs, '{red}') AND regexp_matches(lo.obs, '{green}') THEN
            CASE
                WHEN regexp_matches(lo.obs, '{exit_nz}') THEN FALSE
                WHEN regexp_matches(lo.obs, '{exit_z}') THEN TRUE
                ELSE FALSE
            END
        WHEN regexp_matches(lo.obs, '{green}') THEN TRUE
        ELSE NULL
    END AS last_test_green,
    (len(list_filter(coalesce(mp.paths, []), p -> regexp_matches(p, '{test_path}'))) > 0)
        AS model_touches_tests,
    regexp_matches(f.model_patch, '{new_test}') AS model_creates_tests,
    (len(f.model_patch) = 0) AS empty_model_patch,
    f.model_files::INTEGER AS model_files,
    f.gold_files::INTEGER AS gold_files,
    CASE
        WHEN len(list_distinct(list_concat(coalesce(mp.paths, []), coalesce(gp.paths, [])))) = 0
            THEN 0.0
        ELSE len(list_filter(
                list_distinct(coalesce(mp.paths, [])),
                p -> list_contains(coalesce(gp.paths, []), p)
             ))::DOUBLE
             / len(list_distinct(list_concat(coalesce(mp.paths, []), coalesce(gp.paths, []))))
    END AS patch_file_jaccard
FROM flags f
LEFT JOIN last_test lt ON lt.trajectory_id = f.trajectory_id
LEFT JOIN last_obs lo ON lo.trajectory_id = f.trajectory_id
LEFT JOIN model_paths mp ON mp.trajectory_id = f.trajectory_id
LEFT JOIN gold_paths gp ON gp.trajectory_id = f.trajectory_id
"""


def _esc(pattern: str) -> str:
    return pattern.replace("'", "''")


def build_anatomy_sql(file_path: Path, harness: str, teacher: str, source: str) -> str:
    sql = FEATURE_SQL
    replacements = {
        "file_sql": str(file_path).replace("'", "''"),
        "harness": harness.replace("'", "''"),
        "teacher": teacher.replace("'", "''"),
        "source": source.replace("'", "''"),
        "tail_n": str(TAIL_N),
        "tests_pass": _esc(TESTS_PASS_RE),
        "fix_complete": _esc(FIX_COMPLETE_RE),
        "uncertain": _esc(UNCERTAIN_RE),
        "hidden_mention": _esc(HIDDEN_MENTION_RE),
        "unseen_gap": _esc(UNSEEN_GAP_RE),
        "test_cmd": _esc(TEST_CMD_RE),
        "test_path": _esc(TEST_PATH_RE),
        "green": _esc(GREEN_RE),
        "red": _esc(RED_RE),
        "exit_nz": _esc(EXIT_NONZERO_RE),
        "exit_z": _esc(EXIT_ZERO_RE),
        "new_test": _esc(NEW_TEST_RE),
    }
    for key, value in replacements.items():
        sql = sql.replace("{" + key + "}", value)
    return sql


def process_shard(
    con: duckdb.DuckDBPyConnection,
    file_path: Path,
    *,
    force: bool,
    parts_dir: Path = PARTS_DIR,
) -> tuple[str, int]:
    part = part_path_for(file_path, parts_dir)
    if part.exists() and part.stat().st_size > 0 and not force:
        return "skip", 0

    harness, teacher, source = parse_shard(file_path).harness, parse_shard(file_path).teacher, parse_shard(file_path).source
    src_labeled = con.execute(
        f"SELECT count(*) FROM read_parquet({_sql_str(file_path)}) WHERE resolved IN (0, 1)"
    ).fetchone()[0]

    tmp = Path(f"{part}.{os.getpid()}.tmp")
    tmp.parent.mkdir(parents=True, exist_ok=True)
    try:
        if src_labeled == 0:
            # Don't unnest messages on shards with no labeled outcomes.
            con.execute(
                f"""
                COPY (
                    SELECT
                        CAST(NULL AS VARCHAR) AS harness,
                        CAST(NULL AS VARCHAR) AS teacher,
                        CAST(NULL AS VARCHAR) AS source,
                        CAST(NULL AS VARCHAR) AS trajectory_id,
                        CAST(NULL AS VARCHAR) AS instance_id,
                        CAST(NULL AS VARCHAR) AS repo,
                        CAST(NULL AS VARCHAR) AS language,
                        CAST(NULL AS VARCHAR) AS category,
                        CAST(NULL AS TINYINT) AS resolved,
                        CAST(NULL AS INTEGER) AS n_messages,
                        CAST(NULL AS BOOLEAN) AS asserts_done,
                        CAST(NULL AS BOOLEAN) AS asserts_tests_pass,
                        CAST(NULL AS BOOLEAN) AS asserts_fix_complete,
                        CAST(NULL AS BOOLEAN) AS expresses_uncertainty,
                        CAST(NULL AS BOOLEAN) AS mentions_hidden_tests,
                        CAST(NULL AS BOOLEAN) AS mentions_unseen_gap,
                        CAST(NULL AS BOOLEAN) AS ran_test_in_tail,
                        CAST(NULL AS BOOLEAN) AS last_test_green,
                        CAST(NULL AS BOOLEAN) AS model_touches_tests,
                        CAST(NULL AS BOOLEAN) AS model_creates_tests,
                        CAST(NULL AS BOOLEAN) AS empty_model_patch,
                        CAST(NULL AS INTEGER) AS model_files,
                        CAST(NULL AS INTEGER) AS gold_files,
                        CAST(NULL AS DOUBLE) AS patch_file_jaccard
                    WHERE FALSE
                ) TO {_sql_str(tmp)} (FORMAT PARQUET)
                """
            )
            os.replace(tmp, part)
            return "done", 0

        con.execute(
            f"COPY ({build_anatomy_sql(file_path, harness, teacher, source)}) "
            f"TO {_sql_str(tmp)} (FORMAT PARQUET)"
        )
        out_rows, out_tids = con.execute(
            f"SELECT count(*), count(DISTINCT trajectory_id) FROM read_parquet({_sql_str(tmp)})"
        ).fetchone()
        if out_rows != src_labeled or out_tids != src_labeled:
            raise RuntimeError(
                f"row guard failed: labeled={src_labeled} out={out_rows} "
                f"distinct_trajectories={out_tids}"
            )
    except Exception:
        tmp.unlink(missing_ok=True)
        raise
    os.replace(tmp, part)
    return "done", int(out_rows)


def merge_anatomy(
    con: duckdb.DuckDBPyConnection,
    parts_dir: Path = PARTS_DIR,
    out_path: Path = OUT_PARQUET,
) -> tuple[int, int, int] | None:
    parts = sorted(parts_dir.glob("*.parquet"))
    if not parts:
        return None
    glob_sql = str(parts_dir / "*.parquet").replace("'", "''")
    tmp = Path(str(out_path) + ".tmp")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    con.execute(
        f"""
        COPY (
            SELECT * FROM read_parquet('{glob_sql}')
            ORDER BY harness, teacher, source, trajectory_id
        ) TO {_sql_str(tmp)} (FORMAT PARQUET)
        """
    )
    rows, tids = con.execute(
        f"SELECT count(*), count(DISTINCT trajectory_id) FROM read_parquet({_sql_str(tmp)})"
    ).fetchone()
    os.replace(tmp, out_path)
    return len(parts), rows, tids


def wilson(k: int, n: int, z: float = 1.96) -> tuple[float, float, float]:
    """Return (p, lo, hi). p is k/n; lo/hi are the Wilson 95% interval."""
    if n <= 0:
        return (float("nan"), float("nan"), float("nan"))
    p = k / n
    z2 = z * z
    den = 1.0 + z2 / n
    center = (p + z2 / (2.0 * n)) / den
    half = z * math.sqrt(p * (1.0 - p) / n + z2 / (4.0 * n * n)) / den
    return p, max(0.0, center - half), min(1.0, center + half)


def cohen_h(p1: float, p2: float) -> float:
    return 2.0 * math.asin(math.sqrt(p1)) - 2.0 * math.asin(math.sqrt(p2))


def pct_cell(k: int, n: int) -> str:
    p, lo, hi = wilson(k, n)
    if n <= 0 or math.isnan(p):
        return f"— (n=0)"
    return f"{100 * p:.1f}% ({k}/{n}; 95% CI {100 * lo:.1f}–{100 * hi:.1f})"


def classify_last_text(text: str) -> dict[str, bool]:
    """Python mirror of the SQL last-text flags, for unit tests."""
    return {
        "asserts_tests_pass": bool(re.search(TESTS_PASS_RE, text or "")),
        "asserts_fix_complete": bool(re.search(FIX_COMPLETE_RE, text or "")),
        "expresses_uncertainty": bool(re.search(UNCERTAIN_RE, text or "")),
        "mentions_hidden_tests": bool(re.search(HIDDEN_MENTION_RE, text or "")),
        "mentions_unseen_gap": bool(re.search(UNSEEN_GAP_RE, text or "")),
    }


def classify_test_obs(obs: str | None) -> bool | None:
    """Python mirror of the SQL last-test-green CASE."""
    if not obs:
        return None
    red = bool(re.search(RED_RE, obs))
    green = bool(re.search(GREEN_RE, obs))
    if red and not green:
        return False
    if red and green:
        if re.search(EXIT_NONZERO_RE, obs):
            return False
        if re.search(EXIT_ZERO_RE, obs):
            return True
        return False
    if green:
        return True
    return None


def _count(df: pd.DataFrame, col: str, value: Any = True) -> int:
    if col not in df.columns:
        return 0
    series = df[col]
    if value is True:
        return int(series.fillna(False).astype(bool).sum())
    if value is False:
        return int((series == False).sum())  # noqa: E712
    return int((series == value).sum())


def _rate_row(fail: pd.DataFrame, ok: pd.DataFrame, col: str) -> dict[str, Any]:
    kf, nf = _count(fail, col), len(fail)
    ko, no = _count(ok, col), len(ok)
    pf, lo_f, hi_f = wilson(kf, nf)
    po, lo_o, hi_o = wilson(ko, no)
    return {
        "fail_pct": pct_cell(kf, nf),
        "ok_pct": pct_cell(ko, no),
        "delta_pp": 100.0 * (pf - po) if nf and no else float("nan"),
        "cohen_h": cohen_h(pf, po) if nf and no else float("nan"),
        "k_fail": kf,
        "n_fail": nf,
        "k_ok": ko,
        "n_ok": no,
        "p_fail": pf,
        "p_ok": po,
    }


def load_fail_to_pass(con: duckdb.DuckDBPyConnection) -> pd.DataFrame:
    frames = []
    if SCALE_SWE_META.exists():
        frames.append(
            con.execute(
                f"""
                SELECT instance_id,
                       len(fail_to_pass)::INTEGER AS n_ftp,
                       len(pass_to_pass)::INTEGER AS n_ptp,
                       'scale-swe' AS ftp_source
                FROM read_parquet({_sql_str(SCALE_SWE_META)})
                """
            ).fetchdf()
        )
    if SWE_REBENCH.exists():
        frames.append(
            con.execute(
                f"""
                SELECT instance_id,
                       len(FAIL_TO_PASS)::INTEGER AS n_ftp,
                       len(PASS_TO_PASS)::INTEGER AS n_ptp,
                       'swe-rebench-v2' AS ftp_source
                FROM read_parquet({_sql_str(SWE_REBENCH)})
                """
            ).fetchdf()
        )
    if not frames:
        return pd.DataFrame(columns=["instance_id", "n_ftp", "n_ptp", "ftp_source"])
    out = pd.concat(frames, ignore_index=True)
    return out.drop_duplicates(subset=["instance_id"], keep="first")


def pick_q3_sample(inst: pd.DataFrame, n: int = Q3_N, seed: int = Q3_SEED) -> pd.DataFrame:
    """Stratified all_fail sample: Go oversampled, mixed source and FAIL_TO_PASS size."""
    pool = inst[(inst["n_labeled"] >= 3) & (inst["solve_rate"] == 0)].copy()
    if pool.empty:
        pool = inst[(inst["n_fail"] >= 1)].copy()
    pool["lang_bucket"] = pool["language"].fillna("unknown").str.lower().map(
        lambda x: "go" if x == "go" else ("python" if x == "python" else "other")
    )
    pool["ftp_bucket"] = pd.cut(
        pool["n_ftp"].fillna(-1),
        bins=[-2, 0, 1, 4, 10**9],
        labels=["missing", "1", "2-4", "5+"],
    ).astype(str)
    import hashlib

    def _key(s: str) -> int:
        return int.from_bytes(hashlib.md5(f"{seed}:{s}".encode(), usedforsecurity=False).digest()[:8], "big")

    pool["_key"] = pool["instance_id"].map(_key)
    pool = pool.sort_values(["_key", "instance_id"])

    targets = [("go", 16), ("python", 12), ("other", 12)]
    ftp_order = ["1", "2-4", "5+", "missing"]
    picked = []
    used = set()
    for bucket, want in targets:
        sub = pool[pool["lang_bucket"] == bucket]
        # Equal share per FAIL_TO_PASS size, round-robin so a large "1" bucket
        # cannot crowd out 2-4 and 5+.
        queues = {
            ftp: sub[sub["ftp_bucket"] == ftp].to_dict("records")
            for ftp in ftp_order
        }
        idx = {ftp: 0 for ftp in ftp_order}
        got = 0
        while got < want:
            progressed = False
            for ftp in ftp_order:
                if got >= want:
                    break
                q = queues[ftp]
                i = idx[ftp]
                if i >= len(q):
                    continue
                row = q[i]
                idx[ftp] = i + 1
                if row["instance_id"] in used:
                    continue
                picked.append(row)
                used.add(row["instance_id"])
                got += 1
                progressed = True
            if not progressed:
                break
    # fill if short
    for _, row in pool.iterrows():
        if len(picked) >= n:
            break
        if row["instance_id"] in used:
            continue
        picked.append(row)
        used.add(row["instance_id"])
    out = pd.DataFrame(picked[:n])
    return out.drop(columns=["_key"], errors="ignore")


def extract_issue_gold(
    con: duckdb.DuckDBPyConnection,
    instance_ids: list[str],
    files: list[Path],
) -> dict[str, dict[str, Any]]:
    """Pull issue text + gold patch for a small id set, shard by shard."""
    wanted = set(instance_ids)
    found: dict[str, dict[str, Any]] = {}
    id_sql = ",".join(_sql_str(i) for i in instance_ids)
    for file_path in files:
        if not wanted:
            break
        shard = parse_shard(file_path)
        rows = con.execute(
            f"""
            SELECT
                instance_id,
                messages[2].content AS raw_content,
                coalesce(metadata.reference_patch.patch, '') AS gold_patch,
                length(messages[2].content) AS content_len,
                repo,
                language
            FROM read_parquet({_sql_str(file_path)})
            WHERE instance_id IN ({id_sql})
              AND len(messages) >= 2
            """
        ).fetchall()
        for instance_id, raw, gold, content_len, repo, language in rows:
            prev = found.get(instance_id)
            if prev is None or int(content_len or 0) > prev["content_len"]:
                found[instance_id] = {
                    "instance_id": instance_id,
                    "repo": repo,
                    "language": language,
                    "harness_seen": shard.harness,
                    "source": shard.source,
                    "content_len": int(content_len or 0),
                    "issue_text": strip_task_text(raw or ""),
                    "raw_content": (raw or "")[:20000],
                    "gold_patch": gold or "",
                }
                wanted.discard(instance_id)
    return found


def stream_corpus(args: argparse.Namespace) -> int:
    PARTS_DIR.mkdir(parents=True, exist_ok=True)
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
                status, rows = process_shard(con, file_path, force=args.force)
            except Exception as exc:  # noqa: BLE001
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
        if not args.no_merge:
            merged = merge_anatomy(con)
            if merged is None:
                console.print(f"[{utcnow()}] no parts to merge")
            else:
                n_parts, rows, tids = merged
                console.print(
                    f"[{utcnow()}] merged {n_parts} parts → {OUT_PARQUET.relative_to(ROOT)}: "
                    f"{rows:,} rows, {tids:,} distinct trajectories"
                )
    except KeyboardInterrupt:
        console.print(f"[{utcnow()}] interrupted — parts kept, rerun to resume")
        return 130
    finally:
        con.close()
    console.print(
        f"[{utcnow()}] stream finished: processed={done} skipped={skipped} failed={failed} "
        f"in {fmt_duration(time.monotonic() - started)}"
    )
    if failures:
        console.print(f"[red]failed shards[/red]: {', '.join(failures)}")
        return 1
    return 0


def _fmt_delta(row: dict[str, Any]) -> str:
    d = row["delta_pp"]
    h = row["cohen_h"]
    if isinstance(d, float) and math.isnan(d):
        return "—"
    sign = "+" if d >= 0 else ""
    return f"{sign}{d:.1f} pp (h={h:+.2f})"


def build_report(con: duckdb.DuckDBPyConnection) -> str:
    if not OUT_PARQUET.exists():
        raise SystemExit(f"missing {OUT_PARQUET}; run without --skip-stream")
    df = con.execute(f"SELECT * FROM read_parquet({_sql_str(OUT_PARQUET)})").fetchdf()
    fail = df[df["resolved"] == 0]
    ok = df[df["resolved"] == 1]

    ftp = load_fail_to_pass(con)
    df = df.merge(ftp, on="instance_id", how="left")
    fail = df[df["resolved"] == 0]
    ok = df[df["resolved"] == 1]

    inst = (
        df.groupby("instance_id", as_index=False)
        .agg(
            n_rollouts=("trajectory_id", "count"),
            n_fail=("resolved", lambda s: int((s == 0).sum())),
            n_ok=("resolved", lambda s: int((s == 1).sum())),
            language=("language", "first"),
            repo=("repo", "first"),
            source=("source", "first"),
            n_ftp=("n_ftp", "first"),
            ftp_source=("ftp_source", "first"),
        )
    )
    inst["n_labeled"] = inst["n_fail"] + inst["n_ok"]
    inst["solve_rate"] = inst["n_ok"] / inst["n_labeled"].where(inst["n_labeled"] > 0)

    q1_cols = [
        ("asserts_done", "asserts tests pass or the fix is complete"),
        ("asserts_tests_pass", "asserts tests/reproduction pass"),
        ("asserts_fix_complete", "asserts the fix is complete"),
        ("expresses_uncertainty", "expresses any uncertainty"),
        ("mentions_hidden_tests", "mentions 'hidden tests' (SWE-bench-aware)"),
        ("mentions_unseen_gap", "names unverified / withheld tests as a gap"),
        ("ran_test_in_tail", f"ran a repo test command in the last {TAIL_N} messages"),
        ("model_touches_tests", "model patch touches a test path"),
        ("model_creates_tests", "model patch creates a new test file"),
        ("empty_model_patch", "empty model patch"),
    ]
    q1_rows = {name: _rate_row(fail, ok, col) for col, name in q1_cols}

    # last_test_green is nullable. Report three denominators.
    fail_green = fail[fail["last_test_green"].notna()]
    ok_green = ok[ok["last_test_green"].notna()]
    q1_rows["last visible test command was green (given a classified observation)"] = _rate_row(
        fail_green, ok_green, "last_test_green"
    )
    # Among all rollouts: green is True.
    q1_rows["last visible test command was green (all labeled rollouts)"] = _rate_row(
        fail, ok, "last_test_green"
    )
    # Conjunction: stopping-rule analogue of "visible suite green then stop".
    fail = fail.copy()
    ok = ok.copy()
    fail["stop_rule"] = fail["asserts_done"].fillna(False) & fail["last_test_green"].fillna(False)
    ok["stop_rule"] = ok["asserts_done"].fillna(False) & ok["last_test_green"].fillna(False)
    q1_rows["stopping-rule conjunction (asserts done AND last test green)"] = _rate_row(
        fail, ok, "stop_rule"
    )

    # Q2
    fail_ftp = fail[fail["n_ftp"].notna()]
    n_ftp_cov = int(fail_ftp["instance_id"].nunique())
    n_fail_inst = int(fail["instance_id"].nunique())
    ftp_series = fail_ftp.drop_duplicates("instance_id")["n_ftp"]

    def qtile(s: pd.Series, q: float) -> float:
        if s.empty:
            return float("nan")
        return float(s.quantile(q))

    jaccard_fail = fail.loc[~fail["empty_model_patch"].fillna(False), "patch_file_jaccard"]
    jaccard_ok = ok.loc[~ok["empty_model_patch"].fillna(False), "patch_file_jaccard"]

    def j_bucket(s: pd.Series) -> dict[str, int]:
        return {
            "empty_or_zero": int((s.fillna(0) == 0).sum()) if False else 0,
            "j=0": int((s == 0).sum()),
            "0<j<0.5": int(((s > 0) & (s < 0.5)).sum()),
            "j>=0.5": int((s >= 0.5).sum()),
            "j=1": int((s == 1).sum()),
        }

    # Q3 sample (ids only; texts extracted separately)
    sample = pick_q3_sample(inst)
    labels = {}
    if Q3_LABELS_JSON.exists():
        labels = json.loads(Q3_LABELS_JSON.read_text())

    generated = datetime.now(UTC).strftime("%Y-%m-%d %H:%M UTC")
    n_fail, n_ok = len(fail), len(ok)
    n_inst = int(df["instance_id"].nunique())

    def q1_table() -> str:
        lines = [
            "| measure | failed (resolved=0) | resolved=1 | Δ fail−resolved |",
            "|---|---|---|---|",
        ]
        for name, row in q1_rows.items():
            lines.append(
                f"| {name} | {row['fail_pct']} | {row['ok_pct']} | {_fmt_delta(row)} |"
            )
        return "\n".join(lines)

    def harness_table() -> str:
        lines = [
            "| harness | teacher | n_fail | n_ok | fail asserts_done | ok asserts_done | fail last-test green (classified) | fail uncertainty | fail test-path edit |",
            "|---|---|---:|---:|---|---|---|---|---|",
        ]
        for (harness, teacher), g in df.groupby(["harness", "teacher"]):
            f = g[g["resolved"] == 0]
            o = g[g["resolved"] == 1]
            fg = f[f["last_test_green"].notna()]
            lines.append(
                "| {harness} | {teacher} | {nf} | {no} | {ad} | {ado} | {gr} | {unc} | {te} |".format(
                    harness=harness,
                    teacher=teacher,
                    nf=len(f),
                    no=len(o),
                    ad=pct_cell(_count(f, "asserts_done"), len(f)),
                    ado=pct_cell(_count(o, "asserts_done"), len(o)),
                    gr=pct_cell(_count(fg, "last_test_green"), len(fg)),
                    unc=pct_cell(_count(f, "expresses_uncertainty"), len(f)),
                    te=pct_cell(_count(f, "model_touches_tests"), len(f)),
                )
            )
        return "\n".join(lines)

    ftp_bins = [
        ("1", fail_ftp["n_ftp"] == 1),
        ("2–4", fail_ftp["n_ftp"].between(2, 4)),
        ("5–8", fail_ftp["n_ftp"].between(5, 8)),
        ("9+", fail_ftp["n_ftp"] >= 9),
    ]
    ftp_bin_table = [
        "| FAIL_TO_PASS count | failed rollouts | share of failed-with-ftp |",
        "|---|---:|---:|",
    ]
    n_fail_ftp = len(fail_ftp)
    for label, mask in ftp_bins:
        k = int(mask.sum())
        share = 100.0 * k / n_fail_ftp if n_fail_ftp else 0.0
        ftp_bin_table.append(f"| {label} | {k:,} | {share:.1f}% |")

    j_fail = fail["patch_file_jaccard"]
    j_ok = ok["patch_file_jaccard"]
    j_table = [
        "| file Jaccard (model vs gold) | failed | resolved |",
        "|---|---:|---:|",
        f"| n | {len(j_fail):,} | {len(j_ok):,} |",
        f"| median | {j_fail.median():.3f} | {j_ok.median():.3f} |",
        f"| p25 | {j_fail.quantile(0.25):.3f} | {j_ok.quantile(0.25):.3f} |",
        f"| p75 | {j_fail.quantile(0.75):.3f} | {j_ok.quantile(0.75):.3f} |",
        f"| share = 0 (no shared files, including empty patches) | {100 * (j_fail == 0).mean():.1f}% | {100 * (j_ok == 0).mean():.1f}% |",
        f"| share ≥ 0.5 | {100 * (j_fail >= 0.5).mean():.1f}% | {100 * (j_ok >= 0.5).mean():.1f}% |",
        f"| share = 1 (same file set) | {100 * (j_fail == 1).mean():.1f}% | {100 * (j_ok == 1).mean():.1f}% |",
    ]

    nonempty_fail = fail[~fail["empty_model_patch"].fillna(False)]
    near_miss = nonempty_fail[nonempty_fail["patch_file_jaccard"] >= 0.5]

    q3_section = _q3_section(sample, labels)
    q4_section = _q4_section(labels)

    # Headline numbers for the verdict
    ad = q1_rows["asserts tests pass or the fix is complete"]
    unc = q1_rows["expresses any uncertainty"]
    green = q1_rows["last visible test command was green (given a classified observation)"]
    stop = q1_rows["stopping-rule conjunction (asserts done AND last test green)"]
    test_edit = q1_rows["model patch touches a test path"]
    unseen = q1_rows["names unverified / withheld tests as a gap"]

    verdict = _verdict(
        n_fail=n_fail,
        n_ok=n_ok,
        ad=ad,
        unc=unc,
        green=green,
        stop=stop,
        test_edit=test_edit,
        unseen=unseen,
        ftp_median=float(ftp_series.median()) if len(ftp_series) else float("nan"),
        n_ftp_cov=n_ftp_cov,
        n_fail_inst=n_fail_inst,
        j_fail_ge50=float((j_fail >= 0.5).mean()),
        labels=labels,
        sample_n=len(sample),
    )

    md = f"""# Does the certified-hard failure anatomy reproduce on Open-SWE-Traces?

Generated {generated} by `uv run python scripts/failure_anatomy.py`.

## Verdict

{verdict}

## What is measured vs inferred

Measured: last-two-assistant text flags, last test-command observation in a {TAIL_N}-message tail, model-patch test paths, FAIL_TO_PASS list sizes from Scale-SWE and SWE-rebench-V2, file Jaccard of model vs gold patches.

Inferred: that a green in-tree run caused the stop (we see the green run and then a submit; we do not see a counterfactual where the agent would have kept going); that a missed clause lived only in a withheld contract (Q3/Q4, n={len(sample)} manual reads).

The original finding is on 45 synthetic Go units built so that L0 always fails and L2 almost always passes. Open-SWE-Traces tasks are naturally occurring GitHub PRs at one fixed affordance level. A negative on transfer is a valid result.

## Method

Labeled trajectories only (`resolved IN (0, 1)`). Unknown (`resolved = -1`) rows are dropped; they cannot test a stopping rule against an outcome.

Per shard, DuckDB keeps `list_last` / second-last assistant messages (content + reasoning + tool-call args, lowercased) and a {TAIL_N}-message tail. Regexes live in `openswe_traces.failure_anatomy` and are unit-tested. The corpus is never loaded as trajectories.

Last-test pairing: last tool call in the tail whose arguments match a repo test command (`pytest`, `go test`, `cargo test`, `npm test`, `jest`, …); the next `role=tool` observation is the result. Green/red prefers an explicit exit code when both signals fire. `pytest | tail` without `pipefail` can hide a red exit; those rows stay in the classified-green count if the observation still contains a pass summary, and are a known false-green risk.

Test-path edits: paths from `diff --git a/X b/X` matching `tests/`, `_test.go`, `test_*.py`, `*.test.ts`, and siblings. Created tests: `--- /dev/null` plus a test path.

FAIL_TO_PASS lists come from `traces_external/` (Scale-SWE `fail_to_pass`, SWE-rebench-V2 `FAIL_TO_PASS`). Per-test results of the *agent* patch are not in Open-SWE-Traces.

n_fail = **{n_fail:,}**; n_resolved = **{n_ok:,}**; labeled trajectories = **{n_fail + n_ok:,}**; distinct instances = **{n_inst:,}**.

## Q1 — stopping rule

Original (61 L0-fail trials): 46/61 (75%) final message claims tests pass; 0/61 flags uncertainty; 0/61 names unseen tests; 44/45 units stopped because the visible suite went green.

{q1_table()}

Cohen's h: 0.20 is a small effect, 0.50 medium, 0.80 large (Cohen 1988). The original claim is that unjustified confidence is *specific to failures*. The comparison column is that claim's test: a large positive Δ on confidence, or a large negative Δ on uncertainty, would support it. A near-zero Δ means the stopping ritual is shared and only the outcome label makes the confidence unjustified.

### By harness / teacher

{harness_table()}

`openhands/deepseek_v4_flash` and `openhands/qwen36_27b` contribute no labeled rows (all `resolved = -1`).

`asserts_done` tracks finish-message style, not a shared cognitive state. OpenHands almost always emits a "I have successfully implemented" finish tool call (99%+ on both outcomes). SWE-agent Qwen often submits with an empty last turn (26–31% match). Last-test-green does not have that problem: 93–99% of classified last-test observations are green on *failed* rollouts in every labeled combo. That is the number to compare to the original 44/45 visible-suite-green stops.

Test-path edits concentrate in `openhands/qwen35_122b` (43.6% of that combo's failures, 10.9% of its successes). Drop that teacher and the corpus-wide fail rate is about 3%, still above resolved but not a common mode. `empty_model_patch` is 0% because the patch *string* is never empty; `model_files = 0` still occurs (a few hundred rows).

## Q2 — near miss or structurally different?

Original: failing L0 patch already passes a median 78% of the hidden suite; median 2 failing tests of 8; 80% of patches are structurally gold with one clause wrong.

**Per-test results of the agent patch are not in this corpus.** `resolved` is a single bit. FAIL_TO_PASS is the *gold* hidden-suite membership list, not a run of the model patch. The 78% figure cannot be reproduced here. Closest available proxies follow.

FAIL_TO_PASS coverage: {n_ftp_cov:,} / {n_fail_inst:,} failed instances join a FAIL_TO_PASS list ({100 * n_ftp_cov / max(n_fail_inst, 1):.1f}%). Failed rollouts with a join: {n_fail_ftp:,}.

Hidden-suite *size* (FAIL_TO_PASS length) on failed rollouts with a join:

| | value |
|---|---|
| n rollouts | {n_fail_ftp:,} |
| median | {qtile(fail_ftp['n_ftp'], 0.5):.0f} |
| p25 | {qtile(fail_ftp['n_ftp'], 0.25):.0f} |
| p75 | {qtile(fail_ftp['n_ftp'], 0.75):.0f} |
| p90 | {qtile(fail_ftp['n_ftp'], 0.90):.0f} |
| mean | {fail_ftp['n_ftp'].mean():.1f} |

{chr(10).join(ftp_bin_table)}

The original units had a median hidden suite of 8 tests, so "fail 2 of 8" is a minority. On this corpus the median FAIL_TO_PASS length is {qtile(fail_ftp['n_ftp'], 0.5):.0f}. When the hidden suite is one test, a failed rollout fails 100% of required tests by construction; the "already passing 78%" shape is not even well-defined.

Structural proxy (file Jaccard of model patch vs gold), measured:

{chr(10).join(j_table)}

Failed rollouts with a non-empty patch and file Jaccard ≥ 0.5: **{len(near_miss):,} / {len(nonempty_fail):,}** ({100 * len(near_miss) / max(len(nonempty_fail), 1):.1f}%). That is "touched at least half the gold files", not "one clause wrong". Empty failed patches (string length 0): {pct_cell(_count(fail, 'empty_model_patch'), n_fail)}. The FAIL_TO_PASS mean ({fail_ftp['n_ftp'].mean():.1f}) is not usable: SWE-rebench lists run to 10^4–10^5 names on some instances; use the median and the bins.

## Q3 — was the decisive information in the issue text?

Original: ~76% C (clause only in the L2 contract / hidden test), ~18% B (inferable from the repo), ~7% A (in the bug report).

{q3_section}

## Q4 — anti-inferable category

Original: 13/45 units (29%) where the repo pointed at the rejected answer.

{q4_section}

## Reproduce

```bash
uv run python scripts/failure_anatomy.py
uv run pytest tests/test_failure_anatomy.py
```

Resume a partial stream with the same command (existing parts are skipped). Force a recompute with `--force`. Tables only, no stream: `--skip-stream`. Q3 texts: written to `outputs/failure_anatomy_q3_sample.json`. Labels: `outputs/failure_anatomy_q3_labels.json`.

Regexes: `TESTS_PASS_RE`, `FIX_COMPLETE_RE`, `UNCERTAIN_RE`, `UNSEEN_GAP_RE`, `TEST_CMD_RE`, `GREEN_RE`, `RED_RE` in `src/openswe_traces/failure_anatomy.py`.
"""
    return md


def _q3_section(sample: pd.DataFrame, labels: dict) -> str:
    n = len(sample)
    if not labels:
        ids = "\n".join(
            f"- `{r.instance_id}` lang={r.language} source={r.source} n_ftp={r.n_ftp} "
            f"n_labeled={int(r.n_labeled)}"
            for r in sample.itertuples()
        )
        return (
            f"Stratified sample of **{n}** all_fail instances (solve_rate = 0, ≥ 3 labeled rollouts; "
            f"Go oversampled; FAIL_TO_PASS size mixed; seed {Q3_SEED}). "
            "Labels are not in this file yet; they live in `outputs/failure_anatomy_q3_labels.json` "
            "after the manual pass.\n\n"
            "A = the behaviour gold implements is stated in the issue. "
            "B = not stated, but inferable from the repo as shipped to the agent. "
            "C = neither (only in the withheld test / gold author's choice).\n\n"
            f"Sample ids:\n\n{ids}"
        )
    rows = labels.get("instances", labels if isinstance(labels, list) else [])
    if isinstance(rows, dict):
        rows = list(rows.values())
    counts = {"A": 0, "B": 0, "C": 0, "mixed": 0, "unscored": 0}
    lines = [
        "| instance_id | lang | A/B/C | note |",
        "|---|---|---|---|",
    ]
    by_id = {r["instance_id"]: r for r in rows if "instance_id" in r}
    for rec in sample.itertuples():
        lab = by_id.get(rec.instance_id, {})
        abc = lab.get("abc", "unscored")
        counts[abc] = counts.get(abc, 0) + 1
        note = str(lab.get("note", "")).replace("|", "/")
        lines.append(f"| `{rec.instance_id}` | {rec.language} | {abc} | {note} |")
    scored = counts["A"] + counts["B"] + counts["C"] + counts["mixed"]
    c_share = 100.0 * counts["C"] / scored if scored else float("nan")
    # Go-only, matching the original certified-hard language.
    go_rows = [
        by_id[rec.instance_id]
        for rec in sample.itertuples()
        if str(rec.language).lower() == "go" and rec.instance_id in by_id
    ]
    go_c = sum(1 for r in go_rows if r.get("abc") == "C")
    go_n = len(go_rows)
    return (
        f"Stratified sample n={n}, scored={scored}. "
        f"A={counts['A']}, B={counts['B']}, C={counts['C']}, mixed={counts['mixed']}. "
        f"C share among scored = {c_share:.0f}% (original 76%). "
        f"Go subset: C={go_c}/{go_n}. "
        "Mixed means the issue states the main behaviour and gold also lands a load-bearing extra; "
        "those are not C. B and anti-inferable are lower bounds: no repo checkout. "
        "Labels: four disjoint readers (10 instances each), then one audit pass on every C and mixed row "
        "(two recodes: snowflake C to A, autoprefixer C to B; one anti kept on sushi, "
        "De Bruijn exact-match candidate not counted as original-sense anti). "
        f"C+mixed = {counts['C'] + counts['mixed']}/{scored} if extras count as withheld-clause-ish.\n\n"
        + "\n".join(lines)
    )


def _q4_section(labels: dict) -> str:
    if not labels:
        return (
            "Requires the Q3 manual read plus a judgement that the *repo as the agent saw it* "
            "argues for the rejected answer (sibling implementation, in-tree test, module default). "
            "Without a checkout of each repo this is a lower bound from issue+gold only: gold "
            "chooses a sentinel / error string / default the issue never names, *and* the issue "
            "or gold comment cites an existing convention that points the other way. "
            "Prevalence is filled after labels land."
        )
    rows = labels.get("instances", labels if isinstance(labels, list) else [])
    if isinstance(rows, dict):
        rows = list(rows.values())
    anti = [r for r in rows if r.get("anti_inferable")]
    scored = [r for r in rows if r.get("abc") in {"A", "B", "C", "mixed"}]
    n_anti, n_scored = len(anti), len(scored)
    p, lo, hi = wilson(n_anti, n_scored) if n_scored else (float("nan"),) * 3
    lines = [
        f"Anti-inferable among scored Q3 instances: **{n_anti}/{n_scored}** "
        f"({100 * p:.0f}%; 95% CI {100 * lo:.0f}–{100 * hi:.0f}%). Original 13/45 = 29%.",
        "",
        "| instance_id | evidence |",
        "|---|---|",
    ]
    for r in anti:
        lines.append(
            f"| `{r['instance_id']}` | {str(r.get('anti_note', '')).replace('|', '/')} |"
        )
    if n_anti == 0:
        lines.append("| — | none labeled anti-inferable in this sample |")
    return "\n".join(lines)


def _verdict(
    *,
    n_fail: int,
    n_ok: int,
    ad: dict,
    unc: dict,
    green: dict,
    stop: dict,
    test_edit: dict,
    unseen: dict,
    ftp_median: float,
    n_ftp_cov: int,
    n_fail_inst: int,
    j_fail_ge50: float,
    labels: dict,
    sample_n: int,
) -> str:
    # Five sentences, filled from measured numbers.
    s1 = (
        f"The stopping rule appears on failed rollouts: "
        f"{green['fail_pct']} of classified last-test observations are green and "
        f"{unc['fail_pct']} express uncertainty "
        f"(original: visible suite green on 44/45 units, 0/61 uncertain; "
        f"asserts-done is {ad['fail_pct']} but is harness finish-style, 26-99% by combo)."
    )
    s2 = (
        f"Unjustified confidence is not specific to failures: resolved rollouts match "
        f"(last-test green {green['ok_pct']}, uncertainty {unc['ok_pct']}; "
        f"Cohen's h {green['cohen_h']:+.2f} and {unc['cohen_h']:+.2f}), "
        f"and the original L0 bank had no successes to compare against."
    )
    s3 = (
        f"Manufactured all-clear by editing tests is {test_edit['fail_pct']} of failures vs "
        f"{test_edit['ok_pct']} of successes (h={test_edit['cohen_h']:+.2f}), "
        f"concentrated in openhands/qwen35_122b, not a common mode."
    )
    s4 = (
        f"The 78% hidden-suite near-miss cannot be measured (no agent per-test results); "
        f"median FAIL_TO_PASS length is {ftp_median:.0f} vs original 8."
    )
    if labels:
        rows = labels.get("instances", labels if isinstance(labels, list) else [])
        if isinstance(rows, dict):
            rows = list(rows.values())
        c = sum(1 for r in rows if r.get("abc") == "C")
        scored = sum(1 for r in rows if r.get("abc") in {"A", "B", "C", "mixed"})
        anti = sum(1 for r in rows if r.get("anti_inferable"))
        s5 = (
            f"On a stratified sample of {sample_n} all_fail instances, C (decisive clause not in the issue "
            f"and not scored as repo-inferable) is {c}/{scored} vs original 76%; "
            f"anti-inferable is {anti}/{scored} vs original 29%."
        )
    else:
        s5 = (
            f"Q3/Q4 need the {sample_n}-instance manual read of issue text vs gold patch; "
            f"ids are listed below. Until that pass, the information-availability claim is untested on this corpus."
        )
    return " ".join([s1, s2, s3, s4, s5])


def write_q3_sample(con: duckdb.DuckDBPyConnection, files: list[Path]) -> pd.DataFrame:
    df = con.execute(f"SELECT * FROM read_parquet({_sql_str(OUT_PARQUET)})").fetchdf()
    ftp = load_fail_to_pass(con)
    df = df.merge(ftp, on="instance_id", how="left")
    inst = (
        df.groupby("instance_id", as_index=False)
        .agg(
            n_fail=("resolved", lambda s: int((s == 0).sum())),
            n_ok=("resolved", lambda s: int((s == 1).sum())),
            language=("language", "first"),
            repo=("repo", "first"),
            source=("source", "first"),
            n_ftp=("n_ftp", "first"),
            ftp_source=("ftp_source", "first"),
        )
    )
    inst["n_labeled"] = inst["n_fail"] + inst["n_ok"]
    inst["solve_rate"] = inst["n_ok"] / inst["n_labeled"].where(inst["n_labeled"] > 0)
    sample = pick_q3_sample(inst)
    texts = extract_issue_gold(con, sample["instance_id"].tolist(), files)
    records = []
    for rec in sample.itertuples():
        payload = texts.get(rec.instance_id, {})
        rung = None
        issue = payload.get("issue_text", "")
        gold = payload.get("gold_patch", "")
        if issue or gold:
            try:
                rung = classify_task(payload.get("raw_content") or issue, gold).rung
            except Exception:
                rung = None
        records.append(
            {
                "instance_id": rec.instance_id,
                "repo": rec.repo,
                "language": rec.language,
                "source": rec.source,
                "n_labeled": int(rec.n_labeled),
                "n_fail": int(rec.n_fail),
                "n_ftp": None if pd.isna(rec.n_ftp) else int(rec.n_ftp),
                "heuristic_rung": rung,
                "issue_text": issue,
                "gold_patch": gold,
            }
        )
    Q3_SAMPLE_JSON.parent.mkdir(parents=True, exist_ok=True)
    Q3_SAMPLE_JSON.write_text(json.dumps(records, indent=2))
    console.print(f"wrote {len(records)} Q3 sample rows → {Q3_SAMPLE_JSON.relative_to(ROOT)}")
    return sample


def main_failure_anatomy() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--file", action="append", help="Process only these shard(s)")
    parser.add_argument("--limit", type=int, help="Process at most N shards")
    parser.add_argument("--data-glob", default=PARQUET_GLOB)
    parser.add_argument("--force", action="store_true")
    parser.add_argument("--no-merge", action="store_true")
    parser.add_argument("--merge-only", action="store_true")
    parser.add_argument("--skip-stream", action="store_true", help="Reuse merged parquet")
    parser.add_argument("--no-q3-extract", action="store_true")
    parser.add_argument("--threads", type=int, default=4)
    parser.add_argument("--memory-limit", default="6GB")
    args = parser.parse_args()

    if args.merge_only:
        con = connect_ephemeral()
        con.execute(f"SET memory_limit='{args.memory_limit}'")
        con.execute(f"SET threads={args.threads}")
        try:
            merged = merge_anatomy(con)
            if merged is None:
                raise SystemExit(f"No parts found in {PARTS_DIR}")
            n_parts, rows, tids = merged
            console.print(
                f"[{utcnow()}] merged {n_parts} parts → {OUT_PARQUET.relative_to(ROOT)}: "
                f"{rows:,} rows, {tids:,} trajectories"
            )
        finally:
            con.close()
        return

    if not args.skip_stream:
        rc = stream_corpus(args)
        if rc not in (0,):
            sys.exit(rc)

    con = connect_ephemeral()
    con.execute(f"SET memory_limit='{args.memory_limit}'")
    con.execute(f"SET threads={args.threads}")
    try:
        md = build_report(con)
        OUT_MD.parent.mkdir(parents=True, exist_ok=True)
        OUT_MD.write_text(md)
        console.print(f"wrote {OUT_MD.relative_to(ROOT)}")
        if not args.no_q3_extract:
            files = (
                [Path(f).resolve() for f in args.file]
                if args.file
                else list_parquet_files(args.data_glob)
            )
            write_q3_sample(con, files)
            # rebuild so the sample ids in the md match the json
            md = build_report(con)
            OUT_MD.write_text(md)
    finally:
        con.close()


if __name__ == "__main__":
    main_failure_anatomy()

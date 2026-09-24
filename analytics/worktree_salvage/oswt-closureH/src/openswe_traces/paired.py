"""Within-instance paired analysis: per-trajectory behavior features for pair-eligible rollouts.

The pair-eligible population lives in ``outputs/eligible_pairs.parquet``: trajectories on
mixed instances (``n_labeled >= 5``, ``0 < n_resolved < n_labeled``) inside
(instance, harness/teacher combo) groups that have at least one resolved and one
unresolved rollout.

This module streams the corpus shard-by-shard like ``features.py`` but only for eligible
trajectory_ids (semi-join against a registered ``eligible_ids`` view), computes
message-level behavior features in SQL, then computes patch-hunk features in Python (a
real unified-diff parser, not regexes). Parts land in ``outputs/paired_features_parts/``
(atomic, resume-safe) and merge into ``outputs/paired_features.parquet``.

Feature definitions (see PAIRED_SQL plus ``add_patch_features``):

  n_turns                    len(messages) — all roles
  n_assistant_turns          messages with role='assistant'
  n_tool_calls               tool calls across assistant turns
  assistant_chars            total assistant content chars (token proxy; no wall/tokens in data)
  turn_of_first_edit         1-based call index of first edit call; -1 if none
  turn_of_last_edit          1-based call index of last edit call; -1 if none
  first_repro_call           1-based call index of first run/repro command; -1 if none
  last_test_call             1-based call index of last test command; -1 if none
  ran_repro_before_edit      a run command (python/go/node/pytest/script/...) executed before
                             the first edit (or at all, when the trajectory never edits)
  ran_tests_after_last_edit  a test command executed after the last edit
  n_test_runs                test commands (pytest/go test/cargo test/npm test/jest/phpunit/mvn test)
  n_repro_calls              run commands matching REPRO_RE (superset of tests)
  n_tool_errors              tool observations matching ERROR_RE (nonzero returncode /
                             exit code, Traceback, command not found, ...)
  n_files_read               distinct file paths in view calls + read-ish bash commands
  n_files_edited             distinct file paths in edit calls (tool $.path, redirect, sed -i)
  edited_test_file           a model-patch path matches TEST_PATH_RE (the patch is the
                             submitted diff, so this excludes scratch repro scripts)
  touched_test_path          an edit *call* path matches TEST_PATH_RE (looser; includes
                             scratch test scripts that never reach the patch)
  ends_with_submit           last assistant turn matches submit/finish (see features.SUBMIT_RE)
  has_submit_call            any tool call named submit/finish/complete_task_and_submit
  stop_reason                'submit' if either submit signal else 'no_submit' (turn cap vs
                             crash are not distinguishable in this corpus)
  n_model_hunks / n_gold_hunks     parsed @@ hunks
  patch_hunk_coverage        fraction of gold hunks whose file and line range (+-HUNK_MARGIN)
                             is overlapped by a model hunk; null when gold has no hunks
  extra_hunks                model hunks overlapping no gold hunk
  extra_hunk_lines           added+removed lines inside extra hunks
  model_lines_over_gold      model changed lines / gold changed lines
  submitted_empty_patch      model patch has zero hunks AND trajectory ended with submit

Rule-based failure taxonomy (``classify_failure``, precedence order):
  timeout            ended without submit
  no_patch           submitted an empty patch
  test_edit          edited a test file
  wrong_site         coverage == 0 with extra hunks
  partial            0 <= coverage < 1, no extra hunks (stopped early)
  partial_extra      0 < coverage < 1 with extra hunks
  over_edit          coverage == 1, extras >= 2 hunks and more extra lines than gold lines
  complete_but_wrong coverage == 1, extras not large, still failed
  other              no gold hunks or otherwise unclassified

Examples:
  uv run python scripts/paired_features.py --limit 2 --head 5
  uv run python scripts/paired_features.py            # full run (resume-safe)
  uv run python scripts/paired_features.py --merge-only
"""

from __future__ import annotations

import argparse
import math
import os
import re
import sys
import time
import traceback
from dataclasses import dataclass
from pathlib import Path

import pandas as pd
from rich.console import Console

import duckdb

from .data import PARQUET_GLOB, ROOT, connect_ephemeral, list_parquet_files, parse_shard
from .features import (
    BASH_EDIT_RE,
    BASH_TEST_RE,
    EDIT_TOOL_NAMES,
    EDIT_TOOL_VIEW_ONLY,
    SUBMIT_RE,
    _sql_str,
    _tuple_sql,
    fmt_duration,
    utcnow,
)

console = Console()

PARTS_DIR = ROOT / "outputs" / "paired_features_parts"
OUT_PATH = ROOT / "outputs" / "paired_features.parquet"
ELIGIBLE_PATH = ROOT / "outputs" / "eligible_pairs.parquet"

HUNK_MARGIN = 10

SUBMIT_TOOL_NAMES = ("submit", "finish", "complete_task_and_submit")
READ_TOOL_NAMES = ("read_file", "open_file", "view_file", "view", "read")

# A command that runs code: interpreter, test runner, build tool, or script execution.
REPRO_RE = (
    r"\b(pytest|py\.test|python\d*(\.\d+)?|go\s+(test|run|build|vet)|node|deno|bun|"
    r"npm\s+(test|run|exec)|npx|yarn\s+(test|run)|pnpm\s+(test|run)|cargo\s+(test|run|build|check)|"
    r"phpunit|jest|mocha|vitest|mvn\s+(test|verify|package)|gradle(w)?\s+(test|build)|"
    r"tox|nox|rspec|rake\s+test|dotnet\s+test|make\s+test|"
    r"(ba)?sh\s+\S+\.(sh|bash)|\./\S+\.(sh|py|bash))\b"
)
# A command that reads a file without modifying it.
READ_CMD_RE = (
    r"(?:^|&&|\|\||;|\|)\s*(?:sudo\s+)?"
    r"(cat|head|tail|less|more|nl|wc|sed\s+-n|awk|grep|rg|egrep|fgrep|find|ls|tree|stat|file|diff|"
    r"git\s+(show|diff|log|blame|grep))\b"
)
# Path-ish tokens inside a shell command: contain a slash or a common source extension.
PATH_TOKEN_RE = (
    r"[A-Za-z0-9_~.-]*/[A-Za-z0-9_~./-]+|"
    r"[A-Za-z0-9_-]+\.(py|go|js|ts|tsx|jsx|java|kt|rb|php|rs|c|cc|cpp|h|hpp|cs|"
    r"toml|ya?ml|json|xml|md|txt|sh|cfg|ini|sql|html|css|proto|scala|lua|pl|ex|exs|erl|hrl|swift)"
)
REDIRECT_TARGET_RE = r">>?\s*([A-Za-z0-9_~./-]+)"
SED_TARGET_RE = r"sed\s+[^;&|]*-i[^;&|]*?\s+([^\s;&|]+)\s*$"
TEST_PATH_RE = r"(^|/)(tests?|testing|testdata|test_data|conftest|spec)/|test_|_test\.|_spec\.|\.spec\.|\.test\.|conftest"

# Tool-observation error signals across the three harnesses:
#   minisweagent: JSON {"returncode": N, ...}; openhands: "[The command completed with exit code N.]";
#   sweagent: OBSERVATION text containing the command's stderr.
ERROR_RE = (
    r'"returncode"\s*:\s*(-?0*[1-9][0-9]*)'
    r"|exit code\s+(-?0*[1-9][0-9]*)"
    r"|Traceback \(most recent call last\)"
    r"|command (timed out|not found)"
    r"|(?i)(no such file or directory|permission denied|segmentation fault)"
)

PAIRED_SQL = r"""
WITH src AS (
    SELECT
        t.trajectory_id,
        t.instance_id,
        t.repo,
        t.language,
        t.resolved,
        t.messages,
        t.metadata.category AS category,
        coalesce(t.metadata.model_patch.patch, '') AS model_patch,
        coalesce(t.metadata.reference_patch.patch, '') AS gold_patch
    FROM read_parquet('{file_sql}', union_by_name=true) t
    WHERE t.trajectory_id IN (SELECT trajectory_id FROM eligible_ids)
),
msg AS (
    SELECT
        t.trajectory_id,
        count(*) FILTER (WHERE m.role = 'assistant') AS n_assistant_turns,
        sum(length(coalesce(m.content, ''))) FILTER (WHERE m.role = 'assistant') AS assistant_chars,
        count(*) FILTER (
            WHERE m.role = 'tool' AND regexp_matches(coalesce(m.content, ''), '{error_re}')
        ) AS n_tool_errors
    FROM src t, unnest(t.messages) AS u(m)
    GROUP BY 1
),
calls AS (
    SELECT
        t.trajectory_id,
        row_number() OVER (PARTITION BY t.trajectory_id ORDER BY mi, ci) AS call_idx,
        lower(coalesce(tc.function.name, '')) AS tool_name,
        lower(coalesce(json_extract_string(try_cast(tc.function.arguments AS JSON), '$.command'), '')) AS command_text,
        coalesce(
            json_extract_string(try_cast(tc.function.arguments AS JSON), '$.path'),
            json_extract_string(try_cast(tc.function.arguments AS JSON), '$.file_path'),
            json_extract_string(try_cast(tc.function.arguments AS JSON), '$.filename')
        ) AS tool_path
    FROM src t,
         unnest(t.messages) WITH ORDINALITY AS u(m, mi),
         unnest(m.tool_calls) WITH ORDINALITY AS v(tc, ci)
),
classified AS (
    SELECT
        trajectory_id,
        call_idx,
        tool_name,
        CASE
            WHEN tool_name = '{edit_view_only}'
                THEN (command_text <> 'view')
            WHEN tool_name IN ({edit_tool_names}) THEN true
            WHEN regexp_matches(command_text, '{bash_edit_re}') THEN true
            ELSE false
        END AS is_edit,
        regexp_matches(command_text, '{bash_test_re}') AS is_test,
        regexp_matches(command_text, '{repro_re}') AS is_repro,
        (
            (tool_name = '{edit_view_only}' AND command_text = 'view')
            OR tool_name IN ({read_tool_names})
            OR regexp_matches(command_text, '{read_cmd_re}')
        ) AS is_read,
        CASE
            WHEN tool_path IS NOT NULL AND (command_text <> 'view' OR tool_name <> '{edit_view_only}')
                THEN [tool_path]
            WHEN tool_path IS NOT NULL THEN []
            WHEN regexp_matches(command_text, '{bash_edit_re}') THEN
                list_filter(
                    list_concat(
                        regexp_extract_all(command_text, '{redirect_re}', 1),
                        regexp_extract_all(command_text, '{sed_re}', 1)
                    ),
                    p -> regexp_matches(p, '[./]')
                )
            ELSE []
        END AS edit_paths,
        CASE
            WHEN tool_path IS NOT NULL AND tool_name = '{edit_view_only}' AND command_text = 'view'
                THEN [tool_path]
            WHEN tool_path IS NOT NULL AND tool_name IN ({read_tool_names})
                THEN [tool_path]
            WHEN regexp_matches(command_text, '{read_cmd_re}') THEN
                list_filter(
                    regexp_extract_all(command_text, '{path_token_re}'),
                    p -> regexp_matches(p, '[./]') AND NOT regexp_matches(p, '^(s/|http)')
                )
            ELSE []
        END AS read_paths
    FROM calls
),
call_feats AS (
    SELECT
        trajectory_id,
        count(*) AS n_tool_calls,
        count(*) FILTER (WHERE is_edit) AS n_edit_calls,
        count(*) FILTER (WHERE is_test) AS n_test_runs,
        count(*) FILTER (WHERE is_repro) AS n_repro_calls,
        min(call_idx) FILTER (WHERE is_edit) AS first_edit_call,
        max(call_idx) FILTER (WHERE is_edit) AS last_edit_call,
        min(call_idx) FILTER (WHERE is_repro) AS first_repro_call,
        max(call_idx) FILTER (WHERE is_test) AS last_test_call,
        count(*) FILTER (WHERE tool_name IN ({submit_tool_names})) AS n_submit_calls,
        len(list_distinct(flatten(list(edit_paths) FILTER (WHERE is_edit)))) AS n_files_edited,
        list_distinct(flatten(list(edit_paths) FILTER (WHERE is_edit))) AS edited_paths,
        len(list_distinct(flatten(list(read_paths) FILTER (WHERE is_read)))) AS n_files_read
    FROM classified
    GROUP BY 1
),
ends AS (
    SELECT
        trajectory_id,
        regexp_matches(
            lower(
                coalesce(list_last(list_filter(messages, m -> m.role = 'assistant')).content, '')
                || ' ' ||
                coalesce(cast(list_last(list_filter(messages, m -> m.role = 'assistant')).tool_calls AS VARCHAR), '')
            ),
            '{submit_re}'
        ) AS ends_with_submit
    FROM src
)
SELECT
    t.trajectory_id,
    '{harness}' AS harness,
    '{teacher}' AS teacher,
    '{source}' AS source,
    t.instance_id,
    t.repo,
    t.language,
    t.category,
    t.resolved::TINYINT AS resolved,
    len(t.messages)::INTEGER AS n_turns,
    coalesce(msg.n_assistant_turns, 0)::INTEGER AS n_assistant_turns,
    coalesce(msg.assistant_chars, 0)::BIGINT AS assistant_chars,
    coalesce(msg.n_tool_errors, 0)::INTEGER AS n_tool_errors,
    coalesce(cf.n_tool_calls, 0)::INTEGER AS n_tool_calls,
    coalesce(cf.n_edit_calls, 0)::INTEGER AS n_edit_calls,
    coalesce(cf.n_test_runs, 0)::INTEGER AS n_test_runs,
    coalesce(cf.n_repro_calls, 0)::INTEGER AS n_repro_calls,
    coalesce(cf.first_edit_call, -1)::INTEGER AS turn_of_first_edit,
    coalesce(cf.last_edit_call, -1)::INTEGER AS turn_of_last_edit,
    coalesce(cf.first_repro_call, -1)::INTEGER AS first_repro_call,
    coalesce(cf.last_test_call, -1)::INTEGER AS last_test_call,
    coalesce(cf.n_files_edited, 0)::INTEGER AS n_files_edited,
    coalesce(cf.edited_paths, []::VARCHAR[]) AS edited_paths,
    coalesce(cf.n_files_read, 0)::INTEGER AS n_files_read,
    coalesce(cf.n_submit_calls, 0)::INTEGER AS n_submit_calls,
    e.ends_with_submit,
    t.model_patch,
    t.gold_patch
FROM src t
LEFT JOIN msg ON msg.trajectory_id = t.trajectory_id
LEFT JOIN call_feats cf ON cf.trajectory_id = t.trajectory_id
LEFT JOIN ends e ON e.trajectory_id = t.trajectory_id
"""


def build_paired_sql(file_path: Path, harness: str, teacher: str, source: str) -> str:
    """Per-shard extraction SQL; expects an ``eligible_ids(trajectory_id)`` view."""
    return PAIRED_SQL.format(
        file_sql=str(file_path).replace("'", "''"),
        harness=harness.replace("'", "''"),
        teacher=teacher.replace("'", "''"),
        source=source.replace("'", "''"),
        edit_view_only=EDIT_TOOL_VIEW_ONLY,
        edit_tool_names=_tuple_sql(EDIT_TOOL_NAMES),
        read_tool_names=_tuple_sql(READ_TOOL_NAMES),
        submit_tool_names=_tuple_sql(SUBMIT_TOOL_NAMES),
        bash_edit_re=BASH_EDIT_RE.replace("'", "''"),
        bash_test_re=BASH_TEST_RE.replace("'", "''"),
        repro_re=REPRO_RE.replace("'", "''"),
        read_cmd_re=READ_CMD_RE.replace("'", "''"),
        path_token_re=PATH_TOKEN_RE.replace("'", "''"),
        redirect_re=REDIRECT_TARGET_RE.replace("'", "''"),
        sed_re=SED_TARGET_RE.replace("'", "''"),
        submit_re=SUBMIT_RE.replace("'", "''"),
        error_re=ERROR_RE.replace("'", "''"),
    )


# --- unified-diff parsing (pure Python, unit-tested) -------------------------

DIFF_GIT_RE = re.compile(r"^diff --git a/\S+ b/(\S+)")
HUNK_RE = re.compile(r"^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@")


@dataclass(frozen=True)
class Hunk:
    path: str
    start: int  # first new-side line number
    end: int  # last new-side line number (inclusive)
    n_add: int
    n_del: int

    @property
    def changed(self) -> int:
        return self.n_add + self.n_del


def parse_patch(patch: str) -> list[Hunk]:
    """Parse a unified diff into hunks with new-side line ranges."""
    parsed: list[tuple[str, list[int]]] = []
    path = ""
    cur: list[int] | None = None  # [start, end, n_add, n_del]
    for line in patch.splitlines():
        m = DIFF_GIT_RE.match(line)
        if m:
            path = m.group(1)
            cur = None
            continue
        if line.startswith("+++"):
            p = line[4:].strip()
            path = p.removeprefix("b/")
            continue
        if line.startswith("--- "):
            cur = None
            continue
        m = HUNK_RE.match(line)
        if m:
            start = int(m.group(1))
            span = int(m.group(2)) if m.group(2) else 1
            cur = [start, start + max(span - 1, 0), 0, 0]
            parsed.append((path, cur))
            continue
        if cur is not None:
            if line.startswith("+"):
                cur[2] += 1
            elif line.startswith("-"):
                cur[3] += 1
    return [Hunk(p, c[0], c[1], c[2], c[3]) for p, c in parsed]


def _overlaps(h: Hunk, g: Hunk, margin: int) -> bool:
    return h.path == g.path and h.start <= g.end + margin and h.end >= g.start - margin


def hunk_features(model_patch: str, gold_patch: str, margin: int = HUNK_MARGIN) -> dict:
    """Patch-level features: hunk coverage of the gold patch and off-target hunks."""
    model = parse_patch(model_patch)
    gold = parse_patch(gold_patch)
    gold_lines = sum(h.changed for h in gold)
    model_lines = sum(h.changed for h in model)
    covered = sum(1 for g in gold if any(_overlaps(m, g, margin) for m in model))
    extra = [m for m in model if not any(_overlaps(m, g, margin) for g in gold)]
    return {
        "n_model_hunks": len(model),
        "n_gold_hunks": len(gold),
        "patch_hunk_coverage": (covered / len(gold)) if gold else None,
        "extra_hunks": len(extra),
        "extra_hunk_lines": sum(h.changed for h in extra),
        "gold_changed_lines": gold_lines,
        "model_changed_lines": model_lines,
        "model_lines_over_gold": (model_lines / gold_lines) if gold_lines else None,
        "model_patch_paths": sorted({h.path for h in model}),
    }


def is_test_path(path: str) -> bool:
    return bool(re.search(TEST_PATH_RE, path.lower()))


def add_patch_features(df: pd.DataFrame) -> pd.DataFrame:
    """Add the Python-side per-trajectory features to a shard's SQL result frame."""
    if len(df) == 0:
        for col in (
            "n_model_hunks",
            "n_gold_hunks",
            "patch_hunk_coverage",
            "extra_hunks",
            "extra_hunk_lines",
            "gold_changed_lines",
            "model_changed_lines",
            "model_lines_over_gold",
        ):
            df[col] = pd.Series(dtype="float64")
        for col in (
            "edited_test_file",
            "touched_test_path",
            "ran_repro_before_edit",
            "ran_tests_after_last_edit",
            "has_submit_call",
            "submitted_empty_patch",
        ):
            df[col] = pd.Series(dtype="bool")
        df["stop_reason"] = pd.Series(dtype="object")
        return df.drop(columns=["model_patch", "gold_patch"])

    hf = df.apply(
        lambda r: hunk_features(r["model_patch"], r["gold_patch"]), axis=1, result_type="expand"
    )
    df = pd.concat([df, hf], axis=1)
    patch_test = df["model_patch_paths"].apply(lambda ps: any(is_test_path(p) for p in ps))
    call_test = df["edited_paths"].apply(
        lambda ps: any(is_test_path(p) for p in (ps if ps is not None else []))
    )
    df["edited_test_file"] = patch_test.astype(bool)
    df["touched_test_path"] = call_test.astype(bool)

    df["ran_repro_before_edit"] = (
        (df["first_repro_call"] != -1)
        & ((df["turn_of_first_edit"] == -1) | (df["first_repro_call"] < df["turn_of_first_edit"]))
    ).astype(bool)
    df["ran_tests_after_last_edit"] = (
        (df["last_test_call"] != -1)
        & (df["turn_of_last_edit"] != -1)
        & (df["last_test_call"] > df["turn_of_last_edit"])
    ).astype(bool)
    df["has_submit_call"] = df["n_submit_calls"] > 0
    df["stop_reason"] = (df["ends_with_submit"] | df["has_submit_call"]).map(
        {True: "submit", False: "no_submit"}
    )
    df["submitted_empty_patch"] = (df["n_model_hunks"] == 0) & (df["stop_reason"] == "submit")
    return df.drop(columns=["model_patch", "gold_patch", "model_patch_paths"])


FEATURE_COLUMNS = [
    "n_turns",
    "n_assistant_turns",
    "n_tool_calls",
    "assistant_chars",
    "n_tool_errors",
    "n_edit_calls",
    "n_test_runs",
    "n_repro_calls",
    "turn_of_first_edit",
    "turn_of_last_edit",
    "n_files_read",
    "n_files_edited",
    "n_model_hunks",
    "n_gold_hunks",
    "patch_hunk_coverage",
    "extra_hunks",
    "extra_hunk_lines",
    "model_lines_over_gold",
]

FLAG_COLUMNS = [
    "ran_repro_before_edit",
    "ran_tests_after_last_edit",
    "edited_test_file",
    "touched_test_path",
    "ends_with_submit",
    "has_submit_call",
    "submitted_empty_patch",
]


# --- rule-based failure taxonomy ---------------------------------------------

OVER_EDIT_MIN_EXTRA_HUNKS = 2


def classify_failure(
    row: pd.Series | dict, over_edit_min_extra_hunks: int = OVER_EDIT_MIN_EXTRA_HUNKS
) -> str:
    """Assign a FAIL trajectory to one taxonomy bucket. Precedence matters."""
    cov = row.get("patch_hunk_coverage")
    if cov is not None and isinstance(cov, float) and math.isnan(cov):
        cov = None
    if row.get("stop_reason") != "submit":
        return "timeout"
    if row.get("n_model_hunks", 0) == 0:
        return "no_patch"
    if row.get("edited_test_file"):
        return "test_edit"
    if cov is None:
        return "other"
    extra = row.get("extra_hunks", 0)
    if cov == 0:
        return "wrong_site" if extra > 0 else "other"
    if cov < 1:
        return "partial" if extra == 0 else "partial_extra"
    if extra >= over_edit_min_extra_hunks and row.get("extra_hunk_lines", 0) > row.get(
        "gold_changed_lines", 0
    ):
        return "over_edit"
    return "complete_but_wrong"


TAXONOMY = [
    "timeout",
    "no_patch",
    "test_edit",
    "wrong_site",
    "partial",
    "partial_extra",
    "over_edit",
    "complete_but_wrong",
    "other",
]


# --- extraction runner --------------------------------------------------------


def part_path_for_shard(file_path: Path, parts_dir: Path = PARTS_DIR) -> Path:
    rel = parse_shard(file_path).rel
    return parts_dir / ("__".join(rel.with_suffix("").parts) + ".parquet")


def process_shard(
    con: duckdb.DuckDBPyConnection,
    file_path: Path,
    *,
    force: bool = False,
    parts_dir: Path = PARTS_DIR,
) -> tuple[str, int]:
    """Extract features for eligible trajectories in one shard. Returns (status, rows)."""
    part = part_path_for_shard(file_path, parts_dir)
    if part.exists() and part.stat().st_size > 0 and not force:
        return "skip", 0

    shard = parse_shard(file_path)
    expected = con.execute(
        f"SELECT count(*) FROM read_parquet({_sql_str(file_path)}, union_by_name=true) "
        "WHERE trajectory_id IN (SELECT trajectory_id FROM eligible_ids)"
    ).fetchone()[0]

    df = con.execute(
        build_paired_sql(file_path, shard.harness, shard.teacher, shard.source)
    ).fetchdf()
    if len(df) != expected:
        raise RuntimeError(f"row guard failed: expected={expected} got={len(df)}")
    df = add_patch_features(df)

    tmp = Path(f"{part}.{os.getpid()}.tmp")
    tmp.parent.mkdir(parents=True, exist_ok=True)
    try:
        con.register("stage_df", df)
        con.execute(f"COPY stage_df TO {_sql_str(tmp)} (FORMAT PARQUET)")
        con.unregister("stage_df")
    except Exception:
        tmp.unlink(missing_ok=True)
        raise
    os.replace(tmp, part)
    return "done", len(df)


def load_eligible_ids(con: duckdb.DuckDBPyConnection, eligible_path: Path = ELIGIBLE_PATH) -> int:
    df = pd.read_parquet(eligible_path, columns=["trajectory_id"])
    con.register("eligible_ids", df[["trajectory_id"]].drop_duplicates())
    return len(df)


def paired_features(args: argparse.Namespace) -> int:
    PARTS_DIR.mkdir(parents=True, exist_ok=True)
    OUT_PATH.parent.mkdir(parents=True, exist_ok=True)

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
    n_eligible = load_eligible_ids(con, Path(args.eligible))
    console.print(f"[{utcnow()}] eligible trajectories: {n_eligible:,}")

    done = skipped = failed = 0
    elapsed_total = 0.0
    started = time.monotonic()
    failures: list[str] = []

    try:
        for i, file_path in enumerate(files, start=1):
            shard = parse_shard(file_path)
            label = shard.label
            t0 = time.monotonic()
            try:
                status, rows = process_shard(con, file_path, force=args.force)
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

        if not args.no_merge:
            merged = merge_paired_parts(con)
            if merged is None:
                console.print(f"[{utcnow()}] no parts to merge")
            else:
                n_parts, rows = merged
                suffix = (
                    ""
                    if not args.file and not args.limit
                    else " (partial run: rerun without --file/--limit to cover the corpus)"
                )
                console.print(
                    f"[{utcnow()}] merged {n_parts} parts → {OUT_PATH.relative_to(ROOT)}: "
                    f"{rows:,} rows{suffix}"
                )
                if args.head:
                    print_head(con, OUT_PATH, args.head)
    except KeyboardInterrupt:
        console.print(f"[{utcnow()}] interrupted — parts written so far are kept, rerun to resume")
        return 130
    finally:
        con.close()

    console.print(
        f"[{utcnow()}] finished: processed={done} skipped={skipped} failed={failed} "
        f"in {fmt_duration(time.monotonic() - started)}"
    )
    if failures:
        console.print(f"[red]failed shards[/red]: {', '.join(failures)}")
        return 1
    return 0


def merge_paired_parts(
    con: duckdb.DuckDBPyConnection,
    parts_dir: Path = PARTS_DIR,
    out_path: Path = OUT_PATH,
) -> tuple[int, int] | None:
    parts = sorted(parts_dir.glob("*.parquet"))
    if not parts:
        return None
    glob_sql = str(parts_dir / "*.parquet").replace("'", "''")
    tmp = Path(str(out_path) + ".tmp")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    con.execute(
        f"COPY (SELECT * FROM read_parquet('{glob_sql}', union_by_name=true) "
        "ORDER BY harness, teacher, source, trajectory_id) "
        f"TO {_sql_str(tmp)} (FORMAT PARQUET)"
    )
    rows = con.execute(f"SELECT count(*) FROM read_parquet({_sql_str(tmp)})").fetchone()[0]
    os.replace(tmp, out_path)
    return len(parts), rows


def print_head(con: duckdb.DuckDBPyConnection, path: Path, n: int) -> None:
    path_sql = _sql_str(path)
    df = con.execute(
        f"SELECT trajectory_id, resolved, n_turns, n_tool_calls, n_edit_calls, n_test_runs, "
        f"turn_of_first_edit, turn_of_last_edit, ran_repro_before_edit, "
        f"ran_tests_after_last_edit, n_files_read, n_files_edited, edited_test_file, "
        f"n_tool_errors, patch_hunk_coverage, extra_hunks, model_lines_over_gold, "
        f"stop_reason FROM read_parquet({path_sql}) LIMIT {n}"
    ).fetchdf()
    console.print(df.to_string(index=False, max_colwidth=24))
    checks = (
        con.execute(
            f"""
        SELECT
            count(*) AS rows,
            count(DISTINCT trajectory_id) AS tids,
            avg(n_turns) AS avg_turns,
            avg(n_tool_calls) AS avg_calls,
            avg(ran_repro_before_edit::INT) AS repro_rate,
            avg(ran_tests_after_last_edit::INT) AS test_after_rate,
            avg(edited_test_file::INT) AS test_edit_rate,
            avg(patch_hunk_coverage) AS avg_cov,
            avg(extra_hunks) AS avg_extra,
            count(*) FILTER (WHERE stop_reason='submit') AS submitted,
            count(*) FILTER (WHERE n_model_hunks=0) AS empty_patch
        FROM read_parquet({path_sql})
        """
        )
        .fetchdf()
        .iloc[0]
    )
    console.print(dict(checks))


def main() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument("--file", action="append", help="Process only these shard(s); repeatable")
    parser.add_argument("--limit", type=int, help="Process at most N shards (sorted order)")
    parser.add_argument(
        "--data-glob", default=PARQUET_GLOB, help="Parquet glob (default: the corpus)"
    )
    parser.add_argument("--eligible", default=str(ELIGIBLE_PATH))
    parser.add_argument("--force", action="store_true", help="Recompute parts that already exist")
    parser.add_argument("--no-merge", action="store_true", help="Leave shard parts unmerged")
    parser.add_argument(
        "--merge-only", action="store_true", help="Skip processing; merge existing parts"
    )
    parser.add_argument("--head", type=int, default=0, help="Print first N merged rows + summary")
    parser.add_argument("--threads", type=int, default=4)
    parser.add_argument("--memory-limit", default="4GB")
    args = parser.parse_args()

    if args.merge_only:
        con = connect_ephemeral()
        con.execute(f"SET memory_limit='{args.memory_limit}'")
        try:
            merged = merge_paired_parts(con)
            if merged is None:
                raise SystemExit(f"No parts found in {PARTS_DIR}")
            n_parts, rows = merged
            console.print(f"[{utcnow()}] merged {n_parts} parts → {OUT_PATH}: {rows:,} rows")
            if args.head:
                print_head(con, OUT_PATH, args.head)
        finally:
            con.close()
        return

    sys.exit(paired_features(args))


if __name__ == "__main__":
    main()

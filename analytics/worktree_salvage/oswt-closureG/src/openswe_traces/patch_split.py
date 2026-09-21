r"""Gold-patch split by file class: source vs test vs doc vs config, per instance.

Stage 1 (one streaming DuckDB pass, resume-safe): every corpus shard is read once and
aggregated to one row per ``instance_id`` — labeled-rollout counts, one representative
gold patch, and ``hf_dataset_name`` for the upstream-metadata join. Per-shard parts land
in ``outputs/patch_split_parts/`` (atomic, skipped on rerun) and fold into
``outputs/patch_split.parquet``.

File classes (checked in order ``test`` → ``config`` → ``doc`` → ``src``; the first
match wins):

- ``test``: a path component in {test, tests, testing, __tests__, spec, specs, e2e}, or a
  basename matching test conventions (``test_*``, ``*_test``, ``*_tests``, ``Test*``,
  ``*Test``, ``*Tests``, ``*.test.*``, ``*.spec.*``, ``*_spec.*``, ``conftest.py``).
- ``config``: extension yaml/yml/toml/json/lock/ini/cfg/conf/properties/env/xml, or a
  known config basename (Dockerfile, Makefile, Gemfile, requirements*.txt, dotfiles...).
- ``doc``: extension md/markdown/rst/txt/adoc, a changelog/readme/license basename, or a
  ``docs``/``doc`` path component.
- ``src``: everything else.

``n_new_test_funcs`` counts added lines matching per-language test-function definition
patterns (python ``def test_*``, go ``func TestX(``, ts/js ``it(/test(/describe(``, java
``@Test``, rust ``#[test]``-family, c/cpp ``TEST(``, php ``function test*``, ruby
``def test_`` / ``it '...'``, csharp ``[Test]/[Fact]/[Theory]``). It is the proxy for how
much behavior the verifier checks. Unknown languages get the union of all patterns.

``test_only`` is literal: the patch changes no ``src`` lines (``src_added == 0`` and
``src_removed == 0``) and touches at least one file.

Examples:
  uv run python scripts/patch_split.py --limit 3
  uv run python scripts/patch_split.py --skip-stream
"""

from __future__ import annotations

import argparse
import os
import re
import sys
import time
import traceback
from pathlib import Path

import pandas as pd
from rich.console import Console

import duckdb

from .data import DATA_ROOT, PARQUET_GLOB, ROOT, connect_ephemeral, list_parquet_files
from .difficulty import rel
from .features import _sql_str, fmt_duration, utcnow

console = Console()

PARTS_DIR = ROOT / "outputs" / "patch_split_parts"
OUT_PARQUET = ROOT / "outputs" / "patch_split.parquet"

LANGUAGE_ALIASES = {"ts": "typescript", "js": "javascript"}

SPLIT_COLUMNS = [
    "src_added",
    "src_removed",
    "test_added",
    "test_removed",
    "doc_added",
    "doc_removed",
    "config_added",
    "config_removed",
    "n_src_files",
    "n_test_files",
    "n_doc_files",
    "n_config_files",
    "n_new_test_funcs",
    "test_only",
]

# --- file classification -------------------------------------------------------

TEST_DIR_COMPONENTS = frozenset(
    {"test", "tests", "testing", "__tests__", "spec", "specs", "e2e"}
)
TEST_BASENAME_RES = [
    re.compile(r"^test_.+\.", re.IGNORECASE),
    re.compile(r".+_tests?\.", re.IGNORECASE),
    re.compile(r"^Test.+\.java$"),
    re.compile(r".+(?:Test|Tests|IT|ITCase)\.java$"),
    re.compile(r".+\.(?:test|spec)\.", re.IGNORECASE),
    re.compile(r".+_spec\.", re.IGNORECASE),
    re.compile(r"^conftest\.py$"),
]
DOC_EXTENSIONS = frozenset({".md", ".markdown", ".rst", ".txt", ".adoc"})
DOC_BASENAMES = frozenset(
    {
        "changelog",
        "changes",
        "history",
        "news",
        "license",
        "licence",
        "copying",
        "notice",
        "authors",
        "contributors",
        "readme",
        "install",
        "todo",
        "codeowners",
        "contributing",
    }
)
DOC_DIR_COMPONENTS = frozenset({"docs", "doc", "documentation"})
CONFIG_EXTENSIONS = frozenset(
    {
        ".yaml",
        ".yml",
        ".toml",
        ".json",
        ".jsonl",
        ".lock",
        ".ini",
        ".cfg",
        ".conf",
        ".properties",
        ".env",
        ".xml",
        ".gradle",
    }
)
CONFIG_BASENAME_RES = [
    re.compile(r"^requirements.*\.txt$"),
    re.compile(
        r"^(?:dockerfile|makefile|rakefile|gemfile|podfile|vagrantfile|brewfile|"
        r"cmakelists\.txt|manifest\.in|setup\.cfg|\.gitignore|\.gitattributes|"
        r"\.editorconfig|\.dockerignore|\.pre-commit-config\.yaml|\.babelrc|"
        r"\.eslintrc.*|\.prettierrc.*|\.stylelintrc.*|package\.json|"
        r"package-lock\.json|yarn\.lock|pnpm-lock\.yaml|poetry\.lock|pipfile.*|"
        r"cargo\.lock|go\.mod|go\.sum|composer\.(?:json|lock)|gemfile\.lock)$",
        re.IGNORECASE,
    ),
]

FILE_CLASSES = ("src", "test", "doc", "config")


def classify_file(path: str) -> str:
    """Map a diff path to one of {test, config, doc, src}; first match wins."""
    parts = [p for p in path.strip().split("/") if p and p not in (".", "..")]
    if not parts:
        return "src"
    basename = parts[-1]
    stem, dot, ext = basename.rpartition(".")
    if dot:
        ext = "." + ext.lower()
        stem_lower = stem.lower()
    else:
        ext = ""
        stem_lower = basename.lower()

    dir_parts = {p.lower() for p in parts[:-1]}
    if dir_parts & TEST_DIR_COMPONENTS or any(r.match(basename) for r in TEST_BASENAME_RES):
        return "test"
    if ext in CONFIG_EXTENSIONS or any(r.match(basename) for r in CONFIG_BASENAME_RES):
        return "config"
    if (
        ext in DOC_EXTENSIONS
        or stem_lower in DOC_BASENAMES
        or dir_parts & DOC_DIR_COMPONENTS
    ):
        return "doc"
    return "src"


# --- test-function patterns -----------------------------------------------------

TEST_FUNC_PATTERNS: dict[str, list[re.Pattern]] = {
    "python": [
        re.compile(r"^\s*(?:async\s+)?def\s+test\w*\s*\("),
        re.compile(r"^\s*class\s+Test\w*\s*[(:]"),
    ],
    "go": [
        re.compile(r"^\s*func\s+(?:Test|Benchmark|Fuzz|Example)[A-Z0-9_]\w*\s*\("),
    ],
    "typescript": [
        re.compile(r"\b(?:it|test|describe)\s*\(\s*['\"]"),
        re.compile(r"^\s*(?:export\s+)?(?:async\s+)?function\s+test\w*\s*\(", re.IGNORECASE),
        re.compile(r"^\s*(?:it|test|describe)\s*\.\s*\w+\s*\(\s*['\"]"),
    ],
    "java": [
        # One annotation per test method; JUnit3 `void testX()` methods are rare and
        # counting them too would double-count every annotated test.
        re.compile(r"^\s*@(Test|ParameterizedTest|RepeatedTest|TestFactory)\b"),
    ],
    "rust": [
        re.compile(r"^\s*#\[[\w:]*test[\w:]*(?:\([^)]*\))?\]"),
        re.compile(r"^\s*(?:pub\s+)?(?:async\s+)?fn\s+test_\w+\s*\("),
    ],
    "c": [
        re.compile(r"^\s*TEST(?:_F|_P|_CASE)?\s*\("),
    ],
    "php": [
        re.compile(r"\bfunction\s+test\w*\s*\(", re.IGNORECASE),
        re.compile(r"^\s*(?:/\*\*\s*)?@test\b", re.IGNORECASE),
    ],
    "ruby": [
        re.compile(r"^\s*def\s+test_\w+"),
        re.compile(r"^\s*(?:it|test|specify|scenario)\s*['\"]"),
    ],
    "csharp": [
        re.compile(r"^\s*\[(?:Test|Fact|Theory|TestCase|TestMethod)\b"),
        re.compile(r"\bvoid\s+Test\w*\s*\("),
    ],
}
TEST_FUNC_PATTERNS["cpp"] = TEST_FUNC_PATTERNS["c"]
TEST_FUNC_PATTERNS["javascript"] = TEST_FUNC_PATTERNS["typescript"]

_ALL_TEST_FUNC_PATTERNS = [p for pats in TEST_FUNC_PATTERNS.values() for p in pats]


def normalize_language(value: str | None) -> str:
    return LANGUAGE_ALIASES.get((value or "").strip().lower(), (value or "").strip().lower())


def _test_func_patterns(language: str) -> list[re.Pattern]:
    return TEST_FUNC_PATTERNS.get(language, _ALL_TEST_FUNC_PATTERNS)


# --- patch parsing ---------------------------------------------------------------

_DIFF_GIT_RE = re.compile(r"^diff --git a/(\S+) b/(\S+)")
_PLUS_PATH_RE = re.compile(r"^\+\+\+\s+(?:b/)?(\S+)")


def _path_from_block(block: str) -> str:
    """Best-effort path for one diff block: b-side of diff --git, else +++ path."""
    for line in block.splitlines()[:6]:
        m = _PLUS_PATH_RE.match(line)
        if m and m.group(1) != "/dev/null":
            return m.group(1)
    return ""


def split_patch_by_class(patch: str) -> dict[str, dict]:
    """Per-class added/removed line counts and file counts for one unified diff."""
    counts = {c: {"added": 0, "removed": 0, "files": 0, "added_lines": []} for c in FILE_CLASSES}
    blocks = re.split(r"^diff --git ", patch or "", flags=re.MULTILINE)
    for i, block in enumerate(blocks):
        if not block.strip():
            continue
        if i > 0:
            m = _DIFF_GIT_RE.match("diff --git " + block.splitlines()[0])
            path = m.group(2) if m else _path_from_block(block)
        else:
            path = _path_from_block(block)
            has_header = any(
                line.startswith(("--- ", "+++ ", "@@")) for line in block.splitlines()[:6]
            )
            if not has_header and not path:
                continue
        cls = classify_file(path) if path else "src"
        added: list[str] = []
        removed = 0
        in_hunk = False
        for line in block.splitlines():
            if line.startswith("@@"):
                in_hunk = True
                continue
            if not in_hunk:
                continue
            if line.startswith("+") and not line.startswith("+++"):
                added.append(line[1:])
            elif line.startswith("-") and not line.startswith("---"):
                removed += 1
        counts[cls]["added"] += len(added)
        counts[cls]["removed"] += removed
        counts[cls]["files"] += 1
        counts[cls]["added_lines"].extend(added)
    return counts


def count_new_test_funcs(added_lines: list[str], language: str) -> int:
    """Added lines that define a test function, per-language regex (union if unknown)."""
    patterns = _test_func_patterns(language)
    return sum(1 for line in added_lines if any(p.search(line) for p in patterns))


def compute_patch_split(patch: str, language: str) -> dict[str, int]:
    """All split features for one gold patch. Pure; no I/O."""
    counts = split_patch_by_class(patch or "")
    all_added = [ln for c in FILE_CLASSES for ln in counts[c]["added_lines"]]
    out: dict[str, int] = {}
    for c in FILE_CLASSES:
        out[f"{c}_added"] = counts[c]["added"]
        out[f"{c}_removed"] = counts[c]["removed"]
        out[f"n_{c}_files"] = counts[c]["files"]
    out["n_new_test_funcs"] = count_new_test_funcs(all_added, normalize_language(language))
    n_files = sum(counts[c]["files"] for c in FILE_CLASSES)
    out["test_only"] = int(n_files > 0 and out["src_added"] == 0 and out["src_removed"] == 0)
    return out


# --- streaming pass --------------------------------------------------------------

SHARD_SQL = r"""
SELECT
    instance_id,
    max(repo) AS repo,
    max(language) AS language,
    max(hf_dataset_name) AS hf_dataset_name,
    count(*)::INTEGER AS n_rollouts,
    count(*) FILTER (WHERE resolved IN (0, 1))::INTEGER AS n_labeled,
    count(*) FILTER (WHERE resolved = 1)::INTEGER AS n_resolved,
    max(metadata.reference_patch.patch) AS gold_patch
FROM read_parquet('{file_sql}', union_by_name=true)
GROUP BY 1
"""


def build_shard_sql(file_path: Path) -> str:
    return SHARD_SQL.format(file_sql=str(file_path).replace("'", "''"))


def _data_rel(path: Path) -> Path:
    """Shard path relative to DATA_ROOT, tolerating the traces_data symlink."""
    path = Path(path)
    try:
        return path.relative_to(DATA_ROOT)
    except ValueError:
        return path.resolve().relative_to(DATA_ROOT.resolve())


def shard_label(path: Path) -> str:
    rel = _data_rel(path)
    return f"{rel.parts[0]}/{rel.parts[1]}/{rel.parts[2]}/{path.name}"


def part_path_for(file_path: Path, parts_dir: Path = PARTS_DIR) -> Path:
    rel = _data_rel(file_path)
    return parts_dir / ("__".join(rel.with_suffix("").parts) + ".parquet")


def split_dataframe(rows: list[dict]) -> pd.DataFrame:
    df = pd.DataFrame(rows)
    df["has_gold_patch"] = (df["gold_patch"].fillna("") != "").astype("int8")
    for col in SPLIT_COLUMNS:
        df[col] = df[col].astype("int64")
    return df.drop(columns=["gold_patch"])


def process_shard(
    con: duckdb.DuckDBPyConnection,
    file_path: Path,
    *,
    force: bool,
    parts_dir: Path = PARTS_DIR,
) -> tuple[str, int]:
    """Write one per-instance part for a shard; 'skip' when the part already exists."""
    part = part_path_for(file_path, parts_dir)
    if part.exists() and part.stat().st_size > 0 and not force:
        return "skip", 0

    src_rows = con.execute(f"SELECT count(*) FROM read_parquet({_sql_str(file_path)})").fetchone()[0]
    rows = con.execute(build_shard_sql(file_path)).fetchall()
    if not rows:
        raise RuntimeError("empty shard")
    out = []
    for r in rows:
        split = compute_patch_split(str(r[7] or ""), str(r[2] or ""))
        out.append(
            {
                "instance_id": r[0],
                "repo": r[1],
                "language": r[2],
                "hf_dataset_name": r[3],
                "n_rollouts": int(r[4]),
                "n_labeled": int(r[5]),
                "n_resolved": int(r[6]),
                "gold_patch": r[7],
                **split,
            }
        )
    out_df = split_dataframe(out)

    tmp = Path(f"{part}.{os.getpid()}.tmp")
    tmp.parent.mkdir(parents=True, exist_ok=True)
    try:
        out_df.to_parquet(tmp, index=False)
        out_rows, out_instances, out_trajectories = con.execute(
            f"SELECT count(*), count(DISTINCT instance_id), sum(n_rollouts) "
            f"FROM read_parquet({_sql_str(tmp)})"
        ).fetchone()
        if out_rows != out_instances or out_rows == 0 or out_trajectories != src_rows:
            raise RuntimeError(
                f"row guard failed: source={src_rows} out={out_rows} "
                f"instances={out_instances} trajectories={out_trajectories}"
            )
    except Exception:
        tmp.unlink(missing_ok=True)
        raise
    os.replace(tmp, part)
    return "done", int(out_rows)


def merge_parts(
    con: duckdb.DuckDBPyConnection,
    parts_dir: Path = PARTS_DIR,
    out_path: Path = OUT_PARQUET,
) -> tuple[int, int, int] | None:
    parts = sorted(parts_dir.glob("*.parquet"))
    if not parts:
        return None
    glob_sql = _sql_str(parts_dir / "*.parquet")
    tmp = Path(str(out_path) + ".tmp")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    agg = ", ".join(f"max({col})::BIGINT AS {col}" for col in SPLIT_COLUMNS)
    con.execute(
        f"""
        COPY (
            SELECT
                instance_id,
                max(repo) AS repo,
                max(language) AS language,
                max(hf_dataset_name) AS hf_dataset_name,
                sum(n_rollouts)::INTEGER AS n_rollouts,
                sum(n_labeled)::INTEGER AS n_labeled,
                sum(n_resolved)::INTEGER AS n_resolved,
                max(has_gold_patch)::INTEGER AS has_gold_patch,
                {agg}
            FROM read_parquet({glob_sql})
            GROUP BY 1
            ORDER BY 1
        ) TO {_sql_str(tmp)} (FORMAT PARQUET)
        """
    )
    rows, instances = con.execute(
        f"SELECT count(*), count(DISTINCT instance_id) FROM read_parquet({_sql_str(tmp)})"
    ).fetchone()
    os.replace(tmp, out_path)
    return len(parts), int(rows), int(instances)


def stream_split(args: argparse.Namespace) -> int:
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
            label = shard_label(file_path)
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
                f"instances={rows:,} in {elapsed:.1f}s (avg {rate:.1f}s, eta {fmt_duration(eta)})"
            )

        if failures:
            console.print(
                f"[{utcnow()}] {failed} shard(s) failed — not merging partial output; "
                "rerun to resume (finished parts are kept)"
            )
        else:
            merged = merge_parts(con, out_path=args.output)
            if merged is None:
                console.print(f"[{utcnow()}] no parts to merge")
            else:
                n_parts, rows, _ = merged
                partial = bool(args.file or args.limit)
                suffix = (
                    " (partial run: rerun without --file/--limit to cover the corpus)"
                    if partial
                    else ""
                )
                console.print(
                    f"[{utcnow()}] merged {n_parts} parts → {rel(args.output)}: "
                    f"{rows:,} instances{suffix}"
                )
    except KeyboardInterrupt:
        console.print(f"[{utcnow()}] interrupted — parts written so far are kept, rerun to resume")
        return 130
    finally:
        con.close()

    console.print(
        f"[{utcnow()}] patch-split pass finished: processed={done} skipped={skipped} failed={failed} "
        f"in {fmt_duration(time.monotonic() - started)}"
    )
    if failures:
        console.print(f"[red]failed shards[/red]: {', '.join(failures)}")
        return 1
    return 0


def load_split(path: Path = OUT_PARQUET) -> pd.DataFrame:
    con = connect_ephemeral()
    try:
        inst = con.execute(f"SELECT * FROM read_parquet({_sql_str(path)})").df()
    finally:
        con.close()
    inst["language"] = inst["language"].astype("string").str.lower().replace(LANGUAGE_ALIASES)
    inst["solve_rate"] = (
        inst["n_resolved"] / inst["n_labeled"].where(inst["n_labeled"] > 0)
    ).astype("float64")
    return inst


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Stream gold patches once and split them by file class."
    )
    parser.add_argument("--data-glob", default=PARQUET_GLOB)
    parser.add_argument("--file", nargs="*", type=Path)
    parser.add_argument("--limit", type=int)
    parser.add_argument("--force", action="store_true")
    parser.add_argument("--skip-stream", action="store_true")
    parser.add_argument("--memory-limit", default="4GB")
    parser.add_argument("--threads", type=int, default=2)
    parser.add_argument("--output", type=Path, default=OUT_PARQUET)
    args = parser.parse_args()
    if args.skip_stream:
        con = connect_ephemeral()
        merged = merge_parts(con, out_path=args.output)
        con.close()
        if merged:
            console.print(f"merged {merged[0]} parts → {rel(args.output)}: {merged[1]:,} instances")
        return
    sys.exit(stream_split(args))


if __name__ == "__main__":
    main()

"""Data catalog: sync derived parquets and lint the catalog against reality.

Two responsibilities (thin CLIs: ``scripts/sync_derived.py``,
``scripts/catalog_check.py``):

- ``sync_derived`` copies (never symlinks) the small derived parquets produced by the
  sibling closure worktrees into ``outputs/derived/`` and rewrites ``SOURCES.md`` with
  provenance (source path, md5, row count). Idempotent; missing sources are skipped
  with a warning (e.g. oswt-closureE harbor_hub may not exist yet).
- ``check_catalog`` lints ``analytics/schema/DATA_CATALOG.md`` against the registry
  table in that file. It fails when:

  * a parquet under ``outputs/derived/`` is not registered in the catalog,
  * a registered parquet is missing on disk,
  * a view/table/macro defined in ``analytics/schema/*.sql`` (or the ``_config``
    runtime table created by ``openswe_traces.data.init_db``) is not registered,
  * a registered DuckDB object, parquet, or sqlite table lacks a ``### <name>``
    section in the catalog,
  * a table in ``experiments/pipeline/state.db`` is not registered, or a registered
    sqlite table does not exist there.
"""

from __future__ import annotations

import argparse
import hashlib
import re
import shutil
import sqlite3
from datetime import UTC, datetime
from pathlib import Path

from rich.console import Console

ROOT = Path(__file__).resolve().parents[2]
CATALOG_PATH = ROOT / "analytics" / "schema" / "DATA_CATALOG.md"
SCHEMA_DIR = ROOT / "analytics" / "schema"
DERIVED_DIR = ROOT / "outputs" / "derived"
STATE_DB = ROOT / "experiments" / "pipeline" / "state.db"
SOURCES_PATH = DERIVED_DIR / "SOURCES.md"

console = Console()

# Registry types recognized in DATA_CATALOG.md and what each maps to on disk.
REGISTRY_TYPES = (
    "view",
    "table",
    "derived_view",
    "macro",
    "runtime_table",
    "parquet",
    "sqlite_table",
    "external",
)

# (file under outputs/derived/, source worktree, source path relative to the worktree).
DERIVED_SOURCES: dict[str, tuple[str, str]] = {
    "closure_proxies.parquet": ("oswt-closureB", "outputs/closure_proxies.parquet"),
    "rung_features.parquet": ("oswt-closureC", "outputs/rung_features.parquet"),
    "trajectory_frame.parquet": ("oswt-closureC", "outputs/trajectory_frame.parquet"),
    "closure_metrics.parquet": ("oswt-closureA", "outputs/closure_metrics.parquet"),
}
HARBOR_SOURCE = ("oswt-closureE", "traces_external/harbor_hub")


def siblings_dir() -> Path:
    """Directory holding the sibling worktrees; overridable via OSWT_SIBLINGS_DIR."""
    import os

    return Path(os.environ.get("OSWT_SIBLINGS_DIR", ROOT.parent))


# -------------------------------------------------------------------------------------
# sync_derived
# -------------------------------------------------------------------------------------

def _md5(path: Path) -> str:
    h = hashlib.md5()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def _parquet_rows(path: Path) -> int:
    """Row count from parquet metadata only (no data scan)."""
    import pyarrow.parquet as pq

    meta = pq.read_metadata(path)
    return sum(meta.row_group(i).num_rows for i in range(meta.num_row_groups))


def sync_derived() -> list[Path]:
    """Copy derived parquets from sibling worktrees into outputs/derived/.

    Returns the list of files copied (or re-verified). Missing sources are skipped
    with a warning, so the harbor_hub copy handles its own absence.
    """
    base = siblings_dir()
    DERIVED_DIR.mkdir(parents=True, exist_ok=True)
    copied: list[Path] = []

    def _copy(src: Path, dst: Path) -> bool:
        if not src.is_file():
            console.print(f"[yellow]skip[/yellow] missing source {src}")
            return False
        dst.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(src, dst)
        copied.append(dst)
        console.print(f"[green]copied[/green] {src} -> {dst}")
        return True

    for rel, (worktree, src_rel) in DERIVED_SOURCES.items():
        _copy(base / worktree / src_rel, DERIVED_DIR / rel)

    harbor = base / HARBOR_SOURCE[0] / HARBOR_SOURCE[1]
    if harbor.is_dir():
        for src in sorted(harbor.glob("*.parquet")):
            _copy(src, DERIVED_DIR / "harbor_hub" / src.name)
    else:
        console.print(f"[yellow]skip[/yellow] harbor_hub source missing: {harbor}")

    if copied:
        _write_sources(copied)
    else:
        console.print("[red]nothing copied; SOURCES.md not rewritten[/red]")
    return copied


def _write_sources(copied: list[Path]) -> None:
    """Rewrite SOURCES.md with provenance for every copied parquet."""
    rows: list[str] = []
    for dst in sorted(copied):
        rel = dst.relative_to(DERIVED_DIR)
        if rel.parts[0] == "harbor_hub":
            worktree, src_dir = HARBOR_SOURCE
            src = siblings_dir() / worktree / src_dir / rel.parts[1]
        else:
            src = siblings_dir() / DERIVED_SOURCES[rel.name][0] / DERIVED_SOURCES[rel.name][1]
        rows.append(
            f"| `{rel}` | {src.relative_to(siblings_dir())} | "
            f"{_md5(dst)} | {_parquet_rows(dst):,} | see DATA_CATALOG.md |"
        )

    stamp = datetime.now(UTC).strftime("%Y-%m-%d %H:%M UTC")
    SOURCES_PATH.write_text(
        f"""# outputs/derived — copied artifacts and provenance

Rewritten by `uv run python scripts/sync_derived.py` on {stamp} from sibling closure
worktrees (read-only for this worktree). Full per-column documentation, caveats, and
reproduce commands: `analytics/schema/DATA_CATALOG.md` (Part 5).

| file | source path (worktree) | md5 | rows | producing script |
|---|---|---|---|---|
{chr(10).join(rows)}
Notes

- `closure_proxies.parquet` also exists byte-identical at oswt-closureC/outputs/.
- No IRT parquet outputs exist in oswt-closureC/outputs (only the research note
  `analytics/research/irt_summary.md`), so none are copied.
- Intermediate per-shard parts (`closure_proxies_parts/`, `rung_features_parts/`,
  `trajectory_frame_parts/`) are not copied; they are resume caches regenerated by
  the producing scripts.
- The large TB2 trajectory dumps under
  `/home/evan/Documents/open_swe_traces_research/traces_external/` (yoonholee
  `train-*.parquet` ~212 MB, harithoppil `*.jsonl` ~70 MB) are not copied; they are
  documented in DATA_CATALOG.md and read in place from the main checkout.
"""
    )
    console.print(f"[green]wrote[/green] {SOURCES_PATH}")


def sync_derived_main() -> None:
    parser = argparse.ArgumentParser(
        description="Copy derived parquets from sibling closure worktrees into outputs/derived/."
    )
    parser.parse_args()
    files = sync_derived()
    console.print(f"Derived parquets: {len(files)}")


# -------------------------------------------------------------------------------------
# check_catalog (lint)
# -------------------------------------------------------------------------------------

_REGISTRY_RE = re.compile(r"^\|\s*(" + "|".join(REGISTRY_TYPES) + r")\s*\|\s*([^|]+?)\s*\|$")
_SQL_OBJECT_RE = re.compile(r"CREATE\s+OR\s+REPLACE\s+(VIEW|TABLE|MACRO)\s+([A-Za-z_][A-Za-z0-9_]*)")


def parse_registry(text: str) -> dict[str, str]:
    """Parse the registry table in DATA_CATALOG.md -> {name: type}.

    Only the lines between the ``## Registry`` header and the next ``## `` header
    count, so other markdown tables in the catalog are not mistaken for registry
    entries.
    """
    lines = text.splitlines()
    start = next((i for i, ln in enumerate(lines) if ln.strip() == "## Registry"), None)
    if start is None:
        return {}
    end = next(
        (i for i in range(start + 1, len(lines)) if lines[i].startswith("## ")),
        len(lines),
    )
    registry: dict[str, str] = {}
    for line in lines[start:end]:
        m = _REGISTRY_RE.match(line.strip())
        if m:
            registry[m.group(2)] = m.group(1)
    return registry


def parse_schema_objects() -> dict[str, str]:
    """{name: kind} for every CREATE OR REPLACE VIEW/TABLE/MACRO in analytics/schema/*.sql."""
    objects: dict[str, str] = {}
    for sql_file in sorted(SCHEMA_DIR.glob("*.sql")):
        for m in _SQL_OBJECT_RE.finditer(sql_file.read_text(encoding="utf-8")):
            objects[m.group(2)] = m.group(1).lower()
    return objects


def state_db_tables() -> list[str]:
    """Table names in experiments/pipeline/state.db ([] when the db is absent)."""
    if not STATE_DB.is_file():
        return []
    con = sqlite3.connect(f"file:{STATE_DB}?mode=ro", uri=True)
    try:
        return sorted(
            row[0]
            for row in con.execute("SELECT name FROM sqlite_master WHERE type = 'table'")
        )
    finally:
        con.close()


def check_catalog() -> list[str]:
    """Return a list of catalog violations (empty = catalog is consistent)."""
    errors: list[str] = []
    if not CATALOG_PATH.is_file():
        return [f"missing catalog: {CATALOG_PATH}"]

    text = CATALOG_PATH.read_text(encoding="utf-8")
    registry = parse_registry(text)

    def section(name: str) -> bool:
        return any(line.strip() == f"### {name}" for line in text.splitlines())

    # --- parquets: disk <-> registry, both directions ------------------------------
    on_disk = sorted(p.relative_to(DERIVED_DIR).as_posix() for p in DERIVED_DIR.glob("**/*.parquet"))
    registered_parquets = {
        name.removeprefix("outputs/derived/") for name, typ in registry.items() if typ == "parquet"
    }
    for rel in on_disk:
        key = f"outputs/derived/{rel}"
        if rel not in registered_parquets:
            errors.append(f"parquet not documented in DATA_CATALOG.md registry: {key}")
    for rel in sorted(registered_parquets):
        if rel not in on_disk:
            errors.append(f"registered parquet missing on disk: outputs/derived/{rel}")
        if not section(f"outputs/derived/{rel}"):
            errors.append(f"registered parquet has no '### outputs/derived/{rel}' section")

    # --- schema objects: sql <-> registry, both directions -------------------------
    sql_objects = parse_schema_objects()
    for name in sorted(sql_objects):
        if name not in registry:
            errors.append(f"schema object not documented in DATA_CATALOG.md registry: {name}")
    for name, typ in sorted(registry.items()):
        if typ in {"view", "table", "derived_view", "macro"} and name not in sql_objects:
            errors.append(f"registered {typ} '{name}' is not defined in analytics/schema/*.sql")
        if typ in {"view", "table", "derived_view", "macro", "runtime_table"} and not section(name):
            errors.append(f"registered {typ} '{name}' has no '### {name}' section")

    # --- state.db tables: db <-> registry, both directions -------------------------
    db_tables = state_db_tables()
    registered_sqlite = {name for name, typ in registry.items() if typ == "sqlite_table"}
    for table in db_tables:
        if table not in registered_sqlite:
            errors.append(f"state.db table not documented in DATA_CATALOG.md registry: {table}")
    for table in sorted(registered_sqlite):
        if table not in db_tables:
            errors.append(f"registered sqlite_table '{table}' missing from experiments/pipeline/state.db")
        if not section(table):
            errors.append(f"registered sqlite_table '{table}' has no '### {table}' section")

    # --- external dumps: every registered external must have a section ------------
    for name, typ in sorted(registry.items()):
        if typ == "external" and not section(name):
            errors.append(f"registered external dump '{name}' has no '### {name}' section")

    return errors


def catalog_check_main() -> int:
    parser = argparse.ArgumentParser(
        description="Lint DATA_CATALOG.md against outputs/derived/, analytics/schema/*.sql, and state.db."
    )
    parser.parse_args()
    errors = check_catalog()
    if errors:
        for err in errors:
            console.print(f"[red]FAIL[/red] {err}")
        console.print(f"\n[red]{len(errors)} catalog violation(s)[/red]")
        return 1
    console.print("[green]catalog OK[/green] — every parquet, view, and table documented")
    return 0

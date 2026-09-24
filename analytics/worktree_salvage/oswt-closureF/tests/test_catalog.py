"""Catalog lint: DATA_CATALOG.md must cover outputs/derived/, schema SQL, and state.db."""

from pathlib import Path

from openswe_traces.catalog import (
    CATALOG_PATH,
    DERIVED_DIR,
    check_catalog,
    parse_registry,
    parse_schema_objects,
    state_db_tables,
)


def test_catalog_lint_passes() -> None:
    errors = check_catalog()
    assert not errors, "catalog violations:\n" + "\n".join(errors)


def test_registry_parses_real_catalog() -> None:
    text = CATALOG_PATH.read_text(encoding="utf-8")
    registry = parse_registry(text)
    # Every type we lint over must be present at least once.
    for typ in ("view", "table", "derived_view", "parquet", "sqlite_table", "external"):
        assert typ in registry.values(), f"registry has no {typ} entries"


def test_schema_objects_parse() -> None:
    objects = parse_schema_objects()
    for name in ("traces_raw", "traces", "trace_summary_by_language", "osw_instance_rung"):
        assert name in objects, f"missing schema object {name}"


def test_derived_dir_matches_registry() -> None:
    """Every parquet on disk is registered (registry side checked by the lint)."""
    from openswe_traces.catalog import parse_registry

    text = CATALOG_PATH.read_text(encoding="utf-8")
    registered = {
        name.removeprefix("outputs/derived/")
        for name, typ in parse_registry(text).items()
        if typ == "parquet"
    }
    on_disk = {
        p.relative_to(DERIVED_DIR).as_posix() for p in DERIVED_DIR.glob("**/*.parquet")
    }
    missing = on_disk - registered
    assert not missing, f"parquets on disk but not registered: {sorted(missing)}"


def test_state_db_tables_readable() -> None:
    tables = state_db_tables()
    if not tables:
        return  # state.db absent (e.g. fresh clone) — skip silently
    for name in ("units", "trials", "steps", "meta", "repos", "events", "tokens"):
        assert name in tables, f"state.db missing table {name}"


def test_catalog_file_is_tracked() -> None:
    assert CATALOG_PATH.is_file()
    assert Path(DERIVED_DIR).is_dir()

"""SQLite state machine. Every stage is idempotent; rerunning resumes unfinished work."""

from __future__ import annotations

import json
import sqlite3
from collections.abc import Iterable, Mapping
from contextlib import contextmanager
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

PENDING = "pending"
RUNNING = "running"
DONE = "done"
FAILED = "failed"
SKIPPED = "skipped"
REJECTED = "rejected"
PAUSED = "paused"
TERMINAL = frozenset({DONE, SKIPPED, REJECTED})

STAGES = ("prepare", "author", "verifier", "package", "solve", "aggregate")


def _now() -> str:
    return datetime.now(UTC).replace(microsecond=0).isoformat()


def _json(value: Any) -> str:
    return json.dumps(value, default=str)


@dataclass(frozen=True)
class Step:
    repo: str
    unit: str
    stage: str
    status: str
    payload: dict[str, Any]
    error: str
    updated_at: str


@dataclass(frozen=True)
class TrialRow:
    id: int
    repo: str
    unit: str
    level: int
    solver: str
    attempt: int
    reward: float | None
    tokens_in: int
    tokens_out: int
    audit_class: str
    wall_minutes: float | None
    job_dir: str
    excluded: bool
    timeout: bool = False


class PipelineStore:
    def __init__(self, path: Path | str) -> None:
        self.path = Path(path)
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self._con = sqlite3.connect(str(self.path))
        self._con.row_factory = sqlite3.Row
        self._con.execute("PRAGMA journal_mode=WAL")
        self._con.execute("PRAGMA foreign_keys=ON")
        self._init()
        self._migrate()

    def close(self) -> None:
        self._con.close()

    def __enter__(self):
        return self

    def __exit__(self, *exc: object) -> None:
        self.close()

    def _ensure_column(self, table: str, name: str, decl: str) -> None:
        cols = {row[1] for row in self._con.execute(f"PRAGMA table_info({table})")}
        if name not in cols:
            self._con.execute(f"ALTER TABLE {table} ADD COLUMN {name} {decl}")

    def _migrate(self) -> None:
        self._ensure_column("units", "predicted_flip", "INTEGER")
        self._ensure_column("units", "is_control", "INTEGER DEFAULT 0")
        self._ensure_column("units", "author_backend", "TEXT")
        self._ensure_column("trials", "timeout", "INTEGER DEFAULT 0")
        self._con.commit()

    def _init(self) -> None:
        self._con.executescript(
            """
            CREATE TABLE IF NOT EXISTS meta (
              key TEXT PRIMARY KEY,
              value TEXT
            );
            CREATE TABLE IF NOT EXISTS repos (
              name TEXT PRIMARY KEY,
              status TEXT NOT NULL,
              error TEXT,
              updated_at TEXT
            );
            CREATE TABLE IF NOT EXISTS units (
              repo TEXT NOT NULL,
              unit TEXT NOT NULL,
              status TEXT NOT NULL,
              family TEXT,
              closure_json TEXT,
              n_files INTEGER,
              n_lines INTEGER,
              rejected_rule TEXT,
              updated_at TEXT,
              PRIMARY KEY (repo, unit)
            );
            CREATE TABLE IF NOT EXISTS steps (
              repo TEXT NOT NULL,
              unit TEXT NOT NULL DEFAULT '',
              stage TEXT NOT NULL,
              status TEXT NOT NULL,
              payload_json TEXT,
              error TEXT,
              updated_at TEXT,
              PRIMARY KEY (repo, unit, stage)
            );
            CREATE TABLE IF NOT EXISTS trials (
              id INTEGER PRIMARY KEY AUTOINCREMENT,
              repo TEXT,
              unit TEXT,
              level INTEGER,
              solver TEXT,
              attempt INTEGER,
              reward REAL,
              tokens_in INTEGER,
              tokens_out INTEGER,
              audit_class TEXT,
              wall_minutes REAL,
              job_dir TEXT,
              excluded INTEGER DEFAULT 0
            );
            CREATE TABLE IF NOT EXISTS events (
              id INTEGER PRIMARY KEY AUTOINCREMENT,
              ts TEXT,
              kind TEXT,
              message TEXT
            );
            CREATE TABLE IF NOT EXISTS tokens (
              id INTEGER PRIMARY KEY AUTOINCREMENT,
              source TEXT,
              tokens_in INTEGER,
              tokens_out INTEGER,
              ts TEXT
            );
            """
        )
        self._con.commit()

    def get_meta(self, key: str, default: str | None = None) -> str | None:
        row = self._con.execute("SELECT value FROM meta WHERE key = ?", (key,)).fetchone()
        return row["value"] if row else default

    def set_meta(self, key: str, value: str) -> None:
        self._con.execute(
            "INSERT INTO meta(key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
            (key, value),
        )
        self._con.commit()

    def upsert_repo(self, name: str, status: str = PENDING, error: str = "") -> None:
        self._con.execute(
            """
            INSERT INTO repos(name, status, error, updated_at)
            VALUES (?, ?, ?, ?)
            ON CONFLICT(name) DO UPDATE SET
              status = excluded.status,
              error = excluded.error,
              updated_at = excluded.updated_at
            """,
            (name, status, error, _now()),
        )
        self._con.commit()

    def repo_status(self, name: str) -> str:
        row = self._con.execute("SELECT status FROM repos WHERE name = ?", (name,)).fetchone()
        return row["status"] if row else PENDING

    def upsert_unit(
        self,
        repo: str,
        unit: str,
        *,
        status: str = PENDING,
        family: str = "",
        closure: Mapping[str, Any] | None = None,
        n_files: int | None = None,
        n_lines: int | None = None,
        rejected_rule: str = "",
        predicted_flip: int | None = None,
        is_control: bool | None = None,
        author_backend: str | None = None,
    ) -> None:
        self._con.execute(
            """
            INSERT INTO units(
              repo, unit, status, family, closure_json, n_files, n_lines, rejected_rule,
              predicted_flip, is_control, author_backend, updated_at
            )
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(repo, unit) DO UPDATE SET
              status = excluded.status,
              family = COALESCE(NULLIF(excluded.family, ''), units.family),
              closure_json = COALESCE(excluded.closure_json, units.closure_json),
              n_files = COALESCE(excluded.n_files, units.n_files),
              n_lines = COALESCE(excluded.n_lines, units.n_lines),
              rejected_rule = COALESCE(NULLIF(excluded.rejected_rule, ''), units.rejected_rule),
              predicted_flip = COALESCE(excluded.predicted_flip, units.predicted_flip),
              is_control = COALESCE(excluded.is_control, units.is_control),
              author_backend = COALESCE(NULLIF(excluded.author_backend, ''), units.author_backend),
              updated_at = excluded.updated_at
            """,
            (
                repo,
                unit,
                status,
                family,
                _json(closure) if closure is not None else None,
                n_files,
                n_lines,
                rejected_rule,
                predicted_flip,
                None if is_control is None else int(is_control),
                author_backend,
                _now(),
            ),
        )
        self._con.commit()

    def list_units(self, repo: str | None = None) -> list[sqlite3.Row]:
        if repo:
            return list(self._con.execute("SELECT * FROM units WHERE repo = ? ORDER BY unit", (repo,)))
        return list(self._con.execute("SELECT * FROM units ORDER BY repo, unit"))

    def step(self, repo: str, stage: str, unit: str = "") -> Step | None:
        row = self._con.execute(
            "SELECT * FROM steps WHERE repo = ? AND unit = ? AND stage = ?",
            (repo, unit, stage),
        ).fetchone()
        if row is None:
            return None
        payload = {}
        if row["payload_json"]:
            try:
                payload = json.loads(row["payload_json"])
            except json.JSONDecodeError:
                payload = {}
        return Step(
            repo=row["repo"],
            unit=row["unit"],
            stage=row["stage"],
            status=row["status"],
            payload=payload if isinstance(payload, dict) else {},
            error=row["error"] or "",
            updated_at=row["updated_at"] or "",
        )

    def step_done(self, repo: str, stage: str, unit: str = "") -> bool:
        current = self.step(repo, stage, unit)
        return current is not None and current.status in TERMINAL

    def mark_step(
        self,
        repo: str,
        stage: str,
        status: str,
        *,
        unit: str = "",
        payload: Mapping[str, Any] | None = None,
        error: str = "",
    ) -> None:
        self._con.execute(
            """
            INSERT INTO steps(repo, unit, stage, status, payload_json, error, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(repo, unit, stage) DO UPDATE SET
              status = excluded.status,
              payload_json = COALESCE(excluded.payload_json, steps.payload_json),
              error = excluded.error,
              updated_at = excluded.updated_at
            """,
            (repo, unit, stage, status, _json(payload) if payload is not None else None, error, _now()),
        )
        self._con.commit()

    def should_run(self, repo: str, stage: str, unit: str = "") -> bool:
        """True unless the step already finished successfully or was skipped/rejected."""
        current = self.step(repo, stage, unit)
        if current is None:
            return True
        return current.status not in TERMINAL

    @contextmanager
    def running(self, repo: str, stage: str, unit: str = ""):
        self.mark_step(repo, stage, RUNNING, unit=unit)
        try:
            yield
        except Exception as exc:
            self.mark_step(repo, stage, FAILED, unit=unit, error=str(exc)[:2000])
            raise

    def add_event(self, kind: str, message: str) -> None:
        self._con.execute(
            "INSERT INTO events(ts, kind, message) VALUES (?, ?, ?)",
            (_now(), kind, message),
        )
        self._con.commit()

    def events(self, *, kind: str | None = None) -> list[sqlite3.Row]:
        if kind:
            return list(self._con.execute("SELECT * FROM events WHERE kind = ? ORDER BY id", (kind,)))
        return list(self._con.execute("SELECT * FROM events ORDER BY id"))

    def add_tokens(self, source: str, tokens_in: int, tokens_out: int) -> None:
        self._con.execute(
            "INSERT INTO tokens(source, tokens_in, tokens_out, ts) VALUES (?, ?, ?, ?)",
            (source, int(tokens_in), int(tokens_out), _now()),
        )
        self._con.commit()

    def token_sum(self, *, sources: Iterable[str] | None = None) -> int:
        if sources is None:
            row = self._con.execute(
                "SELECT COALESCE(SUM(tokens_in + tokens_out), 0) AS n FROM tokens"
            ).fetchone()
            return int(row["n"])
        wanted = tuple(sources)
        if not wanted:
            return 0
        q = ",".join("?" * len(wanted))
        row = self._con.execute(
            f"SELECT COALESCE(SUM(tokens_in + tokens_out), 0) AS n FROM tokens WHERE source IN ({q})",
            wanted,
        ).fetchone()
        return int(row["n"])

    def composer_tokens(self) -> int:
        return self.token_sum(sources=("cursor-cli", "cursor-agent", "cursor"))

    def add_trial(
        self,
        *,
        repo: str,
        unit: str,
        level: int,
        solver: str,
        attempt: int,
        reward: float | None,
        tokens_in: int = 0,
        tokens_out: int = 0,
        audit_class: str = "",
        wall_minutes: float | None = None,
        job_dir: str = "",
        excluded: bool = False,
        timeout: bool = False,
    ) -> int:
        cur = self._con.execute(
            """
            INSERT INTO trials(
              repo, unit, level, solver, attempt, reward, tokens_in, tokens_out,
              audit_class, wall_minutes, job_dir, excluded, timeout
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (
                repo,
                unit,
                int(level),
                solver,
                int(attempt),
                reward,
                int(tokens_in),
                int(tokens_out),
                audit_class,
                wall_minutes,
                job_dir,
                int(excluded),
                int(timeout),
            ),
        )
        self._con.commit()
        return int(cur.lastrowid)

    def list_trials(
        self,
        *,
        repo: str | None = None,
        unit: str | None = None,
        include_excluded: bool = True,
    ) -> list[TrialRow]:
        sql = "SELECT * FROM trials WHERE 1=1"
        args: list[Any] = []
        if repo:
            sql += " AND repo = ?"
            args.append(repo)
        if unit:
            sql += " AND unit = ?"
            args.append(unit)
        if not include_excluded:
            sql += " AND excluded = 0"
        sql += " ORDER BY id"
        rows = []
        for row in self._con.execute(sql, args):
            rows.append(
                TrialRow(
                    id=row["id"],
                    repo=row["repo"],
                    unit=row["unit"],
                    level=row["level"],
                    solver=row["solver"],
                    attempt=row["attempt"],
                    reward=row["reward"],
                    tokens_in=row["tokens_in"] or 0,
                    tokens_out=row["tokens_out"] or 0,
                    audit_class=row["audit_class"] or "",
                    wall_minutes=row["wall_minutes"],
                    job_dir=row["job_dir"] or "",
                    excluded=bool(row["excluded"]),
                    timeout=bool(row["timeout"]) if "timeout" in row else False,
                )
            )
        return rows

    def level_counts(self, repo: str, unit: str, level: int, solver: str) -> tuple[int, int]:
        """(passes, scored attempts) excluding contaminated/hacked/timeouts."""
        rows = self._con.execute(
            """
            SELECT reward FROM trials
            WHERE repo = ? AND unit = ? AND level = ? AND solver = ? AND excluded = 0
              AND COALESCE(timeout, 0) = 0
              AND audit_class NOT IN ('contaminated', 'checksum', 'hacked', 'd')
            """,
            (repo, unit, level, solver),
        ).fetchall()
        n = len(rows)
        p = sum(1 for r in rows if r["reward"] == 1.0)
        return p, n

    def status_rows(self) -> list[dict[str, Any]]:
        out: list[dict[str, Any]] = []
        for repo in self._con.execute("SELECT * FROM repos ORDER BY name"):
            steps = list(
                self._con.execute(
                    "SELECT stage, unit, status, error FROM steps WHERE repo = ? ORDER BY stage, unit",
                    (repo["name"],),
                )
            )
            out.append(
                {
                    "repo": repo["name"],
                    "status": repo["status"],
                    "error": repo["error"] or "",
                    "steps": [dict(s) for s in steps],
                    "units": [dict(u) for u in self.list_units(repo["name"])],
                }
            )
        return out

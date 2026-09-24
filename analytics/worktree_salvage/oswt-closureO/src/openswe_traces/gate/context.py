"""Per-task context handed to every rule implementation.

Wraps the documentary ``synth.rules.RuleContext`` (markdown notes, recorded
payloads) with the packaged artifacts the static/executed impls need: the
excised symbol set, sibling level dirs, and a docker runner. All filesystem
reads are lazy so constructing a context is cheap.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from openswe_traces.gate.excision import ExcisedInfo, excised_info

_LEVEL_RE = re.compile(r"^(?P<unit>.+?)-L(?P<level>\d+)$")


@dataclass
class GateContext:
    task_dir: Path
    extra: dict[str, Any] = field(default_factory=dict)
    docker_run: Any = None  # pipeline_ext.hack_audit.DockerRun-compatible
    force: bool = False  # ignore cached executed blocks
    image: str | None = None  # override task_image() resolution
    language: str | None = None  # override detect_language()
    _rules_ctx: Any = field(default=None, repr=False)
    _excised: ExcisedInfo | None = field(default=None, repr=False)
    _runs: Any = field(default=None, repr=False)  # memoized exec_rules.GateRuns

    @property
    def rules(self) -> Any:
        """The documentary RuleContext (synth.rules.load_context)."""
        if self._rules_ctx is None:
            from openswe_traces.synth.rules import load_context

            self._rules_ctx = load_context(self.task_dir, extra=self.extra)
        return self._rules_ctx

    @property
    def excised(self) -> ExcisedInfo:
        if self._excised is None:
            self._excised = excised_info(self.task_dir, rules_ctx=self._rules_ctx)
        return self._excised

    @property
    def unit(self) -> str:
        m = _LEVEL_RE.match(self.task_dir.name)
        return m.group("unit") if m else self.task_dir.name

    @property
    def level(self) -> int | None:
        m = _LEVEL_RE.match(self.task_dir.name)
        return int(m.group("level")) if m else None


def level_dirs(task_dir: Path | str) -> list[tuple[int, Path]]:
    """Sibling ``<unit>-L<n>`` dirs of the same unit, sorted by level."""
    td = Path(task_dir)
    m = _LEVEL_RE.match(td.name)
    if not m:
        return []
    out: list[tuple[int, Path]] = []
    for sib in sorted(td.parent.iterdir()):
        if not sib.is_dir():
            continue
        sm = _LEVEL_RE.match(sib.name)
        if sm and sm.group("unit") == m.group("unit"):
            out.append((int(sm.group("level")), sib))
    return sorted(out)

#!/usr/bin/env python3
"""Move a scripts/ops module into the package and leave a shim at the old path.

Every scripts/ops/*.py used to be both a library (other scripts `import trial_ledger` after
putting scripts/ops on sys.path) and a command (`uv run python scripts/ops/trial_guard.py`).
Running shell loops, frozen copies and cron lines call those paths, so a move must keep
both working. For each module named on the command line this:

  1. git-mvs scripts/ops/<old>.py to src/openswe_traces/<new path>.py
  2. rewrites, in every module already in MOVES, bare imports of moved modules into
     package imports under the SAME local name, so no function body changes
  3. drops the sys.path.insert lines that only existed to make bare imports resolve
  4. turns the `if __name__ == "__main__":` block into `def cli():`
  5. writes a shim at the old path that registers the package module under the old name
     (one module object, so private names and mutable state are shared) and runs cli()

Usage: move_module.py <old-stem> [<old-stem> ...]   (run from the repo root)
"""
import pathlib
import re
import subprocess
import sys

# old scripts/ops stem -> new dotted module. Grows one wave at a time.
MOVES = {
    "trial_ledger": "openswe_traces.ladder.ledger",
    "trial_guard": "openswe_traces.ladder.guard",
    "solver_match": "openswe_traces.ladder.match",
    "escalate": "openswe_traces.ladder.escalate",
    "slots": "openswe_traces.ops.slots",
    "roots": "openswe_traces.ops.roots",
    # wave 2: ladder bookkeeping
    "stability_gate": "openswe_traces.ladder.stability_gate",
    "second_screen": "openswe_traces.ladder.second_screen",
    "ladder_purity": "openswe_traces.ladder.purity",
    "quarantine_trial": "openswe_traces.ladder.quarantine",
    "backfill_history": "openswe_traces.ladder.backfill_history",
    "unit_features": "openswe_traces.ladder.unit_features",
    # wave 2: running agents, capacity and budget
    "agents": "openswe_traces.ops.agents",
    "agent_session": "openswe_traces.ops.agent_session",
    "ask_composer": "openswe_traces.ops.ask_composer",
    "cohorts": "openswe_traces.ops.cohorts",
    "composer_budget": "openswe_traces.ops.composer_budget",
    "budget_forecast": "openswe_traces.ops.budget_forecast",
    "devin_cap": "openswe_traces.ops.devin_cap",
    "devin_ratelimit_check": "openswe_traces.ops.devin_ratelimit_check",
    "restore_env_src": "openswe_traces.ops.restore_env_src",
    "reclaim_disk": "openswe_traces.ops.reclaim_disk",
    "docker_gc": "openswe_traces.ops.docker_gc",
    "worktree_gc": "openswe_traces.ops.worktree_gc",
    "reap_orphans": "openswe_traces.ops.reap_orphans",
    "reap_runaway": "openswe_traces.ops.reap_runaway",
    "grok_build_patch": "openswe_traces.ops.grok_build_patch",
    # wave 3: reports read by people
    "status": "openswe_traces.reports.status",
    "ladder_matrix": "openswe_traces.reports.ladder_matrix",
    "build_dashboard": "openswe_traces.reports.dashboard",
    "jobs_status": "openswe_traces.reports.jobs_status",
    "rolling": "openswe_traces.reports.rolling",
    "token_cost": "openswe_traces.reports.token_cost",
    "devin_usage": "openswe_traces.reports.devin_usage",
    "agent_ledger": "openswe_traces.reports.agent_ledger",
    # wave 3: one-off instruments cited in analytics/research notes
    "synthetic_provenance": "openswe_traces.analysis.synthetic_provenance",
    "task_overlap": "openswe_traces.analysis.task_overlap",
    "contract_vs_test": "openswe_traces.analysis.contract_vs_test",
    "literal_digest_audit": "openswe_traces.analysis.literal_digest_audit",
    "reconcile_inferable": "openswe_traces.analysis.reconcile_inferable",
    "arbitrary_failures": "openswe_traces.analysis.arbitrary_failures",
    "classify_gaps": "openswe_traces.analysis.classify_gaps",
    "assertion_density": "openswe_traces.analysis.assertion_density",
    "cheat_validity": "openswe_traces.analysis.cheat_validity",
    "failure_shape": "openswe_traces.analysis.failure_shape",
    "grok_harvest": "openswe_traces.analysis.grok_harvest",
    # wave 4: task authoring
    "pipeline_autogen": "openswe_traces.authoring.autogen",
    "harvest": "openswe_traces.authoring.harvest",
    "stage_units": "openswe_traces.authoring.stage_units",
    "preaudit": "openswe_traces.authoring.preaudit",
    "contract_gap_read": "openswe_traces.authoring.contract_gap_read",
    "task_lint": "openswe_traces.authoring.task_lint",
    "excision_registry": "openswe_traces.authoring.excision_registry",
    "repair_b6": "openswe_traces.authoring.repair_b6",
}

OPS = pathlib.Path("scripts/ops")
SRC = pathlib.Path("src")

SHIM = '''#!/usr/bin/env python3
"""Moved to {new}. This path keeps old commands and `import {old}` working."""
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[2] / "src"))
from {pkg} import {leaf} as _m  # noqa: E402

if __name__ == "__main__":
    _m.cli()
else:
    sys.modules[__name__] = _m
'''


def target(old):
    return SRC / (MOVES[old].replace(".", "/") + ".py")


def rewrite_imports(text):
    """Bare imports of moved modules become package imports under the same local name."""
    names = "|".join(sorted(MOVES, key=len, reverse=True))

    def plain(m):
        indent, old, alias = m.group(1), m.group(2), m.group(3)
        pkg, leaf = MOVES[old].rsplit(".", 1)
        return f"{indent}from {pkg} import {leaf} as {alias or old}"

    text = re.sub(rf"^(\s*)import ({names})(?: as (\w+))?(?=\s*(?:#.*)?$)", plain, text,
                  flags=re.M)
    text = re.sub(rf"^(\s*)from ({names}) import ",
                  lambda m: f"{m.group(1)}from {MOVES[m.group(2)]} import ", text, flags=re.M)
    return text


def drop_path_hacks(text):
    """sys.path.insert(...) lines that point at scripts/ops or src are no longer needed."""
    text = re.sub(r"^(import [\w, ]+); sys\.path\.insert\(0, .*\)\n", r"\1\n", text, flags=re.M)
    return re.sub(r"^[ \t]*sys\.path\.insert\(0, .*(?:__file__|REPO).*\)\n", "", text, flags=re.M)


# Repo-root lookups that count up from the file's own location. Each resolves to the
# checkout root from scripts/ops, and to the wrong place from src/openswe_traces/<pkg>.
_DIRNAME3 = (r"os\.path\.dirname\(os\.path\.dirname\(os\.path\.dirname\(\s*"
             r"os\.path\.abspath\(__file__\)\)\)\)")
REPO_PATTERNS = [
    (r"pathlib\.Path\(__file__\)\.resolve\(\)\.parents\[2\]", "_REPO"),
    (r"Path\(__file__\)\.resolve\(\)\.parents\[2\]", "_REPO"),
    (_DIRNAME3, "str(_REPO)"),
]


def use_repo_constant(text):
    n = 0
    for pat, rep in REPO_PATTERNS:
        text, k = re.subn(pat, rep, text)
        n += k
    if n:
        # After the module docstring and any __future__ import, before the first use.
        m = re.search(r"^(?:from __future__ import .*\n)", text, flags=re.M) or \
            re.search(r"^(?:import |from )", text, flags=re.M)
        at = m.end() if m and m.group(0).startswith("from __future__") else m.start()
        text = text[:at] + "from openswe_traces.paths import REPO as _REPO\n" + text[at:]
    return text


def main_to_cli(text):
    m = re.search(r'^if __name__ == "__main__":\n', text, flags=re.M)
    if not m:
        return text
    return (text[:m.start()] + "def cli():\n" + text[m.end():].rstrip("\n")
            + '\n\n\nif __name__ == "__main__":\n    cli()\n')


def ensure_package(path):
    path.parent.mkdir(parents=True, exist_ok=True)
    for d in [path.parent, *path.parent.parents]:
        if d == SRC:
            break
        init = d / "__init__.py"
        if not init.exists():
            init.write_text("")
            subprocess.run(["git", "add", str(init)], check=True)


def move(old):
    src, dst = OPS / f"{old}.py", target(old)
    ensure_package(dst)
    subprocess.run(["git", "mv", str(src), str(dst)], check=True)
    dst.write_text(main_to_cli(use_repo_constant(drop_path_hacks(dst.read_text()))))
    pkg, leaf = MOVES[old].rsplit(".", 1)
    src.write_text(SHIM.format(new=MOVES[old], old=old, pkg=pkg, leaf=leaf))
    subprocess.run(["git", "add", str(src), str(dst)], check=True)


def main(argv):
    unknown = [a for a in argv if a not in MOVES]
    if not argv or unknown:
        sys.exit(f"usage: move_module.py <stem>...; unknown: {unknown}; known: {list(MOVES)}")
    for old in argv:
        move(old)
    for old in MOVES:
        p = target(old)
        if p.exists():
            p.write_text(rewrite_imports(p.read_text()))
    print("moved:", ", ".join(f"{o} -> {MOVES[o]}" for o in argv))


if __name__ == "__main__":
    main(sys.argv[1:])

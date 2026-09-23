"""Repository locations, defined once.

Modules used to find the repo root by counting `dirname`s up from their own file, which
breaks the moment a file moves. Everything that needs an absolute repo path imports REPO.
Paths that are deliberately relative to the working directory (the jobs tree, supervisor
rosters) stay relative in their modules: ledger rows carry those relative paths, and
worktree runs depend on resolving them against cwd.
"""
import pathlib

REPO = pathlib.Path(__file__).resolve().parents[2]
SWEEPS = REPO / "experiments" / "dose_response"
SUPERVISOR = REPO / "outputs" / "supervisor"

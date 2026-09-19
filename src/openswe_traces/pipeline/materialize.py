"""Rebuild task ``environment/src`` trees from repos.yaml (pinned commit, obfuscation) + excision patches.

Task packages are committed without their source trees (each is a full repo copy). On a fresh
machine run ``scripts/materialize_tasks.py`` after ``uv sync``; it prepares each repo (clone at the
pinned commit, obfuscate if the spec says so, build ``ladder-base:<repo>``) and materialises every
task directory that lacks ``environment/src``.
"""

from __future__ import annotations

import json
import logging
import shutil
import subprocess
from pathlib import Path

from openswe_traces.pipeline.config import PipelineConfig, RepoSpec
from openswe_traces.pipeline.prepare import prepare_repo

log = logging.getLogger("openswe.pipeline.materialize")
IGNORE = shutil.ignore_patterns(".git", "__pycache__")


def unit_base(unit: str) -> str:
    return unit.removesuffix("-cv")


def excision_patch(cfg: PipelineConfig, repo: str, unit: str) -> Path | None:
    for cand in (unit_base(unit), unit):
        p = cfg.authored_dir / repo / cand / "_author" / "excised" / "excision.patch"
        if p.is_file():
            return p
    return None


def task_dirs(*roots: Path) -> list[tuple[str, str, int, Path]]:
    out: list[tuple[str, str, int, Path]] = []
    for root in roots:
        if not root.is_dir():
            continue
        for repo_dir in sorted(p for p in root.iterdir() if p.is_dir()):
            for td in sorted(p for p in repo_dir.iterdir() if p.is_dir()):
                name = td.name
                if "-L" not in name or not (td / "task.toml").is_file():
                    continue
                unit, _, lvl = name.rpartition("-L")
                if not lvl.isdigit():
                    continue
                out.append((repo_dir.name, unit, int(lvl), td))
    return out


def apply_patch(tree: Path, patch: Path) -> None:
    if shutil.which("git"):
        proc = subprocess.run(
            ["git", "apply", "-p1", "--unsafe-paths", "--directory=.", str(patch)],
            cwd=tree,
            capture_output=True,
            text=True,
            check=False,
        )
    else:
        proc = subprocess.run(
            ["patch", "-p1", "--forward", "--batch", "-i", str(patch)],
            cwd=tree,
            capture_output=True,
            text=True,
            check=False,
        )
    if proc.returncode != 0:
        raise RuntimeError(
            f"excision patch failed in {tree}: {(proc.stderr or proc.stdout)[-800:]}"
        )


def shipped_tests(td: Path) -> list[str]:
    """Hidden test files that this level ships inside the environment (L5 one, L6 all)."""
    aff = td / "affordance.json"
    if not aff.is_file():
        return []
    data = json.loads(aff.read_text(encoding="utf-8"))
    level = int(data.get("level", -1))
    hidden = list(data.get("hidden_tests") or [])
    if level == 4:
        return hidden
    if level == 3:
        rep = data.get("representative") or (hidden[0] if hidden else None)
        return [rep] if rep else []
    return []


def materialize_task(
    cfg: PipelineConfig,
    base_tree: Path,
    repo: str,
    unit: str,
    td: Path,
    *,
    dest: Path | None = None,
) -> Path:
    patch = excision_patch(cfg, repo, unit)
    if patch is None:
        raise FileNotFoundError(f"no excision.patch for {repo}/{unit}")
    target = dest or (td / "environment" / "src")
    if target.exists():
        shutil.rmtree(target)
    shutil.copytree(base_tree, target, symlinks=True, ignore=IGNORE)
    apply_patch(target, patch)
    for rel in shipped_tests(td):
        src = td / "tests" / "hidden" / rel
        if src.is_file():
            (target / rel).parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(src, target / rel)
    return target


def materialize_all(
    cfg: PipelineConfig,
    *,
    roots: list[Path] | None = None,
    only_missing: bool = True,
    repos: set[str] | None = None,
) -> int:
    roots = roots or [cfg.tasks_dir, cfg.tasks_dir.parent / "tasks_composerver"]
    specs: dict[str, RepoSpec] = {r.name: r for r in cfg.repos}
    trees: dict[str, Path] = {}
    n = 0
    for repo, unit, _lvl, td in task_dirs(*roots):
        if repos and repo not in repos:
            continue
        if only_missing and (td / "environment" / "src").is_dir():
            continue
        if repo not in specs:
            log.warning("skip %s: repo not in repos.yaml", td)
            continue
        if repo not in trees:
            info = prepare_repo(specs[repo], cfg)
            trees[repo] = Path(info["tree"])
        materialize_task(cfg, trees[repo], repo, unit, td)
        n += 1
        log.info("materialised %s", td)
    return n

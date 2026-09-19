"""Package L2 first; generate L0/L1/L3–L6 on demand via affordance.py."""

from __future__ import annotations

import json
import shutil
from pathlib import Path

from openswe_traces.pipeline.author import resolve_author_dir
from openswe_traces.pipeline.config import PipelineConfig
from openswe_traces.pipeline.ladder import INITIAL_PACKAGE_LEVELS, affordance_level
from openswe_traces.pipeline.prepare import image_tag
from openswe_traces.pipeline.verifier import collect_hidden, verifier_dir_for
from openswe_traces.synth.affordance import (
    HiddenTest,
    build_affordance_levels,
    render_ladder_base_dockerfile,
    render_unsolv_task_toml,
    with_no_web,
    write_hidden_tests,
)
from openswe_traces.synth.obfuscate import NO_WEB_CLAUSE


def _copytree(src: Path, dest: Path) -> None:
    if dest.exists():
        shutil.rmtree(dest)
    shutil.copytree(src, dest, ignore=shutil.ignore_patterns(".git", "__pycache__"))


def task_dir(cfg: PipelineConfig, repo: str, unit: str, level: int) -> Path:
    return cfg.tasks_dir / repo / f"{unit}-L{level}"


def packaged_levels(cfg: PipelineConfig, repo: str, unit: str) -> set[int]:
    root = cfg.tasks_dir / repo
    if not root.is_dir():
        return set()
    found: set[int] = set()
    prefix = f"{unit}-L"
    for child in root.iterdir():
        if child.is_dir() and child.name.startswith(prefix):
            suf = child.name[len(prefix) :]
            if suf.isdigit():
                found.add(int(suf))
    return found


def _l2_instruction(contract: str, bugreport: str) -> str:
    text = contract.rstrip() + "\n"
    if "expected" not in text.lower() and "actual" not in text.lower():
        text += "\n" + bugreport + "\n"
    if "Reproduce with:" not in text:
        text += (
            "\nReproduce with:\n\n```\ntests/test.sh\n```\n\n"
            "That script installs the hidden suite and runs it. Do not skip, delete, "
            "or weaken the tests. Work in `/app`.\n"
        )
    if NO_WEB_CLAUSE not in text:
        text = with_no_web(text)
    return text


def _skeleton(cfg: PipelineConfig, repo: str, unit: str, hidden: list[HiddenTest]) -> Path:
    author = resolve_author_dir(cfg, repo, unit)
    vdir = verifier_dir_for(cfg, repo, unit)
    skel = cfg.work_dir / repo / "units" / unit / "_skel"
    env = skel / "environment"
    env.mkdir(parents=True, exist_ok=True)
    src = env / "src"
    tree = author / "tree"
    if tree.is_dir():
        _copytree(tree, src)
    else:
        src.mkdir(parents=True, exist_ok=True)
    (env / "Dockerfile").write_text(
        render_ladder_base_dockerfile(image_tag(repo)), encoding="utf-8"
    )
    (skel / "task.toml").write_text(render_unsolv_task_toml(), encoding="utf-8")
    bugreport = (author / "bugreport.md").read_text(encoding="utf-8") if (author / "bugreport.md").is_file() else ""
    (skel / "instruction.md").write_text(with_no_web(bugreport), encoding="utf-8")
    write_hidden_tests(skel / "tests", hidden)
    if (vdir / "tests" / "test.sh").is_file():
        shutil.copy2(vdir / "tests" / "test.sh", skel / "tests" / "test.sh")
    for name in ("gold.patch", "cheat.patch"):
        src_p = author / name
        if src_p.is_file():
            (skel / "patches").mkdir(exist_ok=True)
            shutil.copy2(src_p, skel / "patches" / name)
            (skel / "tests").mkdir(exist_ok=True)
            shutil.copy2(src_p, skel / "tests" / name)
    return skel


def _packages_for(hidden: list[HiddenTest]) -> tuple[str, ...]:
    pkgs = sorted({str(Path(h.relpath).parent).replace("\\", "/") or "." for h in hidden})
    return tuple(pkgs) or (".",)


def _closure(author: Path) -> dict:
    path = author / "closure.json"
    if not path.is_file():
        return {}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return {}
    return data if isinstance(data, dict) else {}


def package_levels(
    repo: str,
    unit: str,
    cfg: PipelineConfig,
    levels: list[int],
    *,
    hidden: list[HiddenTest] | None = None,
) -> dict[int, Path]:
    author = resolve_author_dir(cfg, repo, unit)
    vdir = verifier_dir_for(cfg, repo, unit)
    tests = hidden if hidden is not None else collect_hidden(vdir)
    if not tests:
        raise FileNotFoundError(f"no hidden tests for {repo}/{unit}")
    already = packaged_levels(cfg, repo, unit)
    needed = [lv for lv in levels if lv not in already]
    if not needed:
        return {lv: task_dir(cfg, repo, unit, lv) for lv in levels if lv in already}
    skel = _skeleton(cfg, repo, unit, tests)
    bugreport = (author / "bugreport.md").read_text(encoding="utf-8") if (author / "bugreport.md").is_file() else ""
    contract = (author / "contract.md").read_text(encoding="utf-8") if (author / "contract.md").is_file() else ""
    closure = _closure(author)
    dest_root = cfg.tasks_dir / repo
    dest_root.mkdir(parents=True, exist_ok=True)
    aff_levels = [affordance_level(lv) for lv in needed]
    instructions = {}
    for lv in needed:
        aff = affordance_level(lv)
        if lv == 0:
            instructions[aff] = bugreport
        else:
            instructions[aff] = _l2_instruction(contract, bugreport)
    built = build_affordance_levels(
        skel,
        tests,
        levels=aff_levels,
        dest_root=dest_root,
        family=unit,
        instruction_a0=_l2_instruction(contract, bugreport),
        packages=_packages_for(tests),
        changed_symbols=tuple(closure.get("functions") or []),
        changed_files=tuple(closure.get("files") or []),
        name_scheme="L",
        dockerfile_from=image_tag(repo),
        instructions=instructions,
    )
    out: dict[int, Path] = {}
    for aff, path in built.items():
        lv = next(k for k, v in ((k, affordance_level(k)) for k in needed) if v == aff)
        out[lv] = path
        for name in ("gold.patch", "cheat.patch"):
            src = author / name
            if src.is_file():
                (path / "patches").mkdir(exist_ok=True)
                shutil.copy2(src, path / "patches" / name)
    return out


def package_unit(repo: str, unit: str, cfg: PipelineConfig) -> dict[int, Path]:
    start = list(cfg.climb_levels[:1]) or list(INITIAL_PACKAGE_LEVELS)
    return package_levels(repo, unit, cfg, start)


def ensure_level(repo: str, unit: str, cfg: PipelineConfig, level: int) -> Path:
    have = packaged_levels(cfg, repo, unit)
    if level in have:
        return task_dir(cfg, repo, unit, level)
    built = package_levels(repo, unit, cfg, [level])
    return built[level]

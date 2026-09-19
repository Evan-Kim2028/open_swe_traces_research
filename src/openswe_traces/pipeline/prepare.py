"""Clone at commit, strip .git, obfuscate identity, build ladder-base:<repo>, record baseline."""

from __future__ import annotations

import json
import re
import shutil
import subprocess
from collections.abc import Callable
from pathlib import Path
from typing import Any

from openswe_traces.pipeline.config import PipelineConfig, RepoSpec
from openswe_traces.synth.obfuscate import (
    collect_protected_literals,
    go_mod_tidy_offline,
    rename_identity_dirs,
    rewrite_module_path,
    strip_identity_text,
)

OK_RE = re.compile(r"^ok\s+(\S+)")
FAIL_RE = re.compile(r"^FAIL\s+(\S+)")
NOTEST_RE = re.compile(r"^\?\s+(\S+)")
MODULE_RE = re.compile(r"^module\s+(\S+)", re.MULTILINE)

BASE_DOCKERFILE = """FROM golang:1.23
RUN apt-get update && apt-get install -y --no-install-recommends \\
        git tmux ca-certificates patch gcc libc6-dev tree \\
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY src/ /app/
RUN find /app -name .git -type d -prune -exec rm -rf {{}} + || true
RUN GOPROXY=https://proxy.golang.org,direct go mod download
RUN GOPROXY=off GOSUMDB=off GOFLAGS=-mod=mod go build ./...
"""


class BaselineError(RuntimeError):
    """Raised when the offline baseline has no passing packages."""


def image_tag(repo: str) -> str:
    return f"ladder-base:{repo}"


def parse_baseline_output(text: str) -> dict[str, Any]:
    passing: list[str] = []
    failing: list[str] = []
    no_tests: list[str] = []
    for line in (text or "").splitlines():
        m = OK_RE.match(line)
        if m:
            passing.append(m.group(1))
            continue
        m = FAIL_RE.match(line)
        if m:
            failing.append(m.group(1))
            continue
        m = NOTEST_RE.match(line)
        if m:
            no_tests.append(m.group(1))
    return {
        "passing": passing,
        "failing": failing,
        "no_tests": no_tests,
        "n_passing": len(passing),
        "n_failing": len(failing),
    }


def read_module_path(tree: Path) -> str:
    go_mod = tree / "go.mod"
    if not go_mod.is_file():
        return ""
    m = MODULE_RE.search(go_mod.read_text(encoding="utf-8", errors="replace"))
    return m.group(1) if m else ""


def obfuscate_repo_tree(tree: Path, repo: RepoSpec) -> dict[str, Any]:
    old = repo.module or read_module_path(tree)
    slug = re.sub(r"[^a-z0-9]+", "", repo.name.lower()) or "repo"
    new = f"example.internal/{slug}"
    if old:
        rewrite_module_path(tree, old, new)
    if "tikv" in old or "client-go" in old:
        rename_identity_dirs(tree)
    protected = collect_protected_literals(tree)
    hits = strip_identity_text(tree, protected)
    tidy_ok = True
    tidy_err = ""
    try:
        go_mod_tidy_offline(tree)
    except RuntimeError as exc:
        tidy_ok = False
        tidy_err = str(exc)[-1500:]
    return {
        "old_module": old,
        "new_module": new,
        "strip": hits,
        "tidy_ok": tidy_ok,
        "tidy_err": tidy_err,
    }


def clone_at_commit(
    spec: RepoSpec,
    dest: Path,
    *,
    run: Callable[..., subprocess.CompletedProcess[str]] = subprocess.run,
) -> Path:
    if dest.exists():
        shutil.rmtree(dest)
    dest.parent.mkdir(parents=True, exist_ok=True)
    proc = run(
        ["git", "clone", "--filter=blob:none", spec.url, str(dest)],
        capture_output=True,
        text=True,
        timeout=600,
        check=False,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"git clone {spec.url} failed: {(proc.stderr or proc.stdout)[-2000:]}")
    chk = run(
        ["git", "-C", str(dest), "checkout", "--detach", spec.commit],
        capture_output=True,
        text=True,
        timeout=120,
        check=False,
    )
    if chk.returncode != 0:
        raise RuntimeError(f"git checkout {spec.commit} failed: {(chk.stderr or chk.stdout)[-2000:]}")
    git_dir = dest / ".git"
    if git_dir.exists():
        shutil.rmtree(git_dir)
    return dest


def write_base_dockerfile(dest: Path) -> Path:
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_text(BASE_DOCKERFILE, encoding="utf-8")
    return dest


def build_base_image(
    src: Path,
    repo: str,
    *,
    docker: Callable[..., subprocess.CompletedProcess[str]] | None = None,
    timeout: int = 3600,
) -> str:
    tag = image_tag(repo)
    tmp = src.parent / f".image_{repo}"
    if tmp.exists():
        shutil.rmtree(tmp)
    tmp.mkdir(parents=True)
    write_base_dockerfile(tmp / "Dockerfile")
    shutil.copytree(src, tmp / "src", ignore=shutil.ignore_patterns(".git", "__pycache__"))
    runner = docker or (
        lambda *args, **kw: subprocess.run(
            ["docker", *args], capture_output=True, text=True, timeout=kw.get("timeout", timeout), check=False
        )
    )
    proc = runner("build", "-t", tag, "-f", str(tmp / "Dockerfile"), str(tmp), timeout=timeout)
    if proc.returncode != 0:
        raise RuntimeError(f"docker build {tag} failed: {(proc.stderr or proc.stdout)[-2500:]}")
    return tag


def record_baseline(
    tag: str,
    dest_json: Path,
    *,
    docker_run: Callable[..., subprocess.CompletedProcess[str]] | None = None,
    timeout: int = 3600,
) -> dict[str, Any]:
    runner = docker_run or (
        lambda *args, **kw: subprocess.run(
            ["docker", *args], capture_output=True, text=True, timeout=kw.get("timeout", timeout), check=False
        )
    )
    proc = runner(
        "run",
        "--rm",
        "--network",
        "none",
        "-e",
        "GOPROXY=off",
        "-e",
        "GOSUMDB=off",
        tag,
        "go",
        "test",
        "./...",
        "-count=1",
        "-timeout=20m",
        timeout=timeout,
    )
    text = (proc.stdout or "") + (proc.stderr or "")
    parsed = parse_baseline_output(text)
    parsed["returncode"] = proc.returncode
    parsed["tail"] = text[-4000:]
    dest_json.parent.mkdir(parents=True, exist_ok=True)
    dest_json.write_text(json.dumps(parsed, indent=2) + "\n", encoding="utf-8")
    if parsed["n_passing"] <= 0:
        raise BaselineError(f"baseline has 0 passing packages for {tag}")
    return parsed


def prepare_repo(
    spec: RepoSpec,
    cfg: PipelineConfig,
    *,
    clone: Callable[[RepoSpec, Path], Path] | None = None,
    docker_build: Callable[..., str] | None = None,
    docker_baseline: Callable[..., dict[str, Any]] | None = None,
) -> dict[str, Any]:
    work = cfg.work_dir / spec.name
    tree = work / "tree"
    marker = work / "PREPARE_OK"
    if marker.is_file() and tree.is_dir():
        baseline = json.loads((work / "baseline.json").read_text(encoding="utf-8"))
        return {"skipped": True, "tree": str(tree), "baseline": baseline, "image": image_tag(spec.name)}
    work.mkdir(parents=True, exist_ok=True)
    cloner = clone or (lambda s, d: clone_at_commit(s, d))
    cloner(spec, tree)
    obf = obfuscate_repo_tree(tree, spec)
    (work / "obfuscate.json").write_text(json.dumps(obf, indent=2) + "\n", encoding="utf-8")
    builder = docker_build or (lambda src, name: build_base_image(src, name))
    tag = builder(tree, spec.name)
    baseliner = docker_baseline or (
        lambda t, dest: record_baseline(t, dest)
    )
    baseline = baseliner(tag, work / "baseline.json")
    marker.write_text(tag + "\n", encoding="utf-8")
    return {"skipped": False, "tree": str(tree), "baseline": baseline, "image": tag, "obfuscate": obf}

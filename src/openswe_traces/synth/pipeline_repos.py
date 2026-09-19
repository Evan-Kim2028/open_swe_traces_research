"""Prepare overnight pipeline Go repo set: clone, identity-obfuscate, baseline image+tests."""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from datetime import UTC, datetime
from functools import partial
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.obfuscate import identity_pass

PIPE_ROOT = ROOT / "experiments" / "pipeline"
REPOS_ROOT = PIPE_ROOT / "repos"
REPOS_YAML = PIPE_ROOT / "repos.yaml"
REPOS_MD = PIPE_ROOT / "REPOS.md"

EXCLUDE = ("tikv/client-go", "vaskoz/dailycodingproblem-go", "mgechev/revive")
MIN_PASSING = 5
CLONE_WORKERS = 2
BUILD_WORKERS = 2
NICE = 15
TEST_TIMEOUT_MIN = 20
DOCKER_TEST_TIMEOUT_SEC = TEST_TIMEOUT_MIN * 60 + 120
CLONE_TIMEOUT_SEC = 1800
BUILD_TIMEOUT_SEC = 3600

DOCKERFILE_APT = (
    "FROM golang:1.23\n"
    "RUN apt-get update && apt-get install -y --no-install-recommends "
    "git tmux ca-certificates patch gcc libc6-dev && rm -rf /var/lib/apt/lists/*\n"
)


@dataclass(frozen=True)
class RepoSpec:
    name: str
    github: str
    url: str
    new_module: str
    brands: tuple[tuple[str, str], ...]
    mid_hard: int
    hard: int
    mid: int
    n_instances: int
    backup: bool = False


# Ranked 2026-09-18 from outputs/task_difficulty.parquet (Go, mid+hard instance counts).
CANDIDATES: tuple[RepoSpec, ...] = (
    RepoSpec(
        "knative-client",
        "knative/client",
        "https://github.com/knative/client.git",
        "example.internal/eventkit",
        (("Knative", "Eventkit"), ("knative", "eventkit")),
        mid_hard=17,
        hard=9,
        mid=8,
        n_instances=71,
    ),
    RepoSpec(
        "goa",
        "goadesign/goa",
        "https://github.com/goadesign/goa.git",
        "example.internal/apikit",
        (
            ("Goadesign", "Apikit"),
            ("goadesign", "apikit"),
            ("Goa", "Apikit"),
            ("goa", "apikit"),
        ),
        mid_hard=11,
        hard=6,
        mid=5,
        n_instances=57,
    ),
    RepoSpec(
        "go-github",
        "google/go-github",
        "https://github.com/google/go-github.git",
        "example.internal/forgeapi",
        (("GitHub", "ForgeAPI"), ("go-github", "forgeapi")),
        mid_hard=9,
        hard=2,
        mid=7,
        n_instances=71,
    ),
    RepoSpec(
        "kops",
        "kubernetes/kops",
        "https://github.com/kubernetes/kops.git",
        "example.internal/clustkit",
        (("Kubernetes", "ClusterKit"), ("kOps", "Clustkit"), ("kops", "clustkit")),
        mid_hard=9,
        hard=4,
        mid=5,
        n_instances=61,
    ),
    RepoSpec(
        "argo",
        "argoproj/argo",
        "https://github.com/argoproj/argo.git",
        "example.internal/flowkit",
        (
            ("Argoproj", "Flowkit"),
            ("argoproj", "flowkit"),
            ("Argo", "Flowkit"),
            ("argo", "flowkit"),
        ),
        mid_hard=7,
        hard=4,
        mid=3,
        n_instances=88,
    ),
    RepoSpec(
        "nats-server",
        "nats-io/nats-server",
        "https://github.com/nats-io/nats-server.git",
        "example.internal/msgkit",
        (("NATS", "Msgkit"), ("nats-io", "msgkit"), ("Nats", "Msgkit"), ("nats", "msgkit")),
        mid_hard=2,
        hard=0,
        mid=2,
        n_instances=55,
    ),
)

BACKUP = RepoSpec(
    "helm",
    "helm/helm",
    "https://github.com/helm/helm.git",
    "example.internal/chartkit",
    (("Helm", "ChartKit"), ("helm", "chartkit")),
    mid_hard=0,
    hard=0,
    mid=0,
    n_instances=136,
    backup=True,
)


def _run(
    args: list[str],
    *,
    cwd: Path | None = None,
    timeout: int,
    extra_env: dict[str, str] | None = None,
) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    env["GIT_TERMINAL_PROMPT"] = "0"
    env["GOTOOLCHAIN"] = env.get("GOTOOLCHAIN", "local")
    if extra_env:
        env.update(extra_env)
    return subprocess.run(
        args,
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
        env=env,
    )


def _nice_prefix() -> list[str]:
    return ["nice", "-n", str(NICE)]


def primary_mod_dir(tree: Path) -> Path:
    mods = [
        p
        for p in tree.rglob("go.mod")
        if "vendor" not in p.parts and "testdata" not in p.parts
    ]
    if not mods:
        raise FileNotFoundError(f"no go.mod under {tree}")
    if (tree / "go.mod") in mods:
        return tree
    versioned: list[tuple[int, Path]] = []
    for path in mods:
        m = re.fullmatch(r"v(\d+)", path.parent.name)
        if m:
            versioned.append((int(m.group(1)), path.parent))
    if versioned:
        versioned.sort()
        return versioned[-1][1]
    mods.sort(key=lambda p: (len(p.relative_to(tree).parts), str(p)))
    return mods[0].parent


def write_dockerfile(repo_dir: Path, src: Path) -> str:
    rel = primary_mod_dir(src).relative_to(src).as_posix()
    workdir = "/app" if rel in {".", ""} else f"/app/{rel}"
    text = (
        DOCKERFILE_APT
        + "WORKDIR /app\n"
        + "COPY src/ /app/\n"
        + "RUN find /app -name .git -type d -prune -exec rm -rf {} + || true\n"
        + f"WORKDIR {workdir}\n"
        + "ENV GOTOOLCHAIN=auto\n"
        + "RUN GOPROXY=https://proxy.golang.org,direct go mod download || "
        + "GOPROXY=https://proxy.golang.org,direct go mod tidy || true\n"
        + "RUN GOPROXY=https://proxy.golang.org,direct go build ./...\n"
        + "RUN if [ -d integration_tests ]; then cd integration_tests && "
        + "GOPROXY=https://proxy.golang.org,direct GOFLAGS=-mod=mod go build ./... || true; fi\n"
    )
    (repo_dir / "Dockerfile").write_text(text)
    (repo_dir / ".dockerignore").write_text(".git\nsrc/.git\n*.log\nbaseline.json\n")
    return workdir


def clone_repo(spec: RepoSpec) -> dict[str, object]:
    dest = REPOS_ROOT / spec.name
    src = dest / "src"
    dest.mkdir(parents=True, exist_ok=True)
    commit_path = dest / "commit.txt"
    if src.is_dir() and commit_path.is_file() and not (src / ".git").exists():
        return {
            "name": spec.name,
            "skipped": True,
            "commit": commit_path.read_text().strip(),
        }
    if src.exists():
        shutil.rmtree(src)
    print(f"clone {spec.github} -> {src}", flush=True)
    proc = _run(
        [*_nice_prefix(), "git", "clone", "--depth", "1", "--single-branch", spec.url, str(src)],
        timeout=CLONE_TIMEOUT_SEC,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"clone {spec.name} failed: {(proc.stderr or proc.stdout)[-2000:]}")
    sha = _run(["git", "-C", str(src), "rev-parse", "HEAD"], timeout=30)
    if sha.returncode != 0:
        raise RuntimeError(f"rev-parse {spec.name}: {sha.stderr}")
    commit = sha.stdout.strip()
    commit_path.write_text(commit + "\n")
    (dest / "url.txt").write_text(spec.url + "\n")
    git_dir = src / ".git"
    if git_dir.exists():
        shutil.rmtree(git_dir)
    return {"name": spec.name, "skipped": False, "commit": commit}


def obfuscate_repo(spec: RepoSpec) -> dict[str, object]:
    dest = REPOS_ROOT / spec.name
    src = dest / "src"
    ident_path = dest / "identity.json"
    if ident_path.is_file():
        return json.loads(ident_path.read_text())
    print(f"identity-pass {spec.name}", flush=True)
    result = identity_pass(src, new_module=spec.new_module, brand_pairs=spec.brands)
    ident_path.write_text(json.dumps(result, indent=2) + "\n")
    return result


def parse_go_test_json(text: str) -> dict[str, dict[str, object]]:
    packages: dict[str, dict[str, object]] = {}
    for line in text.splitlines():
        line = line.strip()
        if not line.startswith("{"):
            continue
        try:
            ev = json.loads(line)
        except json.JSONDecodeError:
            continue
        pkg = ev.get("Package")
        action = ev.get("Action")
        if not pkg or ev.get("Test"):
            continue
        if action in {"pass", "fail", "skip"}:
            packages[pkg] = {"action": action, "elapsed": ev.get("Elapsed")}
    return packages


def image_exists(tag: str) -> bool:
    proc = _run(["docker", "image", "inspect", tag], timeout=30)
    return proc.returncode == 0


def build_image(spec: RepoSpec) -> dict[str, object]:
    dest = REPOS_ROOT / spec.name
    src = dest / "src"
    tag = f"ladder-base:{spec.name}"
    workdir = write_dockerfile(dest, src)
    log_path = dest / "docker_build.log"
    print(f"docker build {tag} (nice {NICE})", flush=True)
    t0 = time.perf_counter()
    proc = _run(
        [
            *_nice_prefix(),
            "docker",
            "build",
            "-t",
            tag,
            "-f",
            "Dockerfile",
            ".",
        ],
        cwd=dest,
        timeout=BUILD_TIMEOUT_SEC,
        extra_env={"DOCKER_BUILDKIT": "1"},
    )
    elapsed = time.perf_counter() - t0
    log_path.write_text((proc.stdout or "") + "\n" + (proc.stderr or ""))
    ok = proc.returncode == 0
    return {
        "tag": tag,
        "ok": ok,
        "elapsed_sec": round(elapsed, 2),
        "workdir": workdir,
        "tail": ((proc.stderr or proc.stdout) or "")[-2500:],
    }


def run_offline_tests(spec: RepoSpec, tag: str) -> dict[str, object]:
    json_path = REPOS_ROOT / spec.name / "test.jsonl"
    print(f"docker test --network none {tag}", flush=True)
    t0 = time.perf_counter()
    proc = _run(
        [
            "docker",
            "run",
            "--rm",
            "--network",
            "none",
            "-e",
            "GOPROXY=off",
            "-e",
            "GONOSUMCHECK=*",
            "-e",
            "GOTOOLCHAIN=auto",
            tag,
            "go",
            "test",
            "./...",
            "-count=1",
            f"-timeout={TEST_TIMEOUT_MIN}m",
            "-json",
        ],
        timeout=DOCKER_TEST_TIMEOUT_SEC,
    )
    wall = time.perf_counter() - t0
    out = (proc.stdout or "") + (proc.stderr or "")
    json_path.write_text(proc.stdout or "")
    packages = parse_go_test_json(out)
    passing = sorted(p for p, row in packages.items() if row["action"] == "pass")
    failing = sorted(p for p, row in packages.items() if row["action"] == "fail")
    skipped = sorted(p for p, row in packages.items() if row["action"] == "skip")
    return {
        "rc": proc.returncode,
        "wall_seconds": round(wall, 2),
        "packages": packages,
        "packages_passing": len(passing),
        "packages_failing": len(failing),
        "packages_skipped": len(skipped),
        "passing": passing,
        "failing": failing,
        "skipped": skipped,
        "timeout": proc.returncode == -1 or "test timed out" in out.lower(),
        "tail": out[-2000:],
    }


def evaluate(spec: RepoSpec, build: dict[str, object], tests: dict[str, object] | None) -> dict[str, object]:
    dest = REPOS_ROOT / spec.name
    commit = (dest / "commit.txt").read_text().strip()
    identity = {}
    ident_path = dest / "identity.json"
    if ident_path.is_file():
        identity = json.loads(ident_path.read_text())
    passing = int(tests["packages_passing"]) if tests else 0
    build_ok = bool(build.get("ok"))
    kept = build_ok and passing >= MIN_PASSING
    drop_reason = None
    if not build_ok:
        drop_reason = "image build failed (go build ./... or earlier Dockerfile step)"
    elif tests is None:
        drop_reason = "offline tests not run"
    elif passing < MIN_PASSING:
        drop_reason = (
            f"offline tests: {passing} packages passed "
            f"(need >={MIN_PASSING}); fail={tests['packages_failing']} skip={tests['packages_skipped']}"
        )
    notes = (
        f"parquet mid+hard={spec.mid_hard} (hard={spec.hard}, mid={spec.mid}, n={spec.n_instances}); "
        f"identity module {spec.new_module}; no symbol renames"
    )
    if spec.backup:
        notes += "; backup (0 mid/hard instances)"
    row = {
        "name": spec.name,
        "github": spec.github,
        "url": spec.url.removesuffix(".git"),
        "commit": commit,
        "base_image": f"ladder-base:{spec.name}",
        "packages_passing": passing,
        "packages_failing": int(tests["packages_failing"]) if tests else 0,
        "packages_skipped": int(tests["packages_skipped"]) if tests else 0,
        "wall_seconds": tests["wall_seconds"] if tests else None,
        "build_ok": build_ok,
        "build_seconds": build.get("elapsed_sec"),
        "kept": kept,
        "drop_reason": drop_reason,
        "notes": notes,
        "identity_modules": identity.get("modules"),
        "packages": tests["packages"] if tests else {},
        "passing": tests["passing"] if tests else [],
        "failing": tests["failing"] if tests else [],
        "skipped": tests["skipped"] if tests else [],
        "build_tail": build.get("tail", "")[-1500:],
        "test_tail": (tests or {}).get("tail", ""),
        "updated": datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ"),
    }
    (dest / "baseline.json").write_text(json.dumps(row, indent=2) + "\n")
    return row


def prepare_source(spec: RepoSpec) -> None:
    clone_repo(spec)
    obfuscate_repo(spec)
    write_dockerfile(REPOS_ROOT / spec.name, REPOS_ROOT / spec.name / "src")


def build_and_test(spec: RepoSpec, *, reuse_image: bool = False) -> dict[str, object]:
    dest = REPOS_ROOT / spec.name
    baseline = dest / "baseline.json"
    if baseline.is_file() and not reuse_image:
        prev = json.loads(baseline.read_text())
        if prev.get("build_ok") is not None and "packages_passing" in prev:
            print(f"skip build/test {spec.name}: baseline.json exists", flush=True)
            return prev
    tag = f"ladder-base:{spec.name}"
    if reuse_image and image_exists(tag):
        print(f"reuse image {tag}", flush=True)
        build = {"tag": tag, "ok": True, "elapsed_sec": 0, "workdir": "/app", "tail": "reused"}
    else:
        build = build_image(spec)
    tests = None
    if build["ok"]:
        tests = run_offline_tests(spec, build["tag"])
    return evaluate(spec, build, tests)


def _yaml_quote(value: str) -> str:
    if any(c in value for c in ":#{}[]&*?|>!%@`'\"") or value == "" or value.startswith(" "):
        return json.dumps(value)
    return value


def render_repos_yaml(kept: list[dict[str, object]], dropped: list[dict[str, object]]) -> str:
    lines = [
        "# Overnight pipeline repo set. Generated by scripts/prepare_pipeline_repos.py",
        f"# {datetime.now(UTC).strftime('%Y-%m-%dT%H:%M:%SZ')}",
        "keep_rule: build succeeds and >=5 packages pass `go test ./...` with --network none",
        "repos:",
    ]
    for row in kept:
        lines.append(f"  - name: {row['name']}")
        lines.append(f"    url: {row['url']}")
        lines.append(f"    commit: {row['commit']}")
        lines.append(f"    base_image: {row['base_image']}")
        lines.append(f"    packages_passing: {row['packages_passing']}")
        lines.append(f"    notes: {_yaml_quote(str(row['notes']))}")
    lines.append("dropped:")
    if not dropped:
        lines.append("  []")
    for row in dropped:
        reason = row.get("drop_reason") or "unknown"
        lines.append(f"  - name: {row['name']}")
        lines.append(f"    url: {row.get('url', '')}")
        lines.append(f"    reason: {_yaml_quote(str(reason))}")
    return "\n".join(lines) + "\n"


def render_repos_md(kept: list[dict[str, object]], dropped: list[dict[str, object]]) -> str:
    lines = [
        "# Pipeline repo set (Go)",
        "",
        (
            f"Generated {datetime.now(UTC).strftime('%Y-%m-%d %H:%M UTC')} by "
            "`uv run python scripts/prepare_pipeline_repos.py`."
        ),
        "",
        (
            "Source: `outputs/task_difficulty.parquet` mid+hard instance counts among "
            "helm/helm, argoproj/argo, knative/client, google/go-github, kubernetes/kops, "
            "goadesign/goa, nats-io/nats-server. Excluded: tikv/client-go, "
            "vaskoz/dailycodingproblem-go, mgechev/revive."
        ),
        "",
        (
            "Identity pass: module path + brand strings only (`obfuscate.py --identity-only`); "
            "no symbol or directory renames. Images `ladder-base:<name>` from golang:1.23, "
            "`go mod download`, `go build ./...`. Baseline: `docker run --network none "
            "go test ./... -count=1 -timeout 20m`. Keep if build succeeds and ≥5 packages pass."
        ),
        "",
        "## Kept",
        "",
        "| name | url | commit | base_image | packages_passing | notes |",
        "|---|---|---|---|---:|---|",
    ]
    for row in kept:
        sha = str(row["commit"])[:12]
        lines.append(
            f"| `{row['name']}` | {row['url']} | `{sha}` | `{row['base_image']}` | "
            f"{row['packages_passing']} | {row['notes']} |"
        )
    if not kept:
        lines.append("| *(none)* | | | | | |")
    lines.extend(
        [
            "",
            "## Dropped",
            "",
            "| name | reason |",
            "|---|---|",
        ]
    )
    for row in dropped:
        lines.append(f"| `{row['name']}` | {row.get('drop_reason') or 'unknown'} |")
    if not dropped:
        lines.append("| *(none)* | |")
    lines.extend(
        [
            "",
            "## Reproduce",
            "",
            "```",
            "uv run python scripts/prepare_pipeline_repos.py",
            "```",
            "",
            (
                "At most 2 image builds at a time, `nice -n 15`. Does not touch "
                "`experiments/harbor_nex/jobs`."
            ),
            "",
        ]
    )
    return "\n".join(lines)


def write_outputs(rows: list[dict[str, object]]) -> tuple[list[dict[str, object]], list[dict[str, object]]]:
    kept = [r for r in rows if r.get("kept")]
    dropped = [r for r in rows if not r.get("kept")]
    REPOS_YAML.write_text(render_repos_yaml(kept, dropped))
    REPOS_MD.write_text(render_repos_md(kept, dropped))
    return kept, dropped


def run_pipeline(
    *,
    include_backup: bool = True,
    force: bool = False,
    reclone: bool = False,
    retest: bool = False,
) -> list[dict[str, object]]:
    if reclone or force or retest:
        for spec in (*CANDIDATES, BACKUP):
            dest = REPOS_ROOT / spec.name
            if reclone:
                for p in (
                    dest / "src",
                    dest / "identity.json",
                    dest / "commit.txt",
                    dest / "baseline.json",
                    dest / "test.jsonl",
                    dest / "docker_build.log",
                ):
                    if p.is_dir():
                        shutil.rmtree(p)
                    elif p.exists():
                        p.unlink()
            elif force or retest:
                baseline = dest / "baseline.json"
                if baseline.exists():
                    baseline.unlink()
    REPOS_ROOT.mkdir(parents=True, exist_ok=True)
    specs = list(CANDIDATES)
    print("prepare sources (2 clones at a time)", flush=True)
    with ThreadPoolExecutor(max_workers=CLONE_WORKERS) as pool:
        futs = [pool.submit(prepare_source, spec) for spec in specs]
        for fut in as_completed(futs):
            fut.result()
    rows: list[dict[str, object]] = []
    worker = partial(build_and_test, reuse_image=retest)
    print("build+test (max 2 images)" if not retest else "retest existing images (max 2)", flush=True)
    with ThreadPoolExecutor(max_workers=BUILD_WORKERS) as pool:
        futs = {pool.submit(worker, spec): spec for spec in specs}
        for fut in as_completed(futs):
            spec = futs[fut]
            row = fut.result()
            print(
                f"{spec.name}: kept={row.get('kept')} pass={row.get('packages_passing')} "
                f"build={row.get('build_ok')}",
                flush=True,
            )
            rows.append(row)
    rows.sort(key=lambda r: specs.index(next(s for s in specs if s.name == r["name"])))
    kept_n = sum(1 for r in rows if r.get("kept"))
    if include_backup and kept_n < 6:
        print(f"only {kept_n} kept; trying backup helm", flush=True)
        prepare_source(BACKUP)
        row = worker(BACKUP)
        rows.append(row)
    write_outputs(rows)
    print(REPOS_MD.read_text(), flush=True)
    return rows


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Clone, identity-obfuscate, and baseline Go pipeline repos")
    parser.add_argument("--no-backup", action="store_true")
    parser.add_argument("--force", action="store_true", help="rebuild even if baseline.json exists")
    parser.add_argument("--reclone", action="store_true", help="delete src/identity and clone again")
    parser.add_argument("--retest", action="store_true", help="reuse images; rerun offline tests")
    args = parser.parse_args(argv)
    run_pipeline(
        include_backup=not args.no_backup,
        force=args.force,
        reclone=args.reclone,
        retest=args.retest,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

"""Prepare extra Go host trees under experiments/pipeline/repos2/."""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import time
from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.obfuscate import identity_pass

REPOS2_DIR = ROOT / "experiments" / "pipeline" / "repos2"
YAML_PATH = ROOT / "experiments" / "pipeline" / "repos2.yaml"
MD_PATH = ROOT / "experiments" / "pipeline" / "REPOS2.md"
KEEP_MIN_PASSING = 5

DOCKERFILE = """FROM golang:1.23
WORKDIR /app
COPY src/ /app/
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOTOOLCHAIN=auto
RUN go mod download
RUN go build ./...
"""

_PKG_TEXT_RE = re.compile(
    r"^(?P<status>ok|FAIL|\?)\s+(?P<pkg>\S+)\s+(?P<rest>.*)$"
)


@dataclass(frozen=True)
class Repo2Spec:
    name: str
    github: str
    new_module: str
    brands: tuple[tuple[str, str], ...]
    url: str = ""

    def git_url(self) -> str:
        return self.url or f"https://github.com/{self.github}.git"


SPECS: tuple[Repo2Spec, ...] = (
    Repo2Spec(
        name="nats-server",
        github="nats-io/nats-server",
        new_module="example.internal/msgbus",
        brands=(
            ("nats-io", "acme-msg"),
            ("NATS.io", "MsgBus"),
            ("nats.io", "msgbus.example"),
            ("Synadia", "Acme"),
            ("NATS", "MsgBus"),
        ),
    ),
    Repo2Spec(
        name="cobra",
        github="spf13/cobra",
        new_module="example.internal/clikit",
        brands=(
            ("spf13", "acme"),
            ("Cobra", "CliKit"),
            ("cobra", "clikit"),
        ),
    ),
    Repo2Spec(
        name="gin",
        github="gin-gonic/gin",
        new_module="example.internal/httprouter",
        brands=(
            ("gin-gonic", "acme-http"),
            ("Gin-Gonic", "AcmeHttp"),
            ("Gin", "HttpRouter"),
        ),
    ),
)


def repo_dir(spec: Repo2Spec) -> Path:
    return REPOS2_DIR / spec.name


def _run(
    args: Sequence[str],
    *,
    cwd: Path | None = None,
    timeout: int | None = None,
    extra_env: dict[str, str] | None = None,
) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    if extra_env:
        env.update(extra_env)
    return subprocess.run(
        list(args),
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
        env=env,
    )


def clone_shallow(spec: Repo2Spec) -> dict[str, object]:
    dest = repo_dir(spec)
    src = dest / "src"
    dest.mkdir(parents=True, exist_ok=True)
    if src.is_dir() and (src / "go.mod").is_file() and not (src / ".git").exists():
        meta_path = dest / "clone.json"
        if meta_path.is_file():
            return json.loads(meta_path.read_text(encoding="utf-8"))
    if src.exists():
        shutil.rmtree(src)
    proc = _run(
        ["git", "clone", "--depth", "1", spec.git_url(), str(src)],
        timeout=300,
        extra_env={"GIT_TERMINAL_PROMPT": "0"},
    )
    if proc.returncode != 0:
        raise RuntimeError(f"clone {spec.github} failed: {(proc.stderr or proc.stdout)[-2000:]}")
    sha_proc = _run(["git", "-C", str(src), "rev-parse", "HEAD"], timeout=30)
    if sha_proc.returncode != 0:
        raise RuntimeError(f"rev-parse failed for {spec.name}: {sha_proc.stderr}")
    commit = (sha_proc.stdout or "").strip()
    git_dir = src / ".git"
    if git_dir.exists():
        shutil.rmtree(git_dir)
    meta = {
        "name": spec.name,
        "github": spec.github,
        "url": spec.git_url(),
        "commit": commit,
    }
    (dest / "clone.json").write_text(json.dumps(meta, indent=2) + "\n", encoding="utf-8")
    return meta


def apply_identity(spec: Repo2Spec) -> dict[str, object]:
    src = repo_dir(spec) / "src"
    result = identity_pass(
        src,
        new_module=spec.new_module,
        brand_pairs=spec.brands,
    )
    (repo_dir(spec) / "identity.json").write_text(
        json.dumps(result, indent=2) + "\n", encoding="utf-8"
    )
    (repo_dir(spec) / "Dockerfile").write_text(DOCKERFILE, encoding="utf-8")
    return result


def _parse_test_packages(text: str) -> list[dict[str, object]]:
    by_pkg: dict[str, dict[str, object]] = {}
    for line in text.splitlines():
        raw = line.strip()
        if not raw:
            continue
        if raw.startswith("{"):
            try:
                ev = json.loads(raw)
            except json.JSONDecodeError:
                continue
            if not isinstance(ev, dict):
                continue
            action = ev.get("Action")
            pkg = ev.get("Package")
            if action not in {"pass", "fail", "skip"} or not pkg or ev.get("Test"):
                continue
            by_pkg[str(pkg)] = {
                "import_path": str(pkg),
                "status": {"pass": "ok", "skip": "skip", "fail": "fail"}[str(action)],
                "elapsed_sec": ev.get("Elapsed"),
            }
            continue
        tm = _PKG_TEXT_RE.match(raw)
        if not tm:
            continue
        status_raw = tm.group("status")
        pkg = tm.group("pkg")
        if pkg in by_pkg:
            continue
        rest = tm.group("rest").strip()
        status = {"ok": "ok", "FAIL": "fail", "?": "skip"}[status_raw]
        elapsed = None
        em = re.search(r"(\d+(?:\.\d+)?)s", rest)
        if em:
            elapsed = float(em.group(1))
        by_pkg[pkg] = {
            "import_path": pkg,
            "status": status,
            "elapsed_sec": elapsed,
            "detail": rest,
        }
    return [by_pkg[k] for k in sorted(by_pkg)]


def build_image(spec: Repo2Spec) -> dict[str, object]:
    dest = repo_dir(spec)
    tag = f"ladder-base:{spec.name}"
    log_path = dest / "build.log"
    t0 = time.monotonic()
    proc = subprocess.run(
        ["nice", "-n", "10", "docker", "build", "-t", tag, "-f", "Dockerfile", "."],
        cwd=dest,
        capture_output=True,
        text=True,
        timeout=3600,
        check=False,
    )
    wall = time.monotonic() - t0
    log_path.write_text((proc.stdout or "") + "\n" + (proc.stderr or ""), encoding="utf-8")
    return {
        "image": tag,
        "ok": proc.returncode == 0,
        "returncode": proc.returncode,
        "seconds": round(wall, 3),
        "tail": ((proc.stderr or proc.stdout) or "")[-2500:],
    }


def run_offline_tests(spec: Repo2Spec) -> dict[str, object]:
    dest = repo_dir(spec)
    tag = f"ladder-base:{spec.name}"
    log_path = dest / "test.log"
    t0 = time.monotonic()
    proc = subprocess.run(
        [
            "docker",
            "run",
            "--rm",
            "--network",
            "none",
            tag,
            "go",
            "test",
            "./...",
            "-json",
            "-count=1",
            "-timeout",
            "20m",
        ],
        capture_output=True,
        text=True,
        timeout=1500,
        check=False,
    )
    wall = time.monotonic() - t0
    out = (proc.stdout or "") + "\n" + (proc.stderr or "")
    log_path.write_text(out, encoding="utf-8")
    packages = _parse_test_packages(out)
    n_ok = sum(1 for p in packages if p["status"] == "ok")
    n_fail = sum(1 for p in packages if p["status"] == "fail")
    n_skip = sum(1 for p in packages if p["status"] == "skip")
    return {
        "returncode": proc.returncode,
        "seconds": round(wall, 3),
        "packages": packages,
        "packages_passing": n_ok,
        "packages_failing": n_fail,
        "packages_skipped": n_skip,
        "tail": out[-2500:],
    }


def write_baseline(
    spec: Repo2Spec,
    clone_meta: dict[str, object],
    identity: dict[str, object],
    build: dict[str, object],
    tests: dict[str, object] | None,
) -> dict[str, object]:
    n_ok = int((tests or {}).get("packages_passing") or 0)
    keep = bool(build.get("ok")) and n_ok >= KEEP_MIN_PASSING
    dropped = ""
    if not build.get("ok"):
        dropped = "image build failed"
    elif n_ok < KEEP_MIN_PASSING:
        dropped = f"only {n_ok} packages passed offline (need {KEEP_MIN_PASSING})"
    row = {
        "name": spec.name,
        "repo": spec.github,
        "url": clone_meta["url"],
        "commit": clone_meta["commit"],
        "base_image": f"ladder-base:{spec.name}",
        "new_module": spec.new_module,
        "identity": {
            "modules": identity.get("modules"),
            "module_files": identity.get("module_files"),
            "docs": identity.get("docs"),
            "go_comment_hits": identity.get("go_comment_hits"),
        },
        "build_ok": bool(build.get("ok")),
        "build_seconds": build.get("seconds"),
        "test_seconds": (tests or {}).get("seconds"),
        "packages_passing": n_ok,
        "packages_failing": (tests or {}).get("packages_failing", 0),
        "packages_skipped": (tests or {}).get("packages_skipped", 0),
        "packages": (tests or {}).get("packages") or [],
        "keep": keep,
        "notes": dropped,
    }
    (repo_dir(spec) / "baseline.json").write_text(
        json.dumps(row, indent=2) + "\n", encoding="utf-8"
    )
    return row


def _yaml_scalar(value: object) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    if value is None:
        return "null"
    if isinstance(value, (int, float)) and not isinstance(value, bool):
        return str(value)
    text = str(value)
    if text == "" or any(c in text for c in ":#{}[]&*?|>'\"%@`") or text[:1] in "-":
        return json.dumps(text)
    return text


def write_yaml(rows: Sequence[dict[str, object]]) -> None:
    lines = [
        "# Extra Go hosts for the overnight pipeline (identity-obfuscated).",
        "# Reproduce: uv run python scripts/prepare_repos2.py",
        "repos:",
    ]
    for row in rows:
        lines.append(f"  - name: {row['name']}")
        lines.append(f"    url: {row['url']}")
        lines.append(f"    commit: {row['commit']}")
        lines.append(f"    base_image: {row['base_image']}")
        lines.append("    language: go")
        mods = (row.get("identity") or {}).get("modules") or {}
        mapped = next(iter(mods.values()), row.get("new_module") or "") if isinstance(mods, dict) else row.get("new_module")
        lines.append(f"    module: {_yaml_scalar(mapped)}")
        lines.append(f"    packages_passing: {row['packages_passing']}")
        lines.append(f"    keep: {_yaml_scalar(row['keep'])}")
        notes = row.get("notes") or ("kept" if row["keep"] else "dropped")
        lines.append(f"    notes: {_yaml_scalar(notes)}")
    YAML_PATH.parent.mkdir(parents=True, exist_ok=True)
    YAML_PATH.write_text("\n".join(lines) + "\n", encoding="utf-8")


def write_md(rows: Sequence[dict[str, object]]) -> str:
    lines = [
        "# pipeline repos2",
        "",
        "Date: 2026-09-18. Extra Go hosts (nats-io/nats-server, spf13/cobra,",
        "gin-gonic/gin) cloned at HEAD, `.git` stripped, identity-obfuscated",
        "(module path + brand strings only; no symbol/dir renames), then built as",
        "`ladder-base:<name>` (`golang:1.23` + tree + `go mod download` + `go build ./...`).",
        "Offline `go test ./... -count=1 -timeout 20m` with `--network none`.",
        f"Keep if the image builds and ≥ {KEEP_MIN_PASSING} packages pass.",
        "Image builds ran one at a time under `nice -n 10`.",
        "Internet: `git clone`, `go mod download`, and `GOTOOLCHAIN=auto` (gin and",
        "nats-server declare go 1.26; the image is still `FROM golang:1.23`).",
        "",
        "Reproduce:",
        "",
        "```",
        "uv run python scripts/prepare_repos2.py",
        "```",
        "",
        "Output: `experiments/pipeline/repos2/<name>/` (`src/`, `Dockerfile`,",
        "`baseline.json`), plus `experiments/pipeline/repos2.yaml`.",
        "",
        "| repo | HEAD | image | build | pkgs pass/fail/skip | test wall | keep | notes |",
        "|---|---|---|---|---:|---:|---|---|",
    ]
    for row in rows:
        sha = str(row["commit"])[:12]
        test_s = row.get("test_seconds")
        wall = f"{test_s:.1f}s" if isinstance(test_s, (int, float)) else "?"
        build = "yes" if row.get("build_ok") else "no"
        keep = "yes" if row.get("keep") else "no"
        notes = str(row.get("notes") or "").replace("|", "/") or "—"
        pf = (
            f"{row.get('packages_passing', 0)}/"
            f"{row.get('packages_failing', 0)}/"
            f"{row.get('packages_skipped', 0)}"
        )
        lines.append(
            f"| `{row['repo']}` | `{sha}` | `{row['base_image']}` | {build} | {pf} | {wall} | {keep} | {notes} |"
        )
    lines.append("")
    lines.append("Identity (module path + brand strings only):")
    lines.append("")
    lines.append("| repo | old module | new module |")
    lines.append("|---|---|---|")
    for row in rows:
        mods = (row.get("identity") or {}).get("modules") or {}
        if isinstance(mods, dict) and mods:
            for old, new in mods.items():
                lines.append(f"| `{row['repo']}` | `{old}` | `{new}` |")
        else:
            lines.append(f"| `{row['repo']}` | — | `{row.get('new_module')}` |")
    lines.append("")
    md = "\n".join(lines)
    MD_PATH.write_text(md + "\n", encoding="utf-8")
    return md + "\n"


def prepare_one(spec: Repo2Spec, *, skip_docker: bool = False) -> dict[str, object]:
    dest = repo_dir(spec)
    dest.mkdir(parents=True, exist_ok=True)
    baseline_path = dest / "baseline.json"
    if baseline_path.is_file() and not skip_docker:
        prev = json.loads(baseline_path.read_text(encoding="utf-8"))
        if prev.get("build_ok") is not None and "packages" in prev:
            return prev
    clone_meta = clone_shallow(spec)
    identity = apply_identity(spec)
    if skip_docker:
        return write_baseline(
            spec,
            clone_meta,
            identity,
            {"ok": False, "seconds": 0, "image": f"ladder-base:{spec.name}"},
            None,
        )
    build = build_image(spec)
    tests = None
    if build["ok"]:
        tests = run_offline_tests(spec)
    return write_baseline(spec, clone_meta, identity, build, tests)


def prepare_all(*, skip_docker: bool = False) -> list[dict[str, object]]:
    REPOS2_DIR.mkdir(parents=True, exist_ok=True)
    (REPOS2_DIR / ".gitignore").write_text("*/src/\n*.log\n", encoding="utf-8")
    rows: list[dict[str, object]] = []
    for spec in SPECS:
        print(f"=== repos2 {spec.name} ===", flush=True)
        row = prepare_one(spec, skip_docker=skip_docker)
        rows.append(row)
        print(
            f"    build={row.get('build_ok')} pass={row.get('packages_passing')} keep={row.get('keep')}",
            flush=True,
        )
    write_yaml(rows)
    md = write_md(rows)
    print(md, flush=True)
    return rows


def main(argv: list[str] | None = None) -> int:
    import argparse

    parser = argparse.ArgumentParser(description="Prepare repos2 Go hosts")
    parser.add_argument("--skip-docker", action="store_true")
    parser.add_argument("--clone-only", action="store_true")
    args = parser.parse_args(argv)
    if args.clone_only:
        REPOS2_DIR.mkdir(parents=True, exist_ok=True)
        for spec in SPECS:
            print(f"clone {spec.github}", flush=True)
            meta = clone_shallow(spec)
            print(f"  {spec.name} {meta['commit'][:12]}", flush=True)
        return 0
    prepare_all(skip_docker=args.skip_docker)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

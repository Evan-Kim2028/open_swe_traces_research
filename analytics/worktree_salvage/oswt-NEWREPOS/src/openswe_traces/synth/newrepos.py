"""Prepare NEW Go bank candidates under experiments/pipeline/repos2/.

Same recipe as synth/repos2.py — clone at a pinned commit, strip .git,
identity_pass (module path + brand strings, no symbol renames), build
``ladder-base:<name>``, then offline ``go test ./... -count=1`` under
``--network none``. One deliberate difference: the Dockerfile's FROM line
follows the repo's own ``go`` directive (the ``base_dockerfile_for`` fix for
fresh_laptop_setup defect 2) instead of hardcoding golang:1.23.

Keep threshold: image builds and >= repos2.KEEP_MIN_PASSING packages pass.
"""

from __future__ import annotations

import argparse
import json
import time

from openswe_traces.pipeline.prepare import read_go_directive
from openswe_traces.synth.repos2 import (
    REPOS2_DIR,
    Repo2Spec,
    apply_identity,
    build_image,
    clone_shallow,
    run_offline_tests,
    write_baseline,
)

DOCKERFILE_TEMPLATE = """FROM golang:{go_image}
WORKDIR /app
COPY src/ /app/
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOTOOLCHAIN=auto
RUN go mod download
RUN go build ./...
"""

NEW_SPECS: tuple[Repo2Spec, ...] = (
    Repo2Spec(
        name="bbolt",
        github="etcd-io/bbolt",
        commit="4dc08f7187c709d355a8bcf334fcbcf7d7cfedf6",
        new_module="example.internal/boltstore",
        brands=(
            ("go.etcd.io", "acme.example"),
            ("etcd-io", "acme"),
            ("etcd", "acme"),
            ("bbolt", "boltstore"),
            ("Bbolt", "Boltstore"),
            ("BoltDB", "BoltStore"),
            ("boltdb", "boltstore"),
        ),
    ),
    Repo2Spec(
        name="fasthttp",
        github="valyala/fasthttp",
        commit="0e13c85e8ed5b41b489c502eeccdf90a45eb8dfe",
        new_module="example.internal/httpkit",
        brands=(
            ("valyala", "acme"),
            ("Valyala", "Acme"),
            ("fasthttp", "httpkit"),
            ("FastHTTP", "HttpKit"),
        ),
    ),
    Repo2Spec(
        name="viper",
        github="spf13/viper",
        commit="528f7416c4b56a4948673984b190bf8713f0c3c4",
        new_module="example.internal/confkit",
        brands=(
            ("spf13", "acme"),
            ("Viper", "ConfKit"),
            ("viper", "confkit"),
        ),
    ),
    Repo2Spec(
        name="buntdb",
        github="tidwall/buntdb",
        commit="0dbc8c18459aed4e4179bccbe0322bbd6019733f",
        new_module="example.internal/memkv",
        brands=(
            ("tidwall", "acme"),
            ("BuntDB", "MemKV"),
            ("buntdb", "memkv"),
        ),
    ),
    Repo2Spec(
        name="go-git",
        github="go-git/go-git",
        commit="0f3a0a2c25513f2666ac9b88a6745f7b2382f572",
        new_module="example.internal/gitkit",
        brands=(
            ("go-git", "gitkit"),
            ("GoGit", "GitKit"),
            ("go_git", "gitkit"),
        ),
    ),
)


def write_dockerfile_for_tree(spec: Repo2Spec) -> str:
    """repos2 Dockerfile shape, but FROM follows the tree's own go directive."""
    src = REPOS2_DIR / spec.name / "src"
    go_image = read_go_directive(src) or "1.23"
    (REPOS2_DIR / spec.name / "Dockerfile").write_text(
        DOCKERFILE_TEMPLATE.format(go_image=go_image), encoding="utf-8"
    )
    return go_image


def prepare_candidate(spec: Repo2Spec) -> dict[str, object]:
    """Full recipe for one candidate; writes baseline.json under repos2/<name>/."""
    clone_meta = clone_shallow(spec)
    identity = apply_identity(spec)
    go_image = write_dockerfile_for_tree(spec)
    build = build_image(spec)
    tests = None
    if build["ok"]:
        tests = run_offline_tests(spec)
    row = write_baseline(spec, clone_meta, identity, build, tests)
    row["go_image"] = go_image
    dest = REPOS2_DIR / spec.name
    (dest / "baseline.json").write_text(json.dumps(row, indent=2) + "\n", encoding="utf-8")
    return row


def prepare_newrepos(
    names: list[str] | None = None, *, only: set[str] | None = None
) -> list[dict[str, object]]:
    REPOS2_DIR.mkdir(parents=True, exist_ok=True)
    (REPOS2_DIR / ".gitignore").write_text("*/src/\n*.log\n", encoding="utf-8")
    rows: list[dict[str, object]] = []
    for spec in NEW_SPECS:
        if names is not None and spec.name not in names:
            continue
        if only and spec.name not in only:
            continue
        print(f"=== newrepos {spec.name} ({spec.github} @ {spec.commit[:12]}) ===", flush=True)
        t0 = time.monotonic()
        try:
            row = prepare_candidate(spec)
        except Exception as exc:  # noqa: BLE001 - record and continue with the next repo
            row = {"name": spec.name, "repo": spec.github, "error": str(exc)[-1500:], "keep": False}
            (REPOS2_DIR / spec.name).mkdir(parents=True, exist_ok=True)
            (REPOS2_DIR / spec.name / "baseline.json").write_text(
                json.dumps(row, indent=2) + "\n", encoding="utf-8"
            )
        row["wall_seconds"] = round(time.monotonic() - t0, 3)
        rows.append(row)
        print(
            f"    go={row.get('go_image', '?')} build={row.get('build_ok')} "
            f"pass={row.get('packages_passing')} fail={row.get('packages_failing')} "
            f"skip={row.get('packages_skipped')} keep={row.get('keep')} "
            f"wall={row['wall_seconds']:.1f}s",
            flush=True,
        )
    out = REPOS2_DIR / "newrepos_rows.json"
    out.write_text(json.dumps(rows, indent=2) + "\n", encoding="utf-8")
    return rows


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Prepare NEW Go bank candidates")
    parser.add_argument(
        "--only", nargs="*", default=None, help="restrict to these repo names"
    )
    parser.add_argument("--clone-only", action="store_true")
    args = parser.parse_args(argv)
    if args.clone_only:
        for spec in NEW_SPECS:
            if args.only and spec.name not in args.only:
                continue
            meta = clone_shallow(spec)
            print(f"{spec.name} {meta['commit'][:12]}", flush=True)
        return 0
    prepare_newrepos(only=set(args.only) if args.only else None)
    return 0

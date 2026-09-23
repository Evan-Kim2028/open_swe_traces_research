#!/usr/bin/env python3
"""Reclaim Docker storage without touching anything the pipeline needs warm.

``docker system prune -a`` would free ~70GB here and cost hours. Two things it
would take are load-bearing, and neither is obvious from docker's own accounting:

- ``ladder-base:*`` (9 images, 24GB). Every staged unit's Dockerfile starts
  ``FROM ladder-base:<repo>``, and they are also the only surviving copy of the
  pristine upstream tree that ``restore_env_src.py`` rebuilds excisions from.
  Losing one silently un-regenerates every unit cut from that repo.
- The ``ladder-base-gocache`` volume (32.7GB). ``docker system df`` calls it 99%
  reclaimable purely because no container has it mounted at rest, but it is the
  warm Go build cache; without it a kops or helm build recompiles from zero.

What is genuinely disposable:

- ``openswe-audit:*`` images - one per past audit run, derived from a base image
  plus a task, rebuilt on demand. 322 of them held 33GB of unique layers.
- per-trial ``*__env-main`` images from finished trials.
- exited containers and unreferenced build cache.

Any image a container still references is skipped, so a running trial is never
undercut.

Usage::

    docker_gc.py            # dry run
    docker_gc.py --apply
"""

from __future__ import annotations

import argparse
import re
import subprocess

# Repositories that are rebuilt on demand and safe to drop when unreferenced.
DISPOSABLE = (re.compile(r"^openswe-audit$"), re.compile(r".*__env-main$"))
# Never removed, whatever docker's "reclaimable" column says.
PROTECTED_IMAGES = re.compile(r"^ladder-base$")
PROTECTED_VOLUMES = {"ladder-base-gocache", "envio-postgres-data",
                     "buildx_buildkit_default_state"}


def sh(*args: str) -> str:
    return subprocess.run(args, capture_output=True, text=True, timeout=1800).stdout


def referenced() -> set[str]:
    """Images named by any container, running or not."""
    return set(sh("docker", "ps", "-a", "--format", "{{.Image}}").split())


def disposable_images() -> list[str]:
    keep = referenced()
    out = []
    for line in sh("docker", "images", "--format",
                   "{{.Repository}}:{{.Tag}}").splitlines():
        repo = line.rsplit(":", 1)[0]
        if PROTECTED_IMAGES.match(repo):
            continue
        if any(p.match(repo) for p in DISPOSABLE) and line not in keep:
            out.append(line)
    return out


def free_gb() -> int:
    return int(re.sub(r"\D", "", sh("df", "-BG", "--output=avail", "/").splitlines()[-1]))


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--apply", action="store_true")
    args = ap.parse_args()

    before = free_gb()
    imgs = disposable_images()
    print(f"free: {before}G")
    print(f"disposable images: {len(imgs)}")
    print(f"protected volumes kept: {', '.join(sorted(PROTECTED_VOLUMES))}")

    if not args.apply:
        print("\n(dry run — rerun with --apply)")
        return 0

    n = 0
    for img in imgs:
        if subprocess.run(["docker", "rmi", img], capture_output=True).returncode == 0:
            n += 1
    print(f"removed {n} image(s)")
    print(sh("docker", "container", "prune", "-f").strip().splitlines()[-1:])
    print(sh("docker", "builder", "prune", "-f").strip().splitlines()[-1:])

    survivors = sum(1 for line in sh("docker", "images", "--format",
                                     "{{.Repository}}").splitlines()
                    if line == "ladder-base")
    print(f"ladder-base images surviving: {survivors} (must be 9)")
    if survivors < 9:
        print("ALARM: a ladder-base image is missing — rebuild before staging anything")
        return 1
    print(f"free: {before}G -> {free_gb()}G")
    return 0


def cli():
    raise SystemExit(main())


if __name__ == "__main__":
    cli()

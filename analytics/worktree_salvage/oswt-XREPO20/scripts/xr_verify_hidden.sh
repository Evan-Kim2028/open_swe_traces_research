#!/bin/bash
# Verify a hidden test file against the PRISTINE pair tree, in-image, offline.
# Hardlink-copies the tree (instant, same-fs) so the host tree is never touched.
# Usage: xr_verify_hidden.sh <pair> <hidden.go> <relpath-in-tree> [run-regex]
set -euo pipefail
pair="$1"; hidden="$(cd "$(dirname "$2")" && pwd)/$(basename "$2")"; rel="$3"; runre="${4:-.}"
case "$pair" in
  gin)  src="$(dirname "$0")/../experiments/xrepo20/pair/gin/src";  img=ladder-base:gin ;;
  kops) src="$(dirname "$0")/../experiments/xrepo20/pair/kops/src"; img=ladder-base:kops ;;
  *) echo "unknown pair $pair" >&2; exit 2 ;;
esac
src="$(cd "$src" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
cp -a "$src" "$tmp/src"
mkdir -p "$tmp/src/$(dirname "$rel")"
cp "$hidden" "$tmp/src/$rel"
docker run --rm --network none -v "$tmp/src:/app" -w /app "$img" \
  sh -c "GOPROXY=off go test -count=1 -timeout 10m -run '$runre' './$(dirname "$rel")/' 2>&1 | tail -30"

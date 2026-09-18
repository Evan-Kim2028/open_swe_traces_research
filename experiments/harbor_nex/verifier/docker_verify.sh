#!/usr/bin/env bash
# Verify a Harbor Go task: tests fail on the image (buggy) and pass after restoring
# original files from the host repo at BASE.
set -euo pipefail
TASK_DIR=$1
IMAGE=$2
HOST_REPO=$3
shift 3
# remaining: relative files to restore from host repo (clean originals)
RESTORE=("$@")

ROOT=$(cd "$(dirname "$0")/../../.." && pwd)
TASK_DIR=$(cd "$TASK_DIR" && pwd)
HOST_REPO=$(cd "$HOST_REPO" && pwd)

echo "=== build $IMAGE from $TASK_DIR ==="
docker build -t "$IMAGE" -f "$TASK_DIR/environment/Dockerfile" "$TASK_DIR/environment"

run_tests() {
  local extra_args=("$@")
  docker run --rm \
    "${extra_args[@]}" \
    -v "$TASK_DIR/tests/test.sh:/tmp/test.sh:ro" \
    "$IMAGE" bash /tmp/test.sh
}

echo "=== buggy (expect FAIL) ==="
if run_tests; then
  echo "FAIL: buggy tree tests passed (should fail)"
  exit 1
fi
echo "buggy: tests failed as expected"

vol_args=()
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT
for rel in "${RESTORE[@]}"; do
  mkdir -p "$tmpdir/$(dirname "$rel")"
  # host repo is at the clean base commit
  cp "$HOST_REPO/$rel" "$tmpdir/$rel"
  vol_args+=(-v "$tmpdir/$rel:/app/$rel:ro")
done

echo "=== fixed (expect PASS) ==="
if ! run_tests "${vol_args[@]}"; then
  echo "FAIL: fixed tree tests failed (should pass)"
  exit 1
fi
echo "fixed: tests passed as expected"
echo "OK $IMAGE"

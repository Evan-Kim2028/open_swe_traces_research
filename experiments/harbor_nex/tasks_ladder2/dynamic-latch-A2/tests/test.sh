#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
HIDDEN="$TESTS_DIR/hidden"
install_hidden() {
  rel="$1"
  dest="/app/$rel"
  mkdir -p "$(dirname "$dest")"
  cp "$HIDDEN/$rel" "$dest"
}
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "c35cf8b5aef63db8290b587157f0e38eed724aa4f4c25231ca50ff2ae6ae53de  $HIDDEN/internal/latch/latch_dyn_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/latch/latch_dyn_test.go"
install_hidden "internal/latch/latch_dyn_test.go"
echo "c35cf8b5aef63db8290b587157f0e38eed724aa4f4c25231ca50ff2ae6ae53de  /app/internal/latch/latch_dyn_test.go" | sha256sum -c --status || checksum_fail "internal/latch/latch_dyn_test.go"
if go test -race -count=1 -timeout 15m -run '^(TestLatchExclusiveOverlap|TestLatchConcurrentSameKey)$' ./internal/latch/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

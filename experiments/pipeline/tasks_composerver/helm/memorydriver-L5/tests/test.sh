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
echo "9235cb1749c85b2522267b791bac8500bb89433dbc99f69cf8f617a67468044c  $HIDDEN/pkg/storage/driver/memorydriver_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/pkg/storage/driver/memorydriver_bb_prop_test.go"
install_hidden "pkg/storage/driver/memorydriver_bb_prop_test.go"
echo "9235cb1749c85b2522267b791bac8500bb89433dbc99f69cf8f617a67468044c  /app/pkg/storage/driver/memorydriver_bb_prop_test.go" | sha256sum -c --status || checksum_fail "pkg/storage/driver/memorydriver_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestMemoryDriverContractTableProperty|TestMemoryDriverCRUDProperty|TestMemoryDriverListQueryProperty|TestMemoryDriverAdversarialProperty|TestMemoryDriverUnseenRandomProperty|TestMemoryDriverConcurrentProperty)$' ./pkg/storage/driver/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

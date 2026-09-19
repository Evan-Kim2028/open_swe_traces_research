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
echo "a8fe6cfc9777579a2bc765afb8b8d3aca35b22e9ee3930b58aa8d3c2010f060b  $HIDDEN/internal/unionstore/memdbstaging_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/unionstore/memdbstaging_bb_prop_test.go"
install_hidden "internal/unionstore/memdbstaging_bb_prop_test.go"
echo "a8fe6cfc9777579a2bc765afb8b8d3aca35b22e9ee3930b58aa8d3c2010f060b  /app/internal/unionstore/memdbstaging_bb_prop_test.go" | sha256sum -c --status || checksum_fail "internal/unionstore/memdbstaging_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestHiddenMemDBRandomOps|TestHiddenMemDBNestedCompose|TestHiddenMemDBTombstoneLen|TestHiddenMemDBFlaggedTombstoneInspect|TestHiddenMemDBCheckpointRestore|TestHiddenMemDBFlagCleanupRevert|TestHiddenMemDBEmptySet|TestHiddenMemDBHandleSequence)$' ./internal/unionstore/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

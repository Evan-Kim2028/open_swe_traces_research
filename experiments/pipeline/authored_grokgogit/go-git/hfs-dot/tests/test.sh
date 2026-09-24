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
echo "15e9cf6b5e0f7fe101599ca3a4d6f12f6dfa40c9ae15538d4a008bec7ddf8e70  $HIDDEN/internal/pathutil/hfs_dot_bb_test.go" | sha256sum -c --status || checksum_fail "internal/pathutil/hfs_dot_bb_test.go"
install_hidden "internal/pathutil/hfs_dot_bb_test.go"
echo "15e9cf6b5e0f7fe101599ca3a4d6f12f6dfa40c9ae15538d4a008bec7ddf8e70  /app/internal/pathutil/hfs_dot_bb_test.go" | sha256sum -c --status || checksum_fail "internal/pathutil/hfs_dot_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./internal/pathutil/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

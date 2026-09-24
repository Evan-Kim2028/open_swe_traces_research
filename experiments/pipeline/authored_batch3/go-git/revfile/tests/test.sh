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
echo "7f2c0530e4a8064bec9d6b7cd24f0645ce732821acd4b8cbae46e895e9e86f28  $HIDDEN/plumbing/format/revfile/revfile_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/revfile/revfile_bb_test.go"
install_hidden "plumbing/format/revfile/revfile_bb_test.go"
echo "7f2c0530e4a8064bec9d6b7cd24f0645ce732821acd4b8cbae46e895e9e86f28  /app/plumbing/format/revfile/revfile_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/revfile/revfile_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/format/revfile/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

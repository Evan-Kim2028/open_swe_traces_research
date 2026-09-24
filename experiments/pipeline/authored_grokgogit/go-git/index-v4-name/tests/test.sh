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
echo "d3724c764079ac684211ecff0555ea7809d4164bbb784e42c5b04a4a91a47288  $HIDDEN/plumbing/format/index/index_v4_name_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/index/index_v4_name_bb_test.go"
install_hidden "plumbing/format/index/index_v4_name_bb_test.go"
echo "d3724c764079ac684211ecff0555ea7809d4164bbb784e42c5b04a4a91a47288  /app/plumbing/format/index/index_v4_name_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/index/index_v4_name_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/format/index/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

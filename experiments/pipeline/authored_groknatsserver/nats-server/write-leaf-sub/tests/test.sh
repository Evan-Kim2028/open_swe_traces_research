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
echo "fc40f500e656253ba50055fe2eb1ae547b6fa5c1a6c8fc6edf66d16454ecf306  $HIDDEN/server/write_leaf_sub_bb_test.go" | sha256sum -c --status || checksum_fail "server/write_leaf_sub_bb_test.go"
install_hidden "server/write_leaf_sub_bb_test.go"
echo "fc40f500e656253ba50055fe2eb1ae547b6fa5c1a6c8fc6edf66d16454ecf306  /app/server/write_leaf_sub_bb_test.go" | sha256sum -c --status || checksum_fail "server/write_leaf_sub_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./server/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

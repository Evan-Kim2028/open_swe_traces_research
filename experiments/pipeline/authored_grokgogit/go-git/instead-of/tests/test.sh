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
echo "a4beba6067fa92befa8e8a03013184bd5d96ca48b27a3e4c8b92b106e6c12db7  $HIDDEN/config/instead_of_bb_test.go" | sha256sum -c --status || checksum_fail "config/instead_of_bb_test.go"
install_hidden "config/instead_of_bb_test.go"
echo "a4beba6067fa92befa8e8a03013184bd5d96ca48b27a3e4c8b92b106e6c12db7  /app/config/instead_of_bb_test.go" | sha256sum -c --status || checksum_fail "config/instead_of_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./config/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

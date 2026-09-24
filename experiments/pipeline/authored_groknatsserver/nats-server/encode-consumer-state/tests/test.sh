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
echo "180ad6404d684546d619e32279ec63ef125814a37ea415ba6ed5094ccb1774a0  $HIDDEN/server/encode_consumer_state_bb_test.go" | sha256sum -c --status || checksum_fail "server/encode_consumer_state_bb_test.go"
install_hidden "server/encode_consumer_state_bb_test.go"
echo "180ad6404d684546d619e32279ec63ef125814a37ea415ba6ed5094ccb1774a0  /app/server/encode_consumer_state_bb_test.go" | sha256sum -c --status || checksum_fail "server/encode_consumer_state_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./server/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

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
echo "a2dec98c36a50ed82e6c68d112472549b7538244dce145e27a9c29536ccaf0ab  $HIDDEN/server/parse_msg_schedule_bb_test.go" | sha256sum -c --status || checksum_fail "server/parse_msg_schedule_bb_test.go"
install_hidden "server/parse_msg_schedule_bb_test.go"
echo "a2dec98c36a50ed82e6c68d112472549b7538244dce145e27a9c29536ccaf0ab  /app/server/parse_msg_schedule_bb_test.go" | sha256sum -c --status || checksum_fail "server/parse_msg_schedule_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./server/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

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
echo "d98471d974b88e17ce1ff7059b5c83d58094738e6ad97b214a87c5fa85b4088a  $HIDDEN/server/index_placeholders_bb_test.go" | sha256sum -c --status || checksum_fail "server/index_placeholders_bb_test.go"
install_hidden "server/index_placeholders_bb_test.go"
echo "d98471d974b88e17ce1ff7059b5c83d58094738e6ad97b214a87c5fa85b4088a  /app/server/index_placeholders_bb_test.go" | sha256sum -c --status || checksum_fail "server/index_placeholders_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./server/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

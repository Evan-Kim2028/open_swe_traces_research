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
echo "75c9b720f286a5a3d37f9241a7f908ebc5fecac4c4aefe241e43b45c1a53eee6  $HIDDEN/expr/attachsvc_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/expr/attachsvc_bb_test.go"
install_hidden "expr/attachsvc_bb_test.go"
echo "75c9b720f286a5a3d37f9241a7f908ebc5fecac4c4aefe241e43b45c1a53eee6  /app/expr/attachsvc_bb_test.go" | sha256sum -c --status || checksum_fail "expr/attachsvc_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./expr/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

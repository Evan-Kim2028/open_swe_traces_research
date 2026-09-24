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
echo "8e20f09bbc9ef944e164acc824d6df1f838c40d606c851c1eaefd3b424c5a2e6  $HIDDEN/expr/svcerrors_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/expr/svcerrors_bb_test.go"
install_hidden "expr/svcerrors_bb_test.go"
echo "8e20f09bbc9ef944e164acc824d6df1f838c40d606c851c1eaefd3b424c5a2e6  /app/expr/svcerrors_bb_test.go" | sha256sum -c --status || checksum_fail "expr/svcerrors_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./expr/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

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
echo "0f4d5c7f00a011883d70ccaf81e967bfcea41e0c3c4c7458da9518c85ce65c8b  $HIDDEN/internal/locate/onregionerror_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/locate/onregionerror_bb_prop_test.go"
install_hidden "internal/locate/onregionerror_bb_prop_test.go"
echo "0f4d5c7f00a011883d70ccaf81e967bfcea41e0c3c4c7458da9518c85ce65c8b  /app/internal/locate/onregionerror_bb_prop_test.go" | sha256sum -c --status || checksum_fail "internal/locate/onregionerror_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestOnRegionErrorRetryProperty|TestOnRegionErrorSendFailProperty|TestOnRegionErrorTerminalProperty|TestOnRegionErrorContractExamples)$' ./internal/locate/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

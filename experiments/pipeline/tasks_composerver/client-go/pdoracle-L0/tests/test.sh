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
echo "00d0d11a0500ddc6fda9c8fbc181cb68a53ad3ca6c32865c51fbfd46e1f44ef4  $HIDDEN/oracle/oracles/pdoracle_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/oracle/oracles/pdoracle_bb_prop_test.go"
install_hidden "oracle/oracles/pdoracle_bb_prop_test.go"
echo "00d0d11a0500ddc6fda9c8fbc181cb68a53ad3ca6c32865c51fbfd46e1f44ef4  /app/oracle/oracles/pdoracle_bb_prop_test.go" | sha256sum -c --status || checksum_fail "oracle/oracles/pdoracle_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestPdOracleUntilExpiredProperty|TestPdOracleIsExpiredProperty|TestPdOracleGetStaleTimestampProperty|TestPdOracleNonFutureStaleProperty|TestPdOracleTimestampMonotonicProperty|TestPdOracleAsyncFutureProperty)$' ./oracle/oracles/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

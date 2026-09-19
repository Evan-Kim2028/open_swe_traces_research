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
echo "ac2e9ddf74cdcdb76daed6eea6b5771951cb7c77f423488e899a305d11c39511  $HIDDEN/pkg/ignore/ignorerules_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/pkg/ignore/ignorerules_bb_prop_test.go"
install_hidden "pkg/ignore/ignorerules_bb_prop_test.go"
echo "ac2e9ddf74cdcdb76daed6eea6b5771951cb7c77f423488e899a305d11c39511  /app/pkg/ignore/ignorerules_bb_prop_test.go" | sha256sum -c --status || checksum_fail "pkg/ignore/ignorerules_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestIgnoreContractTableProperty|TestIgnoreParseAndDefaultsProperty|TestIgnoreOracleAgreementProperty|TestIgnoreUnseenRandomProperty)$' ./pkg/ignore/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

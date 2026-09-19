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
echo "103d61e0060499939db0fc8cce428e39bbd660e0b57c8c0462181bdf8a7ce5a3  $HIDDEN/pkg/provenance/provenance_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/pkg/provenance/provenance_bb_prop_test.go"
install_hidden "pkg/provenance/provenance_bb_prop_test.go"
echo "103d61e0060499939db0fc8cce428e39bbd660e0b57c8c0462181bdf8a7ce5a3  /app/pkg/provenance/provenance_bb_prop_test.go" | sha256sum -c --status || checksum_fail "pkg/provenance/provenance_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestProvContractTable|TestProvDigestProperty|TestProvDigestFileProperty|TestProvParseMessageBlockProperty|TestProvSignVerifyRoundTrip|TestProvVerifyProperty)$' ./pkg/provenance/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

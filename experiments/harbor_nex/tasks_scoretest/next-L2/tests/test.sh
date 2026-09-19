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
echo "48f9886ae60382de9af81968866b9c5dcac13f967b156c18745899fb831f8d1c  $HIDDEN/internal/unionstore/next_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/unionstore/next_bb_prop_test.go"
install_hidden "internal/unionstore/next_bb_prop_test.go"
echo "48f9886ae60382de9af81968866b9c5dcac13f967b156c18745899fb831f8d1c  /app/internal/unionstore/next_bb_prop_test.go" | sha256sum -c --status || checksum_fail "internal/unionstore/next_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestIteratorNextOrderProperty|TestIteratorNextContractExamples|TestIteratorNextUnmentionedRandom)$' ./internal/unionstore/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

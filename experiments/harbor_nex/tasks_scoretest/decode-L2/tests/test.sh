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
echo "2a5cd288d104f084b4c8538cd0435063ce118274f230582c3bae979ee9602d91  $HIDDEN/internal/mockstore/mockkv/decode_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/mockstore/mockkv/decode_bb_prop_test.go"
install_hidden "internal/mockstore/mockkv/decode_bb_prop_test.go"
echo "2a5cd288d104f084b4c8538cd0435063ce118274f230582c3bae979ee9602d91  /app/internal/mockstore/mockkv/decode_bb_prop_test.go" | sha256sum -c --status || checksum_fail "internal/mockstore/mockkv/decode_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestLockBinaryRoundTripProperty|TestValueBinaryRoundTripProperty|TestDecodeContractExamples|TestDecodeUnmentionedRandom)$' ./internal/mockstore/mockkv/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

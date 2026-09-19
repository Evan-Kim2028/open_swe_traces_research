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
echo "d4cff4357bc54666d1994244e54c142804f76330619b940c0442211f2333aa08  $HIDDEN/binding/bodydecoders_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/binding/bodydecoders_bb_prop_test.go"
install_hidden "binding/bodydecoders_bb_prop_test.go"
echo "d4cff4357bc54666d1994244e54c142804f76330619b940c0442211f2333aa08  /app/binding/bodydecoders_bb_prop_test.go" | sha256sum -c --status || checksum_fail "binding/bodydecoders_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestBDContractTableProperty|TestBDBindBodyProperty|TestBDJSONUseNumberProperty|TestBDJSONDisallowUnknownProperty|TestBDFormatDecodeValidateProperty|TestBDProtoBufProperty|TestBDBSONProperty|TestBDPlainAdversarialProperty|TestBDJSONUnseenRandomProperty)$' ./binding/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0

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
echo "c5276981a7dfccf05c441ea732430d69d422456aa534f4e18009049a378919b0  $HIDDEN/binding/requestbinders_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/binding/requestbinders_bb_prop_test.go"
install_hidden "binding/requestbinders_bb_prop_test.go"
echo "c5276981a7dfccf05c441ea732430d69d422456aa534f4e18009049a378919b0  /app/binding/requestbinders_bb_prop_test.go" | sha256sum -c --status || checksum_fail "binding/requestbinders_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestBBFormBinderQueryAndBody|TestBBFormBinderToleratesNonMultipart|TestBBFormPostBodyOnly|TestBBFormPostIgnoresQuery|TestBBQueryBinderURLOnly|TestBBQueryConversionError|TestBBHeaderBinderCanonical|TestBBHeaderBinderCaseInsensitive|TestBBUriBinderParams|TestBBUriNestedStruct|TestBBBinderNamesLowercase|TestBBValidationAfterMapping|TestBBValidationAfterMappingRandom)$' ./binding/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
